package testpilot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// AssessmentFactory is a caller's prepared, immutable assessment of one Case: whatever model and
// query it was prepared from stay its own, and Testpilot sees only the Binding and the Assessors.
// New returns state no other Run or replay has been or will be given; WithAssessment reads Binding
// once.
type AssessmentFactory interface {
	Binding() AssessmentBinding
	New(context.Context) (Assessor, error)
}

// Assessor is the passive assessment state of one Run or one replay. It is given every recorded
// Run Event once, in order, each as its own snapshot, and then the closure; it reaches no Driver,
// Slot or schedule. Testpilot calls it on a goroutine of its own, one call at a time, beside the Run
// and never inside it: an Assessor that is slow, fails, panics or never returns costs its own
// Assessment and nothing else. Each call should return when its context is canceled, which happens
// when the Run or replay it serves is over.
//
// Observe answers what the events accepted so far have Established, which Testpilot keeps whatever
// happens next. An Observe that fails ends the events: Close is still asked for its conclusions, of
// which only violations resting on accepted events are kept.
//
// An Assessor embeds SingleUse, which is what makes it the state of one Run or replay only.
type Assessor interface {
	Observe(context.Context, *testpilotspb.RunEvent) (Established, error)
	Close(context.Context, AssessmentClosure) (*AssessmentOutcome, error)
	claim() bool
}

// Established is what the events an Assessor has accepted settle for good: violations, which no
// later event takes back. Nonconformance says no modeled execution explains them; Violations are
// properties already violated. Each rests only on events up to the one just accepted, and the first
// report of each is the one kept. Satisfaction is never established early: it is Close's to say.
type Established struct {
	Nonconformance *ConformanceAssessment
	Violations     []PropertyAssessment
}

// SingleUse is embedded by value in every Assessor. It lets Testpilot take the state exactly once:
// a factory that hands the same state to a second Run or replay, through any binding and however
// much later, is refused with ErrAssessorState.
type SingleUse struct{ claimed atomic.Bool }

func (s *SingleUse) claim() bool { return s != nil && s.claimed.CompareAndSwap(false, true) }

// AssessmentBinding is what a factory was prepared for. Case is the CaseFingerprint of its Case,
// which WithAssessment holds against the admitted one; Model and Query are carried by every
// Assessment, so a Run assessed under one binding is not replayed under another.
type AssessmentBinding struct {
	Case   string
	Model  string
	Query  string
	Limits AssessmentLimits
}

// AssessmentLimits are the ceilings one assessment is evaluated under. An assessment that needs
// more than them fails with AssessmentLimitExceeded rather than concluding from a part.
type AssessmentLimits struct {
	// MaxEvents is the most Run Events an Assessor is given.
	MaxEvents int64
	// MaxProperties is the most property conclusions an Assessment carries.
	MaxProperties int64
	// MaxDuration is the most time one assessment spends inside its Assessor's Observe and Close
	// calls, added up. Time in which the Assessor is not being called is not charged, so a live Run
	// and a replay of it charge the same calls. The call that crosses the ceiling ends the
	// assessment, returned or not, with no conclusion.
	MaxDuration time.Duration
}

// AssessmentClosure is what a closed Run says beyond its events.
type AssessmentClosure struct {
	Disposition testpilotspb.RunDisposition
	Cleanup     testpilotspb.CleanupStatus
	// EvaluationFailureSequence is the event the Contract's own evaluation failed on, or zero.
	EvaluationFailureSequence int64
}

// ConformanceStatus is conformant when a modeled execution explains the recorded trace,
// nonconformant when none does, and inconclusive when the evidence cannot tell.
type ConformanceStatus string

const (
	ConformanceConformant    ConformanceStatus = "conformant"
	ConformanceNonconformant ConformanceStatus = "nonconformant"
	ConformanceInconclusive  ConformanceStatus = "inconclusive"
)

// PropertyStatus is one property's conclusion; evidence that cannot settle it leaves it
// inconclusive.
type PropertyStatus string

const (
	PropertySatisfied    PropertyStatus = "satisfied"
	PropertyViolated     PropertyStatus = "violated"
	PropertyInconclusive PropertyStatus = "inconclusive"
)

