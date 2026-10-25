//! Strict ITF admission at the source typed-value boundary.
#![forbid(unsafe_code)]

use std::collections::BTreeMap;
use umpire_apalache::decode_itf;
use umpire_tla::{
    Bounded, Type,
    values::{Schema, Value},
};

fn ticks(low: i128) -> Type {
    Type::Optional(Bounded {
        name: "Ticks".into(),
        field: None,
        backing: "u8".into(),
        min: 0,
        max: 255,
        low,
        high: 3,
    })
}

#[test]
fn typed_itf_checks_declarations_shapes_and_limits() -> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema {
        fields: BTreeMap::from([("x".into(), Type::Int { min: -2, max: 2 })]),
        observers: BTreeMap::from([("tick".into(), Type::Bool)]),
    };
    let params = BTreeMap::from([("world".into(), Type::Bool)]);
    let raw = r##"{"#meta":{"format":"ITF"},"vars":["x","tick"],"params":["world"],"states":[{"#meta":{"index":0},"x":{"#bigint":"2"},"tick":false,"world":true}]}"##;
    let trace = decode_itf(raw, &schema, &params, 0)?;
    assert_eq!(trace.states[0].state["x"], Value::Int(2));
    assert_eq!(trace.parameters["world"], Value::Bool(true));
    for (from, to) in [
        ("\"ITF\"", "\"other\""),
        ("\"format\":\"ITF\"", "\"format\":\"ITF\",\"version\":99"),
        ("\"x\",\"tick\"", "\"x\",\"x\",\"tick\""),
        ("\"params\":[\"world\"]", "\"params\":[]"),
        ("\"index\":0", "\"index\":1"),
        ("\"x\":{\"#bigint\":\"2\"}", "\"x\":3"),
        ("\"x\":{\"#bigint\":\"2\"}", "\"x\":2"),
        ("\"#bigint\":\"2\"", "\"#bigint\":\"170141183460469231731687303715884105728\""),
        ("\"#bigint\":\"2\"", "\"#bigint\":\"2\",\"extra\":0"),
        ("\"#bigint\":\"2\"", "\"#bigint\":\"3\""),
        ("\"#bigint\":\"2\"", "\"#bigint\":\"-3\""),
        ("\"#bigint\":\"2\"", "\"#bigint\":2"),
        ("\"tick\":false,", ""),
        ("\"tick\":false", "\"tick\":false,\"tick\":false"),
        ("\"tick\":false", "\"tick\":false,\"__umpire_unknown\":false"),
        ("\"world\":true", "\"world\":0"),
        (",\"world\":true", ""),
        ("\"index\":0", "\"index\":\"0\""),
        ("\"states\":", "\"unknown\":0,\"states\":"),
    ] {
        let bad = raw.replace(from, to);
        assert_ne!(bad, raw, "ineffective tamper {from}");
        assert!(decode_itf(&bad, &schema, &params, 0).is_err(), "accepted {bad}");
    }
    assert!(decode_itf(&" ".repeat(4 * 1024 * 1024 + 1), &schema, &params, 0).is_err());
    assert!(
        decode_itf(&format!("{}0{}", "[".repeat(70), "]".repeat(70)), &schema, &params, 0).is_err()
    );
    Ok(())
}

#[test]
fn typed_itf_preserves_composites_and_refuses_retired_set_shapes()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema {
        fields: BTreeMap::from([(
            "x".into(),
            Type::Tuple(vec![
                Type::Enum { name: "Mode".into(), variants: vec!["Mode::A".into()] },
                Type::Unit,
                Type::Seq(Box::new(Type::Bool)),
            ]),
        )]),
        observers: BTreeMap::new(),
    };
    let raw = r##"{"#meta":{"format":"ITF"},"vars":["x"],"states":[{"#meta":{"index":0},"x":{"#tup":["Mode::A",[],[true,false]]}}]}"##;
    let decoded = decode_itf(raw, &schema, &BTreeMap::new(), 0)?;
    assert!(matches!(&decoded.states[0].state["x"], Value::Seq(values) if values.len()==3));
    for bad in [
        raw.replace(",[],", ",[true],"),
        raw.replace(",[],", ",{\"#set\":[]},"),
        raw.replace("[true,false]", "{\"#set\":[true,false]}"),
        raw.replace("\"Mode::A\"", "\"Mode::B\""),
        raw.replace("\"#tup\"", "\"#set\""),
        raw.replace("[true,false]", "[true,0]"),
    ] {
        assert!(decode_itf(&bad, &schema, &BTreeMap::new(), 0).is_err(), "accepted {bad}");
    }
    Ok(())
}

#[test]
fn declared_wire_types_must_match_the_source_schema() -> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema {
        fields: BTreeMap::from([("x".into(), Type::Int { min: 0, max: 2 })]),
        observers: BTreeMap::new(),
    };
    let raw = r##"{"#meta":{"format":"ITF","varTypes":{"x":"Int"}},"vars":["x"],"states":[{"#meta":{"index":0},"x":{"#bigint":"2"}}]}"##;
    decode_itf(raw, &schema, &BTreeMap::new(), 0)?;
    assert!(decode_itf(&raw.replace("\"Int\"", "\"Bool\""), &schema, &BTreeMap::new(), 0).is_err());
    Ok(())
}

#[test]
fn observer_only_safety_assignment_keeps_strict_schema_empty_state_refusal()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema {
        fields: BTreeMap::new(),
        observers: BTreeMap::from([("flag".into(), Type::Bool)]),
    };
    assert!(schema.validate().is_err());
    let raw = r##"{"#meta":{"format":"ITF"},"vars":["flag"],"states":[{"#meta":{"index":0},"flag":true}]}"##;
    assert_eq!(decode_itf(raw, &schema, &BTreeMap::new(), 0)?.states[0].state, BTreeMap::new());
    Ok(())
}

#[test]
fn optional_integer_itf_uses_the_exact_declared_code_set() -> Result<(), Box<dyn std::error::Error>>
{
    let mut schema = Schema {
        fields: BTreeMap::from([("x".into(), ticks(1))]),
        observers: BTreeMap::from([("before".into(), ticks(1))]),
    };
    for code in [-1, 0, 1, 2, 3, 4, 5] {
        let raw = format!(
            r##"{{"#meta":{{"format":"ITF","varTypes":{{"x":"Int","before":"Int"}}}},"vars":["x","before"],"states":[{{"#meta":{{"index":0}},"x":{{"#bigint":"{code}"}},"before":{{"#bigint":"{code}"}}}}]}}"##
        );
        let decoded = decode_itf(&raw, &schema, &BTreeMap::new(), 0);
        assert_eq!(decoded.is_ok(), [0, 2, 3, 4].contains(&code), "code {code}");
    }
    schema.fields.insert("x".into(), ticks(0));
    let raw = r##"{"#meta":{"format":"ITF"},"vars":["x","before"],"states":[{"#meta":{"index":0},"x":{"#bigint":"1"},"before":{"#bigint":"2"}}]}"##;
    assert_eq!(decode_itf(raw, &schema, &BTreeMap::new(), 0)?.states[0].state["x"], Value::Int(1));
    for value in ["true", "\"1\"", "[]", "{\"#bigint\":\"1\",\"extra\":0}"] {
        assert!(
            decode_itf(&raw.replace("{\"#bigint\":\"1\"}", value), &schema, &BTreeMap::new(), 0)
                .is_err()
        );
    }
    Ok(())
}
