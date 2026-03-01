// setup configures a freshly scaffolded project by stripping unused
// postgres infrastructure based on user input.
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

// postgresMode describes how (or whether) the project uses PostgreSQL.
type postgresMode int

const (
	postgresNone     postgresMode = iota // no postgres at all
	postgresDocker                       // postgres runs via docker compose
	postgresExternal                     // postgres is managed externally
)

func main() {
	mode := prompt()

	switch mode {
	case postgresDocker:
		slog.Info("kept postgresql with docker, no changes")

	case postgresExternal:
		removeComposeService("compose.yaml", "postgres")
		removeMakeTargets("Makefile", "db", "db-down")
		slog.Info("configured postgresql for external use")

	case postgresNone:
		removeDir("pkg/db")
		removeComposeService("compose.yaml", "postgres")
		removeMatchingLines(".env.example", "DATABASE_URL", "postgres", "Postgres")
		removeMakeTargets("Makefile", "db", "db-down", "migrate", "migrate-down", "migrate-new")
		slog.Info("removed all postgresql infrastructure")
	}

	// Remove the setup command itself and tidy modules so setup-only
	// dependencies (like the yaml package) are pruned automatically.
	removeSelf()
	tidyModules()
}

// prompt asks the user how they want to configure postgres and returns
// the selected mode.
func prompt() postgresMode {
	scanner := bufio.NewScanner(os.Stdin)
	ask := func(q string) string {
		fmt.Print(q)
		scanner.Scan()
		return strings.TrimSpace(scanner.Text())
	}

	if !strings.EqualFold(ask("Use PostgreSQL? [y/N] "), "y") {
		return postgresNone
	}

	if strings.EqualFold(ask("Docker or external? [docker/external] "), "external") {
		return postgresExternal
	}

	return postgresDocker
}
