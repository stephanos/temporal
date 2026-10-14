package fixture.finiteschemahelper

import framework.dsl.*
import fixture.finiteschemaroles.{Phase, Snapshot}

abstract class Generic[S, O, F](using Finite[S], Finite[O], Finite[F]) extends Machine[S, O, F]
object Helper:
  val phase: Finite[Phase] = new Finite[Phase]:
    def values = throw AssertionError("helper role schema enumerated catalog")
  val snapshot: Finite[Snapshot] = new Finite[Snapshot]:
    def values = throw AssertionError("helper record schema enumerated catalog")
  def forward[A](witness: Finite[A]): Finite[A] = witness
