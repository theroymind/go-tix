package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/theroymind/go-tix/contracts/sketch"
	"github.com/theroymind/go-tix/contracts/validation"
)

const fixtureDir = "testdata/fight-camera"

// fixture is the sketch testdata/fight-camera/sketch.yml holds, in the order it was typed: not sorted by name.
func fixture() sketch.Sketch {
	value := sketch.New()
	value.Tasks = []sketch.Task{
		{Name: "score-strikes", Files: []sketch.File{
			{Path: "client/camera.odin", Items: []sketch.Item{
				{Type: sketch.ItemTypeFn, Name: "score_strike", Lines: []sketch.Line{
					{Type: sketch.LineTypeIn, Text: "actor Combatant_Id"},
					{Type: sketch.LineTypeOut, Text: "Camera_Move"},
					{Type: sketch.LineTypeAsk, Text: "who anchors a kill"},
				}},
				{Type: sketch.ItemTypeType, Name: "Camera_Move"},
			}},
			{Path: "client/arena.odin"},
		}},
		{Name: "frame-kills"},
	}
	return value
}

func TestSaveWritesTheTypedOrder(t *testing.T) {
	dir := t.TempDir()
	expected, err := os.ReadFile(Path(fixtureDir))
	if err != nil {
		t.Fatal(err)
	}

	if err := Save(dir, fixture()); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	actual, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Errorf("saved file =\n%s\nwant\n%s", actual, expected)
	}
	assertOnlySketch(t, dir)
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value sketch.Sketch
	}{
		{"A sketch with every level", fixture()},
		{"A sketch with no tasks", sketch.New()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := Save(dir, tt.value); err != nil {
				t.Fatalf("Save() = %v", err)
			}
			first, _ := os.ReadFile(Path(dir))

			loaded, err := Load(dir)
			if err != nil {
				t.Fatalf("Load() = %v", err)
			}
			if !reflect.DeepEqual(loaded, tt.value) {
				t.Errorf("Load() = %+v, want %+v", loaded, tt.value)
			}
			if err := Save(dir, loaded); err != nil {
				t.Fatalf("Save() again = %v", err)
			}
			second, _ := os.ReadFile(Path(dir))
			if string(second) != string(first) {
				t.Errorf("saving the loaded sketch changed the file:\n%s\nwas\n%s", second, first)
			}
		})
	}
}

func TestSaveTrims(t *testing.T) {
	dir := t.TempDir()
	padded := fixture()
	padded.Tasks[0].Name = " score-strikes "
	padded.Tasks[0].Files[0].Path = "client/camera.odin\t"
	padded.Tasks[0].Files[0].Items[0].Name = " score_strike"
	padded.Tasks[0].Files[0].Items[0].Lines[2].Text = "who anchors a kill  "

	if err := Save(dir, padded); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	loaded, err := Load(dir)
	if err != nil || !reflect.DeepEqual(loaded, fixture()) {
		t.Errorf("Load() = %+v, %v, want %+v", loaded, err, fixture())
	}
	if padded.Tasks[0].Name != " score-strikes " {
		t.Errorf("Save() changed the caller's sketch: task name %q", padded.Tasks[0].Name)
	}
}

func TestSaveCreatesTheFeatureDir(t *testing.T) {
	dir := FeatureDir(t.TempDir(), "fight-camera")

	if err := Save(dir, fixture()); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	if _, err := Load(dir); err != nil {
		t.Errorf("Load() = %v", err)
	}
}

func TestSaveReplacesAnEarlierSketch(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, fixture()); err != nil {
		t.Fatal(err)
	}
	replacement := sketch.New()

	if err := Save(dir, replacement); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	loaded, err := Load(dir)
	if err != nil || !reflect.DeepEqual(loaded, replacement) {
		t.Errorf("Load() = %+v, %v, want %+v", loaded, err, replacement)
	}
	assertOnlySketch(t, dir)
}

func TestSaveRefusesAnInvalidSketch(t *testing.T) {
	dir := t.TempDir()
	invalid := fixture()
	invalid.Tasks[0].Files[0].Items[1].Name = "score_strike"

	err := Save(dir, invalid)

	if !errors.Is(err, validation.ErrInvalid) {
		t.Errorf("Save() = %v, want an error wrapping %v", err, validation.ErrInvalid)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("Save() left %d entries, want none", len(entries))
	}
}

