//! The committed Temporal workload's primary-prefix selection control.
//! Runtime selection is applied after preparation and retains its own preimage.

use super::{History, Workload, native_crash};
use crate::{
    campaign::{prepare::Prepared, scenario::Bound},
    evidence::{canonical, record::Record, store::Opened},
    replay::cluster,
};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{collections::BTreeMap, path::PathBuf};

const CAPSULE: &str = "evidence/primary-crash/";

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(tag = "kind", deny_unknown_fields)]
pub(crate) enum Request {
    #[serde(rename = "sqlite-durable-selection")]
    SQLiteSelection {
        primary_result: PathBuf,
        selection: native_crash::Selection,
    },
    #[serde(rename = "temporal-committed-workflow-crash-reopen")]
    TemporalRecovery { primary: Primary },
    #[serde(rename = "temporal-named-network-reset")]
    TemporalRetry,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Primary {
    pub artifact_path: PathBuf,
    pub record_sha256: String,
    pub prefix_sha256: String,
    pub storage_state_sha256: String,
    pub state: native_crash::State,
    pub context: native_crash::Context,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Forcing {
    pub primary_record_sha256: String,
    pub prefix_sha256: String,
    pub storage_state_sha256: String,
    pub storage_sha256: String,
    pub stdout_sha256: String,
    pub state: native_crash::State,
    pub context: native_crash::Context,
    pub expected: History,
}

impl Primary {
    pub(super) fn validate(&self) -> Result<(), String> {
        if !self.artifact_path.is_absolute()
            || [
                &self.record_sha256,
                &self.prefix_sha256,
                &self.storage_state_sha256,
            ]
            .into_iter()
            .any(|digest| !canonical::valid_digest(digest))
        {
            return Err(
                "Temporal primary requires an authenticated Rust artifact reference".into(),
            );
        }
        native_crash::validate_state(&self.state, &self.context, "temporal", 256)
    }
}

pub(crate) fn from_provenance(raw: &[u8]) -> Result<Option<Request>, String> {
    let provenance: Value = crate::qualify::spec::strict(raw)?;
    serde_json::from_value(
        provenance
            .get("runtime_workload")
            .cloned()
            .unwrap_or(Value::Null),
    )
    .map_err(|error| error.to_string())
}

pub(crate) fn attach(prepared: &mut Prepared, request: &Request) -> Result<(), String> {
    match request {
        Request::SQLiteSelection {
            primary_result,
            selection,
        } => {
            if !primary_result.is_absolute() {
                return Err("SQLite runtime primary Result path must be absolute".into());
            }
            selection.validate()?;
            let raw = prepared
                .scenario
                .as_deref()
                .ok_or("SQLite selection requires its fixed base scenario")?;
            let scenario: Value = crate::qualify::spec::strict(raw)?;
            if scenario["workload"] != json!({"kind":"sqlite-durable-recovery","selection":null}) {
                return Err(
                    "SQLite selection must remain outside its compiled base scenario".into(),
                );
            }
        }
        Request::TemporalRecovery { primary } => {
            primary.validate()?;
            let raw = prepared
                .scenario
                .as_deref()
                .ok_or("Temporal recovery requires its fixed base scenario")?;
            let scenario: Value = crate::qualify::spec::strict(raw)?;
            if scenario["workload"]
                != json!({"kind":"temporal-committed-workflow-crash-reopen","forcing":null})
            {
                return Err(
                    "runtime selection requires the unchanged registered Temporal base scenario"
                        .into(),
                );
            }
        }
        Request::TemporalRetry => {
            if prepared.scenario.is_some() {
                return Err("named reset is the original standalone workload".into());
            }
            validate_retry(&prepared.configuration)?;
        }
    }
    let mut provenance: Value = crate::qualify::spec::strict(&prepared.provenance)?;
    provenance["runtime_workload"] =
        serde_json::to_value(request).map_err(|error| error.to_string())?;
    prepared.provenance = canonical::encode(&provenance)?;
    Ok(())
}

struct Source {
    record: Record,
    inputs: cluster::Inputs,
    payloads: BTreeMap<String, Vec<u8>>,
    forcing: Forcing,
}

fn source(primary: &Primary) -> Result<Source, String> {
    primary.validate()?;
    let opened = Opened::open(&primary.artifact_path)?;
    opened.authenticate_all()?;
    let record = opened.manifest().record.clone();
    if record.hash()? != primary.record_sha256 || record.profile != "gomad.wasi-cooperative/v1" {
        return Err("Temporal primary record identity or lineage changed".into());
    }
    let inputs = cluster::inputs(&opened)?;
    cluster::bind(&opened, &inputs)?;
    if !matches!(
        inputs.configuration.workload,
        Some(Workload::TemporalRecovery { forcing: None })
    ) || inputs.configuration.incarnations.len() != 2
    {
        return Err("Temporal primary is not the unselected original recovery workload".into());
    }
    let mut payloads = BTreeMap::new();
    payloads.insert("record.json".into(), record.bytes()?);
    payloads.insert(
        "cluster-replay.json".into(),
        opened.payload(cluster::FILE, 16 << 20)?,
    );
    for (name, maximum) in [
        ("choice", 1 << 30),
        ("storage_state", 256 << 20),
        ("storage", 128 << 20),
        ("stdout", 8 << 20),
    ] {
        payloads.insert(
            name.into(),
            opened.payload(&format!("evidence/cluster/temporal/1/{name}"), maximum)?,
        );
    }
    let choice = &payloads["choice"];
    let state = &payloads["storage_state"];
    if canonical::digest(choice) != primary.prefix_sha256
        || canonical::digest(state) != primary.storage_state_sha256
    {
        return Err("Temporal primary prefix or storage-state identity changed".into());
    }
    let catalogue = native_crash::temporal(state, &payloads["storage"])?;
    if catalogue.context != primary.context
        || !catalogue
            .enumeration
            .states
            .as_ref()
            .is_some_and(|states| states.contains(&primary.state))
    {
        return Err(
            "Temporal selected state/context differs from complete actual primary catalogue".into(),
        );
    }
    let expected = super::read_history(&payloads["stdout"], "GOMAD_TEMPORAL_OBSERVATION ")?;
    let forcing = Forcing {
        primary_record_sha256: primary.record_sha256.clone(),
        prefix_sha256: primary.prefix_sha256.clone(),
        storage_state_sha256: primary.storage_state_sha256.clone(),
        storage_sha256: canonical::digest(&payloads["storage"]),
        stdout_sha256: canonical::digest(&payloads["stdout"]),
        state: primary.state.clone(),
        context: primary.context.clone(),
        expected,
    };
    Ok(Source {
        record,
        inputs,
        payloads,
        forcing,
    })
}

pub(crate) fn bind(prepared: &Prepared, bound: &mut Bound) -> Result<(), String> {
    let request = from_provenance(&prepared.provenance)?;
    if let Some(Request::SQLiteSelection {
        primary_result,
        selection,
    }) = &request
    {
        return super::sqlite::bind(prepared, bound, primary_result, selection);
    }
    let Some(Request::TemporalRecovery { primary }) = request else {
        return Ok(());
    };
    let source = source(&primary)?;
    if source.record.identity.target != canonical::digest(&prepared.module)
        || source.record.identity.toolchain != prepared.toolchain
        || source.record.seed != bound.request.broker.seed
        || source.record.identity.configuration
            != canonical::digest(&canonical::encode(&bound.configuration)?)
    {
        return Err("Temporal primary belongs to another module, choice identity, candidate or base configuration".into());
    }
    let identity = &source.inputs.configuration.incarnations[0].choice_identity;
    let current = prepared
        .choice_identity
        .as_ref()
        .ok_or("Temporal choice identity missing")?;
    if canonical::encode(identity)? != canonical::encode(current)? {
        return Err("Temporal primary choice identity changed".into());
    }
    let raw = &source.payloads["choice"];
    let projected = linked::inspect_choices(raw, identity).map_err(str::to_owned)?;
    let commit = bound
        .request
        .incarnations
        .iter_mut()
        .find(|item| item.handle.node == "temporal" && item.handle.incarnation == 1)
        .ok_or("Temporal commit boot missing")?;
    let model = commit
        .request
        .model
        .as_mut()
        .ok_or("Temporal commit model missing")?;
    model.configuration.mode = 3;
    model.configuration.replay = Some(serde_json::from_value(json!({"schema":"gomad.replay-trace/v1","execution_identity":identity,
        "trace_sha256":primary.prefix_sha256,"tape_sha256":projected.tape_sha256,"decisions":projected.decisions,
        "trace_bytes":raw.len(),"trace_frames":raw.len().div_ceil(65536),"prefix_control":{"cutoff":projected.decisions,"override":null}})).map_err(|error|error.to_string())?);
    model.replay_trace = Some(raw.clone());
    let storage = commit
        .request
        .storage
        .as_mut()
        .ok_or("Temporal durable commit storage missing")?;
    let mut encoded = serde_json::to_value(&*storage).map_err(|error| error.to_string())?;
    let context = canonical::encode(&BTreeMap::from([("temporal", &primary.context)]))?;
    encoded["descriptor"]["crash_selection"] =
        json!({"temporal":primary.state.selected_operations});
    encoded["descriptor"]["crash_context"] = json!({"bytes":context.len(),"chunks":context.len().div_ceil(65536),"sha256":canonical::digest(&context)});
    encoded["context"] = json!(context);
    *storage = serde_json::from_value(encoded).map_err(|error| error.to_string())?;
    bound
        .configuration
        .incarnations
        .iter_mut()
        .find(|item| item.handle == commit.handle)
        .ok_or("retained Temporal commit boot missing")?
        .storage = Some(storage.clone());
    bound.configuration.workload = Some(Workload::TemporalRecovery {
        forcing: Some(source.forcing),
    });
    Ok(())
}

pub(crate) fn payloads(prepared: &Prepared) -> Result<BTreeMap<String, Vec<u8>>, String> {
    let request = from_provenance(&prepared.provenance)?;
    if let Some(Request::SQLiteSelection {
        primary_result,
        selection,
    }) = &request
    {
        return super::sqlite::payloads(primary_result, selection);
    }
    let Some(Request::TemporalRecovery { primary }) = request else {
        return Ok(BTreeMap::new());
    };
    Ok(source(&primary)?
        .payloads
        .into_iter()
        .map(|(name, bytes)| (format!("{CAPSULE}{name}"), bytes))
        .collect())
}

pub(crate) fn authenticate(opened: &Opened, forcing: &Forcing) -> Result<(), String> {
    native_crash::selected_binding(opened, "temporal", 1, &forcing.state, &forcing.context)?;
    let read = |name: &str, maximum| opened.payload(&format!("{CAPSULE}{name}"), maximum);
    let raw = read("record.json", 1 << 20)?;
    let record: Record = crate::qualify::spec::strict(&raw)?;
    if record.bytes()? != raw
        || record.hash()? != forcing.primary_record_sha256
        || record.identity.implementation != opened.manifest().record.identity.implementation
        || record.identity.toolchain != opened.manifest().record.identity.toolchain
        || record.identity.target != opened.manifest().record.identity.target
        || record.seed != opened.manifest().record.seed
    {
        return Err("retained Temporal primary capsule identity changed".into());
    }
    let cluster_raw = read("cluster-replay.json", 16 << 20)?;
    if record.streams.get(cluster::FILE).is_none_or(|stream| {
        stream.sha256 != canonical::digest(&cluster_raw) || stream.bytes != cluster_raw.len() as u64
    }) {
        return Err("retained Temporal primary cluster capsule binding changed".into());
    }
    let inputs: cluster::Inputs = crate::qualify::spec::strict(&cluster_raw)?;
    if canonical::digest(&canonical::encode(&inputs.configuration)?)
        != record.identity.configuration
        || !matches!(
            inputs.configuration.workload,
            Some(Workload::TemporalRecovery { forcing: None })
        )
    {
        return Err("retained Temporal primary configuration changed".into());
    }
    let mut bytes = BTreeMap::new();
    for (name, maximum, expected) in [
        ("choice", 1 << 30, &forcing.prefix_sha256),
        ("storage_state", 256 << 20, &forcing.storage_state_sha256),
        ("storage", 128 << 20, &forcing.storage_sha256),
        ("stdout", 8 << 20, &forcing.stdout_sha256),
    ] {
        let raw = read(name, maximum)?;
        if canonical::digest(&raw) != *expected
            || record
                .streams
                .get(&format!("evidence/cluster/temporal/1/{name}"))
                .is_none_or(|stream| stream.sha256 != *expected || stream.bytes != raw.len() as u64)
        {
            return Err("retained Temporal primary capsule payload binding changed".into());
        }
        bytes.insert(name, raw);
    }
    let catalogue = native_crash::temporal(&bytes["storage_state"], &bytes["storage"])?;
    if catalogue.context != forcing.context
        || !catalogue
            .enumeration
            .states
            .as_ref()
            .is_some_and(|states| states.contains(&forcing.state))
        || super::read_history(&bytes["stdout"], "GOMAD_TEMPORAL_OBSERVATION ")? != forcing.expected
    {
        return Err("retained Temporal primary catalogue/context/history changed".into());
    }
    let commit = inputs
        .configuration
        .incarnations
        .first()
        .ok_or("Temporal primary commit absent")?;
    linked::inspect_choices(&bytes["choice"], &commit.choice_identity).map_err(str::to_owned)?;
    Ok(())
}

pub(crate) fn matches_request(request: &Request, forcing: &Forcing) -> bool {
    match request {
        Request::TemporalRecovery { primary } => {
            primary.record_sha256 == forcing.primary_record_sha256
                && primary.prefix_sha256 == forcing.prefix_sha256
                && primary.storage_state_sha256 == forcing.storage_state_sha256
                && primary.state == forcing.state
                && primary.context == forcing.context
        }
        Request::TemporalRetry | Request::SQLiteSelection { .. } => false,
    }
}

pub(super) fn validate_retry(
    config: &crate::replay::preflight::Configuration,
) -> Result<(), String> {
    use sha2::Digest as _;
    let model = config
        .model
        .as_ref()
        .ok_or("named reset requires cooperative model")?;
    let expected = [
        "gomad-target",
        "-test.v",
        "-test.count=1",
        "-test.timeout=72h",
        "-test.run=^TestTemporalWorkflowRecoversNamedNetworkReset$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite",
    ];
    let network = serde_json::to_value(
        config
            .network
            .as_ref()
            .ok_or("named reset requires original network model")?,
    )
    .map_err(|error| error.to_string())?;
    if config.limits.memory_bytes != 4 << 30
        || config.limits.fuel != 1 << 50
        || config.limits.output_bytes != 256 << 20
        || config.arguments.iter().map(String::as_str).ne(expected)
        || model["limit"] != 1u64 << 30
        || model["digest_only"].as_bool().unwrap_or(false)
        || model["epoch_nanos"] != 946684800000000000u64
        || model["read_step_nanos"] != 0
        || model["entropy_key"] != json!(sha2::Sha256::digest(b"gomad-wasm-prototype/v1").to_vec())
        || model["limits"]["calls"] != 1u64 << 28
        || model["limits"]["transcript_bytes"] != 1u64 << 30
        || network["profile"] != "gomad.simulation-network-stream/v3"
        || network["limits"]
            != json!({"listeners":128,"connections":4096,"bytes":128<<20,"connection_bytes":4<<20,"stall_nanos":10_000_000_000u64,"transitions":1<<20})
        || network["faults"]
            != json!([{"name":"rpc-reset","action":"reset","at_nanos":40_000_000_000u64,"port":0,"delay_nanos":0}])
    {
        return Err("named reset runtime intent requires its complete original standalone selection and bounds".into());
    }
    Ok(())
}
