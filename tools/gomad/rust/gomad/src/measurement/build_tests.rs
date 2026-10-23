use super::*;
use std::os::unix::fs::MetadataExt;
struct Temporary(PathBuf);
impl Temporary {
    fn new() -> Self {
        Self(directory(&std::env::temp_dir(), "gomad-native-build-contract").unwrap())
    }
}
impl Drop for Temporary {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}
fn features(values: &[&str]) -> Vec<String> {
    values.iter().map(|value| (*value).into()).collect()
}
fn cargo_frame(actual: &[String], opt: &str, debug: bool) -> serde_json::Value {
    serde_json::json!({"reason":"compiler-artifact","target":{"name":"gomad-measure","kind":["bin"]},"profile":{"opt_level":opt,"debug_assertions":debug},"executable":"/release/gomad-measure","features":actual})
}
fn stream(frame: &serde_json::Value, finished: Option<bool>) -> Vec<u8> {
    let mut bytes = serde_json::to_vec(frame).unwrap();
    if let Some(success) = finished {
        bytes.extend(
            serde_json::to_vec(&serde_json::json!({"reason":"build-finished","success":success}))
                .unwrap(),
        );
    }
    bytes
}
#[test]
fn terminal_exit_cargo_artifact_controls() {
    let release = cargo_frame(&[], "3", false);
    let mut missing = release.clone();
    missing.as_object_mut().unwrap().remove("features");
    for (label, data, valid) in [
        ("release", stream(&release, Some(true)), true),
        (
            "debug-build",
            stream(&cargo_frame(&[], "0", true), Some(true)),
            false,
        ),
        (
            "diagnostic-feature",
            stream(
                &cargo_frame(&features(&["helper-diagnostic-profile"]), "3", false),
                Some(true),
            ),
            false,
        ),
        ("missing-features", stream(&missing, Some(true)), false),
        ("failed-build", stream(&release, Some(false)), false),
        ("missing-finish", stream(&release, None), false),
        (
            "missing-helper",
            serde_json::to_vec(&serde_json::json!({"reason":"build-finished","success":true}))
                .unwrap(),
            false,
        ),
        ("malformed", b"{".to_vec(), false),
        (
            "duplicate",
            {
                let mut data = stream(&release, None);
                data.extend(stream(&release, Some(true)));
                data
            },
            false,
        ),
    ] {
        let result = cargo_artifact(&data, &Role::Release);
        assert_eq!(result.is_ok(), valid, "{label}: {result:?}");
        if valid {
            assert_eq!(result.unwrap(), Path::new("/release/gomad-measure"));
        }
    }
}
#[test]
fn diagnostic_admission_cargo_features() {
    for original in [
        features(&["diagnostic-profile"]),
        features(&["diagnostic-profile", "guest-profile"]),
        features(&["diagnostic-fuel-off", "diagnostic-profile"]),
    ] {
        let role = Role::Diagnostic(original);
        let wanted = cargo_features(&role).unwrap();
        let mut extra = wanted.clone();
        extra.push("unexpected".into());
        for (actual, accepted) in [
            (wanted.clone(), true),
            (wanted[..wanted.len() - 1].to_vec(), false),
            (wanted[1..].to_vec(), false),
            (extra, false),
        ] {
            let data = stream(&cargo_frame(&actual, "3", false), Some(true));
            assert_eq!(cargo_artifact(&data, &role).is_ok(), accepted, "{actual:?}");
            assert_eq!(
                cargo_artifact(&data, &Role::Release).is_ok(),
                actual.is_empty()
            );
        }
    }
}
#[test]
fn warm_preparation_diagnostic_cargo_feature_controls() {
    assert!(
        cargo_artifact(
            &stream(
                &cargo_frame(&features(&["helper-diagnostic-profile"]), "3", false),
                Some(true)
            ),
            &Role::Warm
        )
        .is_ok()
    );
    for actual in [
        vec![],
        features(&["helper-guest-profile"]),
        features(&["helper-diagnostic-profile", "fuel-off"]),
    ] {
        assert!(
            cargo_artifact(
                &stream(&cargo_frame(&actual, "3", false), Some(true)),
                &Role::Warm
            )
            .is_err()
        );
    }
}
#[test]
fn diagnostic_build_rejects_unadmitted_features() {
    for invalid in [
        vec![],
        features(&["guest-profile"]),
        features(&["diagnostic-profile", "diagnostic-profile"]),
        features(&["diagnostic-profile", "unexpected"]),
        features(&["diagnostic-fuel-off", "diagnostic-profile", "guest-profile"]),
    ] {
        assert!(Role::Diagnostic(invalid).features().is_err());
    }
}
#[test]
fn terminal_exit_compiler_environment_controls() {
    let inputs = BTreeMap::from([(PathBuf::from("/pinned/rustc"), "a".repeat(64))]);
    for (entries, valid) in [
        (
            features(&[
                "RUSTC=/pinned/rustc",
                "RUSTC_WRAPPER=",
                "RUSTC_WORKSPACE_WRAPPER=",
            ]),
            true,
        ),
        (
            features(&["RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="]),
            false,
        ),
        (
            features(&[
                "RUSTC=/other/rustc",
                "RUSTC_WRAPPER=",
                "RUSTC_WORKSPACE_WRAPPER=",
            ]),
            false,
        ),
        (
            features(&[
                "RUSTC=/pinned/rustc",
                "RUSTC=/other/rustc",
                "RUSTC_WRAPPER=",
                "RUSTC_WORKSPACE_WRAPPER=",
            ]),
            false,
        ),
        (
            features(&["RUSTC=/pinned/rustc", "RUSTC_WORKSPACE_WRAPPER="]),
            false,
        ),
        (
            features(&[
                "RUSTC=/pinned/rustc",
                "RUSTC_WRAPPER=/wrapper",
                "RUSTC_WORKSPACE_WRAPPER=",
            ]),
            false,
        ),
        (
            features(&[
                "RUSTC=/pinned/rustc",
                "RUSTC_WRAPPER=",
                "RUSTC_WORKSPACE_WRAPPER=/wrapper",
            ]),
            false,
        ),
    ] {
        assert_eq!(
            compiler_environment(Path::new("/pinned/rustc"), &inputs, &entries),
            valid
        );
    }
}
#[test]
fn terminal_exit_stages_hardlinked_cargo_helper() {
    let temp = Temporary::new();
    let artifact = temp.0.join("cargo-helper");
    let alias = temp.0.join("cargo-alias");
    let helper = temp.0.join("admitted");
    let data = b"#!/bin/sh\nexit 0\n";
    fs::write(&artifact, data).unwrap();
    fs::set_permissions(&artifact, fs::Permissions::from_mode(0o755)).unwrap();
    fs::hard_link(&artifact, &alias).unwrap();
    stage_executable(&artifact, &helper, &canonical::digest(data)).unwrap();
    assert_eq!(bounded(&artifact, LIMIT).unwrap(), data);
    assert_eq!(bounded(&helper, LIMIT).unwrap(), data);
    assert_eq!(
        fs::metadata(&helper).unwrap().permissions().mode() & 0o777,
        0o755
    );
    assert_ne!(
        fs::metadata(&helper).unwrap().ino(),
        fs::metadata(&artifact).unwrap().ino()
    );
    assert_eq!(fs::metadata(&helper).unwrap().nlink(), 1);
    assert!(stage_executable(&artifact, &helper, &canonical::digest(data)).is_err());
    assert_eq!(bounded(&helper, LIMIT).unwrap(), data);
}
#[test]
fn stage_refuses_drift_symlinks_and_empty_input() {
    let temp = Temporary::new();
    let artifact = temp.0.join("cargo-helper");
    let helper = temp.0.join("admitted");
    fs::write(&artifact, b"changed").unwrap();
    fs::set_permissions(&artifact, fs::Permissions::from_mode(0o755)).unwrap();
    assert!(stage_executable(&artifact, &helper, &canonical::digest(b"original")).is_err());
    assert!(!helper.exists());
    let alias = temp.0.join("symlink");
    std::os::unix::fs::symlink(&artifact, &alias).unwrap();
    assert!(stage_executable(&alias, &helper, &canonical::digest(b"changed")).is_err());
    fs::write(&artifact, []).unwrap();
    assert!(stage_executable(&artifact, &helper, &canonical::digest(b"")).is_err());
}
#[test]
fn actual_command_enforces_argv_directory_environment_and_exit() {
    let temp = Temporary::new();
    let environment = BTreeMap::from([("VISIBLE".into(), "a b".into())]);
    let output = command(
        &temp.0,
        "probe",
        &temp.0,
        &environment,
        &features(&[
            "/bin/sh",
            "-c",
            "printf '%s\\n' \"$PWD\" \"$VISIBLE\" \"${RUSTFLAGS-unset}\" \"$1\"",
            "probe",
            "quoted argument",
        ]),
    )
    .unwrap();
    assert_eq!(
        String::from_utf8(output).unwrap(),
        format!(
            "{}\na b\nunset\nquoted argument\n",
            temp.0.canonicalize().unwrap().display()
        )
    );
    assert!(
        command(
            &temp.0,
            "failed",
            &temp.0,
            &environment,
            &features(&["/bin/sh", "-c", "exit 7"])
        )
        .is_err()
    );
    let inherited = inherited(&features(&[
        "PATH=/bin",
        "RUSTFLAGS=foreign",
        "RUSTC=/foreign",
        "RUSTC_WRAPPER=foreign",
        "CARGO_HOME=/cache",
        "GOMAD_WASM_PROFILE_ROOT=hidden",
    ]));
    assert_eq!(
        inherited,
        BTreeMap::from([
            ("PATH".into(), "/bin".into()),
            ("CARGO_HOME".into(), "/cache".into())
        ])
    );
}
#[test]
fn compiled_native_identity_owners() {
    let temp = Temporary::new();
    let path = compiled_owners(&temp.0).unwrap();
    let value: serde_json::Value = serde_json::from_slice(&bounded(&path, LIMIT).unwrap()).unwrap();
    assert_eq!(value["schema"], "gomad.native-compiled-owner-identities/v1");
    assert_eq!(value["owners"].as_object().unwrap().len(), 3);
    for digest in value["owners"].as_object().unwrap().values() {
        assert_eq!(digest.as_str().unwrap().len(), 64);
    }
}

