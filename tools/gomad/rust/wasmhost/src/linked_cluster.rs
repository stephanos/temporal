//! Bytes-owned cluster lifecycle execution over the helper's shared broker.
pub use crate::model::inspection::Handle;
use crate::{cluster, engine, linked, metrics::Metrics, namespace, protocol};
use cluster::handoff::{Broker, Command, Member};
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
        atomic::{AtomicBool, AtomicU64, Ordering},
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

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
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
    recovery_source: Option<(Handle, String, u64)>,
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct LinkControl {
    pub action: String,
    pub from: String,
    pub to: String,
    pub delay: u64,
    pub symmetric: bool,
}

pub enum CleanupMode {
    Shutdown,
    Error(String),
    ContextCancellation,
}

struct Observation {
    marker: Vec<u8>,
    send: mpsc::SyncSender<()>,
    receive: mpsc::Receiver<()>,
}

struct Cleanup {
    after: Option<Handle>,
    mode: CleanupMode,
}

#[derive(Clone, Default)]
pub struct Activity {
    launched_stores: Arc<AtomicU64>,
    remote_callback_attempts: Arc<AtomicU64>,
}

#[derive(Serialize)]
pub struct ActivitySnapshot {
    pub launched_stores: u64,
    pub remote_callback_attempts: u64,
}

impl Activity {
    #[must_use]
    pub fn snapshot(&self) -> ActivitySnapshot {
        ActivitySnapshot {
            launched_stores: self.launched_stores.load(Ordering::Acquire),
            remote_callback_attempts: self.remote_callback_attempts.load(Ordering::Acquire),
        }
    }
}

pub struct Prepared {
    activity: Activity,
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
    #[cfg(feature = "determinism-fixture")]
    fixture_fault: Option<(linked::FixtureFault, Option<Handle>)>,
    cancellation: linked::Cancellation,
    simulation: Option<crate::linked_simulation::Shared>,
    observations: BTreeMap<Handle, Observation>,
    cleanup: Option<Cleanup>,
    initial_links: Vec<LinkControl>,
    action_limit: usize,
    #[cfg(test)]
    shutdown_reinsertion: Option<(bool, Arc<AtomicBool>)>,
    committed_temporal_reopens: bool,
}

impl Prepared {
    #[must_use]
    pub fn compile_cache_directory(&self) -> Option<&str> {
        self.incarnations
            .values()
            .next()
            .and_then(|sealed| sealed.execution.compile_cache_directory())
    }

    /// # Errors
    /// Refuses an invalid host compile-cache selection.
    pub fn compile_cache(&mut self, directory: Option<&str>) -> Result<()> {
        for sealed in self.incarnations.values_mut() {
            sealed.execution.compile_cache(directory)?;
        }
        Ok(())
    }

    #[must_use]
    pub fn activity(&self) -> Activity {
        self.activity.clone()
    }

    /// Retain the actual precrash state for the registered SQLite reference oracle.
    /// # Errors
    /// Refuses another incarnation graph or a missing enumerated storage owner.
    pub fn sqlite_crash_reference(&mut self) -> Result<()> {
        self.native_crash_reference("sqlite", 2, &[1, 2, 3])
    }

    /// Retain the actual precrash state for the registered Temporal recovery oracle.
    /// # Errors
    /// Refuses another incarnation graph or a missing enumerated storage owner.
    pub fn temporal_crash_reference(&mut self) -> Result<()> {
        let expected: &[u64] = if self.incarnations.len() == 3 {
            &[1, 2, 3]
        } else {
            &[1, 2]
        };
        self.native_crash_reference("temporal", 1, expected)
    }

    fn native_crash_reference(&mut self, node: &str, source: u64, expected: &[u64]) -> Result<()> {
        let handles = self
            .incarnations
            .keys()
            .map(|handle| (handle.node.as_str(), handle.incarnation))
            .collect::<Vec<_>>();
        if handles
            != expected
                .iter()
                .map(|incarnation| (node, *incarnation))
                .collect::<Vec<_>>()
        {
            return Err(Error::msg(
                "native crash reference requires its registered incarnation graph",
            ));
        }
        self.incarnations
            .iter_mut()
            .find(|(handle, _)| handle.incarnation == source)
            .and_then(|(_, sealed)| sealed.execution.request.storage.as_mut())
            .ok_or_else(|| Error::msg("native crash reference requires durable storage"))?
            .retain_native_crash_reference()
    }

