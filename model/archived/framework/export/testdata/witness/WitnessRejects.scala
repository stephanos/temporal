package fixture.witness

import framework.dsl.*
import fixture.witness.worker.{begin, finish}

val repeatedTail =
  Subject.witness(begin(), finish(Input.first), finish(Input.first)).records(Note.completed)
val unnamed: Vector[Query] = Vector(Subject.witness(finish(Input.first)).records(Note.completed))
val shortSteps = Subject
  .witness(begin(), finish(Input.first))
  .records(Note.completed)
  .limits(Limits("shortSteps", steps = 1, actions = 2, search = 8))
val shortActions = Subject
  .witness(begin(), finish(Input.first))
  .records(Note.completed)
  .limits(Limits("shortActions", steps = 2, actions = 1, search = 8))
val wrongOwner = Subject.witness(Together.own(_.left, finish(Input.first))).records(Note.completed)
val wrongComposition = Together
  .witness(Reversed.own(_.right, finish(Input.first)))
  .records(_.left -> Subject, Note.completed)
val wrongMember = Together
  .witness(Together.own(_.left, finish(Input.first)))
  .records(_.right -> Subject, Note.completed)
val wrongSharedState = Reversed
  .witness(Reversed.own(_.right, finish(Input.first)))
  .records(_.left -> Subject, Note.completed)
val repeatedComposed = Together
  .witness(
    Together.own(_.left, finish(Input.first)),
    Together.own(_.left, finish(Input.first))
  )
  .records(_.left -> Subject, Note.completed)
val repeatedSync = Synced
  .witness(Synced.synced(_.left -> begin), Synced.synced(_.right -> begin))
  .records(_.left -> Subject, Note.completed)
val collisionProperty = Subject.property("collision") expect(check(after.records(Note.completed)))
val collisionScenario = Subject.scenario("collisionScenario").actions(finish(Input.first))
val collisionPropertyQuery: Query = query("ordinaryProperty") find collisionProperty in
  Subject.scenario("ordinarySchedule").actions(finish(Input.first)) limits bounded
val collisionScenarioQuery: Query = query("ordinaryScenario") find
  Subject
    .property("ordinaryPromise")
    .expect(check(after.records(Note.other))) in collisionScenario limits bounded
val collision = Subject.witness("collision")(finish(Input.first)).records(Note.completed)
val collisionWithScenario =
  Subject.witness("collisionScenario")(finish(Input.first)).records(Note.completed)
val collisionNamedQuery: Query = query("collisionQuery") find
  Subject.property("queryPromise").expect(check(after.records(Note.other))) in
  Subject.scenario("querySchedule").actions(finish(Input.first)) limits bounded
val collisionWithQuery =
  Subject.witness("collisionQuery")(finish(Input.first)).records(Note.completed)
val unbound = Subject.witness(worker.outside()).records(Note.completed)
val emptyName = Subject.witness("")(finish(Input.first)).records(Note.completed)
// A member selector that names no direct state field is refused where it is written, by the
// native selector, so it is a compiler refusal (compiler/CompilerRejects.scala), not a lift one.
//
val repeatedSyncInput = SyncedInput
  .witness(
    SyncedInput.synced(_.left -> finish(Input.first)),
    SyncedInput.synced(_.left -> finish(Input.first))
  )
  .records(_.left -> Subject, Note.completed)