// ConformanceAssessment says whether the model explains the trace. SupportingEventSequences are the
// Run Events that decided it, ascending; Detail is bounded human-readable text and no stable API.
type ConformanceAssessment struct {
	Status                   ConformanceStatus
	SupportingEventSequences []int64
	Detail                   string
}

// PropertyAssessment is one property's conclusion, apart from conformance.
type PropertyAssessment struct {
	ID                       string
	Status                   PropertyStatus
	SupportingEventSequences []int64
	Detail                   string
}

// AssessmentOutcome is what an Assessor concludes.
type AssessmentOutcome struct {
	Conformance ConformanceAssessment
	Properties  []PropertyAssessment
}

// Assessment is what an Assessor concluded about one Run, beside the Contract's Verdict and never
// part of it. Model and Query are the identities of the binding it was made under. When Failure
// is set, only the violations the Assessor had established are kept; every other conclusion is
// inconclusive or absent.
type Assessment struct {
	Model       string
	Query       string
	Conformance ConformanceAssessment
	Properties  []PropertyAssessment
	Failure     *AssessmentFailure
}

// AssessmentFailure says why an Assessment is incomplete. EventSequence is the Run Event the
// Assessor failed on, or zero when it failed on none.
type AssessmentFailure struct {
	Code          AssessmentFailureCode
	Detail        string
	EventSequence int64
}

// AssessmentFailureCode is a stable classification of an assessment failure.
type AssessmentFailureCode string

const (
	AssessmentObserveFailed  AssessmentFailureCode = "observe_failed"
	AssessmentCloseFailed    AssessmentFailureCode = "close_failed"
	AssessmentOutcomeInvalid AssessmentFailureCode = "outcome_invalid"
	AssessmentLimitExceeded  AssessmentFailureCode = "limit_exceeded"
)

var (
	// ErrAssessorState says a factory gave a Run or replay no Assessor, or one another Run or replay
	// was given.
	ErrAssessorState = errors.New("assessment factory returned no fresh Assessor")
	// ErrForeignAssessment says a Run was assessed under another model or query than the one it is
	// being replayed under.
	ErrForeignAssessment = errors.New("the Run was assessed under another model or query")
)

const (
	maxAssessmentProperties  = 10000
	maxAssessmentIdentity    = 256
	maxAssessmentDetailBytes = 1024
	maxAssessmentDuration    = 24 * time.Hour
)

