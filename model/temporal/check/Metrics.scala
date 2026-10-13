// The source metrics of a Model: its lines and its classified string literals (see
// SourceMetrics.scala). The check's second entry point, beside its main class `run`:
//
//   scala-cli run model/temporal/check --main-class temporal.check.metrics -- \
//     model/temporal/features/activity/standalone [more directories]
package temporal.check
import framework.check.Tools

@main def metrics(directories: String*): Unit =
  val root = Tools.here.directory
  if directories.isEmpty then
    System.err.println(
      "usage: scala-cli run model/temporal/check --main-class temporal.check.metrics -- <directory>..."
    )
    sys.exit(2)
  print(SourceMetrics.report(root, directories.map(d => root.resolve(d).normalize)))
