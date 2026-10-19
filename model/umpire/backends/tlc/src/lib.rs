//! Pinned TLC execution, complete reachable sets and typed counterexamples.

mod decode;
mod module;
mod runner;

pub use decode::*;
pub use runner::*;

/// A backend admission, execution or decoding failure; never a property verdict.
#[derive(Debug, thiserror::Error)]
#[error("TLC refusal: {0}")]
pub struct Refusal(pub String);

impl From<std::io::Error> for Refusal {
    fn from(error: std::io::Error) -> Self {
        Self(error.to_string())
    }
}
