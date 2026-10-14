mod abi;
mod cache;
#[cfg(feature = "diagnostic-profile")]
mod diagnostic;
mod engine;
#[cfg(feature = "guest-profile")]
mod guest_profile;
mod metrics;
mod model;
mod protocol;
mod transcript;

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
