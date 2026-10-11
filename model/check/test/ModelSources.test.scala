package umpire.check

import java.nio.file.Files

class ModelSourcesSuite extends munit.FunSuite:
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
    assert(!selected.includes(root.resolve("model/temporal/features/kind/beta/Model.scala")))
    val integration = root.resolve("model/temporal/capabilities/Properties.test.scala")
    assert(!selected.includes(integration))
    assertEquals(selected.options, Seq("--exclude", integration.toAbsolutePath.toString))
    assert(selected.includes(root.resolve("model/temporal/Bounds.scala")))
    assert(ModelSources(root, None).includes(integration))
    assertEquals(ModelSources(root, None).options, Seq.empty)
    val failure = intercept[GateError](ModelSources(root, Some("kind/absent")))
    assert(failure.getMessage.contains("kind/absent"))
    val held = Files.createDirectories(root.resolve("ir"))
    val lifted = Files.createDirectories(root.resolve("lifted"))
    Files.writeString(held.resolve("kind-alpha.json"), "before"): Unit
    Files.writeString(lifted.resolve("kind-alpha.json"), "before"): Unit
    val full = ModelSources(root, None)
    assertEquals(
      full.states(held, lifted),
      Vector("kind/alpha" -> "current", "kind/beta" -> "missing")
    )
    Files.writeString(lifted.resolve("kind-alpha.json"), "after"): Unit
    assertEquals(selected.states(held, lifted), Vector("kind/alpha" -> "behind"))

  test("narrowed settlement checks only produced files and leaves unlisted files untouched"):
    val root = Scratch.directory("selected-ir-settlement")
    val tree = Files.createDirectories(root.resolve("ir"))
    val produced = Files.createDirectories(root.resolve("produced"))
    Files.writeString(tree.resolve("unlisted.json"), "unlisted\n")
    Files.writeString(produced.resolve("selected.json"), "selected\n")
    Gate.settle(tree, produced, root, update = true, selected = true)
    assertEquals(Files.readString(tree.resolve("unlisted.json")), "unlisted\n")
    assertEquals(Files.readString(tree.resolve("selected.json")), "selected\n")
    Gate.settle(tree, produced, root, update = false, selected = true)
    val failure = intercept[GateError](Gate.settle(tree, produced, root, update = false))
    assert(failure.getMessage.contains("unlisted.json"))
