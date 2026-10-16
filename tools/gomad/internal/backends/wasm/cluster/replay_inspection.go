package simulation

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// InspectRetained creates an isolated retained replay and terminates at the
// selected predecision boundary, without projecting an unconsumed suffix as a
// completed cluster execution.
func InspectRetained(ctx context.Context, payload target.BackendPayload, module RetainedModule, directory string, installed InstalledHelper, resolved cluster.WASMInspectionResolution, check string) (result wasi.Result, retErr error) {
	bundle, p, retained, err := verifyRetained(payload)
	if err != nil {
		return result, err
	}
	if err := bindRetainedPrimary(retained, module.SHA256); err != nil {
		return result, err
	}
	control, err := InspectionControl(retained.WASM.Inspection, resolved, check)
	if err != nil {
		return result, err
	}
	admission := retainedAdmission{bundle: bundle, provider: p}
	spec, p, err := admission.reopen(module, directory, installed)
	if err != nil {
		return result, err
	}
	p.inspectionTarget, p.inspectionControl = resolved.Target, control
	opened, err := p.Open(ctx, spec)
	if err != nil {
		return result, err
	}
	e := opened.(*execution)
	defer func() {
		if e.cluster != nil {
			retErr = errors.Join(retErr, e.cluster.Shutdown(ctx))
		} else {
			retErr = errors.Join(retErr, e.Shutdown(ctx))
		}
	}()
	find := func() bool {
		observations := p.CompletedResults()
		for _, observed := range observations {
			if observed.Result.Termination == "inspection_aborted" {
				if target, err := e.lookup(resolved.Target); err == nil {
					select {
					case <-target.done:
					case <-ctx.Done():
						return false
					}
					observations = p.CompletedResults()
				}
				break
			}
		}
		for _, observed := range observations {
			if observed.Handle == resolved.Target && observed.Result.Termination == "inspected_prefix" {
				result = observed.Result
				return true
			}
		}
		return false
	}
	selectedFailure := func() error {
		for _, observed := range p.CompletedResults() {
			if observed.Handle != resolved.Target {
				continue
			}
			switch observed.Result.Termination {
			case "debug_error", "probe_fuel_limit", "capacity", "fuel":
				return fmt.Errorf("inspection evaluation %s: %s", observed.Result.Termination, observed.Result.Message)
			case "instrumentation_divergence":
				return fmt.Errorf("%w: %s", cluster.ErrWASMInstrumentationDivergence, observed.Result.Message)
			}
		}
		return nil
	}
	for _, transition := range spec.Replay.Transitions {
		if err := selectedFailure(); err != nil {
			return result, err
		}
		if find() {
			break
		}
		var stepErr error
		switch transition.Action {
		case cluster.LifecycleStart, cluster.LifecycleRestart:
			var node cluster.NodeSpec
			for _, n := range spec.Nodes {
				if n.ID == transition.Handle.Node {
					node = n
					break
				}
			}
			stepErr = e.Start(node, transition.Handle)
		case cluster.LifecycleWait:
			_, stepErr = e.Wait(ctx, transition.Handle)
		case cluster.LifecycleStop:
			_, stepErr = e.Stop(ctx, transition.Handle)
		case cluster.LifecycleCrash:
			_, stepErr = e.Crash(ctx, transition.Handle)
		default:
			stepErr = errors.New("unsupported retained inspection lifecycle")
		}
		if stepErr != nil {
			if err := selectedFailure(); err != nil {
				return result, err
			}
			if !find() {
				var boundary *wasi.BoundaryError
				if errors.As(stepErr, &boundary) && (boundary.Kind == "debug_error" || boundary.Kind == "probe_fuel_limit" || boundary.Kind == "capacity" || boundary.Kind == "fuel") {
					return result, stepErr
				}
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				return result, fmt.Errorf("%w: %v", cluster.ErrWASMInstrumentationDivergence, stepErr)
			}
			break
		}
	}
	if err := selectedFailure(); err != nil {
		return result, err
	}
	if !find() {
		return result, fmt.Errorf("%w: validated inspection boundary not reached", cluster.ErrWASMInstrumentationDivergence)
	}
	if e.cluster == nil {
		return result, errors.New("inspection requires retained shared helper")
	}
	finished, err := e.cluster.Finish(ctx)
	if err != nil {
		return result, fmt.Errorf("finish inspected cluster: %w", err)
	}
	if !finished.GroupGone {
		return result, errors.New("inspection helper not reaped")
	}
	metadata, err := cluster.DecodeWASMInspection(finished.InspectionMetadata)
	if err != nil {
		return result, err
	}
	if err := cluster.VerifyWASMInspectionPrefix(retained.WASM.Inspection, metadata, resolved.Boundary); err != nil {
		return result, err
	}
	result.InspectionMetadata = finished.InspectionMetadata
	return result, nil
}
