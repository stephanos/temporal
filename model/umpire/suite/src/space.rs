//! A Model's explorations: questions over everything reachable from chosen starts, answered by
//! the checker within a state budget.

use umpire_checker::{Checker, Report};
use umpire_dsl::{CheckOutcome, Path, Registered};

use crate::{
    Refusal,
    search::{self, Rooted, Trace},
};

/// Start states and a state budget; scopes stay the Model's own.
#[derive(Clone, Debug)]
pub struct Space<S> {
    starts: Vec<S>,
    budget: usize,
}

impl<S, const N: usize> From<[S; N]> for Space<S> {
    fn from(starts: [S; N]) -> Self {
        Vec::from(starts).into()
    }
}

impl<S> From<Vec<S>> for Space<S> {
    fn from(starts: Vec<S>) -> Self {
        Self { starts, budget: Checker::BUDGET }
    }
}

impl<S> Space<S> {
    /// Admits at most `n` states; the default is the checker's.
    #[must_use]
    pub const fn budget(mut self, n: usize) -> Self {
        self.budget = n;
        self
    }
}

/// The Model's own claims over the space. A cut exploration answers unknown, never verified.
pub fn verify_all<S: Registered>(space: &Space<S>) -> Result<Report<S::World>, Refusal> {
    let explored = Checker::explore_from(&S::world(), space.starts.clone(), space.budget)?;
    Ok(explored.verify_all()?)
}

/// The shortest path to a reached state where `predicate` fails. A counterexample found before
/// a cut stands; without one, a cut answers unknown.
pub fn refuted<S: Registered>(
    space: &Space<S>,
    predicate: fn(&S) -> bool,
) -> Result<Trace<S::World>, Refusal> {
    let world = S::world();
    let rooted = Rooted { world: &world, starts: space.starts.clone(), query: Some(predicate) };
    let explored = Checker::explore(&rooted, space.budget)?;
    let mut report = explored.verify_all()?;
    search::failed(&report, 1)?;
    let query = report.claims.swap_remove(0);
    match (query.outcome, query.path) {
        (CheckOutcome::Refuted, Some(path)) => Ok(path),
        (CheckOutcome::Verified, _) => Err(Refusal::Holds { reached: report.counts.reached }),
        _ => Err(Refusal::Unknown { budget: space.budget }),
    }
}

/// The shortest admitted path from the space's starts through the waypoints in order, each
/// within its gap, searched as one exploration within the space's budget.
// By value, so `find(&space, path![..])` reads as the question it is.
#[allow(clippy::needless_pass_by_value)]
pub fn find<S: Registered>(space: &Space<S>, path: Path<S>) -> Result<Trace<S::World>, Refusal> {
    search::through(&S::world(), space.starts.clone(), &path, space.budget)
}
