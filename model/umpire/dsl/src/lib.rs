//! State machines and named claims consumed by the Umpire checker.

#![warn(missing_docs)]

mod action;
mod catalog;
pub mod claim;
pub mod closable;
mod finite;
mod machine;
pub mod outcome;
mod path;
#[doc(hidden)]
pub mod path_build;
mod path_input;
mod phased;
mod progress;
mod scope;

pub use action::{Actor, ActorProjection, ActorValue, RegisteredActor};
pub use catalog::{ActionCatalog, CATALOG_LIMIT, CatalogError, CatalogVariant};
pub use claim::{Claim, Hole, StepCheckPredicate, StepClaim, StepPredicate, Verdict};
pub use finite::Finite;
pub use machine::{Associated, Machine, Registered, Step, StepError, Steps};
pub use outcome::{Outcome, OutcomeClass, Rejection};
#[doc(hidden)]
pub use path::kind as waypoint_kind;
pub use path::{
    ActionDomain, ActionPredicate, Alternative, Gap, InputDomain, InputRange, IntoWaypoint,
    OutcomePredicate, Path, PathNode, Pattern, PatternError, Waypoint,
};
pub use path_input::IntoWorldPath;
#[doc(hidden)]
pub use path_input::kind as world_path_kind;
pub use phased::{Phase, Phased};
pub use progress::{Progress, ProgressEvaluation, ProgressEvaluator};
pub use scope::{Assumption, AssumptionExt, Scope};
#[doc(hidden)]
pub use umpire_dsl_macros::__path_parse;
pub use umpire_dsl_macros::{Action, Finite, State, World, machine};

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
