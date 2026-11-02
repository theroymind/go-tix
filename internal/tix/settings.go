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
	optionForce       = "force"
	pairSeparator     = "="
	longOptionStart   = "--"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
)

const (
	tableHeader           = "ID\tTYPE\tSTATUS\tPRIORITY\tTITLE"
	tableRow              = "%s\t%s\t%s\t%s\t%s\n"
	tableAbsent           = "-"
	tableMinimumWidth     = 0
	tableTabWidth         = 0
	tablePadding          = 2
	tablePaddingCharacter = ' '
)

var (
	ErrExpectedOneTitle     = errors.New("expected exactly one title")
	ErrExpectedOneReference = errors.New("expected exactly one ticket reference")
	ErrInvalidOption        = errors.New("is invalid")
	ErrExpectedPair         = errors.New("expected KEY=VALUE")
	ErrExpectedPairs        = errors.New("expected a ticket reference and at least one KEY=VALUE")
	ErrUnmetCriteria        = errors.New("has unmet criteria")
	ErrListField            = errors.New("is a list; set takes single-value fields")
)
