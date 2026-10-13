package framework.lint

import framework.inspect.{Index, LiftError}

import java.nio.file.{Files, Path}
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}

final private[framework] class ConventionInspector(
    prefixes: Map[String, String],
    policy: ModulePolicy
) extends Inspector:
  val errors = mutable.ArrayBuffer.empty[LiftError]

  def inspect(using quotes: Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    errors ++= Conventions(Index(tastys, prefixes), policy)
      .refusals(_ => false)
      .sortBy((file, line, error) => (file, line, error.message))
      .map(_._3)
      .distinct

// Framework and tests participate in lint without becoming semantic lifting inputs.
def lintConventions(jars: String, classpathFiles: Seq[String], policy: ModulePolicy): Unit =
  val specs = jars
    .split(",")
    .toList
    .map(_.split("=", 2) match
      case Array(jar, prefix) => jar -> prefix
      case Array(jar)         => jar -> "")
  val scratch = framework.check.Scratch.directory("umpire-conventions")
  val prefixes = mutable.Map.empty[String, String]
  val batches = specs.zipWithIndex.map { case ((jar, prefix), i) =>
    val zip = ZipFile(jar)
    try
      zip.entries.asScala
        .filter(_.getName.endsWith(".tasty"))
        .map { e =>
          val entry = s"$i/${e.getName}"
          val path = scratch.resolve(entry)
          Files.createDirectories(path.getParent): Unit
          val input = zip.getInputStream(e)
          try Files.copy(input, path): Unit
          finally input.close()
          prefixes(entry) = prefix
          path.toString
        }
        .toList
        .sorted
    finally zip.close()
  }
  if batches.exists(_.isEmpty) then
    System.err.println("conventions: no source TASTy to inspect")
    sys.exit(1)
  val classpath = specs.map(_._1) ++ classpathFiles.flatMap(file =>
    Files.readString(Path.of(file)).trim.split(java.io.File.pathSeparator).toList
  )
  val failures = batches.map { tastys =>
    val checker = ConventionInspector(prefixes.toMap, policy)
    val inspected = TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(checker)
    for error <- checker.errors do System.err.println(s"conventions: ${error.getMessage}")
    !inspected || checker.errors.nonEmpty
  }
  if failures.exists(identity) then sys.exit(1)
