//! The shortest path through a pattern over one budgeted Model exploration: explored rows,
//! typed pattern nodes and a label search that keeps every non-dominated arrival.

use std::{
    collections::{HashMap, VecDeque},
    panic::{AssertUnwindSafe, catch_unwind},
};

use umpire_checker::{Checker, Path, PathStep, Report, StateId, Transition};
use umpire_dsl::{
    ActionCatalog, Alternative, CatalogError, CheckOutcome, Claim, Gap, Machine, PathNode,
    StepError, Steps, Verdict, Waypoint,
};

use crate::Refusal;

/// A path as the checker's `paths/` goldens spell it.
pub(crate) type Trace<M> = Path<<M as Machine>::State, <M as Machine>::Action>;

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

const SEARCH_LIMIT: usize = 1_000_000;

/// Where a pattern's first node may match.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Anchor {
    /// At an explored start state.
    Starts,
    /// At any explored state, reached by the shortest prefix from a start.
    Anywhere,
}

/// A path through the explored rows and, per step, the row it took: (source, ordinal).
pub(crate) struct Witness<M: Machine> {
    pub(crate) trace: Trace<M>,
    pub(crate) rows: Vec<(StateId, usize)>,
}

/// One exploration's retained rows, searched for typed patterns. A cut exploration searches
/// only below its frontier: the shortest depth at which a row's destination was not admitted.
pub(crate) struct Graph<'a, M: Machine> {
    explored: &'a umpire_checker::Result<M>,
    states: Vec<&'a M::State>,
    rows: Vec<Vec<&'a Transition<M>>>,
    frontier: usize,
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq)]
struct Position {
    state: StateId,
    next: usize,
    age: u32,
    depth: usize,
}

struct Label {
    position: Position,
    distance: usize,
    parent: Option<(usize, Option<usize>)>,
}

const EMPTY: usize = usize::MAX;

/// The non-dominated label ids per (state, node) cell: one dense slot for the usual single
/// label, spilling to a map only while a bounded gap keeps several ages alive.
struct Nondominated {
    one: Vec<usize>,
    many: HashMap<usize, Vec<usize>>,
}

impl Nondominated {
    fn new(cells: usize) -> Self {
        Self { one: vec![EMPTY; cells], many: HashMap::new() }
    }

    fn spilled(&self, cell: usize) -> Option<&Vec<usize>> {
        if self.many.is_empty() { None } else { self.many.get(&cell) }
    }

    /// Whether any kept label satisfies `dominates`.
    fn any(&self, cell: usize, dominates: impl Fn(usize) -> bool) -> bool {
        let one = self.one[cell];
        one != EMPTY
            && (dominates(one)
                || self.spilled(cell).is_some_and(|ids| ids.iter().any(|id| dominates(*id))))
    }

    /// Adds `id`, dropping every label it dominates.
    fn insert(&mut self, cell: usize, id: usize, dominated: impl Fn(usize) -> bool) {
        let one = self.one[cell];
        let Some(more) = self.spilled(cell) else {
            if one == EMPTY || dominated(one) {
                self.one[cell] = id;
            } else {
                self.many.insert(cell, vec![id]);
            }
            return;
        };
        let mut kept: Vec<_> = std::iter::once(one)
            .chain(more.iter().copied())
            .filter(|old| !dominated(*old))
            .collect();
        kept.push(id);
        self.one[cell] = kept.remove(0);
        if kept.is_empty() {
            self.many.remove(&cell);
        } else {
            self.many.insert(cell, kept);
        }
    }
}

struct Arc {
    target: usize,
    row: Option<usize>,
}

pub(crate) use umpire_checker::SplitMix64 as Rng;

impl<'a, M: Machine> Graph<'a, M> {
    pub(crate) fn new(explored: &'a umpire_checker::Result<M>) -> Result<Self, Refusal> {
        let states: Vec<_> = explored.states().map(|(_, state)| state).collect();
        let rows: Vec<Vec<_>> = explored
            .states()
            .map(|(id, _)| explored.rows(id).map(Iterator::collect))
            .collect::<Result<_, _>>()?;
        let mut frontier = usize::MAX;
        if explored.counts.cut {
            frontier = 0;
            for edges in &rows {
                if let Some(row) = edges.iter().find(|row| row.target.is_none()) {
                    let depth = explored.path_to(row.source)?.steps.len() + 1;
                    frontier = if frontier == 0 { depth } else { frontier.min(depth) };
                }
            }
        }
        Ok(Self { explored, states, rows, frontier })
    }

