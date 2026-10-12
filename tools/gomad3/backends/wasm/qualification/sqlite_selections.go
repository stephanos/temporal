package qualification

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/backends/wasm/simulation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3sim"
)

// ValidateSQLiteSelectionCoverage checks coverage against an actual complete
// native enumeration. It does not enumerate states, execute guests or qualify
// reports: each selected state's existing Result still needs sealed selection
// binding, deterministic retained replay and measured cost/budget acceptance.
func ValidateSQLiteSelectionCoverage(enumeration volume.CrashEnumeration, selected []volume.CrashState) error {
	if !enumeration.Complete || enumeration.Capacity != "" || len(enumeration.States) == 0 || len(enumeration.States) > 32 || len(selected) != len(enumeration.States) {
		return errors.New("complete bounded native SQLite crash-state coverage required")
	}
	expected := make(map[string]volume.CrashState, len(enumeration.States))
	for _, state := range enumeration.States {
		if state.Identity == "" || state.Volume == "" || state.PendingSHA256 == "" {
			return errors.New("native SQLite crash state lacks identity")
		}
		if _, duplicate := expected[state.Identity]; duplicate {
			return errors.New("duplicate native SQLite crash state")
		}
		expected[state.Identity] = state
	}
	for _, state := range selected {
		native, found := expected[state.Identity]
		if !found || !reflect.DeepEqual(native, state) {
			return errors.New("extra, repeated or altered selected SQLite crash state")
		}
		delete(expected, state.Identity)
	}
	if len(expected) != 0 {
		return errors.New("missing selected SQLite crash state")
	}
	return nil
}

func validateSQLiteBootSelections(bindings []simulation.RetainedStorageCrashSelection, state volume.CrashState) error {
	if len(bindings) != 3 {
		return errors.New("SQLite selection requires exactly three sealed boots")
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		boot := string(binding.Boot)
		if seen[boot] || boot != "setup-live" && boot != "append-crash" && boot != "recover" {
			return errors.New("unknown or repeated SQLite selected-state boot")
		}
		seen[boot] = true
		limits := binding.Limits
		if limits == nil || limits.States == 0 || limits.States > 32 || limits.Operations == 0 || limits.Operations > 8 || limits.Depth == 0 || limits.Depth > 8 || limits.Bytes == 0 || limits.Bytes > 4<<20 || limits.Work == 0 || limits.Work > 1<<16 {
			return errors.New("SQLite selected-state native enumeration limits missing or unsupported")
		}
		if boot != "append-crash" {
			if binding.Selection != nil {
				return errors.New("SQLite selection applied to a different lifecycle phase")
			}
			continue
		}
		operations, exists := binding.Selection[state.Volume]
		if !exists || len(binding.Selection) != 1 || !reflect.DeepEqual(operations, state.SelectedOperations) {
			return errors.New("SQLite selected-state operations differ from sealed append boot")
		}
	}
	return nil
}

