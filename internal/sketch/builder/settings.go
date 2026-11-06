package builder

const (
	commandUp    = ".."
	commandTop   = "/"
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
var sessionCommands = []string{commandUp, commandTop, commandUndo, commandEdit, commandWhere, commandExit}
