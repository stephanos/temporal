use gomad::{
    evidence::canonical,
    measurement::evidence::{self, ComparisonMask, ReadLimits},
};
use serde_json::{Value, json};
use std::{
    fs::{File, OpenOptions},
    os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt},
    path::{Path, PathBuf},
    process::{Command, Output, Stdio},
    sync::atomic::{AtomicU64, Ordering},
    time::{Duration, Instant},
};

static NEXT: AtomicU64 = AtomicU64::new(0);
struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-measurement-public-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&path)
            .unwrap();
        Self(path)
    }
    fn write(&self, name: &str, bytes: &[u8]) -> PathBuf {
        let path = self.0.join(name);
        std::fs::write(&path, bytes).unwrap();
        path
    }
    fn request(&self, name: &str, module: &str, sampling: bool) -> PathBuf {
        let module = self.write(&format!("{name}.wat"), module.as_bytes());
        let mut guest = json!({"module":module,"module_sha256":canonical::digest(&std::fs::read(&module).unwrap()),
            "limits":{"memory_bytes":65536,"fuel":500_000_000,"wall_millis":5000,"output_bytes":1024,"evidence_bytes":1<<20},
            "model":null,"namespace":null,"storage":null});
        if sampling {
            guest["guest_profile"] = json!({"interval_micros":100,"max_samples":10000});
        }
        self.write(
            &format!("{name}.json"),
            &serde_json::to_vec(&json!({"kind":"standalone","guest":guest,"network":null}))
                .unwrap(),
        )
    }
    fn invoke(&self, binary: &str, args: &[&std::ffi::OsStr]) -> Output {
        let index = NEXT.fetch_add(1, Ordering::Relaxed);
        let stdout_path = self.0.join(format!("stdout-{index}"));
        let stderr_path = self.0.join(format!("stderr-{index}"));
        let capture = |path: &Path| -> File {
            OpenOptions::new()
                .write(true)
                .create_new(true)
                .mode(0o600)
                .open(path)
                .unwrap()
        };
        let mut command = Command::new(binary);
        for (name, _) in std::env::vars_os() {
            if name
                .to_str()
                .is_some_and(|name| name.starts_with("GOMAD_") || name.starts_with("QUALDRV"))
            {
                command.env_remove(name);
            }
        }
        let mut child = command
            .args(args)
            .stdin(Stdio::null())
            .stdout(capture(&stdout_path))
            .stderr(capture(&stderr_path))
            .spawn()
            .unwrap();
        let deadline = Instant::now() + Duration::from_secs(30);
        let status = loop {
            if let Some(status) = child.try_wait().unwrap() {
                break status;
            }
            if Instant::now() >= deadline {
                child.kill().unwrap();
                let _ = child.wait();
                panic!("public native producer exceeded 30-second subprocess bound");
            }
            std::thread::sleep(Duration::from_millis(10));
        };
        assert!(std::fs::metadata(&stdout_path).unwrap().len() <= 65536);
        assert!(std::fs::metadata(&stderr_path).unwrap().len() <= 65536);
        Output {
            status,
            stdout: std::fs::read(stdout_path).unwrap(),
            stderr: std::fs::read(stderr_path).unwrap(),
        }
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn read(path: &Path) -> Value {
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}
fn assert_variant_artifacts(root: &Path) -> Value {
    let catalog = read(&root.join("variant-artifacts.json"));
    assert_eq!(catalog["schema"], "gomad.native-variant-artifacts/v1");
    assert_eq!(catalog["availability_only"], true);
    assert_eq!(catalog["catalog_errors"], json!([]));
    let entries = catalog["artifacts"].as_object().unwrap();
    assert_eq!(
        entries.keys().map(String::as_str).collect::<Vec<_>>(),
        vec![
            "canonical-wasi",
            "choice",
            "guest-stderr",
            "guest-stdout",
            "helper-stderr",
            "network"
        ]
    );
    for (name, suffix) in [
        ("canonical-wasi", "-canonical.bin"),
        ("choice", "-choice.bin"),
        ("network", "-network.jsonl"),
        ("helper-stderr", "-helper-stderr.txt"),
        ("guest-stdout", "-guest-stdout.txt"),
        ("guest-stderr", "-guest-stderr.txt"),
    ] {
        let entry = &entries[name];
        assert_eq!(entry["original_suffix"], suffix);
        if entry["status"] == "available" {
            let raw = std::fs::read(root.join(entry["file"].as_str().unwrap())).unwrap();
            assert_eq!(entry["bytes"], raw.len());
            assert_eq!(entry["sha256"], canonical::digest(&raw));
            assert!(entry["source"].as_str().is_some());
        } else if entry["status"] == "digest-only" {
            assert_eq!(name, "canonical-wasi");
            assert!(entry["file"].is_null());
            assert!(entry["bytes"].as_u64().is_some());
            assert!(
                entry["sha256"]
                    .as_str()
                    .is_some_and(canonical::valid_digest)
            );
            assert_eq!(entry["source"], "outcome:local_model.transcript_sha256");
            assert!(!root.join("wasi").exists());
            assert!(!root.join("transcript").exists());
        } else {
            assert_eq!(entry["status"], "unavailable");
            assert_ne!(entry["cause"].as_str().unwrap(), "");
            assert!(entry.get("file").is_none());
            assert!(entry.get("bytes").is_none());
            assert!(entry.get("sha256").is_none());
        }
    }
    for (file, binding) in catalog["files"].as_object().unwrap() {
        let raw = std::fs::read(root.join(file)).unwrap();
        assert_eq!(binding["bytes"], raw.len());
        assert_eq!(binding["sha256"], canonical::digest(&raw));
    }
    catalog
}

fn private_tree(path: &Path) {
    assert_eq!(
        std::fs::metadata(path).unwrap().permissions().mode() & 0o777,
        0o700
    );
    for entry in std::fs::read_dir(path).unwrap() {
        let entry = entry.unwrap();
        if entry.file_type().unwrap().is_dir() {
            private_tree(&entry.path());
        } else {
            assert_eq!(
                entry.metadata().unwrap().permissions().mode() & 0o777,
                0o600,
                "{}",
                entry.path().display()
            );
        }
    }
}
const SUCCESS: &str = "(module (memory (export \"memory\") 1) (func (export \"_start\")))";
const TRAP: &str = "(module (memory (export \"memory\") 1) (func (export \"_start\") unreachable))";

#[test]
fn actual_public_native_run_retains_authenticated_raw_observation() {
    let scratch = Scratch::new();
    let input = scratch.request("success", SUCCESS, false);
    let output = scratch.0.join("observed");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-measure"),
        &[
            "--linked-run".as_ref(),
            input.as_os_str(),
            output.as_os_str(),
        ],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    private_tree(&output);
    let observed = evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
    assert_eq!(observed.root.outcome["termination"], "exit");
    assert_eq!(observed.modules.len(), 1);
    compare_self(&observed);
    let catalog = assert_variant_artifacts(&output);
    for name in ["guest-stdout", "guest-stderr", "network"] {
        assert_eq!(catalog["artifacts"][name]["status"], "available");
        assert!(
            catalog["artifacts"][name]["source"]
                .as_str()
                .unwrap()
                .starts_with("closed-sink-empty:")
        );
    }
    for name in ["canonical-wasi", "choice", "helper-stderr"] {
        assert_eq!(catalog["artifacts"][name]["status"], "unavailable");
    }
    let report = read(&output.join("report.json"));
    assert_eq!(
        report["diagnostic_only"],
        cfg!(feature = "helper-diagnostic-profile")
    );
    #[cfg(not(feature = "helper-diagnostic-profile"))]
    {
        assert_eq!(report["features"], json!([]));
        assert!(report["attribution"].is_null());
        assert!(!output.join("profile.json").exists());
        assert!(observed.profile.is_null());
    }
    #[cfg(feature = "helper-diagnostic-profile")]
    {
        assert!(output.join("profile.json").is_file());
        gomad::profile::validate(&observed.profile).unwrap();
    }
}
fn compare_self(run: &evidence::NativeRun) {
    evidence::compare(run, run, ComparisonMask::Exact).unwrap();
}

#[test]
fn actual_native_trap_keeps_full_terminal_and_stream_catalogue() {
    let scratch = Scratch::new();
    let input = scratch.request("trap", TRAP, false);
    let output = scratch.0.join("trap-observed");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-measure"),
        &[
            "--linked-run".as_ref(),
            input.as_os_str(),
            output.as_os_str(),
        ],
    );
    assert!(
        result.status.success(),
        "diagnostic collection must retain a native terminal: {}",
        String::from_utf8_lossy(&result.stderr)
    );
    let observed = evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
    assert_ne!(observed.root.outcome["termination"], "exit");
    assert!(
        observed.root.outcome["message"]
            .as_str()
            .is_some_and(|message| !message.is_empty())
    );
    private_tree(&output);
}

