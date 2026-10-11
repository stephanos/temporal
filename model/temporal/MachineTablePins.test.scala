package framework

import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.security.MessageDigest
import org.json4s.*
import org.json4s.jackson.JsonMethods.parse

class MachineTablePins extends munit.FunSuite:
  private val pins = parse(Files.readString(Path.of("model/testdata/table-pins/pins.json")))
  private def items(value: JValue): List[JValue] = value match
    case JArray(values) => values
    case other          => fail(s"expected pin array, got $other")
  private def string(value: JValue): String = value match
    case JString(text) => text
    case other         => fail(s"expected pin string, got $other")
  private def integer(value: JValue): Int = value match
    case JInt(number) => number.toInt
    case other        => fail(s"expected pin integer, got $other")
  private def subjects(value: JValue): Set[(String, String)] =
    items(value).map(p => string(p \ "kind") -> string(p \ "owner")).toSet
  private def hash(bytes: Array[Byte]): String =
    MessageDigest.getInstance("SHA-256").digest(bytes).map(b => f"${b & 0xff}%02x").mkString

  private def key(value: Any): String = StepTables.spell(value)

  private def catalog(actual: Seq[String], pin: JValue, label: String): Unit =
    assertEquals(actual.size, integer(pin \ "size"), label)
    assertEquals(hash(actual.mkString("\n").getBytes(UTF_8)), string(pin \ "sha256"), label)

  private def catalogs[S, O, F](machine: Machine[S, O, F], pin: JValue): Unit =
    val header = pin \ "catalogs"
    val states = machine.fs.values
    catalog(states.map(key), header \ "states", machine.name + " states")
    catalog(machine.fo.values.map(key), header \ "outcomes", machine.name + " outcomes")
    catalog(machine.ff.values.map(key), header \ "facts", machine.name + " facts")
    catalog(Seq(key(machine.init)), header \ "starts", machine.name + " starts")
    catalog(states.filter(machine.end).map(key), header \ "ends", machine.name + " ends")
    val aliases = MachineLaws.bindingKeys(machine).map(_.split('.').last)
    val classes = machine.bindings
      .zip(aliases)
      .flatMap: (binding, alias) =>
        val name = if binding.decl.name.nonEmpty then binding.decl.name else alias
        classesOf(binding.decl).map(inputs => (name +: inputs.map(key)).mkString("-"))
    val orderedClasses = classes.sorted
    catalog(orderedClasses, header \ "actions", machine.name + " classes")
    assertEquals(
      aliases,
      items(pin \ "binding_order").map(string).toVector,
      machine.name
    )

  test("current exports match independent native pins; behind features are reported"):
    val files = temporal.CompiledInventory.files
    val artifacts = items(pins \ "artifacts").filter(a =>
      files.exists(f => string(a \ "artifact").equals(f.name + ".json"))
    )
    for file <- files if !artifacts.exists(a => string(a \ "artifact").equals(file.name + ".json"))
    do println(s"feature ${file.name}: missing independent native baseline (unmodified)")
    val sealedFiles = files.filter(file =>
      artifacts
        .find(a => string(a \ "artifact").equals(file.name + ".json"))
        .exists(artifact =>
          val path = IrFile.path(Path.of("."), file.name)
          val current = Files.isRegularFile(path) && hash(Files.readAllBytes(path))
            .equals(string(artifact \ "ir_sha256"))
          if !current then println(s"feature ${file.name}: behind (IR/independent native baseline)")
          current
        )
    )
    for file <- sealedFiles do
      val artifact = artifacts.find(a => string(a \ "artifact").equals(file.name + ".json")).get
      val pinned = subjects(artifact \ "subjects")
      val bytes = Files.readAllBytes(IrFile.path(Path.of("."), file.name))
      assertEquals(
        hash(bytes),
        string(artifact \ "ir_sha256"),
        file.name + " independent IR identity"
      )
      val lifted = parse(new String(bytes, UTF_8))
      val irSubjects = Seq("machine" -> "machines", "composition" -> "compositions")
        .flatMap((kind, field) =>
          (lifted \ field).children.map(decl => kind -> string(decl \ "name"))
        )
        .toSet
      val exported = file
        .subjects()
        .map:
          case machine: Machine[?, ?, ?]   => "machine" -> machine.name
          case composition: Composition[?] => "composition" -> composition.name
          case other => fail(s"unrecognized exported subject ${IrFile.identity(other)}")
      assertEquals(irSubjects, pinned, file.name + " IR/pins")
      val declarationsMatch = file
        .subjects()
        .forall(model =>
          Seq("machines", "compositions")
            .flatMap(field => (lifted \ field).children)
            .exists(d =>
              string(d \ "name").equals(model.name) &&
                (string(d \ "family") + "." + model.name.updated(0, model.name.head.toUpper))
                  .equals(IrFile.identity(model))
            )
        )
      val current = exported.toSet == pinned && declarationsMatch
      if !current then println(s"feature ${file.name}: behind (export/native baseline)")
      else assertEquals(exported.toSet, pinned, file.name + " export/pins")
      for model <- file.subjects() if current do
        val field = model match
          case _: Machine[?, ?, ?] => "machines"
          case _: Composition[?]   => "compositions"
          case other => fail(s"unrecognized exported subject ${IrFile.identity(other)}")
        val declaration =
          (lifted \ field).children.find(d => string(d \ "name").equals(model.name)).get
        val name = string(declaration \ "name")
        assertEquals(
          IrFile.identity(model),
          string(declaration \ "family") + "." + name.updated(0, name.head.toUpper),
          file.name + " full declaration identity"
        )
      for machine <- file.subjects().collect { case m: Machine[?, ?, ?] => m } if current do
        val pin = items(artifact \ "subjects")
          .find(p =>
            string(p \ "kind").equals("machine") && string(p \ "owner").equals(machine.name)
          )
          .get
        catalogs(machine, pin)
