package framework.residualclient

import framework.dsl.*
import framework.residualhelpers.Helpers.*

def unchanged(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Bindings(Client.send ~> unchanged)

object Client:
  val fromLocalEntity = localEntity()
  val fromLocalObservation = localObservation(fromLocalEntity)
  val fromLocalBounds = localBounds()
  val inferred = entity("id")
  val empty = emptyEntity("emptyId")
  val read = observation(inferred)
  val emptyRead = emptyObservation(empty)
  val one = bounds(1)
  val emptyOne = emptyBounds(1)
  val send = inferredAction(Actor("caller"))
  val emptySend = emptyAction(Actor("caller"))
  // Qualified: framework.dsl's binary-public inferredAssumption is visible in framework.*.
  val fair = framework.residualhelpers.Helpers.inferredAssumption(send)
  val emptyFair = emptyAssumption(emptySend)
  val alias = fair
  val entityAlias = inferred
  val observationAlias = read
  val limitsAlias = one
  val actionAlias = send
  val entityCopy = copyEntity(inferred)
  val explicitEmptyEntity = emptyEntityCopy(inferred)
  val observationCopy = copyObservation(read)
  val explicitEmptyObservation = emptyObservationCopy(read)
  val boundsCopy = copyBounds(one)
  val explicitEmptyBounds = emptyBoundsCopy(one)
  def main(args: Array[String]): Unit =
    assert(fromLocalEntity.sourceName == "fromLocalEntity")
    assert(fromLocalObservation.sourceName == "fromLocalObservation")
    assert(fromLocalBounds.toIR.name == "fromLocalBounds")
    assert(entityAlias.sourceName == "inferred")
    assert(empty.sourceName == "")
    assert(observationAlias.sourceName == "read")
    assert(emptyRead.sourceName == "")
    assert(limitsAlias.toIR.name == "one")
    assert(emptyOne.toIR.name == "")
    assert(actionAlias.toIR.name == "send")
    assert(emptySend.toIR.name == "")
    assert(alias.toIR.id == "framework.residualclient.Client.fair")
    assert(alias.toIR.name == "fair")
    assert(emptyFair.toIR.name == "")
    assert(entityCopy.sourceName == "inferred")
    assert(explicitEmptyEntity.sourceName == "")
    assert(observationCopy.sourceName == "read")
    assert(explicitEmptyObservation.sourceName == "")
    assert(boundsCopy.toIR.name == "one")
    assert(boundsCopy.toIR.steps == 2)
    assert(explicitEmptyBounds.toIR.name == "")
    assert(inferred.productElement(0).equals(""))
    assert(read.productElement(0).equals(""))
    assert(one.productElement(0).equals(""))
    println("helper-controls:passed")

val progress = Subject.leadsTo("helper")(s => !s, s => s, 1, Client.fair)
