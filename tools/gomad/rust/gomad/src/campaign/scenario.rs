use crate::{
    evidence::canonical,
    replay::{cluster, preflight},
};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    net::IpAddr,
    time::Duration,
};

macro_rules! scenario_limits {
    ($($name:ident: $maximum:expr),+ $(,)?) => {
        #[derive(Deserialize, Serialize)]
        #[serde(deny_unknown_fields)]
        struct Limits { $( $name: u64, )+ }
        impl Limits {
            fn validate(&self) -> Result<(), String> {
                $(if self.$name == 0 || self.$name > $maximum {
                    return Err(format!("invalid simulation {} limit", stringify!($name)));
                })+
                Ok(())
            }
        }
    };
}

scenario_limits! {
    nodes: 64, directional_links: 4096, volumes: 64, scenario_actions: 4096,
    scenario_decisions: 4096, fault_actions: 4096, history_operations: 65536,
    observations: 65536, oracle_results: 4096, scenario_evidence_bytes: 64 << 20,
    boot_config_bytes: 1 << 20, volume_capacity_bytes: 1 << 30,
    observation_bytes: 64 << 20, network_listeners: 4096,
    network_connections: 65536, network_bytes: 1 << 30,
    network_connection_bytes: 1 << 30, network_stall_nanos: i64::MAX as u64 - 1,
    network_transitions: 1 << 20, volume_operations: 1 << 20,
    volume_transitions: 1 << 20, crash_states: 65536, crash_depth: 256,
    crash_bytes: 128 << 20, crash_wall_nanos: 60_000_000_000,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Scenario {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    workload: Option<crate::qualify::workloads::Workload>,
    schema: String,
    spec: Spec,
    boots: Vec<Boot>,
    actions: Vec<helper::Step>,
    #[serde(default, deserialize_with = "nullable_vec")]
    incarnation_boots: Vec<Binding>,
    #[serde(default, deserialize_with = "nullable_vec")]
    output_crashes: Vec<Boundary>,
    #[serde(default, deserialize_with = "nullable_vec")]
    output_stops: Vec<Boundary>,
    #[serde(default, deserialize_with = "nullable_vec")]
    network_faults: Vec<NetworkFault>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Spec {
    schema: String,
    backend: String,
    fidelity: String,
    #[serde(default)]
    seed: u64,
    limits: Limits,
    nodes: Vec<Node>,
    #[serde(default, deserialize_with = "nullable_vec")]
    links: Vec<Link>,
    #[serde(default, deserialize_with = "nullable_vec")]
    volumes: Vec<Volume>,
    #[serde(default)]
    faults: Option<Value>,
    #[serde(default)]
    exploration: Option<Value>,
    #[serde(default)]
    scenario_choice_plan: Option<Value>,
    #[serde(default)]
    replay: Option<Value>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Node {
    id: String,
    boot: String,
    address: String,
    #[serde(default)]
    config: Option<Binary>,
    #[serde(default, deserialize_with = "nullable_vec")]
    volumes: Vec<Mount>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Mount {
    volume: String,
    path: String,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Link {
    from: String,
    to: String,
    #[serde(default)]
    enabled: bool,
    #[serde(default)]
    delay_nanos: u64,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Volume {
    id: String,
    capacity_bytes: u64,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Boot {
    id: String,
    #[serde(default)]
    args: Option<Vec<String>>,
    #[serde(default)]
    storage: Option<Storage>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Binding {
    handle: helper::Handle,
    boot: String,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Boundary {
    handle: helper::Handle,
    marker: Binary,
}

#[derive(Deserialize, Serialize)]
#[serde(untagged)]
enum Binary {
    Base64(String),
    Bytes(Vec<u8>),
}

impl Binary {
    fn is_empty(&self) -> bool {
        match self {
            Self::Base64(value) => value.is_empty(),
            Self::Bytes(value) => value.is_empty(),
        }
    }
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
struct NetworkFault {
    name: String,
    action: String,
    from: String,
    to: String,
    #[serde(default)]
    at_nanos: u64,
    #[serde(default)]
    delay_nanos: u64,
    #[serde(default)]
    symmetric: bool,
}

// Field order and omission rules here bind the existing helper's Go storage-model identity.
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Storage {
    volumes: Vec<StorageVolume>,
    limits: StorageLimits,
    #[serde(
        default,
        deserialize_with = "nullable_vec",
        skip_serializing_if = "Vec::is_empty"
    )]
    faults: Vec<StorageFault>,
    #[serde(
        default,
        deserialize_with = "nullable_vec",
        skip_serializing_if = "Vec::is_empty"
    )]
    sqlite_lock_databases: Vec<String>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
struct StorageVolume {
    #[serde(rename = "ID")]
    id: String,
    path: String,
    capacity_bytes: u64,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
struct StorageLimits {
    pending_operations: u64,
    transitions: u64,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct StorageFault {
    name: String,
    resource: String,
    operation: String,
    occurrence: u64,
    action: String,
    #[serde(default, skip_serializing_if = "is_zero")]
    bytes: u64,
}

#[expect(
    clippy::trivially_copy_pass_by_ref,
    reason = "Serde skip_serializing_if predicates borrow their field"
)]
fn is_zero(value: &u64) -> bool {
    *value == 0
}

fn nullable_vec<'de, D, T>(deserializer: D) -> Result<Vec<T>, D::Error>
where
    D: serde::Deserializer<'de>,
    T: Deserialize<'de>,
{
    Ok(Option::<Vec<T>>::deserialize(deserializer)?.unwrap_or_default())
}

fn valid_id(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 128
        && value.as_bytes()[0].is_ascii_lowercase()
        && value.split('-').all(|part| {
            !part.is_empty()
                && part
                    .bytes()
                    .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit())
        })
}

fn clean_path(value: &str) -> bool {
    value.starts_with('/')
        && value != "/"
        && value.len() <= 4096
        && !value.contains(['\0', '\\'])
        && !value.ends_with('/')
        && value[1..]
            .split('/')
            .all(|part| !matches!(part, "" | "." | ".."))
}

pub(crate) fn canonical_base64(value: &str) -> Result<Vec<u8>, String> {
    let bytes = marker(&Binary::Base64(value.into()))?;
    if super::exploration::base64(&bytes) != value {
        return Err("noncanonical base64 payload".into());
    }
    Ok(bytes)
}

fn marker(value: &Binary) -> Result<Vec<u8>, String> {
    let value = match value {
        Binary::Bytes(bytes) if !bytes.is_empty() && bytes.len() <= 4096 => {
            return Ok(bytes.clone());
        }
        Binary::Bytes(_) => return Err("invalid WASM lifecycle marker".into()),
        Binary::Base64(value) => value,
    };
    let value: Vec<_> = value
        .bytes()
        .filter(|byte| !matches!(byte, b'\r' | b'\n'))
        .collect();
    if value.is_empty() || value.len() % 4 != 0 || value.len() > 4096_usize.div_ceil(3) * 4 {
        return Err("invalid WASM lifecycle marker".into());
    }
    let mut decoded = Vec::new();
    for (index, chunk) in value.as_chunks::<4>().0.iter().enumerate() {
        let mut digits = [0u8; 4];
        let mut padding = 0;
        for (offset, byte) in chunk.iter().copied().enumerate() {
            digits[offset] = match byte {
                b'A'..=b'Z' => byte - b'A',
                b'a'..=b'z' => byte - b'a' + 26,
                b'0'..=b'9' => byte - b'0' + 52,
                b'+' => 62,
                b'/' => 63,
                b'=' if offset >= 2 && index + 1 == value.len() / 4 => {
                    padding += 1;
                    0
                }
                _ => return Err("invalid WASM lifecycle marker base64".into()),
            };
            if padding != 0 && byte != b'=' {
                return Err("invalid WASM lifecycle marker padding".into());
            }
        }
        decoded.push(digits[0] << 2 | digits[1] >> 4);
        if padding < 2 {
            decoded.push(digits[1] << 4 | digits[2] >> 2);
        }
        if padding == 0 {
            decoded.push(digits[2] << 6 | digits[3]);
        }
    }
    if decoded.len() > 4096 {
        return Err("WASM lifecycle marker capacity".into());
    }
    Ok(decoded)
}

fn parse(raw: &[u8]) -> Result<Scenario, String> {
    if raw.is_empty() || raw.len() > 256 << 10 {
        return Err("WASM scenario size exceeds bounds".into());
    }
    serde_json::from_slice::<UniqueNames>(raw).map_err(|error| error.to_string())?;
    let scenario: Scenario = serde_json::from_slice(raw).map_err(|error| error.to_string())?;
    scenario.validate()?;
    Ok(scenario)
}

pub(super) struct UniqueNames;

impl<'de> Deserialize<'de> for UniqueNames {
    fn deserialize<D: serde::Deserializer<'de>>(decoder: D) -> Result<Self, D::Error> {
        struct Visitor;
        impl<'de> serde::de::Visitor<'de> for Visitor {
            type Value = UniqueNames;
            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("JSON without duplicate object names")
            }
            fn visit_map<M: serde::de::MapAccess<'de>>(
                self,
                mut map: M,
            ) -> Result<UniqueNames, M::Error> {
                let mut names = BTreeSet::new();
                while let Some(name) = map.next_key::<String>()? {
                    if !names.insert(name) {
                        return Err(serde::de::Error::custom("duplicate scenario object name"));
                    }
                    map.next_value::<UniqueNames>()?;
                }
                Ok(UniqueNames)
            }
            fn visit_seq<S: serde::de::SeqAccess<'de>>(
                self,
                mut sequence: S,
            ) -> Result<UniqueNames, S::Error> {
                while sequence.next_element::<UniqueNames>()?.is_some() {}
                Ok(UniqueNames)
            }
            fn visit_bool<E: serde::de::Error>(self, _: bool) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
            fn visit_u64<E: serde::de::Error>(self, _: u64) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
            fn visit_i64<E: serde::de::Error>(self, _: i64) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
            fn visit_f64<E: serde::de::Error>(self, _: f64) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
            fn visit_str<E: serde::de::Error>(self, _: &str) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
            fn visit_unit<E: serde::de::Error>(self) -> Result<UniqueNames, E> {
                Ok(UniqueNames)
            }
        }
        decoder.deserialize_any(Visitor)
    }
}

impl Scenario {
    #[expect(
        clippy::too_many_lines,
        reason = "Admission preserves the complete public scenario grammar before preparing any guest"
    )]
    fn validate(&self) -> Result<(), String> {
        if let Some(workload) = &self.workload {
            workload.validate()?;
        }
        let spec = &self.spec;
        if spec.fidelity == "hard_isolation" {
            return Err("one-helper WASM clusters cannot provide hard isolation".into());
        }
        if self.schema != "gomad-wasm.scenario/v1"
            || spec.schema != "gomad.simulation-spec/v8"
            || spec.backend != "wasm"
            || spec.fidelity != "simulation_model"
            || spec.seed != 0
            || spec.faults.is_some()
            || spec.exploration.is_some()
            || spec.scenario_choice_plan.is_some()
            || spec.replay.is_some()
        {
            return Err("unsupported WASM scenario schema or control plan".into());
        }
        spec.limits.validate()?;
        if spec.nodes.is_empty()
            || spec.nodes.len() as u64 > spec.limits.nodes
            || spec.links.len() as u64 > spec.limits.directional_links
            || spec.volumes.len() as u64 > spec.limits.volumes
            || self.boots.is_empty()
            || self.boots.len() > 4096
            || self.actions.is_empty()
            || self.actions.len() as u64 > spec.limits.scenario_actions
            || self.network_faults.len() as u64 > spec.limits.fault_actions
        {
            return Err("WASM scenario capacity".into());
        }
        let mut volumes = BTreeMap::new();
        let mut capacity = 0u64;
        let mut previous = "";
        for volume in &spec.volumes {
            capacity = capacity
                .checked_add(volume.capacity_bytes)
                .ok_or("volume capacity overflow")?;
            if !valid_id(&volume.id)
                || volume.id.as_str() <= previous
                || volume.capacity_bytes == 0
                || capacity > spec.limits.volume_capacity_bytes
            {
                return Err("invalid simulation volume".into());
            }
            previous = &volume.id;
            volumes.insert(volume.id.as_str(), volume.capacity_bytes);
        }
        let mut boots = BTreeMap::new();
        for boot in &self.boots {
            if boot.id.is_empty() || boots.insert(boot.id.as_str(), boot).is_some() {
                return Err("invalid or duplicate WASM boot ID".into());
            }
        }
        let mut nodes = BTreeMap::new();
        let mut addresses = BTreeSet::new();
        let mut owners = BTreeSet::new();
        previous = "";
        for node in &spec.nodes {
            let address: IpAddr = node
                .address
                .parse()
                .map_err(|_| "invalid simulation node address")?;
            if !valid_id(&node.id)
                || node.id.as_str() <= previous
                || !valid_id(&node.boot)
                || !boots.contains_key(node.boot.as_str())
                || node.config.as_ref().is_some_and(|value| !value.is_empty())
                || address.is_unspecified()
                || address.to_string() != node.address
                || !addresses.insert(address)
            {
                return Err("invalid WASM scenario node".into());
            }
            previous = &node.id;
            let mut last_path = "";
            for mount in &node.volumes {
                if !volumes.contains_key(mount.volume.as_str())
                    || !clean_path(&mount.path)
                    || mount.path.as_str() <= last_path
                    || !owners.insert(mount.volume.as_str())
                {
                    return Err("invalid simulation volume mount".into());
                }
                last_path = &mount.path;
            }
            nodes.insert(node.id.as_str(), node);
        }
        if owners.len() != volumes.len() {
            return Err("every simulation volume must have one owner".into());
        }
        let mut previous_link = None;
        for link in &spec.links {
            let key = (&link.from, &link.to);
            if !nodes.contains_key(link.from.as_str())
                || !nodes.contains_key(link.to.as_str())
                || link.from == link.to
                || previous_link.is_some_and(|previous| previous >= key)
            {
                return Err("invalid or unsorted simulation link".into());
            }
            previous_link = Some(key);
        }
        let mut bindings = BTreeMap::new();
        for binding in &self.incarnation_boots {
            if !nodes.contains_key(binding.handle.node.as_str())
                || !boots.contains_key(binding.boot.as_str())
                || binding.handle.incarnation < 2
                || bindings
                    .insert(binding.handle.clone(), binding.boot.as_str())
                    .is_some()
            {
                return Err("invalid WASM incarnation boot".into());
            }
        }
        let mut boundaries = BTreeMap::new();
        for (action, items) in [
            (helper::Action::Crash, &self.output_crashes),
            (helper::Action::Stop, &self.output_stops),
        ] {
            for boundary in items {
                marker(&boundary.marker)?;
                if !nodes.contains_key(boundary.handle.node.as_str())
                    || boundary.handle.incarnation == 0
                    || boundaries.insert(boundary.handle.clone(), action).is_some()
                {
                    return Err("invalid WASM lifecycle boundary".into());
                }
            }
        }
        let mut current = BTreeMap::<&str, (u64, bool)>::new();
        let mut used_bindings = BTreeSet::new();
        let mut used_boundaries = BTreeSet::new();
        for step in &self.actions {
            let node = nodes
                .get(step.handle.node.as_str())
                .ok_or("invalid WASM lifecycle reference")?;
            let state = current.entry(&step.handle.node).or_default();
            match step.action {
                helper::Action::Start | helper::Action::Restart => {
                    if state.1
                        || state.0.checked_add(1) != Some(step.handle.incarnation)
                        || (step.action == helper::Action::Start) != (state.0 == 0)
                    {
                        return Err("invalid WASM start/restart order".into());
                    }
                    *state = (step.handle.incarnation, true);
                    if bindings.contains_key(&step.handle) {
                        used_bindings.insert(step.handle.clone());
                    }
                }
                helper::Action::Wait | helper::Action::Crash | helper::Action::Stop => {
                    if !state.1 || state.0 != step.handle.incarnation {
                        return Err("invalid WASM terminal order".into());
                    }
                    if step.action != helper::Action::Wait {
                        let boot = boots[bindings.get(&step.handle).copied().unwrap_or(&node.boot)];
                        let storage_crash = step.action == helper::Action::Crash
                            && boot.storage.as_ref().is_some_and(|storage| {
                                storage
                                    .faults
                                    .iter()
                                    .any(|fault| fault.action == "crash-before")
                            });
                        if boundaries.get(&step.handle) == Some(&step.action) {
                            used_boundaries.insert(step.handle.clone());
                        } else if !storage_crash {
                            return Err("undeclared WASM lifecycle boundary".into());
                        }
                    }
                    state.1 = false;
                }
            }
        }
        if used_bindings.len() != bindings.len()
            || used_boundaries.len() != boundaries.len()
            || current.len() != nodes.len()
            || current.values().any(|state| state.1)
        {
            return Err("unused WASM lifecycle binding or unterminated node".into());
        }
        for node in &spec.nodes {
            let storage = &boots[node.boot.as_str()].storage;
            let mounts = storage
                .as_ref()
                .map_or(&[][..], |storage| storage.volumes.as_slice());
            if mounts.len() != node.volumes.len()
                || node.volumes.iter().any(|mount| {
                    !mounts.iter().any(|volume| {
                        volume.id == mount.volume
                            && volume.path == mount.path
                            && volumes.get(volume.id.as_str()) == Some(&volume.capacity_bytes)
                    })
                })
            {
                return Err("WASM boot durable mounts differ from scenario".into());
            }
        }
        for binding in &self.incarnation_boots {
            let initial = &boots[nodes[binding.handle.node.as_str()].boot.as_str()].storage;
            let restart = &boots[binding.boot.as_str()].storage;
            if canonical::encode(initial)? != canonical::encode(restart)? {
                return Err("WASM restart durable model differs from initial boot".into());
            }
        }
        Ok(())
    }
}

