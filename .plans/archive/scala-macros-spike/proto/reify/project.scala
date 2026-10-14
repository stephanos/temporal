// The reifier: Scala 3 macros that lift a step function, guard or rule to Umpire IR expression JSON
// while the Model compiles. Zero dependencies; it emits ProtoJSON-shaped strings that the run-time
// exporter parses into the generated IR classes.
//> using scala 3.9.0
//> using options -deprecation -feature -unchecked
