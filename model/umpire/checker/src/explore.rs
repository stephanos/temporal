use std::{
    any::Any,
    panic::{AssertUnwindSafe, catch_unwind},
};

use fixedbitset::FixedBitSet;
use umpire_dsl::{
    CheckOutcome, Claim, Machine, Progress, Step, StepClaim, Verdict, claim::Evaluation,
};

use crate::{
    ClaimResult, FullStates, Path, PathStep, Refusal, Report, StateId, Visited, report::aggregate,
};

/// Work observed during an exploration; disabled actions emit no edge.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Counts {
    /// Distinct initial states admitted before the budget was exhausted.
    pub starts: usize,
    /// Distinct admitted states, shared across all starts.
    pub reached: usize,
    /// Emitted choices, including rejecting outcomes and repeated destinations.
    pub emitted: usize,
    /// Calls to `Machine::checked_step`, including disabled actions and caught panics.
    pub evaluated: usize,
    /// Step evaluations returning no choices (disabled actions).
    pub refused: usize,
    /// Maximum shortest-path distance among admitted states.
    pub depth: usize,
    /// A new state could not be admitted under the state budget.
    pub cut: bool,
}

/// One ordered result of evaluating a state/action pair.
pub struct Transition<M: Machine> {
    /// The source state's discovery id.
    pub source: StateId,
    /// The evaluated action.
    pub action: M::Action,
    /// The result's zero-based position within the action's choices.
    pub choice: usize,
    /// The actual domain outcome, facts and destination, including a budget-rejected destination.
    pub emitted: Step<M::State, M::Outcome, M::Fact>,
    /// The admitted destination id, or `None` when the state budget cut this edge.
    pub target: Option<StateId>,
}

/// Entry points for world exploration and negative-control claims.
pub struct Checker;

/// A bounded graph with claim verdicts, ordered rows and predecessor paths.
pub struct Result<M: Machine> {
    /// Statistics for the work actually performed.
    pub counts: Counts,
    pub(crate) visited: FullStates<M::State>,
    parents: Vec<Option<(StateId, M::Action, usize)>>,
    depths: Vec<usize>,
    pub(crate) transitions: Vec<Transition<M>>,
    pub(crate) completed: Vec<Vec<bool>>,
    pub(crate) actions: Vec<M::Action>,
    pub(crate) catalog_limit: usize,
    pub(crate) progress: Vec<Progress<M>>,
    pub(crate) claims: Vec<ClaimResult<M::State, M::Action>>,
    warnings: Vec<String>,
}

impl Checker {
    /// The default state budget, about a million states.
    pub const BUDGET: usize = 1 << 20;

    /// The default maximum number of concrete catalog actions.
    pub const CATALOG_LIMIT: usize = umpire_dsl::CATALOG_LIMIT;

    /// Explores ordered starts with a shared visited set and FIFO action/choice expansion.
    ///
    /// The budget limits admitted states, not edge evaluations; zero is refused.
    pub fn explore<M: Machine>(
        world: &M,
        budget: usize,
    ) -> std::result::Result<Result<M>, Refusal> {
        Self::explore_claims(
            world,
            world.starts(),
            budget,
            Self::CATALOG_LIMIT,
            &world.invariants(),
            None,
        )
    }

    /// Checks an additional state predicate with the default million-state budget.
    pub fn refuted<M: Machine>(
        world: &M,
        claim: Claim<M>,
    ) -> std::result::Result<Result<M>, Refusal> {
        let mut invariants = world.invariants();
        let supplemental = invariants.len();
        invariants.push(claim);
        Self::explore_claims(
            world,
            world.starts(),
            Self::BUDGET,
            Self::CATALOG_LIMIT,
            &invariants,
            Some(supplemental),
        )
    }

    /// Explores the given ordered starts admitting at most `catalog_limit` concrete actions,
    /// in place of the default [`Checker::CATALOG_LIMIT`]. A larger catalog is refused before
    /// any Model step; later progress queries revalidate it under the same limit.
    pub fn explore_within<M: Machine>(
        world: &M,
        starts: Vec<M::State>,
        budget: usize,
        catalog_limit: usize,
    ) -> std::result::Result<Result<M>, Refusal> {
        Self::explore_claims(world, starts, budget, catalog_limit, &world.invariants(), None)
    }

