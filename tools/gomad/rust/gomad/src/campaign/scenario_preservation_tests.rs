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
