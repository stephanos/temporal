package framework.checker

import framework.dsl.{Checker as _, *}
import framework.dsl.Finite.given
import java.lang.management.ManagementFactory
import java.lang.ref.{Reference, WeakReference}
import scala.jdk.CollectionConverters.*

object DiscoveryLifetimeFixture:
  given Ok[Boolean] = Ok(true)

  class Model(seed: Int) extends Machine[Boolean, Nothing]:
    var value = seed

    object user extends Actor:
      lazy val advance = action
      lazy val alias = advance
      def current: Int = value

    object transitions extends Transitions:
      on(user.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

    object invariants:
      val bounded =
        check(value < 2)

    object monitors:
      val watched = monitor(false)(seen => seen || value == 2)(identity)

    object progress:
      val arrives = leadsTo(from = value == 0, to = value == 2, within = 2)

object DiscoveryCatalogueFixture:
  given Ok[Boolean] = Ok(true)

  object catalogue:
    var offered: Action[EmptyTuple] = null // scalafix:ok DisableSyntax.null
    var wire: Channel[Boolean] = null // scalafix:ok DisableSyntax.null
    var missing: Hole = null // scalafix:ok DisableSyntax.null
    var watched: Monitor[State, Boolean, Nothing, Boolean] = null // scalafix:ok DisableSyntax.null

  class Model extends Machine[Boolean, Nothing]:
    var value = 0

    object user extends Actor:
      lazy val advance = action

    object transitions extends Transitions

class DiscoveryLifetimeTest extends munit.FunSuite:
  import DiscoveryLifetimeFixture.{*, given}

  private def collect(label: String, refs: Vector[WeakReference[?]]): Unit =
    def collections =
      ManagementFactory.getGarbageCollectorMXBeans.asScala.map(_.getCollectionCount.max(0L)).sum
    val before = collections
    var rounds = 0
    while rounds < 40 && (rounds == 0 || refs.exists(_.get != null)) do
      System.gc()
      Thread.sleep(25)
      rounds += 1
    val observed = collections - before
    val survivors = refs.count(_.get != null)
    println(s"GC $label: rounds=$rounds collections=$observed survivors=$survivors/${refs.size}")
    assert(observed > 0, s"$label: no collection observed")
    assertEquals(survivors, 0, label)

  private def batch(checked: Boolean): Vector[WeakReference[Model]] = Vector.fill(16):
    val model = new Model(0)
    if checked then
      val report = Checker.verifyAll(model, budget = 3)
      assertEquals(report.result("bounded").verdict, Verdict.refuted)
      assertEquals(report.result("arrives").verdict, Verdict.verified)
    new WeakReference(model)

  test("checked declarations and owners release with constructed controls"):
    collect("constructed", batch(false))
    collect("checked", batch(true))

  private def retained(): (Array[Result], WeakReference[Model]) =
    val model = new Model(0)
    val report = Checker.verifyAll(model, budget = 3)
    (Array(report.result("bounded")), new WeakReference(model))

  private def exerciseResult(held: Array[Result], owner: WeakReference[Model]): Unit =
    collect("other-checked", batch(true))
    val result = held(0)
    val model = Option(owner.get).getOrElse(fail("retained Result lost its owner"))
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(result.path.steps.map(s => NativeCheck.shown(s.action)), Vector("alias", "alias"))
    assertEquals(
      result.path.states,
      Vector(model.init, model.init.copy(value = 1), model.init.copy(value = 2))
    )
    assertEquals(Checker.verifyAll(model, budget = 1).result("bounded").verdict, Verdict.unknown)
    assertEquals(NativeCheck.claimName(model.monitors.watched, model), "watched")
    Reference.reachabilityFence(result)

  private def retainState(held: Array[Result]): Array[State] =
    val state = held(0).path.states.head match
      case value: State => value
      case other        => fail(s"unexpected result state $other")
    held(0) = null // scalafix:ok DisableSyntax.null
    Array(state)

  private def exerciseState(held: Array[State], owner: WeakReference[Model]): Unit =
    collect("state-other-checked", batch(true))
    val model = Option(owner.get).getOrElse(fail("retained State lost checked owner"))
    assert(held(0).eq(model.init))
    assertEquals(NativeCheck.actionName(model.user.advance.decl), "alias")
    assertEquals(Checker.verifyAll(model, budget = 3).result("arrives").verdict, Verdict.verified)
    Reference.reachabilityFence(held)

  test("a retained Result preserves names and states until its release"):
    val (held, owner) = retained()
    exerciseResult(held, owner)
    val state = retainState(held)
    exerciseState(state, owner)
    state(0) = null // scalafix:ok DisableSyntax.null
    collect("released-result-and-state", Vector(owner))

  private def catalogue(): (Array[ActionDecl], Array[AnyRef], Vector[WeakReference[?]]) =
    val model = new DiscoveryCatalogueFixture.Model
    val wire = channel[Boolean](1, Order.fifo, Loss.lossy)
    val missing = hole
    val watch = monitor[State, Boolean, Nothing, Boolean](false)((seen, _, _) =>
      seen || model.value == 1
    )(identity)
    val offered = model.user.advance
    val entries = DiscoveryCatalogueFixture.catalogue
    entries.offered = offered
    entries.wire = wire
    entries.missing = missing
    entries.watched = watch
    val _ = NativeCheck.explore(model, model.initialStates)
    assertEquals(NativeCheck.actionName(offered.decl), "offered")
    assertEquals(NativeCheck.actionName(wire.deliver.decl), "wire.deliver")
    assertEquals(NativeCheck.holeName(missing), "missing")
    assertEquals(NativeCheck.claimName(watch, model), "watched")
    entries.offered = null // scalafix:ok DisableSyntax.null
    entries.wire = null // scalafix:ok DisableSyntax.null
    entries.missing = null // scalafix:ok DisableSyntax.null
    entries.watched = null // scalafix:ok DisableSyntax.null
    (
      Array(wire.deliver.decl),
      Array(missing, watch, offered.decl),
      Vector(model, offered.decl, missing, watch, wire, wire.deliver.decl, wire.lose.decl)
        .map(value => new WeakReference(value))
    )

  private def exerciseAction(held: Array[ActionDecl], channel: WeakReference[?]): Unit =
    collect("catalogue-unrelated", batch(false))
    assert(channel.get != null, "retained delivery action lost its channel")
    assertEquals(NativeCheck.actionName(held(0)), "wire.deliver")
    val wire = channel.get match
      case value: Channel[?] =>
        value.asInstanceOf[Channel[Boolean]] // scalafix:ok DisableSyntax.asInstanceOf
      case _ => fail("retained action lost channel")
    object mailbox extends Machine[Boolean, Nothing]:
      var network = wire.empty.send(true)
      var received = false

      object transitions extends Transitions:
        on(wire.deliver): message =>
          effects.receive(message)
        on(wire.lose): message =>
          effects.lose(message)

      object effects:
        def receive(message: Boolean): Unit = received = message
        def lose(message: Boolean): Unit = received = !message
    val effects = NativeCheck.machineEffects(mailbox)
    for (decl, received) <- Vector(wire.deliver.decl -> true, wire.lose.decl -> false) do
      val run = effects.find((action, _) => action eq decl).get._2
      val steps = run(mailbox.init, Class(decl, List(true)))
      assertEquals(steps.size, 1)
      assertEquals(mailbox.at(steps.head.state)(mailbox.network.isEmpty), true)
      assertEquals(mailbox.at(steps.head.state)(mailbox.received), received)
      assertEquals(run(steps.head.state, Class(decl, List(true))), Nil)
    Reference.reachabilityFence(held)

  test("scanned channel backreferences and callback claims release after live actions"):
    val (held, named, refs) = catalogue()
    exerciseAction(held, refs(4))
    assertEquals(DeclarationLifetime.name(named(0)), Some("missing"))
    assertEquals(DeclarationLifetime.name(named(1)), Some("watched"))
    assertEquals(DeclarationLifetime.name(named(2)), Some("offered"))
    named.indices.foreach(i => named(i) = null) // scalafix:ok DisableSyntax.null
    collect("catalogue-owner-hole-claim", refs.take(4))
    held(0) = null // scalafix:ok DisableSyntax.null
    collect("catalogue-channel", refs.drop(4))

  test("discovery names keep equal declarations distinct and retain their first spelling"):
    final case class Equal(value: Int)
    val first = Equal(1)
    val second = Equal(1)
    assertEquals(first, second)
    assert(!(first eq second))
    DeclarationLifetime.spell(first, "first")
    DeclarationLifetime.spell(second, "second")
    DeclarationLifetime.spell(first, "alias")
    def discarded(): WeakReference[Equal] =
      val value = Equal(1)
      DeclarationLifetime.spell(value, "discarded")
      new WeakReference(value)
    collect("identity-equal-discarded", Vector(discarded()))
    assertEquals(DeclarationLifetime.name(first), Some("first"))
    assertEquals(DeclarationLifetime.name(second), Some("second"))
    Reference.reachabilityFence(first)
    Reference.reachabilityFence(second)
