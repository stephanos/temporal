package framework.dsl
package compose

import Joined.*

// Compositions the runtime refuses when it reads them, each with a located message.
object Refused:
  // Subject restricted to start: a derivation, so it holds Subject's vars.
  object SubjectAgain extends Derived(Subject.restrict(hand.start))

  enum TwiceFact:
    case left(value: Fact)
    case right(value: Fact)

  // Two members that hold one Machine's vars.
  object Doubled extends Composition[TwiceFact]:
    val left = Subject
    val right = SubjectAgain

  // A sync of an action both members bind, without saying whose.
  object Ambiguous extends Composition[JoinedFact]:
    val left = Subject
    val right = Other

    object syncs:
      val started = sync(hand.start, hand.start)

  // A sync of an action one member does not bind.
  object Unbound extends Composition[JoinedFact]:
    val left = Subject
    val right = Other

    object syncs:
      val finished = sync(left(hand.finish), right(hand.finish))
