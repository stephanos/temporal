package propertygroups.foundations.crossed

import framework.dsl.*
import propertygroups.*

object Crossed extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object monitors:
    val watched = seal
  object rules extends Rules
  object properties:
    object safety:
      val directProgress =
        Crossed.leadsTo("directProgress")(_ == State.open, _ == State.closed, 1, fair)
      val aliasProgress = directProgress
      val factoryProgress = eventual(Crossed)
      val mixedBundle = mixed(Crossed)
      val reversedBundle = reversed(Crossed)
      val bundleAlias = mixedBundle
      val selectedProgress = mixedBundle.progress
      val scenarioBundle = propertygroups.scenarioBundle(Crossed)
    object liveness:
      val directProperty = property("directProperty").expect(check(after.state == State.closed))
      val aliasProperty = directProperty
      val factoryProperty = shared(Crossed)
      val neverProperty = property("neverProperty").never(after => after.state == State.open)
      val neverAlias = neverProperty
      val staysProperty = property("staysProperty").stays(_ == State.closed)
      val staysAlias = staysProperty
      val unlessProperty = property("unlessProperty")
        .stays(_ == State.closed)
        .unless(after => after.state == State.open)
      val subtypeBundle = subtypes(Crossed)
      val subtypeAlias = subtypeBundle
      val selectedNever = subtypeBundle.never
      val mixedBundle = mixed(Crossed)
      val reversedBundle = reversed(Crossed)
      val watched = seal
      val monitorAlias = watched
      val monitorFactory = attached
      val queryBundle = propertygroups.queryBundle(Crossed)
      val misplacedEffect = effect[State, Outcome, Nothing](using summon, Ok(Outcome.done)) {}
      def misplacedStep(s: State): List[Step[State, Outcome, Nothing]] = step(s)