fn command_receipt(role: Role) -> BuildReceipt {
    BuildReceipt {
        schema: BUILD_SCHEMA.into(),
        crate_root: "/crate".into(),
        executable: "/target/release/gomad-measure".into(),
        executable_sha256: "e".repeat(64),
        rustc: "/pinned/rustc".into(),
        cargo: "/pinned/cargo".into(),
        inputs: BTreeMap::from([(PathBuf::from("/pinned/cargo"), "a".repeat(64))]),
        command: cargo_command(Path::new("/pinned/cargo"), Path::new("/crate"), &role).unwrap(),
        environment: BTreeMap::new(),
        role,
        rustc_version: String::new(),
        cargo_version: String::new(),
        identity: NativeIdentity {
            schema: "gomad.native-measurement-identity/v1".into(),
            role: "guest-profile".into(),
            features: features(&["diagnostic-profile", "guest-profile"]),
            host_implementation_sha256: "a".repeat(64),
            linked_implementation_sha256: "b".repeat(64),
            runtime_implementation_sha256: env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into(),
            configuration: format!("{}-guest-profile", gomad_wasmhost::linked::CONFIGURATION),
            source_inputs: BTreeMap::from([
                ("host".into(), inputs::SOURCE_INPUTS.into()),
                (
                    "helper".into(),
                    gomad_wasmhost::linked::SOURCE_INPUTS.into(),
                ),
            ]),
            executable_sha256: "e".repeat(64),
        },
    }
}
#[test]
fn diagnostic_admission_command_every_argument_is_bound() {
    let role = Role::Diagnostic(features(&["diagnostic-profile", "guest-profile"]));
    let receipt = command_receipt(role.clone());
    assert!(exact_cargo_command(&receipt, &role));
    assert_eq!(
        receipt.command,
        features(&[
            "/pinned/cargo",
            "build",
            "--manifest-path",
            "/crate/Cargo.toml",
            "--release",
            "--locked",
            "--jobs=1",
            "--no-default-features",
            "--message-format=json",
            "--bin",
            "gomad-measure",
            "--features",
            "helper-diagnostic-profile,helper-guest-profile"
        ])
    );
    for index in 0..receipt.command.len() {
        let mut changed = receipt.clone();
        changed.command[index] = "wrong".into();
        assert!(!exact_cargo_command(&changed, &role), "argument {index}");
    }
}
#[test]
fn identity_role_features_source_and_executable_are_bound() {
    let role = Role::Diagnostic(features(&["diagnostic-profile", "guest-profile"]));
    let identity = command_receipt(role.clone()).identity;
    assert!(validate_identity(&identity, &role, &"e".repeat(64)).is_ok());
    for index in 0..7 {
        let mut changed = identity.clone();
        match index {
            0 => changed.role = "ordinary".into(),
            1 => changed.features.clear(),
            2 => changed.source_inputs.remove("host").map(|_| ()).unwrap(),
            3 => changed.executable_sha256 = "f".repeat(64),
            4 => changed.host_implementation_sha256.clear(),
            5 => changed.linked_implementation_sha256.clear(),
            _ => changed.schema = "foreign/v1".into(),
        }
        assert!(validate_identity(&changed, &role, &"e".repeat(64)).is_err());
    }
}
#[test]
fn duplicate_compiler_environment_json_is_refused() {
    #[derive(Deserialize)]
    struct Environment {
        #[serde(deserialize_with = "unique_environment")]
        environment: BTreeMap<String, String>,
    }
    let good: Environment =
        serde_json::from_str(r#"{"environment":{"RUSTC":"/pinned/rustc","RUSTC_WRAPPER":""}}"#)
            .unwrap();
    assert_eq!(good.environment["RUSTC"], "/pinned/rustc");
    assert!(
        serde_json::from_str::<Environment>(
            r#"{"environment":{"RUSTC":"/pinned/rustc","RUSTC":"/other/rustc"}}"#
        )
        .is_err()
    );
}

#[expect(
    clippy::too_many_lines,
    reason = "Keep the bounded command or admission transaction and its refusal order together."
)]
fn synthetic_receipts(
    root: &Path,
    role: Role,
    raw_command_mutation: bool,
) -> (CheckoutProof, PathBuf, BuildReceipt) {
    let selected = root.join("selected");
    let actual = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    for (relative, manifest) in [
        ("tools/gomad/rust/gomad", inputs::SOURCE_INPUTS),
        (
            "tools/gomad/rust/wasmhost",
            gomad_wasmhost::linked::SOURCE_INPUTS,
        ),
    ] {
        for line in manifest.lines() {
            let (_, name) = line.split_once('\t').unwrap();
            let from = actual.join(relative).join(name);
            let to = selected.join(relative).join(name);
            fs::create_dir_all(to.parent().unwrap()).unwrap();
            fs::copy(from, to).unwrap();
        }
    }
    let proof = inputs::fixture_checkout(selected);
    let crate_root = proof.repo().join("tools/gomad/rust/gomad");
    let build = root.join("raw-build");
    fs::create_dir_all(build.join("target/release")).unwrap();
    let target = build.join("target");
    let artifact = target.join("release/gomad-measure");
    let helper = root.join("admitted-helper");
    let rustc = root.join("rustc");
    let cargo = root.join("cargo");
    for path in [&artifact, &helper, &rustc, &cargo] {
        fs::write(path, b"synthetic receipt fixture; never executed").unwrap();
        fs::set_permissions(path, fs::Permissions::from_mode(0o755)).unwrap();
    }
    let environment = BTreeMap::from([
        (
            "HOME".into(),
            root.join("home").to_string_lossy().into_owned(),
        ),
        (
            "CARGO_TARGET_DIR".into(),
            target.to_string_lossy().into_owned(),
        ),
        ("RUSTC".into(), rustc.to_string_lossy().into_owned()),
        ("RUSTC_WRAPPER".into(), String::new()),
        ("RUSTC_WORKSPACE_WRAPPER".into(), String::new()),
    ]);
    let command = cargo_command(&cargo, &crate_root, &role).unwrap();
    let mut artifact_frame = cargo_frame(&cargo_features(&role).unwrap(), "3", false);
    artifact_frame["executable"] = serde_json::to_value(&artifact).unwrap();
    fs::write(
        build.join("cargo-build.stdout"),
        stream(&artifact_frame, Some(true)),
    )
    .unwrap();
    let mut raw_command = command.clone();
    if raw_command_mutation {
        raw_command[6] = "--jobs=8".into();
    }
    write_json(&build.join("cargo-build.json"),&serde_json::json!({"command":raw_command,"environment":environment,"directory":crate_root,"exit_code":0})).unwrap();
    let rustc_version = "rustc 1.99.1 synthetic receipt fixture".to_string();
    let cargo_version = "cargo 1.99.1 synthetic receipt fixture".to_string();
    fs::write(build.join("rustc-version.stdout"), &rustc_version).unwrap();
    fs::write(build.join("cargo-version.stdout"), &cargo_version).unwrap();
    let identity = NativeIdentity {
        schema: "gomad.native-measurement-identity/v1".into(),
        role: role.identity_role().unwrap().into(),
        features: role.features().unwrap(),
        host_implementation_sha256: "a".repeat(64),
        linked_implementation_sha256: "b".repeat(64),
        runtime_implementation_sha256: env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into(),
        configuration: format!(
            "{}-{}",
            gomad_wasmhost::linked::CONFIGURATION,
            role.identity_role().unwrap()
        ),
        source_inputs: BTreeMap::from([
            ("host".into(), inputs::SOURCE_INPUTS.into()),
            (
                "helper".into(),
                gomad_wasmhost::linked::SOURCE_INPUTS.into(),
            ),
        ]),
        executable_sha256: hash(&artifact).unwrap(),
    };
    write_json(&build.join("artifact-identity.stdout"), &identity).unwrap();
    let mut inputs = semantic_inputs(&proof, &environment).unwrap();
    inputs.insert(rustc.clone(), hash(&rustc).unwrap());
    inputs.insert(cargo.clone(), hash(&cargo).unwrap());
    for label in [
        "rustc-path",
        "cargo-path",
        "rustc-version",
        "cargo-version",
        "cargo-build",
        "artifact-identity",
    ] {
        for suffix in ["stdout", "stderr", "json"] {
            let path = build.join(format!("{label}.{suffix}"));
            if !path.exists() {
                fs::write(&path, b"synthetic retained process fixture").unwrap();
            }
            inputs.insert(path.clone(), hash(&path).unwrap());
        }
    }
    let receipt = BuildReceipt {
        schema: BUILD_SCHEMA.into(),
        crate_root,
        executable: artifact,
        executable_sha256: identity.executable_sha256.clone(),
        rustc,
        cargo,
        inputs,
        command,
        environment,
        role,
        rustc_version,
        cargo_version,
        identity,
    };
    (proof, helper, receipt)
}
fn publish_synthetic_admission(
    root: &Path,
    helper: &Path,
    receipt: &BuildReceipt,
) -> (PathBuf, PathBuf) {
    let raw = root.join("helper-build.json");
    write_json(&raw, receipt).unwrap();
    let admission = Admission {
        schema: ADMISSION_SCHEMA.into(),
        build_receipt: raw.clone(),
        build_receipt_sha256: hash(&raw).unwrap(),
        cargo_artifact: receipt.executable.clone(),
        cargo_artifact_sha256: receipt.executable_sha256.clone(),
        executable: helper.into(),
        executable_sha256: hash(helper).unwrap(),
        identity: receipt.identity.clone(),
    };
    let path = root.join("helper-admission.json");
    write_json(&path, &admission).unwrap();
    (raw, path)
}
#[test]
fn diagnostic_admission_helper_evidence_full_mutation_matrix() {
    for mutation in [
        "valid",
        "cluster-valid",
        "cluster-extra-feature",
        "artifact",
        "helper",
        "source",
        "compiler",
        "receipt",
        "wrapper",
        "wrong-compiler",
        "old-toolchain",
        "missing-feature",
        "extra-feature",
        "raw-command",
    ] {
        let temp = Temporary::new();
        let role = if mutation == "cluster-valid" {
            Role::Warm
        } else {
            Role::Diagnostic(features(&["diagnostic-profile", "guest-profile"]))
        };
        let expected = if mutation.starts_with("cluster-") {
            Role::Warm
        } else {
            Role::Diagnostic(features(&["diagnostic-profile", "guest-profile"]))
        };
        let (proof, helper, mut receipt) =
            synthetic_receipts(&temp.0, role, mutation == "raw-command");
        match mutation {
            "wrapper" => {
                receipt
                    .environment
                    .insert("RUSTC_WRAPPER".into(), "/wrong".into());
            }
            "wrong-compiler" => {
                receipt.environment.insert("RUSTC".into(), "/wrong".into());
            }
            "old-toolchain" => {
                receipt.rustc_version = "rustc 1.94.1 fixture".into();
                receipt.cargo_version = "cargo 1.94.1 fixture".into();
            }
            "missing-feature" => receipt.role = Role::Diagnostic(features(&["guest-profile"])),
            "extra-feature" => {
                receipt.role = Role::Diagnostic(features(&[
                    "diagnostic-profile",
                    "guest-profile",
                    "unexpected",
                ]));
            }
            _ => {}
        }
        let (raw, admission) = publish_synthetic_admission(&temp.0, &helper, &receipt);
        let changed = match mutation {
            "artifact" => Some(receipt.executable.clone()),
            "helper" => Some(helper.clone()),
            "source" => Some(receipt.crate_root.join("src/lib.rs")),
            "compiler" => Some(receipt.rustc.clone()),
            "receipt" => Some(raw),
            _ => None,
        };
        if let Some(path) = changed {
            fs::write(path, b"tampered").unwrap();
        }
        let result = validate_admission(&proof, &admission, &expected);
        assert_eq!(
            result.is_ok(),
            matches!(mutation, "valid" | "cluster-valid"),
            "{mutation}: {result:?}"
        );
    }
}
#[test]
fn native_admission_refusal_order_and_binding_variants() {
    for mutation in [
        "same-artifact",
        "wrong-schema",
        "wrong-admission-identity",
        "linked-helper",
        "helper-mode",
        "omitted-source",
        "raw-cargo-features",
        "missing-build-log",
        "new-source",
        "source-and-artifact",
    ] {
        let temp = Temporary::new();
        let role = Role::Warm;
        let (proof, helper, mut receipt) = synthetic_receipts(&temp.0, role.clone(), false);
        if mutation == "omitted-source" {
            receipt
                .inputs
                .remove(&receipt.crate_root.join("src/lib.rs"));
        }
        if mutation == "missing-build-log" {
            let target = Path::new(&receipt.environment["CARGO_TARGET_DIR"]);
            receipt
                .inputs
                .remove(&target.parent().unwrap().join("cargo-build.stdout"));
        }
        let (_, path) = publish_synthetic_admission(&temp.0, &helper, &receipt);
        let mut admission: Admission =
            serde_json::from_slice(&bounded(&path, LIMIT).unwrap()).unwrap();
        match mutation {
            "same-artifact" => admission.executable = admission.cargo_artifact.clone(),
            "wrong-schema" => admission.schema = "foreign/v1".into(),
            "wrong-admission-identity" => admission.identity.role = "ordinary".into(),
            "linked-helper" => fs::hard_link(&helper, temp.0.join("helper-alias")).unwrap(),
            "helper-mode" => {
                fs::set_permissions(&helper, fs::Permissions::from_mode(0o700)).unwrap();
            }
            "new-source" => {
                fs::write(receipt.crate_root.join("src/injected.rs"), b"additional").unwrap();
            }
            "source-and-artifact" => {
                fs::write(receipt.crate_root.join("src/lib.rs"), b"source tampered").unwrap();
                fs::write(&receipt.executable, b"artifact tampered").unwrap();
            }
            "raw-cargo-features" => {
                let target = Path::new(&receipt.environment["CARGO_TARGET_DIR"]);
                let raw = target.parent().unwrap().join("cargo-build.stdout");
                fs::write(raw, stream(&cargo_frame(&[], "3", false), Some(true))).unwrap();
            }
            _ => {}
        }
        fs::write(&path, serde_json::to_vec(&admission).unwrap()).unwrap();
        let result = validate_admission(&proof, &path, &role);
        assert!(result.is_err(), "{mutation}");
        if mutation == "source-and-artifact" {
            assert!(
                result.unwrap_err().contains("source closure changed"),
                "source refusal must precede raw Cargo artifact checks"
            );
        }
    }
}

