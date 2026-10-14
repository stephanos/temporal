// The run-time exporter of the spike: constructs the Model, reads the IR its capture points baked in
// at compile time, and writes it with the lifter's own printer (generated IR classes, ProtoJSON,
// Jackson's indented layout), so the bytes compare with a lift.
//> using jar ../../repo/model/build/ir-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-json4s:0.12.2
package framework.reify

import com.fasterxml.jackson.core.JsonGenerator
import com.fasterxml.jackson.core.util.DefaultPrettyPrinter
import io.temporal.server.api.umpire.v1 as ir
import java.nio.file.{Files, Path}
import org.json4s.*
import org.json4s.jackson.JsonMethods
import scalapb.json4s.{Parser, Printer}

final private class Pretty extends DefaultPrettyPrinter:
  override def createInstance(): DefaultPrettyPrinter = Pretty()
  override def writeObjectFieldValueSeparator(g: JsonGenerator): Unit = g.writeRaw(": ")

private def json(value: JValue): String =
  JsonMethods.mapper.writer(Pretty()).writeValueAsString(JsonMethods.asJsonNode(value)) + "\n"

// The functions of every reified JSON array, by name, refusing two different bodies of one name.
def merged(arrays: Seq[String]): Seq[ir.Function] =
  val byName = scala.collection.mutable.LinkedHashMap.empty[String, ir.Function]
  for array <- arrays; f <- JsonMethods.parse(array).children do
    val fn = Parser().fromJsonString[ir.Function](JsonMethods.compact(f))
    byName.get(fn.name) match
      case Some(other) if !other.equals(fn) => sys.error(s"${fn.name} reified twice, differently")
      case _                          => byName(fn.name) = fn
  byName.values.toSeq.sortBy(_.name)

// usage: export <out.json> [<lifted.json> <lifted-subset.json>]
@main def exportKettle(out: String, compare: String*): Unit =
  val fns = merged(samples.Kettle.Model.rules.reified.toSeq)
  Files.writeString(Path.of(out), json(Printer().toJson(ir.Model(functions = fns)))): Unit
  compare match
    case Seq(lifted, subset) =>
      val model = Parser().fromJsonString[ir.Model](Files.readString(Path.of(lifted)))
      val names = fns.map(_.name).toSet
      val kept = model.functions.filter(f => names(f.name))
      Files.writeString(Path.of(subset), json(Printer().toJson(ir.Model(functions = kept)))): Unit
      println(s"reified ${fns.size} functions: ${fns.map(_.name).mkString(", ")}")
      println(s"lifted counterparts: ${kept.size}; equal as messages: ${kept.equals(fns)}")
    case _ => ()
