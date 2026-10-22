//! One helper owns fresh incarnation stores, shared prepared modules, broker
//! scheduling, World, clock and network under the cluster baton.
mod manual;
pub mod transport;

use super::handoff::Coordinator;
use crate::cluster::network::{
    Config as NetworkConfig, Endpoint, Failure, ModelV2 as NetworkModel, Shared, validate_model,
};
use crate::engine::State;
use crate::protocol::{self, Execute, Outcome};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
use std::io::{BufRead, Write};
use std::panic::{AssertUnwindSafe, catch_unwind};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use transport::{Broken, SharedInput, SharedOutput, TaggedInput, TaggedOutput, lock};
use wasmtime::{Engine, Error, InstancePre, Result, Store};

pub const MAX_INSTANCES: usize = 64;
const STORE_THREAD_STACK: usize = 8 << 20;

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct InstanceConfig {
    pub id: String,
    pub memory_bytes: u64,
    pub fuel: u64,
    pub output_bytes: u64,
    #[serde(default)]
    pub local_model: Option<crate::model::Config>,
    /// The instance's own network profile (fn-178.12); required exactly when
    /// the cluster has a network.
    #[serde(default)]
    pub network_v2: Option<NetworkModel>,
    #[serde(default)]
    pub module_path: Option<String>,
    #[serde(default)]
    pub module_sha256: Option<String>,
    #[serde(default)]
    pub namespace: Option<crate::namespace::transport::NamespaceDescriptor>,
    #[serde(default)]
    pub storage: Option<crate::storage_transport::StorageDescriptor>,
    #[serde(default)]
    pub stdout_progress: bool,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ClusterExecute {
    pub schema: String,
    #[serde(rename = "type")]
    pub kind: String,
    pub module_path: String,
    pub module_sha256: String,
    #[serde(default)]
    pub compile_cache: Option<String>,
    pub instances: Vec<InstanceConfig>,
    /// Go `BrokerConfig`'s World fields: the in-process broker (fn-178.7).
    #[serde(default)]
    pub broker: Option<crate::cluster::handoff::Config>,
    /// Go `BrokerConfig`'s network fields, served by one shared Rust owner.
    #[serde(default)]
    pub network: Option<NetworkConfig>,
    #[serde(default)]
    pub manual_start: bool,
    #[serde(default)]
    pub boots: Vec<InstanceConfig>,
    #[serde(default)]
    pub output_crashes: Vec<manual::Boundary>,
    #[serde(default)]
    pub output_stops: Vec<manual::Boundary>,
    #[serde(default)]
    pub output_crash_group: Option<manual::CrashGroup>,
    #[serde(skip)]
    shared: Option<Arc<Mutex<Shared>>>,
}

fn valid_id(id: &str) -> bool {
    (1..=128).contains(&id.len())
        && id
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'.' | b'_' | b'-'))
}

impl ClusterExecute {
    /// The single-execution request whose rules every instance must satisfy.
    fn single(&self, instance: &InstanceConfig) -> Execute {
        Execute {
            schema: self.schema.clone(),
            kind: "execute".into(),
            module_path: instance
                .module_path
                .clone()
                .unwrap_or_else(|| self.module_path.clone()),
            module_sha256: instance
                .module_sha256
                .clone()
                .unwrap_or_else(|| self.module_sha256.clone()),
            memory_bytes: instance.memory_bytes,
            fuel: instance.fuel,
            output_bytes: instance.output_bytes,
            compile_cache: self.compile_cache.clone(),
            local_model: None,
            namespace: None,
            storage: None,
            #[cfg(feature = "guest-profile")]
            guest_profile: None,
        }
    }

