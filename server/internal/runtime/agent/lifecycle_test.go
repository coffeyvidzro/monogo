package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/ai"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const testMediaControlToken = "0123456789abcdef0123456789abcdef"

func TestLifecycleDuplicateChannelAnswerAttachesOnce(t *testing.T) {
	db := newLifecycleDB()
	media := newLifecycleMediaServer(t)
	freeSwitch := newLifecycleFreeSWITCHServer(t, false)
	runtime := newLifecycleRuntime(t, db, media.URL, freeSwitch)

	channelID := uuid.NewString()
	call := sqlc.Call{
		ID:             db.callID,
		OrganizationID: db.organizationID,
		VoiceAgentID:   &db.agent.ID,
	}
	event := calling.LifecycleEvent{
		CallID:     call.ID,
		ChannelID:  channelID,
		Type:       calling.LifecycleAnswered,
		OccurredAt: time.Now().UTC(),
	}

	if err := runtime.HandleLifecycle(context.Background(), call, event); err != nil {
		t.Fatalf("first HandleLifecycle() error = %v", err)
	}
	if err := runtime.HandleLifecycle(context.Background(), call, event); err != nil {
		t.Fatalf("duplicate HandleLifecycle() error = %v", err)
	}

	if got := db.createCount(); got != 1 {
		t.Fatalf("durable session creates = %d, want 1", got)
	}
	if got := media.createCount(); got != 1 {
		t.Fatalf("media session creates = %d, want 1", got)
	}
	if got := freeSwitch.audioForkStarts(); got != 1 {
		t.Fatalf("audio fork starts = %d, want 1", got)
	}
	if got := freeSwitch.audioForkCommand(); strings.Count(got, " ") != 6 {
		t.Fatalf("audio fork command = %q, want no metadata argument", got)
	}
	if got := freeSwitch.audioClockStarts(); got != 1 {
		t.Fatalf("audio clock starts = %d, want 1", got)
	}
	if got := freeSwitch.sessionVariable(); got != db.sessionID.String() {
		t.Fatalf("Voice Agent session channel variable = %q, want %q", got, db.sessionID)
	}
	if got := db.sessionState(); got != "active" {
		t.Fatalf("durable session state = %q, want active", got)
	}
}

func TestLifecycleAudioForkFailureCleansUpAttachment(t *testing.T) {
	db := newLifecycleDB()
	media := newLifecycleMediaServer(t)
	freeSwitch := newLifecycleFreeSWITCHServer(t, true)
	runtime := newLifecycleRuntime(t, db, media.URL, freeSwitch)

	call := sqlc.Call{
		ID:             db.callID,
		OrganizationID: db.organizationID,
		VoiceAgentID:   &db.agent.ID,
	}
	event := calling.LifecycleEvent{
		CallID:     call.ID,
		ChannelID:  uuid.NewString(),
		Type:       calling.LifecycleAnswered,
		OccurredAt: time.Now().UTC(),
	}

	err := runtime.HandleLifecycle(context.Background(), call, event)
	if err == nil || !strings.Contains(err.Error(), "start Voice Agent audio fork") {
		t.Fatalf("HandleLifecycle() error = %v", err)
	}

	if got := media.createCount(); got != 1 {
		t.Fatalf("media session creates = %d, want 1", got)
	}
	if got := media.stopCount(); got != 1 {
		t.Fatalf("media session stops = %d, want 1", got)
	}
	if got := freeSwitch.audioForkStarts(); got != 1 {
		t.Fatalf("audio fork starts = %d, want 1", got)
	}
	if got := db.completeCount(); got != 1 {
		t.Fatalf("durable session completions = %d, want 1", got)
	}
	if got := db.sessionState(); got != "failed" {
		t.Fatalf("durable session state = %q, want failed", got)
	}
	if got := freeSwitch.sessionVariable(); got != "" {
		t.Fatalf("Voice Agent session channel variable = %q, want empty", got)
	}
}

