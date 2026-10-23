package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/model"
	modelv1 "umpire.local/model/umpire/model/v1"
	runv1 "umpire.local/model/umpire/run/v1"
)

// session is one Run against one Host. Every Host call goes through it, from one goroutine, in
// the order events were received.
type session struct {
	host        *model.Host
	run         *Run
	opts        Options
	request     uint64
	termination runv1.Termination
	limit       uint64
	stopped     bool
	progress    progress
}

func (r *Runner) newSession(ctx context.Context, mode Mode, iteration int, opts Options, world *anypb.Any, seed uint64) (*session, error) {
	host, err := r.newHost(ctx, r.wasm)
	if err != nil {
		return nil, fmt.Errorf("model host: %w", err)
	}
	return &session{
		host: host,
		run:  &Run{ModelHash: r.hash, ProtocolVersion: ProtocolVersion, World: world, Seed: seed},
		opts: opts,
		progress: progress{
			out: opts.Progress, mode: mode, iteration: iteration, seed: seed,
		},
	}, nil
}

func (s *session) close(ctx context.Context, err error) error {
	return errors.Join(err, s.host.Close(context.WithoutCancel(ctx)))
}

// finish sets the Explained verdict when no event ended the Run, applies redaction and reports.
func (s *session) finish() *Run {
	if s.run.Verdict == nil {
		s.run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: s.termination, Limit: s.limit}}}
	}
	if s.opts.Redact != nil {
		for _, event := range s.run.Events {
			if s.opts.Redact(event) {
				event.Redacted = true
			}
		}
	}
	s.progress.finished(s.run)
	return s.run
}

// append records an event at the next position under the current request. The first terminal
// event decides the verdict; later events, such as faults while closing, never change it.
func (s *session) append(event *runv1.Event) {
	event.Sequence = uint64(len(s.run.Events))
	event.Request = s.request
	s.run.Events = append(s.run.Events, event)
	if s.run.Verdict == nil {
		s.run.Verdict = terminal(event)
	}
}

func (s *session) fault(driverEvent uint64, reason string) {
	s.append(&runv1.Event{DriverEvent: driverEvent, Kind: &runv1.Event_DriverFault{DriverFault: &runv1.DriverFault{Reason: reason}}})
}

func (s *session) eventsSpent() bool { return len(s.run.Events) >= s.opts.EventBudget }

// spend ends the Run on its event budget.
func (s *session) spend() {
	s.termination, s.limit, s.stopped = runv1.Termination_TERMINATION_EVENT_BUDGET, uint64(s.opts.EventBudget), true
}

// terminal is the verdict an event decides, or nil when the Run goes on.
func terminal(event *runv1.Event) *runv1.Verdict {
	switch {
	case event.GetJudgment().GetResponse().GetUnexplained() != nil:
		return &runv1.Verdict{Outcome: &runv1.Verdict_Unexplained{Unexplained: &runv1.Unexplained{Event: event.Sequence}}}
	case event.GetRefusal() != nil:
		refusal := event.GetRefusal()
		if refusal.Method == modelv1.Method_METHOD_PROPOSE && refusal.GetRefusal().GetCode() == modelv1.RefusalCode_REFUSAL_CODE_QUIESCENT {
			return &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUIESCENT}}}
		}
		return refused(event.Sequence, fmt.Sprintf("model refused %s: %s", refusal.Method, refusal.GetRefusal().GetCode()))
	case event.GetDriverFault() != nil:
		return refused(event.Sequence, "driver fault: "+event.GetDriverFault().GetReason())
	}
	return nil
}

func refused(event uint64, reason string) *runv1.Verdict {
	return &runv1.Verdict{Outcome: &runv1.Verdict_Refused{Refused: &runv1.Refused{Event: event, Reason: reason}}}
}

