package framework.fixture.expectcapture

import java.util.Base64
import framework.dsl.*
import framework.fixture.expect.*

// Prints the predicate each expectation of testdata/expect captured under the reify plugin
// (ExpectCaptureSuite): one `captured:<name>:<root>:<model>` line per Property or standalone value,
// its model the captured closure in Base64. A predicate that captured nothing fails the run.
object Captures:
  private def of(predicate: Option[AnyRef]): Captured[AnyRef] =
    val f = predicate.getOrElse(throw IllegalStateException("no predicate"))
    PredicateBindings
      .captured(f)
      .getOrElse(throw IllegalStateException(s"$f captured no predicate"))

  private def property(p: Property[?]): Captured[AnyRef] =
    of(p.decl.holds.orElse(p.decl.holds2))

  def main(args: Array[String]): Unit =
    val captured = Vector(
      "heartbeat" -> property(heartbeat),
      "finished" -> property(finished),
      "reset" -> property(reset),
      "settled" -> property(settled),
      "explicit" -> property(explicit),
      "forwarded" -> property(forwarded),
      "beatingNow" -> of(Some(beatingNow))
    )
    for (name, c) <- captured do
      // The captured callable is the very predicate the Property or value holds.
      val held = name match
        case "beatingNow" => beatingNow
        case _            =>
          val p = Map[String, Property[?]](
            "heartbeat" -> heartbeat,
            "finished" -> finished,
            "reset" -> reset,
            "settled" -> settled,
            "explicit" -> explicit,
            "forwarded" -> forwarded
          )(name)
          p.decl.holds.orElse(p.decl.holds2).get
      require(c.callable.eq(held), s"$name captured another callable")
      println(
        s"captured:$name:${c.root}:${Base64.getEncoder.encodeToString(c.model.toByteArray)}"
      )
    // The core twins are ordinary lambdas: nothing is captured for them.
    for p <- Vector(heartbeatCore, resetCore, forwardedCore) do
      require(
        PredicateBindings.captured(p.decl.holds.orElse(p.decl.holds2).get).isEmpty,
        s"${p.name} is a core lambda, yet captured a predicate"
      )
    println("captures:passed")
