package ticket

import (
	"errors"
	"fmt"
	"io"
)

var ErrIDSpaceExhausted = errors.New("could not generate a unique id")

func IsValidID(id string) bool {
	return idExpression.MatchString(id)
}

func IsValidPrefix(prefix string) bool {
	return prefixExpression.MatchString(prefix)
}

func GenerateID(prefix string, random io.Reader, isTaken func(id string) bool) (string, error) {
	suffix := make([]byte, IDSuffixLength)
	for range MaximumIDAttempts {
		if _, err := io.ReadFull(random, suffix); err != nil {
			return "", fmt.Errorf("read random bytes: %w", err)
		}
		for index := range suffix {
			suffix[index] = IDCharacters[int(suffix[index])%len(IDCharacters)]
		}
		id := prefix + IDSeparator + string(suffix)
		if !isTaken(id) {
			return id, nil
		}
	}
	return "", ErrIDSpaceExhausted
}
