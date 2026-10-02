package temporal
package testsupport

import com.google.gson.{JsonElement, JsonParser}
import java.nio.file.{Files, Path, Paths}
import scala.jdk.CollectionConverters.*

/** The Lean oracle: the dumps model/go/leandump writes into model/go/parity/testdata/lean, read in
  * place so the Go and Scala experiments compare against one copy. Gson comes with
  * protobuf-java-util, so reading them adds no dependency. */
object Lean:
  /** The repository root, found by walking up from the working directory. */
  lazy val root: Path =
    Iterator.iterate(Paths.get("").toAbsolutePath)(_.getParent).takeWhile(_ != null)
      .find(p => Files.isDirectory(p.resolve("model/go/parity")))
      .getOrElse(sys.error("no model/go/parity above the working directory"))

  /** The dumps are git-ignored and only a Lean build regenerates them, so a checkout without them
    * skips the test that reads one; a dump missing from a present directory still fails. */
  def text(name: String): String =
    val dumps = root.resolve("model/go/parity/testdata/lean")
    munit.Assertions.assume(Files.isDirectory(dumps), s"no Lean dumps in $dumps; model/go/leandump/dump.sh writes them")
    Files.readString(dumps.resolve(name))
  def json(name: String): JsonElement = JsonParser.parseString(text(name))

  extension (e: JsonElement)
    def apply(field: String): JsonElement = e.getAsJsonObject.get(field)
    def str: String = e.getAsString
    def strOpt: Option[String] = Option(e).filterNot(_.isJsonNull).map(_.getAsString)
    def arr: Vector[JsonElement] = e.getAsJsonArray.asScala.toVector
    def strs: Vector[String] = arr.map(_.getAsString)
    def has(field: String): Boolean = e.getAsJsonObject.has(field)
