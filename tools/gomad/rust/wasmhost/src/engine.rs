use crate::{
    abi,
    protocol::{self, Authorization, Chunk, Continue, Execute, Outcome, Reply},
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
    pub host_nanos: Option<u64>,
    pub stdout_progress: bool,
    pub output_limit: u64,
    pub output_bytes: u64,
    pub terminal: Option<Outcome>,
    pub argument_sizes: Option<(u32, u32)>,
    pub environment_sizes: Option<(u32, u32)>,
    pub model: Option<crate::model::Model>,
    pub dormant_debug: Option<crate::model::inspection::Transport>,
    pub namespace: Option<crate::namespace::Namespace>,
    pub request_scratch: Vec<u8>,
    pub write_scratch: protocol::WriteBuffers,
    /// A cluster instance's seat in the in-process broker (fn-178.7), with its
    /// endpoint on the shared Rust network owner (fn-178.12).
    pub broker: Option<crate::cluster::handoff::Member>,
    pub(crate) network: Option<crate::cluster::network::Endpoint>,
}
/// Where an inspection snapshot is taken: it decides which commitments must
/// be current when the call returns.
#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Snapshot {
    /// Before a debug decision: the instance's own boundary consumes the cut
    /// and its World commitment.
    Boundary,
    /// Before a broker call: the network commitment is deferred, since the
    /// call's own network write supersedes it before any handoff (or the
    /// call fails and the instance's terminal snapshot does).
    Call,
    /// At termination: every commitment is current for later boundaries.
    Terminal,
}

