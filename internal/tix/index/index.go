// Package index provides tix search's vector index: the notes' title and summary embedded through an Embedder, kept
// in one JSON file per repository under the state directory, updated for the notes that changed, and ranked against a
// query by brute-force cosine similarity.
package index

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"
)

// Index is the embedded notes of one repository, built with one model. Entries are sorted by id.
type Index struct {
	SchemaVersion int     `json:"schema_version"`
	Model         string  `json:"model"`
	Entries       []Entry `json:"entries"`
}

// Entry is one note's text as it was embedded, and its vector.
type Entry struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Summary string    `json:"summary"`
	Vector  []float32 `json:"vector"`
}

// Note is a note to index: its id, type, title and summary.
type Note struct {
	ID      string
	Type    string
	Title   string
	Summary string
}

// Text is what gets embedded for a note: its title, then its summary on the next line when it has one.
func (note Note) Text() string {
	if strings.TrimSpace(note.Summary) == "" {
		return note.Title
	}
	return note.Title + textSeparator + note.Summary
}

func (entry Entry) note() Note {
	return Note{ID: entry.ID, Type: entry.Type, Title: entry.Title, Summary: entry.Summary}
}

// Path is the index file for the repository, the directory holding .tix, under the state directory.
func Path(stateDirectory, repository string) string {
	return filepath.Join(stateDirectory, cli.StateApplication, Directory, cli.RepositoryKey(repository)+FileExtension)
}

// Read loads the index at path. A missing file is ErrNoIndex.
func Read(path string) (Index, error) {
	text, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Index{}, fmt.Errorf("%w (%s)", ErrNoIndex, path)
	}
	if err != nil {
		return Index{}, err
	}
	var loaded Index
	if err := strictjson.Unmarshal(text, &loaded); err != nil {
		return Index{}, fmt.Errorf("%s: %w", path, err)
	}
	if loaded.SchemaVersion != SchemaVersion {
		return Index{}, fmt.Errorf("%s: %w %d", path, ErrSchemaVersion, loaded.SchemaVersion)
	}
	return loaded, nil
}

// CheckModel fails with ErrModelMismatch when the index was built with a model other than model.
func (value Index) CheckModel(model string) error {
	if value.Model != model {
		return fmt.Errorf("%w (index: %s, config: %s)", ErrModelMismatch, value.Model, model)
	}
	return nil
}

// Write stores the index at path atomically, so a failure leaves the previous file as it was.
func Write(path string, value Index) error {
	if err := os.MkdirAll(filepath.Dir(path), DirectoryPermission); err != nil {
		return err
	}
	temporary := path + temporaryFile
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FilePermission)
	if err != nil {
		return err
	}
	if err := cli.WriteJSON(file, value); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// Update returns the index for notes, built with embedder and its model: the entries of previous whose title and
// summary are unchanged are kept, every other note is embedded again, and a note no longer in notes drops out. An
// index built with another model is rebuilt whole. It returns how many notes were embedded. When embedding fails the
// previous index is returned unchanged with the error.
func Update(previous Index, notes []Note, embedder Embedder, model string) (Index, int, error) {
	kept := map[string]Entry{}
	if previous.Model == model {
		for _, entry := range previous.Entries {
			kept[entry.ID] = entry
		}
	}
	var changed []Note
	for _, note := range notes {
		if entry, exists := kept[note.ID]; !exists || entry.note() != note {
			changed = append(changed, note)
		}
	}
	vectors, err := embedNotes(embedder, changed)
	if err != nil {
		return previous, 0, err
	}
	updated := Index{SchemaVersion: SchemaVersion, Model: model, Entries: []Entry{}}
	for _, note := range notes {
		entry, exists := kept[note.ID]
		if vector, isChanged := vectors[note.ID]; isChanged {
			entry = Entry{ID: note.ID, Type: note.Type, Title: note.Title, Summary: note.Summary, Vector: vector}
		} else if !exists {
			continue
		}
		updated.Entries = append(updated.Entries, entry)
	}
	slices.SortFunc(updated.Entries, func(left, right Entry) int { return strings.Compare(left.ID, right.ID) })
	return updated, len(changed), nil
}

// embedNotes embeds the notes' texts in batches and returns each note's vector by id.
func embedNotes(embedder Embedder, notes []Note) (map[string][]float32, error) {
	vectors := map[string][]float32{}
	for start := 0; start < len(notes); start += embedBatchSize {
		batch := notes[start:min(start+embedBatchSize, len(notes))]
		texts := make([]string, len(batch))
		for position, note := range batch {
			texts[position] = note.Text()
		}
		embedded, err := embedder.Embed(texts)
		if err != nil {
			return nil, err
		}
		if len(embedded) != len(batch) {
			return nil, fmt.Errorf("%w: %d for %d texts", ErrEmbeddingCount, len(embedded), len(batch))
		}
		for position, note := range batch {
			vectors[note.ID] = embedded[position]
		}
	}
	return vectors, nil
}
