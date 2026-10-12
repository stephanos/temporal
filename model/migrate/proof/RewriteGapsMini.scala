package umpire.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import Fn167Task4Gaps.*

// fn-167.4: the gap migration on copied mini sources (prove-mini.sh). The acceptance input is
// `gaps-mini.lint.json`: one hole, and the mini's two capability waivers with the reasons their
// except/overriding declarations state, read from the copy's own `before` lift. The hole's
// declaration is resolved here by hand; the IR-backed resolver is fn-167.7's adapter.
object RewriteGapsMini:
  private val product = "model/temporal/features/activity_mini/product/Product.scala"

  def main(args: Array[String]): Unit = args.toList match
    case List(protocol)           => rewrite(Path.of(protocol).toRealPath())
    case List("compare", scratch) => compare(Path.of(scratch).toRealPath())
    case _ => sys.error("provide the copied-mini protocol directory, or `compare <scratch>`")

  private def rewrite(protocol: Path): Unit =
    require(
      protocol != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "the source proof must not rewrite the authored mini"
    )
    val mini = protocol.getParent
    val scratch = mini.getParent.getParent.getParent.getParent
    val files = Files
      .walk(mini)
      .iterator
      .asScala
      .toSeq
      .filter(p => p.toString.endsWith(".scala") && !p.toString.endsWith(".test.scala"))
      .sortBy(_.toString)
    val sources = files.map(p => mini.relativize(p).toString -> Files.readString(p)).toMap
    val acceptance = Files.readString(Path.of("model/migrate/proof/gaps-mini.lint.json"))
    val inputs = Inputs(
      Fn167Task4GapInputs.acceptances(acceptance),
      Map(
        ("unproduced", "activityProduct", "fact statusScheduled") -> Placement(
          "temporal.features.activity.mini.product.ActivityProduct",
          "unproduced(Fact.statusScheduled)"
        )
      ),
      Map.empty,
      Fn167Task4GapInputs.waivers(
        Files.readString(scratch.resolve("before/activity-mini.waivers.json"))
      ),
      sources
    )
    val migrated = migrate(inputs).fold(p => sys.error(p.mkString("\n")), identity)
    require(migrated.inserted.size == 1, migrated.inserted.toString)
    require(
      migrated.dropped.count(_.startsWith("capability-waiver")) == 2,
      migrated.dropped.toString
    )
    // A second application converges, and a changed waiver reason refuses everything.
    val again =
      migrate(inputs.copy(sources = migrated.sources)).fold(p => sys.error(p.mkString), identity)
    require(
      again.sources == migrated.sources && again.inserted.isEmpty,
      "the migration must converge"
    )
    val reworded = inputs.waivers.map(w => w.copy(because = w.because + " (reworded)"))
    require(
      migrate(inputs.copy(waivers = reworded)).isLeft,
      "a mismatched waiver reason must refuse"
    )
    val changed = migrated.sources.filter((file, text) => sources(file) != text)
    require(changed.keySet == Set("product/Product.scala"), changed.keySet.toString)
    for (file, text) <- changed do
      val before = sources(file)
      val at = before.linesIterator.indexWhere(_.startsWith("object ActivityProduct")) + 1
      val added = text.linesIterator.size - before.linesIterator.size
      Files.writeString(mini.resolve(file), text): Unit
      Files.writeString(scratch.resolve("gap-insertion.txt"), s"$product $at $added\n"): Unit
      println(s"inserted ${migrated.inserted.head} above line $at of $file (+$added lines)")
    migrated.dropped.foreach(d => println(s"dropped $d"))

  // The after lift equals the before lift but for the positions below the inserted lines, which move
  // down by their count; the waivers are byte for byte the same.
  private def compare(scratch: Path): Unit =
    val Array(file, at, added) =
      Files.readString(scratch.resolve("gap-insertion.txt")).trim.split(' ')
    def read(stage: String, name: String) = Files.readString(scratch.resolve(s"$stage/$name"))
    val after = read("after", "activity-mini.json").linesIterator.toVector
    val line = "^(\\s*\"line\": )([0-9]+)(,?)$".r
    val restored = after.indices.map { i =>
      after(i) match
        case line(prefix, n, comma)
            if i > 0 && after(i - 1).contains(s"\"file\": \"$file\"") &&
              n.toInt >= at.toInt + added.toInt =>
          s"$prefix${n.toInt - added.toInt}$comma"
        case other => other
    }
    val moved = restored.zip(after).count(_ != _)
    require(
      restored.mkString("\n") + "\n" == read("before", "activity-mini.json"),
      "IR delta beyond moved lines"
    )
    require(
      read("after", "activity-mini.waivers.json") == read("before", "activity-mini.waivers.json")
    )
    require(moved > 0, "the insertion must move the positions below it")
    println(
      s"PASS: mini IR equal but for $moved positions moved by $added lines below $file:$at; waivers identical"
    )
