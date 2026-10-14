package temporal.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import framework.check.*

class MakeFeaturesSuite extends munit.FunSuite:
  test("the Model jar and transitive classpath come from one build and survive compiler failure"):
    val root = Scratch.directory("umpire-make-one-build")
    Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
    for name <- Seq(
        "framework",
        "temporal/foundations",
        "temporal/actors",
        "temporal/capabilities",
        "temporal/realize"
      )
    do Files.createDirectories(root.resolve("model/" + name)): Unit
    val folder = Files.createDirectories(root.resolve("model/temporal/features/activity_mini"))
    Files.writeString(folder.resolve("ActivityMini.scala"), "package fixture\n"): Unit
    val build = Files.createDirectories(root.resolve("model/build"))
    for name <- Seq("ir-scalapb.jar", "api-scalapb.jar") do
      Files.writeString(build.resolve(name), "cached"): Unit
    val log = root.resolve("commands.log")
    val bin = Files.createDirectories(root.resolve("bin"))
    Stubs(bin)
      .tool("mise", "shift; shift; exec \"$@\"")
      .tool(
        "scala-cli",
        """echo "$*" >> "$COMMANDS_LOG"
        |destination=
        |while [ "$#" -gt 0 ]; do
        |  case "$1" in -o|--compilation-output) shift; destination=$1;; esac
        |  shift
        |done
        |[ -z "$destination" ] || printf jar > "$destination"
        |printf '/classes:/a dependency/api.jar:/transitive/scala.jar\n'
        |[ "${FAIL_BUILD:-}" != yes ]
        |""".stripMargin
      )
    val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
      "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv("PATH")),
      "COMMANDS_LOG" -> log.toString
    )
    val tools = Tools(root, environment)
    val args = Seq("--no-print-directory", "model/build/model-scala.jar", "FEATURES=activity_mini")
    val result = tools.run("make", args).orFail()
    val commands = Files.readAllLines(log).asScala.toVector
    assertEquals(commands.size, 1, result.output)
    assert(commands.head.contains("--server=false"), commands.head)
    assertEquals(Files.readString(build.resolve("model-scala.jar")), "jar")
    assertEquals(
      Files.readString(build.resolve("model-scala.classpath")),
      "/classes:/a dependency/api.jar:/transitive/scala.jar\n"
    )
    Files.writeString(build.resolve("model-scala.jar"), "previous jar"): Unit
    Files.writeString(build.resolve("model-scala.classpath"), "previous classpath"): Unit
    Files.setLastModifiedTime(
      folder.resolve("ActivityMini.scala"),
      java.nio.file.attribute.FileTime.fromMillis(System.currentTimeMillis + 2000)
    ): Unit
    val failed = tools.withEnvironment("FAIL_BUILD" -> "yes").run("make", args)
    assert(failed.failed, failed.output)
    assertEquals(Files.readAllLines(log).size, 2, failed.output)
    assertEquals(Files.readString(build.resolve("model-scala.jar")), "previous jar")
    assertEquals(Files.readString(build.resolve("model-scala.classpath")), "previous classpath")

  test("Case Make entry points run only in selected generated folders"):
    for target <- Seq("umpire-check-cases", "umpire-gen-cases") do
      val root = Scratch.directory("umpire-make-feature-scope")
      Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
      for name <- Seq(
          "framework",
          "temporal/foundations",
          "temporal/actors",
          "temporal/capabilities",
          "temporal/realize"
        )
      do Files.createDirectories(root.resolve("model/" + name)): Unit
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
        .tool(
          "go",
          """case "$PWD" in */features/lamp/generated) ;; *) exit 9;; esac
          |[ -f ir.json ] || exit 8
          |echo "$PWD|$*" >> "$COMMANDS_LOG"
          |case " $* " in *" --update "*) printf published > answer.case.json;; esac
          |""".stripMargin
        )
      val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv("PATH")),
        "COMMANDS_LOG" -> log.toString
      )
      val tools = Tools(root, environment)
      val arguments = Seq(target, "FEATURES=lamp", "GOOS=linux", "GOARCH=amd64", "GOPATH=/tmp")
      val result = tools.run("make", arguments)
      assertEquals(result.exit, 0, result.output)
      val update = if target == "umpire-gen-cases" then " --update" else ""
      assertEquals(
        Files.readString(log),
        s"$selected|run ${root.resolve("tools/umpire/cmd/umpire-gen-cases")}$update\n"
      )
      assertEquals(
        Files.readString(selected.resolve("answer.case.json")),
        if update.isEmpty then "original" else "published"
      )
      assertEquals(Files.readString(sibling), "original")
      assertEquals(Files.getLastModifiedTime(sibling), siblingTime)

  test(
    "pinned Make entry points scope check/update to source owners, including unlifted no-pin features"
  ):
    val targets = Seq(
      ("umpire-check-fixtures", "functional", false),
      ("umpire-gen-fixtures", "functional", true),
      ("canary-check-case", "canary", false),
      ("canary-gen-case", "canary", true)
    )
    for (target, kind, update) <- targets; selection <- Seq(Some("lamp"), Some("fresh"), None) do
      val root = Scratch.directory("umpire-make-pinned-scope")
      Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
      for name <- Seq(
          "framework",
          "temporal/foundations",
          "temporal/actors",
          "temporal/capabilities",
          "temporal/realize"
        )
      do Files.createDirectories(root.resolve("model/" + name)): Unit
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
      ) ++ Seq(
        selected.resolve("Lamp.scala"),
        sibling.resolve("Sibling.scala"),
        fresh.resolve("Fresh.scala")
      )
      val before =
        owned.map(file => file -> (Files.readString(file), Files.getLastModifiedTime(file))).toMap
      val log = root.resolve("commands.log")
      val bin = Files.createDirectories(root.resolve("bin"))
      Stubs(bin)
        .tool("mise", "shift; shift; exec \"$@\"")
        .tool(
          "go",
          """case "$PWD" in */features/lamp|*/features/sibling|*/features/fresh) ;; *) exit 9;; esac
          |echo "$PWD|$*" >> "$COMMANDS_LOG"
          |""".stripMargin
        )
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
      val expected = features
        .map(name =>
          val folder = root.resolve("model/temporal/features/" + name)
          s"$folder|run ${root.resolve("tools/umpire/cmd/umpire-gen-cases")} --kind $kind${
              if update then " --update" else ""
            }\n"
        )
        .mkString
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

  test(
    "model Make targets read cached jars, never generate them, and name gen-model-api when one is missing"
  ):
    val root = Scratch.directory("umpire-make-cached-api")
    Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
    for name <- Seq(
        "framework",
        "temporal/foundations",
        "temporal/actors",
        "temporal/capabilities",
        "temporal/realize"
      )
    do Files.createDirectories(root.resolve("model/" + name)): Unit
    val folder = Files.createDirectories(root.resolve("model/temporal/features/activity_mini"))
    Files.writeString(folder.resolve("ActivityMini.scala"), "package fixture\n")
    val build = Files.createDirectories(root.resolve("model/build"))
    val log = root.resolve("commands.log")
    val bin = Files.createDirectories(root.resolve("bin"))
    Stubs(bin)
      .tool("mise", "shift; shift; exec \"$@\"")
      .tool("scala-cli", "echo \"scala-cli $*\" >> \"$COMMANDS_LOG\"")
    val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
      "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv("PATH")),
      "COMMANDS_LOG" -> log.toString
    )
    val tools = Tools(root, environment)
    val targets = Seq(
      Seq("model-mini"),
      Seq("model"),
      Seq("compile-model"),
      Seq("lift-model"),
      Seq("umpire-check-equivalence", "BASE=b", "MAP=m")
    )
    for target <- targets do
      val missing = tools.run("make", target :+ "FEATURES=activity_mini")
      assertEquals(missing.exit, 2, missing.output)
      assert(
        missing.output.contains("ir-scalapb.jar is missing; run make gen-model-api"),
        missing.output
      )
      assert(!Files.exists(log), target.mkString(" "))
    // With both jars cached, their inputs being newer does not regenerate them.
    for jar <- Seq("ir-scalapb.jar", "api-scalapb.jar") do
      Files.writeString(build.resolve(jar), "cached")
    Files.createDirectories(root.resolve("proto"))
    Files.writeString(root.resolve("proto/api.binpb"), "newer")
    val compiled = tools.run("make", Seq("compile-model", "FEATURES=activity_mini"))
    assertEquals(compiled.exit, 0, compiled.output)
    val ran = Files.readString(log)
    assert(ran.startsWith("scala-cli compile "), ran)
    assert(!ran.contains("--generate-"), ran)
    assertEquals(Files.readString(build.resolve("api-scalapb.jar")), "cached")

  test("model Make callers preserve compiler rejection for headers and direct -Werror builds"):
    val targets = Seq(
      "compile-model",
      "test-model",
      "model",
      "model-check-changed",
      "lift-model",
      "lint-model-conventions",
      "lint-model-models",
      "fix-model",
      "check-model-boundary"
    )
    val result = Tools.here
      .withoutEnvironment("FEATURES", "MAKEFLAGS", "MFLAGS")
      .run("make", Seq("--no-print-directory", "--dry-run", "FEATURES=activity_mini") ++ targets)
      .orFail()
    val commands = result.output.linesIterator.filter(_.contains("scala-cli ")).toVector
    assert(commands.nonEmpty, result.output)
    assert(commands.exists(_.contains("temporal.check.changed")), result.output)
    assert(commands.exists(_.contains("irgen-conventions.jar")), result.output)
    assert(commands.exists(_.contains("--test model/framework/dsl")), result.output)
    for command <- commands do
      val normal = command.contains("model/project.scala") ||
        (!command.contains("scala-cli --power fix") &&
          (command.contains("model/framework/export") || command.contains("model/framework/dsl")))
      assertEquals(
        command.contains("--server=false"),
        normal,
        command
      )
