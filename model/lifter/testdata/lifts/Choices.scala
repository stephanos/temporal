// Named choices (model/SEMANTICS.md, Named choices): each step function of `chosen` names its
// results with `choose`, and its twin in `unchosen` writes the same steps as an unnamed list. The
// lifter's tests lift both and require one IR of the two but for the `choice` of each named step,
// positions and the names of the functions, and require the names in the order written.
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

given Accepted[Outcome] = Accepted(Outcome.accepted)

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

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

// The admission specimen's committed admission: the message consumed, or retained and delivered
// again.
def admitNamed(s: Admission): List[AdmissionStep] =
  val next = s.copy(phase = Phase.started, active = oneMore(s.active))
  choose(
    committed -> accept(
      next.copy(message = Message.empty),
      Fact.statusStarted,
      Fact.attemptAdmitted
    ),
    redelivered -> accept(
      next.copy(message = Message.redelivery),
      Fact.statusStarted,
      Fact.attemptAdmitted
    ).because("the channel may deliver the message again")
  )
def admitUnnamed(s: Admission): List[AdmissionStep] =
  val next = s.copy(phase = Phase.started, active = oneMore(s.active))
  List(
    Step(
      Outcome.accepted,
      next.copy(message = Message.empty),
      List(Fact.statusStarted, Fact.attemptAdmitted)
    ),
    Step(
      Outcome.accepted,
      next.copy(message = Message.redelivery),
      List(Fact.statusStarted, Fact.attemptAdmitted),
      "the channel may deliver the message again"
    )
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
def pauseUnnamed(s: Admission): List[AdmissionStep] = List(
  Step(Outcome.accepted, s),
  Step(Outcome.accepted, s.copy(phase = Phase.paused, message = Message.empty)),
  Step(Outcome.rejected, s, List(Fact.admissionRejected), "the pause comes too late"),
  Step(
    Outcome.accepted,
    s.copy(phase = Phase.paused),
    List(Fact.statusPaused),
    "the pause is recorded"
  )
)

// A choose in a branch of a match and of an if.
def pollNamed(s: Admission): List[AdmissionStep] = s.phase match
  case Phase.scheduled =>
    if s.message == Message.queued then
      choose(
        committed -> accept(s.copy(phase = Phase.started), Fact.statusStarted),
        held -> stay(s).because("the worker polls again")
      )
    else disabled
  case Phase.paused => choose(held -> stay(s), dropped -> accept(s.copy(message = Message.empty)))
  case _            => disabled
def pollUnnamed(s: Admission): List[AdmissionStep] = s.phase match
  case Phase.scheduled =>
    if s.message == Message.queued then
      List(
        Step(Outcome.accepted, s.copy(phase = Phase.started), List(Fact.statusStarted)),
        Step(Outcome.accepted, s, Nil, "the worker polls again")
      )
    else Nil
  case Phase.paused =>
    List(Step(Outcome.accepted, s), Step(Outcome.accepted, s.copy(message = Message.empty)))
  case _ => Nil

// Alternatives that answer different outcomes.
def answerNamed(s: Admission): List[AdmissionStep] = choose(
  committed -> accept(s.copy(active = Active.none), Fact.statusStarted),
  refused -> List(Step(Outcome.rejected, s, List(Fact.admissionRejected)))
)
def answerUnnamed(s: Admission): List[AdmissionStep] = List(
  Step(Outcome.accepted, s.copy(active = Active.none), List(Fact.statusStarted)),
  Step(Outcome.rejected, s, List(Fact.admissionRejected))
)

val chosen = machine[Admission, Outcome, Fact] {
  starts(Admission(Phase.scheduled, Message.queued, Active.none))
  ends(_ => true)
  steps(admit ~> admitNamed, pause ~> pauseNamed, poll ~> pollNamed, answer ~> answerNamed)
}

val unchosen = machine[Admission, Outcome, Fact] {
  starts(Admission(Phase.scheduled, Message.queued, Active.none))
  ends(_ => true)
  steps(admit ~> admitUnnamed, pause ~> pauseUnnamed, poll ~> pollUnnamed, answer ~> answerUnnamed)
}
