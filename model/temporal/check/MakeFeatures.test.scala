package temporal.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import framework.check.*

class MakeFeaturesSuite extends munit.FunSuite:
  private val reifyCacheKey =
    "-P:umpire-reify:cache-key=[0-9a-f]{64}".r

  private def reifyCacheKeys(command: String): Vector[String] =
    command
      .split("\\s+")
      .toVector
      .map(token =>
        if token.length >= 2 &&
          ((token.head == '\'' && token.last == '\'') ||
            (token.head == '"' && token.last == '"'))
        then token.substring(1, token.length - 1)
        else token
      )
      .filter(reifyCacheKey.pattern.matcher(_).matches())

  private def reifyInputs(root: Path): Unit =
    for name <- Seq(
        "model/framework/inspect",
        "model/framework/reify/body",
        "model/framework/reify/plugin/resources"
      )
    do Files.createDirectories(root.resolve(name)): Unit
    for (name, content) <- Seq(
        "model/framework/inspect/LiftableSubset.scala" -> "package framework.inspect\n",
        "model/framework/inspect/LiftError.scala" -> "package framework.inspect\n",
        "model/framework/reify/body/BodyContext.scala" -> "package framework.reify\n",
        "model/framework/reify/plugin/project.scala" -> "//> using scala 3.9.0\n",
        "model/framework/reify/plugin/resources/plugin.properties" ->
          "pluginClass=fixture.Plugin\n"
      )
    do Files.writeString(root.resolve(name), content): Unit

  test(
    "the Model jar and transitive classpath come from one build and survive compiler failure"
  ):
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
    reifyInputs(root)
    val folder = Files.createDirectories(
      root.resolve("model/temporal/features/activity_mini")
    )
    Files.writeString(
      folder.resolve("ActivityMini.scala"),
      "package fixture\n"
    ): Unit
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
        |case "$destination" in
        |  *model/build/reify.*.jar)
        |    if [ "${FAIL_PLUGIN:-}" != yes ]; then
        |      { printf 'jar:'; cat model/framework/reify/plugin/project.scala; } > "$destination"
        |    fi;;
        |  ?*) printf jar > "$destination";;
        |esac
        |printf '/classes:/a dependency/api.jar:/transitive/scala.jar\n'
        |[ "${FAIL_BUILD:-}" != yes ]
        |""".stripMargin
      )
    val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
      "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv(
        "PATH"
      )),
      "COMMANDS_LOG" -> log.toString
    )
    val tools = Tools(root, environment)
    val args = Seq(
      "--no-print-directory",
      "model/build/model-scala.jar",
      "FEATURES=activity_mini"
    )
    val result = tools.run("make", args).orFail()
    val commands = Files.readAllLines(log).asScala.toVector
    assertEquals(commands.size, 2, result.output)
    assert(commands.head.contains("--assembly"), commands.head)
    assert(commands.head.contains("--preamble=false"), commands.head)
    assert(
      commands.head.contains("--main-class-in-manifest=false"),
      commands.head
    )
    assert(
      commands.head.contains("--provided org.scala-lang:scala3-compiler_3"),
      commands.head
    )
    assert(
      commands.head.contains("model/framework/reify/plugin"),
      commands.head
    )
    assert(
      commands.head.matches(".*model/build/reify\\.[^.]+\\.jar.*"),
      commands.head
    )
    assert(commands(1).contains("--server=false"), commands(1))
    assert(
      commands(1).contains(
        s"-Xplugin:${root.resolve("model/build/reify.jar")}"
      ),
      commands(1)
    )
    assert(
      commands(1).contains("-P:umpire-reify:mode=realization"),
      commands(1)
    )
    val firstCacheKey = ModelSources
      .reifyOptions(build.resolve("reify.jar"))
      .last
    assertEquals(
      reifyCacheKeys(commands(1)),
      Vector(firstCacheKey),
      commands(1)
    )
    assertEquals(Files.readString(build.resolve("model-scala.jar")), "jar")
    assertEquals(
      Files.readString(build.resolve("model-scala.classpath")),
      "/classes:/a dependency/api.jar:/transitive/scala.jar\n"
    )
    tools.run("make", args).orFail(): Unit
    assertEquals(
      Files.readAllLines(log).size,
      2,
      "a warm Model package reuses both outputs"
    )
    Files.writeString(
      root.resolve("model/framework/reify/plugin/project.scala"),
      "//> using scala 3.9.0\n// changed plugin bytes\n"
    ): Unit
    Files.setLastModifiedTime(
      build.resolve("reify.jar"),
      java.nio.file.attribute.FileTime
        .fromMillis(System.currentTimeMillis() - 4000)
    ): Unit
    tools.run("make", args).orFail(): Unit
    assertEquals(
      Files.readAllLines(log).size,
      4,
      "a changed plugin input rebuilds the plugin and then the Models"
    )
    val changedCommand = Files.readAllLines(log).get(3)
    val secondCacheKey = ModelSources
      .reifyOptions(build.resolve("reify.jar"))
      .last
    assertEquals(
      reifyCacheKeys(changedCommand),
      Vector(secondCacheKey),
      changedCommand
    )
    assertNotEquals(secondCacheKey, firstCacheKey)
    Files.writeString(build.resolve("model-scala.jar"), "previous jar"): Unit
    Files.writeString(
      build.resolve("model-scala.classpath"),
      "previous classpath"
    ): Unit
    val previousPlugin = Files.readAllBytes(build.resolve("reify.jar"))
    Files.setLastModifiedTime(
      folder.resolve("ActivityMini.scala"),
      java.nio.file.attribute.FileTime
        .fromMillis(System.currentTimeMillis + 2000)
    ): Unit
    val failed = tools.withEnvironment("FAIL_BUILD" -> "yes").run("make", args)
    assert(failed.failed, failed.output)
    assertEquals(Files.readAllLines(log).size, 5, failed.output)
    assertEquals(
      Files.readString(build.resolve("model-scala.jar")),
      "previous jar"
    )
    assertEquals(
      Files.readString(build.resolve("model-scala.classpath")),
      "previous classpath"
    )
    assert(
      Files
        .readAllBytes(build.resolve("reify.jar"))
        .sameElements(previousPlugin)
    )

    Files.setLastModifiedTime(
      root.resolve("model/framework/reify/plugin/project.scala"),
      java.nio.file.attribute.FileTime
        .fromMillis(System.currentTimeMillis() + 4000)
    ): Unit
    val missingPlugin =
      tools.withEnvironment("FAIL_PLUGIN" -> "yes").run("make", args)
    assert(missingPlugin.failed, missingPlugin.output)
    assert(
      missingPlugin.output.contains(
        "model/build/reify.jar is missing: scala-cli did not package the reify compiler plugin"
      ),
      missingPlugin.output
    )
    assertEquals(Files.readAllLines(log).size, 6, missingPlugin.output)
    assert(
      Files
        .readAllBytes(build.resolve("reify.jar"))
        .sameElements(previousPlugin)
    )
    assertEquals(
      Files.readString(build.resolve("model-scala.jar")),
      "previous jar"
    )
    assertEquals(
      Files.readString(build.resolve("model-scala.classpath")),
      "previous classpath"
    )

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
        val folder = Files.createDirectories(
          root.resolve("model/temporal/features/" + name)
        )
        Files.writeString(
          folder.resolve(name.capitalize + ".scala"),
          "package fixture\n"
        )
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
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv(
          "PATH"
        )),
        "COMMANDS_LOG" -> log.toString
      )
      val tools = Tools(root, environment)
      val arguments = Seq(
        target,
        "FEATURES=lamp",
        "GOOS=linux",
        "GOARCH=amd64",
        "GOPATH=/tmp"
      )
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
    for
      (target, kind, update) <- targets;
      selection <- Seq(Some("lamp"), Some("fresh"), None)
    do
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
        val folder = Files.createDirectories(
          root.resolve("model/temporal/features/" + name)
        )
        Files.writeString(
          folder.resolve(name.capitalize + ".scala"),
          "package fixture\n"
        )
        if lifted then
          val generated = Files.createDirectories(folder.resolve("generated"))
          Files.writeString(generated.resolve("ir.json"), "{}")
          Files.writeString(generated.resolve("answer.case.json"), "original")
        folder
      val selected = feature("lamp", lifted = true)
      val sibling = feature("sibling", lifted = true)
      val fresh = feature("fresh", lifted = false)
      val owned = Seq(selected, sibling).flatMap(folder =>
        Seq(
          folder.resolve("generated/ir.json"),
          folder.resolve("generated/answer.case.json")
        )
      ) ++ Seq(
        selected.resolve("Lamp.scala"),
        sibling.resolve("Sibling.scala"),
        fresh.resolve("Fresh.scala")
      )
      val before =
        owned
          .map(file => file -> (Files.readString(file), Files.getLastModifiedTime(file)))
          .toMap
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
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv(
          "PATH"
        )),
        "COMMANDS_LOG" -> log.toString
      )
      val tools = Tools(root, environment)
      val options = Seq("GOOS=linux", "GOARCH=amd64", "GOPATH=/tmp")
      val arguments =
        Seq(target) ++ options ++ selection.toSeq.map("FEATURES=" + _)
      val result = tools.run("make", arguments)
      assertEquals(result.exit, 0, s"$target $selection: ${result.output}")
      val features =
        selection.fold(ModelSources(root, None).features)(Vector(_))
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
        val invalid =
          tools.run("make", Seq(target, "FEATURES=absent") ++ options)
        assertEquals(invalid.exit, 2)
        assert(
          invalid.output.contains("feature folder not found: absent"),
          invalid.output
        )
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
    reifyInputs(root)
    val folder = Files.createDirectories(
      root.resolve("model/temporal/features/activity_mini")
    )
    Files.writeString(folder.resolve("ActivityMini.scala"), "package fixture\n")
    val build = Files.createDirectories(root.resolve("model/build"))
    val log = root.resolve("commands.log")
    val bin = Files.createDirectories(root.resolve("bin"))
    Stubs(bin)
      .tool("mise", "shift; shift; exec \"$@\"")
      .tool(
        "scala-cli",
        """echo "scala-cli $*" >> "$COMMANDS_LOG"
          |destination=
          |while [ "$#" -gt 0 ]; do
          |  case "$1" in -o) shift; destination=$1;; esac
          |  shift
          |done
          |[ -z "$destination" ] || printf jar > "$destination"
          |""".stripMargin
      )
    val environment = (System.getenv().asScala.toMap - "FEATURES") ++ Map(
      "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv(
        "PATH"
      )),
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
        missing.output.contains(
          "ir-scalapb.jar is missing; run make gen-model-api"
        ),
        missing.output
      )
      assert(!Files.exists(log), target.mkString(" "))
    // With both jars cached, their inputs being newer does not regenerate them.
    for jar <- Seq("ir-scalapb.jar", "api-scalapb.jar") do
      Files.writeString(build.resolve(jar), "cached")
    Files.createDirectories(root.resolve("proto"))
    Files.writeString(root.resolve("proto/api.binpb"), "newer")
    val compiled =
      tools.run("make", Seq("compile-model", "FEATURES=activity_mini"))
    assertEquals(compiled.exit, 0, compiled.output)
    val ran = Files.readString(log)
    assert(ran.contains("scala-cli --power package "), ran)
    assert(ran.contains("scala-cli compile "), ran)
    assert(
      ran.contains(s"-Xplugin:${root.resolve("model/build/reify.jar")}"),
      ran
    )
    assert(ran.contains("-P:umpire-reify:mode=realization"), ran)
    val expectedCacheKey = ModelSources
      .reifyOptions(build.resolve("reify.jar"))
      .last
    val pluginCommands = ran.linesIterator
      .filter(_.contains("-Xplugin:"))
      .toVector
    assert(pluginCommands.nonEmpty, ran)
    for command <- pluginCommands do
      assertEquals(
        reifyCacheKeys(command),
        Vector(expectedCacheKey),
        command
      )
    assert(!ran.contains("--generate-"), ran)
    assertEquals(Files.readString(build.resolve("api-scalapb.jar")), "cached")

  test(
    "a no-scope change check starts without packaging Model compiler inputs"
  ):
    val root = Scratch.directory("umpire-make-no-scope")
    Files.copy(Path.of("Makefile").toAbsolutePath, root.resolve("Makefile"))
    val log = root.resolve("commands.log")
    val bin = Files.createDirectories(root.resolve("bin"))
    Stubs(bin)
      .tool("mise", "shift; shift; exec \"$@\"")
      .tool("scala-cli", "echo \"$*\" >> \"$COMMANDS_LOG\"")
    val tools = Tools(
      root,
      System.getenv().asScala.toMap ++ Map(
        "PATH" -> (bin.toString + java.io.File.pathSeparator + System.getenv(
          "PATH"
        )),
        "COMMANDS_LOG" -> log.toString
      )
    )
    val checked = tools.run(
      "make",
      Seq("--no-print-directory", "model-check-changed", "BASE=HEAD")
    )
    assertEquals(checked.exit, 0, checked.output)
    val commands = Files.readAllLines(log).asScala.toVector
    assertEquals(commands.size, 1, commands.mkString("\n"))
    assert(commands.head.contains("temporal.check.changed"), commands.head)
    assert(!commands.head.contains("--assembly"), commands.head)
    assert(!commands.head.contains("model/build/reify.jar"), commands.head)

  test(
    "model Make callers preserve compiler rejection for headers and direct -Werror builds"
  ):
    val expectedReifyDigest =
      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    val expectedCacheKey =
      s"-P:umpire-reify:cache-key=$expectedReifyDigest"
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
      .run(
        "make",
        Seq(
          "--no-print-directory",
          "--dry-run",
          "FEATURES=activity_mini",
          s"MODEL_REIFY_DIGEST=$expectedReifyDigest"
        ) ++ targets
      )
      .orFail()
    val commands =
      result.output
        .replace("\\\n", " ")
        .linesIterator
        .filter(_.contains("scala-cli "))
        .toVector
    assert(commands.nonEmpty, result.output)
    assert(commands.exists(_.contains("temporal.check.changed")), result.output)
    val nativeTests = commands.find(_.contains("framework.check.nativeModelTests")).get
    assert(!nativeTests.contains("-Xplugin:"), nativeTests)
    assert(commands.exists(_.contains("irgen-conventions.jar")), result.output)
    assert(
      commands.exists(_.contains("--test model/framework/dsl")),
      result.output
    )
    val modelCommands = commands.filter(_.contains("model/project.scala"))
    assert(modelCommands.nonEmpty, result.output)
    val pluginModelCommands = modelCommands.filter(_.contains("-Xplugin:"))
    assert(pluginModelCommands.nonEmpty, result.output)
    for command <- pluginModelCommands do
      assert(command.contains("-P:umpire-reify:mode=realization"), command)
      assertEquals(
        reifyCacheKeys(command),
        Vector(expectedCacheKey),
        command
      )
    val nativeModelCommands = modelCommands.filterNot(_.contains("-Xplugin:"))
    assert(nativeModelCommands.nonEmpty, result.output)
    for command <- nativeModelCommands do
      assert(
        command.contains("model-conventions") || command.contains("scalafix"),
        command
      )
    val frameworkCommands = commands.filter(command =>
      !command.contains("model/project.scala") &&
        (command.contains("model/framework/export") ||
          command.contains("model/framework/check") ||
          command.contains("model/framework/dsl") ||
          command.contains(
            "scala-cli --power package --server=false --assembly"
          ))
    )
    for command <- frameworkCommands do assert(!command.contains("-Xplugin:"), command)
    assert(
      commands.exists(command =>
        command.contains("RealizationConversionTest") && !command.contains(
          "-Xplugin:"
        )
      ),
      result.output
    )
    for command <- commands do
      val normal = command.contains("model/project.scala") ||
        (command.contains("scala-cli compile ") && command.contains(
          "--test model/framework/check"
        ) &&
          !command.contains("--print-classpath")) ||
        (command.contains("scala-cli --power fix ") && command.contains("model/framework/check")) ||
        (command.contains("scala-cli run ") && command.contains("model/framework/check")) ||
        command.contains("framework.check.nativeModelTests") ||
        command.contains(
          "scala-cli --power package --server=false --assembly"
        ) ||
        (!command.contains("scala-cli --power fix") &&
          (command.contains("model/framework/export") ||
            command.contains("model/framework/dsl") ||
            command.contains("model/framework/reify/plugin")))
      assertEquals(
        command.contains("--server=false"),
        normal,
        command
      )