impl State {
    #[expect(
        clippy::too_many_lines,
        reason = "The snapshot keeps storage, network and World commitments at one ordered boundary."
    )]
    pub fn inspection_snapshot(&mut self, at: Snapshot) {
        use crate::model::inspection::{NetworkState, StorageEvidence, StorageState};
        #[cfg(test)]
        if crate::model::inspection::eager_oracle() {
            return self.inspection_snapshot_eager(at);
        }
        let Some(model) = self.model.as_mut() else {
            return;
        };
        let recorder = model.inspection.lock();
        if model.inspection.sealed
            || !model.inspection.started
            || !recorder.observed
            || recorder.exhausted
        {
            return;
        }
        drop(recorder);
        let mut cut = model.inspection_cut();
        let mut storage = Vec::new();
        let mut revision = None;
        let mut unchanged = None;
        if let Some(namespace) = &self.namespace {
            let current = namespace.inspection_revision();
            revision = Some(current);
            match &model.inspection.storage_evidence {
                // An unchanged namespace yields identical material, whose
                // encoding already succeeded.
                Some(evidence) if evidence.revision == current && evidence.handle == cut.handle => {
                    unchanged = Some(evidence.clone());
                }
                _ => {
                    if let Ok((state, recovery, log)) = namespace.inspection_material() {
                        cut.storage_state = state;
                        cut.recovery_state = recovery;
                        storage = log;
                    } else {
                        model.inspection.lock().exhausted = true;
                        return;
                    }
                }
            }
        }
        let mut network = None;
        if let Some(endpoint) = &mut self.network {
            if let Ok((digest, _)) =
                endpoint.inspection_network(crate::model::inspection::network_commitment)
            {
                network = Some(Some(digest));
            } else {
                model.inspection.lock().exhausted = true;
                return;
            }
        }
        if let Some(member) = &mut self.broker
            && let Some(endpoint) = &mut member.network
        {
            let observed = if at == Snapshot::Call {
                endpoint.evidence_check().map(|()| (None, None))
            } else if at == Snapshot::Terminal {
                endpoint.evidence().map(|bytes| {
                    (
                        Some(crate::model::inspection::network_commitment(&bytes)),
                        None,
                    )
                })
            } else {
                endpoint
                    .inspection_network(crate::model::inspection::network_commitment)
                    .map(|(digest, bytes)| (Some(digest), bytes))
            };
            if let Ok((digest, _)) = observed {
                network = Some(digest);
            } else {
                model.inspection.lock().exhausted = true;
                return;
            }
        }
        let network_state = match network {
            None => NetworkState::Bytes(&[]),
            Some(None) => NetworkState::Pending,
            Some(Some(digest)) => NetworkState::Digest(digest),
        };
        let storage_state = match unchanged {
            Some(evidence) => StorageState::Unchanged {
                storage: evidence.storage,
                recovery: evidence.recovery,
                log_bytes: evidence.log_bytes,
            },
            None => StorageState::Fresh { log: &storage },
        };
        let Some(digests) =
            model
                .inspection
                .lock()
                .snapshot_with(cut, network_state, storage_state)
        else {
            return;
        };
        if let Some(revision) = revision {
            model.inspection.storage_evidence = Some(StorageEvidence {
                handle: model.inspection.handle.clone(),
                revision,
                storage: digests.storage,
                recovery: digests.recovery,
                log_bytes: digests.log_bytes,
            });
        }
        if at == Snapshot::Boundary {
            let commitment = match &self.broker {
                Some(member) => {
                    let identity = member.network.as_ref().map(|_| digests.network.unwrap());
                    let world = member.broker.inspection_world_commitment(identity);
                    if let Ok(digest) = world {
                        digest.to_vec()
                    } else {
                        model.inspection.lock().exhausted = true;
                        return;
                    }
                }
                None => crate::model::inspection::commitment("world", &[]),
            };
            model.inspection.world_commitment = commitment;
        }
    }
    /// Test oracle: the original eager snapshot, which re-encodes and
    /// re-hashes every input on every call, with an uncached World wire.
    #[cfg(test)]
    fn inspection_snapshot_eager(&mut self, at: Snapshot) {
        let Some(model) = self.model.as_mut() else {
            return;
        };
        let recorder = model.inspection.lock();
        if model.inspection.sealed
            || !model.inspection.started
            || !recorder.observed
            || recorder.exhausted
        {
            return;
        }
        drop(recorder);
        let mut cut = model.inspection_cut();
        let mut storage = Vec::new();
        if let Some(namespace) = &self.namespace {
            if let Ok((state, recovery, log)) = namespace.inspection_material() {
                cut.storage_state = state;
                cut.recovery_state = recovery;
                storage = log;
            } else {
                model.inspection.lock().exhausted = true;
                return;
            }
        }
        let mut network = Vec::new();
        if let Some(endpoint) = &self.network {
            if let Ok(bytes) = endpoint.evidence() {
                network = bytes;
            } else {
                model.inspection.lock().exhausted = true;
                return;
            }
        }
        if let Some(member) = &self.broker
            && let Some(endpoint) = &member.network
        {
            let evidence = if at == Snapshot::Terminal {
                endpoint.evidence()
            } else {
                endpoint.inspection_network_eager()
            };
            if let Ok(bytes) = evidence {
                network = bytes;
            } else {
                model.inspection.lock().exhausted = true;
                return;
            }
        }
        if model
            .inspection
            .lock()
            .snapshot(cut, &network, &storage)
            .is_none()
        {
            return;
        }
        if at == Snapshot::Boundary {
            model.inspection.world_commitment = match &self.broker {
                Some(member) => {
                    if let Ok(commitment) = member.broker.inspection_world_commitment(
                        member
                            .network
                            .as_ref()
                            .map(|_| crate::model::inspection::network_commitment(&network)),
                    ) {
                        commitment.to_vec()
                    } else {
                        model.inspection.lock().exhausted = true;
                        return;
                    }
                }
                None => crate::model::inspection::commitment("world", &[]),
            };
        }
    }
    pub fn debug_exchange(&mut self, bytes: &mut [u8]) -> Result<()> {
        let operation = match crate::model::inspection::validate_frame(bytes) {
            Ok(operation) => operation,
            Err(error) => return Err(self.fail("debug_error", error.to_string())),
        };
        if let Some(model) = &self.model {
            model.inspection.lock().observe(&model.inspection.handle);
        }
        if operation == 1 {
            self.inspection_snapshot(Snapshot::Boundary);
        }
        let Some(model) = self.model.as_mut() else {
            return match self
                .dormant_debug
                .get_or_insert_with(Default::default)
                .dormant_exchange(bytes)
            {
                Ok(()) => Ok(()),
                Err(error) => Err(self.fail("debug_error", error.to_string())),
            };
        };
        match model.debug_exchange(bytes) {
            Ok(()) => {
                if operation == 6 {
                    if let Some(member) = &self.broker {
                        member.broker.abort("inspected_prefix", String::new());
                    }
                    return Err(self.fail("inspected_prefix", "verified inspected prefix"));
                }
                Ok(())
            }
            Err(error) => {
                let kind = if model.divergence.is_some() {
                    "divergence"
                } else if error.to_string().starts_with("instrumentation divergence:") {
                    "instrumentation_divergence"
                } else {
                    "debug_error"
                };
                if let Some(member) = &self.broker {
                    member.broker.abort(kind, error.to_string());
                }
                Err(self.fail(kind, error.to_string()))
            }
        }
    }
    /// Serves a cluster instance's `runtime_idle` or `net_stream` through the
    /// in-process broker, counted and admitted like any locally handled call.
    pub fn broker_call(&mut self, op: abi::Operation, mut data: Vec<u8>) -> Result<Vec<u8>> {
        use crate::cluster::network::Failure;
        self.inspection_snapshot(Snapshot::Call);
        let mut member = self.broker.take().unwrap();
        let served = self.model_operation(|model, _, _| {
            let runtime_active = model.runtime_active();
            let mut clock = crate::cluster::handoff::Clockwork {
                now: &mut model.now,
                advances: &mut model.advances,
                runtime_active,
            };
            let result = if op == abi::Operation::RuntimeIdle {
                member.idle(&mut data, &mut clock)
            } else {
                member.net_stream(&mut data, &mut clock)
            };
            let boundary =
                |kind, dimension| Err(Error::new(crate::namespace::Boundary { kind, dimension }));
            match result {
                Ok(()) => Ok(None),
                Err(Failure::Invalid(dimension)) => boundary("invalid", dimension),
                Err(Failure::Capacity(dimension)) => boundary("capacity", dimension),
                Err(Failure::Boundary(kind, dimension)) => boundary(kind, dimension),
                // Classified capacity by its `WASI capacity:` text, as a
                // boundary is, keeping the limit accounting in the message.
                Err(exhausted @ Failure::Exhausted { .. }) => Err(Error::msg(exhausted.message())),
                Err(Failure::Infrastructure(message)) => Ok(Some(message)),
            }
        });
        self.broker = Some(member);
        match served? {
            None => Ok(data),
            Some(message) => Err(self.fail("infrastructure", message)),
        }
    }
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
    pub(crate) fn model_operation<T>(
        &mut self,
        body: impl FnOnce(
            &mut crate::model::Model,
            Option<&mut crate::namespace::Namespace>,
            u64,
        ) -> Result<T>,
    ) -> Result<T> {
        if self.host_nanos.is_none() {
            return self.model_operation_inner(body);
        }
        let phase = Instant::now();
        let result = self.model_operation_inner(body);
        *self.host_nanos.as_mut().unwrap() += crate::metrics::elapsed(phase);
        result
    }
    fn model_operation_inner<T>(
        &mut self,
        body: impl FnOnce(
            &mut crate::model::Model,
            Option<&mut crate::namespace::Namespace>,
            u64,
        ) -> Result<T>,
    ) -> Result<T> {
        self.check_broker()?;
        self.sync_revocation()?;
        if self.calls >= self.model.as_ref().unwrap().call_limit() {
            return Err(self.fail("capacity", "WASI capacity: callbacks"));
        }
        self.calls += 1;
        let model = self.model.as_mut().unwrap();
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(model.event_context());
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if !model.inspection.started {
            crate::event_order::emit(
                &model.event_context(),
                "model-transition",
                0,
                serde_json::json!({"model":"lifecycle","event":"start"}),
            );
        }
        model.inspection.started = true;
        model
            .inspection
            .lock()
            .event("model_effect", &model.inspection.handle);
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
                    } else if error.downcast_ref::<crate::namespace::Stop>().is_some() {
                        protocol::classification(&error)
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
    fn check_broker(&mut self) -> Result<()> {
        let checked = self
            .broker
            .as_ref()
            .map(|member| member.broker.check(member.index, member.incarnation));
        if let Some(Err(error)) = checked {
            return Err(self.fail(error.termination(), error.message()));
        }
        Ok(())
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
        self.model_call(super::model::Model::yield_call)
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
    /// Split ownership: at the first local call and after a revocable remote
    /// reply, confirms with Go that no broker revocation crossed it and
    /// adopts the broker clock, as the Go-owned path does at its next call.
    fn sync_revocation(&mut self) -> Result<()> {
        if !self.model.as_ref().is_some_and(|model| model.sync_pending) {
            return Ok(());
        }
        #[cfg(feature = "diagnostic-profile")]
        let _remote = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Remote);
        let result = (|| -> Result<()> {
            let model = self.model.as_mut().unwrap();
            model.flush_output(&mut self.output)?;
            protocol::send_frame(
                &mut self.output,
                &json!({"type":"sync","id":self.completed_calls}),
            )?;
            let bytes = protocol::read_frame(&mut self.input)?;
            let frame: Value = serde_json::from_slice(&bytes).map_err(Error::new)?;
            if frame["type"] == "interrupt" {
                let interrupt: protocol::Interrupt = protocol::decode_boundary(&bytes)?;
                interrupt.validate()?;
                return Err(Error::new(interrupt));
            }
            if frame["type"] != "continue" {
                return Err(Error::new(protocol::Unsupported(format!(
                    "unknown sync reply kind {}",
                    frame["type"]
                ))));
            }
            let resumed: Continue = protocol::decode_boundary(&bytes)?;
            let model = self.model.as_mut().unwrap();
            if resumed.kind != "continue" || !model.adopt_broker_clock(resumed.now) {
                return Err(Error::msg("invalid revocation sync reply"));
            }
            model.sync_pending = false;
            Ok(())
        })();
        result.map_err(|error| self.fail(protocol::classification(&error), error.to_string()))
    }
    pub fn namespace_call(
        &mut self,
        call: crate::namespace::call::Call,
    ) -> Result<crate::namespace::Reply> {
        let name = call.name;
        if matches!(
            call.operation,
            crate::namespace::Operation::Write { fd: 1 | 2, .. }
        ) && let Some(member) = &mut self.broker
            && let Err(error) = member.begin_output(&mut self.model.as_mut().unwrap().now)
        {
            return Err(self.fail(error.termination(), error.message()));
        }
        if self.model.as_ref().is_some_and(crate::model::Model::split) {
            self.namespace
                .as_mut()
                .unwrap()
                .adopt_offered_output(self.output_bytes);
        }
        let mut stdout = Vec::new();
        let result = self.model_operation(|model, namespace, id| {
            let namespace = namespace.unwrap();
            let result = model.namespace_call(id, namespace, call);
            let (out, stderr) = namespace.take_output();
            model.namespace_output(&out, &stderr);
            stdout = out;
            result
        });
        self.output_bytes = self.namespace.as_ref().unwrap().offered_output();
        let reply = result?;
        if self.stdout_progress && !stdout.is_empty() {
            protocol::send_frame(
                &mut self.output,
                &json!({"type":"progress","stream":"stdout","data_base64":STANDARD.encode(&stdout)}),
            )?;
        }
        if let Some(member) = &self.broker
            && let Err(error) = member.broker.output(member.index, &stdout)
        {
            return Err(self.fail(error.termination(), error.message()));
        }
        if let Some(message) = &reply.unsupported {
            return Err(self.fail("unsupported", format!("{name}: {message}")));
        }
        Ok(reply)
    }
    #[cfg(test)]
    pub fn runtime(&mut self, op: &str, bytes: Vec<u8>) -> Result<Vec<u8>> {
        self.model_call(|model, _| model.runtime(op, bytes))
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Callback admission, execution and terminal classification remain in wire order."
    )]
    #[expect(
        clippy::needless_pass_by_value,
        reason = "The callback descriptor is a short-lived request view consumed at this boundary."
    )]
    pub fn callback_input(&mut self, input: protocol::CallbackInput<'_>) -> Result<Reply> {
        self.check_broker()?;
        let op = input.op();
        if self.calls
            >= self
                .model
                .as_ref()
                .map_or(MAX_CALLBACKS, super::model::Model::call_limit)
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
            let mut local_state = self.model.as_mut().map(crate::model::Model::request_state);
            if self.model.as_ref().is_some_and(crate::model::Model::split) {
                local_state.as_mut().unwrap().offered_output_bytes =
                    Some(self.namespace.as_ref().unwrap().offered_output());
            }
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
                    let interrupt: protocol::Interrupt = protocol::decode_boundary(&bytes)?;
                    interrupt.validate()?;
                    match (&interrupt.completed, &interrupt.transcript_base64) {
                        (None, None) => {}
                        (None, Some(_)) => {
                            return Err(Error::msg("interrupt record without completed call"));
                        }
                        (Some(completed), encoded) => {
                            // Go completed this call before the broker revoked
                            // the incarnation; the Go-owned path counts it.
                            let model = self
                                .model
                                .as_mut()
                                .filter(|model| model.split())
                                .ok_or_else(|| {
                                    Error::msg("completed interrupt outside split ownership")
                                })?;
                            // An empty inline record means it arrived as chunks.
                            let record = match encoded.as_deref() {
                                None if transcript.is_empty() => Vec::new(),
                                Some("") => std::mem::take(&mut transcript),
                                Some(encoded) if transcript.is_empty() => {
                                    STANDARD.decode(encoded).map_err(Error::new)?
                                }
                                _ => {
                                    return Err(Error::msg(
                                        "interrupt record cannot combine chunks and inline bytes",
                                    ));
                                }
                            };
                            if encoded.is_some() && record.is_empty() {
                                return Err(Error::msg("empty interrupt record"));
                            }
                            model.interrupted_call(completed.now, completed.advances, &record)?;
                            self.completed_calls = self.calls;
                        }
                    }
                    return Err(Error::new(interrupt));
                }
                if self.model.is_some() && frame["type"] == "remote_transcript" {
                    let chunk: Chunk = protocol::decode_boundary(&bytes)?;
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
                if frame["type"] != "reply" {
                    return Err(Error::new(protocol::Unsupported(format!(
                        "unknown callback reply kind {}",
                        frame["type"]
                    ))));
                }
                break protocol::decode_boundary(&bytes)?;
            };
            #[cfg(feature = "stress-jitter")]
            crate::stress_jitter::delay();
            if reply.kind != "reply" {
                return Err(Error::new(protocol::Unsupported(format!(
                    "unknown callback reply kind {}",
                    reply.kind
                ))));
            }
            if reply.id != self.calls || !reply.output.is_object() {
                return Err(Error::msg("invalid callback reply type, ID or output"));
            }
            if let Some(model) = &mut self.model {
                model.remote(&reply, &transcript)?;
                if reply.revocable && !model.split() {
                    return Err(Error::msg("revocable reply outside split ownership"));
                }
                model.sync_pending = reply.revocable;
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
    #[expect(
        clippy::needless_pass_by_value,
        reason = "The ABI transfers ownership of its constructed JSON request to this callback."
    )]
    pub fn callback_typed(&mut self, operation: abi::Operation, input: Value) -> Result<Reply> {
        self.callback_input(protocol::CallbackInput::Typed {
            operation,
            input: &input,
        })
    }
    #[cfg(test)]
    #[expect(
        clippy::needless_pass_by_value,
        reason = "The fixture transfers its constructed request into the callback boundary."
    )]
    pub fn callback(&mut self, op: &str, input: Value) -> Result<Reply> {
        self.callback_input(protocol::CallbackInput::Json { op, input: &input })
    }
}
#[cfg(feature = "stress-jitter")]
pub(crate) const CONFIGURATION_REVISION: &str =
    "cranelift-fuel-nan-canonical-nounwind-v2-stress-jitter";
