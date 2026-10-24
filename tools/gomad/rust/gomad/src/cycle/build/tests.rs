use super::*;
use serde_json::json;

#[test]
fn dry_run_state_and_fixed_role_commands() {
    let mut dry = DryRun::default();
    dry.enable(&json!({}));
    assert!(!dry.enabled());
    for value in [json!({}), json!({"dry_run":false}), json!({"dry_run":1})] {
        assert!(!dry.enable(&value));
        assert!(!dry.enabled());
    }
    assert!(dry.enable(&json!({"dry_run":true})));
    assert!(dry.enabled());
    assert!(!dry.enable(&json!({"dry_run":false})));
    let repo = Path::new("/checkout");
    let recipes: [(&str, &str, &str, &[&str], &str); 13] = [
        ("release", "build", "gomad", &["--bin", "gomad"], ""),
        (
            "host-fixture",
            "build",
            "gomad",
            &["--bin", "gomad"],
            "determinism-fixture",
        ),
        (
            "ordinary-helper-test",
            "test",
            "gomad-wasmhost",
            &["--lib", "--no-run"],
            "",
        ),
        (
            "ordinary-host-test",
            "test",
            "gomad",
            &["--lib", "--no-run"],
            "",
        ),
        (
            "stress",
            "build",
            "gomad",
            &["--bin", "gomad"],
            "determinism-fixture,helper-stress-jitter",
        ),
        (
            "stress-release",
            "build",
            "gomad",
            &["--bin", "gomad"],
            "helper-stress-jitter",
        ),
        (
            "conformance-reference",
            "build",
            "gomad-wasmhost",
            &["--bin", "gomad-wasmhost"],
            "conformance",
        ),
        (
            "reference-transport",
            "build",
            "gomad-wasmhost",
            &["--bin", "gomad-wasmhost"],
            "reference-helper",
        ),
        (
            "runtime-harness",
            "build",
            "gomad",
            &["--example", "runtime-qualification-replay"],
            "",
        ),
        (
            "interception-harness",
            "build",
            "gomad",
            &["--example", "interception-streaming-replay"],
            "",
        ),
        (
            "fn222-support",
            "build",
            "gomad",
            &["--example", "fn222-support"],
            "",
        ),
        (
            "counterfactual-fixture",
            "build",
            "gomad",
            &["--bin", "gomad"],
            "determinism-fixture,helper-determinism-fixture",
        ),
        (
            "counterfactual-harness",
            "test",
            "gomad",
            &["--lib", "--no-run"],
            "determinism-fixture,helper-determinism-fixture",
        ),
    ];
    for (name, verb, package, target, features) in recipes {
        let mut expected = vec![
            "/cargo",
            verb,
            "--release",
            "--locked",
            "--no-default-features",
            "-p",
            package,
        ];
        expected.extend_from_slice(target);
        expected.extend([
            "--message-format=json",
            "--manifest-path",
            "/checkout/tools/gomad/rust/Cargo.toml",
        ]);
        if !features.is_empty() {
            expected.extend(["--features", features]);
        }
        assert_eq!(
            cargo_command(Path::new("/cargo"), repo, Role::parse(name).unwrap()),
            expected,
            "{name}"
        );
    }
    assert!(Role::parse("foreign").unwrap_err().contains("unknown role"));
    assert_eq!(
        Role::parse("conformance-reference")
            .unwrap()
            .helper_features(),
        vec!["conformance", "reference-helper"]
    );
    assert_eq!(
        Role::parse("counterfactual-harness")
            .unwrap()
            .helper_features(),
        vec!["determinism-fixture"]
    );
}

