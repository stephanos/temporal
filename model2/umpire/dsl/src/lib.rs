//! State machines and named claims consumed by the Umpire checker.

#![warn(missing_docs)]

mod claim;
mod machine;
mod progress;
mod scope;
mod spell;

pub use claim::{Claim, Hole, StepClaim, StepPredicate, Verdict};
pub use machine::{Machine, Step, Steps};
pub use progress::Progress;
pub use scope::Scope;
pub use spell::Spell;

/// The checker's aggregate claim result, distinct from a domain's step outcome.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Outcome {
    /// The claim holds throughout the explored machine.
    Verified,
    /// A counterexample refutes the claim.
    Refuted,
    /// The exploration ended before the claim could be decided.
    Unknown,
}
