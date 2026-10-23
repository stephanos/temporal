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

#[cfg(feature = "guest-profile")]
pub use crate::guest_profile::Config as GuestProfileConfig;
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
pub use crate::model::{Config as ModelConfig, CreationSite, Decision, Divergence, ModelLimits};
pub use crate::module_metadata::{ModuleImport, ModuleMetadata, module_metadata};
pub use crate::namespace::{
    Image as NamespaceImage, ImageEntry as NamespaceEntry, Limits as NamespaceLimits,
};
pub use crate::protocol::Outcome;

#[cfg(feature = "stress-jitter")]
pub use crate::stress_jitter::statistics as stress_jitter_statistics;

pub const SOURCE_INPUTS: &str =
    include_str!(concat!(env!("OUT_DIR"), "/measurement-source-inputs.tsv"));

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

pub struct ExecutionTiming {
    pub authorization_started: std::time::Instant,
    pub terminal_classified: std::time::Instant,
    pub returned: std::time::Instant,
}
impl ExecutionTiming {
    #[must_use]
    pub fn authorization_to_terminal_nanos(&self) -> u64 {
        u64::try_from(
            self.terminal_classified
                .duration_since(self.authorization_started)
                .as_nanos(),
        )
        .unwrap_or(u64::MAX)
    }
    #[must_use]
    pub fn terminal_to_return_nanos(&self) -> u64 {
        u64::try_from(
            self.returned
                .duration_since(self.terminal_classified)
                .as_nanos(),
        )
        .unwrap_or(u64::MAX)
    }
}

pub struct Execution {
    pub timing: Option<ExecutionTiming>,
    #[cfg(feature = "guest-profile")]
    pub guest_profile_error: Option<String>,
    #[cfg(feature = "diagnostic-profile")]
    pub external_owner_attempts: u64,
    pub authorization_to_terminal_nanos: Option<u64>,
    pub outcome: Outcome,
    pub evidence: BTreeMap<String, Vec<u8>>,
    pub event_order: Option<Vec<u8>>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum OutputStream {
    Stdout,
    Stderr,
}

/// Consumes accepted output synchronously. Calls are serialized per request.
/// Implementations must bound their I/O and must not reenter the request's sink.
pub trait OutputConsumer: Send + Sync {
    /// # Errors
    /// A write failure stops execution and retains the accepted evidence prefix.
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()>;
}

pub struct Prepared {
    pub(crate) request: Request,
    pub(crate) execute: protocol::Execute,
    pub(crate) model: Option<model::Model>,
    pub(crate) namespace: Option<namespace::Namespace>,
    pub(crate) conditions: Conditions,
    cancellation: Cancellation,
    #[cfg(feature = "determinism-fixture")]
    fixture_timing_delays: [Duration; 2],
    output: Option<Arc<dyn OutputConsumer>>,
    network: Option<crate::cluster::network::Endpoint>,
    #[cfg(feature = "guest-profile")]
    guest_profile: Option<GuestProfileConfig>,
}

impl Prepared {
    #[cfg(feature = "determinism-fixture")]
    /// Admits bounded request-local delays for native terminal lifecycle controls.
    ///
    /// # Errors
    /// Rejects delays longer than thirty seconds.
    pub fn fixture_timing_delays(
        &mut self,
        preparation: Duration,
        cleanup: Duration,
    ) -> wasmtime::Result<()> {
        if [preparation, cleanup]
            .into_iter()
            .any(|delay| delay > Duration::from_secs(30))
        {
            return Err(Error::msg(
                "fixture timing delay exceeds original control bound",
            ));
        }
        self.fixture_timing_delays = [preparation, cleanup];
        Ok(())
    }

    /// # Errors
    /// Admits request-local bounded guest sampling in a guest-profile role only.
    #[cfg(feature = "guest-profile")]
    pub fn guest_profile(&mut self, config: GuestProfileConfig) -> wasmtime::Result<()> {
        config.validate()?;
        if self.guest_profile.is_some() {
            return Err(Error::msg("linked guest profile already configured"));
        }
        self.guest_profile = Some(config);
        Ok(())
    }

    #[must_use]
    pub fn compile_cache_directory(&self) -> Option<&str> {
        self.execute.compile_cache.as_deref()
    }

