package framework.inspect

import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.{ZipEntry, ZipFile, ZipOutputStream}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*

class SuiteInventoryTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes

  private val flags = Vector(
    "-Werror",
    "-deprecation",
    "-feature",
    "-unchecked",
    "-Wunused:all",
    "-Wvalue-discard",
    "-Wnonunit-statement",
    "-Wsafe-init",
    "-Wimplausible-patterns",
    "-Yretain-trees",
    "-language:strictEquality"
  )
  private lazy val root = Files
    .createTempDirectory(
      Files.createDirectories(Path.of(".flow/tmp/fn19513")),
      "fixtures-"
    )
    .toRealPath()

  private def sha(path: Path): String =
    MessageDigest
      .getInstance("SHA-256")
      .digest(Files.readAllBytes(path))
      .map(byte => f"${byte & 0xff}%02x")
      .mkString

  private def run(label: String, command: Vector[String]): String =
    val log = root.resolve(label + ".log")
    val _ = Files.writeString(root.resolve(label + ".command"), command.mkString("\n") + "\n")
    val process = new ProcessBuilder(command*)
      .redirectErrorStream(true)
      .redirectOutput(log.toFile)
      .start()
    if !process.waitFor(600, java.util.concurrent.TimeUnit.SECONDS) then
      val _ = process.destroyForcibly()
      val _ = process.waitFor()
      fail(s"timeout: $command")
    val _ = Files.writeString(root.resolve(label + ".exit"), process.exitValue().toString + "\n")
    assertEquals(process.exitValue(), 0, Files.readString(log))
    Files.readString(log)

  private def compile(label: String, files: Vector[Path], classpath: String): (Path, String) =
    val output = root.resolve(label + ".jar")
    val command = Vector(
      "mise",
      "exec",
      "--",
      "scala-cli",
      "compile",
      "--server=false",
      "--suppress-outdated-dependency-warning",
      "--scala",
      "3.9.0",
      "--jvm",
      "27",
      "--workspace",
      root.resolve(label + "-workspace").toString,
      "--compilation-output",
      output.toString,
      "--print-class-path"
    ) ++ (if files.exists(_.toString.endsWith(".test.scala")) then Vector("--test")
          else Vector.empty) ++
      (if classpath.isEmpty then Vector.empty else Vector("--classpath", classpath)) ++
      flags.flatMap(flag => Vector("--scalac-option", flag)) ++ files.map(_.toString)
    val log = run(label, command)
    output -> log.linesIterator.filter(line => !line.startsWith("Picked up ")).toVector.last

  private def write(name: String, text: String): Path =
    val path = root.resolve(name)
    val _ = Files.createDirectories(path.getParent)
    val _ = Files.writeString(path, text)
    path

  private lazy val native = compile(
    "native",
    Vector(
      Path.of("model/framework/dsl"),
      Path.of("model/framework/realize")
    ),
    ""
  )

  private lazy val sentinel = root.resolve("sentinel")
  private def mark(name: String): String =
    s"java.nio.file.Files.writeString(java.nio.file.Path.of(\"$sentinel\"), \"$name\\n\", java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.APPEND): Unit"

  private lazy val helperSource = write(
    "Helpers.scala",
    s"""package inventorylib
    |class Helpers:
    |  ${mark("helper")}
    |  abstract class Base extends munit.FunSuite:
    |    object unit:
    |      inline def retainedInline: Int = 17
    |    test("inherited"):
    |      ${mark("test")}
    |      assertEquals(1, 1)
    |trait Extra
    |object Bindings:
    |  val helpers: Helpers = new Helpers
    |  val inferred = new Helpers
    |class HelperLeaf extends Bindings.helpers.Base
    |""".stripMargin
  )
  private lazy val helper = compile("helper", Vector(helperSource), native._2)
  private lazy val leafSource = write(
    "Leaf.test.scala",
    """package inventoryclient
    |class Leaf extends inventorylib.Bindings.helpers.Base with inventorylib.Extra
    |""".stripMargin
  )
  private lazy val leaf =
    compile("leaf", Vector(leafSource), helper._1.toString + java.io.File.pathSeparator + native._2)

  private def artifact(path: Path): SuiteInventory.Artifact =
    SuiteInventory.Artifact(path, sha(path))

  private def request: SuiteInventory.Request =
    val libraries = (Vector(native._1) ++ native._2
      .split(java.io.File.pathSeparator)
      .iterator
      .map(Path.of(_))
      .filter(path => Files.isRegularFile(path) && path.toString.endsWith(".jar"))
      .toVector).distinct
    SuiteInventory.Request(
      "inventory-client",
      Vector(artifact(leaf._1)),
      Vector(artifact(helper._1)),
      libraries.map(artifact),
      Vector(SuiteInventory.Source(leafSource, sha(leafSource))),
      Vector.empty,
      flags,
      SuiteInventory.Toolchain("3.9.0", System.getProperty("java.runtime.version"))
    )

  private def admitted(input: SuiteInventory.Request): SuiteInventory.Result =
    val result = SuiteInventory.inspect(input).fold(error => fail(error.message), identity)
    val _ = Files.writeString(
      root.resolve("inventory-evidence.log"),
      s"module=${result.input.module} fingerprint=${result.fingerprint} candidates=${result.input.candidates} hierarchy=${result.input.hierarchy} suites=${result.suites.map(_.facts.binaryName)}\n",
      java.nio.file.StandardOpenOption.CREATE,
      java.nio.file.StandardOpenOption.APPEND
    )
    result

  private def refused(input: SuiteInventory.Request, fragments: String*): Unit =
    val error =
      SuiteInventory.inspect(input).swap.toOption.getOrElse(fail("accepted invalid metadata"))
    val _ = Files.writeString(
      root.resolve("refusals.log"),
      error.message + "\n",
      java.nio.file.StandardOpenOption.CREATE,
      java.nio.file.StandardOpenOption.APPEND
    )
    assert(fragments.forall(error.message.contains), error.message)

  private lazy val matrixSource = write(
    "Matrix.test.scala",
    s"""package inventory
    |import framework.dsl.*
    |import framework.dsl.Finite.given
    |import framework.dsl.outcomes.Outcome
    |abstract class Parent extends munit.FunSuite:
    |  ${mark("constructor")}
    |  test("ordinary"):
    |    ${mark("test")}
    |    assertEquals(1, 1)
    |trait Transitive extends Parent
    |object Types:
    |  type Alias = Transitive
    |class Direct extends munit.FunSuite:
    |  test("direct"):
    |    assertEquals(1, 1)
    |class Indirect extends Types.Alias
    |class Empty extends munit.FunSuite
    |abstract class Abstract extends Parent
    |class Arguments(val value: Int) extends munit.FunSuite
    |class PrivateConstructor private () extends Parent
    |private class Private extends Parent
    |object Nested:
    |  ${mark("object")}
    |  class `a+b` extends Parent
    |class InnerOwner:
    |  class Inner extends Parent
    |object Singleton extends Parent
    |class Flag extends Machine[Outcome, Nothing]:
    |  ${mark("model")}
    |  var value = 0
    |  object transitions extends Transitions
    |  object invariants:
    |    val constant = check(true)
    |class ModelDirect extends ModelSuite(new Flag):
    |  object unit:
    |    test("model"):
    |      assertEquals(model.value, 0)
    |class ModelChild extends ModelDirect
    |class ExplicitLeaf extends inventorylib.Bindings.helpers.Base
    |class InferredLeaf extends inventorylib.Bindings.inferred.Base
    |object PositiveControl:
    |  def main(args: Array[String]): Unit =
    |    val _ = new Flag
    |    val _ = Nested
    |""".stripMargin
  )
  private lazy val duplicates = Vector(
    write("a/Repeated.test.scala", "package duplicate.a\nclass First extends munit.FunSuite\n"),
    write("b/Repeated.test.scala", "package duplicate.b\nclass Second extends munit.FunSuite\n")
  )
  private lazy val matrix = compile(
    "matrix",
    matrixSource +: duplicates,
    helper._1.toString + java.io.File.pathSeparator + native._2
  )
  private def matrixRequest: SuiteInventory.Request = request.copy(
    module = "matrix",
    candidates = Vector(artifact(matrix._1)),
    sources = (matrixSource +: duplicates).map(path => SuiteInventory.Source(path, sha(path)))
  )

  private def changedJar(original: Path, name: String)(keep: String => Boolean): Path =
    rewriteJar(original, name)((entry, bytes) => if keep(entry) then Some(bytes) else None)

  private def rewriteJar(original: Path, name: String)(
      change: (String, Array[Byte]) => Option[Array[Byte]]
  ): Path =
    val target = root.resolve(name + ".jar")
    val zip = new ZipFile(original.toFile)
    val out = new ZipOutputStream(Files.newOutputStream(target))
    try
      zip
        .entries()
        .asScala
        .filterNot(_.isDirectory)
        .foreach: entry =>
          val input = zip.getInputStream(entry)
          val bytes = try input.readAllBytes()
          finally input.close()
          change(entry.getName, bytes).foreach: result =>
            out.putNextEntry(new ZipEntry(entry.getName))
            out.write(result)
            out.closeEntry()
    finally
      out.close()
      zip.close()
    target

  private def zipBytes(path: Path, name: String): Array[Byte] =
    val zip = new ZipFile(path.toFile)
    try
      val stream = zip.getInputStream(zip.getEntry(name))
      try stream.readAllBytes()
      finally stream.close()
    finally zip.close()

  private def unpack(path: Path, name: String): SuiteInventory.Artifact =
    val directory = Files.createDirectories(root.resolve(name))
    val zip = new ZipFile(path.toFile)
    try
      zip
        .entries()
        .asScala
        .filterNot(_.isDirectory)
        .foreach: entry =>
          val target = directory.resolve(entry.getName)
          val _ = Files.createDirectories(target.getParent)
          val _ = Files.write(target, zipBytes(path, entry.getName))
    finally zip.close()
    val stream = Files.walk(directory)
    val files = try
      stream
        .iterator()
        .asScala
        .filter(Files.isRegularFile(_))
        .map(file =>
          directory.relativize(file).toString.replace('\\', '/') -> Files.readAllBytes(file)
        )
        .toVector
        .sortBy(_._1)
    finally stream.close()
    val encoded = new java.io.ByteArrayOutputStream()
    val data = new java.io.DataOutputStream(encoded)
    files.foreach: (entry, bytes) =>
      val name = entry.getBytes(java.nio.charset.StandardCharsets.UTF_8)
      data.writeInt(name.length)
      data.writeLong(bytes.length.toLong)
      data.write(name)
      data.write(bytes)
    data.close()
    val hash = MessageDigest
      .getInstance("SHA-256")
      .digest(encoded.toByteArray)
      .map(byte => f"${byte & 0xff}%02x")
      .mkString
    SuiteInventory.Artifact(directory, hash)

  private def withoutInterface(bytes: Array[Byte]): Array[Byte] =
    val buffer = java.nio.ByteBuffer.wrap(bytes)
    def u2(index: Int): Int = buffer.getShort(index).toInt & 0xffff
    var offset = 10
    var slot = 1
    val count = u2(8)
    while slot < count do
      val tag = bytes(offset).toInt & 0xff
      offset += 1
      tag match
        case 1                                  => offset += 2 + u2(offset)
        case 3 | 4 | 9 | 10 | 11 | 12 | 17 | 18 => offset += 4
        case 5 | 6                              =>
          offset += 8
          slot += 1
        case 7 | 8 | 16 | 19 | 20 => offset += 2
        case 15                   => offset += 3
        case other                => fail(s"unexpected fixture constant-pool tag $other")
      slot += 1
    assertEquals(u2(offset + 6), 1)
    bytes.take(offset + 6) ++ Array[Byte](0, 0) ++ bytes.drop(offset + 10)

  test("classfile-existing Base omitted from TASTy refuses while the leaf remains"):
    val partial = changedJar(helper._1, "omitted-base")(_ != "inventorylib/Helpers.tasty")
    val result = SuiteInventory.inspect(request.copy(hierarchy = Vector(artifact(partial))))
    assert(
      result.isLeft,
      "accepted incomplete author graph: Helpers$Base has classfile bytes but no typed declaration"
    )
    assert(result.swap.toOption.get.message.contains("inventorylib.Helpers$Base"))

  test("missing interface declaration refuses independently of the superclass"):
    val partial = changedJar(helper._1, "omitted-interface")(_ != "inventorylib/Extra.tasty")
    refused(
      request.copy(hierarchy = Vector(artifact(partial))),
      "inventorylib.Extra",
      "typed declaration"
    )

  test("classfile interface omissions and typed edge disagreements refuse independently"):
    val missing = changedJar(helper._1, "missing-interface-class")(_ != "inventorylib/Extra.class")
    refused(request.copy(hierarchy = Vector(artifact(missing))), "inventorylib.Extra")
    val missingEdge = rewriteJar(leaf._1, "omitted-interface-edge")((entry, bytes) =>
      Some(if entry == "inventoryclient/Leaf.class" then withoutInterface(bytes) else bytes)
    )
    refused(
      request.copy(candidates = Vector(artifact(missingEdge))),
      "inventoryclient.Leaf",
      "parent edges disagree"
    )

  test("swapped TASTy generations corrupted bytes and incompatible metadata refuse"):
    val alternate =
      write("alternate/Helpers.scala", Files.readString(helperSource).replace("= 17", "= 23"))
    val generation = compile("alternate-helper", Vector(alternate), native._2)
    val swapped = rewriteJar(helper._1, "swapped-generation")((entry, bytes) =>
      Some(if entry == "inventorylib/Helpers.tasty" then zipBytes(generation._1, entry) else bytes)
    )
    refused(
      request.copy(hierarchy = Vector(artifact(swapped))),
      "Helpers",
      "mixed classfile/TASTy generation"
    )
    val corruptClass = rewriteJar(helper._1, "corrupt-classfile")((entry, bytes) =>
      Some(if entry == "inventorylib/Extra.class" then Array[Byte](1, 2, 3) else bytes)
    )
    refused(
      request.copy(hierarchy = Vector(artifact(corruptClass))),
      "Extra.class",
      "malformed classfile"
    )
    val corruptTasty = rewriteJar(helper._1, "corrupt-tasty")((entry, bytes) =>
      Some(if entry == "inventorylib/Extra.tasty" then Array[Byte](1, 2, 3) else bytes)
    )
    refused(
      request.copy(hierarchy = Vector(artifact(corruptTasty))),
      "Extra.tasty",
      "incompatible TASTy"
    )
    val incompatible = rewriteJar(helper._1, "incompatible-classfile")((entry, bytes) =>
      Some(if entry == "inventorylib/Extra.class" then
        bytes.take(6) ++ Array[Byte](0x7f, 0x7f) ++ bytes.drop(8)
      else bytes)
    )
    refused(request.copy(hierarchy = Vector(artifact(incompatible))), "Extra.class", "classfile")

  test("fresh library ownership preserves complete typed ancestors and source-less positions"):
    val result = admitted(request)
    assertEquals(result.suites.map(_.facts.binaryName), Vector("inventoryclient.Leaf"))
    val suite = result.suites.head
    assertEquals(
      suite.ancestors.map(_.binaryName).toSet,
      Set("inventoryclient.Leaf", "inventorylib.Helpers$Base", "inventorylib.Extra")
    )
    val base = suite.ancestors.find(_.binaryName == "inventorylib.Helpers$Base").get
    assertEquals(base.location.artifact.path, helper._1.toRealPath())
    assertEquals(base.location.artifact.sha256, sha(helper._1))
    assertEquals(base.location.source, None)
    assert(base.location.start >= 0 && base.location.end > base.location.start)
    assert(base.location.startLine >= 0)
    assert(base.tree.show.contains("retainedInline"))
    assert(base.tree.show.contains("17"))
    val saved = root.resolve("saved-library-source")
    val _ = Files.move(helperSource, saved)
    try
      assert(!Files.exists(helperSource))
      assertEquals(admitted(request).suites.map(_.facts.binaryName), Vector("inventoryclient.Leaf"))
      val _ = Files.copy(saved, helperSource)
      val permissions = Files.getPosixFilePermissions(helperSource)
      try
        val _ = Files.setPosixFilePermissions(
          helperSource,
          java.util.Set.of[java.nio.file.attribute.PosixFilePermission]()
        )
        assert(
          !Files.isReadable(helperSource),
          "permission-denied source fixture must be unreadable"
        )
        assertEquals(
          admitted(request).suites.map(_.facts.binaryName),
          Vector("inventoryclient.Leaf")
        )
      finally
        val _ = Files.setPosixFilePermissions(helperSource, permissions)
        Files.delete(helperSource)
    finally
      val _ = Files.move(saved, helperSource)

  test("exact JVM eligibility preserves no-test and Scala-private public bytecode"):
    val result = admitted(matrixRequest)
    assertEquals(
      result.suites.map(_.facts.binaryName),
      Vector(
        "duplicate.a.First",
        "duplicate.b.Second",
        "inventory.Direct",
        "inventory.Empty",
        "inventory.ExplicitLeaf",
        "inventory.Indirect",
        "inventory.InferredLeaf",
        "inventory.ModelChild",
        "inventory.ModelDirect",
        "inventory.Nested$a$plusb",
        "inventory.Private"
      )
    )
    val excluded = result.exclusions.map(_.facts.binaryName).toSet
    assert(
      Set(
        "inventory.Abstract",
        "inventory.Parent",
        "inventory.Transitive",
        "inventory.Arguments",
        "inventory.PrivateConstructor",
        "inventory.InnerOwner$Inner",
        "inventory.Singleton$"
      ).subsetOf(excluded)
    )
    assertEquals(
      result.suites
        .find(_.facts.binaryName == "inventory.Indirect")
        .get
        .ancestors
        .map(_.binaryName)
        .toSet,
      Set("inventory.Indirect", "inventory.Transitive", "inventory.Parent")
    )
    val first = result.suites.find(_.facts.binaryName == "duplicate.a.First").get
    val second = result.suites.find(_.facts.binaryName == "duplicate.b.Second").get
    assertEquals(first.declaration.location.source.get.path, duplicates.head.toRealPath())
    assertEquals(second.declaration.location.source.get.path, duplicates(1).toRealPath())

  test("candidate skips follow hierarchy completion and do not exclude descendants"):
    val combined = request.copy(
      candidates = Vector(artifact(helper._1), artifact(leaf._1)),
      hierarchy = Vector.empty,
      sources =
        Vector(helperSource, leafSource).map(path => SuiteInventory.Source(path, sha(path))),
      skippedRoots = Vector(helperSource)
    )
    assertEquals(admitted(combined).suites.map(_.facts.binaryName), Vector("inventoryclient.Leaf"))
    val skipped = admitted(request.copy(skippedRoots = Vector(leafSource)))
    assertEquals(skipped.suites.size, 0)
    assertEquals(skipped.exclusions.map(_.reason), Vector("skipped source"))

  test("distinct required owners refuse in candidate and hierarchy combinations"):
    val duplicate = root.resolve("helper-copy.jar")
    val _ = Files.copy(helper._1, duplicate)
    val other = artifact(duplicate)
    val cases = Vector(
      request.copy(candidates = Vector(artifact(leaf._1), artifact(helper._1), other)),
      request.copy(
        candidates = Vector(artifact(leaf._1), artifact(helper._1)),
        hierarchy = Vector(other)
      ),
      request.copy(hierarchy = Vector(artifact(helper._1), other)),
      request.copy(hierarchy = Vector(other, artifact(helper._1)))
    )
    cases.foreach(input => refused(input, "conflicting binary owners", "inventorylib."))
    val repeated =
      admitted(request.copy(hierarchy = Vector(artifact(helper._1), artifact(helper._1))))
    assertEquals(repeated.fingerprint, admitted(request).fingerprint)
    val alias = root.resolve("helper-alias.jar")
    val _ = Files.createSymbolicLink(alias, helper._1)
    assertEquals(
      admitted(request.copy(hierarchy = Vector(artifact(helper._1), artifact(alias)))).fingerprint,
      admitted(request).fingerprint
    )

  test("directory outputs use verified entries and reject escaping symbolic entries"):
    val output = unpack(leaf._1, "leaf-classes")
    assertEquals(
      admitted(request.copy(candidates = Vector(output))).suites.map(_.facts.binaryName),
      Vector("inventoryclient.Leaf")
    )
    val link = output.path.resolve("escape.class")
    val _ = Files.createSymbolicLink(link, helper._1)
    try refused(request.copy(candidates = Vector(output)), "symbolic artifact entry")
    finally Files.delete(link)

  test("missing changed malformed and incompatible inputs refuse with context"):
    val input = request
    val malformed = write("corrupt.jar", "not a jar")
    val cases = Vector(
      input.copy(candidates = Vector.empty) -> "missing candidate outputs",
      input.copy(candidates =
        Vector(SuiteInventory.Artifact(root.resolve("missing.jar"), "absent"))
      ) -> "missing.jar",
      input.copy(candidates =
        Vector(input.candidates.head.copy(sha256 = "wrong"))
      ) -> "changed artifact",
      input.copy(candidates = Vector(artifact(malformed))) -> "zip",
      input.copy(hierarchy = Vector.empty) -> "inventorylib.Helpers$Base",
      input.copy(sources = Vector(input.sources.head.copy(sha256 = "wrong"))) -> "changed source",
      input.copy(sources = input.sources ++ input.sources) -> "conflicting source ownership",
      input.copy(sources = Vector.empty) -> "source outside explicit manifest",
      input.copy(sources =
        Vector(SuiteInventory.Source(helperSource, sha(helperSource)))
      ) -> "source outside explicit manifest",
      input.copy(compilerFlags = Vector.empty) -> "missing compiler flags",
      input.copy(toolchain = input.toolchain.copy(scalaVersion = "3.8.0")) -> "incompatible Scala",
      input.copy(toolchain = input.toolchain.copy(javaVersion = "other")) -> "incompatible Java"
    )
    cases.foreach((value, fragment) => refused(value, fragment))

  test("result binding includes module flags source manifest and skipped roots"):
    val original = admitted(request)
    val changes = Vector(
      request.copy(module = "other"),
      request.copy(compilerFlags = flags.reverse),
      request.copy(skippedRoots = Vector(leafSource))
    )
    changes.foreach(input => assert(admitted(input).fingerprint != original.fingerprint))
    assertEquals(original.input.module, "inventory-client")
    assertEquals(original.input.compilerFlags, flags)

  test("inspection has no author initialization and separate execution fires each sentinel"):
    intercept[ClassNotFoundException](Class.forName("framework.checker.Checker"))
    val _ = admitted(matrixRequest)
    val _ = admitted(request)
    assert(!Files.exists(sentinel), "author initialized during compile/inspection")
    val cp = Vector(matrix._1.toString, leaf._1.toString, helper._1.toString, native._2)
      .mkString(java.io.File.pathSeparator)
    val execution = run(
      "independent-junit",
      Vector(
        "mise",
        "exec",
        "--",
        "java",
        "-cp",
        cp,
        "org.junit.runner.JUnitCore",
        "inventory.Indirect",
        "inventory.Private",
        "inventory.Nested$a$plusb",
        "inventoryclient.Leaf"
      )
    )
    assert(execution.contains("OK (4 tests)"), execution)
    val _ = run(
      "independent-model-object",
      Vector("mise", "exec", "--", "java", "-cp", cp, "inventory.PositiveControl")
    )
    assertEquals(
      Files.readString(sentinel).linesIterator.toSet,
      Set("helper", "constructor", "object", "model", "test")
    )
