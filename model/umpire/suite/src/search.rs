//! The shortest path through a pattern's waypoints, asked of the checker as one exploration of
//! the Model's states paired with how far along the pattern each path has come.

use std::fmt;

use umpire_checker::{Checker, Path as CheckerPath, PathStep, Report};
use umpire_dsl::{CheckOutcome, Claim, Gap, Machine, Path, StepError, Steps, Verdict, Waypoint};

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

/// A Model state with the pattern's partial matches that end at it: for each waypoint index,
/// the fewest steps since the waypoint before it was matched, or `None` when no path waits for
/// that waypoint. Index 0 always waits; the last index, once set, means the pattern is complete.
#[derive(Clone, PartialEq, Eq, Hash)]
pub(crate) struct Along<S> {
    state: S,
    waiting: Box<[Option<u32>]>,
}

impl<S: fmt::Display> fmt::Display for Along<S> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.state.fmt(out)
    }
}

/// The world paired with a pattern. Each Model step is one step here, so a shortest path here
/// is a shortest Model path that has the waypoints in order within their gaps.
struct Through<'a, M: Machine> {
    world: &'a M,
    starts: Vec<M::State>,
    /// Each waypoint with the gap before it; the first gap is from the starts, any length.
    waypoints: Vec<(Gap, &'a Waypoint<M::State>)>,
}

impl<M: Machine> Through<'_, M>
where
    M::State: PartialEq,
{
    /// Matches every waiting waypoint the state satisfies, in order, so several consecutive
    /// waypoints may match the same state.
    fn arrive(&self, state: M::State, mut waiting: Box<[Option<u32>]>) -> Along<M::State> {
        let complete = self.waypoints.len();
        for (index, (_, waypoint)) in self.waypoints.iter().enumerate() {
            if waiting[index].is_some() && waypoint.judge(&state) {
                waiting[index + 1] = Some(0);
            }
        }
        if waiting[complete].is_some() {
            // One completed form per Model state keeps the explored pairs few.
            waiting = (0..=complete).map(|index| (index == complete).then_some(0)).collect();
        }
        Along { state, waiting }
    }

    /// Ages every partial match by one step, dropping those whose gap is spent.
    fn aged(&self, waiting: &[Option<u32>]) -> Box<[Option<u32>]> {
        let complete = self.waypoints.len();
        (0..=complete)
            .map(|index| {
                let steps = waiting[index]?;
                match self.waypoints.get(index).map(|(gap, _)| gap.bound()) {
                    Some(Some(bound)) => (steps < bound).then_some(steps + 1),
                    _ => Some(0),
                }
            })
            .collect()
    }
}

