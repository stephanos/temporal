package fixture.constructedinbox

import framework.dsl.*

final case class Helped(box: Inbox[Message])
// A named witness; the reify plugin does not compile an anonymous `new Finite[T]:`.
final class Supplied[T](read: () => IndexedSeq[T])(using Data.Structural[T]) extends Finite[T]:
  def values: IndexedSeq[T] = read()
object Helpers:
  given helped: Finite[Helped] =
    given contents: Finite[Inbox[Message]] = first.contents
    Supplied[Helped](() => Finite.derived[Helped].values)
  def forward[A](value: Finite[A]): Finite[A] = value
