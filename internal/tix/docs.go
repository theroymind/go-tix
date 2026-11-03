package tix

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/theroymind/railyard/internal/cli"
	"github.com/theroymind/railyard/internal/tix/backend"
)

var docsSpecification = cli.Specification{
	Options: []cli.Option{
		{Name: optionKind, Type: cli.OptionTypeString},
		{Name: optionStatus, Type: cli.OptionTypeString},
		{Name: optionJSON, Type: cli.OptionTypeBoolean},
	},
	AllowsPositionals: true,
}

var docSpecification = cli.Specification{
	Options:           []cli.Option{{Name: optionJSON, Type: cli.OptionTypeBoolean}},
	AllowsPositionals: true,
}

// runDocs dispatches on its first positional, since cli.Program has no nested commands: docs ls lists the docs.
func runDocs(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, docsSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedDocsLs)
	}
	if len(parsed.Positionals) != 1 || parsed.Positionals[0] != subcommandLs {
		return expectedDocSubcommand(ErrExpectedDocsLs, parsed.Positionals[0])
	}
	store, err := openDocStore()
	if err != nil {
		return err
	}
	kind, _ := parsed.String(optionKind)
	docs, err := store.ListDocs(kind)
	if err != nil {
		return err
	}
	if status, exists := parsed.String(optionStatus); exists {
		docs = withStatus(docs, status)
	}
	return writeDocs(stdout, docs, parsed.IsEnabled(optionJSON))
}

// runDoc dispatches on its first positional: doc show REF prints the doc's text, or the doc as JSON.
func runDoc(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, docSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 || (parsed.Positionals[0] == subcommandShow && len(parsed.Positionals) != 2) {
		return cli.NewUsageError(ErrExpectedDocShow)
	}
	if parsed.Positionals[0] != subcommandShow {
		return expectedDocSubcommand(ErrExpectedDocShow, parsed.Positionals[0])
	}
	store, err := openDocStore()
	if err != nil {
		return err
	}
	id, err := store.ResolveDoc(parsed.Positionals[1])
	if err != nil {
		return err
	}
	if parsed.IsEnabled(optionJSON) {
		shown, err := store.ReadDoc(id)
		if err != nil {
			return err
		}
		return cli.WriteJSON(stdout, shown)
	}
	text, err := store.ReadDocText(id)
	if err != nil {
		return err
	}
	_, err = stdout.Write(text)
	return err
}

// openDocStore opens the backend and fails with backend.ErrNoDocs when it holds no docs.
func openDocStore() (backend.DocStore, error) {
	store, err := openBackend()
	if err != nil {
		return nil, err
	}
	docStore, hasDocs := store.(backend.DocStore)
	if !hasDocs {
		return nil, backend.ErrNoDocs
	}
	return docStore, nil
}

// resolveAny returns the ticket or doc that reference names, and the function that applies changes to it. A
// reference that names no ticket is tried as a doc when the backend has docs.
func resolveAny(store backend.Backend, reference string) (string, func(id string, changes []backend.Change) error, error) {
	id, err := store.Resolve(reference)
	docStore, hasDocs := store.(backend.DocStore)
	if err == nil || !hasDocs || !errors.Is(err, backend.ErrNoMatch) {
		return id, store.Apply, err
	}
	docID, docErr := docStore.ResolveDoc(reference)
	if errors.Is(docErr, backend.ErrNoMatch) {
		return "", nil, err
	}
	return docID, docStore.ApplyDoc, docErr
}

func withStatus(docs []backend.Doc, status string) []backend.Doc {
	matching := []backend.Doc{}
	for _, doc := range docs {
		if doc.Status == status {
			matching = append(matching, doc)
		}
	}
	return matching
}

// writeDocs prints the docs as a JSON array when asked, and as an ID KIND STATUS TITLE table otherwise.
func writeDocs(stdout io.Writer, docs []backend.Doc, isJSON bool) error {
	if isJSON {
		return cli.WriteJSON(stdout, docs)
	}
	table := tabwriter.NewWriter(stdout, tableMinimumWidth, tableTabWidth, tablePadding, tablePaddingCharacter, 0)
	if _, err := fmt.Fprintln(table, docTableHeader); err != nil {
		return err
	}
	for _, row := range docs {
		status := row.Status
		if status == "" {
			status = tableAbsent
		}
		if _, err := fmt.Fprintf(table, docTableRow, row.ID, row.Kind, status, row.Title); err != nil {
			return err
		}
	}
	return table.Flush()
}

func expectedDocSubcommand(expected error, got string) error {
	return cli.NewUsageError(fmt.Errorf("%w, got %s", expected, strconv.Quote(got)))
}
