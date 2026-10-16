package framework.dsl

object MachineLawsHoleFixture:
  enum State derives CanEqual:
    case beforeHole, known, invalid

  enum Answer:
    case good, invalid

  given Finite[State] with
    def values: IndexedSeq[State] = Vector(State.beforeHole, State.known)

  given Finite[Answer] with
    def values: IndexedSeq[Answer] = Vector(Answer.good)

  given Finite[String] with
    def values: IndexedSeq[String] = Vector("known")

  object caller extends Actor:
    val interrupt = action(this)

  def subject(run: State => List[Step[State, Answer, String]]) =
    new Machine[State, Answer, String]:
      val init = MachineLawsHoleFixture.State.beforeHole
      def end(s: MachineLawsHoleFixture.State) = s == MachineLawsHoleFixture.State.known
      object rules extends Bindings(caller.interrupt ~> run)

class MachineLawsHoleTest extends munit.FunSuite:
  import MachineLawsHoleFixture.*

  test("declared row holes stay unknown without masking repeatability or later row failures"):
    val first = hole
    val second = hole
    assertEquals(first.name, second.name)
    val known = (s: State) => List(Step[State, Answer, String](Answer.good, s, List("known")))
    var differentCalls = 0
    var mixedCalls = 0
    val cases = Vector[(String, State => List[Step[State, Answer, String]], String)](
      ("stable declared hole", _ => first.reached, "inapplicable"),
      (
        "different same-name holes",
        _ =>
          differentCalls += 1
          if differentCalls % 2 == 0 then second.reached else first.reached
        ,
        "failed"
      ),
      (
        "mixed known and hole",
        s =>
          mixedCalls += 1
          if mixedCalls % 2 == 0 then known(s) else first.reached
        ,
        "failed"
      ),
      ("ordinary exception", _ => throw IllegalStateException("ordinary failure"), "failed"),
      (
        "invalid state after hole",
        s =>
          if s == State.beforeHole then first.reached
          else List(Step[State, Answer, String](Answer.good, State.invalid, List("known"))),
        "failed"
      ),
      (
        "invalid outcome after hole",
        s =>
          if s == State.beforeHole then first.reached
          else List(Step[State, Answer, String](Answer.invalid, s, List("known"))),
        "failed"
      ),
      (
        "invalid fact after hole",
        s =>
          if s == State.beforeHole then first.reached
          else List(Step[State, Answer, String](Answer.good, s, List("invalid"))),
        "failed"
      )
    )
    for (label, run, disposition) <- cases do
      val result = MachineLaws
        .check(subject(run), Some(Vector("caller.interrupt")))
        .find(_.law == "repeatable ordered relation")
        .get
      assertEquals(result.disposition, disposition, label)
      if disposition == "inapplicable" then assert(result.reason.contains("unknown"), result)
      if label == "ordinary exception" then
        assert(result.reason.contains("ordinary failure"), result)
      if label.startsWith("invalid") then
        val domain = label.split(" ")(1)
        val reason = if domain == "state" then "leaves the finite state domain"
        else s"outside the $domain domain"
        assert(result.reason.contains(reason), result)
