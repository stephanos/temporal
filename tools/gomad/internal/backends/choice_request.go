package backends

import (
	"bytes"
	"errors"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func ChoiceRequestForCapability(capability *ChoiceCapability) (*ChoiceRequest, error) {
	if capability == nil {
		return nil, nil
	}
	request := &ChoiceRequest{
		Mode: capability.Mode, ExecutionIdentity: capability.ExecutionIdentity, Limit: capability.Limit,
		Trace: capability.Trace, TraceSHA256: capability.TraceSHA256, TapeSHA256: capability.TapeSHA256,
		Decisions: capability.Decisions, PrefixControl: capability.PrefixControl,
	}
	if capability.ReplayPlan == nil {
		return request, nil
	}
	if request.Mode != choice.ModePrefix || request.PrefixControl != nil {
		return nil, errors.Join(choice.ErrInvalidReplayPlan, errors.New("controller tape requires prefix mode without explicit controls"))
	}
	plan, err := choice.ValidatePrefixReplayPlan(*capability.ReplayPlan, request.ExecutionIdentity)
	if err != nil {
		return nil, err
	}
	records, err := choice.TraceRecordCount(request.Trace)
	if err != nil {
		return nil, err
	}
	trace, err := choice.DecodeStoredTrace(choice.Profile, request.Trace, choice.TerminalMetadata{
		State: choice.TerminalComplete, Limit: request.Limit, Records: records, SHA256: request.TraceSHA256,
	})
	if err != nil {
		return nil, err
	}
	source, err := choice.ProjectReplayPlan(trace, request.ExecutionIdentity)
	if err != nil {
		return nil, err
	}
	control := &ChoicePrefixControl{Cutoff: uint64(len(plan.Decisions))}
	var expected choice.ReplayPlan
	if control.Cutoff != 0 && plan.Decisions[control.Cutoff-1].RankOverride {
		last := plan.Decisions[control.Cutoff-1]
		if last.Ordinal >= uint64(len(source.Decisions)) {
			return nil, choice.ErrInvalidReplayPlan
		}
		override := &ChoiceRankOverride{Ordinal: last.Ordinal, Rank: last.Selected, Reconstruct: source.Decisions[last.Ordinal].Selected == last.Selected}
		control.Override = override
		if override.Reconstruct {
			expected, err = choice.BuildForcedRankPrefix(source, last.Ordinal, last.Selected)
		} else {
			expected, err = choice.BuildRankPrefix(source, last.Ordinal, last.Selected)
		}
	} else {
		expected, err = source.Prefix(control.Cutoff)
	}
	if err != nil {
		return nil, err
	}
	if expected.SHA256 != plan.SHA256 || !bytes.Equal(expected.Bytes, plan.Bytes) {
		return nil, errors.Join(choice.ErrInvalidReplayPlan, errors.New("controller prefix differs from retained source trace"))
	}
	request.TapeSHA256, request.Decisions, request.PrefixControl = plan.SHA256, control.Cutoff, control
	return request, nil
}
