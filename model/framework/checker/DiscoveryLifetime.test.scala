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
  import DiscoveryLifetimeFixture.*

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
    assertEquals(result.path.states.map(_.asInstanceOf[State].selectDynamic("value")), Vector(0, 1, 2))
    assertEquals(Checker.verifyAll(model, budget = 1).result("bounded").verdict, Verdict.unknown)
    assertEquals(NativeCheck.claimName(model.monitors.watched, model), "watched")
    Reference.reachabilityFence(result)

  test("a retained Result preserves names and states until its release"):
    val (held, owner) = retained()
    exerciseResult(held, owner)
    held(0) = null // scalafix:ok DisableSyntax.null
    collect("released-result", Vector(owner))

  private def catalogue(): (Array[ActionDecl], Vector[WeakReference[?]]) =
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
      Vector(model, offered.decl, missing, watch, wire, wire.deliver.decl, wire.lose.decl)
        .map(value => new WeakReference(value))
    )

  private def exerciseAction(held: Array[ActionDecl], channel: WeakReference[?]): Unit =
    collect("catalogue-unrelated", batch(false))
    assert(channel.get != null, "retained delivery action lost its channel")
    assertEquals(NativeCheck.actionName(held(0)), "wire.deliver")
    Reference.reachabilityFence(held)

  test("scanned channel backreferences and callback claims release after live actions"):
    val (held, refs) = catalogue()
    exerciseAction(held, refs(4))
    collect("catalogue-owner-hole-claim", refs.take(4))
    held(0) = null // scalafix:ok DisableSyntax.null
    collect("catalogue-channel", refs.drop(4))
