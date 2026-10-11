package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad3sim"
)

const HistoryObservationSchema = "gomad.wasm.history-observation/v1"
const SQLiteObservationSchema = "gomad.wasm.sqlite-observation/v1"

// OraclePlan is concrete scenario intent over newly observed guest output. It
// never supplies an already observed oracle result as the replay expectation.
type OraclePlan struct {
	Schema       string               `json:"schema"`
	Prefix       string               `json:"prefix"`
	MaximumBytes uint64               `json:"maximum_bytes"`
	Histories    []HistoryComparison  `json:"histories"`
	Convergence  []HistoryConvergence `json:"convergence"`
	SQLite       []SQLiteComparison   `json:"sqlite,omitempty"`
}
type SQLiteComparison struct {
	Name              string               `json:"name"`
	Acknowledged      gomad3sim.NodeHandle `json:"acknowledged"`
	Recovered         gomad3sim.NodeHandle `json:"recovered"`
	ExpectedEvents    []string             `json:"expected_events"`
	ExpectedValue     string               `json:"expected_value"`
	ExpectedIntegrity string               `json:"expected_integrity"`
}
type HistoryComparison struct {
	Name           string               `json:"name"`
	Expected       gomad3sim.NodeHandle `json:"expected"`
	Actual         gomad3sim.NodeHandle `json:"actual"`
	ExpectedEvents []string             `json:"expected_events"`
}
type HistoryConvergence struct {
	Name    string                 `json:"name"`
	Handles []gomad3sim.NodeHandle `json:"handles"`
}
type historyObservation struct {
	ClusterID string   `json:"cluster_id"`
	History   []byte   `json:"history"`
	EventIDs  []int64  `json:"event_ids"`
	Types     []string `json:"types"`
}

func validateOraclePlan(plan *OraclePlan, spec gomad3sim.Spec) error {
	if plan == nil {
		return nil
	}
	if plan.Schema != HistoryObservationSchema && plan.Schema != SQLiteObservationSchema || len(plan.Prefix) == 0 || len(plan.Prefix) > 256 || bytes.ContainsAny([]byte(plan.Prefix), "\r\n") || plan.MaximumBytes == 0 || plan.MaximumBytes > 1<<20 || plan.MaximumBytes > spec.Limits.ScenarioEvidenceBytes || len(plan.Histories)+len(plan.Convergence)+len(plan.SQLite) == 0 {
		return errors.New("invalid WASM oracle observation plan")
	}
	if plan.Schema == SQLiteObservationSchema && (plan.Prefix != "HISTORY " || len(plan.Histories)+len(plan.Convergence) != 0) || plan.Schema == HistoryObservationSchema && len(plan.SQLite) != 0 {
		return errors.New("mixed or invalid WASM oracle observation schemas")
	}
	known := func(h gomad3sim.NodeHandle) bool {
		for _, n := range spec.Nodes {
			if n.ID == h.Node && h.Incarnation != 0 {
				return true
			}
		}
		return false
	}
	names := map[string]bool{}
	checkName := func(name string) error {
		if names[name] {
			return errors.New("duplicate WASM oracle name")
		}
		if _, err := gomad3sim.StateInvariant(name, true, nil, plan.MaximumBytes); err != nil {
			return err
		}
		names[name] = true
		return nil
	}
	for _, comparison := range plan.Histories {
		if !known(comparison.Expected) || !known(comparison.Actual) || comparison.Expected == comparison.Actual || len(comparison.ExpectedEvents) == 0 {
			return errors.New("WASM history oracle lacks bound guest or event intent")
		}
		for _, suffix := range []string{".history", ".events", ".identity"} {
			if err := checkName(comparison.Name + suffix); err != nil {
				return err
			}
		}
		if _, err := gomad3sim.NoDuplicateOrLost(comparison.Name+".events", comparison.ExpectedEvents, comparison.ExpectedEvents, plan.MaximumBytes); err != nil {
			return err
		}
	}
	for _, convergence := range plan.Convergence {
		if len(convergence.Handles) < 2 {
			return errors.New("WASM convergence requires independent observations")
		}
		seen := map[gomad3sim.NodeHandle]bool{}
		for _, h := range convergence.Handles {
			if !known(h) || seen[h] {
				return errors.New("invalid WASM convergence participant")
			}
			seen[h] = true
		}
		if err := checkName(convergence.Name); err != nil {
			return err
		}
	}
	for _, comparison := range plan.SQLite {
		if !known(comparison.Acknowledged) || !known(comparison.Recovered) || comparison.Acknowledged == comparison.Recovered || len(comparison.ExpectedEvents) == 0 || comparison.ExpectedValue == "" || strings.ContainsAny(comparison.ExpectedValue, ",:\r\n") || comparison.ExpectedIntegrity != "ok" {
			return errors.New("SQLite oracle lacks bound acknowledgment, recovery or committed intent")
		}
		for _, id := range comparison.ExpectedEvents {
			if !sqliteID(id) {
				return errors.New("invalid SQLite oracle event intent")
			}
		}
		for _, suffix := range []string{".events", ".integrity", ".acknowledged"} {
			if err := checkName(comparison.Name + suffix); err != nil {
				return err
			}
		}
		if _, err := gomad3sim.NoDuplicateOrLost(comparison.Name+".events", comparison.ExpectedEvents, comparison.ExpectedEvents, plan.MaximumBytes); err != nil {
			return err
		}
	}
	if uint64(len(names)) > spec.Limits.OracleResults {
		return errors.New("WASM oracle plan exceeds result capacity")
	}
	return nil
}

