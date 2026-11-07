package command

import (
	"fmt"
	"io"

	cli "github.com/theroymind/railyard-cli"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/internal/sketch/plan"
	"github.com/theroymind/go-tix/internal/sketch/refs"
	"github.com/theroymind/go-tix/internal/sketch/repo"
)

// runPlan prints the build brief for the notes under the repo root, or with --feature only that feature's, as
// markdown. With a feature, in a tix repo, the brief starts with the feature's title, description and acceptance
// criteria from tix; when tix can't give them, it warns on stderr and goes on without. It then fails naming each
// note that didn't read.
func runPlan(arguments []string, stdout, stderr io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{{Name: optionFeature, Type: cli.OptionTypeString}},
	})
	if err != nil {
		return err
	}
	feature, _ := parsed.String(optionFeature)
	root, loaded, err := loadNotes(feature)
	if err != nil {
		return err
	}
	if feature != "" && len(loaded.Notes) == 0 {
		return fmt.Errorf("%w: %s", ErrNoNotes, feature)
	}
	tixFeature := lookUpFeature(refs.New(root), feature, stderr)
	if _, err := fmt.Fprint(stdout, plan.Markdown(feature, tixFeature, loaded.Notes)); err != nil {
		return err
	}
	return unreadable(loaded.Problems)
}

// runGraph prints the Mermaid flowchart of the notes under the repo root, or with --feature only that feature's,
// and then fails naming each note that didn't read. Mermaid is the only format, so --mermaid may be left out.
func runGraph(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, cli.Specification{
		Options: []cli.Option{
			{Name: optionFeature, Type: cli.OptionTypeString},
			{Name: optionMermaid, Type: cli.OptionTypeBoolean},
		},
	})
	if err != nil {
		return err
	}
	feature, _ := parsed.String(optionFeature)
	_, loaded, err := loadNotes(feature)
	if err != nil {
		return err
	}
	if feature != "" && len(loaded.Notes) == 0 {
		return fmt.Errorf("%w: %s", ErrNoNotes, feature)
	}
	if _, err := fmt.Fprint(stdout, plan.Mermaid(loaded.Notes)); err != nil {
		return err
	}
	return unreadable(loaded.Problems)
}

// loadNotes finds the repo root and loads its notes, keeping only feature's when feature isn't empty. What an
// empty result means is left to the caller.
func loadNotes(feature string) (string, repo.Repo, error) {
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return "", repo.Repo{}, err
	}
	loaded, err := repo.Load(root)
	if err != nil {
		return "", repo.Repo{}, err
	}
	if feature != "" {
		loaded.Notes = notesOf(loaded.Notes, feature)
	}
	return root, loaded, nil
}

// lookUpFeature returns the feature's ticket from tix, or nil when there is no feature or the repo isn't a tix
// repo. When tix can't give it, it warns on stderr and returns nil, so the plan still prints.
func lookUpFeature(resolver refs.Resolver, feature string, stderr io.Writer) *ticket.Ticket {
	if feature == "" || !resolver.IsTixRepo() {
		return nil
	}
	value, err := resolver.Ticket(feature)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, planWarningFormat, programName, err)
		return nil
	}
	return &value
}