#[cfg(feature = "helper-diagnostic-profile")]
#[test]
fn actual_profile_public_input_output_and_retained_root_keep_are_coherent() {
    let scratch = Scratch::new();
    let input = scratch.request("profile", SUCCESS, false);
    let output = scratch.0.join("direct");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-profile"),
        &[input.as_os_str(), output.as_os_str()],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let direct = evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
    let requested = scratch.0.join("requested");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-profile"),
        &[
            "retained".as_ref(),
            input.as_os_str(),
            requested.as_os_str(),
            "--keep".as_ref(),
        ],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let store = requested.join(".gomad-measure/linked-profile");
    let name = read(&store.join("latest.json"))
        .as_str()
        .unwrap()
        .to_string();
    let retained = store.join(name);
    assert!(retained.join(".keep").is_file());
    assert!(retained.join(".complete").is_file());
    let rerun = read(&retained.join("rerun.json"));
    assert_eq!(rerun["requested_root"], requested.to_str().unwrap());
    assert_eq!(
        rerun["command"].as_array().unwrap().last().unwrap(),
        "--keep"
    );
    let retained_run = evidence::read_profile_run(&retained, ReadLimits::default()).unwrap();
    evidence::compare(&direct, &retained_run, ComparisonMask::Exact).unwrap();
    private_tree(&retained);
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-profile"),
        &[
            "retained".as_ref(),
            input.as_os_str(),
            requested.as_os_str(),
        ],
    );
    assert!(result.status.success());
    assert!(
        retained.exists(),
        "explicit keep must retain superseded run"
    );
}

