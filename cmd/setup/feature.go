package main

import (
	"errors"
	"fmt"
	"slices"
)

// tag names a marked section in a project file, such as "postgres" in
// "# [postgres]".
type tag string

// Marker tags of the infrastructure choices. When adding marked sections to
// project files, define the new tag here and add it to knownTags. Optional
// packages need no entry: each one's name is its tag.
const (
	tagServer         tag = "server"
	tagClient         tag = "client"
	tagDocker         tag = "docker"
	tagPostgres       tag = "postgres"
	tagPostgresDocker tag = "postgres-docker"
	tagSetup          tag = "setup"
)

// knownTags returns every tag a project file may use: the infrastructure
// tags, then the tags of each optional package. A marker with any other tag
// is a mistake and aborts setup.
func knownTags() []tag {
	tags := []tag{
		tagServer,
		tagClient,
		tagDocker,
		tagPostgres,
		tagPostgresDocker,
		tagSetup,
	}

	for _, p := range optionalPackages {
		tags = append(tags, p.removal().tags...)
	}

	return tags
}

// hosting describes how PostgreSQL is managed.
type hosting string

const (
	hostingDocker   hosting = "docker"
	hostingExternal hosting = "external"
)

// MarshalText implements encoding.TextMarshaler so a hosting can serve as a
// flag default.
func (h hosting) MarshalText() ([]byte, error) {
	return []byte(h), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, accepting only the
// known hosting modes.
func (h *hosting) UnmarshalText(text []byte) error {
	value := hosting(text)
	if value != hostingDocker && value != hostingExternal {
		return fmt.Errorf("unknown hosting %q, want %s or %s", text, hostingDocker, hostingExternal)
	}

	*h = value

	return nil
}

// Project files referenced by removal specs. Using constants turns a
// misspelled name into a compile error.
const (
	fileMain         = "cmd/template/main.go"
	fileMakefile     = "Makefile"
	fileEnvExample   = ".env.example"
	fileCompose      = "compose.yaml"
	fileDockerfile   = "Dockerfile"
	fileDockerignore = ".dockerignore"
	fileGolangci     = ".golangci.yml"
	fileReadme       = "README.md"
)

// markedFiles lists every file that may contain markers. Removing a feature
// removes its tagged sections from each of these files, and setup strips the
// remaining markers so none reach the finished project.
var markedFiles = []string{
	fileMain,
	fileMakefile,
	fileEnvExample,
	fileCompose,
	fileReadme,
}

// feature describes what to delete when a feature is disabled.
type feature struct {
	paths []string // files and directories, removed recursively
	tags  []tag    // marked sections, removed from every marked file
}

// Removal specs for the infrastructure choices in the setup form.
var (
	serverRemoval = feature{
		paths: []string{"pkg/middleware", "pkg/respond", "pkg/apperr", "pkg/handler", "pkg/service"},
		tags:  []tag{tagServer},
	}

	clientRemoval = feature{
		paths: []string{"pkg/client", "pkg/template"},
		tags:  []tag{tagClient},
	}

	dockerRemoval = feature{
		paths: []string{fileDockerfile, fileCompose, fileDockerignore},
		tags:  []tag{tagDocker},
	}

	postgresRemoval = feature{
		paths: []string{"pkg/db", "pkg/service", "pkg/handler/handler.go"},
		tags:  []tag{tagPostgres, tagPostgresDocker},
	}

	// postgresDockerRemoval keeps PostgreSQL support but drops the bundled
	// database container, for external hosting or when Docker is disabled.
	postgresDockerRemoval = feature{
		tags: []tag{tagPostgresDocker},
	}

	// setupRemoval deletes the setup command itself. It always runs last.
	setupRemoval = feature{
		paths: []string{"cmd/setup"},
		tags:  []tag{tagSetup},
	}
)

// optionalPackage describes a selectable package shown in the setup form.
type optionalPackage struct {
	// name identifies the package in the -packages flag, names its
	// directory under pkg/, and is its marker tag.
	name  string
	label string // describes the package in the form
}

// removal returns what to delete when the package is deselected: its
// directory and every section marked with its name.
func (p optionalPackage) removal() feature {
	return feature{
		paths: []string{"pkg/" + p.name},
		tags:  []tag{tag(p.name)},
	}
}

// optionalPackages is the single source of truth for all toggleable
// packages. Add new entries here; the form, the -packages flag, the known
// tags, and removal derive from it automatically.
var optionalPackages = []optionalPackage{
	{
		name:  "worker",
		label: "Worker Pool: bounded concurrency via errgroup",
	},
	{
		name:  "retry",
		label: "Retry: exponential backoff with jitter",
	},
	{
		name:  "state",
		label: "State: file-backed JSON with file locking",
	},
	{
		name:  "cycle",
		label: "Cycle: thread-safe round-robin rotator",
	},
	{
		name:  "fake",
		label: "Fake Data: fake data generation helpers",
	},
	{
		name:  "ptr",
		label: "Pointer Helpers: nil-safe dereference and comparison",
	},
}

// packageNames returns the names of all optional packages in form order.
func packageNames() []string {
	names := make([]string, len(optionalPackages))
	for i, p := range optionalPackages {
		names[i] = p.name
	}

	return names
}

// config holds the choices that decide which features the project keeps.
type config struct {
	server    bool
	client    bool
	docker    bool
	postgres  bool
	pgHosting hosting
	packages  []string // names of the optional packages to keep
}

// errNoWorkload reports a config that would leave the entrypoint with
// nothing to run, so cmd/template would not compile.
var errNoWorkload = errors.New("at least one of the server or the client must be enabled")

// validate reports whether c describes a project that builds.
func (c config) validate() error {
	if !c.server && !c.client {
		return errNoWorkload
	}

	return nil
}

// removal returns everything to delete for the features c disables, ending
// with the setup command itself.
func (c config) removal() feature {
	var removal feature
	for _, f := range c.disabled() {
		removal.paths = append(removal.paths, f.paths...)
		removal.tags = append(removal.tags, f.tags...)
	}

	return removal
}

// disabled returns the removal spec of every feature c turns off, followed
// by setupRemoval.
func (c config) disabled() []feature {
	var features []feature

	if !c.server {
		features = append(features, serverRemoval)
	}

	if !c.client {
		features = append(features, clientRemoval)
	}

	if !c.docker {
		features = append(features, dockerRemoval)
	}

	switch {
	case !c.postgres:
		features = append(features, postgresRemoval)
	case !c.docker || c.pgHosting == hostingExternal:
		features = append(features, postgresDockerRemoval)
	}

	for _, p := range optionalPackages {
		if !slices.Contains(c.packages, p.name) {
			features = append(features, p.removal())
		}
	}

	return append(features, setupRemoval)
}
