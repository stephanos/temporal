use std::{
    collections::HashSet,
    hash::{BuildHasherDefault, DefaultHasher},
};

use super::{
    AnswerPlan, Any, Event, EventKind, Evidence, EvidenceMatch, Machine, Pending, Prefix,
    Projection, ProposalDecision, Rc, ReachTarget, Realize, Runtime, SearchLimit, Traversal,
    prefix_error, protocol, reserve, search_error,
};

type SearchResult<M> = Result<Option<(Prefix<M>, Vec<Node<M>>, bool)>, protocol::Refusal>;
type Search<M> = Traversal<Node<M>, ()>;

struct Observation<'a, M: Machine> {
    actual: &'a Any,
    continues: bool,
    latest: bool,
    kind: &'a Rc<EventKind<M>>,
}

pub(super) struct Node<M: Machine> {
    prefix: Prefix<M>,
    state: M::State,
    edges: Vec<Rc<Evidence<M>>>,
    cursors: Vec<usize>,
    blocked: Option<SearchLimit>,
    expand: bool,
}

impl<M: Machine> Node<M> {
    fn copy(&self) -> Result<Self, SearchLimit> {
        Ok(Self {
            prefix: copy_prefix(&self.prefix)?,
            state: self.state.clone(),
            edges: copy_shared(&self.edges)?,
            cursors: copy_values(&self.cursors)?,
            blocked: self.blocked,
            expand: self.expand,
        })
    }
}

fn copy_shared<T>(source: &[Rc<T>]) -> Result<Vec<Rc<T>>, SearchLimit> {
    let mut result = Vec::new();
    result.try_reserve(source.len()).map_err(|_| SearchLimit::Allocation)?;
    result.extend(source.iter().map(Rc::clone));
    Ok(result)
}

fn copy_values<T: Clone>(source: &[T]) -> Result<Vec<T>, SearchLimit> {
    let mut result = Vec::new();
    result.try_reserve(source.len()).map_err(|_| SearchLimit::Allocation)?;
    result.extend_from_slice(source);
    Ok(result)
}

fn copy_prefix<M: Machine>(prefix: &Prefix<M>) -> Result<Prefix<M>, SearchLimit> {
    Ok(Prefix { edges: copy_shared(&prefix.edges)?, events: copy_shared(&prefix.events)? })
}

