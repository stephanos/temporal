package framework.inboxalias

import framework.dsl.*
import fixture.constructedinbox.*
import fixture.constructedinbox.given
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object AliasNative:
  def main(args: Array[String]): Unit =
    val witness = summon[Finite[Aliased]]
    val types =
      Signature.stateTypes(Seq(witness -> Some("fixture.constructedinbox.Aliased")), "state alias")
    println(
      "alias:" + Base64.getEncoder.encodeToString(
        ir.Model(types = types, channels = Seq(first.toIR)).toByteArray
      )
    )
    for (name, domain) <- Seq(
        "Duplicate" -> summon[Finite[Duplicate]],
        "DuplicateCases" -> summon[Finite[DuplicateCases]],
        "Missing" -> summon[Finite[Missing]]
      )
    do
      try
        Signature.stateSchema(domain, Some(s"fixture.constructedinbox.$name"), s"state $name"): Unit
        throw AssertionError(s"$name accepted invalid ownership")
      catch
        case error: IllegalArgumentException =>
          assert(error.getMessage.contains(s"state $name"))
          assert(error.getMessage.contains(if name == "Missing" then "ownership"
          else "holds a channel in one field"))
    println("native-refusals:passed")
