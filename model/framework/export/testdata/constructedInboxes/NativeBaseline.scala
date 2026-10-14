package framework.inboxbaseline

import framework.dsl.*
import fixture.constructedinbox.*
import fixture.constructedinbox.given
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object NativeBaseline:
  def main(args: Array[String]): Unit =
    val schema = Signature.stateSchema(
      summon[Finite[First]],
      Some("fixture.constructedinbox.First"),
      "state First"
    )
    val actual = schema.types.find(_.name == schema.identity).get.getRecord.fields.head.getType
    assert(
      actual.getChannel == "fixture.constructedinbox.first",
      "state field still carries its message placeholder"
    )
    val types = Signature.stateTypes(
      Seq(
        summon[Finite[First]] -> Some("fixture.constructedinbox.First"),
        summon[Finite[Second]] -> Some("fixture.constructedinbox.Second"),
        summon[Finite[Bounded]] -> Some("fixture.constructedinbox.Bounded"),
        summon[Finite[Opaque]] -> Some("fixture.constructedinbox.Opaque"),
        summon[Finite[fixture.constructedinbox.Choice]] -> Some("fixture.constructedinbox.Choice"),
        summon[Finite[Nested]] -> Some("fixture.constructedinbox.Nested")
      ),
      "state oracle"
    )
    val channels = Seq(first, second, small, large, left, right).map(_.toIR).sortBy(_.id)
    println(
      "inboxes:" + Base64.getEncoder.encodeToString(
        ir.Model(types = types, channels = channels).toByteArray
      )
    )
