package propertygroups.foundations.incompatible

import framework.dsl.*
import propertygroups.*

enum Other:
  case value

object Incompatible extends Machine[Other, Outcome, Nothing]:
  def init = Other.value
  def end(s: Other) = s == Other.value
  object properties:
    object safety:
      val invalid = shared(Incompatible)
    object liveness:
      val invalid = eventual(Incompatible)
