package framework.bindinghelper

import framework.dsl.*
import fixture.bindingdiscovery.*
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object HelperNative:
  def main(args: Array[String]): Unit =
    val value = CoreBindings.collect(HelperMachine)
    val repeated = CoreBindings.collect(HelperMachine)
    assert(value.equals(repeated))
    assert(
      value.functions.map(_.name).toSet == Set(
        "fixture.bindingdiscovery.Helpers$.forward",
        "fixture.bindingdiscovery.Helpers$.leaf",
        "helperMachine.second"
      )
    )
    assert(
      HelperMachine.bindings.head.captured.get.callable.eq(HelperMachine.bindings.head.function)
    )
    println(
      "helper:" + Base64.getEncoder.encodeToString(
        ir.Model(
          machines = Vector(ir.Machine(name = "helperMachine", steps = value.steps)),
          functions = value.functions,
          types = value.types
        ).toByteArray
      )
    )
