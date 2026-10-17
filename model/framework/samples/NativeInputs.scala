package samples

import framework.dsl.*
import framework.dsl.Finite.given

object NativeInputs:
  final case class Request(amount: Int = 0, ceiling: Int = 2)

  given Ok[Boolean] = Ok(true)

  object caller extends Actor:
    val submit = action.inputs[Request](
      inputField[Request](_.amount).in(domains.oneOf(2, 0)),
      inputField[Request](_.ceiling).in(domains.oneOf(1 to 2))
    )

  object Accepted extends Machine[Boolean, Nothing]:
    var seen = Set.empty[Int]

    override def end = seen.nonEmpty

    object transitions extends Transitions:
      on(caller.submit): in =>
        if in.amount <= in.ceiling && !seen.contains(in.amount) then effects.submit(in)

    object effects:
      def submit(in: Request) =
        seen = seen.incl(in.amount)

    object postconditions:
      val zeroWasAccepted = after(caller.submit(amount = 0)):
        oneOf:
          branch:
            check(seen == Set(0))
          branch:
            check(seen == Set(0, 2))
