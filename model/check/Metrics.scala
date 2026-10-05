// The source metrics of a Model: its lines and its classified string literals (see
// SourceMetrics.scala). The check's second entry point, beside its main class `run`:
//
//   scala-cli run model/check --main-class umpire.check.metrics -- \
//     model/temporal/features/standaloneactivity [more directories]
package umpire.check

@main def metrics(directories: String*): Unit =
  val root = Tools.here.directory
  if directories.isEmpty then
    System.err.println(
      "usage: scala-cli run model/check --main-class umpire.check.metrics -- <directory>..."
    )
    sys.exit(2)
  print(SourceMetrics.report(root, directories.map(d => root.resolve(d).normalize)))
