package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/ir"
)

// AcceptedSuffix ends the checked-in file of accepted findings beside an IR file, `<file>.lint.json`,
// which the reader leaves out of a directory's IR files. The lifter writes nothing there: an author
// does, with a reason for each acceptance.
const AcceptedSuffix = ir.AcceptedSuffix

// AcceptedPath is the file of accepted findings beside the IR file at irPath.
func AcceptedPath(irPath string) string { return strings.TrimSuffix(irPath, ".json") + AcceptedSuffix }

// Accepted is the findings of one IR file an author accepts. Each acceptance names a kind, the owner
// and the subjects it accepts there, and why. A finding is accepted by its kind, owner and subject
// alone, never its message or position, so moving a line leaves it accepted; an acceptance that
// matches no finding is stale, and the gate fails on it as on a finding no acceptance matches.
// An acceptance names any kind lint reports, so a kind added later records its accepted findings, and
// their reasons, the same way.
type Accepted struct {
	Accepted []Acceptance `json:"accepted"`
}

// Acceptance is one reason, and the findings of one kind and owner it accepts.
type Acceptance struct {
	Kind     Kind     `json:"kind"`
	Owner    string   `json:"owner"`
	Subjects []string `json:"subjects"`
	Because  string   `json:"because"`
}

// ReadAccepted reads the accepted findings beside the IR file at irPath, or none where there is no
// such file. An acceptance with no reason, no subject, an unknown kind or a kind no reason excuses
// (LawWaivedWithoutReason, ReasonNamesNoLaw), and a finding accepted twice, are errors of the file.
func ReadAccepted(irPath string) (*Accepted, error) {
	path := AcceptedPath(irPath)
	encoded, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Accepted{}, nil
	}
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var a Accepted
	if err := decoder.Decode(&a); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := a.check(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &a, nil
}

func (a *Accepted) check() error {
	var problems []error
	seen := map[[3]string]bool{}
	for i, x := range a.Accepted {
		switch {
		case !slices.Contains(order, x.Kind):
			problems = append(problems, fmt.Errorf("acceptance %d names no kind lint reports: %q", i, x.Kind))
		case x.Owner == "":
			problems = append(problems, fmt.Errorf("acceptance %d of %s names no owner", i, x.Kind))
		case strings.TrimSpace(x.Because) == "":
			problems = append(problems, fmt.Errorf("acceptance %d of %s %s gives no reason", i, x.Kind, x.Owner))
		case len(x.Subjects) == 0:
			problems = append(problems, fmt.Errorf("acceptance %d of %s %s accepts nothing", i, x.Kind, x.Owner))
		case x.Kind == LawWaivedWithoutReason || x.Kind == ReasonNamesNoLaw:
			// A waiver with no reason, or of a law no capability brings, excuses nothing: it is fixed
			// in the declaration, never accepted.
			problems = append(problems, fmt.Errorf("acceptance %d accepts %s, which is fixed in the declaration, not accepted", i, x.Kind))
		default:
		}
		for _, s := range x.Subjects {
			key := [3]string{string(x.Kind), x.Owner, s}
			if seen[key] {
				problems = append(problems, fmt.Errorf("%s %s %q is accepted twice", x.Kind, x.Owner, s))
			}
			seen[key] = true
		}
	}
	return errors.Join(problems...)
}

// Verdict is what the accepted findings make of a run's findings: those accepted, with their reason,
// those not, and each accepted subject no finding matches.
type Verdict struct {
	Accepted   []Judged
	Unaccepted []Finding
	Stale      []string
}

// Judged is a finding and the reason it is accepted.
type Judged struct {
	Finding
	Because string
}

// Failed is whether the gate fails on the verdict: on a finding nothing accepts or an acceptance
// that matches nothing. A count is never a reason to fail.
func (v Verdict) Failed() bool { return len(v.Unaccepted) > 0 || len(v.Stale) > 0 }

// Judge matches findings against the accepted ones.
func (a *Accepted) Judge(findings []Finding) Verdict {
	because := map[[3]string]string{}
	for _, x := range a.Accepted {
		for _, s := range x.Subjects {
			because[[3]string{string(x.Kind), x.Owner, s}] = x.Because
		}
	}
	var v Verdict
	matched := map[[3]string]bool{}
	for _, f := range findings {
		key := [3]string{string(f.Kind), f.Owner, f.Subject}
		if reason, ok := because[key]; ok {
			matched[key] = true
			v.Accepted = append(v.Accepted, Judged{Finding: f, Because: reason})
			continue
		}
		v.Unaccepted = append(v.Unaccepted, f)
	}
	for _, x := range a.Accepted {
		for _, s := range x.Subjects {
			if !matched[[3]string{string(x.Kind), x.Owner, s}] {
				v.Stale = append(v.Stale, fmt.Sprintf("%s %s %q matches no finding", x.Kind, x.Owner, s))
			}
		}
	}
	return v
}

// Encode is the accepted findings as their file holds them: indented JSON, ending in a newline,
// with no character escaped that a reader would not escape.
func (a *Accepted) Encode() ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(a); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
