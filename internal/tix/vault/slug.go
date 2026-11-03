package vault

import (
	"strings"
	"unicode"
)

// Slugify turns a title into a note's file name without ".md": lowercase, each run of characters outside a-z and
// 0-9 replaced by one "-", and no "-" at either end. Letters outside ASCII count as separators. The result is empty
// when the title holds no ASCII letter or digit.
func Slugify(title string) string {
	var slug strings.Builder
	isSeparatorPending := false
	for _, character := range strings.ToLower(title) {
		if !isSlugCharacter(character) {
			isSeparatorPending = slug.Len() > 0
			continue
		}
		if isSeparatorPending {
			slug.WriteString(slugSeparator)
			isSeparatorPending = false
		}
		slug.WriteRune(character)
	}
	return slug.String()
}

func isSlugCharacter(character rune) bool {
	return character <= unicode.MaxASCII && (unicode.IsLower(character) || unicode.IsDigit(character))
}
