package plan

import "strings"

const (
	rootFolderLabel = "repo root"
	danglingSuffix  = " (dangling)"
	planTitle       = "Plan"
	danglingClass   = "dangling"
)

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
