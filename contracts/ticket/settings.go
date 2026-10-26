package ticket

import (
	"regexp"
	"strconv"
)

const (
	IDCharacters      = "0123456789abcdefghjkmnpqrstvwxyz"
	IDSuffixLength    = 4
	IDSeparator       = "-"
	MaximumIDAttempts = 10
	prefixSource      = "[a-z][a-z0-9]{0,7}"
	PrefixPattern     = "^" + prefixSource + "$"
)

const (
	problemInvalidID     = "invalid ticket id"
	problemTitleRequired = "title is required"
	yamlIntegerTag       = "!!int"
)

var (
	prefixExpression = regexp.MustCompile(PrefixPattern)
	idExpression     = regexp.MustCompile(
		"^" + prefixSource + regexp.QuoteMeta(IDSeparator) +
			"[" + IDCharacters + "]{" + strconv.Itoa(IDSuffixLength) + "}$",
	)
)
