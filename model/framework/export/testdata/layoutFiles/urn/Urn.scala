package temporal.features.urn

import framework.dsl.*

final case class Water(hot: Boolean)

enum Outcome:
  case accepted

enum FacetName:
  case heating

enum FamilyName:
  case boil

object user extends Actor:
  val flip = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
