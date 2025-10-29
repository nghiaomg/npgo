package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"npgo/internal/packagejson"
	"npgo/internal/registry"
	"npgo/internal/ui"
)

type Dependency struct {
	Name         string
	Spec         string
	Resolved     string
	TarballURL   string
	Dependencies map[string]*Dependency
	RawDeps      map[string]string
}

type Resolver struct {
	cache       map[string]*Dependency
	debug       bool
	concurrency int
	onProgress  func(string)
}

func NewResolver() *Resolver {
	return &Resolver{cache: make(map[string]*Dependency), debug: false, concurrency: 32}
}

func NewResolverWithDebug(debug bool) *Resolver {
	return &Resolver{cache: make(map[string]*Dependency), debug: debug, concurrency: 32}
}

func NewResolverWithOptions(debug bool, concurrency int, onProgress func(string)) *Resolver {
	if concurrency <= 0 {
		concurrency = 32
	}
	return &Resolver{cache: make(map[string]*Dependency), debug: debug, concurrency: concurrency, onProgress: onProgress}
}

func (r *Resolver) ResolveDependencies(pkg *packagejson.PackageJSON) ([]*Dependency, error) {
	var deps []*Dependency

	for name, spec := range pkg.Dependencies {
		dep, err := r.resolveDependency(name, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", name, err)
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

func (r *Resolver) ResolveDevDependencies(pkg *packagejson.PackageJSON, include bool) ([]*Dependency, error) {
	if !include {
		return []*Dependency{}, nil
	}

	var deps []*Dependency

	for name, spec := range pkg.DevDependencies {
		dep, err := r.resolveDependency(name, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dev dependency %s: %w", name, err)
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

func (r *Resolver) resolveDependency(name, spec string) (*Dependency, error) {
	if cached, exists := r.cache[name+"@"+spec]; exists {
		return cached, nil
	}

	version := normalizeVersion(spec)
	if r.debug {
		ui.InstallStep("🧭", fmt.Sprintf("Resolving %s (spec: %s → %s)", name, spec, version))
	}

	metadata, err := r.getMetadataCached(name, version)
	if err != nil {
		if r.debug {
			ui.ErrorMessage(fmt.Errorf("resolve failed %s@%s: %v", name, version, err))
		}
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	// merge dependencies + optionalDependencies
	raw := make(map[string]string)
	if metadata.Dependencies != nil {
		for k, v := range metadata.Dependencies {
			raw[k] = v
		}
	}
	if metadata.OptionalDependencies != nil {
		for k, v := range metadata.OptionalDependencies {
			raw[k] = v
		}
	}

	dep := &Dependency{
		Name:         name,
		Spec:         spec,
		Resolved:     metadata.Version,
		TarballURL:   metadata.TarballURL,
		Dependencies: make(map[string]*Dependency),
		RawDeps:      raw,
	}

	r.cache[name+"@"+spec] = dep

	return dep, nil
}

func normalizeVersion(spec string) string {
	spec = strings.TrimPrefix(spec, "^")
	spec = strings.TrimPrefix(spec, "~")
	spec = strings.TrimPrefix(spec, ">=")
	spec = strings.TrimPrefix(spec, "<=")
	spec = strings.TrimPrefix(spec, ">")
	spec = strings.TrimPrefix(spec, "<")

	// lite normalization to increase cache hit
	if spec == "1.x" || spec == "1.*" {
		return "1"
	}
	if strings.HasSuffix(spec, ".x") || strings.HasSuffix(spec, ".*") {
		spec = strings.TrimSuffix(strings.TrimSuffix(spec, ".x"), ".*")
	}

	if strings.Contains(spec, " ") {
		parts := strings.Fields(spec)
		spec = parts[0]
	}

	if spec == "" || spec == "*" || spec == "latest" {
		return "latest"
	}

	return spec
}

// per-version metadata cache under ~/.npgo/registry-cache/versions
func (r *Resolver) getMetadataCached(name, version string) (*registry.PackageMetadata, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".npgo", "registry-cache", "versions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	safe := strings.ReplaceAll(strings.ReplaceAll(name, "/", "-"), "\\", "-")
	p := filepath.Join(dir, fmt.Sprintf("%s@%s.json", safe, version))
	if b, err := os.ReadFile(p); err == nil {
		var md registry.PackageMetadata
		if json.Unmarshal(b, &md) == nil && md.Version != "" {
			return &md, nil
		}
	}
	md, err := registry.FetchMetadata(name, version)
	if err != nil {
		return nil, err
	}
	if data, err := json.MarshalIndent(md, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0644)
	}
	return md, nil
}

func (r *Resolver) GetAllDependencies() []*Dependency {
	var deps []*Dependency
	for _, dep := range r.cache {
		deps = append(deps, dep)
	}

	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})

	return deps
}

func (r *Resolver) BuildGraph(root map[string]string) (map[string]*Dependency, error) {
	graph := make(map[string]*Dependency)
	seen := sync.Map{}
	sem := make(chan struct{}, r.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var visit func(name, spec string)
	visit = func(name, spec string) {
		key := name + "@" + spec
		if _, loaded := seen.LoadOrStore(key, true); loaded {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			dep, err := r.resolveDependency(name, spec)
			<-sem
			if err != nil {
				if r.debug {
					ui.ErrorMessage(fmt.Errorf("resolve %s: %v", key, err))
				}
				return
			}
			if r.onProgress != nil {
				r.onProgress(name + "@" + dep.Resolved)
			}
			mu.Lock()
			graph[name+"@"+dep.Resolved] = dep
			mu.Unlock()
			for cn, cs := range dep.RawDeps {
				visit(cn, cs)
			}
		}()
	}
	for n, s := range root {
		visit(n, s)
	}
	wg.Wait()
	return graph, nil
}

func TopoOrder(graph map[string]*Dependency) ([]*Dependency, error) {
	indeg := make(map[*Dependency]int)
	children := make(map[*Dependency][]*Dependency)
	nodes := make([]*Dependency, 0, len(graph))
	for _, d := range graph {
		nodes = append(nodes, d)
		if d.Dependencies != nil {
			for _, c := range d.Dependencies {
				indeg[c]++
				children[d] = append(children[d], c)
			}
		}
	}
	q := make([]*Dependency, 0)
	for _, d := range nodes {
		if indeg[d] == 0 {
			q = append(q, d)
		}
	}
	order := make([]*Dependency, 0, len(nodes))
	for len(q) > 0 {
		d := q[0]
		q = q[1:]
		order = append(order, d)
		for _, c := range children[d] {
			indeg[c]--
			if indeg[c] == 0 {
				q = append(q, c)
			}
		}
	}
	seen := make(map[*Dependency]bool)
	uniq := make([]*Dependency, 0, len(nodes))
	for _, d := range order {
		if !seen[d] {
			seen[d] = true
			uniq = append(uniq, d)
		}
	}
	// If there is a cycle, append remaining nodes deterministically instead of failing
	if len(uniq) != len(nodes) {
		for _, d := range nodes {
			if !seen[d] {
				seen[d] = true
				uniq = append(uniq, d)
			}
		}
	}
	return uniq, nil
}
