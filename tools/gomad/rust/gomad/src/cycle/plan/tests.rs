use super::*;
use crate::cycle::model::{Mode, PlanConfig};
use serde_json::{Value, json};
use std::path::PathBuf;

fn config() -> PlanConfig {
    PlanConfig {
        binary: PathBuf::from("/sealed/release"),
        config_path: PathBuf::from("/sealed/driver.json"),
        config_sha256: "f".repeat(64),
    }
}

fn oracle() -> Value {
    let bytes = include_bytes!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../../../.flow/artifacts/fn242-stage-table.json"
    ));
    assert_eq!(
        crate::evidence::canonical::digest(bytes),
        "03adbaca3601a01b1be71a9e80e0a155d99e543fa3b6d7e8411fd83c33afed79"
    );
    let value: Value = crate::qualify::spec::strict(bytes).unwrap();
    assert_eq!(value["schema"], "gomad.fn242-stage-inventory/v1");
    assert_eq!(
        value["source_sha256"]["tools/gomad/rust/driver/driver.py"],
        "153c0a969c57bfe983c2b22412c56c88618afc1eb5b00ed9e795ea6fa64c78de"
    );
    assert_eq!(
        crate::evidence::canonical::digest(value["raw_dag_tsv"].as_str().unwrap().as_bytes()),
        value["source_sha256"][".flow/artifacts/wasm-qualification-driver/parallel/dag.tsv"]
    );
    assert_eq!(
        crate::evidence::canonical::digest(
            value["raw_semantic_map_json"].as_str().unwrap().as_bytes()
        ),
        value["source_sha256"]["tools/gomad/rust/driver/semantic-map.json"]
    );
    value
}

#[test]
fn original_96_stage_contract_and_mapped_argv() {
    let expected = oracle();
    let mut rebound = config();
    rebound.binary = "/candidate/bin/gomad".into();
    rebound.config_path = "/candidate/frozen/config.json".into();
    rebound.config_sha256 = "a".repeat(64);
    for cfg in [config(), rebound] {
        for mode in [Mode::Parallel, Mode::Serial] {
            let actual = derive(&cfg, mode).unwrap();
            assert_eq!(actual.stages.len(), 96);
            for (stage, original) in actual
                .stages
                .iter()
                .zip(expected["stages"].as_array().unwrap())
            {
                assert_eq!(stage.contract.label, original["label"]);
                let value = serde_json::to_value(&stage.contract).unwrap();
                for key in [
                    "cores",
                    "mem_gib",
                    "estimated_seconds",
                    "locks",
                    "scratch",
                    "flags",
                    "tier",
                    "dependencies",
                    "isolation_class",
                ] {
                    assert_eq!(value[key], original[key], "{} {key}", stage.contract.label);
                }
                assert_eq!(value["repetitions"], original["original"]["repetitions"]);
                assert_eq!(
                    value["required_subgates"],
                    original["original"]["required_subgates"]
                );
                assert_eq!(value["adapter"], original["adapter"]);
                for key in [
                    "primary_timeout_s",
                    "static_timeout_s",
                    "command_default_timeout_s",
                ] {
                    assert_eq!(
                        value["bounds"][key], original["bounds"][key],
                        "{} {key}",
                        stage.contract.label
                    );
                }
                if stage.contract.label.starts_with("runtime-")
                    && stage.contract.label != "runtime-aggregate"
                {
                    for key in ["instances", "replays", "seed", "shard_wall_seconds"] {
                        assert_eq!(value["bounds"]["runtime"][key], original["bounds"][key]);
                    }
                }
                let old = original["argv"].as_array().unwrap();
                assert_eq!(
                    &old[..2],
                    &[
                        json!("/usr/bin/python3"),
                        json!(
                            "/home/agent/wt/fn242-obligation-inventory/tools/gomad/rust/driver/driver.py"
                        )
                    ]
                );
                let mut mapped = vec![
                    json!(cfg.binary),
                    json!("qualification-cycle"),
                    json!("internal"),
                    json!("driver"),
                ];
                let mut tail = old[2..].to_vec();
                assert_eq!(tail[2], json!("/sealed/driver.json"));
                assert_eq!(tail[4], json!("f".repeat(64)));
                tail[2] = json!(cfg.config_path);
                tail[4] = json!(cfg.config_sha256);
                mapped.extend(tail);
                assert_eq!(serde_json::to_value(&stage.argv).unwrap(), json!(mapped));
                assert_eq!(
                    serde_json::to_value(&stage.dependencies).unwrap(),
                    if mode == Mode::Serial {
                        original["serial"]["dependencies"].clone()
                    } else {
                        original["dependencies"].clone()
                    }
                );
                assert_eq!(
                    serde_json::to_value(stage.isolation_class).unwrap(),
                    if mode == Mode::Serial {
                        json!("solo")
                    } else {
                        original["isolation_class"].clone()
                    }
                );
            }
        }
    }
}

#[test]
fn serial_only_changes_predecessors_and_isolation() {
    let parallel = derive(&config(), Mode::Parallel).unwrap();
    let serial = derive(&config(), Mode::Serial).unwrap();
    for (left, right) in parallel.stages.iter().zip(&serial.stages) {
        assert_eq!(left.contract, right.contract);
        assert_eq!(left.argv, right.argv);
    }
}

