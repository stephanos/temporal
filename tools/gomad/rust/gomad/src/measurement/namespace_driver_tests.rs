use super::*;

#[test]
fn frozen_namespace_sample_names_separate_prime_record_and_replay() {
    assert_eq!(
        row("baseline", 0, -1, false, false).name,
        "seed0-baseline-rep-1"
    );
    assert_eq!(
        row("candidate", 1_000_003, 0, true, true).name,
        "seed1000003-candidate-rep0-replay"
    );
    assert_eq!(
        row("candidate", 3_000_009, 4, false, true).selection,
        ACTIVITY
    );
}

#[test]
fn namespace_driver_refuses_unclean_output_and_legacy_implicit_paths() {
    for path in ["relative", "/", "/absolute/../escape", "/absolute//double"] {
        assert!(clean(Path::new(path)).is_err(), "{path}");
    }
}

#[test]
fn native_digest_projection_requires_exact_canonical_payload_witness() {
    let raw = b"GMWT\x01\0\0\0".to_vec();
    let mut control = NativeRun {
        request: json!({"kind":"standalone","guest":{"model":{"configuration":{"digest_only":true}}}}),
        report: Value::Null,
        profile: Value::Null,
        root: evidence::NativeExecution {
            outcome: json!({"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":0,"peak_memory_bytes":0,"output_bytes":0,"local_model":{"transcript_sha256":canonical::digest(&raw),"transcript_bytes":raw.len()}}),
            streams: BTreeMap::new(),
            handle: None,
        },
        incarnations: Vec::new(),
        executed_steps: None,
        cutoff: None,
        modules: Vec::new(),
    };
    let mut retained = control.clone();
    retained.request["guest"]["model"]["configuration"]["digest_only"] = false.into();
    retained.root.streams.insert("transcript".into(), raw);
    digest_retention(&control, &retained).unwrap();
    control.root.outcome["local_model"]["transcript_sha256"] = "a".repeat(64).into();
    assert!(digest_retention(&control, &retained).is_err());
}

#[test]
fn dispatched_child_sample_reaches_parent_admission_without_a_nested_retention_store() {
    let path = std::env::temp_dir().join(format!("gomad-child-dispatch-{}", std::process::id()));
    fs::create_dir(&path).unwrap();
    let parent = super::super::retention::RetainedRun::begin(
        &path,
        "parent",
        &BTreeMap::new(),
        false,
        &|| false,
    )
    .unwrap();
    let sample = parent.path().join("sample");
    let call = Invocation {
        name: "namespace-sample".into(),
        repo: path.clone(),
        go_command: "go".into(),
        inputs: BTreeMap::from([
            (
                "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT".into(),
                sample.to_string_lossy().into_owned(),
            ),
            (
                "GOMAD_WASM_NAMESPACE_COMMON".into(),
                parent
                    .path()
                    .join("missing-common.json")
                    .to_string_lossy()
                    .into_owned(),
            ),
        ]),
        environ: Vec::new(),
        keep: false,
    };
    let proof = super::super::inputs::fixture_checkout(path.clone());
    let expected = run_sample(&call, &proof, &sample).unwrap_err();
    let mut stderr = Vec::new();
    assert_eq!(
        super::super::dispatch(call, proof, &mut Vec::new(), &mut stderr),
        1
    );
    assert!(String::from_utf8(stderr).unwrap().contains(&expected));
    assert!(!sample.exists());
    drop(parent);
    fs::remove_dir_all(path).unwrap();
}

#[test]
#[ignore = "requires an explicit pinned Go compiler and a fresh retained runtime custody root"]
fn actual_runtime_preparation_reconstructs_and_refuses_mutated_observations() {
    let root = PathBuf::from(std::env::var_os("GOMAD_TEST_MEASUREMENT_ROOT").unwrap());
    let compiler = std::env::var("GOMAD_TEST_PREPARATION_COMPILER").unwrap();
    fs::DirBuilder::new().mode(0o700).create(&root).unwrap();
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap()
        .to_owned();
    let proof = crate::measurement::inputs::authenticate_checkout(&repo).unwrap();
    let call = Invocation {
        name: "runtime-cost-profile".into(),
        repo,
        go_command: compiler.clone(),
        inputs: BTreeMap::from([
            ("GOMAD_STOCK_GO".into(), compiler),
            (
                "GOMAD_WASM_TOOLCHAIN_CACHE".into(),
                root.join("cache").to_string_lossy().into_owned(),
            ),
        ]),
        environ: std::env::vars()
            .map(|(key, value)| format!("{key}={value}"))
            .collect(),
        keep: true,
    };
    let prepared =
        module::prepare(&proof, &call, &root.join("module"), &Selection::Runtime).unwrap();
    module::authenticate_prepared(&prepared).unwrap();
    assert_eq!(
        prepared.native_input(0).unwrap()["guest"]["model"]["configuration"]["limit"],
        64_u64 << 20
    );
    let common = common(&prepared, prepared.module_path.clone(), &root).unwrap();
    let reconstructed = load_prepared(&common).unwrap();
    assert_eq!(reconstructed.guest.auxiliary, prepared.guest.auxiliary);
    let directory = root.join("module/preparation-observations");
    let key = prepared.guest.auxiliary.keys().next().unwrap();
    let path = directory.join(key);
    let original = fs::read(&path).unwrap();
    fs::write(&path, b"altered").unwrap();
    assert!(load_prepared(&common).is_err());
    fs::write(&path, original).unwrap();
    load_prepared(&common).unwrap();
    let saved = root.join("saved-observations");
    fs::rename(&directory, &saved).unwrap();
    assert!(load_prepared(&common).is_err());
    fs::rename(&saved, &directory).unwrap();
    load_prepared(&common).unwrap();
    println!(
        "retained runtime namespace custody probe: {}",
        root.display()
    );
}
