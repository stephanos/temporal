package testpilot_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	runtime "go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	cp "go.temporal.io/server/model/scalav2/goir/testpilot/internal/producer"
	"go.temporal.io/server/model/scalav2/explore"
	"go.temporal.io/server/model/scalav2/goir"
	"go.temporal.io/server/model/scalav2/goir/conformance"
	"go.temporal.io/server/model/scalav2/goir/internal/golden"
	lower "go.temporal.io/server/model/scalav2/goir/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var captureMigrationGoldens = flag.String("capture-goldens", "", "exclusively create a new migration artifact capture directory")

func putJSON(t *testing.T, files map[string][]byte, key string, value any) {
	t.Helper()
	encoded, err := golden.JSON(value)
	require.NoError(t, err, key)
	require.NotContains(t, files, key)
	files[key] = encoded
}

func putProto(t *testing.T, files map[string][]byte, key string, value proto.Message) {
	t.Helper()
	encoded, err := golden.Proto(value)
	require.NoError(t, err, key)
	require.NotContains(t, files, key)
	files[key] = encoded
}

func putCase(t *testing.T, files map[string][]byte, key string, c *testpilotspb.Case) {
	t.Helper()
	putProto(t, files, key+"/case.json", c)
	putProto(t, files, key+"/program.json", c.GetProgram())
	putProto(t, files, key+"/contract.json", c.GetContract())
	identity, err := recordedrun.CaseIdentity(files[key+"/case.json"])
	require.NoError(t, err)
	fingerprint, err := runtime.CaseFingerprint(c)
	require.NoError(t, err)
	putJSON(t, files, key+"/identity.json", struct{ Canonical, Fingerprint string }{identity, fingerprint})
}

func artifactError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func artifactModel(t *testing.T, files map[string][]byte, key string, m *modelirspb.Model) {
	t.Helper()
	producer, err := lower.NewProducer(m)
	require.NoError(t, err, key)
	receipts := map[string]goir.Receipt{}
	for _, receipt := range goir.Check(m, goir.DefaultScope).Receipts {
		if receipt.Subject == goir.QuerySubject {
			receipts[receipt.Key.Name] = receipt
		}
	}
	for _, query := range m.GetQueries() {
		qkey := key + "/queries/" + query.GetName()
		identity := cp.IdentityFor("temporal.case", "scala."+strings.TrimSuffix(filepath.Base(key), ".json"), query.GetName())
		lowered, err := producer.Lower(query.GetName(), identity)
		if err != nil {
			putJSON(t, files, qkey+"/error.json", artifactError(err))
			continue
		}
		c := lowered.Case
		lowered.Case = nil
		putJSON(t, files, qkey+"/lowering.json", lowered)
		if c == nil {
			continue
		}
		putCase(t, files, qkey, c)
		factory, err := conformance.Prepare(m, receipts[query.GetName()].Key, c, conformance.Limits{MaxEvents: 2048, MaxProperties: 16, MaxDuration: time.Minute, MaxCandidates: 1 << 16, MaxWork: 1 << 22, MaxReadings: 1 << 22})
		if err != nil {
			putJSON(t, files, qkey+"/assessment-error.json", artifactError(err))
		} else {
			binding := factory.Binding()
			require.Equal(t, assessmentIdentity(t, m), binding.Model)
			putJSON(t, files, qkey+"/assessment.json", binding)
		}
	}
	for _, query := range m.GetQueries() {
		if query.GetExploration() == nil {
			continue
		}
		plan, err := explore.New(m, query.GetExploration().GetName())
		require.NoError(t, err)
		ekey := key + "/explorations/" + plan.Name
		putJSON(t, files, ekey+"/plan.json", plan)
		for i, candidate := range plan.Candidates {
			ckey := fmt.Sprintf("%s/%03d", ekey, i)
			artifactCandidate(t, files, ckey, plan, candidate)
			for index := range max(0, len(candidate.Actions)-1) {
				reduced, err := plan.Reduce(candidate, index)
				rkey := fmt.Sprintf("%s/reduce-%03d", ckey, index)
				if err != nil {
					putJSON(t, files, rkey+"/error.json", artifactError(err))
					continue
				}
				artifactCandidate(t, files, rkey, plan, reduced)
			}
		}
	}
}