#[test]
fn record_probe_preserves_boolean_refusal_contract() {
    let root = std::env::temp_dir().join(format!("gomad-build-probe-{}", std::process::id()));
    std::fs::create_dir_all(&root).unwrap();
    let path = root.join("record.json");
    for (bytes, expected) in [
        (b"{\"schema\":\"gomad.dry-run-build/v1\"}".as_slice(), true),
        (b"{\"schema\":\"foreign\"}".as_slice(), false),
        (b"[]".as_slice(), false),
        (b"invalid".as_slice(), false),
        (
            b"{\"schema\":\"gomad.dry-run-build/v1\",\"schema\":\"foreign\"}".as_slice(),
            false,
        ),
    ] {
        std::fs::write(&path, bytes).unwrap();
        assert_eq!(is_record(&path), expected);
    }
    std::fs::write(&path, b"{\"schema\":\"gomad.dry-run-build/v1\"}").unwrap();
    let alias = root.join("alias.json");
    std::os::unix::fs::symlink(&path, &alias).unwrap();
    assert!(is_record(&alias));
    std::fs::remove_file(&alias).unwrap();
    let relative = Path::new("tools/gomad/rust/gomad/src/cycle/build/../../../../../Cargo.toml");
    assert!(!is_record(relative));
    let cwd = std::env::current_dir().unwrap();
    let relative_name = PathBuf::from("../".repeat(cwd.components().count() - 1))
        .join(path.strip_prefix("/").unwrap());
    assert!(is_record(&relative_name));
    std::fs::remove_file(&path).unwrap();
    assert!(!is_record(&path));
    std::fs::remove_dir(&root).unwrap();
}

fn witness(path: &Path) -> serde_json::Value {
    use std::os::unix::fs::MetadataExt as _;
    let bytes = read_file(path, FILE_BOUND).unwrap();
    let stat = std::fs::metadata(path).unwrap();
    json!({"raw_file":path,"sha256":digest(&bytes),"bytes":bytes.len(),"stat":{"dev":stat.dev(),"ino":stat.ino(),"uid":stat.uid(),"mode":stat.mode(),"mtime":[stat.mtime(),stat.mtime_nsec()],"ctime":[stat.ctime(),stat.ctime_nsec()]}})
}
fn retain(root: &Path, name: &str, bytes: &[u8]) -> serde_json::Value {
    let directory = root.join("mutants");
    std::fs::create_dir_all(&directory).unwrap();
    let path = directory.join(name);
    std::fs::write(&path, bytes).unwrap();
    witness(&path)
}
fn encoded_frames(frames: &[serde_json::Value]) -> Vec<u8> {
    frames
        .iter()
        .flat_map(|v| {
            let mut row = serde_json::to_vec(v).unwrap();
            row.push(b'\n');
            row
        })
        .collect()
}
fn report(id: &str, mutation: &str, error: &str, facts: &serde_json::Value) {
    eprintln!(
        "BUILD_PROOF {}",
        json!({"id":id,"mutation":mutation,"observed_error":error,"valid_control":"returned","source":"tools/gomad/rust/gomad/src/cycle/build/mod.rs","inputs":facts,"input_sha256":digest(&crate::evidence::canonical::encode(facts).unwrap()),"test_source_sha256":digest(include_bytes!("tests.rs")),"source_sha256":digest(include_bytes!("mod.rs")),"identity_source_sha256":digest(include_bytes!("identity.rs"))})
    );
}

