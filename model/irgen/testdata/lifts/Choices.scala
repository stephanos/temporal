// Named choices (model/SEMANTICS.md, Named choices): each step function of `chosen` names its
// results with `choose`. The lifter's tests require each name on the step its alternative wrote, in
// the order written, and an alternative that calls a function to call a copy of it whose every step
// carries the name. An unnamed list of several steps is refused (Rejects.scala), so fn-120.1's
// unnamed twins of these functions are retired; the Go tooling holds the names inert.
package fixture.choices

import umpire.*

given Family = Family("fixture.choices")

enum Phase derives Finite:
  case scheduled, started, paused

enum Message derives Finite:
  case empty, queued, redelivery

enum Active derives Finite:
  case none, one, two

final case class Admission(phase: Phase, message: Message, active: Active) derives Finite

enum Outcome derives Finite:
  case accepted, rejected

given Ok[Outcome] = Ok(Outcome.accepted)

enum Fact derives Finite:
  case statusStarted, attemptAdmitted, statusPaused, admissionRejected

type AdmissionStep = Step[Admission, Outcome, Fact]

val committed = choice
val redelivered = choice
val held = choice
val dropped = choice
val refused = choice

val admit = action(Party("matching"))
val pause = action(Party("user"))
val poll = action(Party("worker"))
val answer = action(Party("worker"))
val retry = action(Party("matching"))
val resume = action(Party("user"))

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

// The admission specimen's committed admission: the message consumed, or retained and delivered
// again.
def admitNamed(s: Admission): List[AdmissionStep] =
  val next = s.copy(phase = Phase.started, active = oneMore(s.active))
  choose(
    committed -> enter(
      next.copy(message = Message.empty),
      Fact.statusStarted,
      Fact.attemptAdmitted
    ),
    redelivered -> enter(
      next.copy(message = Message.redelivery),
      Fact.statusStarted,
      Fact.attemptAdmitted
    ).because("the channel may deliver the message again")
  )

// Each spelling of one step: stay, List(Step(...)) with its defaults, Step with every field, and
// List(Step(...)).because(...).
def pauseNamed(s: Admission): List[AdmissionStep] = choose(
  held -> stay(s),
  dropped -> List(Step(Outcome.accepted, s.copy(phase = Phase.paused, message = Message.empty))),
  refused -> List(
    Step(Outcome.rejected, s, List(Fact.admissionRejected), "the pause comes too late")
  ),
  committed -> List(Step(Outcome.accepted, s.copy(phase = Phase.paused), List(Fact.statusPaused)))
    .because("the pause is recorded")
)

// A choose in a branch of a match and of an if.
def pollNamed(s: Admission): List[AdmissionStep] = s.phase match
  case Phase.scheduled =>
    if s.message == Message.queued then
      choose(
        committed -> enter(s.copy(phase = Phase.started), Fact.statusStarted),
        held -> stay(s).because("the worker polls again")
      )
    else disabled
  case Phase.paused => choose(held -> stay(s), dropped -> enter(s.copy(message = Message.empty)))
  case _            => disabled

// Alternatives that answer different outcomes.
def answerNamed(s: Admission): List[AdmissionStep] = choose(
  committed -> enter(s.copy(active = Active.none), Fact.statusStarted),
  refused -> List(Step(Outcome.rejected, s, List(Fact.admissionRejected)))
)

// A step two actions share: no step while paused, a refusal once started, or the admission. Resume
// takes it unnamed; the alternatives of retry call it, directly and through another function.
def admitted(s: Admission, m: Message): List[AdmissionStep] =
  if s.phase == Phase.paused then disabled
  else if s.phase == Phase.started then
    List(Step(Outcome.rejected, s, List(Fact.admissionRejected)))
  else enter(s.copy(phase = Phase.started, message = m), Fact.statusStarted)
def redeliveredStep(s: Admission): List[AdmissionStep] = admitted(s, Message.redelivery)

def retryNamed(s: Admission): List[AdmissionStep] = choose(
  committed -> admitted(s, Message.empty),
  held -> stay(s),
  redelivered -> redeliveredStep(s)
)
def resumeStep(s: Admission): List[AdmissionStep] = admitted(s, Message.queued)

val chosen = machine[Admission, Outcome, Fact] {
  starts(Admission(Phase.scheduled, Message.queued, Active.none))
  ends(_ => true)
  steps(
    admit ~> admitNamed,
    pause ~> pauseNamed,
    poll ~> pollNamed,
    answer ~> answerNamed,
    retry ~> retryNamed,
    resume ~> resumeStep
  )
}
