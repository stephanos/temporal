use crate::{
    abi,
    protocol::{self, Authorization, Execute, Outcome, Reply},
};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::HashSet;
use std::io::{BufRead, BufReader, Read, Write};
use std::time::Instant;
use wasmtime::{
    CodeBuilder, Config, Engine, Error, ExternType, Linker, Module, ResourceLimiter, Result, Store,
    Trap, ValType,
};

/// Whole cooperative Temporal workloads make one host callback per scheduler
/// decision; the environment's own call limit is the operative bound.
const MAX_CALLBACKS: u64 = 1 << 28;

pub struct Limits {
    pub max: usize,
    pub peak: usize,
    pub failure: Option<&'static str>,
}
impl ResourceLimiter for Limits {
    fn memory_growing(&mut self, _: usize, desired: usize, maximum: Option<usize>) -> Result<bool> {
        if desired > self.max || maximum.is_some_and(|max| desired > max) {
            self.failure = Some("capacity");
            return Err(Error::msg("guest linear memory capacity"));
        }
        self.peak = self.peak.max(desired);
        Ok(true)
    }
    fn memory_grow_failed(&mut self, error: Error) -> Result<()> {
        self.failure = Some("infrastructure");
        Err(error)
    }
    fn table_growing(&mut self, _: usize, desired: usize, maximum: Option<usize>) -> Result<bool> {
        if desired > 1_000_000 || maximum.is_some_and(|max| desired > max) {
            self.failure = Some("capacity");
            return Err(Error::msg("guest table capacity"));
        }
        Ok(true)
    }
    fn table_grow_failed(&mut self, error: Error) -> Result<()> {
        self.failure = Some("infrastructure");
        Err(error)
    }
    fn instances(&self) -> usize {
        1
    }
    fn memories(&self) -> usize {
        1
    }
    fn tables(&self) -> usize {
        8
    }
}
pub struct State {
    pub memory: Option<wasmtime::Memory>,
    pub input: Box<dyn BufRead + Send>,
    pub output: Box<dyn Write + Send>,
    pub limits: Limits,
    pub calls: u64,
    pub completed_calls: u64,
    pub output_limit: u64,
    pub output_bytes: u64,
    pub terminal: Option<Outcome>,
    pub argument_sizes: Option<(u32, u32)>,
    pub environment_sizes: Option<(u32, u32)>,
    pub model: Option<crate::model::Model>,
    pub namespace: Option<crate::namespace::Namespace>,
    pub request_scratch: Vec<u8>,
    pub write_scratch: protocol::WriteBuffers,
}
impl State {
    pub fn fail(&mut self, kind: &'static str, message: impl Into<String>) -> Error {
        let message = message.into();
        self.terminal = Some(Outcome::new(kind, message.clone()));
        Error::msg(message)
    }
    fn model_call<T>(
        &mut self,
        body: impl FnOnce(&mut crate::model::Model, u64) -> Result<T>,
    ) -> Result<T> {
        self.model_operation(|model, _, id| body(model, id))
    }
    fn model_operation<T>(
        &mut self,
        body: impl FnOnce(
            &mut crate::model::Model,
            Option<&mut crate::namespace::Namespace>,
            u64,
        ) -> Result<T>,
    ) -> Result<T> {
        if self.calls >= self.model.as_ref().unwrap().call_limit() {
            return Err(self.fail("capacity", "WASI capacity: callbacks"));
        }
        self.calls += 1;
        let model = self.model.as_mut().unwrap();
        match body(model, self.namespace.as_mut(), self.calls) {
            Ok(result) => {
                self.completed_calls = self.calls;
                if let Err(error) = model.flush(&mut self.output, false) {
                    return Err(self.fail(protocol::classification(&error), error.to_string()));
                }
                Ok(result)
            }
            Err(error) => {
                let kind =
                    if let Some(boundary) = error.downcast_ref::<crate::namespace::Boundary>() {
                        boundary.kind
                    } else if model.divergence.is_some() {
                        "divergence"
                    } else if error.to_string().contains("capacity:") {
                        "capacity"
                    } else {
                        "invalid"
                    };
                Err(self.fail(kind, error.to_string()))
            }
        }
    }
    pub fn clock_time_get(&mut self, clock_id: u32, precision: u64) -> Result<u64> {
        self.model_call(|model, id| model.clock_read(id, clock_id, precision))?
            .ok_or_else(|| {
                self.fail(
                    "unsupported",
                    format!("clock_time_get: {}", crate::model::CPU_CLOCK_UNSUPPORTED),
                )
            })
    }
    pub fn strings(&mut self, operation: abi::Operation) -> Result<(Vec<u8>, Vec<u32>)> {
        self.model_call(|model, id| model.strings(id, operation))
    }
    pub fn random(&mut self, length: usize) -> Result<Vec<u8>> {
        self.model_call(|model, id| model.random(id, length))
    }
    pub fn yield_call(&mut self) -> Result<()> {
        self.model_call(|model, id| model.yield_call(id))
    }
    pub fn poll_call(
        &mut self,
        subscriptions: &[crate::transcript::Subscription],
    ) -> Result<crate::model::PollResult> {
        let result = self.model_operation(|model, namespace, id| {
            model.poll_owned_call(id, subscriptions, namespace.as_deref())
        })?;
        if !result.unsupported.is_empty() {
            return Err(self.fail(
                "unsupported",
                format!("poll_oneoff: {}", result.unsupported),
            ));
        }
        Ok(result)
    }
    pub fn runtime_typed(&mut self, op: abi::Operation, bytes: Vec<u8>) -> Result<Vec<u8>> {
        self.model_call(|model, _| model.runtime_typed(op, bytes))
    }
    pub fn namespace_call(
        &mut self,
        call: crate::namespace_call::Call,
    ) -> Result<crate::namespace::Reply> {
        let name = call.name;
        let result = self.model_operation(|model, namespace, id| {
            let namespace = namespace.unwrap();
            let result = model.namespace_call(id, namespace, call);
            let (stdout, stderr) = namespace.take_output();
            model.namespace_output(&stdout, &stderr);
            result
        });
        self.output_bytes = self.namespace.as_ref().unwrap().offered_output();
        let reply = result?;
        if let Some(message) = &reply.unsupported {
            return Err(self.fail("unsupported", format!("{name}: {message}")));
        }
        Ok(reply)
    }
    #[cfg(test)]
    pub fn runtime(&mut self, op: &str, bytes: Vec<u8>) -> Result<Vec<u8>> {
        self.model_call(|model, _| model.runtime(op, bytes))
    }
    pub fn callback_input(&mut self, input: protocol::CallbackInput<'_>) -> Result<Reply> {
        let op = input.op();
        if self.calls
            >= self
                .model
                .as_ref()
                .map_or(MAX_CALLBACKS, |model| model.call_limit())
        {
            return Err(self.fail(
                "capacity",
                if self.model.is_some() {
                    "WASI capacity: callbacks"
                } else {
                    "callback capacity"
                },
            ));
        }
        self.calls += 1;
        #[cfg(test)]
        if let Some(model) = &mut self.model {
            match model.handle_input(self.calls, &input) {
                Ok(Some(reply)) => {
                    self.completed_calls = self.calls;
                    if let Err(error) = model.flush(&mut self.output, false) {
                        return Err(self.fail(protocol::classification(&error), error.to_string()));
                    }
                    if reply.unsupported {
                        return Err(self.fail("unsupported", format!("{op}: {}", reply.message)));
                    }
                    return Ok(reply);
                }
                Ok(None) => (),
                Err(error) => {
                    let kind = if model.divergence.is_some() {
                        "divergence"
                    } else if error.to_string().contains("capacity:") {
                        "capacity"
                    } else {
                        "invalid"
                    };
                    return Err(self.fail(kind, error.to_string()));
                }
            }
        }
        #[cfg(feature = "diagnostic-profile")]
        crate::diagnostic::remote_callback();
        let result = (|| {
            if let Some(model) = &mut self.model {
                model.flush_output(&mut self.output)?;
            }
            #[cfg(feature = "diagnostic-profile")]
            let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
            let local_state = self.model.as_mut().map(crate::model::Model::request_state);
            #[cfg(feature = "diagnostic-profile")]
            drop(encoding);
            #[cfg(feature = "diagnostic-profile")]
            let remote_send = crate::diagnostic::RemoteSend::new();
            protocol::send_callback(
                &mut self.output,
                &mut self.request_scratch,
                self.calls,
                &input,
                local_state,
            )?;
            #[cfg(feature = "diagnostic-profile")]
            drop(remote_send);
            let mut transcript = Vec::new();
            let reply: Reply = loop {
                let bytes = {
                    #[cfg(feature = "diagnostic-profile")]
                    let _wait =
                        crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::RemoteReadWait);
                    protocol::read_frame(&mut self.input)?
                };
                #[cfg(feature = "diagnostic-profile")]
                let _encoding =
                    crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
                let frame: Value = serde_json::from_slice(&bytes).map_err(Error::new)?;
                if frame["type"] == "interrupt" {
                    let interrupt: protocol::Interrupt =
                        serde_json::from_slice(&bytes).map_err(Error::new)?;
                    interrupt.validate()?;
                    return Err(Error::new(interrupt));
                }
                if self.model.is_some() && frame["type"] == "remote_transcript" {
                    #[derive(serde::Deserialize)]
                    #[serde(deny_unknown_fields)]
                    struct Chunk {
                        #[serde(rename = "type")]
                        _kind: String,
                        data_base64: String,
                    }
                    let chunk: Chunk = serde_json::from_slice(&bytes).map_err(Error::new)?;
                    if chunk.data_base64.len() > (64 * 1024usize).div_ceil(3) * 4 {
                        return Err(Error::msg("remote transcript chunk capacity"));
                    }
                    let data = STANDARD.decode(chunk.data_base64).map_err(Error::new)?;
                    if data.is_empty()
                        || data.len() > 64 * 1024
                        || transcript.len() + data.len() > protocol::FRAME_LIMIT
                    {
                        return Err(Error::msg("remote transcript chunk capacity"));
                    }
                    transcript.extend(data);
                    continue;
                }
                break serde_json::from_slice(&bytes).map_err(Error::new)?;
            };
            if reply.kind != "reply" || reply.id != self.calls || !reply.output.is_object() {
                return Err(Error::msg("invalid callback reply type, ID or output"));
            }
            if let Some(model) = &mut self.model {
                model.remote(&reply, &transcript)?;
                self.completed_calls = self.calls;
                model.flush(&mut self.output, false)?;
            }
            Ok(reply)
        })();
        match result {
            Ok(reply) if reply.unsupported => {
                Err(self.fail("unsupported", format!("{op}: {}", reply.message)))
            }
            Ok(reply) => Ok(reply),
            Err(error) => Err(self.fail(protocol::classification(&error), error.to_string())),
        }
    }
    pub fn callback_typed(&mut self, operation: abi::Operation, input: Value) -> Result<Reply> {
        self.callback_input(protocol::CallbackInput::Typed {
            operation,
            input: &input,
        })
    }
    #[cfg(test)]
    pub fn callback(&mut self, op: &str, input: Value) -> Result<Reply> {
        self.callback_input(protocol::CallbackInput::Json { op, input: &input })
    }
}
pub(crate) const CONFIGURATION_REVISION: &str = match (
    cfg!(feature = "guest-profile"),
    cfg!(feature = "diagnostic-fuel-off"),
) {
    (false, false) => "cranelift-fuel-nan-canonical-nounwind-v2",
    (true, false) => "cranelift-fuel-nan-canonical-nounwind-v2-guest-profile",
    (false, true) => "cranelift-fuel-nan-canonical-nounwind-v2-fuel-off",
    (true, true) => "cranelift-fuel-nan-canonical-nounwind-v2-guest-profile-fuel-off",
};

