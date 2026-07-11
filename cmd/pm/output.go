package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

// printJSON writes v to stdout as indented JSON.
func printJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	// codeql[go/clear-text-logging] -- intended CLI stdout output; any KisAPIKeyID here is a config index (1-9), not a secret (see prior dismissals of alerts #4/#5)
	fmt.Println(string(out))
	return nil
}

// parseFlags parses args with fs, wrapping the error so every subcommand
// doesn't need its own wrapcheck-satisfying boilerplate around fs.Parse.
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	return nil
}