pub(crate) fn run_cases(owner: &super::super::process::CycleOwner) -> Result<(), String> {
    let root = PathBuf::from(std::env::var("GOMAD_BUILD_CASE_ROOT").map_err(|e| e.to_string())?);
    let target =
        PathBuf::from(std::env::var("GOMAD_BUILD_CASE_TARGET").map_err(|e| e.to_string())?);
    let cargo = PathBuf::from(std::env::var("GOMAD_BUILD_CASE_CARGO").map_err(|e| e.to_string())?);
    std::fs::create_dir_all(&root).map_err(|e| e.to_string())?;
    let mut runner = Runner::create(
        owner,
        &root.join("commands"),
        Some("cycle-build-proof".into()),
    )?;
    let cancelled = AtomicBool::new(false);
    let repo = default_repo()?;
    if std::env::var("GOMAD_BUILD_CASE_MULTI_HOST").is_ok_and(|value| value == "1") {
        return shared_target_host_roles(&root, &target, &cargo, &mut runner, &cancelled, &repo);
    }
    let mut dry = DryRun::default();
    dry.enable(&json!({}));
    let path = DryRun::build(
        "release",
        &repo,
        &root.join("roles"),
        &cargo,
        &target,
        &mut runner,
        &cancelled,
        "cycle-build-proof",
    )?;
    for (role, role_root, executable, message) in [
        (
            "foreign",
            root.join("unknown"),
            cargo.clone(),
            "unknown role",
        ),
        (
            "release",
            root.join("roles"),
            cargo.clone(),
            "output must be fresh",
        ),
        (
            "release",
            root.join("failed"),
            PathBuf::from("/usr/bin/false"),
            "cargo build failed",
        ),
    ] {
        let error = DryRun::build(
            role,
            &repo,
            &role_root,
            &executable,
            &target,
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )
        .unwrap_err();
        assert!(error.contains(message), "{role}: {error}");
        if message == "cargo build failed" {
            assert!(error.contains("exit 1"), "{error}");
        }
        report(
            if role == "foreign" {
                "orchestration:dry_run:126"
            } else if message == "output must be fresh" {
                "orchestration:dry_run:128"
            } else {
                "orchestration:dry_run:135"
            },
            message,
            &error,
            &json!({"control":{"role":"release","record":witness(&path),"argv":cargo_command(&cargo,&repo,Role::parse("release").unwrap())},"mutant":{"role":role,"root":role_root,"target":target,"argv":Role::parse(role).ok().map(|v|cargo_command(&executable,&repo,v))}}),
        );
    }
    let disabled_enabled = dry.enabled();
    let disabled_config = dry.config.clone();
    let error = dry
        .read_record(
            &path,
            None,
            None,
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )
        .unwrap_err();
    assert!(error.contains("outside a dry-run"), "{error}");
    let record = file_json(&path)?;
    let snapshot = Snapshot::observe(&repo, &mut runner, &cancelled, "cycle-build-proof")?;
    dry.enable(&json!({"dry_run":true,"repo":repo,"source_sha256":source_digest(&snapshot),"head":snapshot.head,"symlinks":snapshot.symlinks,"identity":{"host":record["host_sha256"],"helper":record["helper_sha256"]}}));
    let receipt = dry.read_record(
        &path,
        None,
        Some("release"),
        &mut runner,
        &cancelled,
        "cycle-build-proof",
    )?;
    report(
        "orchestration:dry_run:48",
        "configuration disabled",
        &error,
        &json!({"control":{"enabled":dry.enabled(),"config":dry.config,"record":witness(&path)},"mutant":{"enabled":disabled_enabled,"config":disabled_config}}),
    );
    let record_bytes = read_file(&path, JSON_BOUND)?;
    let cargo_path = path.parent().unwrap().join("cargo.stdout");
    let cargo_bytes = read_file(&cargo_path, JSON_BOUND)?;
    for (mutation, message) in [
        ("unfinished", "did not complete"),
        ("unsuccessful", "did not complete"),
        ("duplicate-finished", "did not complete"),
        ("trailing", "did not complete"),
        ("helper-missing", "linked build identity"),
        ("helper-duplicate", "linked build identity"),
        ("helper-foreign-duplicate", "linked build identity"),
        ("host-foreign-duplicate", "host build identity"),
        ("host-wrong", "host build identity"),
        ("artifact-missing", "Cargo artifact role"),
        ("artifact-duplicate", "Cargo artifact role"),
        ("artifact-features", "Cargo artifact role"),
        ("artifact-profile", "Cargo artifact role"),
    ] {
        let mutant_bytes = frame_mutation(&cargo_bytes, mutation);
        std::fs::write(&cargo_path, &mutant_bytes).map_err(|e| e.to_string())?;
        let result = dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        );
        std::fs::write(&cargo_path, &cargo_bytes).map_err(|e| e.to_string())?;
        let error = result.unwrap_err();
        assert!(error.contains(message), "{mutation}: {error}");
        report(
            match message {
                "did not complete" => "orchestration:dry_run:60",
                "linked build identity" => "orchestration:dry_run:64",
                "host build identity" => "orchestration:dry_run:91",
                _ => "orchestration:dry_run:79",
            },
            mutation,
            &error,
            &json!({"control":witness(&cargo_path),"mutant":retain(&root,&format!("cargo-{mutation}.stdout"),&mutant_bytes),"record":witness(&path)}),
        );
        dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )?;
    }
    for (field, changed, message) in [
        ("schema", json!("foreign"), "foreign dry-run"),
        ("role", json!("foreign"), "foreign dry-run"),
        ("binary_sha256", json!("0".repeat(64)), "binary changed"),
        ("features", json!(["foreign"]), "compiled role features"),
        (
            "helper_features",
            json!(["foreign"]),
            "compiled role features",
        ),
        (
            "host_sha256",
            json!("0".repeat(64)),
            "compiled implementation",
        ),
        (
            "runtime_implementation_sha256",
            json!("0".repeat(64)),
            "compiled implementation",
        ),
    ] {
        let mut record = strict_json(&record_bytes)?;
        record[field] = changed;
        std::fs::write(&path, serde_json::to_vec(&record).unwrap()).map_err(|e| e.to_string())?;
        let result = dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        );
        std::fs::write(&path, &record_bytes).map_err(|e| e.to_string())?;
        let error = result.unwrap_err();
        assert!(error.contains(message), "{field}: {error}");
        report(
            match message {
                "foreign dry-run" => "orchestration:dry_run:50",
                "binary changed" => "orchestration:dry_run:52",
                "compiled role features" => "orchestration:dry_run:55",
                _ => "orchestration:dry_run:86",
            },
            field,
            &error,
            &json!({"control_record_bytes":record_bytes,"mutant_record_bytes":serde_json::to_vec(&record).unwrap(),"field":field}),
        );
        dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )?;
    }
    let error = dry
        .read_record(
            &path,
            None,
            Some("host-fixture"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )
        .unwrap_err();
    assert!(error.contains("foreign dry-run"), "{error}");
    let original_frames = frames(&cargo_bytes)?;
    build_observation(&original_frames, Role::parse("release")?, &repo)?;
    for mutation in ["artifact-missing", "artifact-duplicate"] {
        let changed = frames(&frame_mutation(&cargo_bytes, mutation))?;
        let error = build_observation(&changed, Role::parse("release")?, &repo).unwrap_err();
        assert!(error.contains("artifact missing or ambiguous"));
        report(
            "orchestration:dry_run:144",
            mutation,
            &error,
            &json!({"control":witness(&cargo_path),"mutant":retain(&root,&format!("producer-{mutation}.stdout"),&encoded_frames(&changed)),"role":"release"}),
        );
    }
    let mut missing = original_frames.clone();
    for row in &mut missing {
        if row["reason"] == "build-script-executed" {
            row["env"] = json!([]);
        }
    }
    let error = build_observation(&missing, Role::parse("release")?, &repo).unwrap_err();
    assert!(error.contains("build identities not observed"));
    report(
        "orchestration:dry_run:160",
        "actual Cargo build-script identities removed",
        &error,
        &json!({"control":witness(&cargo_path),"mutant":retain(&root,"producer-identities-removed.stdout",&encoded_frames(&missing)),"role":"release"}),
    );
    let frames = frames(&cargo_bytes)?;
    let helper = frames
        .iter()
        .find(|v| {
            env_pairs(v)
                .unwrap_or_default()
                .contains_key("GOMAD_LINKED_IMPLEMENTATION_SHA256")
        })
        .unwrap();
    let out = PathBuf::from(helper["out_dir"].as_str().unwrap());
    for name in ["identity-compiler.txt", "identity-bindings.tsv"] {
        let context_path = out.join(name);
        let original = read_file(&context_path, JSON_BOUND)?;
        let mutant_context = if name == "identity-compiler.txt" {
            b"rustc foreign\n".as_slice()
        } else {
            b"PROFILE\tdebug\nTARGET\tforeign\n".as_slice()
        };
        std::fs::write(&context_path, mutant_context).map_err(|e| e.to_string())?;
        let result = dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        );
        std::fs::write(&context_path, &original).map_err(|e| e.to_string())?;
        let error = result.unwrap_err();
        assert!(error.contains("compiler/feature context"));
        report(
            "orchestration:dry_run:70",
            name,
            &error,
            &json!({"control":{"path":context_path,"bytes":original},"mutant":{"path":context_path,"bytes":mutant_context},"record":witness(&path)}),
        );
        dry.read_record(
            &path,
            None,
            Some("release"),
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )?;
    }
    let mut candidate = json!({"dry_run":true,"repo":repo,
        "files":receipt["inputs"], "head":receipt["head"],"source_sha256":receipt["source_sha256"],
        "symlinks":receipt["symlinks"],"identity":{"host":receipt["host_sha256"],"helper":receipt["helper_sha256"]},
        "binary":receipt["binary"],"binary_sha256":receipt["binary_sha256"]});
    let candidate_control = retain(
        &root,
        "candidate-control.json",
        &serde_json::to_vec(&candidate).unwrap(),
    );
    dry.read_record(
        &path,
        Some(&candidate),
        Some("release"),
        &mut runner,
        &cancelled,
        "cycle-build-proof",
    )?;
    for (key, message) in [
        ("source_sha256", "source custody"),
        ("head", "source custody"),
        ("binary", "release differs"),
        ("binary_sha256", "release differs"),
    ] {
        let mut changed = candidate.clone();
        changed[key] = json!("changed");
        let error = dry
            .read_record(
                &path,
                Some(&changed),
                None,
                &mut runner,
                &cancelled,
                "cycle-build-proof",
            )
            .unwrap_err();
        assert!(error.contains(message), "{key}: {error}");
        report(
            match message {
                "source custody" => "orchestration:dry_run:98",
                "candidate helper" => "orchestration:dry_run:101",
                "candidate host" => "orchestration:dry_run:103",
                _ => "orchestration:dry_run:107",
            },
            key,
            &error,
            &json!({"control":candidate_control,"mutant":retain(&root,&format!("candidate-{key}.json"),&serde_json::to_vec(&changed).unwrap()),"record":witness(&path)}),
        );
    }
    for (key, message) in [("helper", "candidate helper"), ("host", "candidate host")] {
        let mut changed = candidate.clone();
        changed["identity"][key] = json!("0".repeat(64));
        let error = dry
            .read_record(
                &path,
                Some(&changed),
                None,
                &mut runner,
                &cancelled,
                "cycle-build-proof",
            )
            .unwrap_err();
        assert!(error.contains(message), "{key}: {error}");
        report(
            match message {
                "source custody" => "orchestration:dry_run:98",
                "candidate helper" => "orchestration:dry_run:101",
                "candidate host" => "orchestration:dry_run:103",
                _ => "orchestration:dry_run:107",
            },
            key,
            &error,
            &json!({"control":candidate_control,"mutant":retain(&root,&format!("candidate-identity-{key}.json"),&serde_json::to_vec(&changed).unwrap()),"record":witness(&path)}),
        );
    }
    candidate["files"]
        .as_object_mut()
        .unwrap()
        .remove(path.to_str().unwrap());
    let error = dry
        .read_record(
            &path,
            Some(&candidate),
            None,
            &mut runner,
            &cancelled,
            "cycle-build-proof",
        )
        .unwrap_err();
    assert!(error.contains("incompletely sealed"), "{error}");
    report(
        "orchestration:dry_run:105",
        "missing record pin",
        &error,
        &json!({"control":candidate_control,"mutant":retain(&root,"candidate-incomplete-seal.json",&serde_json::to_vec(&candidate).unwrap()),"record":witness(&path)}),
    );
    let helper = DryRun::build(
        "reference-transport",
        &repo,
        &root.join("roles"),
        &cargo,
        &target,
        &mut runner,
        &cancelled,
        "cycle-build-proof",
    )?;
    let admitted = dry.read_record(
        &helper,
        None,
        Some("reference-transport"),
        &mut runner,
        &cancelled,
        "cycle-build-proof",
    )?;
    assert!(admitted["host_sha256"].is_null());
    assert_eq!(admitted["helper_features"], json!(["reference-helper"]));
    eprintln!("BUILD_CONTROL helper reference-transport returned");
    Ok(())
}

