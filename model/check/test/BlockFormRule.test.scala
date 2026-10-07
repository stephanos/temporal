package umpire.check

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}

class BlockFormRuleSuite extends munit.FunSuite:
  test("an on whose cases are in parentheses is found at its line, and a block is not"):
    val source =
      """object rules extends Rules(_.phase):
        |  on(client.pause)(when(started) ~> effects.pause)
        |  on(client.unpause) {
        |    when(paused) ~> effects.unpause
        |  }
        |  on(worker.poll, worker.respond)(
        |    when(started) ~> effects.poll
        |  )
        |  on(client.control(Control.pause)) { when(started) ~> effects.pause }
        |""".stripMargin
    assertEquals(BlockFormRule.parenthesized(source), Vector(2, 6))

  test("an on in a string or a comment, a selection's on and a derivation's block are no finding"):
    val source =
      """// on(x)(case) in a comment.
        |val text = "on(x)(case)"
        |val start = action(this).on(activity)(using given)
        |object Stuck extends Derived(Switch.rebind(on(hand.press) {
        |  always ~> Switch.effects.wear
        |}))
        |""".stripMargin
    assertEquals(BlockFormRule.parenthesized(source), Vector.empty)

  test("the gate's --check-block-form reads the Temporal Models and names each finding"):
    val root = Files.createTempDirectory("umpire-block-form-rule")
    def write(file: String, text: String): Path =
      val path = root.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(path, text)
    write("model/temporal/features/Rules.scala", "object r:\n  on(a.b)(always ~> e)\n"): Unit
    write("model/irgen/testdata/lifts/Rules.scala", "object r:\n  on(a.b)(always ~> e)\n"): Unit
    val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
    val status =
      Gate.main(
        Seq("--check-block-form"),
        Tools(root, Map.empty),
        PrintStream(out),
        PrintStream(err)
      )
    assertEquals(status, 1)
    assertEquals(
      err.toString.linesIterator.map(_.takeWhile(_ != ' ')).toVector,
      Vector("model/temporal/features/Rules.scala:2:")
    )
    assert(err.toString.contains("block-form rule: write the rule as a block"), err.toString)
