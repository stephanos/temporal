package fixture.finiteschemahelper

import framework.dsl.*
import fixture.finiteschemaroles.{Phase, Snapshot}

// Native state needs its admission evidence: a generic S is refused without StateData.
abstract class Generic[S, O, F](using Finite[S], Finite[O], Finite[F], StateData[S])
    extends Machine[S, O, F]
// A values-only witness; the reify plugin does not compile an anonymous `new Finite[T]:`.
final class Unread[T](message: String)(using Data.Structural[T]) extends Finite[T]:
  def values = throw AssertionError(message)
object Helper:
  val phase: Finite[Phase] = Unread[Phase]("helper role schema enumerated catalog")
  val snapshot: Finite[Snapshot] = Unread[Snapshot]("helper record schema enumerated catalog")
  def forward[A](witness: Finite[A]): Finite[A] = witness
