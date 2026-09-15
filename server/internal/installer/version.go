package installer

import (
	"os"
	"path/filepath"
	"strings"
)

func DefaultVersion() string {
	bundleDir, err := findBundleDir()
	if err != nil {
		return "preview"
	}
	content, err := os.ReadFile(filepath.Join(bundleDir, "VERSION"))
	if err != nil {
		return "preview"
	}
	version := strings.TrimSpace(string(content))
	if version == "" {
		return "preview"
	}
	return version
}
