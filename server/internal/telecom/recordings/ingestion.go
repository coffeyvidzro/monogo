package recordings

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
)

// DefaultStagingPath is the shared FreeSWITCH/worker recording volume.
const DefaultStagingPath = "/var/lib/freeswitch/recordings"

type IngestionConfig struct {
	Interval    time.Duration
	Lease       time.Duration
	BaseRetry   time.Duration
	MaxRetry    time.Duration
	MaxAttempts int32
	BatchSize   int32
	StagingPath string
}

func DefaultIngestionConfig(stagingPath string) IngestionConfig {
	return IngestionConfig{Interval: 5 * time.Second, Lease: 2 * time.Minute, BaseRetry: 5 * time.Second, MaxRetry: 15 * time.Minute, MaxAttempts: 10, BatchSize: 50, StagingPath: stagingPath}
}

type ingestionRepository interface {
	ListForUpload(context.Context, time.Time, time.Time, int32) ([]sqlc.Recording, error)
	CompleteUpload(context.Context, sqlc.Recording, string, string, string, string, int64) (sqlc.Recording, error)
	RetryUpload(context.Context, sqlc.Recording, time.Time, string) error
	Fail(context.Context, sqlc.Recording) (sqlc.Recording, error)
}

type IngestionJob struct {
	repo    ingestionRepository
	storage *ObjectStorage
	config  IngestionConfig
	now     func() time.Time
}

func NewIngestionJob(repo ingestionRepository, storage *ObjectStorage, config IngestionConfig) (*IngestionJob, error) {
	if repo == nil || storage == nil {
		return nil, fmt.Errorf("recording ingestion requires repository and storage")
	}
	defaults := DefaultIngestionConfig(config.StagingPath)
	if config.Interval <= 0 {
		config.Interval = defaults.Interval
	}
	if config.Lease <= 0 {
		config.Lease = defaults.Lease
	}
	if config.BaseRetry <= 0 {
		config.BaseRetry = defaults.BaseRetry
	}
	if config.MaxRetry <= 0 {
		config.MaxRetry = defaults.MaxRetry
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = defaults.MaxAttempts
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaults.BatchSize
	}
	config.StagingPath = filepath.Clean(strings.TrimSpace(config.StagingPath))
	if !filepath.IsAbs(config.StagingPath) {
		return nil, fmt.Errorf("recording staging path must be absolute")
	}
	return &IngestionJob{repo: repo, storage: storage, config: config, now: time.Now}, nil
}

func (j *IngestionJob) Run(ctx context.Context) error {
	j.runPass(ctx)
	ticker := time.NewTicker(j.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			j.runPass(ctx)
		}
	}
}

func (j *IngestionJob) runPass(ctx context.Context) {
	if err := j.Ingest(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("recording ingestion pass failed: %v", err)
	}
}

func (j *IngestionJob) Ingest(ctx context.Context) error {
	now := j.now().UTC()
	items, err := j.repo.ListForUpload(ctx, now, now.Add(j.config.Lease), j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim recording uploads: %w", err)
	}
	for _, item := range items {
		if err := j.ingestOne(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (j *IngestionJob) ingestOne(ctx context.Context, recording sqlc.Recording) error {
	path, err := j.safeSourcePath(recording)
	if err != nil {
		return j.retryOrFail(ctx, recording, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return j.retryOrFail(ctx, recording, fmt.Errorf("open staged recording: %w", err))
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return j.retryOrFail(ctx, recording, fmt.Errorf("stat staged recording: %w", err))
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return j.retryOrFail(ctx, recording, fmt.Errorf("staged recording is not a non-empty regular file"))
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "bin"
	}
	key := fmt.Sprintf("recordings/%s/%s/%s.%s", recording.OrganizationID, j.now().UTC().Format("2006/01/02"), recording.ID, ext)
	contentType := mime.TypeByExtension("." + ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := j.storage.client.Put(ctx, key, contentType, file, info.Size()); err != nil {
		return j.retryOrFail(ctx, recording, err)
	}
	if _, err := j.repo.CompleteUpload(ctx, recording, key, "s3", j.storage.client.Bucket(), ext, info.Size()); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("complete recording upload %s: %w", recording.ID, err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove staged recording %s: %v", recording.ID, err)
	}
	return nil
}

func (j *IngestionJob) safeSourcePath(recording sqlc.Recording) (string, error) {
	if recording.SourcePath == nil {
		return "", fmt.Errorf("recording source path is missing")
	}
	path := filepath.Clean(strings.TrimSpace(*recording.SourcePath))
	relative, err := filepath.Rel(j.config.StagingPath, path)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("recording source path is outside staging directory")
	}
	resolvedRoot, err := filepath.EvalSymlinks(j.config.StagingPath)
	if err != nil {
		return "", fmt.Errorf("resolve recording staging directory: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve staged recording: %w", err)
	}
	resolvedRelative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || resolvedRelative == "." || resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("recording source path resolves outside staging directory")
	}
	return resolvedPath, nil
}

func (j *IngestionJob) retryOrFail(ctx context.Context, recording sqlc.Recording, cause error) error {
	if recording.UploadAttempts+1 >= j.config.MaxAttempts {
		if _, err := j.repo.Fail(ctx, recording); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("fail recording upload %s: %w", recording.ID, err)
		}
		return nil
	}
	delay := j.config.BaseRetry
	for attempt := int32(0); attempt < recording.UploadAttempts && delay < j.config.MaxRetry; attempt++ {
		delay *= 2
	}
	if delay > j.config.MaxRetry {
		delay = j.config.MaxRetry
	}
	message := cause.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	if err := j.repo.RetryUpload(ctx, recording, j.now().UTC().Add(delay), message); err != nil {
		return fmt.Errorf("schedule recording upload retry %s: %w", recording.ID, err)
	}
	return nil
}
