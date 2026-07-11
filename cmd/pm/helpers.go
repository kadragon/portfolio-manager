package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// resolveAccountByName finds an account by exact name, falling back to a
// unique case-insensitive substring match. Mirrors
// OrderExecutionService.findAccount (internal/services/order_execution_service.go)
// so "-account ISA" ergonomics are consistent across cmd/rebalance-order and
// cmd/pm.
func resolveAccountByName(ctx context.Context, c *container.Container, name string) (models.Account, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Account{}, fmt.Errorf("-account is required")
	}
	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		return models.Account{}, fmt.Errorf("list accounts: %w", err)
	}
	for _, a := range accounts {
		if a.Name == name {
			return a, nil
		}
	}

	lower := strings.ToLower(name)
	var matches []models.Account
	for _, a := range accounts {
		if strings.Contains(strings.ToLower(a.Name), lower) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return models.Account{}, fmt.Errorf("no account matches %q", name)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, a := range matches {
			names[i] = a.Name
		}
		return models.Account{}, fmt.Errorf("account name %q is ambiguous: matches %s", name, strings.Join(names, ", "))
	}
}

// resolveGroupRef resolves ref as a group UUID first, falling back to an
// exact name match — so -group flags accept either a UUID or a group name.
func resolveGroupRef(ctx context.Context, c *container.Container, ref string) (models.Group, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return models.Group{}, fmt.Errorf("-group is required")
	}
	if id, err := uuidx.Parse(ref); err == nil {
		g, err := c.Groups.GetByID(ctx, id)
		if err != nil {
			return models.Group{}, fmt.Errorf("get group: %w", err)
		}
		if g != nil {
			return *g, nil
		}
	}
	groups, err := c.Groups.ListAll(ctx)
	if err != nil {
		return models.Group{}, fmt.Errorf("list groups: %w", err)
	}
	for _, g := range groups {
		if g.Name == ref {
			return g, nil
		}
	}
	return models.Group{}, fmt.Errorf("no group matches %q", ref)
}
