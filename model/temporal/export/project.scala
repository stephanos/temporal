// Temporal's half of the lifter. Temporal's lifter is built from both halves, the framework's first:
// `scala-cli <command> model/framework/export model/temporal/export`. The framework's project.scala
// is then the build's own, with its settings, and this file adds the Temporal API's classes, which
// the realizations of the Temporal Models name and the lift reads, and Temporal's coverage registry,
// which one of the lifter's suites reads (model/temporal/check/Coverage.scala).
//> using jar ../../build/api-scalapb.jar
//> using file ../../framework/check/Registry.scala
//> using file ../check/Coverage.scala
