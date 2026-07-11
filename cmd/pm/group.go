package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

func runGroup(ctx context.Context, c *container.Container, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pm group list|add|update|delete [flags]")
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return groupList(ctx, c)
	case "add":
		return groupAdd(ctx, c, rest)
	case "update":
		return groupUpdate(ctx, c, rest)
	case "delete":
		return groupDelete(ctx, c, rest)
	default:
		return fmt.Errorf("unknown group verb %q", verb)
	}
}

func groupList(ctx context.Context, c *container.Container) error {
	groups, err := c.Groups.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}
	return printJSON(groups)
}

func groupAdd(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm group add", flag.ExitOnError)
	name := fs.String("name", "", "group name (required)")
	target := fs.Float64("target", 0, "target percentage (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("-name is required")
	}
	group, err := c.Groups.Create(ctx, *name, *target)
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	return printJSON(group)
}

// groupUpdate applies only the flags the caller explicitly passed (via
// fs.Visit); the repository's Update treats nil pointers as unchanged.
func groupUpdate(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm group update", flag.ExitOnError)
	idRaw := fs.String("id", "", "group id (required)")
	name := fs.String("name", "", "new name")
	target := fs.Float64("target", 0, "new target percentage")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })

	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}

	var namePtr *string
	if seen["name"] {
		namePtr = name
	}
	var targetPtr *float64
	if seen["target"] {
		targetPtr = target
	}

	updated, err := c.Groups.Update(ctx, id, namePtr, targetPtr)
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	return printJSON(updated)
}

func groupDelete(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm group delete", flag.ExitOnError)
	idRaw := fs.String("id", "", "group id (required)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := uuidx.Parse(*idRaw)
	if err != nil {
		return fmt.Errorf("invalid -id: %w", err)
	}
	if err := c.Groups.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return printJSON(map[string]string{"status": "deleted", "id": id.String()})
}
