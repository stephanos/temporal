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
