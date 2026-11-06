// Package refs finds a feature's directory: beside the feature's tix note inside a tix repo, found by running
// tix path, and docs/features/<feature> from the repo root outside one.
package refs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/internal/sketch/store"
)

const (
	tixDirectory = ".tix"
	tixCommand   = "tix"
	pathCommand  = "path"
)

var (
	ErrNotATicket   = errors.New("is not a tix ticket")
	ErrTixNotFound  = errors.New("this is a tix repo, but tix isn't on PATH; install it with go install ./cmd/tix")
	ErrRelativePath = errors.New("tix path printed a path that isn't absolute")
)

// Runner runs a child command in a directory and returns its stdout, as cli.RunChildIn does.
type Runner func(directory, name string, arguments []string, stdin io.Reader) ([]byte, error)

// Resolver finds feature directories from a repo root, running tix through Run.
type Resolver struct {
	Root string
	Run  Runner
}

// New returns a Resolver for root that runs the tix on PATH.
func New(root string) Resolver {
	return Resolver{Root: root, Run: cli.RunChildIn}
}

// FeatureDir returns the directory a feature's sketch lives in. Inside a tix repo, one whose root holds .tix, the
// feature must be a ticket, and the directory is the one holding its note; a feature that isn't one fails with
// ErrNotATicket, naming it, and a missing tix fails with ErrTixNotFound. Outside one, any feature name works,
// and the directory is store.FeatureDir's.
func (resolver Resolver) FeatureDir(feature string) (string, error) {
	if !resolver.IsTixRepo() {
		return store.FeatureDir(resolver.Root, feature), nil
	}
	output, err := resolver.Run(resolver.Root, tixCommand, []string{pathCommand, feature}, nil)
	if errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("%w: %w", ErrTixNotFound, err)
	}
	if err != nil {
		return "", fmt.Errorf("feature %q %w: %w", feature, ErrNotATicket, err)
	}
	path := strings.TrimSpace(string(output))
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("feature %q: %w: %q", feature, ErrRelativePath, path)
	}
	return filepath.Dir(path), nil
}

// IsTixRepo reports whether the root holds a .tix directory.
func (resolver Resolver) IsTixRepo() bool {
	info, err := os.Stat(filepath.Join(resolver.Root, tixDirectory))
	return err == nil && info.IsDir()
}
