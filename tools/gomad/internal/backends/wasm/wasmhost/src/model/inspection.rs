use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, VecDeque};
use std::sync::{Arc, Mutex};
use wasmtime::{Error, Result};

pub const MAX_METADATA: usize = 64 << 20;
pub const MAX_OUTPUT: usize = 8 << 20;
pub const MAX_FRAME: usize = 32 + 64 * 1024;
pub const STATE_ENCODING: &str = "gomad.inspection-state-sha256/v1";
pub const WORLD_ENCODING: &str = "gomad.inspection-world-sha256/v1";
const WORLD_HEADER: &[u8] = b"gomad.inspection-world-commitments/v1\0";
pub const BOUNDARY_ENCODING: &str = "gomad.inspection-boundaries-binary/v1";
pub const LIFECYCLE_ENCODING: &str = "gomad.inspection-lifecycle-binary/v1";
const BOUNDARY_HEADER: &[u8] = b"GMWIB\x01\0\0";
const LIFECYCLE_HEADER: &[u8] = b"GMWIL\x01\0\0";

mod binary {
    use super::*;
    pub fn serialize<S: serde::Serializer>(
        bytes: &[u8],
        serializer: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        serializer.serialize_str(&STANDARD.encode(bytes))
    }
    pub fn deserialize<'de, D: serde::Deserializer<'de>>(
        deserializer: D,
    ) -> std::result::Result<Vec<u8>, D::Error> {
        let value = String::deserialize(deserializer)?;
        if value.len() > MAX_METADATA.div_ceil(3) * 4 {
            return Err(serde::de::Error::custom("inspection byte capacity"));
        }
        STANDARD.decode(value).map_err(serde::de::Error::custom)
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq, PartialOrd, Ord, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Handle {
    pub node: String,
    pub incarnation: u64,
}
#[derive(Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Witness {
    pub allocations: u64,
    pub allocated_bytes: u64,
    pub gc_cycles: u64,
    pub gc_phase: u64,
    pub goroutines: u64,
    pub run_queue: u64,
    pub global_run_queue: u64,
    pub seeded_draws: u64,
    pub seeded_draw_counters: [u64; 7],
    pub choices: u64,
    pub observations: u64,
    #[serde(with = "binary")]
    pub runtime_state: Vec<u8>,
}
impl Witness {
    pub fn decode(bytes: &[u8]) -> Result<Self> {
        if bytes.len() != 128 {
            return Err(Error::msg("invalid compulsory runtime witness"));
        }
        let u64_at = |at| u64::from_be_bytes(bytes[at..at + 8].try_into().unwrap());
        let u32_at = |at| u32::from_be_bytes(bytes[at..at + 4].try_into().unwrap()) as u64;
        let mut counters = [0; 7];
        for (index, counter) in counters.iter_mut().enumerate() {
            *counter = u64_at(40 + index * 8);
        }
        let choices = u64_at(104);
        let records = u64_at(0);
        if choices > records || u64_at(112) == 0 {
            return Err(Error::msg("invalid runtime witness counters"));
        }
        Ok(Self {
            allocations: u64_at(16),
            allocated_bytes: u64_at(96),
            gc_cycles: u32_at(24),
            gc_phase: bytes[28] as u64,
            goroutines: u64_at(112),
            run_queue: u32_at(32),
            global_run_queue: u64_at(120),
            seeded_draws: counters
                .iter()
                .fold(0u64, |total, n| total.wrapping_add(*n)),
            seeded_draw_counters: counters,
            choices,
            observations: records - choices,
            runtime_state: bytes.to_vec(),
        })
    }
}
#[derive(Clone, Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Streams {
    #[serde(with = "binary")]
    pub choice: Vec<u8>,
    #[serde(with = "binary")]
    pub wasi: Vec<u8>,
    #[serde(with = "binary")]
    pub network: Vec<u8>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub network_encoding: String,
    #[serde(with = "binary")]
    pub storage: Vec<u8>,
    #[serde(with = "binary")]
    pub stdout: Vec<u8>,
    #[serde(with = "binary")]
    pub stderr: Vec<u8>,
}
#[derive(Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Registry {
    pub closed: bool,
    pub closed_order: u64,
    pub entries: Vec<Registration>,
}
#[derive(Clone, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Registration {
    pub id: u32,
    pub kind: String,
    pub name: String,
    pub available_order: u64,
    pub monotonic: bool,
}
#[derive(Clone, Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Incarnation {
    pub handle: Handle,
    pub started_order: u64,
    pub terminal_order: u64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub failure_order: Option<u64>,
    pub registry: Registry,
    pub streams: Streams,
}
#[derive(Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Cut {
    pub handle: Handle,
    pub state_encoding: String,
    pub choice_records: u64,
    pub choice_bytes: u64,
    pub wasi_bytes: u64,
    pub network_bytes: u64,
    #[serde(with = "binary")]
    pub network_state: Vec<u8>,
    pub storage_bytes: u64,
    pub stdout_bytes: u64,
    pub stderr_bytes: u64,
    #[serde(with = "binary")]
    pub clock_run: Vec<u8>,
    #[serde(with = "binary")]
    pub storage_state: Vec<u8>,
    #[serde(with = "binary")]
    pub recovery_state: Vec<u8>,
    pub witness: Witness,
}
#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Boundary {
    pub handle: Handle,
    pub decision: u64,
    pub trace: u64,
    pub order: u64,
    pub time_nanos: u64,
    pub world_bytes: u64,
    pub lifecycle_bytes: u64,
    pub cuts: Vec<Cut>,
}
#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Evidence {
    pub schema: String,
    pub boundary_encoding: String,
    pub lifecycle_encoding: String,
    pub state_encoding: String,
    pub world_encoding: String,
    pub complete: bool,
    #[serde(with = "binary")]
    pub world: Vec<u8>,
    #[serde(with = "binary")]
    pub lifecycle: Vec<u8>,
    pub incarnations: Vec<Incarnation>,
    #[serde(with = "binary")]
    pub boundary_frames: Vec<u8>,
    #[serde(skip)]
    pub boundaries: Vec<Boundary>,
}
pub struct Recorder {
    pub evidence: Evidence,
    order: u64,
    bytes: usize,
    pub exhausted: bool,
    pub sealed: bool,
    pub observed: bool,
    cuts: BTreeMap<Handle, Cut>,
}
impl Default for Recorder {
    fn default() -> Self {
        Self {
            evidence: Evidence {
                schema: "gomad.wasm-inspection/v3".into(),
                boundary_encoding: BOUNDARY_ENCODING.into(),
                lifecycle_encoding: LIFECYCLE_ENCODING.into(),
                state_encoding: STATE_ENCODING.into(),
                world_encoding: WORLD_ENCODING.into(),
                complete: false,
                world: WORLD_HEADER.to_vec(),
                lifecycle: LIFECYCLE_HEADER.to_vec(),
                incarnations: Vec::new(),
                boundary_frames: [BOUNDARY_HEADER, &[0; 4]].concat(),
                boundaries: Vec::new(),
            },
            order: 0,
            bytes: 4096,
            exhausted: false,
            sealed: false,
            observed: false,
            cuts: BTreeMap::new(),
        }
    }
}
impl Recorder {
    pub fn observe(&mut self, handle: &Handle) {
        self.observed = true;
        self.owner(handle);
    }
    fn reserve(&mut self, bytes: usize) -> bool {
        if self.exhausted || bytes > MAX_METADATA - self.bytes.min(MAX_METADATA) {
            self.exhausted = true;
            return false;
        }
        self.bytes += bytes;
        true
    }
    pub fn event(&mut self, action: &str, handle: &Handle) -> u64 {
        if self.sealed {
            return self.order;
        }
        self.order += 1;
        let tag = match action {
            "start" => 1,
            "decision" => 2,
            "terminal" => 3,
            "register" => 4,
            "registry_closed" => 5,
            "model_effect" => 6,
            "controller_effect" => 7,
            "crash" => 8,
            "stop" => 9,
            "restart" => 10,
            _ => {
                self.exhausted = true;
                return self.order;
            }
        };
        let mut value = vec![tag];
        if put_handle(&mut value, handle, action == "controller_effect").is_err() {
            self.exhausted = true;
            return self.order;
        }
        if self.reserve(encoded_growth(self.evidence.lifecycle.len(), value.len())) {
            self.evidence.lifecycle.extend(value);
        }
        self.order
    }
    fn owner(&mut self, handle: &Handle) -> &mut Incarnation {
        if let Some(index) = self
            .evidence
            .incarnations
            .iter()
            .position(|n| n.handle == *handle)
        {
            return &mut self.evidence.incarnations[index];
        }
        let started_order = self.event("start", handle);
        self.reserve(4096);
        self.evidence.incarnations.push(Incarnation {
            handle: handle.clone(),
            started_order,
            ..Default::default()
        });
        self.evidence.incarnations.last_mut().unwrap()
    }
    pub fn stream(&mut self, handle: &Handle, name: &str, bytes: &[u8]) {
        if self.sealed || self.exhausted || bytes.is_empty() {
            return;
        }
        let streams = &self.owner(handle).streams;
        let prior = match name {
            "choice" => streams.choice.len(),
            "transcript" => streams.wasi.len(),
            "storage" => streams.storage.len(),
            "stdout" => streams.stdout.len(),
            "stderr" => streams.stderr.len(),
            _ => return,
        };
        if !self.reserve(encoded_growth(prior, bytes.len())) {
            return;
        }
        let streams = &mut self.owner(handle).streams;
        match name {
            "choice" => streams.choice.extend_from_slice(bytes),
            "transcript" => streams.wasi.extend_from_slice(bytes),
            "storage" => streams.storage.extend_from_slice(bytes),
            "stdout" => streams.stdout.extend_from_slice(bytes),
            "stderr" => streams.stderr.extend_from_slice(bytes),
            _ => {}
        }
    }
    pub fn snapshot(&mut self, mut cut: Cut, network: &[u8], storage: &[u8]) {
        if self.sealed || self.exhausted {
            return;
        }
        let handle = cut.handle.clone();
        let prior = self.owner(&handle).streams.storage.len();
        if !self.reserve(base64_len(storage.len()).saturating_sub(base64_len(prior))) {
            return;
        }
        let owner = self.owner(&handle);
        owner.streams.storage = storage.to_vec();
        cut.state_encoding = STATE_ENCODING.into();
        cut.storage_state = commitment("storage", &cut.storage_state);
        cut.recovery_state = commitment("recovery", &cut.recovery_state);
        cut.network_state = commitment("network", network);
        let streams = &owner.streams;
        cut.choice_records = streams.choice.len() as u64 / 96;
        cut.choice_bytes = streams.choice.len() as u64;
        cut.wasi_bytes = streams.wasi.len() as u64;
        cut.network_bytes = streams.network.len() as u64;
        cut.storage_bytes = streams.storage.len() as u64;
        cut.stdout_bytes = streams.stdout.len() as u64;
        cut.stderr_bytes = streams.stderr.len() as u64;
        self.cuts.insert(handle, cut);
    }
    pub fn boundary(
        &mut self,
        handle: &Handle,
        witness: Witness,
        time: u64,
        clock: Vec<u8>,
        world: &[u8],
    ) -> Result<Boundary> {
        if self.sealed || self.exhausted {
            return Err(Error::msg("incomplete inspection metadata"));
        }
        let order = self.event("decision", handle);
        let cut = self.cuts.entry(handle.clone()).or_insert_with(|| Cut {
            handle: handle.clone(),
            state_encoding: STATE_ENCODING.into(),
            storage_state: commitment("storage", &[]),
            recovery_state: commitment("recovery", &[]),
            network_state: commitment("network", &[]),
            ..Default::default()
        });
        cut.witness = witness.clone();
        cut.clock_run = clock;
        if self.evidence.boundaries.len() >= 65536 {
            self.exhausted = true;
            return Err(Error::msg("inspection boundary capacity"));
        }
        let world = commitment("world", world);
        let boundary = Boundary {
            handle: handle.clone(),
            decision: witness.choices,
            trace: witness.choices + witness.observations,
            order,
            time_nanos: time,
            world_bytes: (self.evidence.world.len() + 8 + world.len()) as u64,
            lifecycle_bytes: self.evidence.lifecycle.len() as u64,
            cuts: self.cuts.values().cloned().collect(),
        };
        let frame = match boundary_frame(&boundary) {
            Ok(frame) => frame,
            Err(error) => {
                self.exhausted = true;
                return Err(error);
            }
        };
        let charge = encoded_growth(self.evidence.world.len(), world.len() + 8)
            + encoded_growth(self.evidence.boundary_frames.len(), frame.len());
        if !self.reserve(charge) {
            return Err(Error::msg("inspection metadata capacity"));
        }
        self.evidence
            .world
            .extend_from_slice(&(world.len() as u64).to_le_bytes());
        self.evidence.world.extend_from_slice(&world);
        self.evidence.boundary_frames.extend_from_slice(&frame);
        self.evidence.boundary_frames[8..12]
            .copy_from_slice(&((self.evidence.boundaries.len() + 1) as u32).to_le_bytes());
        self.evidence.boundaries.push(boundary.clone());
        Ok(boundary)
    }
    pub fn terminal(&mut self, handle: &Handle, failed: bool) {
        if self.sealed {
            return;
        }
        let order = self.event("terminal", handle);
        let owner = self.owner(handle);
        owner.terminal_order = order;
        owner.failure_order = failed.then_some(order);
    }
    pub fn terminal_network(&mut self, handle: &Handle, bytes: Vec<u8>) {
        if self.sealed || self.exhausted {
            return;
        }
        let prior = self.owner(handle).streams.network.len();
        if !self.reserve(base64_len(bytes.len()).saturating_sub(base64_len(prior))) {
            return;
        }
        let length = bytes.len() as u64;
        let state = commitment("network", &bytes);
        let streams = &mut self.owner(handle).streams;
        streams.network = bytes;
        streams.network_encoding = "gomad.cluster-network-binary/v1".into();
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_bytes = length;
            cut.network_state = state;
        }
    }
    pub fn network_snapshot(&mut self, handle: &Handle, bytes: Vec<u8>) {
        if self.sealed || self.exhausted {
            return;
        }
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_state = commitment("network", &bytes);
        }
    }
    pub fn bytes(&mut self, complete: bool) -> Result<Vec<u8>> {
        if self
            .cuts
            .values()
            .any(|cut| cut.witness.runtime_state.len() != 128)
        {
            self.exhausted = true;
        }
        if self.exhausted {
            self.evidence.boundaries.clear();
            self.evidence.boundary_frames = [BOUNDARY_HEADER, &[0; 4]].concat();
            self.evidence.world = WORLD_HEADER.to_vec();
            self.evidence.lifecycle = LIFECYCLE_HEADER.to_vec();
            for owner in &mut self.evidence.incarnations {
                owner.streams = Streams::default();
            }
        }
        self.evidence.complete = complete && !self.exhausted;
        let bytes = serde_json::to_vec(&self.evidence).map_err(Error::new)?;
        if bytes.len() > MAX_METADATA {
            self.exhausted = true;
            self.evidence.incarnations.clear();
            return self.bytes(false);
        }
        Ok(bytes)
    }
}

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Command {
    pub kind: u32,
    pub id: u32,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Request {
    pub target: Handle,
    pub decision: u64,
    pub expected: Boundary,
    pub world_sha256: String,
    pub lifecycle_sha256: String,
    pub stream_sha256: Vec<BTreeMap<String, String>>,
    pub probes: Vec<u32>,
    pub commands: Vec<Command>,
    pub registries: Vec<Registry>,
}
pub struct Transport {
    pub recorder: Arc<Mutex<Recorder>>,
    pub handle: Handle,
    pub witness: Witness,
    pub request: Option<Request>,
    pub sealed: bool,
    pub output: Vec<u8>,
    pub truncated: bool,
    pub commands: VecDeque<Command>,
    pub world: Vec<u8>,
    pub time: u64,
    pub started: bool,
    pub dropped: u64,
}
impl Default for Transport {
    fn default() -> Self {
        Self {
            recorder: Arc::new(Mutex::new(Recorder::default())),
            handle: Handle {
                node: "guest".into(),
                incarnation: 1,
            },
            witness: Witness::default(),
            request: None,
            sealed: false,
            output: Vec::new(),
            truncated: false,
            commands: VecDeque::new(),
            world: Vec::new(),
            time: 0,
            started: false,
            dropped: 0,
        }
    }
}
impl Transport {
    pub fn dormant_exchange(&mut self, frame: &mut [u8]) -> Result<()> {
        let op = validate_frame(frame)?;
        let payload = &frame[16..];
        match op {
            1 => {
                if payload.len() < 32 + 64 + 128 {
                    return Err(Error::msg("invalid debug predecision length"));
                }
                let count = u32::from_be_bytes(payload[20..24].try_into().unwrap()) as usize;
                let length = 32 + count * 32;
                if !(2..=256).contains(&count) || payload.len() != length + 128 {
                    return Err(Error::msg("invalid debug predecision alternatives"));
                }
                validate_pending(&payload[..length])?;
                self.witness = Witness::decode(&payload[length..])?;
            }
            2 => self.registry(payload)?,
            3 => {
                if payload.len() != 4 || le32(payload, 0) > 1 {
                    return Err(Error::msg("invalid registry closure"));
                }
                self.close_registry(le32(payload, 0) == 1);
            }
            7 => {
                if payload.len() != 32 || payload.iter().any(|b| *b != 0) {
                    return Err(Error::msg("invalid debug enablement mask"));
                }
            }
            10 => self.witness = Witness::decode(payload)?,
            _ => {
                return Err(Error::msg(
                    "inspection unsupported without helper-owned runtime",
                ));
            }
        }
        Ok(())
    }
    pub fn lock(&self) -> std::sync::MutexGuard<'_, Recorder> {
        self.recorder.lock().unwrap_or_else(|e| e.into_inner())
    }
    pub fn frame(&mut self, op: u32, payload: &[u8]) {
        if self.output.len() + 32 + payload.len() > MAX_OUTPUT - 64 {
            self.truncated = true;
            self.dropped += 1;
            return;
        }
        self.output.extend_from_slice(&op.to_le_bytes());
        self.output
            .extend_from_slice(&(payload.len() as u32).to_le_bytes());
        self.output
            .extend_from_slice(&self.witness.choices.to_le_bytes());
        self.output
            .extend_from_slice(&(self.witness.choices + self.witness.observations).to_le_bytes());
        self.output.extend_from_slice(&self.time.to_le_bytes());
        self.output.extend_from_slice(payload);
    }
    pub fn finish_output(&mut self) {
        if self.dropped == 0 {
            return;
        }
        let mut marker = [0u8; 24];
        marker[..4].copy_from_slice(&4u32.to_le_bytes());
        marker[12..16].copy_from_slice(&8u32.to_le_bytes());
        marker[16..24].copy_from_slice(&self.dropped.to_le_bytes());
        self.output.extend_from_slice(&4u32.to_le_bytes());
        self.output.extend_from_slice(&24u32.to_le_bytes());
        self.output
            .extend_from_slice(&self.witness.choices.to_le_bytes());
        self.output
            .extend_from_slice(&(self.witness.choices + self.witness.observations).to_le_bytes());
        self.output.extend_from_slice(&self.time.to_le_bytes());
        self.output.extend_from_slice(&marker);
    }
    pub fn registry(&mut self, payload: &[u8]) -> Result<()> {
        if payload.len() < 16 {
            return Err(Error::msg("invalid debug registration"));
        }
        let kind = le32(payload, 0);
        let id = le32(payload, 4);
        let flags = le32(payload, 8);
        let size = le32(payload, 12) as usize;
        let name = std::str::from_utf8(&payload[16..]).map_err(Error::new)?;
        if !(1..=128).contains(&size)
            || payload.len() != 16 + size
            || flags & !1 != 0
            || flags != 0 && kind != 4
            || !name
                .bytes()
                .all(|b| b.is_ascii_alphanumeric() || matches!(b, b'.' | b'_' | b'-' | b'/'))
        {
            return Err(Error::msg("invalid debug registration name or flags"));
        }
        let mut recorder = self.lock();
        let order = recorder.event("register", &self.handle);
        let owner = recorder.owner(&self.handle);
        if owner.registry.closed
            || id as usize != owner.registry.entries.len()
            || id >= 256
            || owner
                .registry
                .entries
                .iter()
                .any(|entry| entry.name == name)
        {
            return Err(Error::msg(
                "duplicate, late or oversized debug registration",
            ));
        }
        let kind = match kind {
            3 => "dump",
            4 => "check",
            5 => "probe",
            _ => return Err(Error::msg("invalid debug registration kind")),
        };
        recorder.reserve(256 + name.len());
        let owner = recorder.owner(&self.handle);
        owner.registry.entries.push(Registration {
            id,
            kind: kind.into(),
            name: name.into(),
            available_order: order,
            monotonic: flags != 0,
        });
        Ok(())
    }
    pub fn close_registry(&mut self, ok: bool) {
        let mut recorder = self.lock();
        let order = recorder.event("registry_closed", &self.handle);
        let registry = &mut recorder.owner(&self.handle).registry;
        if !ok {
            recorder.exhausted = true;
            return;
        }
        if !registry.closed {
            registry.closed = true;
            registry.closed_order = order;
        }
    }
}
pub fn validate_frame(frame: &[u8]) -> Result<u32> {
    if frame.len() < 16
        || frame.len() > MAX_FRAME
        || le32(frame, 0) != 1
        || le32(frame, 8) != 0
        || le32(frame, 12) as usize != frame.len() - 16
    {
        return Err(Error::msg("invalid debug exchange frame"));
    }
    Ok(le32(frame, 4))
}
pub fn validate_pending(data: &[u8]) -> Result<Vec<[u8; 32]>> {
    if data.len() < 32 {
        return Err(Error::msg("invalid debug pending decision"));
    }
    let word = |at| u32::from_be_bytes(data[at..at + 4].try_into().unwrap());
    let count = word(20) as usize;
    if word(0) != 1
        || !matches!(word(4), 1 | 2)
        || word(16) & !4 != 0
        || !(2..=256).contains(&count)
        || word(24) as usize >= count
        || data.len() != 32 + count * 32
    {
        return Err(Error::msg("invalid debug pending decision"));
    }
    let mut identities: Vec<[u8; 32]> = data[32..]
        .chunks_exact(32)
        .map(|bytes| bytes.try_into().unwrap())
        .collect();
    identities.sort();
    if identities.contains(&[0u8; 32]) || identities.windows(2).any(|pair| pair[0] == pair[1]) {
        return Err(Error::msg("invalid debug pending alternative identities"));
    }
    Ok(identities)
}
pub fn le32(bytes: &[u8], at: usize) -> u32 {
    u32::from_le_bytes(bytes[at..at + 4].try_into().unwrap())
}
pub fn digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
fn base64_len(length: usize) -> usize {
    length.div_ceil(3) * 4
}
fn encoded_growth(prior: usize, extra: usize) -> usize {
    base64_len(prior + extra) - base64_len(prior)
}
fn put_handle(bytes: &mut Vec<u8>, handle: &Handle, controller: bool) -> Result<()> {
    if handle.node.is_empty()
        || handle.node.len() > 1024
        || handle.node.contains(['\0', '\r', '\n'])
        || if controller {
            handle.node != "controller" || handle.incarnation != 0
        } else {
            handle.incarnation == 0
        }
    {
        return Err(Error::msg("invalid inspection handle"));
    }
    bytes.extend_from_slice(&(handle.node.len() as u32).to_le_bytes());
    bytes.extend_from_slice(handle.node.as_bytes());
    bytes.extend_from_slice(&handle.incarnation.to_le_bytes());
    Ok(())
}
fn boundary_frame(boundary: &Boundary) -> Result<Vec<u8>> {
    let mut bytes = vec![0; 4];
    put_handle(&mut bytes, &boundary.handle, false)?;
    for word in [
        boundary.decision,
        boundary.trace,
        boundary.order,
        boundary.time_nanos,
        boundary.world_bytes,
        boundary.lifecycle_bytes,
    ] {
        bytes.extend_from_slice(&word.to_le_bytes());
    }
    if boundary.cuts.is_empty() || boundary.cuts.len() > 65536 {
        return Err(Error::msg("invalid inspection cut count"));
    }
    bytes.extend_from_slice(&(boundary.cuts.len() as u32).to_le_bytes());
    for cut in &boundary.cuts {
        if cut.state_encoding != STATE_ENCODING
            || cut.network_state.len() != 32
            || cut.storage_state.len() != 32
            || cut.recovery_state.len() != 32
            || cut.clock_run.len() > 1024
            || Witness::decode(&cut.witness.runtime_state)? != cut.witness
        {
            return Err(Error::msg("invalid inspection cut or compulsory witness"));
        }
        put_handle(&mut bytes, &cut.handle, false)?;
        for word in [
            cut.choice_records,
            cut.choice_bytes,
            cut.wasi_bytes,
            cut.network_bytes,
            cut.storage_bytes,
            cut.stdout_bytes,
            cut.stderr_bytes,
        ] {
            bytes.extend_from_slice(&word.to_le_bytes());
        }
        bytes.extend_from_slice(&(cut.clock_run.len() as u32).to_le_bytes());
        bytes.extend_from_slice(&cut.clock_run);
        bytes.extend_from_slice(&cut.network_state);
        bytes.extend_from_slice(&cut.storage_state);
        bytes.extend_from_slice(&cut.recovery_state);
        bytes.extend_from_slice(&cut.witness.runtime_state);
    }
    let length = bytes.len() - 4;
    if length > MAX_METADATA {
        return Err(Error::msg("inspection frame capacity"));
    }
    bytes[..4].copy_from_slice(&(length as u32).to_le_bytes());
    Ok(bytes)
}
pub fn commitment(kind: &str, bytes: &[u8]) -> Vec<u8> {
    let mut hash = Sha256::new();
    if kind == "world" {
        hash.update(b"gomad.inspection-world-sha256/v1\0");
    } else {
        hash.update(b"gomad.inspection-state-sha256/v1/");
        hash.update(kind.as_bytes());
        hash.update([0]);
    }
    hash.update((bytes.len() as u64).to_le_bytes());
    hash.update(bytes);
    hash.finalize().to_vec()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn witness_at(choices: u64) -> Witness {
        let mut raw = vec![0; 128];
        raw[..8].copy_from_slice(&choices.to_be_bytes());
        raw[104..112].copy_from_slice(&choices.to_be_bytes());
        raw[112..120].copy_from_slice(&1u64.to_be_bytes());
        Witness::decode(&raw).unwrap()
    }

    #[test]
    fn repeated_static_mount_and_world_cuts_fit_bounded_metadata() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "history".into(),
            incarnation: 1,
        };
        recorder.observe(&handle);
        let mount = vec![b'm'; 64 << 10];
        let world = vec![b'w'; 64 << 10];
        for ordinal in 0..2000 {
            recorder.snapshot(
                Cut {
                    handle: handle.clone(),
                    storage_state: mount.clone(),
                    ..Default::default()
                },
                &[],
                &[],
            );
            recorder
                .boundary(&handle, witness_at(ordinal), ordinal, vec![], &world)
                .unwrap();
            recorder.stream(&handle, "choice", &[0; 96]);
        }
        let encoded = recorder.bytes(true).unwrap();
        eprintln!(
            "inspection metadata: {} bytes for 2000 boundaries and repeated 64KiB mount/World",
            encoded.len()
        );
        assert!(recorder.evidence.complete);
        assert_eq!(recorder.evidence.schema, "gomad.wasm-inspection/v3");
        assert_eq!(recorder.evidence.world_encoding, WORLD_ENCODING);
        assert_eq!(recorder.evidence.boundaries.len(), 2000);
        assert!(encoded.len() < 8 << 20);
        assert_eq!(
            recorder.evidence.boundaries[0].cuts[0].storage_state.len(),
            32
        );
        assert_eq!(
            recorder.evidence.incarnations[0].streams.choice.len(),
            2000 * 96
        );
    }
    #[test]
    fn state_commitments_bind_domains_lengths_and_last_bytes() {
        let storage = commitment("storage", b"abc");
        let hex: String = storage.iter().map(|b| format!("{b:02x}")).collect();
        assert_eq!(
            hex,
            "bc21ac8235dac2e82b5ff9401aec3720cbe19d167569211b906be2b290c51b7a"
        );
        for kind in ["storage", "recovery", "network", "world"] {
            assert_eq!(commitment(kind, &[]).len(), 32);
            assert_ne!(commitment(kind, b"abc"), commitment(kind, b"abd"));
            assert_ne!(commitment(kind, b"abc"), commitment(kind, b"abc\0"));
            if kind != "storage" {
                assert_ne!(storage, commitment(kind, b"abc"));
            }
        }
    }
    #[test]
    fn terminal_network_cut_commits_the_retained_terminal_document() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "peer".into(),
            incarnation: 1,
        };
        recorder.snapshot(
            Cut {
                handle: handle.clone(),
                ..Default::default()
            },
            b"before",
            &[],
        );
        recorder.terminal_network(&handle, b"after".to_vec());
        let cut = recorder.cuts.get(&handle).unwrap();
        assert_eq!(cut.network_state, commitment("network", b"after"));
        assert_eq!(cut.network_bytes, 5);
    }
    #[test]
    fn workflow_scale_inventory_fits_without_omitting_boundaries() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "history".into(),
            incarnation: 1,
        };
        recorder.snapshot(
            Cut {
                handle: handle.clone(),
                storage_state: vec![b'm'; 64 << 10],
                ..Default::default()
            },
            &[],
            &[],
        );
        recorder.stream(&handle, "transcript", crate::transcript::HEADER);
        let mut calls = 0;
        for ordinal in 0..51_000 {
            for _ in 0..if ordinal < 19_000 { 8 } else { 7 } {
                recorder.event("model_effect", &handle);
                calls += 1;
                recorder.stream(
                    &handle,
                    "transcript",
                    &crate::transcript::yield_record(calls),
                );
            }
            {
                let cut = recorder.cuts.get_mut(&handle).unwrap();
                cut.choice_records = ordinal;
                cut.choice_bytes = ordinal * 96;
                cut.wasi_bytes = calls * 16 + crate::transcript::HEADER.len() as u64;
            }
            recorder
                .boundary(&handle, witness_at(ordinal), ordinal, vec![], &[])
                .unwrap();
            recorder.stream(&handle, "choice", &[0; 96]);
        }
        recorder.terminal(&handle, false);
        let encoded = recorder.bytes(true).unwrap();
        eprintln!(
            "workflow-scale metadata={} bytes, boundaries={}, model_effects={}, rawchoice={}, rawWASI={}, lifecycle={}, boundary_frames={}",
            encoded.len(),
            recorder.evidence.boundaries.len(),
            calls,
            recorder.evidence.incarnations[0].streams.choice.len(),
            recorder.evidence.incarnations[0].streams.wasi.len(),
            recorder.evidence.lifecycle.len(),
            recorder.evidence.boundary_frames.len()
        );
        assert!(recorder.evidence.complete);
        assert_eq!(calls, 376_000);
        assert_eq!(recorder.evidence.boundaries.len(), 51_000);
        assert!(encoded.len() <= MAX_METADATA);
    }
}
