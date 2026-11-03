//! Typed action rows and immutable entity/read configuration admitted by `registry!`.

use std::marker::PhantomData;
use umpire_runtime::{
    Evidence, EvidenceMatch, capability::Fill, prost_types::Any, protocol::Refusal,
};

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
