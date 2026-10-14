package framework.reify

import scala.quoted.*

// Compile-time capture points. Each returns, as a string constant baked into the call site, the IR
// JSON of the functions it reifies: a JSON array of `Function` messages in ProtoJSON shape.
object Reify:
  // The step function of one `on` block, `<machine>.rules.<action>`, and every function it reaches.
  inline def rule[C](inline action: Any)(inline cases: C): String = ${ ruleImpl('action, 'cases) }

  // One def (by reference) or lambda, and every function it reaches.
  inline def function(inline f: Any): String = ${ functionImpl('f) }

  // The same, expanded during typing (`transparent`), so the presentation compiler and
  // `scala.compiletime.testing.typeCheckErrors` see its refusals.
  transparent inline def functionTransparent(inline f: Any): String = ${ functionImpl('f) }

  def ruleImpl[C](action: Expr[Any], cases: Expr[C])(using Quotes): Expr[String] =
    timed("rule") { Lifter().rule(action, cases) }

  def functionImpl(f: Expr[Any])(using Quotes): Expr[String] =
    timed("function") { Lifter().function(f) }

  // Macro expansion cost, appended per expansion so it survives the Bloop server's swallowed stderr.
  private def timed(kind: String)(body: => (String, String))(using Quotes): Expr[String] =
    val t0 = System.nanoTime()
    val (what, json) = body
    val micros = (System.nanoTime() - t0) / 1000
    val log = sys.env.getOrElse("REIFY_LOG", "/tmp/umpire-macro-spike/out/macro-times.log")
    scala.util.Try(
      java.nio.file.Files.writeString(
        java.nio.file.Path.of(log),
        s"$kind $what $micros\n",
        java.nio.file.StandardOpenOption.CREATE,
        java.nio.file.StandardOpenOption.APPEND
      )
    ): Unit
    Expr(json)
