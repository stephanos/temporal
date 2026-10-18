// Option E (library): statig 0.4 as the authoring surface, Umpire claims bolted on.
// Does not compile; statig has no notion of claims, so the claim half is hypothetical.
//
// statig gives: `#[state_machine]` on an impl, `#[state]` handler functions that `match` the
// event and return `Handled`, `Transition(..)` or `Super`, hierarchical superstates, data on
// `self` or as handler arguments, a macro that only adds code. Pass-through, so rust-analyzer works.
//
// What it costs us: statig's primary notion is "which named state am I in", so `heating_on` and
// `beeping` stop being vars and become handler names; `temperature` stays a field. Claims then
// have to talk about statig's generated `State` enum. And statig executes a machine on events; it
// has no enumeration of enabled actions, no start set and no checker, so the explorer would call
// `handle` once per action per state and read `State` back: workable, but the model is bent to
// fit the library rather than the other way round.

use statig::prelude::*;
use umpire::prelude::*;

const BOILING_TEMPERATURE: i32 = 100;
const COLDEST_TEMPERATURE: i32 = -40;

#[derive(Default)]
pub struct Kettle {
    temperature: i32,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash)]
pub enum Event {
    PressButton,
    DepressButton,
    Heat,
    Failover,
}

#[state_machine(initial = "State::idle()")]
impl Kettle {
    // idle: heating off, not beeping
    #[state]
    fn idle(event: &Event) -> Outcome<State> {
        match event {
            Event::PressButton => Transition(State::heating()),
            _ => Handled, // refused; statig has no refusal, so "handled, no change" stands in
        }
    }

    // heating: heating on, not beeping
    #[state]
    fn heating(&mut self, event: &Event) -> Outcome<State> {
        match event {
            Event::Heat if self.temperature < BOILING_TEMPERATURE => {
                self.temperature += 1;
                Handled
            }
            Event::Failover if self.temperature >= BOILING_TEMPERATURE => Transition(State::beeping()),
            Event::DepressButton => Transition(State::idle()),
            _ => Handled,
        }
    }

    // beeping: heating off, beeping
    #[state]
    fn beeping(event: &Event) -> Outcome<State> {
        match event {
            Event::PressButton => Transition(State::heating()),
            _ => Handled,
        }
    }
}

// Claims, hypothetically, over (Kettle, State). The vars we lost come back as questions about
// which state the machine is in.
impl Claims for Kettle {
    fn starts() -> Vec<(Kettle, State)> {
        (COLDEST_TEMPERATURE..=40).map(|t| (Kettle { temperature: t }, State::idle())).collect()
    }
    fn invariants() -> Vec<Claim<(Kettle, State)>> {
        // beeps only when off: true by construction, since `beeping` is a state with heating off
        vec![]
    }
    fn postconditions() -> Vec<StepClaim<(Kettle, State)>> {
        vec![
            StepClaim::new("warms_only_when_on", |(b, bs), (a, _), _| {
                matches!(bs, State::Heating {}) || a.temperature <= b.temperature
            }),
            StepClaim::after("fails_over_at_boiling", Event::Failover, |_, (a, s)| {
                a.temperature == BOILING_TEMPERATURE && matches!(s, State::Beeping {})
            }),
        ]
    }
    fn progress() -> Vec<Progress<(Kettle, State)>> {
        vec![Progress::new("boils")
            .whenever(|(_, s)| matches!(s, State::Heating {}))
            .eventually(|(_, s)| matches!(s, State::Beeping {}))
            .within(Scope::without(&[Event::DepressButton]))]
    }
}
