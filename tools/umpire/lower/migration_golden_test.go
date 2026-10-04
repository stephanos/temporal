package lower_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	runtime "go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"go.temporal.io/server/tools/umpire/lower"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
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

// artifactModel records what m lowers to through its producer, which has checked m once already:
// the Query receipts it lowers by also key the assessments.
func artifactModel(t *testing.T, files map[string][]byte, key string, m *umpirespb.Model, producer *lower.Producer) {
	t.Helper()
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
		factory, err := conformance.Prepare(m, producer.QueryKey(query.GetName()), c, conformance.Limits{MaxEvents: 2048, MaxProperties: 16, MaxDuration: time.Minute, MaxCandidates: 1 << 16, MaxWork: 1 << 22, MaxReadings: 1 << 22})
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
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(umpiremodel.WithoutTotals(c.Model))
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

func assessmentIdentity(t *testing.T, model *umpirespb.Model) string {
	t.Helper()
	bare := umpiremodel.WithoutTotals(model)
	bare.Source = ""
	// Position fields must be removed, not replaced by empty messages, to preserve the wire identity.
	clearPositionFields(bare.ProtoReflect())
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(bare)
	require.NoError(t, err)
	return "goir.model/v1:sha256:" + golden.Digest(append([]byte("goir.model/v1"), encoded...))
}

func captureArtifacts(t *testing.T, cfg golden.Config, models map[string]*umpirespb.Model, project bool) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	generatedIR := t.TempDir()
	// Each IR Model is generated through the Producer its artifacts were lowered by, which binds and
	// checks it once for both; the Model loaded back from its file must be that one.
	type lowered struct {
		model    *umpirespb.Model
		producer *lower.Producer
	}
	producers := map[string]lowered{}
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
		producer, err := lower.NewProducer(model)
		require.NoError(t, err, key)
		artifactModel(t, files, key, model, producer)
		if strings.HasPrefix(key, "ir/") {
			require.NotContains(t, producers, filepath.Base(path))
			producers[filepath.Base(path)] = lowered{model, producer}
			require.NoError(t, os.WriteFile(filepath.Join(generatedIR, filepath.Base(path)), files["inputs/"+key], 0644))
		}
	}
	generated, err := lower.GenerateCasesWith(generatedIR, func(path string, loaded *umpirespb.Model) (*lower.Producer, error) {
		p, ok := producers[filepath.Base(path)]
		if !ok || !proto.Equal(p.model, loaded) {
			return nil, fmt.Errorf("%s is not a Model whose artifacts were lowered", path)
		}
		return p.producer, nil
	})
	require.NoError(t, err)
	for name, encoded := range generated {
		files["generated/"+name] = encoded
	}
	return files
}

func originalArtifactInputs(t *testing.T) (golden.Config, map[string]*umpirespb.Model) {
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
		original := new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(expected["original/inputs/"+strings.TrimPrefix(path, "model/scalav2/")], original), path)
		mapped, err := cfg.MatchAt(golden.OriginalKey(path), original, current)
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
	// A reduced or retired fixture is no input, so none of its goldens is compared.
	require.NoError(t, compareMigrationArtifactInventory(cfg.Unreduced(expected), actual))
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
	require.NoError(t, compareMigrationArtifactInventory(expected, migrationJobArtifacts(t)))
}

func compareMigrationArtifactInventory(expected, actual map[string][]byte) error {
	projected, err := projectJobArtifacts(expected, actual)
	if err != nil {
		return err
	}
	return golden.Compare(projected, actual)
}

