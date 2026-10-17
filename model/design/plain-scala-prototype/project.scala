// fn-190.1 probe: the plain-Scala Model form on a copy of Kettle and of the Facets fixture, run
// natively. A throwaway prototype with its own minimal DSL (package `plain`); it shares no source
// with model/framework. Same compiler, JVM and strict flags as model/framework/dsl/project.scala.
//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns -language:strictEquality
//> using test.dep org.scalameta::munit:1.2.0
