package tix

import "errors"

const (
	optionPrefix      = "prefix"
	optionBackend     = "backend"
	optionDocs        = "docs"
	optionPriority    = "priority"
	optionTag         = "tag"
	optionAcceptance  = "acceptance"
	optionDescription = "description"
	optionDraft       = "draft"
	optionJSON        = "json"
	optionStatus      = "status"
	optionParent      = "parent"
	optionDependency  = "dependency"
	optionExternalID  = "external-id"
	optionExternalURL = "external-url"
	optionAll         = "all"
	optionForce       = "force"
	optionDepth       = "depth"
	optionStaged      = "staged"
	pairSeparator     = "="
	longOptionStart   = "--"
	integerPattern    = `^-?\d+$`
	problemPriority   = "Invalid input: expected integer, received "
	subcommandSet     = "set"
	subcommandAdd     = "add"
	subcommandLs      = "ls"
	subcommandShow    = "show"
	subcommandHand    = "hand"
	optionType        = "type"
	optionOpen        = "open"
	optionSet         = "set"
	optionFile        = "file"
	optionWithParent  = "with-parent"

	commandAc                 = "ac"
	commandAcceptanceCriteria = "acceptance-criteria"
	commandLs                 = "ls"
	commandList               = "list"
	helpCommand               = "help"
	helpShortFlag             = "-h"
	helpLongFlag              = "--help"
	usageIndent               = "  "
	usageHeading              = "usage:\n"

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
	checkedCriterion  = "%d. [x] %s\n"
	openCriterion     = "%d. [ ] %s\n"
	handCheckMarker   = "Hand check: "
	openHandCheck     = "%s %d %s\n"
	showHeader        = "==> %s <==\n"
)

const (
	tableColumnID         = "ID"
	tableColumnType       = "TYPE"
	tableColumnStatus     = "STATUS"
	tableColumnPriority   = "PRIORITY"
	tableColumnTitle      = "TITLE"
	tableSeparator        = "\t"
	docTableHeader        = "ID	TYPE	STATUS\tTITLE"
	docTableRow           = "%s\t%s\t%s\t%s\n"
	tableAbsent           = "-"
	tableTreeIndent       = "  "
	tableMinimumWidth     = 0
	tableTabWidth         = 0
	tablePadding          = 2
	tablePaddingCharacter = ' '
)

const (
	promptPlain           = "%s: "
	promptWithDefault     = "%s [%s]: "
	promptProblem         = "%v\n"
	initializedMessage    = "wrote %s\n"
	questionPrefix        = "Ticket id prefix"
	questionBackend       = "Backend (yaml or vault)"
	questionDocsDirectory = "Docs directory, relative to this one"
)

// DefaultContextDepth is how many hops tix context walks from its note when --depth isn't given.
const DefaultContextDepth = 2

const findingLine = "%s: %s: %s\n"

// The staged notes tix gate --staged reads: added or modified files under the working directory, NUL-separated,
// with renames split into a deletion and an addition.
const (
	gitExecutable  = "git"
	statusAdded    = "A"
	statusModified = "M"
	nulSeparator   = 0
)

var stagedArguments = []string{"diff", "--cached", "--name-status", "--no-renames", "--relative", "-z", "--diff-filter=AM"}

const (
	cardTableHeader = "DISTANCE\tRELATION\tID\tTYPE\tCOMMITS\tSUMMARY"
	cardTableRow    = "%d\t%s\t%s\t%s\t%s\t%s\n"
	missingSummary  = "(no summary)"
	shortHashLength = 7
	commitSeparator = ","
)

// commandAliases maps each alias Program registers to the command it runs, whose usage it shares.
var commandAliases = map[string]string{commandAcceptanceCriteria: commandAc, commandList: commandLs}

var (
	ErrExpectedOneTitle           = errors.New("expected exactly one title")
	ErrExpectedOneReference       = errors.New("expected exactly one reference")
	ErrInvalidOption              = errors.New("is invalid")
	ErrExpectedPair               = errors.New("expected KEY=VALUE")
	ErrExpectedPairs              = errors.New("expected a ticket or doc reference and at least one KEY=VALUE")
	ErrUnmetCriteria              = errors.New("has unmet criteria")
	ErrListField                  = errors.New("is a list; set takes single-value fields")
	ErrInvalidList                = errors.New("is not a list of strings, such as [a, b]")
	ErrExpectedKeys               = errors.New("expected a ticket or doc reference and at least one KEY")
	ErrExpectedDependencies       = errors.New("expected a ticket reference and at least one dependency reference")
	ErrDependencyCycle            = errors.New("the dependency would form a cycle")
	ErrNotDependency              = errors.New("does not depend on")
	ErrExpectedNumbers            = errors.New("expected a ticket reference and at least one criterion number")
	ErrInvalidNumber              = errors.New("is not a criterion number; expected a whole number from 1")
	ErrExpectedSubcommand         = errors.New("expected ac ls, ac set, ac add or ac hand")
	ErrExpectedHand               = errors.New("expected ac hand, which takes no reference")
	ErrExpectedLsCriteria         = errors.New("expected ac ls REF")
	ErrOpenNeedsLs                = errors.New("--open goes only with ac ls")
	ErrExpectedTicketReferences   = errors.New("expected at least one ticket reference")
	ErrExpectedNumbersOrAll       = errors.New("expected criterion numbers or --all, not both")
	ErrCreated                    = errors.New("was created")
	ErrNoFilesSection             = errors.New("lists no files; the backend's tickets have no files section")
	ErrExpectedSetCriterion       = errors.New("expected ac set REF N TEXT")
	ErrExpectedAddCriterion       = errors.New("expected ac add REF TEXT")
	ErrExpectedDocsLs             = errors.New("expected docs ls")
	ErrExpectedDocSubcommand      = errors.New("expected doc show or doc add")
	ErrExpectedDocShow            = errors.New("expected doc show REF")
	ErrExpectedDocAdd             = errors.New("expected doc add --type T TITLE")
	ErrExpectedOldAndNew          = errors.New("expected supersede OLD NEW")
	ErrSupersedesItself           = errors.New("can't supersede itself")
	ErrTypesDiffer                = errors.New("a doc is superseded only by one of its own type")
	ErrInvalidDepth               = errors.New("is not a depth; expected a whole number of hops from 0")
	ErrSetupEnded                 = errors.New("input ended before setup finished")
	ErrPrefixMismatch             = errors.New("is not a prefix; expected one matching")
	ErrDocsDirectoryNotLocal      = errors.New("is not a directory inside this one")
	ErrDocsNeedsVault             = errors.New("--docs sets the vault's directory, so it needs --backend vault")
	ErrExpectedReferencesOrAll    = errors.New("expected note references or --all, not both")
	ErrNoLint                     = errors.New("the backend has no notes to lint")
	ErrLintFailed                 = errors.New("lint failed")
	ErrExpectedReferences         = errors.New("expected at least one note reference")
	ErrExpectedReferencesOrStaged = errors.New("expected note references or --staged, not both")
	ErrInvalidStaged              = errors.New("unexpected git diff --name-status output")
)
