#[cfg(not(feature = "conformance"))]
mod abi;
#[cfg(not(feature = "conformance"))]
mod cache;
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
mod namespace_call;
mod namespace_image;
#[cfg(not(feature = "conformance"))]
mod namespace_transport;
#[allow(dead_code, reason = "network owner is consumed by fn178")]
mod network;
#[cfg(not(feature = "conformance"))]
mod protocol;
mod transcript;

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
