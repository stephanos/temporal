//! Multi-store cluster host (fn-178.6): one helper compiles the module once and
//! runs N isolated instance stores from one `InstancePre`, each on its own
//! thread under the cluster baton (fn-178.1 design record). Instances do not
//! interact yet; broker scheduling, World and the clock join in fn-178.7 and
//! the shared network owner in fn-178.12.
pub mod baton;
pub mod transport;

use crate::engine::State;
use crate::network::cluster::{
    Config as NetworkConfig, Endpoint, Failure, ModelV2 as NetworkModel, Shared, validate_model,
};
use crate::protocol::{self, Execute, Outcome};
use baton::{Baton, InOrder};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
use std::io::{BufRead, Cursor, Write};
use std::panic::{AssertUnwindSafe, catch_unwind};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use transport::{Broken, SharedInput, SharedOutput, TaggedInput, TaggedOutput, lock};
use wasmtime::{Engine, Error, InstancePre, Result, Store};

pub const MAX_INSTANCES: usize = 64;
const STORE_THREAD_STACK: usize = 8 << 20;

#[derive(Deserialize)]
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
    /// Go `BrokerConfig`'s network fields, served by one shared Rust owner.
    #[serde(default)]
    pub network: Option<NetworkConfig>,
    #[serde(skip)]
    shared: Option<Arc<Mutex<Shared>>>,
}

