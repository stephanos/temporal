package framework.check

import java.nio.file.Files

class ModelSourcesSuite extends munit.FunSuite:
  test("compiler reification is tooling rather than authored Model source"):
    for path <- Seq(
        "model/framework/reify/body/BodySyntax.scala",
        "model/framework/reify/plugin/ReifyPlugin.scala"
      )
    do assert(ModelSources.tool(path), path)
    assert(!ModelSources.tool("model/framework/realize/Conversion.scala"))

  test("flat features join legacy forms automatically without selecting shared folders"):
    val root = Scratch.directory("flat-model-source-selection")
    def file(name: String, content: String = "package temporal\n"): Unit =
      val path = root.resolve("model/temporal/features/" + name)
      Files.createDirectories(path.getParent): Unit
      Files.writeString(path, content): Unit
    file("activity_mini/MiniActivity.scala")
    file("activity/Activity.scala")
    file("activity/standalone/StandaloneActivity.scala")
    file("activity/shared/Shared.scala")
    file("kind/Kind.scala")
    file("kind/alpha/AlphaKind.scala")
    file("kind/shared/Shared.scala", "import temporal.features.activity_mini.Model\n")
    file("kind/product/Product.scala")
    file("kind/protocol/Protocol.scala")
    file("kind/generated/Generated.scala")
    file("shared/Shared.scala")
    file("unrelated/Helper.scala")
    file("unrelated_form/Helper.scala")
    val mini = ModelSources(root, Some("activity_mini"))
    assertEquals(mini.discovered, Vector("activity/standalone", "activity_mini", "kind/alpha"))
    assert(mini.sources.contains("model/temporal/features/activity_mini"))
    assert(!mini.sources.contains("model/temporal/features/kind"))
    assert(mini.sources.contains("model/temporal/features/activity/Activity.scala"))
    assert(mini.sources.contains("model/temporal/features/activity/shared"))
    assert(!mini.sources.contains("model/temporal/features/activity/standalone"))
    assert(!mini.sources.contains("model/temporal/features/unrelated"))
    val legacy = ModelSources(root, Some("kind/alpha"))
    assert(legacy.sources.contains("model/temporal/features/kind/shared"))
    assert(legacy.sources.contains("model/temporal/features/kind/product"))
    assert(!legacy.sources.contains("model/temporal/features/activity_mini"))
    assert(legacy.includes(root.resolve("model/temporal/features/kind/shared/Shared.scala")))
    assertEquals(legacy.options, Seq.empty)
    file("fresh/Fresh.scala", "import temporal.features.activity_mini.Model\n")
    val full = ModelSources(root, None)
    assertEquals(
      full.features,
      Vector("activity/standalone", "activity_mini", "fresh", "kind/alpha")
    )
    assert(full.sources.contains("model/temporal/features/fresh"))
    val fresh = ModelSources(root, Some("fresh"))
    assert(fresh.includes(root.resolve("model/temporal/features/fresh/Fresh.scala")))
    assertEquals(fresh.options, Seq.empty)
    for name <- Seq("absent", "unrelated", "unrelated_form", "shared", "kind/shared") do
      val failure = intercept[GateError](ModelSources(root, Some(name)))
      assert(failure.getMessage.contains(name))
    Seq("ActivityMini", "Activity_mini", "Mini_Activity").foldLeft(
      root.resolve("model/temporal/features/activity_mini/MiniActivity.scala")
    ): (header, name) =>
      val renamed = Files.move(header, header.resolveSibling(name + ".scala"))
      assertEquals(ModelSources(root, Some("activity_mini")).discovered, full.discovered)
      renamed

  test("discovery preserves prerequisites and excludes unlisted integration tests"):
    val root = Scratch.directory("model-source-selection")
    def file(name: String, content: String = "package temporal\n"): Unit =
      val path = root.resolve("model/temporal/" + name)
      Files.createDirectories(path.getParent): Unit
      Files.writeString(path, content): Unit
    file("Bounds.scala", "import temporal.features.kind.beta.Model\n")
    file("IrFiles.test.scala")
    file("features/kind/Kind.scala")
    file("features/kind/alpha/AlphaKind.scala")
    file("features/kind/beta/BetaKind.scala")
    file("features/kind/shared/Shared.scala")
    file("capabilities/Properties.test.scala", "import temporal.features.kind.beta.Model\n")
    val selected = ModelSources(root, Some("kind/alpha"))
    assertEquals(selected.discovered, Vector("kind/alpha", "kind/beta"))
    for source <- Seq(
        "Bounds.scala",
        "IrFiles.test.scala",
        "realize",
        "features/kind/Kind.scala",
        "features/kind/shared",
        "features/kind/alpha"
      )
    do assert(selected.sources.contains("model/temporal/" + source), source)
    assert(!selected.sources.contains("model/temporal/features/kind/beta"))
    assert(!selected.includes(root.resolve("model/temporal/features/kind/beta/Beta.scala")))
    val integration = root.resolve("model/temporal/capabilities/Properties.test.scala")
    assert(!selected.includes(integration))
    assertEquals(selected.options, Seq("--exclude", integration.toAbsolutePath.toString))
    assert(selected.includes(root.resolve("model/temporal/Bounds.scala")))
    assert(ModelSources(root, None).includes(integration))
    assertEquals(ModelSources(root, None).options, Seq.empty)
    val failure = intercept[GateError](ModelSources(root, Some("kind/absent")))
    assert(failure.getMessage.contains("kind/absent"))
    val held = Files.createDirectories(root.resolve("model/temporal/features/kind/alpha/generated"))
    val lifted = Files.createDirectories(root.resolve("lifted"))
    Files.writeString(held.resolve("ir.json"), "before"): Unit
    Files.writeString(lifted.resolve("kind-alpha.json"), "before"): Unit
    val full = ModelSources(root, None)
    assertEquals(
      full.states(lifted),
      Vector("kind/alpha" -> "current", "kind/beta" -> "missing")
    )
    Files.writeString(lifted.resolve("kind-alpha.json"), "after"): Unit
    assertEquals(selected.states(lifted), Vector("kind/alpha" -> "behind"))
