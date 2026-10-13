// P counterpart: p-org/P/Tutorial/1_ClientServer/PSrc/Server.p
// Quint counterpart: quint-co/quint/quint/README.md
package samples

import framework.dsl.*
import framework.dsl.Finite.given

object Bank:
  enum Client:
    case alice, bob

  enum Balance:
    case empty, one, two

  final case class Response(client: Client, balance: Balance)
  final case class Database(alice: Balance, bob: Balance, pending: Option[Response])

  enum Outcome:
    case accepted
    case balance(client: Client, value: Balance)

  given Ok[Outcome] = Ok(Outcome.accepted)

  object clients extends Actor:
    val withdraw = action(this).input[Client]("client")

  object server extends Actor:
    val respond = action(this)

  def debited(balance: Balance): Balance = balance match
    case Balance.empty => Balance.empty
    case Balance.one   => Balance.empty
    case Balance.two   => Balance.one

  object Model extends Machine[Database, Outcome, Nothing]:
    val init = Database(Balance.two, Balance.two, None)
    def end(database: Database) = database.pending == None

    object effects:
      def withdrawAlice(database: Database): List[Step[Database, Bank.Outcome, Nothing]] =
        val balance = debited(database.alice)
        enter(database.copy(alice = balance, pending = Some(Response(Client.alice, balance))))

      def withdrawBob(database: Database): List[Step[Database, Bank.Outcome, Nothing]] =
        val balance = debited(database.bob)
        enter(database.copy(bob = balance, pending = Some(Response(Client.bob, balance))))

      def answer(database: Database): List[Step[Database, Bank.Outcome, Nothing]] =
        database.pending match
          case Some(response) =>
            List(
              Step(
                Outcome.balance(response.client, response.balance),
                database.copy(pending = None)
              )
            )
          case None => enter(database)

    object rules extends Rules:
      on(clients.withdraw(Client.alice))(where(_.pending == None) ~> effects.withdrawAlice)
      on(clients.withdraw(Client.bob))(where(_.pending == None) ~> effects.withdrawBob)
      on(server.respond)(where(_.pending != None) ~> effects.answer)

  val correctBalance: Property[Database] = Model.property.when(server.respond) holds { after =>
    after.outcome match
      case Outcome.balance(Client.alice, value) => value == after.state.alice
      case Outcome.balance(Client.bob, value)   => value == after.state.bob
      case Outcome.accepted                     => false
  }
  val anyRequest: Scenario[Database] = Model.scenario.starts(Model.init).free
  val short = Limits(steps = 4, actions = 4, search = 512)
  val fairResponses = assume.fair(server.respond)
  val everyWithdrawalGetsResponse: Progress[Database] =
    Model.leadsTo("everyWithdrawalGetsResponse")(
      _.pending != None,
      _.pending == None,
      within = 1,
      fairResponses
    )
  val queries = Vector(query verify correctBalance in anyRequest limits short)
