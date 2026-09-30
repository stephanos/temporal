package umpire
package declarations
// The declarations the fn-107 specimens added that this framework evaluates itself: optional values,
// what a channel holds, and a refinement's visible facts. The ones only the IR interpreter checks, a
// channel's deliveries and a declared hole, make this framework's table fail rather than answer
// without them, and a monitor makes its search refuse a Query rather than answer without it.

enum Result derives Finite:
  case succeeded, failed

enum Coarse derives Finite:
  case idle, done

final case class Outline(phase: Coarse) derives Finite

enum ProductFact derives Finite:
  case finished, reset

enum Outcome derives Finite:
  case accepted, deferred

enum Stage derives Finite:
  case idle, staged, done

final case class Detail(stage: Stage) derives Finite

enum DetailFact derives Finite:
  case staged, finished, reset

object fixtures:
  val fifo: Channel[Result] = channel[Result]("fifo", capacity = 2, order = Order.fifo, loss = Loss.reliable,
    duplicates = 1)
  val bag: Channel[Result] = channel[Result]("bag", capacity = 2, order = Order.unordered, loss = Loss.lossy)

  val Family: umpire.Family = umpire.Family("fixture.declarations")
  val go: Action[EmptyTuple] = action("go", Party("fixture"))
  val prepare: Action[EmptyTuple] = internal("prepare")

  val product: Machine[Outline, Outcome, ProductFact] =
    machine[Outline, Outcome, ProductFact](Family, "product") {
      starts(Outline(Coarse.idle))
      ends(p => p.phase == Coarse.done)
      steps(go ~> (p => if p.phase == Coarse.idle then List(Step(Outcome.accepted, Outline(Coarse.done),
        List(ProductFact.finished))) else Nil))
    }

  def productOf(d: Detail): Outline = if d.stage == Stage.done then Outline(Coarse.done) else Outline(Coarse.idle)

  /** A detailed machine whose `prepare` is a stutter of `product` recording `prepared`, and whose `go`
    * records `done`. */
  def detailed(name: String, prepared: List[DetailFact], done: List[DetailFact], sees: Boolean)
      : Machine[Detail, Outcome, DetailFact] =
    machine[Detail, Outcome, DetailFact](Family, name) {
      refines(product)(productOf)
      if sees then visible(f => f != DetailFact.staged)
      starts(Detail(Stage.idle))
      ends(d => d.stage == Stage.done)
      steps(
        prepare ~> (d => if d.stage == Stage.idle then List(Step(Outcome.accepted, Detail(Stage.staged), prepared)) else Nil),
        go ~> (d => if d.stage == Stage.staged then List(Step(Outcome.accepted, Detail(Stage.done), done)) else Nil),
      )
    }

  /** A detailed machine whose `prepare` stutter answers `prepared`, and which names `accepted` as the one
    * outcome the product sees. */
  def answering(name: String, prepared: Outcome): Machine[Detail, Outcome, DetailFact] =
    machine[Detail, Outcome, DetailFact](Family, name) {
      refines(product)(productOf)
      visibleOutcomes(o => o == Outcome.accepted)
      starts(Detail(Stage.idle))
      ends(d => d.stage == Stage.done)
      steps(
        prepare ~> (d => if d.stage == Stage.idle then List(Step(prepared, Detail(Stage.staged))) else Nil),
        go ~> (d => if d.stage == Stage.staged then List(Step(Outcome.accepted, Detail(Stage.done),
          List(DetailFact.finished))) else Nil),
      )
    }

  val few: Limits = Limits("few", steps = 3, actions = 3, search = 64)

  val stagedOnce: Monitor[Detail, Outcome, DetailFact, Boolean] =
    monitor[Detail, Outcome, DetailFact, Boolean]("stagedOnce", false)((seen, _, after) =>
      seen || after.facts.contains(DetailFact.staged))(_ => false)

  val finishedSeen: Monitor[Outline, Outcome, ProductFact, Boolean] =
    monitor[Outline, Outcome, ProductFact, Boolean]("finishedSeen", false)((seen, _, after) =>
      seen || after.facts.contains(ProductFact.finished))(_ => false)

  /** A detailed machine that `stagedOnce` watches. */
  val watched: Machine[Detail, Outcome, DetailFact] =
    machine[Detail, Outcome, DetailFact](Family, "watched") {
      monitors(stagedOnce)
      starts(Detail(Stage.idle))
      ends(d => d.stage == Stage.done)
      steps(
        prepare ~> (d => if d.stage == Stage.idle then List(Step(Outcome.accepted, Detail(Stage.staged),
          List(DetailFact.staged))) else Nil),
        go ~> (d => if d.stage == Stage.staged then List(Step(Outcome.accepted, Detail(Stage.done),
          List(DetailFact.finished))) else Nil),
      )
    }

  /** The product machine, with `finishedSeen` watching it, and an unwatched machine refining it. */
  val watchedProduct: Machine[Outline, Outcome, ProductFact] =
    machine[Outline, Outcome, ProductFact](Family, "watchedProduct") {
      monitors(finishedSeen)
      starts(Outline(Coarse.idle))
      ends(p => p.phase == Coarse.done)
      steps(go ~> (p => if p.phase == Coarse.idle then List(Step(Outcome.accepted, Outline(Coarse.done),
        List(ProductFact.finished))) else Nil))
    }

  val refiningWatched: Machine[Detail, Outcome, DetailFact] =
    machine[Detail, Outcome, DetailFact](Family, "refiningWatched") {
      refines(watchedProduct)(productOf)
      starts(Detail(Stage.idle))
      ends(d => d.stage == Stage.done)
      steps(
        prepare ~> (d => if d.stage == Stage.idle then List(Step(Outcome.accepted, Detail(Stage.staged), Nil)) else Nil),
        go ~> (d => if d.stage == Stage.staged then List(Step(Outcome.accepted, Detail(Stage.done),
          List(DetailFact.finished))) else Nil),
      )
    }

