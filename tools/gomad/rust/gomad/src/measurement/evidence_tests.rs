use super::*;
use std::{
    os::unix::fs::{PermissionsExt, symlink},
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};
static NEXT: AtomicU64 = AtomicU64::new(0);
struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-native-witness-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&path).unwrap();
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o700)).unwrap();
        Self(path)
    }
    fn write(&self, name: &str, bytes: &[u8]) {
        let path = self.0.join(name);
        std::fs::write(&path, bytes).unwrap();
        std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600)).unwrap();
    }
    fn json(&self, name: &str, value: &Value) {
        self.write(name, &serde_json::to_vec(value).unwrap());
    }
    fn read(&self) -> Result<NativeRun, String> {
        read_profile_run(&self.0, ReadLimits::default())
    }
    fn execution(&self, prefix: &str, handle: Option<Value>) {
        self.json(&format!("{prefix}outcome.json"), &outcome());
        self.json(
            &format!("{prefix}streams.json"),
            &json!({"stdout":{"bytes":2,"sha256":canonical::digest(b"ok")}}),
        );
        self.write(&format!("{prefix}stdout"), b"ok");
        if let Some(handle) = handle {
            self.json(&format!("{prefix}handle.json"), &handle);
        }
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}
fn outcome() -> Value {
    json!({"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":50,"peak_memory_bytes":1024,"output_bytes":2,"metrics":{"execute_nanos":100},"local_model":{"virtual_time_nanos":100}})
}
fn fixture() -> Scratch {
    let root = Scratch::new();
    let module = b"\0asm\x01\0\0\0";
    let request = json!({"kind":"standalone","guest":{"module":"/original/module.wasm","module_sha256":canonical::digest(module),"model":null}});
    let raw = serde_json::to_vec(&request).unwrap();
    root.write("request.json", &raw);
    root.json("report.json",&json!({"schema":"gomad.linked-profile-run/v1","request_sha256":canonical::digest(&raw),"host_sha256":"a".repeat(64),"linked_implementation_sha256":"b".repeat(64),"host_implementation_sha256":"c".repeat(64),"execution":{"Ok":{"termination":"exit"}},"diagnostic_only":false,"features":[],"attribution":null}));

    root.json("rootmodules.json",&json!([{"request_module":"/original/module.wasm","file_name":"module-0.wasm","sha256":canonical::digest(module),"bytes":module.len(),"handle":null}]));
    root.write("module-0.wasm", module);
    root.execution("", None);
    root
}

#[test]
fn retained_module_authentication_survives_missing_original_source() {
    let root = fixture();
    let read = root.read().unwrap();
    assert_eq!(read.root.streams["stdout"], b"ok");
    assert_eq!(read.modules.len(), 1);
    compare(&read, &read, ComparisonMask::Exact).unwrap();
}

#[test]
fn raw_native_failures_remain_available_before_acceptance() {
    let root = fixture();
    let mut failed = outcome();
    failed["termination"] = "trap".into();
    failed.as_object_mut().unwrap().remove("exit_code");
    failed["message"] = "native guest trap".into();
    root.json("outcome.json", &failed);
    let mut report: Value =
        serde_json::from_slice(&std::fs::read(root.0.join("report.json")).unwrap()).unwrap();
    report["execution"] = json!({"Err":"native failure"});
    root.json("report.json", &report);
    let read = root.read().unwrap();
    assert_eq!(read.root.outcome["message"], "native guest trap");
    assert_eq!(read.report["execution"]["Err"], "native failure");
}

