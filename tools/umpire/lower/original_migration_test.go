package lower_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/proto"
)

var captureOriginal = flag.String("capture-original", "", "write the original baseline's derived exploration outputs into this archive directory")

const originalLowerOutputs = "lower.json"

type originalLowering struct {
	root              string
	delta             golden.Delta
	archived, current map[string][]byte
	// models are the current Models as produced; ungenerated are them without the generated claims
	// the delta lists, which must lower and explore as expected, the baselines with the delta applied.
	baselines, expected, models, ungenerated map[string]*umpirespb.Model
}

func readOriginalLowering(t *testing.T) originalLowering {
	t.Helper()
	in := originalLowering{expected: map[string]*umpirespb.Model{}, models: map[string]*umpirespb.Model{}, ungenerated: map[string]*umpirespb.Model{}}
	var err error
	in.root, err = golden.Root()
	require.NoError(t, err)
	in.delta, err = golden.OriginalDelta()
	require.NoError(t, err)
	in.archived, err = golden.OriginalArchive(in.root)
	require.NoError(t, err)
	in.current, err = golden.OriginalCurrent(in.root)
	require.NoError(t, err)
	in.baselines, err = golden.OriginalModels(in.archived)
	require.NoError(t, err)
	current, err := golden.OriginalModels(in.current)
	require.NoError(t, err)
	applied := golden.Applied{}
	for key, baseline := range in.baselines {
		in.expected[key], err = in.delta.Expected(key, baseline, applied)
		require.NoError(t, err, key)
		require.Contains(t, current, key)
		in.models[key] = current[key]
		in.ungenerated[key], err = in.delta.Ungenerated(key, current[key])
		require.NoError(t, err, key)
	}
	require.NoError(t, in.delta.Unapplied(applied))
	return in
}

// generatedCases lowers the IR Models of models as `make umpire-gen-cases` lowers model/ir: the
// ordinary admission every checked-in Case goes through.
func generatedCases(t *testing.T, models map[string]*umpirespb.Model) map[string][]byte {
	t.Helper()
	dir := t.TempDir()
	for key, m := range models {
		if !strings.HasPrefix(key, golden.OriginalIR) {
			continue
		}
		encoded, err := golden.Proto(m)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, strings.TrimPrefix(key, golden.OriginalIR)), encoded, 0644))
	}
	generated, err := lower.GenerateCases(dir)
	require.NoError(t, err)
	out := map[string][]byte{}
	for name, encoded := range generated {
		out[golden.OriginalCases+name] = encoded
	}
	return out
}

func casesOf(files map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for key, encoded := range files {
		if strings.HasPrefix(key, golden.OriginalCases) {
			out[key] = encoded
		}
	}
	return out
}

// compareOriginalCases compares Cases and the manifest byte for byte but for the source positions
// they name. Equal bytes also give each Case the same canonical identity and fingerprint.
func compareOriginalCases(expected, actual map[string][]byte, labels ...string) error {
	want, got := map[string][]byte{}, map[string][]byte{}
	for key, encoded := range expected {
		want[key] = golden.Located(encoded, labels...)
	}
	for key, encoded := range actual {
		got[key] = golden.Located(encoded, labels...)
	}
	for _, key := range slices.Sorted(maps.Keys(want)) {
		if g, ok := got[key]; ok && !bytes.Equal(want[key], g) {
			return fmt.Errorf("%s differs from the original baseline: %s", key, golden.FirstDifference(want[key], g))
		}
	}
	return golden.Compare(want, got)
}

func sources(models map[string]*umpirespb.Model) []string {
	var out []string
	for _, m := range models {
		out = append(out, m.GetSource())
	}
	return out
}

