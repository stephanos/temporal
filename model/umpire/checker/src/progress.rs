use std::{
    collections::VecDeque,
    panic::{AssertUnwindSafe, catch_unwind},
};

use petgraph::{algo::tarjan_scc, graph::DiGraph};
use umpire_dsl::{Assumption, Machine, Outcome, Progress, Spell, Verdict};

use crate::{Path, PathStep, Refusal, Result, StateId, Visited, explore::panic_message};

pub(crate) fn validate_scopes<M: Machine>(
    world: &M,
    progress: &[Progress<M>],
    actions: &[M::Action],
) -> std::result::Result<(), Refusal> {
    for (_, assumption) in world.assumptions() {
        validate_assumption(&assumption, actions)?;
    }
    for claim in progress {
        validate_assumption(&claim.scope, actions)?;
    }
    Ok(())
}

fn validate_assumption<A: Spell + Clone + PartialEq>(
    assumption: &Assumption<A>,
    actions: &[A],
) -> std::result::Result<(), Refusal> {
    for action in
        assumption.only.iter().flatten().chain(&assumption.excluded).chain(&assumption.fair)
    {
        if !actions.contains(action) {
            let mut name = String::new();
            action.spell(&mut name);
            return Err(Refusal::ScopeAction { action: name });
        }
    }
    if (assumption.only.is_some() || !assumption.excluded.is_empty())
        && !actions.iter().any(|action| assumption.allows(action))
    {
        return Err(Refusal::EmptyAssumption);
    }
    Ok(())
}

struct ProgressPredicates<A> {
    triggers: Vec<bool>,
    avoids: Vec<bool>,
    assumption: Assumption<A>,
}

impl<M: Machine> Result<M> {
    pub(crate) fn check_progress(
        &mut self,
        world: &M,
        claim: &Progress<M>,
        index: usize,
        actions: &[M::Action],
    ) -> std::result::Result<(), Refusal> {
        let count = self.visited.len();
        let Some(ProgressPredicates { triggers, avoids, assumption }) =
            self.progress_predicates(world, claim, index, actions)?
        else {
            return Ok(());
        };
        let allowed: Vec<_> = actions.iter().map(|action| assumption.allows(action)).collect();
        let fair_actions: Vec<_> = actions
            .iter()
            .enumerate()
            .filter(|(index, action)| allowed[*index] && assumption.is_fair(action))
            .collect();
        let complete: Vec<_> = self
            .completed
            .iter()
            .map(|row| allowed.iter().enumerate().all(|(a, allowed)| !allowed || row[a]))
            .collect();
        let mut outgoing = vec![Vec::new(); count];
        let mut enabled = vec![vec![false; actions.len()]; count];
        for (edge_index, edge) in self.transitions.iter().enumerate() {
            let Some(action) = actions.iter().position(|action| action == &edge.action) else {
                unreachable!("transitions use catalog actions")
            };
            if allowed[action] {
                outgoing[edge.source.0].push(edge_index);
                enabled[edge.source.0][action] = true;
            }
        }
        let mut parents = vec![None; count];
        let mut origins = vec![None; count];
        let mut queue = VecDeque::new();
        for state in 0..count {
            if triggers[state] && avoids[state] {
                origins[state] = Some(state);
                queue.push_back(state);
            }
        }
        while let Some(state) = queue.pop_front() {
            if !complete[state] {
                self.claims[index].outcome = Outcome::Unknown;
            }
            if complete[state] && outgoing[state].is_empty() {
                self.refute_progress(index, state, &parents, &origins, &[]);
                return Ok(());
            }
            for &edge_index in &outgoing[state] {
                if let Some(target) = self.transitions[edge_index].target
                    && avoids[target.0]
                    && origins[target.0].is_none()
                {
                    origins[target.0] = origins[state];
                    parents[target.0] = Some(edge_index);
                    queue.push_back(target.0);
                }
            }
        }
        let mut graph = DiGraph::<StateId, ()>::new();
        let nodes: Vec<_> = (0..count).map(|state| graph.add_node(StateId(state))).collect();
        for state in 0..count {
            if origins[state].is_none() {
                continue;
            }
            for &edge_index in &outgoing[state] {
                if let Some(target) = self.transitions[edge_index].target
                    && avoids[target.0]
                {
                    graph.add_edge(nodes[state], nodes[target.0], ());
                }
            }
        }
        let mut components: Vec<Vec<usize>> = tarjan_scc(&graph)
            .into_iter()
            .map(|component| {
                let mut states: Vec<_> = component.into_iter().map(|node| graph[node].0).collect();
                states.sort_unstable();
                states
            })
            .collect();
        components.sort_by_key(|component| component[0]);
        for component in components {
            let entry = component[0];
            if origins[entry].is_none() {
                continue;
            }
            if let Some(cycle) = self.fair_cycle(&component, &outgoing, &enabled, &fair_actions) {
                self.refute_progress(index, entry, &parents, &origins, &cycle);
                return Ok(());
            }
        }
        Ok(())
    }

