//! Independent test oracles over the checked activity Model; production uses the policy manifest.

use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{AnyMessage as _, Evidence, prost_types::Any, protocol::Refusal};
use umpire_temporal::{
    activity::{self, Action, Phase, Worker},
    policy::{Choice, History, TaskKey},
};

/// Refuses Worlds and attempts outside the bounded heartbeat search domain.
fn heartbeat_world(world: &activity::World, attempt: u8) -> Result<(), Refusal> {
    if !(1..=3).contains(&world.maximum_attempts)
        || !(1..=world.maximum_attempts).contains(&attempt)
    {
        return Err(Refusal::unsupported("heartbeat attempt is outside World.maximum_attempts"));
    }
    for (field, value) in [
        ("schedule_to_start", world.schedule_to_start),
        ("start_to_close", world.start_to_close),
        ("schedule_to_close", world.schedule_to_close),
        ("heartbeat_timeout", world.heartbeat_timeout),
    ] {
        if value.is_some_and(|ticks| !(1..=3).contains(&ticks.0)) {
            return Err(Refusal::unsupported(format!("heartbeat World.{field} is outside 1..=3")));
        }
    }
    Ok(())
}

fn heartbeat_search_limit(error: umpire_checker::search::SearchLimit) -> Refusal {
    Refusal::unsupported(format!("heartbeat successor {error}"))
}

/// Derives successful next heartbeat bits from complete compatible Model histories.
///
/// # Errors
/// Refuses invalid attempts, impossible histories, Model errors or exhausted search bounds.
pub(super) fn heartbeat_successors(
    world: &activity::World,
    attempt: u8,
    history: &[bool],
    max_depth: usize,
    max_work: usize,
) -> Result<Vec<bool>, Refusal> {
    use std::{
        collections::{HashSet, hash_map::DefaultHasher},
        hash::BuildHasherDefault,
    };
    use umpire_checker::search::Traversal;
    heartbeat_world(world, attempt)?;
    let catalog = world
        .checked_catalog(umpire_dsl::CATALOG_LIMIT)
        .map_err(|error| Refusal::unsupported(format!("heartbeat Model catalog: {error}")))?;
    let mut search = Traversal::<_, ()>::new(max_depth, max_work);
    let mut visited = HashSet::<_, BuildHasherDefault<DefaultHasher>>::default();
    for state in world.starts() {
        let position = (state, 0_usize, false);
        visited
            .try_reserve(1)
            .map_err(|_| Refusal::unsupported("heartbeat successor allocation bound"))?;
        if visited.insert(position.clone()) {
            search.root(position).map_err(heartbeat_search_limit)?;
        }
    }
    let mut bits = [false; 2];
    while let Some(parent) = search.next() {
        let (state, cursor, assigned) = search.label(parent).position.clone();
        if assigned && cursor > history.len() {
            bits[usize::from(state.cancel_requested)] = true;
            continue;
        }
        if state.attempt > attempt || umpire_dsl::Phase::closed(&state.phase) {
            continue;
        }
        for action in catalog.actions() {
            let heartbeat = *action == Action::from(Worker::Heartbeat);
            if assigned && matches!(action, Action::Worker(_)) && !heartbeat {
                continue;
            }
            search.charge().map_err(heartbeat_search_limit)?;
            let choices = world
                .checked_step(&state, action.clone())
                .map_err(|error| Refusal::unsupported(format!("heartbeat Model step: {error}")))?;
            for (outcome, _, next) in choices {
                if outcome != Outcome::Accepted {
                    continue;
                }
                if assigned && (next.phase != Phase::Started || next.attempt != attempt) {
                    continue;
                }
                let mut next_cursor = cursor;
                if assigned && heartbeat {
                    if cursor < history.len() && history[cursor] != next.cancel_requested {
                        continue;
                    }
                    next_cursor += 1;
                }
                let next_assigned = assigned
                    || (*action == Action::from(Worker::Poll(attempt))
                        && next.phase == Phase::Started
                        && next.attempt == attempt);
                let position = (next, next_cursor, next_assigned);
                if visited.contains(&position) {
                    continue;
                }
                visited
                    .try_reserve(1)
                    .map_err(|_| Refusal::unsupported("heartbeat successor allocation bound"))?;
                visited.insert(position.clone());
                search.push(parent, position, Some(())).map_err(heartbeat_search_limit)?;
            }
        }
    }
    let successors: Vec<_> =
        [false, true].into_iter().filter(|bit| bits[usize::from(*bit)]).collect();
    if successors.is_empty() {
        return Err(Refusal::unsupported("heartbeat_history has no successful Model continuation"));
    }
    Ok(successors)
}

