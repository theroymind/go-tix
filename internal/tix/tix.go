// Package tix provides the tix commands, which create, list, show and complete the tickets in a .tix directory.
package tix

import "github.com/theroymind/railyard/internal/cli"

const usage = `usage: tix <command> [arguments]

commands:
  init --prefix PREFIX
  add --type TYPE [--priority N] [--tag T]... [--acceptance A]... [--description D] [--draft] TITLE
  show REF [--json]
  ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]
  ready [--type T] [--json]
  done REF
`

var Program = cli.Program{
	Name:  "tix",
	Usage: usage,
	Commands: map[string]cli.Command{
		"init":  runInit,
		"add":   runAdd,
		"show":  runShow,
		"ls":    runLs,
		"ready": runReady,
		"done":  runDone,
	},
}
