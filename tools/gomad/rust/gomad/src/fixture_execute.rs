use crate::{
    evidence::{
        canonical,
        identity::Identity,
        record::{Limits, Outcome, Record, Stream},
        store::Publication,
    },
    execute,
};
use gomad_wasmhost::linked;
use serde::Deserialize;
use serde_json::json;
use sha2::Digest as _;
use std::{collections::BTreeMap, path::Path, sync::Barrier, time::Duration};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    schema: String,
    profile: String,
    read_step_nanos: u64,
    seeds: Vec<u64>,
    cases: Vec<Case>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Case {
    name: String,
    module: String,
    fuel: u64,
    memory_bytes: u64,
    wall_millis: u64,
    modeled: bool,
    #[serde(default)]
    runtime: bool,
    choice_limit: Option<u64>,
    termination: String,
    compiler_sha256: Option<String>,
    module_sha256: Option<String>,
    runtime_implementation_sha256: Option<String>,
}

pub(crate) fn model_configuration(
    seed: u64,
    choice_limit: u64,
    read_step_nanos: u64,
) -> serde_json::Value {
    json!({"entropy_key":vec![0_u8;32],"working_directory_sha256":canonical::digest(b"/"),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":seed,"mode":1,"limit":choice_limit,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":read_step_nanos,"now":0,"limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}})
}

pub(crate) fn namespace() -> linked::NamespaceImage {
    linked::NamespaceImage {
        entries: Vec::new(),
        writable_directories: Vec::new(),
        preopens: vec![b"/".to_vec()],
        working_directory: b"/".to_vec(),
        stdin: Vec::new(),
        limits: linked::NamespaceLimits {
            files: 16,
            descriptors: 16,
            filesystem_bytes: 4096,
            output_bytes: 1024,
        },
        epoch_nanos: 0,
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "Fixture keeps request construction and the corresponding evidence bindings together"
)]
fn run(
    case: &Case,
    seed: u64,
    corpus: &Path,
    profile: &str,
    read_step_nanos: u64,
) -> Result<Publication, String> {
    if Path::new(&case.module).is_absolute()
        || case
            .module
            .split('/')
            .any(|part| matches!(part, "" | "." | ".."))
    {
        return Err("unsafe corpus module path".into());
    }
    let module = std::fs::read(corpus.join(&case.module)).map_err(|error| error.to_string())?;
    let target = canonical::digest(&module);
    if case
        .module_sha256
        .as_ref()
        .is_some_and(|expected| expected != &target)
        || case
            .compiler_sha256
            .as_ref()
            .is_some_and(|digest| !canonical::valid_digest(digest))
    {
        return Err("fixture compiler/module identity mismatch".into());
    }
    let go_guest = case.compiler_sha256.is_some();
    let evidence_limit = if go_guest { 1 << 20 } else { 4096 };
    let choice_limit = case.choice_limit.unwrap_or(4096);
    let limits = Limits {
        memory_bytes: case.memory_bytes,
        fuel: case.fuel,
        wall_millis: case.wall_millis,
        output_bytes: 1024,
        evidence_bytes: evidence_limit,
    };
    let arguments = if case.modeled {
        vec!["fixture".to_owned()]
    } else {
        Vec::new()
    };
    let environment = if case.modeled {
        BTreeMap::from([("PWD".into(), "/".into())])
    } else {
        BTreeMap::new()
    };
    let model = if case.modeled {
        Some(linked::ModelInput {
            configuration: serde_json::from_value({
                let mut configuration = model_configuration(seed, choice_limit, read_step_nanos);
                configuration["now"] = u64::from(go_guest).into();
                configuration["limits"]["calls"] = if go_guest { 65536 } else { 100 }.into();
                configuration["limits"]["transcript_bytes"] = evidence_limit.into();
                configuration
            })
            .map_err(|error| error.to_string())?,
            arguments: arguments.clone(),
            environment: vec!["PWD=/".into()],
            replay_trace: None,
        })
    } else {
        None
    };
    let namespace = case.modeled.then(namespace);
    let result = execute::execute(linked::Request {
        module: module.clone(),
        module_sha256: target.clone(),
        limits: linked::Limits {
            memory_bytes: case.memory_bytes,
            fuel: case.fuel,
            output_bytes: 1024,
            evidence_bytes: evidence_limit,
            wall_timeout: Duration::from_millis(case.wall_millis),
        },
        model,
        namespace,
        storage: None,
    })
    .map_err(|outcome| outcome.message)?;
    if result.outcome.termination != case.termination {
        return Err(format!(
            "{} termination: got {}, expected {}: {}",
            case.name, result.outcome.termination, case.termination, result.outcome.message
        ));
    }
    if go_guest && result.outcome.exit_code != Some(0) {
        return Err(format!(
            "{} Go fixture exit {:?}: {}",
            case.name,
            result.outcome.exit_code,
            String::from_utf8_lossy(result.evidence.get("stderr").map_or(&[], Vec::as_slice)),
        ));
    }
    if case.modeled
        && !case.runtime
        && result.evidence.get("stdout").map(Vec::as_slice) != Some(b"linked-output\n")
    {
        return Err("modeled stdout mismatch".into());
    }
    let identity = Identity {
        target,
        toolchain: case
            .compiler_sha256
            .clone()
            .unwrap_or_else(|| canonical::digest(b"gomad.fixture-wat-source/v1")),
        platform: canonical::digest(
            format!("{}/{}", std::env::consts::OS, std::env::consts::ARCH).as_bytes(),
        ),
        implementation: canonical::digest(&canonical::encode(&json!({
            "host": env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
            "helper": linked::IMPLEMENTATION_SHA256,
        }))?),
        configuration: canonical::digest(&canonical::encode(
            &json!({"limits":limits,"choice_limit":choice_limit,"arguments":arguments,"environment":environment,"modeled":case.modeled,"engine":linked::CONFIGURATION,"profile":profile,"read_step_nanos":read_step_nanos}),
        )?),
    };
    let mut streams = BTreeMap::new();
    let mut payloads = BTreeMap::from([("target/module.wasm".into(), module)]);
    for (name, bytes) in result.evidence {
        let records = match name.as_str() {
            "transcript" => result
                .outcome
                .local_model
                .as_deref()
                .and_then(|model| model["calls"].as_u64())
                .unwrap_or(0),
            "choice" | "diagnostic" => result
                .outcome
                .local_model
                .as_deref()
                .and_then(|model| model["records"].as_u64())
                .unwrap_or(0),
            _ => 0,
        };
        let path = if matches!(name.as_str(), "stdout" | "stderr") {
            format!("output/{name}")
        } else {
            format!("evidence/{name}")
        };
        streams.insert(
            path.clone(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records,
            },
        );
        payloads.insert(path, bytes);
    }
    let mut record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        profile: profile.into(),
        seed,
        arguments,
        environment,
        limits,
        virtual_time_nanos: result
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| model["now"].as_u64())
            .unwrap_or(0),
        outcome: Outcome {
            class: if result.outcome.termination == "exit" && result.outcome.exit_code == Some(0) {
                "success"
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
    if case.runtime {
        use crate::evidence::inspection::{ChoiceEvidence, Details, Payload};
        let implementation = case
            .runtime_implementation_sha256
            .clone()
            .unwrap_or_else(|| canonical::digest(b"inspect-runtime-implementation/v1"));
        if !canonical::valid_digest(&implementation) {
            return Err("invalid fixture runtime implementation identity".into());
        }
        let mut implementation_sha256 = [0; 32];
        for (index, byte) in implementation_sha256.iter_mut().enumerate() {
            *byte = u8::from_str_radix(&implementation[index * 2..index * 2 + 2], 16)
                .map_err(|error| error.to_string())?;
        }
        let identity = linked::ChoiceIdentity {
            target_sha256: sha2::Sha256::digest(&payloads["target/module.wasm"]).into(),
            toolchain_build_key: record.identity.toolchain.clone(),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256,
        };
        let stream = record
            .streams
            .get("evidence/choice")
            .ok_or("missing runtime choices")?;
        let projected = linked::inspect_choices(&payloads["evidence/choice"], &identity)?;
        let details = Details {
            schema: "gomad.inspection-evidence/v1".into(),
            outputs: None,
            choices: Some(ChoiceEvidence {
                profile: "gomad-choice-trace/v4".into(),
                implementation_sha256: case
                    .runtime_implementation_sha256
                    .clone()
                    .unwrap_or_else(|| linked::IMPLEMENTATION_SHA256.into()),
                identity,
                trace: Payload {
                    schema: "gomad-choice-trace/v4".into(),
                    file: "evidence/choice".into(),
                    sha256: stream.sha256.clone(),
                    bytes: stream.bytes,
                    limit: choice_limit,
                },
                records: stream.records,
                terminal_state: if record.outcome.termination == "capacity" {
                    "overflow"
                } else {
                    "complete"
                }
                .into(),
                tape_sha256: (record.outcome.termination != "capacity")
                    .then_some(projected.tape_sha256),
            }),
            world: None,
            simulation: None,
            minimization: None,
        };
        let bytes = canonical::encode(&details)?;
        record.streams.insert(
            "evidence/inspection.json".into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        );
        payloads.insert("evidence/inspection.json".into(), bytes);
    }
    payloads.insert("record.json".into(), record.bytes()?);
    Ok(Publication { record, payloads })
}

pub(crate) fn bundle(root: &Path, corpus: &Path) -> Result<(), String> {
    let bytes = std::fs::read(corpus.join("foundation-execution.json"))
        .map_err(|error| error.to_string())?;
    let input: Corpus = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if input.schema != "gomad.foundation-execution-corpus/v1"
        || input.profile != "gomad.wasi-stock-exploratory/v1"
        || input.read_step_nanos != 1
        || input.seeds != [7, 8]
        || input.cases.len() != 6
    {
        return Err("incomplete foundation execution corpus".into());
    }
    let mut records = Vec::new();
    for case in input.cases {
        let barrier = Barrier::new(2);
        let results = crate::slots::run(&input.seeds, 2, |seed| {
            barrier.wait();
            run(&case, seed, corpus, &input.profile, input.read_step_nanos)
        })?;
        for publication in results {
            let repeated = run(
                &case,
                publication.record.seed,
                corpus,
                &input.profile,
                input.read_step_nanos,
            )?;
            if repeated.record.bytes()? != publication.record.bytes()? {
                return Err(format!(
                    "{}.seed{}: repeated execution record differs",
                    case.name, publication.record.seed
                ));
            }
            records.push(publication);
        }
    }
    crate::fixture::publish_bundle(root, "foundation-execution", records)
}

pub(crate) fn watchdog_cancellation(corpus: &Path) -> Result<(), String> {
    let case = Case {
        name: "prompt-exit".into(),
        module: "targets/execution-exit.wat".into(),
        fuel: 100_000,
        memory_bytes: 65_536,
        wall_millis: 60_000,
        modeled: false,
        runtime: false,
        choice_limit: None,
        termination: "exit".into(),
        compiler_sha256: None,
        module_sha256: None,
        runtime_implementation_sha256: None,
    };
    run(&case, 7, corpus, "gomad.wasi-stock-exploratory/v1", 1)?;
    println!("normal exit cancelled and joined its 60-second watchdog");
    Ok(())
}

pub(crate) fn inspection_bundle(root: &Path, corpus: &Path) -> Result<(), String> {
    let mut publications = Vec::new();
    for (name, module, limit, termination) in [
        ("runtime-choice", "targets/inspect-choice.wat", 4096, "exit"),
        (
            "runtime-overflow",
            "targets/inspect-choice-overflow.wat",
            160,
            "capacity",
        ),
    ] {
        let case = Case {
            name: name.into(),
            module: module.into(),
            fuel: 100_000,
            memory_bytes: 65_536,
            wall_millis: 1000,
            modeled: true,
            runtime: true,
            choice_limit: Some(limit),
            termination: termination.into(),
            compiler_sha256: None,
            module_sha256: None,
            runtime_implementation_sha256: None,
        };
        for seed in [7, 8] {
            let first = run(&case, seed, corpus, "gomad.wasi-cooperative/v1", 0)?;
            let second = run(&case, seed, corpus, "gomad.wasi-cooperative/v1", 0)?;
            if first.record.bytes()? != second.record.bytes()? {
                return Err("repeated choice execution differs".into());
            }
            publications.push(first);
        }
    }
    crate::fixture::publish_bundle(root, "inspect-execution", publications)
}

pub(crate) fn replay_bundle(root: &Path, corpus: &Path) -> Result<(), String> {
    use crate::replay::preflight::{Configuration, FILE, Inputs};
    let mut publications = Vec::new();
    for (name, module, modeled, runtime, termination, profile, step) in [
        (
            "observed-exit",
            "targets/execution-exit.wat",
            false,
            false,
            "exit",
            "gomad.wasi-stock-exploratory/v1",
            1,
        ),
        (
            "observed-modeled",
            "targets/execution-modeled.wat",
            true,
            false,
            "exit",
            "gomad.wasi-stock-exploratory/v1",
            1,
        ),
        (
            "exact-choice",
            "targets/inspect-choice.wat",
            true,
            true,
            "exit",
            "gomad.wasi-cooperative/v1",
            0,
        ),
    ] {
        let case = Case {
            name: name.into(),
            module: module.into(),
            fuel: 100_000,
            memory_bytes: 65_536,
            wall_millis: 1000,
            modeled,
            runtime,
            choice_limit: Some(4096),
            termination: termination.into(),
            compiler_sha256: None,
            module_sha256: None,
            runtime_implementation_sha256: None,
        };
        for seed in [7, 8] {
            let mut publication = run(&case, seed, corpus, profile, step)?;
            let inputs = Inputs {
                schema: "gomad.replay-inputs/v1".into(),
                helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
                host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
                toolchain_sha256: publication.record.identity.toolchain.clone(),
                runtime_implementation_sha256: runtime
                    .then(|| env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into()),
                configuration: Configuration {
                    world_transition_bytes: None,
                    engine: linked::CONFIGURATION.into(),
                    profile: profile.into(),
                    limits: publication.record.limits.clone(),
                    arguments: publication.record.arguments.clone(),
                    environment: publication.record.environment.clone(),
                    derive_seed_environment: false,
                    wall_nanos: None,
                    network: None,
                    runtime_sha256: runtime
                        .then(|| canonical::digest(b"inspect-runtime-implementation/v1")),
                    model: modeled.then(|| {
                        let mut value = model_configuration(seed, 4096, step);
                        value
                            .as_object_mut()
                            .expect("fixture model object")
                            .remove("seed");
                        value
                    }),
                    namespace: modeled.then(namespace),
                    storage: None,
                },
            };
            publication.record.identity.configuration =
                canonical::digest(&canonical::encode(&inputs.configuration)?);
            publication.record.execution_identity = publication.record.identity.digest()?;
            let bytes = canonical::encode(&inputs)?;
            publication.record.streams.insert(
                FILE.into(),
                Stream {
                    bytes: bytes.len() as u64,
                    sha256: canonical::digest(&bytes),
                    records: 1,
                },
            );
            publication.payloads.insert(FILE.into(), bytes);
            publication
                .payloads
                .insert("record.json".into(), publication.record.bytes()?);
            publications.push(publication);
        }
    }
    crate::fixture::publish_bundle(root, "replay-execution", publications)
}

pub(crate) fn inspection_input(root: &Path, input: &Path) -> Result<(), String> {
    let bytes = std::fs::read(input).map_err(|error| error.to_string())?;
    let corpus: Corpus = serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if corpus.schema != "gomad.inspect-execution-corpus/v1"
        || corpus.profile != "gomad.wasi-cooperative/v1"
        || corpus.read_step_nanos != 0
        || corpus.seeds != [7, 8]
        || corpus.cases.is_empty()
    {
        return Err("invalid inspect execution corpus".into());
    }
    let directory = input
        .parent()
        .and_then(Path::parent)
        .ok_or("missing corpus directory")?;
    let mut publications = Vec::new();
    for case in corpus.cases {
        for seed in &corpus.seeds {
            let first = run(
                &case,
                *seed,
                directory,
                &corpus.profile,
                corpus.read_step_nanos,
            )?;
            let second = run(
                &case,
                *seed,
                directory,
                &corpus.profile,
                corpus.read_step_nanos,
            )?;
            if first.record.bytes()? != second.record.bytes()? {
                return Err("repeated inspect execution differs".into());
            }
            publications.push(first);
        }
    }
    crate::fixture::publish_bundle(root, "inspect-execution", publications)
}
