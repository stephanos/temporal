package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// The DSL's public surface against REFERENCE.md: every public top-level definition of dsl/ and
// realize/, and every public extension method, is written somewhere in the reference as code or
// listed in its closing section, and every name that section lists is still public. A name counts
// as written when it appears as an identifier in a code span or block, so the check is about
// coverage, not about each entry's content.
object ReferenceSurface:
  private val modifiers =
    "(?:(?:final|abstract|sealed|open|inline|transparent|implicit|override|case|opaque|infix)\\s+)*"
  private val topLevel =
    s"^$modifiers(?:def|val|lazy val|var|class|trait|object|enum|type)\\s+`?([A-Za-z_][A-Za-z0-9_]*)".r
  private val extensionMember = s"^  ${modifiers}def\\s+`?([A-Za-z_][A-Za-z0-9_]*)".r
  private val identifier = "[A-Za-z_][A-Za-z0-9_]*".r
  private val fenced = "(?s)```.*?```".r
  private val inline = "`([^`\\n]+)`".r
  val appendixHeading = "## Public surface not in this reference"

  // The public names a source defines at the top level or as extension methods. A definition
  // starting with `private` or `protected` is not public; a given is never called by name.
  def publicNames(source: String): Vector[String] =
    val code = Literals.code(source, Literals.literals(source))
    code
      .split("\n", -1)
      .foldLeft((Vector.empty[String], false)) { case ((names, inExtension), line) =>
        if line.nonEmpty && !line.head.isWhitespace then
          (names ++ topLevel.findFirstMatchIn(line).map(_.group(1)), line.startsWith("extension"))
        else if inExtension then
          (names ++ extensionMember.findFirstMatchIn(line).map(_.group(1)), true)
        else (names, inExtension)
      }
      ._1

  // The identifiers the reference writes as code above its closing section.
  def written(reference: String): Set[String] =
    val body = reference.split(appendixHeading, 2).head
    val blocks = fenced.findAllIn(body).toVector
    val spans = inline.findAllMatchIn(fenced.replaceAllIn(body, "")).map(_.group(1)).toVector
    (blocks ++ spans).flatMap(identifier.findAllIn).toSet

  // The names the lists of the closing section hold, under its `###` headings: its introduction
  // may name code of its own.
  def listed(reference: String): Set[String] =
    reference
      .split(appendixHeading, 2)
      .lift(1)
      .flatMap(_.split("\n### ", 2).lift(1))
      .fold(Set.empty[String])(lists =>
        inline.findAllMatchIn(lists).map(_.group(1)).filter(identifier.matches).toSet
      )

class ReferenceSurfaceTest extends munit.FunSuite:
  import ReferenceSurface.*

  private val framework = Path.of("model/framework")

  private def sources: Vector[Path] =
    Vector("dsl", "realize").flatMap: directory =>
      Files
        .list(framework.resolve(directory))
        .iterator()
        .asScala
        .filter(path => path.toString.endsWith(".scala") && !path.toString.endsWith(".test.scala"))
        .toVector
        .sortBy(_.toString)

  test("every public DSL definition is written in REFERENCE.md or listed in its closing section"):
    val reference = Files.readString(framework.resolve("REFERENCE.md"))
    val documented = written(reference) ++ listed(reference)
    val missing = sources.flatMap: path =>
      publicNames(Files.readString(path))
        .filterNot(documented)
        .map(name => s"$name ($path)")
    assertEquals(missing.distinct, Vector.empty)

  test("every name the closing section of REFERENCE.md lists is still public"):
    val reference = Files.readString(framework.resolve("REFERENCE.md"))
    val public = sources.flatMap(path => publicNames(Files.readString(path))).toSet
    assertEquals(listed(reference).filterNot(public).toVector.sorted, Vector.empty)

  test("public names are top-level and extension definitions, outside comments and strings"):
    val source =
      """package p
        |
        |// def commented = 1
        |final case class Shown(x: Int)
        |private[framework] object Hidden
        |protected def guarded = 1
        |inline def word(x: Int): Int = x
        |val text = "def quoted = 2"
        |given Ordering[Shown] = Ordering.by(_.x)
        |object Holder:
        |  def member = 1
        |extension (s: Shown)
        |  def widened: Int = s.x
        |  private def hidden: Int = 0
        |""".stripMargin
    assertEquals(publicNames(source), Vector("Shown", "word", "text", "Holder", "widened"))

  test("written reads code spans and blocks above the closing section, listed reads below it"):
    val reference =
      """Use `whenever(p): eventually(q)` here.
        |
        |```scala
        |object transitions extends Transitions
        |```
        |
        |## Public surface not in this reference
        |
        |Checked by `ReferenceSurfaceTest`.
        |
        |### Types
        |
        |`Owner` `Declaring` and prose
        |""".stripMargin
    assert(Set("whenever", "eventually", "Transitions").subsetOf(written(reference)))
    assert(!written(reference).contains("Owner"))
    assertEquals(listed(reference), Set("Owner", "Declaring"))
