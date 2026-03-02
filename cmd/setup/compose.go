package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type composeFile struct {
	Services map[string]composeService `yaml:"services,omitempty"`
	Networks map[string]composeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]composeVolume  `yaml:"volumes,omitempty"`
}

type composeService struct {
	Image       string                      `yaml:"image,omitempty"`
	Build       *composeBuild               `yaml:"build,omitempty"`
	Profiles    []string                    `yaml:"profiles,omitempty"`
	Environment map[string]string           `yaml:"environment,omitempty"`
	Ports       []string                    `yaml:"ports,omitempty"`
	EnvFile     []string                    `yaml:"env_file,omitempty"`
	Healthcheck *composeHealthcheck         `yaml:"healthcheck,omitempty"`
	DependsOn   map[string]composeDependsOn `yaml:"depends_on,omitempty"`
	Develop     *composeDevelop             `yaml:"develop,omitempty"`
}

type composeBuild struct {
	Context    string `yaml:"context,omitempty"`
	Target     string `yaml:"target,omitempty"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

type composeHealthcheck struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

type composeDependsOn struct {
	Condition string `yaml:"condition,omitempty"`
}

type composeDevelop struct {
	Watch []composeWatch `yaml:"watch,omitempty"`
}

type composeWatch struct {
	Action string   `yaml:"action,omitempty"`
	Path   string   `yaml:"path,omitempty"`
	Ignore []string `yaml:"ignore,omitempty"`
}

type composeNetwork struct {
	Driver string `yaml:"driver,omitempty"`
}

type composeVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

// removeComposeService removes a named service from a Docker Compose file.
// It is a no-op if the service does not exist.
func removeComposeService(path, service string) error {
	data, err := os.ReadFile(path)
	if err != nil {
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
