package grammar

import "github.com/theroymind/go-tix/contracts/sketch"

const (
	VerbTask   = "task"
	VerbFile   = "file"
	VerbRename = "mv"
	VerbRemove = "rm"
	VerbSet    = "set"
)

const (
	taskSeparator  = "/"
	itemSeparator  = ":"
	parentSegment  = ".."
	currentSegment = "."
	backslash      = `\`
	firstLine      = 1
)

const (
	levelFeature = "feature"
	levelTask    = "task"
	levelFile    = "file"
	levelItem    = "item"
)

// aliases are the one-letter spellings of the verbs typed most. A verb with none is typed in full.
var aliases = map[string]string{
	"f": string(sketch.ItemTypeFn),
	"t": string(sketch.ItemTypeType),
	"i": string(sketch.LineTypeIn),
	"o": string(sketch.LineTypeOut),
	"c": string(sketch.LineTypeCase),
	"a": string(sketch.LineTypeAsk),
	"r": string(sketch.LineTypeRule),
}
