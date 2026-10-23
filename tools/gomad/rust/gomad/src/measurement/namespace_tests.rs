use super::super::evidence::NativeExecution;
use super::*;

fn raw() -> NativeRun {
    NativeRun {
        request: json!({"kind":"standalone","guest":{"module_sha256":super::super::workloads::hex(&identity().target_sha256),"model":{"configuration":{"seed":0}},"namespace":{"entries":[]}}}),
        report: json!({"linked_implementation_sha256":"b".repeat(64),"host_sha256":"d".repeat(64),"host_implementation_sha256":"e".repeat(64),"authorization_to_terminal_nanos":100,"diagnostic_only":true,"features":["diagnostic-profile"],"configuration":format!("{}-diagnostic-profile",gomad_wasmhost::linked::CONFIGURATION),"host_os":"linux","host_arch":"x86_64"}),
        profile: json!({"owners":{"namespace":{"count":1}},"abi_ops":{"wasi_snapshot_preview1.fd_read":{"served_by":{"namespace":1}}}}),
        root: NativeExecution {
            outcome: json!({"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":1,"peak_memory_bytes":1,"output_bytes":0,"local_model":{"calls":1,"advances":0}}),
            streams: BTreeMap::from([("transcript".into(), b"GMWT\x01\0\0\0".to_vec())]),
            handle: None,
        },
        incarnations: Vec::new(),
        executed_steps: None,
        cutoff: None,
        modules: Vec::new(),
    }
}
fn identity() -> gomad_wasmhost::linked::ChoiceIdentity {
    let fixture: Value = serde_json::from_str(include_str!(
        "../../../wasmhost/tests/testdata/replay_plan/empty.json"
    ))
    .unwrap();
    serde_json::from_value(fixture["identity"].clone()).unwrap()
}
fn common() -> Common {
    Common {
        requests: SEEDS
            .into_iter()
            .map(|seed| {
                let mut request = raw().request;
                request["guest"]["model"]["configuration"]["seed"] = seed.into();
                (seed, request)
            })
            .collect(),
        environment_identity: "environment".into(),
        runtime_identity: identity().toolchain_build_key,
        choice_identity: identity(),
        current_module_producer_sha256: "c".repeat(64),
        native_role: VariantRole::DiagnosticProfile,
    }
}
fn attach_counter(run: &mut NativeRun) {
    run.report["native_namespace_counter"] = json!({"schema":"gomad.native-namespace-counter/v1","implementation_sha256":"b".repeat(64),"fresh_store_id":1,"namespace_calls":1,"external_owner_calls":{},"counter_domain":COUNTER_DOMAIN,"namespace_boundary":NAMESPACE_BOUNDARY});
}

#[test]
fn missing_actual_counter_blocks_instead_of_inventing_empty_external_calls() {
    let error = collect(common(), Role::Candidate, |seed, _| {
        let mut run = raw();
        run.request["guest"]["model"]["configuration"]["seed"] = seed.into();
        Ok(Observation {
            run,
            fresh_store_id: 1,
            replay: None,
        })
    })
    .err()
    .unwrap();
    assert!(error.contains("actual native namespace/external-owner counter seam is absent"));
}

#[test]
fn actual_counter_envelope_requires_implementation_store_and_positive_native_attribution() {
    let mut run = raw();
    attach_counter(&mut run);
    assert!(counter(&run, 1).unwrap().is_empty());
    for mutation in 0..8 {
        let mut changed = run.clone();
        match mutation {
            0 => {
                changed.report["native_namespace_counter"]["implementation_sha256"] =
                    "d".repeat(64).into();
            }
            1 => changed.report["native_namespace_counter"]["fresh_store_id"] = 2.into(),
            2 => changed.report["native_namespace_counter"]["namespace_calls"] = 2.into(),
            3 => changed.profile["owners"]["namespace"]["count"] = 0.into(),
            4 => {
                changed.profile["abi_ops"]["wasi_snapshot_preview1.fd_read"]["served_by"]["namespace"] =
                    0.into();
            }
            5 => {
                changed.report["native_namespace_counter"]
                    .as_object_mut()
                    .unwrap()
                    .remove("external_owner_calls");
            }
            6 => {
                changed.report["native_namespace_counter"]["counter_domain"] =
                    "completed filesystem calls".into();
            }
            _ => {
                changed.report["native_namespace_counter"]["namespace_boundary"] =
                    "unbound boundary".into();
            }
        }
        assert!(counter(&changed, 1).is_err(), "mutation {mutation}");
    }
    run.report["native_namespace_counter"]["external_owner_calls"] = json!({"fd_write":0});
    assert_eq!(
        counter(&run, 1).unwrap().len(),
        1,
        "explicit zero-valued entry must remain visible for rejection"
    );
}

#[test]
fn operations_require_retained_valid_native_transcript() {
    let mut run = raw();
    let count = operation_retention(&run).unwrap();
    assert_eq!(count.calls, 0);
    run.root.streams.clear();
    assert!(
        operation_retention(&run)
            .unwrap_err()
            .contains("actual retained native transcript")
    );
    run.root
        .streams
        .insert("transcript".into(), b"malformed".to_vec());
    assert!(operation_retention(&run).is_err());
}

#[test]
fn immutable_four_seed_requests_and_actual_timing_are_required() {
    let mut bad = common();
    bad.requests.remove(&0);
    assert!(validate_common(&bad).is_err());
    let expected = common();
    let mut run = raw();
    attach_counter(&mut run);
    let observed = Observation {
        run: run.clone(),
        fresh_store_id: 1,
        replay: None,
    };
    assert!(sample(&expected, Role::Candidate, 0, 1, &observed).is_ok());
    run.report
        .as_object_mut()
        .unwrap()
        .remove("authorization_to_terminal_nanos");
    assert!(
        sample(
            &expected,
            Role::Candidate,
            0,
            1,
            &Observation {
                run,
                fresh_store_id: 1,
                replay: None
            }
        )
        .unwrap_err()
        .contains("actual native authorization")
    );
    let mut changed = observed;
    changed.run.request["guest"]["unexpected_configuration"] = true.into();
    assert!(sample(&expected, Role::Candidate, 0, 1, &changed).is_err());
}

#[test]
fn common_seed_requests_preserve_every_other_input_and_candidate_counter_entries() {
    let mut changed = common();
    changed.requests.get_mut(&1_000_003).unwrap()["guest"]["arguments"] = json!(["changed"]);
    assert!(validate_common(&changed).unwrap_err().contains("immutable"));
    let mut observed = raw();
    attach_counter(&mut observed);
    observed.report["native_namespace_counter"]["external_owner_calls"] =
        json!({"linked-external-callback":0});
    assert!(
        sample(
            &common(),
            Role::Candidate,
            0,
            1,
            &Observation {
                run: observed,
                fresh_store_id: 1,
                replay: None
            }
        )
        .unwrap_err()
        .contains("external callback")
    );
}
