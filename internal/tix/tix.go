// Package tix provides the tix commands, which create, show and complete the tickets in a .tix directory.
package tix

import "github.com/theroymind/railyard/internal/cli"

const usage = `usage: tix <command> [arguments]

commands:
  init --prefix PREFIX
  add --type TYPE [--priority N] [--tag T]... [--acceptance A]... [--description D] [--draft] TITLE
  show REF [--json]
  done REF
`

var Program = cli.Program{
	Name:  "tix",
	Usage: usage,
	Commands: map[string]cli.Command{
		"init": runInit,
		"add":  runAdd,
		"show": runShow,
		"done": runDone,
	},
}
