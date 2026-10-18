use std::{
    any::Any,
    panic::{AssertUnwindSafe, catch_unwind},
};

use fixedbitset::FixedBitSet;
use framework_dsl::{Claim, Machine, Outcome, Step, StepClaim, Verdict};

use crate::{ClaimResult, FullStates, Path, PathStep, Report, StateId, Visited, report::aggregate};

/// Work observed during an exploration; disabled actions emit no edge.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Counts {
    /// Distinct initial states admitted before the budget was exhausted.
    pub starts: usize,
    /// Distinct admitted states, shared across all starts.
    pub reached: usize,
    /// Emitted choices, including rejecting outcomes and repeated destinations.
    pub emitted: usize,
    /// Calls to `Machine::step`, including disabled actions and caught panics.
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

/// A bounded graph with safety verdicts, ordered rows and predecessor paths.
pub struct Result<M: Machine> {
    /// Statistics for the work actually performed.
    pub counts: Counts,
    visited: FullStates<M::State>,
    parents: Vec<Option<(StateId, M::Action, usize)>>,
    depths: Vec<usize>,
    transitions: Vec<Transition<M>>,
    claims: Vec<ClaimResult<M::State, M::Action>>,
    warnings: Vec<String>,
}

impl Checker {
    /// Explores ordered starts with a shared visited set and FIFO action/choice expansion.
    ///
    /// # Panics
    /// Panics if `budget` is zero. The budget limits admitted states, not edge evaluations.
    pub fn explore<M: Machine>(world: &M, budget: usize) -> Result<M> {
        Self::explore_claims(world, budget, &world.invariants())
    }

    /// Checks an additional state predicate with the default million-state budget.
    pub fn refuted<M: Machine>(world: &M, claim: Claim<M>) -> Result<M> {
        let mut invariants = world.invariants();
        invariants.push(claim);
        Self::explore_claims(world, 1 << 20, &invariants)
    }

    fn explore_claims<M: Machine>(world: &M, budget: usize, invariants: &[Claim<M>]) -> Result<M> {
        assert!(budget > 0, "state budget must be at least one");
        let postconditions = world.postconditions();
        let actions = world.actions();
        let mut result = Result {
            counts: Counts::default(),
            visited: FullStates::default(),
            parents: Vec::new(),
            depths: Vec::new(),
            transitions: Vec::new(),
            claims: invariants
                .iter()
                .map(|claim| unchecked(claim.name))
                .chain(postconditions.iter().map(|claim| unchecked(claim.name)))
                .collect(),
            warnings: Vec::new(),
        };
        result.seed(world, budget, invariants);
        let mut fired = FixedBitSet::with_capacity(actions.len());
        let mut triggered = FixedBitSet::with_capacity(postconditions.len());
        let mut cursor = 0;
        'search: while !result.counts.cut && cursor < result.visited.len() {
            let source = StateId(cursor);
            let Some(before) = result.visited.get(source).cloned() else {
                unreachable!("BFS cursor has an admitted state")
            };
            for (action_index, action) in actions.iter().enumerate() {
                result.counts.evaluated += 1;
                let emitted =
                    catch_unwind(AssertUnwindSafe(|| world.step(&before, action.clone())));
                let choices = match emitted {
                    Ok(choices) => choices,
                    Err(error) => {
                        let mut path = result.path(source);
                        path.failure = Some(format!(
                            "action {action:?} panicked: {}",
                            panic_message(error.as_ref())
                        ));
                        result.claims.push(ClaimResult {
                            name: format!("step_{cursor}_{action_index}"),
                            outcome: Outcome::Refuted,
                            path: Some(path),
                            holes: Vec::new(),
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
            }
            cursor += 1;
        }
        result.finish(&actions, &postconditions, &fired, &triggered);
        result
    }
}

fn unchecked<S, A>(name: &str) -> ClaimResult<S, A> {
    ClaimResult { name: name.into(), outcome: Outcome::Verified, path: None, holes: Vec::new() }
}

fn panic_message(error: &(dyn Any + Send)) -> &str {
    error
        .downcast_ref::<&str>()
        .copied()
        .or_else(|| error.downcast_ref::<String>().map(String::as_str))
        .unwrap_or("non-string panic payload")
}

impl<M: Machine> Result<M> {
    fn seed(&mut self, world: &M, budget: usize, invariants: &[Claim<M>]) {
        for start in world.starts() {
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
        self.counts.reached = self.visited.len();
        if self.counts.cut {
            for claim in &mut self.claims {
                if claim.outcome != Outcome::Refuted {
                    claim.outcome = Outcome::Unknown;
                }
            }
        }
        for (index, action) in actions.iter().enumerate() {
            if !fired.contains(index) {
                self.warnings.push(format!("action {action:?} never fired"));
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
    pub fn rows(&self, state: StateId) -> impl Iterator<Item = &Transition<M>> {
        self.transitions.iter().filter(move |edge| edge.source == state)
    }

    /// Returns safety/step claims and diagnostics without re-running model code.
    pub fn verify_all(&self) -> Report<M> {
        Report { counts: self.counts, claims: self.claims.clone(), warnings: self.warnings.clone() }
    }

    /// The aggregate safety verdict for this exploration.
    pub fn outcome(&self) -> Outcome {
        aggregate(self.counts.cut, &self.claims)
    }

    fn state(&self, id: StateId) -> &M::State {
        match self.visited.get(id) {
            Some(state) => state,
            None => unreachable!("path ids are admitted states"),
        }
    }

    fn path(&self, mut id: StateId) -> Path<M::State, M::Action> {
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
        Path { start: self.state(id).clone(), steps, failure: None }
    }

    fn check_state(&mut self, world: &M, state: StateId, invariants: &[Claim<M>]) {
        for (index, claim) in invariants.iter().enumerate() {
            if self.claims[index].outcome == Outcome::Refuted {
                continue;
            }
            let verdict =
                catch_unwind(AssertUnwindSafe(|| (claim.predicate)(world, self.state(state))));
            self.record(index, verdict, self.path(state));
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
            if self.claims[offset + index].outcome == Outcome::Refuted {
                continue;
            }
            let verdict = catch_unwind(AssertUnwindSafe(|| {
                claim.holds(world, self.state(edge.source), &edge.action, &edge.emitted)
            }));
            let mut path = self.path(edge.source);
            path.steps.push(PathStep {
                action: edge.action.clone(),
                choice: edge.choice,
                after: edge.emitted.2.clone(),
            });
            self.record(offset + index, verdict, path);
        }
    }

    fn record(
        &mut self,
        index: usize,
        verdict: std::result::Result<Verdict, Box<dyn Any + Send>>,
        mut path: Path<M::State, M::Action>,
    ) {
        let claim = &mut self.claims[index];
        match verdict {
            Ok(Verdict::Holds) => {}
            Ok(Verdict::Unknown(hole)) => {
                claim.outcome = Outcome::Unknown;
                if !claim.holes.contains(&hole.reason) {
                    claim.holes.push(hole.reason);
                }
            }
            Ok(Verdict::Fails) => {
                claim.outcome = Outcome::Refuted;
                claim.path = Some(path);
            }
            Err(error) => {
                path.failure = Some(format!(
                    "claim {} panicked: {}",
                    claim.name,
                    panic_message(error.as_ref())
                ));
                claim.outcome = Outcome::Refuted;
                claim.path = Some(path);
            }
        }
    }
}
