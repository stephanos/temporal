//! The shortest path through a pattern's waypoints, asked of the checker one leg at a time.

use umpire_checker::Checker;
use umpire_dsl::{Claim, Gap, Machine, Path, StepError, Steps, Verdict, Waypoint};

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

/// The path from `starts` to the first waypoint, then from each reached waypoint to the next,
/// each leg the shortest within its gap. Shortest per leg need not be shortest overall.
pub(crate) fn through<M: Machine>(
    world: &M,
    starts: Vec<M::State>,
    path: &Path<M::State>,
    budget: usize,
) -> Result<Trace<M>, Refusal>
where
    M::State: PartialEq,
{
    let mut found: Option<Trace<M>> = None;
    let mut from = starts;
    let legs = std::iter::once((Gap::Any, path.first())).chain(path.legs());
    for (waypoint, (gap, judge)) in legs.enumerate() {
        let trace = match leg(world, from.clone(), gap, judge, budget)? {
            Ok(trace) => trace,
            Err(cut) => {
                let last = found.as_ref().map(end).or(from.first());
                let state = last.map_or_else(|| "no start state".into(), ToString::to_string);
                return Err(Refusal::Unbridged { waypoint, state, cut });
            }
        };
        from = vec![end(&trace).clone()];
        match &mut found {
            Some(found) => found.steps.extend(trace.steps),
            None => found = Some(trace),
        }
    }
    let Some(found) = found else { unreachable!("a path has at least one waypoint") };
    Ok(found)
}

/// The shortest path from `starts` to the waypoint within the gap, or whether the search was
/// cut by its budget.
fn leg<M: Machine>(
    world: &M,
    starts: Vec<M::State>,
    gap: Gap,
    waypoint: &Waypoint<M::State>,
    budget: usize,
) -> Result<Result<Trace<M>, bool>, Refusal>
where
    M::State: PartialEq,
{
    let explored = Checker::explore(&Rooted { world, starts, query: None }, budget)?;
    // Discovery order is breadth-first, so the first match is a nearest one.
    let Some((id, _)) = explored.states().find(|(_, state)| waypoint.judge(state)) else {
        return Ok(Err(explored.counts.cut));
    };
    let trace = explored.path_to(id)?;
    // A match beyond the bound means every state within it was admitted, so no cut matters.
    let within = gap.bound().is_none_or(|bound| trace.steps.len() <= bound as usize);
    Ok(if within { Ok(trace) } else { Err(false) })
}

fn end<S, A>(trace: &umpire_checker::Path<S, A>) -> &S {
    trace.steps.last().map_or(&trace.start, |step| &step.after)
}
