use super::*;

pub(super) fn validate(
    root: &str,
    variant: &str,
    o: &Value,
    inputs: Option<&Value>,
) -> Result<(), String> {
    require(
        catalog().contains(&(root.to_owned(), variant.to_owned())),
        "unknown lifecycle case",
    )?;
    match root {
        MODEL if variant.starts_with("memory-capacity-") => capacity::validate(variant, o)?,
        DRAIN if variant == "held-peer-release" => held_drain(o)?,
        OPERATOR | DRAIN => {
            let operator = root == OPERATOR;
            let mut names = vec![
                "activity",
                "exit_code",
                "steps",
                "terminal",
                "store_results",
            ];
            if operator {
                names.push("incarnations");
            } else {
                names.extend(["joined_incarnations", "message"]);
            }
            fields(o, &names)?;
            let code = if variant == "trap" {
                Value::Null
            } else {
                json!(17)
            };
            require(
                o["terminal"] == variant && o["exit_code"] == code,
                "failed wait terminal precedence",
            )?;
            steps(
                &o["steps"],
                if operator {
                    json!([step("start", "a"), step("wait", "a")])
                } else {
                    json!([step("start", "a"), step("start", "b"), step("wait", "a")])
                },
            )?;
            let owners = if operator {
                vec!["a/1"]
            } else {
                vec!["a/1", "b/1"]
            };
            require(
                o[if operator {
                    "incarnations"
                } else {
                    "joined_incarnations"
                }] == json!(owners),
                "failed wait owners",
            )?;
            let actual = stores(&o["store_results"], &o["activity"], Some(&owners), operator)?;
            require(
                actual[0]["terminal"] == variant
                    && actual[0]["exit_code"] == code
                    && (operator || actual[0]["message"] == o["message"]),
                "first failure ownership",
            )?;
        }
        LAZY => {
            fields(
                o,
                &[
                    "declared_nodes",
                    "incarnations",
                    "activity",
                    "store_results",
                ],
            )?;
            require(
                o["declared_nodes"] == json!(["a", "b"]) && o["incarnations"] == json!(["a/1"]),
                "lazy dormant owner",
            )?;
            let actual = stores(&o["store_results"], &o["activity"], Some(&["a/1"]), true)?;
            require(
                actual[0]["terminal"] == "exit" && actual[0]["exit_code"] == 0,
                "lazy store terminal",
            )?;
        }
        CLEANUP => cleanup(variant, o)?,
        HOST | INTERNAL => {
            if variant == "cancelled-before-start" {
                fields(o, &["terminal", "steps", "activity", "store_results"])?;
                require(
                    o["terminal"] == "cancelled",
                    "pre-start cancellation classification",
                )?;
                steps(&o["steps"], json!([]))?;
                stores(&o["store_results"], &o["activity"], Some(&[]), true)?;
            } else {
                fields(
                    o,
                    &[
                        "terminal",
                        "exit_code",
                        "watchdog_timeout",
                        "cancelled",
                        "bound_millis",
                        "parent_cancel_after_millis",
                        "joined_incarnations",
                        "all_stores_dropped_before_return",
                        "activity",
                        "store_results",
                    ],
                )?;
                let watchdog = matches!(variant, "watchdog" | "internal deadline");
                require(
                    o["terminal"]
                        == if watchdog {
                            "watchdog_timeout"
                        } else {
                            "cancelled"
                        }
                        && o["exit_code"].is_null()
                        && o["watchdog_timeout"] == watchdog
                        && o["cancelled"] == !watchdog,
                    "host termination classification",
                )?;
                require(
                    o["bound_millis"] == if watchdog { 20 } else { 1000 }
                        && o["parent_cancel_after_millis"]
                            == if watchdog { Value::Null } else { json!(20) },
                    "host termination bounds",
                )?;
                let actual = stores(&o["store_results"], &o["activity"], None, false)?;
                require(
                    integer(&o["joined_incarnations"])? == actual.len() as u64
                        && actual.len() <= 1
                        && o["all_stores_dropped_before_return"] == true,
                    "host termination store leak",
                )?;
            }
        }
        MODEL | ATTRIBUTION => {
            fields(o, &["admission_error"])?;
            let expected = match variant {
                "missing-model" => "cluster requires helper-owned model",
                "missing-namespace" => "cluster requires sealed namespace",
                "unknown-step" => "unknown cluster lifecycle handle",
                "zero-incarnation" | "duplicate-handle" => "invalid or duplicate cluster handle",
                "model-seed" => "cluster model seed mismatch",
                _ => return Err("unknown owner admission".into()),
            };
            require(
                o["admission_error"] == expected,
                "typed owner admission precedence",
            )?;
        }
        CALLBACK => {
            fields(o, &["error_kind", "message", "activity"])?;
            require(
                o["error_kind"] == "unsupported"
                    && o["message"] == "linked cluster forbids transport callbacks",
                "native callback refusal",
            )?;
            activity(&o["activity"], 0, 1)?;
        }
        DEADLINE => {
            fields(
                o,
                &[
                    "sealed_millis",
                    "bounded_millis",
                    "zero_refused",
                    "activity",
                ],
            )?;
            require(
                o["sealed_millis"] == 660000
                    && o["bounded_millis"] == 420000
                    && o["zero_refused"] == true,
                "sealed deadline extension",
            )?;
            activity(&o["activity"], 0, 0)?;
        }
        RESTART | RETRY => {
            let restart = root == RESTART;
            let mut names = vec![
                "refusal",
                "recovery_input_unchanged",
                "storage_before",
                "storage_after_refusal",
                "retry_storage",
                "recovery_image",
                "activity",
                "retry_incarnation",
                "retry_slot",
            ];
            if restart {
                names.extend([
                    "before_incarnation",
                    "before_slot",
                    "after_refusal_incarnation",
                    "after_refusal_slot",
                    "peer_incarnation",
                    "peer_slot",
                    "refused_storage_unchanged",
                ]);
            } else {
                names.extend(["refused_incarnation_owned", "current_after_refusal"]);
            }
            fields(o, &names)?;
            require(
                o["refusal"]
                    == if !restart || variant == "missing-storage-image" {
                        "cluster restart lacks recovery image"
                    } else {
                        "WASI invalid: broker restart lifecycle"
                    },
                "restart refusal classification",
            )?;
            if restart {
                for (key, n) in [
                    ("before_incarnation", 1),
                    ("before_slot", 1),
                    ("after_refusal_incarnation", 1),
                    ("after_refusal_slot", 1),
                    ("peer_incarnation", 1),
                    ("peer_slot", 2),
                ] {
                    require(
                        integer(&o[key])? == n,
                        "restart consumed owner/peer lifecycle",
                    )?;
                }
                require(
                    o["refused_storage_unchanged"] == true,
                    "refused restart mutated storage",
                )?;
            } else {
                require(
                    o["current_after_refusal"] == 1 && o["refused_incarnation_owned"] == false,
                    "refused incarnation retained",
                )?;
            }
            require(
                o["retry_incarnation"] == 2 && o["retry_slot"] == 3,
                "retry consumed incarnation slot",
            )?;
            activity(&o["activity"], 0, 0)?;
            recovery(o)?;
        }
        GRACE => {
            fields(
                o,
                &[
                    "execution_deadline_millis",
                    "cleanup_grace_millis",
                    "elapsed_nanos",
                    "cleanup_error",
                    "acknowledged_stdout",
                    "authored_steps",
                    "joined_incarnations",
                    "all_stores_dropped_before_return",
                    "activity",
                    "store_results",
                ],
            )?;
            require(
                o["execution_deadline_millis"] == 900000
                    && o["cleanup_grace_millis"] == 5000
                    && (5_000_000_000..10_000_000_000).contains(&integer(&o["elapsed_nanos"])?),
                "actual cleanup grace bound",
            )?;
            require(
                o["cleanup_error"] == "actual scenario failure; cluster cleanup deadline exceeded",
                "cleanup masked original failure",
            )?;
            steps(&o["authored_steps"], json!([step("start", "a")]))?;
            require(
                o["joined_incarnations"] == 1
                    && o["acknowledged_stdout"] == json!(b"mark")
                    && o["all_stores_dropped_before_return"] == true,
                "cleanup ownership",
            )?;
            let actual = stores(&o["store_results"], &o["activity"], Some(&["a/1"]), true)?;
            require(
                actual[0]["stdout"] == json!(b"mark"),
                "cleanup actual output",
            )?;
        }
        NETWORK => network(variant, o)?,
        SQLITE | ORACLE => sqlite(root, variant, o, inputs.ok_or("missing SQLite preimages")?)?,
        BOOT => boot(variant, o, inputs.ok_or("missing boot preimages")?)?,
        "TestStorageCrashBoundOnlyByCrashBefore|TestOperatorAcceptsDeclaredStorageCrash" => {
            fields(o, &["accepted", "error"])?;
            let valid = variant == "crash-before";
            require(
                o["accepted"] == valid
                    && o["error"]
                        == if valid {
                            Value::Null
                        } else {
                            json!("undeclared WASM lifecycle boundary")
                        },
                "storage crash boundary",
            )?;
        }
        _ => return Err("unhandled lifecycle root".into()),
    }
    Ok(())
}

