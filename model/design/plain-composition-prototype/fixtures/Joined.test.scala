package framework.dsl
package compose

import framework.dsl.Finite.given

// The framework's composition fixtures (dsl/Witness.test.scala `Together`, dsl/Expect.test.scala
// `Reordered`, dsl/IrFile.test.scala and dsl/Rules.test.scala's one-member derivations) in the plain
// form: two plain Machines that both bind `start`, composed with `start` synced, and a derivation
// that replaces one member.
object Joined:
  enum Phase derives CanEqual:
    case open, closed

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  enum Fact:
    case completed, reopened

  enum Foreign:
    case completed

  object hand extends Actor:
    val start = action
    val finish = action

  // The left member: closes on start, reopens on finish.
  object Subject extends Machine[Outcome, Fact]:
    var phase = Phase.open

    // a helper of the member, which a composition's claim calls on the member
    def isClosed: Boolean = phase == Phase.closed

    override def end = isClosed

    object transitions extends Transitions:
      on(hand.start):
        if phase == Phase.open then effects.close()
      on(hand.finish):
        if isClosed then effects.reopen()

    object effects:
      def close() =
        phase = Phase.closed
        record(Fact.completed)
      def reopen() =
        phase = Phase.open
        record(Fact.reopened)

  // The right member: closes on start, once.
  object Other extends Machine[Outcome, Foreign]:
    var phase = Phase.open

    object transitions extends Transitions:
      on(hand.start):
        if phase == Phase.open then effects.close()

    object effects:
      def close() =
        phase = Phase.closed
        record(Foreign.completed)
      def acknowledge() =
        record(Foreign.completed)

  // Other, late: it acknowledges a start and stays open.
  object LateOther
      extends Derived(
        Other.rebind(
          on(hand.start):
            if Other.phase == Phase.open then Other.effects.acknowledge()
        )
      )

  // The composition's facts: one case per member, carrying that member's fact (fn-190 R17).
  enum JoinedFact:
    case left(value: Fact)
    case right(value: Foreign)

  object Together extends Composition[JoinedFact]:
    val left = Subject
    val right = Other

    override def end = left.end && right.phase == Phase.closed

    object syncs:
      // both bind start, so each end names its member
      val started = sync(left(hand.start), right(hand.start))

    object invariants:
      val rightClosedWithLeft =
        if left.isClosed then check(right.phase == Phase.closed)

    object postconditions:
      val startClosesBoth =
        after(syncs.started):
          check(before.left.phase == Phase.open && before.right.phase == Phase.open)
          check(left.isClosed && right.phase == Phase.closed)
          records(JoinedFact.left(Fact.completed))
          records(JoinedFact.right(Foreign.completed))
      // finish is left's alone: no sync pairs it, so the action names its step
      val finishMovesLeftAlone =
        after(hand.finish):
          check(!left.isClosed)
          check(right.phase == before.right.phase)

    object queries:
      val reopened =
        after(hand.finish):
          records(JoinedFact.left(Fact.reopened))
      val reopenedFound = find(reopened)

  // The same members in the other order (Expect.test.scala `Reordered`): the composed state lists
  // right first, and member facts keep their case.
  object Reordered extends Composition[JoinedFact]:
    val right = Other
    val left = Subject

    object syncs:
      val started = sync(left(hand.start), right(hand.start))

    object invariants:
      val rightClosedWithLeft =
        if left.isClosed then check(right.phase == Phase.closed)

  // The members with no sync: each start steps its member alone.
  object Apart extends Composition[JoinedFact]:
    val left = Subject
    val right = Other

  // Together with its right member replaced (IrFile.test.scala `JoinedReplacement`, Rules.test.scala
  // `Odd`): the same syncs, end, member order and fact type. Its own claims read the replacement
  // through the vars it shares with Other.
  object LateTogether extends Composition(Together.withMember(right = LateOther)):
    object invariants:
      val rightStaysOpen =
        check(Together.right.phase == Phase.open)