/// Checks a supplied observable successor declaration independently of its response rows.
///
/// # Errors
/// Refuses malformed keys, missing/duplicate/impossible bits, or incomplete Model search.
pub(super) fn validate_heartbeat_successors(
    world: &activity::World,
    declaration: &Any,
    max_depth: usize,
    max_work: usize,
) -> Result<(), Refusal> {
    let declaration: umpire_temporal_api::umpire::temporal::v1::ActivityTaskSuccessors =
        declaration.unpack()?;
    let key = declaration
        .key
        .as_ref()
        .ok_or_else(|| Refusal::malformed("heartbeat successor key is missing"))?;
    if key.scenario.is_empty() || key.activity_type.is_empty() {
        return Err(Refusal::malformed("heartbeat successor key identity is missing"));
    }
    let bits = &declaration.cancel_requested;
    if bits.is_empty() || bits.len() > 2 || (bits.len() == 2 && bits[0] == bits[1]) {
        return Err(Refusal::malformed(
            "heartbeat successor domain requires unique observable bits",
        ));
    }
    let attempt = u8::try_from(key.attempt)
        .map_err(|_| Refusal::unsupported("heartbeat attempt is outside World"))?;
    let expected =
        heartbeat_successors(world, attempt, &key.heartbeat_history, max_depth, max_work)?;
    let mut actual = declaration.cancel_requested.clone();
    actual.sort_unstable();
    if actual != expected {
        return Err(Refusal::unsupported(
            "observable successor domain differs from checked Model histories",
        ));
    }
    Ok(())
}

fn allocation() -> Refusal {
    Refusal::unsupported("policy inventory allocation failed")
}

fn search_limit(error: umpire_checker::search::SearchLimit) -> Refusal {
    Refusal::unsupported(format!("policy inventory {error}"))
}

/// One independent full-state checked-step inventory, before policy filtering.
pub(super) struct Inventory {
    /// Every reached full state in canonical breadth-first order.
    pub(super) states: Vec<activity::Model>,
    /// Every emitted action/outcome/fact/choice, including accepted stays and rejections.
    pub(super) edges: Vec<Evidence<activity::World>>,
}

impl Inventory {
    /// Exhausts the Model's emitted graph under explicit traversal bounds.
    ///
    /// # Errors
    /// Refuses catalog/step errors, allocation or incomplete depth/work search.
    pub(super) fn derive(
        world: &activity::World,
        depth: usize,
        work: usize,
    ) -> Result<Self, Refusal> {
        heartbeat_world(world, 1)?;
        let catalog = world
            .checked_catalog(umpire_dsl::CATALOG_LIMIT)
            .map_err(|error| Refusal::unsupported(format!("policy inventory catalog: {error}")))?;
        let mut search = umpire_checker::search::Traversal::<_, ()>::new(depth, work);
        let mut states = Vec::new();
        let mut edges = Vec::new();
        for state in world.starts() {
            if !states.contains(&state) {
                states.try_reserve(1).map_err(|_| allocation())?;
                search.root(state.clone()).map_err(search_limit)?;
                states.push(state);
            }
        }
        while let Some(parent) = search.next() {
            let source = search.label(parent).position.clone();
            for action in catalog.actions() {
                search.charge().map_err(search_limit)?;
                let steps = world.checked_step(&source, action.clone()).map_err(|error| {
                    Refusal::unsupported(format!("policy inventory step: {error}"))
                })?;
                for (choice, (outcome, facts, target)) in steps.into_iter().enumerate() {
                    if !states.contains(&target) {
                        states.try_reserve(1).map_err(|_| allocation())?;
                        search.push(parent, target.clone(), Some(())).map_err(search_limit)?;
                        states.push(target.clone());
                    }
                    edges.try_reserve(1).map_err(|_| allocation())?;
                    edges.push(Evidence {
                        source: source.clone(),
                        action: action.clone(),
                        choice,
                        outcome,
                        facts,
                        target,
                    });
                }
            }
        }
        Ok(Self { states, edges })
    }
}

