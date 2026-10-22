//! Cluster execution owns instance handoff, broker scheduling and shared networking.
#[cfg(not(feature = "conformance"))]
pub(crate) mod execution;
#[cfg(not(feature = "conformance"))]
pub(crate) mod handoff;
pub(crate) mod network;

#[cfg(not(feature = "conformance"))]
pub(crate) use execution::{execute, requested, transport};