// CaseFingerprint identifies a Case by its content. It is taken of the message, not of a
// producer's canonical bytes, which Testpilot never sees; it is compared, never recorded.
func CaseFingerprint(source *testpilotspb.Case) (string, error) {
	if source == nil {
		return "", preparationError(errors.New("a Case is required"), "case")
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	if err != nil {
		return "", preparationError(err, "case")
	}
	fingerprint := sha256.Sum256(append([]byte("testpilot.case/v1"), encoded...))
	return hex.EncodeToString(fingerprint[:]), nil
}

// AssessedCase is a prepared Case with a caller's assessment bound to it. Its Run and Evaluate are
// the prepared Case's own, with a fresh Assessor beside the Contract each time.
type AssessedCase struct {
	prepared *PreparedCase
	factory  AssessmentFactory
	binding  AssessmentBinding
	clock    assessmentClock
}

// assessmentClock is the time the duration ceiling is charged in.
type assessmentClock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time                         { return time.Now() }
func (systemClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// WithAssessment binds factory to p, which is unchanged and keeps the Contract path alone.
// Rejections expose *PreparationError through errors.As.
func (p *PreparedCase) WithAssessment(factory AssessmentFactory) (*AssessedCase, error) {
	if p == nil || p.source == nil {
		return nil, preparationError(errors.New("prepared Case is required"), "assessment")
	}
	if ir.IsNil(factory) {
		return nil, preparationError(errors.New("assessment factory is required"), "assessment.factory")
	}
	var binding AssessmentBinding
	if err := guarded("Binding", func() error {
		binding = factory.Binding()
		return nil
	}); err != nil {
		return nil, preparationError(err, "assessment.factory")
	}
	fingerprint, err := CaseFingerprint(p.source)
	if err != nil {
		return nil, err
	}
	if binding.Case != fingerprint {
		return nil, preparationError(ir.Invalid(ir.TypeMismatch, "assessment.binding.case", "assessment factory is bound to another Case"), "assessment")
	}
	for _, identity := range []struct{ path, value string }{{"assessment.binding.model", binding.Model}, {"assessment.binding.query", binding.Query}} {
		if !validAssessmentIdentity(identity.value) {
			return nil, preparationError(errors.New("identity must be non-empty text of at most 256 bytes"), identity.path)
		}
	}
	for _, limit := range []struct {
		path           string
		value, ceiling int64
	}{
		{"assessment.binding.limits.max_events", binding.Limits.MaxEvents, execution.ProgramCeiling().GetMaxRunEvents()},
		{"assessment.binding.limits.max_properties", binding.Limits.MaxProperties, maxAssessmentProperties},
		{"assessment.binding.limits.max_duration", int64(binding.Limits.MaxDuration), int64(maxAssessmentDuration)},
	} {
		if limit.value <= 0 || limit.value > limit.ceiling {
			return nil, preparationError(ir.Invalid(ir.LimitExceeded, limit.path, fmt.Sprintf("ceiling must be between 1 and %d", limit.ceiling)), "assessment")
		}
	}
	return &AssessedCase{prepared: p, factory: factory, binding: binding, clock: systemClock{}}, nil
}

func validAssessmentIdentity(value string) bool {
	return value != "" && len(value) <= maxAssessmentIdentity && utf8.ValidString(value)
}

// guarded is the one boundary every call into a caller's factory or Assessor crosses: a panic
// there comes back as an error naming the callback, and goes no further.
func guarded(callback string, call func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("assessment %s panicked: %v", callback, recovered)
		}
	}()
	return call()
}

// Run is the prepared Case's Run with a fresh Assessor beside it, and returns what the Assessor
// concluded with the Run and the Verdict, which are what they would be without it. A factory that
// gives no fresh Assessor fails the Run before the Driver is opened.
func (c *AssessedCase) Run(ctx context.Context, driver Driver) (*testpilotspb.Run, *testpilotspb.Verdict, *Assessment, error) {
	if c == nil {
		return nil, nil, nil, errors.New("assessed Case is required")
	}
	var session *assessmentSession
	defer func() { session.stop() }()
	run, verdict, err := c.prepared.run(ctx, driver, func() (execution.EventObserver, error) {
		var err error
		session, err = c.open(ctx)
		return session, err
	})
	if run == nil {
		return nil, verdict, nil, err
	}
	return run, verdict, session.finish(run), err
}

