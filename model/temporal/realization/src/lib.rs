//! Temporal realization facade for the mini activity guest.

extern crate self as umpire_temporal;

pub use umpire_temporal_realization_macros::{answer, observe, propose, realization};

/// One Model action and its optional expected outcome.
pub struct Step<A> {
    /// Action evidenced by the observation.
    pub action: A,
    /// Required outcome, or any enabled outcome when absent.
    pub expected: Option<umpire_dsl::Outcome>,
}

/// A callback's delivery evidence and raw typed outgoing request.
pub struct Reply<A> {
    /// Actions evidenced by callback delivery.
    pub delivery: Vec<A>,
    /// Response action whose enablement the runtime checks.
    pub action: A,
    /// `Any::pack(&typed_request)`; generation adds the Temporal Reply envelope once.
    pub request: umpire_runtime::google::protobuf::Any,
}

/// Declared coverage for one distinct protobuf use.
pub struct Context {
    /// `observed response`, `incoming callback`, or `outgoing request`.
    pub kind: &'static str,
    /// Canonical RPC path when the message has one RPC owner.
    pub method: Option<&'static str>,
    /// Protobuf message identity.
    pub message: &'static str,
    /// Classified protobuf paths in descriptor order.
    pub fields: &'static [Field],
}

/// One declared field classification in a realization context.
pub struct Field {
    /// Raw protobuf field path.
    pub path: &'static str,
    /// `mapped`, `ignored`, or `unsupported`.
    pub class: &'static str,
    /// Explicit reason for an ignored path.
    pub reason: Option<&'static str>,
}

pub mod support;

#[path = "../../foundation/mod.rs"]
pub mod foundation;

#[path = "../../feature/activity/api.rs"]
pub mod activity_api;

mod annotations;

pub mod probes;

#[path = "../../feature/activity/temporal.rs"]
mod temporal;

pub use annotations::{AnnotationRuntime, annotation_runtime};
pub use temporal::{Realization, activity, runtime};