impl<M: Machine> Machine for Through<'_, M>
where
    M::State: PartialEq,
{
    type State = Along<M::State>;
    type Action = M::Action;
    type Outcome = M::Outcome;
    type Fact = M::Fact;

    fn starts(&self) -> Vec<Self::State> {
        let fresh = || (0..=self.waypoints.len()).map(|index| (index == 0).then_some(0)).collect();
        self.starts.iter().map(|start| self.arrive(start.clone(), fresh())).collect()
    }

    fn actions(&self) -> Vec<Self::Action> {
        self.world.actions()
    }

    fn internal(&self, action: &Self::Action) -> bool {
        self.world.internal(action)
    }

    fn step(
        &self,
        along: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact> {
        let steps = self.world.step(&along.state, action);
        self.paired(along, steps)
    }

    fn checked_step(
        &self,
        along: &Self::State,
        action: Self::Action,
    ) -> Result<Steps<Self::State, Self::Outcome, Self::Fact>, StepError> {
        let steps = self.world.checked_step(&along.state, action)?;
        Ok(self.paired(along, steps))
    }
}

impl<M: Machine> Through<'_, M>
where
    M::State: PartialEq,
{
    fn paired(
        &self,
        along: &Along<M::State>,
        steps: Steps<M::State, M::Outcome, M::Fact>,
    ) -> Steps<Along<M::State>, M::Outcome, M::Fact> {
        steps
            .into_iter()
            .map(|(outcome, facts, after)| {
                (outcome, facts, self.arrive(after, self.aged(&along.waiting)))
            })
            .collect()
    }
}

/// The shortest path from `starts` through every waypoint in order, each within its gap.
pub(crate) fn through<M: Machine>(
    world: &M,
    starts: Vec<M::State>,
    path: &Path<M::State>,
    budget: usize,
) -> Result<Trace<M>, Refusal>
where
    M::State: PartialEq,
{
    let waypoints: Vec<_> = std::iter::once((Gap::Any, path.first())).chain(path.legs()).collect();
    let complete = waypoints.len();
    let paired = Through { world, starts, waypoints };
    let explored = Checker::explore(&paired, budget)?;
    failed(&explored.verify_all()?, 0)?;
    // Discovery order is breadth-first, so the first state that has come furthest along is a
    // nearest one, and for an unmatched waypoint it is where the waypoint before it matched.
    let furthest = |along: &Along<M::State>| along.waiting.iter().rposition(Option::is_some);
    let reached = explored.states().filter_map(|(_, along)| furthest(along)).max().unwrap_or(0);
    let first = explored.states().find(|(_, along)| furthest(along) == Some(reached));
    let state = match (reached, first, paired.starts.as_slice()) {
        (_, Some((id, _)), _) if reached == complete => {
            return Ok(unpaired(explored.path_to(id)?));
        }
        (0, _, []) => "no start state".into(),
        (0, _, [only]) => only.to_string(),
        (0, _, starts) => format!("the space's {} starts", starts.len()),
        (_, Some((_, along)), _) => along.state.to_string(),
        (_, None, _) => unreachable!("a waypoint past the first was matched at an admitted state"),
    };
    let cut = explored.counts.cut && !settled(&paired, &explored);
    Err(Refusal::Unbridged { waypoint: reached, state, cut })
}

/// Whether a cut exploration still settles that no path exists: when the first waypoint is one
/// state and every later gap is bounded, every completion begins where that state was first
/// reached and is at most the sum of the bounds longer, so no longer path needs exploring.
fn settled<M: Machine>(
    paired: &Through<'_, M>,
    explored: &umpire_checker::Result<Through<'_, M>>,
) -> bool
where
    M::State: PartialEq,
{
    let Some((_, Waypoint::Is(anchor))) = paired.waypoints.first() else { return false };
    let Some(span) = paired.waypoints[1..]
        .iter()
        .try_fold(0usize, |span, (gap, _)| Some(span + gap.bound()? as usize))
    else {
        return false;
    };
    let Some((id, _)) = explored.states().find(|(_, along)| along.state == *anchor) else {
        return false;
    };
    let Ok(prefix) = explored.path_to(id) else { return false };
    // Breadth-first, every path shorter than the deepest admitted state was explored in full.
    prefix.steps.len() + span < explored.counts.depth
}

fn unpaired<S, A>(path: CheckerPath<Along<S>, A>) -> CheckerPath<S, A> {
    CheckerPath {
        start: path.start.state,
        steps: path
            .steps
            .into_iter()
            .map(|step| PathStep {
                action: step.action,
                choice: step.choice,
                after: step.after.state,
            })
            .collect(),
        cycle_start: path.cycle_start,
        failure: path.failure,
    }
}

/// Refuses an answer resting on a skipped edge: the checker records a panicking step as a
/// refuted claim of its own and explores on without that edge. Claims before `from` are the
/// question's own.
pub(crate) fn failed<M: Machine>(report: &Report<M>, from: usize) -> Result<(), Refusal> {
    let skipped = report.claims[from..].iter().find(|claim| claim.outcome == CheckOutcome::Refuted);
    let Some(path) = skipped.and_then(|claim| claim.path.as_ref()) else { return Ok(()) };
    let state = path.steps.last().map_or(&path.start, |step| &step.after).to_string();
    let failure = path.failure.clone().unwrap_or_else(|| "a step was skipped".into());
    Err(Refusal::Skipped { failure, state })
}
