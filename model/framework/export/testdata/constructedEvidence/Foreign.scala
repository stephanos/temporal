package fixture.constructedevidence

import framework.dsl.*
import framework.dsl.Finite.given

val _ = () => summon[CanEqual[Fact, Fact]]

object Foreign:
  def text(fact: Fact): String = if fact == Fact.first then "foreign first" else "foreign rest"

object Opaque:
  val callable: Fact => String = _ => "opaque"
