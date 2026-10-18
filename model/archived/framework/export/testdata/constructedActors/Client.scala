package framework.actorclient

import java.util.Base64
import framework.dsl.*
import fixture.actorhelpers.Helpers
import io.temporal.server.api.umpire.v1 as ir

object Client:
  val direct = action(Helpers.`Named-helper`)
  val forwarded = action(Helpers.actor("helper"))
  val empty = action(Helpers.actor(""))
  val forwardedFair = assume("forwarded").fair(forwarded)
  def main(args: Array[String]): Unit =
    println(
      "direct:" + Base64.getEncoder.encodeToString(ir.Model(actions = Seq(direct.toIR)).toByteArray)
    )
    assert(direct.toIR.actor == "named-helper")
    assert(Helpers.getters == 0)
    assert(Helpers.`Named-helper`.name == "public-helper")
    assert(Helpers.getters == 1)
    assert(forwarded.toIR.actor == "helper")
    assert(empty.toIR.actor == "")
    assert(Helpers.calls == 2)
    println("helper-controls:passed")

def unchanged(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Bindings(Client.direct ~> unchanged)
val progress = Subject.leadsTo("direct")(s => !s, s => s, 1)
val helperProgress = Subject.leadsTo("helper")(s => !s, s => s, 1, Client.forwardedFair)