/// # Errors
/// Refuses malformed, unknown, unbounded and inconsistent scenario declarations.
/// The scenario's evidence bound for one cluster execution.
pub(crate) fn evidence_limit(normalized: &[u8]) -> Result<u64, String> {
    Ok(parse(normalized)?.spec.limits.scenario_evidence_bytes)
}

pub(super) fn original_choice_bytes(raw: Option<&[u8]>, requested: u64) -> Result<u64, String> {
    if let Some(raw) = raw {
        let scenario = parse(raw)?;
        if matches!(
            scenario.workload,
            Some(
                crate::qualify::workloads::Workload::TemporalRecovery { .. }
                    | crate::qualify::workloads::Workload::TemporalInspection
            )
        ) {
            if requested != 0 && requested != 1 << 30 {
                return Err(
                    "registered Temporal workload requires its original 1GiB choice bound".into(),
                );
            }
            return Ok(1 << 30);
        }
        if requested == 0 {
            return Ok(8 << 20);
        }
    }
    Ok(requested.max(160))
}

/// # Errors
/// Refuses malformed, unknown, unbounded and inconsistent scenario declarations.
pub fn normalize(raw: &[u8]) -> Result<Vec<u8>, String> {
    let normalized = canonical::encode(&parse(raw)?)?;
    if normalized.len() > 256 << 10 {
        return Err("normalized WASM scenario size exceeds bounds".into());
    }
    Ok(normalized)
}

