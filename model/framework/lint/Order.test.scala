package framework.lint

class OrderFixtures extends munit.FunSuite with LintFixtures:
  // fn-126 R4: the declaration-order lint, before anything is lifted. Each kind at its line, and
  // none of the reads beside them that initialize nothing yet (initOrder/Forward.scala). Every other
  // lift of these tests, and the gate's of the Models, runs it too and is refused nothing.
  test(
    "the declaration-order lint refuses a read before its declaration, a cycle, an order and a place"
  ):
    val jar = packaged("initOrder", materialize("initOrder"))
    val out = scratch.resolve("initOrder-out")
    val at = stored("initOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val forward = (line: Int) =>
      s"lift: ${at}Forward.scala:$line: late is read while Forward initializes, before it is " +
        s"declared at ${at}Forward.scala:18, so it is still null here: declare it before the " +
        "declaration that reads it"
    val feature = s"${at}InitOrder.scala"
    assertEquals(
      refused(result),
      Seq(
        // (a)
        forward(7),
        forward(17),
        // (d): a Model declaration beside the feature file
        s"lift: ${at}Forward.scala:23: stray is a Scenario, which a feature declares in its " +
          s"feature file, $feature: in the `queries` object of its machine's object there",
        s"lift: ${at}Forward.scala:29: strayInCompanion is a Scenario, which a feature declares " +
          s"in its feature file, $feature: in the `queries` object of its machine's object there",
        // (d): a machine in a type's companion, a Property at the top level, a machine object in
        // an object of the signature
        s"lift: $feature:22: BulbLamp holds a Model declaration inside Bulb, the companion of a " +
          "type: a machine object sits at the top level of a feature file, and its sections " +
          "directly in it",
        s"lift: $feature:28: loose is a Property, declared at the top level of a feature file: " +
          "it belongs in the `properties` object of its machine's object",
        s"lift: $feature:32: Inner holds a Model declaration inside Holder, an object of the " +
          "signature: a machine object sits at the top level of a feature file, and its " +
          "sections directly in it",
        // (c)
        s"lift: $feature:68: properties belongs before queries at $feature:65: object Backwards " +
          "reads its header, then states, then refinement, then effects, then monitors, then " +
          "rules, then properties, then capabilities, then queries",
        s"lift: $feature:71: Late belongs before Backwards at $feature:59: a feature file reads " +
          "its header, then its types, then its signature, then its machine and composition " +
          "objects",
        // (d)
        s"lift: $feature:77: extras holds a Model declaration in Misplaced, and is none of its " +
          "sections, states, refinement, effects, monitors, rules, properties, capabilities, " +
          "queries: its declarations belong in them",
        s"lift: $feature:82: lit is a Property, and belongs in the `properties` object of its " +
          "machine's object, not in Misplaced",
        s"lift: $feature:85: switchLit is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        // (c): a Scenario after a Query; (d): a Query over another object's Scenario
        s"lift: $feature:99: late belongs before first at $feature:98: Asked.queries reads " +
          "its Scenarios, then its Queries",
        s"lift: $feature:100: borrowed is declared over flipped, which Switch.queries declares: " +
          "it belongs in Switch.queries",
        // (b) through a base class: the body of the class a `capabilities` section extends
        s"lift: ${at}Realization.scala:16: an initialization cycle: Dimmer.capabilities -> " +
          "DimmerRealization -> Dimmer.capabilities, each read while the one before it " +
          "initializes, so one of them is read half made: read it in a def, a lambda or a lazy " +
          "val, or move what is read into an object of its own"
      )
    )

  // fn-126 R4 (e) and R17: each kind at its line, and none of the correct object forms beside them,
  // nor two objects' timers of one name, which their objects name apart (sectionOrder/SectionOrder.scala).
  test(
    "the declaration-order lint refuses sections out of order or out of place"
  ):
    val jar = packaged("sectionOrder", materialize("sectionOrder"))
    val out = scratch.resolve("sectionOrder-out")
    val at = stored("sectionOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val f = s"${at}SectionOrder.scala"
    val order =
      "object Backwards reads its header, then states, then refinement, then effects, " +
        "then monitors, then rules, then properties, then capabilities, then queries"
    val handBound =
      "a machine object says when each action fires in its `rules`, " +
        "`on(action) { when(p) ~> effects.x }`, and a derivation binds one in `rebind`"
    assertEquals(
      refused(result),
      Seq(
        s"lift: $f:39: the section capabilities sits at the top level of a feature file: a " +
          "machine's sections sit directly in its machine or composition object",
        s"lift: $f:101: states belongs before effects at $f:98: $order",
        s"lift: $f:107: rules belongs before properties at $f:104: $order",
        s"lift: $f:114: stray is a step function, and belongs in the `effects` object of its " +
          "machine's object, not in Misfiled",
        s"lift: $f:115: watched is a monitor, assumption, hole or channel, and belongs in the " +
          "`monitors` object of its machine's object, not in Misfiled",
        s"lift: $f:116: lit is vocabulary of Misfiled, declared outside its sections: it belongs " +
          "in the `states` object of its machine's object",
        s"lift: $f:117: toProduct is a member of Misfiled's refinement: declare it in " +
          "`object refinement extends Refinement(product)`, which holds the machine's refinement",
        s"lift: $f:118: strayBlock is a step function, and belongs in the `effects` object of its " +
          "machine's object, not in Misfiled",
        s"lift: $f:127: dim is a monitor, assumption, hole or channel, declared directly in " +
          "Misfiled.properties: it belongs in the `monitors` object of its machine's object",
        s"lift: $f:129: borrowed is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        s"lift: $f:135: a step function is bound by hand, `action ~> step`, in HandBound: $handBound",
        s"lift: $f:139: kept is a step function, declared inside HandBound.effects.more: it " +
          "belongs in the `effects` object of its machine's object",
        s"lift: $f:153: armed is read while Guarded.rules initializes, before it is declared at " +
          s"$f:154, so it is still null here: declare it before the declaration that reads it",
        s"lift: $f:159: a step function is bound by hand, `action ~> step`, in Escaped: $handBound",
        s"lift: $f:166: a step function is bound by hand, `action ~> step`, in Cored: $handBound",
        s"lift: $f:174: syncs belongs before properties at $f:171: object BackwardsComposition " +
          "reads its header, then states, then refinement, then effects, then monitors, then " +
          "syncs, then properties, then capabilities, then queries"
      )
    )

  // A Model folder must name its feature file after itself (misnamed/Lamp.scala): a file of another
  // name in a folder with no feature file would otherwise be held to no reading order.
  test(
    "the declaration-order lint refuses a Model in a folder with no feature file"
  ):
    val jar = packaged("misnamed", materialize("misnamed"))
    val out = scratch.resolve("misnamed-out")
    val at = stored("misnamed")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val home =
      "a Model folder declares its Models in its feature file, the file named after " +
        s"the folder, in $at"
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}Lamp.scala:16: Switch holds a Model declaration in a file not named after " +
          s"its folder: $home",
        s"lift: ${at}Lamp.scala:27: loose is a Property, declared in a file not named after its " +
          s"folder: $home",
        s"lift: ${at}Lamp.scala:32: Spare holds a Model declaration in a file not named after " +
          s"its folder: $home"
      )
    )
