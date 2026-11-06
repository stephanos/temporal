//! Typed action rows and immutable entity/read configuration admitted by `registry!`.

use std::marker::PhantomData;
use umpire_dsl::Machine;
use umpire_runtime::{
    Evidence, EvidenceMatch, ReachTarget, capability::Fill, prost_types::Any, protocol::Refusal,
};

/// Checked relation evaluation and protocol refusal conversion.
pub mod relations;

/// The caller responsible for an RPC action.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Direction {
    /// An action proposed by a client.
    Client,
    /// An action selected for a worker row.
    Worker,
}

/// The declared external meaning of one actual Model variant.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ActionKind {
    /// One descriptor-admitted RPC and its caller.
    Rpc {
        /// Canonical descriptor service/method path.
        method: &'static str,
        /// Declared caller.
        direction: Direction,
    },
    /// An explicit abstract event with its own evidence predicate.
    Event {
        /// Authored event meaning.
        reason: &'static str,
    },
}

/// Classification shared by proposal, worker and probe consumers.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ActionRow {
    /// Actual actor enum name.
    pub actor: &'static str,
    /// Actual variant name, independent of payload values.
    pub variant: &'static str,
    /// Descriptor RPC or authored environmental meaning.
    pub kind: ActionKind,
}

/// One complete action inventory, sharing the existing typed filler capability.
pub trait Registry<A>: Fill<A> {
    /// Classify an actual action without constructing or caching a request.
    fn row(&self, action: &A) -> ActionRow;

    /// Decode actual RPC evidence and match the complete checked witness prefix.
    /// Empty prefixes ask the authored predicates for root admission. Positive
    /// admissions must agree on their kind and cursor before Reach can explore.
    ///
    /// # Errors
    /// Preserves envelope, concrete request/result and feature admission refusals.
    fn check_registered(
        &self,
        observation: &Any,
        path: &[&Evidence<Self::World>],
    ) -> Result<EvidenceMatch, Refusal>;
}

/// A descriptor method whose input/output identities are checked during expansion.
pub struct Rpc<Q, S> {
    /// Canonical descriptor service/method path.
    pub method: &'static str,
    /// Concrete generated request and response identity witnesses.
    pub types: PhantomData<fn(Q) -> S>,
}

pub use crate::lens::config::{Entity, Read};

/// A feature's declared refinement of an independently authored protocol Model: a full-state map
/// that keeps every latent field, exact action, outcome and fact maps, and the online
/// observable target of a full protocol state. Offline checks own the comparison; the
/// declaration constructs no state from a snapshot.
pub struct Refines<P: Machine, F: Machine, T> {
    /// Total map of reachable full protocol states; `None` marks an unrepresentable state.
    pub state: fn(&P::State) -> Option<F::State>,
    /// The feature action each protocol action must reproduce.
    pub action: fn(&P::Action) -> F::Action,
    /// The feature outcome of each emitted protocol outcome.
    pub outcome: fn(&P::Outcome) -> F::Outcome,
    /// The feature fact of each emitted protocol fact.
    pub fact: fn(&P::Fact) -> F::Fact,
    /// The read's observable target of a full protocol state's public view.
    pub view: fn(&P::State) -> T,
}

/// A binding whose registry declares exactly one protocol refinement, checked against its read
/// target.
pub trait Refined {
    /// The independently authored protocol Model.
    type Protocol: Machine;
    /// The feature Model the binding realizes.
    type Feature: Machine;
    /// The read's observable target.
    type Target: ReachTarget<<Self::Feature as Machine>::State>;
    /// The admitted declaration.
    fn refines() -> Refines<Self::Protocol, Self::Feature, Self::Target>;
}
