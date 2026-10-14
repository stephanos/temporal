package framework.references

import java.util.Base64
import fixture.composedprogress.{flowing, graded}
import framework.dsl.Reify
import io.temporal.server.api.umpire.v1 as ir

object Native:
  private val captured = Reify.capture((value: Boolean) => !value)
  def main(args: Array[String]): Unit =
    assert(captured.callable(false))
    assert(!captured.callable(true))
    assert(captured.function.params.size == 1)
    val refs = (flowing.composed ++ graded.composed).map(_.toIR("native references"))
    println(
      "references:" + Base64.getEncoder.encodeToString(
        ir.Model(assumptions = Seq(ir.Assumption(name = "native", fairRefs = refs))).toByteArray
      )
    )