#[cfg(feature = "helper-guest-profile")]
#[test]
fn actual_guest_sampler_output_reaches_weighted_collector() {
    let scratch = Scratch::new();
    let input=scratch.request("sampled",r#"(module (memory (export "memory") 1) (func (export "_start") (local $i i32) (local.set $i (i32.const 20000000)) (loop $again (local.set $i (i32.sub (local.get $i) (i32.const 1))) (br_if $again (local.get $i)))))"#,true);
    let output = scratch.0.join("sampled");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-profile"),
        &[input.as_os_str(), output.as_os_str()],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let raw = std::fs::read(output.join("guest-profile.json")).unwrap();
    let attribution = gomad::measurement::guest::decode(&raw).unwrap();
    let raw: Value = serde_json::from_slice(&raw).unwrap();
    let expected_rows: u64 = raw["threads"]
        .as_array()
        .unwrap()
        .iter()
        .map(|thread| thread["samples"]["stack"].as_array().unwrap().len() as u64)
        .sum();
    assert_eq!(attribution.sample_rows, expected_rows);
    assert!(
        attribution.guest_sample_weight > 0,
        "actual sampler retained no guest stack weight"
    );
    assert_eq!(
        attribution.self_sample_weight.values().sum::<u64>(),
        attribution.guest_sample_weight
    );
    private_tree(&output);
    evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
}

#[cfg(feature = "helper-diagnostic-fuel-off")]
#[test]
fn actual_fuel_off_role_records_compiled_feature_not_environment_claim() {
    let scratch = Scratch::new();
    let input = scratch.request("fuel-off", SUCCESS, false);
    let output = scratch.0.join("fuel-off");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-profile"),
        &[input.as_os_str(), output.as_os_str()],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let report = read(&output.join("report.json"));
    assert!(
        report["features"]
            .as_array()
            .unwrap()
            .iter()
            .any(|feature| feature == "diagnostic-fuel-off")
    );
    assert!(
        report["configuration"]
            .as_str()
            .unwrap()
            .contains("fuel-off")
    );
    evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
}

#[test]
fn actual_measure_retained_command_preserves_requested_root_and_failed_diagnostics() {
    let scratch = Scratch::new();
    let input = scratch.request("retained-native", SUCCESS, false);
    let requested = scratch.0.join("retained-native");
    let result = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-measure"),
        &[
            "--retained-linked-run".as_ref(),
            input.as_os_str(),
            requested.as_os_str(),
        ],
    );
    assert!(
        result.status.success(),
        "{}",
        String::from_utf8_lossy(&result.stderr)
    );
    let store = requested.join(".gomad-measure/linked-profile");
    let latest = read(&store.join("latest.json"));
    let retained = store.join(latest.as_str().unwrap());
    let rerun = read(&retained.join("rerun.json"));
    assert_eq!(rerun["command"][1], "--retained-linked-run");
    assert_eq!(rerun["command"][3], requested.to_str().unwrap());
    evidence::read_profile_run(&retained, ReadLimits::default()).unwrap();
    let mut invalid = read(&input);
    invalid["guest"]["limits"]["memory_bytes"] = 0.into();
    let invalid_input = scratch.write(
        "refused-native.json",
        &serde_json::to_vec(&invalid).unwrap(),
    );
    let failed = scratch.invoke(
        env!("CARGO_BIN_EXE_gomad-measure"),
        &[
            "--retained-linked-run".as_ref(),
            invalid_input.as_os_str(),
            requested.as_os_str(),
        ],
    );
    assert!(!failed.status.success());
    assert_eq!(read(&store.join("latest.json")), latest);
    let failures: Vec<_> = std::fs::read_dir(&store)
        .unwrap()
        .map(Result::unwrap)
        .filter(|entry| {
            entry.file_type().unwrap().is_dir()
                && entry.file_name().to_str().unwrap().starts_with("run-")
                && entry.path() != retained
        })
        .collect();
    assert_eq!(failures.len(), 1);
    let failed = failures[0].path();
    assert!(!failed.join(".complete").exists());
    assert!(failed.join("request.json").is_file());
    assert!(failed.join("rootmodules.json").is_file());
    let report = read(&failed.join("report.json"));
    assert!(
        report["execution"]["Err"]
            .as_str()
            .is_some_and(|error| !error.is_empty())
    );
    let catalog = assert_variant_artifacts(&failed);
    assert_eq!(catalog["failure"], report["execution"]["Err"]);
    assert_eq!(catalog["transport"], "in-process-linked-run");
    for entry in catalog["artifacts"].as_object().unwrap().values() {
        assert_eq!(entry["status"], "unavailable");
    }
    for name in ["wasi", "choice", "network", "stdout", "stderr"] {
        assert!(
            !failed.join(name).exists(),
            "invented absent stream: {name}"
        );
    }
    private_tree(&failed);
}

