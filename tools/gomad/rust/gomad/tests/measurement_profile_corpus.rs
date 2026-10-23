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
        let mut child = Command::new(binary)
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
