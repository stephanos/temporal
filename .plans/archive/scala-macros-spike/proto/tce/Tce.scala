//> using scala 3.9.0
//> using jar ../reify.jar
package tce
import scala.compiletime.testing.typeCheckErrors

// Whether a refusal is visible to a typer-only check (what munit tests and the presentation
// compiler see): a plain inline macro expands after pickling, a transparent one while typing.
@main def tce(): Unit =
  val plain = typeCheckErrors("""framework.reify.Reify.function((s: Int) => { var x = s; x })""")
  val transparent = typeCheckErrors("""framework.reify.Reify.functionTransparent((s: Int) => { var x = s; x })""")
  println(s"plain inline:       ${plain.map(e => s"line ${e.lineContent.trim} col ${e.column}: ${e.message}")}")
  println(s"transparent inline: ${transparent.map(e => s"col ${e.column}: ${e.message}")}")
