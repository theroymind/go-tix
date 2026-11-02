package ticket

import "regexp"

const (
	IDCharacters      = "0123456789abcdefghjkmnpqrstvwxyz"
	IDSuffixLength    = 4
	IDSeparator       = "-"
	MaximumIDAttempts = 10
	prefixSource      = "[a-z][a-z0-9]{0,7}"
	PrefixPattern     = "^" + prefixSource + "$"
	IDPattern         = "^[a-z0-9][a-z0-9-]*$"
)

const (
	DirectoryName  = ".tix"
	ConfigFileName = "config.yml"
	FileExtension  = ".yml"
)

const (
	problemInvalidID     = "invalid ticket id"
	problemTitleRequired = "title is required"
	yamlIntegerTag       = "!!int"
)

var (
	prefixExpression = regexp.MustCompile(PrefixPattern)
	idExpression     = regexp.MustCompile(IDPattern)
)
