// Package rankeval provides the rankeval command, which scores ranking models on a vault's done tasks. A card
// is relevant to a task when its commits, as of the commit before the task's first, touched a file the task's
// own commits touched, and each model is measured by its precision at the pack's card count.
package rankeval

import (
	"errors"

	"github.com/theroymind/railyard/internal/cli"
)

const usage = `usage: rankeval <command> [arguments]

commands:
  run --vault PATH [--model NAME]... [--endpoint URL] [--depth N] [--top K] [--runs N] [--notes DIR] [--thinking]
      NAME is none for the structural order, haiku for claude -p, or a model the endpoint serves.
      Defaults: --model none, --endpoint http://localhost:8080, --depth 2, --top 5, --runs 1, --notes docs.
      --thinking lets a thinking model think; without it the request asks the endpoint to turn thinking off.
`

var Program = cli.Program{
	Name:  "rankeval",
	Usage: usage,
	Commands: map[string]cli.Command{
		"run": runRun,
	},
}

const (
	optionVault    = "vault"
	optionModel    = "model"
	optionEndpoint = "endpoint"
	optionDepth    = "depth"
	optionTop      = "top"
	optionRuns     = "runs"
	optionNotes    = "notes"
	optionThinking = "thinking"
)

const (
	DefaultEndpoint       = "http://localhost:8080"
	DefaultDepth          = 2
	DefaultTop            = 5
	DefaultRuns           = 1
	DefaultNotesDirectory = "docs"
	ModelNone             = "none"
	MinimumScore          = 0
	MaximumScore          = 3
)

const (
	tixExecutable          = "tix"
	gitExecutable          = "git"
	claudeExecutable       = "claude"
	claudeResultType       = "result"
	chatCompletionsPath    = "/v1/chat/completions"
	chatRoleUser           = "user"
	chatTemperature        = 0
	chatMaximumTokens      = 4096
	millisecondsPerSecond  = 1000
	jsonObjectStart        = "{"
	jsonObjectEnd          = "}"
	skipReasonNoCommits    = "the task has no commits"
	skipReasonRootCommit   = "the task's first commit has no parent"
	skipReasonNoCodeFiles  = "the task's commits touched no code files"
	skipReasonNoCandidates = "no card has commits before the cut"
	skipReasonNoRelevant   = "no candidate card is relevant"
	relationRankStrong     = 0
	relationRankWikilink   = 1
	relationRankBacklink   = 2
	relationRankUnknown    = 3
	promptSectionSeparator = "\n\n"
	promptListItemPrefix   = "- "
	promptCardFieldJoiner  = " | "
	promptFilesJoiner      = ", "
	promptCriteriaHeading  = "Acceptance criteria:\n"
	promptCardsHeading     = "Cards:\n"
	promptTitleLabel       = "Task: "
	promptSummaryLabel     = "Summary: "
	promptCardIDLabel      = "id: "
	promptCardTypeLabel    = "type: "
	promptCardTitleLabel   = "title: "
	promptCardSummaryLabel = "summary: "
	promptCardFilesLabel   = "files: "
	promptMissingSummary   = "(no summary)"
	promptInstruction      = "You rank notes for a software task. Each card below is one note from the project's vault " +
		"and the files it expects to touch. Score each card for how likely the task's code change touches the code " +
		"behind that note: 3 when it certainly does, 2 when it probably does, 1 when it might, 0 when it is unrelated."
	promptAnswerInstruction = "Answer with JSON only, no prose and no code fence, in the shape " +
		`{"scores":[{"id":"<card id>","score":<0 to 3>}]}` + ", with every card id exactly once."
)

var claudeModels = []string{"haiku", "sonnet", "opus"}

var claudeArguments = []string{
	"--print",
	"--output-format=json",
	"--strict-mcp-config",
	"--tools=",
}

var (
	ErrInvalidNumber     = errors.New("is not a whole number; expected at least")
	ErrNoCommits         = errors.New("no commits to order")
	ErrNoParent          = errors.New("commit has no parent")
	ErrInvalidAnswer     = errors.New("invalid ranking answer")
	ErrUnknownCard       = errors.New("scores a card that was not asked about")
	ErrDuplicateCard     = errors.New("scores a card twice")
	ErrMissingCard       = errors.New("leaves a card unscored")
	ErrEndpointFailed    = errors.New("endpoint request failed")
	ErrEmptyCompletion   = errors.New("the model returned no choices")
	ErrClaudeFailed      = errors.New("claude reported an error")
	ErrUnknownClaudeType = errors.New("unknown claude output type")
	ErrNoStartCard       = errors.New("the walk has no start card")
)
