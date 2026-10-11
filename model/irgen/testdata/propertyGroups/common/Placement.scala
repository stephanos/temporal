package propertygroups

import framework.*

final case class Mixed(safety: Property[State], progress: Progress[State])
final case class Reversed(progress: Progress[State], safety: Property[State])
final case class Subtypes(
    never: Never[State, Outcome, Nothing],
    stays: Stays[State, Outcome, Nothing]
)
final case class ScenarioBundle(property: Property[State], scenario: Scenario[State])
final case class QueryBundle(progress: Progress[State], query: Query)
final case class ProgressPair(first: Progress[State], last: Progress[State])

def mixed(m: Machine[State, Outcome, Nothing]): Mixed = Mixed(
  m.property("mixedSafety").holds(after => after.state == State.closed),
  m.leadsTo("mixedProgress")(_ == State.open, _ == State.closed, 2, fair)
)
def reversed(m: Machine[State, Outcome, Nothing]): Reversed =
  val claims = mixed(m)
  Reversed(claims.progress, claims.safety)
def subtypes(m: Machine[State, Outcome, Nothing]): Subtypes = Subtypes(
  m.property("bundleNever").never(after => after.state == State.open),
  m.property("bundleStays").stays(_ == State.closed)
)
def attached: Monitor[State, Outcome, Nothing, Boolean] = seal
def scenarioBundle(m: Machine[State, Outcome, Nothing]): ScenarioBundle =
  ScenarioBundle(shared(m), m.scenario("bundledScenario").free)
def queryBundle(m: Machine[State, Outcome, Nothing]): QueryBundle =
  QueryBundle(
    eventual(m),
    query("bundledQuery").verify(shared(m)).in(m.scenario("any").free).limits(limits)
  )
