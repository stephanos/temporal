//! The shortest path through a pattern's waypoints over one budgeted Model exploration.

use std::{
    collections::HashSet,
    panic::{AssertUnwindSafe, catch_unwind},
};

use umpire_checker::{Checker, PathStep, Report, StateId};
use umpire_dsl::{
    ActionCatalog, CatalogError, CheckOutcome, Claim, Gap, Machine, Path, StepError, Steps,
    Verdict, Waypoint,
};

use crate::Refusal;

/// A path as the checker's `paths/` goldens spell it.
pub(crate) type Trace<M> = umpire_checker::Path<<M as Machine>::State, <M as Machine>::Action>;

/// The world seen from chosen states: the same actions and steps without the world's claims,
/// so exploring it asks only what is reachable and, given a query, where the query fails.
pub(crate) struct Rooted<'a, M: Machine> {
    pub(crate) world: &'a M,
    pub(crate) starts: Vec<M::State>,
    pub(crate) query: Option<fn(&M::State) -> bool>,
}

impl<M: Machine> Machine for Rooted<'_, M> {
    type State = M::State;
    type Action = M::Action;
    type Outcome = M::Outcome;
    type Fact = M::Fact;

    fn starts(&self) -> Vec<Self::State> {
        self.starts.clone()
    }

    fn actions(&self) -> Vec<Self::Action> {
        self.world.actions()
    }

    fn checked_catalog(&self, limit: usize) -> Result<ActionCatalog<Self::Action>, CatalogError> {
        self.world.checked_catalog(limit)
    }

    fn internal(&self, action: &Self::Action) -> bool {
        self.world.internal(action)
    }

    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact> {
        self.world.step(state, action)
    }

    fn checked_step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Result<Steps<Self::State, Self::Outcome, Self::Fact>, StepError> {
        self.world.checked_step(state, action)
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        // A claim is a plain `fn`, so it reads the query back from the world it is given.
        let judge = |rooted: &Self, state: &M::State| {
            rooted.query.map_or(Verdict::Holds, |query| query(state).into())
        };
        self.query.map(|_| Claim::new("query", judge)).into_iter().collect()
    }
}

impl<M: Machine> Rooted<'_, M> {
    /// Explores from these starts under the selected catalog limit.
    pub(crate) fn explore(
        &self,
        budget: usize,
        catalog_limit: usize,
    ) -> Result<umpire_checker::Result<Self>, Refusal> {
        Ok(Checker::explore_within(self, self.starts.clone(), budget, catalog_limit)?)
    }
}

struct Along<A> {
    state: StateId,
    next: usize,
    age: u32,
    depth: usize,
    parent: Option<(usize, A, usize)>,
}

struct Through<'a, M: Machine> {
    states: Vec<&'a M::State>,
    waypoints: Vec<(Gap, &'a Waypoint<M::State>)>,
    best: Vec<Vec<Option<u32>>>,
    labels: Vec<Along<M::Action>>,
    furthest: Option<usize>,
}

impl<M: Machine> Through<'_, M> {
    fn arrive(&mut self, mut along: Along<M::Action>) -> Result<Option<usize>, Refusal> {
        loop {
            let best = &mut self.best[along.state.0][along.next];
            // FIFO arrival guarantees an earlier label is no longer. Less waiting age then
            // dominates future arrivals, but must never invalidate an older queued label.
            if best.is_some_and(|age| age <= along.age) {
                return Ok(None);
            }
            *best = Some(along.age);
            let id = self.labels.len();
            if self.furthest.is_none_or(|old| self.labels[old].next < along.next) {
                self.furthest = Some(id);
            }
            if along.next == self.waypoints.len() {
                self.labels.push(along);
                return Ok(Some(id));
            }
            let state = self.states[along.state.0];
            let matches =
                catch_unwind(AssertUnwindSafe(|| self.waypoints[along.next].1.judge(state)))
                    .map_err(|error| Refusal::Predicate {
                        waypoint: along.next,
                        state: state.to_string(),
                        failure: error
                            .downcast_ref::<&str>()
                            .copied()
                            .or_else(|| error.downcast_ref::<String>().map(String::as_str))
                            .unwrap_or("non-string panic payload")
                            .into(),
                    })?;
            let advanced = Along {
                state: along.state,
                next: along.next + 1,
                age: 0,
                depth: along.depth,
                parent: along.parent.clone(),
            };
            self.labels.push(along);
            if !matches {
                return Ok(None);
            }
            along = advanced;
        }
    }

    fn trace(&self, mut id: usize) -> Trace<M> {
        let mut steps = Vec::new();
        while let Some((parent, action, choice)) = &self.labels[id].parent {
            steps.push(PathStep {
                action: action.clone(),
                choice: *choice,
                after: self.states[self.labels[id].state.0].clone(),
            });
            id = *parent;
        }
        steps.reverse();
        Trace::<M> {
            start: self.states[self.labels[id].state.0].clone(),
            steps,
            cycle_start: None,
            failure: None,
        }
    }
}