    /// # Errors
    /// Refuses an invalid host compile-cache selection.
    pub fn compile_cache(&mut self, directory: Option<&str>) -> wasmtime::Result<()> {
        if directory.is_some_and(|directory| !std::path::Path::new(directory).is_absolute()) {
            return Err(Error::msg("invalid compile cache directory"));
        }
        self.execute.compile_cache = directory.map(str::to_owned);
        Ok(())
    }

    /// The execution wall bound the host's process-wide deadline backstop arms.
    #[must_use]
    pub fn wall_timeout(&self) -> Duration {
        self.request.limits.wall_timeout
    }

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

    pub fn output(&mut self, output: Arc<dyn OutputConsumer>) {
        self.output = Some(output);
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
    output: Option<Arc<Output>>,
}

struct Output {
    consumer: Arc<dyn OutputConsumer>,
    failure: Mutex<Option<String>>,
    cancellation: Cancellation,
}

impl Sink {
    pub(crate) fn new(maximum: u64) -> Self {
        Self {
            streams: Arc::new(Mutex::new(BTreeMap::new())),
            maximum,
            output: None,
        }
    }

    fn with_output(
        mut self,
        consumer: Option<Arc<dyn OutputConsumer>>,
        cancellation: Cancellation,
    ) -> Self {
        self.output = consumer.map(|consumer| {
            Arc::new(Output {
                consumer,
                failure: Mutex::new(None),
                cancellation,
            })
        });
        self
    }

    pub(crate) fn streaming(&self) -> bool {
        self.output.is_some()
    }

    fn output_failure(&self) -> Option<String> {
        self.output.as_ref().and_then(|output| {
            output
                .failure
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner)
                .clone()
        })
    }

    pub(crate) fn into_streams(self) -> wasmtime::Result<BTreeMap<String, Vec<u8>>> {
        Arc::try_unwrap(self.streams)
            .map_err(|_| Error::msg("linked evidence still borrowed"))?
            .into_inner()
            .map_err(|_| Error::msg("linked evidence lock poisoned"))
    }
    fn retained_bytes(streams: &BTreeMap<String, Vec<u8>>, offered: u64) -> wasmtime::Result<u64> {
        streams
            .values()
            .try_fold(offered, |total, value| {
                total.checked_add(value.len() as u64)
            })
            .ok_or_else(|| Error::msg("linked evidence capacity overflow"))
    }
    pub(crate) fn append(&self, stream: &str, bytes: &[u8]) -> wasmtime::Result<()> {
        let output = self.output.as_ref().and_then(|output| {
            let stream = match stream {
                "stdout" => OutputStream::Stdout,
                "stderr" => OutputStream::Stderr,
                _ => return None,
            };
            Some((output, stream))
        });
        // Acquire before evidence admission to preserve each stream's order.
        let mut forwarding = output.map(|(output, _)| {
            output
                .failure
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner)
        });
        let mut streams = self
            .streams
            .lock()
            .map_err(|_| Error::msg("linked evidence lock poisoned"))?;
        let retained = Self::retained_bytes(&streams, bytes.len() as u64)?;
        if retained > self.maximum {
            return Err(Error::new(model::RemoteCapacity(
                "linked evidence capacity",
            )));
        }
        streams
            .entry(stream.into())
            .or_default()
            .extend_from_slice(bytes);
        drop(streams);
        if let (Some((output, stream)), Some(failure)) = (output, forwarding.as_mut())
            && failure.is_none()
            && !bytes.is_empty()
            && let Err(error) = output.consumer.write(stream, bytes)
        {
            let message = format!("linked output consumer: {error}");
            **failure = Some(message.clone());
            output.cancellation.cancel();
            return Err(Error::new(io::Error::other(message)));
        }
        Ok(())
    }
}

struct NoRemote {
    #[cfg(feature = "diagnostic-profile")]
    attempts: Arc<std::sync::atomic::AtomicU64>,
}
impl Write for NoRemote {
    fn write(&mut self, _: &[u8]) -> io::Result<usize> {
        #[cfg(feature = "diagnostic-profile")]
        self.attempts
            .try_update(Ordering::Relaxed, Ordering::Relaxed, |value| {
                value.checked_add(1)
            })
            .map_err(|_| io::Error::other("native external-owner counter overflow"))?;
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
        #[cfg(feature = "determinism-fixture")]
        fixture_timing_delays: [Duration::ZERO; 2],
        output: None,
        network: None,
        #[cfg(feature = "guest-profile")]
        guest_profile: None,
    })
}

