package main

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// tag identifies a marked section in a file (e.g. "# [postgres]").
type tag string

func (t tag) String() string {
	return string(t)
}

// Marker tags used across the project. Define new tags here when adding
// marked sections to project files.
const (
	tagDocker         tag = "docker"
	tagPostgres       tag = "postgres"
	tagPostgresDocker tag = "postgres-docker"
	tagSetup          tag = "setup"
)

// hosting describes how PostgreSQL is managed.
type hosting string

const (
	hostingDocker   hosting = "docker"
	hostingExternal hosting = "external"
)

// File paths referenced by removal specs. Using constants prevents silent
// typos — a misspelled constant name is a compile error.
const (
	fileMakefile     = "Makefile"
	fileDockerfile   = "Dockerfile"
	fileDockerignore = ".dockerignore"
	fileCompose      = "compose.yaml"
	fileEnvExample   = ".env.example"
	fileEnvConfig    = "pkg/env/config.go"
)

// section identifies a tagged block inside a file.
type section struct {
	file string
	tag  tag
}

// feature describes a set of files, directories, config sections, and compose
// services to remove when a feature is disabled.
type feature struct {
	dirs     []string
	files    []string
	sections []section
	compose  []string
}

// optionalPkg describes a selectable package shown in the setup form.
// The feature field contains everything needed to remove the package.
type optionalPkg struct {
	label   string
	feature feature
}

// Infrastructure removal specs. Compose service removal is safe even when
// compose.yaml has already been deleted (removeComposeService handles this).
var (
	dockerRemoval = feature{
		files: []string{fileDockerfile, fileCompose, fileDockerignore},
		sections: []section{
			{
				file: fileMakefile,
				tag:  tagDocker,
			},
		},
	}

	postgresRemoval = feature{
		dirs:    []string{"pkg/db"},
		compose: []string{"postgres"},
		sections: []section{
			{
				file: fileMakefile,
				tag:  tagPostgres,
			},
			{
				file: fileMakefile,
				tag:  tagPostgresDocker,
			},
			{
				file: fileEnvExample,
				tag:  tagPostgres,
			},
			{
				file: fileEnvConfig,
				tag:  tagPostgres,
			},
		},
	}

	postgresExternalRemoval = feature{
		compose: []string{"postgres"},
		sections: []section{
			{
				file: fileMakefile,
				tag:  tagPostgresDocker,
			},
		},
	}
)

// optionalPackages is the single source of truth for all toggleable packages.
// Add new entries here — the form and removal logic derive automatically.
var optionalPackages = []optionalPkg{
	{
		label: "HTTP Client: TLS fingerprint, proxy, cookies, HTTP/2",
		feature: feature{
			dirs: []string{"pkg/client"},
		},
	},
	{
		label: "Worker Pool: bounded concurrency via errgroup",
		feature: feature{
			dirs: []string{"pkg/worker"},
		},
	},
	{
		label: "Retry: exponential backoff with jitter",
		feature: feature{
			dirs: []string{"pkg/retry"},
		},
	},
	{
		label: "State: file-backed JSON with file locking",
		feature: feature{
			dirs: []string{"pkg/state"},
		},
	},
	{
		label: "Cycle: thread-safe round-robin rotator",
		feature: feature{
			dirs: []string{"pkg/cycle"},
		},
	},
	{
		label: "Fake Data: fake data generation helpers",
		feature: feature{
			dirs: []string{"pkg/fake"},
		},
	},
}

// theme returns a minimal monotone theme with checkbox-style indicators.
func theme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")
	t.Blurred.SelectedPrefix = t.Focused.SelectedPrefix
	t.Blurred.UnselectedPrefix = t.Focused.UnselectedPrefix
	return t
}
