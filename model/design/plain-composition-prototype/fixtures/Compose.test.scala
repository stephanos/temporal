package framework.dsl
package compose

import Joined.*
import NativeCheck.Verdict
import Probe.*

class ComposeSuite extends munit.FunSuite:
  private def verdicts(results: Vector[NativeCheck.Result]): Vector[(String, Verdict)] =
    results.map(r => r.subject -> r.verdict)

  private val init = Composed(Vector(Subject.init, Other.init))
  private def at(left: Phase, right: Phase): Composed =
    Composed(Vector(Subject.init.copy(phase = left), Other.init.copy(phase = right)))

  test("members are the vals that name Machines, in declaration order"):
    val together = ComposeRuntime(Together)
    assertEquals(together.members.map(_._1.toString), Vector("left = subject", "right = other"))
    val reordered = ComposeRuntime(Reordered)
    assertEquals(reordered.members.map(_._1.toString), Vector("right = other", "left = subject"))
    assertEquals(together.starts, Vector(init))

  test("a sync steps both members in one step, each fact in its member's case"):
    val steps = ComposeRuntime(Together).steps(init).toVector
    assertEquals(steps.map((t, _) => ComposeRuntime(Together).shown(t)), Vector("started"))
    val (_, step) = steps.head
    assertEquals(step.state, at(Phase.closed, Phase.closed))
    assertEquals(
      step.facts,
      List(JoinedFact.left(Fact.completed), JoinedFact.right(Foreign.completed))
    )
    assertEquals(step.outcome, Outcome.accepted)

  test("an action no sync pairs steps its member alone; a paired one never does"):
    val runtime = ComposeRuntime(Together)
    val from = at(Phase.closed, Phase.closed)
    val steps = runtime.steps(from).toVector
    assertEquals(steps.map((t, _) => runtime.shown(t)), Vector("left.finish"))
    assertEquals(steps.head._2.state, at(Phase.open, Phase.closed))
    assertEquals(steps.head._2.facts, List(JoinedFact.left(Fact.reopened)))
    // left is open again, but its start is paired with right's, which is closed
    assertEquals(runtime.steps(at(Phase.open, Phase.closed)).toVector, Vector.empty)
    val apart = ComposeRuntime(Apart)
    assertEquals(
      apart.steps(init).toVector.map((t, _) => apart.shown(t)),
      Vector("left.start", "right.start")
    )

  test("claims read the members' vars and helpers by ordinary calls"):
    val runtime = ComposeRuntime(Together)
    assert(runtime.at(at(Phase.closed, Phase.open))(Together.left.isClosed))
    assert(!runtime.at(init)(Together.left.isClosed))
    assert(runtime.end(at(Phase.closed, Phase.closed)))
    assert(!runtime.end(init))
    // outside an evaluation the members read as their initial states
    assertEquals(Together.left.phase, Phase.open)

  test("every claim of the sections is checked by default; the witness is found"):
    val (search, results, warnings) = checkAll(Together)
    assertEquals(search.states.size, 3)
    assertEquals(
      verdicts(results),
      Vector(
        "together.invariants.rightClosedWithLeft" -> Verdict.verified,
        "together.postconditions.startClosesBoth" -> Verdict.verified,
        "together.postconditions.finishMovesLeftAlone" -> Verdict.verified,
        "together.reopenedFound" -> Verdict.found
      )
    )
    assertEquals(results.last.detail, Vector("witness: started, left.finish"))
    assertEquals(warnings, Vector.empty)

  test("member order changes the composed state, not the member cases or the verdicts"):
    val (_, results, _) = checkAll(Reordered)
    assertEquals(verdicts(results), Vector("reordered.invariants.rightClosedWithLeft" -> Verdict.verified))
    val runtime = ComposeRuntime(Reordered)
    val (_, step) = runtime.steps(Composed(Vector(Other.init, Subject.init))).toVector.head
    assertEquals(
      step.facts,
      List(JoinedFact.left(Fact.completed), JoinedFact.right(Foreign.completed))
    )

  test("a refuted claim is a test's negative control"):
    val sameSides = invariant(check(Together.left.phase == Together.right.phase))
    val result = assertRefuted(Together, sameSides)
    assertEquals(result.subject, "together.sameSides")
    assertEquals(result.detail.head, "counterexample: started, left.finish")

  test("without the sync, Together's invariant is refuted on the members alone"):
    val (_, results, warnings) = checkAll(Apart, Some(Together))
    val r = results.find(_.subject == "apart.invariants.rightClosedWithLeft").get
    assertEquals(r.verdict, Verdict.refuted)
    assertEquals(r.detail.head, "counterexample: left.start")
    // Together's sync is no step of Apart, so the postcondition about it is vacuous there
    assertEquals(
      warnings,
      Vector("apart.postconditions.startClosesBoth: no step takes its after target")
    )

  test("a derivation replaces one member, keeping syncs, order, end and the fact type"):
    val _ = summon[LateTogether.type <:< Composition[JoinedFact]]
    val runtime = ComposeRuntime(LateTogether)
    assertEquals(
      runtime.members.map((m, actual) => s"${m.name} = ${actual.name}"),
      Vector("left = subject", "right = lateOther")
    )
    assertEquals(runtime.syncs.map(_._1.name), Vector("started"))
    val (_, step) = runtime.steps(init).toVector.head
    assertEquals(step.state, at(Phase.closed, Phase.open))
    assertEquals(
      step.facts,
      List(JoinedFact.left(Fact.completed), JoinedFact.right(Foreign.completed))
    )
    assert(!runtime.end(at(Phase.closed, Phase.open)))
    val (_, own, _) = checkAll(LateTogether)
    assertEquals(verdicts(own), Vector("lateTogether.invariants.rightStaysOpen" -> Verdict.verified))
    // the source's claims read the replacement through the vars it shares with Other
    val (_, inherited, _) = checkAll(LateTogether, Some(Together))
    assertEquals(
      verdicts(inherited),
      Vector(
        "lateTogether.invariants.rightClosedWithLeft" -> Verdict.refuted,
        "lateTogether.postconditions.startClosesBoth" -> Verdict.refuted,
        "lateTogether.postconditions.finishMovesLeftAlone" -> Verdict.verified,
        "lateTogether.reopenedFound" -> Verdict.found
      )
    )

  test("a sync of actions one member each binds names them alone; the poll names the sync's step"):
    import Scheduling.*
    val runtime = ComposeRuntime(Queue)
    assertEquals(runtime.syncs.map((s, a, b) => (s.name, a._1, b._1)), Vector(("admit", 0, 1)))
    assertEquals(runtime.own.map((i, d) => s"${runtime.name(i)}.${NativeCheck.actionName(d)}"),
      Vector("queue.schedule"))
    val (search, results, warnings) = checkAll(Queue)
    assertEquals(search.states.size, 6)
    assertEquals(
      verdicts(results),
      Vector(
        "queue.postconditions.deliveredWasScheduled" -> Verdict.verified,
        "queue.startedWithOneLeftFound" -> Verdict.found
      )
    )
    assertEquals(
      results.last.detail,
      Vector("witness: queue.schedule, queue.schedule, admit")
    )
    assertEquals(warnings, Vector.empty)

  test("the runtime refuses what it cannot run, with a located message"):
    import Refused.*
    assertEquals(
      refusal(ComposeRuntime(Doubled)),
      "doubled: left and right hold the same vars (subjectAgain's are subject's): a Machine's " +
        "vars hold one state, so each member holds vars of its own"
    )
    assert(
      refusal(ComposeRuntime(Ambiguous)).endsWith(
        "sync started: left and right bind start; name the member, left(start)"
      ),
      refusal(ComposeRuntime(Ambiguous))
    )
    assert(refusal(ComposeRuntime(Ambiguous)).startsWith("Refused.test.scala:"))
    assert(
      refusal(ComposeRuntime(Unbound)).endsWith("sync finished: right (other) binds no finish"),
      refusal(ComposeRuntime(Unbound))
    )
    assertEquals(
      refusal(Together.withMember(rigth = LateOther)),
      "rigth is not a member of together: its members are left, right"
    )
    assertEquals(
      refusal(Together.withMember(right = Subject)),
      "together.withMember(right = subject): subject does not hold other's vars; a replacement " +
        "is other or a derivation of it, so the claims that read right read it"
    )

  test("the compiler refuses a member it cannot read, a member fact without its case, a misplaced before"):
    val repeated = compileErrors(
      """object Repeated extends Composition[JoinedFact] { val right = Subject; val left = Subject }"""
    )
    assert(repeated.contains("left names Subject, which right names already"), repeated)
    val notMachine = compileErrors(
      """object Clocked extends Composition[JoinedFact] { val left = Subject; val clock = hand }"""
    )
    assert(notMachine.contains("clock names hand, which is neither a Machine nor a Composition"), notMachine)
    val untagged = compileErrors(
      """object Untagged extends Composition[Foreign] { val left = Subject }"""
    )
    assert(untagged.contains("Foreign declares a case for it, `case left(value: Fact)`"), untagged)
    val unwrapped = compileErrors(
      """object Unwrapped extends Composition[JoinedFact] { val left = Subject; object postconditions { val x = records(Fact.completed) } }"""
    )
    assert(unwrapped.contains("Required: framework.dsl.compose.Joined.JoinedFact"), unwrapped)
    val beforeInInvariant = compileErrors(
      """object Early extends Composition[JoinedFact] { val left = Subject; object invariants { val x = check(before.left.phase == Phase.open) } }"""
    )
    assert(beforeInInvariant.contains("before is the state a step started from"), beforeInInvariant)
    val misspelt = compileErrors(
      """object Misspelt extends Composition[JoinedFact] { val left = Subject; object postconditions { val x = check(before.left.phse == Phase.open) } }"""
    )
    assert(misspelt.contains("value phse is not a member of"), misspelt)
    val notEndpoint = compileErrors(
      """object Paired extends Composition[JoinedFact] { val left = Subject; val right = Other; object syncs { val s = sync(left, right) } }"""
    )
    assert(notEndpoint.contains("Required: framework.dsl.compose.Endpoint"), notEndpoint)
