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
    val name = "\"name\": \""
    // Reads from `i` at nesting depth `depth` of the array, inside a string or not, to its end.
    @scala.annotation.tailrec
    def scan(i: Int, depth: Int, inString: Boolean, names: List[String]): List[String] =
      if i >= json.length then names
      else
        val c = json.charAt(i)
        if inString then
          if c == '\\' then scan(i + 2, depth, true, names)
          else scan(i + 1, depth, c != '"', names)
        else if depth == 1 && json.startsWith(name, i) then
          val from = i + name.length
          val to = json.indexOf('"', from)
          scan(to + 1, depth, false, json.substring(from, to) :: names)
        else
          c match
            case '"'               => scan(i + 1, depth, true, names)
            case '{' | '['         => scan(i + 1, depth + 1, false, names)
            case '}'               => scan(i + 1, depth - 1, false, names)
            case ']' if depth == 0 => names
            case ']'               => scan(i + 1, depth - 1, false, names)
            case _                 => scan(i + 1, depth, false, names)
    val start = json.indexOf("\"machines\": [")
    if start < 0 then Set.empty else scan(json.indexOf('[', start) + 1, 0, false, Nil).toSet

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
