use super::*;
use crate::campaign::{prepare, scenario};
use serde::Serialize;
use std::{
    fs,
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Serialize)]
struct SourceSeal {
    path: &'static str,
    sha256: &'static str,
}

const SOFT: SourceSeal = SourceSeal {
    path: "tests/gomadfunctional/testdata/softassert/softassert_test.go",
    sha256: "3bc1fe2c5b6cc5f59cce0dec742bec77efa0a931e5976c7dec741b0b3a1274a5",
};
const SERVICE: SourceSeal = SourceSeal {
    path: "tools/gomad/internal/backends/wasm/cluster/testdata/service/main.go",
    sha256: "35278d6687f89f68716a020ecadfe6c6aaa73bed5c0741963cd50f9d87fb13d6",
};
const CAPACITY: SourceSeal = SourceSeal {
    path: "tools/gomad/internal/backends/wasm/cluster/testdata/service/output_capacity_test.go",
    sha256: "4c144724dc5c299d679f761cc24f00a43bd03f67d652a74e0520a024558afbaf",
};

struct Producer {
    root: PathBuf,
    prepared: prepare::Prepared,
    sources: Vec<SourceSeal>,
}

fn checkout() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .unwrap()
}

fn verify_sources(sources: &[SourceSeal]) {
    for source in sources {
        assert_eq!(
            canonical::digest(&fs::read(checkout().join(source.path)).unwrap()),
            source.sha256,
            "original 1d695 fixture changed: {}",
            source.path
        );
    }
}

fn original_config() -> Value {
    let mut entropy = [0u8; 32];
    entropy[..3].copy_from_slice(&[1, 2, 3]);
    json!({"Profile":"gomad.wasi-cooperative/v1","Args":["service","server"],
        "Environment":["PWD=/workspace","TZ=UTC"],"WorkingDirectory":"/workspace",
        "WritableDirectories":["/workspace","/tmp"],"EntropyKey":entropy,
        "Clock":{"EpochNanos":946_684_800_000_000_000_u64},
        "Limits":{"OutputBytes":1<<20,"TranscriptBytes":32<<20,"Calls":100_000,"Descriptors":128,
            "Files":1024,"FilesystemBytes":8<<20,"PendingEvents":4096},
        "network_v2":{"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":128,
            "bytes":8<<20,"connection_bytes":4<<20,"stall_nanos":10_000_000_000_u64,"transitions":10000}}})
}

