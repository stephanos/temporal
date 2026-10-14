package framework.check
package lint

class BlockFormRuleSuite extends munit.FunSuite:
  test("an on whose cases are in parentheses is found at its line, and a block is not"):
    val source =
      """object rules extends Rules:
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
