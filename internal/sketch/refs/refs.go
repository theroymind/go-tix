// Package refs looks features up in tix: whether the repo is a tix repo, which tickets it holds, and a feature's
// tasks, found by running tix ls.
package refs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cli "github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"

	"github.com/theroymind/go-tix/contracts/ticket"
)

const (
	tixDirectory = ".tix"
	tixCommand   = "tix"
	listCommand  = "ls"
	parentOption = "--parent"
	allOption    = "--all"
	jsonOption   = "--json"
)

var (
	ErrTixNotFound = errors.New("this is a tix repo, but tix isn't on PATH; install it with go install ./cmd/tix")
	ErrTixOutput   = errors.New("tix ls printed tickets sketch can't read")
)

// Runner runs a child command in a directory and returns its stdout, as cli.RunChildIn does.
type Runner func(directory, name string, arguments []string, stdin io.Reader) ([]byte, error)

// Resolver looks features up in tix from a repo root, running tix through Run.
type Resolver struct {
	Root string
	Run  Runner
}

// New returns a Resolver for root that runs the tix on PATH.
func New(root string) Resolver {
	return Resolver{Root: root, Run: RunQuiet}
}

// RunQuiet is a Runner that runs a child command as cli.RunChildIn does, but keeps its stderr out of the
// terminal: the error it returns ends with the child's last stderr line, so sketch's own error already says what
// tix said.
func RunQuiet(directory, name string, arguments []string, stdin io.Reader) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	child := exec.Command(name, arguments...)
	child.Dir, child.Stdin, child.Stdout, child.Stderr = directory, stdin, &stdout, &stderr
	if err := child.Run(); err != nil {
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		if cause := strings.TrimSpace(lines[len(lines)-1]); cause != "" {
			return stdout.Bytes(), fmt.Errorf("%w: %s: %w: %s", cli.ErrChildFailed, name, err, cause)
		}
		return stdout.Bytes(), fmt.Errorf("%w: %s: %w", cli.ErrChildFailed, name, err)
	}
	return stdout.Bytes(), nil
}

// Tasks returns the ids of the feature's child tickets, done ones included, in tix's order.
// Outside a tix repo, a feature has no tasks to offer, and Tasks returns none.
func (resolver Resolver) Tasks(feature string) ([]string, error) {
	if !resolver.IsTixRepo() {
		return nil, nil
	}
	return resolver.list(parentOption, feature)
}

// Tickets returns the set of every ticket id in the repo, done ones included, from one run of tix ls. Call it
// only in a tix repo.
func (resolver Resolver) Tickets() (map[string]bool, error) {
	ids, err := resolver.list()
	if err != nil {
		return nil, err
	}
	tickets := make(map[string]bool, len(ids))
	for _, id := range ids {
		tickets[id] = true
	}
	return tickets, nil
}

// list runs tix ls --all --json with filters in front, and returns the ids of the tickets it prints, in order.
func (resolver Resolver) list(filters ...string) ([]string, error) {
	arguments := append([]string{listCommand}, filters...)
	arguments = append(arguments, allOption, jsonOption)
	output, err := resolver.Run(resolver.Root, tixCommand, arguments, nil)
	if errors.Is(err, exec.ErrNotFound) {
		return nil, fmt.Errorf("%w: %w", ErrTixNotFound, err)
	}
	if err != nil {
		return nil, err
	}
	var tickets []ticket.Ticket
	if err := strictjson.Unmarshal(output, &tickets); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTixOutput, err)
	}
	ids := make([]string, 0, len(tickets))
	for index, value := range tickets {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("%w: ticket %d: %w", ErrTixOutput, index, err)
		}
		ids = append(ids, value.ID)
	}
	return ids, nil
}

// IsTixRepo reports whether the root holds a .tix directory.
func (resolver Resolver) IsTixRepo() bool {
	info, err := os.Stat(filepath.Join(resolver.Root, tixDirectory))
	return err == nil && info.IsDir()
}
