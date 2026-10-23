use crate::evidence::{canonical, record::Limits, store::Opened};
use gomad_wasmhost::{
    linked, linked_cluster as helper, linked_inspection as inspection,
    linked_simulation as simulation,
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::Digest as _;
use std::collections::{BTreeMap, BTreeSet};

pub(crate) mod oracles;
pub use oracles::Plan as OraclePlan;

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

pub use linked::ModuleImport as ImportBinding;

pub(crate) fn module_imports(bytes: &[u8]) -> Result<Vec<ImportBinding>, String> {
    linked::module_metadata(bytes)
        .map(|metadata| metadata.imports)
        .map_err(|error| error.to_string())
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Incarnation {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub workload_kind: String,
    pub handle: helper::Handle,
    pub module: Payload,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub imports: Option<Vec<ImportBinding>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub storage_crash_selection: Option<BTreeMap<String, Vec<u64>>>,
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

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct OriginalLimits {
    pub network_transitions: u64,
    pub observation_bytes: u64,
}

pub(crate) fn storage_input(item: &Incarnation) -> Result<Option<linked::StorageInput>, String> {
    let Some(selection) = item
        .storage_crash_selection
        .as_ref()
        .filter(|selection| !selection.is_empty())
    else {
        return Ok(item.storage.clone());
    };
    let storage = item
        .storage
        .as_ref()
        .ok_or("storage crash selection requires declared storage volumes")?;
    let mut value = serde_json::to_value(storage).map_err(|error| error.to_string())?;
    value["descriptor"]["crash_selection"] = json!(selection);
    serde_json::from_value(value)
        .map(Some)
        .map_err(|error| error.to_string())
}

#[derive(Clone, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Configuration {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub original_limits: Option<OriginalLimits>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub oracle_plan: Option<OraclePlan>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub workload: Option<crate::qualify::workloads::Workload>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub simulation_original: Option<Box<Configuration>>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub nodes: Vec<String>,
    pub engine: String,
    pub profile: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub world_encoding: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub world_profile: Option<String>,
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

impl Configuration {
    pub(crate) fn encoded_world_format(&self) -> Result<&str, String> {
        if let Some(limits) = &self.original_limits {
            if limits.network_transitions == 0
                || limits.network_transitions > 1 << 20
                || limits.observation_bytes == 0
                || limits.observation_bytes > 64 << 20
            {
                return Err("invalid original cluster scenario limits".into());
            }
            if self.world_profile.as_deref() == Some("") {
                let broker = &self.broker.limits;
                if broker.max_requests != limits.network_transitions * 2
                    || broker.max_events != limits.network_transitions * 2
                    || broker.max_queued_events != (limits.network_transitions * 2).min(65536)
                    || broker.max_transitions != limits.network_transitions * 8
                    || broker.max_payload_bytes != limits.observation_bytes
                    || broker.max_string_bytes != 1024
                {
                    return Err("original scenario limits differ from derived broker bounds".into());
                }
            }
        }
        if let Some(profile) = &self.world_profile {
            if profile != &self.world_encoding
                || !matches!(profile.as_str(), "" | "gomad/world/compressed-snapshot/v2")
            {
                return Err("retained World profile/encoding mismatch".into());
            }
            Ok("gomad/world/compressed-snapshot/v2")
        } else if self.world_encoding.is_empty() {
            Ok("")
        } else {
            Err("explicit World encoding requires retained profile".into())
        }
    }
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
    pub execution: Execution,
    pub exact: bool,
    pub available: bool,
}

pub struct Execution {
    oracle_configuration: Configuration,
    world_encoding: String,
    pub(crate) prepared: helper::Prepared,
    simulation: Option<simulation::Plan>,
}

pub(crate) fn execute(execution: Execution) -> Result<helper::Execution, String> {
    crate::backstop::within(execution.prepared.wall_timeout(), || {
        if let Some(plan) = execution.simulation {
            Ok(simulation::execute(execution.prepared, plan, 65536)?.cluster)
        } else {
            helper::execute_prepared(execution.prepared).map_err(|error| error.to_string())
        }
    })
}

pub(crate) struct Projection {
    configuration: Configuration,
    world_encoding: String,
    simulation: bool,
}

impl Execution {
    pub(crate) fn projection(&self) -> Projection {
        Projection {
            configuration: self.oracle_configuration.clone(),
            world_encoding: self.world_encoding.clone(),
            simulation: self.simulation.is_some(),
        }
    }
}

impl Projection {
    pub(crate) fn project(self, observed: helper::Execution) -> Result<linked::Execution, String> {
        let oracle_report = oracles::assess(&observed, &self.configuration)?;
        let primary = self.simulation.then(|| {
            observed
                .incarnations
                .first()
                .and_then(|item| item.execution.evidence.get("choice"))
                .cloned()
                .unwrap_or_default()
        });
        let mut projected = project_workload(
            observed,
            self.configuration.workload.as_ref(),
            &self.world_encoding,
        )?;
        oracles::apply(&mut projected, &oracle_report)?;
        if let Some(primary) = primary {
            projected.evidence.insert("choice".into(), primary);
        }
        Ok(projected)
    }
}

pub(crate) fn execute_projected(execution: Execution) -> Result<linked::Execution, String> {
    let projection = execution.projection();
    projection.project(execute(execution)?)
}

fn simulation_plan(opened: &Opened, inputs: &Inputs) -> Result<Option<simulation::Plan>, String> {
    use crate::campaign::exploration::{simulation_record, simulation_record_wire};
    let details = crate::evidence::inspection::Details::read(opened)?;
    let profile = details
        .as_ref()
        .and_then(|details| details.simulation.as_ref());
    let Some(profile) = profile else {
        if ["simulation/plan.json", "simulation/record.json"]
            .iter()
            .any(|name| opened.manifest().files.contains_key(*name))
        {
            return Err("retained simulation inputs require a simulation profile".into());
        }
        return Ok(None);
    };
    for descriptor in [&profile.plan, &profile.record] {
        let stream = opened
            .manifest()
            .record
            .streams
            .get(&descriptor.file)
            .ok_or("retained simulation payload is not record bound")?;
        if stream.bytes != descriptor.bytes || stream.sha256 != descriptor.sha256 {
            return Err("retained simulation stream identity mismatch".into());
        }
    }
    let raw = opened.payload(
        &profile.plan.file,
        simulation_record_wire::MAXIMUM_PLAN_BYTES as u64,
    )?;
    let record = opened.payload(
        &profile.record.file,
        simulation_record::MAXIMUM_RECORD_BYTES as u64,
    )?;
    simulation_record::validate_artifact(profile, &raw, &record)?;
    let retained = simulation_record_wire::validate_retained_plan(&raw)?;
    if retained.base_seed != opened.manifest().record.seed {
        return Err("retained simulation seed differs from execution".into());
    }
    if retained.overrides.iter().flatten().any(|forced| {
        forced.dimension == crate::campaign::exploration::simulation::Dimension::Runtime
    }) {
        let primary = inputs
            .configuration
            .incarnations
            .first()
            .ok_or("retained simulation primary incarnation missing")?;
        let retained_primary = inputs
            .retained
            .iter()
            .find(|item| item.handle == primary.handle)
            .ok_or("retained simulation primary evidence missing")?;
        let trace = retained_primary
            .evidence
            .get("choice")
            .map(|payload| payload.read(opened, 1 << 30))
            .transpose()?
            .unwrap_or_default();
        let (config, candidate) = simulation_record_wire::candidate_for_artifact(
            profile,
            &raw,
            Some(&trace),
            Some(&primary.choice_identity),
        )
        .map_err(|error| format!("retained runtime forcing authentication: {error}"))?;
        let execution = simulation_record_wire::execution_for_candidate(
            &config,
            &candidate,
            &primary.choice_identity,
        )
        .map_err(|error| format!("retained runtime forcing authentication: {error}"))?;
        if execution.simulation_plan != raw {
            return Err("retained simulation forcing differs from its actual choice tape".into());
        }
    }
    simulation::Plan::decode(&raw).map(Some)
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

pub(crate) fn bind(opened: &Opened, inputs: &Inputs) -> Result<(), String> {
    let record = &opened.manifest().record;
    if inputs.helper_sha256 != linked::IMPLEMENTATION_SHA256
        || inputs.host_sha256 != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || inputs.runtime_implementation_sha256 != env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256")
        || inputs.toolchain_sha256 != record.identity.toolchain
        || record.identity.implementation != super::preflight::implementation()?
        || record.identity.platform != super::preflight::platform()
        || inputs.configuration.engine != linked::Conditions::compiled().configuration()
        || inputs.configuration.profile != record.profile
        || record.profile != "gomad.wasi-cooperative/v1"
        || inputs.configuration.limits != record.limits
        || inputs.configuration.broker.seed != record.seed
        || canonical::digest(&canonical::encode(&inputs.configuration)?)
            != record.identity.configuration
    {
        return Err("WASM retained cluster implementation/configuration identity mismatch".into());
    }
    if let Some(crate::qualify::workloads::Workload::SQLite {
        selection: Some(selection),
    }) = &inputs.configuration.workload
    {
        crate::qualify::workloads::sqlite::authenticate(opened, selection)?;
    }
    if let Some(crate::qualify::workloads::Workload::TemporalRecovery {
        forcing: Some(forcing),
    }) = &inputs.configuration.workload
    {
        crate::qualify::workloads::recovery::authenticate(opened, forcing)?;
    }
    let original = inputs.configuration.simulation_original.as_deref();
    if let Some(original) = original {
        if original.simulation_original.is_some() {
            return Err("nested simulation replay recipe".into());
        }
        let mut expected = original.clone();
        crate::campaign::scenario::retain_execution_prefix(
            &mut expected,
            &inputs.configuration.steps,
            true,
        )?;
        if canonical::encode(&expected)? != canonical::encode(&inputs.configuration)? {
            return Err("simulation replay recipe differs from its actual lifecycle".into());
        }
    }
    let recipe = original.unwrap_or(&inputs.configuration);
    if recipe.incarnations.is_empty()
        || recipe.incarnations.len() > 65536
        || inputs.retained.len() != inputs.configuration.incarnations.len()
    {
        return Err("retained cluster incarnation count mismatch".into());
    }
    if record.outcome.termination == "exit" && record.outcome.exit_code == Some(0) {
        let mut live = BTreeSet::new();
        for step in &inputs.configuration.steps {
            if matches!(step.action, helper::Action::Start | helper::Action::Restart) {
                live.insert(step.handle.clone());
            } else {
                live.remove(&step.handle);
            }
        }
        if !live.is_empty() {
            return Err("retained completed cluster lacks terminal lifecycle action".into());
        }
    }
    let primary = &recipe.incarnations[0].module;
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
    retained: Option<&Retained>,
    control: Option<&inspection::Request>,
) -> Result<helper::Incarnation, String> {
    let module = item.module.read(opened, 512 << 20)?;
    if let Some(imports) = &item.imports
        && module_imports(&module)? != *imports
    {
        return Err("retained cluster import binding mismatch".into());
    }
    let identity = &item.choice_identity;
    if identity.toolchain_build_key != inputs.toolchain_sha256
        || identity.goos != "wasip1"
        || identity.goarch != "wasm"
        || sha2::Sha256::digest(&module).as_slice() != identity.target_sha256.as_slice()
        || identity.runtime_implementation_sha256() != item.runtime_sha256
    {
        return Err("WASM retained cluster choice identity mismatch".into());
    }
    let descriptor = retained.and_then(|retained| retained.evidence.get("choice"));
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
    let raw = if let (Some(retained), Some(descriptor)) = (retained, descriptor) {
        let raw = descriptor.read(opened, item.limits.evidence_bytes)?;
        let projected = linked::inspect_choices(&raw, identity).map_err(str::to_owned)?;
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
        Some(raw)
    } else if retained.is_none() && inputs.configuration.simulation_original.is_some() {
        None
    } else {
        return Err("missing retained incarnation choice trace".into());
    };
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
                replay_trace: raw,
            }),
            namespace: Some(item.namespace.clone()),
            storage: storage_input(item)?,
        },
    })
}

