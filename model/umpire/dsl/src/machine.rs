use std::{
    fmt::{Debug, Display},
    hash::Hash,
};

use smallvec::SmallVec;

use crate::{
    ActionCatalog, Assumption, CatalogError, Claim, OutcomeClass, Progress, Scope, StepClaim,
};

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

    /// Admits the action catalog under a concrete-action limit before any step is evaluated.
    /// Engines use this method; `actions` is the unchecked interface for direct callers.
    ///
    /// The default checks the length of `actions`, so it cannot refuse before that list is
    /// built. Generated input-bearing machines count every declared domain first.
    ///
    /// # Errors
    /// Refuses a catalog above `limit`, or one whose count overflows.
    fn checked_catalog(&self, limit: usize) -> Result<ActionCatalog<Self::Action>, CatalogError> {
        ActionCatalog::checked(self.actions(), limit)
    }

    /// Complete declared outcome kinds, or `None` when the world has not declared them.
    /// `Some(Vec::new())` explicitly declares that the outcome catalog is empty.
    fn outcome_catalog(&self) -> Option<Vec<OutcomeClass<Self::Outcome>>> {
        None
    }

    /// Complete declared fault actions, independent of internal-action classification.
    /// `None` is undeclared metadata, distinct from a declared empty catalog.
    fn fault_catalog(&self) -> Option<Vec<Self::Action>> {
        None
    }

    /// Identifies environment-independent actions; this does not imply fairness.
    fn internal(&self, action: &Self::Action) -> bool;

    /// Returns enabled choices in written order; an empty list disables the action.
    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact>;

    /// Evaluates enabled choices, refusing an invalid Model step before publishing any edge.
    /// Engines use this method; `step` is the unchecked interface for direct callers.
    ///
    /// # Errors
    /// Returns a Model error when a rejecting handler changed its state.
    fn checked_step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Result<Steps<Self::State, Self::Outcome, Self::Fact>, StepError> {
        Ok(self.step(state, action))
    }

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

/// Associates a dynamic state with the machine whose constants govern its behavior.
pub trait Associated: Clone + Eq + Hash + Display {
    /// The machine this state belongs to.
    type World: Machine<State = Self>;
}

impl<S: Registered> Associated for S {
    type World = <S as Registered>::World;
}

/// A state whose machine has no constants, as `#[machine]` registers it, so a test can start
/// from a state alone.
pub trait Registered: Clone + Eq + Hash + Display {
    /// The machine this state belongs to.
    type World: Machine<State = Self>;
    /// Builds that machine.
    fn world() -> Self::World;
}

/// An emitted edge `(outcome, facts, next)`, including an edge with a rejecting outcome.
pub type Step<S, O, F> = (O, Vec<F>, S);

/// Enabled edges in written order, storing one edge inline; empty disables the action.
pub type Steps<S, O, F> = SmallVec<[Step<S, O, F>; 1]>;

/// An invalid Model implementation, distinct from an enabled rejecting edge.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum StepError {
    /// A handler both rejected the request and changed its state.
    RejectedAfterMutation,
}

impl Display for StepError {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::RejectedAfterMutation => out.write_str("rejecting step changed state"),
        }
    }
}

impl std::error::Error for StepError {}
