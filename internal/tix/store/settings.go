package store

import "io/fs"

const (
	temporaryFileSuffix = ".tmp"
	yamlIndent          = 2
)

const (
	DirectoryPermission fs.FileMode = 0o777
	filePermission      fs.FileMode = 0o666
)
