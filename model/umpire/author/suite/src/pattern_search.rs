//! Typed pattern matching over retained checker rows.

use std::{
    collections::{HashMap, VecDeque},
    panic::{AssertUnwindSafe, catch_unwind},
};

use umpire_checker::{Path, PathStep, StateId, Transition};
use umpire_dsl::{Alternative, Gap, Machine, PathNode, Waypoint};

use crate::Refusal;

const SEARCH_LIMIT: usize = 1_000_000;

pub(crate) struct Witness<M: Machine> {
    pub(crate) trace: Path<M::State, M::Action>,
    pub(crate) rows: Vec<(StateId, usize)>,
}

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

struct Arc {
    target: usize,
    row: Option<usize>,
}

pub(crate) struct Rng(u64);

impl Rng {
    pub(crate) const fn new(seed: u64) -> Self {
        Self(seed)
    }

    pub(crate) fn index(&mut self, len: usize) -> usize {
        assert!(len > 0);
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut value = self.0;
        value = (value ^ (value >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        value = (value ^ (value >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        value ^= value >> 31;
        usize::try_from(value % u64::try_from(len).unwrap_or(u64::MAX)).unwrap_or(0)
    }
}

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
        prefix: bool,
    ) -> Vec<PathNode<M::State, M::Action, M::Outcome>> {
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
            failure: error
                .downcast_ref::<&str>()
                .copied()
                .or_else(|| error.downcast_ref::<String>().map(String::as_str))
                .unwrap_or("non-string panic payload")
                .into(),
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
    ) -> Result<Vec<(Position, Option<usize>)>, Refusal> {
        let Some(node) = nodes.get(position.next) else { return Ok(Vec::new()) };
        let mut next = Vec::new();
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
                        && self.judge(waypoint, position.state, || {
                            outcome.as_ref().is_none_or(|predicate| predicate(&row.emitted.0))
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
                        if !self.judge(waypoint, position.state, || {
                            exclude.as_ref().is_some_and(|predicate| predicate(&row.action))
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
        Ok(next)
    }

    pub(crate) fn shortest(
        &self,
        alternative: &Alternative<M::State, M::Action, M::Outcome>,
        prefix: bool,
    ) -> Result<Witness<M>, Refusal> {
        let nodes = Self::nodes(alternative, prefix);
        let ordinals = Self::ordinals(&nodes);
        let mut labels: Vec<Label> = Vec::new();
        let mut best: HashMap<(StateId, usize), Vec<usize>> = HashMap::new();
        let mut queue = VecDeque::new();
        for state in 0..self.explored.counts.starts {
            if labels.len() >= SEARCH_LIMIT {
                return Err(Refusal::SearchLimit { limit: SEARCH_LIMIT });
            }
            let position = Position { state: StateId(state), next: 0, age: 0, depth: 0 };
            best.insert((position.state, position.next), vec![labels.len()]);
            queue.push_back(labels.len());
            labels.push(Label { position, distance: 0, parent: None });
        }
        let mut furthest = None;
        while let Some(id) = queue.pop_front() {
            let position = labels[id].position;
            let distance = labels[id].distance;
            if furthest.is_none_or(|old: usize| labels[old].position.next < position.next) {
                furthest = Some(id);
            }
            if position.next == nodes.len() {
                return Ok(self.witness(&labels, id));
            }
            for (next, row) in
                self.successors(&nodes, &ordinals, position, distance < self.frontier)?
            {
                let next_distance = distance + usize::from(row.is_some());
                let nondominated = best.entry((next.state, next.next)).or_default();
                if nondominated.iter().any(|old| {
                    labels[*old].distance <= next_distance && labels[*old].position.age <= next.age
                }) {
                    continue;
                }
                if labels.len() >= SEARCH_LIMIT {
                    return Err(Refusal::SearchLimit { limit: SEARCH_LIMIT });
                }
                let next_id = labels.len();
                nondominated.retain(|old| {
                    !(next_distance <= labels[*old].distance
                        && next.age <= labels[*old].position.age)
                });
                nondominated.push(next_id);
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
        let state = furthest.map_or_else(
            || "no start state".into(),
            |id| self.states[labels[id].position.state.0].to_string(),
        );
        Err(Refusal::Unbridged {
            waypoint,
            state,
            cut: self.explored.counts.cut && !self.settled(alternative),
        })
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
        prefix: bool,
        horizon: usize,
        rng: &mut Rng,
    ) -> Result<Witness<M>, Refusal> {
        let nodes = Self::nodes(alternative, prefix);
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
        while cursor < positions.len() {
            let position = positions[cursor];
            for (mut next, row) in self.successors(
                &nodes,
                &ordinals,
                position,
                position.depth < horizon.min(self.frontier),
            )? {
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

    use super::{Graph, Rng};

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
        let witness = graph.shortest(&request, false)?;
        assert_eq!(witness.trace.steps.len(), 1);
        assert_eq!(witness.trace.steps[0].choice, 1);
        assert_eq!(witness.rows, vec![(umpire_checker::StateId(0), 2)]);
        Ok(())
    }

    #[test]
    fn only_explicit_gaps_consume_intervening_edges() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        assert!(graph.shortest(&pattern(vec![state(0), state(1)]), false).is_err());
        let empty = graph.shortest(
            &pattern(vec![
                state(0),
                PathNode::Gap { gap: Gap::AtMost(0), exclude: None },
                state(0),
            ]),
            false,
        )?;
        assert!(empty.trace.steps.is_empty());
        let excluded = pattern(vec![
            state(0),
            PathNode::Gap { gap: Gap::Any, exclude: Some(Rc::new(|action| *action == 1)) },
            action(1),
            state(1),
        ]);
        assert_eq!(graph.shortest(&excluded, false)?.trace.steps.len(), 1);
        Ok(())
    }

    #[test]
    fn finds_a_global_shortest_match_and_searches_before_the_first_node()
    -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 3)?;
        let graph = Graph::new(&explored)?;
        let request =
            pattern(vec![PathNode::Gap { gap: Gap::Any, exclude: None }, action(1), state(1)]);
        assert_eq!(graph.shortest(&request, false)?.trace.steps.len(), 1);
        assert!(graph.shortest(&pattern(vec![state(2)]), false).is_err());
        assert_eq!(graph.shortest(&pattern(vec![state(2)]), true)?.trace.steps[0].action, 2);
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
            graph.shortest(&request, false),
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
            let first = graph.walk(&request, false, 5, &mut Rng::new(seed))?;
            let second = graph.walk(&request, false, 5, &mut Rng::new(seed))?;
            assert_eq!(first.rows, second.rows);
            assert!(first.trace.steps.len() <= 5);
            found_loop |= first.trace.steps.iter().any(|step| step.action == 0);
            experiments.insert(first.rows);
        }
        assert!(found_loop);
        assert!(experiments.len() > 1);
        assert!(matches!(
            graph.walk(&request, false, 0, &mut Rng::new(1)),
            Err(crate::Refusal::WalkBound { bound: 0 })
        ));
        Ok(())
    }

    #[test]
    fn cut_graph_answers_only_settled_searches() -> Result<(), crate::Refusal> {
        let explored = Checker::explore(&World, 2)?;
        let graph = Graph::new(&explored)?;
        assert_eq!(
            graph.shortest(&pattern(vec![state(0), action(1)]), false)?.trace.steps.len(),
            1
        );
        assert!(matches!(
            graph.shortest(&pattern(vec![state(0), action(2)]), false),
            Err(crate::Refusal::Unbridged { cut: true, .. })
        ));
        assert!(matches!(
            graph.shortest(
                &pattern(vec![
                    state(0),
                    PathNode::Gap { gap: Gap::AtMost(0), exclude: None },
                    state(2)
                ]),
                true
            ),
            Err(crate::Refusal::Unbridged { cut: false, .. })
        ));
        Ok(())
    }
}
