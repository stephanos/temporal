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
      .find(p => Files.isDirectory(p.resolve("model/go/parity/testdata/lean")))
      .getOrElse(sys.error("no model/go/parity/testdata/lean above the working directory"))

  def text(name: String): String = Files.readString(root.resolve("model/go/parity/testdata/lean").resolve(name))
  def json(name: String): JsonElement = JsonParser.parseString(text(name))

  extension (e: JsonElement)
    def apply(field: String): JsonElement = e.getAsJsonObject.get(field)
    def str: String = e.getAsString
    def strOpt: Option[String] = Option(e).filterNot(_.isJsonNull).map(_.getAsString)
    def arr: Vector[JsonElement] = e.getAsJsonArray.asScala.toVector
    def strs: Vector[String] = arr.map(_.getAsString)
    def has(field: String): Boolean = e.getAsJsonObject.has(field)
