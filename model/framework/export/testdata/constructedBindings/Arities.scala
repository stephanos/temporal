package framework.bindingarity

import framework.dsl.*

enum Reply:
  case ok

object Evaluation:
  var catalogs = 0
  var starts = 0
  var refinements = 0

given Finite[Boolean] with
  def values: IndexedSeq[Boolean] =
    Evaluation.catalogs += 1
    Vector(false, true)

val action0 = action(Actor("user"))
val action1 = action(Actor("user")).input[Boolean]("i1")
val action2 = action(Actor("user")).input[Boolean]("i1").input[Boolean]("i2")
val action3 = action(Actor("user")).input[Boolean]("i1").input[Boolean]("i2").input[Boolean]("i3")
val action4 = action(Actor("user"))
  .input[Boolean]("i1")
  .input[Boolean]("i2")
  .input[Boolean]("i3")
  .input[Boolean]("i4")
val action6 = action(Actor("user"))
  .input[Boolean]("i1")
  .input[Boolean]("i2")
  .input[Boolean]("i3")
  .input[Boolean]("i4")
  .input[Boolean]("i5")
  .input[Boolean]("i6")

abstract class AritySupport extends Machine[Boolean, Reply, Nothing]:
  def init: Boolean =
    Evaluation.starts += 1
    false
object ArityMachine extends AritySupport:
  def end(s: Boolean): Boolean = s
  override private[framework] def refinementEvidence: Option[Refinement[Boolean, ?]] =
    Evaluation.refinements += 1
    None
  object rules
      extends Bindings(
        action0 ~> ((s: Boolean) => List(Step(Reply.ok, s))),
        action1 ~> ((s: Boolean, i1: Boolean) => List(Step(Reply.ok, s && i1))),
        action2 ~> ((s: Boolean, i1: Boolean, i2: Boolean) => List(Step(Reply.ok, s && i1 && i2))),
        action3 ~> ((s: Boolean, i1: Boolean, i2: Boolean, i3: Boolean) =>
          List(Step(Reply.ok, s && i1 && i2 && i3))
        ),
        action4 ~> ((s: Boolean, i1: Boolean, i2: Boolean, i3: Boolean, i4: Boolean) =>
          List(Step(Reply.ok, s && i1 && i2 && i3 && i4))
        ),
        action6 ~> (
          (
              s: Boolean,
              i1: Boolean,
              i2: Boolean,
              i3: Boolean,
              i4: Boolean,
              i5: Boolean,
              i6: Boolean
          ) => List(Step(Reply.ok, s && i1 && i2 && i3 && i4 && i5 && i6))
        )
      )

object ArityNative:
  def main(args: Array[String]): Unit =
    val first = CoreBindings.collect(ArityMachine)
    assert(CoreBindings.collect(ArityMachine).equals(first))
    assert(first.steps.size == 6)
    assert(first.functions.map(_.params.size).sorted == Vector(1, 2, 3, 4, 5, 7))
    assert(Evaluation.catalogs == 0 && Evaluation.starts == 0 && Evaluation.refinements == 0)
    val derived = ArityMachine.restrict(action0, action6)
    assert(Evaluation.starts == 1 && Evaluation.refinements == 0)
    CoreBindings.collect(derived): Unit
    assert(Evaluation.catalogs == 0 && Evaluation.starts == 1 && Evaluation.refinements == 0)
    for binding <- ArityMachine.bindings do
      assert(binding.function.eq(binding.captured.get.callable))
      assert(
        !effectOf[Boolean, Reply, Nothing](binding.decl, binding.function)(
          false,
          List.fill(binding.decl.domains.size)(true)
        ).head.state
      )
    println(
      "arity-evaluation:0,1,2,3,4,6; construction-start=1; conversion-catalogs=0; conversion-refinement=0"
    )