#[cfg(test)]
mod fidelity_tests {
    use super::*;

    #[test]
    fn single_linked_helper_refuses_hard_isolation() {
        let mut limits = serde_json::Map::new();
        for name in [
            "nodes",
            "directional_links",
            "volumes",
            "scenario_actions",
            "scenario_decisions",
            "fault_actions",
            "history_operations",
            "observations",
            "oracle_results",
            "scenario_evidence_bytes",
            "boot_config_bytes",
            "volume_capacity_bytes",
            "observation_bytes",
            "network_listeners",
            "network_connections",
            "network_bytes",
            "network_connection_bytes",
            "network_stall_nanos",
            "network_transitions",
            "volume_operations",
            "volume_transitions",
            "crash_states",
            "crash_depth",
            "crash_bytes",
            "crash_wall_nanos",
        ] {
            limits.insert(name.into(), json!(1));
        }
        limits.insert("scenario_actions".into(), json!(2));
        let mut scenario: Scenario = serde_json::from_value(json!({
            "schema": "gomad-wasm.scenario/v1",
            "spec": {
                "schema": "gomad.simulation-spec/v8", "backend": "wasm",
                "fidelity": "simulation_model", "limits": limits,
                "nodes": [{"id":"a", "boot":"guest", "address":"10.0.0.1"}]
            },
            "boots": [{"id":"guest"}],
            "actions": [
                {"action":"start", "handle":{"node":"a", "incarnation":1}},
                {"action":"wait", "handle":{"node":"a", "incarnation":1}}
            ]
        }))
        .expect("bounded ordinary scenario");
        scenario.validate().expect("ordinary simulation model");
        scenario.spec.fidelity = "hard_isolation".into();
        let failure = scenario
            .validate()
            .expect_err("one helper cannot promise hard isolation");
        assert!(failure.contains("hard isolation"), "{failure}");
    }
}

pub struct Bound {
    pub configuration: cluster::Configuration,
    pub request: helper::Request,
}

pub(super) fn simulation_crash_bounds(raw: &[u8], bound: &mut Bound) -> Result<(), String> {
    let scenario = parse(raw)?;
    let limits = &scenario.spec.limits;
    let work = limits
        .crash_states
        .checked_mul(limits.volume_operations)
        .ok_or("capacity: simulation crash work bound overflow")?;
    for incarnation in &mut bound.request.incarnations {
        if let Some(storage) = &mut incarnation.request.storage {
            let mut value = serde_json::to_value(&*storage).map_err(|error| error.to_string())?;
            value["descriptor"]["crash_limits"] = json!({
                "States":limits.crash_states,"Operations":limits.volume_operations,
                "Depth":limits.crash_depth,"Bytes":limits.crash_bytes,"Work":work
            });
            *storage = serde_json::from_value(value).map_err(|error| error.to_string())?;
            let retained = bound
                .configuration
                .incarnations
                .iter_mut()
                .find(|item| item.handle == incarnation.handle)
                .ok_or("simulation storage incarnation missing its configuration")?;
            retained.storage = Some(storage.clone());
        }
    }
    Ok(())
}