#[test]
fn bounded_process_observations_survive_exit_signal_timeout_capacity_and_spawn_failure() {
    for (label, argv, timeout, limit, termination) in [
        (
            "exit",
            features(&["/bin/sh", "-c", "printf out; printf err >&2; exit 7"]),
            Duration::from_secs(10),
            1024,
            "exit_failure",
        ),
        (
            "signal",
            features(&["/bin/sh", "-c", "kill -TERM $$"]),
            Duration::from_secs(10),
            1024,
            "signal",
        ),
        (
            "timeout",
            features(&["/bin/sh", "-c", "while :; do :; done"]),
            Duration::from_millis(1),
            1024,
            "timeout",
        ),
        (
            "capacity",
            features(&[
                "/bin/sh",
                "-c",
                "printf 12345678901234567890; printf abcdefghijklmnopqrstuvwxyz >&2",
            ]),
            Duration::from_secs(10),
            8,
            "output_capacity",
        ),
        (
            "spawn",
            features(&["/definitely-absent-native-compiler"]),
            Duration::from_secs(10),
            1024,
            "infrastructure_failure",
        ),
    ] {
        let temp = Temporary::new();
        let observed = observe_command(
            &temp.0,
            label,
            &temp.0,
            &BTreeMap::new(),
            &argv,
            timeout,
            limit,
        );
        assert_eq!(observed.termination, termination, "{observed:?}");
        assert!(observed.error.is_some());
        assert!(observed.resource_usage.is_none());
        let retained: CommandObservation =
            serde_json::from_slice(&bounded(&temp.0.join(format!("{label}.json")), LIMIT).unwrap())
                .unwrap();
        assert_eq!(retained.termination, termination);
        assert_eq!(retained.command, argv);
        assert_eq!(retained.pid, observed.pid);
        for captured in [&retained.stdout, &retained.stderr] {
            if let Some(digest) = &captured.sha256 {
                let bytes = bounded(&captured.path, limit).unwrap();
                assert_eq!(canonical::digest(&bytes), *digest);
                assert_eq!(Some(bytes.len() as u64), captured.captured_bytes);
                assert!(captured.bytes.unwrap() >= captured.captured_bytes.unwrap());
            }
        }
        if label == "exit" {
            assert_eq!(retained.exit_code, Some(7));
            assert!(retained.pid.is_some());
        }
        if label == "signal" {
            assert_eq!(retained.signal, Some(15));
        }
        if label == "capacity" {
            // The fixture writes stdout before stderr and the observer kills the
            // group as soon as either stream passes the bound, so stdout is
            // always over capacity while stderr holds either nothing or all of
            // its write, depending on whether the kill preceded it.
            assert!(retained.stdout.truncated);
            assert_eq!(retained.stdout.bytes, Some(20));
            assert_eq!(retained.stdout.captured_bytes, Some(8));
            match retained.stderr.bytes {
                Some(26) => {
                    assert!(retained.stderr.truncated);
                    assert_eq!(retained.stderr.captured_bytes, Some(8));
                }
                Some(0) => {
                    assert!(!retained.stderr.truncated);
                    assert_eq!(retained.stderr.captured_bytes, Some(0));
                }
                other => panic!("stderr was split by the kill: {other:?}"),
            }
        }
        if label == "spawn" {
            assert!(retained.pid.is_none());
        }
    }
}

