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
	optionDepth       = "depth"
	optionTop         = "top"
	pairSeparator     = "="
	longOptionStart   = "--"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
	subcommandSet     = "set"
	subcommandAdd     = "add"
	subcommandLs      = "ls"
	subcommandShow    = "show"
	optionType        = "type"

	typeNamesLegend        = "default|project: "
	typeNamesHintSeparator = ", "
	typeNameSeparator      = "|"

	dependencyPathSeparator = " -> "
	dependenciesHint        = "change dependencies with tix dep and tix undep"
	criteriaHint            = "change criteria with tix ac set and tix ac add"
	listHint                = "write it as a list, such as [a, b]"
	listStart               = "["
	listEnd                 = "]"
	wikilinkStart           = "[["
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

// DefaultContextDepth is how many hops tix context walks from its note when --depth isn't given.
const DefaultContextDepth = 2

// DefaultSearchTop is how many hits tix search prints when --top isn't given.
const DefaultSearchTop = 10

const (
	hitTableHeader = "SCORE\tID\tTYPE\tSUMMARY"
	hitTableRow    = "%.3f\t%s\t%s\t%s\n"
	indexedMessage = "indexed %d notes (%d embedded) at %s\n"
)

const (
	cardTableHeader = "DISTANCE\tRELATION\tID\tTYPE\tCOMMITS\tSUMMARY"
	cardTableRow    = "%d\t%s\t%s\t%s\t%s\t%s\n"
	missingSummary  = "(no summary)"
	shortHashLength = 7
	commitSeparator = ","
)

var (
	ErrExpectedOneTitle      = errors.New("expected exactly one title")
	ErrExpectedOneReference  = errors.New("expected exactly one reference")
	ErrInvalidOption         = errors.New("is invalid")
	ErrExpectedPair          = errors.New("expected KEY=VALUE")
	ErrExpectedPairs         = errors.New("expected a ticket or doc reference and at least one KEY=VALUE")
	ErrUnmetCriteria         = errors.New("has unmet criteria")
	ErrListField             = errors.New("is a list; set takes single-value fields")
	ErrInvalidList           = errors.New("is not a list of strings, such as [a, b]")
	ErrExpectedKeys          = errors.New("expected a ticket or doc reference and at least one KEY")
	ErrExpectedDependencies  = errors.New("expected a ticket reference and at least one dependency reference")
	ErrDependencyCycle       = errors.New("the dependency would form a cycle")
	ErrNotDependency         = errors.New("does not depend on")
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
	ErrInvalidDepth          = errors.New("is not a depth; expected a whole number of hops from 0")
	ErrInvalidTop            = errors.New("is not a count; expected a whole number from 1")
	ErrExpectedOneQuery      = errors.New("expected exactly one query")
	ErrNoEmbeddings          = errors.New("config names no embeddings endpoint and model")
)
