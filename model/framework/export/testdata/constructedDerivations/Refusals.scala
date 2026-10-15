package fixture.constructedderivations

import framework.dsl.*

object EmptyGroup extends Derived(Base.extend(on(hand.more) {}))
object ClassMerge extends Derived(ClassAmbiguous.rebind(hand.one ~> Base.effects.one))
object DerivedAfterReplaced extends Derived(Replaced.rebind(hand.go ~> Base.effects.first))
object DistinctNamed extends Derived(Distinct.rebind(hand.go ~> Base.effects.replacement))
object HiddenInvalid
    extends Derived(Distinct.rebind(hand.go ~> Base.effects.replacement).restrict())
object OverwrittenInvalid
    extends Derived(
      Distinct
        .rebind(hand.go ~> Base.effects.replacement)
        .rebind(on(hand.go)(always ~> Base.effects.first))
    )
object DisabledExtended extends Derived(Disabled.extend(hand.more ~> Base.effects.first))
object RestrictedBareExtended
    extends Derived(Base.restrict().extend(hand.more ~> Base.effects.first))
object DuplicateReplace
    extends Derived(
      Base.rebind(
        on(hand.go)(always ~> Base.effects.first),
        on(hand.go)(always ~> Base.effects.first)
      )
    )
object DuplicateExtend
    extends Derived(
      Base.extend(
        on(hand.more)(always ~> Base.effects.first),
        on(hand.more)(always ~> Base.effects.first)
      )
    )
object UnknownReplace extends Derived(Base.rebind(on(hand.more)(always ~> Base.effects.first)))
object AlreadyBound extends Derived(Base.extend(on(hand.go)(always ~> Base.effects.first)))
object HiddenUnknown
    extends Derived(Base.rebind(on(hand.more)(always ~> Base.effects.first)).restrict())
object Anonymous
    extends Derived(Base.rebind(on(hand.go)(always ~> ((s: Boolean) => List(Step(false, s))))))
