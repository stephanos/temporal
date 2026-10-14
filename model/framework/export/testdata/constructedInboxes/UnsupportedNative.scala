package framework.inboxunsupported

import framework.dsl.*
import fixture.inboxunsupported.*
import fixture.inboxunsupported.given

object UnsupportedNative:
  def main(args: Array[String]): Unit =
    for (name, domain) <- Seq(
        "Forwarded" -> summon[Finite[Forwarded]],
        "Aliased" -> summon[Finite[Aliased]]
      )
    do
      try
        Signature.stateSchema(domain, Some(s"fixture.inboxunsupported.$name"), s"state $name"): Unit
        throw AssertionError(s"$name guessed channel ownership")
      catch
        case error: IllegalArgumentException =>
          assert(error.getMessage.contains(s"state $name"))
          assert(error.getMessage.contains("ownership"))
    println("unsupported:passed")
