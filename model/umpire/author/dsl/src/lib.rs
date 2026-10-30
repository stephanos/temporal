//! State machines and named claims consumed by the Umpire checker.

#![forbid(unsafe_code)]
#![warn(missing_docs)]

mod action;
mod catalog;
pub mod claim;
pub mod closable;
pub mod countdown;
mod finite;
mod machine;
pub mod outcome;
mod path;
#[doc(hidden)]
pub mod path_build;
mod phased;
mod progress;
mod scope;
mod value;

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
pub use phased::{Phase, Phased};
pub use progress::{Progress, ProgressEvaluation, ProgressEvaluator};
pub use scope::{Assumption, AssumptionExt, Scope};
#[doc(hidden)]
pub use umpire_dsl_macros::__path_parse;
pub use umpire_dsl_macros::{Action, Finite, State, World, machine};
#[doc(hidden)]
pub use value::{ActionValues, IntoValue, StartFailure, Value, Values};

/// The admitted Model a `#[machine]` macro embeds, for the lifted back ends; never in the guest.
#[cfg(not(target_arch = "wasm32"))]
#[doc(hidden)]
pub mod __lifted {
    pub use umpire_ir::*;
    /// A `#[machine]` World, which carries its module's admitted Model.
    pub trait Lifts {
        /// The embedded Model, or its located subset refusal.
        fn __umpire_lifted() -> Embedded;
    }
    /// A `#[machine]` World whose module the front end admitted: typed access to its state
    /// and outcomes by the embedded Model's field order, for the interpreted checker.
    pub trait Interprets: Lifts + crate::Machine {
        /// The state field at `slot`, in declaration order.
        fn __umpire_get(state: &Self::State, slot: usize) -> value::Value;
        /// Replaces the state field at `slot`.
        fn __umpire_set(state: &mut Self::State, slot: usize, value: &value::Value);
        /// A state from every field's value, in declaration order.
        fn __umpire_state(values: &[value::Value]) -> Self::State;
        /// An edge outcome from its value.
        fn __umpire_outcome(value: &value::Value) -> Self::Outcome;
        /// An authored edge fact from its admitted value.
        fn __umpire_fact(_value: &value::Value) -> Self::Fact {
            panic!("the Model declares no fact decoder")
        }
        /// An edge outcome's value.
        fn __umpire_outcome_value(outcome: &Self::Outcome) -> value::Value;
        /// Each `check!` the embedded Model's claims index, as its failure reports it.
        fn __umpire_checks() -> &'static [&'static str];
        /// One callback per claim of the embedded Model, each judging its own claim through
        /// `W`, since a claim callback is a plain function of the World.
        fn __umpire_callbacks<W: Evaluates<Self>>() -> Callbacks<W, Self>;
    }
    /// A catalog action's exported values as its concrete identity.
    pub trait ConcreteOf {
        /// The qualified variant of `action` and its payload values in declaration order.
        fn of<M: crate::Machine<Action: crate::ActionValues<M>>>(action: &M::Action) -> Concrete;
    }
    impl ConcreteOf for Concrete {
        fn of<M: crate::Machine<Action: crate::ActionValues<M>>>(action: &M::Action) -> Self {
            let (action, fields) = crate::ActionValues::<M>::action_value(action);
            Self { action: action.into(), fields: literals(fields) }
        }
    }
    /// A selected World's exported values, its starts' and its catalog's, as one binding.
    pub trait BindingOf {
        /// The values of `world`, each of `starts` and each catalog action, in their orders.
        fn of<M>(world: &M, starts: &[M::State], actions: &[M::Action]) -> Binding
        where
            M: crate::Machine<State: crate::Values, Action: crate::ActionValues<M>> + crate::Values;
    }
    impl BindingOf for Binding {
        fn of<M>(world: &M, starts: &[M::State], actions: &[M::Action]) -> Self
        where
            M: crate::Machine<State: crate::Values, Action: crate::ActionValues<M>> + crate::Values,
        {
            Self {
                world: literals(crate::Values::values(world)),
                starts: starts.iter().map(|state| literals(crate::Values::values(state))).collect(),
                actions: actions.iter().map(Concrete::of::<M>).collect(),
            }
        }
    }
    fn literals(values: Vec<(&'static str, crate::Value)>) -> Vec<(String, Literal)> {
        values.into_iter().map(|(name, value)| (name.into(), value.into())).collect()
    }
    /// A World that judges the claims of `M`'s embedded Model by their position.
    pub trait Evaluates<M: crate::Machine> {
        /// Records the checks of the invariant at `index` for `state`.
        fn __umpire_invariant(&self, index: usize, state: &M::State);
        /// Records the checks of the postcondition at `index` for one edge.
        fn __umpire_postcondition(
            &self,
            index: usize,
            before: &M::State,
            after: &M::State,
            outcome: &M::Outcome,
        );
        /// The progress claim at `index`, evaluated at `state`.
        fn __umpire_progress(
            &self,
            index: usize,
            state: &M::State,
        ) -> crate::ProgressEvaluation<M::Action>;
    }
    /// An invariant's check-recording callback and its verdict-only adapter.
    pub type InvariantCallbacks<W, M> = (
        fn(&W, &<M as crate::Machine>::State),
        fn(&W, &<M as crate::Machine>::State) -> crate::Verdict,
    );
    /// A postcondition's check-recording callback.
    pub type PostconditionCallback<W, M> = fn(
        &W,
        &<M as crate::Machine>::State,
        &<M as crate::Machine>::State,
        &<M as crate::Machine>::Action,
        &<M as crate::Machine>::Outcome,
        &[<M as crate::Machine>::Fact],
    );
    /// A progress claim's evaluator.
    pub type ProgressCallback<W, M> =
        fn(
            &W,
            &<M as crate::Machine>::State,
        ) -> crate::ProgressEvaluation<<M as crate::Machine>::Action>;
    /// Exactly one callback per claim of the embedded Model, in its order.
    pub struct Callbacks<W, M: crate::Machine> {
        /// Per invariant.
        pub invariants: Vec<InvariantCallbacks<W, M>>,
        /// Per postcondition.
        pub postconditions: Vec<PostconditionCallback<W, M>>,
        /// Per progress claim.
        pub progress: Vec<ProgressCallback<W, M>>,
    }
}

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
