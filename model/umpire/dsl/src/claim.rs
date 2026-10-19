//! Scoped recording and registration of state and edge checks.

use std::cell::Cell;

use crate::{Machine, Step};

/// A claim verdict and the first failed check from that same invocation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Evaluation {
    /// The invocation's local judgment.
    pub verdict: Verdict,
    /// The first failed expression, when checks refuted the claim.
    pub failure: Option<&'static str>,
}

#[derive(Clone, Copy, Default)]
struct Frame {
    active: bool,
    failure: Option<&'static str>,
    hole: Option<Hole>,
}

thread_local! {
    static FRAME: Cell<Frame> = const { Cell::new(Frame { active: false, failure: None, hole: None }) };
}

struct Restore(Frame);

impl Drop for Restore {
    fn drop(&mut self) {
        FRAME.set(self.0);
    }
}

/// Runs unit-returning checks in an isolated scope, restoring enclosing checks on unwind.
pub fn evaluate(checks: impl FnOnce()) -> Evaluation {
    let _restore = Restore(FRAME.replace(Frame { active: true, ..Frame::default() }));
    checks();
    let frame = FRAME.get();
    Evaluation {
        verdict: if frame.failure.is_some() {
            Verdict::Fails
        } else if let Some(hole) = frame.hole {
            Verdict::Unknown(hole)
        } else {
            Verdict::Holds
        },
        failure: frame.failure,
    }
}

/// Records the first failed expression; direct calls outside evaluation have no effect.
pub fn fail(expression: &'static str) {
    let mut frame = FRAME.get();
    if frame.active && frame.failure.is_none() {
        frame.failure = Some(expression);
        FRAME.set(frame);
    }
}

/// Records the first reached knowledge gap; direct calls outside evaluation have no effect.
pub fn unknown(reason: &'static str) {
    let mut frame = FRAME.get();
    if frame.active && frame.hole.is_none() {
        frame.hole = Some(Hole { reason });
        FRAME.set(frame);
    }
}

/// Evaluates an expression once and records a failed check without interrupting execution.
#[macro_export]
macro_rules! check {
    ($expression:expr $(,)?) => {{
        if !$expression {
            $crate::claim::fail(stringify!($expression));
        }
    }};
}

/// Records a reached gap without interrupting execution.
#[macro_export]
macro_rules! unknown {
    ($reason:expr $(,)?) => {{
        $crate::claim::unknown($reason);
    }};
}

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
    checks: Option<fn(&M, &M::State)>,
}

impl<M: Machine> Claim<M> {
    /// Names a state predicate with access to the world's constants.
    pub fn new(name: &'static str, predicate: fn(&M, &M::State) -> Verdict) -> Self {
        Self { name, predicate, checks: None }
    }

    /// Names recorded checks with a verdict-only adapter for direct predicate callers.
    pub fn with_checks(
        name: &'static str,
        checks: fn(&M, &M::State),
        predicate: fn(&M, &M::State) -> Verdict,
    ) -> Self {
        Self { name, predicate, checks: Some(checks) }
    }

    /// Evaluates this registered callback once, retaining only its own check evidence.
    pub fn evaluate(&self, world: &M, state: &M::State) -> Evaluation {
        if let Some(checks) = self.checks {
            evaluate(|| checks(world, state))
        } else {
            Evaluation { verdict: (self.predicate)(world, state), failure: None }
        }
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

/// Unit-returning checks over an emitted edge.
pub type StepCheckPredicate<M> = fn(
    &M,
    &<M as Machine>::State,
    &<M as Machine>::State,
    &<M as Machine>::Action,
    &<M as Machine>::Outcome,
    &[<M as Machine>::Fact],
);

enum EdgePredicate<M: Machine> {
    Manual(StepPredicate<M>),
    Checks(StepCheckPredicate<M>),
}

/// A named predicate over an emitted edge, optionally filtered to an action.
pub struct StepClaim<M: Machine> {
    /// The claim's name in reports.
    pub name: &'static str,
    /// The triggering action, or `None` for every emitted edge.
    pub action: Option<M::Action>,
    predicate: EdgePredicate<M>,
}

impl<M: Machine> StepClaim<M> {
    /// Names a predicate checked after every emitted edge.
    pub fn new(name: &'static str, predicate: StepPredicate<M>) -> Self {
        Self { name, action: None, predicate: EdgePredicate::Manual(predicate) }
    }

    /// Names a predicate checked only after the given action.
    pub fn after(name: &'static str, action: M::Action, predicate: StepPredicate<M>) -> Self {
        Self { name, action: Some(action), predicate: EdgePredicate::Manual(predicate) }
    }

    /// Names checks evaluated after every emitted edge.
    pub fn with_checks(name: &'static str, predicate: StepCheckPredicate<M>) -> Self {
        Self { name, action: None, predicate: EdgePredicate::Checks(predicate) }
    }

    /// Names checks evaluated only after the given action.
    pub fn after_checks(
        name: &'static str,
        action: M::Action,
        predicate: StepCheckPredicate<M>,
    ) -> Self {
        Self { name, action: Some(action), predicate: EdgePredicate::Checks(predicate) }
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
        self.evaluate(world, before, action, emitted).verdict
    }

    /// Evaluates an emitted edge once, retaining only this callback's check evidence.
    pub fn evaluate(
        &self,
        world: &M,
        before: &M::State,
        action: &M::Action,
        emitted: &Step<M::State, M::Outcome, M::Fact>,
    ) -> Evaluation {
        if !self.applies_to(action) {
            return Evaluation { verdict: Verdict::Holds, failure: None };
        }
        match self.predicate {
            EdgePredicate::Manual(predicate) => Evaluation {
                verdict: predicate(world, before, &emitted.2, action, &emitted.0, &emitted.1),
                failure: None,
            },
            EdgePredicate::Checks(predicate) => {
                evaluate(|| predicate(world, before, &emitted.2, action, &emitted.0, &emitted.1))
            }
        }
    }
}