#[test]
fn rust_driver_83_isolated_refusal() {
    let stages = contracts();
    let semantic = semantics();
    validate_mapping(&stages, &semantic).unwrap();
    println!("orchestration:driver:83 control=accepted");
    let mut duplicate = semantic.clone();
    let first = duplicate[0].label.clone();
    duplicate[1].label = first;
    assert_eq!(
        validate_mapping(&stages, &duplicate).unwrap_err(),
        "missing/duplicate semantic stage"
    );
    assert_eq!(
        validate_mapping(&stages, &semantic[..95]).unwrap_err(),
        "missing/duplicate semantic stage"
    );
    println!(
        "orchestration:driver:83 mutation=missing-or-duplicate-semantic-stage outcome=refused"
    );
}

#[test]
fn rust_driver_84_isolated_refusal() {
    let stages = contracts();
    let semantic = semantics();
    validate_mapping(&stages, &semantic).unwrap();
    println!("orchestration:driver:84 control=accepted");
    let mut foreign = semantic;
    foreign[0].label = "undeclared".into();
    assert_eq!(
        validate_mapping(&stages, &foreign).unwrap_err(),
        "semantic stage/DAG mismatch"
    );
    println!("orchestration:driver:84 mutation=foreign-semantic-label outcome=refused");
}

#[test]
fn rust_driver_89_isolated_refusal() {
    assert_eq!("serial".parse::<Mode>().unwrap(), Mode::Serial);
    assert_eq!("parallel".parse::<Mode>().unwrap(), Mode::Parallel);
    println!("orchestration:driver:89 control=serial-and-parallel-accepted");
    assert_eq!("other".parse::<Mode>().unwrap_err(), "unknown mode");
    println!("orchestration:driver:89 mutation=unknown-mode outcome=refused");
}

#[test]
fn changed_plan_contract_is_refused() {
    let cfg = config();
    let control = derive(&cfg, Mode::Parallel).unwrap();
    validate(&cfg, &control).unwrap();
    for field in [
        "argument",
        "label",
        "resource",
        "bound",
        "dependency",
        "repetition",
        "substitution",
        "lock",
        "scratch",
        "flag",
        "isolation",
    ] {
        let mut changed = control.clone();
        let stage = &mut changed.stages[0];
        match field {
            "argument" => stage.argv.push("--changed-stage-argument".into()),
            "label" => stage.contract.label = "changed".into(),
            "resource" => stage.contract.cores += 1.0,
            "bound" => stage.contract.bounds.static_timeout_s = Some(1),
            "dependency" => stage.dependencies.push("portable".into()),
            "repetition" => stage.contract.repetitions += 1,
            "substitution" => stage.argv[3] = "unmapped-driver".into(),
            "lock" => stage.contract.locks.push("new-lock".into()),
            "scratch" => stage.contract.scratch.push("new-scratch".into()),
            "flag" => stage.contract.flags.push(crate::cycle::model::Flag::Heavy),
            "isolation" => stage.isolation_class = crate::cycle::model::Isolation::Solo,
            _ => unreachable!(),
        }
        assert!(validate(&cfg, &changed).is_err(), "{field}");
    }
}

#[test]
fn strict_plan_json_refuses_duplicate_unknown_and_invalid_values() {
    let cfg = config();
    let control = derive(&cfg, Mode::Parallel).unwrap();
    let encoded = serde_json::to_vec(&control).unwrap();
    assert_eq!(decode(&cfg, &encoded).unwrap(), control);
    let raw = String::from_utf8(encoded).unwrap();
    for (name, original, replacement) in [
        ("duplicate", r#""mode":"#, r#""mode":"parallel","mode":"#),
        (
            "escaped duplicate",
            r#""mode":"#,
            r#""mode":"parallel","m\u006fde":"#,
        ),
        (
            "nested duplicate",
            r#""label":"#,
            r#""label":"validate","label":"#,
        ),
        ("unknown field", r#""mode":"#, r#""unknown":true,"mode":"#),
        (
            "unknown class",
            r#""isolation_class":"normal""#,
            r#""isolation_class":"unmapped""#,
        ),
        ("unknown flag", r#""flags":[]"#, r#""flags":["unknown"]"#),
        ("nonfinite", r#""cores":2.0"#, r#""cores":NaN"#),
        ("overflow", r#""cores":2.0"#, r#""cores":1e9999"#),
        ("boolean resource", r#""cores":2.0"#, r#""cores":true"#),
        ("negative resource", r#""cores":2.0"#, r#""cores":-1.0"#),
        (
            "negative duration",
            r#""static_timeout_s":3600"#,
            r#""static_timeout_s":-1"#,
        ),
    ] {
        let mutation = raw.replacen(original, replacement, 1);
        assert_ne!(mutation, raw, "{name}");
        assert!(decode(&cfg, mutation.as_bytes()).is_err(), "{name}");
    }
    assert!(
        decode(&cfg, format!("{raw} true").as_bytes())
            .unwrap_err()
            .contains("trailing")
    );
    assert!(decode(&cfg, b"{").is_err());
    assert!(
        decode(&cfg, &vec![b' '; (1 << 20) + 1])
            .unwrap_err()
            .contains("byte bound")
    );
}
