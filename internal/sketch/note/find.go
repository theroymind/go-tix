package note

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/theroymind/go-tix/contracts/sketch"
)

// Find returns the slash-separated path from root of every note under root, in lexical order. It skips hidden
// folders, such as .git, and node_modules.
func Find(root string) ([]string, error) {
	var notes []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(name, hiddenPrefix) || name == nodeModules) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, sketch.NoteSuffix) {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		notes = append(notes, filepath.ToSlash(relative))
		return nil
	})
	return notes, err
}