    fn explore_claims<M: Machine>(
        world: &M,
        starts: Vec<M::State>,
        budget: usize,
        catalog_limit: usize,
        invariants: &[Claim<M>],
        supplemental: Option<usize>,
    ) -> std::result::Result<Result<M>, Refusal> {
        if budget == 0 {
            return Err(Refusal::Budget { given: budget });
        }
        let (actions, empty) = admit(world, catalog_limit)?;
        let postconditions = world.postconditions();
        let progress = world.progress();
        crate::progress::validate_scopes(world, &progress, &actions)?;
        let mut result = Result::new(invariants, &postconditions, &progress, catalog_limit, empty);
        result.seed(world, starts, budget, invariants);
        let mut fired = FixedBitSet::with_capacity(actions.len());
        let mut triggered = FixedBitSet::with_capacity(postconditions.len());
        let mut cursor = 0;
        'search: while !result.counts.cut && cursor < result.visited.len() {
            let source = StateId(cursor);
            result.completed.resize(result.visited.len(), vec![false; actions.len()]);
            let Some(before) = result.visited.get(source).cloned() else {
                unreachable!("BFS cursor has an admitted state")
            };
            for (action_index, action) in actions.iter().enumerate() {
                result.counts.evaluated += 1;
                let emitted =
                    catch_unwind(AssertUnwindSafe(|| world.checked_step(&before, action.clone())));
                let choices = match emitted {
                    Ok(choices) => choices.map_err(|error| Refusal::Step {
                        action: action.to_string(),
                        before: before.to_string(),
                        error,
                    })?,
                    Err(error) => {
                        let mut path = result.path(source);
                        let action_text = action.to_string();
                        path.failure = Some(format!(
                            "action {action_text} panicked: {}",
                            panic_message(error.as_ref())
                        ));
                        result.claims.push(ClaimResult {
                            outcome: CheckOutcome::Refuted,
                            path: Some(path),
                            ..unchecked(&format!("step_{cursor}_{action_index}"))
                        });
                        continue;
                    }
                };
                if choices.is_empty() {
                    result.counts.refused += 1;
                }
                for (choice, emitted) in choices.into_iter().enumerate() {
                    fired.insert(action_index);
                    result.counts.emitted += 1;
                    let edge = Transition {
                        source,
                        action: action.clone(),
                        choice,
                        emitted,
                        target: None,
                    };
                    result.check_edge(
                        world,
                        &edge,
                        invariants.len(),
                        &postconditions,
                        &mut triggered,
                    );
                    let mut edge = edge;
                    if let Some(id) = result.visited.find(&edge.emitted.2) {
                        edge.target = Some(id);
                    } else if result.visited.len() == budget {
                        result.counts.cut = true;
                    } else {
                        let id = result.visited.insert(edge.emitted.2.clone());
                        edge.target = Some(id);
                        result.parents.push(Some((source, action.clone(), choice)));
                        let depth = result.depths[cursor] + 1;
                        result.depths.push(depth);
                        result.counts.depth = result.counts.depth.max(depth);
                        result.check_state(world, id, invariants);
                    }
                    result.transitions.push(edge);
                    if result.counts.cut {
                        break 'search;
                    }
                }
                result.completed[cursor][action_index] = true;
            }
            cursor += 1;
        }
        result.completed.resize(result.visited.len(), vec![false; actions.len()]);
        for (index, claim) in progress.iter().enumerate() {
            let checked = result.evaluate_progress(world, claim, &actions)?;
            result.claims[invariants.len() + postconditions.len() + index] = checked;
        }
        result.finish(&actions, &postconditions, &fired, &triggered);
        result.order_claims(
            &world.claim_order(),
            invariants.len() + postconditions.len() + progress.len(),
            supplemental,
        );
        result.actions = actions;
        result.progress = progress;
        Ok(result)
    }
}

/// Admits the catalog before any step, with a warning for each declared empty variant.
fn admit<M: Machine>(
    world: &M,
    limit: usize,
) -> std::result::Result<(Vec<M::Action>, Vec<String>), Refusal> {
    let catalog = world.checked_catalog(limit)?;
    let empty = catalog
        .variants()
        .unwrap_or_default()
        .iter()
        .filter(|variant| variant.count == 0)
        .map(|variant| format!("variant {} has no actions", variant.name))
        .collect();
    Ok((catalog.into_actions(), empty))
}

pub(crate) fn unchecked<S, A>(name: &str) -> ClaimResult<S, A> {
    ClaimResult {
        name: name.into(),
        outcome: CheckOutcome::Verified,
        verdict: None,
        hint: None,
        path: None,
        holes: Vec::new(),
    }
}

