package framework.dsl

import propertygroups.*

object NativeChecks:
  def check(
      claims: List[Property[State]],
      stable: Property[State],
      progress: List[Progress[State]],
      owner: Machine[State, Outcome, Nothing],
      lonely: Machine[State, Outcome, Nothing],
      bundleOnly: Machine[State, Outcome, Nothing],
      asked: Query,
      watched: Monitor[State, Outcome, Nothing, Boolean]
  ): Unit =
    val states = Vector(State.open, State.closed)
    assert(Finite[State].values.toVector == states)
    assert(claims.map(_.name) == List("closed", "aliased", "bundledOpen", "bundledClosed"))
    assert(claims.head.decl.when.contains(tester.advance))
    assert(claims.tail.forall(_.decl.when.isEmpty))
    assert(claims.forall(_.decl.holds2.isEmpty))
    assert(stable.decl.holds.isEmpty && stable.decl.holds2.nonEmpty && stable.decl.when.isEmpty)
    val expected =
      List(Vector(false, true), Vector(false, true), Vector(true, false), Vector(false, true))
    for (claim, truth) <- claims.zip(expected) do
      assert(claim.decl.machine eq owner)
      val predicate = claim.decl.holds.get.asInstanceOf[Step[State, Outcome, Nothing] => Boolean]
      assert(states.map(s => predicate(Step(Outcome.done, s))) == truth)
    val transition =
      stable.decl.holds2.get.asInstanceOf[(State, Step[State, Outcome, Nothing]) => Boolean]
    assert(
      states.flatMap(before =>
        states.map(after => transition(before, Step(Outcome.done, after)))
      ) ==
        Vector(true, false, false, true)
    )
    assert(
      progress
        .map(_.name) == List("eventuallyClosed", "factoryProgress", "onlyProgress", "bundledOnly")
    )
    assert(progress.map(_.within) == List(1, 2, 3, 4))
    for p <- progress do
      assert(states.map(p.from) == Vector(true, false))
      assert(states.map(p.to) == Vector(false, true))
      assert(p.assumptions.map(_.name) == List("fair"))
    assert(progress.take(2).forall(_.machine eq owner))
    assert(progress(2).machine eq lonely)
    assert(progress.last.machine eq bundleOnly)
    assert(asked.name == "finished" && asked.form == QueryForm.verify)
    assert(asked.property.name == "closed" && (asked.property.machine eq owner))
    assert(asked.scenario.name == "any" && (asked.scenario.machine eq owner))
    assert(asked.scenario.free && asked.scenario.start.isEmpty && asked.scenario.actions.isEmpty)
    assert(asked.limits == Limits("small", 1, 1, 8))
    assert(asked.total.isEmpty)
    assert(watched eq seal)
    assert(!watched.initial && !watched.violated(false) && watched.violated(true))
    val monitorRows = Vector(false, true).flatMap(m =>
      states.flatMap(before =>
        states.map(after => watched.next(m, before, Step(Outcome.done, after)))
      )
    )
    assert(monitorRows == Vector(false, true, false, true, false, true, false, true))
    assert(watched.readAt.isEmpty && !watched.atEnds)
    val files = IrFile.discover("propertygroups")
    assert(
      files.map(_.name) == Vector("bundleonly", "lonely", "specimen-workflow"),
      files.map(_.name).toString
    )
    assert(
      files.map(f => f.name -> f.construct()) ==
        Vector(
          "bundleonly" -> Set("bundleOnly"),
          "lonely" -> Set("lonely"),
          "specimen-workflow" -> Set("subject", "joined")
        )
    )
