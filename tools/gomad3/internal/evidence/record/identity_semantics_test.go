package record

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEveryRecordProjectionInputAffectsItsIdentity(t *testing.T) {
	manifest := fullIdentityManifest()
	assertEveryProjectionInputAffectsIdentity(t, "record", recordProjectionOf(manifest))

	manifest.SimulationProfile = nil
	assertEveryProjectionInputAffectsIdentity(t, "failure", failureProjectionOf(manifest))

	manifest.SimulationProfile = fullIdentityManifest().SimulationProfile
	assertEveryProjectionInputAffectsIdentity(t, "simulation failure", failureProjectionOf(manifest))
}

func TestExecutionIdentityProjectionInclusionsAndExclusions(t *testing.T) {
	baseline := fullIdentityManifest()
	baseline.SimulationProfile = nil
	tests := []struct {
		name           string
		mutate         func(*ExecutionRecord)
		recordChanges  bool
		failureChanges bool
	}{
		{name: "schema", mutate: func(value *ExecutionRecord) { value.SchemaVersion++ }, recordChanges: true, failureChanges: true},
		{name: "artifact kind", mutate: func(value *ExecutionRecord) { value.ArtifactKind += "-changed" }},
		{name: "record hash", mutate: func(value *ExecutionRecord) { value.RecordHash = HashBytes([]byte("changed record")) }},
		{name: "creation time", mutate: func(value *ExecutionRecord) { value.CreatedAt += "-changed" }},
		{name: "campaign", mutate: func(value *ExecutionRecord) { value.CampaignID += "-changed" }},
		{name: "selection ordinal", mutate: func(value *ExecutionRecord) { value.SelectionOrdinal++ }},
		{name: "seed", mutate: func(value *ExecutionRecord) { value.Seed++ }, recordChanges: true},
		{name: "replay mode", mutate: func(value *ExecutionRecord) { value.ReplayMode += "-changed" }},
		{name: "runner", mutate: func(value *ExecutionRecord) { value.Runner.RunnerBuild += "-changed" }, recordChanges: true},
		{name: "toolchain", mutate: func(value *ExecutionRecord) { value.Toolchain.BuildKey += "-changed" }, recordChanges: true, failureChanges: true},
		{name: "target identity", mutate: func(value *ExecutionRecord) { value.Target.Kind += "-changed" }, recordChanges: true, failureChanges: true},
		{name: "target source path", mutate: func(value *ExecutionRecord) { value.Target.Source += "-changed" }},
		{name: "target artifact path", mutate: func(value *ExecutionRecord) { value.Target.File += "-changed" }},
		{name: "backend identity", mutate: func(value *ExecutionRecord) {
			value.Target.Backend.Provenance.SHA256 = HashBytes([]byte("changed provenance"))
		}, recordChanges: true, failureChanges: true},
		{name: "backend provenance path", mutate: func(value *ExecutionRecord) { value.Target.Backend.Provenance.File += "-changed" }},
		{name: "backend evidence path", mutate: func(value *ExecutionRecord) { value.Target.Backend.Evidence.File += "-changed" }},
		{name: "capability identity", mutate: func(value *ExecutionRecord) { value.Target.CapabilityManifest.Facts++ }, recordChanges: true, failureChanges: true},
		{name: "capability artifact path", mutate: func(value *ExecutionRecord) { value.Target.CapabilityManifest.File += "-changed" }},
		{name: "io inventory", mutate: func(value *ExecutionRecord) { value.IOProfile.Inventory += " " }, recordChanges: true, failureChanges: true},
		{name: "io transcript identity", mutate: func(value *ExecutionRecord) { value.IOProfile.Transcript.Records++ }, recordChanges: true, failureChanges: true},
		{name: "io transcript path", mutate: func(value *ExecutionRecord) { value.IOProfile.Transcript.File += "-changed" }},
		{name: "mount identity", mutate: func(value *ExecutionRecord) { value.IOProfile.ReadOnlyMounts.TotalBytes++ }, recordChanges: true, failureChanges: true},
		{name: "mount artifact path", mutate: func(value *ExecutionRecord) { value.IOProfile.ReadOnlyMounts.File += "-changed" }},
		{name: "choice identity", mutate: func(value *ExecutionRecord) { value.ChoiceProfile.Trace.Decisions++ }, recordChanges: true, failureChanges: true},
		{name: "choice artifact path", mutate: func(value *ExecutionRecord) { value.ChoiceProfile.Trace.File += "-changed" }},
		{name: "minimization", mutate: func(value *ExecutionRecord) { value.Minimization.Attempts++ }, recordChanges: true},
		{name: "ordinary environment", mutate: func(value *ExecutionRecord) { value.Environment[0].Value += "-changed" }, recordChanges: true, failureChanges: true},
		{name: "seed environment", mutate: func(value *ExecutionRecord) { value.Environment[1].Value += "-changed" }, recordChanges: true},
		{name: "limits", mutate: func(value *ExecutionRecord) { value.Limits.OutputBytes++ }, recordChanges: true},
		{name: "world identity", mutate: func(value *ExecutionRecord) { value.World.Final.SemanticDigest = HashBytes([]byte("changed world")) }, recordChanges: true, failureChanges: true},
		{name: "world artifact path", mutate: func(value *ExecutionRecord) { value.World.Final.File += "-changed" }},
		{name: "outcome", mutate: func(value *ExecutionRecord) { value.Outcome.Reason += "-changed" }, recordChanges: true, failureChanges: true},
		{name: "stored failure signature", mutate: func(value *ExecutionRecord) { value.Outcome.FailureSignature = HashBytes([]byte("changed failure")) }},
		{name: "replay comparison", mutate: func(value *ExecutionRecord) { *value.Outcome.ReplayMatch = !*value.Outcome.ReplayMatch }},
		{name: "retained stream identity", mutate: func(value *ExecutionRecord) {
			value.Streams.Stdout.RetainedSHA256 = HashBytes([]byte("changed retained stdout"))
		}, recordChanges: true},
		{name: "full stream identity", mutate: func(value *ExecutionRecord) {
			value.Streams.Stdout.FullSHA256 = HashBytes([]byte("changed full stdout"))
		}, recordChanges: true, failureChanges: true},
		{name: "stream artifact path", mutate: func(value *ExecutionRecord) { value.Streams.Stdout.File += "-changed" }},
		{name: "artifact file inventory", mutate: func(value *ExecutionRecord) { value.Files[0].Mode = "0644" }},
		{name: "host observation", mutate: func(value *ExecutionRecord) { value.Host.ElapsedNanos++ }},
	}

	wantRecord, wantFailure := executionIdentityDigests(t, baseline)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneExecutionRecordForIdentityTest(t, baseline)
			test.mutate(&changed)
			gotRecord, gotFailure := executionIdentityDigests(t, changed)
			if changed := gotRecord != wantRecord; changed != test.recordChanges {
				t.Fatalf("record identity changed = %t, want %t", changed, test.recordChanges)
			}
			if changed := gotFailure != wantFailure; changed != test.failureChanges {
				t.Fatalf("failure identity changed = %t, want %t", changed, test.failureChanges)
			}
		})
	}
}

