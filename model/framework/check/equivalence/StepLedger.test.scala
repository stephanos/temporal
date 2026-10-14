package framework.check
package equivalence

import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.GZIPOutputStream
import scala.jdk.CollectionConverters.*

class StepLedgerSuite extends munit.FunSuite:
  private val old = Vector("a.first", "b.last")
  private val now = old.reverse
  private val header =
    Vector("init\tzero", "bindings\ta.first b.last", "state\tzero\tend=false")
  private val first = Vector(
    "step\ta.first[x]\tzero\t{ ok | zero | [one,two] | first ; ok | one | [] | second }",
    "step\ta.first[y]\tzero\t{ ok | one | [two] | another pair }"
  )
  private val last = Vector(
    "step\tb.last[]\tzero\t{ ok | one | [done] | last }"
  )
  private val baseline = header ++ first ++ last
  private val reordered =
    header.updated(1, "bindings\tb.last a.first") ++ last ++ first

  private def digest(rows: Vector[String]): String =
    MessageDigest
      .getInstance("SHA-256")
      .digest(rows.mkString("", "\n", "\n").getBytes(UTF_8))
      .map(b => f"${b & 0xff}%02x")
      .mkString

  private def tables(entries: (String, Vector[String])*): Path =
    val dir = Scratch.directory("step-ledger")
    val index = entries.map { (key, rows) =>
      val hash = digest(rows)
      val out =
        GZIPOutputStream(Files.newOutputStream(dir.resolve(hash + ".steps.gz")))
      try out.write(rows.mkString("", "\n", "\n").getBytes(UTF_8))
      finally out.close()
      s"$key\t$hash\t${rows.size}"
    }
    Files.writeString(
      dir.resolve("index.tsv"),
      index.mkString("", "\n", "\n")
    ): Unit
    dir

  private def ledger = StepLedger(
    Vector(
      BindingOrder(
        "old.machine",
        "new.machine",
        digest(baseline),
        baseline.size.toLong,
        old,
        now,
        Vector(1, 0)
      )
    ),
    Vector.empty
  )

  private def compare(
      rows: Vector[String],
      seal: Option[StepLedger] = Some(ledger)
  ) =
    Equivalence
      .compareTables(
        tables("old.machine" -> baseline),
        tables("new.machine" -> rows),
        RenameMap.parse("old.machine new.machine"),
        seal
      )
      .differences

  test(
    "only the named exact binding vector and complete action blocks may be permuted"
  ):
    assertEquals(compare(reordered), Vector.empty)
    assert(compare(reordered, None).nonEmpty)
    val mutants = Map(
      "fact-order" -> reordered.map(_.replace("[one,two]", "[two,one]")),
      "result-order" -> reordered.map(
        _.replace(
          "ok | zero | [one,two] | first ; ok | one | [] | second",
          "ok | one | [] | second ; ok | zero | [one,two] | first"
        )
      ),
      "reason" -> reordered.map(_.replace("| last }", "| changed }")),
      "state" -> reordered.updated(2, "state\tzero\tend=true"),
      "input" -> reordered.map(_.replace("a.first[x]", "a.first[y]")),
      "missing-block" -> reordered.dropRight(first.size),
      "split-block" -> (reordered ++ first),
      "wrong-vector" -> reordered.updated(1, "bindings\ta.first b.last"),
      "inner-pair-order" -> (reordered.take(4) ++ first.reverse),
      "missing-pair" -> reordered.dropRight(1),
      "duplicate-pair" -> (reordered :+ first.last)
    )
    for (name, rows) <- mutants do assert(compare(rows).nonEmpty, name)
    val nonBijective = ledger.copy(orders = ledger.orders.map(_.copy(permutation = Vector(0, 0))))
    val wrongDigest =
      ledger.copy(orders = ledger.orders.map(_.copy(baselineDigest = "0" * 64)))
    val wrongOldVector =
      ledger.copy(orders = ledger.orders.map(_.copy(oldBindings = now)))
    assert(compare(reordered, Some(nonBijective)).nonEmpty)
    assert(compare(reordered, Some(wrongDigest)).nonEmpty)
    assert(compare(reordered, Some(wrongOldVector)).nonEmpty)

  test("a shared digest does not lend one table's ledger to another table"):
    val result = Equivalence.compareTables(
      tables("old.machine" -> baseline, "old.other" -> baseline),
      tables("new.machine" -> reordered, "old.other" -> reordered),
      RenameMap.parse("old.machine new.machine"),
      Some(ledger)
    )
    assert(result.differences.exists(_.contains("old.other")), result)

  test("unused order seals and unrelated additions remain errors"):
    assert(compare(baseline).nonEmpty)
    val result = Equivalence.compareTables(
      tables("old.machine" -> baseline),
      tables("new.machine" -> reordered, "extra.machine" -> baseline),
      RenameMap.parse("old.machine new.machine"),
      Some(ledger)
    )
    assert(
      result.differences.exists(_.contains("extra.machine is new")),
      result
    )

  test(
    "an addition compares every byte against its independent original table"
  ):
    val original = tables("original.dispatch" -> last)
    val addition = StepAddition(
      "original.dispatch",
      "current.dispatch",
      original,
      digest(last),
      last.size.toLong,
      RenameMap.empty
    )
    val seal = StepLedger(Vector.empty, Vector(addition))
    def added(rows: Vector[String]) = Equivalence
      .compareTables(
        tables(),
        tables("current.dispatch" -> rows),
        RenameMap.empty,
        Some(seal)
      )
      .differences
    assertEquals(added(last), Vector.empty)
    assert(added(last.map(_.replace("[done]", "[other]"))).nonEmpty)
    assert(added(last :+ last.head).nonEmpty)
    assert(
      Equivalence
        .compareTables(
          tables(),
          tables("current.dispatch" -> last),
          RenameMap.empty
        )
        .differences
        .nonEmpty
    )
    assert(
      Equivalence
        .compareTables(tables(), tables(), RenameMap.empty, Some(seal))
        .differences
        .nonEmpty
    )
    val changed = last.map(_.replace("[done]", "[changed]"))
    val out = GZIPOutputStream(
      Files.newOutputStream(original.resolve(digest(last) + ".steps.gz"))
    )
    try out.write(changed.mkString("", "\n", "\n").getBytes(UTF_8))
    finally out.close()
    assert(
      added(changed).exists(_.contains("original payload digest/count changed"))
    )

  test(
    "real dispatch origin binds committed sources execution receipts and compiled members"
  ):
    val admitted = Path.of("model/testdata/equivalence/dispatch-original")
    val dir = Scratch.directory("dispatch-origin-controls")
    val copied = dir.resolve("dispatch-original")
    val stream = Files.walk(admitted)
    try
      for from <- stream.iterator.asScala do
        val to = copied.resolve(admitted.relativize(from))
        if Files.isDirectory(from) then Files.createDirectories(to): Unit
        else Files.copy(from, to): Unit
    finally stream.close()
    val map = dir.resolve("batch.map")
    val json = dir.resolve("ledger.json")
    Files.writeString(map, "# step-ledger ledger.json\n"): Unit
    val ledgerJson = Files.readString(
      Path.of("model/testdata/equivalence/batch-1d.ledger.json")
    )
    Files.writeString(json, ledgerJson): Unit
    assertEquals(StepLedger.read(map).get.additions.size, 4)
    for relative <- Vector(
        "source.sha256",
        "sources/model/project.scala",
        "classpath.txt",
        "dependencies.sha256",
        "compiled-classes.sha256",
        "compiled.jar",
        "tables/index.tsv"
      )
    do
      val path = copied.resolve(relative)
      val bytes = Files.readAllBytes(path)
      Files.write(path, bytes :+ 0.toByte): Unit
      try intercept[GateError](StepLedger.read(map))
      finally Files.write(path, bytes): Unit
    val addition = StepLedger
      .read(map)
      .get
      .additions
      .find(_.currentKey.endsWith("servingWorker"))
      .get
    val payload = copied.resolve(s"tables/${addition.originalDigest}.steps.gz")
    val input = new java.util.zip.GZIPInputStream(Files.newInputStream(payload))
    val rows = try
      new String(input.readAllBytes(), UTF_8).linesIterator.toVector
    finally input.close()
    assertEquals(rows.size.toLong, addition.originalCount)
    val mutant = rows.updated(0, rows.head + "changed")
    val out = GZIPOutputStream(Files.newOutputStream(payload))
    try out.write(mutant.mkString("", "\n", "\n").getBytes(UTF_8))
    finally out.close()
    val compared = Equivalence.compareTables(
      tables(),
      tables(addition.currentKey -> mutant.map(addition.renames.apply)),
      RenameMap.empty,
      Some(StepLedger(Vector.empty, Vector(addition)))
    )
    assert(
      compared.differences.exists(
        _.contains("original payload digest/count changed")
      ),
      compared
    )

  test("the sidecar pointer and schema fail closed"):
    val dir = Scratch.directory("step-ledger-schema")
    val map = dir.resolve("batch.map")
    val json = dir.resolve("ledger.json")
    Files.writeString(map, "# step-ledger ledger.json\n"): Unit
    val malformed = Vector(
      """{"version":2,"orders":[],"additions":[]}""",
      """{"version":1,"version":1,"orders":[],"additions":[]}""",
      """{"version":1,"orders":[],"additions":[],"unknown":true}""",
      """{"version":1,"orders":[],"additions":[{"originalKey":"x"}]}"""
    )
    for specimen <- malformed do
      Files.writeString(json, specimen): Unit
      intercept[GateError](StepLedger.read(map))
    val entry =
      """{"baselineKey":"b","currentKey":"c","baselineDigest":"x","baselineCount":1,"oldBindings":["a"],"currentBindings":["a"],"permutation":[0]}"""
    Files.writeString(
      json,
      s"""{"version":1,"orders":[$entry,$entry],"additions":[]}"""
    ): Unit
    intercept[GateError](StepLedger.read(map))
    Files.writeString(
      map,
      "# step-ledger ledger.json\n# step-ledger ledger.json\n"
    ): Unit
    intercept[GateError](StepLedger.read(map))
    Files.writeString(map, "# step-ledger ../outside.json\n"): Unit
    intercept[GateError](StepLedger.read(map))
    Files.writeString(map, "# step-ledger missing.json\n"): Unit
    intercept[GateError](StepLedger.read(map))
    Files.writeString(map, "old new\n# unrelated comment\n"): Unit
    assertEquals(StepLedger.read(map), None)

  test("full projection mode never admits a steps-only ledger"):
    val map = Path.of("model/testdata/equivalence/batch-1d.map")
    assert(StepLedger.forComparison(map, stepsOnly = true).nonEmpty)
    assertEquals(StepLedger.forComparison(map, stepsOnly = false), None)
    assert(
      compare(
        reordered,
        StepLedger.forComparison(map, stepsOnly = false)
      ).nonEmpty
    )

  test(
    "duplicate original and current table keys fail even when their payloads agree"
  ):
    for rows <- Vector(last, first) do
      val dir = tables("duplicate" -> last, "duplicate" -> rows)
      intercept[GateError](Equivalence.index(dir))
      intercept[GateError](
        Equivalence.compareTables(dir, tables(), RenameMap.empty)
      )
      intercept[GateError](
        Equivalence.compareTables(tables(), dir, RenameMap.empty)
      )
