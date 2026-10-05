package golden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// Since fn-124.5 a Query's expected Run declares what the baseline left to its reader: the Contract's
// Verdict (the baseline wrote it only when violated), how the Run ends and how its cleanup ends (the
// reader took a violated Contract to stop the Run, and any other to complete it, and every cleanup to
// succeed), and each reason by the judge's id rather than its prose. The archive is frozen in the
// schema before that, which the current one cannot decode, so it is read as the baseline's reader read
// it (DeclaredRuns.Declare): each expected Run of an archived IR file and lifter fixture, and of the
// Case manifest, with those values written out. Everything else is read as frozen, and is compared as
// before; the declared values are compared too, against what the baseline's reader made of it.

// DeclaredRuns are the reasons the baseline's expected Runs wrote as prose, each with the IR reason it
// names: "declared_run_expectations": {"reasons": {"<prose>": "REASON_EXPLANATIONS_DISAGREE"}}. Every
// prose reason of the archive must be listed, and every listed one found.
type DeclaredRuns struct {
	Reasons map[string]string `json:"reasons"`
}

// check checks each reason names a reason of the IR.
func (r DeclaredRuns) check() error {
	for prose, name := range r.Reasons {
		if prose == "" || umpirespb.RunExpectation_Reason_value[name] == 0 {
			return fmt.Errorf("declared run reason %q names %q, which is no reason of the IR", prose, name)
		}
	}
	return nil
}

// disposition is what the baseline's reader took a Run to end as: stopped by its Monitor when the
// Contract is expected violated, completed otherwise.
func disposition(contractViolated bool) string {
	if contractViolated {
		return "STOPPED_BY_MONITOR"
	}
	return "COMPLETED"
}

// Declare gives the archive with every expected Run declared, as the current schema reads it. A
// listed reason no archived file writes, or a prose reason the list leaves out, fails.
func (r DeclaredRuns) Declare(archived map[string][]byte) (map[string][]byte, error) {
	out := maps.Clone(archived)
	used := map[string]bool{}
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(archived)) {
		var err error
		switch {
		case key == OriginalCases+"manifest.json":
			out[key], err = r.manifest(archived[key], used)
		case (strings.HasPrefix(key, OriginalIR) || strings.HasPrefix(key, OriginalLifts)) && strings.HasSuffix(key, ".json") && !IsLawSidecar(key):
			out[key], err = r.model(archived[key], used)
		default:
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
	}
	for _, prose := range slices.Sorted(maps.Keys(r.Reasons)) {
		if !used[prose] {
			errs = append(errs, fmt.Errorf("declared run reason %q is written by no archived expected Run", prose))
		}
	}
	return out, errors.Join(errs...)
}

// id is the IR reason a baseline's prose names.
func (r DeclaredRuns) id(prose any, used map[string]bool) (string, error) {
	text, ok := prose.(string)
	name, listed := r.Reasons[text]
	if !ok || !listed {
		return "", fmt.Errorf("expected Run reason %v is not a listed declared run reason", prose)
	}
	used[text] = true
	return name, nil
}

// model declares each expected Run of an IR file, wherever it is.
func (r DeclaredRuns) model(encoded []byte, used map[string]bool) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var m any
	if err := decoder.Decode(&m); err != nil {
		return nil, err
	}
	var walk func(v any) error
	walk = func(v any) error {
		switch v := v.(type) {
		case map[string]any:
			if run, ok := v["expectedRun"].(map[string]any); ok {
				if err := r.run(run, used); err != nil {
					return err
				}
			}
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range v {
				if err := walk(child); err != nil {
					return err
				}
			}
		default:
		}
		return nil
	}
	if err := walk(m); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// run declares one expected Run of an IR file.
func (r DeclaredRuns) run(run map[string]any, used map[string]bool) error {
	for _, field := range []string{"disposition", "cleanup"} {
		if _, ok := run[field]; ok {
			return fmt.Errorf("an archived expected Run declares its %s", field)
		}
	}
	if _, ok := run["contract"]; !ok {
		run["contract"] = "OUTCOME_SATISFIED"
	}
	run["disposition"] = "DISPOSITION_" + disposition(run["contract"] == "OUTCOME_VIOLATED")
	run["cleanup"] = "CLEANUP_SUCCEEDED"
	reasoned := []map[string]any{run}
	monitors, _ := run["monitors"].([]any)
	for _, monitor := range monitors {
		if m, ok := monitor.(map[string]any); ok {
			reasoned = append(reasoned, m)
		}
	}
	for _, claim := range reasoned {
		if prose, ok := claim["reason"]; ok {
			name, err := r.id(prose, used)
			if err != nil {
				return err
			}
			claim["reason"] = name
		}
	}
	return nil
}

var (
	// An expected Run's opening line, and the member after it, in the manifest's indented layout.
	manifestExpected = regexp.MustCompile(`(?m)^( *)"expected": \{\n( *)(?:"contract": "(\w+)",\n *)?"conformance"`)
	manifestReason   = regexp.MustCompile(`"reason": ("(?:[^"\\]|\\.)*")`)
)

// manifest declares each expected Run of the Case manifest, rewriting its lines where they are and
// keeping every other byte, as GenerateCases lays out the declared members: contract, disposition and
// cleanup ahead of conformance, and each reason as the id the IR reason names.
func (r DeclaredRuns) manifest(encoded []byte, used map[string]bool) ([]byte, error) {
	var errs []error
	declared := manifestExpected.ReplaceAllFunc(encoded, func(match []byte) []byte {
		parts := manifestExpected.FindSubmatch(match)
		indent, contract := string(parts[2]), string(parts[3])
		if contract == "" {
			contract = "satisfied"
		}
		return fmt.Appendf(nil, "%s\"expected\": {\n%s\"contract\": %q,\n%s\"disposition\": %q,\n%s\"cleanup\": \"succeeded\",\n%s\"conformance\"",
			parts[1], indent, contract, indent, strings.ToLower(disposition(contract == "violated")), indent, indent)
	})
	if want, got := bytes.Count(encoded, []byte(`"expected": {`)), len(manifestExpected.FindAll(encoded, -1)); want != got {
		errs = append(errs, fmt.Errorf("%d of %d expected Runs are laid out as the manifest lays one out", got, want))
	}
	declared = manifestReason.ReplaceAllFunc(declared, func(match []byte) []byte {
		var prose string
		if err := json.Unmarshal(manifestReason.FindSubmatch(match)[1], &prose); err != nil {
			errs = append(errs, err)
			return match
		}
		name, err := r.id(prose, used)
		if err != nil {
			errs = append(errs, err)
			return match
		}
		return fmt.Appendf(nil, "\"reason\": %q", strings.ToLower(strings.TrimPrefix(name, "REASON_")))
	})
	return declared, errors.Join(errs...)
}