    /// Bind the original committed-workflow variant: two fresh stores recover
    /// from the identical actual image produced by temporal/1.
    /// # Errors
    /// Refuses any other handle set or missing durable storage owner.
    pub fn committed_temporal_reopens(&mut self) -> Result<()> {
        let handles = self
            .incarnations
            .keys()
            .map(|handle| (handle.node.as_str(), handle.incarnation))
            .collect::<Vec<_>>();
        if handles != [("temporal", 1), ("temporal", 2), ("temporal", 3)]
            || self
                .incarnations
                .values()
                .any(|sealed| sealed.execution.request.storage.is_none())
        {
            return Err(Error::msg(
                "committed Temporal recovery requires exactly its three durable incarnations",
            ));
        }
        self.committed_temporal_reopens = true;
        Ok(())
    }

    /// The cluster wall bound the host's process-wide deadline backstop arms.
    #[must_use]
    pub fn wall_timeout(&self) -> Duration {
        self.wall_timeout
    }

    /// # Errors
    /// Installs a controller observation without binding a guest output to a revocation.
    pub fn observe_revocation(&mut self, handle: Handle, marker: Vec<u8>) -> Result<()> {
        if !self.incarnations.contains_key(&handle)
            || marker.is_empty()
            || marker.len() > 4096
            || self.observations.contains_key(&handle)
        {
            return Err(Error::msg("invalid linked controller observation"));
        }
        let (send, receive) = mpsc::sync_channel(1);
        self.observations.insert(
            handle,
            Observation {
                marker,
                send,
                receive,
            },
        );
        Ok(())
    }

    /// # Errors
    /// Cleanup joins every started store without adding authored lifecycle steps.
    pub fn cleanup_after_starts(&mut self, mode: CleanupMode) -> Result<()> {
        if self.cleanup.is_some()
            || self
                .steps
                .iter()
                .take(self.action_limit.min(self.steps.len()))
                .any(|step| !matches!(step.action, Action::Start))
        {
            return Err(Error::msg("invalid cleanup lifecycle"));
        }
        self.cleanup = Some(Cleanup { after: None, mode });
        Ok(())
    }

    /// # Errors
    /// Drives a real acknowledgment before ending a scenario and joining its live stores.
    pub fn cleanup_after_output(
        &mut self,
        handle: Handle,
        marker: Vec<u8>,
        mode: CleanupMode,
    ) -> Result<()> {
        self.cleanup_after_starts(mode)?;
        self.observe_revocation(handle.clone(), marker)?;
        self.cleanup
            .as_mut()
            .ok_or_else(|| Error::msg("missing cleanup"))?
            .after = Some(handle);
        Ok(())
    }

    /// # Errors
    /// A scenario action budget is checked before the next action can mutate a model.
    pub fn action_limit(&mut self, maximum: usize) -> Result<()> {
        if maximum == 0 || maximum > 65536 {
            return Err(Error::msg("invalid cluster action limit"));
        }
        self.action_limit = maximum;
        Ok(())
    }

    /// # Errors
    /// Controller links use the existing bounded fault-action capacity.
    pub fn link_before_start(&mut self, link: LinkControl) -> Result<()> {
        if self.initial_links.len() >= 4096 {
            return Err(Error::msg("cluster controller link capacity"));
        }
        self.initial_links.push(link);
        Ok(())
    }

    pub(crate) fn simulation(
        &mut self,
        controller: crate::linked_simulation::Shared,
    ) -> Result<()> {
        if controller
            .lock()
            .map_err(|_| Error::msg("simulation exploration controller poisoned"))?
            .seed()
            != self.broker.seed
        {
            return Err(Error::msg(
                "simulation exploration seed does not match cluster",
            ));
        }
        for sealed in self.incarnations.values_mut() {
            if let Some(storage) = &mut sealed.execution.request.storage {
                storage.set_simulation(controller.clone());
            }
        }
        self.simulation = Some(controller);
        Ok(())
    }

    pub fn cancellation(&mut self, cancellation: linked::Cancellation) {
        self.cancellation = cancellation;
    }

