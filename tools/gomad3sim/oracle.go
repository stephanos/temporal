package gomad3sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"unicode/utf8"
)

const MaximumOracleNameBytes = 256
const MaximumOperationTextBytes = 4096

var oracleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(?:\.[a-z][a-z0-9-]*)*$`)

type HistoryOperation struct {
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	Kind       string `json:"kind"`
	Invocation uint64 `json:"invocation"`
	Completion uint64 `json:"completion"`
	Input      []byte `json:"input,omitempty"`
	Output     []byte `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

type OracleEvidence struct {
	Label      string `json:"label"`
	Value      []byte `json:"value"`
	FullSHA256 string `json:"full_sha256"`
}

type OracleResult struct {
	Name            string           `json:"name"`
	Passed          bool             `json:"passed"`
	Evidence        []OracleEvidence `json:"evidence"`
	FailureIdentity string           `json:"failure_identity,omitempty"`
	Identity        string           `json:"identity"`
}

func StateInvariant(name string, passed bool, evidence []OracleEvidence, maximumBytes uint64) (OracleResult, error) {
	return newOracleResult(name, passed, evidence, maximumBytes)
}

func ExactHistory(name string, expected, actual []HistoryOperation, maximumBytes uint64) (OracleResult, error) {
	if err := ValidateHistory(expected, maximumBytes); err != nil {
		return OracleResult{}, fmt.Errorf("validate expected history: %w", err)
	}
	if err := ValidateHistory(actual, maximumBytes); err != nil {
		return OracleResult{}, fmt.Errorf("validate actual history: %w", err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return OracleResult{}, fmt.Errorf("encode expected history: %w", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return OracleResult{}, fmt.Errorf("encode actual history: %w", err)
	}
	return newOracleResult(name, slices.EqualFunc(expected, actual, equalHistoryOperation), []OracleEvidence{
		{Label: "actual", Value: actualJSON},
		{Label: "expected", Value: expectedJSON},
	}, maximumBytes)
}

func NoDuplicateOrLost(name string, expected, actual []string, maximumBytes uint64) (OracleResult, error) {
	expected = append([]string(nil), expected...)
	actual = append([]string(nil), actual...)
	for _, values := range [][]string{expected, actual} {
		for _, value := range values {
			if len(value) == 0 || len(value) > MaximumOperationTextBytes || !utf8.ValidString(value) {
				return OracleResult{}, errors.New("duplicate/lost oracle value is invalid")
			}
		}
	}
	sort.Strings(expected)
	sort.Strings(actual)
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return OracleResult{}, fmt.Errorf("encode expected operations: %w", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return OracleResult{}, fmt.Errorf("encode actual operations: %w", err)
	}
	return newOracleResult(name, slices.Equal(expected, actual), []OracleEvidence{
		{Label: "actual", Value: actualJSON},
		{Label: "expected", Value: expectedJSON},
	}, maximumBytes)
}

func EventualConvergence(name string, values map[string][]byte, maximumBytes uint64) (OracleResult, error) {
	labels := make([]string, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	evidence := make([]OracleEvidence, 0, len(labels))
	passed := len(labels) != 0
	var reference []byte
	for index, label := range labels {
		if err := validateID("convergence participant", label); err != nil {
			return OracleResult{}, err
		}
		value := append([]byte(nil), values[label]...)
		evidence = append(evidence, OracleEvidence{Label: label, Value: value})
		if index == 0 {
			reference = value
		} else if !bytes.Equal(reference, value) {
			passed = false
		}
	}
	return newOracleResult(name, passed, evidence, maximumBytes)
}

func ValidateHistory(operations []HistoryOperation, maximumBytes uint64) error {
	seen := make(map[string]struct{}, len(operations))
	var total uint64
	for _, operation := range operations {
		if err := validateID("history operation ID", operation.ID); err != nil {
			return err
		}
		if err := validateID("history actor", operation.Actor); err != nil {
			return err
		}
		if err := validateID("history operation kind", operation.Kind); err != nil {
			return err
		}
		if operation.Invocation == 0 || operation.Completion < operation.Invocation {
			return fmt.Errorf("history operation %q has an invalid interval", operation.ID)
		}
		if operation.Error != "" && len(operation.Output) != 0 {
			return fmt.Errorf("history operation %q contains both output and error", operation.ID)
		}
		if len(operation.Error) > MaximumOperationTextBytes || !utf8.ValidString(operation.Error) {
			return fmt.Errorf("history operation %q has invalid error evidence", operation.ID)
		}
		if _, ok := seen[operation.ID]; ok {
			return fmt.Errorf("history operation ID %q is duplicated", operation.ID)
		}
		seen[operation.ID] = struct{}{}
		total = saturatingAdd(total, uint64(len(operation.Input)))
		total = saturatingAdd(total, uint64(len(operation.Output)))
		total = saturatingAdd(total, uint64(len(operation.Error)))
		if err := checkCapacity("history_bytes", total, maximumBytes); err != nil {
			return err
		}
	}
	return nil
}

func newOracleResult(name string, passed bool, evidence []OracleEvidence, maximumBytes uint64) (OracleResult, error) {
	if len(name) == 0 || len(name) > MaximumOracleNameBytes || !oracleNamePattern.MatchString(name) {
		return OracleResult{}, fmt.Errorf("invalid oracle name %q", name)
	}
	cloned := make([]OracleEvidence, len(evidence))
	var total uint64
	for index, item := range evidence {
		if err := validateID("oracle evidence label", item.Label); err != nil {
			return OracleResult{}, err
		}
		if index != 0 && evidence[index-1].Label >= item.Label {
			return OracleResult{}, errors.New("oracle evidence must be strictly sorted")
		}
		cloned[index] = OracleEvidence{Label: item.Label, Value: append([]byte(nil), item.Value...)}
		identity, err := hashCanonical("gomad3-oracle-evidence/v1", cloned[index].Value)
		if err != nil {
			return OracleResult{}, err
		}
		cloned[index].FullSHA256 = identity
		total = saturatingAdd(total, uint64(len(item.Value)))
		if err := checkCapacity("oracle_evidence_bytes", total, maximumBytes); err != nil {
			return OracleResult{}, err
		}
	}
	result := OracleResult{Name: name, Passed: passed, Evidence: cloned}
	if !passed {
		identity, err := hashCanonical("gomad3-oracle-failure/v1", struct {
			Name     string           `json:"name"`
			Evidence []OracleEvidence `json:"evidence"`
		}{Name: name, Evidence: cloned})
		if err != nil {
			return OracleResult{}, err
		}
		result.FailureIdentity = identity
	}
	identity, err := oracleResultIdentity(result)
	if err != nil {
		return OracleResult{}, err
	}
	result.Identity = identity
	return result, nil
}

func validateOracleResult(result OracleResult, maximumBytes uint64) error {
	rebuilt, err := newOracleResult(result.Name, result.Passed, result.Evidence, maximumBytes)
	if err != nil {
		return err
	}
	if !equalOracleEvidence(rebuilt.Evidence, result.Evidence) || rebuilt.Name != result.Name || rebuilt.Passed != result.Passed || rebuilt.FailureIdentity != result.FailureIdentity || rebuilt.Identity != result.Identity {
		return errors.New("oracle result identity does not match its contents")
	}
	return nil
}

func oracleResultIdentity(result OracleResult) (string, error) {
	result.Identity = ""
	return hashCanonical("gomad3-oracle-result/v1", result)
}

func equalOracleEvidence(left, right []OracleEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Label != right[index].Label || left[index].FullSHA256 != right[index].FullSHA256 || !bytes.Equal(left[index].Value, right[index].Value) {
			return false
		}
	}
	return true
}

func equalHistoryOperation(left, right HistoryOperation) bool {
	return left.ID == right.ID && left.Actor == right.Actor && left.Kind == right.Kind && left.Invocation == right.Invocation && left.Completion == right.Completion && bytes.Equal(left.Input, right.Input) && bytes.Equal(left.Output, right.Output) && left.Error == right.Error
}

func cloneHistoryOperation(operation HistoryOperation) HistoryOperation {
	operation.Input = append([]byte(nil), operation.Input...)
	operation.Output = append([]byte(nil), operation.Output...)
	return operation
}

func cloneHistoryOperations(operations []HistoryOperation) []HistoryOperation {
	cloned := make([]HistoryOperation, len(operations))
	for index, operation := range operations {
		cloned[index] = cloneHistoryOperation(operation)
	}
	return cloned
}
