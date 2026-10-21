use crate::{
    evidence::{
        canonical,
        identity::Identity,
        record::{Limits, Outcome, Record, Stream},
        store::Publication,
    },
    replay::{
        cluster::{self, Configuration, Incarnation, Inputs, Payload, Retained},
        preflight,
    },
};
use gomad_wasmhost::{linked, linked_cluster as helper};
use serde::Deserialize;
use serde_json::json;
use sha2::Digest as _;
use std::{collections::BTreeMap, path::Path, time::Duration};

fn descriptor(file: String, bytes: &[u8], limit: u64) -> Payload {
    Payload {
        file,
        bytes: bytes.len() as u64,
        sha256: canonical::digest(bytes),
        limit,
    }
}

fn publication(corpus: &Path, seed: u64) -> Result<Publication, String> {
    let module = std::fs::read(corpus.join("targets/inspect-choice.wat"))
        .map_err(|error| error.to_string())?;
    let toolchain = canonical::digest(b"gomad.fixture-wat-source/v1");
    let runtime = canonical::digest(b"inspect-runtime-implementation/v1");
    let limits = Limits {
        memory_bytes: 65_536,
        fuel: 100_000,
        wall_millis: 1000,
        output_bytes: 1024,
        evidence_bytes: 1 << 20,
    };
    let broker = helper::BrokerConfig {
        seed,
        pause_on_settle: false,
        limits: helper::WorldLimits {
            max_requests: 100,
            max_events: 100,
            max_queued_events: 100,
            max_transitions: 1000,
            max_payload_bytes: 4096,
            max_string_bytes: 1024,
        },
    };
    let handles: Vec<_> = [("a", 1), ("b", 1), ("a", 2)]
        .into_iter()
        .map(|(node, incarnation)| helper::Handle {
            node: node.into(),
            incarnation,
        })
        .collect();
    let steps: Vec<_> = handles
        .iter()
        .flat_map(|handle| {
            [
                helper::Step {
                    handle: handle.clone(),
                    action: if handle.incarnation == 1 {
                        helper::Action::Start
                    } else {
                        helper::Action::Restart
                    },
                },
                helper::Step {
                    handle: handle.clone(),
                    action: helper::Action::Wait,
                },
            ]
        })
        .collect();
    let mut incarnations = Vec::new();
    for handle in handles {
        let mut model = crate::fixture_execute::model_configuration(seed, 4096, 0);
        model["now"] = json!(1);
        let mut image = crate::fixture_execute::namespace();
        image.epoch_nanos = 0;
        model.as_object_mut().ok_or("fixture model")?.remove("seed");
        let identity = serde_json::from_value(json!({"TargetSHA256": sha2::Sha256::digest(&module).to_vec(),
            "ToolchainBuildKey":toolchain,"GOOS":"wasip1","GOARCH":"wasm",
            "ImplementationSHA256":sha2::Sha256::digest(b"inspect-runtime-implementation/v1").to_vec() }))
            .map_err(|error| error.to_string())?;
        incarnations.push(Incarnation {
            handle,
            module: descriptor("target/module.wasm".into(), &module, 512 << 20),
            limits: limits.clone(),
            arguments: vec!["fixture".into()],
            environment: BTreeMap::from([("PWD".into(), "/".into())]),
            runtime_sha256: runtime.clone(),
            model,
            namespace: image,
            storage: None,
            network: None,
            choice_identity: identity,
        });
    }
    let configuration = Configuration {
        engine: linked::CONFIGURATION.into(),
        profile: "gomad.wasi-cooperative/v1".into(),
        limits,
        broker,
        network: None,
        incarnations,
        steps,
        boundaries: Vec::new(),
        crash_groups: Vec::new(),
    };
    execute_configuration(
        configuration,
        toolchain,
        BTreeMap::from([("target/module.wasm".into(), module)]),
    )
}

