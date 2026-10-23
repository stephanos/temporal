use super::super::soft_assertions::{Report, observe};
use super::*;

#[test]
fn output_capacity_counters_do_not_prove_soft_assertion_completeness() {
    for report_only in [false, true] {
        let complete = output_complete("capacity", "output capacity", None, 4, 4);
        let report = observe(b"PASS", b"", complete, report_only).unwrap();
        assert_eq!(
            report,
            Report {
                messages: vec![],
                report_only,
                inconclusive: true,
            }
        );
        let assertions = Assertions {
            schema: "gomad.cluster-soft-assertions/v1".into(),
            incarnations: vec![IncarnationAssertions {
                handle: helper::Handle {
                    node: "a".into(),
                    incarnation: 1,
                },
                report,
            }],
        };
        assert!(assertions.failed());
    }
}

#[test]
fn incomplete_output_and_invalid_utf8_remain_inconclusive_under_both_policies() {
    for report_only in [false, true] {
        for (output, offered) in [(b"PASS\n".as_slice(), 100), (&[0xff][..], 1)] {
            let complete = output_complete("exit", "", None, offered, output.len() as u64);
            let report = observe(output, b"", complete, report_only).unwrap();
            assert_eq!(
                report,
                Report {
                    messages: vec![],
                    report_only,
                    inconclusive: true,
                }
            );
        }
    }
}

#[test]
fn complete_soft_assertion_overrides_pass_only_without_report_only() {
    for report_only in [false, true] {
        let output = b"PASS\nfailed assertion: invalid state\n";
        let report = observe(
            output,
            b"",
            output_complete("exit", "", None, output.len() as u64, output.len() as u64),
            report_only,
        )
        .unwrap();
        assert_eq!(
            report,
            Report {
                messages: vec!["invalid state".into()],
                report_only,
                inconclusive: false,
            }
        );
        assert_eq!(
            Assertions {
                schema: "gomad.cluster-soft-assertions/v1".into(),
                incarnations: vec![IncarnationAssertions {
                    handle: helper::Handle {
                        node: "a".into(),
                        incarnation: 1,
                    },
                    report,
                }],
            }
            .failed(),
            !report_only
        );
    }
}

#[test]
fn literal_braces_survive_output_capacity_annotation() {
    for output in [
        "failed assertion: invalid {\"left\":1}",
        "2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true}",
    ] {
        let expected = if output.starts_with("failed assertion:") {
            "invalid {\"left\":1}"
        } else {
            "invalid {state}"
        };
        for report_only in [false, true] {
            let report = observe(
                b"",
                output.as_bytes(),
                output_complete(
                    "capacity",
                    "output capacity",
                    None,
                    output.len() as u64,
                    output.len() as u64,
                ),
                report_only,
            )
            .unwrap();
            assert_eq!(
                report,
                Report {
                    messages: vec![expected.into()],
                    report_only,
                    inconclusive: true,
                }
            );
        }
    }
}

#[test]
fn only_declared_model_capacity_interruptions_allow_complete_output() {
    let interrupted = json!({"configured":true,"finished":false});
    for resource in [
        "event-ids",
        "events",
        "payload-bytes",
        "queued-events",
        "request-ids",
        "requests",
        "transition-bytes",
        "transition-sequences",
        "transitions",
    ] {
        let message = format!("WASI capacity: {resource} reached");
        assert!(output_complete(
            "capacity",
            &message,
            Some(&interrupted),
            4,
            4
        ));
        for model in [
            None,
            Some(json!({"configured":false,"finished":false})),
            Some(json!({"configured":true,"finished":true})),
        ] {
            assert!(!output_complete("capacity", &message, model.as_ref(), 4, 4));
        }
        assert!(!output_complete(
            "capacity",
            &message,
            Some(&interrupted),
            5,
            4
        ));
    }
    for message in [
        "output capacity",
        "transcript capacity",
        "WASI capacity: output-bytes",
        "WASI capacity: requests-extra",
    ] {
        assert!(!output_complete(
            "capacity",
            message,
            Some(&interrupted),
            4,
            4
        ));
    }
    for termination in ["cancelled", "watchdog_timeout"] {
        assert!(!output_complete(termination, "", Some(&interrupted), 4, 4));
    }
}