#[test]
fn historical_native_admission_binds_explicit_source_without_current_identity() {
    // Synthetic receipt fixtures exercise validation only. They are never build or run evidence.
    let temp = Temporary::new();
    let role = Role::Warm;
    let (proof, helper, mut receipt) = synthetic_receipts(&temp.0, role.clone(), false);
    receipt.identity.runtime_implementation_sha256 = "c".repeat(64);
    let build = Path::new(&receipt.environment["CARGO_TARGET_DIR"])
        .parent()
        .unwrap();
    let raw_identity = build.join("artifact-identity.stdout");
    fs::write(
        &raw_identity,
        canonical::encode(&serde_json::to_value(&receipt.identity).unwrap()).unwrap(),
    )
    .unwrap();
    receipt
        .inputs
        .insert(raw_identity.clone(), hash(&raw_identity).unwrap());
    let (_, path) = publish_synthetic_admission(&temp.0, &helper, &receipt);
    let historical = validate_historical_admission(proof.repo(), &path, &role).unwrap();
    assert_eq!(
        historical.identity.runtime_implementation_sha256,
        "c".repeat(64)
    );
    assert!(validate_admission(&proof, &path, &role).is_err());
    assert!(validate_historical_admission(Path::new("relative"), &path, &role).is_err());
    assert!(validate_historical_admission(&temp.0.join("other-source"), &path, &role).is_err());
    assert!(validate_historical_admission(proof.repo(), &path, &Role::Release).is_err());
    fs::write(receipt.crate_root.join("src/lib.rs"), b"changed source").unwrap();
    assert!(validate_historical_admission(proof.repo(), &path, &role).is_err());
}

