//! Projection of the actual Rust volume owner's authenticated GMSTORE state.

use crate::evidence::canonical;
use serde::{Deserialize, Serialize};
use sha2::Digest as _;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct Entry {
    pub path: String,
    pub mode: u32,
    pub kind: u64,
    pub mod_time: i64,
    pub data: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct State {
    pub volume: String,
    #[serde(rename = "PendingSHA256")]
    pub pending_sha256: String,
    pub selected_operations: Vec<u64>,
    pub entries: Vec<Entry>,
    pub identity: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct Enumeration {
    pub states: Option<Vec<State>>,
    pub choice_exploration: Option<serde_json::Value>,
    pub complete: bool,
    pub capacity: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct Operation {
    #[serde(rename = "ID")]
    pub id: u64,
    pub kind: String,
    pub dependencies: Option<Vec<u64>>,
    pub inode: u64,
    pub offset: u64,
    pub bytes: u64,
    #[serde(rename = "PayloadSHA256")]
    pub payload_sha256: String,
    #[serde(rename = "EffectSHA256")]
    pub effect_sha256: String,
    pub path: String,
    pub destination: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct Context {
    #[serde(rename = "PendingSHA256")]
    pub pending_sha256: String,
    pub operations: Vec<Operation>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Selection {
    pub primary_result_sha256: String,
    pub state: State,
    pub context: Context,
}

impl Selection {
    pub(crate) fn validate(&self) -> Result<(), String> {
        let native_digest = |value: &str| {
            value
                .strip_prefix("sha256:")
                .is_some_and(canonical::valid_digest)
        };
        if !canonical::valid_digest(&self.primary_result_sha256)
            || !native_digest(&self.state.identity)
            || !native_digest(&self.state.pending_sha256)
            || self.state.volume != "sqlite"
            || self.state.pending_sha256 != self.context.pending_sha256
            || self.context.operations.len() > 8
            || self.state.selected_operations.len() > 8
        {
            return Err("SQLite selection lacks exact primary/context binding".into());
        }
        let mut operations = BTreeMap::new();
        for operation in &self.context.operations {
            if operation.id == 0 || operations.insert(operation.id, operation).is_some() {
                return Err("duplicate or invalid pending operation".into());
            }
        }
        let selected = self
            .state
            .selected_operations
            .iter()
            .copied()
            .collect::<BTreeSet<_>>();
        if selected.len() != self.state.selected_operations.len()
            || selected.iter().any(|id| !operations.contains_key(id))
            || selected.iter().any(|id| {
                operations[id]
                    .dependencies
                    .as_ref()
                    .is_some_and(|dependencies| {
                        dependencies
                            .iter()
                            .any(|dependency| !selected.contains(dependency))
                    })
            })
        {
            return Err("SQLite selection is not dependency closed".into());
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Catalogue {
    pub handle: gomad_wasmhost::linked_cluster::Handle,
    pub volume: String,
    pub storage_state_sha256: String,
    pub enumeration: Enumeration,
    pub context: Context,
}

pub(super) fn sections(raw: &[u8]) -> Result<BTreeMap<u32, &[u8]>, String> {
    if raw.len() < 56 || raw.len() > 8 << 20 || raw.get(..8) != Some(b"GMSTORE\x01") {
        return Err("native storage state header/capacity".into());
    }
    let body = raw.len() - 40;
    let length = u64::from_le_bytes(
        raw[body..body + 8]
            .try_into()
            .map_err(|_| "storage state length")?,
    );
    if length != body as u64 || sha2::Sha256::digest(&raw[..body]).as_slice() != &raw[body + 8..] {
        return Err("native storage state integrity".into());
    }
    let integer = |start| {
        raw.get(start..start + 4)
            .and_then(|bytes| bytes.try_into().ok())
            .map(u32::from_le_bytes)
            .ok_or("truncated native storage state")
    };
    if integer(8)? != 3 || integer(12)? != 6 {
        return Err("native storage state lacks full crash context".into());
    }
    let mut offset = 16usize;
    let mut sections = BTreeMap::new();
    for expected in 1..=6 {
        let tag = integer(offset)?;
        let size = raw
            .get(offset + 4..offset + 12)
            .and_then(|bytes| bytes.try_into().ok())
            .map(u64::from_le_bytes)
            .ok_or("truncated storage section")?;
        offset = offset.checked_add(12).ok_or("storage section overflow")?;
        let end = usize::try_from(size)
            .ok()
            .and_then(|size| offset.checked_add(size))
            .filter(|end| *end <= body)
            .ok_or("storage section capacity")?;
        if tag != expected {
            return Err("unordered native storage sections".into());
        }
        sections.insert(tag, &raw[offset..end]);
        offset = end;
    }
    if offset != body {
        return Err("trailing native storage state".into());
    }
    Ok(sections)
}

pub(crate) fn capture(raw: &[u8], log: &[u8]) -> Result<Catalogue, String> {
    let sections = sections(raw)?;
    let log_binding = sections[&5];
    if log_binding.len() != 40
        || u64::from_le_bytes(
            log_binding[..8]
                .try_into()
                .map_err(|_| "storage log length")?,
        ) != log.len() as u64
        || sha2::Sha256::digest(log).as_slice() != &log_binding[8..]
    {
        return Err("native crash catalogue log binding".into());
    }
    let mut enumerations: BTreeMap<String, Enumeration> = super::super::spec::strict(sections[&3])?;
    let mut contexts: BTreeMap<String, Context> = super::super::spec::strict(sections[&6])?;
    let enumeration = enumerations
        .remove("sqlite")
        .ok_or("missing SQLite native enumeration")?;
    let context = contexts
        .remove("sqlite")
        .ok_or("missing SQLite pending context")?;
    let states = enumeration
        .states
        .as_ref()
        .ok_or("missing SQLite crash states")?;
    if !enumeration.complete
        || !enumeration.capacity.is_empty()
        || enumeration.choice_exploration.is_some()
        || states.len() != 4
        || !enumerations.is_empty()
        || !contexts.is_empty()
    {
        return Err("actual complete reviewed four-state SQLite catalogue required".into());
    }
    let mut identities = BTreeSet::new();
    for state in states {
        let selection = Selection {
            primary_result_sha256: "0".repeat(64),
            state: state.clone(),
            context: context.clone(),
        };
        selection.validate()?;
        if state.identity.is_empty()
            || !identities.insert(&state.identity)
            || state.entries.len() > 32768
        {
            return Err("invalid or duplicate SQLite crash state".into());
        }
    }
    Ok(Catalogue {
        handle: gomad_wasmhost::linked_cluster::Handle {
            node: "sqlite".into(),
            incarnation: 2,
        },
        volume: "sqlite".into(),
        storage_state_sha256: canonical::digest(raw),
        enumeration,
        context,
    })
}

pub(crate) fn marker(raw: &[u8], selected: &State) -> Result<(), String> {
    let sections = sections(raw)?;
    let snapshots: serde_json::Value = super::super::spec::strict(sections[&1])?;
    let snapshot = snapshots
        .as_array()
        .and_then(|rows| rows.iter().find(|row| row["ID"] == "sqlite"))
        .ok_or("recovered SQLite volume missing")?;
    let marker = snapshot["Persisted"]
        .as_array()
        .ok_or("recovered SQLite entries missing")?
        .iter()
        .find(|row| row["Path"] == "/marker");
    let expected = selected
        .entries
        .iter()
        .find(|entry| entry.path == "/marker");
    match (expected, marker) {
        (None, None) => Ok(()),
        (Some(expected), Some(actual)) => {
            let bytes = match expected.data.as_deref() {
                Some(data) => {
                    super::super::cells::base64(data).ok_or("invalid selected marker bytes")?
                }
                None => Vec::new(),
            };
            let (kind, hash) = match expected.kind {
                1 => ("file", format!("sha256:{}", canonical::digest(&bytes))),
                2 => ("directory", String::new()),
                _ => return Err("unsupported marker kind".into()),
            };
            if actual["Kind"] != kind
                || actual["Mode"] != expected.mode
                || actual["ModTime"] != expected.mod_time
                || actual["Size"] != bytes.len() as u64
                || actual["DataSHA256"] != hash
            {
                return Err("recovered marker differs from exact selected native state".into());
            }
            Ok(())
        }
        _ => Err("recovered marker presence differs from exact selected native state".into()),
    }
}
