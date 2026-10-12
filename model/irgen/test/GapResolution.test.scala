package umpire.irgen

import java.nio.file.{Files, Path}
import java.util.zip.ZipFile
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.tasty.inspector.TastyInspector
import umpire.check.{Scratch, Tools}

// fn-167.2: `@gap` resolved to the lint's finding keys by placement, with duplicate, typed-reference
// and expiry refusals, and nothing of it written: the IR and the waivers file of the fixture with its
// gaps are byte for byte those of the fixture with every gap blanked out.
class GapResolutionSuite extends munit.FunSuite:
  override val munitTimeout = 15.minutes
  private val tools = Tools.here
  private lazy val scratch = Scratch.directory("umpire-gap-resolution")
  private val fixture = tools.directory.resolve("model/irgen/testdata/gaps/resolution")
  private val registry = SpecRegistry.of(
    "fn-167-gaps-declared-in-the-model-gap-replaces",
    "fn-5-first",
    "fn-5-second"
  )

  // The framework and the Temporal vocabulary the fixture reads, as their own jar, so the lift reads
  // their TASTy without the features.
  private lazy val scoped: (Path, Path) =
    val roots = Seq(
      "model/project.scala",
      "model/framework",
      "model/temporal/Bounds.scala",
      "model/temporal/capabilities",
      "model/temporal/realize"
    )
    val jar = scratch.resolve("scoped.jar")
    tools
      .scalaCli(
        Seq("--power", "package", "--server=false", "--library") ++ roots ++
          Seq("-f", "-o", jar.toString)
      )
      .orFail(): Unit
    val printed = tools
      .scalaCli(Seq("compile", "--server=false", "--print-class-path") ++ roots)
      .orFail()
      .diagnostics
      .trim
      .split("\n")
      .last
    val classpath = scratch.resolve("scoped.classpath")
    Files.writeString(classpath, s"$jar:${printed.substring(printed.indexOf(':') + 1)}"): Unit
    (jar, classpath)

  private def packaged(name: String, sources: Map[String, String]): (Path, Path) =
    val dir = Files.createDirectory(scratch.resolve(name))
    for (file, text) <- sources do
      Files.createDirectories(dir.resolve(file).getParent): Unit
      Files.writeString(dir.resolve(file), text): Unit
    Files.writeString(
      dir.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = dir.resolve("fixture.jar")
    tools
      .inDirectory(dir)
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          Files.readString(scoped._2).trim,
          ".",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    (dir, jar)

  private def source(file: String) = Files.readString(fixture.resolve(file))

  private lazy val resolution = packaged(
    "resolution",
    Seq("Resolution.scala", "Invalid.scala", "Derivation.scala").map(f => f -> source(f)).toMap
  )
  private lazy val gapfile = packaged(
    "gapfile",
    Map("Gapfile.scala" -> source("Gapfile.scala"), "orphan/Orphan.scala" -> source("Orphan.scala"))
  )

  // Every `@gap(...)(...)` annotation blanked out, its lines kept, so every position stays put.
  private def blanked(text: String): String =
    val out = StringBuilder()
    var i = 0 // scalafix:ok DisableSyntax.var
    while i < text.length do
      if text.startsWith("@gap(", i) then
        var depth = 0 // scalafix:ok DisableSyntax.var
        var lists = 0 // scalafix:ok DisableSyntax.var
        var j = i + 4 // scalafix:ok DisableSyntax.var
        while lists < 2 do
          text(j) match
            case '(' => depth += 1
            case ')' =>
              depth -= 1
              if depth == 0 then lists += 1
            case _ => ()
          j += 1
        out ++= text.substring(i, j).map(c => if c == '\n' then '\n' else ' ')
        i = j
      else
        out += text(i)
        i += 1
    out.toString

  private lazy val unannotated = packaged(
    "unannotated",
    Seq("Resolution.scala", "Invalid.scala", "Derivation.scala")
      .map(f => f -> blanked(source(f)))
      .toMap
  )

  private def root(name: String) = s"fixture.gapresolution.$name"
  private val valid = Seq(
    "Worker",
    "Worker$.queries",
    "Plain",
    "Plain$.queries",
    "Plain$.capabilities",
    "Pair",
    "runs$.one",
    "runs$.two"
  ).map(root)

  // fn-167.3: machines derived from Plain by each operation (Derivation.scala).
  private val derived = Seq(
    "PlainAssumed",
    "PlainUnmonitored",
    "PlainRebound",
    "PlainGuarded",
    "PlainRestricted",
    "PlainExtended",
    "PlainRefined"
  ).map(root)

  // A lift of the roots in this JVM, with the registry the test gives, as `lift` reads its jars.
  private def lifted(built: (Path, Path), roots: Seq[String]): Lifter =
    liftedAs(built, Target.Roots(roots))

  private def liftedAs(built: (Path, Path), target: Target): Lifter =
    val (dir, jar) = built
    val tastyDir = Files.createTempDirectory(scratch, "tasty")
    val prefixes = scala.collection.mutable.Map.empty[String, String]
    val tastys = Seq(jar -> s"$dir/", scoped._1 -> "model/").zipWithIndex.flatMap {
      case ((from, prefix), i) =>
        val zip = ZipFile(from.toFile)
        try
          zip.entries.asScala
            .filter(e => e.getName.endsWith(".tasty") && !e.getName.startsWith("framework/"))
            .map { e =>
              val entry = s"$i/${e.getName}"
              val path = tastyDir.resolve(entry)
              Files.createDirectories(path.getParent): Unit
              val input = zip.getInputStream(e)
              try Files.copy(input, path): Unit
              finally input.close()
              prefixes(entry) = prefix
              path.toString
            }
            .toList
        finally zip.close()
    }.sorted
    val classpath = jar.toString +: Files.readString(scoped._2).trim.split(':').toList
    val lifter = Lifter(target, prefixes.toMap, () => registry)
    assert(TastyInspector.inspectAllTastyFiles(tastys.toList, Nil, classpath)(lifter))
    lifter

  private def line(file: String, needle: String): Int =
    Files.readString(fixture.resolve(file)).linesIterator.indexWhere(_.contains(needle)) + 1

  test("every supported placement resolves to the lint's owner and subject"):
    val lifter = lifted(resolution, valid)
    assertEquals(lifter.errors.toList, Nil)
    val gaps = lifter.gaps("")
    val keys = gaps.map(g => (g.kind, g.owner, g.subject, g.exact)).toList
    val expected = List(
      // Facets: owned by the one machine that mixes each in.
      ("never-enabled", "worker", "move-slow", true),
      ("never-enabled", "worker", "finish", true),
      ("witness-only", "worker", "moves", true),
      ("unasked-property", "worker", "moves", true),
      // The machine's table kinds: an action stands for its every class, phases in catalog order.
      ("silent-rejection", "worker", "move-fast in busy, done", true),
      ("silent-rejection", "worker", "move-slow in busy, done", true),
      ("disabled-by-default", "worker", "finish in idle", true),
      ("must-not-pinned", "worker", "move-fast in busy", true),
      ("unconstrained-result", "worker", "move-fast", true),
      ("unproduced", "worker", "outcome refused", true),
      ("unproduced", "worker", "fact moved", true),
      ("untaken-choice", "worker", "held", true),
      ("unreachable-value", "worker", "stage=done", true),
      ("stuck-state", "worker", "busy-true", true),
      // Realized kinds declared on the machine aggregate under it.
      ("unevidenced-fact", "worker", "moved", true),
      ("unread-observation", "worker", "named-info", true),
      // Query subjects: verify by its Property's machine, find by its Scenario's.
      ("unfired-verify", "worker", "movesVerified", true),
      ("unrealized-find", "worker", "movesFound", true),
      ("unread-refinement", "plain", "plain refines worker", true),
      ("stuck-state", "pair", "busy-false-idle-false", true),
      // A realization owns its binding kinds, its machine its evidence kinds.
      ("unperformed-action", "worker", "finish", true),
      ("uncovered-class", "one", "move-fast", true),
      ("uncovered-class", "one", "move-slow", true),
      (
        "unmodeled-api-value",
        "worker",
        "io.temporal.api.enums.v1.ActivityExecutionStatus ACTIVITY_EXECUTION_STATUS_RUNNING",
        false
      ),
      ("explicit-wait", "worker", "fixture.gapresolution.readInfo", false),
      ("uncovered-class", "two", "move-slow", true),
      ("unreachable-binding", "two", "move-fast", true),
      // The capability waiver, synthesized from `except`.
      ("capability-waiver", "plain", "plain.terminalStatesAreFinal", true)
    )
    assertEquals(
      keys.sortBy(_.toString).mkString("\n"),
      expected.sortBy(_.toString).mkString("\n")
    )
    val silent = gaps.find(_.kind == "silent-rejection").get
    assertEquals(silent.because, "silent reason")
    assertEquals(silent.until, "fn-167 R2")
    assertEquals(silent.expires, Some("fn-167-gaps-declared-in-the-model-gap-replaces"))
    assertEquals(gaps.find(_.kind == "unconstrained-result").get.until, "fn-167")
    assertEquals(gaps.find(_.kind == "unproduced").get.expires, silent.expires)
    // An omitted `until` is empty and expires with nothing.
    assertEquals(gaps.find(_.kind == "disabled-by-default").get.expires, None)
    val at = (g: ResolvedGap) => s"${g.position.file.split('/').last}:${g.position.line}"
    assertEquals(at(silent), s"Resolution.scala:${line("Resolution.scala", "@gap(silent(")}")
    // Each subject's own position, distinct from its annotation's when written apart.
    val mustNot = gaps.find(_.kind == "must-not-pinned").get
    assertEquals(mustNot.subjectAt.line, line("Resolution.scala", "mustNotPinned(caller"))
    assertEquals(mustNot.position.line, line("Resolution.scala", "byDefault(caller.finish") - 1)
    assert(mustNot.subjectAt.line > mustNot.position.line)
    assertEquals(
      gaps.filter(_.declaration == "fixture.gapresolution.Moving").map(_.kind),
      Seq("never-enabled")
    )
    val waiver = gaps.find(_.kind == "capability-waiver").get
    assert(waiver.synthesized)
    assertEquals(waiver.because, "a fixture's waiver reason")
    assertEquals(gaps.count(_.synthesized), 1)
    // The written IR's waivers are what they were.
    assertEquals(
      org.json4s.jackson.JsonMethods.compact(lifter.waivers("")),
      """{"machines":["plain"],"waivers":[{"machine":"plain","subject":"plain.terminalStatesAreFinal","because":"a fixture's waiver reason"}]}"""
    )

  test("the IR and the waivers are those of the fixture without its gaps"):
    def written(built: (Path, Path), name: String): (String, String) =
      val (dir, jar) = built
      val out = dir.resolve(s"$name.json")
      val ran = InProcess.lift(
        tools.directory,
        Seq(s"$jar=$dir/,${scoped._1}=model/", scoped._2.toString, out.toString) ++ valid
      )
      assertEquals(ran.exit, 0, ran.diagnostics)
      (Files.readString(out), Files.readString(dir.resolve(s"$name.waivers.json")))
    val (withGaps, withWaivers) = written(resolution, "with")
    val (without, withoutWaivers) = written(unannotated, "without")
    // The fixtures are copied to directories of their own names, which positions name.
    assertEquals(
      withGaps.replace("/resolution/", "/<fixture>/"),
      without.replace("/unannotated/", "/<fixture>/")
    )
    assertEquals(withWaivers, withoutWaivers)
    assert(withGaps.contains("\"realizations\""))
    assert(!withGaps.contains("silent reason"))

  test("typed references that resolve to nothing, wrong placements and duplicates are refused"):
    val lifter = lifted(
      resolution,
      Seq("Unbound", "Twice", "Expiring", "Asking$.queries", "first", "second")
        .map(n => root(s"invalid$$.$n"))
    )
    val errors = lifter.errors.map(_._2.getMessage).toList
    def refused(needle: String, fragment: String) =
      val at = s"Invalid.scala:${line("Invalid.scala", needle)}:"
      assert(
        errors.exists(e => e.contains(at) && e.contains(fragment)),
        s"expected $at ... $fragment in\n${errors.mkString("\n")}"
      )
    refused("@gap(neverEnabled(caller.finish))", "finish is no action unbound binds")
    refused("@gap(silent(caller.move, in = Mode.fast))", "is no value of")
    refused("@gap(silent[Stage](caller.move))", "name the phases of unbound")
    refused("@gap(uncovered(caller.move))", "uncovered does not apply to the machine unbound")
    refused("@gap(unevidenced(Fact.moved))", "no realization of unbound is lifted")
    refused("@gap(unproduced(Mode.fast))", "is no outcome or fact of unbound")
    refused("@gap(unreachable[Mode]", "is held by no field of unbound's state")
    refused("@gap(unconstrained(caller.move(Mode.fast)))", "is declared by the gap at")
    refused(
      "because = \"second run\"",
      "unperformed-action worker \"finish\" is declared by the gap at"
    )
    refused("@gap(unrealized)(because = \"a verify", "unrealized reads a find Query")
    refused("@gap(witnessOnly)", "witnessOnly does not apply to the Query checked")
    refused("until = \"fn-999999\"", "no Flow spec has that id")
    refused("until = \"fn-5\"", "ambiguous among fn-5-first, fn-5-second")
    refused("until = \"soon\"", "no Flow spec id")
    assertEquals(lifter.models.get("").map(_ => ()), Some(()))

  test("an IR file run refuses a gap on a declaration no IR file lifts"):
    val lifter = liftedAs(gapfile, Target.IrFiles(Nil))
    val errors = lifter.errors.map(_._2.getMessage).toList
    val orphan = line("Orphan.scala", "nothing lifts it")
    assert(
      errors.exists(e =>
        e.contains(s"Orphan.scala:$orphan:") && e.contains("fixture.gaporphan.extras.orphan") &&
          e.contains("no IR file lifts")
      ),
      errors.mkString("\n")
    )
    assertEquals(errors.size, 1, errors.mkString("\n"))
    assertEquals(
      lifter.gaps("gapfile").map(g => (g.kind, g.owner, g.subject)),
      Seq(("unasked-property", "gapfile", "lit"))
    )

  test("each derivation operation and chain records its provenance without changing the IR"):
    val lifter = lifted(resolution, valid ++ derived ++ Seq(root("refused$.Chained")))
    // The chained machine's own gap is refused (below); everything else lifts.
    val errors = lifter.errors.map(_._2.getMessage).toList
    assertEquals(errors.size, 1, errors.mkString("\n"))
    val byName = lifter.derivations("").map(p => p.machine -> p).toMap
    def op(machine: String) =
      val p = byName(machine)
      assertEquals(p.source, "plain", machine)
      assertEquals(p.ops.size, 1, machine)
      p.ops.head
    val assumed = op("plainAssumed")
    assertEquals(assumed.op, "assuming")
    assertEquals(assumed.kept, Seq("move"))
    assertEquals(assumed.addedAssumptions, Seq("fixture.gapresolution.other"))
    assert(assumed.rebound.isEmpty && assumed.droppedMonitors.isEmpty && assumed.added.isEmpty)
    val unmonitored = op("plainUnmonitored")
    assertEquals(unmonitored.droppedMonitors, Seq("fixture.gapresolution.flaggedOnce"))
    assertEquals(unmonitored.droppedRefinement, Some("worker"))
    assertEquals(unmonitored.kept, Seq("move"))
    assertEquals(op("plainRebound").rebound, Seq("move"))
    assertEquals(op("plainGuarded").rebound, Seq("move"))
    val restricted = op("plainRestricted")
    assertEquals(restricted.kept, Seq("move"))
    assertEquals(restricted.droppedRefinement, Some("worker"))
    val extended = op("plainExtended")
    assertEquals((extended.kept, extended.added), (Seq("move"), Seq("finish")))
    assertEquals(op("plainRefined").replacedRefinement, Some("worker" -> "shadow"))
    // A chain records its operations in the order applied and its source's own origins.
    val chained = byName("chained")
    assertEquals(chained.source, "plainAssumed")
    assertEquals(chained.origins, Seq("plainAssumed", "plain"))
    assertEquals(chained.ops.map(_.op), Seq("unmonitored"))
    assertEquals(byName("plainAssumed").origins, Seq("plain"))
    // Gaps the lift keeps on derived machines, with why each may be the derived machine's own.
    val kept = lifter
      .gaps("")
      .filter(_.inheritance.nonEmpty)
      .map(g => (g.owner, g.kind, g.subject, g.inheritance.takeWhile(_ != ':')))
      .sortBy(_.toString)
    assertEquals(
      kept.mkString("\n"),
      Seq(
        ("plainExtended", "never-enabled", "finish", "local"),
        ("plainExtended", "never-enabled", "move-slow", "unproven"),
        ("plainGuarded", "never-enabled", "move-fast", "local"),
        ("plainRebound", "never-enabled", "move-fast", "local"),
        ("plainRefined", "unread-refinement", "plainRefined refines shadow", "unproven"),
        ("plainRestricted", "never-enabled", "move-fast", "unproven"),
        ("plainUnmonitored", "stuck-state", "busy-true", "unproven"),
        ("plainUnmonitored", "unconstrained-result", "move-slow", "unproven")
      ).sortBy(_.toString).mkString("\n")
    )
    // The provenance changes nothing written: the derived machines' IR is their gap-free twin's.
    def written(built: (Path, Path)): String =
      val (dir, jar) = built
      val out = dir.resolve("derived.json")
      val ran = InProcess.lift(
        tools.directory,
        Seq(s"$jar=$dir/,${scoped._1}=model/", scoped._2.toString, out.toString) ++ valid ++
          derived
      )
      assertEquals(ran.exit, 0, ran.diagnostics)
      Files.readString(out).replace(s"$dir/", "<fixture>/")
    assertEquals(written(resolution), written(unannotated))

  test("a gap on a derived machine for a provably inherited finding is refused naming the base"):
    val lifter = lifted(
      resolution,
      valid ++ Seq("Assumed", "Unmonitored", "Chained").map(n => root(s"refused$$.$n"))
    )
    val errors = lifter.errors.map(_._2.getMessage).toList
    def refused(needle: String, fragment: String) =
      val at = s"Derivation.scala:${line("Derivation.scala", needle)}:"
      assert(
        errors.exists(e => e.contains(at) && e.contains(fragment)),
        s"expected $at ... $fragment in\n${errors.mkString("\n")}"
      )
    refused(
      "assumed again: inherited",
      "never-enabled assumed \"move-slow\" is plain's finding, which assumed inherits unchanged " +
        "through assuming: declare the gap on plain"
    )
    refused("unmonitored: rows kept", "is plain's finding")
    refused(
      "chained: inherited",
      "unproduced chained \"outcome refused\" is plain's finding, which chained inherits " +
        "unchanged through unmonitored, assuming: declare the gap on plain"
    )
    assertEquals(errors.size, 3, errors.mkString("\n"))
