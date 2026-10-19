use std::{
    fmt::{Debug, Display},
    hash::Hash,
};

use smallvec::SmallVec;

use crate::{Assumption, Claim, Progress, Scope, StepClaim};

/// A world of constants defining a finite state machine and its named claims.
pub trait Machine: Sized {
    /// The dynamic state, independent of the world's constants.
    type State: Clone + Eq + Hash + Display;
    /// The finite action catalog's value type.
    type Action: Clone + Eq + Hash + Display;
    /// A domain outcome carried by an edge, including rejecting outcomes.
    type Outcome: Debug;
    /// A domain fact emitted by an edge.
    type Fact: Debug;

    /// Enumerates initial states in field order with each domain ascending.
    fn starts(&self) -> Vec<Self::State>;

    /// Enumerates the finite action catalog in declaration order, including disabled actions.
    fn actions(&self) -> Vec<Self::Action>;

    /// Identifies environment-independent actions; this does not imply fairness.
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

    /// Registers eventuality claims under their assumptions.
    fn progress(&self) -> Vec<Progress<Self>> {
        Vec::new()
    }

    /// Registers named assumption builders in declaration order.
    fn assumptions(&self) -> Vec<(&'static str, Assumption<Self::Action>)> {
        self.scopes()
    }

    /// Compatibility registration for manual trait Models using scopes.
    fn scopes(&self) -> Vec<(&'static str, Scope<Self::Action>)> {
        Vec::new()
    }

    /// Names claims in source order across invariant, postcondition and progress kinds.
    /// An empty list retains the manual world's existing registration order.
    fn claim_order(&self) -> Vec<&'static str> {
        Vec::new()
    }
}

/// An emitted edge `(outcome, facts, next)`, including an edge with a rejecting outcome.
pub type Step<S, O, F> = (O, Vec<F>, S);

/// Enabled edges in written order, storing one edge inline; empty disables the action.
pub type Steps<S, O, F> = SmallVec<[Step<S, O, F>; 1]>;
