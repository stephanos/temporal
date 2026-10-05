package lower

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	runtime "go.temporal.io/server/common/testing/testpilot"
)

func generatedFiles(t *testing.T) map[string][]byte {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "model", "cases", "*.json"))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, path := range paths {
		files[filepath.Base(path)], err = os.ReadFile(path)
		require.NoError(t, err)
	}
	return files
}

func TestManifestRejectsInvalidMetadata(t *testing.T) {
	encoded := generatedFiles(t)["manifest.json"]
	for name, mutate := range map[string]func(*Manifest){
		"version":         func(m *Manifest) { m.Version++ },
		"duplicate query": func(m *Manifest) { m.Queries = append(m.Queries, m.Queries[0]) },
		"duplicate path": func(m *Manifest) {
			for i := range m.Queries {
				if m.Queries[i].Standing == Lowered {
					other := m.Queries[i]
					other.Query.Name += "-other"
					m.Queries = append(m.Queries, other)
					break
				}
			}
		},
		"traversal": func(m *Manifest) { m.Queries[0].Model = "../model.json" },
		"missing expected": func(m *Manifest) {
			for i := range m.Queries {
				if m.Queries[i].Standing == Lowered {
					m.Queries[i].Expected = nil
					break
				}
			}
		},
		"unknown status":        expecting(func(e *ExpectedRun) { e.Properties[0].Status = "maybe" }),
		"no contract":           expecting(func(e *ExpectedRun) { e.Contract = "" }),
		"inconclusive contract": expecting(func(e *ExpectedRun) { e.Contract = "inconclusive" }),
		"no disposition":        expecting(func(e *ExpectedRun) { e.Disposition = "" }),
		"unknown disposition":   expecting(func(e *ExpectedRun) { e.Disposition = "abandoned" }),
		"shouted disposition":   expecting(func(e *ExpectedRun) { e.Disposition = "COMPLETED" }),
		"no cleanup":            expecting(func(e *ExpectedRun) { e.Cleanup = "" }),
		"unknown cleanup":       expecting(func(e *ExpectedRun) { e.Cleanup = "skipped" }),
		// ConcludeVerdict stops a Run its rules violate, and concludes no violation of a completed one.
		"violated yet completed": expecting(func(e *ExpectedRun) { e.Contract, e.Disposition = "violated", "completed" }),
		"satisfied yet stopped":  expecting(func(e *ExpectedRun) { e.Contract, e.Disposition = "satisfied", "stopped_by_monitor" }),
		"satisfied with a reason": expecting(func(e *ExpectedRun) {
			e.Properties[0].Status, e.Properties[0].Reason = "satisfied", "hole"
		}),
		"inconclusive without a reason": expecting(func(e *ExpectedRun) {
			e.Properties[0].Status, e.Properties[0].Reason = "inconclusive", ""
		}),
		// ExpectationID is the one spelling of an id: a reason or a disposition spelled otherwise is none.
		"a mixed-case reason": expecting(func(e *ExpectedRun) {
			e.Properties[0].Status, e.Properties[0].Reason = "inconclusive", "Explanations_Disagree"
		}),
		"a mixed-case disposition": expecting(func(e *ExpectedRun) { e.Disposition = "Completed" }),
		"an unknown reason": expecting(func(e *ExpectedRun) {
			e.Properties[0].Status, e.Properties[0].Reason = "inconclusive", "Explanations Disagree"
		}),
		// Conformance short of conformant may name the judge's reason, spelled as a claim's is.
		"conformant with a conformance reason": expecting(func(e *ExpectedRun) {
			e.Conformance, e.ConformanceReason = "conformant", "hole"
		}),
		"an unknown conformance reason": expecting(func(e *ExpectedRun) {
			e.Conformance, e.ConformanceReason = "inconclusive", "half-closed"
		}),
		"a mixed-case conformance reason": expecting(func(e *ExpectedRun) {
			e.Conformance, e.ConformanceReason = "inconclusive", "Incomplete"
		}),
	} {
		t.Run(name, func(t *testing.T) {
			m, err := DecodeManifest(encoded)
			require.NoError(t, err)
			mutate(m)
			invalid, err := json.Marshal(m)
			require.NoError(t, err)
			_, err = DecodeManifest(invalid)
			require.Error(t, err)
		})
	}
	for _, invalid := range [][]byte{[]byte(`{"version":1,"unknown":true}`), append(append([]byte{}, encoded...), []byte("{}")...)} {
		_, err := DecodeManifest(invalid)
		require.Error(t, err)
	}
	// A conformance reason on conformance short of conformant is read back as declared.
	m, err := DecodeManifest(encoded)
	require.NoError(t, err)
	expecting(func(e *ExpectedRun) { e.Conformance, e.ConformanceReason = "inconclusive", "incomplete" })(m)
	valid, err := json.Marshal(m)
	require.NoError(t, err)
	_, err = DecodeManifest(valid)
	require.NoError(t, err)
}

