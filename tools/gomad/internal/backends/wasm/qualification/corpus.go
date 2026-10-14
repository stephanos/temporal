package qualification

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// The corpus directory admits only the exact existing Result identity and its
// bounded terminal receipt. A receipt is never treated as another Result.
func corpusResultEntry(directory, name string) (string, bool, error) {
	if filepath.Base(name) != name {
		return "", false, errors.New("invalid corpus entry path")
	}
	if strings.HasSuffix(name, ".publication.json") {
		if _, err := publicationResultIdentity(strings.TrimSuffix(name, ".publication.json")); err != nil {
			return "", false, err
		}
		return "", false, nil
	}
	if _, err := publicationResultIdentity(name); err != nil {
		return "", false, err
	}
	return filepath.Join(directory, name), true, nil
}

func validateCorpusEntryPairs(names []string) error {
	results, receipts := map[string]bool{}, map[string]bool{}
	for _, name := range names {
		_, isResult, err := corpusResultEntry("", name)
		if err != nil {
			return err
		}
		entries, key := results, name
		if !isResult {
			entries, key = receipts, strings.TrimSuffix(name, ".publication.json")
		}
		if entries[key] {
			return errors.New("repeated corpus Result or publication receipt")
		}
		entries[key] = true
	}
	if len(results) != len(receipts) {
		return errors.New("missing or extra corpus publication receipt")
	}
	for name := range results {
		if !receipts[name] {
			return errors.New("mismatched corpus publication receipt filename")
		}
	}
	return nil
}

// ValidateCorpus is a gate over existing owner reports, not a new report or
// campaign format. Supplementary fixtures and operation-only SQLite results do
// not substitute for the four required actual workload/canary cells.
func ValidateCorpus(results []Result, receipts ...PublicationReceipt) error {
	if len(receipts) != len(results) {
		return errors.New("mandatory corpus requires exactly one terminal publication receipt per Result")
	}
	required := map[string]bool{}
	order := []string{}
	for _, cell := range []struct{ workload, id, profile string }{{"fixture", "ready-select-timer", "controlled"}, {"fixture", "ready-select-timer", "cooperative"}, {"sqlite", "sqlite-durable-recovery", "cooperative"}, {"onebox", "unchanged-workflow-type-smoke", "cooperative"}, {"multi-instance", "independent-temporal-replication-durable-restart", "cooperative"}} {
		for _, variant := range []string{"success", "canary"} {
			key := cell.workload + "/" + cell.id + "/" + cell.profile + "/" + variant
			required[key] = false
			order = append(order, key)
		}
	}
	seenProcesses := map[uint64]bool{}
	var integrated, compiler, helper, engine, model record.SHA256
	var hostOS, hostArch string
	matched := map[string]Manifest{}
	for index, result := range results {
		m := result.Manifest
		key := m.Workload + "/" + m.ID + "/" + m.Profile + "/" + m.Cell
		found, known := required[key]
		if !known || found {
			return fmt.Errorf("unknown or repeated mandatory qualification cell %s", key)
		}
		if m.Provisional {
			return errors.New("mandatory corpus cannot accept provisional source binding")
		}
		if result.PreparationReusedWithinProducer {
			return errors.New("mandatory fresh cells cannot import another cell's preparation metrics")
		}
		if err := ValidateResult(result); err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		path := "result-" + strings.TrimPrefix(string(record.HashBytes(append(encoded, '\n'))), "sha256:") + ".json"
		if _, err := ValidatePublicationReceipt(path, result, receipts[index]); err != nil {
			return err
		}
		if result.OperatorProcessID == 0 || seenProcesses[result.OperatorProcessID] {
			return errors.New("mandatory cells require distinct fresh operator processes")
		}
		seenProcesses[result.OperatorProcessID] = true
		if !result.Qualification.Deterministic || result.Qualification.Failure != nil || result.Cost == nil || result.CostFailure != "" || result.Budget == nil || !result.Budget.WithinBudget || result.BudgetFailure != "" {
			return fmt.Errorf("mandatory qualification/cost/budget gate failed for %s", key)
		}
		if result.Qualification.TargetSuccess != (m.Cell == "success") || m.Cell == "canary" && result.Cost.ReplayableBugs != 1 {
			return fmt.Errorf("expected workload outcome or replayable canary missing for %s", key)
		}
		if m.Workload == "sqlite" && m.Cell == "success" {
			if result.NativeCrashSelection != nil {
				return errors.New("supplementary selected state cannot substitute for primary durable-only SQLite cell")
			}
			enumeration, err := MatchedNativeCrashEnumeration(result, gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}, "sqlite")
			if err != nil || len(enumeration.States) != 4 {
				return errors.New("mandatory SQLite actual matched reviewed four-state catalogue missing")
			}
		}
		for _, execution := range result.Qualification.Executions {
			if execution.Replay == nil || !execution.Replay.Match || !execution.Replay.Attempted || execution.ArtifactPath == "" {
				return fmt.Errorf("retained matched execution replay missing for %s", key)
			}
		}
		if integrated == "" {
			integrated, compiler, helper, engine, model = m.IntegratedSourceSHA256, m.CompilerSHA256, m.HelperSHA256, m.EngineSHA256, m.ModelSHA256
			hostOS, hostArch = m.HostOS, m.HostArch
		} else if integrated != m.IntegratedSourceSHA256 || compiler != m.CompilerSHA256 || helper != m.HelperSHA256 || engine != m.EngineSHA256 || model != m.ModelSHA256 || hostOS != m.HostOS || hostArch != m.HostArch {
			return errors.New("mandatory corpus contains unmatched candidate/compiler/helper/engine/model/host")
		}
		if previous, ok := matched[m.Workload]; ok {
			if err := Matched(previous, m); err != nil {
				return err
			}
		} else {
			matched[m.Workload] = m
		}
		required[key] = true
	}
	for _, cell := range order {
		if !required[cell] {
			return fmt.Errorf("mandatory actual qualification cell missing: %s", cell)
		}
	}
	return nil
}
