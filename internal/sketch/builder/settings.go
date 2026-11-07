package builder

import "regexp"

const (
	commandUndo  = "undo"
	commandEdit  = "edit"
	commandExit  = "exit"
	commandQuit  = "quit"
	commandWhere = "where"
)

const (
	promptMarker   = "> "
	ellipsis       = "…"
	undoneMessage  = "undone"
	yamlIndent     = 2
	pathSeparator  = "/"
	hiddenPrefix   = "."
	defaultEditor  = "vi"
	editorVariable = "EDITOR"
	editPattern    = "sketch-item-*.yml"
)

// sessionCommands are the builder's own commands, which completion offers beside the grammar's verbs.
var sessionCommands = []string{commandUndo, commandEdit, commandWhere, commandExit}

// identifierPattern matches a name as most languages spell one, which completion offers from the context file.
var identifierPattern = regexp.MustCompile(`[\p{L}_][\p{L}\p{N}_]*`)