func TestMigrationArtifactInventoryRejectsInactiveChanges(t *testing.T) {
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	actual := maps.Clone(expected)
	maps.Copy(actual, migrationJobArtifacts(t))
	require.NoError(t, compareMigrationArtifactInventory(expected, actual))
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
				require.Error(t, compareMigrationArtifactInventory(changed, actual))
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
				require.Error(t, compareMigrationArtifactInventory(changed, actual))
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
	model := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(baseline["original/inputs/ir/nexus-caller.json"], model))
	lower.MigrationComparativeModel(t, model)
	// The functional Queries are the ones the legacy producer realized; a verify Query has nothing to
	// realize and no oracle.
	var functional []*umpirespb.Query
	for _, declaration := range model.GetQueries() {
		if declaration.GetForm() == umpirespb.Query_FORM_FIND {
			functional = append(functional, declaration)
		}
	}
	require.Len(t, functional, 7)
	for _, declaration := range functional {
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

func captureLegacyQuery(t *testing.T, files map[string][]byte, key string, q *umpiremodel.Query) {
	t.Helper()
	table, err := q.Scenario.Machine.Table()
	require.NoError(t, err)
	putJSON(t, files, key+"/table.json", table)
	putJSON(t, files, key+"/ids.json", table.IDs())
	putJSON(t, files, key+"/fingerprint.json", table.TargetFingerprint())
	putJSON(t, files, key+"/claims.json", table.Claims())
	fields := map[string][]umpiremodel.Atom{}
	for _, state := range table.States {
		fields[state] = table.FieldValues(state)
	}
	putJSON(t, files, key+"/fields.json", fields)
	groups, err := q.Property.Lower()
	require.NoError(t, err)
	putJSON(t, files, key+"/property.json", struct {
		Name   string
		Groups []umpiremodel.Group
	}{q.Property.Name, groups})
	putJSON(t, files, key+"/scenario.json", struct {
		Name, Start string
		Actions     []string
		Limits      umpiremodel.Limits
	}{q.Scenario.Name, q.Scenario.Start, q.Scenario.Actions, q.Limits})
	answer, err := q.Answer()
	require.NoError(t, err)
	putJSON(t, files, key+"/answer.json", answer)
}

func clearPositionFields(m protoreflect.Message) {
	position := (&umpirespb.Position{}).ProtoReflect().Descriptor().FullName()
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

// TestMigrationProjectionKeepsLoweredCases checks that the current IR of every Model, which MatchAt
// admits, without the generated claims the original baseline's delta lists, lowers to the mapped
// goldens' Cases: every Query Case byte for byte, every exploration Case
// except its IDs, which carry the digest of the whole candidate Model. It checks it again for the
// Nexus caller changed the way the IR-changing tasks change it. A changed step passes the IR
// comparison, which reads no Function, and fails on its Cases.
func TestMigrationProjectionKeepsLoweredCases(t *testing.T) {
	const path, key = "model/scalav2/ir/nexus-caller.json", "mapped/ir/nexus-caller.json"
	const kernel = "temporal.nexuscaller.kernel.Protocol$.completeStep"
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	cfg, inputs := originalArtifactInputs(t)
	for _, input := range slices.Sorted(maps.Keys(inputs)) {
		t.Run(strings.TrimPrefix(input, "model/scalav2/"), func(t *testing.T) {
			ikey := "mapped/" + strings.TrimPrefix(input, "model/scalav2/")
			// The generated claims the original baseline's delta lists are compared there: their Cases
			// with the new Cases it lists (TestOriginalBaselineCases).
			ungenerated, err := golden.Ungenerated(golden.OriginalKey(input), inputs[input])
			require.NoError(t, err)
			actual, ids := lowerMigrationCases(t, ikey, ungenerated)
			require.NoError(t, compareLoweredCases(cfg, expected, actual, ids, ikey))
		})
	}

	original := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(expected["original/inputs/ir/nexus-caller.json"], original))
	moved := "temporal.nexuscaller.Protocol$.completeStep"
	if i := slices.IndexFunc(cfg.Projection.Functions, func(s golden.Substitution) bool { return s.Old == kernel }); i >= 0 {
		moved = cfg.Projection.Functions[i].New
	} else {
		cfg.Projection.Functions = append(cfg.Projection.Functions, golden.Substitution{Old: kernel, New: moved})
	}
	// Each substitution renames a Function of the frozen IR that declares it, the Nexus caller's or
	// another's.
	originals, err := golden.FrozenModels(expected, "original/inputs/ir/")
	require.NoError(t, err)
	require.NoError(t, cfg.FunctionsRenamed(originals))
	require.NoError(t, cfg.TypesRenamed(map[string]*umpirespb.Model{path: original}))
	line := regexp.MustCompile(`"line":\s*([0-9]+)`)
	encoded, err := protojson.Marshal(inputs[path])
	require.NoError(t, err)
	text := line.ReplaceAllStringFunc(string(encoded), func(at string) string {
		n, err := strconv.Atoi(line.FindStringSubmatch(at)[1])
		require.NoError(t, err)
		return `"line":` + strconv.Itoa(n+3)
	})
	text = strings.ReplaceAll(strings.ReplaceAll(text, `"_$1"`, `"placeholder"`), strconv.Quote(kernel), strconv.Quote(moved))
	admitted := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal([]byte(text), admitted))
	mapped, err := cfg.Match(original, admitted)
	require.NoError(t, err)
	require.True(t, mapped, "the projected IR selects the mapped variant")
	actual, ids := lowerMigrationCases(t, key, admitted)
	require.Contains(t, ids, key+"/queries/retry")
	require.Contains(t, ids, key+"/explorations/nexusDeadlines/000")
	require.NoError(t, compareLoweredCases(cfg, expected, actual, ids, key))

	t.Run("exploration Case with a changed byte", func(t *testing.T) {
		changed := maps.Clone(actual)
		name := key + "/explorations/nexusDeadlines/000/case.json"
		require.Contains(t, string(changed[name]), `"major":1`)
		changed[name] = []byte(strings.Replace(string(changed[name]), `"major":1`, `"major":2`, 1))
		require.Error(t, compareLoweredCases(cfg, expected, changed, ids, key))
	})
	t.Run("Query Case with a changed ID", func(t *testing.T) {
		changed, changedIDs := maps.Clone(actual), maps.Clone(ids)
		name := key + "/queries/retry/case.json"
		id := "temporal.case.scala.nexus-caller.retry"
		require.Contains(t, string(changed[name]), id)
		changed[name] = []byte(strings.ReplaceAll(string(changed[name]), id, id+"Again"))
		changedIDs[key+"/queries/retry"] = "retryAgain"
		require.Error(t, compareLoweredCases(cfg, expected, changed, changedIDs, key))
	})
	// The IR comparison reads no Function, so a flipped guard passes it; the Queries whose Cases it
	// changes no longer lower to the goldens' Cases.
	t.Run("changed step", func(t *testing.T) {
		changed := proto.CloneOf(admitted)
		for _, f := range changed.GetFunctions() {
			if f.GetName() == moved {
				guard := f.GetBody().GetIf()
				require.NotNil(t, guard)
				guard.Then, guard.Else = guard.Else, guard.Then
			}
		}
		_, err := cfg.Match(original, changed)
		require.NoError(t, err)
		lowered, loweredIDs := lowerMigrationCases(t, key, changed)
		require.Error(t, compareLoweredCases(cfg, expected, lowered, loweredIDs, key))
	})
}