#[test]
fn historical_native_admission_does_not_trust_claimed_relocated_checkout() {
    let temp = Temporary::new();
    let role = Role::Warm;
    let (proof, helper, receipt) = synthetic_receipts(&temp.0, role.clone(), false);
    let (_, path) = publish_synthetic_admission(&temp.0, &helper, &receipt);
    assert!(validate_historical_admission(proof.repo(), &path, &role).is_ok());
    let relocated = temp.0.join("relocated");
    fs::create_dir_all(&relocated).unwrap();
    std::os::unix::fs::symlink(proof.repo().join("tools"), relocated.join("tools")).unwrap();
    assert!(validate_historical_admission(&relocated, &path, &role).is_err());
}

#[test]
fn actual_child_records_ancestry_and_enclosed_monotonic_spawn_to_reap() {
    let temp = Temporary::new();
    let before = gomad_wasmhost::linked::monotonic_nanos().unwrap();
    let observed = observe_command(
        &temp.0,
        "ancestry",
        &temp.0,
        &BTreeMap::new(),
        &features(&["/bin/sh", "-c", "printf '%s\\n' \"$PPID\""]),
        Duration::from_secs(5),
        1024,
    );
    let after = gomad_wasmhost::linked::monotonic_nanos().unwrap();
    assert_eq!(observed.error, None);
    assert_eq!(observed.exit_code, Some(0));
    assert_eq!(observed.signal, None);
    assert_eq!(observed.clock.as_deref(), Some("CLOCK_MONOTONIC"));
    let parent = std::process::id();
    assert_eq!(observed.parent_pid, Some(parent));
    assert_ne!(observed.pid, Some(parent));
    assert_eq!(
        fs::read_to_string(&observed.stdout.path).unwrap().trim(),
        parent.to_string()
    );
    let start = observed.started_monotonic_nanos.unwrap();
    let reaped = observed.reaped_monotonic_nanos.unwrap();
    assert!(before <= start && start <= reaped && reaped <= after);
}
