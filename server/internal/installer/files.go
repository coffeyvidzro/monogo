package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func prepareFilesystem() error {
	for _, directory := range []struct {
		path string
		mode os.FileMode
	}{
		{InstallRoot, 0o755},
		{ReleasesDir, 0o755},
		{ConfigRoot, 0o700},
		{CertificateDir, 0o700},
		{LicenseDir, 0o700},
		{StateRoot, 0o700},
		{InstallStateDir, 0o700},
		{ACMEWebroot, 0o755},
		{BackupsDir, 0o700},
		{StagingDir, 0o700},
		{LogRoot, 0o700},
		{RunRoot, 0o700},
	} {
		if err := ensureDirectory(directory.path, directory.mode); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirectory(path string, mode os.FileMode) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, mode); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		return os.Chmod(path, mode)
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing unsafe managed path %s", path)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("secure %s: %w", path, err)
	}
	return nil
}

func copyDir(source, destination string, fileMode os.FileMode) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported bundle file type: %s", path)
		}
		return copyFile(path, target, fileMode)
	})
}

func copyFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}