pub(crate) fn panic_message(error: &(dyn Any + Send)) -> &str {
    error
        .downcast_ref::<&str>()
        .copied()
        .or_else(|| error.downcast_ref::<String>().map(String::as_str))
        .unwrap_or("non-string panic payload")
}

impl<M: Machine> Result<M> {
    fn order_claims(&mut self, claim_order: &[&str], declared: usize, supplemental: Option<usize>) {
        let slots: Vec<_> = self.claims[..declared]
            .iter()
            .enumerate()
            .filter_map(|(index, claim)| {
                (Some(index) != supplemental && claim_order.contains(&claim.name.as_str()))
                    .then_some(index)
            })
            .collect();
        for end in 1..slots.len() {
            for cursor in (1..=end).rev() {
                let rank = |index: usize| {
                    claim_order.iter().position(|name| *name == self.claims[index].name)
                };
                if rank(slots[cursor - 1]) <= rank(slots[cursor]) {
                    break;
                }
                self.claims.swap(slots[cursor - 1], slots[cursor]);
            }
        }
    }

    fn new(
        invariants: &[Claim<M>],
        postconditions: &[StepClaim<M>],
        progress: &[Progress<M>],
        catalog_limit: usize,
        warnings: Vec<String>,
    ) -> Self {
        Self {
            counts: Counts::default(),
            visited: FullStates::default(),
            parents: Vec::new(),
            depths: Vec::new(),
            transitions: Vec::new(),
            completed: Vec::new(),
            actions: Vec::new(),
            catalog_limit,
            progress: Vec::new(),
            claims: invariants
                .iter()
                .map(|claim| unchecked(claim.name))
                .chain(postconditions.iter().map(|claim| unchecked(claim.name)))
                .chain(progress.iter().map(|claim| unchecked(claim.name)))
                .collect(),
            warnings,
        }
    }
    fn seed(&mut self, world: &M, starts: Vec<M::State>, budget: usize, invariants: &[Claim<M>]) {
        for start in starts {
            if self.visited.find(&start).is_some() {
                continue;
            }
            if self.visited.len() == budget {
                self.counts.cut = true;
                break;
            }
            let id = self.visited.insert(start);
            self.parents.push(None);
            self.depths.push(0);
            self.counts.starts += 1;
            self.check_state(world, id, invariants);
        }
    }

    fn finish(
        &mut self,
        actions: &[M::Action],
        postconditions: &[StepClaim<M>],
        fired: &FixedBitSet,
        triggered: &FixedBitSet,
    ) {
        debug_assert!(self.transitions.windows(2).all(|rows| rows[0].source.0 <= rows[1].source.0));
        self.counts.reached = self.visited.len();
        if self.counts.cut {
            for claim in &mut self.claims {
                if claim.outcome != CheckOutcome::Refuted {
                    claim.outcome = CheckOutcome::Unknown;
                }
            }
        }
        for (index, action) in actions.iter().enumerate() {
            if !fired.contains(index) {
                let action_text = action.to_string();
                self.warnings.push(format!("action {action_text} never fired"));
            }
        }
        for (index, claim) in postconditions.iter().enumerate() {
            if claim.action.is_some() && !triggered.contains(index) {
                self.warnings.push(format!("after claim {} never triggered", claim.name));
            }
        }
    }
    /// Admitted states and ids in discovery order.
    pub fn states(&self) -> impl ExactSizeIterator<Item = (StateId, &M::State)> {
        self.visited.iter()
    }

    /// Emitted rows for one state in catalog/choice order, including repeated destinations.
    pub fn rows(
        &self,
        state: StateId,
    ) -> std::result::Result<impl Iterator<Item = &Transition<M>>, Refusal> {
        if self.visited.get(state).is_none() {
            return Err(Refusal::State { given: state.0 });
        }
        let start = self.transitions.partition_point(|edge| edge.source.0 < state.0);
        let end = start + self.transitions[start..].partition_point(|edge| edge.source == state);
        Ok(self.transitions[start..end].iter())
    }

    /// The path by which exploration first reached an admitted state; no admitted path to it is
    /// shorter.
    pub fn path_to(
        &self,
        state: StateId,
    ) -> std::result::Result<Path<M::State, M::Action>, Refusal> {
        if self.visited.get(state).is_none() {
            return Err(Refusal::State { given: state.0 });
        }
        Ok(self.path(state))
    }

    /// Returns claim results and diagnostics without re-running model code.
    pub fn verify_all(&self) -> std::result::Result<Report<M>, Refusal> {
        Ok(Report {
            counts: self.counts,
            claims: self.claims.clone(),
            warnings: self.warnings.clone(),
        })
    }

