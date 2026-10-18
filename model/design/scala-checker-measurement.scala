package framework.checker

import framework.dsl.{Checker as _, *}
import framework.dsl.Finite.given
import java.lang.management.{ManagementFactory, MemoryType}
import java.lang.ref.WeakReference
import java.util.concurrent.atomic.AtomicBoolean
import scala.jdk.CollectionConverters.*
import samples.{Bank, Kettle, ReliableBroadcast, TwoLayeredCache, TwoPhaseCommit, Voting}

object ScalaCheckerMeasurement:
  enum Acceptor:
    case a1, a2, a3, a4, a5, a6, a7
  enum Value:
    case v1, v2
  enum Address:
    case king, donkeykong, peaches, mario
  enum Manager:
    case rm1, rm2, rm3

  private def voting(n: Int): Voting.Model[Acceptor, Value] =
    given Finite[Acceptor] = Finite.of(Acceptor.values.take(n).toSeq*)
    new Voting.Model[Acceptor, Value](
      Value.values.toSet,
      Acceptor.values.take(n).toSeq.combinations(n / 2 + 1).map(_.toSet).toSet
    )

  private def withModel[A](name: String)(
      f: (NativeModel[State, ?, ?], Int => Report) => A
  ): A = name match
    case "kettle" =>
      f(Kettle.Model, budget => Checker.verifyAll(Kettle.Model, budget = budget))
    case "broadcast" =>
      f(
        ReliableBroadcast.Model,
        budget => Checker.verifyAll(ReliableBroadcast.Model, budget = budget)
      )
    case "cache" =>
      f(TwoLayeredCache.Model, budget => Checker.verifyAll(TwoLayeredCache.Model, budget = budget))
    case "bank" =>
      val bank = new Bank.Model[Address](Address.king, 2)
      f(bank, budget => Checker.verifyAll(bank, budget = budget))
    case "twopc" =>
      val twoPhaseCommit = new TwoPhaseCommit.Model[Manager](Manager.values.toSet)
      f(twoPhaseCommit, budget => Checker.verifyAll(twoPhaseCommit, budget = budget))
    case "voting3" =>
      val votingThree = voting(3)
      f(votingThree, budget => Checker.verifyAll(votingThree, budget = budget))
    case "voting5" =>
      val votingFive = voting(5)
      f(votingFive, budget => Checker.verifyAll(votingFive, budget = budget))
    case "voting7" =>
      val votingSeven = voting(7)
      f(votingSeven, budget => Checker.verifyAll(votingSeven, budget = budget))
    case other => throw IllegalArgumentException(s"unknown graph $other")

  private def printReport(value: Report): Unit =
    value.results.foreach: result =>
      println(
        s"RESULT\t${result.subject}\t${result.verdict}\t${result.complete}\t${result.depth}" +
          s"\t${result.longest}\t${result.detail.mkString(" | ")}"
      )
    value.warnings.foreach(w => println(s"WARNING\t$w"))

  private def statistics[S, O, F](m: NativeModel[S, O, F], budget: Int): Unit =
    val roots = m.initialStates
    val graph = NativeCheck.explore(m, roots, budget = budget)
    var steps = 0L
    graph.walk(onEdge = _ =>
      steps += 1; false)
    println(
      s"GRAPH\t${m.name}\t${roots.size}\t$budget\t${graph.states.size}\t$steps" +
        s"\t${graph.depth}\t${graph.complete}\t${graph.cut}\t${graph.unknown.size}"
    )
    java.lang.ref.Reference.reachabilityFence(graph)

  private val heapPools = ManagementFactory.getMemoryPoolMXBeans.asScala
    .filter(_.getType eq MemoryType.HEAP)
    .map(_.getName)
    .toSet
  private def collector = ManagementFactory.getGarbageCollectorMXBeans.asScala.collectFirst:
    case gc: com.sun.management.GarbageCollectorMXBean if gc.getName == "MarkSweepCompact" => gc

  private def fullGc(): (Long, Long, Long, Long) =
    val gc = collector.getOrElse(throw IllegalStateException("requires -XX:+UseSerialGC"))
    val before = Option(gc.getLastGcInfo).fold(0L)(_.getId)
    System.gc()
    val info = Option(gc.getLastGcInfo).getOrElse(throw IllegalStateException("no full GC"))
    require(info.getId > before, "explicit full GC was not observed")
    val bytes = info.getMemoryUsageAfterGc.asScala.iterator
      .filter((name, _) => heapPools(name))
      .map(_._2.getUsed)
      .sum
    (info.getId, bytes, info.getStartTime, info.getEndTime)

  private def heap[S, O, F](m: NativeModel[S, O, F], check: () => Report): Unit =
    val (baselineId, baseline, baselineStart, baselineEnd) = fullGc()
    println(s"HEAP_BASELINE\t$baselineId\t$baseline\t$baselineStart\t$baselineEnd")
    val active = new AtomicBoolean(false)
    val finished = new AtomicBoolean(false)
    val observations = scala.collection.mutable.ArrayBuffer.empty[Long]
    var rejected = 0
    val errors = scala.collection.mutable.ArrayBuffer.empty[Throwable]
    val worker = Thread.currentThread()
    def inside: Boolean = worker.getStackTrace.exists(frame =>
      frame.getClassName == "framework.checker.NativeCheck$" && frame.getMethodName == "verifyAll"
    )
    def observe(): Unit =
      try
        while !finished.get() do
          if active.get() && inside then
            val (id, bytes, start, end) = fullGc()
            val accepted = active.get() && inside
            val marker = if accepted then "LIVE_SAMPLE" else "LIVE_REJECTED_SAMPLE"
            println(
              s"$marker\t${ProcessHandle.current().pid()}:1\t$id\t$bytes\t$start\t$end\t$accepted"
            )
            if accepted then observations += bytes: Unit
            else rejected += 1
          Thread.sleep(10)
      catch
        case error: Throwable =>
          val _ = errors.addOne(error)
    val sampler = new Thread(() => observe(), "measurement-full-gc")
    sampler.start()
    active.set(true)
    val value = try check()
    finally
      active.set(false)
      finished.set(true)
      sampler.join()
    errors.headOption.foreach(throw _)
    println(s"LIVE_REJECTED\t$rejected\tinvocation-ended-or-stack-left-verifyAll")
    require(observations.nonEmpty, "no full GC wholly inside verifyAll; no live heap measured")
    println(s"LIVE_MAX_SAMPLE\t${observations.max}\t${observations.size}")
    printReport(value)
    java.lang.ref.Reference.reachabilityFence(m)

  private def stored(owner: AnyRef, name: String): AnyRef =
    val candidates = owner.getClass.getDeclaredFields.filter(field =>
      field.getName == name || field.getName.endsWith("$$" + name)
    )
    require(candidates.length == 1, s"cannot resolve ${owner.getClass.getName}.$name")
    val field = candidates.head
    require(field.trySetAccessible(), s"cannot read $field")
    println(
      s"CACHE_FIELD\t${field.getDeclaringClass.getName}\t${field.getName}\t${field.getType.getName}"
    )
    field.get(owner).asInstanceOf[AnyRef]

  private def retention(count: Int): Unit =
    require(count > 0)
    def batch(kind: String): Vector[WeakReference[AnyRef]] = Vector.fill(count):
      val model = voting(3)
      kind match
        case "constructed" => ()
        case "initialized" => val _ = model.init
        case "checked"     =>
          val value = Checker.verifyAll(model, budget = 1)
          require(value.results.nonEmpty)
        case other => throw IllegalArgumentException(other)
      new WeakReference[AnyRef](model)

    val lives = stored(VarRuntime, "lives").asInstanceOf[java.util.Map[Vars, ?]]
    val spelled = stored(NativeCheck, "spelled")
      .asInstanceOf[java.util.Map[ActionDecl, String]]
    val channels = stored(NativeCheck, "channels")
      .asInstanceOf[java.util.Map[ActionDecl, ?]]
    val claims = stored(NativeCheck, "claims").asInstanceOf[java.util.Map[?, ?]]
    val holes = stored(NativeCheck, "holes").asInstanceOf[java.util.Map[?, ?]]
    val declared = stored(ClaimEvaluation, "declared").asInstanceOf[java.util.Map[?, ?]]

    def actorPaths(map: java.util.Map[ActionDecl, ?], target: AnyRef): Vector[String] =
      map.keySet.asScala.toVector.flatMap: decl =>
        val actor = decl.actor
        actor.getClass.getDeclaredFields.toVector
          .filterNot(field =>
            java.lang.reflect.Modifier.isStatic(field.getModifiers) || field.getType.isPrimitive
          )
          .flatMap: field =>
            require(field.trySetAccessible(), s"cannot inspect $field")
            Option.when(field.get(actor).asInstanceOf[AnyRef] eq target)(
              s"${actor.getClass.getName}.${field.getName}"
            )

    def attribute(kind: String, refs: Vector[WeakReference[AnyRef]]): Unit =
      var alive = 0
      var attributed = 0
      refs.zipWithIndex.foreach: (ref, index) =>
        Option(ref.get()).foreach: target =>
          alive += 1
          val runtime = lives.keySet.asScala.count(vars => vars.machine eq target)
          val names = NativeCheck.synchronized(actorPaths(spelled, target))
          val channelKeys = NativeCheck.synchronized(actorPaths(channels, target))
          if runtime > 0 || names.nonEmpty || channelKeys.nonEmpty then attributed += 1
          println(
            s"OWNER_ROOT\t$kind\t$index\t$runtime\t${names.size}\t${channelKeys.size}" +
              s"\t${names.mkString(",")}\t${channelKeys.mkString(",")}"
          )
      println(s"RETENTION\t$kind\t${refs.size}\t$alive\t$attributed\t${alive - attributed}")

    val held = scala.collection.mutable.ArrayBuffer.empty[Vector[WeakReference[AnyRef]]]
    Vector("constructed", "initialized", "checked").foreach: kind =>
      val refs = batch(kind)
      held += refs: Unit
      val (id, bytes, start, end) = fullGc()
      println(s"RETENTION_GC\t$kind\t$id\t$bytes\t$start\t$end")
      attribute(kind, refs)
      println(
        s"CACHE_SIZES\t$kind\t${lives.size}\t${spelled.size}\t${channels.size}" +
          s"\t${claims.size}\t${holes.size}\t${declared.size}"
      )
      println(
        s"IDLE_FRAMES\t$kind\t${VarRuntime.current.isEmpty}\t${ClaimEvaluation.current.isEmpty}"
      )
    java.lang.ref.Reference.reachabilityFence(held)

  def main(args: Array[String]): Unit =
    require(args.length >= 2, "graph mode [budget] [repetitions]")
    println(
      s"ENV\t${System.getProperty("java.version")}\t${System.getProperty("java.vm.name")}" +
        s"\t${Runtime.getRuntime.maxMemory()}\t${ManagementFactory.getRuntimeMXBean.getInputArguments}"
    )
    if args(0) == "retention" then retention(args(1).toInt)
    else
      val budget =
        args.lift(2).fold(if args(0) == "cache" then 4096 else Checker.defaultBudget)(_.toInt)
      val repetitions = args.lift(3).fold(3)(_.toInt)
      require(repetitions >= 1)
      withModel(args(0)): (model, verify) =>
        println(
          s"OWNER\t${args(0)}\t${model.getClass.getName}\t${model.initialStates.size}\t$budget"
        )
        args(1) match
          case "stats" => statistics(model, budget)
          case "heap"  => heap(model, () => verify(budget))
          case "time"  =>
            (0 until repetitions).foreach: iteration =>
              val start = System.nanoTime()
              val value = verify(budget)
              val nanos = System.nanoTime() - start
              println(s"TIME\t${args(0)}\t$iteration\t$nanos")
              printReport(value)
          case other => throw IllegalArgumentException(s"unknown mode $other")
