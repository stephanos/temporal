package samples



// Native collections in state (model/design/reachable-exploration.md §6), reified at compile time.
object Ledger:
  enum Acct:
    case alice, bob

  final case class Books(balances: Map[Acct, Int], open: Set[Acct])

  // deposit one unit into an open account
  def deposit(s: Books, a: Acct): Books =
    if s.open.contains(a) then s.copy(balances = s.balances.updated(a, s.balances.getOrElse(a, 0) + 1))
    else s

  // close an account: it leaves the open set and its balance is dropped
  def close(s: Books, a: Acct): Books =
    s.copy(open = s.open.excl(a), balances = s.balances.removed(a))

  val depositIR: String = framework.reify.Reify.function(deposit)
  val closeIR: String = framework.reify.Reify.function(close)
  // A traversal the IR design names but this spike does not reify, refused at compile time:
  // val totalIR = framework.reify.Reify.function((s: Books) => s.balances.values.sum)

@main def printLedger(): Unit =
  println(Ledger.depositIR)
  println(Ledger.closeIR)
