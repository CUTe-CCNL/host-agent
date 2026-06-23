package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

var allowedMethods = map[string]struct{}{
	"GET":    {},
	"POST":   {},
	"PUT":    {},
	"PATCH":  {},
	"DELETE": {},
}

type Route struct {
	PathPrefix string   `yaml:"path_prefix" json:"path_prefix"`
	Methods    []string `yaml:"methods" json:"methods"`
}

type Manifest struct {
	ID          string            `yaml:"id" json:"id"`
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Description string            `yaml:"description" json:"description,omitempty"`
	Command     string            `yaml:"command" json:"command"`
	Args        []string          `yaml:"args" json:"args,omitempty"`
	WorkingDir  string            `yaml:"working_dir" json:"working_dir,omitempty"`
	Env         map[string]string `yaml:"env" json:"env,omitempty"`
	Routes      []Route           `yaml:"routes" json:"routes"`
}

func LoadManifests(directory string) ([]Manifest, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	manifests := make([]Manifest, 0)
	seen := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(directory, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read plugin manifest %s: %w", path, err)
		}

		var manifest Manifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse plugin manifest %s: %w", path, err)
		}
		if err := validateManifest(&manifest); err != nil {
			return nil, fmt.Errorf("validate plugin manifest %s: %w", path, err)
		}

		if previous, exists := seen[manifest.ID]; exists {
			return nil, fmt.Errorf("duplicate plugin id %q in %s and %s", manifest.ID, previous, path)
		}
		seen[manifest.ID] = path

		manifests = append(manifests, manifest)
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].ID < manifests[j].ID
	})

	return manifests, nil
}

func validateManifest(manifest *Manifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("id is required")
	}
	if !pluginIDPattern.MatchString(manifest.ID) {
		return fmt.Errorf("id %q must be a slug containing lowercase letters, numbers, and hyphens", manifest.ID)
	}
	if manifest.Name == "" {
		return fmt.Errorf("name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("version is required")
	}
	if manifest.Command == "" {
		return fmt.Errorf("command is required")
	}
	if len(manifest.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}

	for routeIndex := range manifest.Routes {
		route := &manifest.Routes[routeIndex]
		if route.PathPrefix == "" {
			return fmt.Errorf("route path_prefix is required")
		}
		if !strings.HasPrefix(route.PathPrefix, "/") {
			return fmt.Errorf("route path_prefix %q must start with /", route.PathPrefix)
		}
		if len(route.Methods) == 0 {
			return fmt.Errorf("route %q must allow at least one method", route.PathPrefix)
		}
		for methodIndex, method := range route.Methods {
			normalized := strings.ToUpper(method)
			if _, ok := allowedMethods[normalized]; !ok {
				return fmt.Errorf("method %q is not allowed", method)
			}
			route.Methods[methodIndex] = normalized
		}
	}

	return nil
}
