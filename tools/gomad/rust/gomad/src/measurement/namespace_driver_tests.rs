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
