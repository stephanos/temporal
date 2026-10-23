use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, VecDeque};
use std::sync::{Arc, Mutex};
use wasmtime::{Error, Result};

pub const MAX_METADATA: usize = 64 << 20;
pub const MAX_OUTPUT: usize = 8 << 20;
pub const MAX_FRAME: usize = 32 + 64 * 1024;
pub const STATE_ENCODING: &str = "gomad.inspection-state-sha256/v2";
pub const WORLD_ENCODING: &str = "gomad.inspection-world-sha256/v2";
pub(crate) const WORLD_HEADER: &[u8] = b"gomad.inspection-world-commitments/v2\0";
pub const BOUNDARY_ENCODING: &str = "gomad.inspection-boundaries-binary/v1";
pub const LIFECYCLE_ENCODING: &str = "gomad.inspection-lifecycle-binary/v1";
const BOUNDARY_HEADER: &[u8] = b"GMWIB\x01\0\0";
const LIFECYCLE_HEADER: &[u8] = b"GMWIL\x01\0\0";
const MAX_RECOVERY_CACHE_ENTRIES: usize = 64;
const MAX_RECOVERY_CACHE_BYTES: usize = MAX_METADATA - std::mem::size_of::<RecoveryCache>();

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
    /// # Errors
    /// Refuses malformed compulsory runtime witness bytes and inconsistent counters.
    pub fn decode(bytes: &[u8]) -> Result<Self> {
        let invalid = || Error::msg("invalid compulsory runtime witness");
        let bytes: &[u8; 128] = bytes.try_into().map_err(|_| invalid())?;
        let u64_at = |at| -> Result<u64> {
            Ok(u64::from_be_bytes(
                bytes
                    .get(at..at + 8)
                    .ok_or_else(invalid)?
                    .try_into()
                    .map_err(|_| invalid())?,
            ))
        };
        let u32_at = |at| -> Result<u64> {
            Ok(u64::from(u32::from_be_bytes(
                bytes
                    .get(at..at + 4)
                    .ok_or_else(invalid)?
                    .try_into()
                    .map_err(|_| invalid())?,
            )))
        };
        let mut counters = [0; 7];
        for (index, counter) in counters.iter_mut().enumerate() {
            *counter = u64_at(40 + index * 8)?;
        }
        let choices = u64_at(104)?;
        let records = u64_at(0)?;
        if choices > records || u64_at(112)? == 0 {
            return Err(Error::msg("invalid runtime witness counters"));
        }
        Ok(Self {
            allocations: u64_at(16)?,
            allocated_bytes: u64_at(96)?,
            gc_cycles: u32_at(24)?,
            gc_phase: u64::from(bytes[28]),
            goroutines: u64_at(112)?,
            run_queue: u32_at(32)?,
            global_run_queue: u64_at(120)?,
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
/// The network commitment input of [`Recorder::snapshot_with`].
#[derive(Clone, Copy)]
pub enum NetworkState<'a> {
    /// The instance's current network evidence bytes.
    Bytes(&'a [u8]),
    /// `commitment("network", bytes)` of the instance's current evidence.
    Digest([u8; 32]),
    /// Deferred: superseded by an eager write before it can be observed.
    Pending,
}

/// The storage commitment input of [`Recorder::snapshot_with`].
#[derive(Clone, Copy)]
pub enum StorageState<'a> {
    /// Fresh material: the cut carries the raw state and recovery bytes.
    Fresh { log: &'a [u8] },
    /// Material identical to the handle's previous fresh snapshot, whose
    /// commitments and retained log length the caller kept.
    Unchanged {
        storage: [u8; 32],
        recovery: [u8; 32],
        log_bytes: usize,
    },
}

/// The commitments one snapshot recorded.
#[derive(Clone, Copy)]
pub struct SnapshotDigests {
    pub network: Option<[u8; 32]>,
    pub storage: [u8; 32],
    pub recovery: [u8; 32],
    pub log_bytes: usize,
}

struct RecoveryCommitment {
    node: Box<str>,
    incarnation: u64,
    material: Box<[u8]>,
    digest: [u8; 32],
}

struct RecoveryCache {
    entries: [Option<RecoveryCommitment>; MAX_RECOVERY_CACHE_ENTRIES],
    bytes: usize,
    next: usize,
}

impl Default for RecoveryCache {
    fn default() -> Self {
        Self {
            entries: std::array::from_fn(|_| None),
            bytes: 0,
            next: 0,
        }
    }
}

impl RecoveryCache {
    fn commitment(&mut self, handle: &Handle, material: Vec<u8>) -> Vec<u8> {
        if let Some(index) = self.entries.iter().position(|entry| {
            entry.as_ref().is_some_and(|entry| {
                entry.node.as_ref() == handle.node && entry.incarnation == handle.incarnation
            })
        }) {
            let entry = self.entries[index].as_ref().unwrap();
            if entry.material.as_ref() == material {
                return entry.digest.to_vec();
            }
            self.evict(index);
        }
        let Some(required) = material
            .len()
            .checked_add(handle.node.len())
            .filter(|required| *required <= MAX_RECOVERY_CACHE_BYTES)
        else {
            return commitment("recovery", &material);
        };
        while self.bytes > MAX_RECOVERY_CACHE_BYTES - required
            || self.entries.iter().all(Option::is_some)
        {
            self.evict(self.next);
            self.next = (self.next + 1) % MAX_RECOVERY_CACHE_ENTRIES;
        }
        let digest = commitment("recovery", &material);
        let index = self.entries.iter().position(Option::is_none).unwrap();
        self.entries[index] = Some(RecoveryCommitment {
            node: handle.node.as_str().into(),
            incarnation: handle.incarnation,
            material: material.into_boxed_slice(),
            digest: digest.as_slice().try_into().unwrap(),
        });
        self.bytes += required;
        digest
    }

    fn evict(&mut self, index: usize) {
        if let Some(entry) = self.entries[index].take() {
            self.bytes -= entry.node.len() + entry.material.len();
        }
    }
}

pub struct Recorder {
    pub evidence: Evidence,
    order: u64,
    bytes: usize,
    pub exhausted: bool,
    pub sealed: bool,
    pub observed: bool,
    cuts: BTreeMap<Handle, Cut>,
    recovery: RecoveryCache,
    /// Handles whose cut holds a deferred network commitment.
    pending_network: std::collections::BTreeSet<Handle>,
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
            recovery: RecoveryCache::default(),
            pending_network: std::collections::BTreeSet::new(),
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
    #[cfg(test)]
    pub fn snapshot(&mut self, cut: Cut, network: &[u8], storage: &[u8]) -> Option<[u8; 32]> {
        self.snapshot_with(
            cut,
            NetworkState::Bytes(network),
            StorageState::Fresh { log: storage },
        )
        .map(|digests| digests.network.unwrap())
    }
    /// [`Self::snapshot`] with commitments the caller already holds for
    /// unchanged material, or with the network commitment deferred.
    ///
    /// A deferred (`Pending`) network commitment may only stand until the
    /// instance's next eager network write, which always precedes any
    /// boundary that could observe it; [`Self::boundary`] refuses one.
    pub fn snapshot_with(
        &mut self,
        mut cut: Cut,
        network: NetworkState<'_>,
        storage: StorageState<'_>,
    ) -> Option<SnapshotDigests> {
        if self.sealed || self.exhausted {
            return None;
        }
        let handle = cut.handle.clone();
        let prior = self.owner(&handle).streams.storage.len();
        let (storage_state, recovery_state) = match storage {
            StorageState::Fresh { log } => {
                if !self.reserve(base64_len(log.len()).saturating_sub(base64_len(prior))) {
                    return None;
                }
                self.owner(&handle).streams.storage = log.to_vec();
                let storage_state: [u8; 32] = commitment("storage", &cut.storage_state)
                    .try_into()
                    .unwrap();
                let recovery_state: [u8; 32] = self
                    .recovery
                    .commitment(&handle, std::mem::take(&mut cut.recovery_state))
                    .try_into()
                    .unwrap();
                (storage_state, recovery_state)
            }
            StorageState::Unchanged {
                storage,
                recovery,
                log_bytes,
            } => {
                // Unchanged material re-reserves nothing and rewrites the
                // identical log; anything else is a caller error.
                if prior != log_bytes {
                    self.exhausted = true;
                    return None;
                }
                (storage, recovery)
            }
        };
        cut.state_encoding = STATE_ENCODING.into();
        cut.storage_state = storage_state.to_vec();
        cut.recovery_state = recovery_state.to_vec();
        let network = match network {
            NetworkState::Bytes(bytes) => Some(commitment("network", bytes).try_into().unwrap()),
            NetworkState::Digest(digest) => Some(digest),
            NetworkState::Pending => None,
        };
        if let Some(digest) = network {
            cut.network_state = digest.to_vec();
            self.supersede_network(&handle);
        } else {
            cut.network_state = Vec::new();
            self.pending_network.insert(handle.clone());
        }
        let streams = &self.owner(&handle).streams;
        cut.choice_records = streams.choice.len() as u64 / 96;
        cut.choice_bytes = streams.choice.len() as u64;
        cut.wasi_bytes = streams.wasi.len() as u64;
        cut.network_bytes = streams.network.len() as u64;
        cut.storage_bytes = streams.storage.len() as u64;
        cut.stdout_bytes = streams.stdout.len() as u64;
        cut.stderr_bytes = streams.stderr.len() as u64;
        let log_bytes = streams.storage.len();
        self.cuts.insert(handle, cut);
        Some(SnapshotDigests {
            network,
            storage: storage_state,
            recovery: recovery_state,
            log_bytes,
        })
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The boundary count is rejected at 65536 before encoding its increment."
    )]
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
        // Deferred network commitments are superseded before any boundary;
        // one left behind fails the frame's cut validation below.
        debug_assert!(
            self.pending_network.is_empty(),
            "boundary would observe a deferred network commitment"
        );
        let cut = self.cuts.entry(handle.clone()).or_insert_with(|| Cut {
            handle: handle.clone(),
            state_encoding: STATE_ENCODING.into(),
            storage_state: commitment("storage", &[]),
            recovery_state: commitment("recovery", &[]),
            network_state: commitment("network", &[]),
            ..Default::default()
        });
        cut.witness = witness;
        cut.clock_run = clock;
        if self.evidence.boundaries.len() >= 65536 {
            self.exhausted = true;
            return Err(Error::msg("inspection boundary capacity"));
        }
        if world.len() != 32 {
            return Err(Error::msg("invalid inspection World commitment"));
        }
        let boundary = Boundary {
            handle: handle.clone(),
            decision: cut.witness.choices,
            trace: cut.witness.choices + cut.witness.observations,
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
        self.evidence.world.extend_from_slice(world);
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
        self.terminal_network_encoded(handle, bytes, "gomad.cluster-network-binary/v1");
    }
    pub(crate) fn terminal_network_json(&mut self, handle: &Handle, bytes: Vec<u8>) {
        self.terminal_network_encoded(handle, bytes, "");
    }
    fn terminal_network_encoded(&mut self, handle: &Handle, bytes: Vec<u8>, encoding: &str) {
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
        streams.network_encoding = encoding.into();
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_bytes = length;
            cut.network_state = state;
            self.supersede_network(handle);
        }
    }
    #[cfg(test)]
    pub fn network_snapshot(&mut self, handle: &Handle, bytes: &[u8]) {
        if self.sealed || self.exhausted {
            return;
        }
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_state = commitment("network", bytes);
            self.supersede_network(handle);
        }
    }
    /// [`Self::network_snapshot`] with an already computed commitment.
    pub fn network_digest(&mut self, handle: &Handle, digest: [u8; 32]) {
        if self.sealed || self.exhausted {
            return;
        }
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_state = digest.to_vec();
            self.supersede_network(handle);
        }
    }
    /// An eager network write supersedes `handle`'s deferred commitment.
    fn supersede_network(&mut self, handle: &Handle) {
        #[cfg(test)]
        if MISSED_SUPERSEDE.with(std::cell::Cell::get) {
            return;
        }
        self.pending_network.remove(handle);
    }
    /// Whether `handle` still has a deferred network commitment where an
    /// eager write must already have superseded it. Sealed or exhausted
    /// evidence records no further commitments, so nothing is pending there.
    pub fn deferred_network_pending(&self, handle: &Handle) -> bool {
        !self.sealed && !self.exhausted && self.pending_network.contains(handle)
    }
    /// [`Self::network_snapshot`] deferred: the caller has checked that the
    /// evidence would encode and guarantees an eager write supersedes this
    /// one before another instance can record a boundary.
    pub fn network_pending(&mut self, handle: &Handle) {
        if self.sealed || self.exhausted {
            return;
        }
        if let Some(cut) = self.cuts.get_mut(handle) {
            cut.network_state = Vec::new();
            self.pending_network.insert(handle.clone());
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
    pub world_commitment: Vec<u8>,
    pub time: u64,
    pub started: bool,
    pub dropped: u64,
    /// The storage commitments of this instance's last fresh snapshot, by
    /// namespace revision: unchanged namespaces are not re-encoded.
    pub storage_evidence: Option<StorageEvidence>,
}
/// See [`Transport::storage_evidence`].
#[derive(Clone, PartialEq, Eq)]
pub struct StorageEvidence {
    pub handle: Handle,
    pub revision: (u64, u64),
    pub storage: [u8; 32],
    pub recovery: [u8; 32],
    pub log_bytes: usize,
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
            world_commitment: commitment("world", &[]),
            time: 0,
            started: false,
            dropped: 0,
            storage_evidence: None,
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
        self.recorder
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The output-capacity check bounds payload length below MAX_OUTPUT."
    )]
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
    let mut identities = data[32..].as_chunks::<32>().0.to_vec();
    identities.sort_unstable();
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
#[expect(
    clippy::cast_possible_truncation,
    reason = "Handle node names are checked against 1024 bytes before encoding."
)]
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
#[expect(
    clippy::cast_possible_truncation,
    reason = "Cut counts are at most 65536, clock runs at most 1024 bytes, and the final frame is checked against MAX_METADATA."
)]
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
/// `commitment("network", evidence)` as the fixed-size cut value.
pub fn network_commitment(evidence: &[u8]) -> [u8; 32] {
    commitment("network", evidence).try_into().unwrap()
}

