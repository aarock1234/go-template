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

	"github.com/charmbracelet/huh"
)

// pkg maps an optional package to its directory for removal.
type pkg struct {
	label string
	value string
	dir   string
}

var packages = []pkg{
	{
		label: "HTTP Client: TLS fingerprint, proxy, cookies, HTTP/2 support",
		value: "client",
		dir:   "pkg/client",
	},
	{
		label: "Worker Pool: bounded concurrency via errgroup",
		value: "worker",
		dir:   "pkg/worker",
	},
	{
		label: "Retry: exponential backoff with jitter",
		value: "retry",
		dir:   "pkg/retry",
	},
	{
		label: "State: file-backed JSON with file locking",
		value: "state",
		dir:   "pkg/state",
	},
	{
		label: "Cycle: thread-safe round-robin rotator",
		value: "cycle",
		dir:   "pkg/cycle",
	},
	{
		label: "Fake Data: fake data generation helpers",
		value: "fake",
		dir:   "pkg/fake",
	},
}

// setup holds the user's configuration choices collected from the form.
type setup struct {
	docker   bool
	postgres bool
	hosting  string
	packages []string
}

func main() {
	var (
		useDocker    = true
		usePostgres  = true
		pgHosting    string
		selectedPkgs []string
	)

	err := huh.NewForm(
		// Infrastructure
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Include Docker?").
				Description("Multi-stage Dockerfile, compose for dev, hot reload").
				Options(
					huh.NewOption("Yes", true).Selected(true),
					huh.NewOption("No", false),
				).
				Value(&useDocker),

			huh.NewSelect[bool]().
				Title("Include PostgreSQL?").
				Description("pgx pool, sqlc queries, goose migrations").
				Options(
					huh.NewOption("Yes", true).Selected(true),
					huh.NewOption("No", false),
				).
				Value(&usePostgres),
		),

		// PostgreSQL hosting (only when both docker and postgres are enabled)
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("PostgreSQL hosting").
				Options(
					huh.NewOption("Docker", "docker"),
					huh.NewOption("External", "external"),
				).
				Value(&pgHosting),
		).WithHideFunc(func() bool {
			return !usePostgres || !useDocker
		}),

		// Optional packages
		huh.NewGroup(
			packageSelect(&selectedPkgs),
		),
	).WithTheme(huh.ThemeBase()).Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	apply(setup{
		docker:   useDocker,
		postgres: usePostgres,
		hosting:  pgHosting,
		packages: selectedPkgs,
	})
}

// packageSelect builds a multi-select field for optional packages.
// All packages are pre-selected (batteries-included default).
func packageSelect(value *[]string) *huh.MultiSelect[string] {
	opts := make([]huh.Option[string], len(packages))
	for i, p := range packages {
		opts[i] = huh.NewOption(p.label, p.value).Selected(true)
	}

	return huh.NewMultiSelect[string]().
		Title("Packages to include").
		Description("Pre-selected. Deselect any you don't need.").
		Options(opts...).
		Value(value)
}

// apply removes disabled features and cleans up the project.
func apply(s setup) {
	// Docker
	if !s.docker {
		warn(removeFeature(feature{
			files:    []string{"Dockerfile", "compose.yaml", ".dockerignore"},
			sections: []section{{file: "Makefile", tag: "docker"}},
		}))
	}

	// PostgreSQL
	if !s.postgres {
		f := feature{
			dirs: []string{"pkg/db"},
			sections: []section{
				{file: "Makefile", tag: "postgres"},
				{file: "Makefile", tag: "postgres-docker"},
				{file: ".env.example", tag: "postgres"},
				{file: "pkg/env/config.go", tag: "postgres"},
			},
		}
		if s.docker {
			f.compose = []string{"postgres"}
		}
		warn(removeFeature(f))
	} else if !s.docker || s.hosting == "external" {
		// PostgreSQL kept but forced/chosen external
		f := feature{
			sections: []section{
				{file: "Makefile", tag: "postgres-docker"},
			},
		}
		if s.docker {
			f.compose = []string{"postgres"}
		}
		warn(removeFeature(f))
	}

	// Packages
	selected := make(map[string]bool, len(s.packages))
	for _, p := range s.packages {
		selected[p] = true
	}

	for _, p := range packages {
		if !selected[p.value] {
			warn(removeFeature(feature{dirs: []string{p.dir}}))
		}
	}

	// Cleanup
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
