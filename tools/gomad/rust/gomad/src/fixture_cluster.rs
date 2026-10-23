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
use std::{collections::BTreeMap, path::Path};

fn descriptor(file: String, bytes: &[u8], limit: u64) -> Payload {
    Payload {
        file,
        bytes: bytes.len() as u64,
        sha256: canonical::digest(bytes),
        limit,
    }
}

fn inspection_lifecycle() -> (Vec<helper::Handle>, Vec<helper::Step>) {
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
    (handles, steps)
}

pub(crate) fn publication(corpus: &Path, seed: u64) -> Result<Publication, String> {
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
        transition_bytes: None,
        limits: helper::WorldLimits {
            max_requests: 100,
            max_events: 100,
            max_queued_events: 100,
            max_transitions: 1000,
            max_payload_bytes: 4096,
            max_string_bytes: 1024,
        },
    };
    let (handles, steps) = inspection_lifecycle();
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
            workload_kind: String::new(),
            handle,
            imports: None,
            storage_crash_selection: None,
            module: descriptor("target/module.wasm".into(), &module, 512 << 20),
            limits: limits.clone(),
            wall_nanos: None,
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
        original_limits: None,
        oracle_plan: None,
        workload: None,
        simulation_original: None,
        world_encoding: String::new(),
        world_profile: None,
        wall_nanos: None,
        nodes: Vec::new(),
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
pub(crate) fn execute_configuration(
    mut configuration: Configuration,
    toolchain: String,
    modules: BTreeMap<String, Vec<u8>>,
) -> Result<Publication, String> {
    configuration.encoded_world_format()?;
    for item in &mut configuration.incarnations {
        if let Some(module) = modules.get(&item.module.file)
            && module.starts_with(b"\0asm")
        {
            let actual = cluster::module_imports(module)?;
            if item
                .imports
                .as_ref()
                .is_some_and(|retained| *retained != actual)
            {
                return Err("cluster import binding mismatch".into());
            }
            item.imports = Some(actual);
        }
    }
    let limits = configuration.limits.clone();
    let seed = configuration.broker.seed;
    let target = configuration
        .incarnations
        .first()
        .ok_or("missing cluster incarnation")?
        .module
        .sha256
        .clone();
    let prepared = prepare_configuration(&configuration, &modules)?;
    crate::fixture::execution_started().map_err(|error| error.message)?;
    let execution = crate::backstop::within(prepared.wall_timeout(), || {
        helper::execute_prepared(prepared).map_err(|error| error.to_string())
    })?;
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
    let assessment = cluster::oracles::capacity_observation(&execution, &configuration);
    if !assessment["capacity_failure"].is_null() {
        return Err(format!(
            "capacity: output capacity; {}",
            assessment["capacity_failure"]["detail"]
        ));
    }
    let oracle_report = cluster::oracles::assess(&execution, &configuration)?;
    let mut observed = cluster::project_encoded(
        execution,
        configuration.encoded_world_format()?,
        &configuration.broker,
    )?;
    cluster::oracles::apply(&mut observed, &oracle_report)?;
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
pub(crate) struct Forcing {
    trace_path: String,
    mode: u32,
    identity: linked::ChoiceIdentity,
}

pub(crate) fn prepare_configuration(
    configuration: &Configuration,
    modules: &BTreeMap<String, Vec<u8>>,
) -> Result<helper::Prepared, String> {
    prepare_configuration_forced(configuration, modules, &BTreeMap::new())
}

fn forced_trace(
    item: &Incarnation,
    model: &mut serde_json::Value,
    forced: Option<&Forcing>,
) -> Result<Option<Vec<u8>>, String> {
    Ok(if let Some(forced) = forced {
        if !matches!(forced.mode, 2 | 3)
            || !Path::new(&forced.trace_path).is_absolute()
            || canonical::encode(&forced.identity)? != canonical::encode(&item.choice_identity)?
        {
            return Err("invalid live cluster forcing binding".into());
        }
        let raw = std::fs::read(&forced.trace_path).map_err(|error| error.to_string())?;
        let trace = linked::inspect_choices(&raw, &forced.identity).map_err(str::to_owned)?;
        model["mode"] = json!(forced.mode);
        model["replay"] = json!({"schema":"gomad.replay-trace/v1", "execution_identity":forced.identity,
                "trace_sha256":canonical::digest(&raw),"tape_sha256":trace.tape_sha256,"decisions":trace.decisions,
                "trace_bytes":raw.len(),"trace_frames":(raw.len() as u64).div_ceil(65536),
                "prefix_control":if forced.mode == 3 {Some(json!({"cutoff":trace.decisions,"override":null}))} else {None}});
        Some(raw)
    } else {
        None
    })
}

pub(crate) fn prepare_configuration_forced(
    configuration: &Configuration,
    modules: &BTreeMap<String, Vec<u8>>,
    forcing: &BTreeMap<String, Forcing>,
) -> Result<helper::Prepared, String> {
    cluster::oracles::validate(configuration)?;
    if configuration.engine != linked::CONFIGURATION
        || configuration.profile != "gomad.wasi-cooperative/v1"
    {
        return Err("live cluster requires explicit pinned cooperative engine".into());
    }
    configuration.encoded_world_format()?;
    if forcing.keys().any(|key| {
        !configuration
            .incarnations
            .iter()
            .any(|item| key == &format!("{}/{}", item.handle.node, item.handle.incarnation))
    }) {
        return Err("unknown live cluster forcing owner".into());
    }
    let seed = configuration.broker.seed;
    let limits = &configuration.limits;
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
        if let Some(imports) = &item.imports
            && cluster::module_imports(module)? != *imports
        {
            return Err("cluster import binding mismatch".into());
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
        let replay_trace = forced_trace(
            item,
            &mut model,
            forcing.get(&format!("{}/{}", item.handle.node, item.handle.incarnation)),
        )?;
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
                    wall_timeout: crate::replay::preflight::wall_timeout(
                        &item.limits,
                        item.wall_nanos,
                    )?,
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
                    replay_trace,
                }),
                namespace: Some(item.namespace.clone()),
                storage: cluster::storage_input(item)?,
            },
        });
    }
    helper::prepare(helper::Request {
        nodes: configuration.nodes.clone(),
        broker: configuration.broker.clone(),
        network: configuration.network.clone(),
        incarnations: executions,
        steps: configuration.steps.clone(),
        boundaries: configuration.boundaries.clone(),
        crash_groups: configuration.crash_groups.clone(),
        wall_timeout: crate::replay::preflight::wall_timeout(limits, configuration.wall_nanos)?,
        evidence_bytes: limits.evidence_bytes,
    })
    .map_err(|error| error.to_string())
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

