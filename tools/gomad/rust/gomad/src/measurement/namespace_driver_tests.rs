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
