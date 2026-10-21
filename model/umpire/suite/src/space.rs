//! A Model's explorations: questions over everything reachable from chosen starts, answered by
//! the checker within a state budget.

use std::{fmt, rc::Rc};

use umpire_checker::{Checker, Report};
use umpire_dsl::{Associated, CheckOutcome, IntoWorldPath, Machine, Path, Registered};

use crate::{
    Refusal,
    search::{self, Rooted, Trace},
};

/// Start states and a state budget; scopes stay the Model's own.
pub struct Space<S: Associated> {
    world: WorldSource<S::World>,
    pub(crate) starts: Vec<S>,
    pub(crate) budget: usize,
    pub(crate) alternatives: usize,
    pub(crate) walk_bound: usize,
}

enum WorldSource<W> {
    Selected(Rc<W>),
    Implicit(fn() -> W),
}

impl<S: Associated> Clone for Space<S> {
    fn clone(&self) -> Self {
        let world = match &self.world {
            WorldSource::Selected(world) => WorldSource::Selected(world.clone()),
            WorldSource::Implicit(make) => WorldSource::Implicit(*make),
        };
        Self {
            world,
            starts: self.starts.clone(),
            budget: self.budget,
            alternatives: self.alternatives,
            walk_bound: self.walk_bound,
        }
    }
}

#[allow(
    clippy::missing_fields_in_debug,
    reason = "preserve Space's existing Debug output without requiring World: Debug"
)]
impl<S: Associated + fmt::Debug> fmt::Debug for Space<S> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.debug_struct("Space")
            .field("starts", &self.starts)
            .field("budget", &self.budget)
            .field("alternatives", &self.alternatives)
            .field("walk_bound", &self.walk_bound)
            .finish()
    }
}

impl<S: Registered, const N: usize> From<[S; N]> for Space<S> {
    fn from(starts: [S; N]) -> Self {
        Vec::from(starts).into()
    }
}

impl<S: Registered> From<Vec<S>> for Space<S> {
    fn from(starts: Vec<S>) -> Self {
        Self {
            world: WorldSource::Implicit(S::world),
            starts,
            budget: Checker::BUDGET,
            alternatives: Checker::BUDGET,
            walk_bound: 256,
        }
    }
}

impl<S: Associated> Space<S> {
    /// Selects one owned world and its declared start states.
    pub fn new(world: S::World) -> Self {
        let starts = world.starts();
        Self::with_world(world, starts)
    }

    /// Selects one owned world and explicit start states, including an empty set.
    /// The start type must belong to that world.
    ///
    /// ```compile_fail
    /// use umpire_suite::Space;
    /// macro_rules! model {
    ///     ($name:ident) => {
    ///         #[umpire_dsl::machine]
    ///         mod $name {
    ///             use umpire_dsl::{Action, State};
    ///             #[derive(State)]
    ///             pub struct Model {}
    ///             #[derive(Action)]
    ///             pub enum User {}
    ///             impl Model {
    ///                 #[on(User)]
    ///                 fn user(&mut self, action: User) { match action {} }
    ///             }
    ///         }
    ///     };
    /// }
    /// model!(first);
    /// model!(second);
    /// let _ = Space::with_world(first::world(), [second::Model {}]);
    /// ```
    pub fn with_world(world: S::World, starts: impl Into<Vec<S>>) -> Self {
        Self {
            world: WorldSource::Selected(Rc::new(world)),
            starts: starts.into(),
            budget: Checker::BUDGET,
            alternatives: Checker::BUDGET,
            walk_bound: 256,
        }
    }

    pub(crate) fn world(&self) -> Rc<S::World> {
        match &self.world {
            WorldSource::Selected(world) => world.clone(),
            WorldSource::Implicit(make) => Rc::new(make()),
        }
    }

    /// Admits at most `n` distinct Model states; pattern progress does not spend this budget.
    /// The default is the checker's.
    #[must_use]
    pub const fn budget(mut self, n: usize) -> Self {
        self.budget = n;
        self
    }

    /// Admits at most `n` authored alternatives, independently of the Model-state budget.
    /// The default is 1,048,576; concrete pattern storage has its own admission ceiling.
    #[must_use]
    pub const fn alternative_budget(mut self, n: usize) -> Self {
        self.alternatives = n;
        self
    }

    /// Limits each sampled walk to `n` Model edges, independently of both admission budgets.
    /// The default is 256; a selected alternative that cannot finish is explicitly refused.
    #[must_use]
    pub const fn walk_bound(mut self, n: usize) -> Self {
        self.walk_bound = n;
        self
    }
}

/// The Model's own claims over the space. A cut exploration answers unknown, never verified.
pub fn verify_all<S: Associated>(space: &Space<S>) -> Result<Report<S::World>, Refusal> {
    let world = space.world();
    let explored = Checker::explore_from(world.as_ref(), space.starts.clone(), space.budget)?;
    Ok(explored.verify_all()?)
}

/// The shortest path to a reached state where `predicate` fails. A counterexample found before
/// a cut stands; without one, a cut answers unknown.
pub fn refuted<S: Associated>(
    space: &Space<S>,
    predicate: fn(&S) -> bool,
) -> Result<Trace<S::World>, Refusal> {
    let world = space.world();
    let rooted =
        Rooted { world: world.as_ref(), starts: space.starts.clone(), query: Some(predicate) };
    let explored = Checker::explore(&rooted, space.budget)?;
    let mut report = explored.verify_all()?;
    let length =
        report.claims.first().and_then(|claim| claim.path.as_ref()).map(|path| path.steps.len());
    search::failed(&report, 1, length)?;
    let query = report.claims.swap_remove(0);
    match (query.outcome, query.path) {
        (CheckOutcome::Refuted, Some(path)) => Ok(path),
        (CheckOutcome::Verified, _) => Err(Refusal::Holds { reached: report.counts.reached }),
        _ => Err(Refusal::Unknown { budget: space.budget }),
    }
}

/// The shortest admitted path from the space's starts through the waypoints in order, each
/// within its gap, searched over one Model exploration within the space's state budget.
// By value, so `find(&space, path![..])` reads as the question it is.
#[allow(clippy::needless_pass_by_value)]
pub fn find<S: Associated, A, O, K>(
    space: &Space<S>,
    path: Path<S, A, O>,
) -> Result<Trace<S::World>, Refusal>
where
    Path<S, A, O>: IntoWorldPath<S, K>,
    <S::World as Machine>::Action: 'static,
{
    let path = path.into_world_path();
    if path.is_state_only() {
        drop(path.alternatives(space.alternatives)?);
        return search::through(space.world().as_ref(), space.starts.clone(), &path, space.budget);
    }
    crate::covered::find_pattern(space, &path)
}
