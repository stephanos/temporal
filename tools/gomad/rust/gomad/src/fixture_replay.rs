use crate::{
    evidence::{
        canonical,
        identity::Identity,
        inspection::{ChoiceEvidence, Details, Payload},
        record::{Outcome, Record, Stream},
        store::{Opened, Publication, Store},
    },
    replay::preflight::{Configuration, FILE, Inputs, implementation, platform, wall_timeout},
};
use gomad_wasmhost::linked;
use serde::Deserialize;
use sha2::Digest as _;
use std::{collections::BTreeMap, path::Path};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    schema: String,
    cases: Vec<Case>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    module_path: String,
    module_sha256: String,
    toolchain_build_key: String,
    seed: u64,
    choice_identity: Option<linked::ChoiceIdentity>,
    configuration: Configuration,
}

#[expect(
    clippy::too_many_lines,
    reason = "Fixture keeps request construction and the corresponding evidence bindings together"
)]
fn execute(case: Case, conditions: linked::Conditions) -> Result<Publication, String> {
    if !Path::new(&case.module_path).is_absolute() {
        return Err("fixture module must be absolute".into());
    }
    let module = std::fs::read(&case.module_path).map_err(|error| error.to_string())?;
    if module.len() > 512 << 20 || canonical::digest(&module) != case.module_sha256 {
        return Err("fixture module identity mismatch".into());
    }
    let configuration = case.configuration;
    let model = configuration
        .model
        .as_ref()
        .map(|value| {
            let mut value = value.clone();
            let object = value
                .as_object_mut()
                .ok_or("fixture model is not an object")?;
            if object.insert("seed".into(), case.seed.into()).is_some() {
                return Err("fixture seed must come from the execution".to_owned());
            }
            Ok(linked::ModelInput {
                configuration: serde_json::from_value(value).map_err(|error| error.to_string())?,
                arguments: configuration.arguments.clone(),
                environment: configuration
                    .environment
                    .iter()
                    .map(|(key, value)| format!("{key}={value}"))
                    .collect(),
                replay_trace: None,
            })
        })
        .transpose()?;
    let mut prepared = linked::prepare(linked::Request {
        module: module.clone(),
        module_sha256: case.module_sha256.clone(),
        model,
        namespace: configuration.namespace.clone(),
        storage: configuration.storage.clone(),
        limits: linked::Limits {
            memory_bytes: configuration.limits.memory_bytes,
            fuel: configuration.limits.fuel,
            output_bytes: configuration.limits.output_bytes,
            evidence_bytes: configuration.limits.evidence_bytes,
            wall_timeout: wall_timeout(&configuration.limits, configuration.wall_nanos)?,
        },
    })
    .map_err(|error| format!("{}: {error}", case.name))?;
    if let Some(network) = configuration.network.clone() {
        prepared
            .network(network)
            .map_err(|error| error.to_string())?;
    }
    prepared.conditions(conditions);
    let result =
        linked::execute_prepared(prepared).map_err(|error| format!("{}: {error}", case.name))?;
    let mut streams = BTreeMap::new();
    let mut payloads = BTreeMap::from([("target/module.wasm".into(), module)]);
    for (name, bytes) in &result.evidence {
        let path = format!(
            "{}/{name}",
            if matches!(name.as_str(), "stdout" | "stderr") {
                "output"
            } else {
                "evidence"
            }
        );
        let records = result
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| {
                model[if name == "transcript" {
                    "calls"
                } else {
                    "records"
                }]
                .as_u64()
            })
            .unwrap_or(0);
        streams.insert(
            path.clone(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(bytes),
                records,
            },
        );
        payloads.insert(path, bytes.clone());
    }
    if let Some(identity) = case.choice_identity {
        if !payloads.contains_key("evidence/choice") {
            let complete_empty = result.outcome.local_model.as_deref().is_some_and(|model| {
                model["configured"].as_bool() == Some(true)
                    && model["finished"].as_bool() == Some(true)
                    && model["records"].as_u64() == Some(0)
            });
            if !complete_empty {
                return Err("fixture has no choices".into());
            }
            payloads.insert("evidence/choice".into(), Vec::new());
            streams.insert(
                "evidence/choice".into(),
                Stream {
                    bytes: 0,
                    sha256: canonical::digest(&[]),
                    records: 0,
                },
            );
        }
        if identity.target_sha256.as_slice()
            != sha2::Sha256::digest(&payloads["target/module.wasm"]).as_slice()
        {
            return Err("fixture choice module identity mismatch".into());
        }
        let trace = payloads
            .get("evidence/choice")
            .ok_or("fixture has no choices")?;
        let projected = linked::inspect_choices(trace, &identity)?;
        let limit = configuration
            .model
            .as_ref()
            .and_then(|model| model["limit"].as_u64())
            .ok_or("fixture has no choice bound")?;
        let details = Details {
            schema: "gomad.inspection-evidence/v1".into(),
            outputs: None,
            choices: Some(ChoiceEvidence {
                profile: "gomad-choice-trace/v3".into(),
                implementation_sha256: linked::IMPLEMENTATION_SHA256.into(),
                identity,
                trace: Payload {
                    schema: "gomad-choice-trace/v3".into(),
                    file: "evidence/choice".into(),
                    sha256: canonical::digest(trace),
                    bytes: trace.len() as u64,
                    limit,
                },
                records: projected.records,
                terminal_state: "complete".into(),
                tape_sha256: Some(projected.tape_sha256),
            }),
            world: None,
            simulation: None,
            minimization: None,
        };
        let bytes = canonical::encode(&details)?;
        streams.insert(
            "evidence/inspection.json".into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        );
        payloads.insert("evidence/inspection.json".into(), bytes);
    }
    let inputs = Inputs {
        schema: "gomad.replay-inputs/v1".into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: (configuration.profile == "gomad.wasi-cooperative/v1")
            .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
        toolchain_sha256: case.toolchain_build_key.clone(),
        configuration,
    };
    let identity = Identity {
        target: case.module_sha256,
        toolchain: case.toolchain_build_key,
        platform: platform(),
        implementation: implementation()?,
        configuration: canonical::digest(&canonical::encode(&inputs.configuration)?),
    };
    let bytes = canonical::encode(&inputs)?;
    streams.insert(
        FILE.into(),
        Stream {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            records: 1,
        },
    );
    payloads.insert(FILE.into(), bytes);
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        seed: case.seed,
        profile: inputs.configuration.profile,
        arguments: inputs.configuration.arguments,
        environment: inputs.configuration.environment,
        limits: inputs.configuration.limits,
        virtual_time_nanos: result
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| model["now"].as_u64())
            .unwrap_or(0),
        outcome: Outcome {
            class: if result.outcome.termination == "exit" {
                if result.outcome.exit_code == Some(0) {
                    "success"
                } else {
                    "target_failure"
                }
            } else {
                "runner_failure"
            }
            .into(),
            termination: result.outcome.termination.into(),
            exit_code: result.outcome.exit_code,
            reason: result.outcome.message,
        },
        streams,
    };
    payloads.insert("record.json".into(), record.bytes()?);
    Ok(Publication { record, payloads })
}