// ungeneratedCases is a complete generated tree of Cases without the generated Queries the delta
// lists: their manifest entries, and the Case files of those it lists as new Cases, each of which must
// be lowered to the file `make umpire-gen-cases` names; a generated Query it does not list must not be
// lowered. Every Query of a new IR file is dropped, and each it lowers must be a listed new Case. The
// manifest is encoded again as GenerateCases encodes it.
func ungeneratedCases(delta golden.Delta, cases map[string][]byte) (map[string][]byte, error) {
	const manifestKey = golden.OriginalCases + "manifest.json"
	var manifest lower.Manifest
	decoder := json.NewDecoder(bytes.NewReader(cases[manifestKey]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("%s: %w", manifestKey, err)
	}
	type query struct{ model, owner, name string }
	generated, listed := map[query]bool{}, map[query]string{}
	for _, r := range delta.Replacements {
		generated[query{strings.TrimPrefix(r.Model, golden.OriginalIR), r.Machine, r.Generated()}] = true
	}
	for _, c := range delta.NewCases {
		listed[query{model: strings.TrimPrefix(c.Model, golden.OriginalIR), name: c.Query}] = strings.TrimPrefix(c.File(), golden.OriginalCases)
	}
	out, seen := maps.Clone(cases), map[query]bool{}
	var errs []error
	manifest.Queries = slices.DeleteFunc(manifest.Queries, func(e lower.GeneratedCase) bool {
		q := query{e.Model, e.Query.Owner, e.Query.Name}
		if !generated[q] && !slices.Contains(delta.NewIRFiles, golden.OriginalIR+e.Model) {
			return false
		}
		seen[q] = true
		file, isListed := listed[query{model: e.Model, name: e.Query.Name}]
		switch {
		case isListed && (e.Standing != lower.Lowered || e.File != file):
			errs = append(errs, fmt.Errorf("new Case %s of %s is %s to %q, not lowered to %s", e.Query.Name, e.Model, e.Standing, e.File, file))
		case !isListed && e.Standing == lower.Lowered:
			errs = append(errs, fmt.Errorf("generated Query %s of %s lowers to %s, which the delta does not list as a new Case", e.Query.Name, e.Model, e.File))
		default:
			// A listed Case lowered to its file, or an unlisted Query lowered to none, is as the delta says.
		}
		delete(out, golden.OriginalCases+e.File)
		return true
	})
	for _, q := range slices.SortedFunc(maps.Keys(generated), func(a, b query) int { return strings.Compare(a.model+" "+a.name, b.model+" "+b.name) }) {
		if !seen[q] {
			errs = append(errs, fmt.Errorf("generated Query %s of %s has no manifest entry", q.name, q.model))
		}
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	out[manifestKey] = append(encoded, '\n')
	return out, errors.Join(errs...)
}

// TestOriginalBaselineCases holds every checked-in Case and the manifest to the Cases the current
// model/ir lowers to, and those, without the generated Queries the delta lists, to the archived Cases:
// when the delta attaches entities or replaces claims, to the Cases the baseline lowers to with
// exactly that delta.
func TestOriginalBaselineCases(t *testing.T) {
	in := readOriginalLowering(t)
	expected := casesOf(in.archived)
	if in.delta.Rederives() {
		require.NoError(t, compareOriginalCases(expected, generatedCases(t, in.baselines)), "the archive lowers as it was frozen")
		expected = generatedCases(t, in.expected)
	}
	labels := append(sources(in.baselines), sources(in.models)...)
	lowered := generatedCases(t, in.models)
	require.NoError(t, compareOriginalCases(lowered, casesOf(in.current)), "checked in as lowered from the current IR")
	ungenerated, err := ungeneratedCases(in.delta, lowered)
	require.NoError(t, err)
	require.NoError(t, compareOriginalCases(expected, ungenerated, labels...), "lowered from the current IR")
	require.NoError(t, compareOriginalCases(ungenerated, generatedCases(t, in.ungenerated)), "the generated Queries are all the tree adds")
}

// TestOriginalBaselineNewCasesAreExact changes which generated Queries the delta lists as new Cases,
// and the tree they lower to: the tree without them no longer reads as the delta says.
func TestOriginalBaselineNewCasesAreExact(t *testing.T) {
	in := readOriginalLowering(t)
	lowered := casesOf(in.current)
	_, err := ungeneratedCases(in.delta, lowered)
	require.NoError(t, err)
	require.NotEmpty(t, in.delta.NewCases)
	listed := in.delta.NewCases[0]
	for name, change := range map[string]func(*golden.Delta, map[string][]byte){
		"unlisted new Case": func(d *golden.Delta, _ map[string][]byte) { d.NewCases = d.NewCases[1:] },
		"listed Case not lowered": func(d *golden.Delta, _ map[string][]byte) {
			d.NewCases = append(d.NewCases, golden.NewCase{Model: "ir/activity.json", Query: "activityProduct.terminalStatesAreFinal"})
		},
		"generated Query without an entry": func(d *golden.Delta, _ map[string][]byte) {
			d.Replacements = append(d.Replacements, golden.Replacement{Model: "ir/activity.json", Machine: "activityProtocol", Law: "missing", Verdict: "found"})
		},
		"lowered to another file": func(_ *golden.Delta, m map[string][]byte) {
			key := golden.OriginalCases + "manifest.json"
			m[key] = bytes.Replace(m[key], []byte(`"file": "`+strings.TrimPrefix(listed.File(), golden.OriginalCases)+`"`), []byte(`"file": "elsewhere.json"`), 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := in.delta
			d.NewCases, d.Replacements = slices.Clone(d.NewCases), slices.Clone(d.Replacements)
			changed := maps.Clone(lowered)
			change(&d, changed)
			_, err := ungeneratedCases(d, changed)
			require.Error(t, err)
		})
	}
}

func TestOriginalBaselineCasesRejectChanges(t *testing.T) {
	in := readOriginalLowering(t)
	archived := casesOf(in.archived)
	moved := maps.Clone(archived)
	const retry = golden.OriginalCases + "activity-retry-case.json"
	moved[retry] = bytes.ReplaceAll(moved[retry], []byte("standaloneactivity/Claims.scala"), []byte("standaloneactivity/Queries.scala"))
	require.NoError(t, compareOriginalCases(archived, moved), "a Case names its Query's new file")
	for name, change := range map[string]func(map[string][]byte){
		"Case byte": func(m map[string][]byte) {
			m[retry] = bytes.Replace(m[retry], []byte(`"major":1`), []byte(`"major":2`), 1)
		},
		"definition ID": func(m map[string][]byte) {
			m[retry] = bytes.ReplaceAll(m[retry], []byte("query.retry"), []byte("query.retried"))
		},
		"fingerprint": func(m map[string][]byte) {
			m[retry] = bytes.Replace(m[retry], []byte(`"behaviorFingerprint":"sha256:`), []byte(`"behaviorFingerprint":"sha256:0`), 1)
		},
		"manifest field": func(m map[string][]byte) {
			key := golden.OriginalCases + "manifest.json"
			m[key] = bytes.Replace(m[key], []byte(`"status": "satisfied"`), []byte(`"status": "violated"`), 1)
		},
		"Case dropped": func(m map[string][]byte) { delete(m, retry) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := maps.Clone(archived)
			change(changed)
			require.Error(t, compareOriginalCases(archived, changed))
		})
	}
}

// deriveExplorations streams every exploration of one Model: its plan, and each candidate and
// reduction with its lowered Case. A candidate's own digest covers positions and Function names, so
// the candidate is named by the digest of its Model as the comparison reads it, everywhere its
// digest appears, and its Case identity is derived again from those bytes.
func deriveExplorations(m *umpirespb.Model, project func(*umpirespb.Model) (*umpirespb.Model, error), s *golden.Stream) error {
	label := m.GetSource()
	for _, query := range m.GetQueries() {
		if query.GetExploration() == nil {
			continue
		}
		plan, err := explore.New(m, query.GetExploration().GetName())
		if err != nil {
			return err
		}
		if err := s.Add("plan "+plan.Name, []string{plan.Name, plan.Query, fmt.Sprint(plan.Runs, plan.Edits, len(plan.Candidates))}, label); err != nil {
			return err
		}
		candidate := func(key string, c *explore.Candidate) error {
			projected, err := project(c.Model)
			if err != nil {
				return err
			}
			digest, err := golden.ProjectedDigest(projected)
			if err != nil {
				return err
			}
			named := *c
			named.Digest, named.Identity = digest, ""
			var encoded []byte
			if c.Case != nil {
				if encoded, err = golden.Proto(c.Case); err != nil {
					return err
				}
				encoded = golden.Located(bytes.ReplaceAll(encoded, []byte(c.Digest), []byte(digest)), label)
				if named.Identity, err = recordedrun.CaseIdentity(encoded); err != nil {
					return err
				}
			}
			return errors.Join(s.Add(key, &named, label), s.Add(key+" case", encoded, label))
		}
		for i, c := range plan.Candidates {
			key := fmt.Sprintf("%s/%03d", plan.Name, i)
			if err := candidate(key, c); err != nil {
				return err
			}
			for index := range max(0, len(c.Actions)-1) {
				reduced, err := plan.Reduce(c, index)
				rkey := fmt.Sprintf("%s/reduce-%03d", key, index)
				if err != nil {
					if err := s.Add(rkey+" error", []byte(err.Error()), label); err != nil {
						return err
					}
					continue
				}
				if err := candidate(rkey, reduced); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func explorationDigests(t *testing.T, models map[string]*umpirespb.Model, project func(*umpirespb.Model) (*umpirespb.Model, error)) golden.Derived {
	t.Helper()
	d := golden.Derived{}
	for key, m := range models {
		s := &golden.Stream{}
		require.NoError(t, deriveExplorations(m, project, s), key)
		d["explorations/"+key] = s.Digest()
	}
	return d
}

func TestCaptureOriginalBaseline(t *testing.T) {
	if *captureOriginal == "" {
		t.Skip("explicit -capture-original=<archive directory> required")
	}
	in := readOriginalLowering(t)
	require.NoError(t, golden.WriteDerived(*captureOriginal, originalLowerOutputs, explorationDigests(t, in.baselines, in.delta.ProjectBaseline)))
}

// TestOriginalBaselineExplorations holds every exploration of the current Models, its candidates,
// reductions and their Cases, to the original baseline's: of the current Models without the generated
// claims the delta lists, since a candidate is a whole Model, to the baseline's with the delta.
func TestOriginalBaselineExplorations(t *testing.T) {
	if *captureOriginal != "" {
		t.Skip("capture is separate from verification")
	}
	in := readOriginalLowering(t)
	expected, err := golden.ReadDerived(in.root, originalLowerOutputs)
	require.NoError(t, err)
	if in.delta.Rederives() {
		require.NoError(t, golden.CompareDerived(expected, explorationDigests(t, in.baselines, in.delta.ProjectBaseline), nil),
			"the archive explores as it was frozen")
		expected = explorationDigests(t, in.expected, in.delta.ProjectBaseline)
	}
	explain := func(key string) string {
		model := strings.TrimPrefix(key, "explorations/")
		return golden.Explain(expected[key], func(original bool, s *golden.Stream) error {
			if original {
				return deriveExplorations(in.expected[model], in.delta.ProjectBaseline, s)
			}
			return deriveExplorations(in.ungenerated[model], in.delta.ProjectCurrent, s)
		})
	}
	require.NoError(t, golden.CompareDerived(expected, explorationDigests(t, in.ungenerated, in.delta.ProjectCurrent), explain))
}

// TestOriginalBaselineExplorationsRejectChanges moves the Nexus caller as a migration may, which the
// projection admits, and changes what it explores, which it does not.
func TestOriginalBaselineExplorationsRejectChanges(t *testing.T) {
	const key = golden.OriginalIR + "nexus-caller.json"
	in := readOriginalLowering(t)
	archived, err := golden.ReadDerived(in.root, originalLowerOutputs)
	require.NoError(t, err)
	digest := func(m *umpirespb.Model, project func(*umpirespb.Model) (*umpirespb.Model, error)) string {
		return explorationDigests(t, map[string]*umpirespb.Model{key: m}, project)["explorations/"+key]
	}
	moved := proto.CloneOf(in.baselines[key])
	moved.Source = "model: moved roots"
	for _, q := range moved.GetQueries() {
		q.Position = &umpirespb.Position{File: "model/temporal/nexuscaller/Queries.scala", Line: 7}
	}
	require.Equal(t, archived["explorations/"+key], digest(moved, in.delta.ProjectCurrent))
	changed := proto.CloneOf(moved)
	var exploring *umpirespb.Query
	for _, q := range changed.GetQueries() {
		if q.GetExploration() != nil {
			exploring = q
		}
	}
	require.NotNil(t, exploring)
	exploring.GetExploration().GetVariations()[0].GetChoices()[0].Priority += 5
	require.NotEqual(t, archived["explorations/"+key], digest(changed, in.delta.ProjectCurrent))
}
