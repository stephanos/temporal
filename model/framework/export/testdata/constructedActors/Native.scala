package framework.actorcontrols

import java.util.Base64
import framework.dsl.*
import fixture.constructedactor.*
import io.temporal.server.api.umpire.v1 as ir

object Native:
  private var reads = 0
  private var arguments = 0
  private var calls = 0
  object Getter extends Actor("constructor"):
    override def name: String =
      reads += 1
      "getter"
  private def argument(): String =
    arguments += 1
    "argument"
  private def helper(value: Actor): Actor =
    calls += 1
    value
  val value = Actor(argument())
  val alias = value
  val emptyValue = Actor("")
  val emptyAlias = emptyValue
  val emptyAliasAction = action("emptyAlias", emptyAlias)
  val helped = action("helped", helper(alias))
  val getterAction = action("getter", Getter)
  val moduleAlias: Actor = Named
  val aliased = action("aliased", moduleAlias)
  val constructed = action("constructed", new Actor(named = "new"))
  val omitted = action("omitted", new Actor())

  class Base extends Actor("base"):
    val inherited = action("inherited", this)
    val earlyFailure: String = try
      inherited.toIR
      "accepted"
    catch case error: IllegalArgumentException => error.getMessage
    def converted: ir.Action = inherited.toIR
  object Child extends Base
  trait Early extends Actor:
    val earlyFailure: String =
      val value = action("early", this).declaredAt("framework.actorcontrols.EarlyChild.early")
      try
        value.toIR
        "accepted"
      catch case error: IllegalArgumentException => error.getMessage
  object EarlyChild extends Actor with Early

  def main(args: Array[String]): Unit =
    val actions = Vector(
      Default.defaultOwn,
      Empty.emptyOwn,
      Named.namedOwn,
      Outer.Nested.nestedOwn,
      `Odd-name`.oddOwn,
      fixture.constructedactor.Inherited.inheritedOwn,
      Actions.defaultRef,
      Actions.emptyRef,
      Actions.namedRef,
      Actions.system,
      Actions.factory,
      Actions.emptyFactory,
      Actions.alias,
      Actions.modified
    )
    val model = Signature.collect(actions, Nil)
    println("actors:" + Base64.getEncoder.encodeToString(model.toByteArray))
    assert(Named.name == "native-name")
    assert(Named.toString == "native-name")
    assert(Actions.emptyFactory.decl.actor.name == "actor")
    assert(Actions.alias.decl.actor eq Actions.factoryValue)
    assert(helped.decl.actor eq value)
    assert(emptyAliasAction.decl.actor eq emptyValue)
    assert(emptyAliasAction.toIR.actor == "")
    assert(helped.toIR.actor == "argument")
    assert(arguments == 1 && calls == 1)
    assert(getterAction.toIR.actor == "getter")
    assert(reads == 0)
    assert(Getter.name == "getter" && Getter.toString == "getter")
    assert(reads == 2)
    assert(aliased.toIR.actor == "named")
    assert(constructed.toIR.actor == "new")
    assert(omitted.toIR.actor == "")
    assert(Child.converted.actor == "child")
    assert(Child.earlyFailure.contains("action framework.actorcontrols.Native.Base.inherited"))
    assert(Child.earlyFailure.contains("no retained Actor module name"))
    assert(EarlyChild.earlyFailure.contains("framework.actorcontrols.EarlyChild.early"))
    assert(EarlyChild.earlyFailure.contains("no retained Actor module name"))
    assert(Child.name == "base")
    println("controls:passed")