    /// The aggregate claim verdict for this exploration.
    pub fn outcome(&self) -> CheckOutcome {
        aggregate(self.counts.cut, &self.claims)
    }

    pub(crate) fn state(&self, id: StateId) -> &M::State {
        match self.visited.get(id) {
            Some(state) => state,
            None => unreachable!("path ids are admitted states"),
        }
    }

    pub(crate) fn path(&self, mut id: StateId) -> Path<M::State, M::Action> {
        let mut steps = Vec::new();
        while let Some((parent, action, choice)) = &self.parents[id.0] {
            steps.push(PathStep {
                action: action.clone(),
                choice: *choice,
                after: self.state(id).clone(),
            });
            id = *parent;
        }
        steps.reverse();
        Path { start: self.state(id).clone(), steps, cycle_start: None, failure: None }
    }

    fn check_state(&mut self, world: &M, state: StateId, invariants: &[Claim<M>]) {
        for (index, claim) in invariants.iter().enumerate() {
            if self.claims[index].outcome == CheckOutcome::Refuted {
                continue;
            }
            let verdict =
                catch_unwind(AssertUnwindSafe(|| claim.evaluate(world, self.state(state))));
            self.record(index, verdict, |result| result.path(state));
        }
    }

    fn check_edge(
        &mut self,
        world: &M,
        edge: &Transition<M>,
        offset: usize,
        postconditions: &[StepClaim<M>],
        triggered: &mut FixedBitSet,
    ) {
        for (index, claim) in postconditions.iter().enumerate() {
            if !claim.applies_to(&edge.action) {
                continue;
            }
            triggered.insert(index);
            if self.claims[offset + index].outcome == CheckOutcome::Refuted {
                continue;
            }
            let verdict = catch_unwind(AssertUnwindSafe(|| {
                claim.evaluate(world, self.state(edge.source), &edge.action, &edge.emitted)
            }));
            self.record(offset + index, verdict, |result| {
                let mut path = result.path(edge.source);
                path.steps.push(PathStep {
                    action: edge.action.clone(),
                    choice: edge.choice,
                    after: edge.emitted.2.clone(),
                });
                path
            });
        }
    }

    fn record(
        &mut self,
        index: usize,
        verdict: std::result::Result<Evaluation, Box<dyn Any + Send>>,
        make_path: impl FnOnce(&Self) -> Path<M::State, M::Action>,
    ) {
        let mut path = if matches!(verdict, Ok(Evaluation { verdict: Verdict::Fails, .. }) | Err(_))
        {
            Some(make_path(self))
        } else {
            None
        };
        let claim = &mut self.claims[index];
        match verdict {
            Ok(Evaluation { verdict: Verdict::Holds, .. }) => {}
            Ok(Evaluation { verdict: Verdict::Unknown(hole), .. }) => {
                claim.outcome = CheckOutcome::Unknown;
                if !claim.holes.contains(&hole.reason) {
                    claim.holes.push(hole.reason);
                }
            }
            Ok(Evaluation { verdict: Verdict::Fails, failure }) => {
                if let (Some(path), Some(expression)) = (&mut path, failure) {
                    let state = path.steps.last().map_or(&path.start, |step| &step.after);
                    let state_text = state.to_string();
                    path.failure = Some(format!("check!({expression}) failed at {state_text}"));
                }
                claim.outcome = CheckOutcome::Refuted;
                claim.path = path;
            }
            Err(error) => {
                if let Some(path) = &mut path {
                    path.failure = Some(format!(
                        "claim {} panicked: {}",
                        claim.name,
                        panic_message(error.as_ref())
                    ));
                }
                claim.outcome = CheckOutcome::Refuted;
                claim.path = path;
            }
        }
    }
}

// Kept after every panic site in this file so adding it leaves linked guest bytes unchanged.
impl Checker {
    /// Checks an additional state predicate from the given ordered starts in place of the
    /// world's own, with the default million-state budget.
    pub fn refuted_from<M: Machine>(
        world: &M,
        starts: Vec<M::State>,
        claim: Claim<M>,
    ) -> std::result::Result<Result<M>, Refusal> {
        let mut invariants = world.invariants();
        let supplemental = invariants.len();
        invariants.push(claim);
        Self::explore_claims(
            world,
            starts,
            Self::BUDGET,
            Self::CATALOG_LIMIT,
            &invariants,
            Some(supplemental),
        )
    }
}