#[cfg(all(
    test,
    not(any(
        feature = "helper-determinism-fixture",
        feature = "helper-stress-jitter"
    ))
))]
mod world_limit_tests {
    use super::*;
    use crate::evidence::store::{Opened, Store};

    #[test]
    fn retained_cluster_world_byte_bound_survives_actual_public_replay() {
        let corpus =
            std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
        let initial = publication(&corpus, 7).unwrap();
        let original: Inputs = serde_json::from_slice(&initial.payloads[cluster::FILE]).unwrap();
        let modules = initial
            .payloads
            .into_iter()
            .filter(|(name, _)| name.starts_with("target/"))
            .collect::<BTreeMap<_, _>>();
        let root = std::env::temp_dir().join(format!(
            "gomad-world-cap-replay-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        for limit in [1 << 20, 32 << 20] {
            let mut configuration = original.configuration.clone();
            configuration.broker.transition_bytes = Some(limit);
            let publication = execute_configuration(
                configuration,
                original.toolchain_sha256.clone(),
                modules.clone(),
            )
            .unwrap();
            let published = Store {
                root: root.join(limit.to_string()),
                maximum_bytes: 64 << 20,
                target_pool: None,
                fault: None,
            }
            .publish(publication)
            .unwrap();
            let opened = Opened::open(&published.path).unwrap();
            let raw = opened.payload(cluster::FILE, 16 << 20).unwrap();
            let retained: Inputs = serde_json::from_slice(&raw).unwrap();
            assert_eq!(retained.configuration.broker.transition_bytes, Some(limit));
            let world = opened.payload("evidence/world", 64 << 20).unwrap();
            helper::validate_encoded_world(
                &world,
                retained.configuration.encoded_world_format().unwrap(),
                &retained.configuration.broker,
            )
            .unwrap();
            let mut output = Vec::new();
            let status =
                crate::cli::replay::run(&[published.path.to_str().unwrap().into()], &mut output)
                    .unwrap_or_else(|error| panic!("{}", error.message));
            assert_eq!(
                status,
                i32::from(opened.manifest().record.outcome.class != "success")
            );
            assert!(
                String::from_utf8(output)
                    .unwrap()
                    .contains("reproduced=true")
            );
            assert_eq!(opened.payload(cluster::FILE, 16 << 20).unwrap(), raw);
        }
        std::fs::remove_dir_all(root).unwrap();
    }
}

#[cfg(all(
    test,
    not(any(
        feature = "helper-determinism-fixture",
        feature = "helper-stress-jitter"
    ))
))]
#[path = "fixture_cluster_lifecycle_tests.rs"]
mod lifecycle_preservation_tests;
