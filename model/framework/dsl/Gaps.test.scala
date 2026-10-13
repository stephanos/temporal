package framework

import scala.compiletime.testing.typeCheckErrors

class GapsSuite extends munit.FunSuite:
  enum Phase:
    case ready, done
  enum Fact:
    case finished

  test("every lint kind has an inert typed subject, including the optional kind"):
    val move = action(Actor("caller"))
    val taken = choice
    val subjects: Seq[Subject] = Seq(
      silent(move, in = Phase.ready),
      byDefault(move, in = Phase.done),
      neverEnabled(move),
      unconstrained(move()),
      unproduced(Fact.finished),
      unperformed(move),
      unevidenced(Fact.finished),
      uncovered(move()),
      unrealized,
      unread,
      witnessOnly,
      untaken(taken),
      unasked,
      unfired,
      unreachable(Phase.done),
      stuck(Phase.done),
      unreadObservation(Observation(on = Entity())),
      unreachableBinding(move()),
      mustNotPinned(move, in = Phase.ready),
      unmapped(com.google.protobuf.struct.NullValue.NULL_VALUE),
      explicitWait()
    )
    assertEquals(
      subjects.map(_.kind),
      Seq(
        "silent-rejection",
        "disabled-by-default",
        "never-enabled",
        "unconstrained-result",
        "unproduced",
        "unperformed-action",
        "unevidenced-fact",
        "uncovered-class",
        "unrealized-find",
        "unread-refinement",
        "witness-only",
        "untaken-choice",
        "unasked-property",
        "unfired-verify",
        "unreachable-value",
        "stuck-state",
        "unread-observation",
        "unreachable-binding",
        "must-not-pinned",
        "unmodeled-api-value",
        "explicit-wait"
      )
    )
    val annotation = new gap(subjects*)(because = "a known source gap")
    assertEquals(annotation.because, "a known source gap")
    assertEquals(annotation.until, "")
    assertEquals(annotation.subjects.size, subjects.size)

  test("subject references cannot be authored as string identities"):
    for errors <- Seq(
        typeCheckErrors("framework.silent(\"caller.move\")"),
        typeCheckErrors("framework.unconstrained(\"caller.move\")"),
        typeCheckErrors("framework.unproduced(\"finished\")"),
        typeCheckErrors("framework.untaken(\"choice\")"),
        typeCheckErrors("framework.unmapped(\"API_VALUE\")"),
        typeCheckErrors("framework.explicitWait(\"poll\")")
      )
    do assert(errors.nonEmpty)
