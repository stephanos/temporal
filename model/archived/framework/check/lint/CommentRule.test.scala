package framework.check
package lint

class CommentRuleSuite extends munit.FunSuite:
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

  test("the repository's comments keep to the rule"):
    assertEquals(CommentRule.findings(Tools.here.directory), Vector.empty)
