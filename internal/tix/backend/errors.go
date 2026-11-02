package backend

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
	ErrNoCriterion        = errors.New("has no criterion")
	ErrInvalidFile        = errors.New("is invalid")
	ErrInvalidYAML        = errors.New("is not valid YAML")
	ErrMultipleDocuments  = errors.New("source contains multiple documents")
	ErrDuplicateKey       = errors.New("duplicate mapping key")
	ErrUnsupported        = errors.New("is not supported by the backend")
	ErrNotImplemented     = errors.New("is not implemented yet")
)

func AlreadyInitialized(directory string) error {
	return fmt.Errorf("%s %w", directory, ErrAlreadyInitialized)
}

func NoMatch(reference string) error {
	return fmt.Errorf("%w %s", ErrNoMatch, strconv.Quote(reference))
}

func Ambiguous(reference string, matches []string) error {
	return fmt.Errorf("%s %w: %s", strconv.Quote(reference), ErrAmbiguous, strings.Join(matches, matchSeparator))
}

// NoCriterion reports a 1-based criterion index the ticket doesn't have.
func NoCriterion(id string, index int) error {
	return fmt.Errorf("%s %w %d", id, ErrNoCriterion, index)
}

func InvalidFile(path string, problems error) error {
	return fmt.Errorf("%s %w:\n%w", path, ErrInvalidFile, problems)
}

func invalidYAML(path string, problems error) error {
	return fmt.Errorf("%s %w:\n%w", path, ErrInvalidYAML, problems)
}

// Unsupported reports a change that the kind of backend can't hold.
func Unsupported(change Change, kind Kind) error {
	return fmt.Errorf("%T %w %s", change, ErrUnsupported, kind)
}

func notImplemented(kind Kind) error {
	return fmt.Errorf("backend %s %w", kind, ErrNotImplemented)
}
