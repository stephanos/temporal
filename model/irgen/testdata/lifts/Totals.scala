// The total each Query asserts: written infix and dotted, before and after `.expect(...)`, and as the
// Int parameter of a shared def over a machine, supplied as a literal at each call; and none, which
// the lifter counts (fn-126 decision 28). The lifter's tests lift `queries`, `lampTotals`,
// `plainLampTotals`, `lampCounted` and `plainLampCounted` and require each spelling's record to
// differ from its twin's in nothing but its name and position, and the shared def's instances in
// nothing but their machine and total. Two claims one shared def declares together, as a case-class bundle
// its Queries read by field, lift (`bundledLamp`, `bundledPlainLamp`) as the same claims declared
// directly (`directLamp`, `directPlainLamp`) do, but for the words `bundled` and `direct` in their names.
package fixture.totals

import umpire.*

// A lamp's state, named apart from the machine object `Lamp`.
final case class LampState(lit: Boolean) derives Finite

enum Dim derives Finite:
  case low, high

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

val flip = action(Actor("user"))
val dim = action(Actor("user")).input[Dim]("level")

def flipStep(l: LampState): List[Step[LampState, Outcome, Nothing]] = enter(LampState(!l.lit))
def dimStep(l: LampState, level: Dim): List[Step[LampState, Outcome, Nothing]] =
  if l.lit then enter(l) else disabled

// 2 states; its action classes are flip and dim's two levels, 3 in all.
object Lamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(false)
  def end(lamp: State) = true

  object rules extends Bindings(flip ~> flipStep, dim ~> dimStep)

// 2 states; its one action class is flip.
object PlainLamp extends Machine[LampState, Outcome, Nothing]:
  val init = LampState(false)
  def end(lamp: State) = true

  object rules extends Bindings(flip ~> flipStep)

val two = Limits(steps = 2, actions = 2, search = 64)

val litOnce = Lamp.property holds (after => after.state.lit)
val flipTwice = Lamp.scenario.actions(flip, flip)

// Pinned: 2 states x min(2 steps, 2 scheduled flips) = 4.
val infixTotal = query find litOnce in flipTwice limits two total 4
val dottedTotal = (query find litOnce in flipTwice limits two).total(4)

val run = realize.RunExpectation(
  realize.Conformance.conformant,
  realize.PropertyOutcome.satisfied,
  realize.PropertyOutcome.satisfied,
  realize.Disposition.completed,
  realize.Cleanup.succeeded
)
val totalThenExpect = (query find litOnce in flipTwice limits two total 4).expect(run)
val expectThenTotal = (query find litOnce in flipTwice limits two).expect(run).total(4)

// A Query declared once over a machine, whose total each call supplies. Free: 2 states x the
// machine's action classes x 2 steps, so 12 on `lamp` (3 classes) and 4 on `plainLamp` (1 class).
def lampQueries(m: Machine[LampState, Outcome, Nothing], total: Int): Vector[Query] = Vector(
  query(s"${m.name}.anyLit") find (m.property(s"${m.name}.lit") holds (after =>
    after.state.lit
  )) in m.scenario("any").free limits two total total
)

// No total: the lifter counts the 4 the author writes for `infixTotal`.
val countedTotal = query find litOnce in flipTwice limits two

// The shared def's Query with no total: the lifter counts 12 on `lamp` and 4 on `plainLamp`.
def countedQueries(m: Machine[LampState, Outcome, Nothing]): Vector[Query] = Vector(
  query(s"${m.name}.anyLit") find (m.property(s"${m.name}.lit") holds (after =>
    after.state.lit
  )) in m.scenario("any").free limits two
)

val queries: Vector[Query] =
  Vector(infixTotal, dottedTotal, totalThenExpect, expectThenTotal, countedTotal)
val lampTotals: Vector[Query] = lampQueries(Lamp, 12)
val plainLampTotals: Vector[Query] = lampQueries(PlainLamp, 4)
val lampCounted: Vector[Query] = countedQueries(Lamp)
val plainLampCounted: Vector[Query] = countedQueries(PlainLamp)

// ### Two claims a shared def declares together, as a bundle read back by field

// The claims every lamp is held to, declared together by `lampClaims`.
final case class LampClaims(lit: Property[LampState], unlit: Property[LampState])

def lampClaims(m: Machine[LampState, Outcome, Nothing]): LampClaims = LampClaims(
  m.property("bundledLit") holds (after => after.state.lit),
  m.property("bundledUnlit") holds (after => !after.state.lit)
)

// The claims of `m`, each read from the bundle by field. Free, so 12 on `lamp` and 4 on `plainLamp`.
def bundledClaimQueries(m: Machine[LampState, Outcome, Nothing], total: Int): Vector[Query] =
  val claims = lampClaims(m)
  val any = m.scenario("bundledAny").free
  Vector(
    query(s"${m.name}.bundledLit") find claims.lit in any limits two total total,
    query(s"${m.name}.bundledUnlit") find claims.unlit in any limits two total total
  )

// The same claims declared directly.
def directClaimQueries(m: Machine[LampState, Outcome, Nothing], total: Int): Vector[Query] =
  val any = m.scenario("directAny").free
  Vector(
    query(s"${m.name}.directLit") find (m.property("directLit") holds (after =>
      after.state.lit
    )) in any limits two total total,
    query(s"${m.name}.directUnlit") find (m.property("directUnlit") holds (after =>
      !after.state.lit
    )) in any limits two total total
  )

val bundledLamp: Vector[Query] = bundledClaimQueries(Lamp, 12)
val bundledPlainLamp: Vector[Query] = bundledClaimQueries(PlainLamp, 4)
val directLamp: Vector[Query] = directClaimQueries(Lamp, 12)
val directPlainLamp: Vector[Query] = directClaimQueries(PlainLamp, 4)
