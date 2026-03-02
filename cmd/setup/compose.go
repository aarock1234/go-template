package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// composeFile is the minimal representation needed to remove services.
type composeFile struct {
	Services map[string]yaml.Node `yaml:"services,omitempty"`
}

// removeComposeService removes a named service from a Docker Compose file.
// It is a no-op if the service does not exist.
func removeComposeService(path, service string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read compose file: %w", err)
	}

	var compose composeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("parse compose file: %w", err)
	}

	if _, exists := compose.Services[service]; !exists {
		return nil
	}

	delete(compose.Services, service)

	out, err := yaml.Marshal(&compose)
	if err != nil {
		return fmt.Errorf("marshal compose file: %w", err)
	}

	return os.WriteFile(path, out, 0644)
}
