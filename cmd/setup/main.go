// setup is an interactive project configurator.
// Run with: go run ./cmd/setup
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	green = "\033[0;32m"
	reset = "\033[0m"
)

type config struct {
	postgres bool
	docker   bool
}

func main() {
	cfg := configure()
	fmt.Println()

	switch {
	case !cfg.postgres:
		apply(os.RemoveAll("pkg/db"), "✓ Removed pkg/db/")
		apply(removeSvcBlock("compose.yaml", "postgres"), "✓ Removed postgres from compose.yaml")
		apply(dropLines(".env.example", "DATABASE_URL", "postgres", "Postgres"), "✓ Removed DATABASE_URL from .env.example")
		apply(dropMakeTargets("Makefile", "db", "db-down", "migrate", "migrate-down", "migrate-new"), "✓ Removed db/migration Make targets")
		apply(modTidy(), "✓ Cleaned Go dependencies")
	case !cfg.docker:
		apply(removeSvcBlock("compose.yaml", "postgres"), "✓ Removed postgres from compose.yaml (kept app)")
		apply(dropMakeTargets("Makefile", "db", "db-down"), "✓ Removed make db / db-down targets")
	default:
		fmt.Println("✓ Kept PostgreSQL with Docker (no changes)")
	}
}

func configure() config {
	s := bufio.NewScanner(os.Stdin)
	ask := func(q string) string {
		fmt.Printf("%s%s%s", green, q, reset)
		s.Scan()
		return strings.TrimSpace(s.Text())
	}

	if !strings.EqualFold(ask("Use PostgreSQL? [y/N] "), "y") {
		return config{}
	}

	return config{
		postgres: true,
		docker:   !strings.EqualFold(ask("Docker or external? [docker/external] "), "external"),
	}
}

func apply(err error, msg string) {
	if err == nil {
		fmt.Println(msg)
	}
}

// removeSvcBlock removes a named service block from compose.yaml.
func removeSvcBlock(path, svc string) error {
	return transformLines(path, func(lines []string) []string {
		var out []string
		skip := false
		prefix := "  " + svc + ":"
		for _, line := range lines {
			if strings.HasPrefix(line, prefix) {
				skip = true
				continue
			}
			if skip && len(line) > 0 && !strings.HasPrefix(line, "   ") {
				skip = false
			}
			if !skip {
				out = append(out, line)
			}
		}
		return out
	})
}

// dropLines removes any line containing one of the given substrings.
func dropLines(path string, subs ...string) error {
	return transformLines(path, func(lines []string) []string {
		return slices.DeleteFunc(lines, func(line string) bool {
			for _, sub := range subs {
				if strings.Contains(line, sub) {
					return true
				}
			}
			return false
		})
	})
}

// dropMakeTargets removes named target blocks and their .PHONY entries.
func dropMakeTargets(path string, targets ...string) error {
	drop := make(map[string]bool, len(targets))
	for _, t := range targets {
		drop[t] = true
	}

	return transformLines(path, func(lines []string) []string {
		var out []string
		skip := false
		for _, line := range lines {
			if strings.HasPrefix(line, ".PHONY:") {
				parts := slices.DeleteFunc(strings.Fields(line), func(p string) bool {
					return p != ".PHONY:" && drop[p]
				})
				out = append(out, strings.Join(parts, " "))
				continue
			}

			if isTarget(line, drop) {
				skip = true
				continue
			}

			if skip && !strings.HasPrefix(line, "\t") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				skip = false
			}

			if !skip {
				out = append(out, line)
			}
		}
		return out
	})
}

func isTarget(line string, targets map[string]bool) bool {
	name, _, ok := strings.Cut(line, ":")
	return ok && targets[strings.TrimSpace(name)]
}

func transformLines(path string, fn func([]string) []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return os.WriteFile(path, []byte(strings.Join(fn(strings.Split(string(data), "\n")), "\n")), 0644)
}

func modTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
