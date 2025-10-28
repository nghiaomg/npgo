package resolver

import (
	"fmt"
	"sort"
	"strings"

	"npgo/internal/packagejson"
	"npgo/internal/registry"
)

// Dependency represents a single dependency with its resolved version
type Dependency struct {
	Name         string
	Spec         string // Original version spec (e.g., "^1.0.0")
	Resolved     string // Resolved version (e.g., "1.2.3")
	TarballURL   string
	Dependencies map[string]*Dependency
}

// Resolver resolves dependencies
type Resolver struct {
	cache map[string]*Dependency
}

// NewResolver creates a new dependency resolver
func NewResolver() *Resolver {
	return &Resolver{
		cache: make(map[string]*Dependency),
	}
}

// ResolveDependencies resolves all dependencies from package.json
func (r *Resolver) ResolveDependencies(pkg *packagejson.PackageJSON) ([]*Dependency, error) {
	var deps []*Dependency

	// Process regular dependencies
	for name, spec := range pkg.Dependencies {
		dep, err := r.resolveDependency(name, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", name, err)
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

// ResolveDevDependencies resolves dev dependencies
func (r *Resolver) ResolveDevDependencies(pkg *packagejson.PackageJSON, include bool) ([]*Dependency, error) {
	if !include {
		return []*Dependency{}, nil
	}

	var deps []*Dependency

	// Process dev dependencies
	for name, spec := range pkg.DevDependencies {
		dep, err := r.resolveDependency(name, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dev dependency %s: %w", name, err)
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

// resolveDependency resolves a single dependency
func (r *Resolver) resolveDependency(name, spec string) (*Dependency, error) {
	// Check cache
	if cached, exists := r.cache[name+"@"+spec]; exists {
		return cached, nil
	}

	// Normalize version spec
	version := normalizeVersion(spec)

	// Fetch metadata from registry
	metadata, err := registry.FetchMetadata(name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	dep := &Dependency{
		Name:         name,
		Spec:         spec,
		Resolved:     metadata.Version,
		TarballURL:   metadata.TarballURL,
		Dependencies: make(map[string]*Dependency),
	}

	// Cache the dependency
	r.cache[name+"@"+spec] = dep

	return dep, nil
}

// normalizeVersion normalizes version specs to be registry-compatible
func normalizeVersion(spec string) string {
	// Remove prefix characters
	spec = strings.TrimPrefix(spec, "^")
	spec = strings.TrimPrefix(spec, "~")
	spec = strings.TrimPrefix(spec, ">=")
	spec = strings.TrimPrefix(spec, "<=")
	spec = strings.TrimPrefix(spec, ">")
	spec = strings.TrimPrefix(spec, "<")

	// Handle range specs
	if strings.Contains(spec, " ") {
		parts := strings.Fields(spec)
		// Take the first version in the range
		spec = parts[0]
	}

	// Handle "latest" or empty
	if spec == "" || spec == "*" || spec == "latest" {
		return "latest"
	}

	return spec
}

// GetAllDependencies returns all dependencies in a flat list
func (r *Resolver) GetAllDependencies() []*Dependency {
	var deps []*Dependency
	for _, dep := range r.cache {
		deps = append(deps, dep)
	}

	// Sort by name for consistent output
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})

	return deps
}