fn producer(name: &str, soft: bool, test_binary: bool) -> Producer {
    #[expect(
        clippy::assertions_on_constants,
        reason = "The opt-in guest producer enforces its compiled release role at runtime while ordinary debug test builds remain valid."
    )]
    assert!(
        !cfg!(debug_assertions),
        "actual Go receiving roots require a release fixture build"
    );
    let root = PathBuf::from(
        std::env::var_os("GOMAD_TEST_BROAD_GUEST_ROOT").expect("explicit retained root"),
    )
    .join(name);
    assert!(root.is_absolute());
    fs::create_dir(&root).unwrap();
    let compiler = PathBuf::from(std::env::var_os("GOMAD_STOCK_GO").expect("stock Go 1.27.1"));
    assert!(compiler.is_absolute());
    let sources = if soft {
        vec![SOFT]
    } else if test_binary {
        vec![SERVICE, CAPACITY]
    } else {
        vec![SERVICE]
    };
    verify_sources(&sources);
    let config = root.join("original-config.json");
    fs::write(&config, canonical::encode(&original_config()).unwrap()).unwrap();
    let installation = root.join("installation.json");
    let arguments = vec![
        "--compiler".into(),
        compiler.to_string_lossy().into_owned(),
        "--runtime-root".into(),
        checkout()
            .join("tools/gomad/internal/model/runtimesource")
            .to_string_lossy()
            .into_owned(),
        "--profile=gomad.wasi-cooperative/v1".into(),
        "--memory-bytes=268435456".into(),
        "--fuel=4000000000".into(),
        "--config".into(),
        config.to_string_lossy().into_owned(),
        "--cache".into(),
        root.join("preparation").to_string_lossy().into_owned(),
        "--output".into(),
        installation.to_string_lossy().into_owned(),
    ];
    crate::cli::install::run(&arguments, &mut Vec::new())
        .unwrap_or_else(|error| panic!("{}", error.message));
    let package = if soft {
        "./tests/gomadfunctional/testdata/softassert"
    } else {
        "./tools/gomad/internal/backends/wasm/cluster/testdata/service"
    };
    let mut options = crate::cli::explore::parse(
        &[
            "--choices".into(),
            "--diagnostics".into(),
            if test_binary {
                "go-test".into()
            } else {
                "go-run".into()
            },
            package.into(),
        ],
        &mut false,
    )
    .unwrap();
    options.installation = Some(installation);
    options.working_dir = if soft {
        checkout()
    } else {
        checkout().join("tools/gomad")
    };
    if !soft {
        options.source = "./internal/backends/wasm/cluster/testdata/service".into();
    }
    options.build_tags = if soft {
        vec!["test_dep".into()]
    } else {
        vec![]
    };
    options.output_limit = 1 << 20;
    options.execution_timeout = Duration::from_secs(if soft { 300 } else { 60 });
    let prepared = prepare::prepare(&options).unwrap();
    verify_sources(&sources);
    if !soft {
        assert!(
            prepared.module.len() <= 8 << 20,
            "original small guest module admission"
        );
    }
    assert_eq!(prepared.configuration.limits.memory_bytes, 256 << 20);
    assert_eq!(prepared.configuration.limits.fuel, 4_000_000_000);
    assert_eq!(
        prepared.configuration.model.as_ref().unwrap()["limit"],
        8 << 20
    );
    retain_preparation(&root, &prepared);
    Producer {
        root,
        prepared,
        sources,
    }
}

fn retain_preparation(root: &Path, prepared: &prepare::Prepared) {
    fs::write(root.join("prepared-provenance.json"), &prepared.provenance).unwrap();
    fs::write(root.join("prepared-module.wasm"), &prepared.module).unwrap();
    fs::rename(
        root.join("prepared-module.wasm"),
        root.join("prepared-module.wasm.unavailable"),
    )
    .unwrap();
    fs::rename(
        root.join("installation.json"),
        root.join("installation.json.unavailable"),
    )
    .unwrap();
    for (name, raw) in &prepared.auxiliary {
        let path = root.join("preparation-proof").join(name);
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        fs::write(path, raw).unwrap();
    }
}

fn original_limits() -> Value {
    json!({"nodes":64,"directional_links":4096,"volumes":64,"scenario_actions":4096,
        "scenario_decisions":4096,"fault_actions":4096,"history_operations":65536,"observations":65536,
        "oracle_results":4096,"scenario_evidence_bytes":64<<20,"boot_config_bytes":1<<20,
        "volume_capacity_bytes":1<<30,"observation_bytes":64<<20,"network_listeners":4096,
        "network_connections":65536,"network_bytes":1<<30,"network_connection_bytes":4<<20,
        "network_stall_nanos":10_000_000_000_u64,"network_transitions":1<<20,"volume_operations":1<<20,
        "volume_transitions":1<<20,"crash_states":65536,"crash_depth":256,"crash_bytes":128<<20,
        "crash_wall_nanos":60_000_000_000_u64})
}

fn scenario(nodes: Value, boots: Value, actions: Value, links: Value) -> Value {
    let mut spec = json!({"schema":"gomad.simulation-spec/v8", "backend":"wasm",
        "fidelity":"simulation_model", "seed":7, "limits":original_limits()});
    let spec_fields = spec.as_object_mut().unwrap();
    spec_fields.insert("nodes".into(), nodes);
    spec_fields.insert("links".into(), links);
    Value::Object(serde_json::Map::from_iter([
        ("schema".into(), json!("gomad-wasm.scenario/v1")),
        ("spec".into(), spec),
        ("boots".into(), boots),
        ("actions".into(), actions),
        ("incarnation_boots".into(), json!([])),
        ("network_faults".into(), json!([])),
    ]))
}