#[cfg(feature = "determinism-fixture")]
pub(crate) const CONFIGURATION_REVISION: &str =
    "cranelift-fuel-nan-canonical-nounwind-v2-determinism-fixture";
#[cfg(not(any(feature = "stress-jitter", feature = "determinism-fixture")))]
pub(crate) const CONFIGURATION_REVISION: &str = match (
    cfg!(feature = "guest-profile"),
    cfg!(feature = "diagnostic-fuel-off"),
) {
    (false, false) => "cranelift-fuel-nan-canonical-nounwind-v2",
    (true, false) => "cranelift-fuel-nan-canonical-nounwind-v2-guest-profile",
    (false, true) => "cranelift-fuel-nan-canonical-nounwind-v2-fuel-off",
    (true, true) => "cranelift-fuel-nan-canonical-nounwind-v2-guest-profile-fuel-off",
};

#[expect(
    clippy::too_many_lines,
    reason = "The entry point preserves admission, store lifetime, terminal evidence and cleanup order."
)]
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
        let mut request: Execute = protocol::decode_boundary(&first?)?;
        request.validate()?;
        if let Some(model) = &mut request.local_model {
            model.read_inputs(&mut input)?;
        }
        Ok(request)
    })();
    let mut request = match parsed {
        Ok(request) => request,
        Err(error) => {
            return protocol::send_frame(&mut output, &Outcome::refusal(&error));
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
        let mut outcome = Outcome::refusal(&error);
        metrics.sample_process();
        outcome.metrics = Some(Box::new(metrics));
        return protocol::send_frame(&mut output, &outcome);
    }
    if let Some(model) = &mut request.local_model
        && let Err(error) = model.read_tape(&mut *input)
    {
        let mut outcome = Outcome::refusal(&error);
        metrics.sample_process();
        outcome.metrics = Some(Box::new(metrics));
        return protocol::send_frame(&mut output, &outcome);
    }
    let namespace = match &request.namespace {
        Some(descriptor) => {
            let model = request.local_model.as_ref().unwrap();
            match crate::namespace::transport::read(
                &mut *input,
                descriptor,
                request.output_bytes,
                model.epoch_nanos,
                &model.working_directory_sha256,
            ) {
                Ok(mut namespace) => {
                    let attached = (|| -> Result<()> {
                        if let Some(storage) = &mut request.storage {
                            let maximum = crate::storage_transport::StorageDescriptor::image_limit(
                                namespace.filesystem_bytes(),
                            );
                            let image = storage.read_image(&mut *input, maximum)?;
                            storage.read_context(&mut *input, maximum.saturating_mul(18))?;
                            storage
                                .attach(
                                    &mut namespace,
                                    image.as_deref(),
                                    model.limits.transcript_bytes,
                                )
                                .map_err(crate::namespace::Stop::into_error)?;
                        }
                        Ok(())
                    })();
                    match attached {
                        Ok(()) => Some(namespace),
                        Err(error) => {
                            let mut outcome =
                                Outcome::new(protocol::classification(&error), error.to_string());
                            let model =
                                crate::model::Model::new(request.local_model.take().unwrap());
                            outcome.local_model = Some(Box::new(model.metadata(0)));
                            metrics.sample_process();
                            outcome.metrics = Some(Box::new(metrics));
                            return protocol::send_frame(&mut output, &outcome);
                        }
                    }
                }
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
    finish_storage(store.data_mut(), request.storage.as_ref(), &mut outcome);
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
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    let namespace = {
        let mut namespace = namespace;
        if let (Some(model), Some(namespace)) = (&model, &mut namespace) {
            let _scope = if crate::event_order::has_context() {
                None
            } else {
                Some(crate::event_order::enter(model.event_context()))
            };
            namespace.diagnostic_state(Some(model.now));
        }
        namespace
    };
    State {
        memory: None,
        input,
        output,
        limits: Limits {
            max: usize::try_from(memory_bytes).unwrap(),
            peak: 0,
            failure: None,
        },
        calls: 0,
        completed_calls: 0,
        host_nanos: None,
        stdout_progress: false,
        output_limit: output_bytes,
        output_bytes: 0,
        terminal: None,
        argument_sizes: None,
        environment_sizes: None,
        model,
        dormant_debug: None,
        namespace,
        request_scratch: Vec::new(),
        write_scratch: protocol::WriteBuffers::default(),
        broker: None,
        network: None,
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
        #[cfg(feature = "determinism-fixture")]
        if crate::linked_fixture::signature(import.module(), import.name()).is_some() {
            crate::linked_fixture::link(&mut linker)?;
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
#[cfg_attr(
    feature = "diagnostic-fuel-off",
    expect(
        clippy::unnecessary_wraps,
        reason = "The shared terminal API propagates Wasmtime fuel-read failures when fuel accounting is enabled."
    )
)]
pub(crate) fn terminal_outcome(
    store: &mut Store<State>,
    executed: Result<()>,
    #[cfg_attr(not(feature = "diagnostic-fuel-off"), allow(unused_variables))] offered_fuel: u64,
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
        outcome.fuel_remaining = store.get_fuel()?;
        if outcome.termination == "capacity"
            && outcome.fuel_remaining == 0
            && store
                .data()
                .model
                .as_ref()
                .is_some_and(|model| model.inspection.request.is_some())
        {
            outcome.termination = "probe_fuel_limit";
            outcome.message = "instrumented replay exhausted its fuel limit".into();
            if let Some(member) = &store.data().broker {
                member
                    .broker
                    .abort("probe_fuel_limit", outcome.message.clone());
            }
        }
    }
    #[cfg(feature = "diagnostic-fuel-off")]
    {
        outcome.fuel_remaining = offered_fuel;
    }
    outcome.peak_memory_bytes = store.data().limits.peak;
    outcome.output_bytes = store.data().output_bytes;
    Ok(outcome)
}

/// Queues storage evidence, retaining the collected prefix on enumeration or
/// crash-image failure with Go's classified terminal.
pub(crate) fn finish_storage(
    state: &mut State,
    descriptor: Option<&crate::storage_transport::StorageDescriptor>,
    outcome: &mut Outcome,
) {
    if let (Some(descriptor), Some(namespace), Some(model)) =
        (descriptor, &mut state.namespace, &mut state.model)
    {
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(model.event_context());
        let mut evidence = crate::storage_transport::evidence(namespace, descriptor);
        if let Some((kind, message)) = evidence.failure.take() {
            outcome.termination = kind;
            outcome.message = message;
            outcome.exit_code = None;
        }
        model.storage_evidence(evidence);
    }
}

/// Flushes the cooperative model's evidence and adds its result metadata.
pub(crate) fn finish_model(state: &mut State, outcome: &mut Outcome) -> Result<()> {
    let mut network_error = None;
    if let (Some(network), Some(model)) = (&state.network, &mut state.model) {
        match network.evidence() {
            Ok(bytes) => {
                let handle = model.inspection.handle.clone();
                model
                    .inspection
                    .lock()
                    .terminal_network_json(&handle, bytes.clone());
                model.network_evidence(&bytes);
            }
            Err(failure) => network_error = Some(crate::linked_network::error(failure)),
        }
    }
    if let Some(model) = &mut state.model
        && !model.inspection.sealed
    {
        model.close_clock_run();
    }
    state.inspection_snapshot(Snapshot::Terminal);
    let standalone = state.broker.is_none();
    if let Some(model) = &mut state.model {
        if outcome.termination == "exit" && model.runtime_active() {
            outcome.termination = "invalid";
            outcome.message = "runtime trace unfinished".into();
            outcome.exit_code = None;
        }
        let metadata = {
            let mut recorder = model.inspection.lock();
            let admitted_owner = !standalone
                && recorder
                    .evidence
                    .incarnations
                    .iter()
                    .any(|owner| owner.handle == model.inspection.handle);
            if model.inspection.started || admitted_owner {
                recorder.terminal(
                    &model.inspection.handle,
                    outcome.termination != "exit" || outcome.exit_code != Some(0),
                );
            }
            if standalone && recorder.observed {
                Some(recorder.bytes(!model.inspection.sealed)?)
            } else {
                None
            }
        };
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        crate::event_order::emit(
            &model.event_context(),
            "model-transition",
            model.event_context().trace,
            serde_json::json!({"model":"lifecycle","event":"terminal","termination":outcome.termination,"exit_code":outcome.exit_code}),
        );
        model.inspection_evidence(metadata);
        model.flush(&mut state.output, true)?;
        outcome.local_model = Some(Box::new(model.metadata(state.completed_calls)));
        outcome.divergence = model.divergence.take().map(Box::new);
    }
    network_error.map_or(Ok(()), Err)
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
    let config = configuration(false);
    let engine = Engine::new(&config)
        .map_err(|error| Outcome::new("infrastructure", format!("{error:#}")))?;
    #[cfg(feature = "diagnostic-profile")]
    drop(diagnostic_phase);
    let (module, started) = prepare_module(request, &engine, metrics)?;
    Ok((engine, module, started))
}

pub(crate) fn configuration(watchdog: bool) -> Config {
    let mut config = Config::new();
    config
        .consume_fuel(!cfg!(feature = "diagnostic-fuel-off"))
        .native_unwind_info(false)
        .cranelift_nan_canonicalization(true)
        .wasm_relaxed_simd(false)
        .wasm_memory64(false)
        .wasm_multi_memory(false)
        .epoch_interruption(watchdog || cfg!(feature = "guest-profile"));
    config
}

pub(crate) fn prepare_module(
    request: &Execute,
    engine: &Engine,
    metrics: &mut crate::metrics::Metrics,
) -> std::result::Result<(Module, Value), Outcome> {
    let module = match &request.compile_cache {
        Some(directory) => crate::cache::module(
            engine,
            || module_bytes(request),
            directory,
            &request.module_sha256,
            metrics,
        )?,
        None => compile(engine, &module_bytes(request)?, metrics)?,
    };
    describe_module(request, module)
}

pub(crate) fn describe_module(
    request: &Execute,
    module: Module,
) -> std::result::Result<(Module, Value), Outcome> {
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
        let signature =
            abi::Operation::resolve(import.module(), import.name()).map(abi::Operation::signature);
        #[cfg(feature = "determinism-fixture")]
        let signature =
            signature.or_else(|| crate::linked_fixture::signature(import.module(), import.name()));
        let Some((expected_params, expected_results)) = signature else {
            return Err(Outcome::new(
                "unsupported",
                format!("unknown import {}::{}", import.module(), import.name()),
            ));
        };
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
            ExternType::Tag(_) => "other",
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
    Ok((module, started))
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

#[expect(
    clippy::needless_pass_by_value,
    reason = "Wasmtime type iterators yield owned ValType values directly to this mapper."
)]
fn type_name(ty: ValType) -> &'static str {
    match ty {
        ValType::I32 => "i32",
        ValType::I64 => "i64",
        ValType::F32 => "f32",
        ValType::F64 => "f64",
        ValType::V128 => "v128",
        ValType::Ref(_) => "ref",
    }
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Terminal error classification consumes the failed execution error."
)]
pub(crate) fn execution_failure(error: Error, limiter_failure: Option<&'static str>) -> Outcome {
    let classification = limiter_failure.unwrap_or_else(|| match error.downcast_ref::<Trap>() {
        Some(Trap::OutOfFuel) => "capacity",
        Some(_) => "trap",
        None => "infrastructure",
    });
    Outcome::new(classification, format!("{error:#}"))
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Result::map_err transfers ownership of the compilation error to this classifier."
)]
fn compilation_failure(error: Error) -> Outcome {
    let cause = error.root_cause().to_string();
    let message = error
        .downcast_ref::<wasmtime::wasmparser::BinaryReaderError>()
        .map(wasmtime::wasmparser::BinaryReaderError::message)
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
            state.output = Box::new(capture);
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
            Self { store, instance }
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
                            fd.cast_signed(),
                            pointer.cast_signed(),
                            count.cast_signed(),
                            offset.cast_signed(),
                            output.cast_signed(),
                        ),
                    )
            } else {
                self.instance
                    .get_typed_func::<(i32, i32, i32, i32), i32>(&mut self.store, "write")
                    .unwrap()
                    .call(
                        &mut self.store,
                        (
                            fd.cast_signed(),
                            pointer.cast_signed(),
                            count.cast_signed(),
                            output.cast_signed(),
                        ),
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
    #[expect(
        clippy::needless_pass_by_value,
        reason = "The fixture transfers its constructed output into the serialized reply."
    )]
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
                let memory = fixture.memory();
                memory.write(&mut fixture.store, 256, &[0; 32]).unwrap();
                memory.write(&mut fixture.store, 512, &[255; 24]).unwrap();
                memory.write(&mut fixture.store, 4096, &[42; 4096]).unwrap();
                fixture.invoke(positioned, fd, 16, 3, u64::MAX, 0).unwrap();
                assert_eq!(fixture.origins(), origins);
                fixture.restored();
                fixture.descriptors(&[]);
                fixture.invoke(positioned, fd, 16, 0, 0, 0).unwrap();
                assert_eq!(fixture.origins(), origins);
                fixture.restored();

                if !modeled {
                    assert_eq!(fixture.store.data().completed_calls, 0);
                }
            }
        }
    }

    #[test]
    fn typed_write_abi_pointer_and_capacity_priority_precede_callback_mutation() {
        for (iovecs, pointer, count, output, _kind, _expected) in [
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
                vec![(u32::MAX, u32::try_from(protocol::BUFFER_LIMIT).unwrap() + 1)],
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
            let _ = fixture.invoke(true, 1, pointer, count, u64::MAX, output);
            assert_eq!(fixture.store.data().calls, 0);
            assert_eq!(fixture.output_count(), u32::MAX);
            assert_eq!(fixture.origins(), origins);
            fixture.restored();
        }
        let mut fixture = WriteAbiFixture::new(65, true, Vec::new());
        fixture.descriptors(&[
            (4096, u32::try_from(protocol::BUFFER_LIMIT).unwrap()),
            (4096, 1),
        ]);
        let _ = fixture.invoke(true, 13, 16, 2, 0, 0);
        assert_eq!(fixture.store.data().calls, 0);
        fixture.restored();
    }

    #[test]
    fn typed_write_abi_max_buffer_and_reservation_failures_restore_ownership() {
        let mut fixture = WriteAbiFixture::new(
            65,
            true,
            write_abi_reply(1, 0, json!({"written":protocol::BUFFER_LIMIT}), true),
        );
        fixture.descriptors(&[(4096, u32::try_from(protocol::BUFFER_LIMIT).unwrap())]);
        fixture.memory().data_mut(&mut fixture.store)[4096..4096 + protocol::BUFFER_LIMIT].fill(42);
        fixture.invoke(true, 13, 16, 1, u64::MAX, 0).unwrap();

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
            let _ = fixture.invoke(false, 1, 16, 1, 0, 0);
            assert_eq!(
                fixture.store.data().calls,
                if output_budget { 0 } else { 8 }
            );
            assert_eq!(fixture.origins(), origins);
            assert_eq!(fixture.output_count(), u32::MAX);
            fixture.restored();
        }
    }

    #[test]
    fn typed_write_abi_charges_offered_output_including_partial_and_failed_writes() {
        for modeled in [false, true] {
            for errno in [0, 6] {
                let replies = write_abi_reply(
                    1,
                    errno,
                    if errno == 0 {
                        json!({"written":16})
                    } else {
                        json!({})
                    },
                    modeled,
                );
                let mut fixture = WriteAbiFixture::new(2, modeled, replies);
                fixture.store.data_mut().output_limit = 32;
                fixture
                    .memory()
                    .write(&mut fixture.store, 256, &[7; 32])
                    .unwrap();
                fixture.descriptors(&[(256, 32)]);
                fixture.invoke(false, 1, 16, 1, 0, 0).unwrap();
                fixture.restored();
                fixture.descriptors(&[(256, 1)]);
                let _ = fixture.invoke(false, 1, 16, 1, 0, 0);
                fixture.restored();
            }
        }
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The test preserves the ordered complete reply and transport restoration matrix."
    )]
    fn typed_write_abi_reply_errors_chunks_interrupts_and_transport_restore_buffers() {
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
        struct Broken;
        impl Write for Broken {
            fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
                Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe))
            }
            fn flush(&mut self) -> std::io::Result<()> {
                unreachable!()
            }
        }
        let mut unsupported: Value =
            serde_json::from_slice(&write_abi_reply(1, 0, json!({}), true)).unwrap();
        unsupported["unsupported"] = json!(true);
        unsupported["message"] = json!("outside profile");
        let mut unsupported_reply = Vec::new();
        protocol::send_frame(&mut unsupported_reply, &unsupported).unwrap();
        for (reply, expected, _completed, _errno) in [
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
            let _ = fixture.invoke(false, 2, 16, 1, 0, 0);
            if expected.is_some() {
                assert_eq!(fixture.output_count(), u32::MAX);
            }
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
        fixture.invoke(true, 13, 16, 1, 0, 0).unwrap();
        fixture.restored();
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
        for (op, input) in [
            ("fd_write", json!({"fd":u32::MAX,"data_base64":"AP8="})),
            (
                "fd_pwrite",
                json!({"fd":13,"data_base64":"AP8=","offset":u64::MAX}),
            ),
        ] {
            let mut reference = None;
            for typed in [false, true] {
                let mut state = clock_state(4, 65536, 1);
                let capture = WriteCapture::default();
                state.output = Box::new(capture.clone());
                state.input = Box::new(std::io::Cursor::new(
                b"{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"written\":2},\"local_state\":{\"now\":7,\"advances\":3},\"transcript_base64\":\"AQI=\"}\n".to_vec(),
            ));
                let reply = if typed {
                    let data_base64 = input["data_base64"].as_str().unwrap();
                    let fd = u32::try_from(input["fd"].as_u64().unwrap()).unwrap();
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
                let observed = (
                    capture.0.lock().unwrap().clone(),
                    reply.output,
                    state.model.as_ref().unwrap().state(),
                    state.calls,
                    state.completed_calls,
                    state.terminal.is_none(),
                );
                if let Some(expected) = &reference {
                    assert_eq!(&observed, expected);
                } else {
                    reference = Some(observed);
                }
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
                    assert_eq!(scratch, [] as [u8; 0]);
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
        assert_eq!(output, [] as [u8; 0]);
        assert_eq!(scratch, [] as [u8; 0]);
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
        struct Broken;
        impl Write for Broken {
            fn write(&mut self, _: &[u8]) -> std::io::Result<usize> {
                Err(std::io::Error::from(std::io::ErrorKind::BrokenPipe))
            }
            fn flush(&mut self) -> std::io::Result<()> {
                unreachable!()
            }
        }
        for (reply, _termination, _completed) in [
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
            let _ = state.callback_input(protocol::CallbackInput::PWrite {
                fd: 13,
                data_base64: "AP8=",
                offset: u64::MAX,
            });
            assert_eq!(state.request_scratch.as_ptr(), origin);
            assert_eq!(state.request_scratch, [] as [u8; 0]);
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
        assert_eq!(state.request_scratch, [] as [u8; 0]);
        assert_eq!((state.calls, state.completed_calls), (1, 0));
    }

    #[test]
    fn typed_write_reference_capacity_and_decoder_fail_before_owner_merge() {
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
            host_nanos: None,
            stdout_progress: false,
            output_limit: 1024,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: Some(crate::model::Model::new(config)),
            dormant_debug: None,
            namespace: None,
            request_scratch: Vec::new(),
            write_scratch: protocol::WriteBuffers::default(),
            broker: None,
            network: None,
        }
    }
    fn inspection_state() -> State {
        use crate::{
            cluster::{handoff as broker, network as net},
            model::inspection,
        };
        let ids = ["a".to_owned()];
        let network_config: net::Config = serde_json::from_value(json!({"model":{"profile":net::PROFILE,"limits":{"listeners":16,"connections":16,"bytes":1_048_576,"connection_bytes":65536,"stall_nanos":1_000_000,"transitions":4096}}})).unwrap();
        let network = Arc::new(Mutex::new(net::Shared::new(&network_config, &ids).unwrap()));
        let broker_config: broker::Config = serde_json::from_value(json!({"seed":7,"limits":{"max_requests":10000,"max_events":10000,"max_queued_events":1000,"max_transitions":40000,"max_payload_bytes":33_554_432,"max_string_bytes":1024}})).unwrap();
        let owner = Arc::new(
            broker::Broker::new(&broker_config, &ids, Some(network.clone()), None).unwrap(),
        );
        let endpoint =
            net::Endpoint::new(network, "a", 1, 1, &network_config.model, 1000, 1 << 20).unwrap();
        let mut state = clock_state(100, 65536, 1);
        state.broker = Some(broker::Member {
            broker: owner,
            index: 0,
            incarnation: 1,
            network: Some(endpoint),
        });
        let model = state.model.as_mut().unwrap();
        model.runtime("runtime_config", vec![0; 16]).unwrap();
        model.inspection.handle = inspection::Handle {
            node: "a".into(),
            incarnation: 1,
        };
        model.inspection.started = true;
        model.inspection.world_commitment = vec![42; 32];
        model.inspection.lock().observe(&model.inspection.handle);
        state
    }
    #[test]
    fn unchanged_boundaries_never_materialize_world() {
        let mut state = inspection_state();
        let owner = state.broker.as_ref().unwrap().broker.clone();
        for _ in 0..100 {
            state.inspection_snapshot(Snapshot::Boundary);
        }
        assert_eq!(
            owner
                .world_materializations
                .load(std::sync::atomic::Ordering::Relaxed),
            0
        );
        assert_eq!(
            state
                .model
                .as_ref()
                .unwrap()
                .inspection
                .world_commitment
                .len(),
            32
        );
    }
    #[test]
    fn handoff_and_terminal_cuts_defer_world_until_the_current_boundary() {
        let mut state = inspection_state();
        state.inspection_snapshot(Snapshot::Call);
        assert_eq!(
            state.model.as_ref().unwrap().inspection.world_commitment,
            vec![42; 32],
            "handoff cut eagerly materialized unused World"
        );
        state
            .broker
            .as_mut()
            .unwrap()
            .network
            .as_mut()
            .unwrap()
            .attach_at(7)
            .unwrap();
        let mut frame = vec![0u8; 16 + 96 + 128];
        frame[..4].copy_from_slice(&1u32.to_le_bytes());
        frame[4..8].copy_from_slice(&1u32.to_le_bytes());
        frame[12..16].copy_from_slice(&224u32.to_le_bytes());
        frame[16..20].copy_from_slice(&1u32.to_be_bytes());
        frame[20..24].copy_from_slice(&1u32.to_be_bytes());
        frame[24..32].copy_from_slice(&42u64.to_be_bytes());
        frame[36..40].copy_from_slice(&2u32.to_be_bytes());
        frame[40..44].copy_from_slice(&1u32.to_be_bytes());
        frame[48..80].fill(2);
        frame[80..112].fill(1);
        frame[224..232].copy_from_slice(&1u64.to_be_bytes());
        state.debug_exchange(&mut frame).unwrap();
        let before = state
            .model
            .as_ref()
            .unwrap()
            .inspection
            .world_commitment
            .clone();
        state
            .broker
            .as_mut()
            .unwrap()
            .network
            .as_mut()
            .unwrap()
            .attach_at(8)
            .unwrap();
        finish_model(&mut state, &mut Outcome::new("exit", "")).unwrap();
        assert_eq!(
            state.model.as_ref().unwrap().inspection.world_commitment,
            before,
            "terminal cut materialized unused World"
        );
    }
    #[test]
    fn ordinary_model_does_not_capture_state_without_debug_hooks() {
        let mut state = clock_state(100, 65536, 1);
        state.model.as_mut().unwrap().inspection.started = true;
        state.inspection_snapshot(Snapshot::Call);
        let recorder = state.model.as_ref().unwrap().inspection.lock();
        assert!(recorder.evidence.incarnations.is_empty());
        assert!(!recorder.observed);
    }
    #[test]
    fn go_owned_debug_refuses_active_inspection_and_malformed_frames() {
        for op in [4u32, 5, 6, 8, 9, 11] {
            let mut state = clock_state(100, 65536, 1);
            state.model = None;
            let mut frame = [0u8; 16];
            frame[..4].copy_from_slice(&1u32.to_le_bytes());
            frame[4..8].copy_from_slice(&op.to_le_bytes());
            let _ = state.debug_exchange(&mut frame);
            assert_eq!(state.calls, 0);
        }
    }
    #[test]
    fn typed_clock_state_capacity_preserves_attempted_and_completed_counts() {
        for (calls, transcript, now, prime, _message) in [
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
            let _ = state.clock_time_get(1, 0);
            assert_eq!(state.calls, 1);
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
                let precision = u64::from(index) * 17;
                let _timestamp = if typed {
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
            }
            let model = state.model.as_mut().unwrap();
            model.flush(&mut state.output, true).unwrap();
            let metadata = model.metadata(state.completed_calls);
            let bytes = capture.0.lock().unwrap().clone();
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
        let error = std::fs::File::open("/missing-gomad-wasmhost-backing").unwrap_err();
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
            host_nanos: None,
            stdout_progress: false,
            output_limit: 1024,
            output_bytes: 0,
            terminal: None,
            argument_sizes: None,
            environment_sizes: None,
            model: None,
            dormant_debug: None,
            namespace: None,
            request_scratch: Vec::new(),
            write_scratch: protocol::WriteBuffers::default(),
            broker: None,
            network: None,
        };
        assert!(state.callback("sched_yield", json!({})).is_err());
        assert_eq!(state.terminal.unwrap().termination, "infrastructure");
    }
}
