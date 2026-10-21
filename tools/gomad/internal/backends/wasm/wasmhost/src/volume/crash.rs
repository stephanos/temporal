use super::hash::Hasher;
use super::state::{Error, Fs, Result, VolumeState, apply_persistence_effect};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Copy, Debug, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct CrashEnumerationLimits {
    pub states: u64,
    pub operations: u64,
    pub depth: u64,
    pub bytes: u64,
    pub work: u64,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub(crate) struct CrashExploration {
    pub volume: String,
    #[serde(rename = "PendingSHA256")]
    pub pending_sha256: String,
    #[serde(with = "optional_bytes")]
    pub cursor: Option<Vec<u8>>,
    #[serde(with = "nullable_strings")]
    pub seen: Vec<String>,
    pub identity: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub(crate) struct CrashEntry {
    pub path: String,
    pub mode: u32,
    pub kind: u64,
    pub mod_time: i64,
    #[serde(with = "optional_bytes")]
    pub data: Option<Vec<u8>>,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub(crate) struct CrashState {
    pub volume: String,
    #[serde(rename = "PendingSHA256")]
    pub pending_sha256: String,
    pub selected_operations: Vec<u64>,
    pub entries: Vec<CrashEntry>,
    pub identity: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub(crate) struct CrashEnumeration {
    pub states: Option<Vec<CrashState>>,
    pub choice_exploration: Option<CrashExploration>,
    pub complete: bool,
    pub capacity: String,
}

impl CrashEnumeration {
    fn empty() -> Self {
        Self {
            states: None,
            choice_exploration: None,
            complete: false,
            capacity: String::new(),
        }
    }
}

impl Fs {
    #[cfg(not(feature = "conformance"))]
    pub(crate) fn explore_persistence(
        &mut self,
        controller: &crate::linked_simulation::Shared,
    ) -> std::result::Result<(), String> {
        use crate::linked_simulation::{Dimension, domain_hash};
        use serde_json::json;

        for (id, volume) in &mut self.volumes {
            let pending: BTreeSet<_> = volume
                .pending
                .iter()
                .map(|operation| operation.id)
                .collect();
            let ready: Vec<_> = volume
                .pending
                .iter()
                .filter(|operation| {
                    operation
                        .deps
                        .iter()
                        .all(|dependency| !pending.contains(dependency))
                })
                .map(|operation| operation.id)
                .collect();
            if ready.is_empty() {
                continue;
            }
            let context = volume.pending_identity();
            let site = domain_hash(
                "gomad-storage-persistence-site/v1",
                &json!({"volume":id,"pending":context}),
            )?;
            let alternatives = std::iter::once(None)
                .chain(ready.iter().copied().map(Some))
                .map(|operation| {
                    domain_hash(
                        "gomad-storage-persistence-alternative/v1",
                        &json!({
                            "volume":id,"pending":context,"operation":operation
                        }),
                    )
                })
                .collect::<std::result::Result<Vec<_>, _>>()?;
            let selected = controller
                .lock()
                .map_err(|_| "simulation exploration controller poisoned")?
                .choose(Dimension::Storage, &site, alternatives, 0)?;
            if selected > 0 {
                volume
                    .persist_selected(&BTreeSet::from([ready[selected as usize - 1]]))
                    .map_err(|error| super::faults::error_text(&error))?;
            }
        }
        Ok(())
    }

    pub(crate) fn enumerate_crash_states(
        &self,
        id: &str,
        limits: CrashEnumerationLimits,
        exploration: Option<&CrashExploration>,
    ) -> Result<CrashEnumeration> {
        let volume = self.volumes.get(id).ok_or(Error::Errno("no-entry"))?;
        if [
            limits.states,
            limits.operations,
            limits.depth,
            limits.bytes,
            limits.work,
        ]
        .contains(&0)
        {
            return Err(Error::Invalid("crash enumeration limits must be nonzero"));
        }
        let pending = volume.pending_identity();
        let mut cursor = vec![0; volume.pending.len()];
        let mut seen = BTreeSet::new();
        let mut seen_order = Vec::new();
        if let Some(exploration) = exploration {
            exploration.validate(id, &pending, volume.pending.len())?;
            cursor = exploration.cursor.clone().unwrap_or_default();
            seen.extend(exploration.seen.iter().cloned());
            seen_order.clone_from(&exploration.seen);
        }
        if volume.pending.len() as u64 > limits.operations {
            return Ok(volume.capacity_page(&pending, cursor, seen_order, "operations"));
        }
        let mut page = CrashEnumeration::empty();
        let mut retained = 0;
        for _ in 0..limits.work {
            let selected: BTreeSet<_> = volume
                .pending
                .iter()
                .zip(&cursor)
                .filter(|(_, value)| **value != 0)
                .map(|(operation, _)| operation.id)
                .collect();
            let (next, complete) = increment_cursor(&cursor);
            if selected.len() as u64 > limits.depth {
                return Ok(volume.capacity_page(&pending, cursor, seen_order, "depth"));
            }
            if volume.selection_valid(&selected) {
                let state = volume.crash_state(&selected, &pending)?;
                if !seen.contains(&state.identity) {
                    let state_bytes = state.retained_bytes();
                    if state_bytes > limits.bytes - retained {
                        return Ok(volume.capacity_page(&pending, cursor, seen_order, "bytes"));
                    }
                    retained += state_bytes;
                    seen.insert(state.identity.clone());
                    seen_order.push(state.identity.clone());
                    page.states.get_or_insert_with(Vec::new).push(state);
                }
            }
            cursor = next;
            if complete {
                page.complete = true;
                return Ok(page);
            }
            if page.states.as_ref().map_or(0, Vec::len) as u64 == limits.states {
                page.capacity = "states".into();
                page.choice_exploration =
                    Some(CrashExploration::new(id, &pending, cursor, seen_order));
                return Ok(page);
            }
        }
        page.capacity = "work".into();
        page.choice_exploration = Some(CrashExploration::new(id, &pending, cursor, seen_order));
        Ok(page)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; seeded selection is retained alongside the WASI subset."
    )]
    pub(crate) fn select_crash_operations(&self, seed: u64) -> BTreeMap<String, Vec<u64>> {
        let mut result = BTreeMap::new();
        let mut state = seed;
        for (id, volume) in &self.volumes {
            let pending: BTreeSet<_> = volume
                .pending
                .iter()
                .map(|operation| operation.id)
                .collect();
            let mut selected = BTreeSet::new();
            for operation in &volume.pending {
                let eligible = operation.deps.iter().all(|dependency| {
                    !pending.contains(dependency) || selected.contains(dependency)
                });
                state ^= state << 13;
                state ^= state >> 7;
                state ^= state << 17;
                if eligible && state & 1 != 0 {
                    selected.insert(operation.id);
                    result
                        .entry(id.clone())
                        .or_insert_with(Vec::new)
                        .push(operation.id);
                }
            }
        }
        result
    }

    pub(crate) fn crash_volumes(&mut self, selections: &BTreeMap<String, Vec<u64>>) -> Result<()> {
        if !self.configured {
            return Err(Error::Invalid("simulation volumes are not configured"));
        }
        let mut next = BTreeMap::new();
        for (id, volume) in &self.volumes {
            let mut selected = BTreeSet::new();
            for operation in selections.get(id).into_iter().flatten() {
                if !selected.insert(*operation) {
                    return Err(Error::Invalid(
                        "crash selection contains a duplicate operation",
                    ));
                }
            }
            if !volume.selection_valid(&selected) {
                return Err(Error::Invalid("crash selection is not dependency-closed"));
            }
            if selected
                .iter()
                .any(|operation| !volume.has_pending(*operation))
            {
                return Err(Error::Invalid(
                    "crash selection contains an unknown operation",
                ));
            }
            let mut candidate = volume.persisted.clone();
            for operation in &volume.pending {
                if selected.contains(&operation.id) {
                    apply_persistence_effect(&mut candidate, &operation.effect)?;
                }
            }
            next.insert(id.clone(), candidate);
        }
        if selections.keys().any(|id| !self.volumes.contains_key(id)) {
            return Err(Error::Invalid(
                "crash selection refers to an unknown volume",
            ));
        }
        self.preflight_rebuild()?;
        for (id, volume) in &mut self.volumes {
            volume.persisted = next.remove(id).unwrap();
            volume.pending.clear();
            volume.last.clear();
        }
        self.rebuild_after_lifecycle()
    }

    pub(crate) fn flush_volumes(&mut self) -> Result<()> {
        self.preflight_rebuild()?;
        for volume in self.volumes.values_mut() {
            let selected = volume
                .pending
                .iter()
                .map(|operation| operation.id)
                .collect();
            volume.persist_selected(&selected)?;
        }
        self.rebuild_after_lifecycle()
    }

    pub(crate) fn advance_lifecycle(
        &mut self,
        graceful: bool,
        selections: &BTreeMap<String, Vec<u64>>,
    ) -> Result<Fs> {
        let allocations = self
            .rebuild_inode_allocations()
            .checked_mul(2)
            .ok_or(Error::Errno("no-space"))?;
        self.preflight_inode_allocations(allocations)?;
        if graceful {
            self.flush_volumes()?;
        } else {
            self.crash_volumes(selections)?;
        }
        let next = self.clone_durable()?;
        self.unavailable = true;
        Ok(next)
    }

    pub(crate) fn clone_durable(&self) -> Result<Fs> {
        self.preflight_inode_allocations(self.rebuild_inode_allocations())?;
        let mut next = Fs::new(0);
        next.next_inode = self.next_inode;
        next.limits = self.limits;
        next.configured = self.configured;
        for (id, volume) in &self.volumes {
            next.volumes.insert(
                id.clone(),
                VolumeState {
                    id: id.clone(),
                    mount: volume.mount.clone(),
                    capacity_bytes: volume.capacity_bytes,
                    persisted: volume.persisted.clone(),
                    pending: Vec::new(),
                    history: volume.history.clone(),
                    next_operation: volume.next_operation,
                    last: BTreeMap::new(),
                },
            );
        }
        next.rebuild_after_lifecycle()?;
        Ok(next)
    }
}

impl VolumeState {
    fn crash_state(&self, selected: &BTreeSet<u64>, pending: &str) -> Result<CrashState> {
        let mut state = self.persisted.clone();
        let mut selected_operations = Vec::new();
        for operation in &self.pending {
            if selected.contains(&operation.id) {
                apply_persistence_effect(&mut state, &operation.effect)?;
                selected_operations.push(operation.id);
            }
        }
        let entries = state
            .paths
            .iter()
            .map(|(path, inode)| {
                let object = &state.objects[inode];
                CrashEntry {
                    path: path.clone(),
                    mode: object.mode,
                    kind: object.kind as u64,
                    mod_time: object.mod_time,
                    data: (!object.data.is_empty()).then(|| object.data.to_vec()),
                }
            })
            .collect();
        let mut result = CrashState {
            volume: self.id.clone(),
            pending_sha256: pending.into(),
            selected_operations,
            entries,
            identity: String::new(),
        };
        result.identity = result.identity();
        Ok(result)
    }

    fn capacity_page(
        &self,
        pending: &str,
        cursor: Vec<u8>,
        seen: Vec<String>,
        capacity: &str,
    ) -> CrashEnumeration {
        CrashEnumeration {
            states: None,
            choice_exploration: Some(CrashExploration::new(&self.id, pending, cursor, seen)),
            complete: false,
            capacity: capacity.into(),
        }
    }
}

impl CrashExploration {
    fn new(volume: &str, pending: &str, cursor: Vec<u8>, seen: Vec<String>) -> Self {
        let mut exploration = Self {
            volume: volume.into(),
            pending_sha256: pending.into(),
            cursor: (!cursor.is_empty()).then_some(cursor),
            seen,
            identity: String::new(),
        };
        exploration.identity = exploration.identity();
        exploration
    }

    fn identity(&self) -> String {
        let mut hasher = Hasher::new();
        hasher.string("gomad-volume-crash-exploration/v1");
        hasher.string(&self.volume);
        hasher.string(&self.pending_sha256);
        hasher.bytes(self.cursor.as_deref().unwrap_or(&[]));
        for identity in &self.seen {
            hasher.string(identity);
        }
        hasher.finish()
    }

    fn validate(&self, volume: &str, pending: &str, operations: usize) -> Result<()> {
        let cursor = self.cursor.as_deref().unwrap_or(&[]);
        if self.volume != volume
            || self.pending_sha256 != pending
            || cursor.len() != operations
            || self.identity != self.identity()
        {
            return Err(Error::Invalid(
                "crash enumeration exploration does not match pending volume state",
            ));
        }
        if cursor.iter().any(|value| *value > 1) {
            return Err(Error::Invalid(
                "crash enumeration exploration cursor is invalid",
            ));
        }
        let mut seen = BTreeSet::new();
        for identity in &self.seen {
            if !valid_digest(identity) {
                return Err(Error::Invalid(
                    "crash enumeration exploration contains an invalid state identity",
                ));
            }
            if !seen.insert(identity) {
                return Err(Error::Invalid(
                    "crash enumeration exploration contains duplicate state identities",
                ));
            }
        }
        Ok(())
    }
}

impl CrashState {
    fn identity(&self) -> String {
        let mut hasher = Hasher::new();
        hasher.string("gomad-volume-crash-state/v1");
        hasher.string(&self.volume);
        for entry in &self.entries {
            hasher.string(&entry.path);
            hasher.u64(u64::from(entry.mode));
            hasher.u64(entry.kind);
            hasher.u64(entry.mod_time.cast_unsigned());
            hasher.bytes(entry.data.as_deref().unwrap_or(&[]));
        }
        hasher.finish()
    }
    fn retained_bytes(&self) -> u64 {
        let mut bytes = (self.volume.len() + self.pending_sha256.len() + self.identity.len())
            as u64
            + self.selected_operations.len() as u64 * 8;
        for entry in &self.entries {
            bytes += entry.path.len() as u64 + entry.data.as_ref().map_or(0, Vec::len) as u64 + 32;
        }
        bytes
    }
}

fn increment_cursor(cursor: &[u8]) -> (Vec<u8>, bool) {
    let mut next = cursor.to_vec();
    for value in next.iter_mut().rev() {
        if *value == 0 {
            *value = 1;
            return (next, false);
        }
        *value = 0;
    }
    (next, true)
}

pub(super) fn valid_digest(value: &str) -> bool {
    value.strip_prefix("sha256:").is_some_and(|digest| {
        digest.len() == 64
            && digest
                .bytes()
                .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
    })
}

mod optional_bytes {
    use base64::{Engine as _, engine::general_purpose::STANDARD};
    use serde::{Deserialize, Serialize};
    #[expect(
        clippy::ref_option,
        reason = "Serde serializes the optional field through a shared reference"
    )]
    pub fn serialize<S: serde::Serializer>(
        value: &Option<Vec<u8>>,
        serializer: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        value
            .as_ref()
            .map(|value| STANDARD.encode(value))
            .serialize(serializer)
    }
    pub fn deserialize<'de, D: serde::Deserializer<'de>>(
        deserializer: D,
    ) -> std::result::Result<Option<Vec<u8>>, D::Error> {
        Option::<String>::deserialize(deserializer)?
            .map(|value| STANDARD.decode(value).map_err(serde::de::Error::custom))
            .transpose()
    }
}

mod nullable_strings {
    use serde::{Deserialize, Serialize};
    pub fn serialize<S: serde::Serializer>(
        value: &[String],
        serializer: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        (!value.is_empty()).then_some(value).serialize(serializer)
    }
    pub fn deserialize<'de, D: serde::Deserializer<'de>>(
        deserializer: D,
    ) -> std::result::Result<Vec<String>, D::Error> {
        Ok(Option::<Vec<String>>::deserialize(deserializer)?.unwrap_or_default())
    }
}