    fn fair_cycle(
        &self,
        component: &[usize],
        outgoing: &[Vec<usize>],
        enabled: &[Vec<bool>],
        fair_actions: &[(usize, &M::Action)],
    ) -> Option<Vec<usize>> {
        let entry = component[0];
        let mut member = vec![false; outgoing.len()];
        for &state in component {
            member[state] = true;
        }
        let internal_edges: Vec<_> = component
            .iter()
            .flat_map(|&state| outgoing[state].iter().copied())
            .filter(|&edge| self.transitions[edge].target.is_some_and(|target| member[target.0]))
            .collect();
        if internal_edges.is_empty() {
            return None;
        }
        let mut required = Vec::new();
        for &(action, value) in fair_actions {
            if component
                .iter()
                .any(|&state| self.completed[state][action] && !enabled[state][action])
            {
                continue;
            }
            if let Some(&edge) =
                internal_edges.iter().find(|&&edge| &self.transitions[edge].action == value)
            {
                required.push(edge);
            } else {
                return None;
            }
        }
        let mut cycle = Vec::new();
        let mut current = entry;
        for &destination in component {
            let route = self.progress_route(current, destination, &member, outgoing);
            cycle.extend(route);
            current = destination;
        }
        for edge in required {
            if cycle
                .iter()
                .any(|&taken| self.transitions[taken].action == self.transitions[edge].action)
            {
                continue;
            }
            cycle.extend(self.progress_route(
                current,
                self.transitions[edge].source.0,
                &member,
                outgoing,
            ));
            cycle.push(edge);
            current = self.progress_target(edge);
        }
        if cycle.is_empty() {
            cycle.push(internal_edges[0]);
            current = self.progress_target(internal_edges[0]);
        }
        cycle.extend(self.progress_route(current, entry, &member, outgoing));
        Some(cycle)
    }

    fn progress_predicates(
        &mut self,
        world: &M,
        claim: &Progress<M>,
        index: usize,
        actions: &[M::Action],
    ) -> std::result::Result<Option<ProgressPredicates<M::Action>>, Refusal> {
        let count = self.visited.len();
        let mut triggers = vec![false; count];
        let mut avoids = vec![false; count];
        let mut assumption = claim.evaluator.is_none().then(|| claim.scope.clone());
        let mut inconsistent = false;
        for state in 0..count {
            let Some(evaluator) = claim.evaluator else {
                for (predicate, values, target) in
                    [(claim.trigger, &mut triggers, false), (claim.target, &mut avoids, true)]
                {
                    let Some(verdict) =
                        self.progress_evaluation(claim.name, index, StateId(state), |state| {
                            predicate(world, state)
                        })
                    else {
                        return Ok(None);
                    };
                    match verdict {
                        Verdict::Holds => values[state] = !target,
                        Verdict::Fails => values[state] = target,
                        Verdict::Unknown(hole) => {
                            self.claims[index].outcome = Outcome::Unknown;
                            if !self.claims[index].holes.contains(&hole.reason) {
                                self.claims[index].holes.push(hole.reason);
                            }
                        }
                    }
                }
                continue;
            };
            let Some(evaluation) =
                self.progress_evaluation(claim.name, index, StateId(state), |state| {
                    evaluator(world, state)
                })
            else {
                return Ok(None);
            };
            validate_assumption(&evaluation.assumption, actions)?;
            if assumption.as_ref().is_some_and(|fixed| fixed != &evaluation.assumption) {
                inconsistent = true;
            }
            if assumption.is_none() {
                assumption = Some(evaluation.assumption);
            }
            for (verdict, values, target) in
                [(evaluation.trigger, &mut triggers, false), (evaluation.target, &mut avoids, true)]
            {
                match verdict {
                    Verdict::Holds => values[state] = !target,
                    Verdict::Fails => values[state] = target,
                    Verdict::Unknown(hole) => {
                        self.claims[index].outcome = Outcome::Unknown;
                        if !self.claims[index].holes.contains(&hole.reason) {
                            self.claims[index].holes.push(hole.reason);
                        }
                    }
                }
            }
        }
        if inconsistent {
            self.claims[index].outcome = Outcome::Unknown;
            self.claims[index].holes.push("progress assumption differs across reached states");
            return Ok(None);
        }
        Ok(Some(ProgressPredicates {
            triggers,
            avoids,
            assumption: assumption.unwrap_or_default(),
        }))
    }

