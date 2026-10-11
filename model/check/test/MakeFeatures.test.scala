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
      val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
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

  test("pinned Make entry points scope check/update to source owners, including unlifted no-pin features"):
    val targets = Seq(
      ("umpire-check-fixtures", "functional", false),
      ("umpire-gen-fixtures", "functional", true),
      ("canary-check-case", "canary", false),
      ("canary-gen-case", "canary", true)
    )
    for (target, kind, update) <- targets; selection <- Seq(Some("lamp"), Some("fresh"), None) do
      val root = Scratch.directory("umpire-make-pinned-scope")
      Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
      for name <- Seq("framework", "temporal/foundations", "temporal/actors", "temporal/capabilities", "temporal/realize") do
        Files.createDirectories(root.resolve("model/" + name)): Unit
      def feature(name: String, lifted: Boolean): Path =
        val folder = Files.createDirectories(root.resolve("model/temporal/features/" + name))
        Files.writeString(folder.resolve(name.capitalize + ".scala"), "package fixture\n")
        if lifted then
          val generated = Files.createDirectories(folder.resolve("generated"))
          Files.writeString(generated.resolve("ir.json"), "{}")
          Files.writeString(generated.resolve("answer.case.json"), "original")
        folder
      val selected = feature("lamp", lifted = true)
      val sibling = feature("sibling", lifted = true)
      val fresh = feature("fresh", lifted = false)
      val owned = Seq(selected, sibling).flatMap(folder =>
        Seq(folder.resolve("generated/ir.json"), folder.resolve("generated/answer.case.json"))
      ) ++ Seq(selected.resolve("Lamp.scala"), sibling.resolve("Sibling.scala"), fresh.resolve("Fresh.scala"))
      val before = owned.map(file => file -> (Files.readString(file), Files.getLastModifiedTime(file))).toMap
      val log = root.resolve("commands.log")
      val bin = Files.createDirectories(root.resolve("bin"))
      Stubs(bin)
        .tool("mise", "shift; shift; exec \"$@\"")
        .tool("go", """case "$PWD" in */features/lamp|*/features/sibling|*/features/fresh) ;; *) exit 9;; esac
          |echo "$PWD|$*" >> "$COMMANDS_LOG"
          |""".stripMargin)
      val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv("PATH")),
        "COMMANDS_LOG" -> log.toString
      )
      val tools = Tools(root, environment)
      val options = Seq("GOOS=linux", "GOARCH=amd64", "GOPATH=/tmp")
      val arguments = Seq(target) ++ options ++ selection.toSeq.map("FEATURES=" + _)
      val result = tools.run("make", arguments)
      assertEquals(result.exit, 0, s"$target $selection: ${result.output}")
      val features = selection.fold(ModelSources(root, None).features)(Vector(_))
      val expected = features.map(name =>
        val folder = root.resolve("model/temporal/features/" + name)
        s"$folder|run ${root.resolve("tools/umpire/cmd/umpire-gen-cases")} --kind $kind${if update then " --update" else ""}\n"
      ).mkString
      assertEquals(Files.readString(log), expected)
      if selection.isEmpty then assert(features.contains("fresh"))
      for (file, (content, time)) <- before do
        assertEquals(Files.readString(file), content)
        assertEquals(Files.getLastModifiedTime(file), time)
      assert(!Files.exists(fresh.resolve("generated")))
      if selection.contains("lamp") then
        val calls = Files.readString(log)
        val invalid = tools.run("make", Seq(target, "FEATURES=absent") ++ options)
        assertEquals(invalid.exit, 2)
        assert(invalid.output.contains("feature folder not found: absent"), invalid.output)
        assertEquals(Files.readString(log), calls)
