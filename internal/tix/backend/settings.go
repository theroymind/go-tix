package backend

import "io/fs"

const (
	configFieldPrefix     = "prefix"
	configFieldBackend    = "backend"
	problemPrefixMismatch = "prefix must match "
	temporaryFileSuffix   = ".tmp"
	yamlIndent            = 2
)

const (
	DirectoryPermission fs.FileMode = 0o777
	FilePermission      fs.FileMode = 0o666
)

const (
	KindYAML  Kind = "yaml"
	KindVault Kind = "vault"
)

// Kinds are the backends a config may name.
var Kinds = []Kind{KindYAML, KindVault}
