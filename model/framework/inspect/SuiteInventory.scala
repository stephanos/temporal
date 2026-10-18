package framework.inspect

import java.nio.file.Path

private[framework] object SuiteInventory:
  case class Artifact(path: Path, sha256: String)
  case class Source(path: Path, sha256: String)
  case class Toolchain(scalaVersion: String, javaVersion: String)
  case class Request(
      module: String,
      candidates: Vector[Artifact],
      hierarchy: Vector[Artifact],
      trusted: Vector[Artifact],
      sources: Vector[Source],
      skippedRoots: Vector[Path],
      compilerFlags: Vector[String],
      toolchain: Toolchain
  )
  case class Diagnostic(message: String)
  case class Result(module: String)

  def inspect(request: Request): Either[Diagnostic, Result] =
    Right(Result(request.module))