pub fn execute() -> Result<()> {
    let mut input: Box<dyn BufRead + Send> = Box::new(BufReader::new(std::io::stdin()));
    let mut output: Box<dyn Write + Send> = Box::new(std::io::stdout());
    let first = protocol::read_frame(&mut input);
    if let Ok(frame) = &first
        && crate::cluster::requested(frame)
    {
        return crate::cluster::execute(frame, input, output);
    }
    let parsed: Result<Execute> = (|| {
        let mut request: Execute =
            serde_json::from_slice(&first?).map_err(|e| Error::msg(e.to_string()))?;
        request.validate()?;
        if let Some(model) = &mut request.local_model {
            model.read_inputs(&mut input)?;
        }
        Ok(request)
    })();
    let mut request = match parsed {
        Ok(request) => request,
        Err(error) => {
            return protocol::send_frame(
                &mut output,
                &Outcome::new(protocol::classification(&error), error.to_string()),
            );
        }
    };
    let mut metrics = crate::metrics::Metrics::default();
    let phase = Instant::now();
    let prepared = prepare(&request, &mut metrics);
    metrics.prepare_nanos = crate::metrics::elapsed(phase);
    let (engine, module, started) = match prepared {
        Ok(value) => value,
        Err(mut outcome) => {
            metrics.sample_process();
            outcome.metrics = Some(Box::new(metrics));
            return protocol::send_frame(&mut output, &outcome);
        }
    };
    if let Err(error) = protocol::send_frame(&mut output, &started) {
        if protocol::classification(&error) == "capacity" {
            let mut outcome = Outcome::new("capacity", error.to_string());
            metrics.sample_process();
            outcome.metrics = Some(Box::new(metrics));
            return protocol::send_frame(&mut output, &outcome);
        }
        return Err(error);
    }
    if let Err(error) = authorize(&mut *input, &request.module_sha256) {
        let mut outcome = Outcome::new(protocol::classification(&error), error.to_string());
        metrics.sample_process();
        outcome.metrics = Some(Box::new(metrics));
        return protocol::send_frame(&mut output, &outcome);
    }
    if let Some(model) = &mut request.local_model
        && let Err(error) = model.read_tape(&mut *input)
    {
        let mut outcome = Outcome::new(protocol::classification(&error), error.to_string());
        metrics.sample_process();
        outcome.metrics = Some(Box::new(metrics));
        return protocol::send_frame(&mut output, &outcome);
    }
    let namespace = match &request.namespace {
        Some(descriptor) => {
            let model = request.local_model.as_ref().unwrap();
            match crate::namespace_transport::read(
                &mut *input,
                descriptor,
                request.output_bytes,
                model.epoch_nanos,
                &model.working_directory_sha256,
            ) {
                Ok(namespace) => Some(namespace),
                Err(error) => {
                    let mut outcome = Outcome::new("invalid", error.to_string());
                    let model = crate::model::Model::new(request.local_model.take().unwrap());
                    outcome.local_model = Some(Box::new(model.metadata(0)));
                    metrics.sample_process();
                    outcome.metrics = Some(Box::new(metrics));
                    return protocol::send_frame(&mut output, &outcome);
                }
            }
        }
        None => None,
    };
    let mut store = Store::new(
        &engine,
        new_state(
            input,
            output,
            request.memory_bytes,
            request.output_bytes,
            request.local_model.map(crate::model::Model::new),
            namespace,
        ),
    );
    store.limiter(|state| &mut state.limits);
    #[cfg(not(feature = "diagnostic-fuel-off"))]
    store.set_fuel(request.fuel)?;
    #[cfg(feature = "guest-profile")]
    let guest_profile = request
        .guest_profile
        .as_ref()
        .map(|config| crate::guest_profile::Session::start(&mut store, &module, config))
        .transpose()?;
    #[cfg(feature = "guest-profile")]
    if guest_profile.is_none() {
        store.set_epoch_deadline(u64::MAX);
    }
    let linker = linker(&engine, &module)?;
    let executed = (|| {
        let phase = Instant::now();
        #[cfg(feature = "diagnostic-profile")]
        let diagnostic_phase =
            crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Instantiate);
        let instance = linker.instantiate(&mut store, &module);
        #[cfg(feature = "diagnostic-profile")]
        drop(diagnostic_phase);
        metrics.instantiate_nanos = crate::metrics::elapsed(phase);
        let instance = instance?;
        let start = instance.get_typed_func::<(), ()>(&mut store, "_start")?;
        let phase = Instant::now();
        #[cfg(feature = "diagnostic-profile")]
        let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Execute);
        let execution = start.call(&mut store, ());
        #[cfg(feature = "diagnostic-profile")]
        drop(diagnostic_phase);
        metrics.execute_nanos = crate::metrics::elapsed(phase);
        execution
    })();
    let mut outcome = terminal_outcome(&mut store, executed, request.fuel)?;
    finish_model(store.data_mut(), &mut outcome)?;
    metrics.sample_process();
    outcome.metrics = Some(Box::new(metrics));
    let sent = protocol::send_frame(&mut store.data_mut().output, &outcome);
    #[cfg(feature = "guest-profile")]
    if let Some(profile) = guest_profile
        && let Err(error) = profile.finish()
    {
        eprintln!("gomad-wasm-guest-profile warning: {error:#}");
    }
    sent
}
/// Fresh per-store state; the cluster host builds every instance store with it.
pub(crate) fn new_state(
    input: Box<dyn BufRead + Send>,
    output: Box<dyn Write + Send>,
    memory_bytes: u64,
    output_bytes: u64,
    model: Option<crate::model::Model>,
    namespace: Option<crate::namespace::Namespace>,
) -> State {
    State {
        memory: None,
        input,
        output,
        limits: Limits {
            max: memory_bytes as usize,
            peak: 0,
            failure: None,
        },
        calls: 0,
        completed_calls: 0,
        output_limit: output_bytes,
        output_bytes: 0,
        terminal: None,
        argument_sizes: None,
        environment_sizes: None,
        model,
        namespace,
        request_scratch: Vec::new(),
        write_scratch: protocol::WriteBuffers::default(),
    }
}

/// Links every admitted import to the ABI dispatcher.
pub(crate) fn linker(engine: &Engine, module: &Module) -> Result<Linker<State>> {
    let mut linker = Linker::new(engine);
    let mut defined = HashSet::new();
    for import in module.imports() {
        if !defined.insert((import.module().to_owned(), import.name().to_owned())) {
            continue;
        }
        abi::Operation::resolve(import.module(), import.name())
            .unwrap()
            .link(&mut linker)?;
    }
    Ok(linker)
}

