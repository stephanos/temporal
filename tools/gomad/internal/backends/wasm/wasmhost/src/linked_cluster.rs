//! Bytes-owned cluster lifecycle execution over the helper's shared broker.
pub use crate::model::inspection::Handle;
use crate::{cluster, engine, linked, metrics::Metrics, namespace, protocol};
use cluster::handoff::{Broker, Member};
pub use cluster::handoff::{Config as BrokerConfig, WorldLimits};
pub use cluster::network::{Config as NetworkConfig, ModelV2 as NetworkModel};
use cluster::network::{Endpoint, Shared};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    io::{self, Cursor, Write},
    panic::{AssertUnwindSafe, catch_unwind},
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
        mpsc,
    },
    thread::{self, JoinHandle},
    time::Duration,
};
use wasmtime::{Engine, Error, InstancePre, Result, Store};

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum Action {
    Start,
    Restart,
    Wait,
    Crash,
    Stop,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Step {
    pub action: Action,
    pub handle: Handle,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Boundary {
    pub handle: Handle,
    pub action: Action,
    pub marker: Vec<u8>,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct CrashGroup {
    pub nodes: Vec<String>,
    pub marker: Vec<u8>,
}

pub struct Incarnation {
    pub handle: Handle,
    pub request: linked::Request,
    pub network: Option<NetworkModel>,
}

pub struct Request {
    pub nodes: Vec<String>,
    pub broker: BrokerConfig,
    pub network: Option<NetworkConfig>,
    pub incarnations: Vec<Incarnation>,
    pub steps: Vec<Step>,
    pub boundaries: Vec<Boundary>,
    pub crash_groups: Vec<CrashGroup>,
    pub wall_timeout: Duration,
    pub evidence_bytes: u64,
}

struct Sealed {
    execution: linked::Prepared,
    initial_namespace: namespace::Image,
    network: Option<NetworkModel>,
    transcript_limit: u64,
}

pub struct Prepared {
    incarnations: BTreeMap<Handle, Sealed>,
    broker: BrokerConfig,
    network: Option<NetworkConfig>,
    steps: Vec<Step>,
    boundaries: Vec<Boundary>,
    crash_groups: Vec<CrashGroup>,
    ids: Vec<String>,
    wall_timeout: Duration,
    evidence_bytes: u64,
    conditions: linked::Conditions,
    cancellation: linked::Cancellation,
}

impl Prepared {
    pub fn cancellation(&mut self, cancellation: linked::Cancellation) {
        self.cancellation = cancellation;
    }

    pub fn conditions(&mut self, conditions: linked::Conditions) {
        self.conditions = conditions;
    }

    /// # Errors
    /// A remaining caller budget can only shorten the original bound.
    pub fn bound_wall_timeout(&mut self, remaining: Duration) -> Result<()> {
        if remaining.is_zero() {
            return Err(Error::msg("execution wall budget exhausted"));
        }
        self.wall_timeout = self.wall_timeout.min(remaining);
        Ok(())
    }
}

pub struct IncarnationResult {
    pub handle: Handle,
    pub execution: linked::Execution,
    pub interrupted: bool,
}

pub struct Execution {
    pub outcome: protocol::Outcome,
    pub incarnations: Vec<IncarnationResult>,
    pub world: Vec<u8>,
    pub network: Vec<u8>,
    pub inspection: Vec<u8>,
    pub virtual_elapsed_nanos: u64,
    pub event_order: Option<Vec<u8>>,
}

fn valid_id(id: &str) -> bool {
    (1..=128).contains(&id.len())
        && id
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'-'))
}

fn lifecycle(request: &Request, handles: &BTreeSet<Handle>) -> Result<Vec<String>> {
    if request.steps.is_empty() || request.steps.len() > 65536 {
        return Err(Error::msg("cluster lifecycle capacity"));
    }
    let mut current: BTreeMap<String, (u64, bool)> = BTreeMap::new();
    let mut used = BTreeSet::new();
    for step in &request.steps {
        if !handles.contains(&step.handle) {
            return Err(Error::msg("unknown cluster lifecycle handle"));
        }
        let state = current
            .entry(step.handle.node.clone())
            .or_insert((0, false));
        match step.action {
            Action::Start | Action::Restart => {
                if state.1
                    || state.0.checked_add(1) != Some(step.handle.incarnation)
                    || (step.action == Action::Start) != (state.0 == 0)
                    || !used.insert(step.handle.clone())
                {
                    return Err(Error::msg("invalid cluster incarnation lifecycle"));
                }
                *state = (step.handle.incarnation, true);
            }
            Action::Wait | Action::Crash | Action::Stop => {
                if !state.1 || state.0 != step.handle.incarnation {
                    return Err(Error::msg("invalid cluster terminal lifecycle"));
                }
                state.1 = false;
            }
        }
    }
    if used != *handles || current.values().any(|state| state.1) || current.len() > 64 {
        return Err(Error::msg("incomplete cluster incarnation lifecycle"));
    }
    let mut boundaries = BTreeSet::new();
    for boundary in &request.boundaries {
        if !handles.contains(&boundary.handle)
            || !matches!(boundary.action, Action::Crash | Action::Stop)
            || boundary.marker.is_empty()
            || boundary.marker.len() > 4096
            || !boundaries.insert(boundary.handle.clone())
            || !request
                .steps
                .iter()
                .any(|step| step.handle == boundary.handle && step.action == boundary.action)
        {
            return Err(Error::msg("invalid cluster output boundary"));
        }
    }
    if request.crash_groups.len() > 1 {
        return Err(Error::msg("invalid cluster output crash group"));
    }
    for group in &request.crash_groups {
        let nodes: BTreeSet<_> = group.nodes.iter().collect();
        if nodes.len() < 2
            || nodes.len() != group.nodes.len()
            || group.marker.is_empty()
            || group.marker.len() > 4096
            || nodes.iter().any(|node| {
                !handles.contains(&Handle {
                    node: (*node).clone(),
                    incarnation: 1,
                })
            })
        {
            return Err(Error::msg("invalid cluster output crash group"));
        }
    }
    if request.nodes.is_empty() {
        return Ok(current.into_keys().collect());
    }
    let nodes: BTreeSet<_> = request.nodes.iter().cloned().collect();
    if nodes.len() > 64
        || nodes.len() != request.nodes.len()
        || nodes.iter().any(|node| !valid_id(node))
        || current.keys().any(|node| !nodes.contains(node))
    {
        return Err(Error::msg("invalid declared cluster nodes"));
    }
    Ok(nodes.into_iter().collect())
}

/// # Errors
/// Authenticates every incarnation's module, tape, namespace and storage before engine creation.
pub fn prepare(mut request: Request) -> Result<Prepared> {
    if request.incarnations.is_empty()
        || request.incarnations.len() > 65536
        || request.evidence_bytes == 0
        || request.evidence_bytes > 1 << 30
        || request.wall_timeout.is_zero()
    {
        return Err(Error::msg("invalid linked cluster bounds"));
    }
    request.broker.validate().map_err(Error::msg)?;
    if request.broker.pause_on_settle {
        return Err(Error::msg("linked cluster uses retained network faults"));
    }
    let mut handles = BTreeSet::new();
    for incarnation in &request.incarnations {
        if !valid_id(&incarnation.handle.node)
            || incarnation.handle.incarnation == 0
            || !handles.insert(incarnation.handle.clone())
        {
            return Err(Error::msg("invalid or duplicate cluster handle"));
        }
    }
    let ids = lifecycle(&request, &handles)?;
    if let Some(network) = &request.network {
        Shared::new(network, &ids).map_err(|failure| Error::msg(failure.message()))?;
    }
    let mut incarnations = BTreeMap::new();
    for incarnation in std::mem::take(&mut request.incarnations) {
        let model = incarnation
            .request
            .model
            .as_ref()
            .ok_or_else(|| Error::msg("cluster requires helper-owned model"))?;
        if model.configuration.seed != request.broker.seed {
            return Err(Error::msg("cluster model seed mismatch"));
        }
        match (&request.network, &incarnation.network) {
            (Some(_), Some(network)) => {
                cluster::network::validate_model(network)
                    .map_err(|failure| Error::msg(failure.message()))?;
                if !network.faults.is_empty() {
                    return Err(Error::msg("broker node local fault schedule"));
                }
            }
            (None, None) => {}
            _ => return Err(Error::msg("cluster boot network profile")),
        }
        let initial_namespace = incarnation
            .request
            .namespace
            .clone()
            .ok_or_else(|| Error::msg("cluster requires sealed namespace"))?;
        let transcript_limit = model.configuration.limits.transcript_bytes;
        let execution = linked::prepare(incarnation.request)?;
        incarnations.insert(
            incarnation.handle,
            Sealed {
                execution,
                initial_namespace,
                network: incarnation.network,
                transcript_limit,
            },
        );
    }
    Ok(Prepared {
        incarnations,
        broker: request.broker,
        network: request.network,
        steps: request.steps,
        boundaries: request.boundaries,
        crash_groups: request.crash_groups,
        ids,
        wall_timeout: request.wall_timeout,
        evidence_bytes: request.evidence_bytes,
        conditions: linked::Conditions::default(),
        cancellation: linked::Cancellation::default(),
    })
}

struct NoRemote;
impl Write for NoRemote {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            "linked cluster forbids transport callbacks",
        ))
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

