package temporal
package features.activity
package mini
package product

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

enum Phase(val status: Fact) extends Recorded[Fact]:
  case scheduled extends Phase(Fact.statusScheduled), Waiting
  case started extends Phase(Fact.statusStarted), Held
  case cancelRequested extends Phase(Fact.statusCancelRequested), Held
  case completed extends Phase(Fact.statusCompleted), Succeeded
  case failed extends Phase(Fact.statusFailed), Failed
  case canceled extends Phase(Fact.statusCanceled), Canceled
  case timedOut extends Phase(Fact.statusTimedOut), TimedOut

final case class State(phase: Phase)

enum Fact:
  case statusScheduled, statusStarted, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTimedOut

object ActivityProduct
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Dispatching,
      Completion:
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]
  val entity = activity

  object rules extends Facets(dispatching.rules, completion.rules)

  object properties:
    val terminalRemains = property.once(_.phase.in[Closed]).keeps(_.phase)

  object capabilities extends FacetCapabilities(dispatching.capabilities, completion.capabilities)

  object queries:
    capabilities.bound(three)
    val completed = witness(client.start(), activityWorker.poll, activityWorker.respondCompleted)
      .records(Fact.statusCompleted).ends(_.phase == Phase.completed).limits(three)
    val terminalIsFinal = query verify properties.terminalRemains in scenario("any").free limits three
