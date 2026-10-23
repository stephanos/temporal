use super::*;
use std::cell::Cell;

struct Temporary(PathBuf);
impl Temporary {
    fn source_checkout() -> Self {
        use std::os::unix::fs::MetadataExt;
        let compiled = Path::new(env!("CARGO_MANIFEST_DIR"));
        let device = std::fs::metadata(compiled.join("src/measurement/inputs.rs"))
            .unwrap()
            .dev();
        let temporary = Self::new();
        if std::fs::metadata(&temporary.0).unwrap().dev() == device {
            return temporary;
        }
        drop(temporary);
        Self::git_metadata(compiled.ancestors().nth(4).unwrap(), device)
    }
    fn git_metadata(repo: &Path, device: u64) -> Self {
        use std::os::unix::fs::MetadataExt;
        let output = std::process::Command::new("git")
            .args(["rev-parse", "--path-format=absolute", "--git-common-dir"])
            .current_dir(repo)
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "Git common-dir required for source fixture"
        );
        let common =
            std::fs::canonicalize(String::from_utf8(output.stdout).unwrap().trim()).unwrap();
        assert_eq!(
            std::fs::metadata(&common).unwrap().dev(),
            device,
            "closed source fixture requires temp_dir or Git common-dir on the compiled source filesystem"
        );
        let namespace = common.join("gomad-admission-test-tmp");
        match std::fs::DirBuilder::new().mode(0o700).create(&namespace) {
            Ok(()) => {}
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => {}
            Err(error) => panic!("private source fixture namespace: {error}"),
        }
        let pin = crate::evidence::store::open::pin(&namespace).unwrap();
        assert_eq!(pin.metadata().unwrap().dev(), device);
        let root = namespace.join(format!(
            "fixture-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .unwrap();
        Self(root)
    }
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-cluster-admission-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&root)
            .unwrap();
        Self(root)
    }
}
impl Drop for Temporary {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.0);
        if let Some(parent) = self.0.parent().filter(|parent| {
            parent
                .file_name()
                .is_some_and(|name| name == "gomad-admission-test-tmp")
        }) {
            let _ = std::fs::remove_dir(parent);
        }
    }
}

#[test]
fn original_four_phases_by_four_drifts_stop_dispatch_or_observation_at_boundary() {
    for replay in [false, true] {
        for incarnation in [1, 2] {
            let phase = LifecyclePhase {
                incarnation,
                handles: ["active", "standby"]
                    .map(|node| Handle {
                        node: node.into(),
                        incarnation,
                    })
                    .into(),
            };
            let label = phase_label(&phase, replay).unwrap();
            assert_eq!(
                label,
                match (incarnation, replay) {
                    (1, false) => "cluster-crash",
                    (2, false) => "cluster-reopened",
                    (1, true) => "cluster-prefix-replay",
                    _ => "cluster-full-replay",
                }
            );
            for drift in ["none", "before", "after", "resource"] {
                let directory = Temporary::new();
                let input = directory.0.join("module");
                let memory = directory.0.join("meminfo");
                std::fs::write(&input, b"admitted").unwrap();
                std::fs::write(&memory, b"MemAvailable: 8388608 kB\n").unwrap();
                let pins = BTreeMap::from([(input.clone(), canonical::digest(b"admitted"))]);
                if drift == "before" {
                    std::fs::write(&input, b"changed").unwrap();
                }
                if drift == "resource" {
                    std::fs::write(&memory, b"MemAvailable: 8388607 kB\n").unwrap();
                }
                let dispatched = Cell::new(false);
                let observed = Cell::new(false);
                let verify = || module::verify_inputs(&pins);
                let result = boundary(
                    true,
                    verify,
                    || {
                        let raw =
                            std::fs::read_to_string(&memory).map_err(|error| error.to_string())?;
                        let available = host::linux_available_memory(&raw)?;
                        if available < 8_u64 << 30 {
                            Err("at least 8GiB MemAvailable".into())
                        } else {
                            Ok(())
                        }
                    },
                    || Ok(()),
                )
                .and_then(|()| {
                    dispatched.set(true);
                    if drift == "after" {
                        std::fs::write(&input, b"changed").unwrap();
                    }
                    boundary(
                        false,
                        verify,
                        || panic!("post-boundary must not re-admit resource"),
                        || {
                            observed.set(true);
                            Ok(())
                        },
                    )
                });
                assert_eq!(result.is_ok(), drift == "none", "{label}/{drift}");
                assert_eq!(
                    dispatched.get(),
                    matches!(drift, "none" | "after"),
                    "{label}/{drift}"
                );
                assert_eq!(observed.get(), drift == "none", "{label}/{drift}");
                if let Err(error) = result {
                    assert!(
                        error.contains(if drift == "resource" {
                            "8GiB"
                        } else {
                            "changed"
                        }),
                        "{error}"
                    );
                }
            }
        }
    }
}

