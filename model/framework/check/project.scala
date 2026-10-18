// The checks' framework half: the one runner of external tools, the change-scoped check over a
// system's coverage registry, the source lint rules (lint/) and the step-table equivalence
// (equivalence/). It is tooling: it runs scala-cli, protoc and go and reads and writes files with the
// standard library alone, so it builds before anything it generates exists. No Model imports it, and
// it names no system: Temporal's half, model/temporal/check, adds its gate, its coverage registry and
// the entry points, in a build of both directories, this one first.
// It is built without -Werror: scala-cli exits 0 when -Werror turns a warning into an error and
// then runs the classes it wrote, and nothing stands in front of the gate to read its own build.
//> using scala 3.9.0
//> using file ../../OutputPaths.scala
//> using file ../checker/BudgetPolicy.scala
//> using jvm 27
//> using options -deprecation -feature -unchecked -Wunused:imports -language:strictEquality
//> using test.dep org.scalameta::munit:1.2.0