// expecting changes the first lowered Query's expected Run.
func expecting(change func(*ExpectedRun)) func(*Manifest) {
	return func(m *Manifest) {
		for i := range m.Queries {
			if m.Queries[i].Standing == Lowered {
				change(m.Queries[i].Expected)
				return
			}
		}
	}
}

// A Query whose expected Run pairs a Contract verdict with a disposition the judge's aggregation never
// leaves a Run in is refused at the Query's line, before any Case is written for it.
func TestGeneratingRefusesAnExpectationTheJudgeCannotConclude(t *testing.T) {
	m := loaded(t, "nexus-control")
	p, err := NewProducer(m)
	require.NoError(t, err)
	at := slices.IndexFunc(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "forgedCompletion" })
	require.GreaterOrEqual(t, at, 0)
	query := m.GetQueries()[at]
	_, _, err = generateCase(p, "nexus-control.json", query)
	require.NoError(t, err)
	query.GetExpectedRun().Disposition = umpirespb.RunExpectation_DISPOSITION_COMPLETED
	_, _, err = generateCase(p, "nexus-control.json", query)
	require.EqualError(t, err, fmt.Sprintf("%s:%d: Query forgedCompletion expects a violated Contract on a Run completed, which no Run's rules conclude",
		query.GetPosition().GetFile(), query.GetPosition().GetLine()))
}

// A Query's declared conformance reason reaches its manifest entry as ExpectationID spells it; a
// Query that declares none leaves the entry's empty.
func TestGeneratingCarriesADeclaredConformanceReason(t *testing.T) {
	m := loaded(t, "nexus-control")
	p, err := NewProducer(m)
	require.NoError(t, err)
	at := slices.IndexFunc(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "forgedCompletion" })
	require.GreaterOrEqual(t, at, 0)
	query := m.GetQueries()[at]
	entry, _, err := generateCase(p, "nexus-control.json", query)
	require.NoError(t, err)
	require.Empty(t, entry.Expected.ConformanceReason)
	query.GetExpectedRun().ConformanceReason = umpirespb.RunExpectation_REASON_INCOMPLETE
	entry, _, err = generateCase(p, "nexus-control.json", query)
	require.NoError(t, err)
	require.Equal(t, "incomplete", entry.Expected.ConformanceReason)
}

// A declared conformance reason is held to the Assessment's by equality; one undeclared is not
// compared.
func TestExpectedRunChecksADeclaredConformanceReason(t *testing.T) {
	expected := &ExpectedRun{Contract: "satisfied", Disposition: "completed", Cleanup: "succeeded", Conformance: "inconclusive",
		ConformanceReason: "incomplete", Properties: []ExpectedClaim{{ID: "settles", Status: "satisfied"}}}
	run := &testpilotspb.Run{Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	verdict := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}
	assessed := func(reason string) *runtime.Assessment {
		return &runtime.Assessment{Conformance: runtime.ConformanceAssessment{Status: runtime.ConformanceInconclusive, Reason: reason, Detail: "the prose"},
			Properties: []runtime.PropertyAssessment{{ID: "settles", Status: runtime.PropertySatisfied}}}
	}
	require.NoError(t, expected.Check(run, verdict, assessed("incomplete")))
	require.EqualError(t, expected.Check(run, verdict, assessed("hole")), "the conformance reason is hole, expected incomplete: the prose")
	expected.ConformanceReason = ""
	require.NoError(t, expected.Check(run, verdict, assessed("hole")))
}

