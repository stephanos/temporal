use crate::protocol::{self, Reply};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::{HashMap, HashSet};
use std::io::{BufRead, Write};
use wasmtime::{Error, Result};

const CHUNK: usize = 64 * 1024;
const MAX_TIME: u64 = i64::MAX as u64;
pub const CPU_CLOCK_UNSUPPORTED: &str = "CPU clocks outside stock WASI profile";

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
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
    pub network: Option<NetworkModel>,
    #[serde(default)]
    pub digest_only: bool,
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
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NetworkModel {
    seed: u64,
    max_delay_nanos: u64,
    #[serde(default)]
    faults: Vec<Fault>,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Fault {
    at_nanos: u64,
    action: String,
    port: u32,
}
struct Network {
    config: NetworkModel,
    next_fault: usize,
    partitioned: HashSet<u32>,
    resetting: HashSet<(u32, u32)>,
    open: HashSet<(u32, u32)>,
    sequence: u64,
}
#[derive(Serialize)]
struct NetworkEvent<'a> {
    seq: u64,
    at: u64,
    op: &'a str,
    local: u32,
    remote: u32,
    #[serde(skip_serializing_if = "is_zero")]
    bytes: u64,
    action: &'a str,
    #[serde(skip_serializing_if = "is_zero")]
    delay: u64,
}
fn is_zero(value: &u64) -> bool {
    *value == 0
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

pub struct Model {
    config: Config,
    pub now: u64,
    pub advances: u64,
    pub transcript_bytes: u64,
    transcript_hash: Sha256,
    network_bytes: u64,
    network: Option<Network>,
    records: u64,
    cursor: usize,
    configured: bool,
    finished: bool,
    peak: u32,
    elapsed: u64,
    terminal: Vec<u8>,
    pending: HashMap<&'static str, Vec<u8>>,
    pub divergence: Option<Divergence>,
}
impl Config {
    pub fn validate(&self) -> Result<()> {
        if !(1..=3).contains(&self.mode)
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
        if let Some(network) = &self.network
            && (network.max_delay_nanos > MAX_TIME / 2
                || network.faults.len() > 4096
                || network.faults.iter().enumerate().any(|(i, f)| {
                    f.at_nanos > MAX_TIME
                        || !matches!(f.action.as_str(), "partition" | "heal" | "reset")
                        || i > 0 && f.at_nanos < network.faults[i - 1].at_nanos
                }))
        {
            return Err(Error::msg("invalid network model bounds or schedule"));
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
        let mut config = config;
        let network = config.network.take().map(|config| Network {
            config,
            next_fault: 0,
            partitioned: HashSet::new(),
            resetting: HashSet::new(),
            open: HashSet::new(),
            sequence: 0,
        });
        Self {
            now: config.now,
            config,
            advances: 0,
            transcript_bytes: 0,
            transcript_hash: Sha256::new(),
            network_bytes: 0,
            network,
            records: 0,
            cursor: 0,
            configured: false,
            finished: false,
            peak: 0,
            elapsed: 0,
            terminal: Vec::new(),
            pending: HashMap::new(),
            divergence: None,
        }
    }
    pub fn call_limit(&self) -> u64 {
        self.config.limits.calls
    }
    #[cfg(test)]
    pub fn state(&self) -> Value {
        json!({"now":self.now,"advances":self.advances,"transcript_bytes":self.transcript_bytes})
    }
    pub fn request_state(&self) -> protocol::RequestState {
        protocol::RequestState {
            advances: self.advances,
            now: self.now,
            transcript_bytes: self.transcript_bytes,
        }
    }
    pub fn metadata(&self, calls: u64) -> Value {
        json!({"now":self.now,"advances":self.advances,"calls":calls,"records":self.records,
            "cursor":self.cursor,"configured":self.configured,"finished":self.finished,
            "peak":self.peak,"elapsed":self.elapsed,"terminal_base64":STANDARD.encode(&self.terminal),
            "transcript_bytes":self.transcript_bytes,"network_bytes":self.network_bytes,
            "transcript_sha256":format!("{:x}",self.transcript_hash.clone().finalize())})
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
        for stream in ["choice", "diagnostic", "transcript", "network"] {
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
        self.now = state.now;
        self.advances = state.advances;
        self.transcript(&bytes);
        Ok(())
    }
    pub fn clock_time_get(
        &mut self,
        id: u64,
        clock_id: Option<u64>,
        call: &[u8],
    ) -> Result<Option<u64>> {
        if !self.config.digest_only
            && self.transcript_bytes + call.len() as u64 + 1029
                > self.config.limits.transcript_bytes
        {
            return Err(Error::msg("WASI capacity: transcript-bytes"));
        }
        let timestamp = if matches!(clock_id, Some(0 | 1)) {
            if self.config.read_step_nanos > MAX_TIME - self.config.epoch_nanos - self.now {
                return Err(Error::msg("WASI capacity: clock-nanoseconds"));
            }
            let timestamp = self.now
                + if clock_id == Some(0) {
                    self.config.epoch_nanos
                } else {
                    0
                };
            self.now += self.config.read_step_nanos;
            Some(timestamp)
        } else {
            None
        };
        #[cfg(feature = "diagnostic-profile")]
        let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
        let reply = match timestamp {
            Some(timestamp) => format!(
                "{{\"type\":\"reply\",\"id\":{id},\"errno\":0,\"output\":{{\"timestamp\":{timestamp}}}}}\n"
            ),
            None => format!(
                "{{\"type\":\"reply\",\"id\":{id},\"errno\":0,\"output\":{{}},\"unsupported\":true,\"message\":\"{CPU_CLOCK_UNSUPPORTED}\"}}\n"
            ),
        };
        #[cfg(feature = "diagnostic-profile")]
        drop(encoding);
        self.transcript(call);
        self.transcript(reply.as_bytes());
        Ok(timestamp)
    }
    #[cfg(test)]
    pub fn handle(&mut self, id: u64, op: &str, input: &Value) -> Result<Option<Reply>> {
        self.handle_input(id, &protocol::CallbackInput::Json { op, input })
    }
    pub fn handle_input(
        &mut self,
        id: u64,
        callback: &protocol::CallbackInput<'_>,
    ) -> Result<Option<Reply>> {
        let op = callback.op();
        let local = op.starts_with("runtime_") && op != "runtime_net_stream"
            || matches!(op, "clock_time_get" | "sched_yield")
            || op == "poll_oneoff"
                && callback
                    .json()
                    .and_then(|input| input["subscriptions"].as_array())
                    .is_some_and(|s| s.iter().all(|s| s["type"] == 0));
        if !local {
            return Ok(None);
        }
        let input = callback.json().unwrap();
        #[cfg(feature = "diagnostic-profile")]
        let encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
        let input_bytes = serde_json::to_vec(input).map_err(Error::new)?;
        let call = format!(
            "{{\"type\":\"call\",\"id\":{id},\"op\":\"{op}\",\"input\":{}}}\n",
            String::from_utf8(input_bytes.clone()).unwrap()
        );
        #[cfg(feature = "diagnostic-profile")]
        drop(encoding);
        if op == "clock_time_get" {
            let timestamp = self.clock_time_get(id, input["clock_id"].as_u64(), call.as_bytes())?;
            return Ok(Some(Reply {
                kind: "reply".into(),
                id,
                errno: 0,
                output: timestamp
                    .map_or_else(|| json!({}), |timestamp| json!({"timestamp":timestamp})),
                unsupported: timestamp.is_none(),
                message: if timestamp.is_none() {
                    CPU_CLOCK_UNSUPPORTED.into()
                } else {
                    String::new()
                },
                local_state: None,
                transcript_base64: None,
            }));
        }
        if !op.starts_with("runtime_") {
            let reservation = 1028
                + if op == "poll_oneoff" {
                    input_bytes.len() as u64 * 4
                } else {
                    0
                };
            if !self.config.digest_only
                && self.transcript_bytes + call.len() as u64 + reservation + 1
                    > self.config.limits.transcript_bytes
            {
                return Err(Error::msg("WASI capacity: transcript-bytes"));
            }
        }
        let (output, errno, unsupported) = if op.starts_with("runtime_") {
            let encoded = input["data_base64"]
                .as_str()
                .ok_or_else(|| Error::msg("WASI invalid: runtime input"))?;
            let data = STANDARD.decode(encoded).map_err(Error::new)?;
            (
                json!({"data_base64":STANDARD.encode(self.runtime(op, data)?)}).to_string(),
                0,
                "",
            )
        } else if op == "poll_oneoff" {
            self.poll(input)?
        } else {
            ("{}".to_owned(), 0, "")
        };
        let reply = Reply {
            kind: "reply".into(),
            id,
            errno,
            output: serde_json::from_str(&output).map_err(Error::new)?,
            unsupported: !unsupported.is_empty(),
            message: unsupported.into(),
            local_state: None,
            transcript_base64: None,
        };
        if !op.starts_with("runtime_") {
            let suffix = if unsupported.is_empty() {
                String::new()
            } else {
                format!(
                    ",\"unsupported\":true,\"message\":{}",
                    serde_json::to_string(unsupported).unwrap()
                )
            };
            self.transcript(call.as_bytes());
            self.transcript(format!("{{\"type\":\"reply\",\"id\":{id},\"errno\":{errno},\"output\":{output}{suffix}}}\n").as_bytes());
        }
        Ok(Some(reply))
    }
    pub fn runtime(&mut self, op: &str, mut data: Vec<u8>) -> Result<Vec<u8>> {
        if op == "runtime_config" {
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
            "runtime_decision" => {
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
            "runtime_observation" => {
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
            "runtime_finish" => {
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
            "runtime_idle" => {
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
            "runtime_net" => self.net(&mut data)?,
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
    fn net(&mut self, data: &mut [u8]) -> Result<()> {
        if data.len() != 32 || get32(data, 0) != 1 || data[24..].iter().any(|b| *b != 0) {
            return Err(Error::msg("WASI invalid: runtime network layout"));
        }
        let op = get32(data, 4);
        let local = get32(data, 8);
        let remote = get32(data, 12);
        let bytes = get64(data, 16);
        if op == 0 {
            if local != 0 || remote != 0 || bytes != 0 {
                return Err(Error::msg("WASI invalid: runtime network query"));
            }
            set32(data, 4, self.network.is_some() as u32);
            return Ok(());
        }
        let n = self
            .network
            .as_mut()
            .ok_or_else(|| Error::msg("WASI invalid: runtime network operation"))?;
        if !matches!(op, 1 | 2) || op == 1 && bytes != 0 {
            return Err(Error::msg("WASI invalid: runtime network operation"));
        }
        while n.next_fault < n.config.faults.len()
            && n.config.faults[n.next_fault].at_nanos <= self.now
        {
            let f = &n.config.faults[n.next_fault];
            n.next_fault += 1;
            match f.action.as_str() {
                "partition" => {
                    n.partitioned.insert(f.port);
                }
                "heal" => {
                    n.partitioned.remove(&f.port);
                }
                _ => {
                    for key in &n.open {
                        if f.port == 0 || key.0 == f.port || key.1 == f.port {
                            n.resetting.insert(*key);
                        }
                    }
                }
            }
        }
        let key = (local.min(remote), local.max(remote));
        let partition = n.partitioned.contains(&0)
            || n.partitioned.contains(&key.0)
            || n.partitioned.contains(&key.1);
        let mut delay = 0;
        if n.config.max_delay_nanos != 0 {
            let mut z = n
                .config
                .seed
                .wrapping_add((n.sequence + 1).wrapping_mul(0x9e3779b97f4a7c15));
            z = (z ^ (z >> 30)).wrapping_mul(0xbf58476d1ce4e5b9);
            z = (z ^ (z >> 27)).wrapping_mul(0x94d049bb133111eb);
            z ^= z >> 31;
            delay = z % (n.config.max_delay_nanos + 1);
        }
        let refused = partition || op == 2 && n.resetting.contains(&key);
        if refused {
            delay = 0;
            if op == 2 {
                n.resetting.remove(&key);
                n.open.remove(&key);
            }
        } else {
            n.open.insert(key);
            if op == 1 {
                n.resetting.remove(&key);
            }
        }
        let mut line = serde_json::to_vec(&NetworkEvent {
            seq: n.sequence,
            at: self.now,
            op: if op == 1 { "connect" } else { "write" },
            local,
            remote,
            bytes,
            action: if refused {
                if op == 1 { "refuse" } else { "reset" }
            } else {
                "deliver"
            },
            delay,
        })
        .map_err(Error::new)?;
        line.push(b'\n');
        if self.network_bytes + line.len() as u64 > self.config.limits.transcript_bytes {
            return Err(Error::msg("WASI capacity: network-transcript-bytes"));
        }
        n.sequence += 1;
        self.network_bytes += line.len() as u64;
        self.queue("network", &line);
        set32(data, 4, refused as u32);
        set64(data, 24, delay);
        Ok(())
    }
    fn poll(&mut self, input: &Value) -> Result<(String, u16, &'static str)> {
        let subscriptions = input["subscriptions"].as_array().unwrap();
        if subscriptions.is_empty() {
            return Ok(("{}".into(), 28, ""));
        }
        if subscriptions.len() as u64 > self.config.limits.pending_events {
            return Err(Error::msg("WASI capacity: pending-events"));
        }
        let mut deadlines = Vec::new();
        let mut earliest = u64::MAX;
        for sub in subscriptions {
            let clock = sub["clock_id"].as_u64().unwrap();
            let timeout = sub["timeout"].as_u64().unwrap();
            let flags = sub["flags"].as_u64().unwrap();
            if clock > 1 {
                return Ok(("{}".into(), 0, "CPU poll clock outside stock WASI profile"));
            }
            if flags & !1 != 0 {
                return Ok(("{}".into(), 28, ""));
            }
            let deadline = if flags == 1 {
                (if clock == 0 {
                    timeout.saturating_sub(self.config.epoch_nanos)
                } else {
                    timeout
                })
                .max(self.now)
            } else {
                if timeout > MAX_TIME - self.config.epoch_nanos - self.now {
                    return Ok(("{}".into(), 28, ""));
                }
                self.now + timeout
            };
            if deadline > MAX_TIME - self.config.epoch_nanos {
                return Ok(("{}".into(), 28, ""));
            }
            earliest = earliest.min(deadline);
            deadlines.push(deadline);
        }
        if earliest > self.now {
            self.advances += 1;
        }
        self.now = earliest;
        let events: Vec<_> = subscriptions
            .iter()
            .zip(deadlines)
            .filter(|(_, at)| *at <= self.now)
            .map(|(sub, _)| {
                format!(
                    "{{\"userdata\":{},\"errno\":0,\"type\":0,\"nbytes\":0,\"flags\":0}}",
                    sub["userdata"]
                )
            })
            .collect();
        Ok((format!("{{\"events\":[{}]}}", events.join(",")), 0, ""))
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
            "diagnostics":false,"epoch_nanos":1000,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}))
        .unwrap();
        config.validate().unwrap();
        config
            .read_tape(&mut std::io::Cursor::new(Vec::new()))
            .unwrap();
        Model::new(config)
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
        assert!(report["nested"]["encoding"]["count"].as_u64().unwrap() >= 3);
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
                100_000
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
            let literal_reply = match timestamp {
                Some(timestamp) => format!("{{\"type\":\"reply\",\"id\":7,\"errno\":0,\"output\":{{\"timestamp\":{timestamp}}}}}\n"),
                None => "{\"type\":\"reply\",\"id\":7,\"errno\":0,\"output\":{},\"unsupported\":true,\"message\":\"CPU clocks outside stock WASI profile\"}\n".into(),
            };
            assert_eq!(
                typed.pending["transcript"],
                format!("{call}{literal_reply}").as_bytes()
            );
            assert_eq!(typed.now, if timestamp.is_some() { 4 } else { 1 });
            assert_eq!(typed.pending, reference.pending);
            assert_eq!(typed.metadata(7), reference.metadata(7));
        }
    }
    #[test]
    fn typed_clock_reservation_digest_and_read_step_failures_preserve_state() {
        let call = b"{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":0}}\n";
        for (digest, capacity, accepted) in [
            (false, call.len() as u64 + 1028, false),
            (false, call.len() as u64 + 1029, true),
            (true, 1, true),
        ] {
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
                    request_scratch: Vec::new(),
                    write_scratch: protocol::WriteBuffers::default(),
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
        let transcript = String::from_utf8(m.pending["transcript"].clone()).unwrap();
        assert!(transcript.starts_with("{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":0}}\n"));
        assert!(transcript.contains(
            "{\"events\":[{\"userdata\":7,\"errno\":0,\"type\":0,\"nbytes\":0,\"flags\":0}]}"
        ));
        let reply:Reply=serde_json::from_value(json!({"type":"reply","id":5,"errno":0,"output":{},
            "local_state":{"now":20,"advances":3},"transcript_base64":STANDARD.encode(b"remote\n")})).unwrap();
        m.remote(&reply, &[]).unwrap();
        assert_eq!((m.now, m.advances), (20, 3));
        assert_eq!(m.transcript_bytes, transcript.len() as u64 + 7);
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
        m.config.limits.transcript_bytes = 100;
        assert!(
            m.handle(3, "clock_time_get", &json!({"clock_id":1,"precision":0}))
                .err()
                .unwrap()
                .to_string()
                .contains("transcript-bytes")
        );
        assert_eq!(m.now, 1);
        m.queue("network", &vec![5; CHUNK + 1]);
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
        assert_eq!(m.pending["network"].len(), 1);
    }
    #[test]
    fn network_faults_and_splitmix_are_ordered_by_virtual_time() {
        let mut m = model();
        bootstrap(&mut m);
        m.network = Some(Network {
            config: NetworkModel {
                seed: 7,
                max_delay_nanos: 10,
                faults: vec![
                    Fault {
                        at_nanos: 5,
                        action: "reset".into(),
                        port: 0,
                    },
                    Fault {
                        at_nanos: 8,
                        action: "partition".into(),
                        port: 20,
                    },
                    Fault {
                        at_nanos: 10,
                        action: "heal".into(),
                        port: 20,
                    },
                ],
            },
            next_fault: 0,
            partitioned: HashSet::new(),
            resetting: HashSet::new(),
            open: HashSet::new(),
            sequence: 0,
        });
        let request = |op| {
            let mut bytes = vec![0; 32];
            set32(&mut bytes, 0, 1);
            set32(&mut bytes, 4, op);
            set32(&mut bytes, 8, 10);
            set32(&mut bytes, 12, 20);
            bytes
        };
        let mut bytes = request(1);
        m.net(&mut bytes).unwrap();
        assert_eq!(get32(&bytes, 4), 0);
        m.now = 5;
        let mut bytes = request(2);
        m.net(&mut bytes).unwrap();
        assert_eq!(get32(&bytes, 4), 1);
        m.now = 8;
        let mut bytes = request(1);
        m.net(&mut bytes).unwrap();
        assert_eq!(get32(&bytes, 4), 1);
        m.now = 10;
        let mut bytes = request(1);
        m.net(&mut bytes).unwrap();
        assert_eq!(get32(&bytes, 4), 0);
        let lines = String::from_utf8(m.pending["network"].clone()).unwrap();
        assert!(lines.contains("\"action\":\"reset\"}"));
        assert_eq!(lines.lines().count(), 4);
    }
}
