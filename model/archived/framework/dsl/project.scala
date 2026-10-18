// The framework's Model-facing core: the DSL here and the generic realization core in ../realize.
// It builds on its own, with no system's API or Models on its classpath
// (`scala-cli compile model/framework/dsl model/framework/realize`); a system's Models build on it
// with their own settings (Temporal's: model/project.scala).
//> using scala 3.9.0
//> using file ../../OutputPaths.scala
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns -Yretain-trees -language:strictEquality
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
//> using jar ../../build/ir-scalapb.jar
//> using dep org.scalameta::munit:1.2.0
