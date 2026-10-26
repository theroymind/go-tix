package store

import "io/fs"

const (
	DirectoryName       = ".tix"
	ConfigFileName      = "config.yml"
	TicketFileExtension = ".yml"
	temporaryFileSuffix = ".tmp"
	yamlIndent          = 2
)

const (
	DirectoryPermission fs.FileMode = 0o777
	filePermission      fs.FileMode = 0o666
)