struct Completed {
    result: IncarnationResult,
    image: Option<Vec<u8>>,
}

fn interrupted(outcome: &protocol::Outcome) -> bool {
    if matches!(outcome.termination, "crash" | "stop" | "storage_crash") {
        return true;
    }
    let dimension = outcome
        .message
        .strip_prefix("WASI capacity: ")
        .and_then(|message| message.split_once(' '))
        .map(|(dimension, _)| dimension);
    outcome.termination == "capacity"
        && matches!(
            dimension,
            Some(
                "event-ids"
                    | "events"
                    | "payload-bytes"
                    | "queued-events"
                    | "request-ids"
                    | "requests"
                    | "transition-bytes"
                    | "transition-sequences"
                    | "transitions"
            )
        )
        && outcome
            .local_model
            .as_ref()
            .is_some_and(|model| model["configured"] == true && model["finished"] == false)
}

#[expect(
    clippy::too_many_arguments,
    reason = "A fresh store binds its shared owners and exact incarnation identity"
)]
fn run_instance(
    engine: &Engine,
    pre: &InstancePre<engine::State>,
    mut sealed: Sealed,
    broker: &Arc<Broker>,
    network: Option<&Arc<Mutex<Shared>>>,
    handle: Handle,
    index: usize,
    slot: u64,
) -> Result<Completed> {
    let sink = linked::Sink::new(sealed.execution.request.limits.evidence_bytes);
    let mut model = sealed
        .execution
        .model
        .take()
        .ok_or_else(|| Error::msg("missing cluster model"))?;
    model.linked_evidence = Some(sink.clone());
    model.inspection.handle = handle.clone();
    model.inspection.recorder = broker.inspection.clone();
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    {
        model.event_owner = Some(handle.clone());
    }
    let endpoint = match (network, &sealed.network) {
        (Some(shared), Some(profile)) => Some(
            Endpoint::new(
                shared.clone(),
                &handle.node,
                handle.incarnation,
                slot,
                profile,
                sealed.initial_namespace.epoch_nanos,
                sealed.transcript_limit,
            )
            .map_err(|failure| Error::msg(failure.message()))?,
        ),
        _ => None,
    };
    let limits = &sealed.execution.request.limits;
    let mut state = engine::new_state(
        Box::new(Cursor::new(Vec::<u8>::new())),
        Box::new(NoRemote),
        limits.memory_bytes,
        limits.output_bytes,
        Some(model),
        sealed.execution.namespace.take(),
    );
    state.broker = Some(Member {
        broker: broker.clone(),
        index,
        incarnation: handle.incarnation,
        network: endpoint,
    });
    let mut store = Store::new(engine, state);
    store.limiter(|state| &mut state.limits);
    store.set_fuel(limits.fuel)?;
    store.set_epoch_deadline(1);
    let executed = broker
        .check(index, handle.incarnation)
        .map_err(|failure| {
            store
                .data_mut()
                .fail(failure.termination(), failure.message())
        })
        .and_then(|()| pre.instantiate(&mut store))
        .and_then(|instance| {
            instance
                .get_typed_func::<(), ()>(&mut store, "_start")?
                .call(&mut store, ())
        });
    let mut outcome = engine::terminal_outcome(&mut store, executed, limits.fuel)?;
    if matches!(
        outcome.termination,
        "inspected_prefix" | "instrumentation_divergence" | "debug_error" | "probe_fuel_limit"
    ) {
        broker.abort(outcome.termination, outcome.message.clone());
    }
    if outcome.termination == "infrastructure" && broker.inspection_stopped() {
        outcome.termination = "inspection_aborted";
        outcome.message = "cluster inspection stopped".into();
        if let Some(model) = &mut store.data_mut().model {
            model.inspection.sealed = true;
        }
    }
    let descriptor = sealed
        .execution
        .request
        .storage
        .as_ref()
        .map(linked::StorageInput::descriptor);
    let image = cluster::execution::record_terminal(store.data_mut(), descriptor, &mut outcome)
        .map_err(|(kind, message)| Error::msg(format!("{kind}: {message}")))?;
    drop(store);
    let evidence = sink.into_streams()?;
    Ok(Completed {
        result: IncarnationResult {
            handle,
            interrupted: interrupted(&outcome),
            execution: linked::Execution {
                outcome,
                evidence,
                event_order: None,
            },
        },
        image,
    })
}

