package fixture.schemaoracle

import framework.dsl.*

enum Mode:
  case ready
  case stopped(flag: Boolean)

enum Token:
  case empty
  case impossible(value: Nothing)

final case class Payload(mode: Mode, spare: Option[Boolean])

object bounded:
  opaque type Retries = Int
  object Retries:
    given Finite[Retries] = Finite.upTo(2)
    val zero: Retries = 0

// Native state refuses an opaque Int field, so the retries count its bound with UpTo.
final case class State(
    payload: Option[Payload],
    count: UpTo[2],
    retries: UpTo[2],
    token: Token,
    optionalRetries: Option[UpTo[2]],
    small: Option[UpTo[1]],
    large: Option[UpTo[3]]
)

object Subject extends Machine[State, Boolean, Nothing]:
  val init = State(None, UpTo(0), UpTo(0), Token.empty, None, None, None)
  def end(s: State) = true
  object rules extends Rules
