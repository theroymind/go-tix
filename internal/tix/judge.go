package tix

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/theroymind/go-tix/contracts/ticket"
	"github.com/theroymind/go-tix/contracts/verdict"
	"github.com/theroymind/go-tix/internal/tix/backend"
	"github.com/theroymind/go-tix/internal/tix/index"
	"github.com/theroymind/go-tix/internal/tix/judge"
	"github.com/theroymind/railyard-cli"
)

var judgeSpecification = cli.Specification{AllowsPositionals: true}

// searchFunction returns the ids of the notes closest to a query, best first.
type searchFunction func(query string) ([]string, error)

// judging is what judging drafts needs: the backend, its notes as tix context walks them, search when it is
// available, and the judge.
type judging struct {
	store  backend.Backend
	notes  []backend.ContextNote
	search searchFunction
	judge  judge.Judge
}

// runJudge asks the configured judge about each note the references name, beside its context and search neighbours,
// and prints its verdict. It fails when the judge refuses any of them.
func runJudge(arguments []string, _ io.Reader, stdout io.Writer) error {
	parsed, err := cli.Parse(arguments, judgeSpecification)
	if err != nil {
		return err
	}
	if len(parsed.Positionals) == 0 {
		return cli.NewUsageError(ErrExpectedReferences)
	}
	opened, err := openWorkspace()
	if err != nil {
		return err
	}
	var ids []string
	for _, reference := range parsed.Positionals {
		id, _, err := resolveEither(opened.store, reference)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	session, err := openJudging(opened, stdout)
	if err != nil {
		return err
	}
	return session.judgeNotes(ids, stdout)
}

// openJudging reads the notes and opens search and the configured judge. When search can't run it says why on
// stdout and leaves search out, so that drafts are judged against their context neighbours only.
func openJudging(opened workspace, stdout io.Writer) (judging, error) {
	notes, err := contextNotes(opened.store)
	if err != nil {
		return judging{}, err
	}
	search, err := openSearch(opened)
	if err != nil {
		if err := writeSearchUnavailable(stdout, err); err != nil {
			return judging{}, err
		}
	}
	configured := opened.config.JudgeOrDefault()
	claude := judge.Claude{Command: configured.Command, Model: configured.Model, Directory: opened.root}
	return judging{store: opened.store, notes: notes, search: search, judge: claude}, nil
}

// openSearch returns a search over the repository's index, or why there can be none: no embeddings in the config,
// no index, or an index built with another model.
func openSearch(opened workspace) (searchFunction, error) {
	if opened.config.Embeddings == nil {
		return nil, ErrNoEmbeddings
	}
	embeddings := *opened.config.Embeddings
	path, err := indexPath(opened.root)
	if err != nil {
		return nil, err
	}
	loaded, err := index.Read(path)
	if err != nil {
		return nil, err
	}
	if err := loaded.CheckModel(embeddings.Model); err != nil {
		return nil, err
	}
	embedder := index.OpenAIEmbedder{Endpoint: embeddings.Endpoint, Model: embeddings.Model}
	search := func(query string) ([]string, error) {
		vectors, err := embedder.Embed([]string{query})
		if err != nil {
			return nil, err
		}
		hits, err := index.Rank(loaded, vectors[0], nil, JudgeSearchTop)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, hit := range hits {
			ids = append(ids, hit.Note.ID)
		}
		return ids, nil
	}
	return search, nil
}

func writeSearchUnavailable(stdout io.Writer, cause error) error {
	_, err := fmt.Fprintf(stdout, searchUnavailableLine, cause)
	return err
}

// judgeNotes judges each draft in turn, prints each verdict, and fails with ErrJudgeRefused naming every draft the
// judge refused.
func (session judging) judgeNotes(ids []string, stdout io.Writer) error {
	var refused []string
	for _, id := range ids {
		value, draft, neighbours, err := session.judgeNote(id, stdout)
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if err := session.writeVerdict(stdout, draft, value, neighbours); err != nil {
			return err
		}
		if value.IsRefusal() {
			refused = append(refused, id)
		}
	}
	if len(refused) > 0 {
		return fmt.Errorf("%w: %s", ErrJudgeRefused, strings.Join(refused, judgeListSeparator))
	}
	return nil
}

// judgeNote asks the judge about one draft beside its neighbours, and returns a verdict that names only them.
func (session judging) judgeNote(id string, stdout io.Writer) (verdict.Verdict, judge.Note, []judge.Note, error) {
	draft, err := session.judgeNoteOf(id)
	if err != nil {
		return verdict.Verdict{}, judge.Note{}, nil, err
	}
	neighbourIDs, err := session.neighbourIDs(id, stdout)
	if err != nil {
		return verdict.Verdict{}, judge.Note{}, nil, err
	}
	neighbours := make([]judge.Note, 0, len(neighbourIDs))
	for _, neighbourID := range neighbourIDs {
		neighbour, err := session.judgeNoteOf(neighbourID)
		if err != nil {
			return verdict.Verdict{}, judge.Note{}, nil, err
		}
		neighbours = append(neighbours, neighbour)
	}
	value, err := session.judge.Judge(judge.Prompt(draft, neighbours))
	if err != nil {
		return verdict.Verdict{}, judge.Note{}, nil, err
	}
	if err := judge.CheckNames(value, id, neighbours); err != nil {
		return verdict.Verdict{}, judge.Note{}, nil, err
	}
	return value, draft, neighbours, nil
}

// neighbourIDs returns the notes within JudgeContextDepth hops of the draft, then the notes search ranks closest to
// its title and summary, once each and never the draft, at most JudgeNeighbourLimit. When search fails it says why
// on stdout and gives the context neighbours alone.
func (session judging) neighbourIDs(id string, stdout io.Writer) ([]string, error) {
	cards, err := walkCards(session.notes, id, JudgeContextDepth)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, reached := range cards {
		ids = appendNeighbour(ids, reached.ID, id)
	}
	if session.search != nil {
		found, err := session.search(session.searchQuery(id))
		if err != nil {
			if err := writeSearchUnavailable(stdout, err); err != nil {
				return nil, err
			}
		}
		for _, foundID := range found {
			ids = appendNeighbour(ids, foundID, id)
		}
	}
	return ids[:min(len(ids), JudgeNeighbourLimit)], nil
}

func appendNeighbour(ids []string, candidate, draft string) []string {
	if candidate == draft || slices.Contains(ids, candidate) {
		return ids
	}
	return append(ids, candidate)
}

// searchQuery is the draft's title and summary, as the index embeds a note.
func (session judging) searchQuery(id string) string {
	for _, note := range session.notes {
		if note.ID == id {
			return index.Note{ID: note.ID, Type: note.Type, Title: note.Title, Summary: note.Summary}.Text()
		}
	}
	return id
}

// judgeNoteOf reads a ticket or doc as the judge sees it: its type, status and source text.
func (session judging) judgeNoteOf(id string) (judge.Note, error) {
	noteType := ""
	for _, note := range session.notes {
		if note.ID == id {
			noteType = note.Type
		}
	}
	docStore, hasDocs := session.store.(backend.DocStore)
	if hasDocs {
		doc, err := docStore.ReadDoc(id)
		if err == nil {
			text, err := docStore.ReadDocText(id)
			return judge.Note{ID: id, Type: noteType, Status: doc.Status, Text: string(text)}, err
		}
		if !errors.Is(err, backend.ErrNoMatch) {
			return judge.Note{}, err
		}
	}
	found, err := session.store.Read(id)
	if err != nil {
		return judge.Note{}, err
	}
	text, err := ticketText(session.store, found)
	return judge.Note{ID: id, Type: noteType, Status: string(found.Status), Text: text}, err
}

// ticketText is the ticket's source text, or its description when the backend keeps no source text.
func ticketText(store backend.Backend, found ticket.Ticket) (string, error) {
	textReader, hasText := store.(backend.TextReader)
	if !hasText {
		return found.Title + judgeTextSeparator + found.Description, nil
	}
	text, err := textReader.ReadText(found.ID)
	return string(text), err
}

// writeVerdict prints the verdict on one line, naming the note a refusal duplicates or conflicts with, its type and
// status. A conflict with a doc of the draft's own type that still holds points at tix supersede.
func (session judging) writeVerdict(stdout io.Writer, draft judge.Note, value verdict.Verdict, neighbours []judge.Note) error {
	if !value.IsRefusal() {
		_, err := fmt.Fprintf(stdout, acceptLine, draft.ID, value.Reason)
		return err
	}
	var grounds []string
	if value.DuplicateOf != "" {
		grounds = append(grounds, fmt.Sprintf(duplicateGround, describeNote(value.DuplicateOf, neighbours)))
	}
	if value.ConflictsWith != "" {
		grounds = append(grounds, fmt.Sprintf(conflictGround, describeNote(value.ConflictsWith, neighbours)))
	}
	if _, err := fmt.Fprintf(stdout, refuseLine, draft.ID, strings.Join(grounds, judgeGroundSeparator), value.Reason); err != nil {
		return err
	}
	if value.ConflictsWith == "" || !session.canSupersede(draft, neighbourNamed(value.ConflictsWith, neighbours)) {
		return nil
	}
	_, err := fmt.Fprintf(stdout, supersedeHintLine, draft.ID, value.ConflictsWith, value.ConflictsWith, draft.ID)
	return err
}

// canSupersede reports whether tix supersede could mark the older doc superseded by the draft: both are docs of
// one type and the older one isn't superseded already.
func (session judging) canSupersede(draft, older judge.Note) bool {
	docStore, hasDocs := session.store.(backend.DocStore)
	if !hasDocs || draft.Type != older.Type || older.Status == backend.StatusSuperseded {
		return false
	}
	_, draftErr := docStore.ReadDoc(draft.ID)
	_, olderErr := docStore.ReadDoc(older.ID)
	return draftErr == nil && olderErr == nil
}

func neighbourNamed(id string, neighbours []judge.Note) judge.Note {
	for _, neighbour := range neighbours {
		if neighbour.ID == id {
			return neighbour
		}
	}
	return judge.Note{ID: id}
}

func describeNote(id string, neighbours []judge.Note) string {
	named := neighbourNamed(id, neighbours)
	if named.Status == "" {
		return fmt.Sprintf(noteDescription, id, named.Type)
	}
	return fmt.Sprintf(noteDescriptionWithStatus, id, named.Type, named.Status)
}
