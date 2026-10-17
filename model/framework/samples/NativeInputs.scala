package samples

import framework.dsl.*
import framework.dsl.Finite.given

object NativeInputs:
  final case class Request(amount: Int = 0, ceiling: Int = 2)
  final case class AcceptedState(seen: Set[Int])

  object caller extends Actor:
    val submit = action(this).inputs[Request](
      inputField[Request](_.amount).in(domains.oneOf(2, 0)),
      inputField[Request](_.ceiling).in(domains.oneOf(1 to 2))
    )

  object Accepted extends RecordMachine[AcceptedState, Boolean, Nothing]:
    val init = AcceptedState(Set.empty)
    def end(s: AcceptedState): Boolean = s.seen.nonEmpty

    object rules extends Rules:
      on(caller.submit):
        where((s, in) => in.amount <= in.ceiling && !s.seen.contains(in.amount)) ~>
          ((s: AcceptedState, in: Request) =>
            List(Step(true, s.copy(seen = s.seen.incl(in.amount))))
          )

    object postconditions:
      val zeroWasAccepted = property
        .after(caller.submit(amount = 0))
        .expect:
          oneOf:
            branch:
              check(after.state.seen == Set(0))
            branch:
              check(after.state.seen == Set(0, 2))
