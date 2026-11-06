package store

import "io/fs"

const (
	// FileName is the name of a feature's sketch file inside its directory.
	FileName = "sketch.yml"
	// FeaturesDir is where feature directories live from the repo root, by default.
	FeaturesDir         = "docs/features"
	temporaryPattern    = ".sketch-*.yml"
	yamlIndent          = 2
	filePermission      = fs.FileMode(0o644)
	directoryPermission = fs.FileMode(0o755)
)
