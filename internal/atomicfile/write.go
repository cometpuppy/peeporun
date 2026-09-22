// Package atomicfile writes complete files without exposing partial contents.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data to a temporary file beside path, syncs it, and then
// atomically renames it over path. The previous file remains intact if any
// step before the rename fails.
func WriteFile(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmp != nil {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("set permissions on temporary file for %s: %w", path, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temporary file for %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary file for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, err)
	}
	tmp = nil
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
