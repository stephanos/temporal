use crate::{
    evidence::{canonical, store::Opened},
    replay::cluster,
};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::{collections::BTreeMap, path::Path, time::Instant};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Admission {
    schema: String,
    artifact: String,
    operation: String,
    helper_sha256: String,
    primary_sha256: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ObservationInput {
    schema: String,
    configuration: cluster::Configuration,
    modules: BTreeMap<String, String>,
    #[serde(default)]
    controls: Controls,
    #[serde(default)]
    forcing: BTreeMap<String, crate::fixture_cluster::Forcing>,
}

#[derive(Default, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Controls {
    #[serde(default)]
    initial_links: Vec<LinkControl>,
    #[serde(default)]
    observations: Vec<OutputObservation>,
    cleanup: Option<Cleanup>,
    action_limit: Option<usize>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct LinkControl {
    action: String,
    from: String,
    to: String,
    delay: u64,
    symmetric: bool,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct OutputObservation {
    handle: helper::Handle,
    marker: Vec<u8>,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Cleanup {
    mode: String,
    #[serde(default)]
    reverse_insertion: bool,
    after: Option<OutputObservation>,
}

fn write_private(path: &Path, raw: &[u8]) -> Result<(), String> {
    use std::{
        io::Write as _,
        os::unix::fs::{OpenOptionsExt as _, PermissionsExt as _},
    };
    let parent = path.parent().ok_or("missing output parent")?;
    std::fs::create_dir_all(parent).map_err(|error| error.to_string())?;
    std::fs::set_permissions(parent, std::fs::Permissions::from_mode(0o700))
        .map_err(|error| error.to_string())?;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(raw).map_err(|error| error.to_string())
}

fn apply_controls(prepared: &mut helper::Prepared, controls: Controls) -> Result<(), String> {
    if let Some(maximum) = controls.action_limit {
        prepared
            .action_limit(maximum)
            .map_err(|error| error.to_string())?;
    }
    for link in controls.initial_links {
        prepared
            .link_before_start(
                serde_json::from_value(
                    serde_json::to_value(link).map_err(|error| error.to_string())?,
                )
                .map_err(|error| error.to_string())?,
            )
            .map_err(|error| error.to_string())?;
    }
    for observation in controls.observations {
        prepared
            .observe_revocation(observation.handle, observation.marker)
            .map_err(|error| error.to_string())?;
    }
    if let Some(cleanup) = controls.cleanup {
        let mode = match cleanup.mode.as_str() {
            "shutdown" | "action_capacity" => helper::CleanupMode::Shutdown,
            "scenario_error" => helper::CleanupMode::Error("actual scenario failure".into()),
            "context_cancellation" => helper::CleanupMode::ContextCancellation,
            _ => return Err("invalid live scenario callback result".into()),
        };
        if let Some(after) = cleanup.after {
            if cleanup.reverse_insertion {
                return Err("output cleanup insertion override".into());
            }
            prepared
                .cleanup_after_output(after.handle, after.marker, mode)
                .map_err(|error| error.to_string())?;
        } else {
            prepared
                .cleanup_after_starts(mode)
                .map_err(|error| error.to_string())?;
        }
    }
    Ok(())
}

fn observation(root: &Path, mut input: ObservationInput) -> Result<(), String> {
    if input.schema != "gomad.live-cluster-observation/v1" || root.exists() {
        return Err("invalid or existing live cluster observation root".into());
    }
    let modules = input
        .modules
        .into_iter()
        .map(|(name, path)| {
            if !Path::new(&path).is_absolute() {
                return Err("live cluster module must be absolute".to_owned());
            }
            Ok((
                name,
                std::fs::read(path).map_err(|error| error.to_string())?,
            ))
        })
        .collect::<Result<BTreeMap<_, _>, String>>()?;
    if input
        .controls
        .cleanup
        .as_ref()
        .is_some_and(|cleanup| cleanup.reverse_insertion)
    {
        input.configuration.incarnations.reverse();
        input.configuration.nodes.reverse();
    }
    let mut prepared = crate::fixture_cluster::prepare_configuration_forced(
        &input.configuration,
        &modules,
        &input.forcing,
    )?;
    apply_controls(&mut prepared, input.controls)?;
    let activity = prepared.activity();
    let prepared_activity = activity.snapshot();
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let started = Instant::now();
    let mut execution = crate::backstop::within(prepared.wall_timeout(), || {
        helper::execute_prepared(prepared).map_err(|error| error.to_string())
    })?;
    if !execution.world.is_empty() {
        execution.world = helper::encode_world(
            &execution.world,
            input.configuration.encoded_world_format()?,
        )?;
    }
    let wall_elapsed_nanos = started.elapsed().as_nanos();
    let mut incarnations = Vec::new();
    for item in &execution.incarnations {
        let mut streams = BTreeMap::new();
        for (name, bytes) in &item.execution.evidence {
            crate::evidence::store::open::components(name)?;
            write_private(
                &root.join(format!(
                    "incarnations/{}/{}/{name}",
                    item.handle.node, item.handle.incarnation
                )),
                bytes,
            )?;
            streams.insert(
                name,
                json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)}),
            );
        }
        incarnations.push(json!({"handle":item.handle,"interrupted":item.interrupted,
            "outcome":item.execution.outcome,"authorization_to_terminal_nanos":item.execution.authorization_to_terminal_nanos,"choice_bytes":item.execution.evidence.get("choice").map_or(0,Vec::len),
            "streams":streams}));
    }
    let mut streams = BTreeMap::new();
    for (name, bytes) in [
        ("world", &execution.world),
        ("network", &execution.network),
        ("inspection", &execution.inspection),
    ] {
        write_private(&root.join(format!("streams/{name}")), bytes)?;
        streams.insert(
            name,
            json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)}),
        );
    }
    let oracle_assessment =
        cluster::oracles::capacity_observation(&execution, &input.configuration);
    let report = json!({"schema":"gomad.live-cluster-observation/v1", "execution_returned":true,
        "wall_elapsed_nanos":wall_elapsed_nanos,"outcome":execution.outcome,
        "executed_steps":execution.executed_steps,"incarnations":incarnations,"streams":streams,
        "activity_before_execution":prepared_activity,"activity_after_execution":activity.snapshot(),
        "oracle_assessment":oracle_assessment});
    write_private(&root.join("result.json"), &canonical::encode(&report)?)?;
    println!(
        "{}",
        String::from_utf8(canonical::encode(&report)?).map_err(|error| error.to_string())?
    );
    Ok(())
}

