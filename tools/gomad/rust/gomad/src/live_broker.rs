use crate::{evidence::canonical, replay::preflight::Configuration};
use gomad_wasmhost::linked;
use serde::Deserialize;
use serde_json::json;
use std::{path::Path, time::Instant};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    schema: String,
    module_path: String,
    module_sha256: String,
    seed: u64,
    configuration: Configuration,
    forcing: Option<Forcing>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Forcing {
    mode: u8,
    trace_path: String,
    identity: linked::ChoiceIdentity,
}

pub(crate) fn input(root: &Path, path: &Path) -> Result<(), String> {
    let bytes = std::fs::read(path).map_err(|error| error.to_string())?;
    let input: Input = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.schema != "gomad.live-broker-input/v1" || !Path::new(&input.module_path).is_absolute()
    {
        return Err("invalid live broker input".into());
    }
    let module = std::fs::read(&input.module_path).map_err(|error| error.to_string())?;
    if module.len() > 8 << 20 || canonical::digest(&module) != input.module_sha256 {
        return Err("live broker small module identity/bound mismatch".into());
    }
    let configuration = input.configuration;
    let mut value = configuration
        .model
        .ok_or("live broker requires cooperative model")?;
    if value
        .as_object_mut()
        .ok_or("invalid live broker model")?
        .insert("seed".into(), input.seed.into())
        .is_some()
    {
        return Err("live broker seed must derive from request".into());
    }
    let mut model = linked::ModelInput {
        configuration: serde_json::from_value(value).map_err(|error| error.to_string())?,
        arguments: configuration.arguments,
        environment: configuration
            .environment
            .into_iter()
            .map(|(key, value)| format!("{key}={value}"))
            .collect(),
        replay_trace: None,
    };
    if let Some(forcing) = input.forcing {
        if !matches!(forcing.mode, 2 | 3) || !Path::new(&forcing.trace_path).is_absolute() {
            return Err("invalid live broker forcing".into());
        }
        let trace = std::fs::read(forcing.trace_path).map_err(|error| error.to_string())?;
        if trace.len() as u64 > model.configuration.limit {
            return Err("live broker source trace exceeds choice bound".into());
        }
        let plan = linked::project_choices(&trace, &forcing.identity)?;
        model.configuration.mode = forcing.mode;
        model.configuration.replay = Some(linked::ReplayRequest {
            schema: "gomad.replay-trace/v1".into(),
            execution_identity: forcing.identity,
            trace_sha256: canonical::digest(&trace),
            tape_sha256: canonical::digest(&plan.bytes),
            decisions: plan.decisions.len() as u64,
            trace_bytes: trace.len() as u64,
            trace_frames: (trace.len() as u64).div_ceil(65536),
            prefix_control: None,
        });
        model.replay_trace = Some(trace);
    }
    let wall_started = Instant::now();
    let prepared = linked::prepare(linked::Request {
        module,
        module_sha256: input.module_sha256,
        limits: linked::Limits {
            memory_bytes: configuration.limits.memory_bytes,
            fuel: configuration.limits.fuel,
            output_bytes: configuration.limits.output_bytes,
            evidence_bytes: configuration.limits.evidence_bytes,
            wall_timeout: crate::replay::preflight::wall_timeout(
                &configuration.limits,
                configuration.wall_nanos,
            )?,
        },
        model: Some(model),
        namespace: configuration.namespace,
        storage: configuration.storage,
    })
    .map_err(|error| error.to_string())?;
    let input_admission_nanos = wall_started.elapsed().as_nanos();
    let execution = crate::backstop::within(prepared.wall_timeout(), || {
        linked::execute_prepared(prepared)
    })
    .map_err(|error| error.to_string())?;
    let wall_elapsed_nanos = wall_started.elapsed().as_nanos();
    std::fs::create_dir(root).map_err(|error| error.to_string())?;
    let mut streams = serde_json::Map::new();
    for (name, raw) in execution.evidence {
        if name.contains('/') || name.contains('\\') || name == "." || name == ".." {
            return Err("invalid live broker evidence name".into());
        }
        std::fs::write(root.join(&name), &raw).map_err(|error| error.to_string())?;
        streams.insert(
            name,
            json!({"bytes":raw.len(),"sha256":canonical::digest(&raw)}),
        );
    }
    let result = json!({"schema":"gomad.live-broker-result/v1", "outcome":execution.outcome,
        "owner":"rust-linked", "helper_implementation_sha256":linked::IMPLEMENTATION_SHA256,
        "input_admission_nanos":input_admission_nanos, "wall_elapsed_nanos":wall_elapsed_nanos,
        "authorization_to_terminal_nanos":execution.authorization_to_terminal_nanos, "streams":streams});
    std::fs::write(root.join("result.json"), canonical::encode(&result)?)
        .map_err(|error| error.to_string())
}
