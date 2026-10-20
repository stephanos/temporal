use std::fmt;

use umpire_checker::{Checker, PathStep};
use umpire_dsl::{Machine, Path, Registered, Waypoint};

use crate::{
    Refusal,
    search::{self, Trace},
};

type Action<S> = <<S as Registered>::World as Machine>::Action;

/// An authored path through a Model from a concrete state, each step replayed through the same
/// `Machine::checked_step` the checker uses. It spells as the checker's `paths/` goldens do.
pub struct Run<S: Registered> {
    world: S::World,
    trace: Trace<S::World>,
}

impl<S: Registered> From<S> for Run<S> {
    /// Starts at `state`.
    fn from(state: S) -> Self {
        let trace =
            Trace::<S::World> { start: state, steps: Vec::new(), cycle_start: None, failure: None };
        Self { world: S::world(), trace }
    }
}

impl<S: Registered> Run<S> {
    /// Takes the one step the action emits from the current state.
    pub fn step(self, action: impl Into<Action<S>>) -> Result<Self, Refusal> {
        self.take(action.into(), None)
    }

    /// Takes the action's one step `n` times.
    pub fn repeat(mut self, n: usize, action: impl Into<Action<S>>) -> Result<Self, Refusal> {
        let action = action.into();
        for _ in 0..n {
            self = self.take(action.clone(), None)?;
        }
        Ok(self)
    }

    /// The current state.
    pub fn state(&self) -> &S {
        self.trace.steps.last().map_or(&self.trace.start, |step| &step.after)
    }

    /// The path taken so far.
    pub fn path(&self) -> &Trace<S::World> {
        &self.trace
    }

    /// Fills in a story told by its milestones: from the first waypoint, a state, the shortest
    /// path through every waypoint within its gap is found and replayed.
    // By value, so `Run::along(path![..])` reads as the story it is.
    #[allow(clippy::needless_pass_by_value)]
    pub fn along(path: Path<S>) -> Result<Self, Refusal>
    where
        S: PartialEq,
    {
        let Waypoint::Is(start) = path.first() else { return Err(Refusal::Unanchored) };
        let found = search::through(&S::world(), vec![start.clone()], &path, Checker::BUDGET)?;
        let mut run = Self::from(found.start);
        for step in found.steps {
            run = run.take(step.action, Some(step.choice))?;
        }
        Ok(run)
    }

    /// Takes the given choice, or the only one when `choice` is `None`.
    fn take(mut self, action: Action<S>, choice: Option<usize>) -> Result<Self, Refusal> {
        let before = self.state();
        let steps = self.world.checked_step(before, action.clone()).map_err(|error| {
            umpire_checker::Refusal::Step {
                action: action.to_string(),
                before: before.to_string(),
                error,
            }
        })?;
        let choices = steps.len();
        let choice = match (choice, choices) {
            (_, 0) => None,
            (None, 1) => Some(0),
            (None, _) => {
                return Err(Refusal::Ambiguous {
                    action: action.to_string(),
                    choices,
                    state: before.to_string(),
                });
            }
            (Some(choice), _) => Some(choice),
        };
        let Some((choice, (_, _, after))) =
            choice.and_then(|choice| steps.into_iter().nth(choice).map(|step| (choice, step)))
        else {
            return Err(Refusal::Disabled {
                action: action.to_string(),
                state: before.to_string(),
            });
        };
        self.trace.steps.push(PathStep { action, choice, after });
        Ok(self)
    }
}

impl<S: Registered> fmt::Display for Run<S> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.trace.fmt(out)
    }
}
