package installer

import (
	"os"
	"os/exec"
	"path/filepath"
)

func runCompose(environmentPath string, args ...string) error {
	composeArgs := []string{
		"compose",
		"--env-file", environmentPath,
		"-f", filepath.Join(CurrentPath, "compose.yaml"),
	}
	composeArgs = append(composeArgs, args...)
	cmd := exec.Command("docker", composeArgs...)
	cmd.Dir = CurrentPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
