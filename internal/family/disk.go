package family

import (
	"io/fs"
	"os"
)

// osStat + osDirFS are indirection points so the pure-FS LoadDir can
// sit on top of the concrete disk interactions without family.go
// growing a direct os import.
func osStat(path string) (os.FileInfo, error) { return os.Stat(path) }
func osDirFS(path string) fs.FS               { return os.DirFS(path) }
