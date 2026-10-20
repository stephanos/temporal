// Package determinism holds the WASM backend's determinism inventory: one row
// for every influence on a WASM execution, with the disposition that keeps it
// out of bounded evidence and the tests that enforce that disposition.
//
// The inventory is a reviewed contract, not a proof that nondeterminism is
// absent. Completeness tests derive each site class from the code and fail
// when a site has no row or a row names a site that no longer exists.
package determinism

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Schema is the inventory format identifier.
const Schema = "gomad.wasm-determinism-inventory/v1"

//go:embed inventory.json
var inventoryJSON []byte

// Inventory is the decoded inventory file.
type Inventory struct {
	Schema   string   `json:"schema"`
	Reviewed Reviewed `json:"reviewed"`
	Entries  []Entry  `json:"entries"`
}

// Reviewed records the toolchain, engine and Rust versions the rows were
// reviewed against.
type Reviewed struct {
	GoToolchain string `json:"go_toolchain"`
	Wasmtime    string `json:"wasmtime"`
	Rust        string `json:"rust"`
}

// Entry is one influence on a WASM execution.
type Entry struct {
	ID            string   `json:"id"`
	Layer         string   `json:"layer"`
	Class         string   `json:"class"`
	Site          string   `json:"site"`
	Disposition   string   `json:"disposition"`
	Owner         string   `json:"owner"`
	Rule          string   `json:"rule"`
	Evidence      []string `json:"evidence"`
	Justification string   `json:"justification,omitempty"`
	ReadmeAnchor  string   `json:"readme_anchor,omitempty"`
}

