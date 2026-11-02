package backend

import "slices"

// Kind names a backend in .tix/config.yml.
type Kind string

func (kind Kind) IsKnown() bool {
	return slices.Contains(Kinds, kind)
}

// Opener opens a backend of one kind on the .tix directory and its config.
type Opener func(tixDirectory string, config Config) (Backend, error)

// openers holds each kind's Opener. Backend implementations import this package for its types,
// so they can't be imported by it; the command's main registers them instead.
var openers = map[Kind]Opener{}

// Register makes Open use opener for kind. It panics on an unknown kind, which is a programming error.
func Register(kind Kind, opener Opener) {
	if !kind.IsKnown() {
		panic("backend: Register of unknown kind " + string(kind))
	}
	openers[kind] = opener
}

// Open finds the .tix directory at or above workingDirectory and opens the backend its config names.
func Open(workingDirectory string) (Backend, error) {
	directory, err := FindDirectory(workingDirectory)
	if err != nil {
		return nil, err
	}
	config, err := ReadConfig(directory)
	if err != nil {
		return nil, err
	}
	kind := config.BackendKind()
	opener, exists := openers[kind]
	if !exists {
		return nil, notImplemented(kind)
	}
	return opener(directory, config)
}