    pub fn validate(&mut self) -> Result<()> {
        if self.schema != protocol::SCHEMA || self.kind != "cluster_execute" {
            return Err(Error::msg("unknown cluster execute schema or type"));
        }
        if self.instances.is_empty() || self.instances.len() > MAX_INSTANCES {
            return Err(Error::msg("invalid cluster instance count"));
        }
        let mut ids = BTreeSet::new();
        for instance in &self.instances {
            if !valid_id(&instance.id) || !ids.insert(instance.id.clone()) {
                return Err(Error::msg("invalid or duplicate cluster instance id"));
            }
            self.single(instance).validate()?;
            if let Some(model) = &instance.local_model {
                model.validate()?;
            }
        }
        self.instances.sort_by(|left, right| left.id.cmp(&right.id));
        for boot in &self.boots {
            if !valid_id(&boot.id) || !ids.insert(boot.id.clone()) {
                return Err(Error::msg("invalid or duplicate cluster boot id"));
            }
            self.single(boot).validate()?;
            if let Some(model) = &boot.local_model {
                model.validate()?;
            }
        }
        self.boots.sort_by(|left, right| left.id.cmp(&right.id));
        manual::validate(self)?;
        self.validate_network()
    }

    /// Go `newBrokerNetwork` plus its per-node attachment rules, applied before
    /// anything starts. The broker refuses a node without a stream profile or
    /// with a local fault schedule at its first call; a cluster refuses both
    /// up front, and also a node profile without a cluster network.
    fn validate_network(&mut self) -> Result<()> {
        let refuse = |failure: Failure| Error::msg(failure.message());
        if let Some(broker) = &self.broker {
            broker.validate().map_err(Error::msg)?;
            // The broker serves runtime.idle on the cooperative runtime.
            if self
                .instances
                .iter()
                .any(|instance| instance.local_model.is_none())
            {
                return Err(refuse(Failure::Invalid(
                    "broker node requires cooperative runtime",
                )));
            }
        } else if self.network.is_some() {
            // As in Go, a cluster network belongs to a broker.
            return Err(refuse(Failure::Invalid(
                "cluster network requires a broker",
            )));
        }
        let Some(network) = &self.network else {
            if self
                .instances
                .iter()
                .any(|instance| instance.network_v2.is_some())
            {
                return Err(refuse(Failure::Invalid(
                    "broker node network without cluster network",
                )));
            }
            return Ok(());
        };
        for instance in &self.instances {
            let Some(model) = &instance.network_v2 else {
                return Err(refuse(Failure::Invalid(
                    "broker node requires stream network profile",
                )));
            };
            validate_model(model).map_err(refuse)?;
            if !model.faults.is_empty() {
                return Err(refuse(Failure::Invalid("broker node local fault schedule")));
            }
        }
        let ids: Vec<String> = self
            .instances
            .iter()
            .map(|instance| instance.id.clone())
            .collect();
        let shared = Shared::new(network, &ids).map_err(refuse)?;
        self.shared = Some(Arc::new(Mutex::new(shared)));
        Ok(())
    }
}

#[derive(Serialize)]
struct InstanceSummary {
    id: String,
    #[serde(skip_serializing_if = "first_incarnation")]
    incarnation: u64,
    termination: &'static str,
}

#[expect(
    clippy::trivially_copy_pass_by_ref,
    reason = "Serde skip_serializing_if passes fields by reference"
)]
fn first_incarnation(incarnation: &u64) -> bool {
    *incarnation == 1
}

#[derive(Serialize)]
struct ClusterResult {
    #[serde(rename = "type")]
    kind: &'static str,
    termination: &'static str,
    message: String,
    instances: Vec<InstanceSummary>,
    #[serde(skip_serializing_if = "Option::is_none")]
    metrics: Option<crate::metrics::Metrics>,
    virtual_elapsed_nanos: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    scheduler: Option<crate::cluster::handoff::Metrics>,
}

impl ClusterResult {
    fn new(termination: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: "cluster_result",
            termination,
            message: protocol::bounded_message(message),
            instances: Vec::new(),
            metrics: None,
            virtual_elapsed_nanos: 0,
            scheduler: None,
        }
    }
}

