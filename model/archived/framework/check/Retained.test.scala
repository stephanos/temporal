package framework.check

import java.nio.file.{Files, Path}
import java.nio.file.attribute.FileTime
import scala.jdk.CollectionConverters.*

class RetainedSuite extends munit.FunSuite:
  private def names(parent: Path): Set[String] =
    val stream = Files.list(parent)
    try stream.iterator.asScala.map(_.getFileName.toString).toSet
    finally stream.close()

  private def aged(directory: Path, minutes: Long): Path =
    Files.setLastModifiedTime(
      directory,
      FileTime.fromMillis(System.currentTimeMillis - minutes * 60 * 1000)
    )

  test("a run that passes deletes its own scratch, and one that fails keeps it, marked"):
    val parent = Scratch.directory("umpire-retained")
    val passed = Retained.directory(parent, "passed.")
    Files.writeString(passed.resolve("lift.json"), "{}"): Unit
    Retained.finish(passed = true, Seq(passed))
    assert(!Files.exists(passed))
    val failed = Retained.directory(parent, "failed.")
    Retained.finish(passed = false, Seq(failed))
    assert(Files.isRegularFile(failed.resolveSibling(s"${failed.getFileName}.failed")))
    assertEquals(names(failed), Set.empty[String])

  test("a failing run keeps the newest failed runs, never a recent one or another directory"):
    val parent = Scratch.directory("umpire-retained-prune")
    val earlier = (1 to 12).map: i =>
      val d = Retained.directory(parent, f"earlier$i%02d.")
      Files.writeString(d.resolveSibling(s"${d.getFileName}.failed"), ""): Unit
      aged(d, 120L + i)
    val running = Retained.directory(parent, "running.")
    val evidence = Files.createDirectories(parent.resolve("evidence"))
    aged(evidence, 600)
    val own = Retained.directory(parent, "own.")
    Retained.finish(passed = false, Seq(own))
    val left = names(parent)
    // The own failure and the running one count among the ten; the eight newest earlier ones stay.
    assertEquals(
      earlier.map(_.getFileName.toString).filter(left).toSet,
      earlier.take(8).map(_.getFileName.toString).toSet
    )
    // A pruned run's marks go with it.
    assert(!left(s"${earlier.last.getFileName}.scratch"))
    assert(left(own.getFileName.toString) && left(running.getFileName.toString))
    assert(left("evidence"))