/// A raw history policy collision caused by the executor's deliberate per-task reset.
pub(super) struct TaskResetAlias {
    /// Complete typed checked edges witnessing both aliased task suffixes.
    pub(super) paths: [Vec<Evidence<activity::World>>; 2],
    /// Different prior-task observations, before a successful retry and fresh delivery.
    pub(super) prior_histories: [Vec<bool>; 2],
    /// The single actual next-task key shared by both histories.
    pub(super) next_key: TaskKey,
    /// An example pair of incompatible raw-history assignments at that key.
    pub(super) incompatible_choices: [Choice; 2],
}

/// Reports exact cross-task memory collisions independently of stamped policy names.
///
/// # Errors
/// Refuses if the declared checked World cannot supply either concrete alias path.
pub(super) fn task_reset_aliases(world: &activity::World) -> Result<Vec<TaskResetAlias>, Refusal> {
    let mut aliases = Vec::new();
    let mut state = world
        .starts()
        .into_iter()
        .next()
        .ok_or_else(|| Refusal::unsupported("policy World has no start"))?;
    for next_attempt in 2..=world.maximum_attempts {
        let delivery = alias_edge(world, &state, Worker::Poll(next_attempt - 1))?;
        let owned = History::default().advance(&delivery)?;
        let heartbeat = alias_edge(world, &delivery.target, Worker::Heartbeat)?;
        let observed = owned.advance(&heartbeat)?;
        let retry = Worker::RespondFailed { retryable: true };
        let direct_retry = alias_edge(world, &delivery.target, retry)?;
        let heartbeat_retry = alias_edge(world, &heartbeat.target, retry)?;
        let direct_delivery = alias_edge(world, &direct_retry.target, Worker::Poll(next_attempt))?;
        let heartbeat_delivery =
            alias_edge(world, &heartbeat_retry.target, Worker::Poll(next_attempt))?;
        let left = owned.advance(&direct_retry)?.advance(&direct_delivery)?.task;
        let right = observed.advance(&heartbeat_retry)?.advance(&heartbeat_delivery)?.task;
        if left != right {
            return Err(Refusal::unsupported(
                "retry delivery did not share an exact observable key",
            ));
        }
        let next_key =
            left.ok_or_else(|| Refusal::unsupported("retry delivery has no owned task"))?;
        aliases.try_reserve(1).map_err(|_| allocation())?;
        state = direct_retry.target.clone();
        let second_delivery = alias_edge(world, &delivery.source, Worker::Poll(next_attempt - 1))?;
        aliases.push(TaskResetAlias {
            paths: [
                vec![delivery, direct_retry, direct_delivery],
                vec![second_delivery, heartbeat, heartbeat_retry, heartbeat_delivery],
            ],
            prior_histories: [vec![], vec![false]],
            next_key,
            incompatible_choices: [Choice::Complete, Choice::Fail],
        });
    }
    Ok(aliases)
}

fn alias_edge(
    world: &activity::World,
    source: &activity::Model,
    worker: Worker,
) -> Result<Evidence<activity::World>, Refusal> {
    let action = Action::from(worker);
    let (choice, (outcome, facts, target)) = world
        .checked_step(source, action.clone())
        .map_err(|error| Refusal::unsupported(format!("task-reset alias Model step: {error}")))?
        .into_iter()
        .enumerate()
        .find(|(_, (outcome, _, _))| *outcome == Outcome::Accepted)
        .ok_or_else(|| Refusal::unsupported("task-reset alias path is disabled"))?;
    Ok(Evidence { source: source.clone(), action, choice, outcome, facts, target })
}
