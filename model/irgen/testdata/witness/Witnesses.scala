package fixture.witness

import framework.*
import framework.Finite.given
import framework.realize.*

enum Phase derives Finite:
  case open, closed
enum Answer derives Finite:
  case accepted
enum Note derives Finite:
  case completed, other
enum Input derives Finite:
  case first, second
val _ = () => summon[CanEqual[Phase, Phase]]

object worker extends Actor:
  val begin = action(this)
  val finish = action(this).input[Input]("input")
  val outside = action(this)
import worker.{begin, finish}

def beginStep(s: Phase): List[Step[Phase, Answer, Note]] = List(Step(Answer.accepted, s))
def finishStep(s: Phase, input: Input): List[Step[Phase, Answer, Note]] =
  List(
    Step(Answer.accepted, if input == Input.first then Phase.closed else s, List(Note.completed))
  )
def closed(s: Phase): Boolean = s == Phase.closed

object Subject extends Machine[Phase, Answer, Note]:
  def init = Phase.open
  def end(s: Phase) = closed(s)
  object rules extends Bindings(begin ~> beginStep, finish ~> finishStep)
  object queries:
    val discovered = witness(begin(), finish(Input.first)).records(Note.completed)

object DerivedSubject extends Derived(Subject.restrict(begin, finish))
object Other extends Derived(Subject.restrict(begin, finish))
final case class Joined(left: Phase, right: Phase)
object Together extends Composition[Joined](_.left -> Subject, _.right -> Other):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs
object Reversed extends Composition[Joined](_.left -> Other, _.right -> Subject):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs
object Repeated extends Composition[Joined](_.left -> Subject, _.right -> Subject):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs
object Synced extends Composition[Joined](_.left -> Subject, _.right -> Other):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs:
    sync("both", _.left -> begin, _.right -> begin)
object SyncedInput extends Composition[Joined](_.left -> Subject, _.right -> Other):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs:
    sync("finishBoth", _.left -> finish, _.right -> begin)

val bounded = Limits("bounded", steps = 3, actions = 3, search = 64)
val space = Exploration("edits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)
val expected = RunExpectation(
  Conformance.conformant,
  PropertyOutcome.satisfied,
  PropertyOutcome.satisfied,
  Disposition.completed,
  Cleanup.succeeded
)

val machine = Subject
  .witness("completion")(begin(), finish(Input.first))
  .records(Note.completed)
  .ends(closed)
  .limits(bounded)
  .live(expected)
  .explore(space)
  .total(4)
val coreProperty = Subject.property("completion").when(finish(Input.first)) holds { after =>
  after.records(Note.completed) && closed(after.state)
}
val coreScenario = Subject.scenario("completion").actions(begin(), finish(Input.first))
val core: Query = (query("completion") find coreProperty in coreScenario limits bounded)
  .explore(space)
  .live(expected)
  .total(4)

val machineReordered = Subject
  .witness("completion")(begin(), finish(Input.first))
  .records(Note.completed)
  .live(expected)
  .explore(space)
  .limits(bounded)
  .ends(closed)
  .total(4)

val derived = DerivedSubject.witness("derived")(finish(Input.second)).records(Note.completed)
val composed = Together
  .witness("composed")(Together.own(_.left, finish(Input.first)))
  .records(_.left -> Subject, Note.completed)
val composedProperty = Together
  .property("composed")
  .when(Together.own(_.left, finish(Input.first))) holds { after =>
  after.facts.contains("left_completed")
}
val composedScenario = Together
  .scenario("composed")
  .actions(Together.own(_.left, finish(Input.first)))
val one = Limits("composed", steps = 1, actions = 1, search = 512)
val composedCore: Query = query("composed") find composedProperty in composedScenario limits one

val repeated = Repeated
  .witness("repeated")(Repeated.own(_.right, finish(Input.first)))
  .records(_.right -> Subject, Note.completed)
val reversed = Reversed
  .witness("reversed")(Reversed.own(_.right, finish(Input.first)))
  .records(_.right -> Subject, Note.completed)
val sync =
  Synced.witness("sync")(Synced.synced(_.right -> begin)).records(_.left -> Subject, Note.completed)
val distinct =
  Subject.witness("distinct")(finish(Input.first), finish(Input.second)).records(Note.completed)
val distinctMembers = Together
  .witness("distinctMembers")(
    Together.own(_.left, finish(Input.first)),
    Together.own(_.right, finish(Input.first))
  )
  .records(_.right -> Other, Note.completed)
val listed = Vector(Subject.witness("listed")(finish(Input.second)).records(Note.other))
val distinctSync = SyncedInput
  .witness("distinctSync")(
    SyncedInput.synced(_.left -> finish(Input.first)),
    SyncedInput.synced(_.left -> finish(Input.second))
  )
  .records(_.left -> Subject, Note.completed)
val composedWrongInput = Together
  .witness("composed")(Together.own(_.left, finish(Input.second)))
  .records(_.left -> Subject, Note.completed)
val composedWrongMember = Together
  .witness("composed")(Together.own(_.right, finish(Input.first)))
  .records(_.right -> Other, Note.completed)
