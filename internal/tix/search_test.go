package tix_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/theroymind/go-tix/contracts/card"
	"github.com/theroymind/go-tix/internal/tix"
	"github.com/theroymind/go-tix/internal/tix/index"
	"github.com/theroymind/railyard-cli"
	"github.com/theroymind/railyard-cli/strictjson"
)

const (
	searchModel   = "fake-model"
	scoreRounding = 1000
)

// vocabulary is what the fake endpoint can see: a text's vector counts each word of it.
var vocabulary = []string{"start", "far", "away", "walk", "idea", "specifies", "phase"}

// embeddingServer embeds each input as its vocabulary word counts and remembers every input it was given.
type embeddingServer struct {
	server *httptest.Server
	mutex  sync.Mutex
	inputs []string
}

func startEmbeddingServer(t *testing.T) *embeddingServer {
	t.Helper()
	fake := &embeddingServer{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var received struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		fake.mutex.Lock()
		fake.inputs = append(fake.inputs, received.Input...)
		fake.mutex.Unlock()
		data := []map[string]any{}
		for position, text := range received.Input {
			data = append(data, map[string]any{"index": position, "embedding": wordCounts(text)})
		}
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(map[string]any{"data": data}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *embeddingServer) endpoint() string {
	return fake.server.URL + "/v1"
}

func (fake *embeddingServer) takeInputs() []string {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	taken := fake.inputs
	fake.inputs = nil
	return taken
}

func wordCounts(text string) []float32 {
	counts := make([]float32, len(vocabulary))
	for _, word := range index.Words(text) {
		for position, known := range vocabulary {
			if word == known {
				counts[position]++
			}
		}
	}
	return counts
}

func embeddingsLines(endpoint, model string) string {
	return "embeddings:\n  endpoint: " + endpoint + "\n  model: " + model + "\n"
}

// searchVault is the context vault with the fake endpoint configured and the state directory in a temporary home.
func searchVault(t *testing.T) (string, *embeddingServer) {
	t.Helper()
	fake := startEmbeddingServer(t)
	root := contextVaultWith(t, embeddingsLines(fake.endpoint(), searchModel), nil)
	t.Setenv(cli.StateHomeVariable, t.TempDir())
	return root, fake
}

func runIndex(t *testing.T, root string) string {
	t.Helper()
	exitCode, output := run(t, root, "index")
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("index: exit code = %d, output %q", exitCode, output)
	}
	return output
}

func runSearchJSON(t *testing.T, root string, arguments ...string) []card.Hit {
	t.Helper()
	exitCode, output := run(t, root, append([]string{"search", "--json"}, arguments...)...)
	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("search: exit code = %d, output %q", exitCode, output)
	}
	var hits []card.Hit
	if err := strictjson.Unmarshal([]byte(output), &hits); err != nil {
		t.Fatalf("decoding %s: %v", output, err)
	}
	for _, hit := range hits {
		if err := hit.Validate(); err != nil {
			t.Errorf("hit %+v: %v", hit, err)
		}
	}
	return hits
}

func hitIDs(hits []card.Hit) []string {
	ids := []string{}
	for _, hit := range hits {
		ids = append(ids, hit.ID)
	}
	return ids
}

func TestSearchRanks(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		expected  []string
	}{
		{"The closest note comes first and notes sharing no word tie at 0, by id", []string{"far away"},
			[]string{"far", "dependency", "farther", "idea", "phase", "spec", "start"}},
		{"--top keeps the first N", []string{"far away", "--top", "2"}, []string{"far", "dependency"}},
		{"--type keeps one note type", []string{"far away", "--type", "spec"}, []string{"spec"}},
		{"--type takes a ticket type and reaches the project's name for it", []string{"start", "--type", "epic"}, []string{"phase"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := searchVault(t)
			runIndex(t, root)

			hits := runSearchJSON(t, root, tt.arguments...)

			if !reflect.DeepEqual(hitIDs(hits), tt.expected) {
				t.Errorf("ranked %v, expected %v", hitIDs(hits), tt.expected)
			}
		})
	}
}

func TestSearchHits(t *testing.T) {
	root, _ := searchVault(t)
	runIndex(t, root)

	hits := runSearchJSON(t, root, "far away", "--top", "3")

	for position := range hits {
		hits[position].Score = math.Round(hits[position].Score*scoreRounding) / scoreRounding
	}
	farScore := math.Round(3/math.Sqrt(5*2)*scoreRounding) / scoreRounding
	expected := []card.Hit{
		{ID: "far", Type: "task", Title: "Far", Summary: "Far away.", HasSummary: true, Score: farScore},
		{ID: "dependency", Type: "task", Title: "Dependency", Score: 0},
		{ID: "farther", Type: "task", Title: "Farther", Summary: "Farther still.", HasSummary: true, Score: 0},
	}
	if !reflect.DeepEqual(hits, expected) {
		t.Errorf("hits =\n%+v\nexpected\n%+v", hits, expected)
	}
}

