package umpire.irgen

import java.nio.file.Files
import java.util.zip.ZipFile
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}
import umpire.check.{Scratch, Tools}

class GapsSuite extends munit.FunSuite:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = Scratch.directory("umpire-gaps")
  private lazy val jar =
    val output = scratch.resolve("gaps.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--library",
          "--test",
          "--server=false",
          tools.directory.resolve("model/irgen/testdata/gaps").toString,
          "-f",
          "-o",
          output.toString
        )
      )
      .orFail(): Unit
    output

  private def inspect(check: Index => Unit): Unit =
    val zip = ZipFile(jar.toFile)
    val paths = try
      zip.entries.asScala
        .filter(_.getName.endsWith(".tasty"))
        .map { entry =>
          val path = scratch.resolve(entry.getName)
          Files.createDirectories(path.getParent): Unit
          val input = zip.getInputStream(entry)
          try Files.copy(input, path, java.nio.file.StandardCopyOption.REPLACE_EXISTING): Unit
          finally input.close()
          path.toString
        }
        .toList
    finally zip.close()
    assert(paths.nonEmpty)
    val build = tools.directory.resolve("model/build")
    val classpath = List(jar.toString, build.resolve("model-scala.jar").toString) ++
      Files
        .readString(build.resolve("model-scala.classpath"))
        .trim
        .split(java.io.File.pathSeparator)
        .toList
    val inspector = new Inspector:
      def inspect(using quotes: Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
        check(Index(tastys, Map.empty))
    assert(TastyInspector.inspectAllTastyFiles(paths, Nil, classpath)(inspector))

  test(
    "curried repeated annotations retain typed arguments, source order and one object attribution"
  ):
    inspect { index =>
      val annotations = GapAnnotations(index)
      val checked = annotations.declarations.filter(_.owner.fullName == "fixture.gaps.Checked$")
      assertEquals(checked.map(_.because), List("first reason", "second reason"))
      assertEquals(checked.map(_.until), List("fn-167 R2", ""))
      assert(checked.head.at.start < checked.last.at.start)
      assert(checked.head.subjects.head.show.contains("fixture.gaps.caller.move"))
      assert(checked.head.subjects.head.show.contains("fixture.gaps.Phase.ready"))
      assert(checked.head.subjects.head.show.contains("fixture.gaps.Phase.done"))
      assert(checked.head.at.sourceFile.path.endsWith("Positive.scala"))
      assertEquals(annotations.declarations.count(_.because == "property reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "query reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "refinement reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "facet reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "composition reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "realization reason"), 1)
      assertEquals(annotations.declarations.count(_.because == "repeated user reason"), 2)
      import index.quotes.reflect.*
      import index.given
      val mirrored = index.trees.flatMap { tree =>
        object collect extends TreeAccumulator[List[Symbol]]:
          def foldTree(found: List[Symbol], t: Tree)(owner: Symbol): List[Symbol] = t match
            case d: Definition
                if Set("fixture.gaps.Checked", "fixture.gaps.Checked$")(d.symbol.fullName) =>
              foldOverTree(d.symbol :: found, t)(owner)
            case _ => foldOverTree(found, t)(owner)
        collect.foldTree(Nil, tree)(Symbol.spliceOwner)
      }
      assertEquals(
        mirrored
          .flatMap(_.annotations)
          .count(_.tpe.typeSymbol.fullName == "framework.gap"),
        4
      )
    }

  test("gap admission refuses helpers, tests, lookalikes, foreign subjects and empty metadata"):
    inspect { index =>
      val errors = Conventions(index)
        .refusals(_ => false)
        .map(_._3.getMessage)
        .filter(_.contains("R8.gap"))
      for fragment <- Seq(
          "helper",
          "ordinary",
          "TestOnly",
          "Utility",
          "parameter",
          "Lookalike$.gap",
          "Lookalike$.accepted",
          "Foreign.unasked",
          "at least one subject",
          "non-blank reason"
        )
      do assert(errors.exists(_.contains(fragment)), errors.mkString("\n"))
      assert(errors.forall(line => line.matches("(?s).*\\.scala:[0-9]+:.*")), errors.toString)
      assert(!errors.exists(_.contains("Positive.scala")), errors.mkString("\n"))
    }
