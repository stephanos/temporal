package umpire.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class EntersTest extends munit.FunSuite:
  private def rewrite(source: String, context: List[String] = Nil) =
    def parse(s: String) = dialects.Scala3(Input.String(s)).parse[Source].get
    Rules(List(new Fn159Task7Enters(context.map(parse))))
      .syntacticPatch(SyntacticDocument.fromTree(parse(source)), suppress = false)

  private val header = """import framework.*
enum Phase:
  case idle, busy
final case class State(phase: Phase, count: UpTo[1])
private object Subject extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)):
  object effects:
"""

  test("phase-only definitions, aliases, rule references and choose calls migrate together"):
    val legacy = header + """    def work(s: State) = enter(s.copy(phase = Phase.busy))
    val alias = work
    def pick(s: State) = choose(done -> alias(s), kept -> stay(s))
    def meaningful(s: State) = enter(s.copy(phase = Phase.busy, count = UpTo(1)))
  object rules extends Rules:
    on(hand.press) { always ~> effects.work }
"""
    val expected =
      header + """    def pick(s: State) = choose(done -> enters(Phase.busy)(s), kept -> stay(s))
    def meaningful(s: State) = enter(s.copy(phase = Phase.busy, count = UpTo(1)))
  object rules extends Rules:
    on(hand.press) { always ~> enters(Phase.busy) }
"""
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, expected)
    assertEquals(rewrite(result.fixed).fixed, expected)

  test("Recorded migration requires exactly the declared status, and genuine logic stays"):
    val recorded = header.replace(
      "enum Phase:\n  case idle, busy",
      "enum Phase(val status: Fact) extends Recorded[Fact]:\n  case idle extends Phase(Fact.waiting)\n  case busy extends Phase(Fact.working), Held"
    )
    val legacy =
      recorded + """    def work(s: State) = enter(s.copy(phase = Phase.busy), Fact.working)
    def extra(s: State) = enter(s.copy(phase = Phase.busy), Fact.extra)
    def refused(s: State) = reject(Outcome.refused, s)
  object rules extends Rules:
    on(hand.press) { always ~> effects.work }
"""
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assert(result.fixed.contains("always ~> enters(Phase.busy)"))
    assert(
      result.fixed.contains("def extra(s: State) = enter(s.copy(phase = Phase.busy), Fact.extra)")
    )
    assert(result.fixed.contains("def refused(s: State) = reject(Outcome.refused, s)"))
    val missing = legacy.replace(", Fact.working)", ")")
    assertEquals(rewrite(missing).fixed, missing)
    assert(rewrite(missing).diagnostics.nonEmpty)

  test("unknown phase semantics, incorrect writers and unsafe external users refuse atomically"):
    val body =
      "    def work(s: State) = enter(s.copy(phase = Phase.busy))\n  object rules extends Rules:\n    on(hand.press) { always ~> effects.work }\n"
    for source <- List(
        header.replace("enum Phase:\n  case idle, busy\n", "") + body,
        header.replace(", (s, p) => s.copy(phase = p)", "") + body,
        header.replace("s.copy(phase = p)", "s.copy(count = UpTo(1))") + body,
        header.replace("private object Subject", "object Subject") + body,
        header.replace("import framework.*", "import framework.*\nimport unrelated.*") + body,
        header.replace("import framework.*", "import framework.*\ntrait Recorded[F]") + body,
        header + body + "val escaped = Subject.effects.work\n",
        header + body.replace("effects.work", "effects.work(other)")
      )
    do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
      assert(result.diagnostics.forall(_.message.contains("manually")))

  test(
    "a facet uses the concrete typed writer and explicit plain enum from the supplied source set"
  ):
    val facet = """import framework.*
trait Answering extends Facet[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:
  object answering:
    object effects:
      def backOff(s: AnswerState) = enter(s.copy(phase = AnswerPhase.backingOff))
    object rules extends Rules:
      on(fault.delay) { always ~> effects.backOff }
"""
    val context = List("""enum AnswerPhase:
  case waiting, backingOff
final case class AnswerState(phase: AnswerPhase, attempts: UpTo[2])
object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase](_.phase, (s, p) => s.copy(phase = p)), Answering
""")
    val result = rewrite(facet, context)
    assertEquals(result.diagnostics, Nil)
    assert(result.fixed.contains("always ~> enters(AnswerPhase.backingOff)"))
    assert(!result.fixed.contains("def backOff"))
    assertEquals(rewrite(facet).fixed, facet)
    assert(rewrite(facet).diagnostics.nonEmpty)
