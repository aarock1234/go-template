// setup configures a freshly scaffolded project by stripping unused
// infrastructure based on user input. Source files use comment markers
// (e.g. "# [postgres]" / "# [/postgres]") to delimit feature-specific
// blocks that can be cleanly removed.
//
// Run with: go run ./cmd/setup
package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/aarock1234/go-template/pkg/log" // colorized slog
)

var scanner = bufio.NewScanner(os.Stdin)

func main() {
	configurePostgres()
	// future: configureRedis(), configureAuth(), etc.

	removeSelf()
	tidyModules()
}

// ask prints a question and returns the trimmed response.
func ask(q string) string {
	fmt.Print(q)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

// configurePostgres prompts the user for PostgreSQL preferences and strips
// unused postgres infrastructure accordingly.
func configurePostgres() {
	if !strings.EqualFold(ask("Use PostgreSQL? [y/N] "), "y") {
		removePostgres()
		return
	}

	if strings.EqualFold(ask("Docker or external? [docker/external] "), "external") {
		removePostgresDocker()
		return
	}

	slog.Info("kept postgresql with docker, no changes")
}

// removePostgres strips all postgres infrastructure from the project.
func removePostgres() {
	if err := removeDir("pkg/db"); err != nil {
		slog.Error("remove directory", "error", err)
	}
	if err := removeComposeService("compose.yaml", "postgres"); err != nil {
		slog.Error("remove compose service", "error", err)
	}
	if err := removeMarkedSections("Makefile", "postgres"); err != nil {
		slog.Error("remove makefile sections", "error", err)
	}
	if err := removeMarkedSections(".env.example", "postgres"); err != nil {
		slog.Error("remove env sections", "error", err)
	}
	slog.Info("removed all postgresql infrastructure")
}

// removePostgresDocker strips only the docker-managed postgres pieces,
// keeping the Go database package for an externally managed instance.
func removePostgresDocker() {
	if err := removeComposeService("compose.yaml", "postgres"); err != nil {
		slog.Error("remove compose service", "error", err)
	}
	if err := removeMarkedSections("Makefile", "postgres-docker"); err != nil {
		slog.Error("remove makefile sections", "error", err)
	}
	slog.Info("configured postgresql for external use")
}