fn frame_mutation(bytes: &[u8], mutation: &str) -> Vec<u8> {
    let mut frames = frames(bytes).unwrap();
    match mutation {
        "unfinished" => {
            frames.pop();
        }
        "unsuccessful" => {
            frames.last_mut().unwrap()["success"] = json!(false);
        }
        "duplicate-finished" => {
            frames.push(frames.last().unwrap().clone());
        }
        "trailing" => {
            frames.push(json!({"reason":"unexpected"}));
        }
        "helper-missing" => {
            frames.retain(|v| {
                !env_pairs(v)
                    .unwrap_or_default()
                    .contains_key("GOMAD_LINKED_IMPLEMENTATION_SHA256")
            });
        }
        "helper-duplicate" | "helper-foreign-duplicate" => {
            let mut v = frames
                .iter()
                .find(|v| {
                    env_pairs(v)
                        .unwrap_or_default()
                        .contains_key("GOMAD_LINKED_IMPLEMENTATION_SHA256")
                })
                .unwrap()
                .clone();
            if mutation == "helper-foreign-duplicate" {
                v["package_id"] = json!("path+file:///foreign/wasmhost#gomad-wasmhost@0.1.0");
            }
            frames.insert(0, v);
        }
        "host-foreign-duplicate" => {
            let mut v = frames
                .iter()
                .find(|v| {
                    env_pairs(v)
                        .unwrap_or_default()
                        .contains_key("GOMAD_HOST_IMPLEMENTATION_SHA256")
                })
                .unwrap()
                .clone();
            v["package_id"] = json!("path+file:///foreign/gomad#0.1.0");
            frames.insert(0, v);
        }
        "host-wrong" => {
            let v = frames
                .iter_mut()
                .find(|v| {
                    env_pairs(v)
                        .unwrap_or_default()
                        .contains_key("GOMAD_HOST_IMPLEMENTATION_SHA256")
                })
                .unwrap();
            v["env"][0][1] = json!("0".repeat(64));
        }
        "artifact-missing" => {
            frames.retain(|v| v["executable"].as_str().is_none());
        }
        "artifact-duplicate" => {
            let v = frames
                .iter()
                .find(|v| v["executable"].as_str().is_some())
                .unwrap()
                .clone();
            frames.insert(0, v);
        }
        "artifact-features" | "artifact-profile" => {
            let v = frames
                .iter_mut()
                .find(|v| v["executable"].as_str().is_some())
                .unwrap();
            if mutation == "artifact-features" {
                v["features"] = json!(["foreign"]);
            } else {
                v["profile"]["opt_level"] = json!("0");
            }
        }
        _ => panic!("unknown mutation"),
    }
    let mut out = Vec::new();
    for frame in frames {
        out.extend(serde_json::to_vec(&frame).unwrap());
        out.push(b'\n');
    }
    out
}