fn cleanup(variant: &str, o: &Value) -> Result<(), String> {
    let after = variant == "context-cancel-after-actual-output";
    let mut names = vec![
        "terminal",
        "steps",
        "joined_incarnations",
        "acknowledged_stdout",
        "activity",
        "store_results",
    ];
    if after {
        names.push("store_dropped_before_return");
    } else {
        names.extend([
            "message",
            "acknowledged_handle",
            "all_stores_dropped_before_return",
            "dormant_started",
        ]);
    }
    fields(o, &names)?;
    steps(&o["steps"], json!([step("start", "b"), step("start", "a")]))?;
    require(
        o["joined_incarnations"]
            == if after {
                json!(["a/1", "b/1"])
            } else {
                json!(2)
            },
        "cleanup joined owners",
    )?;
    let actual = stores(
        &o["store_results"],
        &o["activity"],
        Some(&["a/1", "b/1"]),
        true,
    )?;
    require(
        actual[1]["stdout"] == json!(b"mark")
            && o["acknowledged_stdout"] == if after { json!("mark") } else { json!(b"mark") },
        "cleanup actual peer output",
    )?;
    let terminal = match variant {
        "shutdown" => "exit",
        "scenario-error" => "scenario_failure",
        "action-limit" => "capacity",
        _ => "cancelled",
    };
    require(
        o["terminal"] == terminal
            && o[if after {
                "store_dropped_before_return"
            } else {
                "all_stores_dropped_before_return"
            }] == true,
        "cleanup terminal/drop",
    )?;
    if !after {
        require(
            o["acknowledged_handle"] == handle("b") && o["dormant_started"] == false,
            "cleanup dormant acknowledgment",
        )?;
        if variant != "context-cancel" {
            require(
                actual
                    .iter()
                    .all(|r| r["terminal"] == "stop" && r["interrupted"] == true),
                "cleanup store interruption",
            )?;
        }
        if variant == "scenario-error" {
            require(
                o["message"] == "scenario broke",
                "cleanup original scenario failure",
            )?;
        }
        if variant == "action-limit" {
            require(
                o["message"] == "scenario action capacity limit=2 used=2 delta=1",
                "action limit/used/delta",
            )?;
        }
    }
    Ok(())
}