pub(crate) fn retain_execution_prefix(
    configuration: &mut cluster::Configuration,
    steps: &[helper::Step],
    simulation: bool,
) -> Result<(), String> {
    if simulation {
        if configuration.simulation_original.is_none() {
            configuration.simulation_original = Some(Box::new(configuration.clone()));
        }
        let mut remaining = std::collections::VecDeque::from(configuration.steps.clone());
        let mut reordered = Vec::new();
        for actual in steps {
            let position = if remaining.front() == Some(actual) {
                Some(0)
            } else if matches!(
                actual.action,
                helper::Action::Start | helper::Action::Restart
            ) {
                remaining
                    .iter()
                    .take_while(|step| {
                        matches!(step.action, helper::Action::Start | helper::Action::Restart)
                    })
                    .position(|step| step == actual)
            } else {
                None
            };
            let Some(position) = position else {
                break;
            };
            reordered.push(
                remaining
                    .remove(position)
                    .ok_or("simulation lifecycle action missing")?,
            );
        }
        reordered.extend(remaining);
        configuration.steps = reordered;
    }
    retain_prefix(configuration, steps)
}

pub(super) fn retain_prefix(
    configuration: &mut cluster::Configuration,
    steps: &[helper::Step],
) -> Result<(), String> {
    if steps.len() > 65536 {
        return Err("scenario executed lifecycle capacity".into());
    }
    let prefix = steps
        .iter()
        .zip(&configuration.steps)
        .take_while(|(actual, declared)| actual == declared)
        .count();
    let mut started = BTreeSet::new();
    let mut running = BTreeSet::new();
    for step in &steps[..prefix] {
        if matches!(step.action, helper::Action::Start | helper::Action::Restart) {
            started.insert(step.handle.clone());
            running.insert(step.handle.clone());
        } else {
            running.remove(&step.handle);
        }
    }
    let drain: Vec<_> = running
        .into_iter()
        .map(|handle| helper::Step {
            handle,
            action: helper::Action::Wait,
        })
        .collect();
    let implicit_drain =
        prefix == steps.len() && !drain.is_empty() && configuration.steps[prefix..] == drain;
    if !implicit_drain && steps[prefix..] != drain {
        return Err(
            "scenario executed lifecycle is not its declared prefix and ordered drain".into(),
        );
    }
    configuration
        .incarnations
        .retain(|item| started.contains(&item.handle));
    if configuration.incarnations.len() != started.len() {
        return Err("scenario executed lifecycle has an unknown incarnation".into());
    }
    // A failed wait joins the remaining stores without adding scenario actions.
    // Keep their exact declared ordered waits as the replay recipe.
    if implicit_drain {
        return Ok(());
    }
    configuration.boundaries.retain(|boundary| {
        steps
            .iter()
            .any(|step| step.handle == boundary.handle && step.action == boundary.action)
    });
    configuration.steps = steps.to_vec();
    Ok(())
}

#[expect(
    clippy::too_many_lines,
    reason = "Oracle authentication binds each helper outcome fact to its retained output before recomputing the canonical report"
)]
pub(super) fn authenticate_assertions(
    plan: &Value,
    opened: &crate::evidence::store::Opened,
) -> Result<(), String> {
    let bytes = opened
        .manifest()
        .record
        .streams
        .contains_key("evidence/soft-assertions.json")
        .then(|| opened.payload("evidence/soft-assertions.json", 16 << 20))
        .transpose()?;
    let actual: Option<Assertions> = bytes
        .as_deref()
        .map(|bytes| serde_json::from_slice(bytes).map_err(|error| error.to_string()))
        .transpose()?;
    if let Some(actual) = &actual
        && (canonical::encode(actual)?.as_slice() != bytes.as_deref().unwrap_or_default()
            || actual.schema != "gomad.cluster-soft-assertions/v1")
    {
        return Err("unsupported scenario soft assertion evidence".into());
    }
    let report_only = plan["soft_assertion_report_only"]
        .as_bool()
        .ok_or("scenario assertion policy missing")?;
    let raw: Vec<u8> =
        serde_json::from_value(plan["scenario"].clone()).map_err(|error| error.to_string())?;
    let scenario = parse(&raw)?;
    let summary_bytes = opened.payload("evidence/cluster_summary", 128 << 20)?;
    let summary: Value =
        serde_json::from_slice(&summary_bytes).map_err(|error| error.to_string())?;
    if canonical::encode(&summary)? != summary_bytes
        || summary["schema"] != "gomad.cluster-observation/v1"
    {
        return Err("invalid retained scenario observation".into());
    }
    let mut observed = BTreeMap::new();
    for row in summary["incarnations"]
        .as_array()
        .ok_or("missing scenario outcome facts")?
    {
        let handle: helper::Handle =
            serde_json::from_value(row["handle"].clone()).map_err(|error| error.to_string())?;
        if observed.insert(handle, row).is_some() {
            return Err("duplicate scenario outcome facts".into());
        }
    }
    let handles: BTreeSet<_> = scenario
        .actions
        .iter()
        .filter(|step| matches!(step.action, helper::Action::Start | helper::Action::Restart))
        .map(|step| step.handle.clone())
        .collect();
    if observed.keys().any(|handle| !handles.contains(handle)) {
        return Err("unknown scenario outcome facts".into());
    }
    let mut reports = BTreeMap::new();
    for handle in handles {
        let row = observed.get(&handle);
        if row.is_none() && !report_only {
            continue;
        }
        let mut output = Vec::new();
        for name in ["stdout", "stderr"] {
            let file = format!(
                "evidence/cluster/{}/{}/{name}",
                handle.node, handle.incarnation
            );
            let bytes = if let Some(row) = row.filter(|row| !row["streams"][name].is_null()) {
                let bytes = opened.payload(&file, opened.manifest().record.limits.output_bytes)?;
                if row["streams"][name]["bytes"].as_u64() != Some(bytes.len() as u64)
                    || row["streams"][name]["sha256"].as_str()
                        != Some(canonical::digest(&bytes).as_str())
                {
                    return Err("scenario output facts mismatch".into());
                }
                bytes
            } else {
                Vec::new()
            };
            output.push(bytes);
        }
        let complete = if let Some(row) = row {
            output_complete(
                row["termination"]
                    .as_str()
                    .ok_or("scenario terminal fact missing")?,
                row["reason"]
                    .as_str()
                    .ok_or("scenario reason fact missing")?,
                row.get("local_model").filter(|value| !value.is_null()),
                row["offered_output_bytes"]
                    .as_u64()
                    .ok_or("scenario output byte fact missing")?,
                output.iter().map(|bytes| bytes.len() as u64).sum(),
            )
        } else {
            false
        };
        if let Some(report) =
            super::soft_assertions::observe(&output[0], &output[1], complete, report_only)
        {
            reports.insert(handle, report);
        }
    }
    let expected = Assertions {
        schema: "gomad.cluster-soft-assertions/v1".into(),
        incarnations: reports
            .into_iter()
            .map(|(handle, report)| IncarnationAssertions { handle, report })
            .collect(),
    };
    let expected = (!expected.incarnations.is_empty()).then_some(expected);
    if expected != actual
        || expected.as_ref().is_some_and(|expected| {
            expected.incarnations.len() as u64 > scenario.spec.limits.oracle_results
        })
        || bytes
            .as_ref()
            .is_some_and(|bytes| bytes.len() as u64 > scenario.spec.limits.scenario_evidence_bytes)
    {
        return Err("scenario soft assertion evidence mismatch".into());
    }
    let outcome = &opened.manifest().record.outcome;
    if outcome.termination == "exit"
        && outcome.exit_code == Some(0)
        && expected.as_ref().is_some_and(Assertions::failed) != (outcome.class == "target_failure")
    {
        return Err("scenario soft assertion classification mismatch".into());
    }
    Ok(())
}