final case class Both(left: Detail, right: Outline)

import fixtures.*

class DeclarationsTest extends munit.FunSuite:
  test("an optional value is None, then Some of each member, keyed by its constructor") {
    assertEquals(Finite[Option[Result]].values.map(Keys.of).toList, List("None", "Some-succeeded", "Some-failed"))
  }

  test("a FIFO channel holds every sequence of up to its capacity of deliveries, in send order") {
    val keys = fifo.contents.values.map(Keys.of)
    assertEquals(keys.size, 1 + 4 + 16)
    assertEquals(keys.take(6).toList,
      List("[]", "[succeeded-0]", "[succeeded-1]", "[failed-0]", "[failed-1]", "[succeeded-0,succeeded-0]"))
    assertEquals(Keys.of(fifo.empty.send(Result.failed).send(Result.succeeded)), "[failed-0,succeeded-0]")
  }

  test("an unordered channel holds each multiset once, whatever order its messages were sent in") {
    assertEquals(bag.contents.values.map(Keys.of).toList,
      List("[]", "[succeeded-0]", "[failed-0]", "[succeeded-0,succeeded-0]", "[succeeded-0,failed-0]",
        "[failed-0,failed-0]"))
    assertEquals(bag.empty.send(Result.failed).send(Result.succeeded), bag.empty.send(Result.succeeded).send(Result.failed))
    assert(bag.empty.send(Result.failed).send(Result.failed).isFull)
    assert(!bag.empty.send(Result.failed).isFull)
  }

  test("a stutter that records only facts the refined machine does not see refines it") {
    val r = detailed("quiet", List(DetailFact.staged), List(DetailFact.finished), sees = true).refinementCheck
    assertEquals(r.map(_.rows), Right(Vector(RefinementRow("idle-prepare", None), RefinementRow("staged-go", Some("go")))))
  }

  test("a stutter that records a fact the refined machine sees does not refine it") {
    val r = detailed("leaky", List(DetailFact.staged, DetailFact.finished), List(DetailFact.finished), sees = true)
      .refinementCheck
    assertEquals(r.left.map(_.message), Left("the row 'idle-prepare' reads as a stutter of product, and records " +
      "'finished', which product sees; a stutter records no fact the refined machine sees"))
  }

  test("without a visible projection, a stutter's facts are not read") {
    val r = detailed("unprojected", List(DetailFact.staged, DetailFact.finished), List(DetailFact.finished),
      sees = false).refinementCheck
    assert(r.isRight, r.toString)
  }

  test("a carried step's visible facts are facts the product step records") {
    val r = detailed("noisy", List(DetailFact.staged), List(DetailFact.finished, DetailFact.reset), sees = true)
      .refinementCheck
    assert(r.left.exists(_.message.startsWith("the row 'staged-go' steps from 'staged' to 'done'")), r.toString)
  }

  test("a stutter whose outcome the refined machine sees does not refine it") {
    assertEquals(answering("loud", Outcome.accepted).refinementCheck.left.map(_.message), Left("the row " +
      "'idle-prepare' reads as a stutter of product, and has the outcome 'accepted', which product sees; a " +
      "stutter emits no result the refined machine sees"))
  }

  test("a stutter whose outcome the refined machine does not see refines it") {
    val r = answering("hushed", Outcome.deferred).refinementCheck
    assertEquals(r.map(_.rows), Right(Vector(RefinementRow("idle-prepare", None), RefinementRow("staged-go", Some("go")))))
  }

  test("a machine that names the outcomes a refined machine sees must refine one") {
    val m = machine[Detail, Outcome, DetailFact](Family, "unrefinedOutcomes") {
      visibleOutcomes(o => o == Outcome.accepted)
      starts(Detail(Stage.idle))
      ends(_ => true)
      steps(go ~> (_ => Nil))
    }
    assertEquals(check(m).map(_.message),
      List("the machine names the outcomes a refined machine sees, and declares no refinement"))
  }

  test("a machine that names what a refined machine sees must refine one") {
    val m = machine[Detail, Outcome, DetailFact](Family, "unrefined") {
      visible(f => f == DetailFact.finished)
      starts(Detail(Stage.idle))
      ends(_ => true)
      steps(go ~> (_ => Nil))
    }
    assertEquals(check(m).map(_.message),
      List("the machine names the facts a refined machine sees, and declares no refinement"))
  }

  test("a member that replaces an opaque machine refines it") {
    final case class Pair(front: Outline, back: Detail)
    val back = detailed("back", List(DetailFact.staged), List(DetailFact.finished), sees = true)
    val pair = compose[Pair](Family, "pair")("front" -> product, "back" -> back).ends(_ => true)
    assertEquals(check(pair.replaces("back", product)), Nil)
    assertEquals(check(pair.replaces("back", back)).map(_.message),
      List("back replaces back, and its member back does not refine back"))
    assertEquals(check(pair.replaces("side", product)).map(_.message),
      List("side replaces product, and no member fills side"))
  }

  test("a machine that reaches a declared hole has no table here") {
    val unknown = hole("unknown")
    val m = machine[Detail, Outcome, DetailFact](Family, "holey") {
      starts(Detail(Stage.idle))
      steps(go ~> (d => if d.stage == Stage.idle then unknown.reached else Nil))
    }
    assertEquals(m.table.left.map(_.message), Left("the row 'idle-go' reaches the hole unknown, which only the IR " +
      "interpreter reads: lift the machine and check its IR"))
  }

  test("a machine that binds a channel's delivery has no table here") {
    final case class Holding(inbox: Inbox[Result])
    given Finite[Holding] =
      given Finite[Inbox[Result]] = fifo.contents
      Finite.derived
    val m = machine[Holding, Outcome, Nothing](Family, "receiving") {
      starts(Holding(fifo.empty))
      steps(fifo.deliver ~> ((_, _) => Nil))
    }
    assertEquals(m.table.left.map(_.message), Left("it binds fifoDelivery, which only the IR interpreter derives from " +
      "channel fifo: lift the machine and check its IR"))
  }

  test("a Query of a machine a monitor watches has no answer here, because this search reads no monitor") {
    val any = watched.scenario("any").starts(Detail(Stage.idle)).free
    val moves = watched.property("moves") holds (after => after.state.stage != Stage.idle)
    val q = query("watched.moves") verify moves in any limits few
    val refused = "query watched.moves: it reads machines monitors watch (stagedOnce on watched), and this " +
      "framework's search evaluates no monitor, so it has no answer here: lift the Model and check its IR"
    assertEquals(q.answer.left.map(_.toString), Left(refused))
    assertEquals(check(q).map(_.toString), List(refused))
    // A monitor disables no row, so the table itself stays.
    assert(watched.table.isRight, watched.table.toString)
  }

  test("a Query read through a refinement has no answer here when a monitor watches the refined machine") {
    val run = refiningWatched.scenario("run").starts(Detail(Stage.idle)).actions(prepare, go)
    val finishes = watchedProduct.property("finishes") holds (after => after.state.phase == Coarse.done)
    val q = query("refiningWatched.finishes").verify(finishes).in(run)(using Reads.through(refiningWatched,
      watchedProduct)) limits few
    assertEquals(q.answer.left.map(_.message), Left("it reads machines monitors watch (finishedSeen on " +
      "watchedProduct), and this framework's search evaluates no monitor, so it has no answer here: lift the " +
      "Model and check its IR"))
  }

  test("a Query of a composition has no answer here when a monitor watches one of its members") {
    val both = compose[Both](Family, "both")("left" -> watched, "right" -> product).ends(_ => true)
    val any = both.scenario("any").starts(Both(Detail(Stage.idle), Outline(Coarse.idle))).free
    val holds = both.property("anything") holds (_ => true)
    val q = query("both.anything") verify holds in any limits few
    assertEquals(q.answer.left.map(_.message), Left("it reads machines monitors watch (stagedOnce on watched), " +
      "and this framework's search evaluates no monitor, so it has no answer here: lift the Model and check its IR"))
  }

  test("a Query of machines no monitor watches is still answered") {
    val quiet = detailed("unwatched", List(DetailFact.staged), List(DetailFact.finished), sees = true)
    val any = quiet.scenario("any").starts(Detail(Stage.idle)).free
    val moves = quiet.property("moves") holds (after => after.state.stage != Stage.idle)
    val q = query("unwatched.moves") verify moves in any limits few
    assertEquals(q.answer.map(_.outcome), Right(Verdict.verifiedWithinLimits))
  }
