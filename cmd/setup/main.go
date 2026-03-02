// Package main implements the setup command, which configures a freshly
// scaffolded project by stripping unused infrastructure based on user input.
// Source files use comment markers (e.g. "# [tag]" / "# [/tag]") to delimit
// feature-specific blocks that can be cleanly removed.
//
// Run with: go run ./cmd/setup
package main

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/charmbracelet/huh"
)

// setup holds the user's configuration choices collected from the form.
type setup struct {
	docker    bool
	postgres  bool
	pgHosting hosting
	packages  []int
	confirm   bool
}

func main() {
	s := setup{
		docker:   true,
		postgres: true,
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Include Docker?").
				Description("Multi-stage Dockerfile, compose for dev, hot reload").
				Options(
					huh.NewOption("Yes", true).Selected(true),
					huh.NewOption("No", false),
				).
				Value(&s.docker),

			huh.NewSelect[bool]().
				Title("Include PostgreSQL?").
				Description("pgx pool, sqlc queries, goose migrations").
				Options(
					huh.NewOption("Yes", true).Selected(true),
					huh.NewOption("No", false),
				).
				Value(&s.postgres),
		),

		huh.NewGroup(
			huh.NewSelect[hosting]().
				Title("PostgreSQL hosting").
				Options(
					huh.NewOption("Docker", hostingDocker).Selected(true),
					huh.NewOption("External", hostingExternal),
				).
				Value(&s.pgHosting),
		).WithHideFunc(func() bool {
			return !s.postgres || !s.docker
		}),

		huh.NewGroup(
			packageSelect(&s.packages),
		),

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Apply changes?").
				Options(
					huh.NewOption("Apply", true).Selected(true),
					huh.NewOption("Cancel", false),
				).
				Value(&s.confirm),
		),
	).WithTheme(theme()).Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !s.confirm {
		fmt.Println("setup cancelled")
		os.Exit(0)
	}

	apply(s)
}

// packageSelect builds a multi-select field from the optionalPackages registry.
// All packages are pre-selected (batteries-included default).
func packageSelect(value *[]int) *huh.MultiSelect[int] {
	opts := make([]huh.Option[int], len(optionalPackages))
	for i, p := range optionalPackages {
		opts[i] = huh.NewOption(p.label, i).Selected(true)
	}

	return huh.NewMultiSelect[int]().
		Title("Packages to include").
		Description("Pre-selected. Deselect any you don't need.").
		Options(opts...).
		Value(value)
}

// apply removes disabled features and cleans up the project.
func apply(s setup) {
	if !s.docker {
		warn(removeFeature(dockerRemoval))
	}

	if !s.postgres {
		warn(removeFeature(postgresRemoval))
	} else if !s.docker || s.pgHosting == hostingExternal {
		warn(removeFeature(postgresExternalRemoval))
	}

	for i, p := range optionalPackages {
		if !slices.Contains(s.packages, i) {
			warn(removeFeature(p.feature))
		}
	}

	warn(removeSelf())
	warn(tidyModules())

	fmt.Println("\nsetup complete")
}

// warn prints a non-fatal error to stderr.
func warn(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
}
