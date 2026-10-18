package framework.checker

import framework.dsl.*
import framework.dsl.Finite.given
import java.lang.management.ManagementFactory
import java.lang.ref.{Reference, WeakReference}
import java.util.concurrent.{Callable, CountDownLatch, Executors, TimeUnit}
import scala.jdk.CollectionConverters.*

object RuntimeLifetimeFixture:
  given Ok[Boolean] = Ok(true)

  class Model(seed: Int) extends Machine[Boolean, Nothing]:
    var count = seed

    object transitions extends Transitions

class RuntimeLifetimeTest extends munit.FunSuite:
  import RuntimeLifetimeFixture.*

  private def collections: Long =
    ManagementFactory.getGarbageCollectorMXBeans.asScala.map(_.getCollectionCount.max(0L)).sum

  private def collect(label: String, refs: Vector[WeakReference[?]]): Unit =
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

  private def batch(initialized: Boolean): Vector[WeakReference[Model]] = Vector.fill(16):
    val model = new Model(7)
    if initialized then
      val _ = model.init
    new WeakReference(model)

  test("initialized runtime owners release with constructed-only collection controls"):
    collect("constructed", batch(false))
    collect("initialized", batch(true))

  private def retained(): (Array[State], WeakReference[Model], WeakReference[State]) =
    val model = new Model(7)
    val initial = model.init
    val state = initial.copy(count = 19)
    model.count = 91
    (Array(state), new WeakReference(model), new WeakReference(initial))

  private def exerciseRetained(
      held: Array[State],
      owner: WeakReference[Model],
      initial: WeakReference[State]
  ): Unit =
    collect("unrelated", batch(false))
    val state = held(0)
    val model = Option(owner.get).getOrElse(fail("retained State lost its owner"))
    val start = Option(initial.get).getOrElse(fail("live owner lost its initial snapshot"))
    assert(model.init.eq(start))
    assertEquals(start.selectDynamic("count"), 7: Any)
    assertEquals(model.at(state)(model.count), 19)
    assertEquals(model.count, 7)
    Reference.reachabilityFence(state)

  test("a retained State preserves its owner and first snapshot until the State is released"):
    val (held, owner, initial) = retained()
    exerciseRetained(held, owner, initial)
    held(0) = null // scalafix:ok DisableSyntax.null
    collect("released-state", Vector(owner, initial))

  test("same-owner concurrent evaluations share one initial snapshot and serialize restoration"):
    val model = new Model(7)
    val ready = new CountDownLatch(2)
    val start = new CountDownLatch(1)
    val executor = Executors.newFixedThreadPool(2)
    try
      val tasks = Vector(19, 29).map: count =>
        executor.submit(new Callable[State]:
          def call(): State =
            ready.countDown()
            val started = start.await(5, TimeUnit.SECONDS)
            assert(started)
            val initial = model.init
            val state = initial.copy(count = count)
            for _ <- 0 until 100 do
              model.at(state):
                assertEquals(model.count, count)
                Thread.`yield`()
                assertEquals(model.at(initial)(model.count), 7)
                assertEquals(model.count, count)
            initial
        )
      val prepared = ready.await(5, TimeUnit.SECONDS)
      assert(prepared)
      start.countDown()
      val snapshots = tasks.map(_.get(10, TimeUnit.SECONDS))
      assert(snapshots(0).eq(snapshots(1)))
      assertEquals(model.count, 7)
    finally
      start.countDown()
      executor.shutdownNow(): Unit
      val terminated = executor.awaitTermination(5, TimeUnit.SECONDS)
      assert(terminated)
