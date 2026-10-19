use crate::{Assumption, Machine, Scope, Verdict};

/// One ordinary progress-method evaluation at a reached state.
pub struct ProgressEvaluation<A> {
    /// The predicate that starts an eventuality obligation.
    pub trigger: Verdict,
    /// The predicate that fulfills an eventuality obligation.
    pub target: Verdict,
    /// The fixed action restriction returned by the claim.
    pub assumption: Assumption<A>,
}

/// A combined progress predicate evaluated once at each reached state.
pub type ProgressEvaluator<M> =
    fn(&M, &<M as Machine>::State) -> ProgressEvaluation<<M as Machine>::Action>;

/// A named claim that a trigger is eventually followed by a target within a scope.
#[must_use]
pub struct Progress<M: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The predicate that starts an eventuality obligation.
    pub trigger: fn(&M, &M::State) -> Verdict,
    /// The predicate that fulfills an eventuality obligation.
    pub target: fn(&M, &M::State) -> Verdict,
    /// The actions retained while checking the obligation.
    pub scope: Scope<M::Action>,
    /// Combined evaluation for generated claims; manual callbacks remain the fallback.
    pub evaluator: Option<ProgressEvaluator<M>>,
}

impl<M: Machine> Progress<M> {
    /// Names an initially unconditional claim over all actions.
    pub fn new(name: &'static str) -> Self {
        Self {
            name,
            trigger: |_, _| Verdict::Holds,
            target: |_, _| Verdict::Holds,
            scope: Scope::default(),
            evaluator: None,
        }
    }

    /// Registers one combined evaluation per reached state.
    pub fn evaluated(name: &'static str, evaluator: ProgressEvaluator<M>) -> Self {
        Self { evaluator: Some(evaluator), ..Self::new(name) }
    }

    /// Sets the predicate that starts an obligation.
    pub fn whenever(mut self, predicate: fn(&M, &M::State) -> Verdict) -> Self {
        self.trigger = predicate;
        self
    }

    /// Sets the predicate that must eventually hold.
    pub fn eventually(mut self, predicate: fn(&M, &M::State) -> Verdict) -> Self {
        self.target = predicate;
        self
    }

    /// Restricts the actions considered by the eventuality search.
    pub fn within(mut self, scope: Scope<M::Action>) -> Self {
        self.scope = scope;
        self
    }

    /// Restricts a manual claim to the supplied assumption's actions.
    pub fn assuming(mut self, assumption: Assumption<M::Action>) -> Self {
        self.scope = assumption;
        self
    }
}