/// Reads the single `authorize` frame and checks it against `started`.
pub(crate) fn authorize(input: &mut dyn BufRead, module_sha256: &str) -> Result<()> {
    let authorization: Authorization = serde_json::from_slice(&protocol::read_frame(input)?)
        .map_err(|e| Error::msg(e.to_string()))?;
    if authorization.kind != "authorize"
        || authorization.module_sha256 != module_sha256
        || authorization.configuration != CONFIGURATION_REVISION
    {
        return Err(Error::msg("execution authorization identity mismatch"));
    }
    Ok(())
}

/// The store's terminal outcome with fuel, guest memory peak and output bytes.
pub(crate) fn terminal_outcome(
    store: &mut Store<State>,
    executed: Result<()>,
    offered_fuel: u64,
) -> Result<Outcome> {
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
            Err(error) => execution_failure(error, store.data().limits.failure),
        });
    #[cfg(not(feature = "diagnostic-fuel-off"))]
    {
        let _ = offered_fuel;
        outcome.fuel_remaining = store.get_fuel()?;
    }
    #[cfg(feature = "diagnostic-fuel-off")]
    {
        outcome.fuel_remaining = offered_fuel;
    }
    outcome.peak_memory_bytes = store.data().limits.peak;
    outcome.output_bytes = store.data().output_bytes;
    Ok(outcome)
}

/// Flushes the cooperative model's evidence and adds its result metadata.
pub(crate) fn finish_model(state: &mut State, outcome: &mut Outcome) -> Result<()> {
    if let Some(model) = &mut state.model {
        model.flush(&mut state.output, true)?;
        outcome.local_model = Some(Box::new(model.metadata(state.completed_calls)));
        outcome.divergence = model.divergence.take().map(Box::new);
    }
    Ok(())
}

fn module_bytes(request: &Execute) -> std::result::Result<Vec<u8>, Outcome> {
    #[cfg(feature = "diagnostic-profile")]
    let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::ModuleRead);
    let file = std::fs::File::open(&request.module_path)
        .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
    let mut bytes = Vec::new();
    file.take(512 * 1024 * 1024 + 1)
        .read_to_end(&mut bytes)
        .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
    if bytes.len() > 512 * 1024 * 1024 {
        return Err(Outcome::new("capacity", "module exceeds 512 MiB"));
    }
    #[cfg(feature = "diagnostic-profile")]
    drop(diagnostic_phase);
    #[cfg(feature = "diagnostic-profile")]
    let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::ModuleHash);
    if format!("{:x}", Sha256::digest(&bytes)) != request.module_sha256 {
        return Err(Outcome::new("invalid", "module SHA-256 mismatch"));
    }
    #[cfg(feature = "diagnostic-profile")]
    drop(diagnostic_phase);
    Ok(bytes)
}
pub(crate) fn prepare(
    request: &Execute,
    metrics: &mut crate::metrics::Metrics,
) -> std::result::Result<(Engine, Module, Value), Outcome> {
    #[cfg(feature = "diagnostic-profile")]
    let diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::EngineInit);
    let mut config = Config::new();
    config
        .consume_fuel(!cfg!(feature = "diagnostic-fuel-off"))
        .native_unwind_info(false)
        .cranelift_nan_canonicalization(true)
        .wasm_relaxed_simd(false)
        .wasm_memory64(false)
        .wasm_multi_memory(false);
    #[cfg(feature = "guest-profile")]
    config.epoch_interruption(true);
    let engine = Engine::new(&config)
        .map_err(|error| Outcome::new("infrastructure", format!("{error:#}")))?;
    #[cfg(feature = "diagnostic-profile")]
    drop(diagnostic_phase);
    let module = match &request.compile_cache {
        Some(directory) => crate::cache::module(
            &engine,
            || module_bytes(request),
            directory,
            &request.module_sha256,
            metrics,
        )?,
        None => compile(&engine, &module_bytes(request)?, metrics)?,
    };
    let resources = module.resources_required();
    if resources.num_tables > 8
        || resources
            .max_initial_table_size
            .is_some_and(|size| size > 1_000_000)
    {
        return Err(Outcome::new(
            "capacity",
            "initial guest tables exceed limit",
        ));
    }
    let mut imports = Vec::new();
    for import in module.imports() {
        let ExternType::Func(ty) = import.ty() else {
            return Err(Outcome::new("unsupported", "non-function import"));
        };
        let params: Vec<_> = ty.params().map(type_name).collect();
        let results: Vec<_> = ty.results().map(type_name).collect();
        let Some(operation) = abi::Operation::resolve(import.module(), import.name()) else {
            return Err(Outcome::new(
                "unsupported",
                format!("unknown import {}::{}", import.module(), import.name()),
            ));
        };
        let (expected_params, expected_results) = operation.signature();
        if params != expected_params || results != expected_results {
            return Err(Outcome::new(
                "unsupported",
                format!(
                    "invalid import signature {}::{}",
                    import.module(),
                    import.name()
                ),
            ));
        }
        imports.push(json!({"module":import.module(),"name":import.name(),"params":params,"results":results}));
    }
    let mut exports = Vec::new();
    let mut pages = None;
    let mut entry = false;
    for export in module.exports() {
        let kind = match export.ty() {
            ExternType::Memory(ty) => {
                if export.name() != "memory" || ty.is_64() || ty.is_shared() || pages.is_some() {
                    return Err(Outcome::new(
                        "unsupported",
                        "expected one unshared memory32 export",
                    ));
                }
                pages = Some(ty.minimum());
                "memory"
            }
            ExternType::Func(ty) => {
                if export.name() == "_start" {
                    entry = ty.params().len() == 0 && ty.results().len() == 0;
                }
                "func"
            }
            ExternType::Table(_) => "table",
            ExternType::Global(_) => "global",
            _ => "other",
        };
        exports.push(json!({"name":export.name(),"kind":kind}));
    }
    let Some(pages) = pages else {
        return Err(Outcome::new("unsupported", "missing exported memory"));
    };
    if !entry {
        return Err(Outcome::new("unsupported", "missing _start: () -> ()"));
    }
    if pages > request.memory_bytes / 65536 {
        return Err(Outcome::new(
            "capacity",
            "initial guest memory exceeds limit",
        ));
    }
    let started = json!({"schema":protocol::SCHEMA,"type":"started","engine":{"name":"wasmtime","version":"47.0.3","configuration":CONFIGURATION_REVISION,"host_os":std::env::consts::OS,"host_arch":std::env::consts::ARCH},"module_sha256":request.module_sha256,"imports":imports,"exports":exports,"initial_memory_pages":pages});
    Ok((engine, module, started))
}
pub(crate) fn compile(
    engine: &Engine,
    bytes: &[u8],
    metrics: &mut crate::metrics::Metrics,
) -> std::result::Result<Module, Outcome> {
    #[cfg(feature = "diagnostic-profile")]
    let _diagnostic_phase = crate::diagnostic::PhaseSpan::new(crate::diagnostic::Phase::Compile);
    let phase = Instant::now();
    let result = (|| {
        let mut builder = CodeBuilder::new(engine);
        builder
            .wasm_binary_or_text(bytes, None)
            .map_err(|error| Outcome::new("invalid", format!("{error:#}")))?;
        builder.compile_module().map_err(compilation_failure)
    })();
    metrics.compile_nanos += crate::metrics::elapsed(phase);
    result
}

fn type_name(ty: ValType) -> &'static str {
    match ty {
        ValType::I32 => "i32",
        ValType::I64 => "i64",
        ValType::F32 => "f32",
        ValType::F64 => "f64",
        ValType::V128 => "v128",
        _ => "ref",
    }
}

pub(crate) fn execution_failure(error: Error, limiter_failure: Option<&'static str>) -> Outcome {
    let classification = limiter_failure.unwrap_or_else(|| match error.downcast_ref::<Trap>() {
        Some(Trap::OutOfFuel) => "capacity",
        Some(_) => "trap",
        None => "infrastructure",
    });
    Outcome::new(classification, format!("{error:#}"))
}