// Check holds a Run, its Verdict and its Assessment to the expectation by equality, and names each
// thing that differs; prose is never compared.
func TestExpectedRunChecksEachDeclaredValueByEquality(t *testing.T) {
	expected := &ExpectedRun{Contract: "violated", Disposition: "stopped_by_monitor", Cleanup: "succeeded", Conformance: "inconclusive",
		Properties: []ExpectedClaim{{ID: "forgedSuccess", Status: "violated", Reason: "every_explanation_violates"}, {ID: "watch", Status: "satisfied"}}}
	run := func() *testpilotspb.Run {
		return &testpilotspb.Run{Disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
			Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	}
	verdict := func() *testpilotspb.Verdict {
		return &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}
	}
	assessment := func() *runtime.Assessment {
		return &runtime.Assessment{Conformance: runtime.ConformanceAssessment{Status: runtime.ConformanceInconclusive, Reason: "incomplete", Detail: "any prose"},
			Properties: []runtime.PropertyAssessment{{ID: "watch", Status: runtime.PropertySatisfied},
				{ID: "forgedSuccess", Status: runtime.PropertyViolated, Reason: "every_explanation_violates", Detail: "other prose"}}}
	}
	require.NoError(t, expected.Check(run(), verdict(), assessment()))
	// A Run with no diagnostics says so by naming none.
	moved := run()
	moved.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED
	require.EqualError(t, expected.Check(moved, verdict(), assessment()), "the disposition is completed, expected stopped_by_monitor")
	for name, test := range map[string]struct {
		change func(*testpilotspb.Run, *testpilotspb.Verdict, *runtime.Assessment)
		says   string
	}{
		"disposition": {func(r *testpilotspb.Run, _ *testpilotspb.Verdict, _ *runtime.Assessment) {
			r.Disposition = testpilotspb.RUN_DISPOSITION_COMPLETED
		}, "the disposition is completed, expected stopped_by_monitor"},
		// A Run decoded from newer bytes may hold a value its enum does not name.
		"unknown disposition": {func(r *testpilotspb.Run, _ *testpilotspb.Verdict, _ *runtime.Assessment) {
			r.Disposition = 99
			r.Diagnostics = []*testpilotspb.RunDiagnostic{{Code: "late"}}
		}, "the disposition is unknown(99), expected stopped_by_monitor: [code:\"late\"]"},
		"cleanup": {func(r *testpilotspb.Run, _ *testpilotspb.Verdict, _ *runtime.Assessment) {
			r.Cleanup.Status = testpilotspb.CLEANUP_STATUS_TIMED_OUT
		}, "the cleanup is timed_out, expected succeeded"},
		"verdict": {func(_ *testpilotspb.Run, v *testpilotspb.Verdict, _ *runtime.Assessment) {
			v.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE
		}, "the Contract's Verdict is inconclusive, expected violated"},
		"conformance": {func(_ *testpilotspb.Run, _ *testpilotspb.Verdict, a *runtime.Assessment) {
			a.Conformance.Status = runtime.ConformanceConformant
		}, "the conformance is conformant, expected inconclusive"},
		"status": {func(_ *testpilotspb.Run, _ *testpilotspb.Verdict, a *runtime.Assessment) {
			a.Properties[1].Status = runtime.PropertyInconclusive
		}, "forgedSuccess's status is inconclusive, expected violated: other prose"},
		"reason": {func(_ *testpilotspb.Run, _ *testpilotspb.Verdict, a *runtime.Assessment) {
			a.Properties[1].Reason = "explanations_disagree"
		}, "forgedSuccess's reason is explanations_disagree, expected every_explanation_violates"},

		"omitted claim": {func(_ *testpilotspb.Run, _ *testpilotspb.Verdict, a *runtime.Assessment) {
			a.Properties = a.Properties[1:]
		}, "the assessment omits watch"},
		"failure": {func(_ *testpilotspb.Run, _ *testpilotspb.Verdict, a *runtime.Assessment) {
			a.Failure = &runtime.AssessmentFailure{Code: runtime.AssessmentCloseFailed, Detail: "broke"}
		}, "the assessment failed: close_failed broke"},
	} {
		t.Run(name, func(t *testing.T) {
			r, v, a := run(), verdict(), assessment()
			test.change(r, v, a)
			require.ErrorContains(t, expected.Check(r, v, a), test.says)
		})
	}
}

func TestCasePublicationDetectsDriftAndPreservesTheOldTreeOnInvalidInput(t *testing.T) {
	files := generatedFiles(t)
	for _, drift := range []string{"stale", "missing", "obsolete", "symlink"} {
		t.Run(drift, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "cases")
			require.NoError(t, SyncCases(root, files, true))
			switch drift {
			case "stale":
				require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}"), 0644))
			case "missing":
				require.NoError(t, os.Remove(filepath.Join(root, "manifest.json")))
			case "obsolete":
				require.NoError(t, os.WriteFile(filepath.Join(root, "obsolete.json"), []byte("{}"), 0644))
			case "symlink":
				require.NoError(t, os.Remove(filepath.Join(root, "manifest.json")))
				require.NoError(t, os.Symlink("absent", filepath.Join(root, "manifest.json")))
			default:
				t.Fatal(drift)
			}
			require.Error(t, SyncCases(root, files, false))
			require.NoError(t, SyncCases(root, files, true))
			require.NoError(t, SyncCases(root, files, false))
			broken := maps.Clone(files)
			for name := range broken {
				if name != "manifest.json" {
					broken[name] = []byte("{")
					break
				}
			}
			require.Error(t, SyncCases(root, broken, true))
			require.NoError(t, SyncCases(root, files, false))
		})
	}
}

