package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
)

// templateModule is the module path of the template repository. Scaffolding
// with gonew rewrites go.mod and import paths but leaves string literals
// alone, so this still names the template in a scaffolded project. That is
// what lets setup find the places that must name the project instead.
const templateModule = "github.com/aarock1234/go-template"

// fileGoMod declares the project's module path on its module line.
const fileGoMod = "go.mod"

// moduleFiles lists the files outside Go sources that name the module path.
// Scaffolding leaves them pointing at the template, so setup points them at
// the project. Add a file here when it hardcodes the module path.
var moduleFiles = []string{fileGolangci}

// adoptModulePath points every file in moduleFiles at the project's module
// path. It changes nothing when the project is the template itself. It
// returns an error if the module path cannot be read or a file cannot be
// rewritten.
func adoptModulePath() error {
	module, err := readModulePath()
	if err != nil {
		return err
	}

	if module == templateModule {
		return nil
	}

	return replaceModulePath(moduleFiles, templateModule, module)
}

// readModulePath returns the module path declared by go.mod in the working
// directory, taken from the first line that starts with "module ". It returns
// an error if the file cannot be read or has no such line.
func readModulePath() (string, error) {
	data, err := os.ReadFile(fileGoMod)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", fileGoMod, err)
	}

	for line := range strings.Lines(string(data)) {
		if module, found := strings.CutPrefix(line, "module "); found {
			return strings.TrimSpace(module), nil
		}
	}

	return "", fmt.Errorf("read %s: no module line", fileGoMod)
}

// replaceModulePath replaces every occurrence of the module path from with to
// in each file in paths, leaving all other bytes as they are. Individual
// failures are joined.
func replaceModulePath(paths []string, from, to string) error {
	var errs []error

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", path, err))

			continue
		}

		data = bytes.ReplaceAll(data, []byte(from), []byte(to))

		if err := os.WriteFile(path, data, 0o644); err != nil {
			errs = append(errs, fmt.Errorf("write %s: %w", path, err))
		}
	}

	return errors.Join(errs...)
}
