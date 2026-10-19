#[cfg(not(feature = "conformance"))]
mod abi;
#[cfg(not(feature = "conformance"))]
mod cache;
#[cfg(any(test, not(feature = "conformance")))]
mod clock;
#[cfg(any(test, not(feature = "conformance")))]
mod cluster;
#[cfg(any(test, feature = "conformance"))]
mod conformance;
#[cfg(all(not(feature = "conformance"), feature = "diagnostic-profile"))]
mod diagnostic;
#[cfg(not(feature = "conformance"))]
mod engine;
#[cfg(all(not(feature = "conformance"), feature = "guest-profile"))]
mod guest_profile;
mod local_model;
#[cfg(not(feature = "conformance"))]
mod metrics;
#[cfg(not(feature = "conformance"))]
mod model;
mod namespace;
#[cfg(any(test, not(feature = "conformance")))]
mod network;
#[cfg(not(feature = "conformance"))]
mod protocol;
#[cfg(not(feature = "conformance"))]
mod storage_transport;
mod transcript;
mod volume;

#[cfg(any(test, not(feature = "conformance")))]
mod world;

#[cfg(feature = "conformance")]
fn main() {
    if let Err(error) = conformance::execute() {
        eprintln!("wasmhost conformance transport: {error}");
        std::process::exit(1);
    }
}

#[cfg(not(feature = "conformance"))]
fn main() {
    #[cfg(feature = "diagnostic-profile")]
    let session = diagnostic::Session::start();
    let result = engine::execute();
    #[cfg(feature = "diagnostic-profile")]
    drop(session);
    if let Err(error) = result {
        eprintln!("wasmhost transport: {error}");
        std::process::exit(1);
    }
}
