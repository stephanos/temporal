package fixture.constructedinbox

import framework.dsl.*

final case class Helped(box: Inbox[Message])
object Helpers:
  given helped: Finite[Helped] =
    given contents: Finite[Inbox[Message]] = first.contents
    new Finite[Helped]:
      def values = Finite.derived[Helped].values
  def forward[A](value: Finite[A]): Finite[A] = value