fn bind(producer: &Producer, scenario: &Value, seed: u64) -> Configuration {
    let normalized = scenario::normalize(&canonical::encode(scenario).unwrap()).unwrap();
    let mut configuration = scenario::bind(
        &normalized,
        &producer.prepared.configuration,
        &producer.prepared.module,
        producer.prepared.choice_identity.as_ref().unwrap(),
        producer.prepared.network.as_ref(),
        seed,
    )
    .unwrap()
    .configuration;
    for incarnation in &mut configuration.incarnations {
        incarnation.workload_kind = if producer
            .sources
            .iter()
            .any(|source| source.path == SOFT.path || source.path == CAPACITY.path)
        {
            "go-test"
        } else {
            "go-run"
        }
        .into();
    }
    configuration
}

fn modules(producer: &Producer) -> BTreeMap<String, Vec<u8>> {
    BTreeMap::from([(
        "target/module.wasm".into(),
        producer.prepared.module.clone(),
    )])
}

fn policy(configuration: &mut Configuration, report_only: bool, maximum: u64) {
    if report_only {
        configuration.oracle_plan = Some(serde_json::from_value(json!({"schema":"gomad.wasm.soft-assertion-observation/v1",
            "maximum_bytes":maximum,"soft_assertions":configuration.incarnations.iter().map(|item|json!({"handle":item.handle,"report_only":true})).collect::<Vec<_>>()})).unwrap());
    }
}

fn observation(
    producer: &Producer,
    root: &str,
    case: &str,
    configuration: &Configuration,
    value: Value,
) {
    let mut row = json!({"schema":"gomad.retained-reader-observation/v1","original_root":root,"case":case,
        "inputs":{"original_commit":"1d695061b6382009d0659238c652a5a765b1d5f5","sources":producer.sources,
            "module_sha256":canonical::digest(&producer.prepared.module),"toolchain_sha256":producer.prepared.toolchain,
            "prepared_provenance_sha256":canonical::digest(&producer.prepared.provenance),"configuration":configuration}});
    row.as_object_mut()
        .unwrap()
        .insert("observed".into(), value);
    fs::write(
        producer
            .root
            .join(format!("{}.json", case.replace('/', "-"))),
        canonical::encode(&row).unwrap(),
    )
    .unwrap();
    println!("GOMAD_RETAINED_OBSERVATION {row}");
}

