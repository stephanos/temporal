use std::{fmt, rc::Rc};

use umpire_checker::{Checker, PathStep};
use umpire_dsl::{Associated, Machine, PathNode, Registered, Waypoint};

use crate::{
    Refusal, WorldPath,
    search::{self, Anchor, Graph, Rooted, Trace},
};

type Action<S> = <<S as Associated>::World as Machine>::Action;

/// An authored path through a Model from a concrete state. Explicit steps use
/// `Machine::checked_step`; searched paths retain explored rows. Display matches checker paths.
pub struct Run<S: Associated> {
    world: Rc<S::World>,
    trace: Trace<S::World>,
}

impl<S: Registered> From<S> for Run<S> {
    /// Starts at `state`.
    fn from(state: S) -> Self {
        Self::new(S::world(), state)
    }
}

impl<S: Registered> Run<S> {
    /// Finds a story using the state's registered world.
    pub fn along(path: WorldPath<S>) -> Result<Self, Refusal>
    where
        Action<S>: 'static,
    {
        Self::along_with_budget(path, Checker::BUDGET)
    }

    /// Finds a story using the registered world within a state budget.
    // By value, matching `Run::along(path![..])`.
    #[allow(clippy::needless_pass_by_value)]
    pub fn along_with_budget(path: WorldPath<S>, budget: usize) -> Result<Self, Refusal>
    where
        Action<S>: 'static,
    {
        Self::along_with_world(S::world, &path, budget)
    }
}

impl<S: Associated> Run<S> {
    /// Starts at `state` in one owned world.
    pub fn new(world: S::World, state: S) -> Self {
        let trace =
            Trace::<S::World> { start: state, steps: Vec::new(), cycle_start: None, failure: None };
        Self { world: Rc::new(world), trace }
    }

    pub(crate) fn retained(world: Rc<S::World>, trace: Trace<S::World>) -> Self {
        Self { world, trace }
    }

    /// Takes the one step the action emits from the current state.
    pub fn step(self, action: impl Into<Action<S>>) -> Result<Self, Refusal> {
        self.take(action.into())
    }

    /// Takes the action's one step `n` times.
    pub fn repeat(mut self, n: usize, action: impl Into<Action<S>>) -> Result<Self, Refusal> {
        let action = action.into();
        for _ in 0..n {
            self = self.take(action.clone())?;
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
    /// path through every waypoint within its gap is found over retained Model rows.
    // By value, so `Run::along(path![..])` reads as the story it is.
    pub fn along_in(world: S::World, path: WorldPath<S>) -> Result<Self, Refusal>
    where
        Action<S>: 'static,
    {
        Self::along_in_with_budget(world, path, Checker::BUDGET)
    }

    /// Finds the shortest path, admitting at most `budget` distinct Model states.
    // By value, matching `Run::along(path![..])`.
    #[allow(clippy::needless_pass_by_value)]
    pub fn along_in_with_budget(
        world: S::World,
        path: WorldPath<S>,
        budget: usize,
    ) -> Result<Self, Refusal>
    where
        Action<S>: 'static,
    {
        Self::along_with_world(|| world, &path, budget)
    }

    fn along_with_world(
        world: impl FnOnce() -> S::World,
        path: &WorldPath<S>,
        budget: usize,
    ) -> Result<Self, Refusal>
    where
        Action<S>: 'static,
    {
        let count = path.alternative_count()?;
        if count != 1 {
            return Err(Refusal::StoryAlternatives { count });
        }
        let alternatives = path.alternatives(1)?;
        let Some(PathNode::State(first)) = alternatives[0].nodes.first() else {
            return Err(Refusal::Unanchored);
        };
        let Waypoint::Is(start) = first.as_ref() else { return Err(Refusal::Unanchored) };
        let world = Rc::new(world());
        let rooted = Rooted { world: world.as_ref(), starts: vec![start.clone()], query: None };
        let explored = rooted.explore(budget, Checker::CATALOG_LIMIT)?;
        let found = match Graph::new(&explored)?.shortest(&alternatives[0], Anchor::Starts) {
            Ok(found) => found,
            Err(error @ Refusal::Unbridged { .. }) => {
                search::failed(&explored.verify_all()?, 0, None)?;
                return Err(error);
            }
            Err(error) => return Err(error),
        };
        search::failed(&explored.verify_all()?, 0, Some(found.trace.steps.len()))?;
        Ok(Self::retained(world.clone(), found.trace))
    }

    fn take(mut self, action: Action<S>) -> Result<Self, Refusal> {
        let before = self.state();
        let steps = self.world.checked_step(before, action.clone()).map_err(|error| {
            umpire_checker::Refusal::Step {
                action: action.to_string(),
                before: before.to_string(),
                error,
            }
        })?;
        let choices = steps.len();
        match choices {
            0 => {
                return Err(Refusal::Disabled {
                    action: action.to_string(),
                    state: before.to_string(),
                });
            }
            1 => {}
            _ => {
                return Err(Refusal::Ambiguous {
                    action: action.to_string(),
                    choices,
                    state: before.to_string(),
                });
            }
        }
        let (_, _, after) =
            steps.into_iter().next().unwrap_or_else(|| unreachable!("one step was checked"));
        self.trace.steps.push(PathStep { action, choice: 0, after });
        Ok(self)
    }
}

impl<S: Associated> fmt::Display for Run<S> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.trace.fmt(out)
    }
}
