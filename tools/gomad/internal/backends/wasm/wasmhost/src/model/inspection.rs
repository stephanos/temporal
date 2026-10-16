use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, VecDeque};
use std::sync::{Arc, Mutex};
use wasmtime::{Error, Result};

pub const MAX_METADATA: usize = 64 << 20;
pub const MAX_OUTPUT: usize = 8 << 20;
pub const MAX_FRAME: usize = 32 + 64 * 1024;

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
    pub complete: bool,
    #[serde(with = "binary")]
    pub world: Vec<u8>,
    #[serde(with = "binary")]
    pub lifecycle: Vec<u8>,
    pub incarnations: Vec<Incarnation>,
    pub boundaries: Vec<Boundary>,
}
pub struct Recorder {
    pub evidence: Evidence,
    order: u64,
    bytes: usize,
    pub exhausted: bool,
    pub sealed: bool,
    cuts: BTreeMap<Handle, Cut>,
}
impl Default for Recorder {
    fn default() -> Self {
        Self {
            evidence: Evidence {
                schema: "gomad3.wasm-inspection/v1".into(),
                complete: false,
                world: b"gomad3.inspection-world/v1\0".to_vec(),
                lifecycle: b"gomad3.inspection-lifecycle/v1\0".to_vec(),
                incarnations: Vec::new(),
                boundaries: Vec::new(),
            },
            order: 0,
            bytes: 128,
            exhausted: false,
            sealed: false,
            cuts: BTreeMap::new(),
        }
    }
}
impl Recorder {
    fn reserve(&mut self, bytes: usize) -> bool {
        if self.exhausted || bytes > MAX_METADATA / 2 - self.bytes.min(MAX_METADATA / 2) {
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
        let value = serde_json::to_vec(&(self.order, action, handle)).unwrap();
        if self.reserve(value.len() + 8) {
            self.evidence
                .lifecycle
                .extend_from_slice(&(value.len() as u64).to_le_bytes());
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
        self.evidence.incarnations.push(Incarnation {
            handle: handle.clone(),
            started_order,
            ..Default::default()
        });
        self.evidence.incarnations.last_mut().unwrap()
    }
    pub fn stream(&mut self, handle: &Handle, name: &str, bytes: &[u8]) {
        if self.sealed || bytes.is_empty() || !self.reserve(bytes.len()) {
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
        if !self.reserve(storage.len().saturating_sub(prior)) {
            return;
        }
        let owner = self.owner(&handle);
        owner.streams.storage = storage.to_vec();
        cut.network_state = network.to_vec();
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
            ..Default::default()
        });
        cut.witness = witness.clone();
        cut.clock_run = clock;
        let count = self.cuts.len();
        let state_bytes = self
            .cuts
            .values()
            .try_fold(0usize, |total, cut| {
                total
                    .checked_add(cut.storage_state.len())?
                    .checked_add(cut.recovery_state.len())?
                    .checked_add(cut.network_state.len())?
                    .checked_add(cut.clock_run.len())
            })
            .unwrap_or(MAX_METADATA);
        if self.evidence.boundaries.len() >= 65536 {
            self.exhausted = true;
            return Err(Error::msg("inspection boundary capacity"));
        }
        if !self.reserve(512 * count + state_bytes + world.len() + 8) {
            return Err(Error::msg("inspection metadata capacity"));
        }
        self.evidence
            .world
            .extend_from_slice(&(world.len() as u64).to_le_bytes());
        self.evidence.world.extend_from_slice(world);
        let boundary = Boundary {
            handle: handle.clone(),
            decision: witness.choices,
            trace: witness.choices + witness.observations,
            order,
            time_nanos: time,
            world_bytes: self.evidence.world.len() as u64,
            lifecycle_bytes: self.evidence.lifecycle.len() as u64,
            cuts: self.cuts.values().cloned().collect(),
        };
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
        if self.sealed || !self.reserve(bytes.len()) {
            return;
        }
        let length = bytes.len() as u64;
        self.owner(handle).streams.network = bytes;
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_bytes = length;
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
            self.evidence.world = b"gomad3.inspection-world/v1\0".to_vec();
            self.evidence.lifecycle = b"gomad3.inspection-lifecycle/v1\0".to_vec();
            for owner in &mut self.evidence.incarnations {
                owner.streams = Streams::default();
            }
        }
        self.evidence.complete = complete && !self.exhausted;
        let bytes = serde_json::to_vec(&self.evidence).map_err(Error::new)?;
        if bytes.len() > MAX_METADATA {
            self.exhausted = true;
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
pub fn le32(bytes: &[u8], at: usize) -> u32 {
    u32::from_le_bytes(bytes[at..at + 4].try_into().unwrap())
}
pub fn digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
