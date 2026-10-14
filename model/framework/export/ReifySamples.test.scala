package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, RetainedScratch, Tools}

class ReifySamplesSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "reify-samples.")
  private lazy val build =
    val out = scratch.resolve("samples.jar")
    val compiled = tools.scalaCli(Seq("compile", "--print-class-path", "--compilation-output", out.toString,
      "model/framework/dsl", "model/framework/realize", "model/framework/samples", "--server=false",
      "--scalac-option", "-sourceroot", "--scalac-option", tools.directory.toString), Output.KeptApart).orFail()
    val path = scratch.resolve("classpath")
    Files.writeString(path, compiled.output): Unit
    (out, path)

  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for row <- node.fields().asScala if row.getKey != "position" do
        out.set[JsonNode](row.getKey, withoutPositions(row.getValue)): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node

  for (name, count, more) <- Seq(
      ("Bank", 15, Seq.empty),
      ("Kettle", 13, Seq.empty),
      ("ReliableBroadcast", 37, Seq("everyoneDelivers")),
      ("TwoLayeredCache", 20, Seq("agreement")),
      ("TwoPhaseCommit", 45, Seq("commitReachesEveryone")),
      ("Voting", 37, Seq.empty)
    )
  do
    test(s"the shared old-host engine preserves all $count $name functions against pre-port IR"):
      val out = scratch.resolve(name + ".json")
      val (samples, classpath) = build
      val roots = (Seq("queries") ++ more).map(s"samples.$name$$." + _)
      val ran = InProcess.lift(tools.directory, Seq(
        s"$samples=",
        classpath.toString, out.toString) ++ roots)
      assertEquals(ran.exit, 0, ran.diagnostics)
      val actual = mapper.readTree(Files.readString(out))
      val expected = mapper.readTree(Files.readString(tools.directory.resolve(
        s"model/framework/export/testdata/reify/samples/$name.json")))
      assertEquals(actual.path("functions").size(), count)
      assertEquals(withoutPositions(actual.path("functions")), expected.path("functions"))
      assertEquals(withoutPositions(actual.path("types")), expected.path("types"))
