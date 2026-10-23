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
    let raw = crate::qualify::cells::base64("R01XU05BUDEAAAABAAAAAAAAAAAAAAAAACAAAAAAAAAAIAAAAAAAAAAD0JAAAAAAAIAAAAAAAAAEAAAA/////w0jTM9SQwAAAAAAAAAAAAEAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFVkPwAshKEN5wBwZmobp69AiLehiWr2Xs5nRltJYTzTVQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==").unwrap();
    let broker: helper::BrokerConfig = serde_json::from_value(json!({"seed":0,"limits":{
        "max_requests":2_097_152,"max_events":2_097_152,"max_queued_events":250_000,
        "max_transitions":8_388_608,"max_payload_bytes":67_108_864,"max_string_bytes":u32::MAX}}))
    .unwrap();
    const PROFILE: &str = "gomad/world/compressed-snapshot/v2";
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
                ])
            }
            "duplicate-boot" => scenario["boots"] = json!([{"id":"main"},{"id":"main"}]),
            "active-restart" => {
                scenario["actions"][2] = json!(
                    {"action":"restart","handle":{"node":"a","incarnation":2}}
                )
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
        let raw = canonical::encode(&value).unwrap();
        let normalized = normalize(&raw);
        assert_eq!(
            normalized.is_ok(),
            action == Some("crash-before"),
            "{action:?}: {normalized:?}"
        );
        if let Ok(normalized) = normalized {
            let sealed = parse(&normalized).unwrap();
            storage_input(sealed.boots[0].storage.as_ref().unwrap()).unwrap();
        }
    }
}