// lowerMigrationCases lowers every Query and every exploration candidate of m as the goldens do, keyed
// by the directory of its Case under key, with the varying part of the Case's IDs.
func lowerMigrationCases(t *testing.T, key string, m *umpirespb.Model) (actual map[string][]byte, ids map[string]string) {
	t.Helper()
	actual, ids = map[string][]byte{}, map[string]string{}
	producer, err := lower.NewProducer(m)
	require.NoError(t, err)
	set := "scala." + strings.TrimSuffix(filepath.Base(key), ".json")
	for _, query := range m.GetQueries() {
		qkey := key + "/queries/" + query.GetName()
		lowered, err := producer.Lower(query.GetName(), cp.IdentityFor("temporal.case", set, query.GetName()))
		if err == nil && lowered.Case != nil {
			putCase(t, actual, qkey, lowered.Case)
			ids[qkey] = query.GetName()
		}
		if query.GetExploration() == nil {
			continue
		}
		plan, err := explore.New(m, query.GetExploration().GetName())
		require.NoError(t, err)
		lowerCandidate := func(ckey string, c *explore.Candidate) {
			if c.Rejection == "" {
				putCase(t, actual, ckey, c.Case)
				ids[ckey] = c.Digest
			}
		}
		for i, candidate := range plan.Candidates {
			ckey := fmt.Sprintf("%s/explorations/%s/%03d", key, plan.Name, i)
			lowerCandidate(ckey, candidate)
			for index := range max(0, len(candidate.Actions)-1) {
				if reduced, err := plan.Reduce(candidate, index); err == nil {
					lowerCandidate(fmt.Sprintf("%s/reduce-%03d", ckey, index), reduced)
				}
			}
		}
	}
	return actual, ids
}

