package plan

import "strings"

const (
	rootFolderLabel = "repo root"
	danglingSuffix  = " (dangling)"
	planTitle       = "Plan"
	danglingClass   = "dangling"
	criteriaTitle   = "Acceptance criteria"
	criterionLine   = "%d. [%s] %s"
	decisionLine    = "- %s: %s"
	answerPrefix    = "  > "
	openLine        = "- %s:%d: %s"

	maxHeadingIndent = 3 // more indent makes a line code, not a heading or a fence
	maxHeadingLevel  = 6
	headingDemotion  = 3 // a note's # goes under its ### in the brief
)

// The titles of the brief's question sections.
const (
	DecisionsTitle = "Decisions"
	OpenTitle      = "Open questions"
)

var fenceMarkers = []string{"```", "~~~"}

// classDefs style new and changed notes and dangling links apart, in colours that read on light and dark themes.
const classDefs = `  classDef new fill:#d4f7dc,stroke:#2da44e,color:#000
  classDef changed fill:#fff3c4,stroke:#bf8700,color:#000
  classDef dangling fill:#ffe0e0,stroke:#cf222e,color:#000,stroke-dasharray:4 3
`

// labelEscapes turns the characters Mermaid reads as syntax inside a quoted label into its #code; entities.
var labelEscapes = strings.NewReplacer(
	`#`, "#35;",
	`"`, "#quot;",
	`|`, "#124;",
	`[`, "#91;",
	`]`, "#93;",
	`{`, "#123;",
	`}`, "#125;",
	`<`, "#lt;",
	`>`, "#gt;",
	"`", "#96;",
)
