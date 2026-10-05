// The gate is tooling: it runs scala-cli, protoc and go and reads and writes files with the standard
// library alone, so it builds before anything it generates exists. No Model imports it.
// It is built without -Werror: scala-cli exits 0 when -Werror turns a warning into an error and
// then runs the classes it wrote, and nothing stands in front of the gate to read its own build.
//> using scala 3.9.0
//> using jvm 27
//> using options -deprecation -feature -unchecked -Wunused:imports
// Two entry points: `run`, the gate, and `metrics` (Metrics.scala), which scala-cli runs with
// `--main-class umpire.check.metrics`.
//> using mainClass umpire.check.run
//> using test.dep org.scalameta::munit:1.2.0
