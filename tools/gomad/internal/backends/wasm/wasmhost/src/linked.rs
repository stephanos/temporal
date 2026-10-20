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

pub use crate::model::replay::Request as ReplayRequest;
pub use crate::model::replay_plan::{
    ExecutionIdentity as ChoiceIdentity, TraceInspection, inspect as inspect_choices,
};
pub use crate::model::{Config as ModelConfig, ModelLimits};
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
}

pub struct Execution {
    pub outcome: Outcome,
    pub evidence: BTreeMap<String, Vec<u8>>,
}

#[derive(Clone)]
pub(crate) struct Sink {
    streams: Arc<Mutex<BTreeMap<String, Vec<u8>>>>,
    maximum: u64,
}

impl Sink {
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
        if image.epoch_nanos != input.configuration.epoch_nanos
            || image.limits.output_bytes != limits.output_bytes
            || format!("{:x}", Sha256::digest(&image.working_directory))
                != input.configuration.working_directory_sha256
        {
            return Err(Error::msg("linked namespace/model identity mismatch"));
        }
        model = Some(model::Model::new(input.configuration));
    } else if request.namespace.is_some() {
        return Err(Error::msg("linked namespace requires model ownership"));
    }
    let namespace = request
        .namespace
        .take()
        .map(namespace::Namespace::new)
        .transpose()
        .map_err(Error::new)?;
    Ok((execute, model, namespace))
}

/// # Errors
/// Refuses invalid bounded inputs before creating an engine or instantiating the guest.
pub fn execute(mut request: Request) -> wasmtime::Result<Execution> {
    let (execute, mut model, namespace) = admitted(&mut request)?;
    let sink = Sink {
        streams: Arc::new(Mutex::new(BTreeMap::new())),
        maximum: request.limits.evidence_bytes,
    };
    if let Some(model) = &mut model {
        model.linked_evidence = Some(sink.clone());
    }
    let engine = Engine::new(&engine::configuration(true))?;
    let timed_out = AtomicBool::new(false);
    let mut outcome = thread::scope(|scope| -> wasmtime::Result<Outcome> {
        let (done, completed) = mpsc::sync_channel::<()>(0);
        let (watch_engine, watch_timeout, wall_timeout) =
            (&engine, &timed_out, request.limits.wall_timeout);
        let watcher = scope.spawn(move || {
            if completed.recv_timeout(wall_timeout) == Err(mpsc::RecvTimeoutError::Timeout) {
                watch_timeout.store(true, Ordering::Release);
                watch_engine.increment_epoch();
            }
        });
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
            store.set_fuel(request.limits.fuel)?;
            store.set_epoch_deadline(1);
            let linker = engine::linker(&engine, &module)?;
            let executed = if timed_out.load(Ordering::Acquire) {
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
            if timed_out.load(Ordering::Acquire) {
                outcome.termination = "watchdog_timeout";
                outcome.exit_code = None;
                outcome.message = "execution watchdog timeout".into();
            }
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
    if timed_out.load(Ordering::Acquire) {
        outcome.termination = "watchdog_timeout";
        outcome.exit_code = None;
        outcome.message = "execution watchdog timeout".into();
    }
    let evidence = Arc::try_unwrap(sink.streams)
        .map_err(|_| Error::msg("linked evidence still borrowed"))?
        .into_inner()
        .map_err(|_| Error::msg("linked evidence lock poisoned"))?;
    Ok(Execution { outcome, evidence })
}
