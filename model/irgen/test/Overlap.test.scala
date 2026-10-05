package umpire.irgen

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

/**
 * The lifter's fixtures hold Models of their own (fn-114.10). A fixture that copies a live Model's
 * text drifts whenever that Model changes and tests again what the gate already holds: it lifts every
 * live Model under model/temporal into model/ir and compares it byte for byte. So no fixture under
 * testdata may share more than `threshold` substantive lines with the files under model/temporal.
 *
 * A substantive line is one with code on it: not blank, not a comment, not an import or a package
 * clause, and not punctuation alone; whitespace inside it is collapsed. A fixture line counts as
 * shared when it lies in a run of at least `run` consecutive substantive lines that also stand
 * consecutively in one model/temporal file. Shorter runs are the vocabulary every Model writes the
 * same way, such as an `enum Outcome derives Finite:` with its `case accepted`. The threshold is
 * Scripts.scala's: its core records spell the six correlation bounds the Temporal kit's helper
 * produces, which is what the fixture compares the helper with.
 */
class Overlap extends munit.FunSuite:
  private val root = Tools.here.directory
  private val run = 4
  private val threshold = 6

  private def scalaFiles(directory: Path): Seq[Path] =
    val stream = Files.walk(directory)
    try stream.iterator.asScala.filter(_.toString.endsWith(".scala")).toList.sorted
    finally stream.close()

  private val punctuation = """[\s(){}\[\],;:=>.]*""".r

  /** The substantive lines of a file, each with its line number. */
  private def substantive(file: Path): Vector[(String, Int)] =
    Files
      .readAllLines(file)
      .asScala
      .zipWithIndex
      .flatMap { (line, index) =>
        val s = line.trim
        val skipped = s.isEmpty || s.startsWith("//") || s.startsWith("/*") || s.startsWith("*") ||
          s.startsWith("import ") || s.startsWith("package ") || punctuation.matches(s)
        Option.when(!skipped)((s.replaceAll("""\s+""", " "), index + 1))
      }
      .toVector

  private def runs(lines: Vector[String]): Iterator[Vector[String]] =
    lines.sliding(run).filter(_.size == run)

  /** For each fixture with shared lines, the numbers of those lines, beside the files they are in. */
  private def shared(fixtures: Seq[Path], live: Seq[Path]): Map[Path, (Seq[Int], Set[Path])] =
    val owners = live.flatMap(f => runs(substantive(f).map(_._1)).map(_ -> f)).groupMap(_._1)(_._2)
    fixtures.flatMap { fixture =>
      val lines = substantive(fixture)
      val hits = lines.indices
        .sliding(run)
        .filter(_.size == run)
        .flatMap { window =>
          owners.get(window.map(lines(_)._1).toVector).map(window -> _)
        }
        .toSeq
      val numbers = hits.flatMap(_._1).distinct.sorted.map(lines(_)._2)
      Option.when(numbers.nonEmpty)(fixture -> (numbers, hits.flatMap(_._2).toSet))
    }.toMap

  test("no lifter fixture repeats more than the threshold of a live Model's lines"):
    val live = scalaFiles(root.resolve("model/temporal"))
    val fixtures = scalaFiles(root.resolve("model/irgen/testdata"))
    assert(live.nonEmpty && fixtures.nonEmpty)
    val over = shared(fixtures, live).collect {
      case (fixture, (lines, files)) if lines.size > threshold =>
        s"${root.relativize(fixture)} shares ${lines.size} lines (${lines.mkString(", ")}) with " +
          files.map(root.relativize).toSeq.sorted.mkString(", ")
    }
    assert(over.isEmpty, over.toSeq.sorted.mkString("\n"))

  test("a copied run is counted, and a run shorter than the window is not"):
    val scratch =
      Files.createTempDirectory(
        Files.createDirectories(root.resolve("model/build/history")),
        "overlap."
      )
    def write(name: String, text: String) = Files.writeString(scratch.resolve(name), text)
    val model = write(
      "Model.scala",
      "package live\n\nval a = 1\nval b = 2\nval c = 3\n// note\nval d = 4\nval e = 5\n"
    )
    val copied =
      write("Copied.scala", "package fixture\n\nval a = 1\nval b = 2\n\nval c = 3\nval d = 4\n")
    val short = write("Short.scala", "val a = 1\nval b = 2\nval c = 3\nval x = 9\n")
    val found = shared(Seq(copied, short), Seq(model))
    assertEquals(found.keySet, Set(copied))
    assertEquals(found(copied), (Seq(3, 4, 6, 7), Set(model)))
