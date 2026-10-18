//! Breadth-first exploration and safety claims over finite machines.

mod explore;
mod refusal;
mod report;
mod visited;

pub use explore::{Checker, Counts, Result, Transition};
pub use refusal::Refusal;
pub use report::{ClaimResult, Path, PathStep, Report};
pub use visited::{FullStates, StateId, Visited};