#[test]
fn separate_go_executable_pin_refuses_real_drift_before_every_phase_dispatch() {
    let directory = Temporary::new();
    let go = directory.0.join("go");
    let compile = directory.0.join("compile");
    let source = directory.0.join("fixture.go");
    for path in [&go, &compile, &source] {
        std::fs::write(path, b"admitted").unwrap();
    }
    let inputs = json!({"inputs":{"compiler_path":go,"compiler_sha256":canonical::digest(b"admitted"),
        "compiler":BTreeMap::from([(compile.clone(),canonical::digest(b"admitted"))]),
        "sources":BTreeMap::from([(source,canonical::digest(b"admitted"))])}});
    let pins = prepared_inputs(&inputs).unwrap();
    assert_eq!(pins[&compile], canonical::digest(b"admitted"));
    for replay in [false, true] {
        for incarnation in [1, 2] {
            let phase = LifecyclePhase {
                incarnation,
                handles: ["active", "standby"]
                    .map(|node| Handle {
                        node: node.into(),
                        incarnation,
                    })
                    .into(),
            };
            let label = phase_label(&phase, replay).unwrap();
            std::fs::write(&go, b"admitted").unwrap();
            module::verify_inputs(&pins).unwrap();
            std::fs::write(&go, b"changed executable").unwrap();
            let dispatched = Cell::new(false);
            let result = boundary(true, || module::verify_inputs(&pins), || Ok(()), || Ok(()))
                .map(|()| dispatched.set(true));
            assert!(
                result.is_err(),
                "{label}: separate Go executable drift dispatched the phase"
            );
            assert!(!dispatched.get(), "{label}: guest phase was dispatched");
        }
    }
}

#[test]
fn actual_host_drift_is_rejected_at_every_before_and_after_callback() {
    use std::os::unix::fs::MetadataExt;
    let directory = Temporary::new();
    let checkout = Temporary::source_checkout();
    isolated_checkout(&checkout.0);
    let repo = checkout.0.as_path();
    let proof = inputs::authenticate_checkout(repo).unwrap();
    let metadata_fixture = Temporary::git_metadata(repo, std::fs::metadata(repo).unwrap().dev());
    std::fs::write(
        metadata_fixture.0.join("metadata-is-not-source-status"),
        b"retained fixture",
    )
    .unwrap();
    let metadata_namespace = metadata_fixture.0.parent().unwrap().to_path_buf();
    let environment: Vec<_> = std::env::vars()
        .map(|(name, value)| format!("{name}={value}"))
        .collect();
    std::fs::DirBuilder::new()
        .mode(0o700)
        .create(directory.0.join("baseline"))
        .unwrap();
    let host = host::current(&directory.0.join("baseline"), &environment).unwrap();
    let source =
        host::source_snapshot(&proof, &directory.0.join("baseline"), &environment).unwrap();
    let baseline = Baseline::admitted(&host, &source).unwrap();
    drop(metadata_fixture);
    assert!(
        !metadata_namespace.exists(),
        "owned empty metadata namespace was not cleaned"
    );
    baseline
        .matches(&host, SourceIdentity::captured(&source).unwrap())
        .unwrap();
    retained_identity_controls(&baseline, &host, &source, &directory.0.join("baseline"));
    full_reader_controls(&baseline, &host, &source, &directory.0, repo);
    actual_drift_callbacks(repo, &baseline);
}