fn compilation_failure(error: Error) -> Outcome {
    let cause = error.root_cause().to_string();
    let message = error
        .downcast_ref::<wasmtime::wasmparser::BinaryReaderError>()
        .map(|error| error.message())
        .or_else(|| {
            // Function translation wraps parser errors in an unexported WasmError.
            let (offset, message) = cause
                .strip_prefix("Invalid input WebAssembly code at offset ")?
                .split_once(": ")?;
            if offset.is_empty() || !offset.bytes().all(|byte| byte.is_ascii_digit()) {
                return None;
            }
            Some(message)
        });
    let classification = match message {
        Some(
            "memory64 must be enabled for 64-bit memories"
            | "memory64 must be enabled for 64-bit tables"
            | "threads must be enabled for shared memories"
            | "threads support is not enabled"
            | "multiple memories"
            | "relaxed SIMD support is not enabled",
        ) => "unsupported",
        Some(_) => "invalid",
        None => "infrastructure",
    };
    Outcome::new(classification, format!("{error:#}"))
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use std::sync::{Arc, Mutex};

    #[derive(Clone, Default)]
    struct WriteCapture(Arc<Mutex<Vec<u8>>>);
    impl Write for WriteCapture {
        fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(bytes);
            Ok(bytes.len())
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

    type CapturedFrame = (usize, Vec<u8>);

    #[derive(Clone, Default)]
    struct OriginCapture(Arc<Mutex<Vec<CapturedFrame>>>);
    impl Write for OriginCapture {
        fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
            self.0
                .lock()
                .unwrap()
                .push((bytes.as_ptr() as usize, bytes.to_vec()));
            Ok(bytes.len())
        }
        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

    struct WriteAbiFixture {
        store: Store<State>,
        instance: wasmtime::Instance,
        capture: WriteCapture,
    }
    impl WriteAbiFixture {
        fn new(pages: u32, modeled: bool, replies: Vec<u8>) -> Self {
            assert!(replies.len() <= 65536);
            assert!(matches!(pages, 2 | 65));
            let wat = format!(
                r#"(module
                (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
                (import "wasi_snapshot_preview1" "fd_pwrite" (func $pwrite (param i32 i32 i32 i64 i32) (result i32)))
                (memory (export "memory") {pages} {pages})
                (func (export "write") (param i32 i32 i32 i32) (result i32)
                    local.get 0 local.get 1 local.get 2 local.get 3 call $write)
                (func (export "pwrite") (param i32 i32 i32 i64 i32) (result i32)
                    local.get 0 local.get 1 local.get 2 local.get 3 local.get 4 call $pwrite))"#
            );
            assert!(wat.len() < 8192);
            let mut config = Config::new();
            config
                .consume_fuel(true)
                .native_unwind_info(false)
                .cranelift_nan_canonicalization(true)
                .wasm_relaxed_simd(false)
                .wasm_memory64(false)
                .wasm_multi_memory(false);
            let engine = Engine::new(&config).unwrap();
            let module = Module::new(&engine, wat).unwrap();
            let capture = WriteCapture::default();
            let mut state = clock_state(8, 65536, 1);
            state.input = Box::new(std::io::Cursor::new(replies));
            state.output = Box::new(capture.clone());
            state.output_limit = 8192;
            state.limits.max = 5 * 1024 * 1024;
            state.request_scratch = Vec::with_capacity(16384);
            state.write_scratch.bytes = Vec::with_capacity(8192);
            state.write_scratch.encoded = String::with_capacity(16384);
            if !modeled {
                state.model = None;
            }
            let mut store = Store::new(&engine, state);
            store.limiter(|state| &mut state.limits);
            store.set_fuel(1_000_000).unwrap();
            let mut linker = Linker::new(&engine);
            for import in module.imports() {
                abi::Operation::resolve(import.module(), import.name())
                    .unwrap()
                    .link(&mut linker)
                    .unwrap();
            }
            let instance = linker.instantiate(&mut store, &module).unwrap();
            Self {
                store,
                instance,
                capture,
            }
        }
        fn memory(&mut self) -> wasmtime::Memory {
            self.instance.get_memory(&mut self.store, "memory").unwrap()
        }
        fn descriptors(&mut self, iovecs: &[(u32, u32)]) {
            assert!(iovecs.len() <= 4096);
            let memory = self.memory();
            for (index, &(pointer, length)) in iovecs.iter().enumerate() {
                memory
                    .write(&mut self.store, 16 + index * 8, &pointer.to_le_bytes())
                    .unwrap();
                memory
                    .write(&mut self.store, 20 + index * 8, &length.to_le_bytes())
                    .unwrap();
            }
            memory
                .write(&mut self.store, 0, &u32::MAX.to_le_bytes())
                .unwrap();
        }
        fn invoke(
            &mut self,
            positioned: bool,
            fd: u32,
            pointer: u32,
            count: u32,
            offset: u64,
            output: u32,
        ) -> Result<i32> {
            if positioned {
                self.instance
                    .get_typed_func::<(i32, i32, i32, i64, i32), i32>(&mut self.store, "pwrite")
                    .unwrap()
                    .call(
                        &mut self.store,
                        (
                            fd as i32,
                            pointer as i32,
                            count as i32,
                            offset as i64,
                            output as i32,
                        ),
                    )
            } else {
                self.instance
                    .get_typed_func::<(i32, i32, i32, i32), i32>(&mut self.store, "write")
                    .unwrap()
                    .call(
                        &mut self.store,
                        (fd as i32, pointer as i32, count as i32, output as i32),
                    )
            }
        }
        fn origins(&self) -> (usize, usize, usize) {
            let state = self.store.data();
            (
                state.write_scratch.bytes.as_ptr() as usize,
                state.write_scratch.encoded.as_ptr() as usize,
                state.request_scratch.as_ptr() as usize,
            )
        }
        fn restored(&self) {
            let state = self.store.data();
            assert!(
                state.write_scratch.bytes.is_empty()
                    && state.write_scratch.encoded.is_empty()
                    && state.request_scratch.is_empty()
            );
            assert!(state.write_scratch.bytes.capacity() <= protocol::BUFFER_LIMIT);
            assert!(
                state.write_scratch.encoded.capacity() <= protocol::BUFFER_LIMIT.div_ceil(3) * 4
            );
            assert!(state.request_scratch.capacity() <= protocol::FRAME_LIMIT);
            assert!(self.store.get_fuel().unwrap() > 0);
            assert!(state.limits.peak <= 5 * 1024 * 1024);
        }
        fn output_count(&mut self) -> u32 {
            let mut bytes = [0; 4];
            self.memory().read(&self.store, 0, &mut bytes).unwrap();
            u32::from_le_bytes(bytes)
        }
    }
    fn write_abi_reply(id: u64, errno: u32, output: Value, modeled: bool) -> Vec<u8> {
        let mut reply = json!({"type":"reply","id":id,"errno":errno,"output":output});
        if modeled {
            reply["local_state"] = json!({"now":1,"advances":0});
            reply["transcript_base64"] = json!("");
        }
        let mut bytes = serde_json::to_vec(&reply).unwrap();
        bytes.push(b'\n');
        assert!(bytes.len() <= 65536);
        bytes
    }

    #[test]
    fn typed_write_abi_gathers_scattered_empty_and_unsigned_inputs_with_state_reuse() {
        for modeled in [false, true] {
            let replies = [
                write_abi_reply(1, 0, json!({"written":4152}), modeled),
                write_abi_reply(2, 0, json!({"written":0}), modeled),
            ]
            .concat();
            let mut first = WriteAbiFixture::new(2, modeled, replies.clone());
            let mut second = WriteAbiFixture::new(2, modeled, replies);
            assert_ne!(first.origins().0, second.origins().0);
            assert_ne!(first.origins().1, second.origins().1);
            assert_ne!(first.origins().2, second.origins().2);
            for (fixture, positioned, fd) in [(&mut first, false, 1), (&mut second, true, u32::MAX)]
            {
                let origins = fixture.origins();
                fixture.descriptors(&[(256, 32), (512, 24), (4096, 4096)]);
                let bytes = [vec![0; 32], vec![255; 24], vec![42; 4096]].concat();
                let memory = fixture.memory();
                memory.write(&mut fixture.store, 256, &[0; 32]).unwrap();
                memory.write(&mut fixture.store, 512, &[255; 24]).unwrap();
                memory.write(&mut fixture.store, 4096, &[42; 4096]).unwrap();
                assert_eq!(
                    fixture.invoke(positioned, fd, 16, 3, u64::MAX, 0).unwrap(),
                    0
                );
                assert_eq!(fixture.output_count(), 4152);
                assert_eq!(fixture.origins(), origins);
                fixture.restored();
                fixture.descriptors(&[]);
                assert_eq!(fixture.invoke(positioned, fd, 16, 0, 0, 0).unwrap(), 0);
                assert_eq!(fixture.output_count(), 0);
                assert_eq!(fixture.origins(), origins);
                fixture.restored();
                let mut expected = Vec::new();
                for (id, data, offset) in [(1, bytes.as_slice(), u64::MAX), (2, &[][..], 0)] {
                    let mut input = json!({"data_base64":STANDARD.encode(data),"fd":fd});
                    if positioned {
                        input["offset"] = json!(offset);
                    }
                    let mut frame = json!({"type":"call","id":id,"op":if positioned {"fd_pwrite"} else {"fd_write"},"input":input});
                    if modeled {
                        frame["local_state"] = json!({"advances":0,"now":1,"transcript_bytes":0});
                    }
                    protocol::send_frame(&mut expected, &frame).unwrap();
                }
                assert_eq!(*fixture.capture.0.lock().unwrap(), expected);
                assert_eq!(fixture.store.data().calls, 2);
                assert_eq!(
                    fixture.store.data().completed_calls,
                    if modeled { 2 } else { 0 }
                );
                assert_eq!(
                    fixture.store.data().output_bytes,
                    if fd == 1 { 4152 } else { 0 }
                );
            }
        }
    }

    #[test]
    fn typed_write_abi_pointer_and_capacity_priority_precede_callback_mutation() {
        for (iovecs, pointer, count, output, kind, expected) in [
            (
                vec![],
                u32::MAX,
                4097,
                u32::MAX,
                "capacity",
                "iovec capacity",
            ),
            (
                vec![],
                u32::MAX,
                1,
                u32::MAX,
                "trap",
                "guest memory range out of bounds",
            ),
            (
                vec![(u32::MAX, 1)],
                16,
                1,
                u32::MAX,
                "trap",
                "guest memory range out of bounds",
            ),
            (
                vec![(u32::MAX, protocol::BUFFER_LIMIT as u32 + 1)],
                16,
                1,
                u32::MAX,
                "capacity",
                "ABI buffer capacity",
            ),
            (
                vec![(256, 1)],
                16,
                1,
                u32::MAX,
                "trap",
                "guest memory range out of bounds",
            ),
        ] {
            let mut fixture = WriteAbiFixture::new(2, true, Vec::new());
            fixture.descriptors(&iovecs);
            let origins = fixture.origins();
            assert!(
                fixture
                    .invoke(true, 1, pointer, count, u64::MAX, output)
                    .is_err()
            );
            assert_eq!(
                fixture.store.data().terminal.as_ref().unwrap().message,
                expected
            );
            assert_eq!(
                fixture.store.data().terminal.as_ref().unwrap().termination,
                kind
            );
            assert_eq!(
                (
                    fixture.store.data().calls,
                    fixture.store.data().completed_calls,
                    fixture.store.data().output_bytes
                ),
                (0, 0, 0)
            );
            assert_eq!(fixture.output_count(), u32::MAX);
            assert_eq!(fixture.origins(), origins);
            assert!(fixture.capture.0.lock().unwrap().is_empty());
            fixture.restored();
        }
        let mut fixture = WriteAbiFixture::new(65, true, Vec::new());
        fixture.descriptors(&[(4096, protocol::BUFFER_LIMIT as u32), (4096, 1)]);
        assert!(fixture.invoke(true, 13, 16, 2, 0, 0).is_err());
        assert_eq!(
            fixture.store.data().terminal.as_ref().unwrap().message,
            "iovec buffer capacity"
        );
        assert_eq!(
            fixture.store.data().terminal.as_ref().unwrap().termination,
            "capacity"
        );
        assert_eq!(
            (
                fixture.store.data().calls,
                fixture.store.data().output_bytes
            ),
            (0, 0)
        );
        assert!(fixture.capture.0.lock().unwrap().is_empty());
        fixture.restored();
    }

    #[test]
    fn typed_write_abi_max_buffer_and_reservation_failures_restore_ownership() {
        let mut fixture = WriteAbiFixture::new(
            65,
            true,
            write_abi_reply(1, 0, json!({"written":protocol::BUFFER_LIMIT}), true),
        );
        fixture.descriptors(&[(4096, protocol::BUFFER_LIMIT as u32)]);
        fixture.memory().data_mut(&mut fixture.store)[4096..4096 + protocol::BUFFER_LIMIT].fill(42);
        assert_eq!(fixture.invoke(true, 13, 16, 1, u64::MAX, 0).unwrap(), 0);
        assert_eq!(fixture.output_count(), protocol::BUFFER_LIMIT as u32);
        let frame: Value = serde_json::from_slice(&fixture.capture.0.lock().unwrap()).unwrap();
        assert_eq!(
            frame["input"],
            json!({"fd":13,"offset":u64::MAX,"data_base64":STANDARD.encode(vec![42;protocol::BUFFER_LIMIT])})
        );
        let mut expected = Vec::new();
        protocol::send_frame(
            &mut expected,
            &json!({"id":1,"input":frame["input"],"local_state":{"advances":0,"now":1,"transcript_bytes":0},"op":"fd_pwrite","type":"call"}),
        )
        .unwrap();
        assert_eq!(*fixture.capture.0.lock().unwrap(), expected);
        assert_eq!(fixture.store.data().calls, 1);
        assert_eq!(fixture.store.data().output_bytes, 0);
        fixture.restored();
        for output_budget in [true, false] {
            let mut fixture = WriteAbiFixture::new(2, true, Vec::new());
            fixture.descriptors(&[(256, 32)]);
            let origins = fixture.origins();
            if output_budget {
                fixture.store.data_mut().output_limit = 31;
            } else {
                fixture.store.data_mut().calls = 8;
            }
            assert!(fixture.invoke(false, 1, 16, 1, 0, 0).is_err());
            assert_eq!(
                fixture.store.data().terminal.as_ref().unwrap().termination,
                "capacity"
            );
            assert_eq!(
                fixture.store.data().terminal.as_ref().unwrap().message,
                if output_budget {
                    "output capacity"
                } else {
                    "WASI capacity: callbacks"
                }
            );
            assert_eq!(
                fixture.store.data().calls,
                if output_budget { 0 } else { 8 }
            );
            assert_eq!(
                fixture.store.data().output_bytes,
                if output_budget { 0 } else { 32 }
            );
            assert_eq!(fixture.origins(), origins);
            assert!(fixture.capture.0.lock().unwrap().is_empty());
            assert_eq!(fixture.output_count(), u32::MAX);
            fixture.restored();
        }
    }

    #[test]
    fn typed_write_abi_charges_output_by_bytes_actually_written() {
        for modeled in [false, true] {
            // A 32-byte write is accepted partially, then the 16-byte remainder
            // is retried; the declared 32-byte bound must not report capacity.
            let replies = [
                write_abi_reply(1, 0, json!({"written":16}), modeled),
                write_abi_reply(2, 0, json!({"written":16}), modeled),
                write_abi_reply(3, 6, json!({}), modeled),
            ]
            .concat();
            let mut fixture = WriteAbiFixture::new(2, modeled, replies);
            fixture.store.data_mut().output_limit = 32;
            fixture
                .memory()
                .write(&mut fixture.store, 256, &[7; 32])
                .unwrap();
            fixture.descriptors(&[(256, 32)]);
            assert_eq!(fixture.invoke(false, 1, 16, 1, 0, 0).unwrap(), 0);
            assert_eq!(fixture.output_count(), 16);
            assert_eq!(fixture.store.data().output_bytes, 16);
            fixture.descriptors(&[(272, 16)]);
            assert_eq!(fixture.invoke(false, 1, 16, 1, 0, 0).unwrap(), 0);
            assert_eq!(fixture.output_count(), 16);
            assert_eq!(fixture.store.data().output_bytes, 32);
            assert!(fixture.store.data().terminal.is_none());
            fixture.restored();
            // A failed write (errno) charges nothing; the bound stays exact.
            fixture.descriptors(&[]);
            assert_eq!(fixture.invoke(false, 2, 16, 0, 0, 0).unwrap(), 6);
            assert_eq!(fixture.store.data().output_bytes, 32);
            // At the bound, any further byte is still rejected before the callback.
            fixture.descriptors(&[(256, 1)]);
            let sent = fixture.capture.0.lock().unwrap().len();
            assert!(fixture.invoke(false, 1, 16, 1, 0, 0).is_err());
            let terminal = fixture.store.data().terminal.as_ref().unwrap();
            assert_eq!(
                (terminal.termination, terminal.message.as_str()),
                ("capacity", "output capacity")
            );
            assert_eq!(fixture.store.data().output_bytes, 32);
            assert_eq!(fixture.capture.0.lock().unwrap().len(), sent);
            fixture.restored();
        }
    }

    #[test]
    fn typed_write_abi_reply_errors_chunks_interrupts_and_transport_restore_buffers() {
        let mut unsupported: Value =
            serde_json::from_slice(&write_abi_reply(1, 0, json!({}), true)).unwrap();
        unsupported["unsupported"] = json!(true);
        unsupported["message"] = json!("outside profile");
        let mut unsupported_reply = Vec::new();
        protocol::send_frame(&mut unsupported_reply, &unsupported).unwrap();
        for (reply, expected, completed, errno) in [
            (write_abi_reply(1, 5, json!({}), true), None, 1, Some(5)),
            (
                write_abi_reply(1, 0, json!({"written":33}), true),
                Some(("invalid", "reply written out of range")),
                1,
                None,
            ),
            (
                write_abi_reply(1, 0, json!({"written":"32"}), true),
                Some(("invalid", "invalid unsigned reply field written")),
                1,
                None,
            ),
            (
                write_abi_reply(2, 0, json!({"written":32}), true),
                Some(("invalid", "invalid callback reply type, ID or output")),
                0,
                None,
            ),
            (
                write_abi_reply(1, 0, json!({"written":32}), false),
                Some(("invalid", "missing remote model state")),
                0,
                None,
            ),
            (
                unsupported_reply,
                Some(("unsupported", "fd_write: outside profile")),
                1,
                None,
            ),
            (
                b"{\"type\":\"interrupt\",\"termination\":\"stop\",\"message\":\"modeled\"}\n"
                    .to_vec(),
                Some(("stop", "modeled")),
                0,
                None,
            ),
            (
                b"{\"type\":\"interrupt\",\"termination\":\"crash\",\"message\":\"modeled\"}\n"
                    .to_vec(),
                Some(("crash", "modeled")),
                0,
                None,
            ),
            (
                b"{\"type\":\"remote_transcript\",\"data_base64\":\"\"}\n".to_vec(),
                Some(("invalid", "remote transcript chunk capacity")),
                0,
                None,
            ),
        ] {
            let mut fixture = WriteAbiFixture::new(2, true, reply);
            fixture.descriptors(&[(256, 32)]);
            let origins = fixture.origins();
            let result = fixture.invoke(false, 2, 16, 1, 0, 0);
            if let Some((kind, message)) = expected {
                assert!(result.is_err());
                let terminal = fixture.store.data().terminal.as_ref().unwrap();
                assert_eq!(
                    (terminal.termination, terminal.message.as_str()),
                    (kind, message)
                );
            } else {
                assert_eq!(result.unwrap(), errno.unwrap());
            }
            assert_eq!(fixture.output_count(), u32::MAX);
            assert_eq!(
                (
                    fixture.store.data().calls,
                    fixture.store.data().completed_calls,
                    fixture.store.data().output_bytes
                ),
                // A failed write wrote nothing; terminal failures keep the reservation.
                (1, completed, if errno.is_some() { 0 } else { 32 })
            );
            assert_eq!(fixture.origins(), origins);
            fixture.restored();
        }
        let replies = [
            b"{\"type\":\"remote_transcript\",\"data_base64\":\"AQI=\"}\n".to_vec(),
            write_abi_reply(1, 0, json!({"written":32}), true),
        ]
        .concat();
        let mut fixture = WriteAbiFixture::new(2, true, replies);
        fixture.descriptors(&[(256, 32)]);
        assert_eq!(fixture.invoke(true, 13, 16, 1, 0, 0).unwrap(), 0);
        assert_eq!(fixture.output_count(), 32);
        assert_eq!(
            fixture.store.data().model.as_ref().unwrap().state()["transcript_bytes"],
            2
        );
        fixture.restored();
        struct Broken;
        impl Write for Broken {
            fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
                Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe))
            }
            fn flush(&mut self) -> std::io::Result<()> {
                unreachable!()
            }
        }
        let mut fixture = WriteAbiFixture::new(2, true, Vec::new());
        fixture.descriptors(&[(256, 32)]);
        fixture.store.data_mut().output = Box::new(Broken);
        let origins = fixture.origins();
        assert!(fixture.invoke(true, 13, 16, 1, 0, 0).is_err());
        assert_eq!(
            fixture.store.data().terminal.as_ref().unwrap().termination,
            "infrastructure"
        );
        assert_eq!(
            (
                fixture.store.data().calls,
                fixture.store.data().completed_calls
            ),
            (1, 0)
        );
        assert_eq!(fixture.origins(), origins);
        fixture.restored();

        struct ShortOutput {
            bytes: WriteCapture,
            fail_after_prefix: bool,
            sent: usize,
        }
        impl Write for ShortOutput {
            fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
                if self.fail_after_prefix && self.sent != 0 {
                    return Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe));
                }
                let count = bytes.len().min(3);
                self.bytes.write_all(&bytes[..count])?;
                self.sent += count;
                Ok(count)
            }
            fn flush(&mut self) -> std::io::Result<()> {
                self.bytes.flush()
            }
        }
        for fail_after_prefix in [false, true] {
            let mut fixture =
                WriteAbiFixture::new(2, true, write_abi_reply(1, 0, json!({"written":32}), true));
            fixture.descriptors(&[(256, 32)]);
            let origins = fixture.origins();
            let bytes = WriteCapture::default();
            fixture.store.data_mut().output = Box::new(ShortOutput {
                bytes: bytes.clone(),
                fail_after_prefix,
                sent: 0,
            });
            let result = fixture.invoke(true, 13, 16, 1, 0, 0);
            assert_eq!(result.is_err(), fail_after_prefix);
            let mut expected = Vec::new();
            protocol::send_frame(&mut expected, &json!({"id":1,"input":{"data_base64":STANDARD.encode([0;32]),"fd":13,"offset":0},"local_state":{"advances":0,"now":1,"transcript_bytes":0},"op":"fd_pwrite","type":"call"})).unwrap();
            if fail_after_prefix {
                assert_eq!(
                    fixture.store.data().terminal.as_ref().unwrap().termination,
                    "infrastructure"
                );
                assert_eq!(bytes.0.lock().unwrap().as_slice(), &expected[..3]);
                assert_eq!(fixture.output_count(), u32::MAX);
                assert_eq!(fixture.store.data().completed_calls, 0);
            } else {
                assert_eq!(result.unwrap(), 0);
                assert_eq!(*bytes.0.lock().unwrap(), expected);
                assert_eq!(fixture.output_count(), 32);
                assert_eq!(fixture.store.data().completed_calls, 1);
            }
            assert_eq!(fixture.store.data().calls, 1);
            assert_eq!(fixture.origins(), origins);
            fixture.restored();
        }
    }
    #[test]
    fn typed_write_byte_and_base64_buffers_reuse_independent_live_states() {
        let mut first = clock_state(8, 65536, 1);
        let mut second = clock_state(8, 65536, 1);
        for state in [&mut first, &mut second] {
            state.write_scratch.bytes = Vec::with_capacity(8192);
            state.write_scratch.encoded = String::with_capacity(16384);
        }
        let origins = [
            (
                first.write_scratch.bytes.as_ptr() as usize,
                first.write_scratch.encoded.as_ptr() as usize,
            ),
            (
                second.write_scratch.bytes.as_ptr() as usize,
                second.write_scratch.encoded.as_ptr() as usize,
            ),
        ];
        assert_ne!(origins[0].0, origins[1].0);
        assert_ne!(origins[0].1, origins[1].1);
        for (state, origin) in [(&mut first, origins[0]), (&mut second, origins[1])] {
            for lengths in [&[32usize, 24, 4096][..], &[][..], &[24][..]] {
                let buffers = &mut state.write_scratch;
                buffers.begin(lengths.iter().sum()).unwrap();
                for &length in lengths {
                    buffers.bytes.extend(vec![42; length]);
                }
                buffers.encode().unwrap();
                assert_eq!(buffers.encoded, STANDARD.encode(&buffers.bytes));
                assert_eq!(
                    buffers.bytes.as_ptr() as usize,
                    origin.0,
                    "guest-byte allocation was not reused"
                );
                assert_eq!(
                    buffers.encoded.as_ptr() as usize,
                    origin.1,
                    "base64 allocation was not reused"
                );
                buffers.clear();
                assert!(buffers.bytes.is_empty() && buffers.encoded.is_empty());
                assert!(buffers.bytes.capacity() <= protocol::BUFFER_LIMIT);
                assert!(buffers.encoded.capacity() <= protocol::BUFFER_LIMIT.div_ceil(3) * 4);
            }
        }
        let buffers = &mut first.write_scratch;
        buffers.begin(protocol::BUFFER_LIMIT).unwrap();
        buffers.bytes.resize(protocol::BUFFER_LIMIT, 42);
        buffers.encode().unwrap();
        assert_eq!(buffers.encoded, STANDARD.encode(&buffers.bytes));
        assert!(buffers.bytes.capacity() <= protocol::BUFFER_LIMIT);
        assert!(buffers.encoded.capacity() <= protocol::BUFFER_LIMIT.div_ceil(3) * 4);
        assert_eq!(
            buffers
                .begin(protocol::BUFFER_LIMIT + 1)
                .unwrap_err()
                .to_string(),
            "iovec buffer capacity"
        );
        assert!(buffers.bytes.is_empty() && buffers.encoded.is_empty());
        buffers.bytes = Vec::with_capacity(protocol::BUFFER_LIMIT + 1);
        assert_eq!(
            buffers.encode().unwrap_err().to_string(),
            "iovec buffer capacity"
        );
        buffers.clear();
        assert!(buffers.bytes.capacity() <= protocol::BUFFER_LIMIT);
        buffers.encoded = String::with_capacity(protocol::BUFFER_LIMIT.div_ceil(3) * 4 + 1);
        assert_eq!(
            buffers.encode().unwrap_err().to_string(),
            "iovec base64 capacity"
        );
        buffers.clear();
        assert!(buffers.encoded.capacity() <= protocol::BUFFER_LIMIT.div_ceil(3) * 4);
    }
    #[test]
    fn typed_write_frame_uses_independent_state_scratch_and_reuses_it() {
        let mut first = clock_state(8, 65536, 1);
        let mut second = clock_state(8, 65536, 1);
        first.request_scratch = Vec::with_capacity(1024);
        second.request_scratch = Vec::with_capacity(1024);
        let origins = [
            first.request_scratch.as_ptr() as usize,
            second.request_scratch.as_ptr() as usize,
        ];
        assert_ne!(origins[0], origins[1]);
        for (state, origin) in [(&mut first, origins[0]), (&mut second, origins[1])] {
            let capture = OriginCapture::default();
            state.output = Box::new(capture.clone());
            state.model = None;
            state.input = Box::new(std::io::Cursor::new(b"{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{}}\n".to_vec()));
            state
                .callback_input(protocol::CallbackInput::Write {
                    fd: 13,
                    data_base64: "AP8=",
                })
                .unwrap();
            state
                .callback_input(protocol::CallbackInput::PWrite {
                    fd: 13,
                    data_base64: "AP8=",
                    offset: u64::MAX,
                })
                .unwrap();
            let frames = capture.0.lock().unwrap();
            assert_eq!(frames[0].1, b"{\"id\":1,\"input\":{\"data_base64\":\"AP8=\",\"fd\":13},\"op\":\"fd_write\",\"type\":\"call\"}\n");
            assert_eq!(frames[1].1, b"{\"id\":2,\"input\":{\"data_base64\":\"AP8=\",\"fd\":13,\"offset\":18446744073709551615},\"op\":\"fd_pwrite\",\"type\":\"call\"}\n");
            assert_eq!(
                frames[0].0, origin,
                "wire did not originate in owned State scratch"
            );
            assert_eq!(
                frames[1].0, origin,
                "sequential write did not reuse scratch"
            );
            assert_eq!(state.request_scratch.as_ptr() as usize, origin);
        }
    }

    #[test]
    fn typed_write_reference_keeps_literal_wire_and_three_field_request_state() {
        for (op, input, expected) in [
            (
                "fd_write",
                json!({"fd":u32::MAX,"data_base64":"AP8="}),
                "{\"id\":1,\"input\":{\"data_base64\":\"AP8=\",\"fd\":4294967295},\"local_state\":{\"advances\":0,\"now\":1,\"transcript_bytes\":0},\"op\":\"fd_write\",\"type\":\"call\"}\n",
            ),
            (
                "fd_pwrite",
                json!({"fd":13,"data_base64":"AP8=","offset":u64::MAX}),
                "{\"id\":1,\"input\":{\"data_base64\":\"AP8=\",\"fd\":13,\"offset\":18446744073709551615},\"local_state\":{\"advances\":0,\"now\":1,\"transcript_bytes\":0},\"op\":\"fd_pwrite\",\"type\":\"call\"}\n",
            ),
        ] {
            for typed in [false, true] {
                let mut state = clock_state(4, 65536, 1);
                let capture = WriteCapture::default();
                state.output = Box::new(capture.clone());
                state.input = Box::new(std::io::Cursor::new(
                b"{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":2},\"local_state\":{\"now\":7,\"advances\":3},\"transcript_base64\":\"AQI=\"}\n".to_vec(),
            ));
                let reply = if typed {
                    let data_base64 = input["data_base64"].as_str().unwrap();
                    let fd = input["fd"].as_u64().unwrap() as u32;
                    let callback = if op == "fd_pwrite" {
                        protocol::CallbackInput::PWrite {
                            fd,
                            data_base64,
                            offset: input["offset"].as_u64().unwrap(),
                        }
                    } else {
                        protocol::CallbackInput::Write { fd, data_base64 }
                    };
                    state.callback_input(callback).unwrap()
                } else {
                    state.callback(op, input.clone()).unwrap()
                };
                assert_eq!(capture.0.lock().unwrap().as_slice(), expected.as_bytes());
                assert_eq!(reply.output, json!({"written":2}));
                assert_eq!((state.calls, state.completed_calls), (1, 1));
                assert_eq!(
                    state.model.as_ref().unwrap().state(),
                    json!({"advances":3,"now":7,"transcript_bytes":2})
                );
                assert!(state.terminal.is_none());
            }
        }
    }

    #[test]
    fn typed_write_serialization_matches_value_counterfactual_and_bounds_retained_capacity() {
        let scattered = [vec![0; 32], vec![255; 24], vec![42; 4096]].concat();
        let payloads = [
            String::new(),
            "AP8=".to_owned(),
            "\"\\\n\0é".to_owned(),
            STANDARD.encode([0; 32]),
            STANDARD.encode([255; 24]),
            STANDARD.encode([42; 4096]),
            STANDARD.encode(scattered),
            STANDARD.encode(vec![42; protocol::BUFFER_LIMIT]),
        ];
        for model in [false, true] {
            for data in payloads.iter().map(String::as_str) {
                for offset in [None, Some(0), Some(u64::MAX)] {
                    let input = offset.map_or(
                        protocol::CallbackInput::Write {
                            fd: u32::MAX,
                            data_base64: data,
                        },
                        |offset| protocol::CallbackInput::PWrite {
                            fd: u32::MAX,
                            data_base64: data,
                            offset,
                        },
                    );
                    let value = offset.map_or_else(
                        || json!({"fd":u32::MAX,"data_base64":data}),
                        |offset| json!({"fd":u32::MAX,"data_base64":data,"offset":offset}),
                    );
                    let mut state = clock_state(4, 65536, 1);
                    let mut expected =
                        json!({"type":"call","id":u64::MAX,"op":input.op(),"input":value});
                    if model {
                        expected["local_state"] = state.model.as_ref().unwrap().state();
                    }
                    let mut reference = Vec::new();
                    protocol::send_frame(&mut reference, &expected).unwrap();
                    let mut actual = Vec::new();
                    let mut scratch = Vec::new();
                    protocol::send_callback(
                        &mut actual,
                        &mut scratch,
                        u64::MAX,
                        &input,
                        model.then(|| state.model.as_mut().unwrap().request_state()),
                    )
                    .unwrap();
                    assert_eq!(actual, reference);
                    assert!(scratch.is_empty());
                    assert!(scratch.capacity() <= protocol::FRAME_LIMIT);
                }
            }
        }
        let data = "x".repeat(protocol::FRAME_LIMIT);
        let mut scratch = Vec::with_capacity(1024);
        let mut output = Vec::new();
        let error = protocol::send_callback(
            &mut output,
            &mut scratch,
            1,
            &protocol::CallbackInput::PWrite {
                fd: 13,
                data_base64: &data,
                offset: 0,
            },
            None,
        )
        .unwrap_err();
        assert_eq!(protocol::classification(&error), "capacity");
        assert!(output.is_empty());
        assert!(scratch.is_empty());
        assert!(scratch.capacity() <= protocol::FRAME_LIMIT);
        let mut oversized = Vec::with_capacity(protocol::FRAME_LIMIT + 1);
        protocol::send_callback(
            &mut output,
            &mut oversized,
            1,
            &protocol::CallbackInput::Write {
                fd: 13,
                data_base64: "",
            },
            None,
        )
        .unwrap();
        assert!(oversized.capacity() <= protocol::FRAME_LIMIT);
    }

    #[test]
    fn typed_write_shared_decoder_chunks_interrupts_and_transport_restore_scratch() {
        for (reply, termination, completed) in [
            (
                "{\"type\":\"remote_transcript\",\"data_base64\":\"AQI=\"}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":2},\"local_state\":{\"now\":7,\"advances\":3},\"transcript_base64\":\"\"}\n",
                None,
                1,
            ),
            (
                "{\"type\":\"interrupt\",\"termination\":\"stop\",\"message\":\"modeled\"}\n",
                Some("stop"),
                0,
            ),
            (
                "{\"type\":\"remote_transcript\",\"data_base64\":\"\"}\n",
                Some("invalid"),
                0,
            ),
            (
                "{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{},\"unsupported\":true,\"message\":\"outside profile\",\"local_state\":{\"now\":7,\"advances\":3},\"transcript_base64\":\"\"}\n",
                Some("unsupported"),
                1,
            ),
        ] {
            let mut state = clock_state(4, 65536, 1);
            state.request_scratch = Vec::with_capacity(1024);
            let origin = state.request_scratch.as_ptr();
            state.input = Box::new(std::io::Cursor::new(reply.as_bytes().to_vec()));
            let result = state.callback_input(protocol::CallbackInput::PWrite {
                fd: 13,
                data_base64: "AP8=",
                offset: u64::MAX,
            });
            assert_eq!(result.is_err(), termination.is_some());
            assert_eq!(
                state.terminal.as_ref().map(|outcome| outcome.termination),
                termination
            );
            assert_eq!((state.calls, state.completed_calls), (1, completed));
            assert_eq!(state.request_scratch.as_ptr(), origin);
            assert!(state.request_scratch.is_empty());
            if termination.is_none() {
                assert_eq!(
                    state.model.as_ref().unwrap().state(),
                    json!({"now":7,"advances":3,"transcript_bytes":2})
                );
            }
        }
        struct Broken;
        impl Write for Broken {
            fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
                Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe))
            }
            fn flush(&mut self) -> std::io::Result<()> {
                unreachable!()
            }
        }
        let mut state = clock_state(4, 65536, 1);
        state.request_scratch = Vec::with_capacity(1024);
        let origin = state.request_scratch.as_ptr();
        state.output = Box::new(Broken);
        assert!(
            state
                .callback_input(protocol::CallbackInput::Write {
                    fd: 13,
                    data_base64: ""
                })
                .is_err()
        );
        assert_eq!(
            state.terminal.as_ref().unwrap().termination,
            "infrastructure"
        );
        assert_eq!(state.request_scratch.as_ptr(), origin);
        assert!(state.request_scratch.is_empty());
        assert_eq!((state.calls, state.completed_calls), (1, 0));
    }

    #[test]
    fn typed_write_reference_capacity_and_decoder_fail_before_owner_merge() {
        for (reply, expected) in [
            (
                "{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{}}\n",
                "invalid callback reply type, ID or output",
            ),
            (
                "{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n",
                "missing remote model state",
            ),
            (
                "{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{},\"local_state\":{\"now\":0,\"advances\":0},\"transcript_base64\":\"\"}\n",
                "invalid remote clock state",
            ),
        ] {
            let mut state = clock_state(4, 65536, 1);
            state.input = Box::new(std::io::Cursor::new(reply.as_bytes().to_vec()));
            assert!(
                state
                    .callback("fd_pwrite", json!({"fd":13,"data_base64":"","offset":0}))
                    .is_err()
            );
            assert_eq!((state.calls, state.completed_calls), (1, 0));
            assert_eq!(
                state.model.as_ref().unwrap().state(),
                json!({"advances":0,"now":1,"transcript_bytes":0})
            );
            let outcome = state.terminal.as_ref().unwrap();
            assert_eq!(outcome.termination, "invalid");
            assert_eq!(outcome.message, expected);
        }
        let mut state = clock_state(1, 65536, 1);
        state.calls = 1;
        let capture = WriteCapture::default();
        state.output = Box::new(capture.clone());
        assert!(
            state
                .callback("fd_pwrite", json!({"fd":13,"data_base64":"","offset":0}))
                .is_err()
        );
        assert_eq!((state.calls, state.completed_calls), (1, 0));
        assert!(capture.0.lock().unwrap().is_empty());
        assert_eq!(
            state.terminal.as_ref().unwrap().message,
            "WASI capacity: callbacks"
        );

        let mut state = clock_state(4, 65536, 1);
        let capture = WriteCapture::default();
        state.output = Box::new(capture.clone());
        assert!(
            state
                .callback(
                    "fd_pwrite",
                    json!({"fd":13,"data_base64":"x".repeat(protocol::FRAME_LIMIT),"offset":0})
                )
                .is_err()
        );
        assert_eq!((state.calls, state.completed_calls), (1, 0));
        assert!(
            capture.0.lock().unwrap().is_empty(),
            "invalid oversized frame partially reached owner"
        );
        assert_eq!(state.terminal.as_ref().unwrap().termination, "capacity");
    }
    fn clock_state(calls: u64, transcript_bytes: u64, now: u64) -> State {
        let config: crate::model::Config = serde_json::from_value(json!({"seed":7,"mode":1,"limit":4096,"diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":3,"now":now,"limits":{"calls":calls,"transcript_bytes":transcript_bytes,"pending_events":16}})).unwrap();
        config.validate().unwrap();
        State {
            memory: None,
            input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
            output: Box::new(Vec::<u8>::new()),
            limits: Limits {
                max: 65536,
                peak: 65536,
                failure: None,
            },
            calls: 0,
            completed_calls: 0,
            output_limit: 1024,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: Some(crate::model::Model::new(config)),
            namespace: None,
            request_scratch: Vec::new(),
            write_scratch: protocol::WriteBuffers::default(),
        }
    }
    #[test]
    fn typed_clock_state_preserves_global_ids_and_remote_clock_updates() {
        let mut state = clock_state(100, 65536, 1);
        state.runtime("runtime_config", vec![0; 16]).unwrap();
        assert_eq!(state.clock_time_get(0, u64::MAX).unwrap(), 1001);
        assert_eq!((state.calls, state.completed_calls), (2, 2));
        let canonical = b"{\"type\":\"call\",\"id\":3,\"op\":\"fd_sync\",\"input\":{\"fd\":3}}\n{\"type\":\"reply\",\"id\":3,\"errno\":0,\"output\":{}}\n";
        let reply = json!({"type":"reply","id":3,"errno":0,"output":{},"local_state":{"now":20,"advances":3},"transcript_base64":STANDARD.encode(canonical)});
        state.input = Box::new(std::io::Cursor::new(format!("{reply}\n").into_bytes()));
        state.callback("fd_sync", json!({"fd":3})).unwrap();
        assert_eq!(state.clock_time_get(1, 0).unwrap(), 20);
        assert_eq!((state.calls, state.completed_calls), (4, 4));
        assert_eq!(
            (
                state.model.as_ref().unwrap().now,
                state.model.as_ref().unwrap().advances
            ),
            (23, 3)
        );
        assert!(state.clock_time_get(2, 0).is_err());
        assert_eq!((state.calls, state.completed_calls), (5, 5));
        let outcome = state.terminal.as_ref().unwrap();
        assert_eq!(outcome.termination, "unsupported");
        assert_eq!(
            outcome.message,
            "clock_time_get: CPU clocks outside stock WASI profile"
        );
        assert_eq!(state.model.as_ref().unwrap().now, 23);
    }
    #[test]
    fn remote_transcript_respects_transcript_limit_with_capacity() {
        let canonical = b"{\"type\":\"call\",\"id\":1,\"op\":\"fd_sync\",\"input\":{\"fd\":3}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n";
        for (limit, accepted) in [
            (canonical.len() as u64, true),
            (canonical.len() as u64 - 1, false),
        ] {
            let mut state = clock_state(100, limit, 1);
            let reply = json!({"type":"reply","id":1,"errno":0,"output":{},"local_state":{"now":20,"advances":3},"transcript_base64":STANDARD.encode(canonical)});
            state.input = Box::new(std::io::Cursor::new(format!("{reply}\n").into_bytes()));
            let result = state.callback("fd_sync", json!({"fd":3}));
            let model = state.model.as_ref().unwrap();
            if accepted {
                assert!(result.is_ok());
                assert_eq!(model.transcript_bytes, limit);
                continue;
            }
            assert!(result.is_err());
            let outcome = state.terminal.as_ref().unwrap();
            assert_eq!(
                (outcome.termination, outcome.message.as_str()),
                ("capacity", "WASI capacity: transcript-bytes")
            );
            // Rejected before any model mutation.
            assert_eq!(
                (model.transcript_bytes, model.now, model.advances),
                (0, 1, 0)
            );
            assert_eq!((state.calls, state.completed_calls), (1, 0));
        }
    }
    #[test]
    fn typed_clock_state_capacity_preserves_attempted_and_completed_counts() {
        for (calls, transcript, now, prime, message) in [
            (1, 65536, 1, true, "WASI capacity: callbacks"),
            (100, 1, 1, false, "WASI capacity: transcript-bytes"),
            (
                100,
                65536,
                i64::MAX as u64 - 1000,
                false,
                "WASI capacity: clock-nanoseconds",
            ),
        ] {
            let mut state = clock_state(calls, transcript, now);
            if prime {
                state.clock_time_get(1, 0).unwrap();
            }
            let before = state.model.as_ref().unwrap().now;
            assert!(state.clock_time_get(1, 0).is_err());
            assert_eq!(state.calls, 1);
            assert_eq!(state.completed_calls, u64::from(prime));
            assert_eq!(state.model.as_ref().unwrap().now, before);
            let outcome = state.terminal.as_ref().unwrap();
            assert_eq!(outcome.termination, "capacity");
            assert_eq!(outcome.message, message);
        }
    }
    #[test]
    fn typed_clock_streams_bounded_chunks_before_final_flush_matching_json_reference() {
        use std::sync::{Arc, Mutex};
        #[derive(Clone)]
        struct Capture(Arc<Mutex<Vec<u8>>>);
        impl Write for Capture {
            fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
                self.0.lock().unwrap().extend_from_slice(bytes);
                Ok(bytes.len())
            }
            fn flush(&mut self) -> std::io::Result<()> {
                Ok(())
            }
        }
        let mut reference = None;
        for typed in [false, true] {
            let mut state = clock_state(8192, 1 << 20, 1);
            let capture = Capture(Arc::new(Mutex::new(Vec::new())));
            state.output = Box::new(capture.clone());
            for index in 0..4096 {
                let clock_id = index % 2;
                let precision = index as u64 * 17;
                let timestamp = if typed {
                    state.clock_time_get(clock_id, precision).unwrap()
                } else {
                    state
                        .callback(
                            "clock_time_get",
                            json!({"clock_id":clock_id,"precision":precision}),
                        )
                        .unwrap()
                        .output["timestamp"]
                        .as_u64()
                        .unwrap()
                };
                assert_eq!(
                    timestamp,
                    index as u64 * 3 + 1 + if clock_id == 0 { 1000 } else { 0 }
                );
                if index == 2048 {
                    assert!(
                        !capture.0.lock().unwrap().is_empty(),
                        "clock evidence must stream before termination"
                    );
                }
            }
            let model = state.model.as_mut().unwrap();
            model.flush(&mut state.output, true).unwrap();
            let metadata = model.metadata(state.completed_calls);
            let bytes = capture.0.lock().unwrap().clone();
            let mut frames = 0;
            for frame in bytes
                .split(|byte| *byte == b'\n')
                .filter(|frame| !frame.is_empty())
            {
                let frame: Value = serde_json::from_slice(frame).unwrap();
                assert!(
                    STANDARD
                        .decode(frame["data_base64"].as_str().unwrap())
                        .unwrap()
                        .len()
                        <= 64 * 1024
                );
                frames += 1;
            }
            assert!(frames >= 3);
            if let Some((expected_bytes, expected_metadata)) = &reference {
                assert_eq!(&bytes, expected_bytes);
                assert_eq!(&metadata, expected_metadata);
            } else {
                reference = Some((bytes, metadata));
            }
        }
    }
    #[test]
    fn untyped_execution_io_failure_is_infrastructure() {
        let error = std::fs::File::open("/missing-gomad3-wasmhost-backing").unwrap_err();
        assert_eq!(
            execution_failure(Error::new(error), None).termination,
            "infrastructure"
        );
    }
    #[test]
    fn broken_callback_transport_is_infrastructure() {
        let (output, peer) = std::os::unix::net::UnixStream::pair().unwrap();
        drop(peer);
        let mut state = State {
            memory: None,
            input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
            output: Box::new(output),
            limits: Limits {
                max: 65536,
                peak: 0,
                failure: None,
            },
            calls: 0,
            completed_calls: 0,
            output_limit: 1024,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: None,
            namespace: None,
            request_scratch: Vec::new(),
            write_scratch: protocol::WriteBuffers::default(),
        };
        assert!(state.callback("sched_yield", json!({})).is_err());
        assert_eq!(state.terminal.unwrap().termination, "infrastructure");
    }
}
