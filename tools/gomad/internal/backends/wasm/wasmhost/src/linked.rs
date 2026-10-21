use crate::{engine, metrics::Metrics, model, namespace, protocol};
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeMap,
    io::{self, Cursor, Write},
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
        mpsc,
    },
    thread,
    time::Duration,
};
use wasmtime::{Engine, Error, Store};

pub use crate::linked_cancellation::Cancellation;
pub use crate::linked_cluster::NetworkModel;
pub use crate::linked_conditions::Conditions;
pub use crate::linked_storage::Input as StorageInput;
pub use crate::model::replay::Request as ReplayRequest;
pub use crate::model::replay_plan::{
    ExecutionIdentity as ChoiceIdentity, PrefixControl, RankOverride, ReplayPlan as ChoicePlan,
    TraceDecision, TraceInspection, TraceSite, coverage_features as choice_features,
    inspect as inspect_choices, project as project_choices,
};
pub use crate::model::{Config as ModelConfig, Decision, Divergence, ModelLimits};
pub use crate::namespace::{
    Image as NamespaceImage, ImageEntry as NamespaceEntry, Limits as NamespaceLimits,
};
pub use crate::protocol::Outcome;

pub const IMPLEMENTATION_SHA256: &str = env!("GOMAD_LINKED_IMPLEMENTATION_SHA256");
pub const CONFIGURATION: &str = "cranelift-fuel-nan-canonical-nounwind-epoch-v1";

pub struct Limits {
    pub memory_bytes: u64,
    pub fuel: u64,
    pub output_bytes: u64,
    pub evidence_bytes: u64,
    pub wall_timeout: Duration,
}

pub struct ModelInput {
    pub configuration: ModelConfig,
    pub arguments: Vec<String>,
    pub environment: Vec<String>,
    pub replay_trace: Option<Vec<u8>>,
}

pub struct Request {
    pub module: Vec<u8>,
    pub module_sha256: String,
    pub limits: Limits,
    pub model: Option<ModelInput>,
    pub namespace: Option<NamespaceImage>,
    pub storage: Option<StorageInput>,
}

pub struct Execution {
    pub outcome: Outcome,
    pub evidence: BTreeMap<String, Vec<u8>>,
    pub event_order: Option<Vec<u8>>,
}

pub struct Prepared {
    pub(crate) request: Request,
    pub(crate) execute: protocol::Execute,
    pub(crate) model: Option<model::Model>,
    pub(crate) namespace: Option<namespace::Namespace>,
    pub(crate) conditions: Conditions,
    cancellation: Cancellation,
    network: Option<crate::cluster::network::Endpoint>,
}

impl Prepared {
    /// # Errors
    /// Admits a fresh standalone network owner with the retained local fault schedule.
    pub fn network(&mut self, configuration: NetworkModel) -> wasmtime::Result<()> {
        let model = self
            .model
            .as_ref()
            .ok_or_else(|| Error::msg("linked network requires helper-owned model"))?;
        if self.network.is_some() {
            return Err(Error::msg("linked network already configured"));
        }
        self.network = Some(crate::linked_network::prepare(configuration, model)?);
        Ok(())
    }
    pub fn cancellation(&mut self, cancellation: Cancellation) {
        self.cancellation = cancellation;
    }

    pub fn conditions(&mut self, conditions: Conditions) {
        self.conditions = conditions;
    }
    /// # Errors
    /// Refuses an exhausted caller budget; a remaining budget may only shorten the original bound.
    pub fn bound_wall_timeout(&mut self, remaining: Duration) -> wasmtime::Result<()> {
        if remaining.is_zero() {
            return Err(Error::msg("execution wall budget exhausted"));
        }
        self.request.limits.wall_timeout = self.request.limits.wall_timeout.min(remaining);
        Ok(())
    }
}

#[derive(Clone)]
pub(crate) struct Sink {
    streams: Arc<Mutex<BTreeMap<String, Vec<u8>>>>,
    maximum: u64,
}

impl Sink {
    pub(crate) fn new(maximum: u64) -> Self {
        Self {
            streams: Arc::new(Mutex::new(BTreeMap::new())),
            maximum,
        }
    }