func TestExecutionIdentityRetainsNilEmptyAndPointerPresence(t *testing.T) {
	tests := []struct {
		name            string
		nil             func(*ExecutionRecord)
		empty           func(*ExecutionRecord)
		recordDistinct  bool
		failureDistinct bool
	}{
		{name: "target argv", nil: func(value *ExecutionRecord) { value.Target.Argv = nil }, empty: func(value *ExecutionRecord) { value.Target.Argv = []string{} }, recordDistinct: true, failureDistinct: true},
		{name: "target build tags", nil: func(value *ExecutionRecord) { value.Target.BuildTags = nil }, empty: func(value *ExecutionRecord) { value.Target.BuildTags = []string{} }, recordDistinct: true, failureDistinct: true},
		{name: "target adapters", nil: func(value *ExecutionRecord) { value.Target.Adapters = nil }, empty: func(value *ExecutionRecord) { value.Target.Adapters = []TargetAdapter{} }, recordDistinct: true, failureDistinct: true},
		{name: "target compatibility", nil: func(value *ExecutionRecord) { value.Target.Compatibility = nil }, empty: func(value *ExecutionRecord) { value.Target.Compatibility = []CompatibilityPack{} }, recordDistinct: true, failureDistinct: true},
		{name: "build settings", nil: func(value *ExecutionRecord) { value.Target.BuildInfo.Settings = nil }, empty: func(value *ExecutionRecord) { value.Target.BuildInfo.Settings = []BuildSetting{} }, recordDistinct: true, failureDistinct: true},
		{name: "mount mappings", nil: func(value *ExecutionRecord) { value.IOProfile.ReadOnlyMounts.Mappings = nil }, empty: func(value *ExecutionRecord) { value.IOProfile.ReadOnlyMounts.Mappings = []string{} }},
		{name: "environment", nil: func(value *ExecutionRecord) { value.Environment = nil }, empty: func(value *ExecutionRecord) { value.Environment = []Environment{} }, recordDistinct: true},
		{name: "world adapters", nil: func(value *ExecutionRecord) { value.World.Adapters = nil }, empty: func(value *ExecutionRecord) { value.World.Adapters = []WorldAdapter{} }},
		{name: "minimization reductions", nil: func(value *ExecutionRecord) { value.Minimization.Accepted = nil }, empty: func(value *ExecutionRecord) { value.Minimization.Accepted = []MinimizationReduction{} }},
	}
	baseline := fullIdentityManifest()
	baseline.SimulationProfile = nil
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nilValue := cloneExecutionRecordForIdentityTest(t, baseline)
			emptyValue := cloneExecutionRecordForIdentityTest(t, baseline)
			test.nil(&nilValue)
			test.empty(&emptyValue)
			nilRecord, nilFailure := executionIdentityDigests(t, nilValue)
			emptyRecord, emptyFailure := executionIdentityDigests(t, emptyValue)
			if distinct := nilRecord != emptyRecord; distinct != test.recordDistinct {
				t.Fatalf("record identity distinguishes nil and empty = %t, want %t", distinct, test.recordDistinct)
			}
			if distinct := nilFailure != emptyFailure; distinct != test.failureDistinct {
				t.Fatalf("failure identity distinguishes nil and empty = %t, want %t", distinct, test.failureDistinct)
			}
		})
	}

	for _, test := range []struct {
		name   string
		remove func(*ExecutionRecord)
	}{
		{name: "backend", remove: func(value *ExecutionRecord) { value.Target.Backend = nil }},
		{name: "capability manifest", remove: func(value *ExecutionRecord) { value.Target.CapabilityManifest = nil }},
		{name: "io transcript", remove: func(value *ExecutionRecord) { value.IOProfile.Transcript = nil }},
		{name: "read-only mounts", remove: func(value *ExecutionRecord) { value.IOProfile.ReadOnlyMounts = nil }},
		{name: "choice profile", remove: func(value *ExecutionRecord) { value.ChoiceProfile = nil }},
		{name: "outcome signal", remove: func(value *ExecutionRecord) { value.Outcome.Signal = nil }},
		{name: "outcome deadline", remove: func(value *ExecutionRecord) { value.Outcome.Deadline = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneExecutionRecordForIdentityTest(t, baseline)
			test.remove(&changed)
			wantRecord, wantFailure := executionIdentityDigests(t, baseline)
			gotRecord, gotFailure := executionIdentityDigests(t, changed)
			if gotRecord == wantRecord || gotFailure == wantFailure {
				t.Fatal("identity collapsed pointer presence")
			}
		})
	}
}

