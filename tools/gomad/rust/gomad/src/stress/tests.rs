use super::*;
use crate::evidence::{
    identity::Identity,
    record::{Limits, Outcome, Record, Stream},
    store::{Publication, Store},
};
use std::collections::BTreeMap;

use serde_json::json;
use std::{fmt::Write as _, os::unix::fs::PermissionsExt};

#[test]
fn matrix_preserves_workloads_seeds_and_axes() {
    let cases = cases();
    assert_eq!(cases.len(), 41);
    assert_eq!(
        cases
            .iter()
            .filter(|(name, _)| *name == "runtime")
            .map(|(_, seed)| *seed)
            .collect::<Vec<_>>(),
        (0..32).map(|index| index * 1_000_003).collect::<Vec<_>>()
    );
    assert_eq!(AXES, ["cpu", "concurrent", "layout", "handoff"]);
    assert_eq!(
        cell_status(3, 2, &vec![json!({"status":"identical"}); 2]),
        "skipped"
    );
    assert_eq!(
        cell_status(2, 2, &vec![json!({"status":"identical"}); 2]),
        "pass"
    );
    assert_eq!(cell_status(2, 2, &[]), "skipped");
    assert_eq!(
        cell_status(3, 2, &[json!({"status":"diverged"})]),
        "divergence"
    );
}

#[test]
fn comparison_refuses_forcing_and_identity_drift() {
    let base = json!({"fresh":true,"seed":7,"identities":{"module":"a"}});
    for changed in [
        json!({"fresh":false,"seed":7,"identities":{"module":"a"}}),
        json!({"fresh":true,"seed":7,"identities":{"module":"b"}}),
        json!({"fresh":true,"seed":8,"identities":{"module":"a"}}),
    ] {
        assert!(comparable(&base, &changed).is_err());
    }
    comparable(&base, &base).unwrap();
}

#[test]
fn first_event_precedes_later_changes_and_missing_log_cannot_pass() {
    let mut a = vec![
        json!({"type":"header","schema":"gomad.wasm-event-order/v1"}),
        event(1, "clock", 7),
        event(2, "draw-counter", 8),
    ];
    let mut b = a.clone();
    b[1]["value"] = json!(9);
    b[2]["value"] = json!(10);
    let encode = |rows: &mut Vec<serde_json::Value>| {
        let mut raw = String::new();
        for row in rows {
            writeln!(&mut raw, "gomad-wasm-event-order-v1 {row}").unwrap();
        }
        format!(
            "{raw}gomad-wasm-event-order-v1 {{\"type\":\"complete\",\"events\":2,\"bytes\":{}}}\n",
            raw.len()
        )
        .into_bytes()
    };
    let left = report::parse(&encode(&mut a)).unwrap();
    let right = report::parse(&encode(&mut b)).unwrap();
    let difference = report::first(&left, &right).unwrap();
    assert_eq!(difference["kind"], "clock");
    assert_eq!(difference["coordinate"]["Shared"], 1);
    let expected: Value = serde_json::from_str(difference["expected"].as_str().unwrap()).unwrap();
    assert_eq!(expected["value"], 7);
    assert!(report::parse(b"").is_err());
    assert!(
        report::parse(
            b"gomad-wasm-event-order-v1 {\"type\":\"incomplete\",\"reason\":\"capacity\"}\n"
        )
        .is_err()
    );
}

fn event(order: u64, kind: &str, value: u64) -> serde_json::Value {
    json!({"type":"event","order":order,"kind":kind,"instance":"active","incarnation":1,"ordinal":order,"trace":order,"decision":order,"time_nanos":1,"value":value})
}

#[test]
fn qualification_refuses_every_compiled_stress_role() {
    if linked::Conditions::compiled() == linked::Conditions::Ordinary {
        crate::qualify::require_release_role().unwrap();
        assert_eq!(
            linked::Conditions::compiled().configuration(),
            linked::CONFIGURATION
        );
    } else {
        assert!(
            crate::qualify::require_release_role()
                .unwrap_err()
                .contains("qualification refuses")
        );
    }
}

fn stress_test_root() -> PathBuf {
    static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
    let root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join(format!(
        "../../../../.flow/tmp/stress-test-{}-{}",
        std::process::id(),
        NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
    ));
    std::fs::create_dir_all(root.parent().unwrap()).unwrap();
    directory(&root).unwrap();
    root
}

