package tix

import "errors"

const (
	optionPrefix      = "prefix"
	optionType        = "type"
	optionPriority    = "priority"
	optionTag         = "tag"
	optionAcceptance  = "acceptance"
	optionDescription = "description"
	optionDraft       = "draft"
	optionJSON        = "json"
	optionStatus      = "status"
	optionParent      = "parent"
	optionAll         = "all"
	longOptionStart   = "--"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
)

const (
	tableHeader           = "ID\tTYPE\tSTATUS\tPRIORITY\tTITLE"
	tableRow              = "%s\t%s\t%s\t%d\t%s\n"
	tableMinimumWidth     = 0
	tableTabWidth         = 0
	tablePadding          = 2
	tablePaddingCharacter = ' '
)

var (
	ErrExpectedOneTitle     = errors.New("expected exactly one title")
	ErrExpectedOneReference = errors.New("expected exactly one ticket reference")
	ErrInvalidOption        = errors.New("is invalid")
)