var (
	layers       = []string{"guest-runtime", "guest-stdlib", "helper", "host-go", "engine-config", "host-process"}
	classes      = []string{"clock", "entropy", "scheduling", "gc-finalizer", "address", "fs-io", "network-io", "thread-order", "settings"}
	dispositions = []string{"controlled", "recorded", "rejected", "escape"}
	idPattern    = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9_]+)*$`)
)

// Load returns the checked-in inventory.
func Load() (Inventory, error) { return Parse(inventoryJSON) }

// Parse decodes and validates an inventory document. Unknown fields, a reused
// id, an unknown enum value or an escape without justification fail.
func Parse(data []byte) (Inventory, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var inventory Inventory
	if err := decoder.Decode(&inventory); err != nil {
		return Inventory{}, fmt.Errorf("decode inventory: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Inventory{}, errors.New("decode inventory: trailing data")
	}
	if inventory.Schema != Schema {
		return Inventory{}, fmt.Errorf("inventory schema %q, want %q", inventory.Schema, Schema)
	}
	if inventory.Reviewed.GoToolchain == "" || inventory.Reviewed.Wasmtime == "" || inventory.Reviewed.Rust == "" {
		return Inventory{}, errors.New("inventory reviewed versions incomplete")
	}
	if len(inventory.Entries) == 0 {
		return Inventory{}, errors.New("inventory has no entries")
	}
	ids := map[string]bool{}
	sites := map[string]string{}
	for index, entry := range inventory.Entries {
		if ids[entry.ID] {
			return Inventory{}, fmt.Errorf("inventory id %q reused", entry.ID)
		}
		ids[entry.ID] = true
		if previous, ok := sites[entry.Site]; ok {
			return Inventory{}, fmt.Errorf("inventory site %q has rows %q and %q", entry.Site, previous, entry.ID)
		}
		sites[entry.Site] = entry.ID
		if err := entry.validate(); err != nil {
			return Inventory{}, fmt.Errorf("inventory entry %d (%q): %w", index, entry.ID, err)
		}
	}
	return inventory, nil
}

func (entry Entry) validate() error {
	switch {
	case !idPattern.MatchString(entry.ID):
		return errors.New("id is not a stable lower-case identifier")
	case !slices.Contains(layers, entry.Layer):
		return fmt.Errorf("unknown layer %q", entry.Layer)
	case !slices.Contains(classes, entry.Class):
		return fmt.Errorf("unknown class %q", entry.Class)
	case !slices.Contains(dispositions, entry.Disposition):
		return fmt.Errorf("unknown disposition %q", entry.Disposition)
	case entry.Site == "" || entry.Owner == "" || entry.Rule == "":
		return errors.New("site, owner and rule are required")
	case len(entry.Evidence) == 0:
		return errors.New("row names no evidence test")
	}
	for _, reference := range entry.Evidence {
		if _, err := parseEvidence(reference); err != nil {
			return err
		}
	}
	if entry.Disposition == "escape" {
		if entry.Justification == "" || entry.ReadmeAnchor == "" {
			return errors.New("escape row needs justification and readme_anchor")
		}
	} else if entry.Justification != "" || entry.ReadmeAnchor != "" {
		return errors.New("justification and readme_anchor are only for escape rows")
	}
	return nil
}

// Sites returns the rows whose site has the given kind prefix ("import",
// "stream", ...), keyed by the site value after the prefix.
func (inventory Inventory) Sites(kind string) map[string]Entry {
	rows := map[string]Entry{}
	for _, entry := range inventory.Entries {
		if value, ok := strings.CutPrefix(entry.Site, kind+":"); ok {
			rows[value] = entry
		}
	}
	return rows
}

// Without returns a copy of the inventory without the row for site. Tests
// use it to prove that removing a row fails the completeness checks.
func (inventory Inventory) Without(site string) Inventory {
	inventory.Entries = slices.DeleteFunc(slices.Clone(inventory.Entries), func(entry Entry) bool { return entry.Site == site })
	return inventory
}

// Complete compares the derived sites of one kind with the rows of that kind
// and reports every site without a row and every row without a site, by name.
func (inventory Inventory) Complete(kind string, derived []string) error {
	return inventory.check(kind, derived, true)
}

// Covers reports every derived site without a row. It is for a table that
// holds a subset of a kind whose full set another check derives (the Go host
// accepts a subset of the helper's import table).
func (inventory Inventory) Covers(kind string, derived []string) error {
	return inventory.check(kind, derived, false)
}

func (inventory Inventory) check(kind string, derived []string, exact bool) error {
	rows := inventory.Sites(kind)
	var problems []error
	seen := map[string]bool{}
	for _, site := range derived {
		seen[site] = true
		if _, ok := rows[site]; !ok {
			problems = append(problems, fmt.Errorf("%s site %q has no inventory row", kind, site))
		}
	}
	for site, entry := range rows {
		if exact && !seen[site] {
			problems = append(problems, fmt.Errorf("inventory row %q names %s site %q that the code no longer has", entry.ID, kind, site))
		}
	}
	slices.SortFunc(problems, func(a, b error) int { return strings.Compare(a.Error(), b.Error()) })
	return errors.Join(problems...)
}

type evidence struct{ kind, path, name string }

// Evidence references name the test that enforces a row:
//
//	go:<package dir relative to tools/gomad>:<TestName>
//	rust:<file relative to wasmhost>:<test fn>
//	corpus:<file relative to the WASM backend directory>
func parseEvidence(reference string) (evidence, error) {
	kind, rest, _ := strings.Cut(reference, ":")
	switch kind {
	case "go", "rust":
		path, name, ok := strings.Cut(rest, ":")
		if !ok || path == "" || name == "" || filepath.IsAbs(path) || strings.Contains(path, "..") {
			return evidence{}, fmt.Errorf("evidence %q is not %s:<path>:<test>", reference, kind)
		}
		return evidence{kind, path, name}, nil
	case "corpus":
		if rest == "" || filepath.IsAbs(rest) || strings.Contains(rest, "..") {
			return evidence{}, fmt.Errorf("evidence %q is not corpus:<path>", reference)
		}
		return evidence{kind, rest, ""}, nil
	}
	return evidence{}, fmt.Errorf("evidence %q has unknown kind", reference)
}

// CheckEvidence verifies that every named evidence test exists. moduleRoot is
// the tools/gomad module directory.
func (inventory Inventory) CheckEvidence(moduleRoot string) error {
	wasmRoot := filepath.Join(moduleRoot, "internal", "backends", "wasm")
	files := map[string][]byte{}
	read := func(path string) ([]byte, error) {
		if data, ok := files[path]; ok {
			return data, nil
		}
		data, err := os.ReadFile(path)
		if err == nil {
			files[path] = data
		}
		return data, err
	}
	var problems []error
	for _, entry := range inventory.Entries {
		for _, reference := range entry.Evidence {
			ref, err := parseEvidence(reference)
			if err == nil {
				err = ref.check(wasmRoot, moduleRoot, read)
			}
			if err != nil {
				problems = append(problems, fmt.Errorf("row %q evidence %q: %w", entry.ID, reference, err))
			}
		}
	}
	return errors.Join(problems...)
}

func (ref evidence) check(wasmRoot, moduleRoot string, read func(string) ([]byte, error)) error {
	switch ref.kind {
	case "corpus":
		info, err := os.Stat(filepath.Join(wasmRoot, filepath.FromSlash(ref.path)))
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("corpus file does not exist")
		}
		return nil
	case "go":
		names, err := filepath.Glob(filepath.Join(moduleRoot, filepath.FromSlash(ref.path), "*_test.go"))
		if err != nil {
			return err
		}
		pattern := regexp.MustCompile(`(?m)^func ` + regexp.QuoteMeta(ref.name) + `\(t \*testing\.T\)`)
		for _, name := range names {
			data, err := read(name)
			if err != nil {
				return err
			}
			if pattern.Match(data) {
				return nil
			}
		}
		return errors.New("go test does not exist")
	default:
		data, err := read(filepath.Join(wasmRoot, "wasmhost", filepath.FromSlash(ref.path)))
		if err != nil {
			return errors.New("rust file does not exist")
		}
		pattern := regexp.MustCompile(`#\[test\]\s*(?:#\[[^\]]*\]\s*)*fn ` + regexp.QuoteMeta(ref.name) + `\(\)`)
		if !pattern.Match(data) {
			return errors.New("rust test does not exist")
		}
		return nil
	}
}