#[derive(Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct IncarnationAssertions {
    handle: helper::Handle,
    report: super::soft_assertions::Report,
}

#[derive(Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Assertions {
    schema: String,
    incarnations: Vec<IncarnationAssertions>,
}

impl Assertions {
    fn failed(&self) -> bool {
        self.incarnations.iter().any(|item| {
            item.report.inconclusive || !item.report.report_only && !item.report.messages.is_empty()
        })
    }
}

pub(crate) fn complete_output(execution: &linked::Execution) -> bool {
    let outcome = &execution.outcome;
    let captured_bytes = ["stdout", "stderr"]
        .into_iter()
        .map(|name| {
            execution
                .evidence
                .get(name)
                .map_or(0, |bytes| bytes.len() as u64)
        })
        .sum::<u64>();
    output_complete(
        outcome.termination,
        &outcome.message,
        outcome.local_model.as_deref(),
        outcome.output_bytes,
        captured_bytes,
    )
}

fn output_complete(
    termination: &str,
    message: &str,
    model: Option<&Value>,
    offered: u64,
    captured: u64,
) -> bool {
    let capacity_interruption = termination == "capacity"
        && model.is_some_and(|model| model["configured"] == true && model["finished"] == false)
        && message
            .strip_prefix("WASI capacity: ")
            .is_some_and(|message| {
                matches!(
                    message.split(' ').next().unwrap_or(""),
                    "event-ids"
                        | "events"
                        | "payload-bytes"
                        | "queued-events"
                        | "request-ids"
                        | "requests"
                        | "transition-bytes"
                        | "transition-sequences"
                        | "transitions"
                )
            });
    (termination != "capacity" || capacity_interruption)
        && !matches!(termination, "cancelled" | "watchdog_timeout")
        && offered == captured
}

fn assertions(
    execution: &helper::Execution,
    configuration: &cluster::Configuration,
    report_only: bool,
    maximum_results: u64,
    maximum_bytes: u64,
) -> Result<Option<Assertions>, String> {
    let observed: BTreeMap<_, _> = execution
        .incarnations
        .iter()
        .map(|item| (&item.handle, &item.execution))
        .collect();
    let mut reports = BTreeMap::new();
    for incarnation in &configuration.incarnations {
        let observed = observed.get(&incarnation.handle);
        if observed.is_none() && !report_only {
            continue;
        }
        let stdout = observed
            .and_then(|execution| execution.evidence.get("stdout"))
            .map_or(&[][..], Vec::as_slice);
        let stderr = observed
            .and_then(|execution| execution.evidence.get("stderr"))
            .map_or(&[][..], Vec::as_slice);
        if let Some(report) = super::soft_assertions::observe(
            stdout,
            stderr,
            observed.is_some_and(|execution| complete_output(execution)),
            report_only,
        ) {
            reports.insert(incarnation.handle.clone(), report);
        }
    }
    if reports.is_empty() {
        return Ok(None);
    }
    if reports.len() as u64 > maximum_results {
        return Err("capacity: scenario soft assertion results".into());
    }
    let reports = Assertions {
        schema: "gomad.cluster-soft-assertions/v1".into(),
        incarnations: reports
            .into_iter()
            .map(|(handle, report)| IncarnationAssertions { handle, report })
            .collect(),
    };
    if canonical::encode(&reports)?.len() as u64 > maximum_bytes {
        return Err("capacity: scenario soft assertion evidence bytes".into());
    }
    Ok(Some(reports))
}

fn output_capacity_failure(
    execution: &helper::Execution,
    configuration: &cluster::Configuration,
    report_only: bool,
    maximum_results: u64,
    maximum_bytes: u64,
) -> Result<Option<super::execution::Failure>, String> {
    if execution.outcome.termination != "capacity"
        || !matches!(
            execution.outcome.message.as_str(),
            "output capacity" | "WASI capacity: offered-output-bytes"
        )
    {
        return Ok(None);
    }
    let Some(selected) = execution
        .incarnations
        .iter()
        .filter(|item| {
            item.execution.outcome.termination == "capacity"
                && matches!(
                    item.execution.outcome.message.as_str(),
                    "output capacity" | "WASI capacity: offered-output-bytes"
                )
        })
        .min_by(|left, right| left.handle.cmp(&right.handle))
    else {
        return Ok(None);
    };
    let report = super::soft_assertions::observe(
        selected
            .execution
            .evidence
            .get("stdout")
            .map_or(&[][..], Vec::as_slice),
        selected
            .execution
            .evidence
            .get("stderr")
            .map_or(&[][..], Vec::as_slice),
        false,
        report_only,
    );
    let finalization = assertions(
        execution,
        configuration,
        report_only,
        maximum_results,
        maximum_bytes,
    )
    .map(|_| ());
    let diagnostic = if let Err(error) = finalization {
        Some(error)
    } else {
        let mut owners = execution.incarnations.iter().collect::<Vec<_>>();
        owners.sort_by(|left, right| left.handle.cmp(&right.handle));
        owners.into_iter().find_map(|owner| {
            let declared = configuration
                .incarnations
                .iter()
                .find(|declared| declared.handle == owner.handle)?;
            let trace = owner.execution.evidence.get("choice");
            let result = if trace.is_none()
                && !owner
                    .execution
                    .outcome
                    .local_model
                    .as_deref()
                    .is_some_and(|model| {
                        model["configured"] == true
                            && model["finished"] == true
                            && model["records"] == 0
                    }) {
                Err("choice trace is not complete".into())
            } else {
                incarnation_choice_features(
                    trace.map_or(&[][..], Vec::as_slice),
                    &declared.choice_identity,
                    owner.execution.outcome.termination,
                )
                .map(|_| ())
            };
            result.err().map(|error| {
                format!(
                    "retain WASM {}/{} choice control: {error}",
                    owner.handle.node, owner.handle.incarnation
                )
            })
        })
    };
    Ok(Some(super::execution::Failure {
        termination: "capacity",
        reason: selected.execution.outcome.message.clone(),
        soft_assertions: report
            .as_ref()
            .map(serde_json::to_value)
            .transpose()
            .map_err(|error| error.to_string())?,
        diagnostics: diagnostic.into_iter().collect(),
    }))
}

fn storage_input(storage: &Storage) -> Result<linked::StorageInput, String> {
    let encoded = serde_json::to_string(storage)
        .map_err(|error| error.to_string())?
        .replace('&', "\\u0026")
        .replace('<', "\\u003c")
        .replace('>', "\\u003e")
        .replace('\u{2028}', "\\u2028")
        .replace('\u{2029}', "\\u2029");
    serde_json::from_value(json!({"descriptor":{"model":storage,
        "model_identity":format!("sha256:{}", canonical::digest(encoded.as_bytes())),
        "return_image":true},"image":null,"context":null}))
    .map_err(|error| error.to_string())
}

fn effective_environment(base: &preflight::Configuration, seed: u64) -> BTreeMap<String, String> {
    let mut environment = base.environment.clone();
    if base.derive_seed_environment {
        environment.insert("GOMADSEED".into(), seed.to_string());
    }
    environment
}