#[test]
fn comparisons_authenticate_retained_streams_and_reject_hidden_forcing() {
    let root = stress_test_root();
    let publish = |name: &str, mode: u64, target: &[u8]| {
        let row = event(1, "clock", 7);
        let raw = format!(
            "gomad-wasm-event-order-v1 {{\"type\":\"header\",\"schema\":\"gomad.wasm-event-order/v1\"}}\ngomad-wasm-event-order-v1 {row}\n"
        );
        let log = format!(
            "{raw}gomad-wasm-event-order-v1 {{\"type\":\"complete\",\"events\":1,\"bytes\":{}}}\n",
            raw.len()
        )
        .into_bytes();
        let configuration = json!({"model":{"mode":mode,"diagnostics":true}});
        let replay = canonical::encode(&json!({"configuration":configuration})).unwrap();
        let payloads: BTreeMap<String, Vec<u8>> = BTreeMap::from([
            ("evidence/event-order".into(), log),
            ("evidence/replay.json".into(), replay),
            ("evidence/choice".into(), vec![0; 96]),
            ("evidence/diagnostic".into(), vec![0; 96]),
        ]);
        let identity = Identity {
            target: canonical::digest(target),
            toolchain: canonical::digest(b"toolchain"),
            platform: canonical::digest(b"platform"),
            implementation: canonical::digest(b"implementation"),
            configuration: canonical::digest(&canonical::encode(&configuration).unwrap()),
        };
        let record = Record {
            schema: "gomad.execution-record/v2".into(),
            execution_identity: identity.digest().unwrap(),
            identity: identity.clone(),
            profile: "gomad.wasi-cooperative/v1".into(),
            seed: 7,
            arguments: Vec::new(),
            environment: BTreeMap::new(),
            limits: Limits {
                fuel: 1,
                memory_bytes: 65536,
                wall_millis: 1000,
                output_bytes: 1024,
                evidence_bytes: 8192,
            },
            virtual_time_nanos: 1,
            outcome: Outcome {
                class: "success".into(),
                termination: "exit".into(),
                exit_code: Some(0),
                reason: String::new(),
            },
            streams: payloads
                .iter()
                .map(|(name, raw)| {
                    (
                        name.clone(),
                        Stream {
                            bytes: raw.len() as u64,
                            sha256: canonical::digest(raw),
                            records: 1,
                        },
                    )
                })
                .collect(),
        };
        let store = Store {
            root: root.join(name),
            maximum_bytes: 1 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        };
        let published = store.publish(Publication { record, payloads }).unwrap();
        json!({"fresh":true,"seed":7,"artifact":published.path,"identities":{"execution":identity}})
    };
    let first = publish("first", 1, b"target");
    let second = publish("second", 1, b"target");
    assert_eq!(
        report::compare(&first, &second).unwrap()["status"],
        "identical"
    );
    let changed = publish("identity", 1, b"changed");
    assert!(
        report::compare(&first, &changed)
            .unwrap_err()
            .contains("identity")
    );
    let forced = publish("forced", 2, b"target");
    assert!(
        report::compare(&forced, &forced)
            .unwrap_err()
            .contains("forced")
    );
    let path = Path::new(second["artifact"].as_str().unwrap()).join("evidence/event-order");
    std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o600)).unwrap();
    std::fs::write(path, b"tamper").unwrap();
    assert!(report::compare(&first, &second).is_err());
    std::fs::remove_dir_all(root).unwrap();
}

