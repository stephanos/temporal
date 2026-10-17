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

  test("author forwarding preserves tagged facts in same-step and transition claims"):
    import WitnessFixture.*
    import ExpectFixture.*
    val named: Step[Joined, String, JoinedFact] => Boolean = expect:
      records(JoinedFact.left(WitnessFixture.Fact.completed))
    def forwarded(step: Step[Joined, String, JoinedFact]): Boolean = named(step)
    val same = Reordered.property.expect:
      check(forwarded(after))
    val transition: (Joined, Step[Joined, String, JoinedFact]) => Boolean = expect.across:
      records(JoinedFact.right(Foreign.completed))
      check(before.left.equals(after.state.left))
    def across(before: Joined, after: Step[Joined, String, JoinedFact]): Boolean =
      transition(before, after)
    val property = Reordered.property.across.expect:
      check(across(before, after))
    assertEquals(same.name, "")
    assertEquals(property.name, "")
    val state = Joined(State.open, State.closed)
    assert(forwarded(Step("accepted", state, List(JoinedFact.left(WitnessFixture.Fact.completed)))))
    assert(!forwarded(Step("accepted", state, List(JoinedFact.right(Foreign.completed)))))
    assert(across(state, Step("accepted", state, List(JoinedFact.right(Foreign.completed)))))
    assert(!across(state, Step("accepted", state, List(JoinedFact.left(WitnessFixture.Fact.completed)))))

  test("method wrappers refuse predicate dependencies confined to the helper scope"):
    val refusals = List(
      compileErrors("""
        import framework.dsl.WitnessFixture.*
        import framework.dsl.ExpectFixture.*
        val named: Step[Joined, String, JoinedFact] => Boolean = expect:
          records(WitnessFixture.JoinedFact.left(WitnessFixture.Fact.completed))
        def forwarded(predicate: Step[Joined, String, JoinedFact] => Boolean,
            step: Step[Joined, String, JoinedFact]): Boolean = predicate(step)
        Reordered.property.expect:
          check(forwarded(named, after))
      """),
      compileErrors("""
        import framework.dsl.WitnessFixture.*
        import framework.dsl.ExpectFixture.*
        val named: (Joined, Step[Joined, String, JoinedFact]) => Boolean = expect.across:
          records(JoinedFact.right(Foreign.completed))
        def forwarded(before: Joined, after: Step[Joined, String, JoinedFact]): Boolean =
          val local = named
          local(before, after)
        Reordered.property.across.expect:
          check(forwarded(before, after))
      """)
    )
    assertEquals(
      refusals.filterNot(
        _.contains("expect: a forwarded predicate must be accessible outside its source helper")
      ),
      Nil
    )

  private case class Job(done: Boolean) derives CanEqual
  private enum Outcome:
    case accepted
  private enum Fact:
    case completed

  private object Subject extends RecordMachine[Job, Outcome, Fact]:
    def init: Job = Job(false)
    def end(state: Job): Boolean = state.done

    object transitions extends Rules

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
    val member: Step[WitnessFixture.Joined, String, WitnessFixture.JoinedFact] => Boolean = expect:
      records(WitnessFixture.JoinedFact.left(WitnessFixture.Fact.completed))
    val composed = WitnessFixture.Together.property.expect:
      check(member(after))
    assertEquals(composed.name, "")
    val joined = WitnessFixture.Joined(WitnessFixture.State.open, WitnessFixture.State.closed)
    assert(member(Step("accepted", joined, List(WitnessFixture.JoinedFact.left(WitnessFixture.Fact.completed)))))