// Evaluate is the prepared Case's Evaluate, and also drives a fresh Assessor through the recorded
// events: the Evaluation carries its Assessment. recorded is the Assessment the Run was given when
// it ran, or nil when it has none; one made under another model or query is ErrForeignAssessment,
// before anything is evaluated.
func (c *AssessedCase) Evaluate(ctx context.Context, run *testpilotspb.Run, recorded *Assessment) (*testpilotspb.Verdict, *Evaluation, error) {
	if c == nil || ir.IsNil(ctx) {
		return nil, nil, errors.New("assessed Case and context are required")
	}
	if recorded != nil && (recorded.Model != c.binding.Model || recorded.Query != c.binding.Query) {
		return nil, nil, fmt.Errorf("%w: recorded model %q query %q, bound model %q query %q", ErrForeignAssessment,
			recorded.Model, recorded.Query, c.binding.Model, c.binding.Query)
	}
	session, err := c.open(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer session.stop()
	verdict, evaluation, err := c.prepared.Evaluate(ctx, run)
	if err != nil {
		return verdict, nil, err
	}
	evaluation.Assessment = session.finish(run)
	return verdict, evaluation, nil
}

// open takes fresh state from the factory and starts the goroutine that owns it.
func (c *AssessedCase) open(ctx context.Context) (*assessmentSession, error) {
	var assessor Assessor
	if err := guarded("New", func() (err error) {
		assessor, err = c.factory.New(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAssessorState, err)
	}
	if ir.IsNil(assessor) {
		return nil, ErrAssessorState
	}
	if !assessor.claim() {
		return nil, fmt.Errorf("%w: its state was given to another Run or replay", ErrAssessorState)
	}
	// The Assessor's calls outlive the caller's cancellation, as the Verdict's do, and end with the
	// session.
	callbacks, cancel := context.WithCancel(context.WithoutCancel(ctx))
	session := &assessmentSession{
		limits: c.binding.Limits, model: c.binding.Model, query: c.binding.Query, clock: c.clock,
		cancel: cancel, wake: make(chan struct{}, 1), concluded: make(chan *Assessment, 1),
	}
	go (&assessorState{limits: c.binding.Limits, assessor: assessor}).serve(callbacks, session)
	return session, nil
}

// assessmentSession is the Run's side of one assessment: it queues event snapshots for the
// goroutine that owns the Assessor, keeps the time that goroutine spends in the Assessor, and
// waits for it only once the Run has closed.
type assessmentSession struct {
	limits       AssessmentLimits
	model, query string
	clock        assessmentClock
	cancel       context.CancelFunc
	wake         chan struct{}
	concluded    chan *Assessment

	mu      sync.Mutex
	pending []*testpilotspb.RunEvent
	queued  int64
	closed  *closedRun
	stopped bool
	// spent is the time of the Assessor's returned calls. calling is the event of the call in
	// progress, since started; a Close is event zero.
	spent   time.Duration
	inCall  bool
	calling int64
	started time.Time
	record  assessmentRecord
}

// assessmentRecord is what an assessment holds for certain while its Assessor is still at work: how
// far the Assessor has accepted the events, what it has established from them, and the first
// failure. The session owns it, so it outlasts an Assessor that never returns.
type assessmentRecord struct {
	observed       int64
	nonconformance *ConformanceAssessment
	violations     []PropertyAssessment
	failure        *AssessmentFailure
}

// closedRun is what the Assessor's goroutine needs of a closed Run once its events are queued.
type closedRun struct {
	closure AssessmentClosure
	events  int64
}

// Observe implements execution.EventObserver. It runs inside the Contract Monitor's callback, so
// it only queues: the Assessor is never called from here.
func (s *assessmentSession) Observe(_ context.Context, event *testpilotspb.RunEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue(event)
}

// queue keeps the next event for the Assessor. One event past the ceiling is kept, which is the one
// the Assessor's goroutine fails on; the rest could only be dropped there.
func (s *assessmentSession) queue(event *testpilotspb.RunEvent) {
	sequence := event.GetSequence()
	s.queued = sequence
	if !s.stopped && sequence <= s.limits.MaxEvents+1 {
		s.pending = append(s.pending, event)
		s.signal()
	}
}

func (s *assessmentSession) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// next hands the Assessor's goroutine one queued event, the closed Run once there is one, and
// whether the session has stopped. One event at a time is all that goroutine ever holds, so an
// Assessor that never returns keeps one snapshot reachable and not the trace.
func (s *assessmentSession) next() (*testpilotspb.RunEvent, *closedRun, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var event *testpilotspb.RunEvent
	if len(s.pending) > 0 {
		event, s.pending[0] = s.pending[0], nil
		if s.pending = s.pending[1:]; len(s.pending) == 0 {
			s.pending = nil
		}
	}
	return event, s.closed, s.stopped
}

// charge runs one Assessor call and adds its time to what the assessment has spent. It reports
// whether that crossed the duration ceiling.
func (s *assessmentSession) charge(sequence int64, call func()) bool {
	s.mu.Lock()
	s.inCall, s.calling, s.started = true, sequence, s.clock.Now()
	s.mu.Unlock()
	call()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spent += s.clock.Now().Sub(s.started)
	s.inCall = false
	return s.spent > s.limits.MaxDuration
}

// fail records a failure. The first one is the assessment's: a later failure does not replace it.
func (s *assessmentSession) fail(code AssessmentFailureCode, detail string, sequence int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failLocked(code, detail, sequence)
}

func (s *assessmentSession) failLocked(code AssessmentFailureCode, detail string, sequence int64) {
	if s.record.failure == nil {
		s.record.failure = &AssessmentFailure{Code: code, Detail: boundedAssessmentText(detail), EventSequence: sequence}
	}
}

// overran records that the call for event sequence crossed the duration ceiling. It reads the same
// whether that call returned late or never.
func (s *assessmentSession) overran(sequence int64) {
	s.fail(AssessmentLimitExceeded, fmt.Sprintf("assessment spent more than its duration ceiling of %s in its Assessor", s.limits.MaxDuration), sequence)
}

func (s *assessmentSession) failed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.record.failure != nil
}

// accept records that the Assessor took event sequence and keeps what it established by it. A
// violation resting on an event the Assessor has not taken is not established, and fails the
// assessment.
func (s *assessmentSession) accept(sequence int64, established Established) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if problem := establishedProblem(established, sequence); problem != "" {
		s.failLocked(AssessmentOutcomeInvalid, problem, sequence)
		return
	}
	s.record.observed = sequence
	if established.Nonconformance != nil && s.record.nonconformance == nil {
		nonconformance := boundedConformance(*established.Nonconformance)
		s.record.nonconformance = &nonconformance
	}
	for _, violation := range established.Violations {
		if slices.ContainsFunc(s.record.violations, func(held PropertyAssessment) bool { return held.ID == violation.ID }) {
			continue
		}
		if limit := s.limits.MaxProperties; int64(len(s.record.violations)) >= limit {
			s.failLocked(AssessmentLimitExceeded, fmt.Sprintf("established violations exceed the assessment's property ceiling of %d", limit), sequence)
			return
		}
		s.record.violations = append(s.record.violations, boundedProperty(violation))
	}
}