fn original_cluster_scenario() -> Value {
    let raw =
        include_str!("../../../../internal/backends/wasm/command/testdata/scenario/network.json")
            .replace("\"a\"", "\"active\"")
            .replace("\"b\"", "\"standby\"");
    let mut scenario: Value = serde_json::from_str(&raw).unwrap();
    scenario["actions"] = json!(
        [("start", 1), ("crash", 1), ("restart", 2), ("wait", 2)]
            .into_iter()
            .flat_map(|(action, incarnation)| ["active", "standby"].map(
                |node| json!({"action":action,"handle":{"node":node,"incarnation":incarnation}})
            ))
            .collect::<Vec<_>>()
    );
    scenario["output_crashes"] = json!(["active","standby"].map(|node|json!({"handle":{"node":node,"incarnation":1},"marker":b"GOMAD_TEMPORAL_OBSERVATION "})));
    scenario["spec"]["limits"] = cluster::original_limits();
    let arguments = json!([
        "-test.v",
        "-test.count=1",
        "-test.timeout=1h",
        "-test.run=^TestIndependentClusterReplicationRecovery$",
        "-persistenceType=sql",
        "-persistenceDriver=sqlite"
    ]);
    scenario["boots"] = json!(["active","standby"].into_iter().flat_map(|node| ["commit","reopen"].map(|phase| json!({"id":format!("{node}-{phase}"),"args":arguments,"storage":{"volumes":[{"ID":node,"Path":"/volume","CapacityBytes":256 << 20}],"limits":{"PendingOperations":100_000,"Transitions":10_000_000},"sqlite_lock_databases":["/volume/temporal.db"]}}))).collect::<Vec<_>>());
    scenario["spec"]["nodes"] = json!([("active","10.0.0.1"),("standby","10.0.0.2")].map(|(node,address)| json!({"id":node,"boot":format!("{node}-commit"),"address":address,"volumes":[{"volume":node,"path":"/volume"}]})));
    scenario["spec"]["volumes"] =
        json!(["active", "standby"].map(|node| json!({"id":node,"capacity_bytes":256 << 20})));
    scenario["incarnation_boots"] = json!(["active", "standby"].map(
        |node| json!({"handle":{"node":node,"incarnation":2},"boot":format!("{node}-reopen")})
    ));
    scenario["network_faults"] = json!([("partition",1_u64),("heal",30_000_000_000)].map(|(action,at)| json!({"Name":format!("temporal-active-to-standby-{action}"),"Action":action,"From":"active","To":"standby","AtNanos":at,"Symmetric":false})));
    scenario
}

