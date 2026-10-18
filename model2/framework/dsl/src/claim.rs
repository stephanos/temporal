use crate::Machine;

/// A named predicate over a machine state.
pub struct Claim<S> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The predicate that must hold for each checked state.
    pub predicate: fn(&S) -> bool,
}

impl<S> Claim<S> {
    /// Names a state predicate.
    pub fn new(name: &'static str, predicate: fn(&S) -> bool) -> Self {
        Self { name, predicate }
    }
}

/// A named predicate over the before state, after state, and action of a step.
pub struct StepClaim<S: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The triggering action, or `None` for every emitted step.
    pub action: Option<S::Action>,
    predicate: StepPredicate<S>,
}

enum StepPredicate<S: Machine> {
    Every(fn(&S, &S, S::Action) -> bool),
    After(fn(&S, &S) -> bool),
}

impl<S: Machine> StepClaim<S> {
    /// Names a predicate checked after every emitted step.
    pub fn new(name: &'static str, predicate: fn(&S, &S, S::Action) -> bool) -> Self {
        Self { name, action: None, predicate: StepPredicate::Every(predicate) }
    }

    /// Names a predicate checked only after the given action.
    pub fn after(name: &'static str, action: S::Action, predicate: fn(&S, &S) -> bool) -> Self {
        Self { name, action: Some(action), predicate: StepPredicate::After(predicate) }
    }

    /// Reports whether this claim is triggered by the given action.
    pub fn applies_to(&self, action: S::Action) -> bool {
        self.action.is_none_or(|expected| expected == action)
    }

    /// Evaluates the predicate, treating steps outside its action filter as satisfied.
    pub fn holds(&self, before: &S, after: &S, action: S::Action) -> bool {
        !self.applies_to(action)
            || match self.predicate {
                StepPredicate::Every(predicate) => predicate(before, after, action),
                StepPredicate::After(predicate) => predicate(before, after),
            }
    }
}
