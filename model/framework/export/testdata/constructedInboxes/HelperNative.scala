package framework.inboxhelper

import framework.dsl.*
import fixture.constructedinbox.*
import fixture.constructedinbox.Helpers.given
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object HelperNative:
  def main(args: Array[String]): Unit =
    val original = summon[Finite[Helped]]
    val forwarded = Helpers.forward(original)
    assert(original eq forwarded)
    val types = Signature.stateTypes(
      Seq(forwarded -> Some("fixture.constructedinbox.Helped")),
      "state helper"
    )
    println(
      "helper:" + Base64.getEncoder.encodeToString(
        ir.Model(types = types, channels = Seq(first.toIR)).toByteArray
      )
    )