fn configuration() -> cluster::Configuration {
    let limits = json!({"fuel":100_000,"memory_bytes":65_536,"wall_millis":1000,
        "output_bytes":1<<20,"evidence_bytes":64<<20});
    let namespace = json!({"entries":[],"writable_directories":[],"preopens":[[47]],
        "working_directory":[47],"stdin":[],"epoch_nanos":0,
        "limits":{"files":1024,"descriptors":128,"filesystem_bytes":8<<20,"output_bytes":1<<20}});
    let owners = ["b", "a"].map(|node| json!({
        "handle":{"node":node,"incarnation":1},
        "module":{"file":"target/module.wasm","bytes":0,"sha256":"a".repeat(64),"limit":512<<20},
        "limits":limits,"arguments":[],"environment":{},"runtime_sha256":"b".repeat(64),
        "model":{},"namespace":namespace,"storage":null,"network":null,
        "choice_identity":{"TargetSHA256":vec![0u8;32],"ToolchainBuildKey":"c".repeat(64),
            "GOOS":"wasip1","GOARCH":"wasm","ImplementationSHA256":vec![0u8;32]}
    }));
    serde_json::from_value(json!({
        "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1","limits":limits,
        "broker":{"seed":0,"limits":{"max_requests":65536,"max_events":65536,
            "max_queued_events":1024,"max_transitions":1<<20,"max_payload_bytes":1<<20,"max_string_bytes":1024}},
        "network":null,"incarnations":owners,"steps":[],"boundaries":[],"crash_groups":[]
    })).unwrap()
}

fn execution(nodes: &[&str]) -> helper::Execution {
    let output = b"PASS\nfailed assertion: invalid {state}\n";
    helper::Execution {
        outcome: linked::Outcome::new("exit", ""),
        executed_steps: vec![],
        incarnations: nodes
            .iter()
            .map(|node| {
                let mut outcome = linked::Outcome::new("exit", "");
                outcome.exit_code = Some(0);
                outcome.output_bytes = output.len() as u64;
                helper::IncarnationResult {
                    handle: helper::Handle {
                        node: (*node).into(),
                        incarnation: 1,
                    },
                    interrupted: false,
                    execution: linked::Execution {
                        terminal_monotonic_nanos: None,
                        admission_terminal_to_return_nanos: None,
                        timing: None,
                        #[cfg(feature = "helper-guest-profile")]
                        guest_profile_error: None,
                        #[cfg(feature = "helper-diagnostic-profile")]
                        external_owner_attempts: 0,
                        authorization_to_terminal_nanos: None,
                        outcome,
                        evidence: BTreeMap::from([("stdout".into(), output.to_vec())]),
                        event_order: None,
                    },
                }
            })
            .collect(),
        world: vec![],
        network: vec![],
        inspection: vec![],
        virtual_elapsed_nanos: 0,
        event_order: None,
    }
}

#[test]
fn assertion_evidence_is_ordered_by_handle_and_refuses_both_capacity_bounds() {
    let configuration = configuration();
    let execution = execution(&["b", "a"]);
    for report_only in [false, true] {
        let reports = assertions(&execution, &configuration, report_only, 2, 4096)
            .unwrap()
            .unwrap();
        assert_eq!(
            reports
                .incarnations
                .iter()
                .map(|item| item.handle.node.as_str())
                .collect::<Vec<_>>(),
            ["a", "b"]
        );
        for item in &reports.incarnations {
            assert_eq!(
                item.report,
                Report {
                    messages: vec!["invalid {state}".into()],
                    report_only,
                    inconclusive: false,
                }
            );
        }
        let bytes = canonical::encode(&reports).unwrap().len() as u64;
        assert!(assertions(&execution, &configuration, report_only, 2, bytes).is_ok());
        assert_eq!(
            assertions(&execution, &configuration, report_only, 1, 4096)
                .err()
                .unwrap(),
            "capacity: scenario soft assertion results"
        );
        assert_eq!(
            assertions(&execution, &configuration, report_only, 2, bytes - 1)
                .err()
                .unwrap(),
            "capacity: scenario soft assertion evidence bytes"
        );
    }
}