pub fn world_chain_commitment(world: &[u8; 32], network: Option<[u8; 32]>) -> Vec<u8> {
    let mut bytes = Vec::with_capacity(65);
    bytes.extend_from_slice(world);
    bytes.push(u8::from(network.is_some()));
    if let Some(network) = network {
        bytes.extend_from_slice(&network);
    }
    commitment("world", &bytes)
}
pub fn commitment(kind: &str, bytes: &[u8]) -> Vec<u8> {
    commitment_parts(kind, &[bytes])
}

pub(crate) fn commitment_parts(kind: &str, parts: &[&[u8]]) -> Vec<u8> {
    #[cfg(test)]
    if kind == "recovery" {
        RECOVERY_HASHES.with(|count| count.set(count.get() + 1));
    }
    let mut hash = Sha256::new();
    if kind == "world" {
        hash.update(b"gomad.inspection-world-sha256/v2\0");
    } else if kind == "network" {
        hash.update(b"gomad.inspection-state-sha256/v2/network\0");
    } else {
        hash.update(b"gomad.inspection-state-sha256/v1/");
        hash.update(kind.as_bytes());
        hash.update([0]);
    }
    hash.update(
        parts
            .iter()
            .map(|part| part.len() as u64)
            .sum::<u64>()
            .to_le_bytes(),
    );
    for part in parts {
        hash.update(part);
    }
    hash.finalize().to_vec()
}

