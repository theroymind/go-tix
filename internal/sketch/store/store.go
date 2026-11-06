// Package store loads and saves a feature's sketch.yml, writing it atomically so a reader never sees a partial
// file.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/theroymind/go-tix/contracts/sketch"
	"go.yaml.in/yaml/v3"
)

var (
	ErrNotFound          = errors.New("no sketch")
	ErrInvalidFile       = errors.New("invalid sketch file")
	ErrMultipleDocuments = errors.New("more than one YAML document")
)

// FeatureDir returns the default directory of feature from the repo root: docs/features/<feature>.
func FeatureDir(root, feature string) string {
	return filepath.Join(root, filepath.FromSlash(FeaturesDir), feature)
}

// Path returns the sketch file inside a feature's directory.
func Path(dir string) string {
	return filepath.Join(dir, FileName)
}

// Load reads, validates and trims the sketch in dir. It returns ErrNotFound when dir holds none, and ErrInvalidFile
// for a file that isn't one valid sketch, including one with a key the schema doesn't name.
func Load(dir string) (sketch.Sketch, error) {
	path := Path(dir)
	text, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return sketch.Sketch{}, fmt.Errorf("%w: %s", ErrNotFound, path)
	}
	if err != nil {
		return sketch.Sketch{}, err
	}
	value, err := decode(text)
	if err != nil {
		return sketch.Sketch{}, fmt.Errorf("%w %s: %w", ErrInvalidFile, path, err)
	}
	return value, nil
}

// Save trims value, validates it and writes it to dir's sketch file through a temporary file and a rename,
// creating dir when it doesn't exist. An invalid value writes nothing.
func Save(dir string, value sketch.Sketch) error {
	value = value.Trimmed()
	if err := value.Validate(); err != nil {
		return err
	}
	text, err := encode(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, directoryPermission); err != nil {
		return err
	}
	return writeAtomically(Path(dir), text)
}

func decode(text []byte) (sketch.Sketch, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	decoder.KnownFields(true)
	var value sketch.Sketch
	if err := decoder.Decode(&value); err != nil {
		return sketch.Sketch{}, err
	}
	var next yaml.Node
	if err := decoder.Decode(&next); !errors.Is(err, io.EOF) {
		if err == nil {
			err = ErrMultipleDocuments
		}
		return sketch.Sketch{}, err
	}
	if err := value.Validate(); err != nil {
		return sketch.Sketch{}, err
	}
	return value.Trimmed(), nil
}

func encode(value sketch.Sketch) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(yamlIndent)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// writeAtomically writes a uniquely named temporary file beside path, so two writers never share one, and
// renames it over path, removing it when any step fails.
func writeAtomically(path string, text []byte) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), temporaryPattern)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(temporary.Name())
		}
	}()
	if _, err = temporary.Write(text); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Chmod(filePermission); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
