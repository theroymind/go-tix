package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const matchSeparator = ", "

var (
	ErrNotInRepository    = errors.New("not in a tix repo (run tix init)")
	ErrAlreadyInitialized = errors.New("already exists")
	ErrNoMatch            = errors.New("no ticket matches")
	ErrAmbiguous          = errors.New("is ambiguous")
	ErrInvalidFile        = errors.New("is invalid")
	ErrInvalidYAML        = errors.New("is not valid YAML")
	ErrIDMismatch         = errors.New("has id")
	ErrMultipleDocuments  = errors.New("source contains multiple documents")
	ErrDuplicateKey       = errors.New("duplicate mapping key")
)

func AlreadyInitialized(directory string) error {
	return fmt.Errorf("%s %w", directory, ErrAlreadyInitialized)
}

func noMatch(reference string) error {
	return fmt.Errorf("%w %s", ErrNoMatch, strconv.Quote(reference))
}

func ambiguous(reference string, matches []string) error {
	return fmt.Errorf("%s %w: %s", strconv.Quote(reference), ErrAmbiguous, strings.Join(matches, matchSeparator))
}

func InvalidFile(path string, problems error) error {
	return fmt.Errorf("%s %w:\n%w", path, ErrInvalidFile, problems)
}

func invalidYAML(path string, problems error) error {
	return fmt.Errorf("%s %w:\n%w", path, ErrInvalidYAML, problems)
}

func idMismatch(path, fileID, expectedID string) error {
	return fmt.Errorf("%s %w %s, expected %s", path, ErrIDMismatch, strconv.Quote(fileID), strconv.Quote(expectedID))
}
