package framework.`export`

import java.nio.file.Path

// What a system supplies to the generic lifter: the first hook of its plug-in seam (fn-169.1). The
// seam's completion, fn-169.2 after fn-141.8, moves the layout conventions, the realization kit's
// lifting and the API descriptors behind it too; until then the lifter still names them
// (model/README.md, "Residual Temporal coupling").
//
// A system's lifter build registers its plug-in as a Java service, in a resource
// `META-INF/services/framework.export.Plugin` that names its class; the lifter reads the first one
// on its classpath, or the generic one, which knows no system.
trait Plugin extends framework.lint.ModulePolicy:
  // The package root of the system's Models and of its tools' halves (`temporal`).
  def root: String

  // Whether a source of package `from`, a test source when `test`, may import or name package `to`
  // under the system's module map; None leaves the pair to the framework's own rules.
  def permitted(from: String, to: String, test: Boolean): Option[Boolean]

  // The module a test source is held to, by its absolute path, when it is not its package's.
  def testModule(file: Path): Option[String]

object Plugin:
  lazy val installed: Plugin =
    java.util.ServiceLoader.load(classOf[Plugin]).findFirst.orElse(Generic)

  // No system: the framework's own rules alone.
  object Generic extends Plugin:
    val root = ""
    def permitted(from: String, to: String, test: Boolean): Option[Boolean] = None
    def testModule(file: Path): Option[String] = None
