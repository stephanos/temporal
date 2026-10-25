use super::*;

fn store() -> Value {
    json!({"handle":{"node":"a","incarnation":1},"terminal":"exit","exit_code":0,"message":"",
        "authorization_to_terminal_nanos":1,"interrupted":false,"stdout":b"mark"})
}

#[test]
fn store_owner_drop_activity_and_authored_order_refusals() {
    let value = json!([store()]);
    let owner = json!({"launched_stores":1,"remote_callback_attempts":0});
    stores(&value, &owner, Some(&["a/1"]), true).unwrap();
    for (key, bad) in [
        ("authorization_to_terminal_nanos", Value::Null),
        ("authorization_to_terminal_nanos", json!(0)),
        ("exit_code", json!(u64::MAX)),
        ("interrupted", json!(1)),
        ("stdout", json!([256])),
    ] {
        let mut changed = value.clone();
        changed[0][key] = bad;
        assert!(
            stores(&changed, &owner, Some(&["a/1"]), true).is_err(),
            "{key}"
        );
    }
    assert!(stores(&json!([store(), store()]), &owner, None, true).is_err());
    assert!(
        stores(
            &value,
            &json!({"launched_stores":0,"remote_callback_attempts":0}),
            None,
            true
        )
        .is_err()
    );
    assert!(
        stores(
            &value,
            &json!({"launched_stores":1,"remote_callback_attempts":1}),
            None,
            true
        )
        .is_err()
    );
    let declared = json!([step("start", "a"), step("wait", "a")]);
    steps(&declared, declared.clone()).unwrap();
    assert!(steps(&json!([step("wait", "a"), step("start", "a")]), declared).is_err());
}

#[test]
fn lifecycle_catalog_and_parser_fail_closed() {
    let required = BTreeSet::from([(LAZY.into(), "native-lazy-store".into())]);
    assert!(observations(b"", &required).is_err());
    let row = json!({"schema":SCHEMA,"original_root":LAZY,"variant":"native-lazy-store","observed":{
        "declared_nodes":["a","b"],"incarnations":["a/1"],"activity":{"launched_stores":1,"remote_callback_attempts":0},
        "store_results":[store()]}});
    let raw = row.to_string();
    observations(raw.as_bytes(), &required).unwrap();
    assert!(observations(format!("{raw}\n{raw}").as_bytes(), &required).is_err());
    let mut changed = row;
    changed["variant"] = json!("foreign");
    assert!(observations(changed.to_string().as_bytes(), &required).is_err());
    assert!(
        observations(
            b"{\"schema\":\"gomad.native-lifecycle-predicate/v1\",\"schema\":\"x\"}",
            &required
        )
        .is_err()
    );
}

#[test]
fn lifecycle_wire_uses_inspection_owner_parser() {
    let handles = BTreeSet::from([Handle {
        node: "a".into(),
        incarnation: 1,
    }]);
    let mut raw = b"GMWIL\x01\0\0".to_vec();
    raw.push(1);
    raw.extend_from_slice(&1u32.to_le_bytes());
    raw.push(b'a');
    raw.extend_from_slice(&1u64.to_le_bytes());
    let admitted = linked_inspection::lifecycle_records(&raw, &handles).unwrap();
    assert_eq!(admitted.values().next().unwrap().1, 1);
    for end in 0..raw.len() {
        if end == 8 {
            continue;
        }
        assert!(linked_inspection::lifecycle_records(&raw[..end], &handles).is_err());
    }
    for (at, byte) in [(0, b'X'), (8, 0), (13, b'b')] {
        let mut changed = raw.clone();
        changed[at] = byte;
        assert!(linked_inspection::lifecycle_records(&changed, &handles).is_err());
    }
}

#[test]
#[ignore = "requires independently captured genuine lifecycle harness stdout"]
fn receive_genuine_lifecycle_rows_and_targeted_mutations() {
    let path = std::env::var("GOMAD_B06_LIFECYCLE_LOG").expect("GOMAD_B06_LIFECYCLE_LOG");
    let raw = std::fs::read(path).unwrap();
    observations(&raw, &catalog()).unwrap();
    for line in std::str::from_utf8(&raw)
        .unwrap()
        .lines()
        .filter(|line| line.contains(SCHEMA))
    {
        let row =
            crate::cycle::custody::strict_json(line[line.find('{').unwrap()..].as_bytes()).unwrap();
        let root = if row.get("original_roots").is_some() {
            STORAGE.join("|")
        } else {
            row["original_root"].as_str().unwrap().into()
        };
        let variant = row["variant"].as_str().unwrap();
        cases::validate(&root, variant, &row["observed"], row.get("inputs")).unwrap();
        let o = row["observed"].as_object().unwrap();
        for key in o.keys() {
            let mut changed = row["observed"].clone();
            changed.as_object_mut().unwrap().remove(key);
            assert!(
                cases::validate(&root, variant, &changed, row.get("inputs")).is_err(),
                "{root}/{variant}/{key}"
            );
        }
    }
}