/// True when the first frame asks for a cluster execution.
pub fn requested(frame: &[u8]) -> bool {
    #[derive(Deserialize)]
    struct Kind<'a> {
        #[serde(rename = "type", borrow)]
        kind: Option<&'a str>,
    }
    serde_json::from_slice::<Kind<'_>>(frame)
        .is_ok_and(|frame| frame.kind == Some("cluster_execute"))
}

fn read_instance_tape(
    model: &mut crate::model::Config,
    tagged: &mut TaggedInput,
    _transport_failure: &Broken,
) -> Result<()> {
    if let Some(request) = &model.replay {
        model.tape = Some(tagged.replay(request, model.mode, model.limit)?.decisions);
    }
    Ok(())
}

/// What a store thread reports back to the coordinator.
struct Report {
    termination: &'static str,
    instantiate_nanos: u64,
    execute_nanos: u64,
    image: Option<Vec<u8>>,
    host_nanos: u64,
    guest_nanos: u64,
}

/// Whole-run failure recorded by the first thread that hits one.
type RunFailure = Arc<Mutex<Option<(&'static str, String)>>>;

fn fail(failure: &RunFailure, termination: &'static str, message: String) {
    let mut failure = lock(failure);
    if failure.is_none() {
        *failure = Some((termination, message));
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "The ordered model transition preserves Go admission, mutation and refusal order"
)]
pub fn execute(
    frame: &[u8],
    input: Box<dyn BufRead + Send>,
    output: Box<dyn Write + Send>,
) -> Result<()> {
    let output: SharedOutput = Arc::new(Mutex::new(output));
    let input: SharedInput = Arc::new(Mutex::new(input));
    let send = |result: &ClusterResult| protocol::send_frame(&mut **lock(&output), result);
    let mut request: ClusterExecute = match serde_json::from_slice(frame)
        .map_err(|error| Error::msg(error.to_string()))
        .and_then(|mut request: ClusterExecute| request.validate().map(|()| request))
    {
        Ok(request) => request,
        Err(error) => {
            return send(&ClusterResult::new(
                protocol::classification(&error),
                error.to_string(),
            ));
        }
    };
    // Fuel-off engines cannot meter store threads; only the timing
    // diagnostic build profiles clusters (fn-185).
    if cfg!(feature = "diagnostic-fuel-off") {
        return send(&ClusterResult::new(
            "unsupported",
            "diagnostic fuel-off helper builds do not run clusters",
        ));
    }
    if cfg!(feature = "guest-profile") {
        return send(&ClusterResult::new(
            "unsupported",
            "diagnostic helper builds do not run clusters",
        ));
    }
    if request.manual_start {
        return manual::execute(request, input, output);
    }
    // Each cooperative instance's model inputs (arguments, environment) follow
    // the execute frame as tagged frames in instance order, before compilation,
    // as in a single execution; any failure is a pre-start refusal.
    let transport_failure = Broken::default();
    for instance in &mut request.instances {
        if let Some(model) = &mut instance.local_model {
            let mut tagged =
                TaggedInput::new(input.clone(), &instance.id, transport_failure.clone());
            if let Err(error) = model.read_inputs(&mut tagged) {
                let termination = if transport_failure.get().is_some() {
                    "invalid"
                } else {
                    protocol::classification(&error)
                };
                return send(&ClusterResult::new(
                    termination,
                    format!("instance {}: {error}", instance.id),
                ));
            }
        }
    }
    let mut metrics = crate::metrics::Metrics::default();
    let phase = Instant::now();
    // Admission against the largest instance; each store's limiter judges its
    // own initial memory at instantiation.
    let largest = request
        .instances
        .iter()
        .max_by_key(|instance| instance.memory_bytes)
        .unwrap();
    let prepared = crate::engine::prepare(&request.single(largest), &mut metrics);
    metrics.prepare_nanos = crate::metrics::elapsed(phase);
    let finish = |mut result: ClusterResult, mut metrics: crate::metrics::Metrics| {
        metrics.sample_process();
        result.metrics = Some(metrics);
        protocol::send_frame(&mut **lock(&output), &result)
    };
    let (engine, module, started) = match prepared {
        Ok(value) => value,
        Err(outcome) => {
            return finish(
                ClusterResult::new(outcome.termination, outcome.message),
                metrics,
            );
        }
    };
    if let Err(error) = protocol::send_frame(&mut **lock(&output), &started) {
        if protocol::classification(&error) == "capacity" {
            return finish(ClusterResult::new("capacity", error.to_string()), metrics);
        }
        return Err(error);
    }
    let authorized = crate::engine::authorize(&mut **lock(&input), &request.module_sha256);
    if let Err(error) = authorized {
        return finish(
            ClusterResult::new(protocol::classification(&error), error.to_string()),
            metrics,
        );
    }
    // Every tape follows authorization in instance order. Each instance's
    // declared frames are read in full first, so a correctly tagged but invalid
    // tape fails only that instance (as its single execution would) and the
    // pipe stays in step for the next one.
    let mut refused: Vec<Option<Outcome>> = Vec::new();
    for instance in &mut request.instances {
        let mut outcome = None;
        if let Some(model) = &mut instance.local_model {
            let mut tagged =
                TaggedInput::new(input.clone(), &instance.id, transport_failure.clone());
            let read = read_instance_tape(model, &mut tagged, &transport_failure);
            if let Some(message) = transport_failure.get() {
                return finish(
                    ClusterResult::new("invalid", format!("instance {}: {message}", instance.id)),
                    metrics,
                );
            }
            if let Err(error) = read {
                outcome = Some(Outcome::refusal(&error));
            }
        }
        refused.push(outcome);
    }
    let linker = crate::engine::linker(&engine, &module)?;
    let pre = Arc::new(linker.instantiate_pre(&module)?);
    let shared = request.shared.take();
    let coordinator = match &request.broker {
        Some(config) => {
            let ids: Vec<String> = request
                .instances
                .iter()
                .map(|instance| instance.id.clone())
                .collect();
            let broker = crate::cluster::handoff::Broker::new(
                config,
                &ids,
                shared.clone(),
                Some((input.clone(), output.clone())),
            );
            match broker {
                Ok(broker) => Coordinator::Broker(Arc::new(broker)),
                Err(error) => return finish(ClusterResult::new("invalid", error), metrics),
            }
        }
        None => Coordinator::in_order(request.instances.len()),
    };
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    coordinator.set_event_owners(
        request
            .instances
            .iter()
            .map(|instance| crate::event_order::Context {
                instance: instance.id.clone(),
                incarnation: 1,
                trace: 0,
                decision: 0,
                time_nanos: instance
                    .local_model
                    .as_ref()
                    .map_or(0, |model| model.now.cast_signed()),
            })
            .collect(),
    );
    let failure: RunFailure = Arc::new(Mutex::new(None));
    let reports: Arc<Mutex<Vec<(String, Report)>>> = Arc::new(Mutex::new(Vec::new()));
    let mut threads = Vec::new();
    for (index, (instance, refused)) in request.instances.into_iter().zip(refused).enumerate() {
        let network = shared.clone();
        let (engine, pre, own_coordinator, own_failure, reports, transport_failure) = (
            engine.clone(),
            pre.clone(),
            coordinator.clone(),
            failure.clone(),
            reports.clone(),
            transport_failure.clone(),
        );
        let input = TaggedInput::new(input.clone(), &instance.id, transport_failure.clone());
        let output = TaggedOutput::new(output.clone(), &instance.id);
        let thread = std::thread::Builder::new()
            .name(format!("instance-{index}"))
            .stack_size(STORE_THREAD_STACK)
            .spawn(move || {
                let (coordinator, failure) = (own_coordinator, own_failure);
                if !coordinator.acquire(index) {
                    if coordinator.inspection_stopped() {
                        let id = instance.id.clone();
                        if let Ok(report) = refuse_instance(
                            Outcome::new("inspection_aborted", "cluster inspection stopped"),
                            output,
                        ) {
                            lock(&reports).push((id, report));
                        }
                    }
                    return;
                }
                let id = instance.id.clone();
                let ran = catch_unwind(AssertUnwindSafe(|| match refused {
                    Some(outcome) => refuse_instance(outcome, output),
                    None => run_instance(
                        &engine,
                        &pre,
                        instance,
                        index,
                        network,
                        &coordinator,
                        input,
                        output,
                        &transport_failure,
                        1,
                        index as u64 + 1,
                        None,
                        None,
                        false,
                        None,
                    ),
                }));
                match ran {
                    Ok(Ok(report)) => {
                        lock(&reports).push((id, report));
                        coordinator.finish(index);
                    }
                    Ok(Err(stop)) => {
                        fail(&failure, stop.0, format!("instance {id}: {}", stop.1));
                        coordinator.abort(stop.0, stop.1);
                    }
                    Err(_) => {
                        fail(
                            &failure,
                            "infrastructure",
                            format!("instance {id}: helper panic"),
                        );
                        coordinator.abort("infrastructure", "helper panic".into());
                    }
                }
            });
        match thread {
            Ok(thread) => threads.push(thread),
            Err(error) => {
                fail(&failure, "infrastructure", format!("store thread: {error}"));
                coordinator.abort("infrastructure", "store thread".into());
                break;
            }
        }
    }
    coordinator.start();
    let completed = coordinator.wait();
    for thread in threads {
        if thread.join().is_err() {
            fail(&failure, "infrastructure", "store thread panicked".into());
        }
    }
    let mut reports = std::mem::take(&mut *lock(&reports));
    reports.sort_by(|(left, _), (right, _)| left.cmp(right));
    for (_, report) in &reports {
        metrics.instantiate_nanos += report.instantiate_nanos;
        metrics.execute_nanos += report.execute_nanos;
    }
    // A failure the broker raised itself (a stall, a refused control frame)
    // is the run's; the instances it aborted only report the abort.
    if let Some(raised) = coordinator.failure() {
        *lock(&failure) = Some(raised);
    }
    // World evidence follows the last instance (fn-178.1 draft): the broker
    // World's expanded wire in 64 KiB chunks; Go wraps and validates it.
    if completed
        && lock(&failure).is_none()
        && let Coordinator::Broker(broker) = &coordinator
    {
        match broker.world_wire() {
            Ok(wire) => {
                for chunk in wire.chunks(64 * 1024) {
                    use base64::Engine as _;
                    let frame = serde_json::json!({"type":"evidence","stream":"world",
                        "data_base64":base64::engine::general_purpose::STANDARD.encode(chunk)});
                    protocol::send_frame(&mut **lock(&output), &frame)?;
                }
            }
            Err(refused) => fail(&failure, refused.termination(), refused.message()),
        }
    }
    let mut result = match lock(&failure).take() {
        Some((termination, message)) => ClusterResult::new(termination, message),
        None if completed => ClusterResult::new("exit", ""),
        None => ClusterResult::new("infrastructure", "cluster ended before every instance"),
    };
    result.instances = reports
        .into_iter()
        .map(|(id, report)| InstanceSummary {
            id,
            incarnation: 1,
            termination: report.termination,
        })
        .collect();
    if let Coordinator::Broker(broker) = &coordinator {
        inspection_evidence(broker, &output)?;
    }
    finish(result, metrics)
}

fn inspection_evidence(
    broker: &crate::cluster::handoff::Broker,
    output: &SharedOutput,
) -> Result<()> {
    let mut recorder = lock(&broker.inspection);
    if !recorder.observed || recorder.evidence.incarnations.is_empty() {
        return Ok(());
    }
    let complete = !recorder.sealed;
    let bytes = recorder.bytes(complete)?;
    for chunk in bytes.chunks(64 * 1024) {
        use base64::Engine as _;
        protocol::send_frame(
            &mut **lock(output),
            &serde_json::json!({"type":"evidence","stream":"inspection",
            "data_base64":base64::engine::general_purpose::STANDARD.encode(chunk)}),
        )?;
    }
    Ok(())
}

/// Runs one instance's store to its terminal and streams its tagged result.
/// `Err` is a whole-run outcome: the instance's result is withheld.
#[allow(clippy::too_many_arguments)]
#[expect(
    clippy::needless_pass_by_value,
    reason = "The worker owns its network and storage descriptors for the full store lifetime"
)]
#[expect(
    clippy::too_many_lines,
    reason = "The ordered model transition preserves Go admission, mutation and refusal order"
)]
fn run_instance(
    engine: &Engine,
    pre: &InstancePre<State>,
    instance: InstanceConfig,
    index: usize,
    network: Option<Arc<Mutex<Shared>>>,
    coordinator: &Coordinator,
    input: TaggedInput,
    output: TaggedOutput,
    transport_failure: &Broken,
    incarnation: u64,
    slot: u64,
    namespace: Option<crate::namespace::Namespace>,
    storage: Option<crate::storage_transport::StorageDescriptor>,
    measured: bool,
    on_terminal: Option<&dyn Fn(&Outcome)>,
) -> std::result::Result<Report, (&'static str, String)> {
    if let Coordinator::Broker(broker) = coordinator {
        broker.wait_terminal(index);
    }
    // Each store thread profiles into its own thread profile, merged into
    // the process report when the session closes (fn-185).
    #[cfg(feature = "diagnostic-profile")]
    let _profile = crate::diagnostic::ThreadSession::start(slot, incarnation);
    let endpoint = match (&network, &instance.network_v2, &instance.local_model) {
        (Some(shared), Some(profile), Some(model)) => Some(
            Endpoint::new(
                shared.clone(),
                &instance.id,
                incarnation,
                slot,
                profile,
                model.epoch_nanos,
                model.limits.transcript_bytes,
            )
            .map_err(|failure| ("infrastructure", failure.message()))?,
        ),
        _ => None,
    };
    let mut store = {
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(crate::event_order::owner(
            &instance.id,
            incarnation,
            instance
                .local_model
                .as_ref()
                .map_or(0, |model| model.now.cast_signed()),
        ));
        Store::new(
            engine,
            crate::engine::new_state(
                Box::new(input),
                Box::new(output),
                instance.memory_bytes,
                instance.output_bytes,
                instance.local_model.map(crate::model::Model::new),
                namespace,
            ),
        )
    };
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    if let Some(model) = &mut store.data_mut().model {
        model.event_owner = Some(crate::model::inspection::Handle {
            node: instance.id.clone(),
            incarnation,
        });
    }
    if let Coordinator::Broker(broker) = coordinator {
        store.data_mut().broker = Some(crate::cluster::handoff::Member {
            broker: broker.clone(),
            index,
            incarnation,
            network: endpoint,
        });
        if let Some(model) = &mut store.data_mut().model {
            model.inspection.handle = crate::model::inspection::Handle {
                node: instance.id.clone(),
                incarnation,
            };
            model.inspection.recorder = broker.inspection.clone();
        }
    }
    store.limiter(|state| &mut state.limits);
    store.data_mut().stdout_progress = instance.stdout_progress;
    let whole = |error: Error| ("infrastructure", format!("{error:#}"));
    store.set_fuel(instance.fuel).map_err(whole)?;
    let phase = Instant::now();
    let checked = if let Coordinator::Broker(broker) = coordinator {
        broker.check(index, incarnation).map_err(|failure| {
            store
                .data_mut()
                .fail(failure.termination(), failure.message())
        })
    } else {
        Ok(())
    };
    #[cfg(feature = "diagnostic-profile")]
    let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Instantiate);
    let instantiated = checked.and_then(|()| pre.instantiate(&mut store));
    #[cfg(feature = "diagnostic-profile")]
    drop(diagnostic_phase);
    let instantiate_nanos = crate::metrics::elapsed(phase);
    let mut execute_nanos = 0;
    let executed = instantiated.and_then(|instance| {
        let start = instance.get_typed_func::<(), ()>(&mut store, "_start")?;
        if measured {
            store.data_mut().host_nanos = Some(0);
        }
        let phase = Instant::now();
        #[cfg(feature = "diagnostic-profile")]
        let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Execute);
        let execution = start.call(&mut store, ());
        #[cfg(feature = "diagnostic-profile")]
        drop(diagnostic_phase);
        execute_nanos = crate::metrics::elapsed(phase);
        execution
    });
    let mut outcome =
        crate::engine::terminal_outcome(&mut store, executed, instance.fuel).map_err(whole)?;
    if let Some(message) = transport_failure.get() {
        return Err(("invalid", message));
    }
    if matches!(
        outcome.termination,
        "inspected_prefix" | "instrumentation_divergence" | "debug_error" | "probe_fuel_limit"
    ) {
        coordinator.abort(outcome.termination, outcome.message.clone());
    }
    if outcome.termination == "infrastructure" && coordinator.inspection_stopped() {
        outcome.termination = "inspection_aborted";
        outcome.message = "cluster inspection stopped".into();
        if let Some(model) = &mut store.data_mut().model {
            model.inspection.sealed = true;
        }
    }
    if outcome.termination == "infrastructure" {
        return Err(("infrastructure", outcome.message));
    }
    let state = store.data_mut();
    if let Coordinator::Broker(broker) = coordinator {
        broker.wait_terminal(index);
    }
    let image = record_terminal(state, storage.as_ref(), &mut outcome)?;
    let termination = outcome.termination;
    let host_nanos = state.host_nanos.unwrap_or(0);
    let guest_nanos = execute_nanos.saturating_sub(host_nanos);
    if measured {
        outcome.metrics = Some(Box::new(crate::metrics::Metrics {
            instantiate_nanos,
            execute_nanos,
            ..Default::default()
        }));
    }
    // Automatic handoff must follow the result so the next instance cannot
    // overtake it. Manual control must observe a finished incarnation when
    // Go answers the result with its next command.
    if coordinator.driven()
        && let Some(on_terminal) = on_terminal
    {
        on_terminal(&outcome);
    }
    let draining = matches!(coordinator, Coordinator::Broker(broker) if broker.terminal_failed());
    if coordinator.driven() && !draining {
        coordinator.finish(index);
    }
    if let Err(error) = protocol::send_frame(&mut state.output, &outcome) {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    if coordinator.driven() && draining {
        coordinator.finish(index);
    }
    Ok(Report {
        termination,
        instantiate_nanos,
        execute_nanos,
        image,
        host_nanos,
        guest_nanos,
    })
}

