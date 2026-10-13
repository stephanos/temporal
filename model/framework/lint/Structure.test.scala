package framework.lint

import java.nio.file.Files
import scala.jdk.CollectionConverters.*
import framework.`export`.ModelBuild

class StructureFixtures extends munit.FunSuite with LintFixtures:
  // The template a new feature copies (layout/lamp): a feature with its two levels and a zoom-in,
  // which the lint refuses nothing of and the lifter lifts.
  test("the layout template lifts with no refusal"):
    val tree = materializeTree("layout", excluding = Set("kinds", "r4", "mini"))
    val stream = Files.walk(tree)
    val inventory =
      try
        stream.iterator.asScala
          .filter(Files.isRegularFile(_))
          .map(tree.relativize(_).toString)
          .toList
          .sorted
      finally stream.close()
    assertEquals(
      inventory,
      List(
        "lamp/Lamp.scala",
        "lamp/product/Product.scala",
        "lamp/system/Bulb.scala",
        "lamp/system/Realization.scala",
        "lamp/system/System.scala",
        "project.scala"
      )
    )
    val jar = packaged("layout", tree)
    val explicit = scratch.resolve("layout-explicit.json")
    val direct = lift(
      s"$jar=${stored("layout")},$modelJar=model/",
      modelClasspath.toString,
      explicit.toString,
      "fixture.features.lamp.system.LampRealization$.system"
    )
    assertEquals(direct.exit, 0, direct.diagnostics)
    val out = scratch.resolve("layout-out")
    val result =
      liftIr(out, s"$jar=${stored("layout")},$modelJar=model/", "lamp")
    assertEquals(refused(result), Nil)
    assert(!result.failed, result.diagnostics)
    assertEquals(listed(out), Seq("lamp.json"))
    val lifted = Files.readString(out.resolve("lamp.json"))
    for machine <- Seq(
        "lampProduct",
        "onlyOn",
        "lampSystem",
        "onlyClosed",
        "bulb"
      )
    do
      assert(
        lifted.contains(s"\"$machine\""),
        s"lamp.json lifts no machine $machine"
      )
    val parsed =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(lifted)
    val realization = Option(parsed.get("realizations")).toSeq
      .flatMap(_.elements().asScala.toSeq)
    assertEquals(realization.size, 1)
    val explicitModel =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(explicit))
    assertEquals(parsed.path("realizations"), explicitModel.path("realizations"))
    assertEquals(
      realization.head.get("id").asText(),
      "fixture.features.lamp.system.LampRealization.system"
    )
    assertEquals(realization.head.get("machine").asText(), "lampSystem")
    assertEquals(
      realization.head.get("position").get("file").asText(),
      "model/framework/export/testdata/layout/lamp/system/Realization.scala"
    )

  test("typed generic realization alone requests a file while unrelated values do not"):
    val tree = materializeTree("coreDiscovery")
    val jar = packaged("coreDiscovery", tree)
    val out = scratch.resolve("coreDiscovery-out")
    val result = liftIr(out, s"$jar=${stored("coreDiscovery")},$modelJar=model/")
    assertEquals(result.exit, 0, result.diagnostics)
    assert(
      Files.isRegularFile(out.resolve("coreonly.json")),
      "the realization-only foundation was not collected"
    )
    // The fixture's own file beside every file the packaged Models declare (ModelBuild).
    assertEquals(listed(out), (ModelBuild.files :+ "coreonly.json").sorted)
    assert(!Files.exists(out.resolve("quiet.json")), "an unrelated value requested an actor file")
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val collected = mapper.readTree(Files.readString(out.resolve("coreonly.json")))
    val explicit = scratch.resolve("coreDiscovery-explicit.json")
    val direct = lift(
      s"$jar=${stored("coreDiscovery")},$modelJar=model/",
      modelClasspath.toString,
      explicit.toString,
      "fixture.foundations.coreonly.Runs$.only"
    )
    assertEquals(direct.exit, 0, direct.diagnostics)
    val expected = mapper.readTree(Files.readString(explicit))
    assertEquals(collected.path("realizations"), expected.path("realizations"))
    assertEquals(collected.path("realizations").size(), 1)
    assertEquals(
      collected.path("realizations").get(0).path("id").asText(),
      "fixture.foundations.coreonly.Runs.only"
    )
    assertEquals(collected.path("realizations").get(0).path("machine").asText(), "idle")

  test(
    "kind scaffolds and their one or two forms lift with local or kind Products"
  ):
    val tree = materializeTree("layout/kinds", "kind-layout")
    val jar = packaged("kind-layout", tree)
    val out = scratch.resolve("kind-layout-out")
    val result = liftIr(
      out,
      s"$jar=$tree/,$modelJar=model/",
      "activity-standalone",
      "nexus-standalone",
      "nexus-workflow",
      "relay-standalone",
      "relay-workflow"
    )
    assertEquals(refused(result), Nil)
    assert(!result.failed, result.diagnostics)
    // The fixture's own files: none states a waiver (fn-166.11; the waivers files were the real
    // features', which shared these files' names while the packaged Models held them).
    assertEquals(
      listed(out),
      Seq(
        "activity-standalone.json",
        "nexus-standalone.json",
        "nexus-workflow.json",
        "relay-standalone.json",
        "relay-workflow.json"
      )
    )
    for form <- Seq("standalone", "workflow") do
      val json = new com.fasterxml.jackson.databind.ObjectMapper()
        .readTree(Files.readString(out.resolve(s"relay-$form.json")))
      val machines = json.get("machines").elements().asScala.toSeq
      assertEquals(
        machines.map(_.get("name").asText()).toSet,
        Set("relayProduct", "relaySystem")
      )
      val actions = json.get("actions").elements().asScala.toSeq
      assertEquals(
        actions.map(_.get("id").asText()).toSet,
        Set(
          "fixture.features.relay.user.start",
          "fixture.features.relay.user.complete"
        )
      )
      assertEquals(
        actions
          .find(_.get("name").asText() == "start")
          .get
          .get("creates")
          .asText(),
        "task"
      )
      assertEquals(
        actions
          .find(_.get("name").asText() == "complete")
          .get
          .get("on")
          .asText(),
        "task"
      )
      assert(
        actions.forall(_.get("actor").asText() == "user")
      )
      assert(
        actions.forall(
          _.get("position").get("file").asText().endsWith("relay/Relay.scala")
        )
      )
      assert(
        machines.forall(_.get("entity").asText() == "task")
      )
      assertEquals(
        machines
          .find(
            _.get("name").asText() == "relaySystem"
          )
          .get
          .get("refines")
          .get("product")
          .asText(),
        "relayProduct"
      )

  private lazy val kindRefusals = Seq(
    (
      "missing-header",
      "nexus/Nexus.scala",
      None,
      "has no general feature file"
    ),
    (
      "wrong-header-package",
      "nexus/Nexus.scala",
      Some("package fixture.features.other\n"),
      "does not mirror"
    ),
    (
      "case-variant-header",
      "nexus/nexus.scala",
      Some("package fixture.features.nexus\n"),
      "one general feature file"
    ),
    (
      "general-vocabulary",
      "activity/Activity.scala",
      Some(
        "package fixture.features.activity\n\nenum Phase:\n  case idle\n"
      ),
      "Phase is level vocabulary declared in activity's root feature file"
    ),
    (
      "wrong-form-package",
      "nexus/standalone/Mismatch.scala",
      Some(
        "package fixture.features.nexus.workflow\nfinal case class Marker(value: Boolean)\n"
      ),
      "does not mirror"
    ),
    (
      "deeper-form",
      "nexus/workflow/system/extra/Extra.scala",
      Some(
        "package fixture.features.nexus.workflow.system.extra\n" +
          "import framework.dsl.*\n" +
          "object Extra extends Derived(fixture.features.nexus.workflow.system.NexusSystem)\n"
      ),
      "no level folder"
    ),
    (
      "foreign-product",
      "relay/workflow/system/System.scala",
      Some(
        Files
          .readString(
            testdata.resolve("layout/kinds/relay/workflow/system/System.scala")
          )
          .replace(
            "fixture.features.relay.shared.product",
            "fixture.features.nexus.workflow.product"
          )
          .replace("RelayProduct", "NexusProduct")
      ),
      "does not refine RelayProduct"
    ),
    (
      "foreign-single-level",
      "nexus/standalone/StandaloneNexus.scala",
      Some(
        Files
          .readString(
            testdata.resolve(
              "layout/kinds/nexus/standalone/StandaloneNexus.scala"
            )
          )
          // Strict equality: the refinement compares phases with the evidence Finite supplies.
          .replace(
            "import framework.dsl.*\n",
            "import framework.dsl.*\nimport framework.dsl.Finite.given\n" +
              "val _ = () => summon[CanEqual[Phase, Phase]]\n"
          )
          .replace(
            "  object effects:",
            "  object refinement extends Refinement(fixture.features.relay.shared.product.RelayProduct):\n" +
              "    def toProduct(s: State) = fixture.features.relay.shared.product.State(\n" +
              "      if s.phase == Phase.idle then fixture.features.relay.shared.product.Phase.idle\n" +
              "      else fixture.features.relay.shared.product.Phase.done)\n\n" +
              "  object effects:"
          )
      ),
      "outside its kind"
    )
  )
  for (name, file, contents, diagnostic) <- kindRefusals do
    test(s"kind classification refuses $name"):
      val variant = materializeTree("layout/kinds", s"kind-$name")
      val changed = variant.resolve(file)
      if name == "case-variant-header" then
        assume(
          !Files.exists(changed),
          "case-variant headers require a case-sensitive filesystem"
        )
      contents match
        case Some(text) =>
          Files.createDirectories(changed.getParent)
          Files.writeString(changed, text)
        case None => Files.delete(changed)
      val jar = packaged(s"kind-$name", variant)
      val out = scratch.resolve(s"kind-$name-out")
      val result = liftIr(out, s"$jar=$variant/,$modelJar=model/")
      assertNotEquals(result.exit, 0, name)
      assertEquals(listed(out), Nil)
      assert(
        refused(result).exists(message => message.contains(file) && message.contains(diagnostic)),
        s"$name did not report $diagnostic:\n${result.diagnostics}"
      )

  for (name, file, line, diagnostic) <- Seq(
      (
        "form-outside-kind",
        "standalone/Standalone.scala",
        3,
        "a form belongs inside a kind"
      ),
      (
        "machine-in-general-file",
        "nexus/Nexus.scala",
        5,
        "Misplaced is a machine object"
      ),
      (
        "second-general-file",
        "nexus/Extra.scala",
        1,
        "one general feature file"
      )
    )
  do
    test(s"R4 refuses $name at its source line"):
      val tree = materializeTree("layout/kinds", s"r4-$name")
      val specimen = testdata.resolve(s"layout/r4/$name/$file")
      val changed = tree.resolve(file)
      Files.createDirectories(changed.getParent)
      Files.copy(
        specimen,
        changed,
        java.nio.file.StandardCopyOption.REPLACE_EXISTING
      )
      val jar = packaged(s"r4-$name", tree)
      val out = scratch.resolve(s"r4-$name-out")
      val result = liftIr(out, s"$jar=$tree/,$modelJar=model/")
      assert(
        refused(result).exists(message =>
          message.startsWith(s"lift: $tree/$file:$line:") && message.contains(
            diagnostic
          )
        ),
        s"$name did not report $file:$line: $diagnostic:\n${result.diagnostics}"
      )
      assertNotEquals(result.exit, 0, name)
      assertEquals(listed(out), Nil)

  test("shared features refuse kind-form nesting"):
    val tree = materializeTree("layout/kinds", "shared-forms")
    val stream = Files.walk(tree.resolve("nexus"))
    try
      for file <- stream.iterator.asScala.toList if Files.isRegularFile(file) do
        Files.writeString(
          file,
          Files
            .readString(file)
            .replace("fixture.features.nexus", "fixture.foundations.nexus")
        )
    finally stream.close()
    val jar = packaged("shared-forms", tree)
    val out = scratch.resolve("shared-forms-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "nexus-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assert(
      refused(result).exists(message =>
        message.contains("nexus/workflow/") && message.contains(
          "no level folder"
        )
      ),
      result.diagnostics
    )

  test(
    "a form's package mirrors its original parent kind before rebasing"
  ):
    val tree = materializeTree("layout/kinds", "wrong-parent-kind")
    Files.move(
      tree.resolve("nexus/workflow"),
      Files.createDirectories(tree.resolve("other")).resolve("workflow")
    )
    val jar = packaged("wrong-parent-kind", tree)
    val out = scratch.resolve("wrong-parent-kind-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "nexus-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assert(
      refused(result).exists(message =>
        message.contains("other/workflow/") && message.contains(
          "does not mirror"
        )
      ),
      result.diagnostics
    )

  for variantName <- Seq("unrefined-sibling", "derived-sibling") do
    test(
      s"an inherited System requires its canonical primary with $variantName"
    ):
      val tree = materializeTree("layout/kinds", variantName)
      val system = tree.resolve("relay/workflow/system/System.scala")
      val text =
        if variantName == "unrefined-sibling" then
          Files
            .readString(system)
            .replaceAll("(?s)  object refinement .*?(?=  object effects:)", "")
            .replace(
              "import fixture.features.relay.shared.product.RelayProduct\n",
              ""
            )
        else
          "package fixture.features.relay.workflow\npackage system\nimport framework.dsl.*\n" +
            "import fixture.features.relay.shared.product.RelayProduct\nobject Copy extends Derived(RelayProduct.unmonitored)\n"
      Files.delete(system)
      Files.writeString(system.resolveSibling("Other.scala"), text)
      if variantName == "derived-sibling" then
        val feature = tree.resolve("relay/workflow/WorkflowRelay.scala")
        Files.writeString(
          feature,
          Files.readString(feature).replace("system.RelaySystem", "system.Copy")
        ): Unit
      val jar = packaged(variantName, tree)
      val out = scratch.resolve(s"$variantName-out")
      val result =
        liftIr(out, s"$jar=$tree/,$modelJar=model/", "relay-workflow")
      assertNotEquals(result.exit, 0)
      assertEquals(listed(out), Nil)
      assert(
        refused(result).exists(_.contains("system/System.scala is missing")),
        result.diagnostics
      )

  test(
    "kind cores and forms refuse unknown or deeper folders containing only types"
  ):
    val tree = materializeTree("layout/kinds", "type-only-folders")
    val misplaced = Seq(
      "relay/extra/Extra.scala" -> "fixture.features.relay.extra",
      "relay/shared/product/extra/Extra.scala" -> "fixture.features.relay.shared.product.extra",
      "relay/workflow/system/extra/Extra.scala" -> "fixture.features.relay.workflow.system.extra"
    )
    for (file, pkg) <- misplaced do
      val path = tree.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(
        path,
        s"package $pkg\nfinal case class Extra(value: Boolean)\n"
      )
    val jar = packaged("type-only-folders", tree)
    val out = scratch.resolve("type-only-folders-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "relay-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    for (file, _) <- misplaced do
      assert(
        refused(result).exists(message =>
          message.contains(file) && message.contains("no level folder")
        ),
        result.diagnostics
      )

  // R20 (a): the folders of a feature with two levels, a missing root feature file or level file,
  // a feature of one level with a subfolder, and a package that does not mirror its folder
  // (layoutRefusals/a).
  test(
    "the structure lint refuses a feature's levels out of their folders"
  ):
    val jar = packaged("layoutRefusals-a", materializeTree("layoutRefusals/a"))
    val out = scratch.resolve("layoutRefusals-a-out")
    val at = stored("layoutRefusals/a")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val two =
      "KettleSystem refines KettleProduct, so kettle has two levels, each in its folder: " +
        "the Product in product/Product.scala, the System in system/System.scala, beside the root " +
        "feature file named after the feature's folder, which holds shared types and the signature " +
        "(model/framework/export/testdata/layout/lamp is the template)"
    val urn = two.replace(
      "KettleSystem refines KettleProduct, so kettle",
      "UrnSystem refines UrnProduct, so urn"
    )
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}kettle/Kettle.scala:26: Stray is a machine object in kettle's root folder, " +
          s"${at}kettle/, whose feature file holds shared types and the signature " +
          "alone: a feature with two levels declares its machines in product/ and system/",
        s"lift: ${at}kettle/system/Heater.scala:12: $two; ${at}kettle/system/System.scala is " +
          "missing",
        s"lift: ${at}kettle/system/element/Element.scala:8: Element is a machine object in " +
          s"${at}kettle/system/element/, which is no level folder of kettle: a feature with two " +
          "levels keeps its Models in product/ and system/, one file per subject beside the " +
          "level's own file, with no folder below them",
        s"lift: ${at}tap/fittings/Washer.scala:5: ${at}tap/fittings/Washer.scala declares package " +
          s"fixture.features.tap, which its folder, ${at}tap/fittings/, does not mirror: a " +
          "feature's subpackages are its folders, named alike, since the structure lint reads a " +
          "source's package and the order lint its path",
        s"lift: ${at}tap/product/Product.scala:6: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no product/ " +
          "folder",
        s"lift: ${at}tap/valve/Valve.scala:8: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no valve/ folder",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/ has no root feature file",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/product/Product.scala is missing"
      )
    )

  // R20 (b): the level files name the feature's Product and System machines and own their
  // Phase, State and Fact vocabulary rather than leaving prefixed level types in the root.
  test(
    "the structure lint refuses invalid level names and root-owned level vocabulary"
  ):
    val jar = packaged("layoutRefusals-b", materializeTree("layoutRefusals/b"))
    val out = scratch.resolve("layoutRefusals-b-out")
    val at = stored("layoutRefusals/b")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}boiler/product/Product.scala:6: BoilerContract is the Product machine in " +
          "boiler's product/Product.scala: name it BoilerProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}boiler/system/System.scala:7: BoilerImplementation is the System machine in " +
          "boiler's system/System.scala: name it BoilerSystem, after the feature and its System " +
          "level",
        s"lift: ${at}kettle/product/Product.scala:6: KettleContract is the Product machine in " +
          "kettle's product/Product.scala: name it KettleProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}pump/system/System.scala:7: PumpProtocol is the System machine in pump's " +
          "system/System.scala: name it PumpSystem, with the same prefix as PumpProduct",
        s"lift: ${at}reservoir/Reservoir.scala:6: ProductPhase is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as Phase in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:9: ProductState is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as State in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:11: ProductFact is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as Fact in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:14: Phase is level vocabulary declared in " +
          "reservoir's root feature file: declare it in product/Product.scala or " +
          "system/System.scala",
        s"lift: ${at}reservoir/Reservoir.scala:17: SystemState is System level vocabulary " +
          "declared in reservoir's root feature file: declare it as State in " +
          "system/System.scala",
        s"lift: ${at}reservoir/Reservoir.scala:19: SystemFact is System level vocabulary " +
          "declared in reservoir's root feature file: declare it as Fact in " +
          "system/System.scala",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no Fact: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no Phase: each level owns Phase, State " +
          "and Fact in its level file",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no State: each level owns Phase, State " +
          "and Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no Fact: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no Phase: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no State: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}urn/system/System.scala:6: UrnSystem is the System machine in urn's " +
          "system/System.scala but does not refine UrnProduct from product/Product.scala"
      )
    )

  test(
    "the structure lint requires independent unambiguous primary level machines"
  ):
    val jar = packaged("layoutRefusals-d", materializeTree("layoutRefusals/d"))
    val out = scratch.resolve("layoutRefusals-d-out")
    val at = stored("layoutRefusals/d")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}brazier/product/Product.scala:6: brazier's product/Product.scala declares " +
          "multiple primary Product machines: BrazierProduct, SpareProduct",
        s"lift: ${at}cistern/product/Product.scala:6: CisternContract is the Product machine in " +
          "cistern's product/Product.scala: name it CisternProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}cistern/system/System.scala:6: cistern's system/System.scala declares no " +
          "primary System machine: declare one <Prefix>System",
        s"lift: ${at}furnace/system/System.scala:6: furnace's system/System.scala declares " +
          "multiple primary System machines: FurnaceSystem, SpareSystem",
        s"lift: ${at}pan/product/Product.scala:6: pan's product/Product.scala declares no " +
          "primary Product machine: declare one <Prefix>Product",
        s"lift: ${at}pan/system/System.scala:6: PanImplementation is the System machine in pan's " +
          "system/System.scala: name it PanSystem, after the feature and its System level",
        s"lift: ${at}stove/system/System.scala:6: StoveSystem is the System machine in stove's " +
          "system/System.scala but does not refine StoveProduct from product/Product.scala"
      )
    )

  // R20 (c): a machine's sections by their closed names, which say what each holds, and
  // one object exports per feature, in its root feature file (layoutRefusals/c); a level folder's
  // file is held to a feature file's order, which has no exports there.
  test("the structure lint refuses an unnamed section"):
    val jar = packaged("layoutRefusals-c", materializeTree("layoutRefusals/c"))
    val out = scratch.resolve("layoutRefusals-c-out")
    val at = stored("layoutRefusals/c")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val product = s"${at}kiln/product/Product.scala"
    val none =
      "and none of its sections, states, refinement, effects, monitors, rules, syncs, " +
        "properties, capabilities, queries: a section of another name sits at the top " +
        "level of the " +
        "file, in the signature, and anything else in one of these"
    assertEquals(
      refused(result),
      Seq(
        s"lift: $product:17: timers is an object in KilnProduct $none",
        s"lift: $product:21: helpers is an object in KilnProduct $none",
        s"lift: ${at}pump/system/System.scala:21: Gauge belongs before PumpSystem at " +
          s"${at}pump/system/System.scala:7: a level folder's file reads its header, then its " +
          "types, then its signature, then its machine and composition objects"
      )
    )
