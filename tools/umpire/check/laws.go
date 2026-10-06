package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/ir"
)

// LawSidecar is what the lifter writes beside an IR file whose Models declare capabilities,
// `<file>.laws.json`: each claim it generated from a law, with the bindings it was expanded from,
// each law a declaration waives, with its reason, and the catalog's laws the declarations read.
type LawSidecar struct {
	Claims  []LawClaim  `json:"claims"`
	Waivers []LawWaiver `json:"waivers"`
	Catalog []LawEntry  `json:"catalog"`
}

// LawClaim is one generated claim: `<machine>.<law>`, the law and capabilities that brought it, the
// action class each action field of those capabilities names, keyed `<capability>.<field>` and
// spelled as a Class's Key, what each of the law's parameters is bound to, and the server code each
// cited binding names.
type LawClaim struct {
	Machine      string              `json:"machine"`
	Name         string              `json:"name"`
	Law          string              `json:"law"`
	Capabilities []string            `json:"capabilities"`
	Actions      map[string]string   `json:"actions"`
	Bindings     map[string]string   `json:"bindings"`
	Cites        map[string][]string `json:"cites"`
	OverriddenBy string              `json:"overriddenBy"`
	Position     string              `json:"position"`
}

// LawWaiver is one law a declaration waives: lifted not at all (`except`) or as the entity's own def
// (`overriding`, by that def), for the reason it gives, at the line that says so.
type LawWaiver struct {
	Machine  string `json:"machine"`
	Law      string `json:"law"`
	Waiver   string `json:"waiver"`
	By       string `json:"by"`
	Because  string `json:"because"`
	Position string `json:"position"`
}

// The two ways a declaration waives a law.
const (
	WaiverExcept     = "except"
	WaiverOverriding = "overriding"
)

// LawEntry is one law of the catalog the declarations of the file read: what it says, the parameters
// each instance backs with a server citation, its instantiating machines in the file, one per state
// type, and where the catalog brings it.
type LawEntry struct {
	Law            string        `json:"law"`
	Capabilities   []string      `json:"capabilities"`
	Cites          []string      `json:"cites"`
	Promises       string        `json:"promises"`
	DoesNotPromise string        `json:"doesNotPromise"`
	Parameters     []string      `json:"parameters"`
	Instantiating  []LawInstance `json:"instantiating"`
	Position       string        `json:"position"`
}

// LawInstance is an instantiating entity of a law: a machine with its own state type that declares
// the capabilities bringing it. A composition and a machine derived from another are none.
type LawInstance struct {
	Machine string `json:"machine"`
	State   string `json:"state"`
}

// LawSidecarPath is the law sidecar beside the IR file at irPath.
func LawSidecarPath(irPath string) string {
	return strings.TrimSuffix(irPath, ".json") + ir.LawSidecarSuffix
}

// ReadLawSidecar reads the law sidecar beside the IR file at irPath, or nil where there is none. A
// field the sidecar does not have, a claim, waiver or law missing what identifies it, and a claim of
// a law its catalog does not list, are errors of the file. A waiver's reason and law are not: an
// empty reason, and a law the catalog does not bring, are what lint reports.
func ReadLawSidecar(irPath string) (*LawSidecar, error) {
	path := LawSidecarPath(irPath)
	encoded, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var s LawSidecar
	if err := decoder.Decode(&s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := s.check(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &s, nil
}

func (s *LawSidecar) check() error {
	var problems []error
	laws := map[string]bool{}
	for i, e := range s.Catalog {
		switch {
		case e.Law == "":
			problems = append(problems, fmt.Errorf("catalog law %d has no name", i))
		case laws[e.Law]:
			problems = append(problems, fmt.Errorf("catalog lists %s twice", e.Law))
		case e.Position == "":
			problems = append(problems, fmt.Errorf("catalog law %s has no position", e.Law))
		default:
		}
		laws[e.Law] = true
		for _, x := range e.Instantiating {
			if x.Machine == "" || x.State == "" {
				problems = append(problems, fmt.Errorf("catalog law %s lists an instantiating machine without its name or state type", e.Law))
			}
		}
	}
	for i, c := range s.Claims {
		switch {
		case c.Machine == "" || c.Law == "":
			problems = append(problems, fmt.Errorf("claim %d names no machine or law", i))
		case c.Name != c.Machine+"."+c.Law:
			problems = append(problems, fmt.Errorf("claim %q of %s's %s is not named <machine>.<law>", c.Name, c.Machine, c.Law))
		case !laws[c.Law]:
			problems = append(problems, fmt.Errorf("claim %s is of %s, which the catalog does not list", c.Name, c.Law))
		case c.Position == "":
			problems = append(problems, fmt.Errorf("claim %s has no position", c.Name))
		default:
		}
	}
	waived := map[[2]string]bool{}
	for i, w := range s.Waivers {
		key := [2]string{w.Machine, w.Law}
		if waived[key] {
			problems = append(problems, fmt.Errorf("%s.%s is waived twice", w.Machine, w.Law))
		}
		waived[key] = true
		switch {
		case w.Machine == "" || w.Law == "":
			problems = append(problems, fmt.Errorf("waiver %d names no machine or law", i))
		case w.Waiver != WaiverExcept && w.Waiver != WaiverOverriding:
			problems = append(problems, fmt.Errorf("waiver of %s.%s is %q, neither %s nor %s", w.Machine, w.Law, w.Waiver, WaiverExcept, WaiverOverriding))
		case (w.Waiver == WaiverOverriding) != (w.By != ""):
			problems = append(problems, fmt.Errorf("waiver of %s.%s names the def that overrides it only when it is %s", w.Machine, w.Law, WaiverOverriding))
		case w.Position == "":
			problems = append(problems, fmt.Errorf("waiver of %s.%s has no position", w.Machine, w.Law))
		default:
		}
	}
	return errors.Join(problems...)
}

// Entry is the catalog's law named law, or nil where the catalog does not list it.
func (s *LawSidecar) Entry(law string) *LawEntry {
	if i := slices.IndexFunc(s.Catalog, func(e LawEntry) bool { return e.Law == law }); i >= 0 {
		return &s.Catalog[i]
	}
	return nil
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
