package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
)

// LawSidecar is what the lifter writes beside an IR file whose Models declare capabilities,
// `<file>.laws.json`: each claim it generated from a law, with the bindings it was expanded from.
type LawSidecar struct {
	Claims []LawClaim `json:"claims"`
}

// LawClaim is one generated claim: `<machine>.<law>`, the law and capabilities that brought it, and
// what each of the law's parameters is bound to.
type LawClaim struct {
	Machine      string            `json:"machine"`
	Name         string            `json:"name"`
	Law          string            `json:"law"`
	Capabilities []string          `json:"capabilities"`
	Bindings     map[string]string `json:"bindings"`
	OverriddenBy string            `json:"overriddenBy"`
	Position     string            `json:"position"`
}

// ReadLawSidecar reads the law sidecar beside the IR file at irPath, or nil where there is none.
func ReadLawSidecar(irPath string) (*LawSidecar, error) {
	encoded, err := os.ReadFile(strings.TrimSuffix(irPath, ".json") + LawSidecarSuffix)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s LawSidecar
	if err := json.Unmarshal(encoded, &s); err != nil {
		return nil, fmt.Errorf("law sidecar of %s: %w", irPath, err)
	}
	return &s, nil
}

// LawViolations names each generated claim a report finds violated: the Query, the law it breaks,
// the capabilities that brought it and their bindings, and the violation, so a Model that breaks a
// law is rejected with the law's name and the entity's binding.
func LawViolations(r *Report, s *LawSidecar) []string {
	if s == nil {
		return nil
	}
	var out []string
	for _, rc := range r.Receipts {
		if rc.Subject != QuerySubject || rc.Kind != Counterexample {
			continue
		}
		i := slices.IndexFunc(s.Claims, func(c LawClaim) bool {
			return c.Machine == rc.Property.Owner && c.Name == rc.Key.Name && c.Name == rc.Property.Name
		})
		if i < 0 {
			continue
		}
		c := s.Claims[i]
		bindings := make([]string, 0, len(c.Bindings))
		for _, k := range slices.Sorted(maps.Keys(c.Bindings)) {
			bindings = append(bindings, k+" = "+c.Bindings[k])
		}
		out = append(out, fmt.Sprintf("%s breaks the law %s of %s (%s): %s",
			c.Name, c.Law, strings.Join(c.Capabilities, " and "), strings.Join(bindings, ", "), rc.Explanation))
	}
	return out
}