/// The shortest path from `starts` through every waypoint in order, each within its gap.
pub(crate) fn through<M: Machine, A, O>(
    world: &M,
    starts: Vec<M::State>,
    path: &Path<M::State, A, O>,
    budget: usize,
    catalog_limit: usize,
) -> Result<Trace<M>, Refusal>
where
    M::State: PartialEq,
{
    let mut unique = Vec::new();
    let mut seen = HashSet::new();
    for start in starts {
        if seen.insert(start.clone()) {
            unique.push(start);
        }
    }
    let rooted = Rooted { world, starts: unique, query: None };
    let explored = rooted.explore(budget, catalog_limit)?;
    let states: Vec<_> = explored.states().map(|(_, state)| state).collect();
    let rows: Vec<Vec<_>> = explored
        .states()
        .map(|(id, _)| explored.rows(id).map(Iterator::collect))
        .collect::<Result<_, _>>()?;
    let frontier = if explored.counts.cut {
        rows.iter()
            .flatten()
            .find(|row| row.target.is_none())
            .map(|row| explored.path_to(row.source).map(|path| path.steps.len() + 1))
            .transpose()?
            .unwrap_or(0)
    } else {
        usize::MAX
    };
    let mut search = Through::<M> {
        best: vec![vec![None; path.len() + 1]; states.len()],
        states,
        waypoints: std::iter::once((Gap::Any, path.first())).chain(path.legs()).collect(),
        labels: Vec::new(),
        furthest: None,
    };
    let report = explored.verify_all()?;
    let mut answer = None;
    for id in 0..explored.counts.starts {
        answer =
            search.arrive(Along { state: StateId(id), next: 0, age: 0, depth: 0, parent: None })?;
        if answer.is_some() {
            break;
        }
    }
    let mut cursor = 0;
    while answer.is_none() && cursor < search.labels.len() {
        let along = &search.labels[cursor];
        if along.depth < frontier {
            let next = along.next;
            let age = match search.waypoints[next].0.bound() {
                Some(bound) if along.age < bound => Some(along.age + 1),
                Some(_) => None,
                None => Some(0),
            };
            if let Some(age) = age {
                let depth = along.depth + 1;
                for row in &rows[along.state.0] {
                    let Some(state) = row.target else { continue };
                    answer = search.arrive(Along {
                        state,
                        next,
                        age,
                        depth,
                        parent: Some((cursor, row.action.clone(), row.choice)),
                    })?;
                    if answer.is_some() {
                        break;
                    }
                }
            }
        }
        cursor += 1;
    }
    if let Some(id) = answer {
        let trace = search.trace(id);
        failed(&report, 0, Some(trace.steps.len()))?;
        return Ok(trace);
    }
    failed(&report, 0, None)?;
    let first = search.furthest.map(|id| &search.labels[id]);
    let reached = first.map_or(0, |along| along.next);
    let state = match (reached, first, rooted.starts.as_slice()) {
        (0, _, []) => "no start state".into(),
        (0, _, [only]) => only.to_string(),
        (0, _, starts) => format!("the space's {} starts", starts.len()),
        (_, Some(along), _) => search.states[along.state.0].to_string(),
        (_, None, _) => unreachable!("a waypoint past the first was matched at an admitted state"),
    };
    let cut = explored.counts.cut && !settled(&search, &explored, frontier);
    Err(Refusal::Unbridged { waypoint: reached, state, cut })
}

/// Whether a cut exploration still settles that no path exists: when the first waypoint is one
/// state and every later gap is bounded, every completion begins where that state was first
/// reached and is at most the sum of the bounds longer, so no longer path needs exploring.
fn settled<M: Machine>(
    search: &Through<'_, M>,
    explored: &umpire_checker::Result<Rooted<'_, M>>,
    frontier: usize,
) -> bool
where
    M::State: PartialEq,
{
    let Some((_, Waypoint::Is(anchor))) = search.waypoints.first() else { return false };
    let Some(span) = search.waypoints[1..]
        .iter()
        .try_fold(0usize, |span, (gap, _)| span.checked_add(gap.bound()? as usize))
    else {
        return false;
    };
    let Some((id, _)) = explored.states().find(|(_, state)| *state == anchor) else {
        return false;
    };
    let Ok(prefix) = explored.path_to(id) else { return false };
    prefix.steps.len().checked_add(span).is_some_and(|length| length < frontier)
}

/// Refuses an answer resting on a skipped edge: the checker records a panicking step as a
/// refuted claim of its own and explores on without that edge. Claims before `from` are the
/// question's own.
pub(crate) fn failed<M: Machine>(
    report: &Report<M>,
    from: usize,
    answer: Option<usize>,
) -> Result<(), Refusal> {
    let skipped = report.claims[from..].iter().find(|claim| {
        claim.outcome == CheckOutcome::Refuted
            && claim
                .path
                .as_ref()
                .is_some_and(|path| answer.is_none_or(|length| path.steps.len() < length))
    });
    let Some(path) = skipped.and_then(|claim| claim.path.as_ref()) else { return Ok(()) };
    let state = path.steps.last().map_or(&path.start, |step| &step.after).to_string();
    let failure = path.failure.clone().unwrap_or_else(|| "a step was skipped".into());
    Err(Refusal::Skipped { failure, state })
}
