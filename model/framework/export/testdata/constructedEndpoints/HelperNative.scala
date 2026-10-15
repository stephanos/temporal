package framework.endpointhelpernative

import java.util.Base64
import framework.dsl.*
import fixture.endpointclient.HelperJar
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val value = MachineEndpoints.collect(HelperJar)
    assert(MachineEndpoints.collect(HelperJar).equals(value))
    assert(HelperJar.init && !HelperJar.end(true))
    assert(
      value.functions.map(_.name) == Vector(
        "fixture.endpointlibrary.Helpers$.forward",
        "fixture.endpointlibrary.Helpers$.leaf"
      )
    )
    assert(
      value.functions.forall(_.position.exists(p => p.file.endsWith("Helpers.scala") && p.line > 0))
    )
    println(
      "endpoints:" + Base64.getEncoder.encodeToString(
        ir.Model(
          machines =
            Seq(ir.Machine(name = "helperJar", starts = value.starts, ends = Some(value.end))),
          functions = value.functions,
          types = value.types
        ).toByteArray
      )
    )