    pub(crate) fn into_streams(self) -> wasmtime::Result<BTreeMap<String, Vec<u8>>> {
        Arc::try_unwrap(self.streams)
            .map_err(|_| Error::msg("linked evidence still borrowed"))?
            .into_inner()
            .map_err(|_| Error::msg("linked evidence lock poisoned"))
    }
    pub(crate) fn append(&self, stream: &str, bytes: &[u8]) -> wasmtime::Result<()> {
        let mut streams = self
            .streams
            .lock()
            .map_err(|_| Error::msg("linked evidence lock poisoned"))?;
        let retained = streams
            .values()
            .try_fold(bytes.len() as u64, |total, value| {
                total.checked_add(value.len() as u64)
            })
            .ok_or_else(|| Error::msg("linked evidence capacity overflow"))?;
        if retained > self.maximum {
            return Err(Error::new(model::RemoteCapacity(
                "linked evidence capacity",
            )));
        }
        streams
            .entry(stream.into())
            .or_default()
            .extend_from_slice(bytes);
        Ok(())
    }
}

struct NoRemote;
impl Write for NoRemote {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            "linked execution requires helper-owned imports",
        ))
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

fn admitted(
    request: &mut Request,
) -> wasmtime::Result<(
    protocol::Execute,
    Option<model::Model>,
    Option<namespace::Namespace>,
)> {
    let limits = &request.limits;
    if request.module.len() > 512 * 1024 * 1024
        || limits.evidence_bytes == 0
        || limits.evidence_bytes > 1 << 30
        || limits.wall_timeout.is_zero()
    {
        return Err(Error::msg("invalid linked execution bounds"));
    }
    if format!("{:x}", Sha256::digest(&request.module)) != request.module_sha256 {
        return Err(Error::msg("module SHA-256 mismatch"));
    }
    let execute = protocol::Execute {
        schema: protocol::SCHEMA.into(),
        kind: "execute".into(),
        module_path: "/linked/module.wasm".into(),
        module_sha256: request.module_sha256.clone(),
        memory_bytes: limits.memory_bytes,
        fuel: limits.fuel,
        output_bytes: limits.output_bytes,
        compile_cache: None,
        local_model: None,
        namespace: None,
        storage: None,
        #[cfg(feature = "guest-profile")]
        guest_profile: None,
    };
    execute.validate()?;
    let mut model = None;
    let transcript_limit = request
        .model
        .as_ref()
        .map(|input| input.configuration.limits.transcript_bytes);
    if let Some(mut input) = request.model.take() {
        if input.configuration.split || input.configuration.tape.is_some() {
            return Err(Error::msg(
                "linked model forbids external owners or caller-projected tapes",
            ));
        }
        input
            .configuration
            .set_inputs(&input.arguments, &input.environment)?;
        match (&input.configuration.replay, input.replay_trace.as_deref()) {
            (Some(replay), Some(trace)) => {
                let plan = model::replay::preflight(
                    trace,
                    replay,
                    input.configuration.mode,
                    input.configuration.limit,
                )?;
                input.configuration.tape = Some(plan.decisions);
            }
            (None, None) => {}
            _ => return Err(Error::msg("linked replay trace/declaration mismatch")),
        }
        let image = request
            .namespace
            .as_ref()
            .ok_or_else(|| Error::msg("linked model requires a sealed namespace"))?;
        if !(1..=1 << 20).contains(&image.limits.files)
            || !(5..=1 << 20).contains(&image.limits.descriptors)
            || !(1..=1 << 30).contains(&image.limits.filesystem_bytes)
            || !(1..=1 << 30).contains(&image.limits.output_bytes)
            || image.epoch_nanos > i64::MAX as u64
        {
            return Err(Error::msg("invalid linked namespace image limits"));
        }
        if image.epoch_nanos != input.configuration.epoch_nanos
            || image.limits.output_bytes != limits.output_bytes
            || format!("{:x}", Sha256::digest(&image.working_directory))
                != input.configuration.working_directory_sha256
        {
            return Err(Error::msg("linked namespace/model identity mismatch"));
        }
        model = Some(model::Model::new_linked(input.configuration));
    } else if request.namespace.is_some() {
        return Err(Error::msg("linked namespace requires model ownership"));
    }
    let mut namespace = request
        .namespace
        .take()
        .map(namespace::Namespace::new)
        .transpose()
        .map_err(Error::new)?;
    if let Some(storage) = &mut request.storage {
        let namespace = namespace
            .as_mut()
            .ok_or_else(|| Error::msg("linked storage requires a sealed namespace"))?;
        let transcript_limit = transcript_limit
            .ok_or_else(|| Error::msg("linked storage requires helper-owned model"))?;
        storage.attach(namespace, transcript_limit)?;
    }
    Ok((execute, model, namespace))
}

/// # Errors
/// Validates retained inputs and helper-owned replay tapes without creating an engine.
pub fn validate(request: Request) -> wasmtime::Result<()> {
    prepare(request).map(|_| ())
}

