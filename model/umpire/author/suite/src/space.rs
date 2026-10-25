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
#[derive(Clone)]
pub struct Space<S, W = ()> {
    world: W,
    pub(crate) starts: Vec<S>,
    pub(crate) budget: usize,
    pub(crate) alternatives: usize,
    pub(crate) walk_bound: usize,
    pub(crate) catalog_limit: usize,
}

#[allow(
    clippy::missing_fields_in_debug,
    reason = "preserve Space's existing Debug output without requiring World: Debug"
)]
impl<S: fmt::Debug, W> fmt::Debug for Space<S, W> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.debug_struct("Space")
            .field("starts", &self.starts)
            .field("budget", &self.budget)
            .field("alternatives", &self.alternatives)
            .field("walk_bound", &self.walk_bound)
            .finish()
    }
}

impl<S, const N: usize> From<[S; N]> for Space<S> {
    fn from(starts: [S; N]) -> Self {
        Vec::from(starts).into()
    }
}

impl<S> From<Vec<S>> for Space<S> {
    fn from(starts: Vec<S>) -> Self {
        Self {
            world: (),
            starts,
            budget: Checker::BUDGET,
            alternatives: Checker::BUDGET,
            walk_bound: 256,
            catalog_limit: Checker::CATALOG_LIMIT,
        }
    }
}

impl<S: Associated> Space<S> {
    /// Selects one owned world and its declared start states.
    pub fn new(world: S::World) -> Space<S, Rc<S::World>> {
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
    pub fn with_world(world: S::World, starts: impl Into<Vec<S>>) -> Space<S, Rc<S::World>> {
        Space {
            world: Rc::new(world),
            starts: starts.into(),
            budget: Checker::BUDGET,
            alternatives: Checker::BUDGET,
            walk_bound: 256,
            catalog_limit: Checker::CATALOG_LIMIT,
        }
    }
}

impl<S, W> Space<S, W> {
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

    /// Admits at most `n` concrete catalog actions; a larger catalog is refused before any
    /// Model step. The default is the checker's 4,096.
    #[must_use]
    pub const fn catalog_limit(mut self, n: usize) -> Self {
        self.catalog_limit = n;
        self
    }
}

mod sealed {
    pub trait Sealed {}

    impl<S, W> Sealed for super::Space<S, W> {}
}

#[doc(hidden)]
pub trait SpaceBinding<S: Associated>: sealed::Sealed {
    fn starts(&self) -> &[S];
    fn state_budget(&self) -> usize;
    fn alternative_budget(&self) -> usize;
    fn walk_bound(&self) -> usize;
    fn catalog_limit(&self) -> usize;
    fn world(&self) -> Rc<S::World>;
}

trait WorldSource<S: Associated> {
    fn world(&self) -> Rc<S::World>;
}

impl<S: Registered> WorldSource<S> for () {
    fn world(&self) -> Rc<S::World> {
        Rc::new(S::world())
    }
}

impl<S: Associated> WorldSource<S> for Rc<S::World> {
    fn world(&self) -> Rc<S::World> {
        self.clone()
    }
}

impl<S: Associated, W: WorldSource<S>> SpaceBinding<S> for Space<S, W> {
    fn starts(&self) -> &[S] {
        &self.starts
    }

    fn state_budget(&self) -> usize {
        self.budget
    }

    fn alternative_budget(&self) -> usize {
        self.alternatives
    }

    fn walk_bound(&self) -> usize {
        self.walk_bound
    }

    fn catalog_limit(&self) -> usize {
        self.catalog_limit
    }

    fn world(&self) -> Rc<S::World> {
        self.world.world()
    }
}

/// The Model's own claims over the space. A cut exploration answers unknown, never verified.
pub fn verify_all<S: Associated>(
    space: &impl SpaceBinding<S>,
) -> Result<Report<S::World>, Refusal> {
    let world = space.world();
    let explored = Checker::explore_within(
        world.as_ref(),
        space.starts().to_vec(),
        space.state_budget(),
        space.catalog_limit(),
    )?;
    Ok(explored.verify_all()?)
}

/// The shortest path to a reached state where `predicate` fails. A counterexample found before
/// a cut stands; without one, a cut answers unknown.
pub fn refuted<S: Associated>(
    space: &impl SpaceBinding<S>,
    predicate: fn(&S) -> bool,
) -> Result<Trace<S::World>, Refusal> {
    let world = space.world();
    let rooted =
        Rooted { world: world.as_ref(), starts: space.starts().to_vec(), query: Some(predicate) };
    let explored = rooted.explore(space.state_budget(), space.catalog_limit())?;
    let mut report = explored.verify_all()?;
    let length =
        report.claims.first().and_then(|claim| claim.path.as_ref()).map(|path| path.steps.len());
    search::failed(&report, 1, length)?;
    let query = report.claims.swap_remove(0);
    match (query.outcome, query.path) {
        (CheckOutcome::Refuted, Some(path)) => Ok(path),
        (CheckOutcome::Verified, _) => Err(Refusal::Holds { reached: report.counts.reached }),
        _ => Err(Refusal::Unknown { budget: space.state_budget() }),
    }
}

/// The shortest admitted path from the space's starts through the waypoints in order, each
/// within its gap, searched over one Model exploration within the space's state budget.
// By value, so `find(&space, path![..])` reads as the question it is.
#[allow(clippy::needless_pass_by_value)]
pub fn find<S: Associated, A, O, K>(
    space: &impl SpaceBinding<S>,
    path: Path<S, A, O>,
) -> Result<Trace<S::World>, Refusal>
where
    Path<S, A, O>: IntoWorldPath<S, K>,
    <S::World as Machine>::Action: 'static,
{
    let path = path.into_world_path();
    if path.is_state_only() {
        drop(path.alternatives(space.alternative_budget())?);
        return search::through(
            space.world().as_ref(),
            space.starts().to_vec(),
            &path,
            space.state_budget(),
            space.catalog_limit(),
        );
    }
    crate::covered::find_pattern(space, &path)
}
