package framework.residualhelpers

import framework.dsl.*

object Helpers:
  def localEntity(): Entity =
    val temporary = Entity(key = "localId")
    temporary
  def localObservation(on: Entity): Observation =
    val temporary = Observation(on = on)
    temporary
  def localBounds(): Limits =
    val temporary = Limits(steps = 1, actions = 1, search = 1)
    temporary
  def entity(key: String): Entity = Entity(key = key)
  def emptyEntity(key: String): Entity = new Entity("", key)
  def observation(on: Entity): Observation = Observation(on = on)
  def emptyObservation(on: Entity): Observation = new Observation("", on)
  def bounds(n: Int): Limits = Limits(steps = n, actions = n, search = n)
  def emptyBounds(n: Int): Limits = new Limits("", n, n, n)
  def inferredAction(actor: Actor): Action[EmptyTuple] = action(actor)
  def emptyAction(actor: Actor): Action[EmptyTuple] = action("", actor)
  def inferredAssumption(a: Action[?]): Assumption = assume.fair(a)
  def emptyAssumption(a: Action[?]): Assumption = assume("").fair(a)
  def copyEntity(value: Entity): Entity = value.copy(key = "copy")
  def emptyEntityCopy(value: Entity): Entity = value.copy(name = "")
  def copyObservation(value: Observation): Observation = value.copy()
  def emptyObservationCopy(value: Observation): Observation = value.copy(name = "")
  def copyBounds(value: Limits): Limits = value.copy(steps = 2)
  def emptyBoundsCopy(value: Limits): Limits = value.copy(name = "")
