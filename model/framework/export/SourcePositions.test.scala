package framework.`export`

import java.nio.file.Files
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}
import framework.check.Tools
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class SourcePositionsFixtures extends munit.FunSuite with RetainedScratch:
  test("module and section refusals retain physical authored source lines"):
    val tools = Tools.here
    val build = tools.directory.resolve("model/build")
    val stored = "model/framework/export/testdata/layoutRefusals/c/"
    val scratch =
      retained(build.resolve("history"), "source-positions.")
    val jar = scratch.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          ModelBuild.copied(tools.directory.resolve(stored), scratch.resolve("sources")).toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val zip = ZipFile(jar.toFile)
    val files = try
      zip.entries.asScala.toList
        .filter(_.getName.endsWith(".tasty"))
        .map: entry =>
          val target = scratch.resolve(entry.getName)
          Files.createDirectories(target.getParent): Unit
          val stream = zip.getInputStream(entry)
          try Files.copy(stream, target): Unit
          finally stream.close()
          target.toString
    finally zip.close()
    val positions = mutable.Map.empty[String, String]
    val refusals = mutable.ArrayBuffer.empty[(String, Int)]
    val ordered = mutable.ArrayBuffer.empty[String]
    val inspector = new Inspector:
      def inspect(using
          quotes: Quotes
      )(tastys: List[Tasty[quotes.type]]): Unit =
        val index = Index(tastys, files.map(_ -> stored).toMap)
        import index.quotes.reflect.*
        val context = Context(index)
        object visit extends TreeTraverser:
          override def traverseTree(tree: Tree)(owner: Symbol): Unit =
            tree match
              case definition: ClassDef
                  if definition.symbol.flags.is(Flags.Module) &&
                    Set("KilnProduct", "timers", "helpers")(
                      definition.name.stripSuffix("$")
                    ) =>
                positions(definition.name.stripSuffix("$")) = context.where(
                  definition.asInstanceOf[context.quotes.reflect.Tree]
                )
              case _ => ()
            super.traverseTree(tree)(owner)
        index.trees.foreach(visit.traverseTree(_)(Symbol.spliceOwner))
        refusals ++= Structure(index)
          .refusals(_ => false)
          .map((file, line, _) => file -> line)
        ordered ++= Order(index).refusals.map(_.getMessage)
    val classpath = jar.toString +: Files
      .readString(build.resolve("model-scala.classpath"))
      .trim
      .split(java.io.File.pathSeparator)
      .toList
    assert(
      TastyInspector.inspectAllTastyFiles(files, Nil, classpath)(inspector)
    )
    val product = stored + "kiln/product/Product.scala"
    assertEquals(
      positions.toMap,
      Map(
        "KilnProduct" -> s"$product:6",
        "timers" -> s"$product:17",
        "helpers" -> s"$product:21"
      )
    )
    assertEquals(
      refusals.toSeq.sorted,
      Seq(
        product -> 17,
        product -> 21
      ).sorted
    )
    assertEquals(
      ordered.toSeq.filter(_.contains("Gauge belongs before")),
      Seq(
        s"${stored}pump/system/System.scala:21: Gauge belongs before PumpSystem at " +
          s"${stored}pump/system/System.scala:7: a level folder's file reads its header, then its types, " +
          "then its signature, then its machine and composition objects"
      )
    )
