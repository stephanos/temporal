//! State machines and named claims consumed by the Umpire checker.

#![warn(missing_docs)]

pub mod claim;
pub mod closable;
mod machine;
pub mod outcome;
mod path;
mod phased;
mod progress;
mod scope;

pub use claim::{Claim, Hole, StepCheckPredicate, StepClaim, StepPredicate, Verdict};
pub use machine::{Machine, Registered, Step, StepError, Steps};
pub use outcome::{Outcome, Rejection};
#[doc(hidden)]
pub use path::kind as waypoint_kind;
pub use path::{Gap, IntoWaypoint, Path, Waypoint};
pub use phased::{Phase, Phased};
pub use progress::{Progress, ProgressEvaluation, ProgressEvaluator};
pub use scope::{Assumption, AssumptionExt, Scope};
pub use umpire_dsl_macros::{Action, State, machine};

/// The checker's aggregate claim result, distinct from a domain's step outcome.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CheckOutcome {
    /// The claim holds throughout the explored machine.
    Verified,
    /// A counterexample refutes the claim.
    Refuted,
    /// The exploration ended before the claim could be decided.
    Unknown,
}
