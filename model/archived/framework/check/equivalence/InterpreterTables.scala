package framework.check
package equivalence

import java.io.{BufferedReader, InputStreamReader, IOException}
import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.{DigestInputStream, MessageDigest}
import java.util.concurrent.TimeUnit
import java.util.zip.GZIPInputStream
import scala.collection.mutable
import scala.jdk.CollectionConverters.*

object InterpreterTables:
  private val suffix = ".jsonl.gz"
  private val shown = 12

  private def require(condition: Boolean, message: => String): Unit =
    if !condition then throw GateError(message)

  private def fields(value: Any): Map[String, Any] = Json.fields(value, "interpreter record")
  private def text(value: Any): String = Json.text(value, "interpreter field")
  private def items(value: Any): Vector[Any] = Json.items(value, "interpreter catalog")
  private def read(path: Path): Map[String, Any] = fields(Json.parse(Files.readString(path)))
  private def hex(bytes: Array[Byte]): String = bytes.map(b => f"${b & 0xff}%02x").mkString
  private def hash(bytes: Array[Byte]): String = hex(
    MessageDigest.getInstance("SHA-256").digest(bytes)
  )

  def digest(path: Path): String =
    val result = MessageDigest.getInstance("SHA-256")
    val source = DigestInputStream(Files.newInputStream(path), result)
    try source.transferTo(java.io.OutputStream.nullOutputStream()): Unit
    finally source.close()
    hex(result.digest())

  private def reader(path: Path): BufferedReader =
    BufferedReader(InputStreamReader(GZIPInputStream(Files.newInputStream(path)), UTF_8))

  private def lines(source: BufferedReader): Iterator[String] =
    Iterator.continually(Option(source.readLine())).takeWhile(_.nonEmpty).flatten

  def verifyStream(path: Path, pin: Map[String, Any]): Map[String, Any] =
    require(pin("status").equals("complete"), s"unavailable table ${pin("owner")}")
    val hashed = MessageDigest.getInstance("SHA-256")
    val source = BufferedReader(
      InputStreamReader(
        DigestInputStream(GZIPInputStream(Files.newInputStream(path)), hashed),
        UTF_8
      )
    )
    val (first, last, count) =
      try
        lines(source).foldLeft((Option.empty[String], Option.empty[String], 0L)):
          case ((first, _, count), line) => (first.orElse(Some(line)), Some(line), count + 1)
      finally source.close()
    require(first.nonEmpty && last.nonEmpty, s"empty table $path")
    val header = fields(Json.parse(first.get))
    val footer = fields(Json.parse(last.get))
    require(
      header("kind").equals("header") && footer("kind").equals("footer"),
      s"header/footer missing $path"
    )
    require(
      Vector(header("artifact"), header("subject_kind"), header("owner"))
        .equals(Vector(pin("artifact"), pin("kind"), pin("owner"))),
      s"table identity changed $path"
    )
    require(
      BigDecimal(items(header("states")).size).equals(pin("states")) && BigDecimal(
        items(header("actions")).size
      ).equals(pin("actions")),
      s"catalog sizes changed $path"
    )
    for (field, expected) <- fields(pin.getOrElse("catalogs", Map.empty[String, Any])) do
      val values = header(field) match
        case _: None.type => Vector.empty
        case value        => items(value).map(text)
      val actual = Map(
        "size" -> BigDecimal(values.size),
        "sha256" -> hash(values.mkString("\n").getBytes(UTF_8))
      )
      require(actual.equals(expected), s"ordered $field catalog differs $path")
    pin
      .get("binding_order")
      .foreach: expected =>
        val bindings = items(fields(header("declaration")).getOrElse("steps", Vector.empty)).map(
          step => text(fields(step)("action")).split('.').last
        )
        require(bindings.equals(expected), s"binding order differs $path")
    val expectedFooter = Map[String, Any]("kind" -> "footer", "status" -> "complete") ++ Vector(
      "states" -> "states",
      "actions" -> "actions",
      "pairs" -> "state_action_pairs",
      "enabled" -> "enabled_pairs",
      "disabled" -> "disabled_pairs",
      "unknown" -> "unknown_pairs",
      "results" -> "results"
    ).map((field, pinned) => field -> pin(pinned))
    require(footer.equals(expectedFooter), s"pair/result completeness changed $path")
    require(BigDecimal(count).equals(pin("lines")), s"ordered record count changed $path")
    val observed = hex(hashed.digest())
    require(
      observed.equals(pin("uncompressed_jsonl_sha256")),
      s"complete ordered table differs $path"
    )
    Map("records" -> BigDecimal(count), "sha256" -> observed)

  def requireUniverse[A](expected: Set[A], actual: Set[A], label: String): Unit =
    require(
      expected.equals(actual),
      s"$label subject universe: missing=${(expected -- actual).toVector.map(_.toString).sorted} extra=${(actual -- expected).toVector.map(_.toString).sorted}"
    )

  private def subjects(entries: Vector[Any], label: String): Set[(String, String)] =
    val keys = entries.map(value =>
      val entry = fields(value)
      text(entry("kind")) -> text(entry("owner"))
    )
    require(keys.distinct.size == keys.size, s"duplicate $label subjects")
    keys.toSet

  def irUniverse(path: Path): Set[(String, String)] =
    val model = read(path)
    val entries =
      Vector("machine" -> "machines", "composition" -> "compositions").flatMap((kind, section) =>
        items(model.getOrElse(section, Vector.empty)).map(value =>
          kind -> text(fields(value)("name"))
        )
      )
    require(entries.distinct.size == entries.size, s"duplicate lifted subjects in $path")
    entries.toSet

  private def files(folder: Path, keep: Path => Boolean): Vector[Path] =
    if !Files.isDirectory(folder) then Vector.empty
    else
      val source = Files.list(folder)
      try source.iterator.asScala.filter(keep).toVector.sortBy(_.toString)
      finally source.close()

  def verifyArtifact(folder: Path, pin: Map[String, Any], ir: Path): Vector[Map[String, Any]] =
    val pinned = items(pin("subjects"))
    val expected = subjects(pinned, "pinned")
    requireUniverse(expected, irUniverse(ir), ir.getFileName.toString)
    val inventory = read(folder.resolve("inventory.json"))
    val observed = items(inventory("subjects"))
    requireUniverse(expected, subjects(observed, "interpreter"), s"${ir.getFileName} interpreter")
    def filename(kind: String, owner: String) = s"$kind-$owner$suffix"
    requireUniverse(
      expected.map(filename),
      files(folder, _.getFileName.toString.endsWith(suffix)).map(_.getFileName.toString).toSet,
      "dump files"
    )
    require(
      inventory("artifact").equals(ir.getFileName.toString) && inventory("artifact").equals(
        pin("artifact")
      ),
      "artifact identity changed"
    )
    require(inventory("scope").equals(pin("scope")), "interpreter scope changed")
    for field <- Vector("declared_machines", "declared_compositions") do
      require(inventory(field).equals(pin(field)), s"$field changed")
    val indexed = observed
      .map(value =>
        val entry = fields(value)
        (text(entry("kind")), text(entry("owner"))) -> entry
      )
      .toMap
    pinned.map: value =>
      val entry = fields(value)
      val kind = text(entry("kind"))
      val owner = text(entry("owner"))
      val current = indexed(kind -> owner)
      for (field, expected) <- entry if !Set("catalogs", "binding_order")(field) do
        require(current.getOrElse(field, None).equals(expected), s"$owner: $field differs")
      val name = filename(kind, owner)
      require(
        Path.of(text(current("path"))).getFileName.toString.equals(name),
        "mismatched dump filename"
      )
      Map("owner" -> owner, "kind" -> kind) ++ verifyStream(folder.resolve(name), entry)

  private def withoutPositions(value: Any): Any = value match
    case m: Map[?, ?] =>
      m.collect {
        case (k: String, v) if !Set("position", "Position")(k) => k -> withoutPositions(v)
      }
    case xs: Vector[?] => xs.map(withoutPositions)
    case other         => other

  private def normalized(line: String, map: RenameMap): String =
    val record = fields(withoutPositions(Json.parse(line)))
    val kept = if record.get("kind").contains("header") then record - "artifact" else record
    map(Equivalence.canonical(kept))

  private def subject(path: Path): (String, String) =
    val parts = path.getFileName.toString.stripSuffix(suffix).split("-", 2)
    require(parts.length == 2, s"malformed dump filename $path")
    val source = reader(path)
    val first = try Option(source.readLine()).filter(_.trim.nonEmpty)
    finally source.close()
    val header = first.fold(Map.empty[String, Any])(line => fields(Json.parse(line)))
    val declaration =
      header.get("declaration").filterNot(_.equals(None)).fold(Map.empty[String, Any])(fields)
    val family = declaration.get("family").fold("")(text)
    parts(0) -> (if family.isEmpty then parts(1) else s"$family.${parts(1)}")

  private def streams(folder: Path): Map[(String, String), Vector[Path]] =
    files(folder, Files.isDirectory(_))
      .flatMap(directory => files(directory, _.getFileName.toString.endsWith(suffix)))
      .groupBy(subject)

  private def compareLines(leftPath: Path, rightPath: Path, map: RenameMap): Vector[String] =
    val left = reader(leftPath)
    try
      val right = reader(rightPath)
      try
        val old = lines(left).map(normalized(_, map)).buffered
        val now = lines(right).map(normalized(_, RenameMap.empty)).buffered
        @scala.annotation.tailrec
        def walk(at: Long, differing: Long, found: Vector[String]): (Long, Long, Vector[String]) =
          if !(old.hasNext && now.hasNext) then (at, differing, found)
          else
            val (a, b) = (old.next(), now.next())
            val changed = !a.equals(b)
            val more = if changed && found.size < shown then
              found :+ s"line ${at + 1} differs\n    baseline: ${a.take(400)}\n    now:      ${b.take(400)}"
            else found
            walk(at + 1, differing + (if changed then 1 else 0), more)
        val (count, differing, found) = walk(0, 0, Vector.empty)
        def remaining(source: Iterator[String]) = source.foldLeft(0L)((count, _) => count + 1)
        val (oldCount, nowCount) = (count + remaining(old), count + remaining(now))
        found ++ Option.when(differing > shown)(s"${differing - shown} more lines differ") ++
          Option.when(oldCount != nowCount)(s"has $nowCount lines, the baseline $oldCount")
      finally right.close()
    finally left.close()

  def compareStreams(base: Path, now: Path, map: RenameMap): Compared =
    val was = streams(base)
    val current = streams(now)
    require(was.nonEmpty, s"$base holds no dumps")
    val differences = mutable.ArrayBuffer.empty[String]
    val matched = mutable.Set.empty[(String, String)]
    val removed = map.removals.filterNot(_.contains('/')).toSet
    val usedRemovals = mutable.Set.empty[String]
    for ((kind, owner), paths) <- was.toVector.sortBy(_._1) do
      if removed(owner) then usedRemovals += owner: Unit
      else
        val target = kind -> map(owner)
        current.get(target) match
          case None          => differences += s"$kind $owner (as ${target._2}) is gone"
          case Some(changed) =>
            matched += target
            for variant <- changed.tail do
              if compareLines(changed.head, variant, RenameMap.empty).nonEmpty then
                differences += s"$kind ${target._2} differs between the changed IR files that hold it"
            for path <- paths do
              differences ++= compareLines(path, changed.head, map).map(line =>
                s"$kind ${target._2} (${path.getParent.getFileName}) $line"
              )
    for (kind, owner) <- (current.keySet -- matched).toVector.sorted do
      differences += s"$kind $owner is new"
    Compared(differences.toVector, usedRemovals.toSet)

  private def artifactFiles(ir: Path): Vector[Path] =
    files(
      ir,
      path =>
        val name = path.getFileName.toString
        name.endsWith(".json") && !name.endsWith(".lint.json") && !name.endsWith(".waivers.json")
    )

  private def checkedPins(pins: Path, dumper: Path): Map[String, Any] =
    val value = read(pins)
    require(value("format_version").equals(BigDecimal(1)), "unsupported pin format")
    require(Files.isRegularFile(dumper), s"inconclusive: inherited dumper missing: $dumper")
    require(
      digest(dumper).equals(fields(value("seam"))("binary_sha256")),
      "inconclusive: inherited dumper identity changed"
    )
    value

  private def heavy[A](tools: Tools)(body: => A): A =
    val command = Seq(
      "-x",
      "/tmp/umpire-heavy-gates.lock",
      tools.find("sh").toString,
      "-c",
      "printf 'locked\\n'; read release"
    )
    val process =
      tools.process("flock", command).redirectError(ProcessBuilder.Redirect.INHERIT).start()
    val ready = BufferedReader(InputStreamReader(process.getInputStream, UTF_8))
    try
      require(
        Option(ready.readLine()).contains("locked"),
        "inconclusive: heavy-gates lock was not acquired"
      )
      body
    finally
      process.getOutputStream.close()
      ready.close()
      process.waitFor(): Unit

  private def fresh(out: Path): Unit =
    require(!Files.exists(out), "refusing to overwrite a replay; choose a fresh scratch directory")
    Files.createDirectories(out): Unit

  private def dump(
      ir: Path,
      dumper: Path,
      out: Path,
      tools: Tools
  ): (Vector[String], Double, String) =
    val before = digest(ir)
    val stem = ir.getFileName.toString.stripSuffix(".json")
    val command = Vector(
      dumper.toAbsolutePath.normalize.toString,
      ir.toAbsolutePath.normalize.toString,
      out.resolve(stem).toAbsolutePath.normalize.toString
    )
    val started = System.nanoTime()
    val builder = tools.process(command.head, command.tail)
    builder.environment().put("BASH_ENV", "/dev/null")
    val process = builder
      .redirectErrorStream(true)
      .redirectOutput(out.resolve(stem + ".log").toFile)
      .redirectInput(ProcessBuilder.Redirect.from(java.io.File("/dev/null")))
      .start()
    val finished = process.waitFor(600, TimeUnit.SECONDS)
    if !finished then
      process.descendants().forEach(child => child.destroyForcibly(): Unit)
      process.destroyForcibly()
      process.waitFor(): Unit
    require(finished, s"inconclusive: native dump timed out: $command")
    require(
      process.exitValue() == 0,
      s"inconclusive: native dump exited ${process.exitValue()}: $command"
    )
    require(digest(ir).equals(before), s"IR input changed during replay $ir")
    (command, (System.nanoTime() - started).toDouble / 1e9, before)

  def replay(
      ir: Path,
      dumper: Path,
      out: Path,
      pins: Path,
      tools: Tools = Tools.here
  ): Map[String, Any] =
    val pinsHash = digest(pins)
    val value = checkedPins(pins, dumper)
    val artifacts = items(value("artifacts")).map(fields)
    requireUniverse(
      artifacts.map(pin => text(pin("artifact"))).toSet,
      artifactFiles(ir).map(_.getFileName.toString).toSet,
      "IR files"
    )
    fresh(out)
    val summaries = heavy(tools):
      artifacts.map: pin =>
        val path = ir.resolve(text(pin("artifact")))
        requireUniverse(
          subjects(items(pin("subjects")), "pinned"),
          irUniverse(path),
          path.getFileName.toString
        )
        val (command, seconds, inputHash) = dump(path, dumper, out, tools)
        val summaries =
          verifyArtifact(out.resolve(path.getFileName.toString.stripSuffix(".json")), pin, path)
        Map[String, Any](
          "artifact" -> path.getFileName.toString,
          "input_sha256" -> inputHash,
          "baseline_ir_sha256" -> pin("ir_sha256"),
          "command" -> command,
          "seconds" -> BigDecimal(seconds),
          "exit_code" -> BigDecimal(0),
          "subjects" -> summaries
        )
    require(digest(pins).equals(pinsHash), "pin inputs changed during replay")
    require(
      digest(dumper).equals(fields(value("seam"))("binary_sha256")),
      "dumper changed during replay"
    )
    val count = summaries
      .flatMap(summary => items(summary("subjects")))
      .map(subject =>
        fields(subject)("records") match
          case n: BigDecimal => n
          case other         => throw GateError(s"invalid record count $other")
      )
      .sum
    Map(
      "status" -> "complete",
      "pins_sha256" -> pinsHash,
      "dumper_sha256" -> digest(dumper),
      "artifacts" -> summaries,
      "validated_records" -> count
    )

  def equivalence(
      ir: Path,
      dumper: Path,
      out: Path,
      pins: Path,
      baseline: Path,
      map: RenameMap,
      tools: Tools = Tools.here
  ): Compared =
    val pinsHash = digest(pins)
    val value = checkedPins(pins, dumper)
    fresh(out)
    heavy(tools):
      artifactFiles(ir).foreach(path => dump(path, dumper, out, tools): Unit)
    require(digest(pins).equals(pinsHash), "pin inputs changed during replay")
    require(
      digest(dumper).equals(fields(value("seam"))("binary_sha256")),
      "dumper changed during replay"
    )
    compareStreams(baseline, out, map)

  def observe(run: () => Map[String, Any]): Map[String, Any] =
    try run()
    catch
      case error: IOException => Map("status" -> "inconclusive", "error" -> error.toString)
      case error: ToolError   => Map("status" -> "inconclusive", "error" -> error.getMessage)
      case error: GateError   =>
        Map(
          "status" -> (if error.getMessage.startsWith("inconclusive:") then "inconclusive"
                       else "different"),
          "error" -> error.getMessage
        )
      case error: NoSuchElementException => Map("status" -> "different", "error" -> error.toString)
