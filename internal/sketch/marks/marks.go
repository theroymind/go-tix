// Package marks says whether a note's target is new or exists, and whether each item it names is new or changes
// code that exists, by searching the target file for the item's name as a whole word.
package marks

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/theroymind/go-tix/contracts/sketch"
)

// Reader marks items against the files under Root, reading each file once.
type Reader struct {
	Root  string
	files map[string]*string
}

// NewReader returns a Reader for the files under root, the repo root a sketch's paths start from.
func NewReader(root string) *Reader {
	return &Reader{Root: root, files: map[string]*string{}}
}

// MarkNote marks value MarkChanged when its target exists and MarkNew when it doesn't, and marks each of its
// items as Mark does.
func (reader *Reader) MarkNote(value *sketch.Note) error {
	text, err := reader.read(value.Target)
	if err != nil {
		return err
	}
	value.Mark = sketch.MarkNew
	if text != nil {
		value.Mark = sketch.MarkChanged
	}
	for index := range value.Items {
		item := &value.Items[index]
		if item.Mark, err = reader.Mark(value.Target, item.Name); err != nil {
			return err
		}
	}
	return nil
}

// Mark returns MarkChanged when the file at the slash-separated repo path holds name as a whole word, so that
// score doesn't match score_strike, and MarkNew otherwise, including when the file doesn't exist yet.
func (reader *Reader) Mark(path, name string) (sketch.Mark, error) {
	text, err := reader.read(path)
	if err != nil {
		return "", err
	}
	if text != nil && containsWord(*text, name) {
		return sketch.MarkChanged, nil
	}
	return sketch.MarkNew, nil
}

// read returns the file's text, or nil for a file that doesn't exist.
func (reader *Reader) read(path string) (*string, error) {
	if text, cached := reader.files[path]; cached {
		return text, nil
	}
	bytes, err := os.ReadFile(filepath.Join(reader.Root, filepath.FromSlash(path)))
	var text *string
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		content := string(bytes)
		text = &content
	}
	reader.files[path] = text
	return text, nil
}

// containsWord reports whether name appears in text with no identifier character directly before or after it.
func containsWord(text, name string) bool {
	if name == "" {
		return false
	}
	for start := 0; ; {
		index := strings.Index(text[start:], name)
		if index < 0 {
			return false
		}
		begin, end := start+index, start+index+len(name)
		if !isIdentifierEnd(text[:begin], true) && !isIdentifierEnd(text[end:], false) {
			return true
		}
		start = begin + 1
	}
}

// isIdentifierEnd reports whether the rune touching name, the last of before or the first of after, is part of an
// identifier.
func isIdentifierEnd(text string, last bool) bool {
	if text == "" {
		return false
	}
	runes := []rune(text)
	character := runes[0]
	if last {
		character = runes[len(runes)-1]
	}
	return character == '_' || unicode.IsLetter(character) || unicode.IsDigit(character)
}
