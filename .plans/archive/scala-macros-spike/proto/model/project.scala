// Build directives of the Temporal Models. scala-cli reads them from any source file; they live here
// so the rest of the tree is plain Scala. The framework's core (framework/dsl, framework/realize)
// builds on its own with its own settings (framework/dsl/project.scala), with no system's API; the
// Temporal Models in temporal/ build on top of it with the settings below and the Temporal API.

//> using scala 3.9.0
//> using jvm 27
//> using jar build/api-scalapb.jar
//> using jar ../reify.jar
// The reifier reads the bodies of defs outside its argument (effects, helpers, givens).
//> using options -Yretain-trees
// The Models' tests read the checked-in IR's JSON (temporal/IrFiles.test.scala).
//> using test.dep org.json4s::json4s-jackson:4.0.7
