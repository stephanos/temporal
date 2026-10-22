pub use wasmtime::wasmparser;

#[cfg(not(feature = "conformance"))]
mod abi;
#[cfg(all(
    feature = "stress-jitter",
    any(feature = "diagnostic-profile", feature = "conformance")
))]
compile_error!("stress-jitter cannot be combined with profiling or conformance features");
#[cfg(all(
    feature = "determinism-fixture",
    any(
        feature = "diagnostic-profile",
        feature = "conformance",
        feature = "stress-jitter"
    )
))]
compile_error!(
    "determinism-fixture cannot be combined with profiling, conformance or stress features"
);
#[cfg(not(feature = "conformance"))]
mod cache;
#[cfg(any(test, not(feature = "conformance")))]
mod clock;
#[cfg(any(test, not(feature = "conformance")))]
mod cluster;
#[cfg(any(test, feature = "conformance"))]
mod conformance;
#[cfg(feature = "determinism-fixture")]
mod determinism_fixture;
#[cfg(all(not(feature = "conformance"), feature = "diagnostic-profile"))]
mod diagnostic;
#[cfg(not(feature = "conformance"))]
mod engine;
#[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
mod event_order;
#[cfg(all(not(feature = "conformance"), feature = "guest-profile"))]
mod guest_profile;
#[cfg(not(feature = "conformance"))]
pub mod linked;
#[cfg(not(feature = "conformance"))]
mod linked_cancellation;
#[cfg(not(feature = "conformance"))]
pub mod linked_cluster;
#[cfg(not(feature = "conformance"))]
mod linked_conditions;
#[cfg(feature = "determinism-fixture")]
mod linked_fixture;
#[cfg(not(feature = "conformance"))]
pub mod linked_inspection;
#[cfg(any(test, not(feature = "conformance")))]
mod linked_network;
#[cfg(any(test, not(feature = "conformance")))]
pub mod linked_simulation;
#[cfg(not(feature = "conformance"))]
mod linked_storage;
mod local_model;
#[cfg(not(feature = "conformance"))]
mod metrics;
#[cfg(not(feature = "conformance"))]
mod model;
#[cfg(not(feature = "conformance"))]
mod module_metadata;
mod namespace;
#[cfg(any(test, not(feature = "conformance")))]
mod network;
#[cfg(not(feature = "conformance"))]
mod protocol;
#[cfg(not(feature = "conformance"))]
mod storage_transport;
#[cfg(feature = "stress-jitter")]
mod stress_jitter;
mod transcript;
mod volume;

#[cfg(any(test, not(feature = "conformance")))]
mod world;

#[cfg(feature = "conformance")]
/// # Errors
/// Returns a conformance transport failure.
pub fn transport() -> std::io::Result<()> {
    conformance::execute()
}

#[cfg(not(feature = "conformance"))]
/// # Errors
/// Returns the helper's existing framed transport failure.
pub fn transport() -> wasmtime::Result<()> {
    #[cfg(feature = "diagnostic-profile")]
    let session = diagnostic::Session::start();
    let result = engine::execute();
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    event_order::finish();
    #[cfg(feature = "diagnostic-profile")]
    drop(session);
    result
}

#[cfg(all(test, not(feature = "conformance")))]
mod clock_conformance;
#[cfg(all(test, not(feature = "conformance")))]
mod network_corpus;
#[cfg(all(test, not(feature = "conformance")))]
mod world_conformance;