fn publish_replay(
    producer: &Producer,
    case: &str,
    configuration: &Configuration,
    replay_requested: bool,
) -> (Value, Value) {
    let publication = crate::fixture_cluster::execute_configuration(
        configuration.clone(),
        producer.prepared.toolchain.clone(),
        modules(producer),
    )
    .unwrap();
    let store = crate::evidence::store::Store {
        root: producer.root.join(case),
        maximum_bytes: 512 << 20,
        target_pool: None,
        fault: None,
    };
    let published = store.publish(publication).unwrap();
    let opened = Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    let inputs = super::inputs(&opened).unwrap();
    oracles::authenticate_retained(&opened, &inputs).unwrap();
    let report = serde_json::from_slice(
        &opened
            .payload("evidence/cluster_oracles", 64 << 20)
            .unwrap(),
    )
    .unwrap();
    let before = opened.manifest().record.bytes().unwrap();
    assert!(!producer.root.join("prepared-module.wasm").exists());
    assert!(!producer.root.join("installation.json").exists());
    let mut output = Vec::new();
    let status = replay_requested.then(|| {
        let status = crate::cli::replay::run(
            &[published.path.to_string_lossy().into_owned()],
            &mut output,
        )
        .unwrap_or_else(|error| panic!("{}", error.message));
        assert_eq!(
            status,
            i32::from(opened.manifest().record.outcome.class != "success")
        );
        status
    });
    let output = String::from_utf8(output).unwrap();
    if replay_requested {
        assert!(
            output.contains("reproduced=true") && output.contains("choice-replay=exact"),
            "{output}"
        );
    }
    assert_eq!(
        Opened::open(&published.path)
            .unwrap()
            .manifest()
            .record
            .bytes()
            .unwrap(),
        before
    );
    (
        report,
        json!({"artifact":published.path,"record_sha256":canonical::digest(&before),"public_replay_exit":status,"public_replay_output":output,"retained_inputs":inputs,"manifest":opened.manifest(),"original_module_absent":true,"installation_absent":true}),
    )
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_soft_assertion_cluster_original_seeds_policies_and_retained_replay() {
    let producer = producer("softassert", true, true);
    let mut signature = None;
    for (report_only, seed) in [(false, 7), (false, 19), (true, 7)] {
        let scenario = scenario(
            json!([{"id":"a","boot":"fixture","address":"10.0.0.1"},{"id":"b","boot":"fixture","address":"10.0.0.2"}]),
            json!([{"id":"fixture","args":["-test.v","-assertion-mode=late"]}]),
            json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"wait","handle":{"node":"a","incarnation":1}},
                {"action":"start","handle":{"node":"b","incarnation":1}},{"action":"wait","handle":{"node":"b","incarnation":1}}]),
            json!([]),
        );
        let mut configuration = bind(&producer, &scenario, seed);
        for item in &mut configuration.incarnations {
            item.arguments[0] = "soft-assertion".into();
        }
        policy(&mut configuration, report_only, 4096);
        let prepared =
            crate::fixture_cluster::prepare_configuration(&configuration, &modules(&producer))
                .unwrap();
        let execution = crate::backstop::within(prepared.wall_timeout(), || {
            helper::execute_prepared(prepared).map_err(|error| error.to_string())
        })
        .unwrap();
        assert_eq!(execution.incarnations.len(), 2);
        for item in &execution.incarnations {
            assert_eq!(item.execution.outcome.exit_code, Some(0));
            assert!(
                item.execution.evidence["stdout"]
                    .windows(4)
                    .any(|window| window == b"PASS")
            );
        }
        let report =
            serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
        assert_eq!(
            report["outcome"],
            if report_only {
                "completed"
            } else {
                "oracle_failed"
            }
        );
        assert_eq!(report["oracles"].as_array().unwrap().len(), 2);
        for oracle in report["oracles"].as_array().unwrap() {
            assert_eq!(oracle["soft_assertion"], true);
            assert_eq!(oracle["report_only"], report_only);
            assert_eq!(oracle["passed"], report_only);
            assert_eq!(
                scenario::canonical_base64(oracle["evidence"][1]["value"].as_str().unwrap())
                    .unwrap(),
                br#"["invalid fixture state"]"#
            );
        }
        if !report_only {
            assert_eq!(report["reason"], "soft_assertion");
            if let Some(signature) = &signature {
                assert_eq!(&report["failure_identity"], signature);
            }
            signature = Some(report["failure_identity"].clone());
        }
        let case = format!("soft/seed={seed}/report-only={report_only}");
        let (retained, replay) = publish_replay(&producer, &case, &configuration, seed == 7);
        assert_eq!(retained, report);
        observation(
            &producer,
            "TestSoftAssertionActualClusterGuestAndRetainedReplay",
            &case,
            &configuration,
            json!({"report":report,"replay":replay}),
        );
    }
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_output_capacity_guest_preserves_original_three_policies() {
    let producer = producer("capacity", false, true);
    for (report_only, maximum) in [(false, 4096), (true, 4096), (true, 1)] {
        let scenario = scenario(
            json!([{"id":"a","boot":"fixture","address":"10.0.0.1"}]),
            json!([{"id":"fixture","args":["-test.run=TestOutputCapacity"]}]),
            json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"wait","handle":{"node":"a","incarnation":1}}]),
            json!([]),
        );
        let mut configuration = bind(&producer, &scenario, 7);
        configuration.limits.output_bytes = 4;
        for item in &mut configuration.incarnations {
            item.arguments[0] = "capacity".into();
            item.limits.output_bytes = 4;
            item.namespace.limits.output_bytes = 4;
        }
        policy(&mut configuration, report_only, maximum);
        let prepared =
            crate::fixture_cluster::prepare_configuration(&configuration, &modules(&producer))
                .unwrap();
        let execution = crate::backstop::within(prepared.wall_timeout(), || {
            helper::execute_prepared(prepared).map_err(|error| error.to_string())
        })
        .unwrap();
        assert_eq!(execution.incarnations.len(), 1);
        let observed = &execution.incarnations[0].execution;
        assert_eq!(observed.outcome.termination, "capacity");
        assert_eq!(observed.outcome.message, "output capacity");
        assert_eq!(observed.evidence["stdout"], b"PASS");
        assert_eq!(observed.outcome.output_bytes, 4);
        assert!(observed.evidence.get("choice").is_none_or(Vec::is_empty));
        let assessment = oracles::capacity_observation(&execution, &configuration);
        let failure = &assessment["capacity_failure"];
        assert_eq!(failure["termination"], "capacity");
        assert_eq!(failure["message"], "output capacity");
        assert_eq!(failure["soft_assertions"]["inconclusive"], true);
        assert_eq!(failure["soft_assertions"]["report_only"], report_only);
        assert!(
            failure["detail"]
                .as_str()
                .unwrap()
                .contains(if maximum == 1 {
                    "oracle_evidence_bytes"
                } else {
                    "choice trace is not complete"
                })
        );
        if maximum == 1 {
            assert_ne!(assessment["oracle_error"].as_str().unwrap(), "");
            assert_eq!(assessment["oracles"], json!([]));
        } else {
            assert_eq!(assessment["oracles"].as_array().unwrap().len(), 1);
            assert_eq!(assessment["oracles"][0]["passed"], false);
            assert_eq!(assessment["oracles"][0]["inconclusive"], true);
        }
        let error = crate::fixture_cluster::execute_configuration(
            configuration.clone(),
            producer.prepared.toolchain.clone(),
            modules(&producer),
        )
        .err()
        .expect("capacity cannot publish retained replay inputs");
        assert!(error.starts_with("capacity: output capacity;"));
        observation(
            &producer,
            "TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure",
            &format!("capacity/report-only={report_only}/maximum={maximum}"),
            &configuration,
            json!({"outcome":observed.outcome,"stdout_base64":crate::campaign::exploration::base64(&observed.evidence["stdout"]),"choice_bytes":observed.evidence.get("choice").map_or(0,Vec::len),"assessment":assessment,"publication_error":error}),
        );
    }
}