func decodeHistoryObservation(stdout []byte, plan *OraclePlan) (historyObservation, error) {
	var line []byte
	count := 0
	for _, candidate := range bytes.Split(stdout, []byte{'\n'}) {
		if bytes.HasPrefix(candidate, []byte(plan.Prefix)) {
			line = candidate[len(plan.Prefix):]
			count++
		}
	}
	if count != 1 || uint64(len(line)) > plan.MaximumBytes {
		return historyObservation{}, errors.New("missing, ambiguous or oversized WASM history observation")
	}
	// Validate object keys separately: encoding/json otherwise permits duplicate
	// keys, which cannot be an unambiguous detached correctness observation.
	d := json.NewDecoder(bytes.NewReader(line))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return historyObservation{}, errors.New("invalid WASM history observation object")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return historyObservation{}, err
		}
		name, ok := key.(string)
		if !ok || seen[name] || name != "cluster_id" && name != "history" && name != "event_ids" && name != "types" {
			return historyObservation{}, errors.New("unknown or duplicate WASM history observation field")
		}
		seen[name] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return historyObservation{}, err
		}
	}
	if _, err := d.Token(); err != nil {
		return historyObservation{}, err
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return historyObservation{}, errors.New("trailing WASM history observation")
	}
	var value historyObservation
	if err := json.Unmarshal(line, &value); err != nil {
		return value, err
	}
	if len(seen) != 4 || value.ClusterID == "" || len(value.History) == 0 || len(value.EventIDs) == 0 || len(value.EventIDs) != len(value.Types) {
		return value, errors.New("incomplete WASM history observation")
	}
	for i, id := range value.EventIDs {
		if id <= 0 || value.Types[i] == "" {
			return value, errors.New("invalid WASM history event observation")
		}
	}
	return value, nil
}

// RecordOracles evaluates sealed intent from real completed/interrupted guest
// outputs, forwarding to the existing cluster correctness and evidence owners.
func (p *Provider) RecordOracles(ctx context.Context, cluster gomad3sim.Cluster) error {
	p.mu.Lock()
	e := p.latest
	p.mu.Unlock()
	if e == nil {
		return errors.New("WASM oracle plan has no execution")
	}
	plan := e.provider.Oracles
	if plan == nil {
		return nil
	}
	if plan.Schema == SQLiteObservationSchema {
		return recordSQLiteOracles(ctx, cluster, e, plan)
	}
	values := map[gomad3sim.NodeHandle]historyObservation{}
	read := func(h gomad3sim.NodeHandle) (historyObservation, error) {
		if value, ok := values[h]; ok {
			return value, nil
		}
		e.mu.Lock()
		result, ok := e.results[h]
		stdout := bytes.Clone(result.Stdout)
		e.mu.Unlock()
		if !ok || !result.Reaped {
			return historyObservation{}, errors.New("WASM oracle guest has not completed and reaped")
		}
		value, err := decodeHistoryObservation(stdout, plan)
		values[h] = value
		return value, err
	}
	history := func(value historyObservation) []gomad3sim.HistoryOperation {
		return []gomad3sim.HistoryOperation{{ID: "committed", Actor: "workflow", Kind: "workflow-history", Invocation: 1, Completion: 2, Output: value.History}}
	}
	for _, comparison := range plan.Histories {
		expected, err := read(comparison.Expected)
		if err != nil {
			return err
		}
		actual, err := read(comparison.Actual)
		if err != nil {
			return err
		}
		exact, err := gomad3sim.ExactHistory(comparison.Name+".history", history(expected), history(actual), plan.MaximumBytes)
		if err != nil {
			return err
		}
		ids := make([]string, len(actual.EventIDs))
		for i, id := range actual.EventIDs {
			ids[i] = strconv.FormatInt(id, 10)
		}
		events, err := gomad3sim.NoDuplicateOrLost(comparison.Name+".events", comparison.ExpectedEvents, ids, plan.MaximumBytes)
		if err != nil {
			return err
		}
		identity, err := gomad3sim.StateInvariant(comparison.Name+".identity", expected.ClusterID == actual.ClusterID, []gomad3sim.OracleEvidence{{Label: "actual", Value: []byte(actual.ClusterID)}, {Label: "expected", Value: []byte(expected.ClusterID)}}, plan.MaximumBytes)
		if err != nil {
			return err
		}
		for _, oracle := range []gomad3sim.OracleResult{exact, events, identity} {
			if err := cluster.RecordOracle(ctx, oracle); err != nil {
				return err
			}
		}
	}
	for _, convergence := range plan.Convergence {
		observations := map[string][]byte{}
		for _, h := range convergence.Handles {
			value, err := read(h)
			if err != nil {
				return err
			}
			observations[fmt.Sprintf("%s-%d", h.Node, h.Incarnation)] = value.History
		}
		oracle, err := gomad3sim.EventualConvergence(convergence.Name, observations, plan.MaximumBytes)
		if err != nil {
			return err
		}
		if err := cluster.RecordOracle(ctx, oracle); err != nil {
			return err
		}
	}
	return nil
}