#[test]
fn private_modes_symlinks_catalogue_sizes_and_digests_are_enforced() {
    for mutation in 0..7 {
        let root = fixture();
        match mutation {
            0 => root.write("stdout", b"changed"),
            1 => root.write("module-0.wasm", b"changed"),
            2 => std::fs::set_permissions(
                root.0.join("stdout"),
                std::fs::Permissions::from_mode(0o644),
            )
            .unwrap(),
            3 => {
                std::fs::remove_file(root.0.join("stdout")).unwrap();
                symlink("module-0.wasm", root.0.join("stdout")).unwrap();
            }
            4 => root.json(
                "streams.json",
                &json!({"stdout":{"bytes":1,"sha256":canonical::digest(b"o")}}),
            ),
            5 => root.write(
                "request.json",
                b"{\"kind\":\"standalone\",\"kind\":\"standalone\"}",
            ),
            _ => root.json(
                "streams.json",
                &json!({"../escape":{"bytes":0,"sha256":canonical::digest(b"")}}),
            ),
        }
        assert!(root.read().is_err(), "mutation {mutation}");
    }
    let root = fixture();
    assert!(
        read_profile_run(
            &root.0,
            ReadLimits {
                total_bytes: 1,
                ..ReadLimits::default()
            }
        )
        .is_err()
    );
}

#[test]
fn all_native_terminal_fields_and_streams_remain_comparable_except_metrics() {
    let root = fixture();
    let before = root.read().unwrap();
    let mut after = before.clone();
    after.root.outcome["metrics"]["execute_nanos"] = 200.into();
    compare(&before, &after, ComparisonMask::Exact).unwrap();
    for mutation in 0..8 {
        let mut after = before.clone();
        match mutation {
            0 => after.root.outcome["message"] = "changed".into(),
            1 => after.root.outcome["fuel_remaining"] = 49.into(),
            2 => after.root.outcome["output_bytes"] = 3.into(),
            3 => after.root.outcome["peak_memory_bytes"] = 2048.into(),
            4 => after.root.outcome["local_model"]["virtual_time_nanos"] = 101.into(),
            5 => {
                after.root.streams.insert("network".into(), vec![1]);
            }
            6 => after.root.outcome["unexpected_native_field"] = "retained".into(),
            _ => after.request["guest"]["inherited_environment"] = json!(["UNDECLARED=1"]),
        }
        assert!(
            compare(&before, &after, ComparisonMask::Exact).is_err(),
            "mutation {mutation}"
        );
    }
}
fn with_environment(run: &NativeRun, gc: &str) -> NativeRun {
    let mut run = run.clone();
    let entries = vec!["PATH=/bin".to_string(), format!("GOGC={gc}")];
    let bytes: usize = entries.iter().map(|entry| entry.len() + 1).sum();
    run.request["guest"]["model"] = json!({"configuration":{"environment":{"bytes":bytes,"count":2,"chunks":1},"entropy_key":[0,1],"epoch_nanos":100,"read_step_nanos":1},"arguments":["program"],"environment":entries});
    run
}
#[test]
fn gc_mask_preserves_derived_environment_arguments_entropy_and_clock() {
    let root = fixture();
    let raw = root.read().unwrap();
    let before = with_environment(&raw, "100");
    let after = with_environment(&raw, "200");
    assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    compare(
        &before,
        &after,
        ComparisonMask::Gc {
            baseline: "100",
            candidate: "200",
        },
    )
    .unwrap();
    for mutation in 0..7 {
        let mut changed = after.clone();
        match mutation {
            0 => changed.request["guest"]["model"]["arguments"] = json!(["changed"]),
            1 => changed.request["guest"]["model"]["environment"][0] = "PATH=/sbin".into(),
            2 => changed.request["guest"]["model"]["configuration"]["epoch_nanos"] = 101.into(),
            3 => changed.request["guest"]["model"]["configuration"]["read_step_nanos"] = 2.into(),
            4 => changed.request["guest"]["model"]["configuration"]["entropy_key"] = json!([2, 3]),
            5 => {
                changed.request["guest"]["model"]["configuration"]["environment"]["bytes"] =
                    0.into();
            }
            _ => changed.request["guest"]["model"]["environment"][1] = "GOGC=300".into(),
        }
        assert!(
            compare(
                &before,
                &changed,
                ComparisonMask::Gc {
                    baseline: "100",
                    candidate: "200"
                }
            )
            .is_err(),
            "mutation {mutation}"
        );
    }
}

