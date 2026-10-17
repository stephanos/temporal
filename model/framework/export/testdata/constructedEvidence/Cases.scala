package fixture.constructedevidence

import framework.dsl.*
import framework.dsl.Finite.given

object Default extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object Partial extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Fact, String] =
    case f @ Fact.second if f == Fact.first => "second guarded"
    case Fact.fourth                        => "last"
  object rules extends Bindings()

object Overlap extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Fact, String] =
    case Fact.third | Fact.fourth => "late"
    case chosen @ Fact.first      => if chosen == Fact.first then "bound" else "never"
    case _                        => "rest"
  object rules extends Bindings()

object Fields extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: PartialFunction[Detailed, String] =
    case Detailed.item(flag) if flag => "specific"
    case Detailed.item(flag)         => if flag then "yes" else "no"
  object rules extends Bindings()

object Constant extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Detailed => String = _ => "constant"
  object rules extends Bindings()

object Ordinary extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Fact => String = Helpers.text
  object rules extends Bindings()

object Conditional extends Machine[Boolean, Boolean, RecordFact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: RecordFact => String = f => if f.flag then "yes" else "no"
  object rules extends Bindings()

object Curried extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Fact => String = f => Helpers.curried(true)(f)
  object rules extends Bindings()

object NoFacts extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object ExplicitNothing extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Nothing => String = _ => "unreachable"
  object rules extends Bindings()

object Scalar extends Machine[Boolean, Boolean, Boolean]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object Forwarded extends Derived(Default.restrict(hand.go))
object Unmonitored extends Derived(Default.unmonitored)
object Repeated extends Derived(Forwarded.restrict(hand.more))
object Inherited extends Derived(Partial.restrict().unmonitored)

object TwoParams extends Machine[Boolean, Boolean, Detailed]:
  val init = false
  def end(s: Boolean) = s
  val evidence: (Detailed, Boolean) => String = (_, flag) => if flag then "yes" else "no"
  object rules extends Bindings()

object NonRootMatch extends Machine[Boolean, Boolean, RecordFact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: RecordFact => String = f =>
    f.flag match
      case true  => "yes"
      case false => "no"
  object rules extends Bindings()

object RuleSource extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean): List[Step[Boolean, Boolean, Fact]] = List(Step(true, s))
  object rules extends Rules:
    on(hand.go)(always ~> effects.keep)

object Mixed
    extends Derived(
      RuleSource
        .rebind(hand.go ~> RuleSource.effects.keep)
        .extend(on(hand.more)(always ~> RuleSource.effects.keep))
        .restrict(hand.more)
        .unmonitored
    )