#[test]
fn report_only_never_treats_a_missing_incarnation_as_complete() {
    let configuration = configuration();
    let execution = execution(&["a"]);
    for report_only in [false, true] {
        let reports = assertions(&execution, &configuration, report_only, 2, 4096)
            .unwrap()
            .unwrap();
        assert!(reports.failed());
        assert_eq!(reports.incarnations.len(), if report_only { 2 } else { 1 });
        if report_only {
            assert_eq!(reports.incarnations[1].handle.node, "b");
            assert_eq!(
                reports.incarnations[1].report,
                Report {
                    messages: vec![],
                    report_only: true,
                    inconclusive: true,
                }
            );
            assert!(reports.failed());
        }
    }
}

#[test]
fn retained_world_profile_authenticates_seed_capacities_and_binary_owner_bytes() {
    const PROFILE: &str = "gomad/world/compressed-snapshot/v2";
    let raw = crate::qualify::cells::base64("R01XU05BUDEAAAABAAAAAAAAAAAAAAAAACAAAAAAAAAAIAAAAAAAAAAD0JAAAAAAAIAAAAAAAAAEAAAA/////w0jTM9SQwAAAAAAAAAAAAEAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFVkPwAshKEN5wBwZmobp69AiLehiWr2Xs5nRltJYTzTVQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==").unwrap();
    let broker: helper::BrokerConfig = serde_json::from_value(json!({"seed":0,"limits":{
        "max_requests":2_097_152,"max_events":2_097_152,"max_queued_events":250_000,
        "max_transitions":8_388_608,"max_payload_bytes":67_108_864,"max_string_bytes":u32::MAX}}))
    .unwrap();
    let compressed = helper::encode_world(&raw, PROFILE).unwrap();
    helper::validate_encoded_world(&compressed, PROFILE, &broker).unwrap();
    helper::validate_encoded_world(&raw, "", &broker).unwrap();
    for (seed, requests) in [
        (1, broker.limits.max_requests),
        (0, broker.limits.max_requests - 1),
    ] {
        let changed: helper::BrokerConfig = serde_json::from_value(json!({"seed":seed,"limits":{
            "max_requests":requests,"max_events":broker.limits.max_events,
            "max_queued_events":broker.limits.max_queued_events,
            "max_transitions":broker.limits.max_transitions,
            "max_payload_bytes":broker.limits.max_payload_bytes,
            "max_string_bytes":broker.limits.max_string_bytes}}))
        .unwrap();
        assert!(helper::validate_encoded_world(&compressed, PROFILE, &changed).is_err());
    }
    assert!(helper::validate_encoded_world(&compressed, "", &broker).is_err());
    assert!(helper::validate_encoded_world(&raw, PROFILE, &broker).is_err());
    let mut trailing = raw;
    trailing.push(0);
    assert!(helper::validate_encoded_world(&trailing, "", &broker).is_err());
}

fn operator_fixture() -> Value {
    let mut scenario: Value = serde_json::from_slice(include_bytes!(
        "../../../../internal/backends/wasm/command/testdata/scenario/durable.json"
    ))
    .unwrap();
    scenario["spec"]["nodes"] = json!([
        {"id":"a","boot":"main","address":"10.0.0.1"},
        {"id":"b","boot":"main","address":"10.0.0.2"}
    ]);
    scenario["spec"]["volumes"] = json!([]);
    scenario["boots"] = json!([{"id":"main"}]);
    scenario["incarnation_boots"] = json!([]);
    scenario["output_crashes"] = json!([]);
    scenario["actions"] = json!([
        {"action":"start","handle":{"node":"a","incarnation":1}},
        {"action":"start","handle":{"node":"b","incarnation":1}},
        {"action":"wait","handle":{"node":"a","incarnation":1}},
        {"action":"wait","handle":{"node":"b","incarnation":1}}
    ]);
    scenario
}

