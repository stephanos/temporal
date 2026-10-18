use std::{fmt::Debug, hash::Hash};

use crate::{Claim, Progress, StepClaim};

/// A finite state machine with explicit actions and named claims.
pub trait Machine: Clone + Eq + Hash {
    /// The finite action catalog's value type.
    type Action: Copy + Eq + Hash + Debug + 'static;

    /// Enumerates the machine's initial states.
    fn starts() -> Vec<Self>;

    /// Enumerates every action, including actions refused in a given state.
    fn actions() -> &'static [Self::Action];

    /// Identifies internal actions subject to weak fairness.
    fn internal(action: Self::Action) -> bool;

    /// Applies an action, returning `None` when the state refuses it.
    fn step(&self, action: Self::Action) -> Option<Self>;

    /// Registers predicates checked at each reachable state.
    fn invariants() -> Vec<Claim<Self>> {
        Vec::new()
    }

    /// Registers predicates checked at each emitted step.
    fn postconditions() -> Vec<StepClaim<Self>> {
        Vec::new()
    }

    /// Registers scoped eventuality claims.
    fn progress() -> Vec<Progress<Self>> {
        Vec::new()
    }
}
