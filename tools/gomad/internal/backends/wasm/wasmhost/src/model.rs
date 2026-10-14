use crate::abi::Operation;
use crate::protocol::{self, Reply};
use crate::transcript::{Event, Subscription};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::io::{BufRead, Write};
use wasmtime::{Error, Result};

const CHUNK: usize = 64 * 1024;
const MAX_TIME: u64 = i64::MAX as u64;
pub const CPU_CLOCK_UNSUPPORTED: &str = "CPU clocks outside stock WASI profile";

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub entropy_key: [u8; 32],
    pub working_directory_sha256: String,
    pub arguments: StringDeclaration,
    pub environment: StringDeclaration,
    pub seed: u64,
    pub mode: u8,
    pub limit: u64,
    pub diagnostics: bool,
    #[serde(default)]
    pub tape: Option<Vec<Decision>>,
    #[serde(default)]
    pub tape_chunks: u64,
    pub epoch_nanos: u64,
    pub read_step_nanos: u64,
    pub now: u64,
    pub limits: ModelLimits,
    #[serde(default)]
    pub digest_only: bool,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StringDeclaration {
    bytes: usize,
    count: usize,
    chunks: usize,
    #[serde(skip)]
    packed: Vec<u8>,
    #[serde(skip)]
    offsets: Vec<u32>,
}
impl StringDeclaration {
    fn validate(&self) -> bool {
        self.bytes <= 4 * 1024 * 1024
            && self.count <= self.bytes
            && self.chunks == self.bytes.div_ceil(CHUNK)
    }
    fn read(&mut self, input: &mut dyn BufRead, stream: &str) -> Result<()> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Chunk {
            #[serde(rename = "type")]
            kind: String,
            stream: String,
            sequence: usize,
            data_base64: String,
        }
        self.packed.reserve(self.bytes);
        for sequence in 0..self.chunks {
            let chunk: Chunk =
                serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
            if chunk.kind != "model_input"
                || chunk.stream != stream
                || chunk.sequence != sequence
                || chunk.data_base64.len() > CHUNK.div_ceil(3) * 4
            {
                return Err(Error::msg("invalid local model input chunk"));
            }
            let bytes = STANDARD.decode(chunk.data_base64).map_err(Error::new)?;
            if bytes.len() != CHUNK.min(self.bytes - self.packed.len()) {
                return Err(Error::msg("invalid local model input chunk length"));
            }
            self.packed.extend(bytes);
        }
        if self.packed.len() != self.bytes || self.bytes != 0 && self.packed.last() != Some(&0) {
            return Err(Error::msg("invalid local model string framing"));
        }
        let mut offset = 0;
        for value in self.packed.split_inclusive(|byte| *byte == 0) {
            std::str::from_utf8(&value[..value.len() - 1]).map_err(Error::new)?;
            self.offsets.push(offset as u32);
            offset += value.len();
        }
        if self.offsets.len() != self.count {
            return Err(Error::msg("invalid local model string count"));
        }
        Ok(())
    }
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelLimits {
    pub calls: u64,
    pub transcript_bytes: u64,
    pub pending_events: u64,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
pub struct Decision {
    ordinal: u64,
    kind: u8,
    site_offset: u64,
    site_missing: bool,
    rank_override: bool,
    alternatives: u32,
    selected: u32,
    data: u32,
    selected_identity: [u8; 32],
    alternative_set_digest: [u8; 32],
}
#[derive(Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct Divergence {
    ordinal: u64,
    reason: u8,
    expected: Option<Decision>,
    observed: Option<Decision>,
    tape_records: u64,
}
/// A remote reply that would exceed a retained-evidence limit; classified as
/// the deterministic `capacity` outcome rather than a protocol violation.
#[derive(Debug)]
pub struct RemoteCapacity(pub &'static str);
impl std::fmt::Display for RemoteCapacity {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(self.0)
    }
}
impl std::error::Error for RemoteCapacity {}

pub struct PollResult {
    pub events: Vec<Event>,
    pub errno: u16,
    pub unsupported: &'static str,
}