// A selection is the complete tree's own bytes for the Queries it names, whatever order names them,
// and it publishes and checks as a tree of its own.
func TestSelectedCasesAreTheCompleteTreesBytes(t *testing.T) {
	// The checked-in tree is what the checked IR lowers to, which TestGeneratedCasesAreCheckedIn holds.
	files, again := generatedFiles(t), generatedFiles(t)

	selected, err := SelectCases(files, []Selected{
		{Model: "nexus-control.json", Query: "forgedCompletion"},
		{Model: "nexus-caller.json", Query: "syncCompletion"},
	})
	require.NoError(t, err)
	reordered, err := SelectCases(again, []Selected{
		{Model: "nexus-caller.json", Query: "syncCompletion"},
		{Model: "nexus-control.json", Query: "forgedCompletion"},
	})
	require.NoError(t, err)
	require.Equal(t, selected, reordered)

	manifest, err := DecodeManifest(selected["manifest.json"])
	require.NoError(t, err)
	complete, err := DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	expected := map[string][]byte{"manifest.json": selected["manifest.json"]}
	var entries []GeneratedCase
	for _, entry := range complete.Queries {
		if entry.File == "nexus-caller-syncCompletion-case.json" || entry.File == "nexus-control-forgedCompletion-case.json" {
			expected[entry.File] = files[entry.File]
			entries = append(entries, entry)
		}
	}
	require.Equal(t, &Manifest{Version: 1, Queries: entries}, manifest)
	require.Equal(t, expected, selected)

	root := filepath.Join(t.TempDir(), "pinned")
	require.NoError(t, SyncCases(root, selected, true))
	require.NoError(t, SyncCases(root, reordered, false))
	require.Error(t, SyncCases(root, files, false), "the complete tree is another tree")
}

// A Query with no Case cannot be pinned: one no Model declares, one named twice, one that lowers to
// nothing, and an empty selection are each refused, and nothing is selected.
func TestSelectingRefusesAQueryWithNoCase(t *testing.T) {
	files := generatedFiles(t)
	manifest, err := DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	var unlowered Selected
	for _, entry := range manifest.Queries {
		if entry.Standing != Lowered {
			unlowered = Selected{Model: entry.Model, Query: entry.Query.Name}
			break
		}
	}
	require.NotEmpty(t, unlowered.Query, "the model tree accounts for a Query that does not lower")
	pinned := Selected{Model: "nexus-caller.json", Query: "syncCompletion"}
	for name, test := range map[string]struct {
		selected []Selected
		detail   string
	}{
		"nothing":           {nil, "no Query selected"},
		"an undeclared one": {[]Selected{pinned, {Model: "nexus-caller.json", Query: "absent"}}, "no Model declares the selected Query nexus-caller.json/absent"},
		"another Model's":   {[]Selected{{Model: "nexus-control.json", Query: "syncCompletion"}}, "no Model declares the selected Query nexus-control.json/syncCompletion"},
		"one named twice":   {[]Selected{pinned, pinned}, "named twice"},
		"one with no Case":  {[]Selected{pinned, unlowered}, "has no Case: " + string(manifestStanding(manifest, unlowered))},
		"a missing Case":    {[]Selected{{Model: "nexus-control.json", Query: "forgedCompletion"}}, "generated Case nexus-control-forgedCompletion-case.json is missing"},
		"a broken manifest": {[]Selected{pinned}, "manifest"},
	} {
		t.Run(name, func(t *testing.T) {
			input := maps.Clone(files)
			switch name {
			case "a missing Case":
				delete(input, "nexus-control-forgedCompletion-case.json")
			case "a broken manifest":
				input["manifest.json"] = []byte(`{"version":2,"queries":[]}`)
			default:
			}
			selected, err := SelectCases(input, test.selected)
			require.ErrorContains(t, err, test.detail)
			require.Nil(t, selected)
		})
	}
}

func manifestStanding(manifest *Manifest, query Selected) Standing {
	for _, entry := range manifest.Queries {
		if entry.Model == query.Model && entry.Query.Name == query.Query {
			return entry.Standing
		}
	}
	return ""
}