#[cfg(test)]
thread_local! {
    static RECOVERY_HASHES: std::cell::Cell<usize> = const { std::cell::Cell::new(0) };
    static EAGER_ORACLE: std::cell::Cell<bool> = const { std::cell::Cell::new(false) };
    static MISSED_SUPERSEDE: std::cell::Cell<bool> = const { std::cell::Cell::new(false) };
}

/// Test fault: eager network writes on this thread leave the deferred
/// commitment pending, as a broken supersede would.
#[cfg(test)]
pub fn set_missed_supersede(missed: bool) {
    MISSED_SUPERSEDE.with(|flag| flag.set(missed));
}

/// Whether this test thread records through the original eager snapshots
/// (the differential oracle for incremental recording).
#[cfg(test)]
pub fn eager_oracle() -> bool {
    EAGER_ORACLE.with(std::cell::Cell::get)
}

#[cfg(test)]
pub fn set_eager_oracle(eager: bool) {
    EAGER_ORACLE.with(|flag| flag.set(eager));
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

    fn uncached_recovery(bytes: &[u8]) -> Vec<u8> {
        let mut hash = Sha256::new();
        hash.update(b"gomad.inspection-state-sha256/v1/recovery\0");
        hash.update((bytes.len() as u64).to_le_bytes());
        hash.update(bytes);
        hash.finalize().to_vec()
    }

    fn snapshot_uncached(
        recorder: &mut Recorder,
        mut cut: Cut,
        network: &[u8],
        storage: &[u8],
    ) -> Option<[u8; 32]> {
        if recorder.sealed || recorder.exhausted {
            return None;
        }
        let handle = cut.handle.clone();
        let prior = recorder.owner(&handle).streams.storage.len();
        if !recorder.reserve(base64_len(storage.len()).saturating_sub(base64_len(prior))) {
            return None;
        }
        let owner = recorder.owner(&handle);
        owner.streams.storage = storage.to_vec();
        cut.state_encoding = STATE_ENCODING.into();
        cut.storage_state = commitment("storage", &cut.storage_state);
        cut.recovery_state = uncached_recovery(&cut.recovery_state);
        cut.network_state = commitment("network", network);
        let network_identity = cut.network_state.as_slice().try_into().unwrap();
        let streams = &owner.streams;
        cut.choice_records = streams.choice.len() as u64 / 96;
        cut.choice_bytes = streams.choice.len() as u64;
        cut.wasi_bytes = streams.wasi.len() as u64;
        cut.network_bytes = streams.network.len() as u64;
        cut.storage_bytes = streams.storage.len() as u64;
        cut.stdout_bytes = streams.stdout.len() as u64;
        cut.stderr_bytes = streams.stderr.len() as u64;
        recorder.cuts.insert(handle, cut);
        Some(network_identity)
    }

    fn assert_recovery_cache_bound(recorder: &Recorder) {
        let cache = &recorder.recovery;
        let used: usize = cache
            .entries
            .iter()
            .flatten()
            .map(|entry| entry.node.len() + entry.material.len())
            .sum();
        assert_eq!(cache.bytes, used);
        assert!(used + std::mem::size_of_val(cache) <= MAX_METADATA);
        let mut owners = std::collections::BTreeSet::new();
        for entry in cache.entries.iter().flatten() {
            assert!(owners.insert((entry.node.as_ref(), entry.incarnation)));
        }
    }

    #[test]
    fn alternating_snapshot_handles_hash_unchanged_recovery_material_once_each() {
        let mut recorder = Recorder::default();
        let material = vec![b'r'; 256 << 10];
        let expected = uncached_recovery(&material);
        RECOVERY_HASHES.with(|count| count.set(0));
        for ordinal in 0..6 {
            let handle = Handle {
                node: if ordinal % 2 == 0 { "a" } else { "b" }.into(),
                incarnation: 1,
            };
            assert!(
                recorder
                    .snapshot(
                        Cut {
                            handle: handle.clone(),
                            recovery_state: material.clone(),
                            storage_state: vec![ordinal],
                            clock_run: vec![ordinal],
                            witness: witness_at(u64::from(ordinal)),
                            ..Default::default()
                        },
                        &[ordinal],
                        &[ordinal]
                    )
                    .is_some()
            );
            assert_eq!(recorder.cuts[&handle].recovery_state, expected);
        }
        assert_eq!(RECOVERY_HASHES.with(std::cell::Cell::get), 2);
    }

    #[test]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The fixture enumerates eight snapshots, so ordinals fit u8 and usize."
    )]
    fn recovery_cache_preserves_uncached_snapshot_and_evidence_bytes() {
        let mut cached = Recorder::default();
        let mut reference = Recorder::default();
        RECOVERY_HASHES.with(|count| count.set(0));
        for (ordinal, (node, incarnation, material)) in [
            ("a", 1, b"alpha".as_slice()),
            ("b", 1, b"alpha"),
            ("a", 1, b"alpha"),
            ("a", 1, b"alphA"),
            ("a", 2, b"alphA"),
            ("b", 1, b""),
            ("b", 1, b""),
            ("a", 1, b"alphA"),
        ]
        .into_iter()
        .enumerate()
        {
            let handle = Handle {
                node: node.into(),
                incarnation,
            };
            let ordinal = ordinal as u64;
            let cut = Cut {
                handle: handle.clone(),
                recovery_state: material.to_vec(),
                storage_state: ordinal.to_le_bytes().to_vec(),
                witness: witness_at(ordinal),
                clock_run: vec![ordinal as u8],
                ..Default::default()
            };
            let network = format!("fresh-network-{ordinal}");
            let log = vec![ordinal as u8; ordinal as usize + 1];
            for recorder in [&mut cached, &mut reference] {
                recorder.stream(&handle, "choice", &[ordinal as u8; 96]);
                recorder.stream(&handle, "transcript", &[ordinal as u8; 16]);
                recorder.stream(&handle, "stdout", &[ordinal as u8]);
            }
            let expected = snapshot_uncached(&mut reference, cut.clone(), network.as_bytes(), &log);
            assert_eq!(
                cached.snapshot(cut.clone(), network.as_bytes(), &log),
                expected
            );
            assert_eq!(
                cached.cuts[&handle].recovery_state,
                uncached_recovery(material)
            );
            assert_eq!(
                serde_json::to_value(&cached.cuts[&handle]).unwrap(),
                serde_json::to_value(&reference.cuts[&handle]).unwrap()
            );
            let world = commitment("world", &ordinal.to_le_bytes());
            for recorder in [&mut cached, &mut reference] {
                recorder
                    .boundary(
                        &handle,
                        witness_at(ordinal),
                        ordinal,
                        vec![ordinal as u8; 2],
                        &world,
                    )
                    .unwrap();
            }
            assert_recovery_cache_bound(&cached);
        }
        assert_eq!(RECOVERY_HASHES.with(std::cell::Cell::get), 5);
        let handle = Handle {
            node: "a".into(),
            incarnation: 2,
        };
        for recorder in [&mut cached, &mut reference] {
            recorder.terminal(&handle, true);
            recorder.terminal_network(&handle, b"terminal-network".to_vec());
            recorder.network_snapshot(&handle, b"fresh-terminal-cut");
        }
        assert_eq!(cached.bytes(true).unwrap(), reference.bytes(true).unwrap());
    }

    #[test]
    fn recovery_cache_bounds_keys_material_entry_pressure_and_oversized_bypass() {
        let mut recorder = Recorder::default();
        let snapshot = |recorder: &mut Recorder, handle: Handle, material: Vec<u8>| {
            let expected = uncached_recovery(&material);
            assert!(
                recorder
                    .snapshot(
                        Cut {
                            handle: handle.clone(),
                            recovery_state: material,
                            witness: witness_at(0),
                            ..Default::default()
                        },
                        &[],
                        &[]
                    )
                    .is_some()
            );
            assert_eq!(recorder.cuts[&handle].recovery_state, expected);
            assert!(!recorder.exhausted);
            assert_recovery_cache_bound(recorder);
        };
        let large_key = Handle {
            node: "k".repeat(1024),
            incarnation: 1,
        };
        snapshot(
            &mut recorder,
            large_key.clone(),
            vec![b'r'; MAX_RECOVERY_CACHE_BYTES - large_key.node.len()],
        );
        assert_eq!(
            recorder.recovery.bytes + std::mem::size_of::<RecoveryCache>(),
            MAX_METADATA
        );
        let handle = Handle {
            node: "a".into(),
            incarnation: 1,
        };
        snapshot(&mut recorder, handle.clone(), vec![]);
        assert_eq!(recorder.recovery.bytes, 1);
        assert!(
            !recorder
                .recovery
                .entries
                .iter()
                .flatten()
                .any(|entry| entry.node.as_ref() == large_key.node)
        );
        RECOVERY_HASHES.with(|count| count.set(0));
        for _ in 0..2 {
            snapshot(
                &mut recorder,
                handle.clone(),
                vec![b'x'; MAX_RECOVERY_CACHE_BYTES],
            );
            assert_eq!(recorder.recovery.bytes, 0);
        }
        assert_eq!(RECOVERY_HASHES.with(std::cell::Cell::get), 2);
        recorder = Recorder::default();
        for incarnation in 1..=MAX_RECOVERY_CACHE_ENTRIES as u64 + 1 {
            let mut material = Vec::with_capacity(1 << 20);
            material.extend_from_slice(b"abc");
            snapshot(
                &mut recorder,
                Handle {
                    node: "a".into(),
                    incarnation,
                },
                material,
            );
        }
        assert_eq!(
            recorder.recovery.entries.iter().flatten().count(),
            MAX_RECOVERY_CACHE_ENTRIES
        );
        assert_eq!(recorder.recovery.bytes, MAX_RECOVERY_CACHE_ENTRIES * 4);
        assert!(
            !recorder
                .recovery
                .entries
                .iter()
                .flatten()
                .any(|entry| entry.incarnation == 1)
        );
        snapshot(&mut recorder, handle, b"again".to_vec());
    }

    #[test]
    fn recovery_cache_keeps_original_snapshot_refusal_order() {
        for refusal in ["sealed", "exhausted", "storage-capacity", "owner-capacity"] {
            let mut cached = Recorder::default();
            let mut reference = Recorder::default();
            let handle = Handle {
                node: "a".into(),
                incarnation: 1,
            };
            let mut cut = Cut {
                handle,
                recovery_state: b"recovery".to_vec(),
                witness: witness_at(0),
                ..Default::default()
            };
            assert_eq!(
                cached.snapshot(cut.clone(), b"network", b"log"),
                snapshot_uncached(&mut reference, cut.clone(), b"network", b"log")
            );
            let retained = cached.recovery.bytes;
            let hashes = RECOVERY_HASHES.with(std::cell::Cell::get);
            for recorder in [&mut cached, &mut reference] {
                match refusal {
                    "sealed" => recorder.sealed = true,
                    "exhausted" => recorder.exhausted = true,
                    "storage-capacity" | "owner-capacity" => recorder.bytes = MAX_METADATA,
                    _ => unreachable!(),
                }
            }
            if refusal == "owner-capacity" {
                cut.handle.incarnation = 2;
            }
            cut.recovery_state = b"different".to_vec();
            assert_eq!(
                cached.snapshot(cut.clone(), b"new-network", b"longer-storage-log"),
                snapshot_uncached(&mut reference, cut, b"new-network", b"longer-storage-log")
            );
            assert_eq!(
                RECOVERY_HASHES.with(std::cell::Cell::get),
                hashes,
                "{refusal}"
            );
            assert_eq!(cached.recovery.bytes, retained);
            assert_eq!(cached.exhausted, reference.exhausted);
            assert_eq!(cached.order, reference.order);
            assert_eq!(cached.bytes, reference.bytes);
            assert_eq!(
                serde_json::to_value(cached.cuts.values().collect::<Vec<_>>()).unwrap(),
                serde_json::to_value(reference.cuts.values().collect::<Vec<_>>()).unwrap()
            );
            assert_eq!(
                cached.bytes(true).unwrap(),
                reference.bytes(true).unwrap(),
                "{refusal}"
            );
        }
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
            let _ = recorder.snapshot(
                Cut {
                    handle: handle.clone(),
                    storage_state: mount.clone(),
                    ..Default::default()
                },
                &[],
                &[],
            );
            recorder
                .boundary(
                    &handle,
                    witness_at(ordinal),
                    ordinal,
                    vec![],
                    &commitment("world", &world),
                )
                .unwrap();
            recorder.stream(&handle, "choice", &[0; 96]);
        }
        let encoded = recorder.bytes(true).unwrap();
        eprintln!(
            "inspection metadata: {} bytes for 2000 boundaries and repeated 64KiB mount/World",
            encoded.len()
        );
        assert!(recorder.evidence.complete);
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
        let hex = storage.iter().fold(String::new(), |mut hex, byte| {
            use std::fmt::Write as _;
            write!(&mut hex, "{byte:02x}").unwrap();
            hex
        });
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
    fn workflow_scale_inventory_fits_without_omitting_boundaries() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "history".into(),
            incarnation: 1,
        };
        let _ = recorder.snapshot(
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
                .boundary(
                    &handle,
                    witness_at(ordinal),
                    ordinal,
                    vec![],
                    &commitment("world", &[]),
                )
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

    /// Supplied commitments and deferred network writes record exactly what
    /// fresh material and evidence bytes record.
    #[test]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The fixture iterates ordinals from zero through five."
    )]
    fn snapshot_with_supplied_and_deferred_commitments_matches_fresh_snapshots() {
        let mut fresh = Recorder::default();
        let mut supplied = Recorder::default();
        let handle = Handle {
            node: "a".into(),
            incarnation: 1,
        };
        let mut retained = None;
        for ordinal in 0..6u64 {
            // Material changes on even ordinals only.
            let material = (ordinal / 2).to_le_bytes().to_vec();
            let network = format!("network-{}", ordinal / 3);
            let log = vec![b'l'; (ordinal / 2) as usize + 1];
            let cut = Cut {
                handle: handle.clone(),
                storage_state: material.clone(),
                recovery_state: material.clone(),
                witness: witness_at(ordinal),
                clock_run: vec![ordinal as u8],
                ..Default::default()
            };
            for recorder in [&mut fresh, &mut supplied] {
                recorder.stream(&handle, "choice", &[ordinal as u8; 96]);
            }
            fresh
                .snapshot(cut.clone(), network.as_bytes(), &log)
                .unwrap();
            let storage = match retained {
                Some(digests) if ordinal % 2 == 1 => {
                    let SnapshotDigests {
                        storage,
                        recovery,
                        log_bytes,
                        ..
                    } = digests;
                    StorageState::Unchanged {
                        storage,
                        recovery,
                        log_bytes,
                    }
                }
                _ => StorageState::Fresh { log: &log },
            };
            let mut cut = cut;
            if ordinal % 2 == 1 {
                cut.storage_state.clear();
                cut.recovery_state.clear();
            }
            let network_state = if ordinal % 3 == 2 {
                NetworkState::Pending
            } else {
                NetworkState::Digest(network_commitment(network.as_bytes()))
            };
            retained = supplied.snapshot_with(cut, network_state, storage);
            if ordinal % 3 == 2 {
                assert!(retained.unwrap().network.is_none());
                assert_eq!(supplied.pending_network.len(), 1);
                assert!(supplied.deferred_network_pending(&handle));
                supplied.exhausted = true;
                assert!(!supplied.deferred_network_pending(&handle));
                supplied.exhausted = false;
                supplied.network_digest(&handle, network_commitment(network.as_bytes()));
                assert!(supplied.pending_network.is_empty());
                assert!(!supplied.deferred_network_pending(&handle));
            }
            assert_eq!(
                serde_json::to_value(&supplied.cuts[&handle]).unwrap(),
                serde_json::to_value(&fresh.cuts[&handle]).unwrap()
            );
            let world = commitment("world", &ordinal.to_le_bytes());
            for recorder in [&mut fresh, &mut supplied] {
                recorder
                    .boundary(&handle, witness_at(ordinal), ordinal, vec![], &world)
                    .unwrap();
            }
        }
        assert_eq!(fresh.bytes(true).unwrap(), supplied.bytes(true).unwrap());
    }

    /// Commitments supplied for a log the recorder does not hold fail closed.
    #[test]
    fn snapshot_with_unchanged_storage_refuses_a_different_retained_log() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "a".into(),
            incarnation: 1,
        };
        let cut = Cut {
            handle,
            witness: witness_at(0),
            ..Default::default()
        };
        let digests = recorder
            .snapshot_with(
                cut.clone(),
                NetworkState::Bytes(b""),
                StorageState::Fresh { log: b"log" },
            )
            .unwrap();
        assert_eq!(digests.log_bytes, 3);
        assert!(
            recorder
                .snapshot_with(
                    cut,
                    NetworkState::Bytes(b""),
                    StorageState::Unchanged {
                        storage: digests.storage,
                        recovery: digests.recovery,
                        log_bytes: 2,
                    },
                )
                .is_none()
        );
        assert!(recorder.exhausted);
    }

    /// A deferred network commitment that is never superseded cannot be
    /// recorded in a boundary frame: the frame is refused, never published.
    #[test]
    #[cfg(not(debug_assertions))]
    fn boundary_refuses_a_deferred_network_commitment() {
        let mut recorder = Recorder::default();
        let handle = Handle {
            node: "a".into(),
            incarnation: 1,
        };
        let cut = Cut {
            handle: handle.clone(),
            witness: witness_at(0),
            ..Default::default()
        };
        recorder
            .snapshot_with(cut, NetworkState::Pending, StorageState::Fresh { log: b"" })
            .unwrap();
        let world = commitment("world", b"");
        assert!(
            recorder
                .boundary(&handle, witness_at(0), 0, vec![], &world)
                .is_err()
        );
        assert!(recorder.exhausted);
        assert!(recorder.evidence.boundaries.is_empty());
    }
}
