package framework.inspect

import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.{ZipEntry, ZipFile, ZipOutputStream}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*

class SuiteInventoryTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes

  private val flags = Vector(
    "-Werror", "-deprecation", "-feature", "-unchecked", "-Wunused:all",
    "-Wvalue-discard", "-Wnonunit-statement", "-Wsafe-init",
    "-Wimplausible-patterns", "-Yretain-trees", "-language:strictEquality"
  )
  private lazy val root = Files.createTempDirectory(
    Files.createDirectories(Path.of(".flow/tmp/fn19513")), "fixtures-"
  ).toRealPath()

  private def sha(path: Path): String =
    MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(path))
      .map(byte => f"${byte & 0xff}%02x").mkString

  private def run(label: String, command: Vector[String]): String =
    val log = root.resolve(label + ".log")
    val process = new ProcessBuilder(command*)
      .redirectErrorStream(true).redirectOutput(log.toFile).start()
    assert(process.waitFor(600, java.util.concurrent.TimeUnit.SECONDS), s"timeout: $command")
    assertEquals(process.exitValue(), 0, Files.readString(log))
    Files.readString(log)

  private def compile(label: String, files: Vector[Path], classpath: String): (Path, String) =
    val output = root.resolve(label + ".jar")
    val command = Vector(
      "mise", "exec", "--", "scala-cli", "compile", "--server=false",
      "--suppress-outdated-dependency-warning", "--scala", "3.9.0", "--jvm", "27",
      "--workspace", root.resolve(label + "-workspace").toString,
      "--compilation-output", output.toString, "--print-class-path"
    ) ++ (if files.exists(_.toString.endsWith(".test.scala")) then Vector("--test") else Vector.empty) ++
      (if classpath.isEmpty then Vector.empty else Vector("--classpath", classpath)) ++
      flags.flatMap(flag => Vector("--scalac-option", flag)) ++ files.map(_.toString)
    val log = run(label, command)
    output -> log.linesIterator.filter(line => !line.startsWith("Picked up ")).toVector.last

  private def write(name: String, text: String): Path =
    val path = root.resolve(name)
    val _ = Files.createDirectories(path.getParent)
    val _ = Files.writeString(path, text)
    path

  private lazy val native = compile("native", Vector(
    Path.of("model/framework/dsl"), Path.of("model/framework/realize")
  ), "")

  private lazy val helperSource = write("Helpers.scala", """package inventorylib
    |class Helpers:
    |  abstract class Base extends munit.FunSuite:
    |    object unit:
    |      inline def retainedInline: Int = 17
    |trait Extra
    |object Bindings:
    |  val helpers: Helpers = new Helpers
    |""".stripMargin)
  private lazy val helper = compile("helper", Vector(helperSource), native._2)
  private lazy val leafSource = write("Leaf.test.scala", """package inventoryclient
    |class Leaf extends inventorylib.Bindings.helpers.Base with inventorylib.Extra
    |""".stripMargin)
  private lazy val leaf = compile("leaf", Vector(leafSource), helper._1.toString + java.io.File.pathSeparator + native._2)

  private def artifact(path: Path): SuiteInventory.Artifact =
    SuiteInventory.Artifact(path, sha(path))

  private def request: SuiteInventory.Request =
    val libraries = (Vector(native._1) ++ native._2.split(java.io.File.pathSeparator).iterator.map(Path.of(_))
      .filter(path => Files.isRegularFile(path) && path.toString.endsWith(".jar"))
      .toVector).distinct
    SuiteInventory.Request(
      "inventory-client", Vector(artifact(leaf._1)), Vector(artifact(helper._1)),
      libraries.map(artifact), Vector(SuiteInventory.Source(leafSource, sha(leafSource))),
      Vector.empty, flags, SuiteInventory.Toolchain("3.9.0", System.getProperty("java.runtime.version"))
    )

  private def changedJar(original: Path, name: String)(keep: String => Boolean): Path =
    val target = root.resolve(name + ".jar")
    val zip = new ZipFile(original.toFile)
    val out = new ZipOutputStream(Files.newOutputStream(target))
    try
      zip.entries().asScala.filter(entry => !entry.isDirectory && keep(entry.getName))
        .foreach: entry =>
          out.putNextEntry(new ZipEntry(entry.getName))
          val input = zip.getInputStream(entry)
          try out.write(input.readAllBytes())
          finally input.close()
          out.closeEntry()
    finally
      out.close()
      zip.close()
    target

  test("classfile-existing Base omitted from TASTy refuses while the leaf remains"):
    val partial = changedJar(helper._1, "omitted-base")(_ != "inventorylib/Helpers.tasty")
    val result = SuiteInventory.inspect(request.copy(hierarchy = Vector(artifact(partial))))
    assert(result.isLeft, "accepted incomplete author graph: Helpers$Base has classfile bytes but no typed declaration")
    assert(result.swap.toOption.get.message.contains("inventorylib.Helpers$Base"))