func newLifecycleRuntime(
	t *testing.T,
	db *lifecycleDB,
	mediaURL string,
	freeSwitchServer *lifecycleFreeSWITCHServer,
) *Runtime {
	t.Helper()

	freeSwitchConfig := freeswitch.DefaultConfig(freeSwitchServer.address(), "secret")
	freeSwitchConfig.ConnectTimeout = time.Second
	freeSwitchConfig.CommandTimeout = time.Second
	freeSwitchConfig.ReconnectMinDelay = time.Second
	freeSwitchConfig.ReconnectMaxDelay = time.Second
	freeSwitchClient, err := freeswitch.New(freeSwitchConfig)
	if err != nil {
		t.Fatalf("freeswitch.New() error = %v", err)
	}
	if err := freeSwitchClient.Connect(context.Background()); err != nil {
		t.Fatalf("FreeSWITCH Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = freeSwitchClient.Close() })

	module := ai.New(sqlc.New(db))
	cfg := DefaultConfig(mediaURL, testMediaControlToken)
	cfg.RequestTimeout = time.Second
	runtime, err := New(module.Orchestration, freeSwitchClient, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return runtime
}

type lifecycleDB struct {
	mu             sync.Mutex
	organizationID uuid.UUID
	callID         uuid.UUID
	agent          sqlc.VoiceAgent
	sessionID      uuid.UUID
	session        *sqlc.VoiceAgentSession
	creates        int
	completes      int
}

func newLifecycleDB() *lifecycleDB {
	now := pgTimestamp(time.Now().UTC())
	organizationID := uuid.New()
	return &lifecycleDB{
		organizationID: organizationID,
		callID:         uuid.New(),
		sessionID:      uuid.New(),
		agent: sqlc.VoiceAgent{
			ID:             uuid.New(),
			OrganizationID: organizationID,
			Name:           "support",
			Engine:         "composable",
			Instructions:   "Help the caller.",
			Status:         "active",
			EngineConfig:   []byte(`{}`),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
}

func (db *lifecycleDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected lifecycle test Exec")
}

func (db *lifecycleDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("unexpected lifecycle test Query")
}

func (db *lifecycleDB) QueryRow(_ context.Context, query string, args ...interface{}) pgx.Row {
	db.mu.Lock()
	defer db.mu.Unlock()

	switch {
	case strings.Contains(query, "-- name: GetActiveVoiceAgentSessionByCallID"):
		if db.session == nil || db.session.State != "active" {
			return lifecycleRow{err: pgx.ErrNoRows}
		}
		return lifecycleRow{values: voiceAgentSessionValues(*db.session)}
	case strings.Contains(query, "-- name: GetVoiceAgentByID"):
		return lifecycleRow{values: voiceAgentValues(db.agent)}
	case strings.Contains(query, "-- name: CreateVoiceAgentSession"):
		db.creates++
		now := pgTimestamp(time.Now().UTC())
		db.session = &sqlc.VoiceAgentSession{
			ID:                   db.sessionID,
			OrganizationID:       db.organizationID,
			CallID:               db.callID,
			VoiceAgentID:         db.agent.ID,
			Engine:               db.agent.Engine,
			InstructionsSnapshot: db.agent.Instructions,
			EngineConfigSnapshot: append([]byte(nil), db.agent.EngineConfig...),
			Voice:                db.agent.Voice,
			Language:             db.agent.Language,
			State:                "active",
			StartedAt:            now,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		return lifecycleRow{values: voiceAgentSessionValues(*db.session)}
	case strings.Contains(query, "-- name: CompleteVoiceAgentSession"):
		if db.session == nil || db.session.State != "active" {
			return lifecycleRow{err: pgx.ErrNoRows}
		}
		db.completes++
		db.session.State = args[0].(string)
		db.session.TurnCount = args[1].(int32)
		db.session.InterruptionCount = args[2].(int32)
		db.session.FirstResponseLatencyMs, _ = args[3].(*int32)
		db.session.AvgTurnLatencyMs, _ = args[4].(*int32)
		db.session.EndedAt = args[5].(pgtype.Timestamptz)
		db.session.UpdatedAt = db.session.EndedAt
		return lifecycleRow{values: voiceAgentSessionValues(*db.session)}
	default:
		return lifecycleRow{err: errors.New("unexpected lifecycle test QueryRow")}
	}
}

func (db *lifecycleDB) createCount() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.creates
}

func (db *lifecycleDB) completeCount() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.completes
}

func (db *lifecycleDB) sessionState() string {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.session == nil {
		return ""
	}
	return db.session.State
}

type lifecycleRow struct {
	values []interface{}
	err    error
}

func (r lifecycleRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destinations = %d, values = %d", len(dest), len(r.values))
	}
	for i := range dest {
		target := reflect.ValueOf(dest[i])
		if !target.IsValid() || target.Kind() != reflect.Pointer || target.IsNil() {
			return fmt.Errorf("scan destination %d is not a writable pointer", i)
		}
		value := reflect.ValueOf(r.values[i])
		if !value.IsValid() {
			target.Elem().Set(reflect.Zero(target.Elem().Type()))
			continue
		}
		if !value.Type().AssignableTo(target.Elem().Type()) {
			return fmt.Errorf(
				"scan value %d has type %s, destination type is %s",
				i,
				value.Type(),
				target.Elem().Type(),
			)
		}
		target.Elem().Set(value)
	}
	return nil
}

func voiceAgentValues(agent sqlc.VoiceAgent) []interface{} {
	return []interface{}{
		agent.ID,
		agent.OrganizationID,
		agent.Name,
		agent.Engine,
		agent.Instructions,
		agent.Voice,
		agent.Language,
		agent.Status,
		agent.EngineConfig,
		agent.CreatedAt,
		agent.UpdatedAt,
	}
}

func voiceAgentSessionValues(record sqlc.VoiceAgentSession) []interface{} {
	return []interface{}{
		record.ID,
		record.OrganizationID,
		record.CallID,
		record.VoiceAgentID,
		record.Engine,
		record.InstructionsSnapshot,
		record.EngineConfigSnapshot,
		record.Voice,
		record.Language,
		record.State,
		record.TurnCount,
		record.InterruptionCount,
		record.FirstResponseLatencyMs,
		record.AvgTurnLatencyMs,
		record.StartedAt,
		record.EndedAt,
		record.CreatedAt,
		record.UpdatedAt,
	}
}

func pgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

type lifecycleMediaServer struct {
	*httptest.Server
	mu      sync.Mutex
	creates int
	deletes int
	stops   int
}

func newLifecycleMediaServer(t *testing.T) *lifecycleMediaServer {
	t.Helper()
	server := &lifecycleMediaServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testMediaControlToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet &&
			strings.HasPrefix(r.URL.Path, "/internal/v1/sessions/") &&
			strings.HasSuffix(r.URL.Path, "/control"):
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = ws.CloseNow() }()
			for {
				kind, payload, err := ws.Read(r.Context())
				if err != nil {
					return
				}
				if kind != websocket.MessageText {
					return
				}
				var command struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(payload, &command) != nil {
					return
				}
				if command.Type == "session.stop" {
					server.mu.Lock()
					server.stops++
					server.mu.Unlock()
					return
				}
			}
		case r.Method == http.MethodPost && r.URL.Path == "/internal/v1/sessions":
			server.mu.Lock()
			server.creates++
			server.mu.Unlock()
			controlURL := "ws" + strings.TrimPrefix(server.URL, "http") +
				"/internal/v1/sessions/" + serverSessionID(r).String() + "/control"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(
				w,
				`{"websocket_url":"ws://media.internal/v1/audio-forks?token=test","control_websocket_url":%q}`,
				controlURL,
			)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/internal/v1/sessions/"):
			server.mu.Lock()
			server.deletes++
			server.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func (s *lifecycleMediaServer) createCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.creates
}

func (s *lifecycleMediaServer) deleteCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deletes
}

func (s *lifecycleMediaServer) stopCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stops
}