func fullIdentityManifest() ExecutionRecord {
	manifest := manifestFixture()
	manifest.Target.Backend = &BackendMetadata{
		Name: "native", ReplayMode: "exact",
		Provenance: BackendPayload{Schema: "provenance/v1", File: "backend/provenance.json", SHA256: HashBytes([]byte("provenance")), Bytes: 10},
		Evidence:   &BackendPayload{Schema: "evidence/v1", File: "backend/evidence.bin", SHA256: HashBytes([]byte("evidence")), Bytes: 8},
	}
	manifest.Target.BuildInfo.MainModule = "example.test/project"
	manifest.Target.BuildInfo.Settings = []BuildSetting{{Key: "GOOS", Value: "darwin"}}
	manifest.Target.CapabilityManifest = &TargetCapabilityManifest{
		Schema: "capabilities/v1", File: "capabilities.json", SHA256: HashBytes([]byte("capabilities")), Bytes: 12, Facts: 2,
		ProducerImplementationSHA256: HashBytes([]byte("producer")), GuardImplementationSHA256: HashBytes([]byte("guard")), CapabilityUniverseSHA256: HashBytes([]byte("universe")),
	}
	manifest.IOProfile.Transcript = &IOTranscript{Schema: "transcript/v1", File: "io/transcript.bin", SHA256: HashBytes([]byte("transcript")), Bytes: 10, Records: 2}
	manifest.IOProfile.ReadOnlyMounts = &ReadOnlyMounts{
		Schema: "mounts/v1", File: "io/mounts.json", SHA256: HashBytes([]byte("mounts")), Bytes: 11, Entries: 2, NotExist: 1, TotalBytes: 4,
		Mappings: []string{"/schema"}, Limits: ReadOnlyMountLimits{PathBytes: 128, Requests: 9, Files: 8, DirectoryEntries: 7, SingleFileBytes: 6, TotalBytes: 5},
	}
	manifest.ChoiceProfile = &ChoiceProfile{
		Name: "choices/v1", ImplementationSHA256: HashBytes([]byte("choices")),
		Trace: ChoiceTrace{Schema: "choice-trace/v1", File: "choices.bin", SHA256: HashBytes([]byte("trace")), Bytes: 13, Records: 5, BranchingRecords: 3, TerminalState: "complete", Limit: 21, TapeSHA256: HashBytes([]byte("tape")), Decisions: 4},
	}
	manifest.SimulationProfile = &SimulationProfile{
		Name: "simulation/v1", ControllerSHA256: HashBytes([]byte("controller")), ExecutionSHA256: HashBytes([]byte("execution")), CandidateSHA256: HashBytes([]byte("candidate")), OutcomeSHA256: HashBytes([]byte("outcome")), FailureSHA256: HashBytes([]byte("simulation failure")),
		Plan:   SimulationPlan{Schema: "simulation-plan/v1", File: "simulation/plan.json", SHA256: HashBytes([]byte("plan")), Bytes: 14},
		Record: SimulationRecord{Schema: "simulation-record/v1", File: "simulation/record.json", SHA256: HashBytes([]byte("simulation record")), Bytes: 15, Limit: 16},
	}
	manifest.Minimization = &Minimization{
		Schema: "minimization/v1", ImplementationSHA256: HashBytes([]byte("minimizer")), ParentRecordHash: HashBytes([]byte("parent record")), ParentFailureSignature: HashBytes([]byte("parent failure")), OriginalCandidateSHA256: HashBytes([]byte("original candidate")), FinalCandidateSHA256: HashBytes([]byte("final candidate")), AttemptBudget: 20, Attempts: 10, OriginalForcedDecisions: 8, FinalForcedDecisions: 3,
		Accepted:     []MinimizationReduction{{Kind: "choice-prefix", BeforeSHA256: HashBytes([]byte("before")), AfterSHA256: HashBytes([]byte("after")), Removed: []MinimizationDecision{{Dimension: "choice", Ordinal: 2, Identity: HashBytes([]byte("removed"))}}}},
		Predicate:    MinimizationPredicate{FailureSignature: HashBytes([]byte("predicate failure")), Domain: "target", Reason: "nonzero_exit", Termination: "exit", ReplayMatch: true, ChoiceReplay: "exact", SimulationReplay: "exact"},
		ChoicePrefix: &ChoicePrefixMinimization{Original: ChoicePrefixControl{Length: 8, SHA256: HashBytes([]byte("original prefix"))}, Final: ChoicePrefixControl{Length: 3, SHA256: HashBytes([]byte("final prefix"))}},
	}
	manifest.Limits.IOTranscriptBytes = 23
	manifest.Limits.ChoiceTraceBytes = 24
	manifest.World.Adapters = []WorldAdapter{{Schema: "world-adapter/v1", InitialDigest: HashBytes([]byte("initial adapter")), FinalDigest: HashBytes([]byte("final adapter"))}}
	manifest.World.Terminal.Detail = "terminal detail"
	signal, deadline, replayMatch := "signal", "deadline", true
	manifest.Outcome.Signal = &signal
	manifest.Outcome.Deadline = &deadline
	manifest.Outcome.ReplayMatch = &replayMatch
	manifest.Streams.Stdout.DiscardedBytes = 1
	manifest.Streams.Stdout.Truncated = true
	manifest.Streams.Stderr.DiscardedBytes = 2
	manifest.Streams.Stderr.Truncated = true
	return manifest
}