pub(crate) fn input(root: &Path, path: &Path) -> Result<(), String> {
    let raw = std::fs::read(path).map_err(|error| error.to_string())?;
    let value: serde_json::Value =
        serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    if value["schema"] == "gomad.live-cluster-observation/v1" {
        return observation(
            root,
            serde_json::from_value(value).map_err(|error| error.to_string())?,
        );
    }
    if value["schema"] == "gomad.cluster-fixture-input/v1" {
        return crate::fixture_cluster::input(root, path);
    }
    let input: Admission = serde_json::from_value(value).map_err(|error| error.to_string())?;
    if input.schema != "gomad.live-cluster-admission/v1"
        || !matches!(
            input.operation.as_str(),
            "verify" | "storage-selections" | "reopen" | "replay"
        )
        || input.helper_sha256 != linked::IMPLEMENTATION_SHA256
    {
        return Err("live cluster admission schema, operation or helper binding".into());
    }
    let opened = Opened::open(Path::new(&input.artifact))?;
    if input.primary_sha256 != opened.manifest().record.identity.target {
        return Err("live cluster primary module binding".into());
    }
    let inputs = cluster::inputs(&opened)?;
    let expected = opened.manifest().record.clone();
    let prepared = cluster::prepare(opened, None)?;
    let result = match input.operation.as_str() {
        "verify" => json!({"verified":true}),
        "storage-selections" => json!({"selections": inputs.configuration.incarnations.iter()
            .map(|item| json!({"handle":item.handle,"storage":item.storage})).collect::<Vec<_>>()}),
        "reopen" => {
            json!({"incarnations":inputs.configuration.incarnations.len(),"available":prepared.available})
        }
        "replay" => {
            crate::fixture::execution_started().map_err(|error| error.message)?;
            let started = Instant::now();
            let observed = cluster::execute(prepared.execution)?;
            let completed: Vec<_> = observed
                .incarnations
                .iter()
                .map(|item| {
                    json!({
                        "handle":item.handle,"interrupted":item.interrupted,
                        "termination":item.execution.outcome.termination,
                        "metrics":item.execution.outcome.metrics,
                    })
                })
                .collect();
            let oracle_report = cluster::oracles::assess(&observed, &inputs.configuration)?;
            let mut observed = cluster::project_encoded(
                observed,
                inputs.configuration.encoded_world_format()?,
                &inputs.configuration.broker,
            )?;
            cluster::oracles::apply(&mut observed, &oracle_report)?;
            for (name, bytes) in &observed.evidence {
                let stream = expected
                    .streams
                    .get(&format!("evidence/{name}"))
                    .ok_or("live replay produced unrecorded stream")?;
                if stream.bytes != bytes.len() as u64 || stream.sha256 != canonical::digest(bytes) {
                    return Err(format!("live cluster replay differs at {name}"));
                }
            }
            if observed.outcome.termination != expected.outcome.termination
                || observed.outcome.exit_code != expected.outcome.exit_code
                || observed.outcome.message != expected.outcome.reason
            {
                return Err("live cluster replay terminal differs".into());
            }
            json!({"reproduced":true,"completed":completed,
                "record_sha256":canonical::digest(&expected.bytes()?),
                "wall_elapsed_nanos":started.elapsed().as_nanos()})
        }
        _ => return Err("live cluster operation".into()),
    };
    if root.exists()
        && std::fs::read_dir(root)
            .map_err(|error| error.to_string())?
            .next()
            .is_some()
    {
        return Err("live cluster admission output is not empty".into());
    }
    println!(
        "{}",
        String::from_utf8(canonical::encode(&result)?).map_err(|error| error.to_string())?
    );
    Ok(())
}
