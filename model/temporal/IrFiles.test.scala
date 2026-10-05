package temporal
// Every IR file's roots, constructed as Scala: the model gate runs this, so a machine whose rules
// overlap, which its `rules` object refuses as it is constructed, fails the gate (fn-126 R16). Every
// machine an IR file lifts must be among those constructed, so none escapes that check.

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import umpire.IrFile

class IrFilesTest extends munit.FunSuite:
  /** Each feature's `object exports`, which declares its IR files as it initializes. */
  val declaring: Seq[AnyRef] = Seq(
    features.standaloneactivity.exports,
    features.nexuscaller.exports,
    features.nexuscaller.closepolicy.exports,
    features.nexusoperation.exports
  )

  /** The checked-in IR files, by name: model/ir/<name>.json, beside no sidecar. */
  def checkedIn: Set[String] =
    val ir = Path.of("model/ir")
    assert(
      Files.isDirectory(ir),
      "model/ir is missing: the Models' tests run at the repository's root"
    )
    val stream = Files.list(ir)
    try
      stream.iterator.asScala
        .map(_.getFileName.toString)
        .filter(n => n.endsWith(".json") && !n.endsWith(".laws.json") && !n.endsWith(".lint.json"))
        .map(_.stripSuffix(".json"))
        .toSet
    finally stream.close()

  /**
   * The names of the machines an IR file lifts: each element of its top-level `machines` array
   * names one with its own `name`, ahead of any nested object's.
   */
  def liftedMachines(file: String): Set[String] =
    val json = Files.readString(Path.of(s"model/ir/$file.json"))
    val start = json.indexOf("\"machines\": [")
    if start < 0 then Set.empty
    else
      val names = Set.newBuilder[String]
      val name = "\"name\": \""
      var depth = 0 // scalafix:ok DisableSyntax.var
      var i = json.indexOf('[', start) + 1 // scalafix:ok DisableSyntax.var
      var inString = false // scalafix:ok DisableSyntax.var
      var done = false // scalafix:ok DisableSyntax.var
      while !done && i < json.length do
        val c = json.charAt(i)
        if inString then
          if c == '\\' then i += 1
          else if c == '"' then inString = false
        else if depth == 1 && json.startsWith(name, i) then
          val from = i + name.length
          names += json.substring(from, json.indexOf('"', from))
          i = json.indexOf('"', from)
        else
          c match
            case '"'       => inString = true
            case '{' | '[' => depth += 1
            case '}'       => depth -= 1
            case ']'       => if depth == 0 then done = true else depth -= 1
            case _         => ()
        i += 1
      names.result()

  test("every IR file is declared by a feature's exports, and every root of each constructs") {
    assertEquals(declaring.size, 4)
    val declared = IrFile.declared
    assertEquals(declared.map(_.name).toSet, checkedIn)
    for file <- declared do
      val lifted = liftedMachines(file.name)
      val unconstructed = lifted -- file.construct(lifted)
      assert(
        unconstructed.isEmpty,
        s"${file.name}.json lifts ${unconstructed.toSeq.sorted.mkString(", ")}, which no root reaches " +
          "and no machine object initialized names, so no rule overlap of theirs is checked"
      )
  }
