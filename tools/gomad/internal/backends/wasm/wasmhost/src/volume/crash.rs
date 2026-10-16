use super::hash::Hasher;
use super::state::{Error, Fs, Result, VolumeState, apply_persistence_effect};
use base64::{Engine, engine::general_purpose::STANDARD};
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
            seen_order = exploration.seen.clone();
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
                    data: (!object.data.is_empty()).then(|| object.data.clone()),
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
            hasher.u64(entry.mod_time as u64);
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
    use super::*;
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
    use super::*;
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

#[cfg(test)]
mod tests {
    use super::super::state::{Kind, VolumeConfig, VolumeLimits};
    use super::*;

    fn fixture() -> Fs {
        let mut fs = Fs::new(0);
        fs.configure_volumes(
            &[VolumeConfig {
                id: "data".into(),
                mount: "/workspace".into(),
                capacity_bytes: 1 << 20,
            }],
            VolumeLimits {
                pending_operations: 1000,
                transitions: 1000,
            },
            0,
        )
        .unwrap();
        let inode = fs.create("/workspace/db", Kind::File, 0o666, 100).unwrap();
        fs.write_at(inode, 0, b"value", 117).unwrap();
        fs
    }

    fn limits(work: u64) -> CrashEnumerationLimits {
        CrashEnumerationLimits {
            states: 64,
            operations: 64,
            depth: 64,
            bytes: 1 << 20,
            work,
        }
    }

    #[test]
    fn enumeration_matches_complete_go_state_and_identity_vector() {
        let fs = fixture();
        let got = fs
            .enumerate_crash_states("data", limits(1 << 16), None)
            .unwrap();
        let corpus: serde_json::Value = serde_json::from_str(include_str!(
            "../../../wasi/testdata/volume-conformance/work-resume-stale.json"
        ))
        .unwrap();
        assert_eq!(
            serde_json::to_value(got).unwrap(),
            corpus["observations"][2]["enumeration"]
        );
    }

    #[test]
    fn work_budget_counts_invalid_and_duplicate_cursors_and_completes() {
        let fs = fixture();
        let all = fs
            .enumerate_crash_states("data", limits(1 << 16), None)
            .unwrap();
        let mut exploration = None;
        let mut states = Vec::new();
        let mut empty_pages = 0;
        for page_number in 0..16 {
            let page = fs
                .enumerate_crash_states("data", limits(1), exploration.as_ref())
                .unwrap();
            let page_states = page.states.as_deref().unwrap_or(&[]);
            empty_pages += usize::from(page_states.is_empty());
            states.extend_from_slice(page_states);
            if page_number == 15 {
                assert!(page.complete);
            } else {
                assert_eq!(page.capacity, "work");
                assert!(!page.complete);
            }
            exploration = page.choice_exploration;
        }
        assert!(empty_pages > 0);
        assert_eq!(Some(states), all.states);
    }

    #[test]
    fn every_enumeration_capacity_matches_complete_go_reply_and_cursor() {
        let mut fs = Fs::new(0);
        fs.configure_volumes(
            &[VolumeConfig {
                id: "data".into(),
                mount: "/workspace".into(),
                capacity_bytes: 1 << 20,
            }],
            VolumeLimits {
                pending_operations: 1000,
                transitions: 1000,
            },
            0,
        )
        .unwrap();
        let inode = fs.create("/workspace/db", Kind::File, 0o666, 100).unwrap();
        fs.write_at(inode, 0, b"committed", 117).unwrap();
        for (fixture, limits) in [
            (
                include_str!("../../../wasi/testdata/volume-conformance/enumeration-states.json"),
                CrashEnumerationLimits {
                    states: 1,
                    ..limits(1 << 16)
                },
            ),
            (
                include_str!(
                    "../../../wasi/testdata/volume-conformance/enumeration-operations.json"
                ),
                CrashEnumerationLimits {
                    operations: 1,
                    ..limits(1 << 16)
                },
            ),
            (
                include_str!("../../../wasi/testdata/volume-conformance/enumeration-depth.json"),
                CrashEnumerationLimits {
                    depth: 1,
                    ..limits(1 << 16)
                },
            ),
            (
                include_str!("../../../wasi/testdata/volume-conformance/enumeration-bytes.json"),
                CrashEnumerationLimits {
                    bytes: 1,
                    ..limits(1 << 16)
                },
            ),
            (
                include_str!("../../../wasi/testdata/volume-conformance/enumeration-work.json"),
                limits(1),
            ),
        ] {
            let corpus: serde_json::Value = serde_json::from_str(fixture).unwrap();
            let before = fs.volume_snapshots();
            let got = fs.enumerate_crash_states("data", limits, None).unwrap();
            assert_eq!(
                serde_json::to_value(&got).unwrap(),
                corpus["observations"][2]["enumeration"],
                "capacity={}",
                got.capacity
            );
            assert_eq!(fs.volume_snapshots(), before);
        }
    }

