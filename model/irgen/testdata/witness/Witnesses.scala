package fixture.witness

import framework.*
import framework.Finite.given
import framework.realize.*

enum Phase:
  case open, closed
enum Answer:
  case accepted
enum Note:
  case completed, other
enum Input:
  case first, second
val _ = () => summon[CanEqual[Phase, Phase]]

object worker extends Actor:
  val begin = action(this)
  val finish = action(this).input[Input]("input")
  val outside = action(this)
import worker.{begin, finish}

def beginStep(s: Phase): List[Step[Phase, Answer, Note]] = List(
  Step(Answer.accepted, s)
)
def finishStep(s: Phase, input: Input): List[Step[Phase, Answer, Note]] =
  List(
    Step(
      Answer.accepted,
      if input == Input.first then Phase.closed else s,
      List(Note.completed)
    )
  )
def closed(s: Phase): Boolean = s == Phase.closed

object Subject extends Machine[Phase, Answer, Note]:
  def init = Phase.open
  def end(s: Phase) = closed(s)
  object rules extends Bindings(begin ~> beginStep, finish ~> finishStep)
  object queries:
    val discovered =
      witness(begin(), finish(Input.first)).records(Note.completed)

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
val space =
  Exploration("edits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)
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
val coreProperty =
  Subject.property("completion").when(finish(Input.first)) holds { after =>
    after.records(Note.completed) && closed(after.state)
  }
val coreScenario =
  Subject.scenario("completion").actions(begin(), finish(Input.first))
val core: Query =
  (query("completion") find coreProperty in coreScenario limits bounded)
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

val derived = DerivedSubject
  .witness("derived")(finish(Input.second))
  .records(Note.completed)
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
val composedCore: Query =
  query("composed") find composedProperty in composedScenario limits one

val repeated = Repeated
  .witness("repeated")(Repeated.own(_.right, finish(Input.first)))
  .records(_.right -> Subject, Note.completed)
val reversed = Reversed
  .witness("reversed")(Reversed.own(_.right, finish(Input.first)))
  .records(_.right -> Subject, Note.completed)
val sync =
  Synced
    .witness("sync")(Synced.synced(_.right -> begin))
    .records(_.left -> Subject, Note.completed)
val distinct =
  Subject
    .witness("distinct")(finish(Input.first), finish(Input.second))
    .records(Note.completed)
val distinctMembers = Together
  .witness("distinctMembers")(
    Together.own(_.left, finish(Input.first)),
    Together.own(_.right, finish(Input.first))
  )
  .records(_.right -> Other, Note.completed)
val listed = Vector(
  Subject.witness("listed")(finish(Input.second)).records(Note.other)
)
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

val conditional = Subject
  .witness("conditional")(finish(Input.first))
  .records(Note.completed)
  .recordsWhen(closed, Note.other)
  .ends(closed)
val conditionalProperty =
  Subject.property("conditional").when(finish(Input.first)) holds { after =>
    after.records(Note.completed) && (!closed(after.state) || after.records(
      Note.other
    )) && closed(after.state)
  }
val conditionalScenario =
  Subject.scenario("conditional").actions(finish(Input.first))
val conditionalCore: Query =
  query("conditional") find conditionalProperty in conditionalScenario limits Limits(
    "conditional",
    steps = 1,
    actions = 1,
    search = 512
  )
val conditionalMember = Together
  .witness("conditionalMember")(Together.own(_.left, finish(Input.first)))
  .records(_.left -> Subject, Note.completed)
  .recordsWhen((s: Joined) => closed(s.right), _.right -> Other, Note.other)

val absent = Subject.witness("absent")(finish(Input.first)).excludes(Note.other).ends(closed)
val absentProperty = Subject.property("absent").when(finish(Input.first)) holds { after =>
  !after.records(Note.other) && closed(after.state)
}
val absentScenario = Subject.scenario("absent").actions(finish(Input.first))
val absentCore: Query = query("absent") find absentProperty in absentScenario limits Limits(
  "absent",
  steps = 1,
  actions = 1,
  search = 512
)
val guarded = Subject
  .witness("guarded")(finish(Input.first))
  .records(Note.completed)
  .excludes(Note.other)
  .recordsWhen(closed, Note.completed)
val absentMember = Together
  .witness("absentMember")(Together.own(_.left, finish(Input.first)))
  .excludes(_.right -> Other, Note.completed)

val guardedProperty = Subject.property("guarded").when(finish(Input.first)) holds { after =>
  after.records(Note.completed) && !after
    .records(Note.other) && (!closed(after.state) || after.records(Note.completed))
}
val guardedScenario = Subject.scenario("guarded").actions(finish(Input.first))
val guardedCore: Query = query("guarded") find guardedProperty in guardedScenario limits Limits(
  "guarded",
  steps = 1,
  actions = 1,
  search = 512
)
val absentMemberProperty =
  Together.property("absentMember").when(Together.own(_.left, finish(Input.first))) holds { after =>
    !after.facts.contains("right_completed")
  }
val absentMemberScenario =
  Together.scenario("absentMember").actions(Together.own(_.left, finish(Input.first)))
val absentMemberCore: Query =
  query("absentMember") find absentMemberProperty in absentMemberScenario limits Limits(
    "absentMember",
    steps = 1,
    actions = 1,
    search = 512
  )
val conditionalMemberProperty =
  Together.property("conditionalMember").when(Together.own(_.left, finish(Input.first))) holds {
    after =>
      after.facts.contains("left_completed") && (!closed(after.state.right) || after.facts.contains(
        "right_other"
      ))
  }
val conditionalMemberScenario =
  Together.scenario("conditionalMember").actions(Together.own(_.left, finish(Input.first)))
val conditionalMemberCore: Query = query(
  "conditionalMember"
) find conditionalMemberProperty in conditionalMemberScenario limits Limits(
  "conditionalMember",
  steps = 1,
  actions = 1,
  search = 512
)