func TestLoadRejects(t *testing.T) {
	tests := []struct {
		name     string
		text     *string
		expected error
	}{
		{"No sketch file", nil, ErrNotFound},
		{"An empty file", new(""), ErrInvalidFile},
		{"An unknown key", new("schema_version: 1\nowner: eric\n"), ErrInvalidFile},
		{"An unknown nested key", new("schema_version: 1\ntasks:\n  - name: t\n    status: done\n"),
			ErrInvalidFile},
		{"An unknown schema version", new("schema_version: 2\n"), validation.ErrInvalid},
		{"A duplicate key", new("schema_version: 1\nschema_version: 1\n"), ErrInvalidFile},
		{"A second document", new("schema_version: 1\n---\nschema_version: 1\n"),
			ErrMultipleDocuments},
		{"An unknown item type", new("schema_version: 1\ntasks:\n  - name: t\n    files:\n" +
			"      - path: a.go\n        items:\n          - type: struct\n            name: A\n"), validation.ErrInvalid},
		{"Malformed YAML", new("schema_version: [1\n"), ErrInvalidFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.text != nil {
				if err := os.WriteFile(filepath.Join(dir, FileName), []byte(*tt.text), filePermission); err != nil {
					t.Fatal(err)
				}
			}

			_, err := Load(dir)

			if !errors.Is(err, tt.expected) {
				t.Errorf("Load() = %v, want an error wrapping %v", err, tt.expected)
			}
		})
	}
}

func TestFeatureDir(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		feature  string
		expected string
	}{
		{"From a repo root", "/repo", "sketch", filepath.FromSlash("/repo/docs/features/sketch")},
		{"From the working directory", ".", "fight-camera", filepath.FromSlash("docs/features/fight-camera")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := FeatureDir(tt.root, tt.feature); actual != tt.expected {
				t.Errorf("FeatureDir() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

// assertOnlySketch fails when dir holds anything but the sketch file, such as a temporary file left behind.
func assertOnlySketch(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != FileName {
		names := make([]string, len(entries))
		for index, entry := range entries {
			names[index] = entry.Name()
		}
		t.Errorf("dir holds %q, want only %q", names, FileName)
	}
}

var errEdit = errors.New("edit refused")

func TestUpdatesAtOnceAllLand(t *testing.T) {
	const updates = 20
	dir := t.TempDir()
	var group sync.WaitGroup
	errs := make([]error, updates)
	for index := range updates {
		group.Go(func() {
			errs[index] = Update(dir, dir, func(value sketch.Sketch) (sketch.Sketch, error) {
				value.Tasks = append(value.Tasks, sketch.Task{Name: fmt.Sprintf("task-%d", index)})
				return value, nil
			})
		})
	}
	group.Wait()

	if err := errors.Join(errs...); err != nil {
		t.Fatalf("Update() = %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Tasks) != updates {
		t.Errorf("Load() holds %d tasks, want %d: an update was lost", len(loaded.Tasks), updates)
	}
	assertOnlySketch(t, dir)
}

func TestUpdateWithoutItsLockDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	err := Update(missing, filepath.Join(missing, "fight-camera"), func(value sketch.Sketch) (sketch.Sketch, error) {
		return value, nil
	})

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Update() = %v, want an error wrapping %v", err, fs.ErrNotExist)
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		existing  bool
		edit      func(sketch.Sketch) (sketch.Sketch, error)
		expected  error
		leavesDir bool
	}{
		{"Starts from an empty sketch", false, func(value sketch.Sketch) (sketch.Sketch, error) {
			if !reflect.DeepEqual(value, sketch.New()) {
				return value, fmt.Errorf("got %+v", value)
			}
			value.Tasks = []sketch.Task{{Name: "frame-kills"}}
			return value, nil
		}, nil, true},
		{"Edits the saved sketch", true, func(value sketch.Sketch) (sketch.Sketch, error) {
			value.Tasks = value.Tasks[1:]
			return value, nil
		}, nil, true},
		{"A refused edit writes nothing and creates no dirs", false,
			func(value sketch.Sketch) (sketch.Sketch, error) { return value, errEdit }, errEdit, false},
		{"An invalid result writes nothing", true, func(value sketch.Sketch) (sketch.Sketch, error) {
			value.SchemaVersion = 2
			return value, nil
		}, validation.ErrInvalid, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := t.TempDir()
			dir := FeatureDir(parent, "fight-camera")
			if tt.existing {
				if err := Save(dir, fixture()); err != nil {
					t.Fatal(err)
				}
			}

			err := Update(parent, dir, tt.edit)

			if !errors.Is(err, tt.expected) {
				t.Fatalf("Update() = %v, want %v", err, tt.expected)
			}
			_, statErr := os.Stat(dir)
			if (statErr == nil) != tt.leavesDir {
				t.Errorf("dir exists = %v, want %v", statErr == nil, tt.leavesDir)
			}
			if entries, _ := os.ReadDir(parent); !tt.leavesDir && len(entries) != 0 {
				t.Errorf("Update() left %d entries in the parent, want none", len(entries))
			}
			if tt.existing && err != nil {
				if loaded, _ := Load(dir); !reflect.DeepEqual(loaded, fixture()) {
					t.Errorf("a failed Update() changed the saved sketch to %+v", loaded)
				}
			}
		})
	}
}
