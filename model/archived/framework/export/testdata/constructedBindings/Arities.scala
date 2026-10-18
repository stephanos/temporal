package framework.bindingarity

import framework.dsl.*

enum Reply:
  case ok

object Evaluation:
  var catalogs = 0
  var starts = 0
  var refinements = 0

// A counting witness. Under the reify plugin a concrete `Finite[Boolean]` subclass, the given's
// anonymous class included, does not compile (the plugin retains a schema on its parent call), so
// it is a small generic named subclass.
final class Counted[T](catalog: Vector[T])(using Data.Structural[T]) extends Finite[T]:
  def values: IndexedSeq[T] =
    Evaluation.catalogs += 1
    catalog
given Finite[Boolean] = Counted(Vector(false, true))

// Native actions declare one complete record for several inputs (fn-165): each record keeps the
// retired input chain's names, order and types, one IR input a field.
final case class In2(i1: Boolean, i2: Boolean)
final case class In3(i1: Boolean, i2: Boolean, i3: Boolean)
final case class In4(i1: Boolean, i2: Boolean, i3: Boolean, i4: Boolean)
final case class In6(i1: Boolean, i2: Boolean, i3: Boolean, i4: Boolean, i5: Boolean, i6: Boolean)

val action0 = action(Actor("user"))
val action1 = action(Actor("user")).input[Boolean]("i1")
val action2 = action(Actor("user")).inputs[In2]
val action3 = action(Actor("user")).inputs[In3]
val action4 = action(Actor("user")).inputs[In4]
val action6 = action(Actor("user")).inputs[In6]

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
        action2 ~> ((s: Boolean, in: In2) => List(Step(Reply.ok, s && in.i1 && in.i2))),
        action3 ~> ((s: Boolean, in: In3) => List(Step(Reply.ok, s && in.i1 && in.i2 && in.i3))),
        action4 ~> ((s: Boolean, in: In4) =>
          List(Step(Reply.ok, s && in.i1 && in.i2 && in.i3 && in.i4))
        ),
        action6 ~> ((s: Boolean, in: In6) =>
          List(Step(Reply.ok, s && in.i1 && in.i2 && in.i3 && in.i4 && in.i5 && in.i6))
        )
      )

object ArityNative:
  def main(args: Array[String]): Unit =
    val first = CoreBindings.collect(ArityMachine)
    assert(CoreBindings.collect(ArityMachine).equals(first))
    assert(first.steps.size == 6)
    // A native binding takes `(s, in)`: the state, then the one declared input, a scalar or the
    // whole record whose fields are the retired chain's inputs in order.
    assert(first.functions.map(_.params.size).sorted == Vector(1, 2, 2, 2, 2, 2))
    val records = first.functions.flatMap(_.params.drop(1)).map(_.getType.getNamed).filter(_.nonEmpty)
    assert(records.sorted == Vector("In2", "In3", "In4", "In6").map("framework.bindingarity." + _))
    assert(
      first.types
        .filter(t => records.contains(t.name))
        .map(t => t.name.stripPrefix("framework.bindingarity.") -> t.getRecord.fields.map(_.name))
        .sortBy(_._1) == Vector(
        "In2" -> Seq("i1", "i2"),
        "In3" -> Seq("i1", "i2", "i3"),
        "In4" -> Seq("i1", "i2", "i3", "i4"),
        "In6" -> Seq("i1", "i2", "i3", "i4", "i5", "i6")
      )
    )
    assert(Evaluation.catalogs == 0 && Evaluation.starts == 0 && Evaluation.refinements == 0)
    // Derivation construction no longer inspects its source's starts (fn-123.10, by-name `Built`
    // starts), so neither construction nor conversion evaluates `init`.
    val derived = ArityMachine.restrict(action0, action6)
    assert(Evaluation.starts == 0 && Evaluation.refinements == 0)
    CoreBindings.collect(derived): Unit
    assert(Evaluation.catalogs == 0 && Evaluation.starts == 0 && Evaluation.refinements == 0)
    for binding <- ArityMachine.bindings do
      assert(binding.function.eq(binding.captured.get.callable))
      assert(
        !effectOf[Boolean, Reply, Nothing](binding.decl, binding.function)(
          false,
          List.fill(binding.decl.domains.size)(true)
        ).head.state
      )
    println(
      "arity-evaluation:0,1,2,3,4,6; construction-start=0; conversion-catalogs=0; conversion-refinement=0"
    )
