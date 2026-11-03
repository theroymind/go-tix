// Package tix provides the tix commands, which create, list, show, change and complete the tickets in a .tix directory.
package tix

import "github.com/theroymind/railyard/internal/cli"

const usage = `usage: tix <command> [arguments]

commands:
  init --prefix PREFIX
  add --type TYPE [--parent REF] [--priority N] [--tag T]... [--dependency REF]... [--acceptance A]... [--description D] [--draft] TITLE
  show REF [--json]
  ls [--status S] [--type T] [--tag T] [--parent REF] [--all] [--json]
  ready [--type T] [--json]
  done [--force] REF
  set REF KEY=VALUE...   (raw write; does not check criteria, use done)
  tick REF N...          (N is a criterion number from show)
  untick REF N...
  ac set REF N TEXT
  ac add REF TEXT
`

var Program = cli.Program{
	Name:  "tix",
	Usage: usage,
	Commands: map[string]cli.Command{
		"init":   runInit,
		"add":    runAdd,
		"show":   runShow,
		"ls":     runLs,
		"ready":  runReady,
		"done":   runDone,
		"set":    runSet,
		"tick":   runTick,
		"untick": runUntick,
		"ac":     runAc,
	},
}
