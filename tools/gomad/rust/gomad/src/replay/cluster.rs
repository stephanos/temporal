use crate::evidence::{canonical, record::Limits, store::Opened};
use gomad_wasmhost::{linked, linked_cluster as helper, linked_inspection as inspection};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::Digest as _;
use std::collections::{BTreeMap, BTreeSet};

pub const FILE: &str = "evidence/cluster-replay.json";

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Payload {
    pub file: String,
    pub bytes: u64,
    pub sha256: String,
    pub limit: u64,
}

impl Payload {
    /// # Errors
    /// Binds the descriptor to the immutable manifest before reading its opened handle.
    pub fn read(&self, opened: &Opened, maximum: u64) -> Result<Vec<u8>, String> {
        if self.limit > maximum || self.bytes > self.limit || !canonical::valid_digest(&self.sha256)
        {
            return Err("cluster payload exceeds caller bound or has invalid identity".into());
        }
        let entry = opened
            .manifest()
            .files
            .get(&self.file)
            .ok_or("missing retained cluster payload")?;
        if entry.bytes != self.bytes || entry.sha256 != self.sha256 {
            return Err("retained cluster payload identity mismatch".into());
        }
        opened.payload(&self.file, self.limit)
    }
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Incarnation {
    pub handle: helper::Handle,
    pub module: Payload,
    pub limits: Limits,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub wall_nanos: Option<u64>,
    pub arguments: Vec<String>,
    pub environment: BTreeMap<String, String>,
    pub runtime_sha256: String,
    pub model: Value,
    pub namespace: linked::NamespaceImage,
    pub storage: Option<linked::StorageInput>,
    pub network: Option<helper::NetworkModel>,
    pub choice_identity: linked::ChoiceIdentity,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Configuration {
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub nodes: Vec<String>,
    pub engine: String,
    pub profile: String,
    pub limits: Limits,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub wall_nanos: Option<u64>,
    pub broker: helper::BrokerConfig,
    pub network: Option<helper::NetworkConfig>,
    pub incarnations: Vec<Incarnation>,
    pub steps: Vec<helper::Step>,
    pub boundaries: Vec<helper::Boundary>,
    pub crash_groups: Vec<helper::CrashGroup>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Retained {
    pub handle: helper::Handle,
    pub interrupted: bool,
    pub evidence: BTreeMap<String, Payload>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Inputs {
    pub schema: String,
    pub helper_sha256: String,
    pub host_sha256: String,
    pub runtime_implementation_sha256: String,
    pub toolchain_sha256: String,
    pub configuration: Configuration,
    pub retained: Vec<Retained>,
}

pub struct Prepared {
    pub opened: Opened,
    pub execution: helper::Prepared,
    pub exact: bool,
    pub available: bool,
}

/// # Errors
/// Decodes bounded canonical cluster inputs from the pinned root.
pub fn inputs(opened: &Opened) -> Result<Inputs, String> {
    if !opened.manifest().record.streams.contains_key(FILE) {
        return Err("retained cluster inputs are not record bound".into());
    }
    let raw = opened.payload(FILE, 16 << 20)?;
    let value: Inputs = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    if canonical::encode(&value)? != raw || value.schema != "gomad.cluster-replay-inputs/v1" {
        return Err("unsupported or noncanonical retained cluster replay inputs".into());
    }
    Ok(value)
}

fn bind(opened: &Opened, inputs: &Inputs) -> Result<(), String> {
    let record = &opened.manifest().record;
    if inputs.helper_sha256 != linked::IMPLEMENTATION_SHA256
        || inputs.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || inputs.runtime_implementation_sha256 != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
        || inputs.toolchain_sha256 != record.identity.toolchain
        || record.identity.implementation != super::preflight::implementation()?
        || record.identity.platform != super::preflight::platform()
        || inputs.configuration.engine != linked::CONFIGURATION
        || inputs.configuration.profile != record.profile
        || record.profile != "gomad.wasi-cooperative/v1"
        || inputs.configuration.limits != record.limits
        || inputs.configuration.broker.seed != record.seed
        || canonical::digest(&canonical::encode(&inputs.configuration)?)
            != record.identity.configuration
    {
        return Err("WASM retained cluster implementation/configuration identity mismatch".into());
    }
    if inputs.configuration.incarnations.is_empty()
        || inputs.configuration.incarnations.len() > 65536
        || inputs.retained.len() != inputs.configuration.incarnations.len()
    {
        return Err("retained cluster incarnation count mismatch".into());
    }
    let primary = &inputs.configuration.incarnations[0].module;
    if primary.file != "target/module.wasm" || primary.sha256 != record.identity.target {
        return Err("retained cluster primary module identity mismatch".into());
    }
    let mut handles = BTreeSet::new();
    for incarnation in &inputs.configuration.incarnations {
        if !handles.insert(incarnation.handle.clone()) {
            return Err("duplicate retained cluster incarnation".into());
        }
    }
    let retained: BTreeSet<_> = inputs
        .retained
        .iter()
        .map(|item| item.handle.clone())
        .collect();
    if handles != retained || retained.len() != inputs.retained.len() {
        return Err("retained cluster handle mismatch".into());
    }
    Ok(())
}

fn incarnation(
    opened: &Opened,
    inputs: &Inputs,
    item: &Incarnation,
    retained: &Retained,
    control: Option<&inspection::Request>,
) -> Result<helper::Incarnation, String> {
    let module = item.module.read(opened, 512 << 20)?;
    let identity = &item.choice_identity;
    if identity.toolchain_build_key != inputs.toolchain_sha256
        || identity.goos != "wasip1"
        || identity.goarch != "wasm"
        || sha2::Sha256::digest(&module).as_slice() != identity.target_sha256.as_slice()
        || identity.runtime_implementation_sha256() != item.runtime_sha256
    {
        return Err("WASM retained cluster choice identity mismatch".into());
    }
    let descriptor = retained
        .evidence
        .get("choice")
        .ok_or("missing retained incarnation choice trace")?;
    let raw = descriptor.read(opened, item.limits.evidence_bytes)?;
    let projected = linked::inspect_choices(&raw, identity).map_err(str::to_owned)?;
    let mut model = item.model.clone();
    let object = model
        .as_object_mut()
        .ok_or("invalid retained cluster model")?;
    if object
        .insert("seed".into(), opened.manifest().record.seed.into())
        .is_some()
        || object.get("mode") != Some(&json!(1))
        || object.contains_key("replay")
        || object.contains_key("inspection")
    {
        return Err("retained cluster must derive seed and replay control".into());
    }
    object.insert(
        "mode".into(),
        json!(if retained.interrupted && control.is_none() {
            3
        } else {
            2
        }),
    );
    object.insert("replay".into(), json!({"schema":"gomad.replay-trace/v1", "execution_identity":identity,
        "trace_sha256":descriptor.sha256,"tape_sha256":projected.tape_sha256,"decisions":projected.decisions,
        "trace_bytes":raw.len(),"trace_frames":(raw.len() as u64).div_ceil(65536),
        "prefix_control":if retained.interrupted && control.is_none() { Some(json!({"cutoff":projected.decisions,"override":null})) } else { None }}));
    let mut configuration: linked::ModelConfig =
        serde_json::from_value(model).map_err(|error| error.to_string())?;
    configuration.inspection = control
        .filter(|control| control.target == item.handle)
        .cloned();
    Ok(helper::Incarnation {
        handle: item.handle.clone(),
        network: item.network.clone(),
        request: linked::Request {
            module,
            module_sha256: item.module.sha256.clone(),
            limits: linked::Limits {
                memory_bytes: item.limits.memory_bytes,
                fuel: item.limits.fuel,
                output_bytes: item.limits.output_bytes,
                evidence_bytes: item.limits.evidence_bytes,
                wall_timeout: super::preflight::wall_timeout(&item.limits, item.wall_nanos)?,
            },
            model: Some(linked::ModelInput {
                configuration,
                arguments: item.arguments.clone(),
                environment: item
                    .environment
                    .iter()
                    .map(|(key, value)| format!("{key}={value}"))
                    .collect(),
                replay_trace: Some(raw),
            }),
            namespace: Some(item.namespace.clone()),
            storage: item.storage.clone(),
        },
    })
}

/// # Errors
/// Authenticates every retained module and incarnation before preparing the helper lifecycle.
pub fn prepare(opened: Opened, control: Option<&inspection::Request>) -> Result<Prepared, String> {
    let inputs = inputs(&opened)?;
    bind(&opened, &inputs)?;
    let mut incarnations = Vec::new();
    for item in &inputs.configuration.incarnations {
        let retained = inputs
            .retained
            .iter()
            .find(|value| value.handle == item.handle)
            .ok_or("missing retained incarnation")?;
        for descriptor in retained.evidence.values() {
            let stream = opened
                .manifest()
                .record
                .streams
                .get(&descriptor.file)
                .ok_or("retained cluster stream is not record bound")?;
            if stream.bytes != descriptor.bytes || stream.sha256 != descriptor.sha256 {
                return Err("retained cluster stream identity mismatch".into());
            }
            descriptor.read(&opened, 1 << 30)?;
        }
        incarnations.push(incarnation(&opened, &inputs, item, retained, control)?);
    }
    if let Some(control) = &control {
        let raw = opened.payload("evidence/inspection", inspection::MAX_METADATA as u64)?;
        let evidence = inspection::Evidence::decode(&raw).map_err(|error| error.to_string())?;
        authenticate_inspection(&opened, &evidence)?;
        if !inputs
            .retained
            .iter()
            .any(|item| item.handle == control.target)
        {
            return Err("unknown cluster inspection target".into());
        }
    }
    opened.authenticate_all()?;
    let configuration = inputs.configuration;
    let execution = helper::prepare(helper::Request {
        nodes: configuration.nodes,
        broker: configuration.broker,
        network: configuration.network,
        incarnations,
        steps: configuration.steps,
        boundaries: configuration.boundaries,
        crash_groups: configuration.crash_groups,
        wall_timeout: super::preflight::wall_timeout(
            &configuration.limits,
            configuration.wall_nanos,
        )?,
        evidence_bytes: configuration.limits.evidence_bytes,
    })
    .map_err(|error| error.to_string())?;
    Ok(Prepared {
        opened,
        execution,
        exact: true,
        available: true,
    })
}

/// # Errors
/// Binds every inspection owner stream to authenticated retained incarnation payloads.
pub fn authenticate_inspection(
    opened: &Opened,
    evidence: &inspection::Evidence,
) -> Result<(), String> {
    let inputs = inputs(opened)?;
    bind(opened, &inputs)?;
    if evidence.incarnations.len() != inputs.retained.len() {
        return Err("cluster inspection incarnation count mismatch".into());
    }
    for observed in &evidence.incarnations {
        let retained = inputs
            .retained
            .iter()
            .find(|item| item.handle == observed.handle)
            .ok_or("unknown cluster inspection incarnation")?;
        for (name, bytes) in [
            ("choice", &observed.streams.choice),
            ("transcript", &observed.streams.wasi),
            ("network", &observed.streams.network),
            ("storage", &observed.streams.storage),
            ("stdout", &observed.streams.stdout),
            ("stderr", &observed.streams.stderr),
        ] {
            let expected = retained
                .evidence
                .get(name)
                .map(|descriptor| descriptor.read(opened, 1 << 30))
                .transpose()?
                .unwrap_or_default();
            if *bytes != expected {
                return Err(format!(
                    "retained cluster inspection stream mismatch: {name}"
                ));
            }
        }
    }
    Ok(())
}

/// # Errors
/// Projects typed cluster observations into a single identity-bound comparison stream.
pub fn project(observed: helper::Execution) -> Result<linked::Execution, String> {
    let mut rows = Vec::new();
    let mut outcome = observed.outcome;
    let mut evidence = BTreeMap::new();
    for item in observed.incarnations {
        let mut streams = BTreeMap::new();
        for (name, raw) in item.execution.evidence {
            streams.insert(
                name.clone(),
                json!({"bytes":raw.len(),"sha256":canonical::digest(&raw)}),
            );
            evidence.insert(
                format!(
                    "cluster/{}/{}/{name}",
                    item.handle.node, item.handle.incarnation
                ),
                raw,
            );
        }
        if outcome.divergence.is_none() {
            outcome.divergence = item.execution.outcome.divergence;
        }
        rows.push(json!({"handle":item.handle,"termination":item.execution.outcome.termination,
            "exit_code":item.execution.outcome.exit_code,"reason":item.execution.outcome.message,"streams":streams}));
    }
    let summary = json!({"schema":"gomad.cluster-observation/v1","incarnations":rows,
        "world_sha256":canonical::digest(&observed.world),"network_sha256":canonical::digest(&observed.network),
        "virtual_elapsed_nanos":observed.virtual_elapsed_nanos});
    evidence.insert("cluster_summary".into(), canonical::encode(&summary)?);
    evidence.insert("world".into(), observed.world);
    evidence.insert("network".into(), observed.network);
    evidence.insert("inspection".into(), observed.inspection);
    outcome.local_model = Some(Box::new(json!({"now":observed.virtual_elapsed_nanos})));
    Ok(linked::Execution {
        outcome,
        evidence,
        event_order: observed.event_order,
    })
}

/// # Errors
/// Returns the selected incarnation's debug output and the shared inspection prefix.
pub fn project_inspection(
    observed: helper::Execution,
    target: &inspection::Handle,
) -> Result<linked::Execution, String> {
    let mut selected = observed
        .incarnations
        .into_iter()
        .find(|item| item.handle == *target)
        .ok_or("cluster inspection target did not execute")?
        .execution;
    selected
        .evidence
        .insert("inspection".into(), observed.inspection);
    selected.event_order = observed.event_order;
    Ok(selected)
}
