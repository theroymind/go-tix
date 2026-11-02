package yamlstore

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrIDMismatch = errors.New("has id")

func idMismatch(path, fileID, expectedID string) error {
	return fmt.Errorf("%s %w %s, expected %s", path, ErrIDMismatch, strconv.Quote(fileID), strconv.Quote(expectedID))
}
