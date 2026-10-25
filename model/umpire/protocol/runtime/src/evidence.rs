//! Typed internal views of actual system evidence.

use crate::{
    AnyMessage as _, Observed, prost_types::Any, protocol::Refusal, realization::Observation,
};
use prost::Name;

/// Evidence from one actual RPC invocation.
pub struct Rpc;

/// A successful read of an explicitly typed observable value.
pub struct Snapshot;

/// Ordered acknowledgment and current-read evidence.
pub struct Reconciled;

/// A declared evidence kind without a producer or transport admission.
pub enum History {}

/// One actual invocation, retaining its original envelope and shared decoder.
pub struct RpcView {
    raw: Observation,
    observed: Observed,
}

impl RpcView {
    /// Admit the invocation envelope, leaving RPC-specific decoding to its owner.
    pub fn new(value: &Any) -> Result<Self, Refusal> {
        let observed = Observed::new(value)?;
        Ok(Self { raw: value.unpack()?, observed })
    }

    /// The unchanged request and result from this invocation.
    pub fn raw(&self) -> &Observation {
        &self.raw
    }
}

impl std::ops::Deref for RpcView {
    type Target = Observed;

    fn deref(&self) -> &Self::Target {
        &self.observed
    }
}

/// A typed successful read, alongside its same-invocation request and result.
pub struct SnapshotView<T> {
    rpc: RpcView,
    value: T,
}

impl<T: Name + Default> SnapshotView<T> {
    /// Admit the RPC envelope and decode its actual successful response.
    pub fn new(value: &Any) -> Result<Self, Refusal> {
        let rpc = RpcView::new(value)?;
        let value = rpc.response()?;
        Ok(Self { rpc, value })
    }
}

impl<T> SnapshotView<T> {
    /// The actual invocation that produced this read.
    pub fn rpc(&self) -> &RpcView {
        &self.rpc
    }

    /// The typed actual response, without inferred Model state.
    pub fn value(&self) -> &T {
        &self.value
    }
}

/// Two actual invocations in declared acknowledgment-then-read order.
pub struct ReconciledView {
    acknowledged: RpcView,
    current: RpcView,
}

impl ReconciledView {
    /// Admit both envelopes in order; RPC-specific correlation remains with their owner.
    pub fn new(acknowledged: &Any, current: &Any) -> Result<Self, Refusal> {
        let acknowledged = RpcView::new(acknowledged)?;
        let current = RpcView::new(current)?;
        Ok(Self { acknowledged, current })
    }

    /// The earlier actual acknowledgment.
    pub fn acknowledged(&self) -> &RpcView {
        &self.acknowledged
    }

    /// The current actual read.
    pub fn current(&self) -> &RpcView {
        &self.current
    }
}