// settle is the one place an Assessment is composed, and every path that ends an assessment goes
// through it. outcome is what Close concluded in time and in a form Testpilot can report, or nil.
// In order of precedence:
//
//  1. An established violation stands. It is a nonconformance or a violated property that Observe
//     reported, or that Close reported resting only on events the Assessor accepted. No later
//     failure, ceiling or conclusion takes it back.
//  2. A failure is reported next, the first of: an Observe or Close that failed or panicked, an
//     outcome Testpilot cannot report, and an exceeded ceiling. A call that crosses the duration
//     ceiling is that failure whatever it returned. Under a failure every conclusion not
//     established under 1 is inconclusive: partial evidence never satisfies and never conforms.
//  3. Without a failure, Close's outcome is the rest of the Assessment as the Assessor concluded
//     it: violated, inconclusive or satisfied.
func (s *assessmentSession) settle(outcome *AssessmentOutcome) *Assessment {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := &s.record
	established := len(record.violations)
	properties := slices.Clone(record.violations)
	if outcome != nil {
		for _, property := range outcome.Properties {
			if !slices.ContainsFunc(record.violations, func(held PropertyAssessment) bool { return held.ID == property.ID }) {
				properties = append(properties, boundedProperty(property))
			}
		}
	}
	if limit := s.limits.MaxProperties; int64(len(properties)) > limit {
		s.failLocked(AssessmentLimitExceeded, fmt.Sprintf("%d properties exceed the assessment's property ceiling of %d", len(properties), limit), 0)
		properties = violatedProperties(properties, limit)
	}
	result := &Assessment{Model: s.model, Query: s.query, Conformance: ConformanceAssessment{Status: ConformanceInconclusive}, Properties: properties, Failure: record.failure}
	switch {
	case record.nonconformance != nil:
		result.Conformance = *record.nonconformance
	case outcome != nil && (record.failure == nil || rests(outcome.Conformance.Status == ConformanceNonconformant, outcome.Conformance.SupportingEventSequences, record.observed)):
		result.Conformance = boundedConformance(outcome.Conformance)
	default:
	}
	if record.failure != nil {
		for index, property := range properties {
			if index >= established && !rests(property.Status == PropertyViolated, property.SupportingEventSequences, record.observed) {
				properties[index] = PropertyAssessment{ID: property.ID, Status: PropertyInconclusive}
			}
		}
	}
	return result
}

