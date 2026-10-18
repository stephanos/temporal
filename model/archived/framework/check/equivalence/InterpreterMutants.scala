package framework.check
package equivalence

import java.io.{BufferedReader, InputStreamReader}
import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.util.zip.{GZIPInputStream, GZIPOutputStream}
import scala.collection.mutable

object InterpreterMutants:
  val names: Vector[String] = Vector(
    "row",
    "result",
    "result-order",
    "fact-order",
    "binding-order",
    "explanation",
    "missing-pair"
  )

  private def fields(value: Any): Map[String, Any] = Json.fields(value, "mutation record")
  private def items(value: Any): Vector[Any] = Json.items(value, "mutation records")
  private def text(value: Any): String = Json.text(value, "mutation text")

  private def native(row: Map[String, Any], index: Int, field: Int)(
      change: Map[String, Any] => Map[String, Any]
  ): Map[String, Any] =
    val steps = items(row("native_steps"))
    val step = fields(steps(index))
    val values = items(step("Fields"))
    row.updated(
      "native_steps",
      steps.updated(
        index,
        step.updated("Fields", values.updated(field, change(fields(values(field)))))
      )
    )

  def mutate(records: Iterator[Map[String, Any]], name: String): Iterator[Map[String, Any]] =
    if !names.contains(name) then throw GateError("unknown mutant " + name)
    if !records.hasNext then throw GateError("empty mutation input")
    val header = records.next()
    val states = items(header("states"))
    val changed = mutable.Set.empty[String]
    def seeded(): Unit = changed += name: Unit
    def other(state: Any): Any = states
      .find(s => !s.equals(state))
      .getOrElse(throw GateError(s"no alternative state for $name"))
    val first =
      if name.equals("binding-order") then
        val declaration = fields(header("declaration"))
        val steps = items(declaration("steps"))
        if steps.size < 2 || steps(0).equals(steps(1)) then
          throw GateError("seed did not alter behavior: " + name)
        seeded()
        header.updated(
          "declaration",
          declaration.updated("steps", steps.take(2).reverse ++ steps.drop(2))
        )
      else header
    val steps = if name.equals("binding-order") then items(fields(header("declaration"))("steps"))
    else Vector.empty
    val tail = records.flatMap: record =>
      if name.equals("binding-order") && record.get("kind").contains("binding") && Vector(
          BigDecimal(0),
          BigDecimal(1)
        ).exists(index => record.get("index").contains(index))
      then
        val index = if record("index").equals(BigDecimal(0)) then 1 else 0
        Iterator.single(record.updated("declaration", steps(index)))
      else if name
          .equals("missing-pair") && changed.isEmpty && record.get("kind").contains("disabled")
      then
        seeded()
        Iterator.empty
      else if !record.get("kind").contains("row") || changed.nonEmpty || name.equals("missing-pair")
      then Iterator.single(record)
      else
        val results = items(record("results"))
        val eligible = name match
          case "result-order" => results.size > 1
          case "fact-order"   =>
            results.exists(result => items(fields(result)("facts")).distinct.size > 1)
          case _ => true
        if !eligible then Iterator.single(record)
        else
          seeded()
          val result = fields(results.head)
          val modified = name match
            case "row" =>
              val source = other(record("source"))
              record
                .updated("source", source)
                .updated("key", text(source) + "-" + text(record("action")))
            case "result" =>
              record.updated(
                "results",
                results.updated(0, result.updated("state", other(result("state"))))
              )
            case "result-order" =>
              record
                .updated("results", results.reverse)
                .updated("native_steps", items(record("native_steps")).reverse)
            case "fact-order" =>
              val index =
                results.indexWhere(result => items(fields(result)("facts")).distinct.size > 1)
              val selected = fields(results(index))
              native(
                record.updated(
                  "results",
                  results
                    .updated(index, selected.updated("facts", items(selected("facts")).reverse))
                ),
                index,
                2
              )(field => field.updated("Items", items(field("Items")).reverse))
            case "explanation" =>
              native(
                record.updated(
                  "results",
                  results.updated(
                    0,
                    result.updated("because", text(result("because")) + " seeded-difference")
                  )
                ),
                0,
                3
              )(field => field.updated("Text", text(field("Text")) + " seeded-difference"))
            case _ => throw GateError("seed did not alter behavior: " + name)
          Iterator.single(modified)
    Iterator.single(first) ++ tail ++ Iterator
      .single(())
      .flatMap: _ =>
        if changed.isEmpty then throw GateError("seed did not alter behavior: " + name)
        Iterator.empty[Map[String, Any]]

  def probe(
      native: Path,
      out: Path,
      pins: Path,
      artifactName: String,
      owner: String,
      missingOwner: String
  ): Map[String, Any] =
    val manifest = fields(Json.parse(Files.readString(pins)))
    val artifact = items(manifest("artifacts"))
      .map(fields)
      .find(_("artifact").equals(artifactName))
      .getOrElse(throw GateError(s"no pinned artifact $artifactName"))
    val subjects = items(artifact("subjects")).map(fields)
    val pin = subjects
      .find(_("owner").equals(owner))
      .getOrElse(throw GateError(s"no pinned subject $owner"))
    val path = native
      .resolve(artifactName.stripSuffix(".json"))
      .resolve(s"${text(pin("kind"))}-$owner.jsonl.gz")
    InterpreterTables.verifyStream(path, pin)
    if Files.exists(out) then throw GateError("refusing to overwrite mutant observations")
    Files.createDirectories(out): Unit
    val observations = names.map: name =>
      val source =
        BufferedReader(InputStreamReader(GZIPInputStream(Files.newInputStream(path)), UTF_8))
      val latest = mutable.ArrayBuffer.empty[(Map[String, Any], String)]
      val original = Iterator
        .continually(Option(source.readLine()))
        .takeWhile(_.nonEmpty)
        .flatten
        .map: line =>
          val record = fields(Json.parse(line))
          latest.clear()
          latest += record -> line
          record
      val mutant = out.resolve(name + ".jsonl.gz")
      val target = GZIPOutputStream(Files.newOutputStream(mutant))
      try
        mutate(original, name).foreach: record =>
          val line =
            latest.headOption.filter(_._1.equals(record)).fold(Equivalence.canonical(record))(_._2)
          target.write((line + "\n").getBytes(UTF_8))
      finally
        target.close()
        source.close()
      rejected(name)(InterpreterTables.verifyStream(mutant, pin): Unit)
    val expected = subjects.map(subject => text(subject("kind")) -> text(subject("owner"))).toSet
    val controls = Vector(
      "missing-machine" -> (expected - ("machine" -> missingOwner)),
      "extra-machine" -> (expected + ("machine" -> "seededExtraMachine"))
    ).map((name, actual) =>
      rejected(name)(InterpreterTables.requireUniverse(expected, actual, artifactName))
    )
    val receipt = Map[String, Any](
      "status" -> "complete",
      "baseline_stream" -> path.toString,
      "pins_sha256" -> InterpreterTables.digest(pins),
      "observations" -> (observations ++ controls)
    )
    Files.writeString(out.resolve("receipt.json"), Equivalence.canonical(receipt) + "\n")
    receipt

  private def rejected(name: String)(run: => Unit): Map[String, Any] =
    val error =
      try
        run
        None
      catch case error: GateError => Some(error.getMessage)
    error.fold(throw GateError("mutant escaped: " + name))(message =>
      Map("mutant" -> name, "status" -> "rejected", "error" -> message)
    )
