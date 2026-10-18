//! Breadth-first exploration, safety claims and fair progress over finite machines.

mod explore;
mod progress;
mod refusal;
mod report;
mod visited;

pub use explore::{Checker, Counts, Result, Transition};
pub use refusal::Refusal;
pub use report::{ClaimResult, Path, PathStep, Report};
pub use visited::{FullStates, StateId, Visited};