func artifactCandidate(t *testing.T, files map[string][]byte, key string, plan *explore.Plan, c *explore.Candidate) {
	t.Helper()
	putJSON(t, files, key+"/candidate.json", c)
	putProto(t, files, key+"/model.json", c.Model)
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(c.Model)
	require.NoError(t, err)
	require.Equal(t, golden.Digest(encoded), c.Digest)
	if c.Rejection != "" {
		return
	}
	identity := cp.IdentityFor("temporal.case", "scala.explore."+plan.Name, c.Digest)
	require.Equal(t, []string{identity.CaseID, identity.ProgramID, identity.ContractID}, []string{c.Case.GetCaseId(), c.Case.GetProgram().GetProgramId(), c.Case.GetContract().GetContractId()})
	canonical, err := recordedrun.CaseIdentity(c.Bytes)
	require.NoError(t, err)
	require.Equal(t, canonical, c.Identity)
	putCase(t, files, key, c.Case)
	proposal, err := plan.Proposal(c)
	require.NoError(t, err)
	require.NotNil(t, proposal.SHA256)
	require.Equal(t, golden.Digest([]byte(proposal.Source)), *proposal.SHA256)
	require.Equal(t, c.Digest+"-regression.json", proposal.Path)
	putJSON(t, files, key+"/proposal.json", proposal)
	trace, err := explore.RenderTrace(c, plan.Query, nil, nil)
	require.NoError(t, err)
	files[key+"/trace.html"] = trace
}

func assessmentIdentity(t *testing.T, model *modelirspb.Model) string {
	t.Helper()
	bare := proto.CloneOf(model)
	bare.Source = ""
	// Position fields must be removed, not replaced by empty messages, to preserve the wire identity.
	clearPositionFields(bare.ProtoReflect())
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(bare)
	require.NoError(t, err)
	return "goir.model/v1:sha256:" + golden.Digest(append([]byte("goir.model/v1"), encoded...))
}

func captureArtifacts(t *testing.T, cfg golden.Config, models map[string]*modelirspb.Model, project bool) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	generatedIR := t.TempDir()
	for _, path := range slices.Sorted(maps.Keys(models)) {
		model := models[path]
		if project {
			var err error
			model, err = cfg.Migrate(model)
			require.NoError(t, err)
			require.Equal(t, assessmentIdentity(t, models[path]), assessmentIdentity(t, model))
		}
		key := strings.TrimPrefix(path, "model/scalav2/")
		putProto(t, files, "inputs/"+key, model)
		artifactModel(t, files, key, model)
		if strings.HasPrefix(key, "ir/") {
			require.NoError(t, os.WriteFile(filepath.Join(generatedIR, filepath.Base(path)), files["inputs/"+key], 0644))
		}
	}
	generated, err := lower.GenerateCases(generatedIR)
	require.NoError(t, err)
	for name, encoded := range generated {
		files["generated/"+name] = encoded
	}
	return files
}

func originalArtifactInputs(t *testing.T) (golden.Config, map[string]*modelirspb.Model) {
	t.Helper()
	cfg, err := golden.Configuration()
	require.NoError(t, err)
	root, err := golden.Root()
	require.NoError(t, err)
	inputs, err := cfg.Inputs(root)
	require.NoError(t, err)
	return cfg, inputs
}

func prefixArtifacts(out, files map[string][]byte, prefix string) {
	for name, data := range files {
		out[prefix+"/"+name] = data
	}
}

func TestCaptureMigrationGoldens(t *testing.T) {
	if *captureMigrationGoldens == "" {
		t.Skip("explicit -capture-goldens=<new directory> required")
	}
	cfg, inputs := originalArtifactInputs(t)
	files := map[string][]byte{}
	prefixArtifacts(files, captureArtifacts(t, cfg, inputs, false), "original")
	prefixArtifacts(files, captureArtifacts(t, cfg, inputs, true), "mapped")
	prefixArtifacts(files, legacyArtifacts(t), "oracles")
	require.NoError(t, golden.Capture(*captureMigrationGoldens, files))
}

