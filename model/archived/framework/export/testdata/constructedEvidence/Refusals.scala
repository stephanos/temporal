package fixture.constructedevidence

import framework.dsl.*
import framework.dsl.Finite.given

object MissingFields extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object PartialFields extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Detailed, String] =
    case Detailed.item(true) => "true only"
  object rules extends Bindings()

object GuardedFields extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Detailed, String] =
    case Detailed.item(flag) if flag => "guarded"
  object rules extends Bindings()

object UnlistedFields extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Detailed, String] =
    case Detailed.early => "early"
  object rules extends Bindings()

object RecordMatch extends Machine[Boolean, Boolean, RecordFact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: RecordFact => String = f =>
    f match
      case RecordFact(_) => "record"
  object rules extends Bindings()

object RecordWildcard extends Machine[Boolean, Boolean, RecordFact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: RecordFact => String = f =>
    f match
      case _ => "record"
  object rules extends Bindings()

object Stepping extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  def forbidden(f: Fact): String =
    val steps = List(Step(true, f == Fact.first))
    if steps.isEmpty then "empty" else "steps"
  val evidence: Fact => String = forbidden
  object rules extends Bindings()

object DefEvidence extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  def evidence(f: Fact): String = if f == Fact.first then "one" else "other"
  object rules extends Bindings()