type sqliteObservation struct {
	Integrity   string
	IDs, Values []string
}

func sqliteID(id string) bool {
	n, err := strconv.ParseUint(id, 10, 64)
	return err == nil && n != 0 && strconv.FormatUint(n, 10) == id
}

func sqliteLine(stdout []byte, prefix string, maximum uint64) (string, error) {
	var value string
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if bytes.HasPrefix(line, []byte(prefix)) {
			if uint64(len(line)) > maximum {
				return "", errors.New("oversized SQLite observation")
			}
			value = string(line[len(prefix):])
			count++
		}
	}
	if count != 1 || value == "" {
		return "", errors.New("missing or ambiguous SQLite observation")
	}
	return value, nil
}

func decodeSQLiteAcknowledgement(stdout []byte, maximum uint64) (string, error) {
	id, err := sqliteLine(stdout, "ACK ", maximum)
	if err != nil {
		return "", err
	}
	if !sqliteID(id) {
		return "", errors.New("invalid SQLite acknowledgement")
	}
	return id, nil
}

func decodeSQLiteObservation(stdout []byte, maximum uint64) (sqliteObservation, error) {
	var value sqliteObservation
	var err error
	value.Integrity, err = sqliteLine(stdout, "INTEGRITY ", maximum)
	if err != nil {
		return value, err
	}
	history, err := sqliteLine(stdout, "HISTORY ", maximum)
	if err != nil {
		return value, err
	}
	for _, row := range strings.Split(history, ",") {
		id, item, ok := strings.Cut(row, ":")
		if !ok || !sqliteID(id) || item == "" || strings.ContainsAny(item, ":\r\n") {
			return value, errors.New("invalid SQLite history observation")
		}
		value.IDs = append(value.IDs, id)
		value.Values = append(value.Values, item)
	}
	return value, nil
}

func recordSQLiteOracles(ctx context.Context, cluster gomad3sim.Cluster, e *execution, plan *OraclePlan) error {
	read := func(handle gomad3sim.NodeHandle) ([]byte, error) {
		e.mu.Lock()
		result, ok := e.results[handle]
		stdout := bytes.Clone(result.Stdout)
		e.mu.Unlock()
		if !ok || !result.Reaped {
			return nil, errors.New("SQLite oracle guest has not completed and reaped")
		}
		return stdout, nil
	}
	for _, comparison := range plan.SQLite {
		ackOutput, err := read(comparison.Acknowledged)
		if err != nil {
			return err
		}
		ack, err := decodeSQLiteAcknowledgement(ackOutput, plan.MaximumBytes)
		if err != nil {
			return err
		}
		recoveredOutput, err := read(comparison.Recovered)
		if err != nil {
			return err
		}
		actual, err := decodeSQLiteObservation(recoveredOutput, plan.MaximumBytes)
		if err != nil {
			return err
		}
		events, err := gomad3sim.NoDuplicateOrLost(comparison.Name+".events", comparison.ExpectedEvents, actual.IDs, plan.MaximumBytes)
		if err != nil {
			return err
		}
		integrity, err := gomad3sim.StateInvariant(comparison.Name+".integrity", actual.Integrity == comparison.ExpectedIntegrity, []gomad3sim.OracleEvidence{{Label: "actual", Value: []byte(actual.Integrity)}, {Label: "expected", Value: []byte(comparison.ExpectedIntegrity)}}, plan.MaximumBytes)
		if err != nil {
			return err
		}
		committed := ack == comparison.ExpectedEvents[len(comparison.ExpectedEvents)-1]
		for _, item := range actual.Values {
			committed = committed && item == comparison.ExpectedValue
		}
		acknowledged, err := gomad3sim.StateInvariant(comparison.Name+".acknowledged", committed, []gomad3sim.OracleEvidence{{Label: "ack", Value: []byte(ack)}, {Label: "expected", Value: []byte(comparison.ExpectedValue)}, {Label: "history", Value: recoveredOutput}}, plan.MaximumBytes)
		if err != nil {
			return err
		}
		for _, oracle := range []gomad3sim.OracleResult{events, integrity, acknowledged} {
			if err := cluster.RecordOracle(ctx, oracle); err != nil {
				return err
			}
		}
	}
	return nil
}