// ValidateSQLiteSelectionResults gates supplementary existing Results, not a
// second report. Every exact native full state needs actual qualified recovery,
// sealed selection inputs, matched source-less replay, costs and terminal receipt.
func ValidateSQLiteSelectionResults(source Result, sourceReceipt PublicationReceipt, results []Result, receipts ...PublicationReceipt) error {
	if err := ValidateResult(source); err != nil {
		return err
	}
	if source.Manifest.Provisional || !source.Qualification.Qualified || source.NativeCrashSelection != nil || source.Cost == nil || source.Budget == nil || !source.Budget.WithinBudget {
		return errors.New("actual frozen native source catalogue support/cost/budget required")
	}
	enumeration, err := MatchedNativeCrashEnumeration(source, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
	if err != nil {
		return err
	}
	encodedSource, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return err
	}
	sourceHash := record.HashBytes(append(encodedSource, '\n'))
	if _, err := ValidatePublicationReceipt("result-"+strings.TrimPrefix(string(sourceHash), "sha256:")+".json", source, sourceReceipt); err != nil {
		return err
	}
	if len(results) == 0 || len(results) != len(receipts) || len(results) != len(enumeration.States) || len(results) > 32 {
		return errors.New("one measured Result/receipt for every native SQLite crash state required")
	}
	selected := make([]volume.CrashState, 0, len(results))
	processes := map[uint64]bool{source.OperatorProcessID: true}
	for _, result := range results {
		if result.NativeCrashSelection == nil {
			return errors.New("selected-state Result lacks exact native full state")
		}
		selected = append(selected, *result.NativeCrashSelection)
	}
	if err := ValidateSQLiteSelectionCoverage(enumeration, selected); err != nil {
		return err
	}
	for index, result := range results {
		if err := ValidateResult(result); err != nil {
			return err
		}
		if result.NativeCrashSourceResultSHA256 != sourceHash {
			return errors.New("selected-state source Result reference changed")
		}
		if result.Manifest.ID != "sqlite-durable-selection-"+strings.TrimPrefix(result.NativeCrashSelection.Identity, "sha256:") {
			return errors.New("selected-state Result cell identity changed")
		}
		if err := Matched(source.Manifest, result.Manifest); err != nil {
			return err
		}
		if source.Manifest.CompilerSHA256 != result.Manifest.CompilerSHA256 || source.Manifest.HelperSHA256 != result.Manifest.HelperSHA256 || source.Manifest.EngineSHA256 != result.Manifest.EngineSHA256 || source.Manifest.RuntimeSHA256 != result.Manifest.RuntimeSHA256 || source.Manifest.ModelSHA256 != result.Manifest.ModelSHA256 {
			return errors.New("selected-state source candidate/toolchain/engine/model mismatch")
		}
		if result.Manifest.Provisional || result.Manifest.Workload != "sqlite" || result.Manifest.Cell != "success" || result.PreparationReusedWithinProducer || result.OperatorProcessID == 0 || processes[result.OperatorProcessID] || !result.Qualification.Qualified || !result.Qualification.Deterministic || !result.Qualification.TargetSuccess || result.Qualification.Failure != nil || result.Cost == nil || result.CostFailure != "" || result.Budget == nil || !result.Budget.WithinBudget || result.BudgetFailure != "" {
			return errors.New("selected-state fresh frozen support/cost/budget gate failed")
		}
		processes[result.OperatorProcessID] = true
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		path := "result-" + strings.TrimPrefix(string(record.HashBytes(append(encoded, '\n'))), "sha256:") + ".json"
		if _, err := ValidatePublicationReceipt(path, result, receipts[index]); err != nil {
			return err
		}
		actual, err := MatchedNativeCrashEnumeration(result, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
		if err != nil || !reflect.DeepEqual(actual, enumeration) {
			return errors.New("selected-state native enumeration differs from actual source catalogue")
		}
		for _, execution := range result.Qualification.Executions {
			if execution.Replay == nil || !execution.Replay.Attempted || !execution.Replay.Match || execution.ArtifactPath == "" {
				return errors.New("selected-state actual retained replay missing")
			}
			if err := validateSQLiteArtifactSelection(execution.ArtifactPath, *result.NativeCrashSelection); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSQLiteArtifactSelection(path string, state volume.CrashState) (retErr error) {
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, opened.Close()) }()
	manifest := opened.Manifest()
	if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || !simulation.IsRetainedSchema(manifest.Target.Backend.Evidence.Schema) {
		return errors.New("selected-state Artifact lacks sealed cluster evidence")
	}
	ref := *manifest.Target.Backend.Evidence
	data, err := opened.ReadPayload(ref.File, 512<<20)
	if err != nil {
		return err
	}
	bindings, err := simulation.RetainedStorageCrashSelections(target.BackendPayload{Reference: ref, Data: data})
	if err != nil {
		return err
	}
	if err := validateSQLiteBootSelections(bindings, state); err != nil {
		return err
	}
	observed, err := simulation.RetainedRecord(target.BackendPayload{Reference: ref, Data: data})
	if err != nil {
		return err
	}
	return validateSQLiteRecoveredMarker(observed, state)
}

func validateSQLiteRecoveredMarker(observed gomad3sim.ClusterRecord, state volume.CrashState) error {
	var expected *volume.CrashEntry
	for index := range state.Entries {
		if state.Entries[index].Path == "/marker" {
			expected = &state.Entries[index]
			break
		}
	}
	for _, snapshot := range observed.Volumes.Snapshot.Volumes {
		if string(snapshot.Volume) != state.Volume || snapshot.Node != "sqlite" {
			continue
		}
		var marker *gomad3sim.VolumeEntrySnapshot
		for index := range snapshot.Persisted {
			if snapshot.Persisted[index].Path == "/marker" {
				marker = &snapshot.Persisted[index]
				break
			}
		}
		if expected == nil && marker == nil {
			return nil
		}
		if expected == nil || marker == nil || marker.Kind != string(expected.Kind) || marker.Mode != expected.Mode || marker.ModTime != expected.ModTime || marker.Size != uint64(len(expected.Data)) || marker.DataSHA256 != string(record.HashBytes(expected.Data)) {
			return errors.New("recovered marker differs from exact reviewed native crash state")
		}
		return nil
	}
	return errors.New("selected-state recovered volume snapshot missing")
}