fn shared_target_host_roles(
    root: &Path,
    target: &Path,
    cargo: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
    repo: &Path,
) -> Result<(), String> {
    let context = "cycle-build-proof";
    let mut dry = DryRun::default();
    let release = DryRun::build(
        "release",
        repo,
        &root.join("roles"),
        cargo,
        target,
        runner,
        cancelled,
        context,
    )?;
    let snapshot = Snapshot::observe(repo, runner, cancelled, context)?;
    let initial = file_json(&release)?;
    dry.enable(&json!({"dry_run":true,"repo":repo,"source_sha256":source_digest(&snapshot),"head":snapshot.head,"symlinks":snapshot.symlinks,"identity":{"host":initial["host_sha256"],"helper":initial["helper_sha256"]}}));
    let first = dry.read_record(&release, None, Some("release"), runner, cancelled, context)?;
    let release_binary = PathBuf::from(first["binary"].as_str().ok_or("release binary")?);
    let shared = target.join("release/gomad");
    let shared_before = witness(&shared);
    assert_eq!(shared_before["sha256"], first["binary_sha256"]);
    let fixture = DryRun::build(
        "host-fixture",
        repo,
        &root.join("roles"),
        cargo,
        target,
        runner,
        cancelled,
        context,
    )?;
    let shared_after = witness(&shared);
    assert_ne!(
        shared_before["sha256"], shared_after["sha256"],
        "second host role must actually overwrite shared Cargo executable"
    );
    assert_eq!(witness(&release_binary)["sha256"], first["binary_sha256"]);
    eprintln!(
        "SHARED_TARGET_OVERWRITE {}",
        json!({"shared_before":shared_before,"shared_after":shared_after,"release_record":witness(&release),"release_binary":witness(&release_binary),"fixture_record":witness(&fixture),"source_sha256":digest(include_bytes!("mod.rs")),"test_source_sha256":digest(include_bytes!("tests.rs"))})
    );
    let second = dry.read_record(
        &fixture,
        None,
        Some("host-fixture"),
        runner,
        cancelled,
        context,
    )?;
    let first_after =
        dry.read_record(&release, None, Some("release"), runner, cancelled, context)?;
    assert_eq!(first_after["binary_sha256"], first["binary_sha256"]);
    assert_eq!(shared_after["sha256"], second["binary_sha256"]);
    let mut files = first_after["inputs"]
        .as_object()
        .ok_or("release inputs")?
        .clone();
    for (path, sha) in second["inputs"].as_object().ok_or("fixture inputs")? {
        if let Some(previous) = files.insert(path.clone(), sha.clone()) {
            assert_eq!(previous, *sha, "role seals must coexist: {path}");
        }
    }
    assert!(
        !files.contains_key(shared.to_str().ok_or("shared target path")?),
        "mutable Cargo executable must not enter role seals"
    );
    let candidate = json!({"repo":repo,"files":files,"source_sha256":first_after["source_sha256"],"head":first_after["head"],"symlinks":first_after["symlinks"],"identity":{"host":first_after["host_sha256"],"helper":first_after["helper_sha256"]},"binary":first_after["binary"],"binary_sha256":first_after["binary_sha256"]});
    for (path, role) in [(&release, "release"), (&fixture, "host-fixture")] {
        dry.read_record(
            path,
            Some(&candidate),
            Some(role),
            runner,
            cancelled,
            context,
        )?;
    }
    published_copy_refusals(root, &release, &release_binary, &dry, runner, cancelled)?;
    eprintln!(
        "SHARED_TARGET_CONTROL {}",
        json!({"roles":["release","host-fixture"],"shared_overwritten":true,"both_admitted_after_second":true,"both_sealed_together":true,"source_sha256":digest(include_bytes!("mod.rs")),"test_source_sha256":digest(include_bytes!("tests.rs"))})
    );
    Ok(())
}

