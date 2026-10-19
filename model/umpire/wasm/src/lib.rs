//! Generated Model protocol, transactional dispatch, and bounded core-wasm ABI.
#![cfg(not(test))]

mod abi;
mod refusal;
mod runtime;
mod stock;

/// Generated core contract and shared well-known types.
#[allow(missing_docs, unreachable_pub, clippy::all, clippy::pedantic)]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/protocol.rs"));
}

pub use abi::Guest;
pub use generated::{google, umpire::model::v1 as protocol};
pub use runtime::{AnswerPlan, Projection, Realize, Runtime, Transaction, Transition};
pub use stock::Model;

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
