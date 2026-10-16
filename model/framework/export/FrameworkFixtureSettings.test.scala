package framework.`export`

import java.nio.file.{Files, Path}
import framework.check.Tools

class FrameworkFixtureSettingsSuite extends munit.FunSuite:
  test("generic witness settings resolve only the fresh core jar without shifting source lines"):
    val build = sys.env.get("UMPIRE_FRAMEWORK_BUILD")
    assume(build.isDefined, "the framework command supplies fresh generic artifacts")
    val original = Files.readString(
      Tools.here.directory.resolve(
        "model/framework/export/testdata/witness/project.scala"
      )
    )
    val rewritten = ModelBuild.fixtureSettings(original)
    val jars = rewritten.linesIterator.collect:
      case line if line.startsWith("//> using jar ") => line.stripPrefix("//> using jar ")
    assertEquals(jars.toVector, Vector(Path.of(build.get).resolve("framework.jar").toString))
    assertEquals(rewritten.linesIterator.size, original.linesIterator.size)
    assertEquals(
      rewritten.linesIterator.toVector(4).length,
      original.linesIterator.toVector(4).length
    )

  test("framework mode holds real mini and request-selector cases before their bodies can run"):
    assume(sys.env.contains("UMPIRE_FRAMEWORK_BUILD"), "requires the framework command")
    val error = intercept[IllegalStateException](ModelBuild.mini)
    assert(error.getMessage.contains("system integration is held"))
    val tests = new ConstructedRealizationsSuite().munitTests()
    for name <- Seq(
        "the real constructed activity mini matches its frozen oracle in two JVMs",
        "an opaque Temporal request selector refuses the constructed root and writes nothing"
      )
    do
      val test = tests.find(_.name.startsWith(name)).getOrElse(fail(s"missing test: $name"))
      assert(test.tags(munit.Ignore), s"system integration case was runnable: ${test.name}")

  test("the actual framework library has no IR packages and unexpected packages fail closed"):
    assume(sys.env.contains("UMPIRE_FRAMEWORK_BUILD"), "requires the framework command")
    assertEquals(ModelBuild.files, Nil)
    val poisoned = Files.createTempFile("framework-ir-package.", ".jar")
    val archive = java.util.jar.JarOutputStream(Files.newOutputStream(poisoned))
    try
      archive.putNextEntry(java.util.jar.JarEntry("fixture/features/unexpected/Probe.class"))
      archive.closeEntry()
    finally archive.close()
    val error = intercept[IllegalArgumentException](ModelBuild.frameworkFiles(poisoned))
    assert(error.getMessage.contains("framework library declares IR packages: unexpected"))
