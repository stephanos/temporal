package expectauthor

import framework.dsl.*

class ExpectExternalTest extends munit.FunSuite:
  test("legacy finishers and private core construction are unavailable to authors"):
    val refusals = List(
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.holds(_ => true)"""),
      compileErrors(
        """framework.dsl.WitnessFixture.Subject.property.holdsAcross((_, _) => true)"""
      ),
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.sameStep(_ => true)"""),
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.transition((_, _) => true)"""),
      compileErrors(
        """framework.dsl.WitnessFixture.Subject.property.expect((s: Step[WitnessFixture.State, WitnessFixture.Outcome, WitnessFixture.Fact]) => s.state.equals(s.state))"""
      )
    )
    assertEquals(
      refusals.zipWithIndex.collect { case (errors, index) if errors.isEmpty => index },
      Nil
    )

  test("final author forwarding retains transitive member ownership even in unreachable branches"):
    import WitnessFixture.*
    import ExpectFixture.*
    val named: Step[Joined, String, String] => Boolean = expect:
      records(_.left -> WitnessFixture.Subject, WitnessFixture.Fact.completed)
    val forwarded: Step[Joined, String, String] => Boolean = expect:
      check(named(after))
    val same = Reordered.property.expect:
      check(forwarded(after))
    val repeated = Repeated.property.expect:
      check(named(after))
    assertEquals(same.name, "")
    assertEquals(repeated.name, "")
    val transition: (Joined, Step[Joined, String, String]) => Boolean = expect.across:
      records(_.right -> Other, Foreign.completed)
      check(before.left.equals(after.state.left))
    val across = Reordered.property.across.expect:
      check(transition(before, after))
    assertEquals(across.name, "")
    val _ = intercept[IllegalArgumentException]:
      Repeated.property.across.expect:
        if false then check(transition(before, after))
    val wrong: Step[Joined, String, String] => Boolean = expect:
      records(_.right -> WitnessFixture.Subject, WitnessFixture.Fact.completed)
    val _ = intercept[IllegalArgumentException]:
      Reordered.property.expect:
        if false then check(wrong(after))
    val _ = intercept[IllegalArgumentException]:
      Reordered.property.never(expect:
        check(wrong(after)))
    val _ = intercept[IllegalArgumentException]:
      Reordered.property
        .stays(_ => true)
        .unless(expect:
          check(wrong(after)))

  private case class Job(done: Boolean) derives CanEqual
  private enum Outcome:
    case accepted
  private enum Fact:
    case completed

  private object Subject extends Machine[Job, Outcome, Fact]:
    def init: Job = Job(false)
    def end(state: Job): Boolean = state.done

    object rules extends Rules

  test("an author outside the framework can build and forward ordinary expectation functions"):
    val predicate: Step[Job, Outcome, Fact] => Boolean = expect: s ?=>
      check(s.step.state.done)
      records(Fact.completed)
    val transition: (Job, Step[Job, Outcome, Fact]) => Boolean = expect.across: (b, a) ?=>
      check(!b.state.done)
      check(a.step.state.done)
    assert(predicate(Step(Outcome.accepted, Job(true), List(Fact.completed))))
    assert(!predicate(Step(Outcome.accepted, Job(true))))
    assert(transition(Job(false), Step(Outcome.accepted, Job(true))))
    val same = Subject.property.expect:
      check(predicate(after))
    val across = Subject.property.across.expect:
      check(transition(before, after))
    assertEquals(same.name, "")
    assertEquals(across.name, "")
    val member: Step[WitnessFixture.Joined, String, String] => Boolean = expect:
      records(_.left -> WitnessFixture.Subject, WitnessFixture.Fact.completed)
    val composed = WitnessFixture.Together.property.expect:
      check(member(after))
    assertEquals(composed.name, "")
    val joined = WitnessFixture.Joined(WitnessFixture.State.open, WitnessFixture.State.closed)
    assert(member(Step("accepted", joined, List("left_completed"))))
