package vaultstore

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/contracts/ticket"
	"github.com/theroymind/railyard/internal/tix/backend"
)

const listSeparator = ", "

var (
	ErrUnknownStatus = errors.New("has a status the profile doesn't map")
	// ErrUnknownType reports a doc type the profile doesn't define.
	ErrUnknownType    = errors.New("is not a doc type in the profile")
	ErrDuplicateSlug  = errors.New("shares its slug with")
	ErrTooManyParents = errors.New("links more than one parent")
	// ErrCriterionOutOfRange reports a criterion index below 1 or past the note's last criterion.
	ErrCriterionOutOfRange = errors.New("is out of range")
	// ErrNoTemplate reports a ticket type that no note type with a template in the profile maps onto.
	ErrNoTemplate = errors.New("has no note type with a template in the profile")
	// ErrNoDocTemplate reports a doc type with no template in the profile.
	ErrNoDocTemplate = errors.New("has no template in the profile")
	// ErrEmptySlug reports a title with no ASCII letter or digit to name a note by.
	ErrEmptySlug = errors.New("has no letter or digit to name a note by")
	// ErrSlugTaken reports a new note's slug that a ticket note, or a file at its path, already has.
	ErrSlugTaken = errors.New("is taken by")
	// ErrNoDestination reports a note type with no existing notes to place a new note beside.
	ErrNoDestination = errors.New("has no notes to place a new note beside")
	// ErrNotKept reports a new note that would read back with a field other than the one given.
	ErrNotKept = errors.New("would not read back")
)

func unknownStatus(path, status string, allowed []string) error {
	quoted := make([]string, len(allowed))
	for index, name := range allowed {
		quoted[index] = strconv.Quote(name)
	}
	return fmt.Errorf("%s %w: %s, expected one of %s", path, ErrUnknownStatus, strconv.Quote(status), strings.Join(quoted, listSeparator))
}

func unknownType(docType string, allowed []string) error {
	quoted := make([]string, len(allowed))
	for index, name := range allowed {
		quoted[index] = strconv.Quote(name)
	}
	return fmt.Errorf("type %s %w, expected one of %s", strconv.Quote(docType), ErrUnknownType, strings.Join(quoted, listSeparator))
}

func duplicateSlug(path, otherPath string) error {
	return fmt.Errorf("%s %w %s", path, ErrDuplicateSlug, otherPath)
}

func tooManyParents(path, property string, links []string) error {
	return fmt.Errorf("%s %w in %s: %s", path, ErrTooManyParents, property, strings.Join(links, listSeparator))
}

func noTemplate(ticketType ticket.Type) error {
	return fmt.Errorf("ticket type %s %w", strconv.Quote(string(ticketType)), ErrNoTemplate)
}

func noDocTemplate(docType string, templated []string) error {
	if len(templated) == 0 {
		return fmt.Errorf("doc type %s %w, and no doc type has one", strconv.Quote(docType), ErrNoDocTemplate)
	}
	quoted := make([]string, len(templated))
	for index, name := range templated {
		quoted[index] = strconv.Quote(name)
	}
	return fmt.Errorf("doc type %s %w; the types with one are %s", strconv.Quote(docType), ErrNoDocTemplate, strings.Join(quoted, listSeparator))
}

func emptySlug(title string) error {
	return fmt.Errorf("title %s %w", strconv.Quote(title), ErrEmptySlug)
}

func slugTaken(slug, path string) error {
	return fmt.Errorf("slug %s %w %s", strconv.Quote(slug), ErrSlugTaken, path)
}

func noDestination(noteType string) error {
	return fmt.Errorf("note type %s %w", strconv.Quote(noteType), ErrNoDestination)
}

func notKept(path, field string) error {
	return fmt.Errorf("%s %w the %s given", path, ErrNotKept, field)
}

// fieldUnsupported reports a ticket field that the vault backend can't set as a property.
func fieldUnsupported(field string) error {
	return fmt.Errorf("SetField %s %w %s", field, backend.ErrUnsupported, backend.KindVault)
}

func criterionOutOfRange(path string, index, count int) error {
	return fmt.Errorf("%s: criterion %d %w, the note has %d", path, index, ErrCriterionOutOfRange, count)
}
