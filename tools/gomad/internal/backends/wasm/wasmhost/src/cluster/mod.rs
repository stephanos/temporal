//! Multi-store cluster host (fn-178.6): one helper compiles the module once and
//! runs N isolated instance stores from one `InstancePre`, each on its own
//! thread under the cluster baton (fn-178.1 design record). Instances do not
//! interact yet; broker scheduling, World and the clock join in fn-178.7 and
//! the shared network owner in fn-178.12.
pub mod baton;
pub mod transport;

use crate::engine::{Limits, State};
use crate::protocol::{self, Authorization, Execute, Outcome};
use baton::{Baton, InOrder};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
use std::io::{BufRead, Write};
use std::panic::{AssertUnwindSafe, catch_unwind};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use transport::{Broken, SharedInput, SharedOutput, TaggedInput, TaggedOutput, lock};
use wasmtime::{Engine, Error, ExternType, InstancePre, Linker, Result, Store};

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
        let mut message = message.into();
        if message.len() > 65536 {
            let mut limit = 65536 - " (message truncated)".len();
            while !message.is_char_boundary(limit) {
                limit -= 1;
            }
            message.truncate(limit);
            message.push_str(" (message truncated)");
        }
        Self {
            kind: "cluster_result",
            termination,
            message,
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
type Failure = Arc<Mutex<Option<(&'static str, String)>>>;

fn fail(failure: &Failure, termination: &'static str, message: String) {
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
    let authorized = (|| -> Result<()> {
        let authorization: Authorization =
            serde_json::from_slice(&protocol::read_frame(&mut **lock(&input))?)
                .map_err(|e| Error::msg(e.to_string()))?;
        if authorization.kind != "authorize"
            || authorization.module_sha256 != request.module_sha256
            || authorization.configuration != crate::engine::CONFIGURATION_REVISION
        {
            return Err(Error::msg("execution authorization identity mismatch"));
        }
        Ok(())
    })();
    if let Err(error) = authorized {
        return finish(
            ClusterResult::new(protocol::classification(&error), error.to_string()),
            metrics,
        );
    }
    let broken = Broken::default();
    for instance in &mut request.instances {
        if let Some(model) = &mut instance.local_model {
            let mut tagged = TaggedInput::new(input.clone(), &instance.id, broken.clone());
            if let Err(error) = model.read_tape(&mut tagged) {
                let termination = if broken.get().is_some() {
                    "invalid"
                } else {
                    protocol::classification(&error)
                };
                return finish(
                    ClusterResult::new(termination, format!("instance {}: {error}", instance.id)),
                    metrics,
                );
            }
        }
    }
    let mut linker = Linker::new(&engine);
    let mut defined = BTreeSet::new();
    for import in module.imports() {
        if !defined.insert((import.module().to_owned(), import.name().to_owned())) {
            continue;
        }
        let name = import.name().to_owned();
        let import_module = import.module().to_owned();
        let ExternType::Func(ty) = import.ty() else {
            unreachable!()
        };
        linker.func_new(
            import.module(),
            import.name(),
            ty,
            move |caller, params, results| {
                crate::abi::invoke(caller, &import_module, &name, params, results)
            },
        )?;
    }
    let pre = Arc::new(linker.instantiate_pre(&module)?);
    let baton = Arc::new(Baton::new(request.instances.len(), Box::new(InOrder)));
    let failure: Failure = Arc::new(Mutex::new(None));
    let reports: Arc<Mutex<Vec<(String, Report)>>> = Arc::new(Mutex::new(Vec::new()));
    let mut threads = Vec::new();
    for (index, instance) in request.instances.into_iter().enumerate() {
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
                let ran = catch_unwind(AssertUnwindSafe(|| {
                    run_instance(&engine, &pre, instance, input, output, &broken)
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
fn run_instance(
    engine: &Engine,
    pre: &InstancePre<State>,
    instance: InstanceConfig,
    input: TaggedInput,
    output: TaggedOutput,
    broken: &Broken,
) -> std::result::Result<Report, (&'static str, String)> {
    let mut store = Store::new(
        engine,
        State {
            input: Box::new(input),
            output: Box::new(output),
            limits: Limits {
                max: instance.memory_bytes as usize,
                peak: 0,
                failure: None,
            },
            calls: 0,
            completed_calls: 0,
            output_limit: instance.output_bytes,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: instance.local_model.map(crate::model::Model::new),
            request_scratch: Vec::new(),
            write_scratch: protocol::WriteBuffers::default(),
        },
    );
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
    let mut outcome = store
        .data_mut()
        .terminal
        .take()
        .unwrap_or_else(|| match executed {
            Ok(()) => {
                let mut result = Outcome::new("exit", "");
                result.exit_code = Some(0);
                result
            }
            Err(error) => crate::engine::execution_failure(error, store.data().limits.failure),
        });
    if let Some(message) = broken.get() {
        return Err(("invalid", message));
    }
    if outcome.termination == "infrastructure" {
        return Err(("infrastructure", outcome.message));
    }
    outcome.fuel_remaining = store.get_fuel().map_err(whole)?;
    outcome.peak_memory_bytes = store.data().limits.peak;
    outcome.output_bytes = store.data().output_bytes;
    let state = store.data_mut();
    let termination = outcome.termination;
    let sent = (|| {
        if let Some(model) = &mut state.model {
            model.flush(&mut state.output, true)?;
            outcome.local_model = Some(Box::new(model.metadata(state.completed_calls)));
            outcome.divergence = model.divergence.take().map(Box::new);
        }
        protocol::send_frame(&mut state.output, &outcome)
    })();
    if let Err(error) = sent {
        return Err(("infrastructure", format!("result transport: {error:#}")));
    }
    Ok(Report {
        termination,
        instantiate_nanos,
        execute_nanos,
    })
}