fn actual_drift_callbacks(repo: &Path, baseline: &Baseline) {
    let helper = std::env::current_exe().unwrap();
    let helper_sha256 = canonical::digest(&std::fs::read(&helper).unwrap());
    for replay in [false, true] {
        for incarnation in [1, 2] {
            for before in [true, false] {
                for drift in ["cpu", "source"] {
                    let output = Temporary::new();
                    let phase = LifecyclePhase {
                        incarnation,
                        handles: ["active", "standby"]
                            .map(|node| Handle {
                                node: node.into(),
                                incarnation,
                            })
                            .into(),
                    };
                    let label = phase_label(&phase, replay).unwrap();
                    let mut bad = baseline.clone();
                    let dirty = repo.join("untracked-phase-drift");
                    if drift == "cpu" {
                        bad.host
                            .witness
                            .cpuinfo
                            .push_str("\nnot the admitted physical CPU bytes\n");
                    } else {
                        std::fs::write(&dirty, b"real untracked source drift").unwrap();
                    }
                    let observer = Observer::new(
                        Spec {
                            schema: "gomad.cluster-phase-admission/v1".into(),
                            repo: repo.into(),
                            files: baseline.host_inputs.clone(),
                            helper: helper.clone(),
                            helper_sha256: helper_sha256.clone(),
                            role: build::Role::Release,
                            receipt: None,
                            memory_mode: String::new(),
                            replay,
                            baseline: bad,
                        },
                        &output.0,
                    )
                    .unwrap();
                    let result = observer.observe(&phase, before);
                    if drift == "source" {
                        std::fs::remove_file(&dirty).unwrap();
                        let receipt = output.0.join("phase-admission").join(format!(
                            "{label}-{}-identity/source.json",
                            if before { "before" } else { "after" }
                        ));
                        let source: SourceIdentity =
                            crate::qualify::spec::strict(&std::fs::read(receipt).unwrap()).unwrap();
                        assert!(source.dirty && source.status.contains("untracked-phase-drift"));
                    }
                    assert!(
                        result.is_err(),
                        "{label}/{before}/{drift}: drift published a phase observation"
                    );
                    assert!(
                        result
                            .unwrap_err()
                            .contains("CPU/boot/source identity changed")
                    );
                    assert!(
                        !output
                            .0
                            .join("phase-admission")
                            .join(format!(
                                "{label}-{}.json",
                                if before { "before" } else { "after" }
                            ))
                            .exists()
                    );
                }
            }
        }
    }
}

fn retained_identity_controls(
    baseline: &Baseline,
    host: &host::HostEvidence,
    source: &host::SourceSnapshot,
    identity_root: &Path,
) {
    let mut receipts = BTreeMap::new();
    for (name, value) in [
        ("host.json", serde_json::to_value(host).unwrap()),
        ("source.json", serde_json::to_value(source).unwrap()),
    ] {
        let bytes = canonical::encode(&value).unwrap();
        super::super::write(&identity_root.join(name), &bytes).unwrap();
        receipts.insert(name.to_owned(), canonical::digest(&bytes));
    }
    let receipt_value = json!(receipts);
    check_identity(baseline, identity_root, &receipt_value).unwrap();
    let witness = identity_root.join("host-witness.json");
    let original_witness = std::fs::read(&witness).unwrap();
    std::fs::write(&witness, b"{}").unwrap();
    assert!(check_identity(baseline, identity_root, &receipt_value).is_err());
    std::fs::write(&witness, original_witness).unwrap();
    check_identity(baseline, identity_root, &receipt_value).unwrap();
    for drift in ["os", "arch", "boot", "cpu", "head", "inventory", "status"] {
        let mut changed = baseline.clone();
        match drift {
            "os" => changed.host.os.push_str("changed"),
            "arch" => changed.host.arch.push_str("changed"),
            "boot" => changed.host.boot_id.push_str("changed"),
            "cpu" => changed.host.witness.cpuinfo.push_str("changed"),
            "head" => changed.source.head = "0".repeat(40),
            "inventory" => changed.source.inputs.clear(),
            _ => changed.source.status = " M changed".into(),
        }
        assert!(
            check_identity(&changed, identity_root, &receipt_value).is_err(),
            "{drift}"
        );
    }
}

