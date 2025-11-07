package fs

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

func EmbedToFS(dest string, fsys embed.FS) ([]string, error) {
	var saved []string

	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}
		_dest := filepath.Join(dest, path)
		if err := os.MkdirAll(filepath.Dir(_dest), os.ModePerm); err != nil {
			return err
		}
		if err := os.WriteFile(_dest, data, os.ModePerm); err != nil {
			return err
		}
		saved = append(saved, _dest)
		return nil
	}); err != nil {
		return nil, err
	}

	return saved, nil
}