func TestSearchTable(t *testing.T) {
	root, _ := searchVault(t)
	runIndex(t, root)

	exitCode, output := run(t, root, "search", "far away", "--top", "3")

	if exitCode != cli.ExitCodeSuccess {
		t.Fatalf("exit code = %d, output %q", exitCode, output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedRows := [][]string{
		{"SCORE", "ID", "TYPE", "SUMMARY"},
		{"0.949", "far", "task", "Far away."},
		{"0.000", "dependency", "task", "(no summary)"},
		{"0.000", "farther", "task", "Farther still."},
	}
	if len(lines) != len(expectedRows) {
		t.Fatalf("output has %d lines, expected %d:\n%s", len(lines), len(expectedRows), output)
	}
	for position, row := range expectedRows {
		for _, cell := range row {
			if !strings.Contains(lines[position], cell) {
				t.Errorf("line %q does not hold %q", lines[position], cell)
			}
		}
	}
}

func TestIndexEmbedsOnlyChangedNotes(t *testing.T) {
	root, fake := searchVault(t)
	runIndex(t, root)
	first := fake.takeInputs()
	if len(first) != len(contextNotes)-1 {
		t.Fatalf("the first index embedded %d texts, expected every mapped note: %q", len(first), first)
	}

	runIndex(t, root)
	if again := fake.takeInputs(); len(again) != 0 {
		t.Errorf("an unchanged vault embedded %q again", again)
	}

	rewordedFar := strings.Replace(contextNotes["tasks/far.md"], "Far away.", "Far away, walk on.", 1)
	if err := os.WriteFile(filepath.Join(root, "docs", "tasks", "far.md"), []byte(rewordedFar), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "docs", "tasks", "farther.md")); err != nil {
		t.Fatal(err)
	}
	output := runIndex(t, root)

	if changed := fake.takeInputs(); !reflect.DeepEqual(changed, []string{"Far\nFar away, walk on."}) {
		t.Errorf("embedded %q, expected only the reworded note", changed)
	}
	if !strings.Contains(output, "(1 embedded)") {
		t.Errorf("output %q doesn't report one embedded", output)
	}
	hits := runSearchJSON(t, root, "away")
	if ids := hitIDs(hits); ids[0] != "far" || slices.Contains(ids, "farther") {
		t.Errorf("after the change the hits are %v; expected far first and farther gone", ids)
	}
}

func TestSearchWithoutAnIndex(t *testing.T) {
	root, _ := searchVault(t)

	exitCode, output := run(t, root, "search", "far")

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "run tix index") {
		t.Errorf("exit code = %d, output %q; expected a failure saying to run tix index", exitCode, output)
	}
}

func TestSearchWithAnIndexFromAnotherModel(t *testing.T) {
	root, fake := searchVault(t)
	runIndex(t, root)
	writeConfig(t, root, contextConfig+embeddingsLines(fake.endpoint(), "other-model"))

	exitCode, output := run(t, root, "search", "far")

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "index: "+searchModel) || !strings.Contains(output, "config: other-model") {
		t.Errorf("exit code = %d, output %q; expected a failure naming both models", exitCode, output)
	}
}

func TestIndexWithAnotherModelRebuilds(t *testing.T) {
	root, fake := searchVault(t)
	runIndex(t, root)
	fake.takeInputs()
	writeConfig(t, root, contextConfig+embeddingsLines(fake.endpoint(), "other-model"))

	runIndex(t, root)

	if rebuilt := fake.takeInputs(); len(rebuilt) != len(contextNotes)-1 {
		t.Errorf("a new model embedded %d texts, expected every mapped note", len(rebuilt))
	}
	if hits := runSearchJSON(t, root, "far", "--top", "1"); hitIDs(hits)[0] != "far" {
		t.Errorf("search after the rebuild ranked %v first", hitIDs(hits))
	}
}

func TestEndpointDown(t *testing.T) {
	root, fake := searchVault(t)
	runIndex(t, root)
	stateHome := os.Getenv(cli.StateHomeVariable)
	indexFile := index.Path(stateHome, root)
	before, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatal(err)
	}
	rewordedFar := strings.Replace(contextNotes["tasks/far.md"], "Far away.", "Far away, walk on.", 1)
	if err := os.WriteFile(filepath.Join(root, "docs", "tasks", "far.md"), []byte(rewordedFar), 0o644); err != nil {
		t.Fatal(err)
	}
	fake.server.Close()

	tests := []struct {
		name      string
		arguments []string
	}{
		{"search exits 1 naming the endpoint", []string{"search", "far"}},
		{"index exits 1 naming the endpoint and leaves the index as it was", []string{"index"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeFailure || !strings.Contains(output, fake.endpoint()) {
				t.Errorf("exit code = %d, output %q; expected a failure naming %s", exitCode, output, fake.endpoint())
			}
			after, err := os.ReadFile(indexFile)
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Errorf("the index changed or went missing (%v)", err)
			}
		})
	}
}

func TestSearchNeedsEmbeddingsConfig(t *testing.T) {
	root := contextVault(t)
	t.Setenv(cli.StateHomeVariable, t.TempDir())

	for _, command := range []string{"index", "search"} {
		exitCode, output := run(t, root, command, "far")
		if command == "index" {
			exitCode, output = run(t, root, command)
		}

		if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "embeddings") {
			t.Errorf("%s: exit code = %d, output %q; expected a failure naming embeddings", command, exitCode, output)
		}
	}
}

func TestIndexWithClaudeSearch(t *testing.T) {
	root := contextVaultWith(t, "claude:\n  model: haiku\n", nil)
	t.Setenv(cli.StateHomeVariable, t.TempDir())

	exitCode, output := run(t, root, "index")

	if exitCode != cli.ExitCodeFailure || !strings.Contains(output, "needs no index") {
		t.Errorf("exit code = %d, output %q; expected a failure saying claude needs no index", exitCode, output)
	}
}

func TestSearchUsage(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		mentions  string
	}{
		{"A query is required", []string{"search"}, tix.ErrExpectedOneQuery.Error()},
		{"--top takes a whole number from 1", []string{"search", "--top", "0", "far"}, tix.ErrInvalidTop.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := searchVault(t)

			exitCode, output := run(t, root, tt.arguments...)

			if exitCode != cli.ExitCodeUsage || !strings.Contains(output, tt.mentions) {
				t.Errorf("exit code = %d, output %q; expected a usage error mentioning %q", exitCode, output, tt.mentions)
			}
		})
	}
}