pub(crate) fn bundle(root: &Path, input: &Path, role: &str) -> Result<(), String> {
    let conditions = match role {
        "ordinary" => linked::Conditions::Ordinary,
        "stress-jitter" => linked::Conditions::stress_replay(),
        "determinism-fixture" => linked::Conditions::determinism_fixture_replay(),
        _ => return Err("unknown linked diagnostic fixture role".into()),
    };
    conditions.validate().map_err(|error| error.to_string())?;
    let raw = std::fs::read(input).map_err(|error| error.to_string())?;
    let corpus: Corpus = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    if corpus.schema != "gomad.replay-fixture-input/v1"
        || corpus.cases.is_empty()
        || corpus.cases.len() > 64
    {
        return Err("invalid replay fixture corpus".into());
    }
    let publications = corpus
        .cases
        .into_iter()
        .map(|case| execute(case, conditions))
        .collect::<Result<Vec<_>, _>>()?;
    crate::fixture::publish_bundle_with_budget(root, "replay-input", publications, 1 << 30)
}

pub(crate) fn divergence(path: &Path, root: &Path, trace: &Path) -> Result<(), String> {
    let opened = Opened::open(path)?;
    opened.authenticate_all()?;
    let mut details = Details::read(&opened)?.ok_or("missing choice metadata")?;
    let choices = details.choices.as_mut().ok_or("missing choice metadata")?;
    let raw = std::fs::read(trace).map_err(|error| error.to_string())?;
    if raw.len() as u64 > choices.trace.limit {
        return Err("fixture trace exceeds bound".into());
    }
    let projected = linked::inspect_choices(&raw, &choices.identity)?;
    choices.trace.sha256 = canonical::digest(&raw);
    choices.trace.bytes = raw.len() as u64;
    choices.records = projected.records;
    choices.tape_sha256 = Some(projected.tape_sha256);
    let mut record = opened.manifest().record.clone();
    let mut payloads = opened
        .manifest()
        .files
        .iter()
        .map(|(name, entry)| Ok((name.clone(), opened.payload(name, entry.bytes)?)))
        .collect::<Result<BTreeMap<_, _>, String>>()?;
    for (name, bytes) in [
        (choices.trace.file.clone(), raw),
        (
            "evidence/inspection.json".into(),
            canonical::encode(&details)?,
        ),
    ] {
        let stream = record
            .streams
            .get_mut(&name)
            .ok_or("unbound fixture stream")?;
        stream.bytes = bytes.len() as u64;
        stream.sha256 = canonical::digest(&bytes);
        payloads.insert(name, bytes);
    }
    payloads.insert("record.json".into(), record.bytes()?);
    let published = Store {
        root: root.into(),
        maximum_bytes: 1 << 30,
        target_pool: None,
        fault: None,
    }
    .publish(Publication { record, payloads })?;
    println!(
        "{}",
        serde_json::json!({"artifact":published.path,"record_sha256":published.record_sha256})
    );
    Ok(())
}
