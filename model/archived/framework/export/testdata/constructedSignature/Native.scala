package framework.signature

import java.util.Base64
import fixture.signature.*
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val actions = Vector(
      caller.zero,
      caller.configure,
      caller.respond,
      caller.integer,
      caller.requiredInput,
      updateAlias,
      clock,
      commit,
      wire.deliver,
      wire.lose,
      other.deliver,
      counters.deliver,
      retries.lose
    )
    val model = Signature.collect(actions, Vector(wire, other, counters, retries))
    assert(
      model.actions
        .find(_.id == "fixture.signature.caller.integer")
        .get
        .inputs
        .head
        .getType
        .ref
        .isInt
    )
    val settingsValue = ir.Value(
      ir.Value.Kind.Record(
        ir.RecordValue(
          "fixture.signature.Settings",
          Seq(
            ir.Value(ir.Value.Kind.Enum(ir.EnumValue("fixture.signature.Mode", "stopped"))),
            ir.Value(ir.Value.Kind.Bool(false)),
            ir.Value(ir.Value.Kind.Int(0))
          )
        )
      )
    )
    val settingDefault = ir.Value(
      ir.Value.Kind.Enum(
        ir.EnumValue("scala.Option[fixture.signature.Settings]", "Some", Seq(settingsValue))
      )
    )
    val none =
      ir.Value(ir.Value.Kind.Enum(ir.EnumValue("scala.Option[fixture.signature.Mode]", "None")))
    val absentDefault = ir.Value(
      ir.Value.Kind.Enum(
        ir.EnumValue("scala.Option[scala.Option[fixture.signature.Mode]]", "Some", Seq(none))
      )
    )
    assert(
      caller.configure.inputMetadata
        .map(_.default)
        .equals(Vector(Some(settingDefault), Some(absentDefault)))
    )
    assert(
      caller
        .configure()
        .values
        .equals(List(Some(Settings(Mode.stopped, false, UpTo[1](0))), Some(None)))
    )
    assert(caller.requiredInput.inputMetadata.head.default.isEmpty)
    assert(updateAlias.decl.sourceBinding.contains("fixture.signature.update"))
    println("signature:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(
      "defaults:" + Base64.getEncoder.encodeToString(
        ir.ActionClass("fixture.signature.caller.configure", Seq(settingDefault, absentDefault))
          .toByteArray
      )
    )