    #[cfg(feature = "determinism-fixture")]
    /// # Errors
    /// Refuses faults without the fixture role or a matching incarnation owner.
    pub fn fixture_fault(
        &mut self,
        fault: linked::FixtureFault,
        owner: Option<Handle>,
    ) -> Result<()> {
        let owner = owner.or_else(|| self.incarnations.keys().next().cloned());
        if owner
            .as_ref()
            .is_some_and(|owner| !self.incarnations.contains_key(owner))
        {
            return Err(Error::msg("unknown fixture fault logical owner"));
        }
        self.fixture_fault = Some((fault, owner));
        Ok(())
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
    pub executed_steps: Vec<Step>,
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
    if used != *handles || current.len() > 64 {
        return Err(Error::msg("incomplete cluster incarnation lifecycle"));
    }
    let mut boundaries = BTreeSet::new();
    for boundary in &request.boundaries {
        if !handles.contains(&boundary.handle)
            || !matches!(boundary.action, Action::Crash | Action::Stop)
            || boundary.marker.is_empty()
            || boundary.marker.len() > 4096
            || !boundaries.insert(boundary.handle.clone())
            || !request.steps.iter().any(|step| {
                step.handle == boundary.handle
                    && (step.action == Action::Wait || step.action == boundary.action)
            })
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
                recovery_source: None,
            },
        );
    }
    Ok(Prepared {
        activity: Activity::default(),
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
        #[cfg(feature = "determinism-fixture")]
        fixture_fault: None,
        cancellation: linked::Cancellation::default(),
        simulation: None,
        observations: BTreeMap::new(),
        cleanup: None,
        initial_links: Vec::new(),
        action_limit: 65536,
        #[cfg(test)]
        shutdown_reinsertion: None,
        committed_temporal_reopens: false,
    })
}

