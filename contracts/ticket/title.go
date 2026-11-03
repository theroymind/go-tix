package ticket

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// TitleMaximumLength is the most characters a title may hold once NormalizeTitle has flattened it.
const TitleMaximumLength = 72

var ErrTitleTooLong = errors.New("title is too long")

// NormalizeTitle flattens text onto one line, replacing each run of whitespace with one space and trimming the ends,
// and fails with ErrTitleTooLong when the result exceeds TitleMaximumLength characters. Validate does not apply it,
// so tickets written with longer or multi-line titles still load.
func NormalizeTitle(text string) (string, error) {
	title := strings.Join(strings.Fields(text), " ")
	if length := utf8.RuneCountInString(title); length > TitleMaximumLength {
		return "", fmt.Errorf("%w: %d characters, maximum %d", ErrTitleTooLong, length, TitleMaximumLength)
	}
	return title, nil
}
