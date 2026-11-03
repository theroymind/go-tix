package tix

import "errors"

const (
	optionPrefix      = "prefix"
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
	subcommandLs      = "ls"
	subcommandShow    = "show"
	optionType        = "type"

	typeNamesHintStart     = "; this project names them "
	typeNamesHintSeparator = ", "
	typeNamePairSeparator  = ":"
)

const (
	criteriaHeading   = "criteria:"
	criteriaSeparator = "\n"
	criterionLine     = "  %d. %s\n"
)

const (
	tableHeader           = "ID\tTYPE\tSTATUS\tPRIORITY\tTITLE"
	tableRow              = "%s\t%s\t%s\t%s\t%s\n"
	docTableHeader        = "ID	TYPE	STATUS\tTITLE"
	docTableRow           = "%s\t%s\t%s\t%s\n"
	tableAbsent           = "-"
	tableMinimumWidth     = 0
	tableTabWidth         = 0
	tablePadding          = 2
	tablePaddingCharacter = ' '
)

var (
	ErrExpectedOneTitle      = errors.New("expected exactly one title")
	ErrExpectedOneReference  = errors.New("expected exactly one reference")
	ErrInvalidOption         = errors.New("is invalid")
	ErrExpectedPair          = errors.New("expected KEY=VALUE")
	ErrExpectedPairs         = errors.New("expected a ticket or doc reference and at least one KEY=VALUE")
	ErrUnmetCriteria         = errors.New("has unmet criteria")
	ErrListField             = errors.New("is a list; set takes single-value fields")
	ErrExpectedNumbers       = errors.New("expected a ticket reference and at least one criterion number")
	ErrInvalidNumber         = errors.New("is not a criterion number; expected a whole number from 1")
	ErrExpectedSubcommand    = errors.New("expected ac set or ac add")
	ErrExpectedSetCriterion  = errors.New("expected ac set REF N TEXT")
	ErrExpectedAddCriterion  = errors.New("expected ac add REF TEXT")
	ErrExpectedDocsLs        = errors.New("expected docs ls")
	ErrExpectedDocSubcommand = errors.New("expected doc show or doc add")
	ErrExpectedDocShow       = errors.New("expected doc show REF")
	ErrExpectedDocAdd        = errors.New("expected doc add --type T TITLE")
	ErrExpectedOldAndNew     = errors.New("expected supersede OLD NEW")
	ErrSupersedesItself      = errors.New("can't supersede itself")
	ErrTypesDiffer           = errors.New("a doc is superseded only by one of its own type")
)