#[test]
fn operator_strict_preflight_preserves_every_original_refusal() {
    for name in [
        "valid",
        "version",
        "duplicate-node",
        "missing-boot",
        "incarnation",
        "ordering",
        "unbound-crash",
        "capacity",
        "fault-capacity",
        "callback",
        "missing-reference",
        "unknown-restart-boot",
        "duplicate-boot",
        "active-restart",
        "incomplete",
    ] {
        let mut scenario = operator_fixture();
        match name {
            "valid" => {}
            "version" => scenario["schema"] = "unknown".into(),
            "duplicate-node" => scenario["spec"]["nodes"][1]["id"] = "a".into(),
            "missing-boot" => scenario["boots"] = Value::Null,
            "incarnation" => scenario["actions"][0]["handle"]["incarnation"] = 2.into(),
            "ordering" => scenario["actions"][0]["action"] = "restart".into(),
            "unbound-crash" => scenario["actions"][2]["action"] = "crash".into(),
            "capacity" => scenario["spec"]["limits"]["nodes"] = 1.into(),
            "fault-capacity" => {
                scenario["spec"]["limits"]["fault_actions"] = 1.into();
                scenario["network_faults"] = json!([{"Name":"one","Action":"","From":"","To":""},{"Name":"two","Action":"","From":"","To":""}]);
            }
            "callback" => scenario["actions"][0]["action"] = "callback".into(),
            "missing-reference" => scenario["actions"][0]["handle"]["node"] = "missing".into(),
            "unknown-restart-boot" => {
                scenario["incarnation_boots"] = json!([
                    {"handle":{"node":"a","incarnation":2},"boot":"missing"}
                ]);
            }
            "duplicate-boot" => scenario["boots"] = json!([{"id":"main"},{"id":"main"}]),
            "active-restart" => {
                scenario["actions"][2] = json!(
                    {"action":"restart","handle":{"node":"a","incarnation":2}}
                );
            }
            "incomplete" => {
                scenario["actions"].as_array_mut().unwrap().truncate(2);
            }
            _ => unreachable!(),
        }
        let normalized = normalize(&canonical::encode(&scenario).unwrap());
        assert_eq!(
            normalized.is_ok(),
            name == "valid",
            "{name}: {normalized:?}"
        );
    }
    let raw = canonical::encode(&operator_fixture()).unwrap();
    for invalid in [
        [raw.as_slice(), b" {}"].concat(),
        [b"{\"unknown\":true,".as_slice(), &raw[1..]].concat(),
    ] {
        assert!(normalize(&invalid).is_err());
    }
}

#[test]
fn operator_arguments_preserve_null_empty_and_override_through_normalization_and_binding() {
    let base: preflight::Configuration = serde_json::from_value(json!({
        "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1",
        "limits":{"fuel":100_000,"memory_bytes":65_536,"wall_millis":1000,
            "output_bytes":1024,"evidence_bytes":1<<20},
        "arguments":["guest","inherited"],"environment":{"PWD":"/workspace"},
        "runtime_sha256":"b".repeat(64),"namespace":null,
        "model":{"entropy_key":vec![0u8;32],"working_directory_sha256":"c".repeat(64),
            "arguments":{"bytes":0,"count":0,"chunks":0},
            "environment":{"bytes":0,"count":0,"chunks":0},
            "seed":7,"mode":1,"limit":1<<20,"diagnostics":false,
            "epoch_nanos":946_684_800_000_000_000_u64,"read_step_nanos":0,"now":1,
            "limits":{"calls":1<<20,"transcript_bytes":1<<20,"pending_events":1024}}
    }))
    .unwrap();
    for (args, expected) in [
        (Value::Null, vec!["guest", "inherited"]),
        (json!([]), vec!["guest"]),
        (json!(["override"]), vec!["guest", "override"]),
    ] {
        let mut value = operator_fixture();
        value["boots"][0]["args"] = args.clone();
        let canonical = normalize(&canonical::encode(&value).unwrap()).unwrap();
        let again = normalize(&canonical).unwrap();
        assert_eq!(canonical, again);
        let sealed = parse(&again).unwrap();
        assert_eq!(
            serde_json::to_value(&sealed.boots[0]).unwrap()["args"],
            args
        );
        let request = boot_request(&sealed.boots[0], &base, b"module", 7).unwrap();
        assert_eq!(request.model.unwrap().arguments, expected);
    }
}

