package vaultstore

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/theroymind/railyard/internal/tix/backend"
)

const listSeparator = ", "

var (
	ErrUnknownStatus  = errors.New("has a status the profile doesn't map")
	ErrDuplicateSlug  = errors.New("shares its slug with")
	ErrTooManyParents = errors.New("links more than one parent")
	// ErrCriterionOutOfRange reports a criterion index below 1 or past the note's last criterion.
	ErrCriterionOutOfRange = errors.New("is out of range")
)

func unknownStatus(path, status string, allowed []string) error {
	quoted := make([]string, len(allowed))
	for index, name := range allowed {
		quoted[index] = strconv.Quote(name)
	}
	return fmt.Errorf("%s %w: %s, expected one of %s", path, ErrUnknownStatus, strconv.Quote(status), strings.Join(quoted, listSeparator))
}

func duplicateSlug(path, otherPath string) error {
	return fmt.Errorf("%s %w %s", path, ErrDuplicateSlug, otherPath)
}

func tooManyParents(path, property string, links []string) error {
	return fmt.Errorf("%s %w in %s: %s", path, ErrTooManyParents, property, strings.Join(links, listSeparator))
}

// createUnsupported reports that the vault backend can't create tickets yet.
func createUnsupported() error {
	return fmt.Errorf("Create %w %s", backend.ErrUnsupported, backend.KindVault)
}

// fieldUnsupported reports a ticket field that the vault backend can't set as a property.
func fieldUnsupported(field string) error {
	return fmt.Errorf("SetField %s %w %s", field, backend.ErrUnsupported, backend.KindVault)
}

func criterionOutOfRange(path string, index, count int) error {
	return fmt.Errorf("%s: criterion %d %w, the note has %d", path, index, ErrCriterionOutOfRange, count)
}