func executionIdentityDigests(t *testing.T, manifest ExecutionRecord) (SHA256, SHA256) {
	t.Helper()
	return projectionIdentity(t, "gomad3-execution-record-v1", recordProjectionOf(manifest)), projectionIdentity(t, "gomad3-failure-signature-v1", failureProjectionOf(manifest))
}

func projectionIdentity(t *testing.T, domain string, projection any) SHA256 {
	t.Helper()
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	return DomainHash(domain, encoded)
}

func cloneExecutionRecordForIdentityTest(t *testing.T, manifest ExecutionRecord) ExecutionRecord {
	t.Helper()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var cloned ExecutionRecord
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

type projectionStep struct {
	kind  byte
	index int
}

func assertEveryProjectionInputAffectsIdentity(t *testing.T, name string, projection any) {
	t.Helper()
	root := reflect.ValueOf(projection)
	baseline := projectionIdentity(t, "projection-test", projection)
	mutations := 0
	var walk func(reflect.Value, []projectionStep, string)
	walk = func(value reflect.Value, steps []projectionStep, path string) {
		switch value.Kind() {
		case reflect.Struct:
			for index := 0; index < value.NumField(); index++ {
				field := value.Type().Field(index)
				walk(value.Field(index), appendProjectionStep(steps, projectionStep{kind: 'f', index: index}), joinProjectionPath(path, field.Name))
			}
		case reflect.Pointer:
			if value.IsNil() {
				t.Fatalf("%s projection pointer %s is nil; full fixture must exercise pointer presence", name, path)
			}
			assertProjectionMutationChanges(t, root, steps, path+" presence", baseline, func(target reflect.Value) { target.SetZero() })
			mutations++
			walk(value.Elem(), appendProjectionStep(steps, projectionStep{kind: 'p'}), path)
		case reflect.Slice:
			if value.IsNil() || value.Len() == 0 {
				t.Fatalf("%s projection slice %s is not populated; full fixture must exercise nil/empty and elements", name, path)
			}
			nilRoot := cloneProjectionValue(root)
			projectionValueAt(nilRoot, steps).SetZero()
			emptyRoot := cloneProjectionValue(root)
			empty := projectionValueAt(emptyRoot, steps)
			empty.Set(reflect.MakeSlice(empty.Type(), 0, 0))
			nilIdentity := projectionIdentity(t, "projection-test", nilRoot.Interface())
			emptyIdentity := projectionIdentity(t, "projection-test", emptyRoot.Interface())
			if nilIdentity == baseline || emptyIdentity == baseline || nilIdentity == emptyIdentity {
				t.Fatalf("%s projection %s collapsed populated, nil, or empty slice semantics", name, path)
			}
			mutations += 2
			for index := 0; index < value.Len(); index++ {
				walk(value.Index(index), appendProjectionStep(steps, projectionStep{kind: 's', index: index}), path)
			}
		case reflect.String:
			assertProjectionMutationChanges(t, root, steps, path, baseline, func(target reflect.Value) { target.SetString(target.String() + "-changed") })
			mutations++
		case reflect.Bool:
			assertProjectionMutationChanges(t, root, steps, path, baseline, func(target reflect.Value) { target.SetBool(!target.Bool()) })
			mutations++
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			assertProjectionMutationChanges(t, root, steps, path, baseline, func(target reflect.Value) { target.SetUint(target.Uint() + 1) })
			mutations++
		default:
			t.Fatalf("unsupported %s projection input %s of kind %s", name, path, value.Kind())
		}
	}
	walk(root, nil, "")
	if mutations == 0 {
		t.Fatalf("%s projection had no independently exercised inputs", name)
	}
}

func assertProjectionMutationChanges(t *testing.T, root reflect.Value, steps []projectionStep, path string, baseline SHA256, mutate func(reflect.Value)) {
	t.Helper()
	changed := cloneProjectionValue(root)
	mutate(projectionValueAt(changed, steps))
	if got := projectionIdentity(t, "projection-test", changed.Interface()); got == baseline {
		t.Fatalf("projection input %s did not affect its identity", path)
	}
}

func cloneProjectionValue(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Struct:
		cloned := reflect.New(value.Type()).Elem()
		for index := 0; index < value.NumField(); index++ {
			cloned.Field(index).Set(cloneProjectionValue(value.Field(index)))
		}
		return cloned
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.New(value.Type().Elem())
		cloned.Elem().Set(cloneProjectionValue(value.Elem()))
		return cloned
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(cloneProjectionValue(value.Index(index)))
		}
		return cloned
	default:
		return value
	}
}

func projectionValueAt(root reflect.Value, steps []projectionStep) reflect.Value {
	value := root
	for _, step := range steps {
		switch step.kind {
		case 'f':
			value = value.Field(step.index)
		case 'p':
			value = value.Elem()
		case 's':
			value = value.Index(step.index)
		default:
			panic("unknown projection path step")
		}
	}
	return value
}

func appendProjectionStep(steps []projectionStep, step projectionStep) []projectionStep {
	cloned := append([]projectionStep(nil), steps...)
	return append(cloned, step)
}

func joinProjectionPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}