func serverSessionID(r *http.Request) uuid.UUID {
	var cfg struct {
		ID uuid.UUID `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&cfg)
	return cfg.ID
}

type lifecycleFreeSWITCHServer struct {
	listener      net.Listener
	failAudioFork bool

	mu              sync.Mutex
	variable        string
	forkStarts      int
	audioClocks     int
	lastForkCommand string
}

func newLifecycleFreeSWITCHServer(t *testing.T, failAudioFork bool) *lifecycleFreeSWITCHServer {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake FreeSWITCH: %v", err)
	}
	server := &lifecycleFreeSWITCHServer{listener: listener, failAudioFork: failAudioFork}
	go server.serve()
	t.Cleanup(func() { _ = listener.Close() })
	return server
}

func (s *lifecycleFreeSWITCHServer) address() string {
	return s.listener.Addr().String()
}

func (s *lifecycleFreeSWITCHServer) sessionVariable() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.variable
}

func (s *lifecycleFreeSWITCHServer) audioForkStarts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.forkStarts
}

func (s *lifecycleFreeSWITCHServer) audioForkCommand() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastForkCommand
}

func (s *lifecycleFreeSWITCHServer) audioClockStarts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.audioClocks
}

func (s *lifecycleFreeSWITCHServer) serve() {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)

	if _, err := fmt.Fprint(conn, "Content-Type: auth/request\n\n"); err != nil {
		return
	}
	command, err := readLifecycleCommand(reader)
	if err != nil || command != "auth secret" {
		return
	}
	if _, err := fmt.Fprint(conn, "Content-Type: command/reply\nReply-Text: +OK accepted\n\n"); err != nil {
		return
	}

	for {
		command, err := readLifecycleCommand(reader)
		if err != nil {
			return
		}
		body := s.handle(command)
		if _, err := fmt.Fprintf(
			conn,
			"Content-Type: api/response\nContent-Length: %d\n\n%s",
			len(body),
			body,
		); err != nil {
			return
		}
	}
}

func (s *lifecycleFreeSWITCHServer) handle(command string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case strings.HasPrefix(command, "api uuid_getvar "):
		if s.variable == "" {
			return "_undef_"
		}
		return s.variable
	case strings.HasPrefix(command, "api uuid_audio_fork ") && strings.Contains(command, " start "):
		s.forkStarts++
		s.lastForkCommand = command
		if s.failAudioFork {
			return "-ERR simulated audio fork failure"
		}
		return "+OK"
	case strings.HasPrefix(command, "api uuid_broadcast ") &&
		strings.Contains(command, " silence_stream://-1 aleg"):
		s.audioClocks++
		return "+OK Message sent"
	case strings.HasPrefix(command, "api uuid_setvar "):
		fields := strings.Fields(command)
		if len(fields) >= 5 && fields[3] == voiceAgentSessionVariable {
			s.variable = fields[4]
		}
		return "+OK"
	case strings.HasPrefix(command, "api uuid_audio_fork ") && strings.HasSuffix(command, " stop"):
		return "+OK"
	default:
		return "+OK"
	}
}

func readLifecycleCommand(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	command := strings.TrimSpace(line)
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(line) == "" {
			return command, nil
		}
	}
}
