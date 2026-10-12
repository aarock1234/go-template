// Setup configures a freshly scaffolded project by removing the features it
// does not need, then deletes itself.
//
// Without -y, an interactive form asks which features to keep, starting from
// the flag values. With -y, the flags alone decide:
//
//	go run ./cmd/setup
//	go run ./cmd/setup -y -server=false -packages=worker,retry
//	make setup ARGS="-y -postgres-hosting=external"
//
// The flags are:
//
//	-y                 apply without prompting
//	-server            keep the HTTP server (default true)
//	-client            keep the HTTP client (default true)
//	-docker            keep the Docker files (default true)
//	-postgres          keep PostgreSQL support (default true)
//	-postgres-hosting  PostgreSQL hosting: docker or external (default docker)
//	-packages          comma-separated names of optional packages to keep:
//	                   worker, retry, state, cycle, fake, ptr (default all)
//
// At least one of the server or the client must be kept.
//
// Project files mark feature-specific code with comments. A block runs from
// a full-line "# [tag]" or "// [tag]" to the matching "[/tag]", and blocks
// may nest. A trailing comment made only of tags, as in
// `"net/http" // [server]`, marks a single line, which is removed when any of
// its tags is. After removing the sections of disabled features, setup strips
// the remaining markers and runs go mod tidy and go fmt.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
)

// errUserAborted reports that the user quit the form without answering.
var errUserAborted = errors.New("user aborted")

func main() {
	if err := run(); err != nil {
		if errors.Is(err, errUserAborted) {
			os.Exit(130)
		}

		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run collects the configuration from flags or the form and applies it.
func run() error {
	c, yes, err := parseFlags(os.Args[1:])
	if err != nil {
		return err
	}

	if !yes {
		confirmed, err := runForm(&c)
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("setup cancelled")

			return nil
		}
	}

	if err := c.validate(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return apply(ctx, c)
}

// parseFlags parses the command line into the initial configuration and
// reports whether -y was given, in which case the configuration is final.
// Invalid flags print usage and exit the process.
func parseFlags(args []string) (c config, yes bool, err error) {
	c.packages = packageNames()

	flags := flag.NewFlagSet("setup", flag.ExitOnError)
	flags.BoolVar(&yes, "y", false, "apply without prompting")
	flags.BoolVar(&c.server, "server", true, "keep the HTTP server")
	flags.BoolVar(&c.client, "client", true, "keep the HTTP client")
	flags.BoolVar(&c.docker, "docker", true, "keep the Docker files")
	flags.BoolVar(&c.postgres, "postgres", true, "keep PostgreSQL support")
	flags.TextVar(&c.pgHosting, "postgres-hosting", hostingDocker, "PostgreSQL `hosting`: docker or external")

	// The backquoted word names the flag's argument in the usage message.
	packagesUsage := fmt.Sprintf("comma-separated `names` of optional packages to keep: %s (default all)", strings.Join(packageNames(), ","))
	flags.Func("packages", packagesUsage, func(list string) error {
		names, err := parsePackages(list)
		if err != nil {
			return err
		}

		c.packages = names

		return nil
	})

	if err := flags.Parse(args); err != nil {
		return config{}, false, err
	}

	if flags.NArg() > 0 {
		return config{}, false, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	return c, yes, nil
}

// parsePackages splits a comma-separated list of optional package names. It
// rejects names missing from optionalPackages. An empty list keeps none.
func parsePackages(list string) ([]string, error) {
	known := packageNames()

	var names []string
	for name := range strings.SplitSeq(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if !slices.Contains(known, name) {
			return nil, fmt.Errorf("unknown package %q, want any of %s", name, strings.Join(known, ","))
		}

		names = append(names, name)
	}

	return names, nil
}

// apply removes everything c disables, strips the remaining markers, deletes
// the setup command, and tidies the module. Malformed markers abort before
// anything changes. A failed file or directory change does not stop the
// remaining steps but makes apply return an error. Failures of go mod tidy
// and go fmt are only warnings.
func apply(ctx context.Context, c config) error {
	removal := c.removal()

	files, err := loadMarkedFiles(markedFiles)
	if err != nil {
		return err
	}

	// Rewrite before removing paths so that a marked file being deleted,
	// such as compose.yaml, is not recreated.
	changeErr := errors.Join(
		rewriteFiles(files, removal.tags),
		removePaths(removal.paths),
	)

	warn(runGo(ctx, "mod", "tidy"))
	warn(runGo(ctx, "fmt", "./..."))

	if changeErr != nil {
		return fmt.Errorf("setup incomplete: %w", changeErr)
	}

	fmt.Println("\nsetup complete")

	return nil
}

// runGo runs the go command with args, streaming its output. Canceling ctx
// stops the command.
func runGo(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

// warn prints a non-fatal error to stderr.
func warn(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
}
