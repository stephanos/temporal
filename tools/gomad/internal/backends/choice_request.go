package backends

import (
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
	control := &ChoicePrefixControl{Cutoff: uint64(len(plan.Decisions))}
	var ordinal uint64
	for _, observed := range trace.Records {
		if observed.Flags&choice.FlagObservation != 0 || observed.Alternatives < 2 {
			continue
		}
		if ordinal+1 == control.Cutoff {
			last := plan.Decisions[ordinal]
			if last.RankOverride {
				control.Override = &ChoiceRankOverride{Ordinal: ordinal, Rank: last.Selected, Reconstruct: observed.Selected == last.Selected}
			}
		}
		ordinal++
	}
	if control.Cutoff > ordinal {
		return nil, errors.Join(choice.ErrInvalidReplayPlan, errors.New("controller prefix exceeds retained source trace"))
	}
	request.TapeSHA256, request.Decisions, request.PrefixControl = plan.SHA256, control.Cutoff, control
	return request, nil
}
