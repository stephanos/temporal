package framework.dsl

import ReliableBroadcastControls.Naive
import samples.ReliableBroadcast

class ReliableBroadcastExpectTest extends munit.FunSuite:
  test("ReliableBroadcast's four native predicates retain both process implications"):
    import ReliableBroadcast.*
    // the Broadcast record a state holds
    def carried(state: State): Broadcast = Model.at(state)(Model.current)
    val initial = Model.init
    val delivered = initial.copy(
      origin = Origin.crashed(Payload.hello),
      p1 = Local.delivered(Payload.hello),
      p2 = Local.delivered(Payload.hello)
    )
    val wrong = delivered.copy(p2 = Local.delivered(Payload.goodbye))
    val safetyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, delivered), true),
      (Step(Outcome.accepted, wrong), false)
    )
    SectionClaims.same(Model, "invariants", "safety", None, safetyCases)(after =>
      onlyBroadcast(carried(after.state))
    )
    SectionClaims.same(Naive, "invariants", "naiveSafety", None, safetyCases)(after =>
      onlyBroadcast(carried(after.state))
    )
    val forwarded =
      initial.copy(p1 = Local.relayed(Payload.hello), p2 = Local.relayed(Payload.hello))
    SectionClaims.across(
      Model,
      "reliability",
      List(
        (initial, Step(Outcome.accepted, initial), true),
        (forwarded, Step(Outcome.accepted, delivered), true),
        (delivered, Step(Outcome.accepted, delivered), true),
        (initial, Step(Outcome.accepted, delivered), false),
        (forwarded.copy(p1 = Local.idle), Step(Outcome.accepted, delivered), false),
        (forwarded.copy(p2 = Local.idle), Step(Outcome.accepted, delivered), false)
      )
    )((before, after) =>
      (deliveredValue(carried(after.state).p1) == None || deliveredValue(
        carried(before).p1
      ) != None || relayed(
        carried(before).p1
      )) &&
        (deliveredValue(carried(after.state).p2) == None || deliveredValue(
          carried(before).p2
        ) != None || relayed(
          carried(before).p2
        ))
    )
    SectionClaims.same(
      Model,
      "queries",
      "survivesCrash",
      Some(process.deliver(Proc.p2)),
      List(
        (Step(Outcome.accepted, delivered), true),
        (Step(Outcome.accepted, delivered.copy(p2 = Local.idle)), false),
        (Step(Outcome.accepted, delivered.copy(origin = Origin.done(Payload.hello))), false),
        (Step(Outcome.accepted, wrong), false)
      )
    )(after =>
      everyoneDelivered(carried(after.state)) &&
        carried(after.state).origin == Origin.crashed(Payload.hello)
    )
