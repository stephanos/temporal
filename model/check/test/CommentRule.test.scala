package framework.check

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}

class CommentRuleSuite extends munit.FunSuite:
  // A repository of the files given, each a path under its root and its text.
  private def repository(files: (String, String)*): Path =
    val root = Files.createTempDirectory("umpire-comment-rule")
    for (file, text) <- files do
      val path = root.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(path, text)
    root

  test("block and scaladoc comments are found at the line they open on, wherever they stand"):
    val source =
      """// A line comment.
        |/** A scaladoc. */
        |val a = 1 /* after code */
        |val b = f(/* inline */ 2)
        |  /*
        |   * Across lines.
        |   */
        |""".stripMargin
    assertEquals(CommentRule.blocks(source), Vector(2, 3, 4, 5))

  test("a /* in a string, a character literal or a line comment is no block comment"):
    val source =
      """// See /* this */ and /** that */.
        |val glob = "**/*.scala"
        |val quote = '"'
        |val text = "/* not a comment */"
        |val escaped = '\''
        |val block = s\"\"\"
        |  /* still the string */
        |\"\"\"
        |""".stripMargin.replace("\\\"", "\"")
    assertEquals(CommentRule.blocks(source), Vector.empty)

  test("the gate's --check-comments reads all of model/, tests and fixtures too, not build output"):
    def run(root: Path): (Int, String) =
      val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
      val status =
        Gate.main(
          Seq("--check-comments"),
          Tools(root, Map.empty),
          PrintStream(out),
          PrintStream(err)
        )
      assertEquals(out.toString, "")
      (status, err.toString)
    val (failed, printed) = run(
      repository(
        "model/framework/Machine.scala" -> "package framework\n\n/** A machine. */\ntrait Machine\n",
        "model/irgen/testdata/lifts/Fixture.scala" -> "/* A fixture. */\nobject Fixture\n",
        "model/build/Generated.scala" -> "/** Generated. */\nobject Generated\n",
        "model/check/.scala-build/Cached.scala" -> "/* Cached. */\nobject Cached\n"
      )
    )
    assertEquals(failed, 1)
    assertEquals(
      printed.linesIterator.map(_.takeWhile(_ != ' ')).toVector,
      Vector("model/irgen/testdata/lifts/Fixture.scala:1:", "model/framework/Machine.scala:3:")
    )
    assert(printed.contains("comment rule: write the comment as // lines"), printed)
    assertEquals(
      run(repository("model/framework/Machine.scala" -> "// A machine.\ntrait M\n")),
      (0, "")
    )

  test("the repository's comments keep to the rule"):
    assertEquals(CommentRule.findings(Tools.here.directory), Vector.empty)
