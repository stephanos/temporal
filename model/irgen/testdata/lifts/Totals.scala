// The total each Query asserts: written infix and dotted, before and after `.expect(...)`, and as the
// Int parameter of a shared def over a machine, supplied as a literal at each call. The lifter's tests
// lift `queries`, `lampTotals` and `plainLampTotals` and require each spelling's record to differ
// from its twin's in nothing but its name and position, and the shared def's instances in nothing
// but their machine and total. Two claims one shared def declares together, as a case-class bundle
// its Queries read by field, lift (`bundledLamp`, `bundledPlainLamp`) as the same claims declared
// directly (`directLamp`, `directPlainLamp`) do, but for the words `bundled` and `direct` in their names.
package fixture.totals

import umpire.*

given Family = Family("fixture.totals")

final case class Lamp(lit: Boolean) derives Finite

enum Dim derives Finite:
  case low, high

enum Outcome derives Finite:
  case accepted

given Accepted[Outcome] = Accepted(Outcome.accepted)

val flip = action(Party("user"))
val dim = action(Party("user")).input[Dim]("level")

def flipStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = accept(Lamp(!l.lit))
def dimStep(l: Lamp, level: Dim): List[Step[Lamp, Outcome, Nothing]] =
  if l.lit then accept(l) else disabled

// 2 states; its action classes are flip and dim's two levels, 3 in all.
val lamp = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> flipStep, dim ~> dimStep)
}

// 2 states; its one action class is flip.
val plainLamp = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> flipStep)
}

val two = Limits(steps = 2, actions = 2, search = 64)

val litOnce = lamp.property holds (after => after.state.lit)
val flipTwice = lamp.scenario.actions(flip, flip)

// Pinned: 2 states x min(2 steps, 2 scheduled flips) = 4.
val infixTotal = query find litOnce in flipTwice limits two total 4
val dottedTotal = (query find litOnce in flipTwice limits two).total(4)

val run = realize.RunExpectation(realize.Conformance.conformant, realize.Outcome.satisfied)
val totalThenExpect = (query find litOnce in flipTwice limits two total 4).expect(run)
val expectThenTotal = (query find litOnce in flipTwice limits two).expect(run).total(4)

/**
 * A Query declared once over a machine, whose total each call supplies. Free: 2 states x the
 * machine's action classes x 2 steps, so 12 on `lamp` (3 classes) and 4 on `plainLamp` (1 class).
 */
def lampQueries(m: Machine[Lamp, Outcome, Nothing], total: Int): Vector[Query] = Vector(
  query(s"${m.name}.anyLit") find (m.property(s"${m.name}.lit") holds (after =>
    after.state.lit
  )) in m.scenario("any").free limits two total total
)

val queries: Vector[Query] = Vector(infixTotal, dottedTotal, totalThenExpect, expectThenTotal)
val lampTotals: Vector[Query] = lampQueries(lamp, 12)
val plainLampTotals: Vector[Query] = lampQueries(plainLamp, 4)

// ### Two claims a shared def declares together, as a bundle read back by field

/** The claims every lamp is held to, declared together by `lampLaws`. */
final case class LampLaws(lit: Property[Lamp], unlit: Property[Lamp])

def lampLaws(m: Machine[Lamp, Outcome, Nothing]): LampLaws = LampLaws(
  m.property("bundledLit") holds (after => after.state.lit),
  m.property("bundledUnlit") holds (after => !after.state.lit)
)

/** The laws of `m`, each read from the bundle by field. Free, so 12 on `lamp` and 4 on `plainLamp`. */
def bundledLawQueries(m: Machine[Lamp, Outcome, Nothing], total: Int): Vector[Query] =
  val laws = lampLaws(m)
  val any = m.scenario("bundledAny").free
  Vector(
    query(s"${m.name}.bundledLit") find laws.lit in any limits two total total,
    query(s"${m.name}.bundledUnlit") find laws.unlit in any limits two total total
  )

/** The same claims declared directly. */
def directLawQueries(m: Machine[Lamp, Outcome, Nothing], total: Int): Vector[Query] =
  val any = m.scenario("directAny").free
  Vector(
    query(s"${m.name}.directLit") find (m.property("directLit") holds (after =>
      after.state.lit
    )) in any limits two total total,
    query(s"${m.name}.directUnlit") find (m.property("directUnlit") holds (after =>
      !after.state.lit
    )) in any limits two total total
  )

val bundledLamp: Vector[Query] = bundledLawQueries(lamp, 12)
val bundledPlainLamp: Vector[Query] = bundledLawQueries(plainLamp, 4)
val directLamp: Vector[Query] = directLawQueries(lamp, 12)
val directPlainLamp: Vector[Query] = directLawQueries(plainLamp, 4)