#[test]
fn only_crash_before_storage_fault_declares_a_lifecycle_crash() {
    for action in [
        Some("crash-before"),
        Some("crash-after"),
        Some("io-error"),
        None,
    ] {
        let mut value = operator_fixture();
        value["spec"]["nodes"] = json!([
            {"id":"a","boot":"main","address":"10.0.0.1",
                "volumes":[{"volume":"data","path":"/workspace"}]}
        ]);
        value["spec"]["volumes"] = json!([{"id":"data","capacity_bytes":1024}]);
        value["actions"] = json!([
            {"action":"start","handle":{"node":"a","incarnation":1}},
            {"action":"crash","handle":{"node":"a","incarnation":1}}
        ]);
        value["boots"][0]["storage"] = json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1024}],
            "limits":{"PendingOperations":100,"Transitions":1000},
            "faults":action.map(|action| vec![json!({
                "name":"crash","resource":"/workspace/data","operation":"sync",
                "occurrence":1,"action":action
            })]).unwrap_or_default()
        });
        if action.is_none() {
            value["boots"][0]["storage"] = Value::Null;
            value["spec"]["volumes"] = json!([]);
            value["spec"]["nodes"][0]["volumes"] = json!([]);
        }
        let raw = canonical::encode(&value).unwrap();
        let normalized = normalize(&raw);
        assert_eq!(
            normalized.is_ok(),
            action == Some("crash-before"),
            "{action:?}: {normalized:?}"
        );
        println!(
            "{}",
            json!({
                "schema":"gomad.native-lifecycle-predicate/v1",
                "original_roots":["TestStorageCrashBoundOnlyByCrashBefore","TestOperatorAcceptsDeclaredStorageCrash"],
                "variant":action.unwrap_or("no-storage"),
                "observed":{"accepted":normalized.is_ok(),"error":normalized.as_ref().err()}
            })
        );
        if let Ok(normalized) = normalized {
            let sealed = parse(&normalized).unwrap();
            storage_input(sealed.boots[0].storage.as_ref().unwrap()).unwrap();
        }
    }
}

fn capacity_finalization_case(
    mode: &str,
    reverse: bool,
) -> (
    super::super::prepare::Prepared,
    cluster::Configuration,
    helper::Execution,
) {
    let report_only = mode == "report-only-budget";
    let capacity_node = if mode == "capacity-later" { "b" } else { "a" };
    let nodes = if mode.starts_with("capacity-") {
        vec!["a", "b"]
    } else {
        vec!["a"]
    };
    let mut scenario = operator_fixture();
    scenario["spec"]["nodes"]
        .as_array_mut()
        .unwrap()
        .retain(|node| nodes.contains(&node["id"].as_str().unwrap()));
    scenario["actions"]
        .as_array_mut()
        .unwrap()
        .retain(|step| nodes.contains(&step["handle"]["node"].as_str().unwrap()));
    if mode.ends_with("budget") {
        scenario["spec"]["limits"]["scenario_evidence_bytes"] = 1.into();
    }
    let raw = canonical::encode(&scenario).unwrap();
    let scenario = parse(&raw).unwrap();
    let mut configuration = configuration();
    configuration
        .incarnations
        .retain(|owner| nodes.contains(&owner.handle.node.as_str()));
    configuration.steps.clone_from(&scenario.actions);
    let mut observed = execution(&nodes);
    observed.executed_steps.clone_from(&scenario.actions);
    observed.outcome = linked::Outcome::new("capacity", "WASI capacity: offered-output-bytes");
    for owner in &mut observed.incarnations {
        owner.execution.evidence = BTreeMap::from([("stdout".into(), b"PASS".to_vec())]);
        owner.execution.outcome.output_bytes = 4;
        if owner.handle.node == capacity_node {
            owner.execution.outcome =
                linked::Outcome::new("capacity", "WASI capacity: offered-output-bytes");
            owner.execution.outcome.output_bytes = 4;
        }
    }
    let base: preflight::Configuration = serde_json::from_value(json!({
        "engine":linked::CONFIGURATION,"profile":"gomad.wasi-cooperative/v1",
        "limits":configuration.limits,"arguments":["guest"],"environment":{},
        "runtime_sha256":"b".repeat(64),"namespace":null,"model":null
    }))
    .unwrap();
    let prepared = super::super::prepare::Prepared {
        auxiliary: BTreeMap::new(),
        module: b"\0asm\x01\0\0\0".to_vec(),
        configuration: base,
        toolchain: "1".repeat(64),
        choice_identity: None,
        scenario: Some(raw),
        provenance: canonical::encode(&json!({"inputs":{"kind":"go-test"}})).unwrap(),
        network: None,
        soft_assertion_report_only: report_only,
        compile_cache_root: None,
    };
    if reverse {
        observed.incarnations.reverse();
    }
    (prepared, configuration, observed)
}