/// # Errors
/// Refuses invalid bounded inputs before creating an engine or instantiating the guest.
pub fn execute(request: Request) -> wasmtime::Result<Execution> {
    execute_prepared(prepare(request)?)
}

/// # Errors
/// Executes previously authenticated inputs with a fresh engine and store.
#[expect(
    clippy::too_many_lines,
    reason = "Authorization, terminal classification and cleanup share the native timing scope"
)]
pub fn execute_prepared(prepared: Prepared) -> wasmtime::Result<Execution> {
    let Prepared {
        request,
        execute,
        mut model,
        namespace,
        conditions,
        cancellation,
        #[cfg(feature = "determinism-fixture")]
        fixture_timing_delays,
        output,
        network,
        #[cfg(feature = "guest-profile")]
        guest_profile,
    } = prepared;
    if cancellation.cancelled() {
        return Ok(Cancellation::execution());
    }
    let mut conditions = conditions.enter()?;
    let sink = Sink::new(request.limits.evidence_bytes).with_output(output, cancellation.clone());
    if let Some(model) = &mut model {
        model.linked_evidence = Some(sink.clone());
    }
    let preparation = std::time::Instant::now();
    let engine = Engine::new(&engine::configuration(true))?;
    let registration = cancellation.attach(&engine, None)?;
    let timed_out = Arc::new(AtomicBool::new(false));
    #[cfg(feature = "diagnostic-profile")]
    let external_owner_attempts = Arc::new(std::sync::atomic::AtomicU64::new(0));
    let mut authorization_started = None;
    let mut terminal_classified = None;
    #[cfg(feature = "guest-profile")]
    let mut guest_profile_error = None;
    let mut outcome = thread::scope(|scope| -> wasmtime::Result<Outcome> {
        let (done, completed) = mpsc::sync_channel::<()>(0);
        let watcher =
            scope.spawn(|| watch(&engine, &timed_out, request.limits.wall_timeout, completed));
        let result = (|| -> wasmtime::Result<Outcome> {
            let mut metrics = Metrics::default();
            let module = match compile_module(&engine, &execute, &request.module, &mut metrics) {
                Ok((module, _)) => module,
                Err(mut outcome) => {
                    metrics.prepare_nanos = crate::metrics::elapsed(preparation);
                    metrics.sample_process();
                    outcome.metrics = Some(Box::new(metrics));
                    return Ok(outcome);
                }
            };
            #[cfg(feature = "determinism-fixture")]
            fixture_timing_delay(fixture_timing_delays[0]);
            metrics.prepare_nanos = crate::metrics::elapsed(preparation);
            authorization_started = Some(std::time::Instant::now());
            let mut store = Store::new(
                &engine,
                engine::new_state(
                    Box::new(Cursor::new(Vec::<u8>::new())),
                    Box::new(NoRemote {
                        #[cfg(feature = "diagnostic-profile")]
                        attempts: Arc::clone(&external_owner_attempts),
                    }),
                    request.limits.memory_bytes,
                    request.limits.output_bytes,
                    model,
                    namespace,
                ),
            );
            store.limiter(|state| &mut state.limits);
            store.data_mut().network = network;
            #[cfg(not(feature = "diagnostic-fuel-off"))]
            store.set_fuel(request.limits.fuel)?;
            store.set_epoch_deadline(1);
            #[cfg(feature = "guest-profile")]
            let guest_profile = guest_profile
                .as_ref()
                .map(|config| {
                    let cancellation = cancellation.clone();
                    let timed_out = Arc::clone(&timed_out);
                    crate::guest_profile::Session::start_interruptible(
                        &mut store,
                        &module,
                        config,
                        move || {
                            if cancellation.cancelled() {
                                Err(Error::msg("seed execution was cancelled"))
                            } else if timed_out.load(Ordering::Acquire) {
                                Err(Error::msg("execution watchdog timeout"))
                            } else {
                                Ok(())
                            }
                        },
                    )
                })
                .transpose()?;
            let linker = engine::linker(&engine, &module)?;
            let executed = if cancellation.cancelled() {
                Err(Error::msg("seed execution was cancelled"))
            } else if timed_out.load(Ordering::Acquire) {
                Err(Error::msg("execution watchdog timeout"))
            } else {
                start_instance(&linker, &mut store, &module, &mut metrics)
            };
            let mut outcome = finish_outcome(&mut store, executed, &request, &timed_out)?;
            terminal_classified = Some(std::time::Instant::now());
            #[cfg(feature = "determinism-fixture")]
            fixture_timing_delay(fixture_timing_delays[1]);
            #[cfg(test)]
            timing_tests::after_terminal();
            #[cfg(feature = "guest-profile")]
            if let Some(profile) = guest_profile {
                guest_profile_error = profile.finish().err().map(|error| error.to_string());
            }
            metrics.sample_process();
            outcome.metrics = Some(Box::new(metrics));
            Ok(outcome)
        })();
        drop(done);
        watcher
            .join()
            .map_err(|_| Error::msg("host panic in execution watchdog"))?;
        result
    })?;
    apply_watchdog(&timed_out, &mut outcome);
    if let Some(message) = sink.output_failure() {
        outcome.termination = "infrastructure";
        outcome.exit_code = None;
        outcome.message = message;
    } else {
        cancellation.apply(&mut outcome);
    }
    let evidence = sink.into_streams()?;
    let event_order = conditions.finish();
    drop(registration);
    drop(engine);
    let returned = std::time::Instant::now();
    let timing = authorization_started.zip(terminal_classified).map(
        |(authorization_started, terminal_classified)| ExecutionTiming {
            authorization_started,
            terminal_classified,
            returned,
        },
    );
    let authorization_to_terminal_nanos = timing
        .as_ref()
        .map(ExecutionTiming::authorization_to_terminal_nanos);
    Ok(Execution {
        timing,
        #[cfg(feature = "guest-profile")]
        guest_profile_error,
        #[cfg(feature = "diagnostic-profile")]
        external_owner_attempts: external_owner_attempts.load(Ordering::Relaxed),
        authorization_to_terminal_nanos,
        outcome,
        evidence,
        event_order,
    })
}

