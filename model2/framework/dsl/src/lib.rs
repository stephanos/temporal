//! State machines and named claims consumed by the Umpire checker.

#![warn(missing_docs)]

mod claim;
mod machine;
mod progress;
mod scope;

pub use claim::{Claim, StepClaim};
pub use machine::Machine;
pub use progress::Progress;
pub use scope::Scope;

/// The verdict for a checked claim, including an incomplete exploration.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Outcome {
    /// The claim holds throughout the explored machine.
    Verified,
    /// A counterexample refutes the claim.
    Refuted,
    /// The exploration ended before the claim could be decided.
    Unknown,
}
