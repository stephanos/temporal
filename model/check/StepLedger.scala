package umpire.check

import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import scala.jdk.CollectionConverters.*

final private[check] case class BindingOrder(
    baselineKey: String,
    currentKey: String,
    baselineDigest: String,
    baselineCount: Long,
    oldBindings: Vector[String],
    currentBindings: Vector[String],
    permutation: Vector[Int]
)

final private[check] case class StepAddition(
    originalKey: String,
    currentKey: String,
    originalTables: Path,
    originalDigest: String,
    originalCount: Long,
    renames: RenameMap
)

// A batch-specific proof ledger, not a relaxed comparison mode. Without one, every order and
// unmatched table remains strict. Its addition origin is the independently admitted original
// dispatch source population, not a capture of the candidate being compared.
final private[check] case class StepLedger(
    orders: Vector[BindingOrder],
    additions: Vector[StepAddition]
)

private[check] object StepLedger:
  private val originalCommit = "fb9ff2a09582b72dc5b703b19ff5b7c514359885"
  private val originalSources =
    "a4f97a73c53efa016ce09f1799c98f385c0daac85422ba56a14e2158fb4faf5c"
  private val originalIndex =
    "6737f58be60849b11bca76c6085bd40f5258f04dade22c8fb3ec9290a1f1f7e2"
  private val originalApi =
    "ca66e13980809d30068c5d8e64302ee1bed71cca8e10379e349b9b744c9fe6f2"
  private val originalClasspath =
    "75349cd13bf0fd306772d74cdd380c19962db4897ea995be9f61a4d5cb117637"
  private val originalDependencies =
    "26426e150da2c0f49c74456f5698f33b1ddb8260ddab7584a022ba56a0168253"
  private val originalClasses =
    "8aa44e5d495652cfbb56fa8c4589326ad4689d0b741366122829e35998fadf44"
  private val originalArtifact =
    "3b5c53679ddda0ad8bc3c444653b88e05302d0201ad0665b8eda405cf584a53b"
  private val dispatchNames = Set(
    "dispatchingActivity",
    "servingWorker",
    "servedDispatch",
    "unservedDispatch"
  )
  private val oldPackage = "temporal.features.activity.standalone.system."
  private val newPackage = "temporal.features.activity.standalone.protocol."

  def hash(bytes: Array[Byte]): String =
    MessageDigest
      .getInstance("SHA-256")
      .digest(bytes)
      .map(b => f"${b & 0xff}%02x")
      .mkString

  def rowsHash(rows: Vector[String]): String = hash(
    rows.mkString("", "\n", "\n").getBytes(UTF_8)
  )

  private def fail(message: String): Nothing = throw GateError(
    s"step ledger: $message"
  )

  private def objectOf(
      value: Any,
      keys: Set[String],
      label: String
  ): Map[String, Any] =
    val fields = Json.fields(value, label)
    if fields.keySet != keys then
      fail(
        s"$label fields must be ${keys.toVector.sorted.mkString(", ")}; got ${fields.keys.toVector.sorted.mkString(", ")}"
      )
    fields

  private def text(fields: Map[String, Any], key: String): String =
    Json.text(fields(key), key)

  private def number(value: Any, label: String): Long = value match
    case n: BigDecimal if n.isWhole && n.isValidLong => n.toLong
    case _                                           => fail(s"$label is no integral Long")

  private def words(value: Any, label: String): Vector[String] =
    Json.items(value, label).map(v => Json.text(v, label))

  private def unique(values: Vector[String], label: String): Unit =
    if values.distinct.size != values.size then fail(s"$label contains duplicates")

  private def contained(parent: Path, relative: String): Path =
    val root = parent.toRealPath()
    val child = Path.of(relative)
    if child.isAbsolute || child.iterator.asScala.exists(_.toString == "..")
    then fail(s"$relative escapes $parent")
    val result = root.resolve(child).normalize()
    if !result.startsWith(root) || !Files.exists(result) then
      fail(s"$relative is missing or escapes $parent")
    val real = result.toRealPath()
    if !real.startsWith(root) then fail(s"$relative escapes $parent through a link")
    real

  def read(mapFile: Path): Option[StepLedger] =
    val pointers = Files
      .readString(mapFile)
      .linesIterator
      .map(_.trim)
      .filter(_.startsWith("# step-ledger"))
      .toVector
    if pointers.isEmpty then None
    else
      if pointers.size != 1 then fail("the map has more than one step-ledger pointer")
      val relative = pointers.head.split("\\s+").toList match
        case List("#", "step-ledger", file) => file
        case _ => fail("the pointer is exactly `# step-ledger <relative-file>`")
      val file = contained(mapFile.toAbsolutePath.getParent, relative)
      val fields = objectOf(
        Json.parse(Files.readString(file)),
        Set("version", "orders", "additions"),
        "ledger"
      )
      if number(fields("version"), "version") != 1 then fail("unsupported version")
      val orders = Json.items(fields("orders"), "orders").map { value =>
        val f = objectOf(
          value,
          Set(
            "baselineKey",
            "currentKey",
            "baselineDigest",
            "baselineCount",
            "oldBindings",
            "currentBindings",
            "permutation"
          ),
          "order"
        )
        val permutation = Json.items(f("permutation"), "permutation").map { n =>
          val value = number(n, "permutation")
          if value < 0 || value > Int.MaxValue then fail("permutation index is out of range")
          value.toInt
        }
        BindingOrder(
          text(f, "baselineKey"),
          text(f, "currentKey"),
          text(f, "baselineDigest"),
          number(f("baselineCount"), "baselineCount"),
          words(f("oldBindings"), "oldBindings"),
          words(f("currentBindings"), "currentBindings"),
          permutation
        )
      }
      unique(orders.map(_.baselineKey), "order baseline keys")
      unique(orders.map(_.currentKey), "order current keys")
      val additions =
        Json.items(fields("additions"), "additions").map { value =>
          val f = objectOf(
            value,
            Set(
              "originalKey",
              "currentKey",
              "originalDigest",
              "originalCount",
              "origin",
              "renames"
            ),
            "addition"
          )
          val originalKey = text(f, "originalKey")
          val currentKey = text(f, "currentKey")
          val name = originalKey.stripPrefix(oldPackage)
          if !dispatchNames(
              name
            ) || originalKey != oldPackage + name || currentKey != newPackage + name
          then
            fail(
              s"$originalKey -> $currentKey is not an admitted original dispatch addition"
            )
          val tables =
            authenticate(contained(file.getParent, text(f, "origin")))
          val digest = text(f, "originalDigest")
          val count = number(f("originalCount"), "originalCount")
          if !Equivalence.index(tables).contains(originalKey -> (digest, count))
          then
            fail(
              s"$originalKey digest/count is not its authenticated original index entry"
            )
          val renames =
            Json.items(f("renames"), "renames").map(v => words(v, "rename"))
          if renames != Vector(
              Vector(oldPackage.dropRight(1), newPackage.dropRight(1))
            )
          then
            fail(
              "dispatch additions use only the independently verified system-to-protocol package move"
            )
          StepAddition(
            originalKey,
            currentKey,
            tables,
            digest,
            count,
            RenameMap.parse(renames.map(_.mkString(" ")).mkString("\n"))
          )
        }
      unique(additions.map(_.currentKey), "addition keys")
      unique(additions.map(_.originalKey), "addition origin keys")
      if orders.map(_.currentKey).exists(additions.map(_.currentKey).contains)
      then fail("a current key is both an order mapping and an addition")
      Some(StepLedger(orders, additions))

  def forComparison(mapFile: Path, stepsOnly: Boolean): Option[StepLedger] =
    if stepsOnly then read(mapFile) else None

  private def authenticate(file: Path): Path =
    val f = objectOf(
      Json.parse(Files.readString(file)),
      Set(
        "version",
        "commit",
        "sourceManifest",
        "sourceManifestSHA256",
        "sourceCount",
        "sourceBytes",
        "sources",
        "tables",
        "tablesIndexSHA256",
        "apiJarSHA256",
        "classpath",
        "classpathSHA256",
        "dependencies",
        "dependenciesSHA256",
        "compiledClasses",
        "compiledClassesSHA256",
        "compiledArtifact"
      ),
      "dispatch origin"
    )
    if number(f("version"), "origin version") != 1 || text(
        f,
        "commit"
      ) != originalCommit ||
      text(f, "sourceManifestSHA256") != originalSources || text(
        f,
        "tablesIndexSHA256"
      ) != originalIndex ||
      text(f, "apiJarSHA256") != originalApi || number(
        f("sourceCount"),
        "sourceCount"
      ) != 103 ||
      number(f("sourceBytes"), "sourceBytes") != 878558
    then
      fail(
        "dispatch origin is not the independently admitted original committed source population"
      )
    def receipt(key: String, admitted: String): Path =
      val path = contained(file.getParent, text(f, key))
      if text(f, key + "SHA256") != admitted || hash(
          Files.readAllBytes(path)
        ) != admitted
      then fail(s"original $key receipt changed")
      path
    val classpath = receipt("classpath", originalClasspath)
    val dependencies = receipt("dependencies", originalDependencies)
    val classes = receipt("compiledClasses", originalClasses)
    val artifact = contained(file.getParent, text(f, "compiledArtifact"))
    if hash(Files.readAllBytes(artifact)) != originalArtifact then
      fail("original compiled artifact changed")
    val dependencyEntries =
      Files.readAllLines(dependencies).asScala.toVector.map { line =>
        line.split("  ", 2).toList match
          case List(digest, path) => digest -> Path.of(path)
          case _                  => fail("malformed dependency receipt")
      }
    val cp = Files
      .readString(classpath)
      .trim
      .split(java.io.File.pathSeparator)
      .toVector
    if cp.size != 21 || cp.tail != dependencyEntries.map(_._2.toString) then
      fail(
        "original execution classpath is not its complete dependency receipt"
      )
    for (digest, originalPath) <- dependencyEntries do
      val path = if originalPath.getFileName.toString == "api-scalapb.jar" then
        Path.of("model/build/api-scalapb.jar")
      else originalPath
      if !Files.isRegularFile(path) || hash(Files.readAllBytes(path)) != digest
      then
        fail(
          s"execution dependency ${originalPath.getFileName} changed or is missing"
        )
      if originalPath.getFileName.toString == "api-scalapb.jar" && digest != originalApi
      then fail("execution API is not the admitted API")
    if dependencyEntries.count(_._1 == originalApi) != 1 then
      fail("execution API dependency is missing")
    val expectedClasses =
      Files.readAllLines(classes).asScala.toVector.map { line =>
        line.split("  ", 2).toList match
          case List(digest, path) if path.startsWith("./") =>
            path.drop(2) -> digest
          case _ => fail("malformed compiled class receipt")
      }
    unique(expectedClasses.map(_._1), "compiled class paths")
    val zip = new java.util.zip.ZipFile(artifact.toFile)
    try
      val entries = zip.entries().asScala.filterNot(_.isDirectory).toVector
      val manifest = "META-INF/MANIFEST.MF"
      if entries.map(_.getName).toSet != expectedClasses
          .map(_._1)
          .toSet + manifest || expectedClasses.size != 1706
      then fail("original compiled member population changed")
      for (path, digest) <- expectedClasses do
        val stream = zip.getInputStream(zip.getEntry(path))
        try
          if hash(stream.readAllBytes()) != digest then fail(s"compiled member $path changed")
        finally stream.close()
      val stream = zip.getInputStream(zip.getEntry(manifest))
      try
        if new String(
            stream.readAllBytes(),
            UTF_8
          ) != "Manifest-Version: 1.0\r\nCreated-By: 25.0.3 (Ubuntu)\r\n\r\n"
        then fail("compiled packaging manifest changed")
      finally stream.close()
    finally zip.close()
    val manifest = contained(file.getParent, text(f, "sourceManifest"))
    if hash(Files.readAllBytes(manifest)) != originalSources then
      fail("original source manifest changed")
    val sources = contained(file.getParent, text(f, "sources"))
    val entries = Files.readAllLines(manifest).asScala.toVector.map { line =>
      line.split("  ", 2).toList match
        case List(digest, relative) if digest.matches("[0-9a-f]{64}") =>
          digest -> relative
        case _ => fail("malformed original source manifest")
    }
    unique(entries.map(_._2), "original source paths")
    val size = entries.foldLeft(0L): (total, entry) =>
      val (digest, relative) = entry
      val bytes = Files.readAllBytes(contained(sources, relative))
      if hash(bytes) != digest then fail(s"original input $relative changed")
      total + bytes.length
    val stream = Files.walk(sources)
    val actual = try
      stream.iterator.asScala
        .filter(Files.isRegularFile(_))
        .map(p => sources.relativize(p).toString)
        .toSet
    finally stream.close()
    if entries.size != 103 || size != 878558 || actual != entries
        .map(_._2)
        .toSet
    then fail("original source input population changed")
    val tables = contained(file.getParent, text(f, "tables"))
    if hash(Files.readAllBytes(tables.resolve("index.tsv"))) != originalIndex
    then fail("original native table index changed")
    tables

  // Validate both vectors and both complete, contiguous block populations before moving anything.
  // No class, pair, result, fact, reason, header or state row is reordered inside a block.
  def reorder(
      seal: BindingOrder,
      raw: Vector[String],
      current: Vector[String],
      map: RenameMap,
      dropped: Set[String],
      digest: String
  ): Vector[String] =
    if seal.baselineDigest != digest || seal.baselineCount != raw.size || rowsHash(
        raw
      ) != digest
    then fail("binding-order baseline digest/count/payload does not match")
    def split(rows: Vector[String], expected: Vector[String], label: String) =
      val at = rows.indexWhere(_.startsWith("step\t"))
      if at < 0 then fail(s"$label has no action blocks")
      val header = rows.take(at)
      val binding = header.filter(_.startsWith("bindings\t"))
      if binding != Vector("bindings\t" + expected.mkString(" ")) then
        fail(s"$label binding vector is not the exact admitted vector")
      unique(expected, s"$label bindings")
      val blocks =
        scala.collection.mutable.ArrayBuffer.empty[(String, Vector[String])]
      for row <- rows.drop(at) do
        if !row.startsWith("step\t") || !row.drop(5).contains('[') then
          fail(s"$label has a non-step inside action blocks")
        val action = row.drop(5).takeWhile(_ != '[')
        if blocks.lastOption.exists(_._1 == action) then
          val last = blocks.last
          blocks(blocks.size - 1) = action -> (last._2 :+ row)
        else blocks += action -> Vector(row)
      if blocks.map(_._1).toVector != expected then
        fail(
          s"$label action blocks are split, missing, extra or in the wrong order"
        )
      (header, blocks.toVector)
    val (header, blocks) = split(raw, seal.oldBindings, "baseline")
    split(current, seal.currentBindings, "current"): Unit
    val kept = blocks.filterNot((action, _) => dropped(action))
    val mapped = kept.map((action, rows) => map(action) -> rows.map(map.apply))
    val indexes = mapped.indices.toVector
    if seal.permutation.sorted != indexes then fail("binding-order permutation is not bijective")
    val reordered = seal.permutation.map(mapped(_))
    if reordered.map(_._1) != seal.currentBindings then
      fail("permutation does not produce the exact current binding vector")
    if mapped.map(_._1) == seal.currentBindings then
      fail("unused binding-order seal changes no order")
    val normalizedHeader = header.map { row =>
      if row.startsWith("bindings\t") then "bindings\t" + seal.currentBindings.mkString(" ")
      else map(row)
    }
    normalizedHeader ++ reordered.flatMap(_._2)
