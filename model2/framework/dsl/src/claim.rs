use crate::{Machine, Step};

/// A reached gap in a claim's knowledge.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Hole {
    /// Why this predicate cannot decide the claim.
    pub reason: &'static str,
}

/// A predicate's local judgment, distinct from the checker's aggregate outcome.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Verdict {
    /// The predicate holds for this state or edge.
    Holds,
    /// The predicate fails for this state or edge.
    Fails,
    /// A reached hole prevents a judgment.
    Unknown(Hole),
}

impl From<bool> for Verdict {
    fn from(holds: bool) -> Self {
        if holds { Self::Holds } else { Self::Fails }
    }
}

/// A named predicate over a world's constants and dynamic state.
pub struct Claim<M: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The predicate evaluated for each checked state.
    pub predicate: fn(&M, &M::State) -> Verdict,
}

impl<M: Machine> Claim<M> {
    /// Names a state predicate with access to the world's constants.
    pub fn new(name: &'static str, predicate: fn(&M, &M::State) -> Verdict) -> Self {
        Self { name, predicate }
    }
}

/// An edge predicate over the world, before/after states, action, outcome, and emitted facts.
pub type StepPredicate<M> = fn(
    &M,
    &<M as Machine>::State,
    &<M as Machine>::State,
    &<M as Machine>::Action,
    &<M as Machine>::Outcome,
    &[<M as Machine>::Fact],
) -> Verdict;

/// A named predicate over an emitted edge, optionally filtered to an action.
pub struct StepClaim<M: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The triggering action, or `None` for every emitted edge.
    pub action: Option<M::Action>,
    predicate: StepPredicate<M>,
}

impl<M: Machine> StepClaim<M> {
    /// Names a predicate checked after every emitted edge.
    pub fn new(name: &'static str, predicate: StepPredicate<M>) -> Self {
        Self { name, action: None, predicate }
    }

    /// Names a predicate checked only after the given action.
    pub fn after(name: &'static str, action: M::Action, predicate: StepPredicate<M>) -> Self {
        Self { name, action: Some(action), predicate }
    }

    /// Reports whether this claim is triggered by the given action.
    pub fn applies_to(&self, action: &M::Action) -> bool {
        self.action.as_ref().is_none_or(|expected| expected == action)
    }

    /// Evaluates an emitted edge, satisfying edges outside the action filter.
    pub fn holds(
        &self,
        world: &M,
        before: &M::State,
        action: &M::Action,
        emitted: &Step<M::State, M::Outcome, M::Fact>,
    ) -> Verdict {
        if self.applies_to(action) {
            (self.predicate)(world, before, &emitted.2, action, &emitted.0, &emitted.1)
        } else {
            Verdict::Holds
        }
    }
}
