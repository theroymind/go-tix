package judge

const (
	draftHeading      = "\n# The draft\n\n"
	neighboursHeading = "\n# Its neighbours\n\n"
	noNeighbours      = "(none)\n"
	noteHeading       = "## %s (type %s, status %s)\n\n"
	noteEnd           = "\n\n"
	noStatus          = "none"
)

const instructions = `You review a draft note before it is committed to a project's design vault. The vault holds ` +
	`decisions, tasks, specs, ideas and phases, each one note named by its id. Read the draft beside the neighbours ` +
	`below, the notes linked to it or closest to it.

Refuse the draft when:
- it restates a neighbour's choice or work without adding a new one: set duplicate_of to that neighbour's id; or
- it contradicts a neighbour that still holds (any status but superseded), such as choosing differently from an ` +
	`accepted decision: set conflicts_with to that neighbour's id.

Otherwise accept it, with duplicate_of and conflicts_with empty. A task that implements a decision, a note that ` +
	`refines or extends a neighbour, and a decision that supersedes one marked as superseded by it are not duplicates ` +
	`or conflicts. Name only ids from the neighbours below, never the draft's own. Give the reason in one or two ` +
	`sentences that name what is restated or contradicted.
`

const (
	claudeResultType = "result"
	jsonNull         = "null"
)

// claudeArguments are claude -p's flags for a judge: one JSON result, no tools, no MCP servers, no saved session.
var claudeArguments = []string{
	"--print",
	"--output-format=json",
	"--strict-mcp-config",
	"--tools=",
	"--no-session-persistence",
}