fn boot_request(
    boot: &Boot,
    base: &preflight::Configuration,
    module: &[u8],
    seed: u64,
) -> Result<linked::Request, String> {
    let program = base
        .arguments
        .first()
        .ok_or("scenario target argument vector is empty")?;
    let arguments = boot.args.as_ref().map_or_else(
        || base.arguments.clone(),
        |args| {
            std::iter::once(program.clone())
                .chain(args.iter().cloned())
                .collect()
        },
    );
    let mut model = base.model.clone().ok_or("missing scenario model")?;
    model
        .as_object_mut()
        .ok_or("invalid scenario model")?
        .insert("seed".into(), seed.into());
    Ok(linked::Request {
        module: module.to_vec(),
        module_sha256: canonical::digest(module),
        limits: linked::Limits {
            memory_bytes: base.limits.memory_bytes,
            fuel: base.limits.fuel,
            output_bytes: base.limits.output_bytes,
            evidence_bytes: base.limits.evidence_bytes,
            wall_timeout: preflight::wall_timeout(&base.limits, base.wall_nanos)?,
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(model).map_err(|error| error.to_string())?,
            arguments,
            environment: effective_environment(base, seed)
                .iter()
                .map(|(key, value)| format!("{key}={value}"))
                .collect(),
            replay_trace: None,
        }),
        namespace: base.namespace.clone(),
        storage: boot.storage.as_ref().map(storage_input).transpose()?,
    })
}

/// # Errors
/// Binds admitted scenario declarations to fresh per-incarnation helper inputs and retained replay inputs.
#[expect(
    clippy::too_many_lines,
    reason = "The adapter binds one prepared target to the complete declared lifecycle"
)]
pub fn bind(
    raw: &[u8],
    base: &preflight::Configuration,
    module: &[u8],
    identity: &linked::ChoiceIdentity,
    network: Option<&helper::NetworkModel>,
    seed: u64,
) -> Result<Bound, String> {
    let scenario = parse(raw)?;
    let model = base
        .model
        .as_ref()
        .ok_or("scenario requires cooperative model")?;
    if base.profile != "gomad.wasi-cooperative/v1"
        || base.arguments.is_empty()
        || model.get("read_step_nanos") != Some(&json!(0))
        || model.get("mode") != Some(&json!(1))
        || model.get("seed").is_some()
        || model.get("replay").is_some_and(|value| !value.is_null())
    {
        return Err("WASM scenario requires fresh cooperative frozen-clock inputs".into());
    }
    let limits = &scenario.spec.limits;
    let used_boots: BTreeSet<_> = scenario
        .spec
        .nodes
        .iter()
        .map(|node| node.boot.as_str())
        .chain(
            scenario
                .incarnation_boots
                .iter()
                .map(|binding| binding.boot.as_str()),
        )
        .collect();
    for boot in &scenario.boots {
        if !used_boots.contains(boot.id.as_str()) {
            linked::prepare(boot_request(boot, base, module, seed)?)
                .map_err(|error| error.to_string())?;
        }
    }
    if network.is_some_and(|network| {
        network.limits.connection_bytes != limits.network_connection_bytes
            || network.limits.stall_nanos != limits.network_stall_nanos
            || !network.faults.is_empty()
    }) {
        return Err("WASM boot network limits differ from simulation specification".into());
    }
    if network.is_none() && !scenario.network_faults.is_empty() {
        return Err("WASM scenario network declarations require network model".into());
    }
    let broker = helper::BrokerConfig {
        seed,
        pause_on_settle: false,
        transition_bytes: base.world_transition_bytes,
        limits: helper::WorldLimits {
            max_requests: limits.network_transitions * 2,
            max_events: limits.network_transitions * 2,
            max_queued_events: 65536.min(limits.network_transitions * 2),
            max_transitions: limits.network_transitions * 8,
            max_payload_bytes: limits.observation_bytes,
            max_string_bytes: 1024,
        },
    };
    let network_config = network.map(|network| serde_json::from_value::<helper::NetworkConfig>(json!({
        "model":network,"addresses":scenario.spec.nodes.iter().map(|node| (node.id.clone(),node.address.clone())).collect::<BTreeMap<_,_>>(),
        "links":scenario.spec.links,"strict_topology":true,
        "faults":scenario.network_faults.iter().map(|fault| json!({"name":fault.name,"action":fault.action,
            "from":fault.from,"to":fault.to,"at_nanos":fault.at_nanos,"delay_nanos":fault.delay_nanos,"symmetric":fault.symmetric})).collect::<Vec<_>>()
    })).map_err(|error| error.to_string())).transpose()?;
    let mut retained = Vec::new();
    let mut incarnations = Vec::new();
    for step in &scenario.actions {
        if !matches!(step.action, helper::Action::Start | helper::Action::Restart) {
            continue;
        }
        let node = scenario
            .spec
            .nodes
            .iter()
            .find(|node| node.id == step.handle.node)
            .ok_or("unknown scenario node")?;
        let boot_id = scenario
            .incarnation_boots
            .iter()
            .find(|binding| binding.handle == step.handle)
            .map_or(node.boot.as_str(), |binding| binding.boot.as_str());
        let boot = scenario
            .boots
            .iter()
            .find(|boot| boot.id == boot_id)
            .ok_or("unknown scenario boot")?;
        let request = boot_request(boot, base, module, seed)?;
        let arguments = request
            .model
            .as_ref()
            .ok_or("missing scenario model")?
            .arguments
            .clone();
        let storage = request.storage.clone();
        let namespace = request
            .namespace
            .clone()
            .ok_or("scenario requires sealed namespace")?;
        incarnations.push(helper::Incarnation {
            handle: step.handle.clone(),
            network: network.cloned(),
            request,
        });
        retained.push(cluster::Incarnation {
            workload_kind: String::new(),
            handle: step.handle.clone(),
            imports: Some(cluster::module_imports(module)?),
            storage_crash_selection: None,
            module: cluster::Payload {
                file: "target/module.wasm".into(),
                bytes: module.len() as u64,
                sha256: canonical::digest(module),
                limit: 512 << 20,
            },
            limits: base.limits.clone(),
            wall_nanos: base.wall_nanos,
            arguments,
            environment: effective_environment(base, seed),
            runtime_sha256: base
                .runtime_sha256
                .clone()
                .ok_or("missing scenario runtime identity")?,
            model: model.clone(),
            namespace,
            storage,
            network: network.cloned(),
            choice_identity: identity.clone(),
        });
    }
    let boundaries = [
        (helper::Action::Crash, &scenario.output_crashes),
        (helper::Action::Stop, &scenario.output_stops),
    ]
    .into_iter()
    .flat_map(|(action, items)| {
        items.iter().map(move |item| {
            Ok(helper::Boundary {
                handle: item.handle.clone(),
                action,
                marker: marker(&item.marker)?,
            })
        })
    })
    .collect::<Result<Vec<_>, String>>()?;
    let nodes: Vec<_> = scenario
        .spec
        .nodes
        .iter()
        .map(|node| node.id.clone())
        .collect();
    let request = helper::Request {
        nodes: nodes.clone(),
        broker: broker.clone(),
        network: network_config.clone(),
        incarnations,
        steps: scenario.actions.clone(),
        boundaries: boundaries.clone(),
        crash_groups: Vec::new(),
        wall_timeout: preflight::wall_timeout(&base.limits, base.wall_nanos)?,
        evidence_bytes: base
            .limits
            .evidence_bytes
            .min(limits.scenario_evidence_bytes),
    };
    let mut execution_limits = base.limits.clone();
    execution_limits.evidence_bytes = request.evidence_bytes;
    let mut bound = Bound {
        configuration: cluster::Configuration {
            original_limits: None,
            oracle_plan: None,
            world_encoding: String::new(),
            world_profile: None,
            simulation_original: None,
            workload: scenario.workload.clone(),
            nodes,
            engine: base.engine.clone(),
            profile: base.profile.clone(),
            limits: execution_limits,
            wall_nanos: base.wall_nanos,
            broker,
            network: network_config,
            incarnations: retained,
            steps: scenario.actions,
            boundaries,
            crash_groups: Vec::new(),
        },
        request,
    };
    if let Some(workload) = &scenario.workload {
        workload.bind(&mut bound)?;
    }
    Ok(bound)
}

