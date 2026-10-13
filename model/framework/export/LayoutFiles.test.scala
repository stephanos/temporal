package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.Tools
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

// The flat layout (fn-160 R7-R9) over model/framework/export/testdata/layoutFiles/urn: one machine per file,
// named after it; a level folder's files grouped by family prefix; the realization layer, its
// `<Subject>Run` names and its kit; and the layers' one-way imports, the Temporal API only in
// realization/. Each refusal is pinned at its source line; the accepted files are refused nothing.
class LayoutFilesFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/framework/export/testdata/layoutFiles/"

  test("the flat layout refuses each misplaced file, machine and import at its line"):
    val scratch = retained(build.resolve("history"), "layout-files.")
    val jar = scratch.resolve("layoutFiles.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          tools.directory.resolve(stored).toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val out = scratch.resolve("ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString,
        "urn"
      )
    )
    assertNotEquals(result.exit, 0, result.output)
    val refused = result.output.linesIterator.filter(_.startsWith("lift:")).toList
    val at = stored + "urn/"
    val prefixes = "Protocol, UrnProtocol, Heating, Boil"
    assertEquals(
      refused,
      List(
        // R9: the Product imports the Protocol.
        s"lift: ${at}product/Product.scala:5: R4.module-import: temporal.features.urn.product may " +
          "not import temporal.features.urn.protocol under the Scala module map",
        // R7: a machine in a file not named after it.
        s"lift: ${at}protocol/BoilOver.scala:6: BoilingOver is declared in ${at}protocol/" +
          "BoilOver.scala: a machine, composition or derivation sits in a file named after it, " +
          "BoilingOver.scala",
        // R8: a realization outside realization/.
        s"lift: ${at}protocol/BoilRun.scala:6: BoilRun is a realization in ${at}protocol/: " +
          "realizations sit in the feature's realization/ folder",
        // R9: the Temporal API outside realization/.
        s"lift: ${at}protocol/BoilWired.scala:4: R4.module-import: temporal.features.urn.protocol " +
          "may not import io.temporal.api.workflowservice.v1 under the Scala module map",
        // R7: a level folder's file with no family prefix.
        s"lift: ${at}protocol/Overflow.scala:6: Overflow.scala in ${at}protocol/ starts with none " +
          s"of its family prefixes, $prefixes: a level folder's files are grouped by the level's " +
          "name or its primary machine's, a facet's (enum FacetName) or a design family's (enum " +
          "FamilyName); name it `<Family><Design>.scala`, renaming its object, or declare its family",
        // R9: the Protocol imports the realization.
        s"lift: ${at}protocol/Protocol.scala:5: R4.module-import: temporal.features.urn.protocol " +
          "may not import temporal.features.urn.realization under the Scala module map",
        // R7: a second machine in a file, naming the file and the machine.
        s"lift: ${at}protocol/Protocol.scala:22: BoilSpare is a second machine, composition or " +
          s"derivation in ${at}protocol/Protocol.scala, beside UrnProtocol: each has a file of its " +
          "own, BoilSpare.scala",
        // R8: realization/ holds realizations and its kit alone.
        s"lift: ${at}realization/Helpers.scala:4: Helpers.scala is in ${at}realization/, the " +
          "realization layer, which holds one file per realization, `<Subject>Run.scala`, and its " +
          "shared helpers in Kit.scala",
        // R8: a realization not named after its subject, naming both.
        s"lift: ${at}realization/SpareRun.scala:7: SpareRun is a realization of BoilDry: name it " +
          "after what it runs, BoilDryRun",
        s"lift: ${at}realization/Starter.scala:7: Starter is a realization of HeatingSlow: name it " +
          "after what it runs, HeatingSlowRun"
      ),
      result.output
    )
    // Accepted: a family prefix (BoilDry), a facet's (HeatingSlow), the level's (ProtocolMember),
    // the main run named after the feature (UrnRun), a subject's run (BoilDryRun) and the kit.
    for accepted <- Seq("BoilDry.scala", "HeatingSlow.scala", "ProtocolMember.scala", "UrnRun", "BoilDryRun", "Kit.scala")
    do assert(!refused.exists(_.contains(s"/$accepted")), accepted)

  // R13 on the kind fixture (layout/kinds): a form's root feature file is `<Form><Kind>.scala`, and
  // what two forms share sits under the kind's `shared/`, never beside the forms.
  test("a misnamed form root and a shared Product beside the forms are refused, naming their files"):
    val scratch = retained(build.resolve("history"), "layout-kinds.")
    val modelJar = build.resolve("model-scala.jar")
    def variant(name: String)(change: java.nio.file.Path => Unit) =
      val tree = LayoutSources.materialize(
        tools.directory.resolve("model/framework/export/testdata/layout/kinds"),
        scratch.resolve(name),
        modelJar,
        Set.empty
      )
      change(tree)
      val jar = scratch.resolve(s"$name.jar")
      tools
        .buildFixture(Seq("--power", "package", "--server=false", "--library", tree.toString, "-f", "-o",
          jar.toString))
        .orFail(): Unit
      val result = InProcess.lift(
        tools.directory,
        Seq(
          "--ir",
          s"$jar=$tree/,$modelJar=model/",
          build.resolve("model-scala.classpath").toString,
          scratch.resolve(s"$name-out").toString
        )
      )
      assertNotEquals(result.exit, 0, result.output)
      tree -> result.output.linesIterator.filter(_.startsWith("lift:")).toList

    val (misnamed, rootRefusals) = variant("misnamed-root") { tree =>
      val form = tree.resolve("relay/standalone")
      Files.move(form.resolve("StandaloneRelay.scala"), form.resolve("Standalone.scala")): Unit
    }
    assert(
      rootRefusals.exists(r =>
        r.startsWith(s"lift: $misnamed/relay/standalone/system/System.scala:") &&
          r.endsWith(s"$misnamed/relay/standalone/ has no root feature file")
      ),
      rootRefusals.mkString("\n")
    )

    val (beside, besideRefusals) = variant("product-beside-forms") { tree =>
      val shared = tree.resolve("relay/shared/product/Product.scala")
      val moved = tree.resolve("relay/product/Product.scala")
      Files.createDirectories(moved.getParent)
      Files.move(shared, moved): Unit
      Files.delete(tree.resolve("relay/shared/product"))
      Files.delete(tree.resolve("relay/shared"))
      val stream = Files.walk(tree)
      try
        stream.iterator.asScala.filter(_.toString.endsWith(".scala")).foreach { file =>
          Files.writeString(
            file,
            Files.readString(file).replace("relay.shared.product", "relay.product")
              .replace("package shared\npackage product", "package product")
          ): Unit
        }
      finally stream.close()
    }
    assert(
      besideRefusals.exists(_.contains(
        s"$beside/relay/product/Product.scala is in $beside/relay/product/, which is no level " +
          "folder of relay: a kind keeps its sources in its general file, shared/product/ and named forms"
      )),
      besideRefusals.mkString("\n")
    )
