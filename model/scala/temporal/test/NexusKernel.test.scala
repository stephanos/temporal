package temporal
package nexuscaller
// The Stainless lemmas quantify over kernel.Actions, one dispatch over every action; the
// machines bind the per-action step functions directly. This test checks the two agree on every
// state and every class, so what Stainless proves is about the machines the pins and Cases use.

import kernel.{Actions, ProductAction, ProtocolAction}
import umpire.*

class NexusKernel extends munit.FunSuite:
  def protocolAction(c: Class): ProtocolAction = (c.decl.name, c.values) match
    case ("schedule", List(a: Timeout, b: Timeout, d: Timeout)) => ProtocolAction.schedule(a, b, d)
    case ("handlerReply", List(r: Reply))                       => ProtocolAction.handlerReply(r)
    case ("complete", List(r: Resolution))                      => ProtocolAction.complete(r)
    case ("transportFault", Nil)                                => ProtocolAction.transportFault
    case ("workerStop", Nil)                                    => ProtocolAction.workerStop
    case ("backoff", Nil)                                       => ProtocolAction.backoff
    case ("scheduleToClose", Nil)                               => ProtocolAction.scheduleToClose
    case ("scheduleToStart", Nil)                               => ProtocolAction.scheduleToStart
    case ("startToClose", Nil)                                  => ProtocolAction.startToClose
    case other                                                  => fail(s"no protocol action for $other")

  def productAction(c: Class): ProductAction = (c.decl.name, c.values) match
    case ("handlerReply", List(r: Reply))  => ProductAction.handlerReply(r)
    case ("complete", List(r: Resolution)) => ProductAction.complete(r)
    case ("transportFault", Nil)           => ProductAction.transportFault
    case ("workerStop", Nil)               => ProductAction.workerStop
    case ("timeout", Nil)                  => ProductAction.timeout
    case other                             => fail(s"no product action for $other")

  def steps(t: Table, state: String, action: String): List[Any] =
    t.row(Table.rowKeyOf(state, action)).fold(Nil)(_.results.toList.map(_.step))

  test("the dispatch gives every protocol row, and nothing where the table has none") {
    val t = nexusProtocol.table.fold(e => fail(e.toString), identity)
    for s <- t.states; a <- t.actions do
      val state = t.stateValueOf(s).get.asInstanceOf[ProtocolState]
      assertEquals(Actions.protocolStep(state, protocolAction(t.classOf(a).get)), steps(t, s, a), s"$s $a")
  }

  test("the dispatch gives every product row, and nothing where the table has none") {
    val t = nexusProduct.table.fold(e => fail(e.toString), identity)
    for s <- t.states; a <- t.actions do
      val state = t.stateValueOf(s).get.asInstanceOf[ProductState]
      assertEquals(Actions.productStep(state, productAction(t.classOf(a).get)), steps(t, s, a), s"$s $a")
  }