pub(crate) fn compile_module(
    engine: &Engine,
    execute: &protocol::Execute,
    bytes: &[u8],
    metrics: &mut Metrics,
) -> Result<(wasmtime::Module, serde_json::Value), Outcome> {
    let module = match &execute.compile_cache {
        Some(directory) => crate::cache::module(
            engine,
            || Ok(bytes),
            directory,
            &execute.module_sha256,
            metrics,
        )?,
        None => engine::compile(engine, bytes, metrics)?,
    };
    engine::describe_module(execute, module)
}

fn start_instance(
    linker: &wasmtime::Linker<engine::State>,
    store: &mut Store<engine::State>,
    module: &wasmtime::Module,
    metrics: &mut Metrics,
) -> wasmtime::Result<()> {
    let phase = std::time::Instant::now();
    let instance = {
        #[cfg(feature = "diagnostic-profile")]
        let _span = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Instantiate);
        linker.instantiate(&mut *store, module)
    };
    metrics.instantiate_nanos = crate::metrics::elapsed(phase);
    let instance = instance?;
    let start = instance.get_typed_func::<(), ()>(&mut *store, "_start")?;
    let phase = std::time::Instant::now();
    let executed = {
        #[cfg(feature = "diagnostic-profile")]
        let _span = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Execute);
        start.call(store, ())
    };
    metrics.execute_nanos = crate::metrics::elapsed(phase);
    executed
}

fn finish_outcome(
    store: &mut Store<engine::State>,
    executed: wasmtime::Result<()>,
    request: &Request,
    timed_out: &AtomicBool,
) -> wasmtime::Result<Outcome> {
    let mut outcome = engine::terminal_outcome(store, executed, request.limits.fuel)?;
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
    apply_watchdog(timed_out, &mut outcome);
    if let Err(error) = engine::finish_model(store.data_mut(), &mut outcome) {
        outcome.termination = protocol::classification(&error);
        outcome.exit_code = None;
        outcome.message = error.to_string();
    }
    Ok(outcome)
}

