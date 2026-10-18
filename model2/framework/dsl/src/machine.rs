use std::{fmt::Debug, hash::Hash};

use crate::{Claim, Progress, StepClaim};

/// A world of constants defining a finite state machine and its named claims.
pub trait Machine: Sized {
    /// The dynamic state, independent of the world's constants.
    type State: Clone + Eq + Hash;
    /// The finite action catalog's value type.
    type Action: Clone + Eq + Hash + Debug;
    /// A domain outcome carried by an edge, including rejecting outcomes.
    type Outcome;
    /// A domain fact emitted by an edge.
    type Fact;

    /// Enumerates initial states in field order with each domain ascending.
    fn starts(&self) -> Vec<Self::State>;

    /// Enumerates the finite action catalog in declaration order, including disabled actions.
    fn actions(&self) -> Vec<Self::Action>;

    /// Identifies internal actions subject to weak fairness.
    fn internal(&self, action: &Self::Action) -> bool;

    /// Returns enabled choices in written order; an empty list disables the action.
    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact>;

    /// Registers predicates checked at each reachable state.
    fn invariants(&self) -> Vec<Claim<Self>> {
        Vec::new()
    }

    /// Registers predicates checked at each emitted step.
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        Vec::new()
    }

    /// Registers scoped eventuality claims.
    fn progress(&self) -> Vec<Progress<Self>> {
        Vec::new()
    }
}

/// An emitted edge `(outcome, facts, next)`, including an edge with a rejecting outcome.
pub type Step<S, O, F> = (O, Vec<F>, S);

/// Enabled edges in written order; an empty list disables the action.
pub type Steps<S, O, F> = Vec<Step<S, O, F>>;