#[test]
fn cluster_incarnations_steps_and_cutoff_are_authenticated_and_compared() {
    let root = fixture();
    let mut request: Value =
        serde_json::from_slice(&std::fs::read(root.0.join("request.json")).unwrap()).unwrap();
    let guest = request["guest"].clone();
    let handle = json!({"slot":0,"incarnation":0});
    request = json!({"kind":"cluster","incarnations":[{"guest":guest,"handle":handle}],"steps":[]});
    let raw = serde_json::to_vec(&request).unwrap();
    root.write("request.json", &raw);
    let mut report: Value =
        serde_json::from_slice(&std::fs::read(root.0.join("report.json")).unwrap()).unwrap();
    report["request_sha256"] = canonical::digest(&raw).into();
    report["retained_incarnations"] = 1.into();
    root.json("report.json", &report);
    let mut modules: Value =
        serde_json::from_slice(&std::fs::read(root.0.join("rootmodules.json")).unwrap()).unwrap();
    modules[0]["handle"] = handle.clone();
    root.json("rootmodules.json", &modules);
    root.json("executed-steps.json", &json!([]));
    root.json("cutoff.json", &json!({"virtual_elapsed_nanos":100}));
    std::fs::create_dir(root.0.join("incarnation-0")).unwrap();
    std::fs::set_permissions(
        root.0.join("incarnation-0"),
        std::fs::Permissions::from_mode(0o700),
    )
    .unwrap();
    root.execution("incarnation-0/", Some(handle));
    let before = root.read().unwrap();
    assert_eq!(before.incarnations.len(), 1);
    for mutation in 0..4 {
        let mut after = before.clone();
        match mutation {
            0 => after.cutoff = Some(json!({"virtual_elapsed_nanos":101})),
            1 => after.executed_steps = Some(json!([{"changed":true}])),
            2 => after.incarnations[0].handle = Some(json!({"slot":1,"incarnation":0})),
            _ => {
                after.incarnations[0]
                    .streams
                    .insert("stderr".into(), b"changed".to_vec());
            }
        }
        assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    }
}

#[test]
fn ordinary_run_refuses_diagnostic_claims_and_unexpected_profile_payload() {
    for mutation in 0..3 {
        let root = fixture();
        let mut report: Value =
            serde_json::from_slice(&std::fs::read(root.0.join("report.json")).unwrap()).unwrap();
        match mutation {
            0 => report["features"] = json!(["diagnostic-profile"]),
            1 => report["attribution"] = json!({"Ok":null}),
            _ => root.json("profile.json", &json!({"invented":true})),
        }
        root.json("report.json", &report);
        assert!(root.read().is_err(), "mutation {mutation}");
    }
}

fn declare_variant_role(run: &mut NativeRun, role: VariantRole) {
    let (features, suffix): (&[&str], &str) = match role {
        VariantRole::Ordinary => (&[], ""),
        VariantRole::DiagnosticProfile => (&["diagnostic-profile"], "-diagnostic-profile"),
        VariantRole::GuestProfile => (&["diagnostic-profile", "guest-profile"], "-guest-profile"),
        VariantRole::DiagnosticFuelOff => (
            &["diagnostic-fuel-off", "diagnostic-profile"],
            "-diagnostic-fuel-off",
        ),
    };
    run.report["diagnostic_only"] = (!features.is_empty()).into();
    run.report["features"] = json!(features);
    run.report["configuration"] =
        format!("{}{suffix}", gomad_wasmhost::linked::CONFIGURATION).into();
    run.report["host_os"] = "linux".into();
    run.report["host_arch"] = "x86_64".into();
}

