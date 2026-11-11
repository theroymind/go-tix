# tix

A per-repo ticket tracker and vault tool for humans and coding agents. One module, `github.com/theroymind/go-tix`, with the `tix` binary.

## Development

```text
make build    # go build cmd/tix into bin/
make test     # go test ./...
make vet      # go vet ./...
make lint     # golangci-lint run
make install  # go install ./cmd/...
make check    # everything CI runs: format, tidy, build, vet, test, lint
make hooks    # point git at .githooks (pre-commit runs format, tidy, vet, lint)
```

## Where the work is

- `docs/tix/SPEC.md` (tix v1) defines the `.tix/` format, and `docs/tix/GLOSSARY.md` its terms. Files tix writes must stay byte-compatible with v1.
- `docs/` is this repo's own tix vault, the starter one `tix init` writes: each feature in `features/<feature>/` with its tasks in `tasks/` inside it, and `ideas/` beside them. `tix ready` names the next task, and `tix lint --all` checks the notes.

## Layout

- `cmd/tix` holds the `main`.
- `contracts/<domain>` holds the public schemas as structs with `Validate` methods: `ticket`, `card`, `timestamp`, `validation`. Railyard imports these, so they are a public API.
- `internal/<app>/` holds each app's own packages.
- Process plumbing (args, subcommands, exit codes, JSON, child processes, state directory) comes from `github.com/theroymind/railyard-cli`, and strict JSON decoding from its `strictjson` package.
- `contracts` never imports `internal`. golangci-lint's `depguard` rule enforces it.

## Code standards

- Every package has one package comment, directly above the `package` clause in one file: `// Package ticket ...`, or `// Command tix ...` for `main`, saying what it provides in a sentence or two. It moves to `doc.go` once it runs past a few lines.
- Errors are sentinel `var Err... = errors.New(...)` values, wrapped with `%w` and checked with `errors.Is`.
- Tests are table-driven.
- JSON at a boundary decodes through `strictjson`, then `Validate`. Unknown `schema_version` or `type` is rejected, never guessed at.
- No app imports another app.
- Ask before adding a dependency or any download, and give its size.
- Run `make check` before reporting a change done.

## Go documentation

Check `go doc <pkg>.<Symbol>` and the package's `example_test.go` under `$(go env GOROOT)/src` before using an API you're unsure of.

## Compatibility

Railyard imports `contracts/*`. A change to an exported name there is a breaking change: tag a new major version, or update Railyard in step.