func TestMigrationGoldens(t *testing.T) {
	if *captureMigrationGoldens != "" {
		t.Skip("capture is separate from verification")
	}
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	cfg, inputs := originalArtifactInputs(t)
	selected := ""
	for path, current := range inputs {
		original := new(modelirspb.Model)
		require.NoError(t, protojson.Unmarshal(expected["original/inputs/"+strings.TrimPrefix(path, "model/scalav2/")], original), path)
		mapped, err := cfg.Match(original, current)
		require.NoError(t, err, path)
		prefix := "original"
		if mapped {
			prefix = "mapped"
		}
		if selected == "" {
			selected = prefix
		}
		require.Equal(t, selected, prefix, "partial IR migration")
		inputs[path] = original
	}
	actual := map[string][]byte{}
	prefixArtifacts(actual, captureArtifacts(t, cfg, inputs, false), "original")
	prefixArtifacts(actual, captureArtifacts(t, cfg, inputs, true), "mapped")
	prefixArtifacts(actual, legacyArtifacts(t), "oracles")
	require.NoError(t, compareMigrationArtifactInventory(expected, actual))
}

func migrationJobArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for key, value := range legacyArtifacts(t) {
		if strings.HasPrefix(key, "job/") {
			files["oracles/"+key] = value
		}
	}
	return files
}

func TestMigrationJobFixturesPreserveOriginalEvidence(t *testing.T) {
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	for key := range expected {
		if !strings.HasPrefix(key, "oracles/job/") {
			delete(expected, key)
		}
	}
	actual := migrationJobArtifacts(t)
	for key, value := range actual {
		if !bytes.Equal(expected[key], value) {
			path := filepath.Join("/Users/stephan/Workspace/temporal/umpire/.flow/tmp/fn115-5/job-diff", key)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, value, 0o600))
		}
	}
	require.NoError(t, compareMigrationArtifactInventory(expected, actual))
}

func compareMigrationArtifactInventory(expected, actual map[string][]byte) error {
	return golden.Compare(expected, actual)
}

func TestMigrationArtifactInventoryRejectsInactiveChanges(t *testing.T) {
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	for _, active := range []string{"original", "mapped"} {
		inactive := "mapped"
		if active == "mapped" {
			inactive = "original"
		}
		for _, suffix := range []string{"/contract.json", "/proposal.json", "/identity.json", "/manifest.json", "/trace.html"} {
			name := ""
			for _, key := range slices.Sorted(maps.Keys(expected)) {
				if strings.HasPrefix(key, inactive+"/") && strings.HasSuffix(key, suffix) {
					name = key
					break
				}
			}
			require.NotEmpty(t, name, suffix)
			t.Run(active+suffix, func(t *testing.T) {
				changed := maps.Clone(expected)
				if suffix == "/trace.html" {
					require.Contains(t, string(changed[name]), "href=\"")
					changed[name] = []byte(strings.Replace(string(changed[name]), "href=\"", "href=\"changed-", 1))
				} else {
					var artifact map[string]any
					require.NoError(t, json.Unmarshal(changed[name], &artifact))
					field := map[string]string{"/contract.json": "contractId", "/proposal.json": "promotionSourceSha256", "/identity.json": "Fingerprint", "/manifest.json": "version"}[suffix]
					require.Contains(t, artifact, field)
					if field == "version" {
						artifact[field] = 2
					} else {
						artifact[field] = "changed"
					}
					changed[name], err = golden.JSON(artifact)
					require.NoError(t, err)
				}
				require.Error(t, compareMigrationArtifactInventory(changed, expected))
			})
		}
		for _, change := range []string{"missing", "extra", "unknown prefix"} {
			t.Run(active+"/"+change, func(t *testing.T) {
				changed := maps.Clone(expected)
				switch change {
				case "missing":
					delete(changed, inactive+"/generated/manifest.json")
				case "extra":
					changed[inactive+"/unexpected.json"] = []byte("{}")
				default:
					changed["unknown/unexpected.json"] = []byte("{}")
				}
				require.Error(t, compareMigrationArtifactInventory(changed, expected))
			})
		}
	}
}