#[test]
fn output_capacity_precedes_assertion_budget_and_choice_finalization_errors() {
    for mode in [
        "default-budget",
        "report-only-budget",
        "capacity-first",
        "capacity-later",
    ] {
        let mut expected_identity = None;
        for reverse in [false, true] {
            let (prepared, configuration, observed) = capacity_finalization_case(mode, reverse);
            let report_only = prepared.soft_assertion_report_only;
            let job = super::super::ledger::Job {
                ordinal: 0,
                seed: 7,
            };
            let result = publication(&prepared, &job, configuration, observed, false)
                .unwrap_or_else(|error| {
                    panic!("{mode} lost capacity and assertion report: {error}")
                });
            let super::super::execution::Completion::Failed(failure) = result else {
                panic!("{mode} acquired a canonical record and publication");
            };
            assert_eq!(failure.termination, "capacity", "{mode}");
            assert_eq!(
                failure.reason, "WASI capacity: offered-output-bytes",
                "{mode}"
            );
            assert!(
                failure.message().contains(if mode.ends_with("budget") {
                    "scenario soft assertion evidence bytes"
                } else {
                    "choice"
                }),
                "{mode}: {}",
                failure.message()
            );
            let report: Report =
                serde_json::from_value(failure.soft_assertions.clone().unwrap()).unwrap();
            assert_eq!(
                report,
                Report {
                    messages: vec![],
                    report_only,
                    inconclusive: true
                },
                "{mode}"
            );
            let mut output = Vec::new();
            assert_eq!(failure.report(true, &mut output).unwrap(), 3);
            let event: Value = serde_json::from_slice(&output).unwrap();
            assert_eq!(event["classification"], "capacity");
            assert_eq!(
                event["soft_assertions"],
                serde_json::to_value(&report).unwrap()
            );
            assert!(event.get("record").is_none());
            assert!(event.get("artifact").is_none());
            let signature = canonical::digest(
                &canonical::encode(&json!({
                    "termination":failure.termination,"reason":failure.reason,
                    "soft_assertions":failure.soft_assertions
                }))
                .unwrap(),
            );
            if let Some(expected) = &expected_identity {
                assert_eq!(
                    &signature, expected,
                    "{mode}: completion order changed failure identity"
                );
            } else {
                expected_identity = Some(signature);
            }
        }
    }
}

#[test]
fn output_capacity_reporting_does_not_reclassify_other_global_failures() {
    for (termination, message) in [
        ("infrastructure", "output capacity"),
        ("capacity", "transcript capacity"),
        ("exit", ""),
    ] {
        let (_, configuration, mut observed) = capacity_finalization_case("capacity-first", false);
        observed.outcome = linked::Outcome::new(termination, message);
        assert!(
            output_capacity_failure(&observed, &configuration, true, 16, 4096)
                .unwrap()
                .is_none()
        );
    }
}

#[test]
fn output_capacity_failure_retains_the_selected_actual_static_message() {
    let (_, configuration, mut observed) = capacity_finalization_case("capacity-later", true);
    let selected = observed
        .incarnations
        .iter_mut()
        .find(|owner| owner.handle.node == "b")
        .unwrap();
    selected.execution.evidence.insert(
        "stderr".into(),
        b"failed assertion: invalid {state}".to_vec(),
    );
    let failure = output_capacity_failure(&observed, &configuration, true, 16, 4096)
        .unwrap()
        .unwrap();
    assert_eq!(failure.termination, "capacity");
    assert_eq!(
        serde_json::from_value::<Report>(failure.soft_assertions.unwrap()).unwrap(),
        Report {
            messages: vec!["invalid {state}".into()],
            report_only: true,
            inconclusive: true
        }
    );
}

#[path = "scenario_sqlite_predicate_tests.rs"]
mod scenario_sqlite_predicate_tests;

#[test]
fn cluster_evidence_bound_is_per_incarnation_not_the_oracle_evidence_bound() {
    let mut scenario = operator_fixture();
    scenario["spec"]["limits"]["scenario_evidence_bytes"] = 1.into();
    let starts = scenario["actions"]
        .as_array()
        .unwrap()
        .iter()
        .filter(|step| matches!(step["action"].as_str(), Some("start" | "restart")))
        .count() as u64;
    assert!(starts > 1);
    let raw = canonical::encode(&scenario).unwrap();
    assert_eq!(
        evidence_limit(&raw, 208 << 20).unwrap(),
        starts * (208 << 20)
    );
    assert!(evidence_limit(&raw, u64::MAX).is_err());
}