/// # Errors
/// Authenticates helper-owned inputs and replay projection without creating an engine.
pub fn prepare(mut request: Request) -> wasmtime::Result<Prepared> {
    let (execute, model, namespace) = admitted(&mut request)?;
    Ok(Prepared {
        request,
        execute,
        model,
        namespace,
        conditions: Conditions::default(),
        cancellation: Cancellation::default(),
        network: None,
    })
}

/// # Errors
/// Refuses invalid bounded inputs before creating an engine or instantiating the guest.
pub fn execute(request: Request) -> wasmtime::Result<Execution> {
    execute_prepared(prepare(request)?)
}

/// # Errors
/// Executes previously authenticated inputs with a fresh engine and store.
pub fn execute_prepared(prepared: Prepared) -> wasmtime::Result<Execution> {
    let Prepared {
        request,
        execute,
        mut model,
        namespace,
        conditions,
        cancellation,
        network,
    } = prepared;
    if cancellation.cancelled() {
        return Ok(Cancellation::execution());
    }
    let mut conditions = conditions.enter()?;
    let sink = Sink::new(request.limits.evidence_bytes);
    if let Some(model) = &mut model {
        model.linked_evidence = Some(sink.clone());
    }
    let engine = Engine::new(&engine::configuration(true))?;
    let _registration = cancellation.attach(&engine, None)?;
    let timed_out = AtomicBool::new(false);
    let mut outcome = thread::scope(|scope| -> wasmtime::Result<Outcome> {
        let (done, completed) = mpsc::sync_channel::<()>(0);
        let watcher =
            scope.spawn(|| watch(&engine, &timed_out, request.limits.wall_timeout, completed));
        let result = (|| -> wasmtime::Result<Outcome> {
            let mut metrics = Metrics::default();
            let module = match engine::compile(&engine, &request.module, &mut metrics)
                .and_then(|module| engine::describe_module(&execute, module))
            {
                Ok((module, _)) => module,
                Err(outcome) => return Ok(outcome),
            };
            let mut store = Store::new(
                &engine,
                engine::new_state(
                    Box::new(Cursor::new(Vec::<u8>::new())),
                    Box::new(NoRemote),
                    request.limits.memory_bytes,
                    request.limits.output_bytes,
                    model,
                    namespace,
                ),
            );
            store.limiter(|state| &mut state.limits);
            store.data_mut().network = network;
            store.set_fuel(request.limits.fuel)?;
            store.set_epoch_deadline(1);
            let linker = engine::linker(&engine, &module)?;
            let executed = if cancellation.cancelled() {
                Err(Error::msg("seed execution was cancelled"))
            } else if timed_out.load(Ordering::Acquire) {
                Err(Error::msg("execution watchdog timeout"))
            } else {
                (|| {
                    let instance = linker.instantiate(&mut store, &module)?;
                    instance
                        .get_typed_func::<(), ()>(&mut store, "_start")?
                        .call(&mut store, ())
                })()
            };
            let mut outcome = engine::terminal_outcome(&mut store, executed, request.limits.fuel)?;
            if let Some(storage) = &request.storage {
                storage.finish(store.data_mut(), &mut outcome);
            }
            if outcome.termination == "storage_crash"
                && let Some(model) = &mut store.data_mut().model
                && let Err(error) = model.interrupted()
            {
                outcome.termination = if model.divergence.is_some() {
                    "divergence"
                } else {
                    "invalid"
                };
                outcome.message = error.to_string();
                outcome.exit_code = None;
            }
            apply_watchdog(&timed_out, &mut outcome);
            if let Err(error) = engine::finish_model(store.data_mut(), &mut outcome) {
                outcome.termination = protocol::classification(&error);
                outcome.exit_code = None;
                outcome.message = error.to_string();
            }
            Ok(outcome)
        })();
        drop(done);
        watcher
            .join()
            .map_err(|_| Error::msg("host panic in execution watchdog"))?;
        result
    })?;
    apply_watchdog(&timed_out, &mut outcome);
    cancellation.apply(&mut outcome);
    let evidence = sink.into_streams()?;
    let event_order = conditions.finish();
    Ok(Execution {
        outcome,
        evidence,
        event_order,
    })
}

fn watch(
    engine: &Engine,
    timed_out: &AtomicBool,
    timeout: Duration,
    completed: mpsc::Receiver<()>,
) {
    if completed.recv_timeout(timeout) == Err(mpsc::RecvTimeoutError::Timeout) {
        timed_out.store(true, Ordering::Release);
        engine.increment_epoch();
    }
    drop(completed);
}

fn apply_watchdog(timed_out: &AtomicBool, outcome: &mut Outcome) {
    if timed_out.load(Ordering::Acquire) {
        outcome.termination = "watchdog_timeout";
        outcome.exit_code = None;
        outcome.message = "execution watchdog timeout".into();
    }
}
