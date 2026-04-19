package provider

import (
	"os"
	"path/filepath"
)

// writeFile is a tiny helper so tests can drop executables into
// t.TempDir without reaching for os.WriteFile directly.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o755)
}