    fn nodes(
        alternative: &Alternative<M::State, M::Action, M::Outcome>,
        anchor: Anchor,
    ) -> Vec<PathNode<M::State, M::Action, M::Outcome>> {
        let prefix = anchor == Anchor::Anywhere;
        let mut nodes = Vec::with_capacity(alternative.nodes.len() + usize::from(prefix));
        if prefix {
            nodes.push(PathNode::Gap { gap: Gap::Any, exclude: None });
        }
        nodes.extend(alternative.nodes.iter().cloned());
        nodes
    }

    fn judge(
        &self,
        index: usize,
        state: StateId,
        predicate: impl FnOnce() -> bool,
    ) -> Result<bool, Refusal> {
        catch_unwind(AssertUnwindSafe(predicate)).map_err(|error| Refusal::Predicate {
            waypoint: index,
            state: self.states[state.0].to_string(),
            failure: umpire_checker::panic_message(error.as_ref()).into(),
        })
    }

    fn ordinals(nodes: &[PathNode<M::State, M::Action, M::Outcome>]) -> Vec<usize> {
        let mut ordinal = 0;
        std::iter::once(0)
            .chain(nodes.iter().map(|node| {
                ordinal += usize::from(matches!(node, PathNode::State(_)));
                ordinal
            }))
            .collect()
    }

    fn successors(
        &self,
        nodes: &[PathNode<M::State, M::Action, M::Outcome>],
        ordinals: &[usize],
        position: Position,
        edges: bool,
        next: &mut Vec<(Position, Option<usize>)>,
    ) -> Result<(), Refusal> {
        next.clear();
        let Some(node) = nodes.get(position.next) else { return Ok(()) };
        let advance = Position { next: position.next + 1, age: 0, ..position };
        let waypoint = ordinals[position.next];
        match node {
            PathNode::State(state) => {
                if self.judge(waypoint, position.state, || {
                    state.judge(self.states[position.state.0])
                })? {
                    next.push((advance, None));
                }
            }
            PathNode::Action { action, outcome, .. } if edges => {
                for (ordinal, row) in self.rows[position.state.0].iter().enumerate() {
                    let Some(state) = row.target else { continue };
                    if row.action == **action
                        && outcome.as_ref().map_or(Ok(true), |predicate| {
                            self.judge(waypoint, position.state, || predicate(&row.emitted.0))
                        })?
                    {
                        next.push((Position { state, ..advance }, Some(ordinal)));
                    }
                }
            }
            PathNode::Gap { gap, exclude } => {
                next.push((advance, None));
                if edges && gap.bound().is_none_or(|bound| position.age < bound) {
                    for (ordinal, row) in self.rows[position.state.0].iter().enumerate() {
                        let Some(state) = row.target else { continue };
                        if !exclude.as_ref().map_or(Ok(false), |predicate| {
                            self.judge(waypoint, position.state, || predicate(&row.action))
                        })? {
                            let age = if gap.bound().is_some() { position.age + 1 } else { 0 };
                            next.push((Position { state, age, ..position }, Some(ordinal)));
                        }
                    }
                }
            }
            PathNode::Action { .. } => {}
        }
        if next.len() > SEARCH_LIMIT {
            return Err(Refusal::SearchLimit { limit: SEARCH_LIMIT });
        }
        Ok(())
    }

