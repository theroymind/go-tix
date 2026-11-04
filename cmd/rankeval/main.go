// Command rankeval scores ranking models on a vault's done tasks and prints the precision and latency of each
// as JSON.
//
// Run rankeval run --vault PATH --model none --model haiku --model NAME, where PATH is the root of a git
// repository that holds a tix vault, and NAME is a model the OpenAI-compatible --endpoint serves.
package main

import (
	"os"

	"github.com/theroymind/railyard/internal/rankeval"
)

func main() {
	os.Exit(int(rankeval.Program.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}
