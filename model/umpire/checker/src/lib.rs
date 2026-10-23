//! Breadth-first exploration, safety claims and fair progress over finite machines.

mod explore;
#[cfg(not(target_arch = "wasm32"))]
mod interpret;
mod progress;
mod refusal;
mod report;
mod visited;

pub use explore::{Checker, Counts, Result, Transition};
#[cfg(not(target_arch = "wasm32"))]
pub use interpret::Interpreted;
pub use refusal::Refusal;
pub use report::{ClaimResult, Path, PathStep, ProgressVerdict, Report};
pub use visited::{FullStates, StateId, Visited};