    fn progress_evaluation<T>(
        &mut self,
        name: &str,
        index: usize,
        state: StateId,
        evaluate: impl FnOnce(&M::State) -> T,
    ) -> Option<T> {
        match catch_unwind(AssertUnwindSafe(|| evaluate(self.state(state)))) {
            Ok(value) => Some(value),
            Err(error) => {
                let mut path = self.path(state);
                path.failure =
                    Some(format!("claim {name} panicked: {}", panic_message(error.as_ref())));
                self.claims[index].outcome = Outcome::Refuted;
                self.claims[index].path = Some(path);
                None
            }
        }
    }

    fn progress_route(
        &self,
        start: usize,
        end: usize,
        member: &[bool],
        outgoing: &[Vec<usize>],
    ) -> Vec<usize> {
        let mut parents = vec![None; member.len()];
        let mut seen = vec![false; member.len()];
        seen[start] = true;
        let mut queue = VecDeque::from([start]);
        while let Some(state) = queue.pop_front() {
            if state == end {
                break;
            }
            for &edge in &outgoing[state] {
                if let Some(target) = self.transitions[edge].target
                    && member[target.0]
                    && !seen[target.0]
                {
                    seen[target.0] = true;
                    parents[target.0] = Some(edge);
                    queue.push_back(target.0);
                }
            }
        }
        assert!(seen[end], "SCC route exists");
        self.progress_suffix(end, &parents)
    }

    fn progress_suffix(&self, mut state: usize, parents: &[Option<usize>]) -> Vec<usize> {
        let mut path = Vec::new();
        while let Some(edge) = parents[state] {
            path.push(edge);
            state = self.transitions[edge].source.0;
        }
        path.reverse();
        path
    }

    fn progress_target(&self, edge: usize) -> usize {
        let Some(target) = self.transitions[edge].target else {
            unreachable!("cycle edges have admitted targets")
        };
        target.0
    }

    fn append_progress(&self, path: &mut Path<M::State, M::Action>, edges: &[usize]) {
        for &edge in edges {
            let edge = &self.transitions[edge];
            path.steps.push(PathStep {
                action: edge.action.clone(),
                choice: edge.choice,
                after: edge.emitted.2.clone(),
            });
        }
    }

    fn refute_progress(
        &mut self,
        index: usize,
        state: usize,
        parents: &[Option<usize>],
        origins: &[Option<usize>],
        cycle: &[usize],
    ) {
        let Some(origin) = origins[state] else {
            unreachable!("counterexamples have obligation origins")
        };
        let mut path = self.path(StateId(origin));
        self.append_progress(&mut path, &self.progress_suffix(state, parents));
        if !cycle.is_empty() {
            path.cycle_start = Some(path.steps.len());
            self.append_progress(&mut path, cycle);
        }
        self.claims[index].outcome = Outcome::Refuted;
        self.claims[index].path = Some(path);
    }
}