fn network(variant: &str, o: &Value) -> Result<(), String> {
    if variant == "valid" {
        fields(o, &["admitted", "profile", "activity"])?;
        require(
            o["admitted"] == true && o["profile"] == "gomad.simulation-network-stream/v3",
            "network admission",
        )?;
        activity(&o["activity"], 0, 0)
    } else {
        let mut names = vec!["admitted", "admission_error", "launched_stores"];
        if variant == "faults without network" {
            names.push("owner");
        }
        fields(o, &names)?;
        let expected = match variant {
            "node without profile" | "no cluster network" => "cluster boot network profile",
            "node local faults" => "broker node local fault schedule",
            "stock node" => "cluster requires helper-owned model",
            "restarted incarnation" => "invalid cluster incarnation lifecycle",
            "faults without network" => "missing field `model`",
            _ => return Err("unknown network refusal".into()),
        };
        require(
            o["admitted"] == false
                && integer(&o["launched_stores"])? == 0
                && o["admission_error"] == expected,
            "network owner refusal",
        )?;
        if variant == "faults without network" {
            require(
                o["owner"] == "network-config-decoder",
                "network refusal provenance",
            )?;
        }
        Ok(())
    }
}

fn sqlite(root: &str, variant: &str, o: &Value, input: &Value) -> Result<(), String> {
    fields(input, &["ack_bytes", "stdout_bytes"])?;
    let good = b"INTEGRITY ok\nHISTORY 1:committed,2:committed\n";
    let expected: &[u8] = match variant {
        "empty" => b"",
        "missing-history" => b"INTEGRITY ok\n",
        "duplicate-history" => b"INTEGRITY ok\nHISTORY 1:committed\nHISTORY 2:committed\n",
        "duplicate-integrity" => b"INTEGRITY ok\nINTEGRITY ok\nHISTORY 1:committed\n",
        "nonnumeric-event" => b"INTEGRITY ok\nHISTORY x:committed\n",
        "trailing-comma" => b"INTEGRITY ok\nHISTORY 1:committed,\n",
        "duplicate-ack" | "three-native-checks" => good,
        _ => return Err("unknown SQLite case".into()),
    };
    let ack = bytes(&input["ack_bytes"])?;
    let stdout = bytes(&input["stdout_bytes"])?;
    require(
        ack == if variant == "duplicate-ack" {
            b"ACK 2\nACK 2\n".as_slice()
        } else {
            b"ACK 2\n".as_slice()
        } && stdout == expected,
        "SQLite input preimages",
    )?;
    let streams = BTreeMap::from([
        ("cluster/sqlite/2/stdout".into(), ack),
        ("cluster/sqlite/3/stdout".into(), stdout),
    ]);
    let result = crate::qualify::workloads::Workload::SQLite { selection: None }.observe(&streams);
    if root == SQLITE {
        fields(o, &["error"])?;
        let error = result.err().ok_or("SQLite malformed input admitted")?;
        require(o["error"] == error, "SQLite refusal classification")
    } else {
        fields(o, &["schema", "checks"])?;
        let observed = result?;
        require(
            observed.passed() && serde_json::to_value(observed).map_err(|e| e.to_string())? == *o,
            "SQLite native oracle forwarding",
        )
    }
}

