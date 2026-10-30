//! Independent partial-observation countdown fixture shared by native and guest controls.
#![forbid(unsafe_code)]

use umpire_dsl::{Machine, Steps};
use umpire_runtime::{
    Evidence, EvidenceMatch, Projection, ReachTarget, Realize, Runtime, Scope, prost_types::Any,
    protocol,
};

pub(crate) const RENEW: u8 = 50;
pub(crate) const GLOBAL: u8 = 51;
pub(crate) const LOCAL: u8 = 52;
pub(crate) const GLOBAL_POLL: u8 = 53;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Phase {
    Active,
    GlobalClosed,
    PendingClosed,
}
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub(crate) struct State {
    pub(crate) phase: Phase,
    pub(crate) global: u8,
    pub(crate) local: u8,
    pub(crate) pending: bool,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub(crate) enum Action {
    Renew,
    Tick,
}
#[derive(Debug, PartialEq)]
pub(crate) enum Outcome {
    Renewed,
    Advanced,
    GlobalClosed,
    PendingClosed,
}
#[derive(Debug, PartialEq)]
pub(crate) enum Emission {
    Renewed,
    FactGlobal,
    FactLocal,
}
impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}
impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}
pub(crate) struct World;
impl Machine for World {
    type State = State;
    type Action = Action;
    type Outcome = Outcome;
    type Fact = Emission;
    fn starts(&self) -> Vec<State> {
        vec![State { phase: Phase::Active, global: 3, local: 2, pending: true }]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Renew, Action::Tick]
    }
    fn internal(&self, action: &Action) -> bool {
        *action == Action::Tick
    }
    fn step(&self, state: &State, action: Action) -> Steps<State, Outcome, Emission> {
        if state.phase != Phase::Active {
            return Steps::new();
        }
        let mut next = state.clone();
        let (outcome, facts) = match action {
            Action::Renew => {
                next.local = 2;
                (Outcome::Renewed, vec![Emission::Renewed])
            }
            Action::Tick => {
                next.global -= 1;
                next.local -= 1;
                if next.global == 0 {
                    next.phase = Phase::GlobalClosed;
                    (Outcome::GlobalClosed, vec![Emission::FactGlobal])
                } else if next.local == 0 && next.pending {
                    next.phase = Phase::PendingClosed;
                    (Outcome::PendingClosed, vec![Emission::FactLocal])
                } else {
                    (Outcome::Advanced, Vec::new())
                }
            }
        };
        [(outcome, facts, next)].into_iter().collect()
    }
}
pub(crate) struct Target(Phase);
impl ReachTarget<State> for Target {
    fn matches(&self, candidate: &State) -> bool {
        candidate.phase == self.0
    }
}
#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub(crate) enum Fault {
    #[default]
    None,
    RejectRenewFact,
    DivergentRead,
    DivergentPoll,
    DivergentAnswer,
}
pub(crate) struct Binding {
    pub(crate) scope: Scope,
    pub(crate) fault: Fault,
}
impl Default for Binding {
    fn default() -> Self {
        Self { scope: Scope { depth: 4, work: 4096, prefix: 128 }, fault: Fault::None }
    }
}
impl umpire_runtime::capability::Identity for Binding {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}
impl umpire_runtime::capability::Fill<Action> for Binding {
    type World = World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}
impl umpire_runtime::capability::Snapshot<World> for Binding {
    type Target = Target;
    fn project(
        &self,
        _: &World,
        actual: &Any,
        _: &State,
    ) -> Result<Projection<Action, Outcome, Target>, protocol::Refusal> {
        let phase = match super::payload(actual)? {
            RENEW => Phase::Active,
            GLOBAL | GLOBAL_POLL => Phase::GlobalClosed,
            LOCAL => Phase::PendingClosed,
            _ => return Err(protocol::Refusal::malformed("unknown countdown observation")),
        };
        Ok(Projection::Reach(Target(phase)))
    }
}
impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Binding {
    type World = World;
    type View = u8;
    fn admit_evidence(&self, actual: &Any) -> Result<u8, protocol::Refusal> {
        super::payload(actual)
    }
    fn check_evidence(&self, actual: &u8, path: &[&Evidence<World>]) -> EvidenceMatch {
        let actions: Vec<_> = path.iter().map(|edge| edge.action).collect();
        let permitted: &[&[Action]] = match *actual {
            RENEW => &[&[Action::Renew], &[Action::Tick, Action::Renew]],
            GLOBAL | GLOBAL_POLL | LOCAL => &[&[Action::Tick, Action::Tick]],
            _ => return EvidenceMatch::Reject,
        };
        if !permitted.iter().any(|expected| expected.starts_with(&actions)) {
            return EvidenceMatch::Reject;
        }
        for edge in path {
            let valid = match edge.outcome {
                Outcome::Renewed => {
                    self.fault != Fault::RejectRenewFact && edge.facts == [Emission::Renewed]
                }
                Outcome::Advanced => edge.facts.is_empty(),
                Outcome::GlobalClosed => edge.facts == [Emission::FactGlobal],
                Outcome::PendingClosed => edge.facts == [Emission::FactLocal],
            };
            if !valid {
                return EvidenceMatch::Reject;
            }
        }
        if !permitted.contains(&actions.as_slice()) {
            return EvidenceMatch::Continue { cursor: path.len() };
        }
        let valid = match (*actual, path.last()) {
            (RENEW, Some(edge)) => edge.outcome == Outcome::Renewed,
            (GLOBAL | GLOBAL_POLL, Some(edge)) => {
                edge.outcome == Outcome::GlobalClosed && edge.facts == [Emission::FactGlobal]
            }
            (LOCAL, Some(edge)) => {
                edge.outcome == Outcome::PendingClosed && edge.facts == [Emission::FactLocal]
            }
            _ => false,
        };
        if valid { EvidenceMatch::Match { cursor: path.len() } } else { EvidenceMatch::Reject }
    }
}
impl umpire_runtime::capability::ReplyPolicy<World> for Binding {
    fn scope(&self) -> Scope {
        self.scope
    }
    fn propose(
        &self,
        _: &World,
        state: &State,
        _: u64,
    ) -> Result<umpire_runtime::Proposal<Action>, protocol::Refusal> {
        if state.phase != Phase::Active {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::Quiescent,
                "countdown closed",
            ));
        }
        Ok(umpire_runtime::Proposal::Read {
            request: super::any(if self.fault == Fault::DivergentRead && state.global == 3 {
                LOCAL
            } else {
                GLOBAL
            }),
        })
    }
    fn matches(&self, _: &Any, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
    fn answer(
        &self,
        _: &World,
        _: &Any,
        state: &State,
    ) -> Result<umpire_runtime::AnswerPlan<Action>, protocol::Refusal> {
        if self.fault != Fault::DivergentAnswer {
            return Err(protocol::Refusal::unsupported("countdown has no callback"));
        }
        Ok(umpire_runtime::AnswerPlan {
            delivery: Vec::new(),
            response_action: Action::Renew,
            reply: super::any(if state.global == 3 { GLOBAL } else { LOCAL }),
        })
    }
    fn continue_polling(
        &self,
        actual: &Any,
        state: &State,
        _: Option<&Action>,
    ) -> Result<bool, protocol::Refusal> {
        Ok(matches!(super::payload(actual)?, RENEW | GLOBAL_POLL)
            && (self.fault != Fault::DivergentPoll || state.global == 3))
    }
}
impl Realize<World> for Binding {}
pub(crate) fn runtime() -> Runtime<World, Binding> {
    Runtime::new(World, Binding::default())
}