/// # Errors
/// Runs a fresh typed cluster and retains the existing cluster replay declaration and streams.
pub(super) fn execute(
    prepared: &super::prepare::Prepared,
    job: &super::ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
) -> Result<super::execution::Completion, String> {
    execute_controlled(
        prepared,
        job,
        timeout,
        cancellation,
        #[cfg(all(test, feature = "helper-determinism-fixture"))]
        None,
    )
}

#[cfg(all(test, feature = "helper-determinism-fixture"))]
pub(super) fn execute_counterfactual(
    prepared: &super::prepare::Prepared,
    job: &super::ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    fault: super::counterfactual::Fault,
) -> Result<super::execution::Completion, String> {
    execute_controlled(prepared, job, timeout, cancellation, Some(fault))
}

fn execute_controlled(
    prepared: &super::prepare::Prepared,
    job: &super::ledger::Job,
    timeout: Duration,
    cancellation: &linked::Cancellation,
    #[cfg(all(test, feature = "helper-determinism-fixture"))] fault: Option<
        super::counterfactual::Fault,
    >,
) -> Result<super::execution::Completion, String> {
    let mut bound = bind(
        prepared
            .scenario
            .as_deref()
            .ok_or("missing campaign scenario")?,
        &prepared.configuration,
        &prepared.module,
        prepared
            .choice_identity
            .as_ref()
            .ok_or("missing scenario choice identity")?,
        prepared.network.as_ref(),
        job.seed,
    )?;
    crate::qualify::workloads::recovery::bind(prepared, &mut bound)?;
    #[cfg(all(test, feature = "helper-determinism-fixture"))]
    if fault.is_some() {
        super::counterfactual::validate_bound(&bound)?;
    }
    let scenario = parse(
        prepared
            .scenario
            .as_deref()
            .ok_or("missing campaign scenario")?,
    )?;
    if prepared.soft_assertion_report_only
        && bound.configuration.incarnations.len() as u64 > scenario.spec.limits.oracle_results
    {
        return Err("capacity: scenario soft assertion policy results".into());
    }
    bound.request.wall_timeout = bound.request.wall_timeout.min(timeout);
    let limits = store_limits(&bound.request.incarnations);
    let mut execution =
        helper::prepare(bound.request).map_err(|error| format!("runner_failure: {error}"))?;
    match &bound.configuration.workload {
        Some(crate::qualify::workloads::Workload::SQLite { .. }) => {
            execution.sqlite_crash_reference()
        }
        Some(crate::qualify::workloads::Workload::TemporalRecovery { .. }) => {
            execution.temporal_crash_reference()
        }
        _ => Ok(()),
    }
    .map_err(|error| error.to_string())?;
    if matches!(
        bound.configuration.workload,
        Some(crate::qualify::workloads::Workload::TemporalRecovery { forcing: None })
    ) {
        execution
            .committed_temporal_reopens()
            .map_err(|error| error.to_string())?;
    }
    execution
        .compile_cache(prepared.compile_cache_root.as_deref())
        .map_err(|error| format!("runner_failure: admit compile cache: {error}"))?;
    execution.conditions(linked::Conditions::compiled());
    #[cfg(all(test, feature = "helper-determinism-fixture"))]
    if let Some(fault) = fault {
        execution
            .fixture_fault(fault.kind, fault.owner)
            .map_err(|error| error.to_string())?;
    }
    execution.cancellation(cancellation.clone());
    #[cfg(feature = "determinism-fixture")]
    crate::fixture::execution_started().map_err(|error| error.message)?;
    crate::qualify::usage::store_begin();
    let started = std::time::Instant::now();
    let execution = crate::backstop::within(execution.wall_timeout(), || {
        helper::execute_prepared(execution)
    });
    let wall = started.elapsed();
    let store_peak = crate::qualify::usage::store_end();
    let execution = execution.map_err(|error| format!("runner_failure: {error}"))?;
    let mut metrics = store_metrics(&execution, &limits, wall)?;
    metrics.store_peak_rss_bytes = store_peak;
    let mut result = publication(prepared, job, bound.configuration, execution, false)?;
    if let super::execution::Completion::Finalized(result) = &mut result {
        result.metrics = metrics;
    }
    Ok(result)
}

/// Each incarnation's own fuel and linear-memory bound, keyed by handle.
pub(crate) fn store_limits(incarnations: &[helper::Incarnation]) -> BTreeMap<String, (u64, u64)> {
    incarnations
        .iter()
        .map(|item| {
            (
                format!("{}/{}", item.handle.node, item.handle.incarnation),
                (item.request.limits.fuel, item.request.limits.memory_bytes),
            )
        })
        .collect()
}

/// Per-store observations of one shared cluster execution; the wall is shared.
pub(crate) fn store_metrics(
    execution: &helper::Execution,
    limits: &BTreeMap<String, (u64, u64)>,
    wall: Duration,
) -> Result<super::execution::Metrics, String> {
    let mut stores: Vec<_> = execution
        .incarnations
        .iter()
        .map(|item| {
            let name = format!("{}/{}", item.handle.node, item.handle.incarnation);
            let (fuel, memory) = limits.get(&name).copied().unwrap_or((0, 0));
            super::execution::StoreMetrics::observed(Some(name), &item.execution, fuel, memory)
        })
        .collect();
    // The cluster network trace is one shared stream; it is counted once, on
    // the first store, so the per-execution sum matches the owner's total.
    if let Some(first) = stores.first_mut() {
        first.trace_bytes = first
            .trace_bytes
            .saturating_add(execution.network.len() as u64);
    }
    Ok(super::execution::Metrics {
        wall_nanos: super::execution::nanos(wall),
        store_peak_rss_bytes: None,
        cache_hit: execution
            .outcome
            .metrics
            .as_ref()
            .map(|metrics| metrics.cache_hit),
        raw_backend: execution
            .outcome
            .metrics
            .as_deref()
            .map(canonical::encode)
            .transpose()?,
        stores,
    })
}

