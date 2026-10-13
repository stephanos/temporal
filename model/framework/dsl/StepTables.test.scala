package framework.dsl

// The step tables (StepTables.scala): every pair of a state and a class is a row, disabled ones
// included, so a changed fact or rule result changes the table; a composition's table holds its
// members and syncs; values are spelled as the table pins spell them.
class StepTablesTest extends munit.FunSuite:
  import RulesFixture.*
  import StepTablesFixture.*

  private def table(model: Model) = StepTables.lines(model).toVector

  test("a machine's table has a row for every state and class, disabled pairs included"):
    val rows = table(Switch).filter(_.startsWith("step\t"))
    val classes = Switch.bindings.map(b => classesOf(b.decl).size).sum
    assertEquals(rows.size, classes * Switch.fs.values.size)
    assert(rows.exists(_.endsWith("\t{  }")), "no disabled pair is written")
    assert(
      table(Switch).contains("bindings\thand.press hand.turn system.tick"),
      table(Switch).take(2)
    )
    assert(rows.exists(_.startsWith("step\thand.turn[down]\ton-0\t{ ok | off-0 | [] |  }")), rows)

  test("a one-fact mutation and a changed rule result change the table"):
    val renamed = table(Switch)
    val worn = table(Derived1)
    assertNotEquals(worn, renamed)
    val changedRows = renamed.zip(worn).filterNot(_ == _)
    assert(changedRows.forall(_._1.startsWith("step\thand.press")), changedRows)
    assertNotEquals(table(Faulted), table(Facts))
    assertEquals(
      table(Faulted).zip(table(Facts)).count(_ != _),
      1
    )

  test("a composition's table holds its members, its syncs and its ends"):
    val lines = table(Twins)
    assertEquals(lines.head, "members\tleft=switch right=unfelt")
    assert(lines.contains("sync\thand.press hand.press"), lines)
    assert(lines.exists(_.startsWith("ends\t")), lines)
    assert(table(Odd).contains("derives\ttwins with stuck"), table(Odd))
    assert(table(Odd).exists(_.startsWith("ends\t")), table(Odd))

  test("values are spelled as the table pins spell them"):
    assertEquals(StepTables.spell(Lamp(Lit.on, UpTo(2))), "on-2")
    assertEquals(StepTables.spell(Some(Lit.off)), "Some-off")
    assertEquals(StepTables.spell(None), "None")
    assertEquals(StepTables.spell(List(Lit.on, Lit.off)), "[on,off]")

object StepTablesFixture:
  import RulesFixture.*
  // A lamp whose press records a fact, and one that records another in one state alone: one
  // changed fact.
  enum Noted:
    case lit, dark

  object Facts extends Machine[Lamp, Said, Noted]:
    val init = Lamp(Lit.off, UpTo(0))
    def end(s: Lamp) = true

    object rules
        extends Bindings(
          hand.press ~> ((s: Lamp) => List(Step(Said.ok, s.copy(light = Lit.on), List(Noted.lit))))
        )

  object Faulted
      extends Derived(
        Facts.rebind(
          hand.press ~> ((s: Lamp) =>
            val fact = if s.equals(Lamp(Lit.broken, UpTo(2))) then Noted.dark else Noted.lit
            List(Step(Said.ok, s.copy(light = Lit.on), List(fact)))
          )
        )
      )

  // The switch whose press wears it out: a changed rule result.
  object Derived1 extends Derived(Switch.rebind(hand.press ~> Switch.effects.wear))