#[test]
fn explicit_variants_exclude_observational_fuel_and_memory_but_exact_retains_them() {
    let root = fixture();
    let mut before = root.read().unwrap();
    let mut after = before.clone();
    declare_variant_role(&mut before, VariantRole::DiagnosticProfile);
    declare_variant_role(&mut after, VariantRole::DiagnosticFuelOff);
    after.root.outcome["fuel_remaining"] = 1000.into();
    after.root.outcome["peak_memory_bytes"] = 4096.into();
    after.root.outcome["metrics"] = json!({"execute_nanos":300});
    assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    compare(&before, &after, ComparisonMask::FuelOff).unwrap();
    assert_eq!(after.root.outcome["fuel_remaining"], 1000);
    assert_eq!(after.root.outcome["peak_memory_bytes"], 4096);
    for mutation in 0..7 {
        let mut changed = after.clone();
        match mutation {
            0 => changed.root.outcome["message"] = "changed".into(),
            1 => changed.root.outcome["termination"] = "trap".into(),
            2 => changed.root.outcome["exit_code"] = 1.into(),
            3 => changed.root.outcome["local_model"]["virtual_time_nanos"] = 101.into(),
            4 => changed.root.outcome["output_bytes"] = 3.into(),
            5 => changed.root.outcome["local_model"]["calls"] = 1.into(),
            _ => {
                changed.root.streams.insert("choice".into(), vec![1]);
            }
        }
        assert!(
            compare(&before, &changed, ComparisonMask::FuelOff).is_err(),
            "mutation {mutation}"
        );
    }
}

#[test]
fn explicit_variants_require_declared_actual_roles_features_and_configuration() {
    let root = fixture();
    let mut before = root.read().unwrap();
    let mut after = before.clone();
    declare_variant_role(&mut before, VariantRole::Ordinary);
    declare_variant_role(&mut after, VariantRole::GuestProfile);
    let compare_variant = |before: &NativeRun, after: &NativeRun| {
        compare(
            before,
            after,
            ComparisonMask::Variant {
                baseline: VariantRole::Ordinary,
                candidate: VariantRole::GuestProfile,
                gc: None,
            },
        )
    };
    compare_variant(&before, &after).unwrap();
    for mutation in 0..7 {
        let mut changed = after.clone();
        match mutation {
            0 => changed.report["features"] = json!(["diagnostic-profile"]),
            1 => changed.report["configuration"] = "invented-profile".into(),
            2 => changed.report["diagnostic_only"] = false.into(),
            3 => changed.report["host_sha256"] = "not-a-digest".into(),
            4 => changed.report["host_arch"] = "aarch64".into(),
            5 => changed.report["host_os"] = "other-os".into(),
            _ => {
                changed.report["features"] =
                    json!(["diagnostic-profile", "guest-profile", "invented"]);
            }
        }
        assert!(
            compare_variant(&before, &changed).is_err(),
            "mutation {mutation}"
        );
    }
}

fn native_record(tag: u8, flags: u8, errno: u16, id: u64, body: &[u8]) -> Vec<u8> {
    let mut bytes = b"GMWT\x01\0\0\0".to_vec();
    bytes.extend([tag, flags]);
    bytes.extend(errno.to_le_bytes());
    bytes.extend(id.to_le_bytes());
    bytes.extend(u32::try_from(body.len()).unwrap().to_le_bytes());
    bytes.extend(body);
    bytes
}

#[test]
fn byte_identical_malformed_native_wasi_transcripts_are_refused_before_comparison() {
    let root = fixture();
    let original = root.read().unwrap();
    for bytes in [
        b"old-json-transcript".to_vec(),
        native_record(5, 0, 0, 0, &[]),
        native_record(5, 0, 0, 1, &[1]),
        native_record(255, 0, 0, 1, &[]),
    ] {
        let mut changed = original.clone();
        changed.root.streams.insert("transcript".into(), bytes);
        assert!(compare(&changed, &changed, ComparisonMask::Exact).is_err());
    }
    let mut before = original;
    before
        .root
        .streams
        .insert("transcript".into(), native_record(5, 0, 0, 1, &[]));
    compare(&before, &before, ComparisonMask::Exact).unwrap();
    for changed_bytes in [
        native_record(6, 0, 0, 1, &[]),
        native_record(5, 0, 1, 1, &[]),
        native_record(5, 0, 0, 2, &[]),
    ] {
        let mut after = before.clone();
        after
            .root
            .streams
            .insert("transcript".into(), changed_bytes);
        assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    }
}