#[test]
fn actual_cluster_cache_comparison_requires_actual_preparation_metrics() {
    let scratch = Scratch::new();
    let standalone = scratch.request("cluster-cache", SUCCESS, false);
    let mut guest = read(&standalone)["guest"].clone();
    guest["model"] = json!({"configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/"),
        "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
        "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
        "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},"arguments":[],"environment":["PWD=/"],"replay_trace":null});
    guest["namespace"] = json!({"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0});
    let handles = ["a", "b"].map(|node| json!({"node":node,"incarnation":1}));
    let mut request = json!({"kind":"cluster","nodes":[],
        "broker":{"seed":0,"pause_on_settle":false,"limits":{"max_requests":100,"max_events":100,"max_queued_events":100,"max_transitions":1000,"max_payload_bytes":4096,"max_string_bytes":1024}},
        "network":null,"incarnations":handles.iter().map(|handle|json!({"handle":handle,"guest":guest,"network":null})).collect::<Vec<_>>(),
        "steps":(["start","wait"].into_iter().flat_map(|action|handles.iter().map(move|handle|json!({"action":action,"handle":handle}))).collect::<Vec<_>>()),
        "boundaries":[],"crash_groups":[],"wall_millis":5000,"evidence_bytes":1<<20});
    let mut runs = Vec::new();
    for cached in [false, true] {
        if cached {
            let cache = scratch.0.join("cache");
            std::fs::DirBuilder::new()
                .mode(0o700)
                .create(&cache)
                .unwrap();
            request["compile_cache"] = json!(cache);
        }
        let input = scratch.write(
            &format!("cluster-{cached}.json"),
            &serde_json::to_vec(&request).unwrap(),
        );
        let output = scratch.0.join(format!("cluster-{cached}"));
        let result = scratch.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                "--linked-run".as_ref(),
                input.as_os_str(),
                output.as_os_str(),
            ],
        );
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
        runs.push(evidence::read_profile_run(&output, ReadLimits::default()).unwrap());
    }
    assert_eq!(runs[1].incarnations.len(), 2, "{}", runs[1].root.outcome);
    assert!(
        runs[1].root.outcome["metrics"]["prepare_nanos"]
            .as_u64()
            .is_some_and(|value| value > 0)
    );
    let role = if cfg!(feature = "helper-diagnostic-fuel-off") {
        evidence::VariantRole::DiagnosticFuelOff
    } else if cfg!(feature = "helper-guest-profile") {
        evidence::VariantRole::GuestProfile
    } else if cfg!(feature = "helper-diagnostic-profile") {
        evidence::VariantRole::DiagnosticProfile
    } else {
        evidence::VariantRole::Ordinary
    };
    let mask = || ComparisonMask::Variant {
        baseline: role,
        candidate: role,
        gc: None,
    };
    evidence::compare(&runs[0], &runs[1], mask()).unwrap();
    let mut missing = runs[1].clone();
    missing
        .root
        .outcome
        .as_object_mut()
        .unwrap()
        .remove("metrics");
    assert!(evidence::compare(&runs[0], &missing, mask()).is_err());
}

#[cfg(feature = "helper-diagnostic-profile")]
#[test]
fn actual_native_admission_refusals_retain_error_phase_counts_and_cleanup() {
    let root = Scratch::new();
    let input = root.request(
        "admission",
        "(module (memory (export \"memory\") 2) (func (export \"_start\") unreachable))",
        false,
    );
    let original = read(&input);
    for stage in ["module-digest", "loaded-module-descriptor"] {
        let mut request = original.clone();
        if stage == "module-digest" {
            request["guest"]["module_sha256"] = json!("0".repeat(64));
        }
        let request_path = root.write(
            &format!("{stage}.json"),
            &serde_json::to_vec(&request).unwrap(),
        );
        let output = root.0.join(stage);
        let result = root.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                std::ffi::OsStr::new("--linked-run"),
                request_path.as_os_str(),
                output.as_os_str(),
            ],
        );
        assert_eq!(result.status.success(), stage != "module-digest");
        let report = read(&output.join("report.json"));
        let refusal = &report["admission_refusal"];
        assert_eq!(refusal["schema"], "gomad.native-admission-refusal/v1");
        assert!(refusal["terminal_to_return_nanos"].as_u64().unwrap() <= 1_000_000_000);
        let profile = read(&output.join("profile.json"));
        for phase in ["instantiate", "execute"] {
            assert_eq!(profile["phases"][phase]["count"], 0);
        }
        if stage == "module-digest" {
            assert_eq!(refusal["stage"], "request-admission");
            assert_eq!(refusal["error"], "module SHA-256 mismatch");
            for phase in ["engine_init", "compile", "deserialize"] {
                assert_eq!(profile["phases"][phase]["count"], 0);
            }
            assert!(!output.join("outcome.json").exists());
        } else {
            assert_eq!(refusal["stage"], "loaded-module-descriptor");
            let outcome = read(&output.join("outcome.json"));
            assert_eq!(outcome["termination"], "capacity");
            assert_eq!(outcome["message"], "initial guest memory exceeds limit");
            assert!(outcome["metrics"]["prepare_nanos"].as_u64().unwrap() > 0);
            assert!(outcome["metrics"]["compile_nanos"].as_u64().unwrap() > 0);
            assert_eq!(outcome["metrics"]["instantiate_nanos"], 0);
            assert_eq!(outcome["metrics"]["execute_nanos"], 0);
            assert_eq!(profile["phases"]["compile"]["count"], 1);
            assert_eq!(read(&output.join("request.json")), original);
        }
    }
}

