package framework.check
package equivalence

import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import java.util.zip.GZIPOutputStream

class InterpreterTablesSuite extends munit.FunSuite:
  private val root = Scratch.directory("umpire-interpreter-fixtures")
  private val original = Vector(
    """{"kind":"header","owner":"choice","subject_kind":"machine","artifact":"fixture.json","states":["a","b"],"actions":["go"],"outcomes":["ok","no"],"facts":["first","second"],"starts":["a"],"ends":["b"],"declaration":{"steps":["go","wait"]}}""",
    """{"kind":"binding","index":0,"declaration":"go"}""",
    """{"kind":"binding","index":1,"declaration":"wait"}""",
    """{"kind":"row","source":"a","action":"go","results":[{"outcome":"ok","state":"b","facts":["first","second"],"because":"why"},{"outcome":"no","state":"a","facts":[],"because":"held"}]}""",
    """{"kind":"disabled","pair":[1,0]}""",
    """{"kind":"footer","status":"complete","states":2,"actions":1,"pairs":2,"enabled":1,"disabled":1,"unknown":0,"results":2}"""
  )
  private val pin = Json.fields(
    Json.parse(
      """{"artifact":"fixture.json","kind":"machine","owner":"choice","states":2,"actions":1,"state_action_pairs":2,"enabled_pairs":1,"disabled_pairs":1,"unknown_pairs":0,"results":2,"lines":6,"status":"complete","uncompressed_jsonl_sha256":"4d58f0f1b2f41eaef9a89cf7a5e91a53308e5a82b6fcdcd6c5f73a01b786ce7a"}"""
    ),
    "pin"
  )

  private def write(path: Path, lines: Seq[String]): Path =
    Files.createDirectories(path.getParent): Unit
    val out = GZIPOutputStream(Files.newOutputStream(path))
    try lines.foreach(line => out.write((line + "\n").getBytes(UTF_8)))
    finally out.close()
    path

  private def stream(lines: Seq[String]): Path =
    write(Files.createTempDirectory(root, "stream").resolve("machine-choice.jsonl.gz"), lines)

  private def dumps(artifacts: (String, Seq[String])*): Path =
    val folder = Files.createTempDirectory(root, "dumps")
    artifacts.foreach((name, lines) =>
      write(folder.resolve(s"$name/machine-choice.jsonl.gz"), lines)
    )
    folder

  private def change(lines: Vector[String], index: Int)(f: Map[String, Any] => Map[String, Any]) =
    lines.updated(index, Equivalence.canonical(f(Json.fields(Json.parse(lines(index)), "record"))))

  for directory <- sys.env.get("INTERPRETER_PARITY") do
    test("the Scala comparator matches Python on the same fixture streams"):
      val manifest = Json.fields(
        Json.parse(Files.readString(Path.of(directory).resolve("manifest.json"))),
        "parity manifest"
      )
      for value <- Json.items(manifest("cases"), "parity cases") do
        val entry = Json.fields(value, "parity case")
        def path(field: String) = Path.of(Json.text(entry(field), field))
        val result = InterpreterTables.observe(() =>
          Json.text(entry("operation"), "operation") match
            case "verify" =>
              InterpreterTables.verifyStream(path("path"), Json.fields(entry("pin"), "pin"))
              Map("status" -> "complete")
            case "compare" =>
              val map = RenameMap.parse(Json.text(entry("map"), "map"))
              val found = InterpreterTables.compareStreams(path("baseline"), path("current"), map)
              Map(
                "status" -> (if found.differences.isEmpty then "equivalent" else "different"),
                "used" -> map.used.toVector.sorted,
                "used_removals" -> found.removalsUsed.toVector.sorted
              )
            case "artifact" =>
              InterpreterTables.verifyArtifact(
                path("folder"),
                Json.fields(entry("pin"), "pin"),
                path("ir")
              )
              Map("status" -> "complete")
            case "replay" =>
              InterpreterTables.replay(path("ir"), path("dumper"), path("out"), path("pins"))
            case other => throw GateError("unknown parity operation " + other)
        )
        assertEquals(result("status"), entry("status"), entry("name").toString)
        for field <- Vector("used", "used_removals") if entry.contains(field) do
          assertEquals(result(field), entry(field), entry("name").toString)

  test("the Python fixture's exact bytes match its historical pin"):
    assertEquals(InterpreterTables.verifyStream(stream(original), pin)("records"), BigDecimal(6))

  test("every Python fixture mutation fails the complete ordered pin"):
    val mutants = Vector(
      "row" -> original.updated(3, original(3).replace("\"source\":\"a\"", "\"source\":\"b\"")),
      "result" -> original
        .updated(3, original(3).replace("\"outcome\":\"ok\"", "\"outcome\":\"no\"")),
      "result-order" -> change(original, 3)(row =>
        row.updated("results", Json.items(row("results"), "results").reverse)
      ),
      "fact-order" -> original
        .updated(3, original(3).replace("[\"first\",\"second\"]", "[\"second\",\"first\"]")),
      "binding-order" -> original.updated(1, original(2)).updated(2, original(1)),
      "explanation" -> original.updated(3, original(3).replace("\"why\"", "\"other\"")),
      "omitted-pair" -> original.patch(4, Nil, 1)
    )
    for (name, lines) <- mutants do
      intercept[GateError](InterpreterTables.verifyStream(stream(lines), pin))
      assert(
        InterpreterTables
          .compareStreams(dumps("fixture" -> original), dumps("fixture" -> lines), RenameMap.empty)
          .differences
          .nonEmpty,
        name
      )

  test("missing and extra subjects are rejected"):
    val expected = Set("machine" -> "choice")
    for actual <- Vector(Set.empty[(String, String)], expected + ("machine" -> "extra")) do
      assert(
        intercept[GateError](
          InterpreterTables.requireUniverse(expected, actual, "fixture")
        ).getMessage.contains("subject universe")
      )

  test("unavailable, empty, malformed and incomplete streams never pass"):
    assert(
      intercept[GateError](
        InterpreterTables.verifyStream(
          stream(original),
          pin.updated("status", "unavailable; NOT PASS")
        )
      ).getMessage.contains("unavailable")
    )
    for lines <- Vector(
        Vector.empty,
        Vector("not JSON"),
        original.dropRight(1),
        original.updated(5, original(5).replace("complete", "incomplete"))
      )
    do intercept[GateError](InterpreterTables.verifyStream(stream(lines), pin))

  test("mapped streams ignore positions and merge identical artifact occurrences"):
    val base = dumps("fixture-race" -> original, "fixture" -> original)
    val renamed = change(original.map(_.replace("\"go\"", "\"start\"")), 0)(header =>
      header.updated(
        "declaration",
        Json
          .fields(header("declaration"), "declaration")
          .updated("position", Map("file" -> "moved.scala", "line" -> BigDecimal(9)))
      )
    )
    val now = dumps("fixture" -> renamed)
    val map = RenameMap.parse("go start\n")
    assertEquals(InterpreterTables.compareStreams(base, now, map).differences, Vector.empty)
    assertEquals(map.used.toSet, Set("go"))
    assert(
      InterpreterTables
        .compareStreams(base, now, RenameMap.empty)
        .differences
        .exists(_.contains("line 4 differs"))
    )

  test("families distinguish subjects while artifact names and removals may change"):
    def family(name: String) = change(original, 0)(header =>
      header.updated(
        "declaration",
        Json.fields(header("declaration"), "declaration").updated("family", name)
      )
    )
    val a = family("a")
    val b = family("b").updated(3, original(3).replace("\"outcome\":\"ok\"", "\"outcome\":\"no\""))
    val base = dumps("one" -> a, "two" -> b)
    def merged(lines: Vector[String]) = change(lines, 0)(_.updated("artifact", "merged.json"))
    assertEquals(
      InterpreterTables
        .compareStreams(base, dumps("one" -> merged(a), "two" -> merged(b)), RenameMap.empty)
        .differences,
      Vector.empty
    )
    val removed =
      InterpreterTables.compareStreams(base, dumps("one" -> a), RenameMap.parse("b.choice -"))
    assertEquals(removed.differences, Vector.empty)
    assertEquals(removed.removalsUsed, Set("b.choice"))

  test("missing, extra and inconsistent duplicate streams fail"):
    val base = dumps("fixture" -> original)
    val changed = original.updated(3, original(3).replace("why", "other"))
    assert(
      InterpreterTables
        .compareStreams(base, dumps("fixture" -> Vector.empty), RenameMap.empty)
        .differences
        .nonEmpty
    )
    val duplicate = InterpreterTables.compareStreams(
      base,
      dumps("one" -> original, "two" -> changed),
      RenameMap.empty
    )
    assert(duplicate.differences.exists(_.contains("differs between")))
    val extra = Files.createTempDirectory(root, "extra")
    write(extra.resolve("one/machine-extra.jsonl.gz"), original.map(_.replace("choice", "extra")))
    assert(
      InterpreterTables
        .compareStreams(base, extra, RenameMap.empty)
        .differences
        .exists(_.contains("is new"))
    )

  test("diagnostics stay bounded while all differing records are counted"):
    val rows = Vector.fill(20000)(original(3))
    val base = dumps("fixture" -> (original.take(3) ++ rows ++ original.takeRight(2)))
    val changed = dumps(
      "fixture" -> (original.take(3) ++ rows.map(_.replace("why", "other")) ++ original.takeRight(
        2
      ))
    )
    val found = InterpreterTables.compareStreams(base, changed, RenameMap.empty).differences
    assertEquals(found.size, 13)
    assert(found.last.contains("19988 more lines differ"))

  test("a missing or changed dumper produces an inconclusive receipt before making output"):
    val pins = root.resolve("pins.json")
    Files.writeString(
      pins,
      """{"format_version":1,"seam":{"binary_sha256":"unchanged"},"artifacts":[]}"""
    )
    for (dumper, error) <- Vector(
        root.resolve("missing-dumper") -> "inherited dumper missing",
        pins -> "inherited dumper identity changed"
      )
    do
      val out = root.resolve("candidate")
      val receipt =
        InterpreterTables.observe(() => InterpreterTables.replay(root, dumper, out, pins))
      assertEquals(receipt("status"), "inconclusive")
      assert(Json.text(receipt("error"), "error").contains(error))
      assert(!Files.exists(out))

  test("the native mutation controls change paired logical and interpreter records"):
    val records = original.map(line => Json.fields(Json.parse(line), "fixture"))
    val steps = Vector(Map[String, Any]("action" -> "go"), Map[String, Any]("action" -> "wait"))
    val nativeSteps = Vector("why", "held").map(reason =>
      Map[String, Any](
        "Fields" -> Vector(
          Map.empty[String, Any],
          Map.empty[String, Any],
          Map("Items" -> Vector("first", "second")),
          Map("Text" -> reason)
        )
      )
    )
    val native = records
      .updated(0, records(0).updated("declaration", Map("steps" -> steps)))
      .updated(1, records(1).updated("declaration", steps(0)))
      .updated(2, records(2).updated("declaration", steps(1)))
      .updated(3, records(3).updated("native_steps", nativeSteps))
    for name <- Vector(
        "row",
        "result",
        "result-order",
        "fact-order",
        "binding-order",
        "explanation",
        "missing-pair"
      )
    do
      val changed = InterpreterMutants.mutate(native.iterator, name).toVector
      assert(!changed.equals(native), name)
      assert(
        InterpreterTables
          .compareStreams(
            dumps("original" -> native.map(Equivalence.canonical)),
            dumps("mutated" -> changed.map(Equivalence.canonical)),
            RenameMap.empty
          )
          .differences
          .nonEmpty,
        name
      )
      if name.equals("explanation") then
        val row = changed(3)
        val result = Json.fields(Json.items(row("results"), "results").head, "result")
        val step = Json.fields(Json.items(row("native_steps"), "steps").head, "step")
        assertEquals(
          Json.fields(Json.items(step("Fields"), "fields")(3), "reason")("Text"),
          result("because")
        )
      if name.equals("binding-order") then
        assertEquals(Json.fields(changed(0)("declaration"), "declaration")("steps"), steps.reverse)
        assertEquals(changed(1)("declaration"), steps(1))
        assertEquals(changed(2)("declaration"), steps(0))
      if name.equals("result-order") then
        assertEquals(changed(3)("native_steps"), nativeSteps.reverse)
      if name.equals("fact-order") then
        val step = Json.fields(Json.items(changed(3)("native_steps"), "steps").head, "step")
        assertEquals(
          Json.fields(Json.items(step("Fields"), "fields")(2), "facts")("Items"),
          Vector("second", "first")
        )
    val folder = Files.createTempDirectory(root, "native-probe")
    val encoded = native.map(Equivalence.canonical).mkString("", "\n", "\n")
    val hash = MessageDigest
      .getInstance("SHA-256")
      .digest(encoded.getBytes(UTF_8))
      .map(b => f"${b & 0xff}%02x")
      .mkString
    val nativePin = pin.updated("uncompressed_jsonl_sha256", hash)
    val pins = folder.resolve("pins.json")
    Files.writeString(
      pins,
      Equivalence.canonical(
        Map(
          "artifacts" -> Vector(
            Map(
              "artifact" -> "fixture.json",
              "subjects" -> Vector(nativePin, nativePin.updated("owner", "heldDispatch"))
            )
          )
        )
      )
    )
    write(folder.resolve("fixture/machine-choice.jsonl.gz"), native.map(Equivalence.canonical))
    val receipt = InterpreterMutants.probe(
      folder,
      folder.resolve("mutants"),
      pins,
      "fixture.json",
      "choice",
      "heldDispatch"
    )
    assertEquals(Json.items(receipt("observations"), "observations").size, 9)
    assert(
      Json
        .items(receipt("observations"), "observations")
        .forall(entry => Json.fields(entry, "observation")("status").equals("rejected"))
    )

  test("artifact inventory, scope, declaration and file mismatches are refused"):
    val folder = Files.createTempDirectory(root, "artifact")
    val ir = folder.resolve("fixture.json")
    val originalIr = Map[String, Any](
      "machines" -> Vector(Map("name" -> "choice")),
      "compositions" -> Vector.empty
    )
    val inventory = Map[String, Any](
      "artifact" -> "fixture.json",
      "scope" -> Map("limit" -> BigDecimal(4)),
      "declared_machines" -> BigDecimal(1),
      "declared_compositions" -> BigDecimal(0),
      "subjects" -> Vector(pin.updated("path", "machine-choice.jsonl.gz"))
    )
    val artifact = inventory.updated("subjects", Vector(pin))
    Files.writeString(ir, Equivalence.canonical(originalIr))
    Files.writeString(folder.resolve("inventory.json"), Equivalence.canonical(inventory))
    write(folder.resolve("machine-choice.jsonl.gz"), original)
    assertEquals(InterpreterTables.verifyArtifact(folder, artifact, ir).size, 1)
    val corrupt = Vector(
      "scope" -> inventory.updated("scope", Map("limit" -> BigDecimal(5))),
      "artifact" -> inventory.updated("artifact", "other.json"),
      "declared machines" -> inventory.updated("declared_machines", BigDecimal(2)),
      "missing" -> inventory.updated("subjects", Vector.empty),
      "duplicate" -> inventory.updated("subjects", Vector(pin, pin)),
      "extra" -> inventory.updated("subjects", Vector(pin, pin.updated("owner", "extra"))),
      "filename" -> inventory.updated("subjects", Vector(pin.updated("path", "other.jsonl.gz"))),
      "metadata" -> inventory.updated(
        "subjects",
        Vector(pin.updated("results", BigDecimal(3)).updated("path", "machine-choice.jsonl.gz"))
      )
    )
    for (_, changed) <- corrupt do
      Files.writeString(folder.resolve("inventory.json"), Equivalence.canonical(changed))
      intercept[GateError](InterpreterTables.verifyArtifact(folder, artifact, ir))
    Files.writeString(folder.resolve("inventory.json"), Equivalence.canonical(inventory))
    intercept[GateError](
      InterpreterTables.verifyArtifact(folder, artifact.updated("subjects", Vector(pin, pin)), ir)
    )
    Files.writeString(
      ir,
      Equivalence.canonical(
        originalIr.updated("machines", Vector(Map("name" -> "choice"), Map("name" -> "choice")))
      )
    )
    intercept[GateError](InterpreterTables.verifyArtifact(folder, artifact, ir))
    Files.writeString(ir, Equivalence.canonical(originalIr))
    val extra = write(folder.resolve("machine-extra.jsonl.gz"), original)
    intercept[GateError](InterpreterTables.verifyArtifact(folder, artifact, ir))
    Files.delete(extra)
    Files.delete(folder.resolve("machine-choice.jsonl.gz"))
    intercept[GateError](InterpreterTables.verifyArtifact(folder, artifact, ir))

  test("replay holds the shared lock and refuses failed or mutating dump processes"):
    val folder = Files.createTempDirectory(root, "replay-process")
    val irFolder = Files.createDirectories(folder.resolve("ir"))
    val ir = irFolder.resolve("fixture.json")
    val input = """{"machines":[{"name":"choice"}],"compositions":[]}"""
    Files.writeString(ir, input)
    val template = Files.createDirectories(folder.resolve("template"))
    write(template.resolve("machine-choice.jsonl.gz"), original)
    val inventory = Map[String, Any](
      "artifact" -> "fixture.json",
      "scope" -> Map.empty[String, Any],
      "declared_machines" -> BigDecimal(1),
      "declared_compositions" -> BigDecimal(0),
      "subjects" -> Vector(pin.updated("path", "machine-choice.jsonl.gz"))
    )
    Files.writeString(template.resolve("inventory.json"), Equivalence.canonical(inventory))
    val artifact = inventory.updated("subjects", Vector(pin)).updated("ir_sha256", "historical")
    val dumper = folder.resolve("dump")
    val pins = folder.resolve("pins.json")
    def configure(body: String): Unit =
      Files.writeString(dumper, "#!/bin/sh\n" + body + "\n")
      dumper.toFile.setExecutable(true): Unit
      Files.writeString(
        pins,
        Equivalence.canonical(
          Map(
            "format_version" -> BigDecimal(1),
            "seam" -> Map("binary_sha256" -> InterpreterTables.digest(dumper)),
            "artifacts" -> Vector(artifact)
          )
        )
      ): Unit
    configure(
      s"if flock -n /tmp/umpire-heavy-gates.lock true; then exit 55; fi\ncp -R '$template' \"$$2\""
    )
    val complete = InterpreterTables.replay(irFolder, dumper, folder.resolve("complete"), pins)
    assertEquals(complete("status"), "complete")
    assertEquals(complete("validated_records"), BigDecimal(6))
    intercept[GateError](
      InterpreterTables.replay(irFolder, dumper, folder.resolve("complete"), pins)
    )
    configure("exit 23")
    val failed = InterpreterTables.observe(() =>
      InterpreterTables.replay(irFolder, dumper, folder.resolve("failed"), pins)
    )
    assertEquals(failed("status"), "inconclusive")
    assert(failed("error").toString.contains("exited 23"))
    configure("printf ' ' >> \"$1\"")
    val changed = InterpreterTables.observe(() =>
      InterpreterTables.replay(irFolder, dumper, folder.resolve("changed"), pins)
    )
    assertEquals(changed("status"), "different")
    assert(changed("error").toString.contains("IR input changed"))
    Files.writeString(ir, input)
    Files.writeString(irFolder.resolve("extra.json"), input)
    val extra = InterpreterTables.observe(() =>
      InterpreterTables.replay(irFolder, dumper, folder.resolve("extra"), pins)
    )
    assertEquals(extra("status"), "different")
    assert(extra("error").toString.contains("subject universe"))