fn predicate_mutants(root: &str, variant: &str, o: &Value) -> Vec<(Value, &'static str)> {
    let mut mutants = Vec::new();
    let mut add = |key: &str, value: Value, reason| {
        let mut changed = o.clone();
        changed[key] = value;
        mutants.push((changed, reason));
    };
    if o["activity"]["launched_stores"]
        .as_u64()
        .is_some_and(|n| n > 0)
    {
        let mut activity = o["activity"].clone();
        activity["launched_stores"] = json!(0);
        add("activity", activity, "native activity ownership");
    }
    match root {
        MODEL if variant.starts_with("memory-capacity-") => {
            add("memory_bytes", json!(65537), "capacity actual resources");
            let mut model = o["local_model"].clone();
            model["calls"] = json!(999);
            add("local_model", model, "interrupted model lifetime/calls");
            let mut request = o["request"].clone();
            request["limits"]["fuel"] = json!(999999);
            add("request", request, "capacity request budget");
        }
        DRAIN if variant == "held-peer-release" => {
            add(
                "after_control_attempts",
                json!(5),
                "post-failure native control attempt",
            );
            add(
                "probe_elapsed_nanos",
                json!(49_999_999),
                "drain release/join bounds",
            );
            add(
                "release_to_return_nanos",
                json!(5_000_000_000u64),
                "drain release/join bounds",
            );
            add(
                "message",
                json!("substituted failure"),
                "drain terminal ownership",
            );
        }
        OPERATOR | DRAIN => {
            add(
                "terminal",
                json!("substituted"),
                "failed wait terminal precedence",
            );
            add(
                "steps",
                json!([step("wait", "a"), step("start", "a")]),
                "authored lifecycle action order",
            );
        }
        LAZY => {
            add("incarnations", json!(["a/1", "b/1"]), "lazy dormant owner");
        }
        CLEANUP => {
            add("terminal", json!("substituted"), "cleanup terminal/drop");
            let field = if variant == "context-cancel-after-actual-output" {
                "store_dropped_before_return"
            } else {
                "all_stores_dropped_before_return"
            };
            add(field, json!(false), "cleanup terminal/drop");
            if variant == "action-limit" {
                add(
                    "message",
                    json!("scenario action capacity limit=2 used=1 delta=1"),
                    "action limit/used/delta",
                );
            }
        }
        HOST | INTERNAL if variant != "cancelled-before-start" => {
            add(
                "cancelled",
                json!(o["cancelled"] != true),
                "host termination classification",
            );
            add("bound_millis", json!(1), "host termination bounds");
            add(
                "all_stores_dropped_before_return",
                json!(false),
                "host termination store leak",
            );
        }
        HOST => {
            add(
                "steps",
                json!([step("start", "a")]),
                "authored lifecycle action order",
            );
        }
        MODEL | ATTRIBUTION => {
            add(
                "admission_error",
                json!("substituted"),
                "typed owner admission precedence",
            );
        }
        CALLBACK => {
            add("error_kind", json!("capacity"), "native callback refusal");
            add(
                "activity",
                json!({"launched_stores":1,"remote_callback_attempts":1}),
                "native activity ownership",
            );
        }
        DEADLINE => {
            add("bounded_millis", json!(660001), "sealed deadline extension");
        }
        RESTART | RETRY => {
            add("retry_slot", json!(4), "retry consumed incarnation slot");
            add(
                "recovery_input_unchanged",
                json!(false),
                "restart recovery preimages",
            );
            let mut after = o["storage_after_refusal"].clone();
            after["context"] = json!([1]);
            add(
                "storage_after_refusal",
                after,
                "refused restart changed storage",
            );
        }
        GRACE => {
            add(
                "elapsed_nanos",
                json!(4_999_999_999u64),
                "actual cleanup grace bound",
            );
            add(
                "elapsed_nanos",
                json!(10_000_000_000u64),
                "actual cleanup grace bound",
            );
            add(
                "cleanup_error",
                json!("cluster cleanup deadline exceeded"),
                "cleanup masked original failure",
            );
        }
        NETWORK if variant == "valid" => {
            add("profile", json!("foreign"), "network admission");
        }
        NETWORK => {
            add("admitted", json!(true), "network owner refusal");
        }
        SQLITE => {
            add(
                "error",
                json!("substituted"),
                "SQLite refusal classification",
            );
        }
        ORACLE => {
            let mut checks = o["checks"].clone();
            checks["acknowledged-durable-recovery.events"] = json!(false);
            add("checks", checks, "SQLite native oracle forwarding");
        }
        BOOT => {
            add(
                "accepted",
                json!(variant != "matching"),
                "selected boot network admission",
            );
        }
        "TestStorageCrashBoundOnlyByCrashBefore|TestOperatorAcceptsDeclaredStorageCrash" => {
            add(
                "accepted",
                json!(variant != "crash-before"),
                "storage crash boundary",
            );
        }
        _ => panic!("missing direct lifecycle mutant for {root}/{variant}"),
    }
    mutants
}

#[test]
#[ignore = "requires independently captured genuine lifecycle harness stdout"]
fn genuine_lifecycle_direct_predicate_mutations() {
    let raw =
        std::fs::read(std::env::var("GOMAD_B06_LIFECYCLE_LOG").expect("GOMAD_B06_LIFECYCLE_LOG"))
            .unwrap();
    observations(&raw, &catalog()).unwrap();
    let mut caught = 0;
    for line in std::str::from_utf8(&raw)
        .unwrap()
        .lines()
        .filter(|line| line.contains(SCHEMA))
    {
        let row =
            crate::cycle::custody::strict_json(line[line.find('{').unwrap()..].as_bytes()).unwrap();
        let root = if row.get("original_roots").is_some() {
            STORAGE.join("|")
        } else {
            row["original_root"].as_str().unwrap().into()
        };
        let variant = row["variant"].as_str().unwrap();
        for (changed, predicate) in predicate_mutants(&root, variant, &row["observed"]) {
            println!(
                "B06_DIRECT_MUTANT root={root} variant={variant} input_sha256={} predicate={predicate}",
                sha(&serde_json::to_vec(&changed).unwrap())
            );
            let error = cases::validate(&root, variant, &changed, row.get("inputs")).unwrap_err();
            assert_eq!(error, predicate, "{root}/{variant}");
            caught += 1;
        }
    }
    assert!(caught >= catalog().len());
}