// compareLoweredCases compares the Case, Program and Contract files of each lowered Case in actual,
// keyed by its directory in ids with the varying part of its IDs, with the goldens' under the
// projection and the source path renames, and requires the goldens to hold no other Case under key.
// A Query Case's identity is compared too, derived again from the golden Case when a rename changed it. An exploration Case's identity must exist on both sides; both of its fields digest
// the Case's bytes, IDs included, so neither is compared.
func compareLoweredCases(cfg golden.Config, expected, actual map[string][]byte, ids map[string]string, key string) error {
	want, got := map[string][]byte{}, map[string][]byte{}
	for dir, id := range ids {
		kind, wantID := golden.QueryCase, id
		files := []string{"case.json", "program.json", "contract.json", "identity.json"}
		if strings.Contains(dir, "/explorations/") {
			var candidate struct{ Digest string }
			if err := json.Unmarshal(expected[dir+"/candidate.json"], &candidate); err != nil {
				return fmt.Errorf("%s: %w", dir, err)
			}
			kind, wantID, files = golden.ExplorationCase, candidate.Digest, files[:3]
			for _, side := range []map[string][]byte{expected, actual} {
				if _, ok := side[dir+"/identity.json"]; !ok {
					return fmt.Errorf("no identity for %s", dir)
				}
			}
		}
		for _, file := range files {
			name := dir + "/" + file
			original, ok := expected[name]
			if !ok {
				return fmt.Errorf("no golden %s", name)
			}
			var err error
			current := actual[name]
			// A path under a merged directory names the directory on both sides, a file split out of
			// another names that file, and a Case whose paths that changes is identified as putCase
			// identifies it.
			if file != "identity.json" {
				original = cfg.MergeSources(cfg.RenameSources(original))
				current = cfg.MergeSources(cfg.UnsplitSources(current))
			} else {
				if renamed := cfg.MergeSources(cfg.RenameSources(expected[dir+"/case.json"])); !bytes.Equal(renamed, expected[dir+"/case.json"]) {
					if original, err = renamedIdentity(renamed); err != nil {
						return fmt.Errorf("%s: %w", name, err)
					}
				}
				if merged := cfg.MergeSources(cfg.UnsplitSources(actual[dir+"/case.json"])); !bytes.Equal(merged, actual[dir+"/case.json"]) {
					if current, err = renamedIdentity(merged); err != nil {
						return fmt.Errorf("%s: %w", name, err)
					}
				}
			}
			if want[name], err = cfg.Projection.Case(kind, original, wantID); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if got[name], err = cfg.Projection.Case(kind, current, id); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	for name := range expected {
		if strings.HasSuffix(name, "/case.json") && strings.HasPrefix(name, key+"/") {
			if _, ok := ids[strings.TrimSuffix(name, "/case.json")]; !ok {
				return fmt.Errorf("golden %s was not lowered", name)
			}
		}
	}
	return golden.Compare(want, got)
}

// renamedIdentity is the identity putCase records for a Case whose source paths were renamed, merged
// or unsplit.
func renamedIdentity(encoded []byte) ([]byte, error) {
	identity, err := recordedrun.CaseIdentity(encoded)
	if err != nil {
		return nil, err
	}
	c := new(testpilotspb.Case)
	if err := protojson.Unmarshal(encoded, c); err != nil {
		return nil, err
	}
	fingerprint, err := runtime.CaseFingerprint(c)
	if err != nil {
		return nil, err
	}
	return golden.JSON(struct{ Canonical, Fingerprint string }{identity, fingerprint})
}