struct NoRemote(Activity);
impl Write for NoRemote {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        self.0
            .remote_callback_attempts
            .fetch_add(1, Ordering::Release);
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

fn instance_state(
    sealed: &mut Sealed,
    broker: &Arc<Broker>,
    network: Option<&Arc<Mutex<Shared>>>,
    handle: &Handle,
    index: usize,
    slot: u64,
    activity: &Activity,
) -> Result<(engine::State, linked::Sink)> {
    let limits = &sealed.execution.request.limits;
    let sink = linked::Sink::new(limits.evidence_bytes)
        .with_storage_bounds(sealed.execution.storage_bounds.clone());
    let mut model = sealed
        .execution
        .model
        .take()
        .ok_or_else(|| Error::msg("missing cluster model"))?;
    model.linked_evidence = Some(sink.clone());
    model.inspection.handle = handle.clone();
    #[cfg(feature = "determinism-fixture")]
    model.bind_fixture_control();
    model.inspection.recorder = broker.inspection.clone();
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    {
        model.event_owner = Some(handle.clone());
    }
    let mut endpoint = match (network, &sealed.network) {
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
    if let Some(endpoint) = &mut endpoint {
        endpoint
            .limit_transition_bytes(broker.transition_bytes())
            .map_err(|failure| Error::msg(failure.message()))?;
    }
    let mut state = engine::new_state(
        Box::new(Cursor::new(Vec::<u8>::new())),
        Box::new(NoRemote(activity.clone())),
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
    Ok((state, sink))
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
    activity: &Activity,
) -> Result<Completed> {
    let authorization_started = std::time::Instant::now();
    broker.wait_terminal(index);
    let (state, sink) =
        instance_state(&mut sealed, broker, network, &handle, index, slot, activity)?;
    let limits = &sealed.execution.request.limits;
    let mut store = Store::new(engine, state);
    activity.launched_stores.fetch_add(1, Ordering::Release);
    store.limiter(|state| &mut state.limits);
    #[cfg(not(feature = "diagnostic-fuel-off"))]
    store.set_fuel(limits.fuel)?;
    store.set_epoch_deadline(1);
    let mut metrics = Metrics::default();
    let executed = broker
        .check(index, handle.incarnation)
        .map_err(|failure| {
            store
                .data_mut()
                .fail(failure.termination(), failure.message())
        })
        .and_then(|()| {
            #[cfg(feature = "diagnostic-profile")]
            let _span = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Instantiate);
            let started = std::time::Instant::now();
            let instance = pre.instantiate(&mut store);
            metrics.instantiate_nanos = crate::metrics::elapsed(started);
            instance
        })
        .and_then(|instance| {
            let start = instance.get_typed_func::<(), ()>(&mut store, "_start")?;
            #[cfg(feature = "diagnostic-profile")]
            let _span = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Execute);
            let started = std::time::Instant::now();
            let result = start.call(&mut store, ());
            metrics.execute_nanos = crate::metrics::elapsed(started);
            result
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
    broker.wait_terminal(index);
    let image = cluster::execution::record_terminal(store.data_mut(), descriptor, &mut outcome)
        .map_err(|(kind, message)| Error::msg(format!("{kind}: {message}")))?;
    drop(store);
    metrics.sample_process();
    outcome.metrics = Some(Box::new(metrics));
    let mut evidence = sink.into_streams()?;
    if let Some((source_handle, image_sha256, image_bytes)) = &sealed.recovery_source {
        evidence.insert("recovery_image_input".into(), serde_json::to_vec(&serde_json::json!({"source_handle":source_handle,"image_sha256":image_sha256,"image_bytes":image_bytes}))?);
    }
    drop(sealed);
    let authorization_to_terminal_nanos = Some(crate::metrics::elapsed(authorization_started));
    Ok(Completed {
        result: IncarnationResult {
            handle,
            interrupted: interrupted(&outcome),
            execution: linked::Execution {
                terminal_monotonic_nanos: None,
                admission_terminal_to_return_nanos: None,
                timing: None,
                #[cfg(feature = "guest-profile")]
                guest_profile_error: None,
                #[cfg(feature = "diagnostic-profile")]
                external_owner_attempts: 0,
                authorization_to_terminal_nanos,
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
    metrics: &mut Metrics,
) -> Result<BTreeMap<String, Arc<InstancePre<engine::State>>>> {
    metrics.cache_hit = true;
    let mut modules = BTreeMap::new();
    for sealed in prepared.incarnations.values() {
        let request = &sealed.execution.request;
        if modules.contains_key(&request.module_sha256) {
            continue;
        }
        let mut observed = Metrics::default();
        let module = linked::compile_module(
            engine,
            &sealed.execution.execute,
            &request.module,
            &mut observed,
        )
        .map_err(|outcome| Error::msg(format!("{}: {}", outcome.termination, outcome.message)))?
        .0;
        metrics.compile_nanos = metrics
            .compile_nanos
            .checked_add(observed.compile_nanos)
            .ok_or_else(|| Error::msg("cluster compile timing overflow"))?;
        metrics.cache_load_nanos = metrics
            .cache_load_nanos
            .checked_add(observed.cache_load_nanos)
            .ok_or_else(|| Error::msg("cluster cache load timing overflow"))?;
        metrics.cache_write_nanos = metrics
            .cache_write_nanos
            .checked_add(observed.cache_write_nanos)
            .ok_or_else(|| Error::msg("cluster cache write timing overflow"))?;
        metrics.cache_hit &= observed.cache_hit;
        let pre = engine::linker(engine, &module)?.instantiate_pre(&module)?;
        modules.insert(request.module_sha256.clone(), Arc::new(pre));
    }
    Ok(modules)
}

#[derive(Clone, Debug)]
pub struct LifecyclePhase {
    pub incarnation: u64,
    pub handles: Vec<Handle>,
}

pub trait LifecycleObserver: Sync {
    /// # Errors
    /// Refuses admission before the first store of this phase starts.
    fn before(&self, phase: &LifecyclePhase) -> std::result::Result<(), String>;
    /// # Errors
    /// Refuses publication after all stores of this phase have joined.
    fn after(&self, phase: &LifecyclePhase) -> std::result::Result<(), String>;
}

struct PhaseObservations<'a> {
    observer: Option<&'a dyn LifecycleObserver>,
    phases: BTreeMap<u64, LifecyclePhase>,
    started: BTreeSet<u64>,
    finished: BTreeSet<u64>,
}
impl<'a> PhaseObservations<'a> {
    fn new(prepared: &Prepared, observer: Option<&'a dyn LifecycleObserver>) -> Self {
        let mut phases = BTreeMap::<u64, LifecyclePhase>::new();
        if observer.is_some() {
            for handle in prepared.incarnations.keys() {
                phases
                    .entry(handle.incarnation)
                    .or_insert_with(|| LifecyclePhase {
                        incarnation: handle.incarnation,
                        handles: Vec::new(),
                    })
                    .handles
                    .push(handle.clone());
            }
        }
        Self {
            observer,
            phases,
            started: BTreeSet::new(),
            finished: BTreeSet::new(),
        }
    }
    fn before(&mut self, handle: &Handle) -> Result<()> {
        if let Some(observer) = self.observer
            && self.started.insert(handle.incarnation)
        {
            observer
                .before(&self.phases[&handle.incarnation])
                .map_err(Error::msg)?;
        }
        Ok(())
    }
    fn after(
        &mut self,
        handle: &Handle,
        completed: &BTreeMap<Handle, IncarnationResult>,
    ) -> Result<()> {
        if let Some(observer) = self.observer {
            let phase = &self.phases[&handle.incarnation];
            if phase
                .handles
                .iter()
                .all(|handle| completed.contains_key(handle))
                && self.finished.insert(handle.incarnation)
            {
                observer.after(phase).map_err(Error::msg)?;
            }
        }
        Ok(())
    }
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
    phase_listener: Option<&dyn LifecycleObserver>,
) -> Result<Execution> {
    let mut phase_observations = PhaseObservations::new(&prepared, phase_listener);
    let execution_started = std::time::Instant::now();
    let mut metrics = Metrics::default();
    let modules = compile(engine, &prepared, &mut metrics)?;
    metrics.prepare_nanos = crate::metrics::elapsed(execution_started);
    let final_network_source = prepared
        .ids
        .iter()
        .enumerate()
        .filter_map(|(index, id)| {
            let source = prepared.incarnations.get(&Handle {
                node: id.clone(),
                incarnation: 1,
            })?;
            Some((
                source.transcript_limit,
                index,
                id.clone(),
                source.network.clone()?,
                source.initial_namespace.epoch_nanos,
            ))
        })
        .max_by_key(|source| source.0);
    let mut running: BTreeMap<Handle, Running> = BTreeMap::new();
    let mut completed = BTreeMap::new();
    let mut images: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    let mut committed_image: Option<Vec<u8>> = None;
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
    for link in std::mem::take(&mut prepared.initial_links) {
        broker
            .command(Command::Link {
                action: link.action,
                from: link.from,
                to: link.to,
                delay: link.delay,
                symmetric: link.symmetric,
            })
            .map_err(|error| Error::msg(error.message()))?;
    }
    let mut script = std::collections::VecDeque::from(std::mem::take(&mut prepared.steps));
    let mut executed_steps = Vec::new();
    let mut action_capacity = false;
    let driven = (|| -> Result<()> {
        while !script.is_empty() {
            if timed_out.load(Ordering::Acquire)
                || prepared.cancellation.cancelled()
                || broker.inspection_stopped()
                || broker.failure().is_some()
                || first_failure
                    .lock()
                    .map_err(|_| Error::msg("cluster failure lock poisoned"))?
                    .is_some()
            {
                break;
            }
            if executed_steps.len() >= prepared.action_limit {
                action_capacity = true;
                break;
            }
            let step = crate::linked_simulation::next_step(
                &mut script,
                prepared.simulation.as_ref(),
                broker.current(),
            )?;
            if matches!(step.action, Action::Start | Action::Restart) {
                phase_observations.before(&step.handle)?;
            }
            executed_steps.push(step.clone());
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
                    let image = restart_image(
                        &images,
                        committed_image.as_ref(),
                        prepared.committed_temporal_reopens,
                        &step.handle.node,
                    )?;
                    if prepared.committed_temporal_reopens {
                        use sha2::Digest as _;
                        sealed.recovery_source = Some((
                            Handle {
                                node: "temporal".into(),
                                incarnation: 1,
                            },
                            format!("{:x}", sha2::Sha256::digest(image)),
                            image.len() as u64,
                        ));
                    }
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
                if let Some(observation) = prepared.observations.get(&step.handle) {
                    broker
                        .observe_output(
                            index,
                            step.handle.incarnation,
                            observation.marker.clone(),
                            observation.send.clone(),
                        )
                        .map_err(|error| Error::msg(error.message()))?;
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
                let activity = prepared.activity.clone();
                let thread = thread::Builder::new()
                    .name(format!("instance-{}-{}", handle.node, handle.incarnation))
                    .stack_size(8 << 20)
                    .spawn(move || {
                        let _conditions = conditions.enter_shared(&session)?;
                        #[cfg(feature = "diagnostic-profile")]
                        let _profile = session.profile(slot, handle.incarnation)?;
                        if !broker.acquire(index) && !broker.terminal_failed() {
                            return Ok(Completed {
                                result: IncarnationResult {
                                    handle,
                                    interrupted: false,
                                    execution: linked::Execution {
                                        terminal_monotonic_nanos: None,
                                        admission_terminal_to_return_nanos: None,
                                        timing: None,
                                        #[cfg(feature = "guest-profile")]
                                        guest_profile_error: None,
                                        #[cfg(feature = "diagnostic-profile")]
                                        external_owner_attempts: 0,
                                        authorization_to_terminal_nanos: None,
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
                                &activity,
                            )
                        }))
                        .map_err(|_| Error::msg("cluster store panicked"))
                        .and_then(std::convert::identity);
                        match &result {
                            Ok(completed) => {
                                let observed = &completed.result.execution.outcome;
                                if let Some((kind, message)) =
                                    cluster::execution::terminal_failure(observed)
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
                                    }
                                    broker.abort_terminal(index, kind, message);
                                }
                            }
                            Err(error) => {
                                broker.abort_terminal(index, "infrastructure", error.to_string());
                            }
                        }
                        broker.finish(index);
                        result
                    })?;
                running.insert(step.handle, Running { thread });
            } else {
                broker
                    .run(index, step.handle.incarnation)
                    .map_err(|failure| Error::msg(failure.message()))?;
                if let Some(observation) = prepared.observations.remove(&step.handle) {
                    observation
                        .receive
                        .recv_timeout(
                            prepared
                                .wall_timeout
                                .saturating_sub(execution_started.elapsed()),
                        )
                        .map_err(|_| Error::msg("controller output observation abandoned"))?;
                    let kind = match step.action {
                        Action::Crash => "crash",
                        Action::Stop => "stop",
                        _ => return Err(Error::msg("observation requires unbound revocation")),
                    };
                    broker
                        .command(Command::Revoke {
                            index,
                            incarnation: step.handle.incarnation,
                            kind,
                        })
                        .map_err(|error| Error::msg(error.message()))?;
                    broker
                        .run(index, step.handle.incarnation)
                        .map_err(|error| Error::msg(error.message()))?;
                }
                let mut observed = joined(
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
                    if prepared.committed_temporal_reopens && step.handle.incarnation == 1 {
                        if image.len() as u64 > prepared.evidence_bytes {
                            return Err(Error::msg(
                                "committed image exceeds existing evidence bound",
                            ));
                        }
                        observed
                            .result
                            .execution
                            .evidence
                            .insert("recovery_image_primary".into(), image.clone());
                        committed_image = Some(image.clone());
                    }
                    images.insert(step.handle.node.clone(), image);
                }
                if step.action == Action::Wait
                    && broker.output_terminal(index, step.handle.incarnation)
                        != Some(observed.result.execution.outcome.termination)
                    && !matches!(
                        observed.result.execution.outcome.termination,
                        "inspected_prefix" | "storage_crash"
                    )
                    && (observed.result.execution.outcome.termination != "exit"
                        || observed.result.execution.outcome.exit_code != Some(0))
                {
                    outcome = protocol::Outcome::new(
                        observed.result.execution.outcome.termination,
                        observed.result.execution.outcome.message.clone(),
                    );
                    outcome.exit_code = observed.result.execution.outcome.exit_code;
                }
                let handle = step.handle;
                completed.insert(handle.clone(), observed.result);
                phase_observations.after(&handle, &completed)?;
            }
        }
        if action_capacity && prepared.cleanup.is_none() {
            prepared.cleanup = Some(Cleanup {
                after: None,
                mode: CleanupMode::Shutdown,
            });
        }
        if let Some(cleanup) = prepared.cleanup.take() {
            if let Some(handle) = cleanup.after {
                let index = prepared
                    .ids
                    .iter()
                    .position(|id| id == &handle.node)
                    .ok_or_else(|| Error::msg("unknown cleanup acknowledgment owner"))?;
                let observation = prepared
                    .observations
                    .remove(&handle)
                    .ok_or_else(|| Error::msg("missing cleanup observation"))?;
                broker
                    .run(index, handle.incarnation)
                    .map_err(|error| Error::msg(error.message()))?;
                observation
                    .receive
                    .recv_timeout(
                        prepared
                            .wall_timeout
                            .saturating_sub(execution_started.elapsed()),
                    )
                    .map_err(|_| Error::msg("cleanup acknowledgment abandoned"))?;
            }
            #[cfg(test)]
            if let Some((reverse, verified)) = prepared.shutdown_reinsertion.take() {
                let states = std::mem::take(&mut running);
                let mut entries: Vec<_> = states.into_iter().collect();
                if reverse {
                    entries.reverse();
                }
                let live = entries.len() == 2
                    && completed.is_empty()
                    && entries
                        .iter()
                        .all(|(_, pending)| !pending.thread.is_finished());
                for (handle, pending) in entries {
                    running.insert(handle, pending);
                }
                if !live {
                    return Err(Error::msg(
                        "root35 handle finished before shutdown reinsertion",
                    ));
                }
                verified.store(true, Ordering::Release);
            }
            match cleanup.mode {
                CleanupMode::ContextCancellation => prepared.cancellation.cancel(),
                mode => {
                    let reason = match mode {
                        CleanupMode::Shutdown => None,
                        CleanupMode::Error(message) => Some(protocol::bounded_message(message)),
                        CleanupMode::ContextCancellation => {
                            return Err(Error::msg("invalid cleanup dispatch"));
                        }
                    };
                    stop_live(
                        &prepared.ids,
                        broker,
                        &mut running,
                        &mut completed,
                        &Error::msg(reason.clone().unwrap_or_else(|| "cluster shutdown".into())),
                    );
                    if action_capacity {
                        outcome = protocol::Outcome::new(
                            "capacity",
                            format!(
                                "scenario action capacity limit={} used={} delta=1",
                                prepared.action_limit,
                                executed_steps.len()
                            ),
                        );
                    } else if let Some(reason) = reason {
                        outcome = protocol::Outcome::new("scenario_failure", reason);
                    }
                }
            }
        }
        Ok(())
    })();
    let driven = driven.and_then(|()| {
        if running.is_empty()
            || broker.failure().is_some()
            || broker.inspection_stopped()
            || first_failure
                .lock()
                .map_err(|_| Error::msg("cluster failure lock poisoned"))?
                .is_some()
        {
            Ok(())
        } else {
            Err(Error::msg("incomplete cluster incarnation lifecycle"))
        }
    });
    let divergent = prepared.simulation.as_ref().is_some_and(|controller| {
        controller
            .lock()
            .is_ok_and(|controller| controller.divergence().is_some())
    });
    if let Err(error) = &driven {
        if divergent && !timed_out.load(Ordering::Acquire) && broker.failure().is_none() {
            stop_live(&prepared.ids, broker, &mut running, &mut completed, error);
        } else {
            broker.abort("infrastructure", error.to_string());
        }
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
    if let Err(error) = driven
        && !divergent
    {
        return Err(error);
    }
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
    let world = if matches!(outcome.termination, "exit" | "trap" | "capacity")
        || (cfg!(feature = "diagnostic-profile") && outcome.termination == "watchdog_timeout")
    {
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
    let network = if matches!(outcome.termination, "exit" | "trap" | "capacity") {
        if let Some(shared) = network {
            let (transcript_limit, index, id, profile, epoch) = final_network_source
                .ok_or_else(|| Error::msg("missing final cluster network source"))?;
            let (incarnation, slot) = broker.incarnation_slot(index);
            Endpoint::new(
                shared.clone(),
                &id,
                incarnation,
                slot,
                &profile,
                epoch,
                transcript_limit,
            )
            .and_then(|endpoint| endpoint.evidence())
            .map_err(|failure| Error::msg(failure.message()))?
        } else {
            Vec::new()
        }
    } else {
        Vec::new()
    };
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
    metrics.sample_process();
    outcome.metrics = Some(Box::new(metrics));
    Ok(Execution {
        outcome,
        executed_steps,
        incarnations: completed.into_values().collect(),
        world,
        network,
        inspection,
        virtual_elapsed_nanos: broker.current(),
        event_order: None,
    })
}

/// Go `ClusterSession.Shutdown` for a session that ends early without a
/// cluster failure: revoke every live incarnation with `stop` in
/// (ID, Incarnation) order, then drive and join each in that order, so their
/// stop terminals and the retained World evidence never depend on start or
/// thread order. An incarnation that already finished is only joined; any
/// other refusal falls back to the abort.
fn stop_live(
    ids: &[String],
    broker: &Arc<Broker>,
    running: &mut BTreeMap<Handle, Running>,
    completed: &mut BTreeMap<Handle, IncarnationResult>,
    error: &Error,
) {
    let index = |handle: &Handle| ids.iter().position(|id| id == &handle.node);
    for handle in running.keys() {
        let revoked = index(handle).is_some_and(|index| {
            broker
                .command(Command::Revoke {
                    index,
                    incarnation: handle.incarnation,
                    kind: "stop",
                })
                .is_ok()
                || broker.finished(index)
        });
        if !revoked {
            broker.abort("infrastructure", error.to_string());
            return;
        }
    }
    let handles: Vec<Handle> = running.keys().cloned().collect();
    for handle in handles {
        let driven =
            index(&handle).is_some_and(|index| broker.run(index, handle.incarnation).is_ok());
        if !driven {
            broker.abort("infrastructure", error.to_string());
            return;
        }
        let Some(pending) = running.remove(&handle) else {
            continue;
        };
        match joined(pending) {
            Ok(observed) => {
                completed.insert(handle, observed.result);
            }
            Err(failure) => {
                broker.abort("infrastructure", failure.to_string());
                return;
            }
        }
    }
}

/// # Errors
/// Executes a fully authenticated lifecycle with fresh stores and one shared broker and engine.
pub fn execute_prepared(prepared: Prepared) -> Result<Execution> {
    execute_prepared_observed(prepared, None)
}

/// # Errors
/// Refuses observer admission at real store start and joined phase boundaries.
pub fn execute_prepared_observed(
    prepared: Prepared,
    observer: Option<&dyn LifecycleObserver>,
) -> Result<Execution> {
    let cancellation = prepared.cancellation.clone();
    if cancellation.cancelled() {
        return Ok(cancelled_execution());
    }
    #[cfg(not(feature = "determinism-fixture"))]
    let mut conditions = prepared.conditions.enter()?;
    #[cfg(feature = "determinism-fixture")]
    let mut conditions = {
        let (fault, owner) = prepared
            .fixture_fault
            .as_ref()
            .map_or((None, None), |(fault, owner)| (Some(*fault), owner.clone()));
        prepared.conditions.enter_with_fault(fault, owner)?
    };
    let session = conditions.session();
    let engine = Engine::new(&engine::configuration(true))?;
    let network = prepared
        .network
        .as_ref()
        .map(|configuration| {
            Shared::new_with_simulation(configuration, &prepared.ids, prepared.simulation.clone())
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
            observer,
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
        executed_steps: Vec::new(),
        incarnations: Vec::new(),
        world: Vec::new(),
        network: Vec::new(),
        inspection: Vec::new(),
        virtual_elapsed_nanos: 0,
        event_order: None,
    }
}

/// Validates retained broker network evidence before replay admission.
///
/// # Errors
/// Rejects malformed wire, foreign profiles, owner identities and World state.
pub fn validate_network_evidence(
    bytes: &[u8],
    transcript_limit: u64,
    model: &NetworkModel,
) -> Result<(), String> {
    if model.profile != crate::cluster::network::PROFILE || !model.faults.is_empty() {
        return Err("retained broker network profile".into());
    }
    let transitions = model.limits.transitions;
    let requests = transitions
        .checked_mul(2)
        .ok_or("network World request limit overflow")?;
    let config = crate::world::Config {
        seed: 0,
        limits: crate::world::Limits {
            max_requests: requests,
            max_events: requests,
            max_queued_events: (1 << 16).min(requests),
            max_transitions: transitions
                .checked_mul(8)
                .ok_or("network World transition limit overflow")?,
            max_payload_bytes: model
                .limits
                .bytes
                .checked_mul(8)
                .ok_or("network World payload limit overflow")?,
            max_string_bytes: 1024,
        },
    };
    crate::cluster::network::evidence::decode::decode(
        bytes,
        transcript_limit,
        &model.limits,
        &config,
    )
    .map(|_| ())
}

/// Validates retained expanded World bytes against the exact broker configuration.
///
/// # Errors
/// Rejects malformed, noncanonical or foreign model state before guest admission.
pub fn validate_world(bytes: &[u8], broker: &BrokerConfig) -> Result<(), String> {
    crate::world::retained::validate_with_transition_limit(
        bytes,
        &crate::world::Config {
            seed: broker.seed,
            limits: crate::world::Limits {
                max_requests: broker.limits.max_requests,
                max_events: broker.limits.max_events,
                max_queued_events: broker.limits.max_queued_events,
                max_transitions: broker.limits.max_transitions,
                max_payload_bytes: broker.limits.max_payload_bytes,
                max_string_bytes: broker.limits.max_string_bytes,
            },
        },
        broker.transition_bytes,
    )
}

/// # Errors
/// Encodes the existing retained compressed World envelope without guest execution.
pub fn encode_world(bytes: &[u8], encoding: &str) -> Result<Vec<u8>, String> {
    match encoding {
        "" => Ok(bytes.to_vec()),
        crate::world::retained::COMPRESSED => crate::world::retained::compress(bytes),
        _ => Err("unknown retained World encoding".into()),
    }
}

/// # Errors
/// Authenticates the retained envelope and replays its bounded World transitions.
pub fn validate_encoded_world(
    bytes: &[u8],
    encoding: &str,
    broker: &BrokerConfig,
) -> Result<(), String> {
    match encoding {
        "" => validate_world(bytes, broker),
        crate::world::retained::COMPRESSED => {
            validate_world(&crate::world::retained::expand(bytes)?, broker)
        }
        _ => Err("unknown retained World encoding".into()),
    }
}

fn restart_image<'a>(
    images: &'a BTreeMap<String, Vec<u8>>,
    committed: Option<&'a Vec<u8>>,
    original: bool,
    node: &str,
) -> Result<&'a Vec<u8>> {
    if original {
        committed.ok_or_else(|| Error::msg("original committed Temporal image absent"))
    } else {
        images
            .get(node)
            .ok_or_else(|| Error::msg("cluster restart lacks recovery image"))
    }
}

#[cfg(test)]
mod live_controller_tests;

#[cfg(test)]
mod committed_image_tests {
    use super::*;
    #[test]
    fn second_store_writes_do_not_replace_the_primary_reopen_image() {
        let original = vec![1, 2, 3];
        let mut latest = BTreeMap::from([("temporal".to_owned(), original.clone())]);
        assert_eq!(
            restart_image(&latest, Some(&original), true, "temporal").unwrap(),
            &[1, 2, 3]
        );
        latest.insert("temporal".into(), vec![4, 5, 6]);
        assert_eq!(
            restart_image(&latest, Some(&original), true, "temporal").unwrap(),
            &[1, 2, 3]
        );
        assert_eq!(
            restart_image(&latest, Some(&original), false, "temporal").unwrap(),
            &[4, 5, 6]
        );
        assert!(restart_image(&latest, None, true, "temporal").is_err());
    }
}
