use crate::{evidence::canonical, replay::preflight::Configuration};
use gomad_wasmhost::linked;
use serde::Deserialize;
use serde_json::{Value, json};
use sha2::Digest as _;
use std::{
    fs::{self, OpenOptions},
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt},
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    schema: String,
    module_path: PathBuf,
    module_sha256: String,
    seed: u64,
    choice_identity: linked::ChoiceIdentity,
    configuration: Configuration,
}

fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .and_then(|mut file| file.write_all(bytes))
        .map_err(|error| error.to_string())
}

fn execute(
    input: &Input,
    module: &[u8],
    storage: bool,
    trace: Option<&[u8]>,
) -> Result<linked::Execution, String> {
    let config = &input.configuration;
    let mut model = config
        .model
        .clone()
        .ok_or("storage prefix requires model")?;
    let object = model.as_object_mut().ok_or("model must be object")?;
    if object.contains_key("seed") || object.contains_key("replay") {
        return Err("model seed/replay must be derived by fixture".into());
    }
    object.insert("seed".into(), json!(input.seed));
    object.insert("mode".into(), json!(if trace.is_some() { 2 } else { 1 }));
    if let Some(raw) = trace {
        let projection =
            linked::inspect_choices(raw, &input.choice_identity).map_err(str::to_owned)?;
        object.insert("replay".into(), json!({"schema":"gomad.replay-trace/v1",
            "execution_identity":input.choice_identity,"trace_sha256":canonical::digest(raw),
            "tape_sha256":projection.tape_sha256,"decisions":projection.decisions,
            "trace_bytes":raw.len(),"trace_frames":raw.len().div_ceil(65536),"prefix_control":null}));
    }
    let mut prepared = linked::prepare(linked::Request {
        module: module.to_vec(),
        module_sha256: input.module_sha256.clone(),
        limits: linked::Limits {
            memory_bytes: config.limits.memory_bytes,
            fuel: config.limits.fuel,
            output_bytes: config.limits.output_bytes,
            evidence_bytes: config.limits.evidence_bytes,
            wall_timeout: Duration::from_millis(config.limits.wall_millis),
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(model).map_err(|error| error.to_string())?,
            arguments: config.arguments.clone(),
            environment: config
                .environment
                .iter()
                .map(|(key, value)| format!("{key}={value}"))
                .collect(),
            replay_trace: trace.map(<[u8]>::to_vec),
        }),
        namespace: config.namespace.clone(),
        storage: if storage {
            config.storage.clone()
        } else {
            None
        },
    })
    .map_err(|error| error.to_string())?;
    prepared.conditions(linked::Conditions::compiled());
    linked::execute_prepared(prepared).map_err(|error| error.to_string())
}

/// # Errors
/// Requires explicit authenticated fixture inputs and retains actual helper-owned replay results.
pub(crate) fn run(root: &Path, path: &Path) -> Result<(), String> {
    let bytes = fs::read(path).map_err(|error| error.to_string())?;
    let input: Input = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.schema != "gomad.replay-storage-prefix-input/v1"
        || !input.module_path.is_absolute()
        || input.configuration.engine != linked::CONFIGURATION
        || input.configuration.storage.is_none()
    {
        return Err("invalid storage-prefix fixture declaration".into());
    }
    let module = fs::read(&input.module_path).map_err(|error| error.to_string())?;
    if canonical::digest(&module) != input.module_sha256
        || input.choice_identity.runtime_implementation_sha256()
            != input
                .configuration
                .runtime_sha256
                .clone()
                .unwrap_or_default()
    {
        return Err("storage-prefix module/runtime identity mismatch".into());
    }
    let target: [u8; 32] = sha2::Sha256::digest(&module).into();
    if target != input.choice_identity.target_sha256 {
        return Err("storage-prefix target identity mismatch".into());
    }
    fs::DirBuilder::new()
        .mode(0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let full = execute(&input, &module, false, None)?;
    let crash = execute(&input, &module, true, None)?;
    let full_trace = full
        .evidence
        .get("choice")
        .ok_or("missing complete choice trace")?;
    let prefix = crash
        .evidence
        .get("choice")
        .ok_or("missing interrupted choice trace")?;
    let full_plan =
        linked::inspect_choices(full_trace, &input.choice_identity).map_err(str::to_owned)?;
    let prefix_plan =
        linked::inspect_choices(prefix, &input.choice_identity).map_err(str::to_owned)?;
    if full.outcome.termination != "exit"
        || full.outcome.exit_code != Some(0)
        || crash.outcome.termination != "storage_crash"
        || full_plan.decisions != 2
        || prefix_plan.decisions != 1
    {
        return Err("storage fixture did not interrupt between its two choices".into());
    }
    let remaining = execute(&input, &module, true, Some(full_trace))?;
    let matching = execute(&input, &module, true, Some(prefix))?;
    let repeated = execute(&input, &module, true, Some(prefix))?;
    let mut rows = Vec::new();
    for (name, execution) in [
        ("full-record", &full),
        ("crash-record", &crash),
        ("full-replay", &remaining),
        ("prefix-replay", &matching),
        ("prefix-repeat", &repeated),
    ] {
        let mut streams = std::collections::BTreeMap::new();
        for (stream, raw) in &execution.evidence {
            let file = format!("{name}-{stream}.bin");
            if stream.contains('/') || stream.contains("..") {
                return Err("unsafe helper stream name".into());
            }
            write(&root.join(&file), raw)?;
            streams.insert(
                stream.clone(),
                json!({"file":file,"bytes":raw.len(),"sha256":canonical::digest(raw)}),
            );
        }
        rows.push(json!({"name":name,"outcome":execution.outcome,"evidence":streams}));
    }
    let report: Value = json!({"schema":"gomad.replay-storage-prefix-result/v1","input_sha256":canonical::digest(&bytes),
        "module_sha256":input.module_sha256,"helper_sha256":linked::IMPLEMENTATION_SHA256,
        "full_projection":full_plan,"prefix_projection":prefix_plan,"executions":rows});
    let raw = canonical::encode(&report)?;
    write(&root.join("storage-prefix-result.json"), &raw)?;
    println!(
        "{}",
        String::from_utf8(raw).map_err(|error| error.to_string())?
    );
    Ok(())
}
