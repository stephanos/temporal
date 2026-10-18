package framework.check
package lint

import java.nio.file.Path

class BoundarySuite extends munit.FunSuite:
  // A repository of the files given, each a path under its root and its text.
  private def repository(files: (String, String)*): Path =
    val root = Scratch.directory("umpire-boundary")
    for (file, text) <- files do
      val path = root.resolve(file)
      java.nio.file.Files.createDirectories(path.getParent)
      java.nio.file.Files.writeString(path, text)
    root

  private def findings(files: (String, String)*): Vector[String] =
    Boundary.findings(repository(files*), Seq("acme", "com.acme.api"))

  test("a framework source that declares, imports or names a system's package crosses it"):
    val found = findings(
      "model/framework/export/Lift.scala" -> "package acme.lift\n",
      "model/framework/dsl/Machine.scala" ->
        "package framework.dsl\n\nimport acme.realize.Kit\nval k = _root_.acme.Bounds\n",
      "model/framework/dsl/Gaps.test.scala" ->
        "package framework.dsl\n\nval e = com.acme.api.enums.Kind.ONE\n"
    )
    assertEquals(
      found.map(_.takeWhile(_ != ' ')),
      Vector(
        "model/framework/dsl/Gaps.test.scala:3:",
        "model/framework/dsl/Machine.scala:3:",
        "model/framework/dsl/Machine.scala:4:",
        "model/framework/export/Lift.scala:1:"
      )
    )
    assert(found.forall(_.contains("boundary: the framework names `")), found)

  test(
    "comments, strings, other packages' members and the fixtures under testdata do not cross it"
  ):
    assertEquals(
      findings(
        "model/framework/export/Realizations.scala" ->
          ("package framework.`export`\n\n// acme.realize.Kit\nval kit = \"acme.realize.Kit\"\n" +
            "val ir = io.acme.ir.Model\nval acmeLike = 1\n"),
        "model/framework/export/testdata/fixture/Fixture.scala" -> "import acme.realize.Kit\n",
        "model/temporal/export/Plugin.scala" -> "package acme.`export`\n"
      ),
      Vector.empty
    )

  test(
    "a framework build that takes a system's jar or a source from outside the framework crosses it"
  ):
    val found = findings(
      "model/framework/export/project.scala" ->
        """//> using jar ../../build/ir-scalapb.jar
          |//> using jar ../../build/api-scalapb.jar
          |//> using file ../check/Tools.scala
          |//> using file ../../OutputPaths.scala
          |//> using file ../../temporal/check/Coverage.scala
          |""".stripMargin
    )
    assertEquals(
      found.map(_.takeWhile(_ != ' ')),
      Vector("model/framework/export/project.scala:2:", "model/framework/export/project.scala:5:")
    )
    assert(found.head.contains("takes ../../build/api-scalapb.jar"), found.head)