// rests reports whether a conclusion is a violation resting only on events the Assessor accepted.
func rests(violation bool, support []int64, observed int64) bool {
	return violation && supportedBy(support, observed)
}

// finish queues the events of the closed Run the live Run did not deliver, which are the ones past
// a Contract evaluation failure or, on a replay, all of them, and waits for the conclusion. A live
// Run and a replay end here on the same record and the same charge, which is what makes them
// agree. The wait ends when the call in progress has crossed the duration ceiling.
func (s *assessmentSession) finish(run *testpilotspb.Run) *Assessment {
	s.mu.Lock()
	for _, event := range run.GetEvents() {
		if event.GetSequence() > s.queued {
			s.queue(proto.CloneOf(event))
		}
	}
	s.closed = &closedRun{events: int64(len(run.GetEvents())), closure: AssessmentClosure{
		Disposition:               run.GetDisposition(),
		Cleanup:                   run.GetCleanup().GetStatus(),
		EvaluationFailureSequence: run.GetEvaluationFailureSequence(),
	}}
	s.signal()
	s.mu.Unlock()

	for {
		s.mu.Lock()
		left, sequence := s.limits.MaxDuration-s.spent, s.calling
		if s.inCall {
			left -= s.clock.Now().Sub(s.started)
		}
		s.mu.Unlock()
		if left < 0 {
			s.overran(sequence)
			return s.settle(nil)
		}
		select {
		case result := <-s.concluded:
			return result
		// Time between calls is not charged, so what is left is read again rather than assumed spent.
		case <-s.clock.After(left + 1):
		}
	}
}

// stop ends the session. The Assessor's context is canceled, what is queued is dropped, and its
// goroutine leaves at its next return, whether or not anyone still waits for it.
func (s *assessmentSession) stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.stopped, s.pending = true, nil
	s.signal()
	s.mu.Unlock()
	s.cancel()
}

// assessorState is the Assessor on the goroutine that owns it from open to conclusion.
type assessorState struct {
	limits   AssessmentLimits
	assessor Assessor
	// spent says the duration ceiling was crossed. The Assessor is not called again.
	spent bool
}

func (s *assessorState) serve(ctx context.Context, session *assessmentSession) {
	for {
		event, closed, stopped := session.next()
		switch {
		// A stopped session has no one waiting: its Assessor is not called again.
		case stopped:
			return
		case event != nil:
			s.observe(ctx, session, event)
		case closed != nil:
			session.concluded <- s.conclude(ctx, session, closed)
			return
		default:
			<-session.wake
		}
	}
}

func (s *assessorState) observe(ctx context.Context, session *assessmentSession, event *testpilotspb.RunEvent) {
	if session.failed() {
		return
	}
	sequence := event.GetSequence()
	if limit := s.limits.MaxEvents; sequence > limit {
		session.fail(AssessmentLimitExceeded, fmt.Sprintf("Run events exceed the assessment's event ceiling of %d", limit), sequence)
		return
	}
	var established Established
	var err error
	s.spent = session.charge(sequence, func() {
		err = guarded("Observe", func() (err error) {
			established, err = s.assessor.Observe(ctx, event)
			return err
		})
	})
	switch {
	case s.spent:
		session.overran(sequence)
	case err != nil:
		session.fail(AssessmentObserveFailed, err.Error(), sequence)
	default:
		session.accept(sequence, established)
	}
}

func (s *assessorState) conclude(ctx context.Context, session *assessmentSession, run *closedRun) *Assessment {
	if s.spent {
		return session.settle(nil)
	}
	var outcome *AssessmentOutcome
	var err error
	if session.charge(0, func() {
		err = guarded("Close", func() (err error) {
			outcome, err = s.assessor.Close(ctx, run.closure)
			return err
		})
	}) {
		session.overran(0)
		return session.settle(nil)
	}
	if err != nil {
		session.fail(AssessmentCloseFailed, err.Error(), 0)
		return session.settle(nil)
	}
	if problem := outcomeProblem(outcome, run.events); problem != "" {
		session.fail(AssessmentOutcomeInvalid, problem, 0)
		return session.settle(nil)
	}
	return session.settle(outcome)
}

