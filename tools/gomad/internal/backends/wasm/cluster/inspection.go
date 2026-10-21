package simulation

import (
	"crypto/sha256"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func ResolveInspection(execution *cluster.WASMExecutionEvidence, request cluster.WASMInspectionRequest, isCluster bool) (cluster.WASMInspectionResolution, error) {
	if err := ValidateInspectionChoices(execution); err != nil {
		return cluster.WASMInspectionResolution{}, err
	}
	return cluster.ResolveWASMInspection(execution, request, isCluster)
}

func ValidateInspectionChoices(execution *cluster.WASMExecutionEvidence) error {
	if execution == nil || execution.Inspection == nil || !execution.Inspection.Complete {
		return nil
	}
	evidence := execution.Inspection
	if err := cluster.ValidateWASMInspection(evidence, execution); err != nil {
		return err
	}
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", cluster.ErrWASMInspectionUnsupported, reason) }
	for _, retained := range execution.Incarnations {
		count, err := choice.TraceRecordCount(retained.ChoiceTrace)
		if err != nil {
			return invalid("choice trace")
		}
		records, err := inspectionChoiceRecords(retained.ChoiceTrace, count)
		if err != nil {
			return invalid("choice trace")
		}
		var mapped []uint64
		predecision := choice.PredecisionOrdinals(records)
		for index, record := range records {
			if record.Flags&choice.FlagDecision != 0 && record.Alternatives >= 2 {
				mapped = append(mapped, predecision[index])
			}
		}
		var decisions uint64
		for _, boundary := range evidence.Boundaries {
			if boundary.Handle == retained.Handle {
				if boundary.Decision >= uint64(len(mapped)) || mapped[boundary.Decision] != boundary.Trace {
					return invalid("decision/trace mapping")
				}
				decisions++
			}
			for _, cut := range boundary.Cuts {
				if cut.Handle != retained.Handle {
					continue
				}
				bytes, err := choice.TracePayloadBytes(cut.ChoiceRecords)
				if err != nil || cut.ChoiceRecords > count || cut.ChoiceBytes != bytes {
					return invalid("choice prefix length")
				}
			}
		}
		if decisions != uint64(len(mapped)) {
			return invalid("incomplete decision boundary inventory")
		}
	}
	return nil
}

func inspectionChoiceRecords(payload []byte, count uint64) ([]choice.Record, error) {
	limit, err := choice.TraceBytes(count)
	if err != nil {
		return nil, err
	}
	// The choice owner uses the same codec scaffold for interrupted prefixes;
	// only raw records escape it, never a successful execution disposition.
	trace, err := choice.DecodeStoredTrace(choice.Profile, payload, choice.TerminalMetadata{State: choice.TerminalComplete, Limit: limit, Records: count, SHA256: sha256.Sum256(payload)})
	if err != nil {
		return nil, err
	}
	return trace.Records, nil
}
