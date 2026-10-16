package fixture.derivationfault

import framework.dsl.*

object fault extends Actor:
  val stop = action(this).responseLoss // a native fault declares one kind (fn-123)
  val restart = action(this).responseLoss

object Source extends Machine[Boolean, Boolean, Nothing], FailureModel:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def first(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
    def replacement(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(false, s))
  object rules extends Rules:
    on(fault.stop)(always ~> effects.first)

object Rebound
    extends Derived(Source.rebind(fault.stop ~> Source.effects.replacement)),
      FailureModel
object Extended
    extends Derived(Source.extend(on(fault.restart)(always ~> Source.effects.first))),
      FailureModel
