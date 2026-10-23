#![cfg(feature = "helper-diagnostic-profile")]

use gomad::evidence::canonical;
use serde_json::{Value, json};
use std::{path::Path, process::Command};

fn invoke(input: &Path, output: &Path) -> std::process::Output {
    Command::new(env!("CARGO_BIN_EXE_gomad-profile"))
        .arg(input)
        .arg(output)
        .output()
        .unwrap()
}
fn guest(module: &Path, fuel: u64) -> Value {
    let module_bytes = std::fs::read(module).unwrap();
    json!({"module":module,"module_sha256":canonical::digest(&module_bytes),
        "limits":{"memory_bytes":65536,"fuel":fuel,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
        "model":{"configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/"),
            "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
            "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},
            "arguments":[],"environment":["PWD=/"],"replay_trace":null},
        "namespace":{"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],
            "limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0},
        "storage":null})
}
fn write(path: &Path, value: &Value) {
    std::fs::write(path, serde_json::to_vec(value).unwrap()).unwrap();
}
fn read(path: &Path) -> Value {
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}

fn reject_profile(root: &Path, name: &str, value: &Value, expected: &str) {
    reject_raw(root, name, &serde_json::to_vec(value).unwrap(), expected);
}
fn reject_raw(root: &Path, name: &str, raw: &[u8], expected: &str) {
    let input = root.join(format!("{name}.json"));
    std::fs::write(&input, raw).unwrap();
    let result = Command::new(env!("CARGO_BIN_EXE_gomad-profile"))
        .arg("check")
        .arg(input)
        .output()
        .unwrap();
    assert!(!result.status.success(), "collector accepted {name}");
    assert!(
        String::from_utf8_lossy(&result.stderr).contains(expected),
        "{name}: {}",
        String::from_utf8_lossy(&result.stderr)
    );
}

fn legacy_mutations(root: &Path, profile: &Value) {
    let raw = serde_json::to_string(profile).unwrap();
    gomad::profile::decode(raw.as_bytes()).unwrap();
    for (name, old, new) in [
        ("duplicate-root", "\"abi\":", "\"abi\":{},\"abi\":"),
        (
            "duplicate-nested",
            "\"cpu_samples\":",
            "\"cpu_samples\":0,\"cpu_samples\":",
        ),
    ] {
        let bad = raw.replacen(old, new, 1);
        assert_ne!(bad, raw);
        reject_raw(root, name, bad.as_bytes(), "duplicate diagnostic field");
    }
    let mut oversized = raw.into_bytes();
    oversized.resize(48 * 1024, b' ');
    reject_raw(root, "oversized", &oversized, "report capacity");
    let op = "wasi_snapshot_preview1.clock_time_get";
    let mut bad = profile.clone();
    bad["abi_ops"][op]["count"] = json!(2);
    bad["abi_ops"][op]["served_by"]["clock"] = json!(2);
    reject_profile(
        root,
        "count-and-serving-count",
        &bad,
        "abi_ops differ from legacy buckets: count",
    );
    for name in ["wall_nanos", "instantiate_wall_nanos", "execute_wall_nanos"] {
        let mut bad = profile.clone();
        bad["abi_ops"][op][name] = json!(bad["abi_ops"][op][name].as_u64().unwrap() + 1);
        reject_profile(
            root,
            &format!("op-{name}"),
            &bad,
            "abi_ops differ from legacy buckets",
        );
    }
    for name in ["abi", "abi_excluded", "residual"] {
        let mut bad = profile.clone();
        bad.as_object_mut().unwrap().remove(name);
        reject_profile(root, &format!("missing-{name}"), &bad, "field inventory");
    }
    for name in ["instantiate_wall_nanos", "execute_wall_nanos"] {
        let mut bad = profile.clone();
        bad["residual"][name] = json!(bad["residual"][name].as_u64().unwrap() + 1);
        reject_profile(
            root,
            &format!("residual-{name}"),
            &bad,
            "residual attribution",
        );
    }
    let mut bad = profile.clone();
    bad["abi_excluded"]["count"] = json!(1);
    reject_profile(
        root,
        "excluded-count",
        &bad,
        "abi_ops differ from legacy buckets",
    );
    let mut bad = profile.clone();
    bad["abi_excluded"]["count"] = json!(u64::MAX);
    reject_profile(root, "legacy-count-overflow", &bad, "total overflow");
    let mut bad = profile.clone();
    bad["phases"]["execute"]["cpu_samples"] =
        json!(bad["phases"]["execute"]["count"].as_u64().unwrap() + 1);
    reject_profile(root, "cpu-samples-exceed-count", &bad, "CPU sample claim");
    let mut bad = profile.clone();
    bad["phases"]["execute"]["cpu_samples"] = json!(0);
    bad["phases"]["execute"]["user_nanos"] = json!(1);
    reject_profile(root, "cpu-without-sample", &bad, "CPU sample claim");
    for (group, entry, field) in [
        ("phases", "execute", "cpu_samples"),
        ("abi", "clock", "wall_nanos"),
        ("nested", "encoding", "count"),
    ] {
        let mut bad = profile.clone();
        bad[group][entry].as_object_mut().unwrap().remove(field);
        reject_profile(
            root,
            &format!("missing-{group}-field"),
            &bad,
            "field inventory",
        );
        let mut bad = profile.clone();
        bad[group][entry]["invented"] = json!(0);
        reject_profile(
            root,
            &format!("extra-{group}-field"),
            &bad,
            "field inventory",
        );
    }
    let mut bad = profile.clone();
    bad["abi"]["invented"] = bad["abi"]["clock"].clone();
    reject_profile(root, "unknown-legacy-bucket", &bad, "field inventory");
    let mut bad = profile.clone();
    bad["abi_ops"][op]["invented"] = json!(0);
    reject_profile(root, "extra-op-field", &bad, "field inventory");
}

fn standalone_fixture(root: &Path) {
    let module = root.join("clock.wat");
    std::fs::write(&module, r#"(module
        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start") (drop (call $clock (i32.const 1) (i64.const 0) (i32.const 0)))))"#).unwrap();
    let input = root.join("standalone.json");
    let request = json!({"kind":"standalone","guest":guest(&module, 10000)});
    write(&input, &request);
    let output = root.join("standalone");
    let result = invoke(&input, &output);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let profile = read(&output.join("profile.json"));
    assert_eq!(
        profile["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["count"],
        1
    );
    assert_eq!(
        profile["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["served_by"]["clock"],
        1
    );
    assert_eq!(read(&output.join("outcome.json"))["termination"], "exit");
    gomad::profile::validate(&profile).unwrap();
    legacy_mutations(root, &profile);
    assert!(
        !invoke(&input, &output).status.success(),
        "retained roots must not be overwritten"
    );
    for (name, field) in [
        ("unattributed", "unattributed_imports"),
        ("overlap", "abi_overlap_errors"),
        ("truncated", "threads_omitted"),
        ("report-capacity", "report_rows_dropped"),
    ] {
        let mut bad = profile.clone();
        bad[field] = json!(1);
        let path = root.join(format!("{name}.json"));
        write(&path, &bad);
        let result = Command::new(env!("CARGO_BIN_EXE_gomad-profile"))
            .arg("check")
            .arg(&path)
            .output()
            .unwrap();
        assert!(!result.status.success(), "collector accepted {name}");
    }
    let mut bad = profile.clone();
    bad["reconciliation"]["difference_nanos"] = json!(1);
    assert!(gomad::profile::validate(&bad).is_err());
    let mut bad = profile.clone();
    bad["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["served_by"] = json!({});
    assert!(gomad::profile::validate(&bad).is_err());

    let mut bad = profile.clone();
    bad["owners"].as_object_mut().unwrap().remove("clock");
    assert!(gomad::profile::validate(&bad).is_err());
    let mut bad = profile.clone();
    bad["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["median_nanos"] = json!(u64::MAX);
    assert!(gomad::profile::validate(&bad).is_err());

    let mut bad = profile;
    let unknown = bad["abi_ops"]["wasi_snapshot_preview1.clock_time_get"].clone();
    bad["abi_ops"]["wasi_snapshot_preview1.unknown"] = unknown;
    assert!(gomad::profile::validate(&bad).is_err());
}
fn cluster_fixture(root: &Path) -> Value {
    let module = root.join("clock.wat");
    let handles: Vec<_> = ["a", "b"]
        .into_iter()
        .map(|node| json!({"node":node,"incarnation":1}))
        .collect();
    let incarnations: Vec<_> = handles
        .iter()
        .map(|handle| json!({"handle":handle,"guest":guest(&module,10000),"network":null}))
        .collect();
    let steps: Vec<_> = ["start", "wait"]
        .into_iter()
        .flat_map(|action| {
            handles
                .iter()
                .map(move |handle| json!({"action":action,"handle":handle}))
        })
        .collect();
    let cluster = json!({"kind":"cluster","nodes":[],"broker":{"seed":0,"pause_on_settle":false,
        "limits":{"max_requests":100,"max_events":100,"max_queued_events":100,"max_transitions":1000,"max_payload_bytes":4096,"max_string_bytes":1024}},
        "network":null,"incarnations":incarnations,"steps":steps,"boundaries":[],"crash_groups":[],"wall_millis":5000,"evidence_bytes":1<<20});
    let input = root.join("cluster.json");
    write(&input, &cluster);
    let output = root.join("cluster");
    let result = invoke(&input, &output);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let profile = read(&output.join("profile.json"));
    gomad::profile::validate(&profile).unwrap();
    assert_eq!(profile["threads"].as_array().unwrap().len(), 2);
    assert_eq!(
        profile["abi_ops"]["wasi_snapshot_preview1.clock_time_get"]["count"],
        2
    );
    assert!(
        profile["wall_basis"]
            .as_str()
            .unwrap()
            .contains("thread-time")
    );

    let mut bad = profile.clone();
    let duplicate = bad["threads"][0].clone();
    bad["threads"].as_array_mut().unwrap().push(duplicate);
    assert!(gomad::profile::validate(&bad).is_err());
    let mut bad = profile;
    bad["threads"][0]["difference_nanos"] = json!(1);
    assert!(gomad::profile::validate(&bad).is_err());

    cluster
}
fn standalone_cutoffs(root: &Path) {
    let module = root.join("fuel.wat");
    std::fs::write(&module, "(module (memory (export \"memory\") 1) (func (export \"_start\") (loop $loop (br $loop))))").unwrap();
    let input = root.join("fuel.json");
    write(
        &input,
        &json!({"kind":"standalone","guest":guest(&module,100)}),
    );
    let output = root.join("fuel");
    let result = invoke(&input, &output);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert_eq!(
        read(&output.join("outcome.json"))["termination"],
        if cfg!(feature = "helper-diagnostic-fuel-off") {
            "watchdog_timeout"
        } else {
            "capacity"
        }
    );
    gomad::profile::validate(&read(&output.join("profile.json"))).unwrap();
    let mut timeout = guest(&module, 1_000_000_000_000);
    timeout["limits"]["wall_millis"] = json!(100);
    let input = root.join("timeout.json");
    write(&input, &json!({"kind":"standalone","guest":timeout}));
    let output = root.join("timeout");
    let result = invoke(&input, &output);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert_eq!(
        read(&output.join("outcome.json"))["termination"],
        "watchdog_timeout"
    );
    gomad::profile::validate(&read(&output.join("profile.json"))).unwrap();
}
fn cluster_cutoff(root: &Path, cluster: Value) {
    let visible = root.join("visible.wat");
    let spinning = root.join("visible-spin.wat");
    for (path, tail) in [(&visible, ""), (&spinning, "(loop $loop (br $loop))")] {
        std::fs::write(
            path,
            format!(
                r#"(module
            (import "gomad_wasm_v2" "net_stream" (func $network (param i32 i32) (result i32)))
            (memory (export "memory") 1)
            (data (i32.const 0) "\00\00\00\02")
            (func (export "_start")
                (if (call $network (i32.const 0) (i32.const 64)) (then unreachable)) {tail}))"#
            ),
        )
        .unwrap();
    }
    let network = json!({"model":{"profile":"gomad.simulation-network-stream/v3","limits":{
        "listeners":8,"connections":8,"bytes":1_048_576,"connection_bytes":1_048_576,
        "stall_nanos":10_000_000_000u64,"transitions":10000}},
        "addresses":{},"links":[],"strict_topology":false,"faults":[]});
    let mut timeout_cluster = cluster;
    timeout_cluster["network"] = network.clone();
    timeout_cluster["wall_millis"] = json!(1000);
    timeout_cluster["incarnations"][0]["guest"] = guest(&visible, 1_000_000_000_000);
    timeout_cluster["incarnations"][1]["guest"] = guest(&spinning, 1_000_000_000_000);
    for incarnation in timeout_cluster["incarnations"].as_array_mut().unwrap() {
        incarnation["network"] = network["model"].clone();
    }
    timeout_cluster["steps"] = json!([
        {"action":"start","handle":{"node":"a","incarnation":1}},
        {"action":"wait","handle":{"node":"a","incarnation":1}},
        {"action":"start","handle":{"node":"b","incarnation":1}},
        {"action":"wait","handle":{"node":"b","incarnation":1}}
    ]);
    let input = root.join("cluster-timeout.json");
    write(&input, &timeout_cluster);
    let output = root.join("cluster-timeout");
    let result = invoke(&input, &output);
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    assert_eq!(
        read(&output.join("outcome.json"))["termination"],
        "watchdog_timeout"
    );
    assert_eq!(
        read(&output.join("incarnation-0/outcome.json"))["termination"],
        "exit"
    );
    let world = std::fs::read(output.join("world")).unwrap();
    assert!(world.len() >= 200);
    assert_eq!(&world[..12], b"GMWSNAP1\0\0\0\x01");
    assert!(
        u64::from_be_bytes(world[88..96].try_into().unwrap()) > 1,
        "watchdog lost nontrivial World history"
    );
    assert_eq!(
        read(&output.join("streams.json"))["world"]["sha256"],
        canonical::digest(&world)
    );
    gomad::profile::validate(&read(&output.join("profile.json"))).unwrap();
}

#[test]
fn linked_profile_cli_corpus() {
    let qualification = Command::new(env!("CARGO_BIN_EXE_gomad"))
        .arg("qualify")
        .output()
        .unwrap();
    assert_eq!(qualification.status.code(), Some(2));
    assert!(String::from_utf8_lossy(&qualification.stderr).contains("qualification is refused"));
    let root = std::env::temp_dir().join(format!("gomad-profile-corpus-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    standalone_fixture(&root);
    let cluster = cluster_fixture(&root);
    standalone_cutoffs(&root);
    cluster_cutoff(&root, cluster);
    std::fs::remove_dir_all(&root).unwrap();
}
