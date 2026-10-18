// Option F (library): smlang 0.8 as the authoring surface.
// Does not compile; smlang has no claims or checker.
//
// smlang gives one `statemachine!` block with a transition table:
//   *Start + Event [ guard ] / action = Next
// Guards and actions are functions on a context trait; states may carry data.
//
// What it costs us: the same as statig but more so. Control becomes named states, `temperature`
// becomes state data threaded through every row, and every guard and effect is a function away
// from the table, so the table reads well while the behaviour lives elsewhere. There is no
// refusal: a missing row is an error from `process_event`. Enumerating enabled actions per state
// is possible from the generated enums. Claims would be written against the generated
// `KettleStates`. Best fit for protocol-shaped machines; a poor fit for var-shaped Models.

use smlang::statemachine;

const BOILING_TEMPERATURE: i32 = 100;

statemachine! {
    name: Kettle,
    derive_states: [Clone, PartialEq, Eq, Hash],
    derive_events: [Clone, Copy, PartialEq, Eq, Hash],
    transitions: {
        *Idle(i32)   + PressButton                      / keep  = Heating(i32),
        Heating(i32) + Heat          [ below_boiling ]  / warm  = Heating(i32),
        Heating(i32) + Failover      [ at_boiling ]     / keep  = Beeping(i32),
        Heating(i32) + DepressButton                    / keep  = Idle(i32),
        Beeping(i32) + PressButton                      / keep  = Heating(i32),
    }
}

pub struct Ctx;

impl KettleStateMachineContext for Ctx {
    fn below_boiling(&self, t: &i32) -> Result<bool, ()> {
        Ok(*t < BOILING_TEMPERATURE)
    }
    fn at_boiling(&self, t: &i32) -> Result<bool, ()> {
        Ok(*t >= BOILING_TEMPERATURE)
    }
    fn warm(&mut self, t: &i32) -> i32 {
        t + 1
    }
    fn keep(&mut self, t: &i32) -> i32 {
        *t
    }
}

// Claims, hypothetically, over `KettleStates`:
//   invariant  beeps_only_when_off: true by construction (Beeping is a state with heating off)
//   postcond.  warms_only_when_on:  |before, after| !matches!(before, Heating(_)) || temp(after) <= temp(before)
//   postcond.  fails_over_at_boiling, after Failover: matches!(after, Beeping(t) if t == 100)
//   progress   boils: whenever Heating(_) eventually Beeping(_), without DepressButton
//
// Starts: (-40..=40).map(KettleStates::Idle)