fn full_reader_controls(
    baseline: &Baseline,
    host: &host::HostEvidence,
    source: &host::SourceSnapshot,
    producer: &Path,
    repo: &Path,
) {
    let evidence = producer.join("evidence");
    let output = evidence.join("sample");
    let phases = output.join("phase-admission");
    for path in [&evidence, &output, &phases] {
        std::fs::DirBuilder::new().mode(0o700).create(path).unwrap();
    }
    let pid = std::process::id();
    super::super::write(
        &evidence.join("sample-driver.json"),
        &canonical::encode(&json!({"pid":pid})).unwrap(),
    )
    .unwrap();
    let helper = std::env::current_exe().unwrap();
    let spec = Spec {
        schema: "gomad.cluster-phase-admission/v1".into(),
        repo: repo.into(),
        files: baseline.host_inputs.clone(),
        helper_sha256: canonical::digest(&std::fs::read(&helper).unwrap()),
        helper,
        role: build::Role::Release,
        receipt: None,
        memory_mode: String::new(),
        replay: false,
        baseline: baseline.clone(),
    };
    let spec_path = producer.join("sample-phase-inputs.json");
    let spec_bytes = canonical::encode(&serde_json::to_value(spec).unwrap()).unwrap();
    super::super::write(&spec_path, &spec_bytes).unwrap();
    for (index, phase) in ["cluster-crash", "cluster-reopened"]
        .into_iter()
        .enumerate()
    {
        for before in [true, false] {
            let suffix = if before { "before" } else { "after" };
            let identity = phases.join(format!("{phase}-{suffix}-identity"));
            std::fs::DirBuilder::new()
                .mode(0o700)
                .create(&identity)
                .unwrap();
            let mut captured: host::HostEvidence =
                serde_json::from_value(serde_json::to_value(host).unwrap()).unwrap();
            captured.inputs.clear();
            for (path, digest) in &host.inputs {
                let target = identity.join(path.file_name().unwrap());
                std::fs::copy(path, &target).unwrap();
                captured.inputs.insert(target, digest.clone());
            }
            let mut receipts = BTreeMap::new();
            for (name, value) in [
                ("host.json", serde_json::to_value(captured).unwrap()),
                ("source.json", serde_json::to_value(source).unwrap()),
            ] {
                let bytes = canonical::encode(&value).unwrap();
                super::super::write(&identity.join(name), &bytes).unwrap();
                receipts.insert(name, canonical::digest(&bytes));
            }
            let handles = ["active", "standby"].map(|node| Handle {
                node: node.into(),
                incarnation: index as u64 + 1,
            });
            let observation = json!({"schema":"gomad.cluster-phase-observation/v1", "phase":phase,"before":before,"passed":true,"process_id":pid,
                "unix_nanos":index*2+if before {1}else{2},"handles":handles,"identity_receipts":receipts});
            super::super::write(
                &phases.join(format!("{phase}-{suffix}.json")),
                &canonical::encode(&observation).unwrap(),
            )
            .unwrap();
        }
    }
    check_observations(&output, false, &spec_path).unwrap();
    for path in [
        &spec_path,
        &evidence.join("sample-driver.json"),
        &phases.join("cluster-crash-before.json"),
    ] {
        public_receipt_mode_refused(&output, &spec_path, path);
    }
    let decoy = evidence.join("sample-phase-inputs.json");
    super::super::write(&decoy, &spec_bytes).unwrap();
    assert!(check_observations(&output, false, &decoy).is_err());
    std::fs::remove_file(&spec_path).unwrap();
    assert!(
        check_observations(&output, false, &spec_path).is_err(),
        "decoy substituted missing producer spec"
    );
    std::os::unix::fs::symlink(&decoy, &spec_path).unwrap();
    assert!(check_observations(&output, false, &spec_path).is_err());
}

fn public_receipt_mode_refused(output: &Path, spec: &Path, path: &Path) {
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o644)).unwrap();
    assert!(check_observations(output, false, spec).is_err());
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600)).unwrap();
    check_observations(output, false, spec).unwrap();
}

fn isolated_checkout(repo: &Path) {
    let original = Path::new(env!("CARGO_MANIFEST_DIR"))
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
            let source = original.join(relative).join(name);
            let target = repo.join(relative).join(name);
            std::fs::create_dir_all(target.parent().unwrap()).unwrap();
            if relative.ends_with("/gomad") && name == "src/measurement/inputs.rs" {
                // Checkout authentication requires this inode; fixture accesses only read it.
                std::fs::hard_link(&source, &target).unwrap();
            } else {
                std::fs::copy(&source, &target).unwrap();
            }
        }
    }
    for args in [
        vec!["init", "--quiet"],
        vec![
            "-c",
            "core.excludesFile=/dev/null",
            "add",
            "--force",
            "--all",
        ],
        vec![
            "-c",
            "user.name=Gomad fixture",
            "-c",
            "user.email=fixture@example.invalid",
            "-c",
            "commit.gpgsign=false",
            "-c",
            "core.hooksPath=/dev/null",
            "commit",
            "--quiet",
            "-m",
            "Exact compiled source fixture",
        ],
    ] {
        let result = std::process::Command::new("git")
            .args(args)
            .current_dir(repo)
            .output()
            .unwrap();
        assert!(
            result.status.success(),
            "{}",
            String::from_utf8_lossy(&result.stderr)
        );
    }
}
