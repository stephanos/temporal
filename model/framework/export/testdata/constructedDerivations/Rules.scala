package fixture.constructedderivations

import framework.dsl.*

object SameNamed extends Derived(Base.rebind(hand.go ~> Base.effects.replacement))
object SameLambda extends Derived(Forwarded.rebind(hand.go ~> Base.effects.replacement))
object DisabledRebound extends Derived(Disabled.rebind(hand.go ~> Base.effects.first))
object ClassRebound extends Derived(Classes.rebind(hand.one ~> Base.effects.one))
object MixedClassRebound extends Derived(MixedClasses.rebind(hand.one ~> Base.effects.one))
object Partial extends Derived(Base.rebind(on(hand.go)(always ~> Base.effects.prefixed(true))))
object Replaced
    extends Derived(Base.rebind(on(hand.go) {
      where((s: Boolean) => !s) ~> Base.effects.replacement
      where((s: Boolean) => s) ~> Base.effects.first
    }))
object Extended extends Derived(Base.extend(on(hand.more)(always ~> Base.effects.replacement)))
object EmptyOperations extends Derived(Base.rebind().extend())
object RestrictedEmpty extends Derived(Base.restrict())
object Repeated
    extends Derived(
      Base.rebind(hand.go ~> Base.effects.replacement).rebind(hand.go ~> Base.effects.first)
    )
object GroupThenBare
    extends Derived(
      Base
        .rebind(on(hand.go)(where((s: Boolean) => !s) ~> Base.effects.replacement))
        .rebind(hand.go ~> Base.effects.first)
    )
object BareThenGroup
    extends Derived(
      Base
        .rebind(hand.go ~> Base.effects.replacement)
        .rebind(on(hand.go)(where((s: Boolean) => !s) ~> Base.effects.first))
    )
object DerivedAgain extends Derived(SameNamed.rebind(hand.go ~> Base.effects.first))
object RestrictedReplaced
    extends Derived(Base.restrict(hand.go).rebind(on(hand.go)(always ~> Base.effects.replacement)))
object ExtendRestrict
    extends Derived(
      Base.extend(on(hand.more)(always ~> Base.effects.replacement)).restrict(hand.more)
    )
object RestrictedExtend
    extends Derived(Base.restrict().extend(on(hand.more)(always ~> Base.effects.replacement)))
object CoreReplaced extends Derived(Core.rebind(on(hand.go)(always ~> Base.effects.replacement)))
object CoreExtended extends Derived(Core.extend(on(hand.more)(always ~> Base.effects.replacement)))
object Mixed
    extends Derived(
      Core
        .extend(on(hand.more)(always ~> Base.effects.replacement))
        .extend(hand.extra ~> Base.effects.second)
    )
object Arities
    extends Derived(
      Base.extend(
        on(hand.one)(always ~> Base.effects.one),
        on(hand.two)(always ~> ((s: Boolean, in: Two) => Base.effects.two(s, in.a, in.b))),
        on(hand.three)(always ~> ((s: Boolean, in: Three) => Base.effects.three(s, in.a, in.b, in.c))),
        on(hand.four)(always ~> ((s: Boolean, in: Four) => Base.effects.four(s, in.a, in.b, in.c, in.d))),
        on(hand.five)(always ~> ((s: Boolean, in: Five) => Base.effects.five(s, in.a, in.b, in.c, in.d, in.e))),
        on(hand.six)(always ~> ((s: Boolean, in: Six) => Base.effects.six(s, in.a, in.b, in.c, in.d, in.e, in.f)))
      )
    )
object CoreAnonymous extends Derived(Core.rebind(hand.go ~> ((s: Boolean) => List(Step(false, s)))))
object Curried
    extends Derived(
      Base.extend(
        on(hand.one)(always ~> ((s: Boolean, flag: Boolean) => Base.effects.curried(s)(flag)))
      )
    )