#[test]
fn actual_closed_native_run_publishes_explicit_canonical_stream_sources() {
    for modeled in [false, true] {
        let scratch = Scratch::new();
        let module = if modeled {
            r#"(module
                (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
                (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
                (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
                (memory (export "memory") 1)
                (func (export "_start")
                    i32.const 0 i32.const 16 call $config drop
                    i32.const 0 i64.const 0 i32.const 32 call $clock drop
                    i32.const 0 i32.const 16 call $finish drop))"#
        } else {
            SUCCESS
        };
        let input = scratch.request("canonical", module, false);
        if modeled {
            let mut request = read(&input);
            request["guest"]["model"] = json!({"configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/"),
                "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},"arguments":[],"environment":["PWD=/"],"replay_trace":null});
            request["guest"]["namespace"] = json!({"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0});
            std::fs::write(&input, serde_json::to_vec(&request).unwrap()).unwrap();
        }
        let output = scratch.0.join("observed");
        let result = scratch.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                "--linked-run".as_ref(),
                input.as_os_str(),
                output.as_os_str(),
            ],
        );
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
        let native = read(&output.join("streams.json"));
        let receipt = read(&output.join("canonical-streams.json"));
        assert_eq!(receipt["schema"], "gomad.native-canonical-streams/v1");
        assert_eq!(
            receipt["native_streams_sha256"],
            canonical::digest(&std::fs::read(output.join("streams.json")).unwrap())
        );
        assert_eq!(
            receipt["outcome_sha256"],
            canonical::digest(&std::fs::read(output.join("outcome.json")).unwrap())
        );
        for name in ["stdout", "stderr", "network"] {
            assert!(native.get(name).is_none());
            assert_eq!(std::fs::read(output.join(name)).unwrap(), b"");
            assert_eq!(
                receipt["streams"][name]["source"],
                format!("closed-sink-empty:{name}")
            );
            assert_eq!(receipt["streams"][name]["sha256"], canonical::digest(b""));
        }
        if modeled {
            assert_eq!(
                std::fs::read(output.join("wasi")).unwrap(),
                std::fs::read(output.join("transcript")).unwrap()
            );
            assert_eq!(
                receipt["streams"]["wasi"]["source"],
                "binary-transcript-alias:transcript"
            );
            let terminal = read(&output.join("outcome.json"))["local_model"]["terminal_base64"]
                .as_str()
                .unwrap()
                .to_owned();
            assert_ne!(terminal, "");
            assert_eq!(std::fs::read(output.join("terminal")).unwrap().len(), 16);
            assert_eq!(
                receipt["streams"]["terminal"]["source"],
                "outcome:local_model.terminal_base64"
            );
        } else {
            assert!(receipt["streams"].get("wasi").is_none());
            assert!(receipt["streams"].get("terminal").is_none());
            assert!(!output.join("wasi").exists());
        }
        if modeled {
            assert_native_argument_shapes_refused(&scratch, &input);
        }
        evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
    }
}

#[test]
fn actual_compiled_identity_producer_replaces_one_success_across_three_runs() {
    let scratch = Scratch::new();
    let requested = scratch.0.join("identity output root");
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    for _ in 0..3 {
        let result = scratch.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                "compiled-identity-owners".as_ref(),
                "--repo".as_ref(),
                repo.as_os_str(),
                "--diagnostic-identity-root".as_ref(),
                requested.as_os_str(),
            ],
        );
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
        let store = requested.join(".gomad-measure/compiled-identity-owners");
        let latest = read(&store.join("latest.json"))
            .as_str()
            .unwrap()
            .to_owned();
        assert!(read(&store.join(latest).join("compiled-owner-identities.json")).is_object());
        assert_eq!(
            std::fs::read_dir(&store)
                .unwrap()
                .map(Result::unwrap)
                .filter(|entry| entry.file_name().to_string_lossy().starts_with("run-"))
                .count(),
            1
        );
    }
}

fn assert_native_argument_shapes_refused(scratch: &Scratch, input: &Path) {
    for mutation in 0..4 {
        let mut malformed = read(input);
        match mutation {
            0 => malformed["guest"]["model"]["arguments"] = Value::Null,
            1 => {
                malformed["guest"]["model"]
                    .as_object_mut()
                    .unwrap()
                    .remove("arguments");
            }
            2 => malformed["guest"]["model"]["arguments"] = json!([Value::Null]),
            _ => malformed["guest"]["model"]["unexpected"] = true.into(),
        }
        let request = scratch.write(
            &format!("malformed-{mutation}.json"),
            &serde_json::to_vec(&malformed).unwrap(),
        );
        let refused = scratch.0.join(format!("refused-{mutation}"));
        let result = scratch.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                "--linked-run".as_ref(),
                request.as_os_str(),
                refused.as_os_str(),
            ],
        );
        assert!(
            !result.status.success(),
            "malformed argument shape {mutation}"
        );
        assert!(
            !refused.exists(),
            "malformed request reached native execution"
        );
    }
}

