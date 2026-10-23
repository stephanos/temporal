#[cfg(test)]
use super::CPU_CLOCK_UNSUPPORTED;
use super::{Decision, MAX_TIME, Model, PollResult, get32, get64, inspection, set32, set64};
use crate::abi::Operation;
#[cfg(test)]
use crate::protocol::{self, Reply};
use crate::transcript::Subscription;
#[cfg(test)]
use base64::{Engine as _, engine::general_purpose::STANDARD};
#[cfg(test)]
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use wasmtime::{Error, Result};

impl Model {
    pub fn namespace_call(
        &mut self,
        id: u64,
        namespace: &mut crate::namespace::Namespace,
        call: crate::namespace::call::Call,
    ) -> Result<crate::namespace::Reply> {
        call.admit(
            id,
            self.config.limits.calls,
            self.transcript_bytes,
            self.config.limits.transcript_bytes,
            self.config.digest_only,
        )
        .map_err(Error::new)?;
        let reply = namespace
            .execute(call.operation, self.now)
            .map_err(crate::namespace::Stop::into_error)?;
        let output = crate::namespace::call::output_bytes(reply.output.as_ref());
        self.append_record(&crate::transcript::record(
            call.tag,
            id,
            call.input,
            &output,
            reply.errno,
            reply.unsupported.as_deref().unwrap_or(""),
        ));
        Ok(reply)
    }
    pub fn strings(&mut self, id: u64, operation: Operation) -> Result<(Vec<u8>, Vec<u32>)> {
        self.reserve_transcript(crate::transcript::RECORD_HEADER)?;
        self.append_record(&crate::transcript::strings_record(id, operation));
        let values = if matches!(operation, Operation::ArgsGet | Operation::ArgsSizesGet) {
            &self.config.arguments
        } else {
            &self.config.environment
        };
        Ok((values.packed.clone(), values.offsets.clone()))
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The entropy request length is checked against four MiB before transcript encoding."
    )]
    pub fn random(&mut self, id: u64, length: usize) -> Result<Vec<u8>> {
        if length > 4 * 1024 * 1024 || self.entropy_offset.checked_add(length as u64).is_none() {
            return Err(Error::msg("WASI capacity: entropy-bytes"));
        }
        self.reserve_transcript(crate::transcript::ENTROPY_BYTES)?;
        let start = self.entropy_offset;
        let mut bytes = Vec::with_capacity(length);
        while bytes.len() < length {
            let mut hash = Sha256::new();
            hash.update(b"gomad.wasi-entropy-sha256-counter/v1");
            hash.update(self.config.entropy_key);
            hash.update((self.entropy_offset / 32).to_le_bytes());
            let block = hash.finalize();
            let skip = (self.entropy_offset % 32) as usize;
            let count = (length - bytes.len()).min(32 - skip);
            bytes.extend_from_slice(&block[skip..skip + count]);
            self.entropy_offset += count as u64;
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        self.event("entropy",id,serde_json::json!({"offset":start,"length":length,"value":crate::event_order::bytes_value(&bytes)}));
        self.append_record(&crate::transcript::entropy_record(id, length as u32, start));
        Ok(bytes)
    }
    pub fn clock_read(&mut self, id: u64, clock_id: u32, precision: u64) -> Result<Option<u64>> {
        #[cfg(feature = "determinism-fixture")]
        {
            let boundaries = self.inspection.lock().evidence.boundaries.len() as u64;
            self.now += self.fixture_fault.clock_shift(boundaries);
        }
        let folds = clock_id <= 1
            && self.transcript_encoder.folds(
                id,
                clock_id,
                precision,
                self.now,
                self.config.read_step_nanos,
            );
        if !folds {
            self.close_clock_run();
            self.reserve_transcript(if clock_id <= 1 {
                crate::transcript::CLOCK_RUN_BYTES
            } else {
                crate::transcript::clock_reservation()
            })?;
        }
        let timestamp = if clock_id <= 1 {
            let timestamp = crate::local_model::clock_timestamp(
                self.now,
                self.config.epoch_nanos,
                self.config.read_step_nanos,
                clock_id,
            )
            .ok_or_else(|| Error::msg("WASI capacity: clock-nanoseconds"))?;
            if self.transcript_bytes == 0 {
                self.transcript(crate::transcript::HEADER);
            }
            if !folds {
                self.transcript_bytes += crate::transcript::CLOCK_RUN_BYTES;
            }
            self.transcript_encoder
                .read(id, clock_id, precision, self.now);
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            self.event("clock",id,serde_json::json!({"clock_id":clock_id,"precision":precision,"timestamp":timestamp}));
            self.now += self.config.read_step_nanos;
            Some(timestamp)
        } else {
            None
        };
        if timestamp.is_none() {
            self.append_record(&crate::transcript::clock_unsupported(
                id, clock_id, precision,
            ));
        }
        Ok(timestamp)
    }
    #[cfg(test)]
    pub fn clock_time_get(
        &mut self,
        id: u64,
        clock_id: Option<u64>,
        call: &[u8],
    ) -> Result<Option<u64>> {
        let value: Value = serde_json::from_slice(call).unwrap();
        self.clock_read(
            id,
            clock_id
                .and_then(|id| u32::try_from(id).ok())
                .unwrap_or(u32::MAX),
            value["input"]["precision"].as_u64().unwrap(),
        )
    }
    #[cfg(test)]
    pub fn handle(&mut self, id: u64, op: &str, input: &Value) -> Result<Option<Reply>> {
        self.handle_input(id, &protocol::CallbackInput::Json { op, input })
    }
    #[cfg(test)]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The JSON reference preserves WASI clock-id conversion to its low 32 bits."
    )]
    pub fn handle_input(
        &mut self,
        id: u64,
        callback: &protocol::CallbackInput<'_>,
    ) -> Result<Option<Reply>> {
        let Some(operation) = callback.operation() else {
            return Ok(None);
        };
        let input = callback.json();
        let local = operation.is_model_local()
            || operation == Operation::PollOneoff
                && input
                    .and_then(|input| input["subscriptions"].as_array())
                    .is_some_and(|subs| subs.iter().all(|sub| sub["type"] == 0));
        if !local {
            return Ok(None);
        }
        let input = input.unwrap();
        let mut errno = 0;
        let mut unsupported = "";
        let output = match operation {
            Operation::RandomGet => {
                json!({"data_base64":STANDARD.encode(self.random(id, usize::try_from(input["length"].as_u64().unwrap()).map_err(|_| Error::msg("WASI capacity: entropy-bytes"))?)?)})
            }
            Operation::ArgsSizesGet
            | Operation::EnvironSizesGet
            | Operation::ArgsGet
            | Operation::EnvironGet => {
                let (packed, offsets) = self.strings(id, operation)?;
                if matches!(
                    operation,
                    Operation::ArgsSizesGet | Operation::EnvironSizesGet
                ) {
                    json!({"count":offsets.len(),"bytes":packed.len()})
                } else {
                    json!({"values":packed.split_inclusive(|byte| *byte==0).map(|value| STANDARD.encode(&value[..value.len()-1])).collect::<Vec<_>>()})
                }
            }
            Operation::ClockTimeGet => {
                let timestamp = self.clock_read(
                    id,
                    input["clock_id"].as_u64().unwrap_or(u64::from(u32::MAX)) as u32,
                    input["precision"].as_u64().unwrap(),
                )?;
                if timestamp.is_none() {
                    unsupported = CPU_CLOCK_UNSUPPORTED;
                }
                timestamp.map_or_else(|| json!({}), |timestamp| json!({"timestamp":timestamp}))
            }
            Operation::SchedYield => {
                self.yield_call(id)?;
                json!({})
            }
            Operation::PollOneoff => {
                let result = self.poll_call(id, &crate::transcript::subscriptions(input)?)?;
                errno = result.errno;
                unsupported = result.unsupported;
                if errno == 0 && unsupported.is_empty() {
                    json!({"events":result.events.iter().map(|event| json!({"userdata":event.userdata,"errno":event.errno,"type":event.kind,"nbytes":event.nbytes,"flags":event.flags})).collect::<Vec<_>>()})
                } else {
                    json!({})
                }
            }
            _ => {
                let encoded = input["data_base64"]
                    .as_str()
                    .ok_or_else(|| Error::msg("WASI invalid: runtime input"))?;
                json!({"data_base64":STANDARD.encode(self.runtime_typed(operation, STANDARD.decode(encoded).map_err(Error::new)?)?)})
            }
        };
        Ok(Some(Reply {
            kind: "reply".into(),
            id,
            errno,
            output,
            unsupported: !unsupported.is_empty(),
            message: unsupported.into(),
            local_state: None,
            transcript_base64: None,
            revocable: false,
        }))
    }
    pub fn yield_call(&mut self, id: u64) -> Result<()> {
        self.reserve_transcript(crate::transcript::RECORD_HEADER)?;
        self.append_record(&crate::transcript::yield_record(id));
        Ok(())
    }
    #[cfg(test)]
    pub fn runtime(&mut self, op: &str, data: Vec<u8>) -> Result<Vec<u8>> {
        let name = op
            .strip_prefix("runtime_")
            .ok_or_else(|| Error::msg("unsupported runtime operation"))?;
        let operation = Operation::resolve(
            if name == "net_stream" {
                "gomad_wasm_v2"
            } else {
                "gomad_wasm_v1"
            },
            name,
        )
        .ok_or_else(|| Error::msg("unsupported runtime operation"))?;
        self.runtime_typed(operation, data)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Keep runtime lifetime checks, decision validation, tape matching, and mutation in their protocol order."
    )]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Decision kinds are 1 or 2 and validated alternative counts and selected indices are at most 256."
    )]
    pub fn runtime_typed(&mut self, op: Operation, mut data: Vec<u8>) -> Result<Vec<u8>> {
        self.close_clock_run();
        if op == Operation::RuntimeConfig {
            if self.configured || data.len() != 16 || data.iter().any(|b| *b != 0) {
                return Err(Error::msg("WASI invalid: runtime bootstrap"));
            }
            set32(&mut data, 0, 1);
            set32(&mut data, 4, u32::from(self.config.diagnostics));
            set64(&mut data, 8, self.config.seed);
            #[cfg(feature = "determinism-fixture")]
            set64(
                &mut data,
                8,
                self.fixture_fault.bootstrap_seed(self.config.seed),
            );
            self.configured = true;
            return Ok(data);
        }
        if !self.configured || self.finished {
            return Err(Error::msg(
                "WASI invalid: runtime operation outside active lifetime",
            ));
        }
        match op {
            Operation::RuntimeDecision => {
                if data.len() < 32 || !matches!(get32(&data, 0), 1 | 2) {
                    return Err(Error::msg("WASI invalid: runtime decision header"));
                }
                let kind = get32(&data, 4);
                let flags = get32(&data, 16);
                let count = get32(&data, 20) as usize;
                let selected = get32(&data, 24) as usize;
                let creation = get32(&data, 0) == 2 && kind == 1;
                let diagnostic_offset = 32 + count * if creation { 48 } else { 32 };
                if !matches!(kind, 1 | 2)
                    || flags & !4 != 0
                    || !(2..=256).contains(&count)
                    || selected >= count
                    || data.len()
                        != diagnostic_offset + if self.config.diagnostics { 96 } else { 0 }
                {
                    return Err(Error::msg(
                        "WASI invalid: runtime decision alternatives or layout",
                    ));
                }
                let alternatives = data[32..32 + count * 32].as_chunks::<32>().0.to_vec();
                let mut ordered = alternatives.clone();
                ordered.sort_unstable();
                if ordered.contains(&[0; 32])
                    || ordered.windows(2).any(|v| v[0] == v[1])
                    || flags != 0 && get64(&data, 8) != 0
                {
                    return Err(Error::msg("WASI invalid: runtime decision identities"));
                }
                let alternative_set_digest = {
                    #[cfg(feature = "diagnostic-profile")]
                    let _hash = crate::diagnostic::WallSpan::new(
                        crate::diagnostic::Nested::AlternativeHash,
                    );
                    let mut hash = Sha256::new();
                    hash.update(b"gomad-choice-alternative-set/v1\0");
                    hash.update((count as u64).to_be_bytes());
                    for identity in &ordered {
                        hash.update(identity);
                    }
                    hash.finalize().into()
                };
                let mut creation_sites = Vec::new();
                if creation {
                    for (index, identity) in alternatives.iter().enumerate() {
                        let start = 32 + count * 32 + index * 16;
                        let missing = data[start + 8];
                        let site_offset = get64(&data, start);
                        if missing & !4 != 0
                            || data[start + 9..start + 16].iter().any(|byte| *byte != 0)
                            || missing != 0 && site_offset != 0
                        {
                            return Err(Error::msg("WASI invalid: runtime creation site"));
                        }
                        creation_sites.push(super::CreationSite {
                            identity: *identity,
                            site_offset,
                            site_missing: missing != 0,
                        });
                    }
                    creation_sites.sort_unstable_by_key(|site| site.identity);
                    if 64 + (self.records + count as u64 + 1) * 96 > self.config.limit {
                        return Err(Error::msg("WASI capacity: runtime-choice-bytes"));
                    }
                }
                let mut observed = Decision {
                    ordinal: self.records,
                    kind: kind as u8,
                    site_offset: get64(&data, 8),
                    site_missing: flags != 0,
                    rank_override: false,
                    alternatives: count as u32,
                    selected: ordered
                        .iter()
                        .position(|v| *v == alternatives[selected])
                        .unwrap() as u32,
                    data: get32(&data, 28),
                    selected_identity: alternatives[selected],
                    alternative_set_digest,
                    creation_sites,
                };
                let tape = self.config.tape.as_deref().unwrap_or(&[]);
                if self.config.mode == 2 || self.config.mode == 3 && self.cursor < tape.len() {
                    let Some(expected) = tape.get(self.cursor).cloned() else {
                        return Err(self.diverge(6, None, Some(observed)));
                    };
                    let reason = if expected.kind != observed.kind {
                        1
                    } else if expected.site_offset != observed.site_offset
                        || expected.site_missing != observed.site_missing
                    {
                        2
                    } else if expected.alternatives != observed.alternatives {
                        3
                    } else if expected.alternative_set_digest != observed.alternative_set_digest {
                        5
                    } else if expected.selected >= count as u32
                        || !expected.rank_override
                            && ordered[expected.selected as usize] != expected.selected_identity
                    {
                        4
                    } else {
                        0
                    };
                    if reason != 0 {
                        return Err(self.diverge(reason, Some(expected), Some(observed)));
                    }
                    observed.selected = expected.selected;
                    observed.selected_identity = ordered[expected.selected as usize];
                    set32(
                        &mut data,
                        24,
                        alternatives
                            .iter()
                            .position(|v| *v == observed.selected_identity)
                            .unwrap() as u32,
                    );
                    self.cursor += 1;
                }
                for (index, site) in observed.creation_sites.iter().enumerate() {
                    let mut record = [0; 96];
                    set64(&mut record, 0, self.records);
                    record[8] = 4;
                    record[9] = if site.site_missing { 4 } else { 0 };
                    set32(&mut record, 12, count as u32);
                    set32(&mut record, 16, index as u32);
                    set64(&mut record, 24, site.site_offset);
                    record[32..64].copy_from_slice(&site.identity);
                    let mut diagnostic = data[diagnostic_offset..].to_vec();
                    if self.config.diagnostics {
                        set64(&mut diagnostic, 0, self.records);
                    }
                    self.append(&record, &diagnostic)?;
                }
                observed.ordinal = self.records;
                #[cfg(feature = "diagnostic-profile")]
                let encoding =
                    crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
                let record = super::choice_wire_generated::encode_decision(&observed);
                #[cfg(feature = "diagnostic-profile")]
                drop(encoding);
                let mut diagnostic = data[diagnostic_offset..].to_vec();
                if self.config.diagnostics {
                    set64(&mut diagnostic, 0, self.records);
                }
                self.append(&record, &diagnostic)?;
            }
            Operation::RuntimeObservation => {
                if data.len() != 96 + if self.config.diagnostics { 96 } else { 0 } {
                    return Err(Error::msg("WASI invalid: runtime observation layout"));
                }
                let count = get32(&data, 12);
                let readiness = u16::from_be_bytes(data[10..12].try_into().unwrap());
                if get64(&data, 0) != self.records
                    || data[8] != 3
                    || data[9] & !6 != 0
                    || data[9] & 2 == 0
                    || count == 0
                    || get32(&data, 16) >= count
                    || data[9] & 4 != 0 && get64(&data, 24) != 0
                    || readiness & 1 == 0
                    || u32::from(readiness >> 6) > count - u32::from(readiness & 2 != 0)
                    || get64(&data, 32) > self.records
                    || data[40..96].iter().any(|b| *b != 0)
                {
                    return Err(Error::msg("WASI invalid: runtime select observation"));
                }
                self.append(&data[..96], &data[96..])?;
            }
            Operation::RuntimeFinish => {
                if data.len() != 16 || get32(&data, 0) != 1 {
                    return Err(Error::msg("WASI invalid: runtime finish layout"));
                }
                if self
                    .config
                    .tape
                    .as_ref()
                    .is_some_and(|t| self.cursor != t.len())
                {
                    return Err(self.diverge(
                        7,
                        Some(self.config.tape.as_ref().unwrap()[self.cursor].clone()),
                        None,
                    ));
                }
                self.peak = get32(&data, 4);
                self.elapsed = get64(&data, 8);
                self.terminal.clone_from(&data);
                self.finished = true;
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                self.event("model-transition",self.records,serde_json::json!({"model":"runtime-finish","terminal":crate::event_order::bytes_value(&data)}));
            }
            Operation::RuntimeIdle => {
                if data.len() != 16 || get32(&data, 0) != 1 || get32(&data, 4) != 0 {
                    return Err(Error::msg("WASI invalid: runtime idle layout"));
                }
                let deadline = get64(&data, 8);
                if deadline == 0 || deadline > MAX_TIME - self.config.epoch_nanos {
                    return Err(Error::msg("WASI invalid: runtime idle deadline"));
                }
                if deadline > self.now {
                    self.advances += 1;
                    self.now = deadline;
                }
                set64(&mut data, 8, self.now);
            }
            _ => return Err(Error::msg("WASI invalid: runtime operation")),
        }
        Ok(data)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "The debug protocol keeps each opcode and its lifetime checks in one dispatch."
    )]
    pub fn debug_exchange(&mut self, frame: &mut [u8]) -> Result<()> {
        use inspection::{Witness, le32};
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let _event_scope = crate::event_order::enter(self.event_context());
        let op = inspection::validate_frame(frame)?;
        let payload = &mut frame[16..];
        self.inspection.lock().observe(&self.inspection.handle);
        self.inspection.time = self.now;
        match op {
            1 => {
                if self.inspection.sealed || payload.len() < 32 + 64 + 128 {
                    return Err(Error::msg("invalid debug predecision lifetime or length"));
                }
                let count = get32(payload, 20) as usize;
                let length = 32 + count * 32;
                if !(2..=256).contains(&count) || payload.len() != length + 128 {
                    return Err(Error::msg("invalid debug predecision alternatives"));
                }
                let selected = self.validate_debug_decision(&payload[..length])?;
                let witness = Witness::decode(&payload[length..])?;
                if !self.config.split && witness.choices + witness.observations != self.records {
                    return Err(Error::msg(
                        "instrumentation divergence: runtime record witness",
                    ));
                }
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                self.witness_event(&witness);
                self.inspection.witness = witness.clone();
                let handle = self.inspection.handle.clone();
                let clock = self
                    .transcript_encoder
                    .pending()
                    .map(|bytes| bytes.to_vec())
                    .unwrap_or_default();
                let boundary = {
                    let mut recorder = self.inspection.lock();
                    recorder.boundary(
                        &handle,
                        witness,
                        self.now,
                        clock,
                        &self.inspection.world_commitment,
                    )
                };
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                if let Ok(boundary) = &boundary {
                    crate::event_order::boundary(
                        &crate::event_order::Context {
                            instance: boundary.handle.node.clone(),
                            incarnation: boundary.handle.incarnation,
                            trace: boundary.trace,
                            decision: boundary.decision,
                            time_nanos: boundary.time_nanos.cast_signed(),
                        },
                        boundary.order,
                    );
                }
                let Some(request) = self.inspection.request.as_ref() else {
                    return Ok(());
                };
                let boundary = boundary?;
                if request.target == handle && boundary.decision == request.decision {
                    self.verify_inspection_cut(request, &boundary)?;
                    let commands = request.commands.clone();
                    self.inspection.commands = commands.into();
                    self.inspection.lock().sealed = true;
                    self.inspection.sealed = true;
                    set32(payload, 24, selected);
                    frame[8..12].copy_from_slice(&1u32.to_le_bytes());
                }
            }
            2 => {
                if self.inspection.sealed {
                    return Err(Error::msg("registration after inspection seal"));
                }
                self.inspection.registry(payload)?;
            }
            3 => {
                if payload.len() != 4 || le32(payload, 0) > 1 || self.inspection.sealed {
                    return Err(Error::msg("invalid registry closure"));
                }
                self.inspection.close_registry(le32(payload, 0) == 1);
            }
            4 => {
                if payload.len() < 16 || le32(payload, 12) as usize != payload.len() - 16 {
                    return Err(Error::msg("invalid debug record length"));
                }
                let kind = le32(payload, 0);
                let id = le32(payload, 4);
                if kind != 4 {
                    let recorder = self.inspection.lock();
                    let owner = recorder
                        .evidence
                        .incarnations
                        .iter()
                        .find(|n| n.handle == self.inspection.handle)
                        .ok_or_else(|| Error::msg("debug record without registry"))?;
                    let entry = owner
                        .registry
                        .entries
                        .get(id as usize)
                        .ok_or_else(|| Error::msg("unknown debug record id"))?;
                    if !matches!(
                        (kind, entry.kind.as_str()),
                        (1, "probe") | (2, "dump") | (3, "check")
                    ) || kind != 1 && !self.inspection.sealed
                    {
                        return Err(Error::msg("debug record outside enabled lifetime"));
                    }
                }
                if self.inspection.request.is_none() {
                    return Err(Error::msg("debug output enabled during recording"));
                }
                self.inspection.frame(op, payload);
            }
            5 => {
                if !self.inspection.sealed || payload.len() != 8 || payload.iter().any(|b| *b != 0)
                {
                    return Err(Error::msg("debug command outside sealed inspection"));
                }
                if let Some(command) = self.inspection.commands.pop_front() {
                    payload[..4].copy_from_slice(&command.kind.to_le_bytes());
                    payload[4..].copy_from_slice(&command.id.to_le_bytes());
                }
            }
            6 => {
                if !self.inspection.sealed
                    || !payload.is_empty()
                    || !self.inspection.commands.is_empty()
                {
                    return Err(Error::msg("invalid inspected-prefix terminal"));
                }
            }
            7 => {
                if payload.len() != 32 || payload.iter().any(|b| *b != 0) {
                    return Err(Error::msg("invalid debug enablement mask"));
                }
                if let Some(request) = &self.inspection.request
                    && request.target == self.inspection.handle
                {
                    for &id in &request.probes {
                        if id >= 256 {
                            return Err(Error::msg("invalid enabled probe id"));
                        }
                        payload[id as usize / 8] |= 1 << (id % 8);
                    }
                }
            }
            8 | 9 | 11 => {
                if !self.inspection.sealed {
                    return Err(Error::msg("runtime dump before prefix seal"));
                }
                self.inspection.frame(op, payload);
            }
            10 => {
                if self.inspection.sealed {
                    return Err(Error::msg("runtime witness after prefix seal"));
                }
                let witness = Witness::decode(payload)?;
                if !self.config.split && witness.choices + witness.observations != self.records {
                    return Err(Error::msg(
                        "instrumentation divergence: observation witness",
                    ));
                }
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                self.witness_event(&witness);
                self.inspection.witness = witness;
            }
            _ => return Err(Error::msg("unknown debug operation")),
        }
        Ok(())
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "validate_pending bounds alternative counts to 256 and seeded and selected indices below that count."
    )]
    fn validate_debug_decision(&self, data: &[u8]) -> Result<u32> {
        let ordered = inspection::validate_pending(data)?;
        let count = get32(data, 20) as usize;
        let seeded = get32(data, 24) as usize;
        if self.inspection.request.is_none() {
            return Ok(seeded as u32);
        }
        let alternatives = data[32..].as_chunks::<32>().0.to_vec();
        let mut hash = Sha256::new();
        hash.update(b"gomad-choice-alternative-set/v1\0");
        hash.update((count as u64).to_be_bytes());
        for identity in &ordered {
            hash.update(identity);
        }
        let digest: [u8; 32] = hash.finalize().into();
        if self.config.mode == 2
            || self.config.mode == 3
                && self.cursor < self.config.tape.as_deref().unwrap_or(&[]).len()
        {
            let expected = self
                .config
                .tape
                .as_deref()
                .unwrap_or(&[])
                .get(self.cursor)
                .ok_or_else(|| Error::msg("instrumentation divergence: pending tape exhausted"))?;
            if u32::from(expected.kind) != get32(data, 4)
                || expected.site_offset != get64(data, 8)
                || expected.site_missing != (get32(data, 16) != 0)
                || expected.alternatives as usize != count
                || expected.alternative_set_digest != digest
                || expected.selected as usize >= count
                || !expected.rank_override
                    && ordered[expected.selected as usize] != expected.selected_identity
            {
                return Err(Error::msg("instrumentation divergence: pending decision"));
            }
            return Ok(alternatives
                .iter()
                .position(|id| *id == ordered[expected.selected as usize])
                .unwrap() as u32);
        }
        Ok(seeded as u32)
    }
    fn verify_inspection_cut(
        &self,
        request: &inspection::Request,
        boundary: &inspection::Boundary,
    ) -> Result<()> {
        let recorder = self.inspection.lock();
        let expected = &request.expected;
        if recorder.exhausted
            || expected.handle != boundary.handle
            || expected.decision != boundary.decision
            || expected.trace != boundary.trace
            || expected.order != boundary.order
            || expected.time_nanos != boundary.time_nanos
            || expected.world_bytes != boundary.world_bytes
            || expected.lifecycle_bytes != boundary.lifecycle_bytes
            || expected.cuts != boundary.cuts
            || request.world_sha256 != inspection::digest(&recorder.evidence.world)
            || request.lifecycle_sha256 != inspection::digest(&recorder.evidence.lifecycle)
            || request.stream_sha256.len() != boundary.cuts.len()
            || request.registries.len() != boundary.cuts.len()
        {
            return Err(Error::msg(
                "instrumentation divergence: sealed global prefix or runtime witness",
            ));
        }
        for (index, cut) in boundary.cuts.iter().enumerate() {
            let owner = recorder
                .evidence
                .incarnations
                .iter()
                .find(|n| n.handle == cut.handle)
                .ok_or_else(|| Error::msg("instrumentation divergence: missing incarnation"))?;
            let streams = &owner.streams;
            if request.registries[index] != owner.registry {
                return Err(Error::msg(
                    "instrumentation divergence: registration prefix",
                ));
            }
            let hashes = &request.stream_sha256[index];
            if hashes.len() != 6
                || [
                    ("choice", &streams.choice),
                    ("wasi", &streams.wasi),
                    ("network", &streams.network),
                    ("storage", &streams.storage),
                    ("stdout", &streams.stdout),
                    ("stderr", &streams.stderr),
                ]
                .iter()
                .any(|(name, bytes)| hashes.get(*name) != Some(&inspection::digest(bytes)))
            {
                return Err(Error::msg(
                    "instrumentation divergence: incarnation prefix stream",
                ));
            }
        }
        Ok(())
    }
    #[cfg(test)]
    pub fn poll_call(&mut self, id: u64, subscriptions: &[Subscription]) -> Result<PollResult> {
        self.poll_owned_call(id, subscriptions, None)
    }
    pub fn poll_owned_call(
        &mut self,
        id: u64,
        subscriptions: &[Subscription],
        namespace: Option<&crate::namespace::Namespace>,
    ) -> Result<PollResult> {
        crate::namespace::call::admit_poll_frames(id, subscriptions).map_err(Error::new)?;
        self.reserve_transcript(crate::transcript::poll_reservation(subscriptions))?;
        let result = self.poll(subscriptions, namespace)?;
        self.append_record(&crate::transcript::poll_record(
            id,
            subscriptions,
            &result.events,
            result.errno,
            result.unsupported,
        ));
        Ok(result)
    }
    fn poll(
        &mut self,
        subscriptions: &[Subscription],
        namespace: Option<&crate::namespace::Namespace>,
    ) -> Result<PollResult> {
        use crate::local_model::PollError;
        let poll = crate::local_model::poll(
            self.now,
            self.config.epoch_nanos,
            self.config.limits.pending_events,
            subscriptions,
            namespace,
        )
        .map_err(|error| match error {
            PollError::Capacity(boundary) => Error::new(boundary),
            PollError::Descriptor => Error::msg("WASI invalid: local descriptor poll"),
        })?;
        if poll.now > self.now {
            self.advances += 1;
        }
        self.now = poll.now;
        Ok(PollResult {
            events: poll.events,
            errno: poll.errno,
            unsupported: poll.unsupported,
        })
    }
}