#[test]
fn unsupported_native_operations_preserve_typed_input_and_reason() {
    let root = fixture();
    let mut before = root.read().unwrap();
    let unsupported = |fd: u32, reason: &str| {
        let mut body = Vec::new();
        let raw = serde_json::to_vec(&json!({"fd":fd})).unwrap();
        for bytes in [
            b"future_operation".as_slice(),
            raw.as_slice(),
            reason.as_bytes(),
        ] {
            body.extend(u32::try_from(bytes.len()).unwrap().to_le_bytes());
            body.extend(bytes);
        }
        native_record(34, 1, 0, 1, &body)
    };
    before
        .root
        .streams
        .insert("transcript".into(), unsupported(3, "outside profile"));
    compare(&before, &before, ComparisonMask::Exact).unwrap();
    for bytes in [
        unsupported(4, "outside profile"),
        unsupported(3, "different reason"),
    ] {
        let mut after = before.clone();
        after.root.streams.insert("transcript".into(), bytes);
        assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    }
}

#[test]
fn only_explicit_gc_off_masks_memory_limit_and_preserves_packed_environment_binding() {
    let root = fixture();
    let raw = root.read().unwrap();
    let mut before = with_environment(&raw, "100");
    let mut after = with_environment(&raw, "off");
    for (run, limit) in [
        (&mut before, "GOMEMLIMIT=1MiB"),
        (&mut after, "GOMEMLIMIT=off"),
    ] {
        let model = &mut run.request["guest"]["model"];
        model["environment"]
            .as_array_mut()
            .unwrap()
            .push(limit.into());
        let values = model["environment"].as_array().unwrap();
        let bytes: usize = values
            .iter()
            .map(|value| value.as_str().unwrap().len() + 1)
            .sum();
        model["configuration"]["environment"] =
            json!({"bytes":bytes,"count":values.len(),"chunks":bytes.div_ceil(64*1024)});
        declare_variant_role(run, VariantRole::DiagnosticProfile);
    }
    let compare_gc = |before: &NativeRun, after: &NativeRun| {
        compare(
            before,
            after,
            ComparisonMask::Variant {
                baseline: VariantRole::DiagnosticProfile,
                candidate: VariantRole::DiagnosticProfile,
                gc: Some(GcMask {
                    baseline: "100",
                    candidate: "off",
                }),
            },
        )
    };
    compare_gc(&before, &after).unwrap();
    let mut changed = after;
    changed.request["guest"]["model"]["configuration"]["environment"]["bytes"] = 0.into();
    assert!(compare_gc(&before, &changed).is_err());
    let mut numeric = with_environment(&raw, "200");
    declare_variant_role(&mut numeric, VariantRole::DiagnosticProfile);
    assert!(
        compare(
            &before,
            &numeric,
            ComparisonMask::Variant {
                baseline: VariantRole::DiagnosticProfile,
                candidate: VariantRole::DiagnosticProfile,
                gc: Some(GcMask {
                    baseline: "100",
                    candidate: "200"
                }),
            }
        )
        .is_err(),
        "numeric GC variant must preserve GOMEMLIMIT"
    );
}

