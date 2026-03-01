// setup is an interactive project configurator.
// Run with: go run ./cmd/setup
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	green = "\033[0;32m"
	reset = "\033[0m"
)

func prompt(msg string) string {
	fmt.Printf("%s%s%s", green, msg, reset)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func main() {
	var changes []string

	usePG := prompt("Use PostgreSQL? [y/N] ")
	if strings.EqualFold(usePG, "y") {
		mode := prompt("Docker or external? [docker/external] ")
		if strings.EqualFold(mode, "external") {
			if err := removePGServiceFromCompose("compose.yaml"); err == nil {
				changes = append(changes, "✓ Removed postgres service from compose.yaml (kept app)")
			}
			if err := removeMakeTargets("Makefile", "db", "db-down"); err == nil {
				changes = append(changes, "✓ Removed make db / db-down targets")
			}
		} else {
			changes = append(changes, "✓ Kept PostgreSQL with Docker (no changes)")
		}
	} else {
		if err := os.RemoveAll("pkg/db"); err == nil {
			changes = append(changes, "✓ Removed pkg/db/")
		}
		if err := removePGServiceFromCompose("compose.yaml"); err == nil {
			changes = append(changes, "✓ Removed postgres service from compose.yaml")
		}
		if err := removeEnvLines(".env.example", "DATABASE_URL", "postgres", "Postgres"); err == nil {
			changes = append(changes, "✓ Removed DATABASE_URL from .env.example")
		}
		if err := removeMakeTargets("Makefile", "db", "db-down", "migrate", "migrate-down", "migrate-new"); err == nil {
			changes = append(changes, "✓ Removed db/migration Make targets")
		}
		if err := runGoModTidy(); err == nil {
			changes = append(changes, "✓ Cleaned Go dependencies (go mod tidy)")
		}
	}

	fmt.Println()
	for _, c := range changes {
		fmt.Println(c)
	}
}

// removePGServiceFromCompose removes the postgres service block from compose.yaml.
// It keeps everything from the start of the file up to (but not including) the
// "  postgres:" key, then resumes at the next top-level key.
func removePGServiceFromCompose(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var out []string
	skip := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  postgres:") {
			skip = true
			continue
		}
		if skip {
			// Resume when we hit the next service at the same indent level
			if len(line) > 0 && line[0] == ' ' && !strings.HasPrefix(line, "   ") {
				skip = false
			} else {
				continue
			}
		}
		out = append(out, line)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// removeMakeTargets removes the given target blocks from a Makefile and
// also strips them from the .PHONY line.
func removeMakeTargets(path string, targets ...string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	targetSet := make(map[string]bool, len(targets))
	for _, t := range targets {
		targetSet[t] = true
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	skip := false
	for _, line := range lines {
		// Check if this is a target line we want to remove
		isTarget := false
		for t := range targetSet {
			if strings.HasPrefix(line, t+":") || strings.HasPrefix(line, t+" ") {
				isTarget = true
				break
			}
		}
		if isTarget {
			skip = true
			continue
		}
		// End of a target block: non-recipe, non-empty line resets skip
		if skip {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue // skip blank lines trailing the target
			}
			if !strings.HasPrefix(line, "\t") {
				skip = false
				// Fall through — process this line normally
			} else {
				continue
			}
		}
		// Strip removed targets from .PHONY
		if strings.HasPrefix(line, ".PHONY:") {
			for t := range targetSet {
				line = strings.ReplaceAll(line, " "+t+" ", " ")
				line = strings.ReplaceAll(line, " "+t+"\n", "\n")
				if strings.HasSuffix(line, " "+t) {
					line = strings.TrimSuffix(line, " "+t)
				}
			}
		}
		out = append(out, line)
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// removeEnvLines removes lines from a .env file that contain any of the given keywords.
func removeEnvLines(path string, keywords ...string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		keep := true
		for _, kw := range keywords {
			if strings.Contains(line, kw) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, line)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// runGoModTidy runs `go mod tidy` in the current directory.
func runGoModTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
