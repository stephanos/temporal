package framework.residualscopes

import framework.dsl.*

object Scopes:
  val given_token = assume
  val given_entity = Entity(key = "id")
  val given_observation = Observation(on = given_entity)
  val given_limits = Limits(steps = 0, actions = 0, search = 0)
  class Local:
    val assumption = assume
    val unknown = hole
  def method(): Assumption =
    val local = assume
    local
  def direct(): Assumption = assume
  def localHole(): Hole =
    val local = hole
    local
  object Anonymous:
    given Action[EmptyTuple] = action(Actor("caller"))
    given Hole = hole
    given Assumption = assume
    given Entity = Entity()
    given Observation = Observation(on = given_entity)
    given Limits = Limits(steps = 0, actions = 0, search = 0)
  object NamedGiven:
    given given_entity: Entity = Entity()
  object Aliases:
    given Assumption = given_token
    given Entity = given_entity
    given Observation = given_observation
    given Limits = given_limits
  val returned = method()
  val returnedDirect = direct()
  val returnedHole = localHole()
  def main(args: Array[String]): Unit =
    assert(given_token.toIR.name == "given_token")
    assert(given_entity.sourceName == "given_entity")
    assert(given_observation.sourceName == "given_observation")
    assert(given_limits.toIR.name == "given_limits")
    assert(Aliases.given_Assumption.toIR.name == "given_token")
    assert(Aliases.given_Entity.sourceName == "given_entity")
    assert(Aliases.given_Observation.sourceName == "given_observation")
    assert(Aliases.given_Limits.toIR.name == "given_limits")
    // A native `assume` spells itself after a local val that declares it (dsl/Assume.scala); the
    // caller's conflicting identity is then refused, as dsl ResidualSignatureTest pins. A helper
    // that returns `assume` directly spells nothing and is still named at its concrete caller.
    assert(returned.sourceName == "local")
    assert(returnedDirect.toIR.id == "framework.residualscopes.Scopes.returnedDirect")
    assert(returnedDirect.toIR.name == "returnedDirect")
    assert(returnedHole.toIR.id == "framework.residualscopes.Scopes.returnedHole")
    def refuses(value: => Any, kind: String): Unit =
      val failed = try
        value
        false
      catch case e: IllegalArgumentException => e.getMessage.contains(kind)
      assert(failed)
    val local = new Local
    refuses(local.assumption.toIR, "assumption")
    refuses(local.unknown.toIR, "hole")
    refuses(method().toIR, "source identity")
    refuses(returned.toIR, "source identity")
    refuses(localHole().toIR, "source identity")
    refuses(Anonymous.given_Assumption.toIR, "assumption")
    refuses(Anonymous.given_Action_EmptyTuple.toIR, "action")
    refuses(Anonymous.given_Hole.toIR, "hole")
    refuses(NamedGiven.given_entity.sourceName, "entity")
    refuses(Anonymous.given_Entity.sourceName, "entity")
    refuses(Anonymous.given_Observation.sourceName, "observation")
    refuses(Anonymous.given_Limits.toIR, "limits")
    println("scope-controls:passed")