fn configure_recovery(
    execution: &mut helper::Prepared,
    workload: Option<&crate::qualify::workloads::Workload>,
) -> Result<(), String> {
    match workload {
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
        workload,
        Some(crate::qualify::workloads::Workload::TemporalRecovery { forcing: None })
    ) {
        execution
            .committed_temporal_reopens()
            .map_err(|error| error.to_string())?;
    }
    Ok(())
}

/// # Errors
/// Authenticates every retained module and incarnation before preparing the helper lifecycle.
pub fn prepare(opened: Opened, control: Option<&inspection::Request>) -> Result<Prepared, String> {
    let inputs = inputs(&opened)?;
    oracles::validate(&inputs.configuration)?;
    bind(&opened, &inputs)?;
    let world = opened.payload("evidence/world", 384 << 20)?;
    if !world.is_empty() {
        helper::validate_encoded_world(
            &world,
            inputs.configuration.encoded_world_format()?,
            &inputs.configuration.broker,
        )?;
    }
    let simulation = simulation_plan(&opened, &inputs)?;
    if simulation.is_some() != inputs.configuration.simulation_original.is_some() {
        return Err("simulation replay requires its original authenticated recipe".into());
    }
    let recipe = inputs
        .configuration
        .simulation_original
        .as_deref()
        .unwrap_or(&inputs.configuration);
    let mut incarnations = Vec::new();
    for item in &recipe.incarnations {
        let retained = inputs
            .retained
            .iter()
            .find(|value| value.handle == item.handle);
        for descriptor in retained
            .into_iter()
            .flat_map(|retained| retained.evidence.values())
        {
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
    let configuration = inputs
        .configuration
        .simulation_original
        .as_deref()
        .unwrap_or(&inputs.configuration)
        .clone();
    let oracle_configuration = configuration.clone();
    let mut execution = helper::prepare(helper::Request {
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
    configure_recovery(&mut execution, configuration.workload.as_ref())?;
    Ok(Prepared {
        opened,
        execution: Execution {
            oracle_configuration,
            prepared: execution,
            world_encoding: inputs.configuration.encoded_world_format()?.into(),
            simulation,
        },
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
    project_encoded(observed, "")
}

/// # Errors
/// Refuses invalid World encoding or observations that exceed the retained stream bounds.
pub fn project_encoded(
    mut observed: helper::Execution,
    encoding: &str,
) -> Result<linked::Execution, String> {
    if !observed.world.is_empty() {
        observed.world = helper::encode_world(&observed.world, encoding)?;
    }
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
        let local_model =
            item.execution.outcome.local_model.as_deref().map(
                |model| json!({"configured":model["configured"],"finished":model["finished"]}),
            );
        rows.push(json!({"handle":item.handle,"termination":item.execution.outcome.termination,
            "exit_code":item.execution.outcome.exit_code,"reason":item.execution.outcome.message,"streams":streams,
            "offered_output_bytes":item.execution.outcome.output_bytes,"local_model":local_model}));
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
        authorization_to_terminal_nanos: None,
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

/// Projects the sealed mandatory oracle against fresh execution output.
/// # Errors
/// Refuses missing, ambiguous or malformed actual workload observations.
pub(crate) fn project_workload(
    observed: helper::Execution,
    workload: Option<&crate::qualify::workloads::Workload>,
    encoding: &str,
) -> Result<linked::Execution, String> {
    let mut projected = project_encoded(observed, encoding)?;
    if let Some(workload) = workload {
        crate::qualify::workloads::project(workload, &mut projected)?;
    }
    Ok(projected)
}

#[cfg(test)]
mod projection_tests {
    use super::*;

    const ENCODING: &str = "gomad/world/compressed-snapshot/v2";

    fn configuration() -> (Configuration, Vec<u8>) {
        // Expanded bytes from the unchanged compressed-world-single-terminal-owner corpus.
        let raw_world = crate::qualify::cells::base64("R01XU05BUDEAAAABAAAAAAAAAAAAAAAAACAAAAAAAAAAIAAAAAAAAAAD0JAAAAAAAIAAAAAAAAAEAAAA/////w0jTM9SQwAAAAAAAAAAAAEAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFVkPwAshKEN5wBwZmobp69AiLehiWr2Xs5nRltJYTzTVQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==").unwrap();
        let limits = json!({"fuel":100_000,"memory_bytes":65_536,"wall_millis":1000,
            "output_bytes":1<<20,"evidence_bytes":64<<20});
        let namespace = json!({"entries":[],"writable_directories":[],"preopens":[[47]],
            "working_directory":[47],"stdin":[],"epoch_nanos":0,
            "limits":{"files":1024,"descriptors":128,"filesystem_bytes":8<<20,"output_bytes":1<<20}});
        let owners = [2, 3].map(|incarnation| json!({
            "handle":{"node":"sqlite","incarnation":incarnation},
            "module":{"file":"target/module.wasm","bytes":0,"sha256":"a".repeat(64),"limit":512<<20},
            "limits":limits,"arguments":[],"environment":{},"runtime_sha256":"b".repeat(64),
            "model":{},"namespace":namespace,"storage":null,"network":null,
            "choice_identity":{"TargetSHA256":vec![0u8;32],"ToolchainBuildKey":"c".repeat(64),
                "GOOS":"wasip1","GOARCH":"wasm","ImplementationSHA256":vec![0u8;32]}
        }));
        let configuration: Configuration = serde_json::from_value(json!({
            "workload":{"kind":"sqlite-durable-recovery","selection":null},
            "oracle_plan":{"schema":"gomad.wasm.history-observation/v1","prefix":"LIVE ","maximum_bytes":1<<20,
                "histories":[{"name":"live-history","expected":{"node":"sqlite","incarnation":2},
                    "actual":{"node":"sqlite","incarnation":3},"expected_events":["1","2"]}]},
            "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1",
            "world_encoding":ENCODING,"world_profile":ENCODING,"limits":limits,
            "broker":{"seed":0,"limits":{"max_requests":2_097_152,"max_events":2_097_152,
                "max_queued_events":250_000,"max_transitions":8_388_608,"max_payload_bytes":67_108_864,"max_string_bytes":u32::MAX}},
            "network":null,"incarnations":owners,"steps":[],"boundaries":[],"crash_groups":[]
        })).unwrap();
        helper::validate_world(&raw_world, &configuration.broker).unwrap();
        (configuration, raw_world)
    }

    fn observe(raw_world: &[u8], bad_live: bool, bad_mandatory: bool) -> helper::Execution {
        let history = |payload| {
            format!(
                "LIVE {}\n",
                json!({"cluster_id":"sqlite",
                "history":payload,"event_ids":[1,2],"types":["a","b"]})
            )
        };
        let outputs = [
            format!("ACK 2\n{}", history("aGlzdG9yeQ==")),
            format!(
                "INTEGRITY ok\nHISTORY {}\n{}",
                if bad_mandatory {
                    "1:committed"
                } else {
                    "1:committed,2:committed"
                },
                history(if bad_live { "YmFk" } else { "aGlzdG9yeQ==" })
            ),
        ];
        let mut outcome = linked::Outcome::new("exit", "");
        outcome.exit_code = Some(0);
        helper::Execution {
            outcome,
            executed_steps: vec![],
            incarnations: outputs
                .into_iter()
                .enumerate()
                .map(|(index, stdout)| {
                    let mut outcome = linked::Outcome::new("exit", "");
                    outcome.exit_code = Some(0);
                    helper::IncarnationResult {
                        handle: helper::Handle {
                            node: "sqlite".into(),
                            incarnation: u64::try_from(index).unwrap() + 2,
                        },
                        interrupted: false,
                        execution: linked::Execution {
                            authorization_to_terminal_nanos: None,
                            outcome,
                            evidence: BTreeMap::from([("stdout".into(), stdout.into_bytes())]),
                            event_order: None,
                        },
                    }
                })
                .collect(),
            world: raw_world.to_vec(),
            network: vec![],
            inspection: vec![],
            virtual_elapsed_nanos: 0,
            event_order: None,
        }
    }

    #[test]
    fn inspection_metadata_cannot_replace_the_actual_session_world() {
        let (_, raw_world) = configuration();
        let mut missing = observe(&[], false, false);
        missing.inspection = br#"{"schema":"retained"}"#.to_vec();
        assert!(project_encoded(missing, "").is_err());

        let mut present = observe(&raw_world, false, false);
        present.inspection = br#"{"schema":"retained"}"#.to_vec();
        let projected = project_encoded(present, "").unwrap();
        assert_eq!(projected.evidence["world"], raw_world);
        assert_eq!(
            projected.evidence["inspection"],
            br#"{"schema":"retained"}"#
        );
        let summary: Value =
            serde_json::from_slice(&projected.evidence["cluster_summary"]).unwrap();
        assert_eq!(summary["world_sha256"], canonical::digest(&raw_world));
        assert_ne!(
            summary["world_sha256"],
            canonical::digest(&projected.evidence["inspection"])
        );
    }

    #[test]
    fn shared_projection_preserves_mandatory_live_oracles_and_retained_world() {
        let (configuration, raw_world) = configuration();
        for (bad_live, bad_mandatory) in [(false, false), (true, false), (false, true)] {
            let projected = Projection {
                configuration: configuration.clone(),
                world_encoding: ENCODING.into(),
                simulation: false,
            }
            .project(observe(&raw_world, bad_live, bad_mandatory))
            .unwrap();
            let mandatory: Value =
                serde_json::from_slice(&projected.evidence["workload-oracles.json"]).unwrap();
            let live: Value =
                serde_json::from_slice(&projected.evidence["cluster_oracles"]).unwrap();
            assert_eq!(
                mandatory["checks"]
                    .as_object()
                    .unwrap()
                    .values()
                    .all(|check| check == true),
                !bad_mandatory
            );
            assert_eq!(live["oracles"].as_array().unwrap().len(), 3);
            assert_eq!(
                live["outcome"],
                if bad_live {
                    "oracle_failed"
                } else {
                    "completed"
                }
            );
            assert_eq!(
                projected.outcome.termination,
                if bad_live { "oracle_failed" } else { "exit" }
            );
            assert_eq!(
                projected.outcome.exit_code,
                if bad_live { None } else { Some(0) }
            );
            if bad_mandatory {
                assert_eq!(
                    projected.outcome.message,
                    "mandatory workload oracle failed"
                );
            } else if !bad_live {
                assert_eq!(projected.outcome.message, "");
            }
            let world = &projected.evidence["world"];
            assert_ne!(world, &raw_world);
            assert!(world.starts_with(b"GMWORLD2"));
            helper::validate_encoded_world(world, ENCODING, &configuration.broker).unwrap();
            let summary: Value =
                serde_json::from_slice(&projected.evidence["cluster_summary"]).unwrap();
            assert_eq!(summary["world_sha256"], canonical::digest(world));
        }
    }
}