fn valid_id(id: &str) -> bool {
    (1..=64).contains(&id.len())
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
            module_path: self.module_path.clone(),
            module_sha256: self.module_sha256.clone(),
            memory_bytes: instance.memory_bytes,
            fuel: instance.fuel,
            output_bytes: instance.output_bytes,
            compile_cache: self.compile_cache.clone(),
            local_model: None,
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
        self.validate_network()
    }

    /// Go `newBrokerNetwork` plus its per-node attachment rules, applied before
    /// anything starts. The broker refuses a node without a stream profile or
    /// with a local fault schedule at its first call; a cluster refuses both
    /// up front, and also a node profile without a cluster network.
    fn validate_network(&mut self) -> Result<()> {
        let refuse = |failure: Failure| Error::msg(failure.message());
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
            if instance.local_model.is_none() {
                return Err(refuse(Failure::Invalid(
                    "broker node requires cooperative runtime",
                )));
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
    termination: &'static str,
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
}

impl ClusterResult {
    fn new(termination: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: "cluster_result",
            termination,
            message: protocol::bounded_message(message),
            instances: Vec::new(),
            metrics: None,
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

/// What a store thread reports back to the coordinator.
struct Report {
    termination: &'static str,
    instantiate_nanos: u64,
    execute_nanos: u64,
}

/// Whole-run failure recorded by the first thread that hits one.
type RunFailure = Arc<Mutex<Option<(&'static str, String)>>>;

fn fail(failure: &RunFailure, termination: &'static str, message: String) {
    let mut failure = lock(failure);
    if failure.is_none() {
        *failure = Some((termination, message));
    }
}

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
    if cfg!(feature = "diagnostic-profile") {
        return send(&ClusterResult::new(
            "unsupported",
            "diagnostic helper builds do not run clusters",
        ));
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
    let broken = Broken::default();
    // Every tape follows authorization in instance order. Each instance's
    // declared frames are read in full first, so a correctly tagged but invalid
    // tape fails only that instance (as its single execution would) and the
    // pipe stays in step for the next one.
    let mut refused: Vec<Option<Outcome>> = Vec::new();
    for instance in &mut request.instances {
        let mut outcome = None;
        if let Some(model) = &mut instance.local_model {
            let mut tagged = TaggedInput::new(input.clone(), &instance.id, broken.clone());
            let mut frames = Vec::new();
            let mut failed = None;
            for _ in 0..model.tape_chunks {
                // Keep reading after a frame-level failure (an oversized chunk
                // is drained by the transport) so every declared chunk leaves
                // the pipe; only an unattributable frame stops the run.
                match protocol::read_frame(&mut tagged) {
                    Ok(frame) => frames.extend(frame),
                    Err(_) if broken.get().is_some() => break,
                    Err(error) => {
                        failed.get_or_insert(error);
                    }
                }
            }
            if let Some(message) = broken.get() {
                return finish(
                    ClusterResult::new("invalid", format!("instance {}: {message}", instance.id)),
                    metrics,
                );
            }
            let read = match failed {
                Some(error) => Err(error),
                None => model.read_tape(&mut Cursor::new(frames)),
            };
            if let Err(error) = read {
                outcome = Some(Outcome::new(
                    protocol::classification(&error),
                    error.to_string(),
                ));
            }
        }
        refused.push(outcome);
    }
    let linker = crate::engine::linker(&engine, &module)?;
    let pre = Arc::new(linker.instantiate_pre(&module)?);
    let baton = Arc::new(Baton::new(request.instances.len(), Box::new(InOrder)));
    let failure: RunFailure = Arc::new(Mutex::new(None));
    let reports: Arc<Mutex<Vec<(String, Report)>>> = Arc::new(Mutex::new(Vec::new()));
    let mut threads = Vec::new();
    let shared = request.shared.take();
    for (index, (instance, refused)) in request.instances.into_iter().zip(refused).enumerate() {
        let network = shared.clone();
        let (engine, pre, own_baton, own_failure, reports, broken) = (
            engine.clone(),
            pre.clone(),
            baton.clone(),
            failure.clone(),
            reports.clone(),
            broken.clone(),
        );
        let input = TaggedInput::new(input.clone(), &instance.id, broken.clone());
        let output = TaggedOutput::new(output.clone(), &instance.id);
        let thread = std::thread::Builder::new()
            .name(format!("instance-{index}"))
            .stack_size(STORE_THREAD_STACK)
            .spawn(move || {
                let (baton, failure) = (own_baton, own_failure);
                if !baton.acquire(index) {
                    return;
                }
                let id = instance.id.clone();
                let ran = catch_unwind(AssertUnwindSafe(|| match refused {
                    Some(outcome) => refuse_instance(outcome, output),
                    None => run_instance(
                        &engine, &pre, instance, index, network, input, output, &broken,
                    ),
                }));
                match ran {
                    Ok(Ok(report)) => {
                        lock(&reports).push((id, report));
                        baton.finish(index);
                    }
                    Ok(Err(stop)) => {
                        fail(&failure, stop.0, format!("instance {id}: {}", stop.1));
                        baton.abort();
                    }
                    Err(_) => {
                        fail(
                            &failure,
                            "infrastructure",
                            format!("instance {id}: helper panic"),
                        );
                        baton.abort();
                    }
                }
            });
        match thread {
            Ok(thread) => threads.push(thread),
            Err(error) => {
                fail(&failure, "infrastructure", format!("store thread: {error}"));
                baton.abort();
                break;
            }
        }
    }
    baton.start();
    let completed = baton.wait();
    for thread in threads {
        if thread.join().is_err() {
            fail(&failure, "infrastructure", "store thread panicked".into());
        }
    }
    let reports = std::mem::take(&mut *lock(&reports));
    for (_, report) in &reports {
        metrics.instantiate_nanos += report.instantiate_nanos;
        metrics.execute_nanos += report.execute_nanos;
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
            termination: report.termination,
        })
        .collect();
    finish(result, metrics)
}

/// Runs one instance's store to its terminal and streams its tagged result.
/// `Err` is a whole-run outcome: the instance's result is withheld.
#[allow(clippy::too_many_arguments)]
fn run_instance(
    engine: &Engine,
    pre: &InstancePre<State>,
    instance: InstanceConfig,
    index: usize,
    network: Option<Arc<Mutex<Shared>>>,
    input: TaggedInput,
    output: TaggedOutput,
    broken: &Broken,
) -> std::result::Result<Report, (&'static str, String)> {
    let endpoint = match (&network, &instance.network_v2, &instance.local_model) {
        (Some(shared), Some(profile), Some(model)) => Some(
            Endpoint::new(
                shared.clone(),
                &instance.id,
                1,
                index as u64 + 1,
                profile,
                model.epoch_nanos,
                model.limits.transcript_bytes,
            )
            .map_err(|failure| ("infrastructure", failure.message()))?,
        ),
        _ => None,
    };
    let mut store = Store::new(
        engine,
        crate::engine::new_state(
            Box::new(input),
            Box::new(output),
            instance.memory_bytes,
            instance.output_bytes,
            instance.local_model.map(crate::model::Model::new),
        ),
    );
    store.data_mut().network = endpoint;
    store.limiter(|state| &mut state.limits);
    let whole = |error: Error| ("infrastructure", format!("{error:#}"));
    store.set_fuel(instance.fuel).map_err(whole)?;
    let phase = Instant::now();
    let instantiated = pre.instantiate(&mut store);
    let instantiate_nanos = crate::metrics::elapsed(phase);
    let mut execute_nanos = 0;
    let executed = instantiated.and_then(|instance| {
        let start = instance.get_typed_func::<(), ()>(&mut store, "_start")?;
        let phase = Instant::now();
        let execution = start.call(&mut store, ());
        execute_nanos = crate::metrics::elapsed(phase);
        execution
    });
    let mut outcome =
        crate::engine::terminal_outcome(&mut store, executed, instance.fuel).map_err(whole)?;
    if let Some(message) = broken.get() {
        return Err(("invalid", message));
    }
    if outcome.termination == "infrastructure" {
        return Err(("infrastructure", outcome.message));
    }
    let state = store.data_mut();
    if let Err(error) = crate::engine::finish_model(state, &mut outcome) {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    if let Some(endpoint) = &state.network {
        network_evidence(endpoint, &mut state.output, &mut outcome)?;
        // Go's `BrokerSession.Finished` revokes a finished incarnation's
        // domains gracefully after its evidence was taken.
        if let Some(shared) = &network {
            lock(shared)
                .revoke(&instance.id, 1, true)
                .map_err(|error| ("infrastructure", error))?;
        }
    }
    let termination = outcome.termination;
    if let Err(error) = protocol::send_frame(&mut state.output, &outcome) {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    Ok(Report {
        termination,
        instantiate_nanos,
        execute_nanos,
    })
}

/// An instance refused before its store existed (an invalid tape): its result
/// is streamed at its turn, like any terminal.
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
