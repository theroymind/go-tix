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
	ErrInvalidChange      = errors.New("change rejected")
	ErrInvalidYAML        = errors.New("is not valid YAML")
	ErrMultipleDocuments  = errors.New("source contains multiple documents")
	ErrDuplicateKey       = errors.New("duplicate mapping key")
	ErrUnsupported        = errors.New("is not supported by the backend")
	ErrNotImplemented     = errors.New("is not implemented yet")
	// ErrRequiredField reports a RemoveField on a field the ticket or doc can't be without.
	ErrRequiredField = errors.New("is required and can't be removed")
	// ErrNoField reports a RemoveField on a field the ticket or doc doesn't hold.
	ErrNoField = errors.New("has no field")
)

func AlreadyInitialized(directory string) error {
	return fmt.Errorf("%s %w", directory, ErrAlreadyInitialized)
}

func NoMatch(reference string) error {
	return fmt.Errorf("%w %s", ErrNoMatch, strconv.Quote(reference))
}

// NoDocMatch reports a reference that names no doc. It is an ErrNoMatch to errors.Is, worded for docs.
func NoDocMatch(reference string) error {
	return noDocMatch{reference: reference}
}

type noDocMatch struct{ reference string }

func (err noDocMatch) Error() string { return "no doc matches " + strconv.Quote(err.reference) }

func (noDocMatch) Is(target error) bool { return target == ErrNoMatch }

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

// InvalidChange reports problems a change would leave in the valid file at path, which is left as it was.
func InvalidChange(path string, problems error) error {
	return fmt.Errorf("%s: %w:\n%w", path, ErrInvalidChange, problems)
}

func invalidYAML(path string, problems error) error {
	return fmt.Errorf("%s %w:\n%w", path, ErrInvalidYAML, problems)
}

// Unsupported reports a change that the kind of backend can't hold.
func Unsupported(change Change, kind Kind) error {
	return fmt.Errorf("%T %w %s", change, ErrUnsupported, kind)
}

// RequiredField reports a field that the file at path can't be without.
func RequiredField(path, field string) error {
	return fmt.Errorf("%s: %s %w", path, field, ErrRequiredField)
}

// NoField reports a field that the file at path doesn't hold.
func NoField(path, field string) error {
	return fmt.Errorf("%s %w %s", path, ErrNoField, field)
}

func notImplemented(kind Kind) error {
	return fmt.Errorf("backend %s %w", kind, ErrNotImplemented)
}
