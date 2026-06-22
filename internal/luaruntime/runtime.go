package luaruntime

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed nvim
var embeddedFS embed.FS

// Extract writes the embedded Lua runtime to dst.
// Existing files are overwritten so the runtime is always up to date.
func Extract(dst string) error {
	return fs.WalkDir(embeddedFS, "nvim", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("nvim", path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0644)
	})
}