fn boot(variant: &str, o: &Value, input: &Value) -> Result<(), String> {
    use crate::qualify::cells::base64;
    fields(input, &["scenario_base64", "network"])?;
    let raw = base64(
        input["scenario_base64"]
            .as_str()
            .ok_or("boot scenario encoding")?,
    )
    .ok_or("boot scenario encoding")?;
    let scenario = crate::cycle::custody::strict_json(&raw)?;
    require(
        scenario["spec"]["nodes"]
            == json!([
        {"id":"a","boot":"main","address":"10.0.0.1","config":null,"volumes":[]},
        {"id":"b","boot":"main","address":"10.0.0.2","config":null,"volumes":[]}])
            && scenario["actions"]
                == json!([step("start","a"),step("start","b"),step("wait","a"),step("wait","b"),
            {"action":"restart","handle":{"node":"a","incarnation":2}},{"action":"wait","handle":{"node":"a","incarnation":2}}]),
        "boot scenario owners/actions",
    )?;
    require(
        scenario["spec"]["limits"]["network_connection_bytes"] == 123
            && scenario["spec"]["limits"]["network_stall_nanos"] == 456,
        "boot scenario network bounds",
    )?;
    let (byte, stall) = match variant {
        "matching" => (123, 456),
        "byte-mismatch" => (124, 456),
        "stall-mismatch" => (123, 457),
        _ => return Err("boot variant".into()),
    };
    let network = json!({"profile":"gomad.simulation-network-stream/v3","faults":[],
        "limits":{"listeners":8,"connections":8,"bytes":1024,"connection_bytes":byte,"stall_nanos":stall,"transitions":128}});
    require(input["network"] == network, "boot network input")?;
    fields(
        o,
        &["configuration", "request_incarnations", "error", "accepted"],
    )?;
    let matching = variant == "matching";
    require(o["accepted"] == matching, "selected boot network admission")?;
    if matching {
        require(
            o["error"].is_null() && o["configuration"].is_object(),
            "matching boot configuration",
        )?;
        for items in [
            &o["configuration"]["incarnations"],
            &o["request_incarnations"],
        ] {
            let items = items.as_array().ok_or("boot incarnations")?;
            let mut owners = items
                .iter()
                .map(|i| i["handle"].clone())
                .collect::<Vec<_>>();
            owners.sort_by_key(|h| {
                (
                    h["node"].as_str().unwrap_or("").to_owned(),
                    h["incarnation"].as_u64().unwrap_or(0),
                )
            });
            require(
                owners
                    == vec![
                        handle("a"),
                        json!({"node":"a","incarnation":2}),
                        handle("b"),
                    ]
                    && items.iter().all(|i| i["network"] == network),
                "boot incarnation model ownership",
            )?;
        }
    } else {
        require(
            o["configuration"].is_null()
                && o["request_incarnations"] == json!([])
                && o["error"] == "WASM boot network limits differ from simulation specification",
            "boot mismatch partial binding",
        )?;
    }
    Ok(())
}
