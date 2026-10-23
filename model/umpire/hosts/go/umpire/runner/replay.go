package runner

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	modelv1 "umpire.local/model/umpire/model/v1"
	runv1 "umpire.local/model/umpire/run/v1"
)

// admit is why a recorded Run cannot be replayed against this Runner's Model, or "".
func (r *Runner) admit(recorded *Run) string {
	switch {
	case recorded.GetModelHash() != r.hash:
		return "the recorded Run is of another model"
	case recorded.GetProtocolVersion() != ProtocolVersion:
		return fmt.Sprintf("protocol version %d is not %d", recorded.GetProtocolVersion(), ProtocolVersion)
	case recorded.GetVerdict().GetOutcome() == nil:
		return "the recorded Run has no verdict"
	}
	return ""
}

// replayRun replays every recorded protocol call against a fresh instance, compares each output
// and the verdict, and makes no Driver call. A mismatch is a Refused Run.
func (r *Runner) replayRun(ctx context.Context, opts Options) (_ *Run, err error) {
	recorded := opts.Recorded
	s, err := r.newSession(ctx, ModeReplay, 1, opts, recorded.GetWorld(), recorded.GetSeed())
	if err != nil {
		return nil, err
	}
	defer func() { err = s.close(ctx, err) }()
	if reason := r.admit(recorded); reason != "" {
		s.run.Verdict = refused(0, "replay: "+reason)
	} else if ok, err := s.replay(ctx, recorded.GetEvents()); err != nil {
		return nil, err
	} else if ok {
		derived := verdictOf(s.run.Events, recorded.GetVerdict().GetExplained().GetTermination())
		s.run.Verdict = derived
		if !proto.Equal(derived, recorded.GetVerdict()) {
			s.run.Verdict = refused(uint64(len(s.run.Events)), "replay: the recorded verdict differs from the replayed events")
		}
	}
	s.progress.finished(s.run)
	return s.run, nil
}

// question replays a recorded prefix, or only Resets without one, then asks one Propose or Judge.
func (r *Runner) question(ctx context.Context, opts Options) (_ *Run, err error) {
	world, seed := opts.World, opts.Seed
	var prefix []*runv1.Event
	if recorded := opts.Recorded; recorded != nil {
		if reason := r.admit(recorded); reason != "" {
			return nil, errors.New("question: " + reason)
		}
		world, seed, prefix = recorded.GetWorld(), recorded.GetSeed(), recorded.GetEvents()
		if opts.Prefix > len(prefix) {
			return nil, fmt.Errorf("question: the recorded Run has %d events, not %d", len(prefix), opts.Prefix)
		}
		if opts.Prefix > 0 {
			prefix = prefix[:opts.Prefix]
		}
		if len(prefix) == 0 {
			return nil, errors.New("question: the recorded Run has no events")
		}
	}
	s, err := r.newSession(ctx, ModeQuestion, 1, opts, world, seed)
	if err != nil {
		return nil, err
	}
	defer func() { err = s.close(ctx, err) }()
	if prefix != nil {
		ok, err := s.replay(ctx, prefix)
		if err != nil {
			return nil, err
		}
		if !ok {
			s.progress.finished(s.run)
			return s.run, nil
		}
		if s.run.Verdict != nil {
			return nil, fmt.Errorf("question: the recorded prefix already ends the Run; choose fewer than %d events", len(prefix))
		}
	} else if _, err := s.call(ctx, 0, 0, &modelv1.ResetRequest{Configuration: world}); err != nil {
		return nil, err
	}
	if s.run.Verdict == nil {
		if opts.Observation != nil {
			_, err = s.call(ctx, 0, 0, &modelv1.JudgeRequest{Observation: opts.Observation})
		} else {
			s.request++
			_, err = s.call(ctx, 0, 0, &modelv1.ProposeRequest{Seed: ProposalSeed(seed, s.request)})
		}
		if err != nil {
			return nil, err
		}
	}
	s.termination = runv1.Termination_TERMINATION_QUESTION
	return s.finish(), nil
}

// replay re-executes recorded events in order, appending what it reproduces. It reports false,
// with a Refused verdict, at the first event that is out of order, crossed, redacted or differs.
func (s *session) replay(ctx context.Context, events []*runv1.Event) (bool, error) {
	var lastDriver uint64
	for i, recorded := range events {
		reason := s.check(i, recorded, &lastDriver)
		var reproduced *runv1.Event
		if reason == "" {
			var err error
			if reproduced, reason, err = s.reproduce(ctx, i, recorded); err != nil {
				return false, err
			}
		}
		if reason == "" && !proto.Equal(reproduced, recorded) {
			reason = fmt.Sprintf("event %d differs from the recording", i)
		}
		if reason != "" {
			s.run.Verdict = refused(uint64(i), "replay: "+reason)
			return false, nil
		}
		s.run.Events = append(s.run.Events, reproduced)
		if s.run.Verdict == nil {
			s.run.Verdict = terminal(reproduced)
		}
	}
	return true, nil
}

