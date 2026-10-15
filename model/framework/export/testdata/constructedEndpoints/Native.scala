package framework.endpointnative

import java.util.Base64
import framework.dsl.*
import fixture.endpoints.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val machines = Vector(
      DirectVal,
      DirectDef,
      Default,
      Explicit,
      OpenOverride,
      Forwarded,
      Outer.Nested,
      HelperRequire,
      HelperEnsuring,
      ChannelEnd,
      HoleEnd
    )
    val contributions = machines.map { machine =>
      val value = MachineEndpoints.collect(machine)
      assert(MachineEndpoints.collect(machine).equals(value))
      val name = machine.headerIdentity.get._2
      name -> value
    }
    val functions = contributions
      .flatMap(_._2.functions)
      .groupBy(_.name)
      .toVector
      .sortBy(_._1)
      .map { (_, copies) => assert(copies.distinct.size == 1); copies.head }
    val types = contributions
      .flatMap(_._2.types)
      .groupBy(_.name)
      .toVector
      .sortBy(_._1)
      .map { (_, copies) => assert(copies.distinct.size == 1); copies.head }
    val model = ir.Model(
      machines = contributions.sortBy(_._1).map { (name, value) =>
        ir.Machine(name = name, starts = value.starts, ends = Some(value.end))
      },
      functions = functions,
      types = types
    )
    assert(
      functions.forall(_.position.exists(p => p.file.endsWith("Endpoints.scala") && p.line > 0))
    )
    val closed = MachineEndpoints.collect(Default).end.getLambda
    assert(closed.params.map(_.name) == Seq("s"))
    assert(
      closed.getBody.getBinary.getRight.getList.items.map(_.getLiteral.getEnum.`case`) == Seq(
        "done",
        "failed"
      )
    )
    assert(MachineEndpoints.collect(DirectVal).end.getLambda.params.map(_.name) == Seq("written"))
    assert(MachineEndpoints.collect(Explicit).end.getLambda.params.map(_.name) == Seq("authored"))
    assert(MachineEndpoints.collect(Forwarded).equals(MachineEndpoints.collect(DirectVal)))
    val chain = DirectVal.unmonitored.unmonitored
    assert(MachineEndpoints.collect(chain).equals(MachineEndpoints.collect(DirectVal)))
    assert(chain.fs eq DirectVal.fs)
    assert(!DirectVal.init && DirectVal.end(true) && DirectDef.end(false))
    assert(Default.end(Snapshot(Phase.done)) && Default.end(Snapshot(Phase.failed)))
    assert(!Default.end(Snapshot(Phase.open)))
    assert(Explicit.end(Snapshot(Phase.open)) && !Explicit.end(Snapshot(Phase.done)))
    assert(OpenOverride.end(OpenState(OpenPhase.open)))
    assert(HelperRequire.init && HelperRequire.end(true))
    assert(!HelperEnsuring.init && HelperEnsuring.end(true))
    assert(ChannelEnd.end(ChannelEnd.init) && HoleEnd.end(true))
    assert(contributions.flatMap(_._2.channels).distinct == Vector("fixture.endpoints.wire"))
    assert(contributions.flatMap(_._2.holes).distinct == Vector("fixture.endpoints.unknown"))
    println("endpoints:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println("endpoint-controls:explicit,default,contracts,nested,derived,built,channels,holes")