    #[test]
    fn stale_invalid_duplicate_and_forged_cursors_are_refused() {
        let mut fs = fixture();
        let cursor = fs
            .enumerate_crash_states("data", limits(1), None)
            .unwrap()
            .choice_exploration
            .unwrap();
        let mut forged = cursor.clone();
        forged.identity.push('0');
        assert!(
            fs.enumerate_crash_states("data", limits(1), Some(&forged))
                .is_err()
        );
        let mut invalid = cursor.clone();
        invalid.cursor.as_mut().unwrap()[0] = 2;
        invalid.identity = invalid.identity();
        assert!(matches!(
            fs.enumerate_crash_states("data", limits(1), Some(&invalid)),
            Err(Error::Invalid(
                "crash enumeration exploration cursor is invalid"
            ))
        ));
        let mut duplicate = cursor.clone();
        duplicate.seen.extend_from_within(..);
        duplicate.identity = duplicate.identity();
        assert!(matches!(
            fs.enumerate_crash_states("data", limits(1), Some(&duplicate)),
            Err(Error::Invalid(
                "crash enumeration exploration contains duplicate state identities"
            ))
        ));
        let mut invalid_identity = cursor.clone();
        invalid_identity.seen[0] = "invalid".into();
        invalid_identity.identity = invalid_identity.identity();
        assert!(
            fs.enumerate_crash_states("data", limits(1), Some(&invalid_identity))
                .is_err()
        );
        fs.write_at(
            fs.lookup("/workspace/db").unwrap().inode,
            0,
            b"changed",
            200,
        )
        .unwrap();
        assert!(matches!(
            fs.enumerate_crash_states("data", limits(1), Some(&cursor)),
            Err(Error::Invalid(
                "crash enumeration exploration does not match pending volume state"
            ))
        ));
        assert!(fs.enumerate_crash_states("data", limits(0), None).is_err());
    }

    #[test]
    fn lifecycle_revokes_handles_keeps_history_and_clones_durable_state() {
        let mut fs = fixture();
        let inode = fs.lookup("/workspace/db").unwrap().inode;
        let handle = fs
            .open(
                "/workspace/db",
                super::super::state::OpenFlags {
                    read: true,
                    ..Default::default()
                },
                0,
                0,
            )
            .unwrap();
        fs.sync_node(inode).unwrap();
        fs.sync_node(fs.lookup("/workspace").unwrap().inode)
            .unwrap();
        fs.write_at(inode, 0, b"lost!", 200).unwrap();
        let before = fs.volume_snapshots();
        for selection in [
            BTreeMap::from([("data".into(), vec![999])]),
            BTreeMap::from([("data".into(), vec![5, 5])]),
            BTreeMap::from([("unknown".into(), vec![])]),
        ] {
            assert!(fs.crash_volumes(&selection).is_err());
            assert_eq!(fs.volume_snapshots(), before);
        }
        assert!(fs.select_crash_operations(0).is_empty());
        let mut next = fs.advance_lifecycle(false, &BTreeMap::new()).unwrap();
        assert!(matches!(
            fs.check_handle(&handle),
            Err(Error::Errno("stale"))
        ));
        assert!(matches!(
            fs.lookup("/workspace/db"),
            Err(Error::Errno("stale"))
        ));
        assert_eq!(next.read_at(inode, 0, 100).unwrap(), b"value");
        let mut clone = next.clone_durable().unwrap();
        clone.write_at(inode, 0, b"clone", 300).unwrap();
        assert_eq!(next.read_at(inode, 0, 100).unwrap(), b"value");
        assert_eq!(next.operations()["data"].len(), 5);
        let graceful = next.advance_lifecycle(true, &BTreeMap::new()).unwrap();
        assert_eq!(graceful.read_at(inode, 0, 100).unwrap(), b"value");
    }
}
