// Package fsutil provides file system utility functions.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
)

// renamer sets the function for renaming a file, defaulting to os.Rename.
// Changing renamer should only be done in tests.
var renamer = os.Rename

// BaseNoExt returns the last element of path with any file extensions removed.
func BaseNoExt(path string) string {
	base := filepath.Base(path)
	if base == "." || base == ".." {
		return base
	}
	if idx := strings.IndexByte(base, '.'); idx != -1 {
		return base[:idx]
	}
	return base
}

// Move moves a file or directory. It first tries to rename src to dst. If the
// rename fails due to the source and destination being on different file
// systems Move copies src to dst, then deletes src.
func Move(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return errors.New("destination already exists")
	}

	// Rename when possible.
	err := renamer(src, dst)
	if err == nil {
		return nil
	}

	// Copy and delete otherwise.
	lerr, _ := err.(*os.LinkError)
	if lerr.Err.Error() == "invalid cross-device link" {
		err := copy.Copy(src, dst, copy.Options{
			Sync:        true,
			OnDirExists: func(src, dst string) copy.DirExistsAction { return copy.Untouchable },
		})
		if err != nil {
			return err
		}
		return os.RemoveAll(src)
	}

	return err
}

// SetFileModes recursively sets the file mode of directory root and its
// contents.
func SetFileModes(root string, dirMode, fileMode int) error {
	return filepath.WalkDir(root,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			mode := fs.FileMode(fileMode)
			if d.IsDir() {
				mode = fs.FileMode(dirMode)
			}

			if err := os.Chmod(path, mode); err != nil {
				return fmt.Errorf("set file mode: %v", err)
			}

			return nil
		},
	)
}
