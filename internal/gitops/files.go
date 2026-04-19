package gitops

import (
	"os"
	"path/filepath"
)

// writeFile creates path's parent directories (if needed) and writes
// content. It is deliberately a separate file from gitops.go so we
// can stub it in tests if we ever need to.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
