package umpire
// The declarations the fn-107 specimens added that this framework evaluates itself: optional values,
// what a channel holds, and a refinement's visible facts. The ones only the IR interpreter checks, a
// channel's deliveries and a declared hole, make this framework's table fail rather than answer
// without them.

import DeclarationsTest.*

object DeclarationsTest:
  enum Result derives Finite:
    case succeeded, failed

  val fifo: Channel[Result] = channel[Result]("fifo", capacity = 2, order = Order.fifo, loss = Loss.reliable,
    duplicates = 1)
  val bag: Channel[Result] = channel[Result]("bag", capacity = 2, order = Order.unordered, loss = Loss.lossy)

  val Family: umpire.Family = umpire.Family("fixture.declarations")
  val go: Action[EmptyTuple] = action("go", Party("fixture"))
  val prepare: Action[EmptyTuple] = internal("prepare")

  enum Coarse derives Finite:
    case idle, done

  final case class Product(phase: Coarse) derives Finite

  enum ProductFact derives Finite:
    case finished, reset

  enum Outcome derives Finite:
    case accepted

  val product: Machine[Product, Outcome, ProductFact] =
    machine[Product, Outcome, ProductFact](Family, "product") {
      starts(Product(Coarse.idle))
      ends(p => p.phase == Coarse.done)
      steps(go ~> (p => if p.phase == Coarse.idle then List(Step(Outcome.accepted, Product(Coarse.done),
        List(ProductFact.finished))) else Nil))
    }

  enum Stage derives Finite:
    case idle, staged, done

  final case class Detail(stage: Stage) derives Finite

  enum DetailFact derives Finite:
    case staged, finished, reset

  def productOf(d: Detail): Product = if d.stage == Stage.done then Product(Coarse.done) else Product(Coarse.idle)

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
    final case class Pair(front: Product, back: Detail)
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
