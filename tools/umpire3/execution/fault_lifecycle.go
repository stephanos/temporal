package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	umpire3fault "go.temporal.io/server/tools/umpire3/execution/fault"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

var errFaultRealizerUnavailable = errors.New("environment does not provide a fault realizer")

type installedFault struct {
	definition protocolexperiment.Fault
	term       umpire3fault.Term
	handle     string
	result     int
	active     bool
	cleaned    bool
}

type faultSet struct {
	realizer         umpire3fault.Realizer
	evidenceProvider umpire3fault.EvidenceProvider
	values           []installedFault
}

func prepareFaults(
	ctx context.Context,
	request Request,
	session Session,
	capabilities []protocolcatalog.CapabilityID,
	limits Limits,
	result *Result,
) (*faultSet, error) {
	if len(request.Experiment.Faults) == 0 {
		return nil, nil
	}
	provider, ok := request.Environment.(umpire3fault.Provider)
	if !ok {
		provider, ok = session.(umpire3fault.Provider)
	}
	if !ok || provider.FaultRealizer() == nil {
		return nil, errFaultRealizerUnavailable
	}
	realizer := provider.FaultRealizer()
	evidenceProvider, ok := realizer.(umpire3fault.EvidenceProvider)
	if !ok {
		return nil, errors.New("fault realizer does not provide realization evidence")
	}
	set := &faultSet{
		realizer: realizer, evidenceProvider: evidenceProvider,
		values: make([]installedFault, 0, len(request.Experiment.Faults)),
	}
	actionIndexes := make(map[string]int64, len(request.Experiment.Actions))
	for index, action := range request.Experiment.Actions {
		actionIndexes[action.Identifier] = int64(index + 1)
	}
	isolationIdentity := result.Environment.IsolationIdentity
	if isolationIdentity == "" {
		isolationIdentity = request.Experiment.ExperimentID
	}
	for _, definition := range request.Experiment.Faults {
		term := umpire3fault.Term{
			Kind: protocolcatalog.FaultKind(definition.Kind),
			Scope: umpire3fault.Scope{
				Namespaces: []string{isolationIdentity}, Endpoints: slices.Clone(definition.Scope.Endpoints),
				TaskQueues: slices.Clone(definition.Scope.TaskQueues), Services: slices.Clone(definition.Scope.Services),
				Routes: slices.Clone(definition.Scope.Routes), Participants: slices.Clone(definition.Scope.Participants),
				Attempts: slices.Clone(definition.Scope.Attempts),
			},
			Occurrence: umpire3fault.Occurrence{First: definition.Occurrence.First, Count: definition.Occurrence.Count},
			Interval: umpire3fault.Interval{
				Start: actionIndexes[definition.Interval.StartAction],
				Stop:  actionIndexes[definition.Interval.StopAction] + 1,
			},
		}
		result.Faults = append(result.Faults, FaultResult{Identifier: definition.Identifier, Kind: definition.Kind})
		resultIndex := len(result.Faults) - 1
		if err := umpire3fault.Preflight(term, capabilities, request.AllowRestrictedFaults); err != nil {
			result.Faults[resultIndex].Error = err.Error()
			return set, fmt.Errorf("preflight fault %q: %w", definition.Identifier, err)
		}
		faultCtx, cancelFault := context.WithTimeout(ctx, limits.FaultTimeout)
		handle, err := realizer.Install(faultCtx, term)
		cancelFault()
		if err != nil {
			result.Faults[resultIndex].Error = err.Error()
			return set, fmt.Errorf("install fault %q: %w", definition.Identifier, err)
		}
		if handle == "" {
			result.Faults[resultIndex].Error = "fault realizer returned an empty handle"
			return set, fmt.Errorf("install fault %q: fault realizer returned an empty handle", definition.Identifier)
		}
		digest := sha256.Sum256([]byte(handle))
		result.Faults[resultIndex].Reference = "fault-installation/" + definition.Identifier + "/sha256:" + hex.EncodeToString(digest[:])
		result.Faults[resultIndex].Installed = true
		set.values = append(set.values, installedFault{
			definition: definition, term: term, handle: handle, result: resultIndex,
		})
	}
	return set, nil
}

func (s *faultSet) beforeAction(ctx context.Context, action string, result *Result) error {
	for index := range s.values {
		value := &s.values[index]
		if value.definition.Interval.StartAction != action || value.active {
			continue
		}
		if err := s.realizer.Activate(ctx, value.handle); err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("activate fault: %w", err))
			return fmt.Errorf("activate fault %q: %w", value.definition.Identifier, err)
		}
		value.active = true
		result.Faults[value.result].Activated = true
	}
	return nil
}

func (s *faultSet) afterAction(ctx context.Context, action string, result *Result) error {
	for index := len(s.values) - 1; index >= 0; index-- {
		value := &s.values[index]
		if value.definition.Interval.StopAction != action || !value.active {
			continue
		}
		evidence, err := s.evidenceProvider.RealizationEvidence(ctx, value.handle)
		if err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("observe fault realization: %w", err))
			return fmt.Errorf("observe fault %q realization: %w", value.definition.Identifier, err)
		}
		if evidence.SourceIdentity == "" || evidence.Reference == "" || evidence.EntityIdentity == "" {
			err := errors.New("fault realization evidence is incomplete")
			appendFaultError(&result.Faults[value.result], err)
			return fmt.Errorf("observe fault %q realization: %w", value.definition.Identifier, err)
		}
		faultResult := &result.Faults[value.result]
		faultResult.SourceIdentity = evidence.SourceIdentity
		faultResult.Reference = evidence.Reference
		faultResult.EntityIdentity = evidence.EntityIdentity
		faultResult.Realized = true
		if err := s.realizer.Release(ctx, value.handle); err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("release fault: %w", err))
			return fmt.Errorf("release fault %q: %w", value.definition.Identifier, err)
		}
		value.active = false
		result.Faults[value.result].Released = true
	}
	return nil
}

func (s *faultSet) cleanup(result *Result, timeout time.Duration) {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), timeout)
	defer cancelCleanup()
	cleanupFailed := false
	for index := len(s.values) - 1; index >= 0; index-- {
		value := &s.values[index]
		faultResult := &result.Faults[value.result]
		if value.active {
			if err := s.realizer.Release(cleanupCtx, value.handle); err != nil {
				appendFaultError(faultResult, fmt.Errorf("release fault during cleanup: %w", err))
				cleanupFailed = true
			} else {
				value.active = false
				faultResult.Released = true
			}
		}
		if err := s.realizer.Cleanup(cleanupCtx, value.handle); err != nil {
			appendFaultError(faultResult, fmt.Errorf("cleanup fault: %w", err))
			cleanupFailed = true
			continue
		}
		value.cleaned = true
		faultResult.CleanupComplete = true
	}
	if cleanupFailed {
		result.Omissions = append(result.Omissions, "fault cleanup incomplete")
		if result.Claim.Kind == ClaimConforming || result.Claim.Kind == ClaimViolating {
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "fault cleanup incomplete"
		}
	}
}

func appendFaultError(result *FaultResult, err error) {
	if result.Error == "" {
		result.Error = err.Error()
		return
	}
	result.Error += "; " + err.Error()
}
