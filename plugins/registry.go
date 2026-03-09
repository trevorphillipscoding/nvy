package plugins

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Plugin{}
)

// Register adds a plugin to the global registry under its canonical name and all aliases.
// Plugins call this from their init() functions; see plugins/all/all.go for the import list.
// Panics if a name or alias is already registered (caught at startup, not runtime).
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()

	names := append([]string{p.Name()}, p.Aliases()...)
	for _, name := range names {
		if _, exists := registry[name]; exists {
			panic(fmt.Sprintf("nvy: plugin name conflict: %q already registered", name))
		}
		registry[name] = p
	}
}

// Get returns the plugin for name (canonical name or alias), or an error listing what is available.
func Get(name string) (Plugin, error) {
	mu.RLock()
	defer mu.RUnlock()

	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime %q — available: %s", name, availableNames())
	}
	return p, nil
}

// All returns one entry per unique plugin, sorted by canonical name.
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	return uniqueSorted()
}

// availableNames returns a sorted, comma-separated list of canonical plugin names.
// Must be called with mu held (e.g. from Get).
func availableNames() string {
	all := uniqueSorted()
	names := make([]string, len(all))
	for i, p := range all {
		names[i] = p.Name()
	}
	return strings.Join(names, ", ")
}

// uniqueSorted returns one entry per unique plugin sorted by canonical name.
// Caller must hold mu (read or write).
func uniqueSorted() []Plugin {
	seen := map[string]bool{}
	var out []Plugin
	for _, p := range registry {
		if !seen[p.Name()] {
			seen[p.Name()] = true
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