    /// The shortest path matching `alternative` from `anchor`. Equal-length choices are
    /// deterministic, generally by start, catalog row, then choice order. A refusal names the
    /// first unmatched waypoint and the state that matched the one before it; whether a cut
    /// leaves that answer open is `cut`.
    ///
    /// Labels are capped by the graph, not a constant: without bounded gaps every
    /// (state, node) cell keeps at most a first arrival and one shorter replacement, so twice
    /// the cells always suffices; only bounded gaps' ages can need more.
    pub(crate) fn shortest(
        &self,
        alternative: &Alternative<M::State, M::Action, M::Outcome>,
        anchor: Anchor,
    ) -> Result<Witness<M>, Refusal> {
        let nodes = Self::nodes(alternative, anchor);
        let ordinals = Self::ordinals(&nodes);
        let width = nodes.len() + 1;
        let cells = self.states.len().checked_mul(width);
        let limit = cells
            .and_then(|cells| cells.checked_mul(2))
            .ok_or(Refusal::SearchLimit { limit: SEARCH_LIMIT })?
            .max(SEARCH_LIMIT);
        let mut best = Nondominated::new(cells.unwrap_or_default());
        let mut labels: Vec<Label> = Vec::new();
        let mut queue = VecDeque::new();
        for state in 0..self.explored.counts.starts {
            let position = Position { state: StateId(state), next: 0, age: 0, depth: 0 };
            best.insert(state * width, labels.len(), |_| false);
            queue.push_back(labels.len());
            labels.push(Label { position, distance: 0, parent: None });
        }
        let mut furthest = None;
        let mut successors = Vec::new();
        while let Some(id) = queue.pop_front() {
            let position = labels[id].position;
            let distance = labels[id].distance;
            if furthest.is_none_or(|old: usize| labels[old].position.next < position.next) {
                furthest = Some(id);
            }
            if position.next == nodes.len() {
                return Ok(self.witness(&labels, id));
            }
            self.successors(
                &nodes,
                &ordinals,
                position,
                distance < self.frontier,
                &mut successors,
            )?;
            for &(next, row) in &successors {
                let next_distance = distance + usize::from(row.is_some());
                let cell = next.state.0 * width + next.next;
                if best.any(cell, |old| {
                    labels[old].distance <= next_distance && labels[old].position.age <= next.age
                }) {
                    continue;
                }
                if labels.len() >= limit {
                    return Err(Refusal::SearchLimit { limit });
                }
                let next_id = labels.len();
                best.insert(cell, next_id, |old| {
                    next_distance <= labels[old].distance && next.age <= labels[old].position.age
                });
                labels.push(Label {
                    position: next,
                    distance: next_distance,
                    parent: Some((id, row)),
                });
                if row.is_some() {
                    queue.push_back(next_id);
                } else {
                    queue.push_front(next_id);
                }
            }
        }
        let reached = furthest.map_or(0, |id| labels[id].position.next);
        let waypoint = ordinals[reached];
        // A cut may have dropped starts, so a cut search names only the admitted ones.
        let cut = self.explored.counts.cut;
        let state = match (waypoint, furthest, self.explored.counts.starts) {
            (_, None, _) | (0, _, 0) => "no start state".into(),
            (0, _, 1) if cut => format!("the admitted start {}", self.states[0]),
            (0, _, 1) => self.states[0].to_string(),
            (0, _, starts) if cut => format!("the {starts} admitted starts"),
            (0, _, starts) => format!("the space's {starts} starts"),
            (_, Some(id), _) => self.states[labels[id].position.state.0].to_string(),
        };
        Err(Refusal::Unbridged { waypoint, state, cut: cut && !self.settled(alternative) })
    }

    fn settled(&self, alternative: &Alternative<M::State, M::Action, M::Outcome>) -> bool {
        let Some(PathNode::State(anchor)) = alternative.nodes.first() else { return false };
        let Waypoint::Is(anchor) = anchor.as_ref() else { return false };
        let Some(span) = alternative.nodes.iter().try_fold(0usize, |span, node| {
            span.checked_add(match node {
                PathNode::Action { .. } => 1,
                PathNode::Gap { gap, .. } => gap.bound()? as usize,
                PathNode::State(_) => 0,
            })
        }) else {
            return false;
        };
        let Some((id, _)) = self.explored.states().find(|(_, state)| *state == anchor) else {
            return false;
        };
        self.explored.path_to(id).is_ok_and(|path| {
            path.steps.len().checked_add(span).is_some_and(|length| length < self.frontier)
        })
    }

    fn witness(&self, labels: &[Label], mut id: usize) -> Witness<M> {
        let mut rows = Vec::new();
        let mut steps = Vec::new();
        while let Some((parent, row)) = labels[id].parent {
            if let Some(ordinal) = row {
                let source = labels[parent].position.state;
                let edge = self.rows[source.0][ordinal];
                rows.push((source, ordinal));
                steps.push(PathStep {
                    action: edge.action.clone(),
                    choice: edge.choice,
                    after: self.states[labels[id].position.state.0].clone(),
                });
            }
            id = parent;
        }
        rows.reverse();
        steps.reverse();
        Witness {
            trace: Path {
                start: self.states[labels[id].position.state.0].clone(),
                steps,
                cycle_start: None,
                failure: None,
            },
            rows,
        }
    }