struct Running {
    thread: JoinHandle<Result<Completed>>,
}

fn joined(running: Running) -> Result<Completed> {
    running
        .thread
        .join()
        .map_err(|_| Error::msg("cluster store panicked"))?
}

fn compile(
    engine: &Engine,
    prepared: &Prepared,
) -> Result<BTreeMap<String, Arc<InstancePre<engine::State>>>> {
    let mut modules = BTreeMap::new();
    for sealed in prepared.incarnations.values() {
        let request = &sealed.execution.request;
        if modules.contains_key(&request.module_sha256) {
            continue;
        }
        let module = engine::compile(engine, &request.module, &mut Metrics::default())
            .and_then(|module| engine::describe_module(&sealed.execution.execute, module))
            .map_err(|outcome| Error::msg(format!("{}: {}", outcome.termination, outcome.message)))?
            .0;
        let pre = engine::linker(engine, &module)?.instantiate_pre(&module)?;
        modules.insert(request.module_sha256.clone(), Arc::new(pre));
    }
    Ok(modules)
}

#[expect(
    clippy::too_many_lines,
    reason = "Lifecycle ordering shares the existing broker activation, run, restart and terminal transitions"
)]
fn execute_inner(
    engine: &Engine,
    mut prepared: Prepared,
    broker: &Arc<Broker>,
    network: Option<&Arc<Mutex<Shared>>>,
    timed_out: &AtomicBool,
    session: &crate::linked_conditions::Session,
) -> Result<Execution> {
    let modules = compile(engine, &prepared)?;
    let mut running: BTreeMap<Handle, Running> = BTreeMap::new();
    let mut completed = BTreeMap::new();
    let mut images: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    let first_failure = Arc::new(Mutex::new(None::<protocol::Outcome>));
    let mut outcome = protocol::Outcome::new("exit", "");
    outcome.exit_code = Some(0);
    broker.defer_starts();
    for group in &prepared.crash_groups {
        broker
            .barrier(&group.nodes, group.marker.clone())
            .map_err(|failure| Error::msg(failure.message()))?;
    }
    broker.start();
    let script = std::mem::take(&mut prepared.steps);
    let driven = (|| -> Result<()> {
        for step in script {
            if timed_out.load(Ordering::Acquire)
                || prepared.cancellation.cancelled()
                || broker.failure().is_some()
                || first_failure
                    .lock()
                    .map_err(|_| Error::msg("cluster failure lock poisoned"))?
                    .is_some()
            {
                break;
            }
            let index = prepared
                .ids
                .iter()
                .position(|id| id == &step.handle.node)
                .ok_or_else(|| Error::msg("unknown cluster node"))?;
            if matches!(step.action, Action::Start | Action::Restart) {
                let mut sealed = prepared
                    .incarnations
                    .remove(&step.handle)
                    .ok_or_else(|| Error::msg("missing cluster incarnation"))?;
                if step.action == Action::Restart
                    && let Some(storage) = &sealed.execution.request.storage
                {
                    let image = images
                        .get(&step.handle.node)
                        .ok_or_else(|| Error::msg("cluster restart lacks recovery image"))?;
                    let mut storage = storage.with_recovery_image(image.clone())?;
                    let mut namespace = namespace::Namespace::new(sealed.initial_namespace.clone())
                        .map_err(Error::new)?;
                    storage.attach(&mut namespace, sealed.transcript_limit)?;
                    sealed.execution.namespace = Some(namespace);
                    sealed.execution.request.storage = Some(storage);
                }
                let slot = if step.action == Action::Restart {
                    broker
                        .restart(index, step.handle.incarnation)
                        .map_err(|failure| Error::msg(failure.message()))?
                        .1
                } else {
                    broker
                        .activate(index)
                        .map_err(|failure| Error::msg(failure.message()))?;
                    index as u64 + 1
                };
                for boundary in prepared
                    .boundaries
                    .iter()
                    .filter(|boundary| boundary.handle == step.handle)
                {
                    let kind = if boundary.action == Action::Crash {
                        "crash"
                    } else {
                        "stop"
                    };
                    broker
                        .boundary(
                            index,
                            step.handle.incarnation,
                            kind,
                            boundary.marker.clone(),
                        )
                        .map_err(|failure| Error::msg(failure.message()))?;
                }
                let pre = modules[&sealed.execution.request.module_sha256].clone();
                let (engine, broker, network, handle) = (
                    engine.clone(),
                    broker.clone(),
                    network.cloned(),
                    step.handle.clone(),
                );
                let (session, conditions) = (session.clone(), prepared.conditions);
                let first_failure = first_failure.clone();
                let thread = thread::Builder::new()
                    .name(format!("instance-{}-{}", handle.node, handle.incarnation))
                    .stack_size(8 << 20)
                    .spawn(move || {
                        let _conditions = conditions.enter_shared(&session)?;
                        if !broker.acquire(index) && !broker.terminal_failed() {
                            return Ok(Completed {
                                result: IncarnationResult {
                                    handle,
                                    interrupted: false,
                                    execution: linked::Execution {
                                        outcome: protocol::Outcome::new(
                                            "inspection_aborted",
                                            "cluster inspection stopped",
                                        ),
                                        evidence: BTreeMap::new(),
                                        event_order: None,
                                    },
                                },
                                image: None,
                            });
                        }
                        let result = catch_unwind(AssertUnwindSafe(|| {
                            run_instance(
                                &engine,
                                &pre,
                                sealed,
                                &broker,
                                network.as_ref(),
                                handle,
                                index,
                                slot,
                            )
                        }))
                        .map_err(|_| Error::msg("cluster store panicked"))
                        .and_then(std::convert::identity);
                        match &result {
                            Ok(completed) => {
                                let observed = &completed.result.execution.outcome;
                                if observed.termination == "trap"
                                    || (observed.termination == "exit"
                                        && observed.exit_code != Some(0))
                                {
                                    let mut failure = first_failure
                                        .lock()
                                        .map_err(|_| Error::msg("cluster failure lock poisoned"))?;
                                    if failure.is_none() && broker.failure().is_none() {
                                        let mut outcome = protocol::Outcome::new(
                                            observed.termination,
                                            observed.message.clone(),
                                        );
                                        outcome.exit_code = observed.exit_code;
                                        *failure = Some(outcome);
                                        broker.abort("trap", "cluster application failed".into());
                                    }
                                }
                                broker.finish(index);
                            }
                            Err(error) => broker.abort("infrastructure", error.to_string()),
                        }
                        result
                    })?;
                running.insert(step.handle, Running { thread });
            } else {
                broker
                    .run(index, step.handle.incarnation)
                    .map_err(|failure| Error::msg(failure.message()))?;
                let observed = joined(
                    running
                        .remove(&step.handle)
                        .ok_or_else(|| Error::msg("cluster incarnation is not running"))?,
                )?;
                if !broker.inspection_stopped()
                    && ((step.action == Action::Crash
                        && !matches!(
                            observed.result.execution.outcome.termination,
                            "crash" | "storage_crash"
                        ))
                        || (step.action == Action::Stop
                            && observed.result.execution.outcome.termination != "stop"))
                {
                    return Err(Error::msg(
                        "cluster deterministic terminal boundary was not reached",
                    ));
                }
                if let Some(image) = observed.image {
                    images.insert(step.handle.node.clone(), image);
                }
                if step.action == Action::Wait
                    && observed.result.execution.outcome.termination != "inspected_prefix"
                    && (observed.result.execution.outcome.termination != "exit"
                        || observed.result.execution.outcome.exit_code != Some(0))
                {
                    outcome = protocol::Outcome::new(
                        observed.result.execution.outcome.termination,
                        observed.result.execution.outcome.message.clone(),
                    );
                    outcome.exit_code = observed.result.execution.outcome.exit_code;
                }
                completed.insert(step.handle, observed.result);
            }
        }
        Ok(())
    })();
    if let Err(error) = &driven {
        broker.abort("infrastructure", error.to_string());
    }
    if timed_out.load(Ordering::Acquire) {
        broker.abort("watchdog_timeout", "execution watchdog timeout".into());
    }
    let mut join_error = None;
    for (handle, pending) in running {
        match joined(pending) {
            Ok(observed) => {
                completed.insert(handle, observed.result);
            }
            Err(error) if driven.is_ok() && !broker.inspection_stopped() => {
                broker.abort("infrastructure", error.to_string());
                join_error.get_or_insert(error);
            }
            Err(_) => {}
        }
    }
    driven?;
    if let Some(error) = join_error {
        return Err(error);
    }
    if let Some((kind, message)) = broker.failure() {
        outcome = protocol::Outcome::new(kind, message);
    } else if let Some(failure) = first_failure
        .lock()
        .map_err(|_| Error::msg("cluster failure lock poisoned"))?
        .take()
    {
        outcome = failure;
    }
    let world = if matches!(outcome.termination, "exit" | "trap") {
        broker
            .world_wire()
            .map_err(|failure| Error::msg(failure.message()))?
    } else {
        Vec::new()
    };
    let inspection = {
        let mut recorder = broker
            .inspection
            .lock()
            .map_err(|_| Error::msg("inspection recorder lock poisoned"))?;
        if recorder.observed && !recorder.evidence.incarnations.is_empty() {
            let complete = !recorder.sealed;
            recorder.bytes(complete)?
        } else {
            Vec::new()
        }
    };
    let network = completed
        .values()
        .rev()
        .find_map(|value| value.execution.evidence.get("network"))
        .cloned()
        .unwrap_or_default();
    let retained = completed
        .values()
        .flat_map(|value| value.execution.evidence.values())
        .try_fold(0u64, |total, value| total.checked_add(value.len() as u64))
        .and_then(|total| {
            total.checked_add((world.len() + network.len() + inspection.len()) as u64)
        });
    if retained.is_none_or(|bytes| bytes > prepared.evidence_bytes) {
        return Err(Error::msg("linked cluster evidence capacity"));
    }
    Ok(Execution {
        outcome,
        incarnations: completed.into_values().collect(),
        world,
        network,
        inspection,
        virtual_elapsed_nanos: broker.current(),
        event_order: None,
    })
}