fn observe_execution(
    producer: &Producer,
    configuration: &Configuration,
) -> (helper::Execution, Value) {
    let prepared =
        crate::fixture_cluster::prepare_configuration(configuration, &modules(producer)).unwrap();
    let activity = prepared.activity();
    let execution = crate::backstop::within(prepared.wall_timeout(), || {
        helper::execute_prepared(prepared).map_err(|error| error.to_string())
    })
    .unwrap();
    let activity = activity.snapshot();
    assert_eq!(
        activity.launched_stores,
        configuration.incarnations.len() as u64
    );
    assert_eq!(activity.remote_callback_attempts, 0);
    let outcomes=execution.incarnations.iter().map(|item|json!({"handle":item.handle,"interrupted":item.interrupted,
        "outcome":item.execution.outcome,"stdout_base64":crate::campaign::exploration::base64(&item.execution.evidence["stdout"]),
        "evidence":item.execution.evidence.iter().map(|(name,bytes)|(name,json!({"bytes":bytes.len(),"sha256":canonical::digest(bytes)}))).collect::<BTreeMap<_,_>>()})).collect::<Vec<_>>();
    let executed_steps = serde_json::to_value(&execution.executed_steps).unwrap();
    (
        execution,
        json!({"activity":activity,"incarnations":outcomes,"executed_steps":executed_steps}),
    )
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_independent_go_guests_preserve_plain_and_compressed_lifecycle_replay() {
    let producer = producer("lifecycle", false, false);
    for compressed in [false, true] {
        let scenario = scenario(
            json!([{"id":"a","boot":"server","address":"10.0.0.1"},{"id":"b","boot":"client","address":"10.0.0.2"}]),
            json!([{"id":"server","args":["server"]},{"id":"client","args":["client","10.0.0.1"]}]),
            json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"start","handle":{"node":"b","incarnation":1}},
                {"action":"wait","handle":{"node":"b","incarnation":1}},{"action":"wait","handle":{"node":"a","incarnation":1}},
                {"action":"restart","handle":{"node":"a","incarnation":2}},{"action":"restart","handle":{"node":"b","incarnation":2}},
                {"action":"wait","handle":{"node":"b","incarnation":2}},{"action":"wait","handle":{"node":"a","incarnation":2}}]),
            json!([{"from":"a","to":"b","enabled":true},{"from":"b","to":"a","enabled":true}]),
        );
        let mut configuration = bind(&producer, &scenario, 7);
        for item in &mut configuration.incarnations {
            item.arguments[0] = "service".into();
        }
        configuration.world_profile = Some(
            if compressed {
                "gomad/world/compressed-snapshot/v2"
            } else {
                ""
            }
            .into(),
        );
        if compressed {
            configuration.world_encoding = "gomad/world/compressed-snapshot/v2".into();
            configuration.broker.limits = helper::WorldLimits {
                max_requests: 2 << 20,
                max_events: 2 << 20,
                max_queued_events: 250_000,
                max_transitions: 8 << 20,
                max_payload_bytes: 64 << 20,
                max_string_bytes: u32::MAX,
            };
        }
        let (execution, proof) = observe_execution(&producer, &configuration);
        assert_eq!(execution.executed_steps.len(), 8);
        assert_eq!(
            canonical::encode(&execution.executed_steps).unwrap(),
            canonical::encode(&configuration.steps).unwrap()
        );
        assert_eq!(execution.incarnations.len(), 4);
        for item in &execution.incarnations {
            assert_eq!(item.execution.outcome.exit_code, Some(0));
            let stdout = std::str::from_utf8(&item.execution.evidence["stdout"]).unwrap();
            assert!(
                stdout.contains("fresh heap 1") && stdout.contains("ordered bytes ordered-byte"),
                "{stdout}"
            );
        }
        let case = format!("lifecycle/compressed={compressed}");
        let (report, replay) = publish_replay(&producer, &case, &configuration, true);
        assert_eq!(report["outcome"], "completed");
        assert_eq!(
            replay["retained_inputs"]["configuration"]["world_profile"],
            configuration.world_profile.as_ref().unwrap().as_str()
        );
        observation(
            &producer,
            if compressed {
                "TestProviderCompressedIndependentGuestsLifecycleReplay"
            } else {
                "TestProviderIndependentGuestsLifecycleReplay"
            },
            &case,
            &configuration,
            json!({"producer":proof,"report":report,"replay":replay}),
        );
    }
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_go_history_oracle_canary_preserves_detached_retained_replay() {
    let producer = producer("canary", false, false);
    for canary in [false, true] {
        let scenario = scenario(
            json!([{"id":"a","boot":"a","address":"10.0.0.1"},{"id":"b","boot":"b","address":"10.0.0.2"}]),
            json!([{"id":"a","args":["observation"]},{"id":"b","args":if canary {vec!["observation","canary"]} else {vec!["observation"]}}]),
            json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"start","handle":{"node":"b","incarnation":1}},
                {"action":"wait","handle":{"node":"a","incarnation":1}},{"action":"wait","handle":{"node":"b","incarnation":1}}]),
            json!([]),
        );
        let mut configuration = bind(&producer, &scenario, 7);
        for item in &mut configuration.incarnations {
            item.arguments[0] = "service".into();
        }
        configuration.oracle_plan=Some(serde_json::from_value(json!({"schema":"gomad.wasm.history-observation/v1",
            "prefix":"GOMAD_HISTORY_OBSERVATION ","maximum_bytes":4096,
            "histories":[{"name":"fixture.recovered","expected":{"node":"a","incarnation":1},"actual":{"node":"b","incarnation":1},"expected_events":["1","2"]}],
            "convergence":[{"name":"fixture.converged","handles":[{"node":"a","incarnation":1},{"node":"b","incarnation":1}]}]})).unwrap());
        let (execution, proof) = observe_execution(&producer, &configuration);
        assert_eq!(execution.incarnations.len(), 2);
        let report =
            serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
        assert_eq!(
            report["outcome"],
            if canary { "oracle_failed" } else { "completed" }
        );
        assert_eq!(report["oracles"].as_array().unwrap().len(), 4);
        let case = format!("canary/enabled={canary}");
        let (retained, replay) = publish_replay(&producer, &case, &configuration, true);
        assert_eq!(retained, report);
        observation(
            &producer,
            "TestProviderDetachedOracleCanarySourceLessReplay",
            &case,
            &configuration,
            json!({"producer":proof,"report":report,"replay":replay}),
        );
    }
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_go_grouped_acknowledgment_crash_preserves_independent_durable_recovery() {
    let producer = producer("grouped", false, false);
    let mut spec = scenario(
        json!([{"id":"a","boot":"a-commit","address":"10.0.0.1","volumes":[{"volume":"a","path":"/workspace"}]},
        {"id":"b","boot":"b-commit","address":"10.0.0.2","volumes":[{"volume":"b","path":"/workspace"}]}]),
        json!([]),
        json!([{"action":"start","handle":{"node":"a","incarnation":1}},{"action":"start","handle":{"node":"b","incarnation":1}},
            {"action":"crash","handle":{"node":"a","incarnation":1}},{"action":"crash","handle":{"node":"b","incarnation":1}},
            {"action":"restart","handle":{"node":"a","incarnation":2}},{"action":"restart","handle":{"node":"b","incarnation":2}},
            {"action":"wait","handle":{"node":"a","incarnation":2}},{"action":"wait","handle":{"node":"b","incarnation":2}}]),
        json!([]),
    );
    spec["spec"]["volumes"] =
        json!([{"id":"a","capacity_bytes":8<<20},{"id":"b","capacity_bytes":8<<20}]);
    spec["boots"]=json!(["a","b"].into_iter().flat_map(|node|["commit","recover"].into_iter().map(move |phase|json!({"id":format!("{node}-{phase}"),
        "args":[if phase=="commit" {"durable"} else {"recover"}],"storage":{"volumes":[{"ID":node,"Path":"/workspace","CapacityBytes":8<<20}],
            "limits":{"PendingOperations":1024,"Transitions":10000}}}))).collect::<Vec<_>>());
    spec["incarnation_boots"] = json!([{"handle":{"node":"a","incarnation":2},"boot":"a-recover"},{"handle":{"node":"b","incarnation":2},"boot":"b-recover"}]);
    let mut configuration = bind(&producer, &spec, 7);
    for item in &mut configuration.incarnations {
        item.arguments[0] = "service".into();
    }
    configuration.crash_groups = vec![helper::CrashGroup {
        nodes: vec!["a".into(), "b".into()],
        marker: b"committed durable-a".to_vec(),
    }];
    let (execution, proof) = observe_execution(&producer, &configuration);
    assert_eq!(execution.incarnations.len(), 4);
    assert_eq!(execution.executed_steps.len(), 8);
    assert_eq!(
        canonical::encode(&execution.executed_steps).unwrap(),
        canonical::encode(&configuration.steps).unwrap()
    );
    for item in &execution.incarnations {
        let stdout = std::str::from_utf8(&item.execution.evidence["stdout"]).unwrap();
        if item.handle.incarnation == 1 {
            assert!(item.interrupted);
            assert!(stdout.contains("committed durable-a"), "{stdout}");
        } else {
            assert_eq!(item.execution.outcome.exit_code, Some(0));
            assert!(
                stdout.contains("fresh heap 1\nrecovered durable-a"),
                "{stdout}"
            );
        }
    }
    let (report, replay) = publish_replay(&producer, "grouped/replay", &configuration, true);
    assert_eq!(report["outcome"], "completed");
    observation(
        &producer,
        "TestProviderGroupedAcknowledgmentCrashSourceLessReplay",
        "grouped/replay",
        &configuration,
        json!({"producer":proof,"report":report,"replay":replay}),
    );
}
