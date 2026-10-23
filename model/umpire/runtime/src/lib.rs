//! Generated Model protocol, transactional dispatch, and bounded core-wasm ABI.
#![cfg(not(test))]

mod abi;
#[path = "annotations.rs"]
mod annotation_dispatch;
mod any;
#[path = "realization.rs"]
mod envelopes;
mod refusal;
mod runtime;
mod stock;

/// Generated core contract and shared well-known types.
#[allow(missing_docs, unreachable_pub, clippy::all, clippy::pedantic)]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/protocol.rs"));
}

pub use abi::Guest;
pub use annotation_dispatch::{AnnotationRuntime, probe_recipe};
pub use envelopes::{Observed, callback, reply};
pub use generated::{
    google,
    umpire::{
        annotations::v1 as annotations, model::v1 as protocol, realization::v1 as realization,
    },
};
pub use runtime::{
    AnswerPlan, Candidate, Projection, Proposal, Realize, Runtime, Transaction, Transition,
};
pub use stock::Model;

impl protocol::Refusal {
    /// Construct a refusal with a stable protocol code.
    pub fn new(code: protocol::RefusalCode, message: impl Into<String>) -> Self {
        Self { code: code as i32, message: message.into() }
    }

    /// Refuse malformed input.
    pub fn malformed(message: impl Into<String>) -> Self {
        Self::new(protocol::RefusalCode::MalformedRequest, message)
    }

    /// Refuse a well-formed but unsupported input.
    pub fn unsupported(message: impl Into<String>) -> Self {
        Self::new(protocol::RefusalCode::Unsupported, message)
    }
}

/// A dispatcher whose effects remain private until its response is published.
pub trait Dispatch {
    /// Owned prospective changes, without imposing Clone on Model outcomes or facts.
    type Transaction;

    /// Computes a response and its optional effects without changing the current run.
    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>);

    /// Installs already validated effects after successful response publication.
    fn commit(&mut self, transaction: Self::Transaction);
}
