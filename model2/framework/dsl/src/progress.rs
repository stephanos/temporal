use crate::{Machine, Scope, Verdict};

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
}

impl<M: Machine> Progress<M> {
    /// Names an initially unconditional claim over all actions.
    pub fn new(name: &'static str) -> Self {
        Self {
            name,
            trigger: |_, _| Verdict::Holds,
            target: |_, _| Verdict::Holds,
            scope: Scope::default(),
        }
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
}
