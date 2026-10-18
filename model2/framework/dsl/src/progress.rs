use crate::{Machine, Scope};

/// A named claim that a trigger is eventually followed by a target within a scope.
#[must_use]
pub struct Progress<S: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The predicate that starts an eventuality obligation.
    pub trigger: fn(&S) -> bool,
    /// The predicate that fulfills an eventuality obligation.
    pub target: fn(&S) -> bool,
    /// The actions retained while checking the obligation.
    pub scope: Scope<S::Action>,
}

impl<S: Machine> Progress<S> {
    /// Names an initially unconditional claim over all actions.
    pub fn new(name: &'static str) -> Self {
        Self { name, trigger: |_| true, target: |_| true, scope: Scope::default() }
    }

    /// Sets the predicate that starts an obligation.
    pub fn whenever(mut self, predicate: fn(&S) -> bool) -> Self {
        self.trigger = predicate;
        self
    }

    /// Sets the predicate that must eventually hold.
    pub fn eventually(mut self, predicate: fn(&S) -> bool) -> Self {
        self.target = predicate;
        self
    }

    /// Restricts the actions considered by the eventuality search.
    pub fn within(mut self, scope: Scope<S::Action>) -> Self {
        self.scope = scope;
        self
    }
}
