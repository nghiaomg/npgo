package packagejson

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
	Scripts         map[string]string `json:"scripts,omitempty"`
	Private         bool              `json:"private,omitempty"`
	Workspaces      interface{}       `json:"workspaces,omitempty"`
}

func Read(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &pkg, nil
}

func Write(path string, pkg *PackageJSON) error {
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	return nil
}

func (p *PackageJSON) GetDependencies() map[string]string {
	deps := make(map[string]string)

	for name, version := range p.Dependencies {
		deps[name] = version
	}

	for name, version := range p.DevDependencies {
		deps[name] = version
	}

	return deps
}

func (p *PackageJSON) HasDependencies() bool {
	return len(p.Dependencies) > 0 || len(p.DevDependencies) > 0
}