fn cluster_test_configuration() -> crate::replay::preflight::Configuration {
    let base: crate::replay::preflight::Configuration = serde_json::from_value(json!({
        "engine":linked::Conditions::compiled().configuration(),"profile":"gomad.wasi-cooperative/v1",
        "limits":{"fuel":100_000,"memory_bytes":65536,"wall_millis":1000,"output_bytes":1024,"evidence_bytes":16_777_216},
        "arguments":["guest"],"environment":{"PWD":"/"},"derive_seed_environment":true,"runtime_sha256":"00".repeat(32),
        "model":{"entropy_key":vec![0;32],"working_directory_sha256":canonical::digest(b"/"),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"mode":1,"limit":4096,"diagnostics":true,"epoch_nanos":0,"read_step_nanos":0,"now":1,"limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},
        "namespace":{"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0},"storage":null
    })).unwrap();
    base
}

#[test]
fn cluster_role_inputs_are_preserved_in_the_retained_configuration() {
    let small = json!({"history":"YQ==","event_ids":[1]});
    cluster::history_oracle(&small, &small).unwrap();
    let large = json!({"history":"YQ==".repeat(1 << 18),"event_ids":[1]});
    assert!(cluster::history_oracle(&small, &large).is_err());
    let mut scenario = original_cluster_scenario();
    cluster::validate_scenario(&canonical::encode(&scenario).unwrap()).unwrap();
    scenario["spec"]["limits"]["crash_states"] = json!(1);
    assert!(cluster::validate_scenario(&canonical::encode(&scenario).unwrap()).is_err());
    scenario["spec"]["limits"] = cluster::original_limits();
    let base = cluster_test_configuration();
    let identity = linked::ChoiceIdentity {
        target_sha256: [0; 32],
        toolchain_build_key: "00".repeat(32),
        goos: "wasip1".into(),
        goarch: "wasm".into(),
        implementation_sha256: [0; 32],
    };
    let network = serde_json::from_value(json!({"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":4096,"bytes":128 << 20,"connection_bytes":4 << 20,"stall_nanos":10_000_000_000_u64,"transitions":1 << 20}})).unwrap();
    let mut bound = crate::campaign::scenario::bind(
        &canonical::encode(&scenario).unwrap(),
        &base,
        b"\0asm\x01\0\0\0\x05\x03\x01\x00\x01",
        &identity,
        Some(&network),
        7,
    )
    .unwrap();
    cluster::validate_original(&bound).unwrap();
    bound.request.network.as_mut().unwrap().faults[0].at_nanos = 2;
    assert!(
        cluster::validate_original(&bound)
            .unwrap_err()
            .contains("fault schedule")
    );
    bound.request.network.as_mut().unwrap().faults[0].at_nanos = 1;
    cluster::bind(&mut bound).unwrap();
    assert!(bound.request.boundaries.is_empty());
    assert_eq!(
        serde_json::to_value(&bound.request.crash_groups).unwrap(),
        json!([{"nodes":["active","standby"],"marker":b"GOMAD_TEMPORAL_OBSERVATION "}])
    );
    assert_eq!(
        serde_json::to_value(&bound.request.crash_groups).unwrap(),
        serde_json::to_value(&bound.configuration.crash_groups).unwrap()
    );
    for incarnation in &bound.request.incarnations {
        let model = incarnation.request.model.as_ref().unwrap();
        let retained = bound
            .configuration
            .incarnations
            .iter()
            .find(|item| item.handle == incarnation.handle)
            .unwrap();
        let node = &incarnation.handle.node;
        let phase = if incarnation.handle.incarnation == 1 {
            "commit"
        } else {
            "reopen"
        };
        let peer = if node == "active" {
            "10.0.0.2"
        } else {
            "10.0.0.1"
        };
        for entry in [
            format!("GOMAD_TEMPORAL_CLUSTER={node}"),
            format!("GOMAD_TEMPORAL_PHASE={phase}"),
            format!("GOMAD_TEMPORAL_REMOTE_FRONTEND={peer}:7233"),
            format!("GOMAD_TEMPORAL_REMOTE_HTTP={peer}:7243"),
            format!("GOMAD_TEMPORAL_REMOTE_HISTORY={peer}:7234"),
        ] {
            assert!(model.environment.contains(&entry));
            let (name, value) = entry.split_once('=').unwrap();
            assert_eq!(retained.environment[name], value);
        }
        assert_eq!(
            retained.model["entropy_key"],
            json!(model.configuration.entropy_key)
        );
        assert!(retained.model.get("seed").is_none());
    }
    assert!(
        cluster::bind(&mut bound)
            .unwrap_err()
            .contains("already bound")
    );
}

#[test]
fn canonical_soft_assertion_failure_cannot_pass_any_stress_workload() {
    let prepared = prepare::Prepared {
        compile_cache_root: None,
        module: b"(module)".to_vec(),
        configuration: cluster_test_configuration(),
        toolchain: canonical::digest(b"toolchain"),
        choice_identity: None,
        scenario: None,
        provenance: br#"{"inputs":{"kind":"go-test"}}"#.to_vec(),
        network: None,
        soft_assertion_report_only: false,
    };
    let mut accepted = Vec::new();
    for assertion in [b"failed assertion: stress witness\n".as_slice(), b"\xff"] {
        let mut terminal = linked::Outcome::new("exit", "");
        terminal.exit_code = Some(0);
        let mut stdout = b"--- PASS: TestWorkflowTypeEncodingSuite\nPASS\n".to_vec();
        stdout.extend_from_slice(assertion);
        let execution = linked::Execution {
            authorization_to_terminal_nanos: None,
            outcome: terminal,
            evidence: BTreeMap::from([("stdout".into(), stdout)]),
            event_order: None,
        };
        let mut result = execution::publication(
            &prepared,
            &Job {
                ordinal: 1,
                seed: 7,
            },
            execution,
        )
        .unwrap();
        assert_eq!(result.publication.record.outcome.class, "target_failure");
        result.publication.payloads.insert(
            "evidence/stress-oracles.json".into(),
            br#"{"passed":true}"#.to_vec(),
        );
        for name in ["workflow", "two-cluster"] {
            if cluster::outcome(name, &prepared, &result.publication).is_ok() {
                accepted.push(name);
            }
        }
        result.publication.record.outcome.exit_code = Some(17);
        result
            .publication
            .payloads
            .insert("output/stdout".into(), b"canary 2\n".to_vec());
        assert!(cluster::canonical_outcome("runtime", &result.publication).is_err());
        result.publication.record.outcome.exit_code = Some(0);
    }
    assert!(
        accepted.is_empty(),
        "canonical assertion failures accepted by {accepted:?}"
    );
}

#[test]
fn runtime_exit17_exception_requires_the_original_canary_and_failure_class() {
    let identity = Identity {
        target: canonical::digest(b"t"),
        toolchain: canonical::digest(b"c"),
        platform: canonical::digest(b"p"),
        implementation: canonical::digest(b"i"),
        configuration: canonical::digest(b"m"),
    };
    let mut publication = Publication {
        record: Record {
            schema: "gomad.execution-record/v2".into(),
            execution_identity: identity.digest().unwrap(),
            identity,
            profile: "gomad.wasi-cooperative/v1".into(),
            seed: 7,
            arguments: Vec::new(),
            environment: BTreeMap::new(),
            limits: Limits {
                fuel: 1,
                memory_bytes: 65536,
                wall_millis: 1,
                output_bytes: 1024,
                evidence_bytes: 1024,
            },
            virtual_time_nanos: 0,
            outcome: Outcome {
                class: "target_failure".into(),
                termination: "exit".into(),
                exit_code: Some(17),
                reason: String::new(),
            },
            streams: BTreeMap::new(),
        },
        payloads: BTreeMap::new(),
    };
    assert!(cluster::canonical_outcome("runtime", &publication).is_err());
    publication
        .payloads
        .insert("output/stdout".into(), b"canary 2\n".to_vec());
    cluster::canonical_outcome("runtime", &publication).unwrap();
    assert!(cluster::canonical_outcome("workflow", &publication).is_err());
    publication.record.outcome.class = "runner_failure".into();
    assert!(cluster::canonical_outcome("runtime", &publication).is_err());
}

#[test]
fn rerun_supervision_restores_axis_bounds_and_launches_the_full_cohort() {
    let root = stress_test_root();
    let binary = root.join("probe");
    std::fs::write(&binary, r"#!/usr/bin/python3
import json, os, pathlib, sys, time
request = json.loads(pathlib.Path(sys.argv[3]).read_text())
root = pathlib.Path(request['root'])
(root.parent / ('arrival-' + str(os.getpid()))).touch()
deadline = time.monotonic() + 5
while len(list(root.parent.glob('arrival-*'))) < request['supervision']['cohort_size']:
    if time.monotonic() > deadline:
        raise RuntimeError('cohort was executed serially or incompletely')
    time.sleep(.005)
(root / 'observation.json').write_text(json.dumps({'process_id':os.getpid(), 'oracle_error':None, 'perturbation_error':None, 'event_order_error':None}))
").unwrap();
    std::fs::set_permissions(&binary, std::fs::Permissions::from_mode(0o700)).unwrap();
    let mut json = false;
    let options =
        crate::cli::explore::parse(&["go-run".into(), "./unused".into()], &mut json).unwrap();
    let mut request = Request {
        workload: "runtime".into(),
        seed: 7,
        axis: "concurrent".into(),
        options,
        root: root.join("original"),
        source_sha256: canonical::digest(b"source"),
        target_pool: root.join("targets"),
        supervision: Supervision {
            cpu_workers: 0,
            cohort_size: 3,
            repetition: 5,
            member: 1,
        },
    };
    let report = process::supervise(&binary, &request, &root.join("rerun")).unwrap();
    assert_eq!(report["passed"], true);
    let runs = report["runs"].as_array().unwrap();
    assert_eq!(runs.len(), 3);
    assert_eq!(
        runs.iter()
            .map(|run| run["process_id"].as_u64().unwrap())
            .collect::<std::collections::BTreeSet<_>>()
            .len(),
        3
    );
    request.axis = "cpu".into();
    request.supervision.cohort_size = 1;
    request.supervision.member = 0;
    request.supervision.cpu_workers = 4;
    let members = process::supervision_requests(&request, &root.join("cpu-rerun")).unwrap();
    assert_eq!(members.len(), 1);
    assert_eq!(members[0].supervision.cpu_workers, 4);
    assert_eq!(members[0].supervision.repetition, 5);
    request.supervision.cpu_workers = 0;
    assert!(process::supervision_requests(&request, &root.join("invalid")).is_err());
    let command = rerun(
        &binary,
        &[
            "stress".into(),
            "--cell".into(),
            "/retained-request.json".into(),
        ],
        &root,
    )
    .unwrap();
    assert!(
        command
            .windows(2)
            .any(|pair| pair == ["--rerun", "/retained-request.json"])
    );
    assert!(!command.iter().any(|argument| argument == "--cell"));
    std::fs::remove_dir_all(root).unwrap();
}