// violatedProperties is what an outcome over its property ceiling keeps: its violations, in the
// order reported, up to the ceiling.
func violatedProperties(properties []PropertyAssessment, limit int64) []PropertyAssessment {
	var kept []PropertyAssessment
	for _, property := range properties {
		if property.Status == PropertyViolated && int64(len(kept)) < limit {
			kept = append(kept, property)
		}
	}
	return kept
}

// outcomeProblem names what makes an outcome one Testpilot cannot report, or nothing. Such an
// outcome is discarded whole: no part of it is known to be sound.
func outcomeProblem(outcome *AssessmentOutcome, events int64) string {
	if outcome == nil {
		return "no outcome"
	}
	switch outcome.Conformance.Status {
	case ConformanceConformant, ConformanceNonconformant, ConformanceInconclusive:
	default:
		return "conformance status is not a conclusion"
	}
	if !supportedBy(outcome.Conformance.SupportingEventSequences, events) {
		return "conformance support names no ascending Run Events"
	}
	seen := make(map[string]struct{}, len(outcome.Properties))
	for index, property := range outcome.Properties {
		if !validAssessmentIdentity(property.ID) {
			return fmt.Sprintf("property %d has an invalid id", index)
		}
		if _, repeated := seen[property.ID]; repeated {
			return fmt.Sprintf("property %q is reported twice", property.ID)
		}
		seen[property.ID] = struct{}{}
		switch property.Status {
		case PropertySatisfied, PropertyViolated, PropertyInconclusive:
		default:
			return fmt.Sprintf("property %q status is not a conclusion", property.ID)
		}
		if !supportedBy(property.SupportingEventSequences, events) {
			return fmt.Sprintf("property %q support names no ascending Run Events", property.ID)
		}
	}
	return ""
}

// establishedProblem names what keeps an Observe's report from being established by the events up
// to accepted, or nothing.
func establishedProblem(established Established, accepted int64) string {
	if nonconformance := established.Nonconformance; nonconformance != nil {
		if nonconformance.Status != ConformanceNonconformant {
			return "established conformance is not a nonconformance"
		}
		if !supportedBy(nonconformance.SupportingEventSequences, accepted) {
			return "conformance support names no ascending Run Events"
		}
	}
	seen := make(map[string]struct{}, len(established.Violations))
	for index, property := range established.Violations {
		if !validAssessmentIdentity(property.ID) {
			return fmt.Sprintf("property %d has an invalid id", index)
		}
		if _, repeated := seen[property.ID]; repeated {
			return fmt.Sprintf("property %q is reported twice", property.ID)
		}
		seen[property.ID] = struct{}{}
		if property.Status != PropertyViolated {
			return fmt.Sprintf("property %q is established without being violated", property.ID)
		}
		if !supportedBy(property.SupportingEventSequences, accepted) {
			return fmt.Sprintf("property %q support names no ascending Run Events", property.ID)
		}
	}
	return ""
}

func supportedBy(sequences []int64, events int64) bool {
	previous := int64(0)
	for _, sequence := range sequences {
		if sequence <= previous || sequence > events {
			return false
		}
		previous = sequence
	}
	return true
}

func boundedConformance(source ConformanceAssessment) ConformanceAssessment {
	return ConformanceAssessment{Status: source.Status, SupportingEventSequences: slices.Clone(source.SupportingEventSequences), Detail: boundedAssessmentText(source.Detail)}
}

func boundedProperty(source PropertyAssessment) PropertyAssessment {
	return PropertyAssessment{ID: source.ID, Status: source.Status, SupportingEventSequences: slices.Clone(source.SupportingEventSequences), Detail: boundedAssessmentText(source.Detail)}
}

func boundedAssessmentText(value string) string {
	return strings.ToValidUTF8(value[:min(len(value), maxAssessmentDetailBytes)], "?")
}
