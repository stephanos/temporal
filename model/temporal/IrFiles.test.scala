package temporal
// Every IR file's roots, constructed as Scala: the model gate runs this, so a machine whose rules
// overlap, which its `rules` object refuses as it is constructed, fails the gate (fn-126 R16).

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

  test("every IR file is declared by a feature's exports, and every root of each constructs") {
    assertEquals(declaring.size, 4)
    val declared = IrFile.declared
    assertEquals(declared.map(_.name).toSet, checkedIn)
    declared.foreach(_.construct())
  }
