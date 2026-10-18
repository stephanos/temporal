package framework.references

import framework.dsl.*
import framework.dsl.Finite.given
import helper.references.Helpers

object Client:
  final case class Cell(flag: Boolean)
  final case class Pair(left: Cell, right: Cell)
  val send = action("send", Actor("caller"))
  val receive = action("receive", Actor("caller"))
  var calls = 0
  def selected: Action[EmptyTuple] =
    calls += 1
    send
  object Tank extends Machine[Cell, Boolean, Nothing]:
    val init = Cell(false)
    def end(s: Cell) = s.flag
    object rules
        extends Bindings(
          send ~> ((s: Cell) => List(Step(true, s))),
          receive ~> ((s: Cell) => List(Step(false, s)))
        )
  object Pipeline extends Composition[Pair](_.left -> Tank, _.right -> Tank):
    object syncs extends Syncs:
      sync(_.left -> send, _.right -> receive)
  object DerivedPipeline extends Composition(Helpers.member(Pipeline, _.right -> Tank))
  def main(args: Array[String]): Unit =
    val alias = Helpers.forward(Helpers.synced(Pipeline, _.left -> selected))
    assert(calls == 0)
    assert(alias.toIR("client").getSync == "send")
    assert(alias.toIR("client").getSync == "send")
    assert(calls == 1)
    assert(Helpers.own(Pipeline, _.right, send).toIR("client").member == "right")
    assert(Helpers.curried(Pipeline)(_.right, send).toIR("client").member == "right")
    assert(Helpers.twoHop(Pipeline, _.right, send).toIR("client").member == "right")
    assert(Helpers.synced(DerivedPipeline, _.left -> send()).toIR("client").selection.isSyncClass)
    val ordinary = Helpers.unrelated[Pair, Cell](_.left)
    assert(MemberPath.field(ordinary).isEmpty)
    val state = Pair(Cell(true), Cell(false))
    assert(ordinary(state) == Cell(true))
    assert(List(state).map(_.left) == List(Cell(true)))
