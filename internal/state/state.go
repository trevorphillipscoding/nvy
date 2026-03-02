// Package state manages nvy's persistent state.
//
// global.json  — active global version per tool:  {"versions": {"go": "1.22.1"}}
// owners.json  — which tool owns each binary:     {"go": "go", "gofmt": "go", "node": "node"}
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trevorphillipscoding/nvy/internal/env"
)

// globalState is the in-memory representation of global.json.
type globalState struct {
	Versions map[string]string `json:"versions"`
}

// SetGlobal writes tool → version into global.json.
func SetGlobal(tool, version string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.Versions[tool] = version
	return save(s)
}

// GetGlobal returns the currently active global version for tool, and whether one is set.
func GetGlobal(tool string) (version string, ok bool) {
	s, err := load()
	if err != nil {
		return "", false
	}
	v, ok := s.Versions[tool]
	return v, ok
}

// AllGlobals returns the full tool → version map.
func AllGlobals() (map[string]string, error) {
	s, err := load()
	if err != nil {
		return nil, err
	}
	// Return a copy so callers can't mutate internal state.
	out := make(map[string]string, len(s.Versions))
	for k, v := range s.Versions {
		out[k] = v
	}
	return out, nil
}

// ── Owner tracking ───────────────────────────────────────────────────────────
// owners.json maps binary name → canonical plugin name.
// e.g. {"go": "go", "gofmt": "go", "node": "node", "npm": "node"}
// Updated each time "nvy global" is run.

type ownerState struct {
	Owners map[string]string `json:"owners"`
}

// RegisterShims records that each binary in binaries belongs to tool.
// Existing entries for other tools are preserved; only this tool's entries are updated.
func RegisterShims(tool string, binaries []string) error {
	s, err := loadOwners()
	if err != nil {
		return err
	}
	for _, b := range binaries {
		s.Owners[b] = tool
	}
	return saveOwners(s)
}

// LookupShim returns the tool name that owns binary, e.g. "npm" → "node".
func LookupShim(binary string) (tool string, ok bool) {
	s, err := loadOwners()
	if err != nil {
		return "", false
	}
	t, ok := s.Owners[binary]
	return t, ok
}

func ownersPath() string { return env.StateDir() + "/owners.json" }

func loadOwners() (*ownerState, error) {
	data, err := os.ReadFile(ownersPath())
	if errors.Is(err, os.ErrNotExist) {
		return &ownerState{Owners: map[string]string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading owners: %w", err)
	}
	var s ownerState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing owners: %w", err)
	}
	if s.Owners == nil {
		s.Owners = map[string]string{}
	}
	return &s, nil
}

func saveOwners(s *ownerState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ownersPath()), 0700); err != nil {
		return err
	}
	tmp := ownersPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, ownersPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ── Global versions ──────────────────────────────────────────────────────────

// load reads global.json. Returns an empty state if the file does not yet exist.
func load() (*globalState, error) {
	data, err := os.ReadFile(env.GlobalStatePath())
	if errors.Is(err, os.ErrNotExist) {
		return &globalState{Versions: map[string]string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var s globalState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	if s.Versions == nil {
		s.Versions = map[string]string{}
	}
	return &s, nil
}

// save writes global.json atomically using a temp file + rename.
func save(s *globalState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	path := env.GlobalStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	// Write to a temp file first, then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("committing state: %w", err)
	}
	return nil
}