// verdictOf is the verdict a sequence of events decides. When no event ends the Run, the
// recorded termination must fit the events: a proposal budget is the number of proposals, the
// last of them accepted; an event budget is the number of events; a question ends with the one
// Propose or Judge it asked.
func verdictOf(events []*runv1.Event, recorded *runv1.Explained) *runv1.Verdict {
	for _, event := range events {
		if verdict := terminal(event); verdict != nil {
			return verdict
		}
	}
	if len(events) == 0 {
		return refused(0, "a Run begins with its Reset")
	}
	fits := false
	switch recorded.GetTermination() {
	case runv1.Termination_TERMINATION_PROPOSAL_BUDGET:
		fits = proposals(events) == recorded.GetLimit() && accepted(events)
	case runv1.Termination_TERMINATION_EVENT_BUDGET:
		fits = uint64(len(events)) == recorded.GetLimit()
	case runv1.Termination_TERMINATION_QUESTION:
		last := events[len(events)-1]
		fits = recorded.GetLimit() == 0 && last.GetDriverEvent() == 0 && (last.GetProposal() != nil || last.GetJudgment() != nil)
	default:
	}
	if !fits {
		return refused(uint64(len(events)), "the recorded termination does not fit the events")
	}
	return &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: recorded.GetTermination(), Limit: recorded.GetLimit()}}}
}

func proposals(events []*runv1.Event) uint64 {
	var count uint64
	for _, event := range events {
		if event.GetProposal() != nil {
			count++
		}
	}
	return count
}

// accepted reports whether the last proposal has a final Explained observation.
func accepted(events []*runv1.Event) bool {
	last := proposals(events)
	for _, event := range slices.Backward(events) {
		judgment := event.GetJudgment().GetResponse()
		switch {
		case event.GetRequest() != last:
			return false
		case judgment.GetExplained() != nil && !judgment.GetContinuePolling():
			return true
		default:
		}
	}
	return false
}

// call makes one protocol call and records it, as its event or as a Refusal. The error is a
// Host failure, never a refusal.
func (s *session) call(ctx context.Context, driverEvent, workerCall uint64, request proto.Message) (*runv1.Event, error) {
	event, err := invoke(ctx, s.host, request, workerCall)
	if err != nil {
		return nil, err
	}
	event.DriverEvent = driverEvent
	s.append(event)
	return event, nil
}

