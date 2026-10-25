//! Temporal realization facade for the mini activity guest.

#![forbid(unsafe_code)]

extern crate self as umpire_temporal;

/// Generated upstream API namespace used by catalog mirrors.
pub use umpire_temporal_api::temporal;

/// Dynamic configuration context for shared catalog derives.
pub type CatalogContext = foundation::dc::Limits;

pub use umpire_temporal_realization_macros::{answer, observe, propose, realization, registry};

/// One Model action and its optional expected outcome.
pub struct Step<A> {
    /// Action evidenced by the observation.
    pub action: A,
    /// Required outcome, or any enabled outcome when absent.
    pub expected: Option<umpire_dsl::Outcome>,
}

/// A callback's delivery evidence and typed inputs for the realization's filler.
pub struct Reply<A, I> {
    /// Actions evidenced by callback delivery.
    pub delivery: Vec<A>,
    /// Response action whose enablement the runtime checks.
    pub action: A,
    /// Explicit inputs passed to the shared typed request constructor.
    pub inputs: I,
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

pub mod registry;

#[path = "../../foundation/mod.rs"]
pub mod foundation;

#[path = "../../feature/activity/api.rs"]
pub mod activity_api;

mod annotations;

pub mod probes;

#[path = "../../feature/activity/temporal.rs"]
mod activity_realization;

pub use activity_realization::{Realization, SCENARIO_HEADER, activity, runtime};
pub use annotations::{AnnotationRuntime, annotation_runtime};