pub struct Model {
    config: Config,
    pub now: u64,
    pub advances: u64,
    pub transcript_bytes: u64,
    transcript_hash: Sha256,
    transcript_encoder: crate::transcript::Encoder,
    records: u64,
    cursor: usize,
    configured: bool,
    finished: bool,
    peak: u32,
    elapsed: u64,
    terminal: Vec<u8>,
    pending: HashMap<&'static str, Vec<u8>>,
    entropy_offset: u64,
    pub divergence: Option<Divergence>,
}
impl Config {
    pub fn validate(&self) -> Result<()> {
        if !(1..=3).contains(&self.mode)
            || !self.arguments.validate()
            || !self.environment.validate()
            || self.working_directory_sha256.len() != 64
            || !self
                .working_directory_sha256
                .bytes()
                .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
            || !(160..=1 << 30).contains(&self.limit)
            || self.mode == 1 && (self.tape.is_some() || self.tape_chunks != 0)
            || self.tape_chunks > (self.limit - 64) / 96
            || self.epoch_nanos > MAX_TIME
            || self.now > MAX_TIME - self.epoch_nanos
            || self.read_step_nanos > 1_000_000_000
            || self.limits.calls == 0
            || self.limits.calls > 1 << 28
            || self.limits.transcript_bytes == 0
            || self.limits.transcript_bytes > 1 << 30
            || self.limits.pending_events == 0
            || self.limits.pending_events > 4096
            || self.diagnostics && 64 + (self.limit - 64) / 96 * 96 > 64 * 1024 * 1024
        {
            return Err(Error::msg("invalid local model configuration"));
        }
        Ok(())
    }
    pub fn read_inputs(&mut self, input: &mut dyn BufRead) -> Result<()> {
        self.arguments.read(input, "arguments")?;
        self.environment.read(input, "environment")?;
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct End {
            #[serde(rename = "type")]
            kind: String,
        }
        let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
        if end.kind != "model_inputs_end" {
            return Err(Error::msg("invalid local model input end"));
        }
        let mut names = std::collections::HashSet::new();
        let mut pwd = None;
        for entry in self.environment.packed.split_inclusive(|byte| *byte == 0) {
            let value = std::str::from_utf8(&entry[..entry.len() - 1]).map_err(Error::new)?;
            let (name, content) = value
                .split_once('=')
                .ok_or_else(|| Error::msg("invalid local model environment"))?;
            if name.is_empty() || !names.insert(name) {
                return Err(Error::msg("invalid local model environment names"));
            }
            if name == "PWD" {
                pwd = Some(content);
            }
        }
        let pwd = pwd.ok_or_else(|| Error::msg("missing local model PWD"))?;
        let clean = pwd == "/"
            || pwd.starts_with('/')
                && !pwd.ends_with('/')
                && pwd[1..]
                    .split('/')
                    .all(|part| !part.is_empty() && part != "." && part != "..");
        if !clean
            || format!("{:x}", Sha256::digest(pwd.as_bytes())) != self.working_directory_sha256
        {
            return Err(Error::msg("invalid local model working directory identity"));
        }
        Ok(())
    }
    pub fn read_tape(&mut self, input: &mut dyn BufRead) -> Result<()> {
        if self.mode != 1 && self.tape.is_none() {
            self.tape = Some(Vec::new());
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Tape {
            #[serde(rename = "type")]
            kind: String,
            decisions: Vec<Decision>,
        }
        for _ in 0..self.tape_chunks {
            let chunk: Tape =
                serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
            if chunk.kind != "tape" || chunk.decisions.is_empty() {
                return Err(Error::msg("invalid tape chunk"));
            }
            let decisions = self.tape.get_or_insert_with(Vec::new);
            if (decisions.len() + chunk.decisions.len()) as u64 > (self.limit - 64) / 96 {
                return Err(Error::msg("tape decision capacity"));
            }
            decisions.extend(chunk.decisions);
        }
        Ok(())
    }
}
impl Model {
    pub fn new(config: Config) -> Self {
        Self {
            now: config.now,
            config,
            advances: 0,
            transcript_bytes: 0,
            transcript_hash: Sha256::new(),
            transcript_encoder: crate::transcript::Encoder::default(),
            records: 0,
            cursor: 0,
            configured: false,
            finished: false,
            peak: 0,
            elapsed: 0,
            terminal: Vec::new(),
            pending: HashMap::new(),
            entropy_offset: 0,
            divergence: None,
        }
    }
    /// Whether the cooperative runtime is configured and not yet finished.
    pub fn runtime_active(&self) -> bool {
        self.configured && !self.finished
    }
    pub fn call_limit(&self) -> u64 {
        self.config.limits.calls
    }
    #[cfg(test)]
    pub fn state(&self) -> Value {
        json!({"now":self.now,"advances":self.advances,"transcript_bytes":self.transcript_bytes})
    }
    pub fn request_state(&mut self) -> protocol::RequestState {
        self.close_clock_run();
        protocol::RequestState {
            advances: self.advances,
            now: self.now,
            transcript_bytes: self.transcript_bytes,
        }
    }
    pub fn metadata(&self, calls: u64) -> Value {
        let mut hash = self.transcript_hash.clone();
        if let Some(bytes) = self.transcript_encoder.pending() {
            hash.update(bytes);
        }
        json!({"now":self.now,"advances":self.advances,"calls":calls,"records":self.records,
            "cursor":self.cursor,"configured":self.configured,"finished":self.finished,
            "peak":self.peak,"elapsed":self.elapsed,"terminal_base64":STANDARD.encode(&self.terminal),
            "transcript_bytes":self.transcript_bytes,
            "transcript_sha256":format!("{:x}",hash.finalize())})
    }
    fn queue(&mut self, stream: &'static str, bytes: &[u8]) {
        self.pending
            .entry(stream)
            .or_default()
            .extend_from_slice(bytes);
    }
    pub fn flush(&mut self, output: &mut dyn Write, all: bool) -> Result<()> {
        #[cfg(feature = "diagnostic-profile")]
        let _flush = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Flush);
        if all {
            self.close_clock_run();
        }
        self.flush_streams(
            output,
            &["choice", "diagnostic", "transcript", "stdout", "stderr"],
            all,
        )
    }
    /// Sends full chunks of the named evidence streams, and with `all` every
    /// pending byte.
    fn flush_streams(
        &mut self,
        output: &mut dyn Write,
        streams: &[&'static str],
        all: bool,
    ) -> Result<()> {
        for &stream in streams {
            if let Some(bytes) = self.pending.get_mut(stream) {
                while bytes.len() >= CHUNK || all && !bytes.is_empty() {
                    let count = bytes.len().min(CHUNK);
                    protocol::send_frame(
                        output,
                        &json!({"type":"evidence","stream":stream,
                        "data_base64":STANDARD.encode(&bytes[..count])}),
                    )?;
                    bytes.drain(..count);
                }
            }
        }
        Ok(())
    }
    pub fn namespace_output(&mut self, stdout: &[u8], stderr: &[u8]) {
        self.queue("stdout", stdout);
        self.queue("stderr", stderr);
    }
    pub fn flush_output(&mut self, output: &mut dyn Write) -> Result<()> {
        self.flush_streams(output, &["stdout", "stderr"], true)
    }
    pub fn namespace_call(
        &mut self,
        id: u64,
        namespace: &mut crate::namespace::Namespace,
        call: crate::namespace_call::Call,
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
            .map_err(Error::new)?;
        let output = crate::namespace_call::output_bytes(reply.output.as_ref());
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
    pub fn transcript(&mut self, bytes: &[u8]) {
        #[cfg(feature = "diagnostic-profile")]
        let hash = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::TranscriptHash);
        self.transcript_hash.update(bytes);
        #[cfg(feature = "diagnostic-profile")]
        drop(hash);
        self.transcript_bytes += bytes.len() as u64;
        if !self.config.digest_only {
            self.queue("transcript", bytes);
        }
    }
    pub fn remote(&mut self, reply: &Reply, chunks: &[u8]) -> Result<()> {
        let state = reply
            .local_state
            .as_ref()
            .ok_or_else(|| Error::msg("missing remote model state"))?;
        if state.now < self.now
            || state.now > MAX_TIME - self.config.epoch_nanos
            || state.advances < self.advances
        {
            return Err(Error::msg("invalid remote clock state"));
        }
        let encoded = reply
            .transcript_base64
            .as_ref()
            .ok_or_else(|| Error::msg("missing remote transcript"))?;
        if !chunks.is_empty() && !encoded.is_empty() {
            return Err(Error::msg(
                "remote transcript cannot combine chunks and inline bytes",
            ));
        }
        let bytes = {
            #[cfg(feature = "diagnostic-profile")]
            let _encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
            if chunks.is_empty() {
                STANDARD.decode(encoded).map_err(Error::new)?
            } else {
                chunks.to_vec()
            }
        };
        if bytes.len() > protocol::FRAME_LIMIT {
            return Err(Error::msg("remote transcript capacity"));
        }
        if !self.config.digest_only
            && self.transcript_bytes + bytes.len() as u64 > self.config.limits.transcript_bytes
        {
            return Err(Error::new(RemoteCapacity(
                "WASI capacity: transcript-bytes",
            )));
        }
        self.close_clock_run();
        self.now = state.now;
        self.advances = state.advances;
        self.transcript(&bytes);
        Ok(())
    }
    fn reserve_transcript(&self, reservation: u64) -> Result<()> {
        let header = if self.transcript_bytes == 0 {
            crate::transcript::HEADER.len() as u64
        } else {
            0
        };
        if !self.config.digest_only
            && self.transcript_bytes + header + reservation > self.config.limits.transcript_bytes
        {
            return Err(Error::new(crate::namespace::Boundary {
                kind: "capacity",
                dimension: "transcript-bytes",
            }));
        }
        Ok(())
    }
    fn close_clock_run(&mut self) {
        if let Some(bytes) = self.transcript_encoder.close() {
            self.transcript_bytes -= crate::transcript::CLOCK_RUN_BYTES;
            self.transcript(&bytes);
        }
    }
    fn append_record(&mut self, bytes: &[u8]) {
        self.close_clock_run();
        if self.transcript_bytes == 0 {
            self.transcript(crate::transcript::HEADER);
        }
        self.transcript(bytes);
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
    pub fn random(&mut self, id: u64, length: usize) -> Result<Vec<u8>> {
        if length > 4 * 1024 * 1024 || self.entropy_offset.checked_add(length as u64).is_none() {
            return Err(Error::msg("WASI capacity: entropy-bytes"));
        }
        self.reserve_transcript(crate::transcript::ENTROPY_BYTES)?;
        let start = self.entropy_offset;
        let mut bytes = Vec::with_capacity(length);
        while bytes.len() < length {
            let mut hash = Sha256::new();
            hash.update(b"gomad3.wasi-entropy-sha256-counter/v1");
            hash.update(self.config.entropy_key);
            hash.update((self.entropy_offset / 32).to_le_bytes());
            let block = hash.finalize();
            let skip = (self.entropy_offset % 32) as usize;
            let count = (length - bytes.len()).min(32 - skip);
            bytes.extend_from_slice(&block[skip..skip + count]);
            self.entropy_offset += count as u64;
        }
        self.append_record(&crate::transcript::entropy_record(id, length as u32, start));
        Ok(bytes)
    }
    pub fn clock_read(&mut self, id: u64, clock_id: u32, precision: u64) -> Result<Option<u64>> {
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
                json!({"data_base64":STANDARD.encode(self.random(id,input["length"].as_u64().unwrap() as usize)?)})
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
                    input["clock_id"].as_u64().unwrap_or(u32::MAX as u64) as u32,
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
    pub fn runtime_typed(&mut self, op: Operation, mut data: Vec<u8>) -> Result<Vec<u8>> {
        self.close_clock_run();
        if op == Operation::RuntimeConfig {
            if self.configured || data.len() != 16 || data.iter().any(|b| *b != 0) {
                return Err(Error::msg("WASI invalid: runtime bootstrap"));
            }
            set32(&mut data, 0, 1);
            set32(&mut data, 4, self.config.diagnostics as u32);
            set64(&mut data, 8, self.config.seed);
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
                if data.len() < 32 || get32(&data, 0) != 1 {
                    return Err(Error::msg("WASI invalid: runtime decision header"));
                }
                let kind = get32(&data, 4);
                let flags = get32(&data, 16);
                let count = get32(&data, 20) as usize;
                let selected = get32(&data, 24) as usize;
                if !matches!(kind, 1 | 2)
                    || flags & !4 != 0
                    || !(2..=256).contains(&count)
                    || selected >= count
                    || data.len() != 32 + count * 32 + if self.config.diagnostics { 96 } else { 0 }
                {
                    return Err(Error::msg(
                        "WASI invalid: runtime decision alternatives or layout",
                    ));
                }
                let alternatives: Vec<[u8; 32]> = data[32..32 + count * 32]
                    .chunks_exact(32)
                    .map(|b| b.try_into().unwrap())
                    .collect();
                let mut ordered = alternatives.clone();
                ordered.sort();
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
                    hash.update(b"gomad3-choice-alternative-set/v1\0");
                    hash.update((count as u64).to_be_bytes());
                    for identity in &ordered {
                        hash.update(identity);
                    }
                    hash.finalize().into()
                };
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
                #[cfg(feature = "diagnostic-profile")]
                let encoding =
                    crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
                let mut record = [0u8; 96];
                set64(&mut record, 0, observed.ordinal);
                record[8] = observed.kind;
                record[9] = 1 | if observed.site_missing { 4 } else { 0 };
                set32(&mut record, 12, observed.alternatives);
                set32(&mut record, 16, observed.selected);
                set32(&mut record, 20, observed.data);
                set64(&mut record, 24, observed.site_offset);
                record[32..64].copy_from_slice(&observed.selected_identity);
                record[64..96].copy_from_slice(&observed.alternative_set_digest);
                #[cfg(feature = "diagnostic-profile")]
                drop(encoding);
                self.append(&record, &data[32 + count * 32..])?;
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
                    || (readiness >> 6) as u32 > count - u32::from(readiness & 2 != 0)
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
                self.terminal = data.clone();
                self.finished = true;
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
    fn append(&mut self, record: &[u8], diagnostic: &[u8]) -> Result<()> {
        if 64 + (self.records + 1) * 96 > self.config.limit {
            return Err(Error::msg("WASI capacity: runtime-choice-bytes"));
        }
        self.queue("choice", record);
        if self.config.diagnostics {
            self.queue("diagnostic", diagnostic);
        }
        self.records += 1;
        Ok(())
    }
    fn diverge(
        &mut self,
        reason: u8,
        expected: Option<Decision>,
        observed: Option<Decision>,
    ) -> Error {
        let names = [
            "",
            "kind",
            "site",
            "alternatives",
            "selected",
            "alternative_set",
            "tape_exhausted",
            "tape_unconsumed",
        ];
        self.divergence = Some(Divergence {
            ordinal: self.cursor as u64,
            reason,
            expected,
            observed,
            tape_records: self.config.tape.as_ref().unwrap().len() as u64,
        });
        Error::msg(format!(
            "WASM choice {}: {}",
            self.cursor, names[reason as usize]
        ))
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
        crate::namespace_call::admit_poll_frames(id, subscriptions).map_err(Error::new)?;
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
fn get32(data: &[u8], at: usize) -> u32 {
    u32::from_be_bytes(data[at..at + 4].try_into().unwrap())
}
fn get64(data: &[u8], at: usize) -> u64 {
    u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
}
fn set32(data: &mut [u8], at: usize, value: u32) {
    data[at..at + 4].copy_from_slice(&value.to_be_bytes());
}
fn set64(data: &mut [u8], at: usize, value: u64) {
    data[at..at + 8].copy_from_slice(&value.to_be_bytes());
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::hint::black_box;
    fn model() -> Model {
        let mut config: Config = serde_json::from_value(json!({"seed":7,"mode":1,"limit":4096,
            "diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}))
        .unwrap();
        config.validate().unwrap();
        config
            .read_tape(&mut std::io::Cursor::new(Vec::new()))
            .unwrap();
        Model::new(config)
    }
    fn namespace() -> crate::namespace::Namespace {
        use crate::namespace::{Image, Limits, Namespace};
        Namespace::new(Image {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: Vec::new(),
            working_directory: b"/".to_vec(),
            stdin: b"q".to_vec(),
            limits: Limits {
                files: 10,
                descriptors: 10,
                filesystem_bytes: 1024,
                output_bytes: 100000,
            },
            epoch_nanos: 1000,
        })
        .unwrap()
    }
    #[test]
    fn namespace_mixed_poll_preserves_order_and_only_advances_for_clock() {
        let mut m = model();
        let n = namespace();
        let fd = |userdata, fd, kind| Subscription {
            userdata,
            kind,
            fd,
            clock_id: 0,
            timeout: 0,
            precision: 0,
            flags: 0,
        };
        let clock = Subscription {
            userdata: 4,
            kind: 0,
            fd: 0,
            clock_id: 1,
            timeout: 10,
            precision: 0,
            flags: 0,
        };
        let result = m
            .poll_owned_call(
                1,
                &[fd(1, 0, 1), fd(2, 99, 1), fd(3, 1, 2), clock],
                Some(&n),
            )
            .unwrap();
        assert_eq!(result.errno, 0);
        assert_eq!(
            result
                .events
                .iter()
                .map(|e| (e.userdata, e.errno, e.kind, e.nbytes, e.flags))
                .collect::<Vec<_>>(),
            [(1, 0, 1, 1, 0), (2, 8, 1, 0, 0), (3, 0, 2, 0, 0)]
        );
        assert_eq!((m.now, m.advances), (1, 0));
        let result = m.poll_owned_call(2, &[clock], Some(&n)).unwrap();
        assert_eq!(result.events[0].userdata, 4);
        assert_eq!((m.now, m.advances), (11, 1));
        assert_eq!(m.transcript_bytes, 241);
    }
    #[test]
    fn namespace_output_flushes_at_chunk_and_before_remote_without_losing_partial() {
        let mut m = model();
        let mut output = Vec::new();
        m.namespace_output(&vec![b'x'; CHUNK - 1], b"err");
        m.flush(&mut output, false).unwrap();
        assert!(output.is_empty());
        m.namespace_output(b"yz", b"");
        m.flush(&mut output, false).unwrap();
        let lines: Vec<Value> = output
            .split(|b| *b == b'\n')
            .filter(|b| !b.is_empty())
            .map(|b| serde_json::from_slice(b).unwrap())
            .collect();
        assert_eq!(lines.len(), 1);
        assert_eq!(
            STANDARD
                .decode(lines[0]["data_base64"].as_str().unwrap())
                .unwrap()
                .len(),
            CHUNK
        );
        output.clear();
        m.flush_output(&mut output).unwrap();
        let lines: Vec<Value> = output
            .split(|b| *b == b'\n')
            .filter(|b| !b.is_empty())
            .map(|b| serde_json::from_slice(b).unwrap())
            .collect();
        assert_eq!(lines.len(), 2);
        assert_eq!(lines[0]["stream"], "stdout");
        assert_eq!(
            STANDARD
                .decode(lines[0]["data_base64"].as_str().unwrap())
                .unwrap(),
            b"z"
        );
        assert_eq!(lines[1]["stream"], "stderr");
        assert_eq!(
            STANDARD
                .decode(lines[1]["data_base64"].as_str().unwrap())
                .unwrap(),
            b"err"
        );
        output.clear();
        m.flush(&mut output, true).unwrap();
        assert!(output.is_empty());
    }
    #[test]
    fn namespace_capacity_refuses_before_mutation_and_preserves_completed_evidence() {
        use crate::namespace::Operation as N;
        let mut m = model();
        let mut n = namespace();
        let call = |data: &[u8]| {
            crate::namespace_call::Call::new(N::Write {
                fd: 1,
                data: data.to_vec(),
                offset: None,
            })
        };
        m.namespace_call(1, &mut n, call(b"ab")).unwrap();
        let (stdout, stderr) = n.take_output();
        m.namespace_output(&stdout, &stderr);
        m.config.limits.transcript_bytes = m.transcript_bytes;
        let before = n.canonical_state();
        let error = m.namespace_call(2, &mut n, call(b"x")).unwrap_err();
        assert_eq!(
            error
                .downcast_ref::<crate::namespace::Boundary>()
                .unwrap()
                .dimension,
            "transcript-bytes"
        );
        assert_eq!(n.canonical_state(), before);
        let mut output = Vec::new();
        m.flush(&mut output, true).unwrap();
        let lines: Vec<Value> = output
            .split(|b| *b == b'\n')
            .filter(|b| !b.is_empty())
            .map(|b| serde_json::from_slice(b).unwrap())
            .collect();
        assert_eq!(
            STANDARD
                .decode(
                    lines.iter().find(|v| v["stream"] == "stdout").unwrap()["data_base64"]
                        .as_str()
                        .unwrap()
                )
                .unwrap(),
            b"ab"
        );
        let transcript = STANDARD
            .decode(
                lines.iter().find(|v| v["stream"] == "transcript").unwrap()["data_base64"]
                    .as_str()
                    .unwrap(),
            )
            .unwrap();
        assert_eq!(&transcript[..8], crate::transcript::HEADER);
        assert_eq!(transcript[8], 8);
        assert_eq!(&transcript[12..20], &1u64.to_le_bytes());
    }
    #[test]
    fn namespace_unsupported_exact_bounds_preserve_message_bytes_and_ids() {
        use crate::namespace::{Image, Limits, Namespace, Open, Operation as N};
        for (operation, limit, message) in [
            (
                N::SetFlags { fd: 3, flags: 2 },
                75,
                "descriptor flags outside volatile profile",
            ),
            (
                N::Open(Open {
                    fd: 3,
                    path: b"x".to_vec(),
                    dirflags: 0,
                    oflags: 0,
                    base: 2,
                    inheriting: 0,
                    flags: 2,
                }),
                96,
                "open flags outside volatile profile",
            ),
            (
                N::Readlink {
                    fd: 3,
                    path: b"x".to_vec(),
                    length: 0,
                },
                92,
                "operation outside stock WASI profile: path_readlink",
            ),
        ] {
            let mut n = Namespace::new(Image {
                entries: Vec::new(),
                writable_directories: Vec::new(),
                preopens: vec![b"/".to_vec()],
                working_directory: b"/".to_vec(),
                stdin: Vec::new(),
                limits: Limits {
                    files: 10,
                    descriptors: 10,
                    filesystem_bytes: 1024,
                    output_bytes: 1024,
                },
                epoch_nanos: 1000,
            })
            .unwrap();
            let before = n.canonical_state();
            let mut m = model();
            m.config.limits.transcript_bytes = limit - 1;
            let error = m
                .namespace_call(
                    1,
                    &mut n,
                    crate::namespace_call::Call::new(operation.clone()),
                )
                .unwrap_err();
            assert_eq!(
                error
                    .downcast_ref::<crate::namespace::Boundary>()
                    .unwrap()
                    .dimension,
                "transcript-bytes"
            );
            assert_eq!(m.transcript_bytes, 0);
            assert_eq!(n.canonical_state(), before);
            m.config.limits.transcript_bytes = limit;
            let reply = m
                .namespace_call(1, &mut n, crate::namespace_call::Call::new(operation))
                .unwrap();
            assert_eq!(reply.unsupported.as_deref(), Some(message));
            assert_eq!(n.canonical_state(), before);
            assert_eq!(m.transcript_bytes, limit);
            let mut output = Vec::new();
            m.flush(&mut output, true).unwrap();
            let frame: Value = serde_json::from_slice(output.strip_suffix(b"\n").unwrap()).unwrap();
            let bytes = STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap();
            assert_eq!(&bytes[12..20], &1u64.to_le_bytes());
            assert_eq!(bytes[9], 1);
            assert_eq!(&bytes[bytes.len() - message.len()..], message.as_bytes());
        }
    }
    #[test]
    fn retired_network_configuration_is_rejected_by_name() {
        let error = serde_json::from_value::<Config>(json!({"seed":7,"mode":1,"limit":4096,
            "diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16},
            "network":{"seed":7,"max_delay_nanos":10}}))
        .err()
        .unwrap();
        assert!(error.to_string().contains("unknown field `network`"));
    }
    #[test]
    fn fragmented_inputs_accept_full_domain_and_reject_malformed_carriage() {
        let mut m = model();
        let argument = vec![1u8; 4 * 1024 * 1024 - 1];
        let mut packed = argument.clone();
        packed.push(0);
        m.config.arguments.bytes = packed.len();
        m.config.arguments.count = 1;
        m.config.arguments.chunks = 64;
        let mut wire = Vec::new();
        for (sequence, bytes) in packed.chunks(CHUNK).enumerate() {
            protocol::send_frame(&mut wire,&json!({"type":"model_input","stream":"arguments","sequence":sequence,"data_base64":STANDARD.encode(bytes)})).unwrap();
        }
        wire.extend_from_slice(b"{\"type\":\"model_input\",\"stream\":\"environment\",\"sequence\":0,\"data_base64\":\"UFdEPS8A\"}\n{\"type\":\"model_inputs_end\"}\n");
        m.config.validate().unwrap();
        m.config
            .read_inputs(&mut std::io::Cursor::new(wire))
            .unwrap();
        assert_eq!(m.config.arguments.packed, packed);
        let literal=b"{\"type\":\"model_input\",\"stream\":\"environment\",\"sequence\":0,\"data_base64\":\"UFdEPS8A\"}\n{\"type\":\"model_inputs_end\"}\n";
        for broken in [
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("\"sequence\":0", "\"sequence\":1"),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("environment", "arguments"),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("UFdEPS8A", "UFdEPS8="),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("UFdEPS8A", "/1dEPS8A"),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("model_inputs_end", "model_input"),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("\"sequence\":0", "\"sequence\":0,\"extra\":0"),
            String::from_utf8(literal.to_vec()).unwrap().replace(
                "\"type\":\"model_inputs_end\"",
                "\"type\":\"model_inputs_end\",\"extra\":0",
            ),
            String::from_utf8(literal.to_vec())
                .unwrap()
                .replace("UFdEPS8A", "UFdEPWEA"),
            "".to_string(),
        ] {
            assert!(
                model()
                    .config
                    .read_inputs(&mut std::io::Cursor::new(broken))
                    .is_err()
            );
        }
        let mut bad = model().config;
        bad.environment.count = 2;
        assert!(bad.read_inputs(&mut std::io::Cursor::new(literal)).is_err());
        bad = model().config;
        bad.arguments.bytes = 4 * 1024 * 1024 + 1;
        assert!(bad.validate().is_err());
        bad = model().config;
        bad.environment.chunks = 2;
        assert!(bad.validate().is_err());
        for packed in [
            b"PWD=/\0PWD=/\0".as_slice(),
            b"PWD=/\0X\0",
            b"PWD=/\0=x\0",
            b"PWD=/./\0",
        ] {
            bad = model().config;
            bad.environment.bytes = packed.len();
            bad.environment.count = packed.iter().filter(|byte| **byte == 0).count();
            let mut wire = Vec::new();
            protocol::send_frame(&mut wire,&json!({"type":"model_input","stream":"environment","sequence":0,"data_base64":STANDARD.encode(packed)})).unwrap();
            wire.extend_from_slice(b"{\"type\":\"model_inputs_end\"}\n");
            assert!(bad.read_inputs(&mut std::io::Cursor::new(wire)).is_err());
        }
    }
    #[test]
    fn entropy_boundaries_and_literal_transcript_preserve_offsets() {
        let mut m = model();
        m.entropy_offset = 30;
        assert_eq!(
            m.random(1, 6).unwrap(),
            [0x48, 0x47, 0x6a, 0x64, 0xd7, 0x20]
        );
        assert_eq!(
            m.pending["transcript"],
            b"GMWT\x01\0\0\0\x07\0\0\0\x01\0\0\0\0\0\0\0\x0c\0\0\0\x06\0\0\0\x1e\0\0\0\0\0\0\0"
        );
        assert!(m.random(2, 4 * 1024 * 1024 + 1).is_err());
        assert_eq!(m.entropy_offset, 36);
        m.entropy_offset = u64::MAX;
        assert!(m.random(2, 1).is_err());
        assert_eq!(m.entropy_offset, u64::MAX);
        assert!(m.random(2, 0).unwrap().is_empty());
        m = model();
        m.config.limits.transcript_bytes = 35;
        assert!(m.random(1, 1).is_err());
        assert_eq!(m.entropy_offset, 0);
        assert_eq!(m.transcript_bytes, 0);
        m.config.limits.transcript_bytes = 36;
        assert!(m.random(1, 1).is_ok());
    }
    #[test]
    fn local_yield_records_versioned_binary_bytes() {
        let mut m = model();
        m.handle(1, "sched_yield", &json!({})).unwrap();
        assert_eq!(
            m.pending["transcript"],
            b"GMWT\x01\0\0\0\x02\0\0\0\x01\0\0\0\0\0\0\0\0\0\0\0"
        );
    }
    #[test]
    fn local_entropy_uses_shared_sha256_counter_golden() {
        let mut m = model();
        let reply = m.handle(1, "random_get", &json!({"length":32})).unwrap();
        assert!(
            reply.is_some(),
            "captured entropy must be answered by local model"
        );
        let bytes = STANDARD
            .decode(reply.unwrap().output["data_base64"].as_str().unwrap())
            .unwrap();
        assert_eq!(
            bytes,
            [
                0xc0, 0x72, 0x8b, 0x91, 0x20, 0xad, 0xdf, 0x2a, 0x1f, 0x40, 0x90, 0xcb, 0x91, 0x6e,
                0xd4, 0x59, 0xe1, 0xac, 0xff, 0x62, 0x10, 0xa7, 0xff, 0xc7, 0x88, 0x4a, 0xa6, 0xd4,
                0xec, 0xa7, 0x48, 0x47
            ]
        );
        assert_eq!(
            m.random(2, 32).unwrap(),
            [
                0x6a, 0x64, 0xd7, 0x20, 0x6e, 0x01, 0x72, 0xd7, 0xa2, 0x57, 0xe0, 0xd0, 0xe7, 0x67,
                0x46, 0x67, 0xf6, 0xcd, 0x8f, 0x1a, 0xf5, 0x81, 0x46, 0x0a, 0xcb, 0xae, 0x65, 0x52,
                0x3c, 0x4c, 0x10, 0x8e
            ]
        );
        assert!(m.random(3, 0).unwrap().is_empty());
        assert_eq!(m.entropy_offset, 64);
    }
    #[test]
    fn consecutive_clock_reads_fold_in_frozen_model() {
        let mut m = model();
        for id in 1..=3 {
            assert_eq!(m.clock_read(id, 1, 7).unwrap(), Some(1));
        }
        m.flush(&mut Vec::new(), true).unwrap();
        assert_eq!(m.transcript_bytes, 52);
        assert_eq!(m.metadata(3)["transcript_sha256"], format!("{:x}", Sha256::digest(b"GMWT\x01\0\0\0\x21\0\0\0\x01\0\0\0\0\0\0\0\x1c\0\0\0\x01\0\0\0\x07\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0\x03\0\0\0\0\0\0\0")));
    }
    #[test]
    fn pending_clock_inspection_preserves_run_and_digest_binds_every_key() {
        let mut reference = None;
        for (count, clock_id, precision, now) in [
            (3, 1, 7, 1),
            (2, 1, 7, 1),
            (3, 0, 7, 1),
            (3, 1, 8, 1),
            (3, 1, 7, 2),
        ] {
            let mut m = model();
            m.now = now;
            for id in 1..=count {
                m.clock_read(id, clock_id, precision).unwrap();
                let _snapshot = m.metadata(id);
            }
            assert_eq!(m.transcript_bytes, 52);
            let before = m.metadata(count)["transcript_sha256"].clone();
            m.flush(&mut Vec::new(), true).unwrap();
            let digest = m.metadata(count)["transcript_sha256"].clone();
            assert_eq!(digest, before);
            if let Some(reference) = &reference {
                assert_ne!(&digest, reference);
            } else {
                reference = Some(digest);
            }
        }
    }
    fn decision() -> Vec<u8> {
        let mut bytes = vec![0; 96];
        set32(&mut bytes, 0, 1);
        set32(&mut bytes, 4, 1);
        set64(&mut bytes, 8, 42);
        set32(&mut bytes, 20, 2);
        set32(&mut bytes, 24, 1);
        bytes[32..64].fill(2);
        bytes[64..96].fill(1);
        bytes
    }
    fn bootstrap(model: &mut Model) {
        model.runtime("runtime_config", vec![0; 16]).unwrap();
    }
    #[cfg(feature = "diagnostic-profile")]
    #[test]
    fn diagnostic_model_hooks_measure_actual_hash_encoding_and_flush() {
        let _session = crate::diagnostic::Session::start();
        let mut m = model();
        bootstrap(&mut m);
        m.runtime("runtime_decision", decision()).unwrap();
        m.handle(3, "clock_time_get", &json!({"clock_id":1,"precision":0}))
            .unwrap();
        m.flush(&mut Vec::new(), true).unwrap();
        let report = crate::diagnostic::snapshot();
        assert_eq!(report["nested"]["alternative_hash"]["count"], 1);
        assert_eq!(report["nested"]["transcript_hash"]["count"], 2);
        assert_eq!(report["nested"]["flush"]["count"], 1);
        assert_eq!(report["nested"]["encoding"]["count"], 3);
        assert_eq!(m.metadata(3)["now"], 1);
        assert_eq!(m.records, 1);
    }
    #[cfg(feature = "diagnostic-profile")]
    #[test]
    #[ignore = "instrumentation overhead calibration only; not a guest or performance gate"]
    fn diagnostic_calibration_preserves_model_bytes_without_timing_feedback() {
        use std::time::Instant;
        let run = |active: bool| {
            let mut m = model();
            m.config.limit = 16 * 1024 * 1024;
            m.config.limits.transcript_bytes = 16 * 1024 * 1024;
            let session = active.then(crate::diagnostic::Session::start);
            bootstrap(&mut m);
            let before = crate::metrics::diagnostic_process_usage();
            let start = Instant::now();
            let mut output = Vec::new();
            let mut id = 1;
            for _ in 0..10_000 {
                m.runtime("runtime_decision", decision()).unwrap();
                id += 1;
                for _ in 0..5 {
                    id += 1;
                    m.handle(id, "clock_time_get", &json!({"clock_id":1,"precision":0}))
                        .unwrap();
                    m.flush(&mut output, false).unwrap();
                }
            }
            let wall = crate::metrics::elapsed(start);
            let after = crate::metrics::diagnostic_process_usage();
            m.flush(&mut output, true).unwrap();
            let metadata = m.metadata(id);
            let profile = active.then(crate::diagnostic::snapshot);
            let cpu = match (before, after) {
                (Some((u0, s0, _)), Some((u1, s1, rss))) => {
                    json!({"user_nanos":u1-u0,"system_nanos":s1-s0,"peak_rss_bytes_high_water":rss})
                }
                _ => Value::Null,
            };
            drop(session);
            (
                output,
                metadata,
                json!({"wall_nanos":wall,"cpu":cpu,"profile":profile}),
            )
        };
        for replicate in 1..=3 {
            let (control_bytes, control_metadata, control) = run(false);
            let (sampled_bytes, sampled_metadata, sampled) = run(true);
            assert_eq!(control_bytes, sampled_bytes);
            assert_eq!(control_metadata, sampled_metadata);
            assert_eq!(
                sampled["profile"]["nested"]["alternative_hash"]["count"],
                10_000
            );
            assert_eq!(
                sampled["profile"]["nested"]["transcript_hash"]["count"],
                10_001
            );
            println!(
                "{}",
                json!({"diagnostic_only":true,"replicate":replicate,"decisions":10_000,
                "clock_reads":50_000,"canonical_equal":true,"control":control,"sampled":sampled,
                "control_scope":"same diagnostic build with no active session; cfg hooks still perform inactive checks",
                "excludes":"Wasmtime/JIT guest instructions, import trampolines, IPC callbacks, final flush and byte comparison",
                "qualification":false})
            );
        }
    }
    #[test]
    fn typed_clock_preserves_json_reference_and_literal_canonical_bytes() {
        for (input, clock_id, timestamp, literal_input) in [
            (
                json!({"clock_id":0,"precision":0}),
                Some(0),
                Some(1001),
                "{\"clock_id\":0,\"precision\":0}",
            ),
            (
                json!({"clock_id":1,"precision":u64::MAX}),
                Some(1),
                Some(1),
                "{\"clock_id\":1,\"precision\":18446744073709551615}",
            ),
            (
                json!({"clock_id":2,"precision":0}),
                Some(2),
                None,
                "{\"clock_id\":2,\"precision\":0}",
            ),
            (
                json!({"clock_id":u32::MAX,"precision":0}),
                Some(u32::MAX as u64),
                None,
                "{\"clock_id\":4294967295,\"precision\":0}",
            ),
            (
                json!({"clock_id":"1","precision":0}),
                None,
                None,
                "{\"clock_id\":\"1\",\"precision\":0}",
            ),
            (json!({"precision":0}), None, None, "{\"precision\":0}"),
            (
                json!({"clock_id":-1,"precision":0}),
                None,
                None,
                "{\"clock_id\":-1,\"precision\":0}",
            ),
        ] {
            let mut typed = model();
            let mut reference = model();
            typed.config.read_step_nanos = 3;
            reference.config.read_step_nanos = 3;
            let call = format!(
                "{{\"type\":\"call\",\"id\":7,\"op\":\"clock_time_get\",\"input\":{literal_input}}}\n"
            );
            assert_eq!(
                typed.clock_time_get(7, clock_id, call.as_bytes()).unwrap(),
                timestamp
            );
            let reply = reference
                .handle(7, "clock_time_get", &input)
                .unwrap()
                .unwrap();
            assert_eq!(reply.unsupported, timestamp.is_none());
            typed.close_clock_run();
            reference.close_clock_run();
            let mut literal = b"GMWT\x01\0\0\0".to_vec();
            literal.extend_from_slice(&[
                if timestamp.is_some() { 33 } else { 1 },
                timestamp.is_none() as u8,
                0,
                0,
                7,
                0,
                0,
                0,
                0,
                0,
                0,
                0,
            ]);
            literal.extend_from_slice(
                &(if timestamp.is_some() { 28_u32 } else { 53_u32 }).to_le_bytes(),
            );
            literal.extend_from_slice(&(clock_id.unwrap_or(u32::MAX as u64) as u32).to_le_bytes());
            literal.extend_from_slice(&input["precision"].as_u64().unwrap().to_le_bytes());
            if timestamp.is_some() {
                literal.extend_from_slice(b"\x01\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0");
            } else {
                literal.extend_from_slice(b"\x25\0\0\0CPU clocks outside stock WASI profile");
            }
            assert_eq!(typed.pending["transcript"], literal);
            assert_eq!(typed.now, if timestamp.is_some() { 4 } else { 1 });
            assert_eq!(typed.pending, reference.pending);
            assert_eq!(typed.metadata(7), reference.metadata(7));
        }
    }
    #[test]
    fn typed_clock_reservation_digest_and_read_step_failures_preserve_state() {
        let call = b"{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":0}}\n";
        for (digest, capacity, accepted) in [(false, 51, false), (false, 52, true), (true, 1, true)]
        {
            let mut typed = model();
            typed.config.digest_only = digest;
            typed.config.limits.transcript_bytes = capacity;
            let result = typed.clock_time_get(1, Some(1), call);
            assert_eq!(result.is_ok(), accepted);
            if accepted {
                assert_eq!(result.unwrap(), Some(1));
                assert!(typed.transcript_bytes > 0);
                assert_eq!(typed.pending.contains_key("transcript"), !digest);
            } else {
                assert_eq!(
                    result.unwrap_err().to_string(),
                    "WASI capacity: transcript-bytes"
                );
                assert_eq!(typed.transcript_bytes, 0);
            }
            assert_eq!((typed.now, typed.advances), (1, 0));
        }
        let mut typed = model();
        typed.now = MAX_TIME - typed.config.epoch_nanos;
        typed.config.read_step_nanos = 1;
        let before = typed.now;
        assert_eq!(
            typed
                .clock_time_get(1, Some(1), call)
                .unwrap_err()
                .to_string(),
            "WASI capacity: clock-nanoseconds"
        );
        assert_eq!(typed.now, before);
        assert_eq!(typed.transcript_bytes, 0);
        assert!(typed.pending.is_empty());
        let unsupported_call = b"{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":2,\"precision\":0}}\n";
        assert_eq!(
            typed.clock_time_get(2, Some(2), unsupported_call).unwrap(),
            None
        );
        assert_eq!(typed.now, before);
    }
    #[test]
    #[ignore = "diagnostic profiling only; not a qualification or performance gate"]
    fn profile_local_import_stages_against_json_reference() {
        use crate::engine::{Limits, State};
        use std::{
            sync::{Arc, Mutex},
            time::Instant,
        };
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
        fn configured(diagnostics: bool) -> Model {
            let mut model = model();
            model.config.limit = 64 * 1024 * 1024;
            model.config.diagnostics = diagnostics;
            model.config.read_step_nanos = 1;
            model.config.limits.calls = 1 << 28;
            model.config.limits.transcript_bytes = 64 * 1024 * 1024;
            model.config.validate().unwrap();
            model
        }
        fn timed(mut body: impl FnMut(usize), iterations: usize) -> u64 {
            let started = Instant::now();
            for index in 0..iterations {
                body(black_box(index));
            }
            crate::metrics::elapsed(started)
        }
        fn sample() -> crate::metrics::Metrics {
            let mut metrics = crate::metrics::Metrics::default();
            metrics.sample_process();
            metrics
        }
        for (alternatives, diagnostics, iterations) in
            [(2, false, 100_000), (32, false, 50_000), (32, true, 50_000)]
        {
            let mut request = vec![0; 32 + alternatives * 32 + if diagnostics { 96 } else { 0 }];
            set32(&mut request, 0, 1);
            set32(&mut request, 4, 1);
            set64(&mut request, 8, 42);
            set32(&mut request, 20, alternatives as u32);
            set32(&mut request, 24, alternatives as u32 - 1);
            for (index, identity) in request[32..32 + alternatives * 32]
                .chunks_exact_mut(32)
                .enumerate()
            {
                identity.fill((alternatives - index) as u8);
            }
            let ordered: Vec<[u8; 32]> =
                (1..=alternatives).map(|index| [index as u8; 32]).collect();
            let baseline_nanos = timed(
                |index| {
                    black_box(index);
                },
                iterations,
            );
            let copy_nanos = timed(
                |_| {
                    black_box(black_box(&request).clone());
                },
                iterations,
            );
            let format_nanos = timed(
                |_| {
                    black_box(format!("runtime_{}", black_box("decision")));
                },
                iterations,
            );
            let base64_roundtrip_nanos = timed(
                |_| {
                    let encoded = STANDARD.encode(black_box(&request));
                    black_box(STANDARD.decode(encoded).unwrap());
                },
                iterations,
            );
            let clone_sort_nanos = timed(
                |_| {
                    let alternatives: Vec<[u8; 32]> = black_box(&request)
                        [32..32 + alternatives * 32]
                        .chunks_exact(32)
                        .map(|identity| identity.try_into().unwrap())
                        .collect();
                    let mut ordered = alternatives.clone();
                    ordered.sort();
                    assert!(
                        !ordered.contains(&[0; 32])
                            && !ordered.windows(2).any(|pair| pair[0] == pair[1])
                    );
                    black_box(
                        ordered
                            .iter()
                            .position(|identity| *identity == alternatives[alternatives.len() - 1])
                            .unwrap(),
                    );
                },
                iterations,
            );
            let alternative_hash_nanos = timed(
                |_| {
                    let mut hash = Sha256::new();
                    hash.update(b"gomad3-choice-alternative-set/v1\0");
                    hash.update((alternatives as u64).to_be_bytes());
                    for identity in black_box(&ordered) {
                        hash.update(identity);
                    }
                    black_box(hash.finalize());
                },
                iterations,
            );
            let mut core = configured(diagnostics);
            bootstrap(&mut core);
            let core_decision_nanos = timed(
                |_| {
                    black_box(core.runtime("runtime_decision", request.clone()).unwrap());
                },
                iterations,
            );
            drop(core);
            let mut core = configured(diagnostics);
            bootstrap(&mut core);
            let mut observation = vec![0; if diagnostics { 192 } else { 96 }];
            observation[8] = 3;
            observation[9] = 2;
            observation[11] = 1;
            set32(&mut observation, 12, 1);
            let core_observation_nanos = timed(
                |index| {
                    set64(&mut observation, 0, index as u64);
                    black_box(
                        core.runtime("runtime_observation", observation.clone())
                            .unwrap(),
                    );
                },
                iterations,
            );
            drop(core);
            let mut pending = configured(diagnostics);
            pending.queue("choice", &[0; 96]);
            pending.queue("transcript", &[0; 96]);
            if diagnostics {
                pending.queue("diagnostic", &[0; 96]);
            }
            let pending_flush_checks_nanos = timed(
                |_| {
                    black_box(&mut pending)
                        .flush(&mut std::io::sink(), false)
                        .unwrap();
                },
                iterations,
            );
            let clock = json!({"clock_id":1,"precision":0});
            let mut clocks = configured(diagnostics);
            let clock_handle_nanos = timed(
                |index| {
                    black_box(
                        clocks
                            .handle(index as u64 + 1, "clock_time_get", &clock)
                            .unwrap(),
                    );
                },
                iterations,
            );
            drop(clocks);
            let mut canonical = None;
            let mut paths = Vec::new();
            for (json_reference, typed_clock) in [(false, false), (true, false), (false, true)] {
                let capture = Capture(Arc::new(Mutex::new(Vec::new())));
                let mut state = State {
                    memory: None,
                    input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
                    output: Box::new(capture.clone()),
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
                    model: Some(configured(diagnostics)),
                    namespace: None,
                    request_scratch: Vec::new(),
                    write_scratch: protocol::WriteBuffers::default(),
                    network: None,
                };
                let invoke = |state: &mut State, op: &str, bytes: Vec<u8>| {
                    let name = format!("runtime_{op}");
                    if json_reference {
                        let reply = state
                            .callback(&name, json!({"data_base64":STANDARD.encode(bytes)}))
                            .unwrap();
                        black_box(
                            STANDARD
                                .decode(reply.output["data_base64"].as_str().unwrap())
                                .unwrap(),
                        );
                    } else {
                        black_box(state.runtime(&name, bytes).unwrap());
                    }
                };
                invoke(&mut state, "config", vec![0; 16]);
                let before = sample();
                let wall_nanos = timed(
                    |index| {
                        invoke(&mut state, "decision", request.clone());
                        set64(&mut observation, 0, index as u64 * 2 + 1);
                        invoke(&mut state, "observation", observation.clone());
                        if typed_clock {
                            black_box(state.clock_time_get(1, 0).unwrap());
                        } else {
                            black_box(state.callback("clock_time_get", clock.clone()).unwrap());
                        }
                    },
                    iterations,
                );
                let after = sample();
                let mut finish = vec![0; 16];
                set32(&mut finish, 0, 1);
                set32(&mut finish, 4, 5);
                set64(&mut finish, 8, iterations as u64);
                invoke(&mut state, "finish", finish);
                let model = state.model.as_mut().unwrap();
                model.flush(&mut state.output, true).unwrap();
                let metadata = model.metadata(state.completed_calls);
                let encoded = capture.0.lock().unwrap();
                let mut streams: HashMap<String, Vec<u8>> = HashMap::new();
                for frame in encoded
                    .split(|byte| *byte == b'\n')
                    .filter(|line| !line.is_empty())
                {
                    let frame: Value = serde_json::from_slice(frame).unwrap();
                    streams
                        .entry(frame["stream"].as_str().unwrap().to_owned())
                        .or_default()
                        .extend(
                            STANDARD
                                .decode(frame["data_base64"].as_str().unwrap())
                                .unwrap(),
                        );
                }
                let evidence_bytes: usize = streams.values().map(Vec::len).sum();
                if let Some((reference_streams, reference_metadata)) = &canonical {
                    assert_eq!(&streams, reference_streams);
                    assert_eq!(&metadata, reference_metadata);
                } else {
                    canonical = Some((streams, metadata));
                }
                paths.push(json!({"path":if typed_clock {"binary_runtime_with_typed_clock"} else if json_reference {"counterfactual_json_runtime_reference"} else {"binary_runtime_with_json_clock"}, "mixed_imports":iterations * 3,"wall_nanos":wall_nanos,"user_nanos":after.process_user_nanos.saturating_sub(before.process_user_nanos),"system_nanos":after.process_system_nanos.saturating_sub(before.process_system_nanos),"resource_available":after.resource_usage_available,"peak_rss_bytes":after.peak_rss_bytes,"evidence_bytes":evidence_bytes}));
            }
            eprintln!(
                "{}",
                json!({"diagnostic_only":true,"excludes":"Wasmtime guest instructions/import trampoline, actual guest runtime loop, parent callbacks, final flush and evidence comparison","alternatives":alternatives,"diagnostics":diagnostics,"iterations":iterations,"stages_nanos":{"baseline_loop":baseline_nanos,"one_request_vec_copy":copy_nanos,"runtime_op_format":format_nanos,"base64_encode_decode_counterfactual":base64_roundtrip_nanos,"alternatives_clone_sort_validate_rank":clone_sort_nanos,"alternative_set_sha256":alternative_hash_nanos,"core_decision_with_request_copy":core_decision_nanos,"core_observation_with_request_copy":core_observation_nanos,"underfull_flush_checks":pending_flush_checks_nanos,"current_clock_handle_json_no_flush":clock_handle_nanos},"paths":paths,"canonical_concatenated_streams_and_metadata_equal":true})
            );
        }
    }
    #[test]
    fn canonical_record_and_forced_rank_preserve_identities() {
        let mut recorded = model();
        bootstrap(&mut recorded);
        recorded.runtime("runtime_decision", decision()).unwrap();
        let bytes = &recorded.pending["choice"];
        assert_eq!(bytes.len(), 96);
        assert_eq!(get64(bytes, 0), 0);
        assert_eq!(get32(bytes, 16), 0);
        assert_eq!(&bytes[32..64], &[1; 32]);
        let mut expected = Decision {
            ordinal: 0,
            kind: 1,
            site_offset: 42,
            site_missing: false,
            rank_override: true,
            alternatives: 2,
            selected: 1,
            data: 0,
            selected_identity: [0; 32],
            alternative_set_digest: bytes[64..].try_into().unwrap(),
        };
        let mut replay = model();
        replay.config.mode = 2;
        replay.config.tape = Some(vec![expected.clone()]);
        bootstrap(&mut replay);
        let answer = replay.runtime("runtime_decision", decision()).unwrap();
        assert_eq!(get32(&answer, 24), 0);
        assert_eq!(get32(&replay.pending["choice"], 16), 1);
        expected.kind = 2;
        let mut mismatch = model();
        mismatch.config.mode = 2;
        mismatch.config.tape = Some(vec![expected]);
        bootstrap(&mut mismatch);
        assert!(mismatch.runtime("runtime_decision", decision()).is_err());
        assert_eq!(mismatch.divergence.as_ref().unwrap().reason, 1);
        assert_eq!(mismatch.records, 0);
    }
    #[test]
    fn clock_idle_poll_and_remote_share_one_time_and_canonical_transcript() {
        let mut m = model();
        bootstrap(&mut m);
        let clock = json!({"clock_id":1,"precision":0});
        assert_eq!(
            m.handle(2, "clock_time_get", &clock)
                .unwrap()
                .unwrap()
                .output["timestamp"],
            1
        );
        let mut idle = vec![0; 16];
        set32(&mut idle, 0, 1);
        set64(&mut idle, 8, 10);
        m.runtime("runtime_idle", idle).unwrap();
        let poll = json!({"subscriptions":[{"userdata":7,"type":0,"clock_id":1,"timeout":5,"precision":0,"flags":0}]});
        m.handle(4, "poll_oneoff", &poll).unwrap();
        assert_eq!((m.now, m.advances), (15, 2));
        let transcript = m.pending["transcript"].clone();
        assert_eq!(&transcript[..52], b"GMWT\x01\0\0\0\x21\0\0\0\x02\0\0\0\0\0\0\0\x1c\0\0\0\x01\0\0\0\0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0");
        assert_eq!(transcript[52], 32);
        assert_eq!(
            &transcript[transcript.len() - 21..],
            b"\x07\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"
        );
        let reply:Reply=serde_json::from_value(json!({"type":"reply","id":5,"errno":0,"output":{},
            "local_state":{"now":20,"advances":3},"transcript_base64":STANDARD.encode(b"\x0d\0\0\0\x05\0\0\0\0\0\0\0\x04\0\0\0\x03\0\0\0")})).unwrap();
        m.remote(&reply, &[]).unwrap();
        assert_eq!((m.now, m.advances), (20, 3));
        assert_eq!(m.transcript_bytes, transcript.len() as u64 + 20);
    }
    #[test]
    fn bounds_trace_transcript_and_evidence_chunks() {
        let mut m = model();
        m.config.limit = 160;
        bootstrap(&mut m);
        m.runtime("runtime_decision", decision()).unwrap();
        assert!(
            m.runtime("runtime_decision", decision())
                .err()
                .unwrap()
                .to_string()
                .contains("runtime-choice-bytes")
        );
        m.config.limits.transcript_bytes = 51;
        assert!(
            m.handle(3, "clock_time_get", &json!({"clock_id":1,"precision":0}))
                .err()
                .unwrap()
                .to_string()
                .contains("transcript-bytes")
        );
        assert_eq!(m.now, 1);
        m.pending.clear();
        m.queue("transcript", &vec![5; CHUNK + 1]);
        let mut output = Vec::new();
        m.flush(&mut output, false).unwrap();
        let frame: Value = serde_json::from_slice(&output).unwrap();
        assert_eq!(
            STANDARD
                .decode(frame["data_base64"].as_str().unwrap())
                .unwrap()
                .len(),
            CHUNK
        );
        assert_eq!(m.pending["transcript"].len(), 1);
    }
}