// invoke makes one typed protocol call and returns the event recording it, without position or
// correlation. Runner and replay share it, so both record a call the same way.
func invoke(ctx context.Context, host *model.Host, request proto.Message, workerCall uint64) (*runv1.Event, error) {
	var event *runv1.Event
	var method modelv1.Method
	var err error
	switch typed := request.(type) {
	case *modelv1.ResetRequest:
		method = modelv1.Method_METHOD_RESET
		var reply *modelv1.ResetResponse
		if reply, err = host.Reset(ctx, typed); err == nil {
			event = &runv1.Event{Kind: &runv1.Event_Reset_{Reset_: &runv1.Reset{Request: typed, Response: reply}}}
		}
	case *modelv1.ProposeRequest:
		method = modelv1.Method_METHOD_PROPOSE
		var reply *modelv1.ProposeResponse
		if reply, err = host.Propose(ctx, typed); err == nil {
			event = &runv1.Event{Kind: &runv1.Event_Proposal{Proposal: &runv1.Proposal{Request: typed, Response: reply}}}
		}
	case *modelv1.JudgeRequest:
		method = modelv1.Method_METHOD_JUDGE
		var reply *modelv1.JudgeResponse
		if reply, err = host.Judge(ctx, typed); err == nil {
			event = &runv1.Event{Kind: &runv1.Event_Judgment{Judgment: &runv1.Judgment{Request: typed, Response: reply}}}
		}
	case *modelv1.AnswerRequest:
		method = modelv1.Method_METHOD_ANSWER
		var reply *modelv1.AnswerResponse
		if reply, err = host.Answer(ctx, typed); err == nil {
			event = &runv1.Event{Kind: &runv1.Event_Answer{Answer: &runv1.Answer{Request: typed, Response: reply, Call: workerCall}}}
		}
	case *modelv1.AnnotateRequest:
		method = modelv1.Method_METHOD_ANNOTATE
		var reply *modelv1.AnnotateResponse
		if reply, err = host.Annotate(ctx, typed); err == nil {
			event = &runv1.Event{Kind: &runv1.Event_Annotation{Annotation: &runv1.Annotation{Request: typed, Response: reply}}}
		}
	default:
		return nil, fmt.Errorf("the loop does not send %T", request)
	}
	var refusal *model.RefusalError
	if errors.As(err, &refusal) {
		data, marshalErr := proto.MarshalOptions{Deterministic: true}.Marshal(request)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return &runv1.Event{Kind: &runv1.Event_Refusal{Refusal: &runv1.Refusal{
			Method: method, Request: data, Refusal: &modelv1.Refusal{Code: refusal.Code, Message: refusal.Message}, Call: workerCall,
		}}}, nil
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

// execute is one driven Run: Reset, then Propose and exchange until the Model is quiescent, a
// budget is spent or an event ends the Run.
func (r *Runner) execute(ctx context.Context, mode Mode, iteration int, opts Options, world *anypb.Any, seed uint64, budget int) (_ *Run, err error) {
	s, err := r.newSession(ctx, mode, iteration, opts, world, seed)
	if err != nil {
		return nil, err
	}
	defer func() { err = s.close(ctx, err) }()
	runCtx, cancel := context.WithTimeout(ctx, opts.RunTimeout)
	defer cancel()
	if _, err := s.call(ctx, 0, 0, &modelv1.ResetRequest{Configuration: world}); err != nil {
		return nil, err
	}
	if s.run.Verdict != nil {
		return s.finish(), nil
	}
	driver, err := r.newDriver(runCtx)
	if err != nil {
		s.fault(0, "driver: "+err.Error())
		return s.finish(), nil
	}
	err = s.drive(ctx, runCtx, driver, seed, budget)
	if closer, ok := driver.(io.Closer); ok {
		if closeErr := closer.Close(); closeErr != nil && err == nil {
			s.fault(0, "driver close: "+closeErr.Error())
		}
	}
	if err != nil {
		return nil, err
	}
	return s.finish(), nil
}

func (s *session) drive(parent, ctx context.Context, driver Driver, seed uint64, budget int) error {
	for proposals := 0; ; proposals++ {
		if proposals == budget {
			s.termination, s.limit = runv1.Termination_TERMINATION_PROPOSAL_BUDGET, uint64(budget)
			return nil
		}
		if s.eventsSpent() {
			s.spend()
			return nil
		}
		s.request++
		s.progress.proposal(len(s.run.Events), s.request)
		event, err := s.call(parent, 0, 0, &modelv1.ProposeRequest{Seed: ProposalSeed(seed, s.request)})
		if err != nil {
			return err
		}
		if event.GetProposal() == nil {
			return nil
		}
		if s.eventsSpent() {
			s.spend()
			return nil
		}
		if err := s.exchange(parent, ctx, driver, event.GetProposal().GetResponse()); err != nil {
			return err
		}
		if s.run.Verdict != nil || s.stopped {
			return nil
		}
	}
}

// exchange has the Driver act on one proposal and serves its events in receive order until it
// completes, faults or an event ends the Run.
func (s *session) exchange(parent, ctx context.Context, driver Driver, proposal *modelv1.ProposeResponse) error {
	exchange, late, err := s.start(parent, ctx, driver, proposal.GetRequest())
	if parent.Err() != nil {
		if exchange != nil {
			s.closeExchange(exchange)
		}
		return parent.Err()
	}
	if late {
		return nil
	}
	if err != nil {
		s.fault(0, "start: "+err.Error())
		return nil
	}
	if exchange == nil {
		s.fault(0, "start returned no exchange")
		return nil
	}
	defer s.closeExchange(exchange)
	var last uint64
	final := false
	for {
		event, open, wait := s.next(ctx, exchange)
		if parent.Err() != nil {
			return parent.Err()
		}
		if reason := s.admitEvent(event, open, wait, last); reason != "" {
			s.fault(event.ID, reason)
			return nil
		}
		last = event.ID
		switch event.Kind {
		case EventDone:
			if !final {
				s.fault(event.ID, "exchange completed before the Model accepted a final observation")
			}
			return nil
		case EventFault:
			s.fault(event.ID, event.Fault)
			return nil
		default:
		}
		decision, accepted, err := s.serve(parent, event)
		if err != nil {
			return err
		}
		final = final || accepted
		respondCtx, cancel := context.WithTimeout(ctx, s.opts.EventTimeout)
		err = exchange.Respond(respondCtx, event.ID, decision)
		cancel()
		if err != nil {
			s.fault(event.ID, "respond: "+err.Error())
			return nil
		}
		if s.run.Verdict != nil {
			return nil
		}
		if s.eventsSpent() {
			s.spend()
			return nil
		}
	}
}

// start sends the proposed request, waiting at most the read timeout. A Start that is late
// or interrupted ends the exchange: its canceled ctx must make it return, and what it started
// is closed and joined before the Run goes on. late reports that the exchange already ended.
func (s *session) start(parent, ctx context.Context, driver Driver, payload *anypb.Any) (_ Exchange, late bool, _ error) {
	type started struct {
		exchange Exchange
		err      error
	}
	startCtx, cancel := context.WithTimeout(ctx, s.opts.EventTimeout)
	defer cancel()
	done := make(chan started, 1)
	request := s.request
	go func() {
		exchange, err := driver.Start(startCtx, request, payload)
		done <- started{exchange, err}
	}()
	select {
	case got := <-done:
		return got.exchange, false, got.err
	case <-startCtx.Done():
	}
	if parent.Err() == nil {
		s.fault(0, "start did not return within the read timeout")
	}
	grace := time.NewTimer(s.opts.EventTimeout)
	defer grace.Stop()
	select {
	case got := <-done:
		if got.exchange != nil {
			s.closeExchange(got.exchange)
		}
	case <-grace.C:
		// Nothing more can be joined; close whatever it starts once it returns.
		s.fault(0, "start did not return after its context ended")
		go func() {
			if got := <-done; got.exchange != nil {
				_ = got.exchange.Close()
			}
		}()
	}
	return nil, true, nil
}

// admitEvent is why a received event ends the exchange as a Driver fault, or "".
func (s *session) admitEvent(event Event, open bool, wait string, last uint64) string {
	switch {
	case wait != "":
		return wait
	case !open:
		return "event stream closed without completion"
	case event.Request != s.request:
		return fmt.Sprintf("event %d correlates with request %d, not %d", event.ID, event.Request, s.request)
	case event.ID <= last:
		return fmt.Sprintf("event %d repeats or precedes event %d", event.ID, last)
	case event.Kind < EventObservation || event.Kind > EventFault:
		return fmt.Sprintf("event %d has unknown kind %d", event.ID, event.Kind)
	default:
		return ""
	}
}

// serve asks the Model about one Observation, Callback or Annotate event and returns the
// decision for the Driver, and whether the Model accepted a final observation.
func (s *session) serve(ctx context.Context, event Event) (Decision, bool, error) {
	var request proto.Message
	var workerCall uint64
	switch event.Kind {
	case EventObservation:
		request = &modelv1.JudgeRequest{Observation: event.Payload}
	case EventCallback:
		request, workerCall = &modelv1.AnswerRequest{Callback: event.Payload}, event.Call
	default:
		request = &modelv1.AnnotateRequest{Input: event.Payload}
	}
	recorded, err := s.call(ctx, event.ID, workerCall, request)
	if err != nil {
		return Decision{}, false, err
	}
	judgment := recorded.GetJudgment().GetResponse()
	switch {
	case recorded.GetRefusal() != nil:
		return Decision{Refusal: recorded.GetRefusal().GetRefusal()}, false, nil
	case judgment != nil:
		return Decision{Judgment: judgment}, judgment.GetExplained() != nil && !judgment.GetContinuePolling(), nil
	case recorded.GetAnswer() != nil:
		return Decision{Reply: recorded.GetAnswer().GetResponse().GetReply()}, false, nil
	default:
		return Decision{Annotation: recorded.GetAnnotation().GetResponse().GetOutput()}, false, nil
	}
}

// next waits for the exchange's next event, at most the read timeout.
func (s *session) next(ctx context.Context, exchange Exchange) (Event, bool, string) {
	timer := time.NewTimer(s.opts.EventTimeout)
	defer timer.Stop()
	select {
	case event, open := <-exchange.Events():
		return event, open, ""
	case <-timer.C:
		return Event{}, false, "no event within the read timeout without Model acceptance"
	case <-ctx.Done():
		return Event{}, false, "the run time budget ended without Model acceptance"
	}
}

// closeExchange cancels and joins the exchange, waiting at most the read timeout for a Driver
// that does not return.
func (s *session) closeExchange(exchange Exchange) {
	done := make(chan error, 1)
	go func() { done <- exchange.Close() }()
	timer := time.NewTimer(s.opts.EventTimeout)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			s.fault(0, "close: "+err.Error())
		}
	case <-timer.C:
		s.fault(0, "close did not return within the read timeout")
	}
}
