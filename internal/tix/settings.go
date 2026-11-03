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
	optionDependency  = "dependency"
	optionAll         = "all"
	optionForce       = "force"
	pairSeparator     = "="
	longOptionStart   = "--"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
	subcommandSet     = "set"
	subcommandAdd     = "add"
)

const (
	criteriaHeading    = "criteria:"
	criteriaSeparator  = "\n"
	criterionLine      = "  %d. %s%s\n"
	criterionMet       = "[x] "
	criterionUnmet     = "[ ] "
	criterionUntracked = ""
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
	ErrExpectedNumbers      = errors.New("expected a ticket reference and at least one criterion number")
	ErrInvalidNumber        = errors.New("is not a criterion number; expected a whole number from 1")
	ErrExpectedSubcommand   = errors.New("expected ac set or ac add")
	ErrExpectedSetCriterion = errors.New("expected ac set REF N TEXT")
	ErrExpectedAddCriterion = errors.New("expected ac add REF TEXT")
)