impl<M: Machine, R: Realize<M>> Runtime<M, R>
where
    M::Outcome: PartialEq,
{
    pub(super) fn search_observations(&self, observation: &Any) -> SearchResult<M> {
        let mut journal = copy_shared(&self.prefix.events).map_err(search_error)?;
        let current = self.current()?.clone();
        reserve(&mut journal, 1)?;
        journal.push(Rc::new(Event {
            before: current.clone(),
            after: current,
            start: 0,
            end: 0,
            kind: Rc::new(EventKind::Observation { actual: observation.clone() }),
            cursors: None,
            continues: false,
        }));
        self.search_journal(&journal, self.pending.is_none())
    }

    fn search_journal(&self, journal: &[Rc<Event<M>>], first: bool) -> SearchResult<M> {
        let scope = Realize::scope(&self.realization);
        let mut traversal = Traversal::<Node<M>, ()>::new(scope.prefix, scope.work);
        if self.frontier.is_empty() {
            traversal
                .root(Node {
                    prefix: copy_prefix(&self.prefix).map_err(search_error)?,
                    state: self.current()?.clone(),
                    edges: Vec::new(),
                    cursors: Vec::new(),
                    blocked: None,
                    expand: false,
                })
                .map_err(search_error)?;
        } else {
            for node in &self.frontier {
                self.validate_prefix_data(
                    &node.prefix,
                    node.prefix.events.last().map_or_else(
                        || self.initial.as_ref().ok_or_else(prefix_error),
                        |event| Ok(&event.after),
                    )?,
                    Self::pending_for_prefix(&node.prefix).as_ref(),
                )
                .map_err(|_| disagreement())?;
                traversal.root(node.copy().map_err(search_error)?).map_err(search_error)?;
            }
        }
        let mut blocked = Vec::new();
        let mut completed = Vec::new();
        while let Some(id) = traversal.next() {
            let node = traversal.label(id).position.copy().map_err(search_error)?;
            if node.blocked.is_some() {
                reserve(&mut blocked, 1)?;
                blocked.push(node);
                continue;
            }
            let index = node.prefix.events.len();
            if index == journal.len() {
                let continues = node.prefix.events.last().is_some_and(|event| event.continues);
                if first {
                    let prefix = copy_prefix(&node.prefix).map_err(search_error)?;
                    let mut frontier = Vec::new();
                    reserve(&mut frontier, 1)?;
                    frontier.push(node);
                    let pending = traversal.pending(Node::copy).map_err(search_error)?;
                    reserve(&mut frontier, pending.len() + blocked.len())?;
                    frontier.extend(pending);
                    frontier.extend(blocked);
                    self.bound_frontier(&frontier)?;
                    return Ok(Some((prefix, frontier, continues)));
                }
                reserve(&mut completed, 1)?;
                completed.push(node);
                continue;
            }
            self.expand_event(
                &mut traversal,
                id,
                node,
                &journal[index],
                index + 1 == journal.len(),
            )?;
        }
        if let Some(limit) = blocked.first().and_then(|node| node.blocked) {
            return Err(search_error(limit));
        }
        if let Some(node) = completed.first() {
            let prefix = copy_prefix(&node.prefix).map_err(search_error)?;
            self.bound_frontier(&completed)?;
            let continues = node.prefix.events.last().is_some_and(|event| event.continues);
            if completed.iter().any(|node| {
                node.prefix.events.last().is_some_and(|event| event.continues) != continues
            }) {
                return Err(disagreement());
            }
            return Ok(Some((prefix, completed, continues)));
        }
        Ok(None)
    }

    fn expand_event(
        &self,
        traversal: &mut Search<M>,
        id: usize,
        node: Node<M>,
        event: &Event<M>,
        latest: bool,
    ) -> Result<(), protocol::Refusal> {
        let kind = &event.kind;
        match kind.as_ref() {
            EventKind::Observation { actual } => {
                self.observe_frontier(
                    traversal,
                    id,
                    node,
                    &Observation { actual, continues: event.continues, latest, kind },
                )?;
            }
            EventKind::Callback { actual, reply, response } => {
                let plan = Realize::answer(&self.realization, &self.machine, actual, &node.state)?;
                if plan.reply != *reply || plan.response_action != *response {
                    return Err(disagreement());
                }
                let Some(replay) = self.replay(&node.state, plan.delivery)? else {
                    return Err(disagreement());
                };
                if self.checked_step(&replay.state, response.clone())?.is_empty() {
                    return Err(disagreement());
                }
                let mut next = node;
                next.edges = replay.edges;
                next.state = replay.state;
                self.finish_event(&mut next, Rc::clone(kind))?;
                traversal.push(id, next, None).map_err(search_error)?;
            }
            EventKind::Proposal { request, action, seed, bootstrap } => {
                if !self
                    .valid_proposal(&node.state, request, action.as_ref(), *seed, *bootstrap)
                    .map_err(|_| disagreement())?
                {
                    return Err(disagreement());
                }
                if action.as_ref().is_some_and(|action| self.machine.internal(action)) {
                    return Err(disagreement());
                }
                if let Some(action) = action
                    && self.checked_step(&node.state, action.clone())?.is_empty()
                {
                    return Err(disagreement());
                }
                let mut next = node;
                self.finish_event(&mut next, Rc::clone(kind))?;
                traversal.push(id, next, None).map_err(search_error)?;
            }
            EventKind::Unexplained(_) => return Err(prefix_error()),
        }
        Ok(())
    }

    fn observe_frontier(
        &self,
        traversal: &mut Search<M>,
        id: usize,
        node: Node<M>,
        observation: &Observation<'_, M>,
    ) -> Result<(), protocol::Refusal> {
        let before = node.prefix.events.last().map_or_else(
            || self.initial.as_ref().ok_or_else(prefix_error),
            |event| Ok(&event.after),
        )?;
        let projection =
            Realize::observe(&self.realization, &self.machine, observation.actual, before)?;
        match projection {
            Projection::Reach(target) => {
                self.reach_frontier(traversal, id, node, observation, &target)?;
            }
            Projection::Current => {
                if node.edges.is_empty()
                    && let Some(next) = self.finish_observation(&node, observation)?
                {
                    traversal.push(id, next, None).map_err(search_error)?;
                }
            }
            Projection::Stutter(target) => {
                if node.edges.is_empty()
                    && target.matches(&node.state)
                    && let Some(next) = self.finish_observation(&node, observation)?
                {
                    traversal.push(id, next, None).map_err(search_error)?;
                }
            }
            Projection::Candidates(candidates) => {
                for candidate in candidates.into_iter().rev() {
                    if candidate.path.is_empty() {
                        continue;
                    }
                    traversal.charge().map_err(search_error)?;
                    if let Some(replay) = self.replay(&node.state, candidate.path)?
                        && candidate.outcome.as_ref().is_none_or(|outcome| {
                            replay.edges.last().is_some_and(|edge| outcome == &edge.outcome)
                        })
                    {
                        let mut next = node.copy().map_err(search_error)?;
                        next.state = replay.state;
                        next.edges = replay.edges;
                        if let Some(next) = self.finish_observation(&next, observation)? {
                            traversal.push(id, next, None).map_err(search_error)?;
                        }
                    }
                }
            }
        }
        Ok(())
    }

    fn reach_frontier(
        &self,
        traversal: &mut Search<M>,
        id: usize,
        node: Node<M>,
        observation: &Observation<'_, M>,
        target: &R::Target,
    ) -> Result<(), protocol::Refusal> {
        let scope = Realize::scope(&self.realization);
        let admission = self.node_evidence(observation.actual, &node.edges)?;
        let Some(cursor) = admission.cursor() else { return Ok(()) };
        if matches!(admission, EvidenceMatch::Stutter { .. })
            && node.edges.last().is_some_and(|edge| edge.source != edge.target)
        {
            return Ok(());
        }
        let mut node = node;
        if node.cursors.is_empty() {
            reserve(&mut node.cursors, 1)?;
            node.cursors.push(cursor);
        }
        if !node.expand
            && matches!(admission, EvidenceMatch::Match { .. } | EvidenceMatch::Stutter { .. })
            && target.matches(&node.state)
        {
            let mut unexamined = node.copy().map_err(search_error)?;
            unexamined.expand = true;
            traversal.push(id, unexamined, None).map_err(search_error)?;
            if let Some(next) = self.finish_observation(&node, observation)? {
                traversal.push(id, next, None).map_err(search_error)?;
            }
            return Ok(());
        }
        let catalog = self.catalog()?;
        for action in &catalog {
            traversal.charge().map_err(search_error)?;
            for (choice, (outcome, facts, target)) in
                self.checked_step(&node.state, action.clone())?.into_iter().enumerate()
            {
                traversal.charge().map_err(search_error)?;
                let edge = Rc::new(Evidence {
                    source: node.state.clone(),
                    action: action.clone(),
                    choice,
                    outcome,
                    facts,
                    target: target.clone(),
                });
                let mut next = node.copy().map_err(search_error)?;
                reserve(&mut next.edges, 1)?;
                next.edges.push(edge);
                let evidence = self.node_evidence(observation.actual, &next.edges)?;
                let Some(cursor) = evidence.cursor() else { continue };
                if matches!(evidence, EvidenceMatch::Stutter { .. })
                    && next.edges.last().is_some_and(|edge| edge.source != edge.target)
                {
                    continue;
                }
                next.state = target;
                next.expand = false;
                reserve(&mut next.cursors, 1)?;
                next.cursors.push(cursor);
                if next.edges.len() > scope.depth {
                    next.blocked = Some(SearchLimit::Depth { limit: scope.depth });
                }
                traversal.push(id, next, Some(())).map_err(search_error)?;
            }
        }
        Ok(())
    }

    fn node_evidence(
        &self,
        actual: &Any,
        edges: &[Rc<Evidence<M>>],
    ) -> Result<EvidenceMatch, protocol::Refusal> {
        let mut refs = Vec::new();
        reserve(&mut refs, edges.len())?;
        refs.extend(edges.iter().map(Rc::as_ref));
        Realize::reach_evidence(&self.realization, actual, &refs)
    }

    pub(super) fn resolved_frontier(&self) -> Result<Vec<Node<M>>, protocol::Refusal> {
        if self.frontier.is_empty() {
            return Ok(Vec::new());
        }
        self.search_journal(&self.prefix.events, false)?
            .map(|(_, nodes, _)| nodes)
            .ok_or_else(prefix_error)
    }

    pub(super) fn check_proposal_decisions(
        &self,
        nodes: &[Node<M>],
        seed: u64,
        expected: &ProposalDecision<M::Action>,
    ) -> Result<(), protocol::Refusal> {
        for node in nodes {
            let decision = self.proposal_decision(&node.state, seed).map_err(|_| disagreement())?;
            if decision != *expected {
                return Err(disagreement());
            }
        }
        Ok(())
    }

    pub(super) fn check_answers(
        &self,
        nodes: &[Node<M>],
        callback: &Any,
        expected: &AnswerPlan<M::Action>,
    ) -> Result<(), protocol::Refusal> {
        for node in nodes {
            let plan = Realize::answer(&self.realization, &self.machine, callback, &node.state)?;
            if plan.reply != expected.reply || plan.response_action != expected.response_action {
                return Err(disagreement());
            }
            let Some(replay) = self.replay(&node.state, plan.delivery)? else {
                return Err(disagreement());
            };
            if !self.catalog()?.contains(&plan.response_action)
                || self.checked_step(&replay.state, plan.response_action)?.is_empty()
            {
                return Err(disagreement());
            }
        }
        Ok(())
    }

    fn finish_observation(
        &self,
        node: &Node<M>,
        observation: &Observation<'_, M>,
    ) -> Result<Option<Node<M>>, protocol::Refusal> {
        let actual = observation.actual;
        let pending = Self::pending_for_prefix(&node.prefix);
        if let Some(pending) = &pending {
            Realize::matches(&self.realization, &pending.request, actual)?;
        }
        let continues = pending.is_some()
            && Realize::continue_polling(
                &self.realization,
                actual,
                &node.state,
                pending.as_ref().and_then(|pending| pending.action.as_ref()),
            )?;
        if !observation.latest && continues != observation.continues {
            return Ok(None);
        }
        let mut next = node.copy().map_err(search_error)?;
        let cursors =
            if next.cursors.is_empty() { None } else { Some(std::mem::take(&mut next.cursors)) };
        self.finish_event(&mut next, Rc::clone(observation.kind))?;
        let event = next.prefix.events.last_mut().ok_or_else(prefix_error)?;
        let event = Rc::get_mut(event).ok_or_else(prefix_error)?;
        event.cursors = cursors;
        event.continues = continues;
        Ok(Some(next))
    }

    fn pending_for_prefix(prefix: &Prefix<M>) -> Option<Pending<M::Action>> {
        let mut pending = None;
        for event in &prefix.events {
            match event.kind.as_ref() {
                EventKind::Proposal { request, action, .. } => {
                    pending = Some(Pending { request: request.clone(), action: action.clone() });
                }
                EventKind::Observation { .. } if !event.continues => pending = None,
                _ => {}
            }
        }
        pending
    }

    fn finish_event(
        &self,
        node: &mut Node<M>,
        kind: Rc<EventKind<M>>,
    ) -> Result<(), protocol::Refusal> {
        let before = node.prefix.events.last().map_or_else(
            || self.initial.clone().ok_or_else(prefix_error),
            |event| Ok(event.after.clone()),
        )?;
        let start = node.prefix.edges.len();
        reserve(&mut node.prefix.edges, node.edges.len())?;
        node.prefix.edges.append(&mut node.edges);
        reserve(&mut node.prefix.events, 1)?;
        node.prefix.events.push(Rc::new(Event {
            before,
            after: node.state.clone(),
            start,
            end: node.prefix.edges.len(),
            kind,
            cursors: None,
            continues: false,
        }));
        node.cursors.clear();
        node.expand = false;
        let scope = Realize::scope(&self.realization);
        if node.prefix.edges.len() + node.prefix.events.len() > scope.prefix {
            return Err(protocol::Refusal::new(
                protocol::RefusalCode::Budget,
                format!("prefix bound {}", scope.prefix),
            ));
        }
        Ok(())
    }

    fn bound_frontier(&self, nodes: &[Node<M>]) -> Result<(), protocol::Refusal> {
        let limit = Realize::scope(&self.realization).prefix;
        let bound = || {
            protocol::Refusal::new(
                protocol::RefusalCode::Budget,
                format!("frontier history bound {limit}"),
            )
        };
        if nodes.len() > limit {
            return Err(bound());
        }
        // Identity accounting must not require guest entropy imports.
        let mut edges = HashSet::<_, BuildHasherDefault<DefaultHasher>>::default();
        let mut calls = HashSet::<_, BuildHasherDefault<DefaultHasher>>::default();
        for node in nodes {
            for edge in node.prefix.edges.iter().chain(&node.edges) {
                let identity = Rc::as_ptr(edge);
                if !edges.contains(&identity) {
                    if edges.len() + calls.len() == limit {
                        return Err(bound());
                    }
                    edges.try_reserve(1).map_err(|_| super::allocation_error())?;
                    edges.insert(identity);
                }
            }
            for event in &node.prefix.events {
                let identity = Rc::as_ptr(&event.kind);
                if !calls.contains(&identity) {
                    if edges.len() + calls.len() == limit {
                        return Err(bound());
                    }
                    calls.try_reserve(1).map_err(|_| super::allocation_error())?;
                    calls.insert(identity);
                }
            }
        }
        Ok(())
    }
}

fn disagreement() -> protocol::Refusal {
    protocol::Refusal::new(
        protocol::RefusalCode::State,
        "compatible branches disagree on control decision",
    )
}
