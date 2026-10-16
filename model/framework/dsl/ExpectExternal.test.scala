package expectauthor

import framework.dsl.*

class ExpectExternalTest extends munit.FunSuite:
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
    val same = Subject.property.holds(predicate)
    val across = Subject.property.holdsAcross(transition)
    assertEquals(same.name, "")
    assertEquals(across.name, "")
    val member: Step[WitnessFixture.Joined, String, String] => Boolean = expect:
      records(_.left -> WitnessFixture.Subject, WitnessFixture.Fact.completed)
    val composed = WitnessFixture.Together.property.holds(member)
    assertEquals(composed.name, "")
    val joined = WitnessFixture.Joined(WitnessFixture.State.open, WitnessFixture.State.closed)
    assert(member(Step("accepted", joined, List("left_completed"))))
