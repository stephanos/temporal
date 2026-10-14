package framework.lint

import java.nio.file.Path

trait ModulePolicy:
  def root: String
  def permitted(from: String, to: String, test: Boolean): Option[Boolean]
  def testModule(file: Path): Option[String]

object ModulePolicy:

  object Generic extends ModulePolicy:
    val root = ""
    def permitted(from: String, to: String, test: Boolean): Option[Boolean] = None
    def testModule(file: Path): Option[String] = None
