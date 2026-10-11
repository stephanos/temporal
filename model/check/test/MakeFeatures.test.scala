package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

class MakeFeaturesSuite extends munit.FunSuite:
  test("Case Make entry points run only in selected generated folders"):
    for target <- Seq("umpire-check-cases", "umpire-gen-cases") do
      val root = Scratch.directory("umpire-make-feature-scope")
      Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
      for name <- Seq("framework", "temporal/foundations", "temporal/actors", "temporal/capabilities", "temporal/realize") do
        Files.createDirectories(root.resolve("model/" + name)): Unit
      def feature(name: String): Path =
        val folder = Files.createDirectories(root.resolve("model/temporal/features/" + name))
        Files.writeString(folder.resolve(name.capitalize + ".scala"), "package fixture\n")
        val generated = Files.createDirectories(folder.resolve("generated"))
        Files.writeString(generated.resolve("ir.json"), "{}")
        Files.writeString(generated.resolve("answer.case.json"), "original")
        generated
      val selected = feature("lamp")
      val sibling = feature("sibling").resolve("answer.case.json")
      val siblingTime = Files.getLastModifiedTime(sibling)
      val log = root.resolve("commands.log")
      val bin = Files.createDirectories(root.resolve("bin"))
      Stubs(bin)
        .tool("mise", "shift; shift; exec \"$@\"")
        .tool("go", """case "$PWD" in */features/lamp/generated) ;; *) exit 9;; esac
          |[ -f ir.json ] || exit 8
          |echo "$PWD|$*" >> "$COMMANDS_LOG"
          |case " $* " in *" --update "*) printf published > answer.case.json;; esac
          |""".stripMargin)
      val environment = System.getenv().asScala.toMap ++ Map(
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv("PATH")),
        "COMMANDS_LOG" -> log.toString
      )
      val tools = Tools(root, environment)
      val arguments = Seq(target, "FEATURES=lamp", "GOOS=linux", "GOARCH=amd64", "GOPATH=/tmp")
      val result = tools.run("make", arguments)
      assertEquals(result.exit, 0, result.output)
      val update = if target == "umpire-gen-cases" then " --update" else ""
      assertEquals(Files.readString(log), s"$selected|run ${root.resolve("tools/umpire/cmd/umpire-gen-cases")}$update\n")
      assertEquals(Files.readString(selected.resolve("answer.case.json")), if update.isEmpty then "original" else "published")
      assertEquals(Files.readString(sibling), "original")
      assertEquals(Files.getLastModifiedTime(sibling), siblingTime)