pub(crate) fn terminal_failure(outcome: &Outcome) -> Option<(&'static str, String)> {
    match outcome.termination {
        "exit" if outcome.exit_code == Some(0) => None,
        "exit" => Some((
            "trap",
            format!(
                "WASM guest exited with code {}",
                outcome.exit_code.unwrap_or(1)
            ),
        )),
        "crash"
        | "stop"
        | "storage_crash"
        | "inspection_aborted"
        | "inspected_prefix"
        | "instrumentation_divergence"
        | "debug_error"
        | "probe_fuel_limit" => None,
        termination => Some((termination, outcome.message.clone())),
    }
}

/// An instance's terminal evidence after its store stopped: recovery
/// storage, terminal network and the local model's result, then the check
/// that no deferred network commitment survives into the instance's finish.
/// Returns the crash image for a later incarnation.
pub(crate) fn record_terminal(
    state: &mut State,
    storage: Option<&crate::storage_transport::StorageDescriptor>,
    outcome: &mut Outcome,
) -> std::result::Result<Option<Vec<u8>>, (&'static str, String)> {
    let mut image = None;
    if outcome.termination != "inspection_aborted"
        && outcome.termination != "inspected_prefix"
        && let (Some(descriptor), Some(namespace), Some(model)) =
            (storage, &mut state.namespace, &mut state.model)
    {
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(model.event_context());
        let mut evidence = crate::storage_transport::recovery_evidence(namespace, descriptor);
        image = evidence.image.take();
        if descriptor.return_image {
            evidence.image.clone_from(&image);
        }
        if let Some((kind, message)) = evidence.failure.take() {
            outcome.termination = kind;
            outcome.message = message;
            outcome.exit_code = None;
        }
        model.storage_evidence(evidence);
    }
    if outcome.termination != "inspection_aborted"
        && outcome.termination != "inspected_prefix"
        && let Some(endpoint) = state
            .broker
            .as_ref()
            .and_then(|member| member.network.as_ref())
    {
        if let Some(model) = &state.model {
            let mut recorder = model.inspection.lock();
            match endpoint.evidence() {
                Ok(bytes) => recorder.terminal_network(&model.inspection.handle, bytes),
                Err(_) => recorder.exhausted = true,
            }
        }
        if let Some(sink) = state
            .model
            .as_ref()
            .and_then(|model| model.linked_evidence.as_ref())
        {
            let bytes = endpoint
                .evidence()
                .map_err(|failure| (failure.termination(), failure.message()))?;
            if bytes.len() as u64 > endpoint.transcript_limit() {
                let capacity = Failure::Capacity("network-v2-evidence-bytes");
                outcome.termination = capacity.termination();
                outcome.message = capacity.message();
                outcome.exit_code = None;
            } else {
                sink.append("network", &bytes)
                    .map_err(|error| (protocol::classification(&error), error.to_string()))?;
            }
        } else {
            network_evidence(endpoint, &mut state.output, outcome)?;
        }
    }
    if let Err(error) = crate::engine::finish_model(state, outcome) {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    // The terminal network evidence above superseded any deferred
    // commitment; an aborted or prefix inspection records none.
    if outcome.termination != "inspection_aborted"
        && outcome.termination != "inspected_prefix"
        && let Some(member) = &state.broker
        && let Err(message) = member.check_network_commitment("instance finish")
    {
        return Err(("infrastructure", message));
    }
    Ok(image)
}

/// An instance refused before its store existed (an invalid tape): its result
/// is streamed at its turn, like any terminal.
#[expect(
    clippy::needless_pass_by_value,
    reason = "Refusal consumes the worker output handle after sending its terminal result"
)]
fn refuse_instance(
    outcome: Outcome,
    mut output: TaggedOutput,
) -> std::result::Result<Report, (&'static str, String)> {
    if let Err(error) = protocol::send_frame(&mut output, &outcome) {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    Ok(Report {
        termination: outcome.termination,
        instantiate_nanos: 0,
        execute_nanos: 0,
        image: None,
        host_nanos: 0,
        guest_nanos: 0,
    })
}

/// Streams an instance's binary network evidence in 64 KiB chunks, or turns
/// its outcome into `network-v2-evidence-bytes` capacity when the evidence
/// exceeds its transcript bytes (as Go's `NetworkV2Evidence` does).
fn network_evidence(
    endpoint: &Endpoint,
    output: &mut Box<dyn Write + Send>,
    outcome: &mut Outcome,
) -> std::result::Result<(), (&'static str, String)> {
    let document = endpoint
        .evidence()
        .map_err(|failure| (failure.termination(), failure.message()))?;
    if document.len() as u64 > endpoint.transcript_limit() {
        let capacity = Failure::Capacity("network-v2-evidence-bytes");
        outcome.termination = capacity.termination();
        outcome.message = capacity.message();
        outcome.exit_code = None;
        return Ok(());
    }
    for chunk in document.chunks(64 * 1024) {
        use base64::Engine as _;
        let frame = serde_json::json!({"type":"evidence","stream":"network",
            "data_base64":base64::engine::general_purpose::STANDARD.encode(chunk)});
        protocol::send_frame(&mut **output, &frame)
            .map_err(|error| ("infrastructure", format!("result transport: {error:#}")))?;
    }
    Ok(())
}