#[expect(
    clippy::too_many_lines,
    reason = "Fixture publication binds the exact recording inputs to every incarnation stream and the resulting record"
)]
fn execute_configuration(
    configuration: Configuration,
    toolchain: String,
    modules: BTreeMap<String, Vec<u8>>,
) -> Result<Publication, String> {
    let limits = configuration.limits.clone();
    let seed = configuration.broker.seed;
    let target = configuration
        .incarnations
        .first()
        .ok_or("missing cluster incarnation")?
        .module
        .sha256
        .clone();
    let mut executions = Vec::new();
    for item in &configuration.incarnations {
        let module = modules
            .get(&item.module.file)
            .ok_or("missing fixture cluster module")?;
        if module.len() as u64 != item.module.bytes
            || canonical::digest(module) != item.module.sha256
            || item.module.bytes > item.module.limit
            || item.module.limit > 512 << 20
        {
            return Err("fixture cluster module identity/bound mismatch".into());
        }
        let mut model = item.model.clone();
        if model
            .as_object_mut()
            .ok_or("invalid fixture cluster model")?
            .insert("seed".into(), seed.into())
            .is_some()
        {
            return Err("fixture cluster seed must derive from broker".into());
        }
        executions.push(helper::Incarnation {
            handle: item.handle.clone(),
            network: item.network.clone(),
            request: linked::Request {
                module: module.clone(),
                module_sha256: item.module.sha256.clone(),
                limits: linked::Limits {
                    memory_bytes: item.limits.memory_bytes,
                    fuel: item.limits.fuel,
                    output_bytes: item.limits.output_bytes,
                    evidence_bytes: item.limits.evidence_bytes,
                    wall_timeout: Duration::from_millis(item.limits.wall_millis),
                },
                model: Some(linked::ModelInput {
                    configuration: serde_json::from_value(model)
                        .map_err(|error| error.to_string())?,
                    arguments: item.arguments.clone(),
                    environment: item
                        .environment
                        .iter()
                        .map(|(key, value)| format!("{key}={value}"))
                        .collect(),
                    replay_trace: None,
                }),
                namespace: Some(item.namespace.clone()),
                storage: item.storage.clone(),
            },
        });
    }
    let execution = helper::execute_prepared(
        helper::prepare(helper::Request {
            broker: configuration.broker.clone(),
            network: configuration.network.clone(),
            incarnations: executions,
            steps: configuration.steps.clone(),
            boundaries: configuration.boundaries.clone(),
            crash_groups: configuration.crash_groups.clone(),
            wall_timeout: Duration::from_millis(limits.wall_millis),
            evidence_bytes: limits.evidence_bytes,
        })
        .map_err(|error| error.to_string())?,
    )
    .map_err(|error| error.to_string())?;
    let mut retained = Vec::new();
    for item in &execution.incarnations {
        let mut evidence: BTreeMap<_, _> = item
            .execution
            .evidence
            .iter()
            .map(|(name, bytes)| {
                (
                    name.clone(),
                    descriptor(
                        format!(
                            "evidence/cluster/{}/{}/{name}",
                            item.handle.node, item.handle.incarnation
                        ),
                        bytes,
                        limits.evidence_bytes,
                    ),
                )
            })
            .collect();
        evidence.entry("choice".into()).or_insert_with(|| {
            descriptor(
                format!(
                    "evidence/cluster/{}/{}/choice",
                    item.handle.node, item.handle.incarnation
                ),
                &[],
                limits.evidence_bytes,
            )
        });
        retained.push(Retained {
            handle: item.handle.clone(),
            interrupted: item.interrupted,
            evidence,
        });
    }
    let mut observed = cluster::project(execution)?;
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
    let mut payloads = modules;
    for (name, bytes) in observed.evidence {
        let path = format!("evidence/{name}");
        streams.insert(
            path.clone(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 0,
            },
        );
        payloads.insert(path, bytes);
    }
    let inputs = Inputs {
        schema: "gomad.cluster-replay-inputs/v1".into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into(),
        toolchain_sha256: toolchain.clone(),
        configuration,
        retained,
    };
    let identity = Identity {
        target,
        toolchain,
        platform: preflight::platform(),
        implementation: preflight::implementation()?,
        configuration: canonical::digest(&canonical::encode(&inputs.configuration)?),
    };
    let bytes = canonical::encode(&inputs)?;
    streams.insert(
        cluster::FILE.into(),
        Stream {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            records: 1,
        },
    );
    payloads.insert(cluster::FILE.into(), bytes);
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        seed,
        profile: "gomad.wasi-cooperative/v1".into(),
        limits,
        arguments: Vec::new(),
        environment: BTreeMap::new(),
        outcome: Outcome {
            class: if observed.outcome.termination == "exit"
                && observed.outcome.exit_code == Some(0)
            {
                "success"
            } else {
                "target_failure"
            }
            .into(),
            termination: observed.outcome.termination.into(),
            exit_code: observed.outcome.exit_code,
            reason: observed.outcome.message.clone(),
        },
        virtual_time_nanos: observed
            .outcome
            .local_model
            .as_deref()
            .and_then(|model| model["now"].as_u64())
            .unwrap_or(0),
        streams,
    };
    payloads.insert("record.json".into(), record.bytes()?);
    Ok(Publication { record, payloads })
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    schema: String,
    toolchain_sha256: String,
    configuration: Configuration,
    modules: BTreeMap<String, String>,
}

pub(crate) fn input(root: &Path, path: &Path) -> Result<(), String> {
    let raw = std::fs::read(path).map_err(|error| error.to_string())?;
    let input: Input = serde_json::from_slice(&raw).map_err(|error| error.to_string())?;
    if input.schema != "gomad.cluster-fixture-input/v1"
        || !canonical::valid_digest(&input.toolchain_sha256)
    {
        return Err("invalid cluster fixture input".into());
    }
    let modules = input
        .modules
        .into_iter()
        .map(|(name, path)| {
            if !Path::new(&path).is_absolute() {
                return Err("fixture module must be absolute".to_owned());
            }
            Ok((
                name,
                std::fs::read(path).map_err(|error| error.to_string())?,
            ))
        })
        .collect::<Result<_, String>>()?;
    let publication = execute_configuration(input.configuration, input.toolchain_sha256, modules)?;
    crate::fixture::publish_bundle_with_budget(
        root,
        "replay-cluster-input",
        vec![publication],
        1 << 30,
    )
}

pub(crate) fn bundle(root: &Path, corpus: &Path) -> Result<(), String> {
    let publications = [7, 8]
        .into_iter()
        .map(|seed| publication(corpus, seed))
        .collect::<Result<Vec<_>, _>>()?;
    crate::fixture::publish_bundle_with_budget(root, "replay-cluster", publications, 1 << 30)
}
