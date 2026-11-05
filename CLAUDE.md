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

## Layout

- `cmd/tix` holds the `main`.
- `contracts/<domain>` holds the public schemas as structs with `Validate` methods: `ticket`, `card`, `timestamp`, `validation`. Railyard imports these, so they are a public API.
- `internal/<app>/` holds each app's own packages.
- Process plumbing (args, subcommands, exit codes, JSON, child processes, state directory) comes from `github.com/theroymind/railyard-cli`, and strict JSON decoding from its `strictjson` package.
- `contracts` never imports `internal`. golangci-lint's `depguard` rule enforces it.

## Code standards

- Every package has a package comment. See [Package comments](#package-comments).
- Errors are sentinel `var Err... = errors.New(...)` values, wrapped with `%w` and checked with `errors.Is`.
- Tests are table-driven.
- JSON at a boundary decodes through `strictjson`, then `Validate`. Unknown `schema_version` or `type` is rejected, never guessed at.
- `gofmt -l .` prints nothing, and `go vet ./...` and `golangci-lint run` stay clean.

## Package comments

Required by Go tooling (staticcheck ST1000) and shown by `go doc`.

- Exactly one file per package carries it, directly above the `package` clause with no blank line.
- The first sentence is the summary: `// Package ticket ...` for libraries, `// Command tix ...` for `main`. It must stand alone, since tools show it in package lists.
- Describe what the package provides, not how it's implemented or how it changed.
- Default to one or two sentences. Once it grows past a few lines, move it to `doc.go`.
- For `main`, describe what the command does and how to run it, not its Go API.

## Go documentation

Check `go doc <pkg>.<Symbol>` and the package's `example_test.go` under `$(go env GOROOT)/src` before using an API you're unsure of.

## Compatibility

Railyard imports `contracts/*`. A change to an exported name there is a breaking change: tag a new major version, or update Railyard in step.

## Review checklist

- [ ] `make check` passes.
- [ ] `gofmt -l .` prints nothing.
- [ ] No app imports another app.
- [ ] `.tix/` files tix writes still match tix v1's format.
- [ ] Ask before adding a dependency or any download, with its size.
