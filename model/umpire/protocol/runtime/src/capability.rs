//! Typed realization capabilities and explicit refusing adapters.

use crate::{
    AnswerPlan, EvidenceMatch, Projection, Proposal, ReachTarget, Scope, prost_types::Any,
    protocol::Refusal,
};
use umpire_dsl::Machine;

/// Admission of the actual identity used when filling a request.
pub trait Identity {
    /// System-owned admitted identity.
    type Id;
    /// Validate the actual identity without changing admission state.
    fn identify(&self, value: &Any) -> Result<Self::Id, Refusal>;
}

/// Packing at the system boundary after typed request construction.
pub trait Request {
    /// Pack this request using its declared system envelope.
    fn pack_request(&self) -> Any;
}
impl<T: prost::Name> Request for T {
    fn pack_request(&self) -> Any {
        <Any as crate::AnyMessage>::pack(self)
    }
}

/// Explicit absence of an outgoing action request.
pub enum NoRequest {}
impl Request for NoRequest {
    fn pack_request(&self) -> Any {
        match *self {}
    }
}

/// One typed request constructor shared by every action consumer.
pub trait Fill<A>: Identity {
    /// The independently declared World.
    type World: Machine<Action = A>;
    /// Explicit inputs selected by the caller's policy.
    type Inputs;
    /// Typed system request, or a typed sum of requests.
    type Request: Request;
    /// Construct the request from declared data and an admitted actual identity.
    fn fill(
        &self,
        _action: &A,
        _world: &Self::World,
        _inputs: &Self::Inputs,
        _identity: &Self::Id,
    ) -> Result<Self::Request, Refusal> {
        Err(Refusal::unsupported("filler unsupported"))
    }
}

/// Projection of admitted observations onto checked full Model states.
pub trait Snapshot<M: Machine> {
    /// Pure compatibility target over full states retained by the bounded frontier.
    type Target: ReachTarget<M::State>;
    /// Validate and project the actual observation without changing run state.
    fn project(
        &self,
        world: &M,
        observation: &Any,
        state: &M::State,
    ) -> Result<Projection<M::Action, M::Outcome, Self::Target>, Refusal>;
}

/// Explicit absence of supported positive evidence.
pub enum NoEvidence {}

/// Admission and matching for one declared internal evidence kind.
pub trait Evidence<K> {
    /// The checked Model whose complete edges are matched.
    type World: Machine;
    /// Admitted typed evidence view.
    type View;
    /// Admit the existing transport before interpreting feature evidence.
    /// Refusals propagate unchanged; an admitted view without a matching predicate rejects.
    fn admit_evidence(&self, _observation: &Any) -> Result<Self::View, Refusal> {
        Err(Refusal::unsupported("evidence kind unsupported"))
    }
    /// Match complete checked edges, retaining caller-owned evidence cursors.
    fn check_evidence(
        &self,
        _view: &Self::View,
        _path: &[&crate::Evidence<Self::World>],
    ) -> EvidenceMatch {
        EvidenceMatch::Reject
    }
}

/// Explicit optional policies around projection and typed request filling.
pub trait ReplyPolicy<M: Machine> {
    /// Finite search and whole-prefix retention metadata.
    fn scope(&self) -> Scope {
        Scope::default()
    }

    /// Whether this realization admits stateful operations.
    fn enabled(&self) -> bool {
        true
    }

    /// Whether a fresh run requires admission before callbacks and action proposals.
    fn needs_bootstrap(&self) -> bool {
        false
    }

    /// Issues the worker program for this realization and its bound World.
    fn program(&self) -> Result<Any, Refusal> {
        Err(Refusal::unsupported("program unsupported"))
    }

    /// Selects a deterministic external request for the current state and seed.
    fn propose(
        &self,
        _world: &M,
        _state: &M::State,
        _seed: u64,
    ) -> Result<Proposal<M::Action>, Refusal> {
        Err(Refusal::unsupported("proposal unsupported"))
    }

    /// Supplies admission independently of the action proposal policy.
    fn bootstrap(
        &self,
        world: &M,
        state: &M::State,
        seed: u64,
    ) -> Result<Proposal<M::Action>, Refusal> {
        self.propose(world, state, seed)
    }

    /// Verifies that an observation belongs to the pending request.
    fn matches(&self, _request: &Any, _observation: &Any) -> Result<(), Refusal> {
        Err(Refusal::unsupported("proposal correlation unsupported"))
    }

    /// Requests another observation after a positively explained intermediate read.
    fn continue_polling(
        &self,
        _observation: &Any,
        _state: &M::State,
        _action: Option<&M::Action>,
    ) -> Result<bool, Refusal> {
        Ok(false)
    }

    /// Validates the complete callback and selects its delivery path and response.
    fn answer(
        &self,
        _world: &M,
        _callback: &Any,
        _state: &M::State,
    ) -> Result<AnswerPlan<M::Action>, Refusal> {
        Err(Refusal::unsupported("answer unsupported"))
    }
}