func TestMigrationCaseGoldenDetectsContractMutation(t *testing.T) {
	c := produced(t, "once", jobSources(), jobOnce...)
	expected := map[string][]byte{}
	putCase(t, expected, "once", c)
	require.NotEmpty(t, c.Contract.Correlated.Transitions)
	c.Contract.Correlated.Transitions[0].State.Value += "-changed"
	actual := map[string][]byte{}
	putCase(t, actual, "once", c)
	require.Error(t, golden.Compare(expected, actual))
	require.True(t, bytes.Equal(expected["once/program.json"], actual["once/program.json"]))
}

func legacyArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	baseline, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	model := new(modelirspb.Model)
	require.NoError(t, protojson.Unmarshal(baseline["original/inputs/ir/nexus-caller.json"], model))
	lower.MigrationComparativeModel(t, model)
	require.Len(t, model.GetQueries(), 7)
	for _, declaration := range model.GetQueries() {
		name := declaration.GetName()
		original := "oracles/nexus/" + name + "/typed"
		var identity cp.Identity
		var source cp.Source
		require.NoError(t, json.Unmarshal(baseline[original+"/identity-input.json"], &identity))
		require.NoError(t, json.Unmarshal(baseline[original+"/source.json"], &source))
		query, realization, err := lower.MigrationFixture(model, name, identity)
		require.NoError(t, err)
		produced, err := cp.Produce(query, identity, realization, source)
		require.NoError(t, err)
		for _, form := range []string{"typed", "keyed"} {
			key := "nexus/" + name + "/" + form
			captureLegacyQuery(t, files, key, query)
			putCase(t, files, key, produced)
			putJSON(t, files, key+"/source.json", source)
			putJSON(t, files, key+"/identity-input.json", identity)
		}
	}

	for name, actions := range map[string][]string{"once": jobOnce, "retried": jobRetried, "closed": {"submit", "take", "check", "close"}} {
		key := "job/" + name
		captureLegacyQuery(t, files, key, jobQuery(t, name, actions...))
		putCase(t, files, key, produced(t, name, jobSources(), actions...))
		putJSON(t, files, key+"/source.json", cp.Source{Path: "job.go", Provenance: "test"})
		putJSON(t, files, key+"/identity-input.json", jobIdentity(name))
		putJSON(t, files, key+"/sources.json", jobSources())
	}
	return files
}

func captureLegacyQuery(t *testing.T, files map[string][]byte, key string, q *goir.Query) {
	t.Helper()
	table, err := q.Scenario.Machine.Table()
	require.NoError(t, err)
	putJSON(t, files, key+"/table.json", table)
	putJSON(t, files, key+"/ids.json", table.IDs())
	putJSON(t, files, key+"/fingerprint.json", table.TargetFingerprint())
	putJSON(t, files, key+"/claims.json", table.Claims())
	fields := map[string][]goir.Atom{}
	for _, state := range table.States {
		fields[state] = table.FieldValues(state)
	}
	putJSON(t, files, key+"/fields.json", fields)
	groups, err := q.Property.Lower()
	require.NoError(t, err)
	putJSON(t, files, key+"/property.json", struct {
		Name   string
		Groups []goir.Group
	}{q.Property.Name, groups})
	putJSON(t, files, key+"/scenario.json", struct {
		Name, Start string
		Actions     []string
		Limits      goir.Limits
	}{q.Scenario.Name, q.Scenario.Start, q.Scenario.Actions, q.Limits})
	answer, err := q.Answer()
	require.NoError(t, err)
	putJSON(t, files, key+"/answer.json", answer)
}

func clearPositionFields(m protoreflect.Message) {
	position := (&modelirspb.Position{}).ProtoReflect().Descriptor().FullName()
	m.Range(func(f protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if f.Message() == nil || f.IsMap() {
			return true
		}
		switch {
		case f.Message().FullName() == position:
			m.Clear(f)
		case f.IsList():
			for i := range value.List().Len() {
				clearPositionFields(value.List().Get(i).Message())
			}
		default:
			clearPositionFields(value.Message())
		}
		return true
	})
}
