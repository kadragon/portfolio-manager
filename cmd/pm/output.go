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