fn published_copy_refusals(
    root: &Path,
    record: &Path,
    binary: &Path,
    dry: &DryRun,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<(), String> {
    let original = read_file(record, JSON_BOUND)?;
    let control = retain(root, "publication-control.json", &original);
    for (field, value) in [
        ("sha256", json!("0".repeat(64))),
        ("executable", json!("/foreign/gomad")),
    ] {
        let mut mutant = strict_json(&original)?;
        mutant["cargo_artifact"][field] = value;
        let bytes = serde_json::to_vec(&mutant).map_err(|e| e.to_string())?;
        let inputs = json!({"control":control,"mutant":retain(root,&format!("publication-{field}.json"),&bytes)});
        fs::write(record, &bytes).map_err(|e| e.to_string())?;
        let result = dry.read_record(
            record,
            None,
            Some("release"),
            runner,
            cancelled,
            "cycle-build-proof",
        );
        fs::write(record, &original).map_err(|e| e.to_string())?;
        let error = result.unwrap_err();
        assert!(
            error.contains("published binary differs from actual Cargo artifact"),
            "{error}"
        );
        dry.read_record(
            record,
            None,
            Some("release"),
            runner,
            cancelled,
            "cycle-build-proof",
        )?;
        eprintln!(
            "SHARED_TARGET_REFUSAL {}",
            json!({"mutation":field,"observed_error":error,"inputs":inputs,"input_sha256":digest(&crate::evidence::canonical::encode(&inputs).unwrap()),"source_sha256":digest(include_bytes!("mod.rs")),"test_source_sha256":digest(include_bytes!("tests.rs"))})
        );
    }
    let original_binary = read_file(binary, FILE_BOUND)?;
    let mut mutant = original_binary.clone();
    mutant[0] ^= 1;
    let inputs =
        json!({"control":witness(binary),"mutant":retain(root,"private-binary-mutant",&mutant)});
    fs::write(binary, &mutant).map_err(|e| e.to_string())?;
    let result = dry.read_record(
        record,
        None,
        Some("release"),
        runner,
        cancelled,
        "cycle-build-proof",
    );
    fs::write(binary, &original_binary).map_err(|e| e.to_string())?;
    let error = result.unwrap_err();
    assert!(error.contains("binary changed after its build"), "{error}");
    dry.read_record(
        record,
        None,
        Some("release"),
        runner,
        cancelled,
        "cycle-build-proof",
    )?;
    eprintln!(
        "SHARED_TARGET_REFUSAL {}",
        json!({"mutation":"private binary bytes","observed_error":error,"inputs":inputs,"input_sha256":digest(&crate::evidence::canonical::encode(&inputs).unwrap()),"source_sha256":digest(include_bytes!("mod.rs")),"test_source_sha256":digest(include_bytes!("tests.rs"))})
    );
    Ok(())
}