#[cfg(feature = "determinism-fixture")]
fn fixture_timing_delay(delay: Duration) {
    if !delay.is_zero() {
        let (_sender, receiver) = mpsc::channel::<()>();
        let _ = receiver.recv_timeout(delay);
    }
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

#[cfg(test)]
mod live_metrics_tests {
    use super::*;

    #[test]
    fn linked_terminal_retains_actual_phase_and_process_metrics() {
        let module = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))".to_vec();
        let mut prepared = prepare(Request {
            module_sha256: format!("{:x}", Sha256::digest(&module)),
            module,
            limits: Limits {
                memory_bytes: 65536,
                fuel: 1000,
                output_bytes: 1024,
                evidence_bytes: 4096,
                wall_timeout: Duration::from_secs(5),
            },
            model: None,
            namespace: None,
            storage: None,
        })
        .unwrap();
        prepared.conditions(Conditions::compiled());
        let result = execute_prepared(prepared).unwrap();
        assert_eq!(result.outcome.termination, "exit");
        assert!(
            result
                .authorization_to_terminal_nanos
                .expect("linked authorization span missing")
                > 0
        );
        let metrics = result
            .outcome
            .metrics
            .expect("linked phase metrics missing");
        assert!(metrics.prepare_nanos > 0);
        assert!(metrics.compile_nanos > 0);
        assert!(metrics.instantiate_nanos > 0);
        assert!(metrics.execute_nanos > 0);
        assert!(metrics.resource_usage_available);
        assert!(metrics.peak_rss_bytes > 0);
    }
}

#[cfg(test)]
mod evidence_bounds_tests {
    use super::Sink;
    use std::collections::BTreeMap;

    #[test]
    fn aggregate_is_combined_and_failed_append_preserves_prefix() {
        let sink = Sink::new(3);
        sink.append("stdout", b"ab").unwrap();
        sink.append("stderr", b"c").unwrap();
        assert!(sink.append("choice", b"d").is_err());
        let streams = sink.into_streams().unwrap();
        assert_eq!(streams.len(), 2);
        assert_eq!(streams["stdout"], b"ab");
        assert_eq!(streams["stderr"], b"c");
    }

    #[test]
    fn aggregate_addition_overflow_is_refused() {
        let streams = BTreeMap::from([("choice".into(), vec![0])]);
        let error = Sink::retained_bytes(&streams, u64::MAX).unwrap_err();
        assert_eq!(error.to_string(), "linked evidence capacity overflow");
    }
}

#[cfg(test)]
#[path = "linked/streaming_tests.rs"]
mod streaming_tests;

#[cfg(test)]
mod timing_tests {
    use super::*;
    use std::{
        cell::RefCell,
        time::{Duration, Instant},
    };
    thread_local! {
        static WORK: RefCell<Option<Box<dyn FnOnce()>>> = RefCell::new(None);
    }
    pub(super) fn after_terminal() {
        let work = WORK.with(|slot| slot.borrow_mut().take());
        if let Some(work) = work {
            work();
        }
    }
    struct Reset;
    impl Drop for Reset {
        fn drop(&mut self) {
            WORK.with(|slot| slot.borrow_mut().take());
        }
    }
    #[test]
    fn actual_native_terminal_excludes_postterminal_work_and_retains_separate_cleanup() {
        let _reset = Reset;
        WORK.with(|slot| {
            *slot.borrow_mut() = Some(Box::new(|| {
                let (_sender, receiver) = std::sync::mpsc::channel::<()>();
                assert_eq!(
                    receiver.recv_timeout(Duration::from_millis(20)),
                    Err(std::sync::mpsc::RecvTimeoutError::Timeout)
                );
            }))
        });
        let module = b"(module (memory (export \"memory\") 1) (func (export \"_start\")))".to_vec();
        let mut prepared = prepare(Request {
            module_sha256: format!("{:x}", Sha256::digest(&module)),
            module,
            limits: Limits {
                memory_bytes: 65536,
                fuel: 1000,
                output_bytes: 1024,
                evidence_bytes: 4096,
                wall_timeout: Duration::from_secs(5),
            },
            model: None,
            namespace: None,
            storage: None,
        })
        .unwrap();
        prepared.conditions(Conditions::compiled());
        let started = Instant::now();
        let observed = execute_prepared(prepared).unwrap();
        let wall = started.elapsed();
        assert_eq!(observed.outcome.termination, "exit");
        let timing = observed.timing.expect("native terminal timing absent");
        assert!(timing.terminal_to_return_nanos() >= 20_000_000);
        assert_eq!(
            observed.authorization_to_terminal_nanos,
            Some(timing.authorization_to_terminal_nanos())
        );
        assert!(
            u128::from(timing.authorization_to_terminal_nanos())
                + u128::from(timing.terminal_to_return_nanos())
                <= wall.as_nanos()
        );
    }
}