fn replay_pair() -> (NativeRun, NativeRun, gomad_wasmhost::linked::ChoiceIdentity) {
    let root = fixture();
    let mut before = root.read().unwrap();
    let fixture: Value = serde_json::from_str(include_str!(
        "../../../wasmhost/tests/testdata/replay_plan/empty.json"
    ))
    .unwrap();
    let identity: gomad_wasmhost::linked::ChoiceIdentity =
        serde_json::from_value(fixture["identity"].clone()).unwrap();
    let target = super::super::workloads::hex(&identity.target_sha256);
    before.request["guest"]["module_sha256"] = target.into();
    before.request["guest"]["model"] = json!({"configuration":{"mode":1},"replay_trace":null});
    before.root.streams.insert("choice".into(), Vec::new());
    before.root.outcome["local_model"] = json!({"cursor":0,"tape_records":0,"tape_sha256":"","next_expected":null,"calls":1,"advances":0,"finished":true,"records":0});
    declare_variant_role(&mut before, VariantRole::Ordinary);
    let mut after = before.clone();
    let inspection = gomad_wasmhost::linked::inspect_choices(&[], &identity).unwrap();
    after.request["guest"]["model"]["configuration"]["mode"] = 2.into();
    after.request["guest"]["model"]["configuration"]["replay"] = json!({"schema":"gomad.replay-trace/v1","execution_identity":fixture["identity"],"trace_sha256":canonical::digest(&[]),"tape_sha256":inspection.tape_sha256,"decisions":0,"trace_bytes":0,"trace_frames":0,"prefix_control":null});
    after.request["guest"]["model"]["replay_trace"] = json!([]);
    after.root.outcome["local_model"]["tape_sha256"] = inspection.tape_sha256.into();
    (before, after, identity)
}

#[test]
fn native_full_replay_authenticates_projection_before_narrow_bookkeeping_translation() {
    let (before, after, identity) = replay_pair();
    compare_replay(&before, &after, &identity).unwrap();
    assert_ne!(
        after.root.outcome["local_model"]["tape_sha256"],
        before.root.outcome["local_model"]["tape_sha256"]
    );
    assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    for mutation in 0..16 {
        let mut changed = after.clone();
        match mutation {
            0 => changed.request["guest"]["model"]["configuration"]["mode"] = 3.into(),
            1 => changed.request["guest"]["model"]["replay_trace"] = json!([0]),
            2 => {
                changed.request["guest"]["model"]["configuration"]["replay"]["trace_sha256"] =
                    "a".repeat(64).into();
            }
            3 => {
                changed.request["guest"]["model"]["configuration"]["replay"]["decisions"] =
                    1.into();
            }
            4 => {
                changed.request["guest"]["model"]["configuration"]["replay"]["tape_sha256"] =
                    "a".repeat(64).into();
            }
            5 => {
                changed.request["guest"]["model"]["configuration"]["replay"]["trace_frames"] =
                    1.into();
            }
            6 => {
                changed.request["guest"]["model"]["configuration"]["replay"]["execution_identity"]
                    ["ToolchainBuildKey"] = "changed".into();
            }
            7 => changed.root.outcome["local_model"]["cursor"] = 1.into(),
            8 => changed.root.outcome["local_model"]["tape_records"] = 1.into(),
            9 => changed.root.outcome["local_model"]["tape_sha256"] = "a".repeat(64).into(),
            10 => {
                changed.root.outcome["local_model"]
                    .as_object_mut()
                    .unwrap()
                    .remove("next_expected");
            }
            11 => changed.root.outcome["local_model"]["calls"] = 2.into(),
            12 => changed.root.outcome["local_model"]["finished"] = false.into(),
            13 => changed.root.outcome["unknown_native_field"] = true.into(),
            14 => changed.report["host_implementation_sha256"] = "d".repeat(64).into(),
            _ => changed.request["guest"]["module_sha256"] = "d".repeat(64).into(),
        }
        assert!(
            compare_replay(&before, &changed, &identity).is_err(),
            "mutation {mutation}"
        );
    }
    let mut changed_identity = identity;
    changed_identity.target_sha256[0] ^= 1;
    assert!(compare_replay(&before, &after, &changed_identity).is_err());
}