    pub(crate) fn walk(
        &self,
        alternative: &Alternative<M::State, M::Action, M::Outcome>,
        anchor: Anchor,
        horizon: usize,
        rng: &mut Rng,
    ) -> Result<Witness<M>, Refusal> {
        let nodes = Self::nodes(alternative, anchor);
        let ordinals = Self::ordinals(&nodes);
        let mut positions = Vec::new();
        let mut index = HashMap::new();
        let mut arcs: Vec<Vec<Arc>> = Vec::new();
        let mut reverse: Vec<Vec<usize>> = Vec::new();
        for state in 0..self.explored.counts.starts {
            if positions.len() >= SEARCH_LIMIT {
                return Err(Refusal::SearchLimit { limit: SEARCH_LIMIT });
            }
            let position = Position { state: StateId(state), next: 0, age: 0, depth: 0 };
            index.insert(position, positions.len());
            positions.push(position);
            arcs.push(Vec::new());
            reverse.push(Vec::new());
        }
        let starts = positions.len();
        let mut work = positions.len();
        let mut cursor = 0;
        let mut successors = Vec::new();
        while cursor < positions.len() {
            let position = positions[cursor];
            self.successors(
                &nodes,
                &ordinals,
                position,
                position.depth < horizon.min(self.frontier),
                &mut successors,
            )?;
            for &(mut next, row) in &successors {
                next.depth += usize::from(row.is_some());
                work += 1;
                if work >= SEARCH_LIMIT {
                    return Err(Refusal::SearchLimit { limit: SEARCH_LIMIT });
                }
                let target = if let Some(id) = index.get(&next) {
                    *id
                } else {
                    work += 1;
                    let id = positions.len();
                    index.insert(next, id);
                    positions.push(next);
                    arcs.push(Vec::new());
                    reverse.push(Vec::new());
                    id
                };
                arcs[cursor].push(Arc { target, row });
                reverse[target].push(cursor);
            }
            cursor += 1;
        }
        let mut viable = vec![false; positions.len()];
        let mut queue = VecDeque::new();
        for (id, position) in positions.iter().enumerate() {
            if position.next == nodes.len() {
                viable[id] = true;
                queue.push_back(id);
            }
        }
        while let Some(id) = queue.pop_front() {
            for &parent in &reverse[id] {
                if !viable[parent] {
                    viable[parent] = true;
                    queue.push_back(parent);
                }
            }
        }
        let starts: Vec<_> = (0..starts).filter(|id| viable[*id]).collect();
        if starts.is_empty() {
            return Err(Refusal::WalkBound { bound: horizon });
        }
        let mut id = starts[rng.index(starts.len())];
        let mut labels = vec![Label { position: positions[id], distance: 0, parent: None }];
        while positions[id].next < nodes.len() {
            let choices: Vec<_> = arcs[id].iter().filter(|arc| viable[arc.target]).collect();
            let chosen = choices[rng.index(choices.len())];
            id = chosen.target;
            labels.push(Label {
                position: positions[id],
                distance: positions[id].depth,
                parent: Some((labels.len() - 1, chosen.row)),
            });
        }
        Ok(self.witness(&labels, labels.len() - 1))
    }
}

#[cfg(test)]
mod tests {
    use std::{collections::HashSet, rc::Rc};

    use umpire_checker::Checker;
    use umpire_dsl::{Alternative, Gap, Machine, PathNode, Steps, Waypoint};

    use super::{Anchor, Graph, Rng};

    #[derive(Debug)]
    struct Outcome(u8);

    struct World;

    impl Machine for World {
        type State = u8;
        type Action = u8;
        type Outcome = Outcome;
        type Fact = ();

        fn starts(&self) -> Vec<u8> {
            vec![0]
        }
        fn actions(&self) -> Vec<u8> {
            vec![0, 1, 2]
        }
        fn internal(&self, _: &u8) -> bool {
            false
        }
        fn step(&self, state: &u8, action: u8) -> Steps<u8, Outcome, ()> {
            match (*state, action) {
                (0, 0) => [(Outcome(0), vec![], 0)].into_iter().collect(),
                (0, 1) => [(Outcome(0), vec![], 1), (Outcome(1), vec![], 1)].into_iter().collect(),
                (0, 2) => [(Outcome(0), vec![], 2)].into_iter().collect(),
                (2, 1) => [(Outcome(0), vec![], 1)].into_iter().collect(),
                _ => Steps::new(),
            }
        }
    }

    type Node = PathNode<u8, u8, Outcome>;

    fn state(value: u8) -> Node {
        PathNode::State(Rc::new(Waypoint::Is(value)))
    }
    fn action(value: u8) -> Node {
        PathNode::Action { action: Rc::new(value), outcome: None, label: value.to_string() }
    }
    fn pattern(nodes: Vec<Node>) -> Alternative<u8, u8, Outcome> {
        Alternative { nodes, choices: vec![] }
    }

