// Build directives for the Scala model layer. scala-cli reads them from any source file; they live
// here so the rest of the tree is plain Scala. The framework in umpire/ builds on its own
// (`scala-cli compile project.scala umpire`); the Temporal Models in temporal/ build on top of it.

//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
//> using jar gen/api-scalapb.jar
//> using test.dep org.scalameta::munit:1.2.0
