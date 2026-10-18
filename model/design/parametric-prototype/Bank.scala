package parametricprototype

import framework.dsl.*
import framework.dsl.Finite.given

object Bank:
  enum Outcome:
    case accepted, invalidCoinsOrInsufficientFunds, overflow

  given Ok[Outcome] = Ok(Outcome.accepted)

  enum Denom:
    case banana, burger

  final case class Balances(of: Map[Int, Map[Denom, Int]]):
    def apply(addr: Int, denom: Denom): Int =
      of.getOrElse(addr, Map.empty).getOrElse(denom, 0)

    def set(addr: Int, denom: Denom, amount: Int): Balances =
      val coins = of.getOrElse(addr, Map.empty)
      val next = if amount == 0 then coins.removed(denom) else coins.updated(denom, amount)
      Balances(if next.isEmpty then of.removed(addr) else of.updated(addr, next))

  final case class Send(fromAddr: Int, toAddr: Int, denom: Denom, amount: Int)

  class Model[Addr: Finite](holder: Addr, supply: Int) extends Machine[Outcome, Nothing]:
    val addresses =
      summon[Finite[Addr]].values

    def index(addr: Addr) =
      addresses.indexOf(addr)

    def balance(addr: Addr, denom: Denom) =
      balances(index(addr), denom)

    object client extends Actor:
      lazy val send =
        action.inputs[Send](
          inputField[Send](_.fromAddr).in(domains.oneOf(addresses.indices)),
          inputField[Send](_.toAddr).in(domains.oneOf(addresses.indices)),
          inputField[Send](_.amount).in(domains.oneOf(-1 to supply + 1))
        )

    var balances = Balances(
      Map(index(holder) -> Map(Denom.banana -> supply, Denom.burger -> supply))
    )
    var opening = oneOf(0 to supply)

    object transitions extends Transitions:
      on(client.send): in =>
        effects.send(in)

    object effects:
      def send(in: Send) =
        import in.*, Outcome.*
        val sender = balances(fromAddr, denom)
        val receiver = balances(toAddr, denom)
        def fits(amount: Int) = -supply <= amount && amount <= supply
        if sender == 0 || sender < amount then reject(invalidCoinsOrInsufficientFunds)
        else if fromAddr == toAddr then ()
        else if !fits(sender - amount) || !fits(receiver + amount) then reject(overflow)
        else
          balances = balances
            .set(fromAddr, denom, sender - amount)
            .set(toAddr, denom, receiver + amount)

    object invariants:
      val totalSupply =
        check(Denom.values.forall(denom => addresses.indices.map(balances(_, denom)).sum == supply))

    object postconditions:
      val failedSendChangesNothing =
        if outcome != Outcome.accepted then check(balances == before.balances)
