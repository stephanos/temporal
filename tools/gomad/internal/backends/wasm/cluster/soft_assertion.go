package simulation

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/evidence/record/softassert"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

type SoftAssertionPolicy struct {
	Handle     gomad3sim.NodeHandle `json:"handle"`
	ReportOnly bool                 `json:"report_only"`
}

func validateSoftAssertionPolicies(plan *OraclePlan, spec gomad3sim.Spec) error {
	seen := map[gomad3sim.NodeHandle]bool{}
	for _, policy := range plan.SoftAssertions {
		known := false
		for _, node := range spec.Nodes {
			known = known || node.ID == policy.Handle.Node
		}
		if !known || policy.Handle.Incarnation == 0 || seen[policy.Handle] {
			return errors.New("invalid soft assertion policy participant")
		}
		seen[policy.Handle] = true
	}
	if uint64(len(seen)) > spec.Limits.OracleResults {
		return errors.New("soft assertion plan exceeds result capacity")
	}
	return nil
}

func (p *Provider) validateSoftAssertionWorkloads(spec gomad3sim.Spec) error {
	if p.Oracles == nil {
		return nil
	}
	for _, policy := range p.Oracles.SoftAssertions {
		if !policy.ReportOnly {
			continue
		}
		var boot gomad3sim.BootID
		for _, node := range spec.Nodes {
			if node.ID == policy.Handle.Node {
				boot = node.Boot
			}
		}
		for _, binding := range p.IncarnationBoots {
			if binding.Handle == policy.Handle {
				boot = binding.Boot
			}
		}
		if p.Boots[boot].WorkloadKind != target.KindGoTest {
			return errors.New("soft assertion report-only mode requires a sealed Go test boot workload")
		}
	}
	return nil
}

func (e *execution) softAssertionOracles() ([]gomad3sim.OracleResult, error) {
	handles := make([]gomad3sim.NodeHandle, 0, len(e.results))
	for handle := range e.results {
		handles = append(handles, handle)
	}
	policies := map[gomad3sim.NodeHandle]SoftAssertionPolicy{}
	maximum := e.limits.ScenarioEvidenceBytes
	if plan := e.provider.Oracles; plan != nil {
		for _, policy := range plan.SoftAssertions {
			policies[policy.Handle] = policy
			if _, ok := e.results[policy.Handle]; !ok {
				handles = append(handles, policy.Handle)
			}
		}
		if len(plan.SoftAssertions) != 0 {
			maximum = plan.MaximumBytes
		}
	}
	slices.SortFunc(handles, func(a, b gomad3sim.NodeHandle) int {
		if a.Node < b.Node {
			return -1
		}
		if a.Node > b.Node {
			return 1
		}
		if a.Incarnation < b.Incarnation {
			return -1
		}
		if a.Incarnation > b.Incarnation {
			return 1
		}
		return 0
	})
	var oracles []gomad3sim.OracleResult
	for _, handle := range handles {
		result, exists := e.results[handle]
		policy, requested := policies[handle]
		complete := exists && result.Reaped && result.Termination != "capacity" && (!result.Cancelled || result.RuntimeInterrupted) && !result.WatchdogTimeout && result.OfferedOutputBytes == uint64(len(result.Stdout))+uint64(len(result.Stderr))
		observation := softassert.Observe(result.Stdout, result.Stderr, complete)
		if len(observation.Messages) == 0 && observation.Complete && !requested {
			continue
		}
		oracle, err := gomad3sim.SoftAssertion(fmt.Sprintf("%s/%d", handle.Node, handle.Incarnation), observation.Messages, !observation.Complete, policy.ReportOnly, maximum)
		if err != nil {
			return nil, err
		}
		oracles = append(oracles, oracle)
	}
	slices.SortFunc(oracles, func(a, b gomad3sim.OracleResult) int {
		return strings.Compare(a.Name, b.Name)
	})
	return oracles, nil
}

func (e *execution) outputCapacityFailure(handle gomad3sim.NodeHandle, result wasi.Result) error {
	if result.Termination != "capacity" || result.Message != "output capacity" {
		return nil
	}
	reportOnly := false
	if plan := e.provider.Oracles; plan != nil {
		for _, policy := range plan.SoftAssertions {
			if policy.Handle == handle {
				reportOnly = policy.ReportOnly
			}
		}
	}
	observation := softassert.Observe(result.Stdout, result.Stderr, false)
	return &backends.FailureError{Termination: backends.Capacity, Message: result.Message,
		SoftAssertions: &record.SoftAssertions{Messages: observation.Messages, ReportOnly: reportOnly, Inconclusive: true}}
}

func validWorkloadKind(kind target.Kind) bool {
	return kind == "" || kind == target.KindGoRun || kind == target.KindGoTest || kind == target.KindExec
}

func validateRetainedSoftAssertions(spec gomad3sim.Spec, provider *Provider) (int, error) {
	plan := spec.Replay
	results := make(map[gomad3sim.NodeHandle]wasi.Result)
	if plan.WASM != nil {
		for _, incarnation := range plan.WASM.Incarnations {
			results[incarnation.Handle] = wasi.Result{Reaped: true}
		}
	}
	for _, output := range plan.Outputs {
		if output.TotalBytes == uint64(len(output.Bytes)) && output.FullSHA256 != string(record.HashBytes(output.Bytes)) {
			return 0, errors.New("retained soft assertion output digest mismatch")
		}
		result, known := results[output.Handle]
		if !known {
			return 0, errors.New("retained soft assertion output lacks a bound incarnation")
		}
		result.OfferedOutputBytes += output.TotalBytes
		result.Cancelled = result.Cancelled || output.Truncated
		switch output.Stream {
		case gomad3sim.OutputStdout:
			result.Stdout = append(result.Stdout, output.Bytes...)
		case gomad3sim.OutputStderr:
			result.Stderr = append(result.Stderr, output.Bytes...)
		}
		results[output.Handle] = result
	}
	execution := execution{provider: Provider{Oracles: provider.Oracles}, results: results, limits: spec.Limits}
	expected, err := execution.softAssertionOracles()
	if err != nil {
		return 0, err
	}
	var actual []gomad3sim.OracleResult
	for _, oracle := range plan.Oracles {
		if oracle.SoftAssertion {
			actual = append(actual, oracle)
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		return 0, errors.New("retained soft assertion evidence differs from authenticated guest output or policy")
	}
	return len(expected), nil
}
