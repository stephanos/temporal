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
    termination: String,
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
    let limits = Limits {
        memory_bytes: case.memory_bytes,
        fuel: case.fuel,
        wall_millis: case.wall_millis,
        output_bytes: 1024,
        evidence_bytes: 4096,
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
            configuration: serde_json::from_value(json!({"entropy_key":vec![0_u8;32],"working_directory_sha256":canonical::digest(b"/"),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"seed":seed,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":read_step_nanos,"now":0,"limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}})).map_err(|error| error.to_string())?,
            arguments: arguments.clone(), environment: vec!["PWD=/".into()], replay_trace: None,
        })
    } else {
        None
    };
    let namespace = case.modeled.then(|| linked::NamespaceImage {
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
    });
    let result = execute::execute(linked::Request {
        module: module.clone(),
        module_sha256: target.clone(),
        limits: linked::Limits {
            memory_bytes: case.memory_bytes,
            fuel: case.fuel,
            output_bytes: 1024,
            evidence_bytes: 4096,
            wall_timeout: Duration::from_millis(case.wall_millis),
        },
        model,
        namespace,
    })
    .map_err(|outcome| outcome.message)?;
    if result.outcome.termination != case.termination {
        return Err(format!(
            "{} termination: got {}, expected {}: {}",
            case.name, result.outcome.termination, case.termination, result.outcome.message
        ));
    }
    if case.modeled && result.evidence.get("stdout").map(Vec::as_slice) != Some(b"linked-output\n")
    {
        return Err("modeled stdout mismatch".into());
    }
    let identity = Identity {
        target,
        toolchain: canonical::digest(b"gomad.fixture-wat-source/v1"),
        platform: canonical::digest(
            format!("{}/{}", std::env::consts::OS, std::env::consts::ARCH).as_bytes(),
        ),
        implementation: canonical::digest(&canonical::encode(&json!({
            "host": env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
            "helper": linked::IMPLEMENTATION_SHA256,
        }))?),
        configuration: canonical::digest(&canonical::encode(
            &json!({"limits":limits,"arguments":arguments,"environment":environment,"modeled":case.modeled,"engine":linked::CONFIGURATION,"profile":profile,"read_step_nanos":read_step_nanos}),
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
    let record = Record {
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
        termination: "exit".into(),
    };
    run(&case, 7, corpus, "gomad.wasi-stock-exploratory/v1", 1)?;
    println!("normal exit cancelled and joined its 60-second watchdog");
    Ok(())
}
