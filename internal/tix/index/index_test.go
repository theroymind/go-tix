package index_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/theroymind/railyard/internal/tix/index"
)

const model = "fake-model"

// countingEmbedder returns a vector per text that encodes its length, and remembers every text it was asked for.
type countingEmbedder struct {
	texts []string
	err   error
}

func (embedder *countingEmbedder) Embed(texts []string) ([][]float32, error) {
	if embedder.err != nil {
		return nil, embedder.err
	}
	embedder.texts = append(embedder.texts, texts...)
	vectors := make([][]float32, len(texts))
	for position, text := range texts {
		vectors[position] = []float32{float32(len(text)), 1}
	}
	return vectors, nil
}

var (
	noteA = index.Note{ID: "a", Type: "task", Title: "A", Summary: "First."}
	noteB = index.Note{ID: "b", Type: "task", Title: "B", Summary: "Second."}
	noteC = index.Note{ID: "c", Type: "idea", Title: "C"}
)

func built(t *testing.T, notes ...index.Note) index.Index {
	t.Helper()
	value, _, err := index.Update(index.Index{}, notes, &countingEmbedder{}, model)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestUpdateEmbedsOnlyWhatChanged(t *testing.T) {
	changedB := index.Note{ID: "b", Type: "task", Title: "B", Summary: "Second, reworded."}
	retitledC := index.Note{ID: "c", Type: "idea", Title: "C again"}
	tests := []struct {
		name          string
		previous      index.Index
		notes         []index.Note
		previousModel string
		expectedTexts []string
		expectedIDs   []string
	}{
		{"An empty index embeds every note", index.Index{}, []index.Note{noteA, noteB, noteC}, model,
			[]string{"A\nFirst.", "B\nSecond.", "C"}, []string{"a", "b", "c"}},
		{"An unchanged note is not embedded again", built(t, noteA, noteB, noteC), []index.Note{noteA, noteB, noteC}, model,
			nil, []string{"a", "b", "c"}},
		{"A changed summary is embedded again", built(t, noteA, noteB, noteC), []index.Note{noteA, changedB, noteC}, model,
			[]string{"B\nSecond, reworded."}, []string{"a", "b", "c"}},
		{"A changed title is embedded again", built(t, noteA, noteB, noteC), []index.Note{noteA, noteB, retitledC}, model,
			[]string{"C again"}, []string{"a", "b", "c"}},
		{"A deleted note drops out", built(t, noteA, noteB, noteC), []index.Note{noteA, noteC}, model,
			nil, []string{"a", "c"}},
		{"A new note is embedded and sorted in", built(t, noteA, noteC), []index.Note{noteC, noteB, noteA}, model,
			[]string{"B\nSecond."}, []string{"a", "b", "c"}},
		{"Another model rebuilds the whole index", built(t, noteA, noteB, noteC), []index.Note{noteA, noteB, noteC}, "other-model",
			[]string{"A\nFirst.", "B\nSecond.", "C"}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := &countingEmbedder{}

			updated, embedded, err := index.Update(tt.previous, tt.notes, embedder, tt.previousModel)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(embedder.texts, tt.expectedTexts) {
				t.Errorf("embedded %q, expected %q", embedder.texts, tt.expectedTexts)
			}
			if embedded != len(tt.expectedTexts) {
				t.Errorf("reported %d embedded, expected %d", embedded, len(tt.expectedTexts))
			}
			var ids []string
			for _, entry := range updated.Entries {
				ids = append(ids, entry.ID)
			}
			if !reflect.DeepEqual(ids, tt.expectedIDs) {
				t.Errorf("entries %v, expected %v", ids, tt.expectedIDs)
			}
			if updated.Model != tt.previousModel || updated.SchemaVersion != index.SchemaVersion {
				t.Errorf("model %q schema %d, expected %q and %d", updated.Model, updated.SchemaVersion, tt.previousModel, index.SchemaVersion)
			}
		})
	}
}

func TestUpdateKeepsThePreviousIndexWhenEmbeddingFails(t *testing.T) {
	previous := built(t, noteA, noteB)
	failure := errors.New("down")

	updated, embedded, err := index.Update(previous, []index.Note{noteA, noteC}, &countingEmbedder{err: failure}, model)

	if !errors.Is(err, failure) || embedded != 0 || !reflect.DeepEqual(updated, previous) {
		t.Errorf("got %+v, %d, %v; expected the previous index, 0 and the failure", updated, embedded, err)
	}
}

func TestUpdateKeepsTheVectorsOfUnchangedNotes(t *testing.T) {
	previous := built(t, noteA)
	previous.Entries[0].Vector = []float32{42, 42}

	updated, _, err := index.Update(previous, []index.Note{noteA, noteB}, &countingEmbedder{}, model)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(updated.Entries[0].Vector, []float32{42, 42}) {
		t.Errorf("a's vector = %v, expected the previous one", updated.Entries[0].Vector)
	}
}

func TestCheckModel(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		expectedErr error
	}{
		{"The same model passes", model, nil},
		{"Another model is a mismatch", "other", index.ErrModelMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := built(t, noteA).CheckModel(tt.model)

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, expected %v", err, tt.expectedErr)
			}
			if err != nil && !strings.Contains(err.Error(), "index: "+model) {
				t.Errorf("error %q doesn't name the index's model", err)
			}
		})
	}
}

func TestReadAndWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "railyard", "index", "repo.json")
	value := built(t, noteA, noteB, noteC)

	if err := index.Write(path, value); err != nil {
		t.Fatal(err)
	}
	loaded, err := index.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(loaded, value) {
		t.Errorf("loaded %+v, expected %+v", loaded, value)
	}
}

func TestReadReportsAMissingIndex(t *testing.T) {
	_, err := index.Read(filepath.Join(t.TempDir(), "missing.json"))

	if !errors.Is(err, index.ErrNoIndex) {
		t.Errorf("error = %v, expected %v", err, index.ErrNoIndex)
	}
}

func TestReadRejectsAnotherSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(path, []byte(`{"schema_version": 99, "model": "m", "entries": []}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := index.Read(path)

	if !errors.Is(err, index.ErrSchemaVersion) {
		t.Errorf("error = %v, expected %v", err, index.ErrSchemaVersion)
	}
}

func TestPathKeepsRepositoriesApart(t *testing.T) {
	first := index.Path("/state", "/work/repo")
	second := index.Path("/state", "/personal/repo")

	if first == second {
		t.Errorf("both repositories map to %s", first)
	}
	if !strings.HasPrefix(first, filepath.Join("/state", "railyard", "index")+string(filepath.Separator)) || !strings.HasSuffix(first, index.FileExtension) {
		t.Errorf("path = %s, expected one under /state/railyard/index ending in %s", first, index.FileExtension)
	}
}
