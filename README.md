# tix

A per-repo ticket tracker for humans and coding agents. Tickets are YAML files in `.tix/`, committed with the code, or notes in an Obsidian vault. `tix ready` answers "what do I pick up next?". See [docs/tix/SPEC.md](docs/tix/SPEC.md).

## Install

```bash
go install github.com/theroymind/go-tix/cmd/tix@latest
```

## Use

```bash
tix init --prefix ab
id=$(tix add --type task --acceptance "..." "title")
tix ready
tix show "$id"
tix done "$id"
```

## Contracts

`contracts/ticket`, `contracts/card`, `contracts/timestamp` and `contracts/validation` are public. Other tools, such as [Railyard](https://github.com/theroymind/railyard), import them to read tix's output.

## Development

```bash
make hooks   # point git at .githooks
make check   # format, tidy, build, vet, test, lint
make build   # build every app into bin/
```