    #[test]
    fn exact_outcome_retains_the_selected_row() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        let request = pattern(vec![
            state(0),
            PathNode::Action {
                action: Rc::new(1),
                outcome: Some(Rc::new(|outcome| outcome.0 == 1)),
                label: "1 => 1".into(),
            },
            state(1),
        ]);
        let witness = graph.shortest(&request, Anchor::Starts)?;
        assert_eq!(witness.trace.steps.len(), 1);
        assert_eq!(witness.trace.steps[0].choice, 1);
        assert_eq!(witness.rows, vec![(umpire_checker::StateId(0), 2)]);
        Ok(())
    }

    #[test]
    fn only_explicit_gaps_consume_intervening_edges() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        assert!(graph.shortest(&pattern(vec![state(0), state(1)]), Anchor::Starts).is_err());
        let empty = graph.shortest(
            &pattern(vec![
                state(0),
                PathNode::Gap { gap: Gap::AtMost(0), exclude: None },
                state(0),
            ]),
            Anchor::Starts,
        )?;
        assert!(empty.trace.steps.is_empty());
        let excluded = pattern(vec![
            state(0),
            PathNode::Gap { gap: Gap::Any, exclude: Some(Rc::new(|action| *action == 1)) },
            action(1),
            state(1),
        ]);
        assert_eq!(graph.shortest(&excluded, Anchor::Starts)?.trace.steps.len(), 1);
        Ok(())
    }

    #[test]
    fn finds_a_global_shortest_match_and_searches_before_the_first_node()
    -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        let request =
            pattern(vec![PathNode::Gap { gap: Gap::Any, exclude: None }, action(1), state(1)]);
        assert_eq!(graph.shortest(&request, Anchor::Starts)?.trace.steps.len(), 1);
        assert!(graph.shortest(&pattern(vec![state(2)]), Anchor::Starts).is_err());
        assert_eq!(
            graph.shortest(&pattern(vec![state(2)]), Anchor::Anywhere)?.trace.steps[0].action,
            2
        );
        Ok(())
    }

    #[test]
    fn bounded_gap_loops_do_not_spend_work_once_their_labels_are_dominated()
    -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        let request = pattern(vec![
            state(0),
            PathNode::Gap {
                gap: Gap::AtMost(1_000_000_000),
                exclude: Some(Rc::new(|action| *action != 0)),
            },
            state(9),
        ]);
        assert!(matches!(
            graph.shortest(&request, Anchor::Starts),
            Err(crate::Refusal::Unbridged { cut: false, .. })
        ));
        Ok(())
    }

    #[test]
    fn seeded_walks_retain_loops_and_refuse_too_short_a_horizon() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        let request = pattern(vec![
            state(0),
            PathNode::Gap { gap: Gap::Any, exclude: None },
            action(1),
            state(1),
        ]);
        let mut found_loop = false;
        let mut experiments = HashSet::new();
        for seed in 0..32 {
            let first = graph.walk(&request, Anchor::Starts, 5, &mut Rng::new(seed))?;
            let second = graph.walk(&request, Anchor::Starts, 5, &mut Rng::new(seed))?;
            assert_eq!(first.rows, second.rows);
            assert!(first.trace.steps.len() <= 5);
            let mut state = first.trace.start;
            assert_eq!(state, 0);
            for step in &first.trace.steps {
                let successors = World.step(&state, step.action);
                let (_, _, after) = &successors[step.choice];
                assert_eq!(*after, step.after);
                state = *after;
            }
            assert_eq!(state, 1);
            assert_eq!(first.trace.steps.last().map(|step| step.action), Some(1));
            found_loop |= first.trace.steps.iter().any(|step| step.action == 0);
            experiments.insert(first.rows);
        }
        assert!(found_loop);
        assert!(experiments.len() > 1);
        assert!(matches!(
            graph.walk(&request, Anchor::Starts, 0, &mut Rng::new(1)),
            Err(crate::Refusal::WalkBound { bound: 0 })
        ));
        Ok(())
    }

    #[test]
    fn cut_graph_answers_only_settled_searches() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 2)?;
        let graph = Graph::new(&explored)?;
        assert_eq!(
            graph.shortest(&pattern(vec![state(0), action(1)]), Anchor::Starts)?.trace.steps.len(),
            1
        );
        assert!(matches!(
            graph.shortest(&pattern(vec![state(0), action(2)]), Anchor::Starts),
            Err(crate::Refusal::Unbridged { cut: true, .. })
        ));
        assert!(matches!(
            graph.shortest(
                &pattern(vec![
                    state(0),
                    PathNode::Gap { gap: Gap::AtMost(0), exclude: None },
                    state(2)
                ]),
                Anchor::Anywhere
            ),
            Err(crate::Refusal::Unbridged { cut: false, .. })
        ));
        Ok(())
    }
}