#[expect(
    clippy::too_many_lines,
    reason = "Publication binds every incarnation stream and shared observation before returning a campaign receipt"
)]
pub(crate) fn publication(
    prepared: &super::prepare::Prepared,
    job: &super::ledger::Job,
    mut configuration: cluster::Configuration,
    execution: helper::Execution,
    simulation: bool,
) -> Result<super::execution::Completion, String> {
    use crate::evidence::{
        identity::Identity,
        record::{Outcome, Record, Stream},
        store::Publication,
    };
    let scenario = parse(
        prepared
            .scenario
            .as_deref()
            .ok_or("missing campaign scenario")?,
    )?;
    if let Some(failure) = output_capacity_failure(
        &execution,
        &configuration,
        prepared.soft_assertion_report_only,
        scenario.spec.limits.oracle_results,
        scenario.spec.limits.scenario_evidence_bytes,
    )? {
        return Ok(super::execution::Completion::Failed(failure));
    }
    let reports = assertions(
        &execution,
        &configuration,
        prepared.soft_assertion_report_only,
        scenario.spec.limits.oracle_results,
        scenario.spec.limits.scenario_evidence_bytes,
    )?;
    let executed_steps = execution.executed_steps.clone();
    retain_execution_prefix(&mut configuration, &executed_steps, simulation)?;
    let observed_handles: BTreeSet<_> = execution
        .incarnations
        .iter()
        .map(|item| &item.handle)
        .collect();
    let configured_handles: BTreeSet<_> = configuration
        .incarnations
        .iter()
        .map(|item| &item.handle)
        .collect();
    if observed_handles != configured_handles {
        return Err(
            "runner_failure: scenario started handles differ from the executed lifecycle".into(),
        );
    }
    let mut retained = Vec::new();
    let mut features = Vec::new();
    for item in &execution.incarnations {
        let configuration = configuration
            .incarnations
            .iter()
            .find(|value| value.handle == item.handle)
            .ok_or("runner_failure: unknown observed scenario incarnation")?;
        let mut evidence = BTreeMap::new();
        for (name, bytes) in &item.execution.evidence {
            evidence.insert(
                name.clone(),
                cluster::Payload {
                    file: format!(
                        "evidence/cluster/{}/{}/{name}",
                        item.handle.node, item.handle.incarnation
                    ),
                    bytes: bytes.len() as u64,
                    sha256: canonical::digest(bytes),
                    limit: configuration.limits.evidence_bytes,
                },
            );
        }
        let trace = item
            .execution
            .evidence
            .get("choice")
            .map_or(&[][..], Vec::as_slice);
        if !matches!(
            item.execution.outcome.termination,
            "cancelled" | "watchdog_timeout"
        ) {
            for feature in incarnation_choice_features(
                trace,
                &configuration.choice_identity,
                item.execution.outcome.termination,
            )? {
                features.push(format!(
                    "{}/{}/{feature}",
                    item.handle.node, item.handle.incarnation
                ));
            }
        }
        evidence
            .entry("choice".into())
            .or_insert_with(|| cluster::Payload {
                file: format!(
                    "evidence/cluster/{}/{}/choice",
                    item.handle.node, item.handle.incarnation
                ),
                bytes: 0,
                sha256: canonical::digest(&[]),
                limit: configuration.limits.evidence_bytes,
            });
        retained.push(cluster::Retained {
            handle: item.handle.clone(),
            interrupted: matches!(
                item.execution.outcome.termination,
                "crash" | "stop" | "storage_crash"
            ),
            evidence,
        });
    }
    features.sort();
    features.dedup();
    let mut observed = cluster::project_workload(
        execution,
        configuration.workload.as_ref(),
        "",
        &configuration.broker,
    )?;
    if let Some(raw) = observed.event_order.take() {
        observed.evidence.insert("event-order".into(), raw);
    }
    for item in &retained {
        observed
            .evidence
            .entry(format!(
                "cluster/{}/{}/choice",
                item.handle.node, item.handle.incarnation
            ))
            .or_default();
    }
    let mut streams = BTreeMap::new();
    let mut payloads = BTreeMap::from([("target/module.wasm".into(), prepared.module.clone())]);
    for (name, bytes) in observed.evidence {
        let file = format!("evidence/{name}");
        streams.insert(
            file.clone(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 0,
            },
        );
        payloads.insert(file, bytes);
    }
    for (file, bytes) in crate::qualify::workloads::recovery::payloads(prepared)? {
        streams.insert(
            file.clone(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        );
        payloads.insert(file, bytes);
    }
    if let Some(reports) = &reports {
        let bytes = canonical::encode(reports)?;
        streams.insert(
            "evidence/soft-assertions.json".into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: reports.incarnations.len() as u64,
            },
        );
        payloads.insert("evidence/soft-assertions.json".into(), bytes);
    }
    let inputs = cluster::Inputs {
        schema: "gomad.cluster-replay-inputs/v1".into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into(),
        toolchain_sha256: prepared.toolchain.clone(),
        configuration,
        retained,
    };
    let identity = Identity {
        target: canonical::digest(&prepared.module),
        toolchain: prepared.toolchain.clone(),
        platform: preflight::platform(),
        implementation: preflight::implementation()?,
        configuration: canonical::digest(&canonical::encode(&inputs.configuration)?),
    };
    let raw = canonical::encode(&inputs)?;
    streams.insert(
        cluster::FILE.into(),
        Stream {
            bytes: raw.len() as u64,
            sha256: canonical::digest(&raw),
            records: 1,
        },
    );
    payloads.insert(cluster::FILE.into(), raw);
    let class = match observed.outcome.termination {
        "exit"
            if reports.as_ref().is_some_and(Assertions::failed)
                || observed.outcome.message == "mandatory workload oracle failed" =>
        {
            "target_failure"
        }
        "exit" if observed.outcome.exit_code == Some(0) => "success",
        "exit" | "trap" => "target_failure",
        "watchdog_timeout" => "watchdog",
        _ => "runner_failure",
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        seed: job.seed,
        profile: inputs.configuration.profile,
        limits: inputs.configuration.limits,
        arguments: Vec::new(),
        environment: BTreeMap::new(),
        virtual_time_nanos: observed
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| model["now"].as_u64())
            .unwrap_or(0),
        outcome: Outcome {
            class: class.into(),
            termination: observed.outcome.termination.into(),
            exit_code: observed.outcome.exit_code,
            reason: if observed.outcome.termination == "cancelled" {
                "runner_cancelled".into()
            } else {
                observed.outcome.message
            },
        },
        streams,
    };
    payloads.insert("record.json".into(), record.bytes()?);
    Ok(super::execution::Completion::Finalized(
        super::execution::Result {
            execution: super::ledger::Execution {
                ordinal: job.ordinal,
                seed: job.seed,
                record: record.clone(),
                artifact: None,
                stored_bytes: 0,
                semantic_probes: Vec::new(),
                choice_features: features,
                divergence: None,
                scenario_steps: Some(executed_steps),
                soft_assertions: None,
            },
            publication: Publication { record, payloads },
            metrics: super::execution::Metrics::default(),
        },
    ))
}

fn incarnation_choice_features(
    trace: &[u8],
    identity: &linked::ChoiceIdentity,
    termination: &str,
) -> Result<Vec<String>, String> {
    let mut features =
        linked::choice_features(trace, identity, "complete").map_err(str::to_owned)?;
    if matches!(termination, "crash" | "stop" | "storage_crash") {
        features.retain(|feature| feature != "terminal/complete");
    }
    Ok(features)
}

#[cfg(test)]
mod interrupted_coverage_tests {
    #[test]
    fn interrupted_incarnations_retain_records_without_completed_terminal() {
        let identity = gomad_wasmhost::linked::ChoiceIdentity {
            target_sha256: [1; 32],
            toolchain_build_key: "02".repeat(32),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [3; 32],
        };
        let mut trace = [0; 96];
        trace[8] = 1;
        trace[9] = 5;
        trace[12..16].copy_from_slice(&1u32.to_be_bytes());
        trace[32..64].fill(1);
        trace[64..96].fill(2);
        let complete = super::incarnation_choice_features(&trace, &identity, "exit").unwrap();
        assert!(
            complete
                .iter()
                .any(|feature| feature == "terminal/complete")
        );
        let records: Vec<_> = complete
            .into_iter()
            .filter(|feature| feature != "terminal/complete")
            .collect();
        assert!(
            records
                .iter()
                .any(|feature| feature.starts_with("record_kind/"))
        );
        for termination in ["crash", "stop", "storage_crash"] {
            assert_eq!(
                super::incarnation_choice_features(&trace, &identity, termination).unwrap(),
                records,
                "{termination} must preserve record features without claiming completion"
            );
        }
    }
}

#[cfg(test)]
#[path = "scenario_preservation_tests.rs"]
mod preservation_tests;