// check is why an event cannot be at position i of its Run, or "". It advances the request.
func (s *session) check(i int, event *runv1.Event, lastDriver *uint64) string {
	switch {
	case event.GetSequence() != uint64(i):
		return fmt.Sprintf("event %d is out of sequence", i)
	case event.GetRedacted():
		return fmt.Sprintf("event %d is redacted", i)
	case event.GetKind() == nil:
		return fmt.Sprintf("event %d has no kind", i)
	case (i == 0) != (event.GetReset_() != nil):
		return fmt.Sprintf("event %d: a Run begins with its one Reset", i)
	case i == 0 && !proto.Equal(event.GetReset_().GetRequest().GetConfiguration(), s.run.GetWorld()):
		return "event 0 does not reset to the recorded world"
	default:
	}
	proposes := event.GetProposal() != nil || event.GetRefusal().GetMethod() == modelv1.Method_METHOD_PROPOSE
	switch {
	case event.GetReset_() != nil:
		if event.GetRequest() != 0 || event.GetDriverEvent() != 0 {
			return "event 0 is correlated"
		}
	case proposes:
		if event.GetRequest() != s.request+1 || event.GetDriverEvent() != 0 {
			return fmt.Sprintf("event %d proposes request %d after request %d", i, event.GetRequest(), s.request)
		}
		s.request, *lastDriver = event.GetRequest(), 0
	case event.GetRequest() != s.request:
		return fmt.Sprintf("event %d is crossed with request %d", i, event.GetRequest())
	default:
	}
	// A fault names the Driver event it concerns, which may be one already served.
	if driver := event.GetDriverEvent(); driver != 0 && event.GetDriverFault() == nil {
		if driver <= *lastDriver {
			return fmt.Sprintf("event %d repeats or precedes Driver event %d", i, *lastDriver)
		}
		*lastDriver = driver
	}
	return ""
}

// reproduce makes the recorded event's call again, or copies a Driver fault, and returns the
// event it produces, or why the recorded request cannot be made.
func (s *session) reproduce(ctx context.Context, i int, recorded *runv1.Event) (*runv1.Event, string, error) {
	if recorded.GetDriverFault() != nil {
		return proto.CloneOf(recorded), "", nil
	}
	request, call, err := requestOf(recorded)
	if err != nil {
		return nil, fmt.Sprintf("event %d: %v", i, err), nil
	}
	if propose, ok := request.(*modelv1.ProposeRequest); ok && propose.GetSeed() != ProposalSeed(s.run.GetSeed(), recorded.GetRequest()) {
		return nil, fmt.Sprintf("event %d proposes with another seed than the Run's", i), nil
	}
	event, err := invoke(ctx, s.host, request, call)
	if err != nil {
		return nil, "", err
	}
	event.Sequence, event.Request, event.DriverEvent = recorded.GetSequence(), recorded.GetRequest(), recorded.GetDriverEvent()
	return event, "", nil
}

// requestOf is the protocol request a recorded event sent, and its worker call correlation.
func requestOf(event *runv1.Event) (proto.Message, uint64, error) {
	switch kind := event.GetKind().(type) {
	case *runv1.Event_Reset_:
		return orEmpty(kind.Reset_.GetRequest(), new(modelv1.ResetRequest)), 0, nil
	case *runv1.Event_Proposal:
		return orEmpty(kind.Proposal.GetRequest(), new(modelv1.ProposeRequest)), 0, nil
	case *runv1.Event_Judgment:
		return orEmpty(kind.Judgment.GetRequest(), new(modelv1.JudgeRequest)), 0, nil
	case *runv1.Event_Answer:
		return orEmpty(kind.Answer.GetRequest(), new(modelv1.AnswerRequest)), kind.Answer.GetCall(), nil
	case *runv1.Event_Annotation:
		return orEmpty(kind.Annotation.GetRequest(), new(modelv1.AnnotateRequest)), 0, nil
	case *runv1.Event_Refusal:
		var request proto.Message
		switch kind.Refusal.GetMethod() {
		case modelv1.Method_METHOD_RESET:
			request = new(modelv1.ResetRequest)
		case modelv1.Method_METHOD_PROPOSE:
			request = new(modelv1.ProposeRequest)
		case modelv1.Method_METHOD_JUDGE:
			request = new(modelv1.JudgeRequest)
		case modelv1.Method_METHOD_ANSWER:
			request = new(modelv1.AnswerRequest)
		case modelv1.Method_METHOD_ANNOTATE:
			request = new(modelv1.AnnotateRequest)
		case modelv1.Method_METHOD_UNSPECIFIED, modelv1.Method_METHOD_EXPLORE, modelv1.Method_METHOD_ROWS:
			return nil, 0, fmt.Errorf("the loop does not call %s", kind.Refusal.GetMethod())
		default:
			return nil, 0, fmt.Errorf("unknown method %d", kind.Refusal.GetMethod())
		}
		if err := proto.Unmarshal(kind.Refusal.GetRequest(), request); err != nil {
			return nil, 0, fmt.Errorf("the refused request does not decode: %w", err)
		}
		return request, kind.Refusal.GetCall(), nil
	default:
		return nil, 0, errors.New("the event made no protocol call")
	}
}

// orEmpty sends an empty request for an event recorded without one; the reproduced event then
// differs from the recording.
func orEmpty[T proto.Message](request, empty T) T {
	if request.ProtoReflect().IsValid() {
		return request
	}
	return empty
}
