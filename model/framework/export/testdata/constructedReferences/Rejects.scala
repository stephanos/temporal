package framework.references

import fixture.composedprogressrejects.{mixed, oneSided, paired, spread, unpaired}

object Rejects:
  private def refuses(expected: String)(body: => Any): Unit =
    try
      body: Unit
      throw new AssertionError(s"accepted $expected")
    catch
      case error: IllegalArgumentException =>
        assert(error.getMessage.contains("native refusal"), error.getMessage)
        assert(error.getMessage.contains(expected), error.getMessage)
  def main(args: Array[String]): Unit =
    assert(mixed.fairness.size == 1)
    assert(mixed.composed.size == 1)
    assert(spread.composed.map(_.toIR("native refusal").composition) == List("pipeline", "mirror"))
    refuses("takes inputs")(oneSided.composed.head.toIR("native refusal"))
    refuses("only with its pair")(paired.composed.head.toIR("native refusal"))
    refuses("no sync")(unpaired.composed.head.toIR("native refusal"))