#[test]
fn actual_digest_only_wasi_preserves_rolling_evidence_without_invented_raw_payload() {
    let scratch = Scratch::new();
    let module = r#"(module
        (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
        (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start")
            i32.const 0 i32.const 16 call $config drop
            i32.const 0 i64.const 0 i32.const 32 call $clock drop
            i32.const 0 i32.const 16 call $finish drop))"#;
    let mut reference = None;
    for digest_only in [false, true] {
        let name = if digest_only { "digest" } else { "raw" };
        let input = scratch.request(name, module, false);
        let mut request = read(&input);
        request["guest"]["model"] = json!({"configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":canonical::digest(b"/"),
            "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
            "seed":0,"mode":1,"limit":4096,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,"digest_only":digest_only,
            "limits":{"calls":100,"transcript_bytes":4096,"pending_events":16}},"arguments":[],"environment":["PWD=/"],"replay_trace":null});
        request["guest"]["namespace"] = json!({"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1024},"epoch_nanos":0});
        std::fs::write(&input, serde_json::to_vec(&request).unwrap()).unwrap();
        let output = scratch.0.join(format!("observed-{name}"));
        let result = scratch.invoke(
            env!("CARGO_BIN_EXE_gomad-measure"),
            &[
                "--linked-run".as_ref(),
                input.as_os_str(),
                output.as_os_str(),
            ],
        );
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
        let outcome = read(&output.join("outcome.json"));
        assert_eq!(outcome["termination"], "exit");
        assert_eq!(outcome["exit_code"], 0);
        let model = &outcome["local_model"];
        let behavior = json!({"calls":model["calls"],"transcript_bytes":model["transcript_bytes"],"transcript_sha256":model["transcript_sha256"],"terminal_base64":model["terminal_base64"],"virtual_time_nanos":model["virtual_time_nanos"]});
        if let Some(reference) = &reference {
            assert_eq!(&behavior, reference);
        } else {
            reference = Some(behavior);
        }
        let native = read(&output.join("streams.json"));
        let canonical_streams = read(&output.join("canonical-streams.json"));
        let wasi = &canonical_streams["streams"]["wasi"];
        assert_eq!(wasi["bytes"], model["transcript_bytes"]);
        assert_eq!(wasi["sha256"], model["transcript_sha256"]);
        let artifacts = assert_variant_artifacts(&output);
        let entry = &artifacts["artifacts"]["canonical-wasi"];
        assert_eq!(entry["bytes"], wasi["bytes"]);
        assert_eq!(entry["sha256"], wasi["sha256"]);
        if digest_only {
            assert!(native.get("transcript").is_none() && native.get("wasi").is_none());
            assert_eq!(
                wasi,
                &json!({"file":null,"bytes":model["transcript_bytes"],"sha256":model["transcript_sha256"],"source":"outcome:local_model.transcript_sha256"})
            );
            assert_eq!(entry["status"], "digest-only");
            assert!(
                artifacts["files"].get("wasi").is_none()
                    && artifacts["files"].get("transcript").is_none()
            );
        } else {
            let raw = std::fs::read(output.join("transcript")).unwrap();
            assert_eq!(std::fs::read(output.join("wasi")).unwrap(), raw);
            assert_eq!(canonical::digest(&raw), wasi["sha256"]);
            assert_eq!(wasi["file"], "wasi");
            assert_eq!(wasi["source"], "binary-transcript-alias:transcript");
            assert_eq!(entry["status"], "available");
        }
        evidence::read_profile_run(&output, ReadLimits::default()).unwrap();
    }
}