#[test]
fn declared_sampling_mask_changes_only_validated_host_sampler_options() {
    let root = fixture();
    let mut before = root.read().unwrap();
    declare_variant_role(&mut before, VariantRole::GuestProfile);
    let mut after = before.clone();
    after.request["guest"]["guest_profile"] = json!({"interval_micros":100,"max_samples":1000});
    assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    compare(
        &before,
        &after,
        ComparisonMask::Sampling {
            interval_micros: 100,
            max_samples: 1000,
        },
    )
    .unwrap();
    for mutation in 0..5 {
        let mut changed = after.clone();
        match mutation {
            0 => changed.request["guest"]["guest_profile"]["interval_micros"] = 101.into(),
            1 => changed.request["guest"]["guest_profile"]["undeclared_option"] = true.into(),
            2 => declare_variant_role(&mut changed, VariantRole::DiagnosticProfile),
            3 => changed.request["guest"]["limits"] = json!({"fuel":999}),
            _ => {
                changed
                    .root
                    .streams
                    .insert("stdout".into(), b"different".to_vec());
            }
        }
        assert!(
            compare(
                &before,
                &changed,
                ComparisonMask::Sampling {
                    interval_micros: 100,
                    max_samples: 1000
                }
            )
            .is_err()
        );
    }
    assert!(
        compare(
            &before,
            &after,
            ComparisonMask::Sampling {
                interval_micros: 0,
                max_samples: 1000
            }
        )
        .is_err()
    );
}

#[test]
fn declared_cache_location_projection_requires_live_private_safe_admission() {
    let root = fixture();
    let mut before = root.read().unwrap();
    declare_variant_role(&mut before, VariantRole::DiagnosticProfile);
    let first = Scratch::new();
    let second = Scratch::new();
    before.request["compile_cache"] = std::fs::canonicalize(&first.0)
        .unwrap()
        .to_string_lossy()
        .into_owned()
        .into();
    before.root.outcome["metrics"] = json!({"prepare_nanos":10,"cache_hit":true,"compile_nanos":0});
    let mut after = before.clone();
    after.request["compile_cache"] = std::fs::canonicalize(&second.0)
        .unwrap()
        .to_string_lossy()
        .into_owned()
        .into();
    let variant = || ComparisonMask::Variant {
        baseline: VariantRole::DiagnosticProfile,
        candidate: VariantRole::DiagnosticProfile,
        gc: None,
    };
    assert!(compare(&before, &after, ComparisonMask::Exact).is_err());
    compare(&before, &after, variant()).unwrap();
    std::fs::set_permissions(&second.0, std::fs::Permissions::from_mode(0o755)).unwrap();
    assert!(compare(&before, &after, variant()).is_err());
    std::fs::set_permissions(&second.0, std::fs::Permissions::from_mode(0o700)).unwrap();
    after.request["compile_cache"] = "relative/cache".into();
    assert!(compare(&before, &after, variant()).is_err());
    let alias = first.0.join("alias");
    symlink(&second.0, &alias).unwrap();
    after.request["compile_cache"] = alias.to_string_lossy().into_owned().into();
    assert!(compare(&before, &after, variant()).is_err());
}

#[test]
fn gc_variant_preserves_native_zero_bootstrap_string_declarations() {
    let root = fixture();
    let raw = root.read().unwrap();
    let mut before = with_environment(&raw, "100");
    let mut after = with_environment(&raw, "200");
    for run in [&mut before, &mut after] {
        run.request["guest"]["model"]["configuration"]["environment"] =
            json!({"bytes":0,"count":0,"chunks":0});
    }
    compare(
        &before,
        &after,
        ComparisonMask::Gc {
            baseline: "100",
            candidate: "200",
        },
    )
    .unwrap();
    after.request["guest"]["model"]["environment"][0] = "PATH=/other".into();
    assert!(
        compare(
            &before,
            &after,
            ComparisonMask::Gc {
                baseline: "100",
                candidate: "200"
            }
        )
        .is_err()
    );
}
