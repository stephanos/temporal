package fixture.constructedactor

import framework.dsl.*
import fixture.constructedactor.Named as Renamed
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object Forms:
  val imported = action(Renamed)
  val namedFactory = action(Actor(name = "named-factory"))
  val emptyValue = Actor("")
  val emptyAlias = emptyValue
  val emptyAliased = action(emptyAlias)
  val constructed = action(new Actor("new"))
  val namedConstructor = action(new Actor(named = "named-new"))
  val omittedConstructor = action(new Actor())
  val moduleValue: Actor = Named
  val moduleAlias = action(moduleValue)
  def helper(name: String): Actor = Actor(name)
  val helped = action(helper("helper"))
  class Base extends Actor("base"):
    val inherited = action(this)
  object Child extends Base
  val importedFair = assume("imported").fair(imported)
  val namedFactoryFair = assume("namedFactory").fair(namedFactory)
  val emptyAliasedFair = assume("emptyAliased").fair(emptyAliased)
  val constructedFair = assume("constructed").fair(constructed)
  val namedConstructorFair = assume("namedConstructor").fair(namedConstructor)
  val omittedConstructorFair = assume("omittedConstructor").fair(omittedConstructor)
  val moduleAliasFair = assume("moduleAlias").fair(moduleAlias)
  val helpedFair = assume("helped").fair(helped)
  val inheritedFair = assume("inherited").fair(Child.inherited)
  def main(args: Array[String]): Unit =
    val model = ir.Model(actions = Seq(imported, namedFactory, emptyAliased).map(_.toIR))
    println("forms:" + Base64.getEncoder.encodeToString(model.toByteArray))
    assert(emptyAliased.decl.actor eq emptyValue)
    assert(moduleAlias.decl.actor eq Named)
    assert(constructed.toIR.actor == "new")
    assert(namedConstructor.toIR.actor == "named-new")
    assert(omittedConstructor.toIR.actor == "")
    assert(moduleAlias.toIR.actor == "named")
    assert(helped.toIR.actor == "helper")

val importedProgress = Subject.leadsTo("imported")(s => !s, s => s, 1, Forms.importedFair)
val namedFactoryProgress =
  Subject.leadsTo("namedFactory")(s => !s, s => s, 1, Forms.namedFactoryFair)
val emptyAliasedProgress =
  Subject.leadsTo("emptyAliased")(s => !s, s => s, 1, Forms.emptyAliasedFair)
val constructedProgress = Subject.leadsTo("constructed")(s => !s, s => s, 1, Forms.constructedFair)
val namedConstructorProgress =
  Subject.leadsTo("namedConstructor")(s => !s, s => s, 1, Forms.namedConstructorFair)
val omittedConstructorProgress =
  Subject.leadsTo("omittedConstructor")(s => !s, s => s, 1, Forms.omittedConstructorFair)
val moduleAliasProgress = Subject.leadsTo("moduleAlias")(s => !s, s => s, 1, Forms.moduleAliasFair)
val helpedProgress = Subject.leadsTo("helped")(s => !s, s => s, 1, Forms.helpedFair)
val inheritedProgress = Subject.leadsTo("inherited")(s => !s, s => s, 1, Forms.inheritedFair)
