// The source metrics of a Model: its lines and its classified string literals (see
// ../gate/SourceMetrics.scala). A project of its own, so the gate keeps its one main class:
//
//   scala-cli run model/metrics -- model/temporal/standaloneactivity [more directories]
//> using scala 3.9.0
//> using jvm 27
//> using options -deprecation -feature -unchecked -Wunused:imports
// It compiles the gate's sources it reads literals with, and the gate's main class beside its own.
//> using file ../gate/Gate.scala ../gate/Tools.scala
//> using file ../gate/ProtoLiterals.scala ../gate/SourceMetrics.scala ../gate/SyntaxRule.scala
//> using mainClass umpire.gate.metrics
package umpire.gate

@main def metrics(directories: String*): Unit =
  val root = Tools.here.directory
  if directories.isEmpty then
    System.err.println("usage: scala-cli run model/metrics -- <directory>...")
    sys.exit(2)
  print(SourceMetrics.report(root, directories.map(d => root.resolve(d).normalize)))
