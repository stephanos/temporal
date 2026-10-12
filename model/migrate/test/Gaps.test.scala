package umpire.migrate

import java.nio.file.{Files, Path}
import Fn167Task4Gaps.*

class GapsSuite extends munit.FunSuite:
  private val machine =
    """package fixture
      |package gaps
      |
      |import framework.*
      |
      |// The machine.
      |@unused
      |object Machine extends Base:
      |  val init = 0
      |  object properties:
      |    val safe = property.holds(_ => true)
      |
      |object Other extends Base:
      |  val init = 1
      |""".stripMargin
  private val sources = Map("Machine.scala" -> machine)
  private val tricky = "a \"quoted\" reason with a \\ backslash, é and a\ttab"
  private val acceptances = Seq(
    Acceptance("never-enabled", "machine", Seq("move-fast", "stop"), "first reason"),
    Acceptance("unasked-property", "machine", Seq("safe"), tricky),
    Acceptance("stuck-state", "machine", Seq("busy"), "first reason"),
    Acceptance("never-enabled", "other", Seq("stop"), "first reason")
  )
  private val placements = Map(
    ("never-enabled", "machine", "move-fast") -> Placement(
      "fixture.gaps.Machine",
      "neverEnabled(caller.move(Mode.fast))"
    ),
    ("never-enabled", "machine", "stop") -> Placement(
      "fixture.gaps.Machine",
      "neverEnabled(caller.stop)"
    ),
    ("unasked-property", "machine", "safe") -> Placement(
      "fixture.gaps.Machine.properties.safe",
      "unasked"
    ),
    ("stuck-state", "machine", "busy") -> Placement("fixture.gaps.Machine", "stuck(State.busy)"),
    ("never-enabled", "other", "stop") -> Placement(
      "fixture.gaps.Other",
      "neverEnabled(caller.stop)"
    )
  )
  private def inputs(
      acceptances: Seq[Acceptance] = acceptances,
      placements: Map[Key, Placement] = placements,
      inherited: Map[Key, Key] = Map.empty,
      waivers: Seq[Waiver] = Nil,
      sources: Map[String, String] = sources
  ) = Inputs(acceptances, placements, inherited, waivers, sources)

  private val expected =
    """package fixture
      |package gaps
      |
      |import framework.*
      |
      |// The machine.
      |@gap(neverEnabled(caller.move(Mode.fast)), neverEnabled(caller.stop), stuck(State.busy))(
      |  because = "first reason"
      |)
      |@unused
      |object Machine extends Base:
      |  val init = 0
      |  object properties:
      |    @gap(unasked)(because = "a \"quoted\" reason with a \\ backslash, é and a\ttab")
      |    val safe = property.holds(_ => true)
      |
      |@gap(neverEnabled(caller.stop))(because = "first reason")
      |object Other extends Base:
      |  val init = 1
      |""".stripMargin

  test("one gap per declaration and exact reason, grouped in acceptance order, reasons preserved"):
    val result = migrate(inputs()).toOption.get
    assertEquals(result.sources("Machine.scala"), expected)
    assertEquals(result.inserted.size, 3)
    assertEquals(result.dropped, Nil)
    // Every byte outside the inserted lines is the original's, and the inputs are untouched.
    val kept =
      result.sources("Machine.scala").linesIterator.filterNot(_.trim.startsWith("@gap(")).toSeq
    val unwrapped = kept.filterNot(l => l.trim.startsWith("because =") || l.trim == ")")
    assertEquals(unwrapped.mkString("\n") + "\n", machine)
    assertEquals(sources("Machine.scala"), machine)

  test("a second application converges"):
    val once = migrate(inputs()).toOption.get
    val twice = migrate(inputs(sources = once.sources)).toOption.get
    assertEquals(twice.sources, once.sources)
    assertEquals(twice.inserted, Nil)

  test(
    "a finding its base's acceptance covers is dropped; one whose base is unaccepted is refused"
  ):
    val derived = acceptances :+ Acceptance("never-enabled", "derived", Seq("stop"), "templated")
    val inherited =
      Map(("never-enabled", "derived", "stop") -> ("never-enabled", "machine", "stop"))
    val result = migrate(inputs(acceptances = derived, inherited = inherited)).toOption.get
    assertEquals(result.sources("Machine.scala"), expected)
    assertEquals(
      result.dropped,
      Seq("never-enabled derived \"stop\": inherited from never-enabled machine \"stop\"")
    )
    val unsafe = Map(("never-enabled", "derived", "stop") -> ("never-enabled", "base", "stop"))
    val refused = migrate(inputs(acceptances = derived, inherited = unsafe))
    assert(
      refused.left.toOption.get.exists(_.contains("which no acceptance covers")),
      refused.toString
    )

  test("unplaceable owners and subjects refuse the whole migration without publishing a source"):
    val unknownOwner =
      acceptances :+ Acceptance("never-enabled", "byIDCompletion", Seq("stop"), "old owner")
    val missing = migrate(inputs(acceptances = unknownOwner))
    assertEquals(
      missing.left.toOption.get,
      Seq("never-enabled byIDCompletion \"stop\": no declaration is resolved for it")
    )
    val undeclared = placements.updated(
      ("stuck-state", "machine", "busy"),
      Placement("fixture.gaps.Gone", "stuck(State.busy)")
    )
    assertEquals(
      migrate(inputs(placements = undeclared)).left.toOption.get,
      Seq("fixture.gaps.Gone is declared by none of the sources")
    )
    val twice = sources + ("Copy.scala" -> machine)
    assert(migrate(inputs(sources = twice)).left.toOption.get.head.contains("is declared 2 times"))
    val duplicated = acceptances :+ Acceptance("stuck-state", "machine", Seq("busy"), "again")
    assert(
      migrate(inputs(acceptances = duplicated)).left.toOption.get.head.contains("accepted 2 times")
    )

  test("conflicting insertions refuse rather than stack a second reason"):
    val other = machine.replace(
      "object Other extends Base:",
      "@gap(neverEnabled(caller.stop))(because = \"another reason\")\nobject Other extends Base:"
    )
    val conflict = migrate(inputs(sources = Map("Machine.scala" -> other)))
    assertEquals(
      conflict.left.toOption.get,
      Seq(
        "fixture.gaps.Other already declares neverEnabled(caller.stop) a gap because \"another reason\""
      )
    )
    val partial = machine.replace(
      "object Other extends Base:",
      "@gap(stuck(State.idle))(because = \"first reason\")\nobject Other extends Base:"
    )
    assert(
      migrate(inputs(sources = Map("Machine.scala" -> partial))).left.toOption.get.head
        .contains("already has a gap because \"first reason\" naming other subjects")
    )

  test("capability waivers reconcile with their declarations and are never inserted"):
    val waived =
      acceptances :+ Acceptance("capability-waiver", "machine", Seq("machine.closed"), "because so")
    val result = migrate(
      inputs(acceptances = waived, waivers = Seq(Waiver("machine", "machine.closed", "because so")))
    ).toOption.get
    assertEquals(result.sources("Machine.scala"), expected)
    assertEquals(
      result.dropped,
      Seq(
        "capability-waiver machine \"machine.closed\": lowered from its except/overriding declaration"
      )
    )
    val mismatched = migrate(
      inputs(
        acceptances = waived,
        waivers = Seq(Waiver("machine", "machine.closed", "other words"))
      )
    )
    assert(
      mismatched.left.toOption.get.head.contains("reconcile them by hand"),
      mismatched.toString
    )
    assert(
      migrate(inputs(acceptances = waived)).left.toOption.get.head.contains("no except/overriding")
    )

  test("the existing acceptance and waiver files parse into the transformation's inputs"):
    val read = Fn167Task4GapInputs.acceptances(
      """{"accepted": [{"kind": "stuck-state", "owner": "machine", "subjects": ["busy", "a.b c"],
        |"because": "why \"so\""}]}""".stripMargin
    )
    assertEquals(
      read,
      Seq(Acceptance("stuck-state", "machine", Seq("busy", "a.b c"), "why \"so\""))
    )
    val mini = Files.readString(Path.of("model/migrate/proof/gaps-mini.lint.json"))
    assert(Fn167Task4GapInputs.acceptances(mini).exists(_.kind == "unproduced"))
    val waivers = Fn167Task4GapInputs.waivers(
      """{"machines": ["m"], "waivers": [{"machine": "m", "subject": "m.p", "because": "r"}]}"""
    )
    assertEquals(waivers, Seq(Waiver("m", "m.p", "r")))