/// # Errors
/// Executes a fully authenticated lifecycle with fresh stores and one shared broker and engine.
pub fn execute_prepared(prepared: Prepared) -> Result<Execution> {
    let cancellation = prepared.cancellation.clone();
    if cancellation.cancelled() {
        return Ok(cancelled_execution());
    }
    let mut conditions = prepared.conditions.enter()?;
    let session = conditions.session();
    let engine = Engine::new(&engine::configuration(true))?;
    let network = prepared
        .network
        .as_ref()
        .map(|configuration| {
            Shared::new(configuration, &prepared.ids)
                .map(|shared| Arc::new(Mutex::new(shared)))
                .map_err(|failure| Error::msg(failure.message()))
        })
        .transpose()?;
    let broker = Arc::new(
        Broker::new(&prepared.broker, &prepared.ids, network.clone(), None).map_err(Error::msg)?,
    );
    let _registration = cancellation.attach(&engine, Some(broker.clone()))?;
    let timed_out = AtomicBool::new(false);
    thread::scope(|scope| {
        let (done, completed) = mpsc::sync_channel::<()>(0);
        let (engine, broker, timed_out, timeout) =
            (&engine, &broker, &timed_out, prepared.wall_timeout);
        let (watch_conditions, watch_session) = (prepared.conditions, session.clone());
        let watcher = scope.spawn(move || -> Result<()> {
            let _conditions = watch_conditions.enter_shared(&watch_session)?;
            if completed.recv_timeout(timeout) == Err(mpsc::RecvTimeoutError::Timeout) {
                timed_out.store(true, Ordering::Release);
                engine.increment_epoch();
                broker.abort("watchdog_timeout", "execution watchdog timeout".into());
            }
            Ok(())
        });
        let mut result = execute_inner(
            engine,
            prepared,
            broker,
            network.as_ref(),
            timed_out,
            &session,
        );
        drop(done);
        watcher
            .join()
            .map_err(|_| Error::msg("cluster watchdog panicked"))??;
        if cancellation.cancelled() && result.is_err() {
            result = Ok(cancelled_execution());
        }
        if timed_out.load(Ordering::Acquire)
            && let Ok(execution) = &mut result
        {
            execution.outcome =
                protocol::Outcome::new("watchdog_timeout", "execution watchdog timeout");
        }
        if let Ok(execution) = &mut result {
            cancellation.apply(&mut execution.outcome);
            execution.event_order = conditions.finish();
        }
        result
    })
}

fn cancelled_execution() -> Execution {
    Execution {
        outcome: linked::Cancellation::execution().outcome,
        incarnations: Vec::new(),
        world: Vec::new(),
        network: Vec::new(),
        inspection: Vec::new(),
        virtual_elapsed_nanos: 0,
        event_order: None,
    }
}
