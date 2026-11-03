//! Typed entity and read declarations outside the Model action graph.

use crate::registry::Rpc;
use umpire_runtime::{capability::Identity, protocol::Refusal};

/// Typed bootstrap admission and immutable receipt/correlation hooks.
pub struct Entity<B: Identity, R, Q, S> {
    /// Bootstrap RPC outside the Model transition graph.
    pub bootstrap: Rpc<Q, S>,
    /// Admit the actual namespace/entity/run identity from one invocation.
    pub admit: fn(&B, &Q, &S) -> Result<B::Id, Refusal>,
    /// Construct a receipt from the admitted identity and actual bootstrap invocation.
    pub receipt: fn(&B, &B::Id, &Q, &S) -> Result<R, Refusal>,
    /// Check the receipt against the actual request and response.
    pub correlate: fn(&B, &R, &Q, &S) -> Result<(), Refusal>,
}

/// Pure read construction, projection and continuation hooks outside the action graph.
pub struct Read<B, W, R, Q, S, T> {
    /// Descriptor-admitted read method.
    pub rpc: Rpc<Q, S>,
    /// Construct from the current World and actual receipt.
    pub fill: fn(&B, &W, &R) -> Result<Q, Refusal>,
    /// Correlate the actual read request and response with their receipt.
    pub correlate: fn(&B, &R, &Q, &S) -> Result<(), Refusal>,
    /// Project only the actual response's observable data.
    pub project: fn(&B, &S) -> Result<T, Refusal>,
    /// Decide finality from the typed observable projection.
    pub finality: fn(&B, &T) -> bool,
    /// Select continuation from that same projection.
    pub continue_polling: fn(&B, &T) -> bool,
}
