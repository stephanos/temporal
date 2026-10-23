use super::*;

fn observation(case: &str, inputs: Value, value: Value) {
    println!(
        "GOMAD_RETAINED_OBSERVATION {}",
        json!({"schema":"gomad.retained-reader-observation/v1", "case":case,"inputs":inputs,"observed":value})
    );
}

fn soft_configuration(report_only: bool) -> Configuration {
    let (mut configuration, _) = super::projection_tests::configuration();
    configuration.workload = None;
    configuration.incarnations.truncate(1);
    configuration.incarnations[0].workload_kind = "go-test".into();
    configuration.oracle_plan = Some(serde_json::from_value(json!({
        "schema":"gomad.wasm.soft-assertion-observation/v1", "maximum_bytes":4096,
        "soft_assertions":[{"handle":configuration.incarnations[0].handle,"report_only":report_only}]
    })).unwrap());
    configuration
}

fn soft_execution(
    configuration: &Configuration,
    stdout: &[u8],
    stderr: &[u8],
) -> helper::Execution {
    let mut execution = super::projection_tests::observe(&[], false, false);
    execution.incarnations.truncate(1);
    let item = &mut execution.incarnations[0];
    item.handle = configuration.incarnations[0].handle.clone();
    item.execution.evidence = BTreeMap::from([
        ("stdout".into(), stdout.to_vec()),
        ("stderr".into(), stderr.to_vec()),
    ]);
    item.execution.outcome.output_bytes = (stdout.len() + stderr.len()) as u64;
    execution
}

fn execution_input(execution: &helper::Execution, configuration: &Configuration) -> Value {
    let item = &execution.incarnations[0];
    json!({"handle":item.handle,"stdout_base64":crate::campaign::exploration::base64(&item.execution.evidence["stdout"]),
        "stderr_base64":crate::campaign::exploration::base64(&item.execution.evidence["stderr"]),
        "termination":item.execution.outcome.termination,"message":item.execution.outcome.message,
        "offered_bytes":item.execution.outcome.output_bytes,"oracle_plan":configuration.oracle_plan})
}

#[test]
fn history_observation_rejects_every_original_ambiguous_input() {
    let (configuration, _) = super::projection_tests::configuration();
    let good = r#"{"cluster_id":"cluster","history":"YQ==","event_ids":[1],"types":["started"]}"#;
    let mut invalid_utf8 =
        b"LIVE {\"cluster_id\":\"cluster\",\"history\":\"YQ==\",\"event_ids\":[1],\"types\":[\""
            .to_vec();
    invalid_utf8.extend_from_slice(b"\xff\"]}");
    let cases = [
        ("missing", vec![]),
        (
            "duplicate-lines",
            format!("LIVE {good}\nLIVE {good}").into_bytes(),
        ),
        (
            "duplicate-key",
            format!("LIVE {},\"history\":\"Yg==\"}}", &good[..good.len() - 1]).into_bytes(),
        ),
        (
            "unknown-key",
            format!("LIVE {},\"unknown\":true}}", &good[..good.len() - 1]).into_bytes(),
        ),
        ("incomplete", b"LIVE {}".to_vec()),
        ("trailing-value", format!("LIVE {good} {{}}").into_bytes()),
        ("invalid-utf8", invalid_utf8),
    ];
    for (case, stdout) in cases {
        let mut execution = super::projection_tests::observe(&[], false, false);
        execution.incarnations[0]
            .execution
            .evidence
            .insert("stdout".into(), stdout);
        let error = oracles::assess(&execution, &configuration)
            .err()
            .expect(case);
        observation(
            &format!("history/{case}"),
            json!({"stdout_base64":crate::campaign::exploration::base64(&execution.incarnations[0].execution.evidence["stdout"])}),
            json!({"error":error}),
        );
    }
    let mut execution = super::projection_tests::observe(&[], false, false);
    for item in &mut execution.incarnations {
        item.execution
            .evidence
            .insert("stdout".into(), format!("LIVE {good}").into_bytes());
    }
    let report =
        serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
    assert_eq!(report["oracles"].as_array().unwrap().len(), 3);
    observation(
        "history/valid",
        json!({"stdout_base64":crate::campaign::exploration::base64(format!("LIVE {good}").as_bytes())}),
        report,
    );
}

#[test]
fn soft_assertion_completeness_and_literal_messages_survive_native_assessment() {
    for report_only in [false, true] {
        let configuration = soft_configuration(report_only);
        for (case, stdout, stderr, complete, message) in [
            ("literal-json", &b"PASS\n"[..], &b"failed assertion: invalid {\"left\":1}"[..],true,Some("invalid {\"left\":1}")),
            ("literal-tagged", &b"PASS\n"[..], &b"2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true}"[..],true,Some("invalid {state}")),
            ("invalid-utf8", &b"\xff"[..], &b""[..],false,None),
            ("offered-mismatch", &b"PASS\n"[..], &b""[..],false,None),
            ("capacity-equal-counters", &b"PASS"[..], &b""[..],false,None),
        ] {
            let mut execution = soft_execution(&configuration, stdout, stderr);
            if case == "offered-mismatch" { execution.incarnations[0].execution.outcome.output_bytes = 100; }
            if case == "capacity-equal-counters" {
                execution.incarnations[0].execution.outcome.termination = "capacity";
                execution.incarnations[0].execution.outcome.message = "output capacity".into();
            }
            let value = serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
            let oracle = &value["oracles"][0];
            assert_eq!(oracle["inconclusive"], !complete, "{case}");
            assert_eq!(oracle["passed"], complete && report_only, "{case}");
            assert_eq!(oracle["report_only"], report_only);
            assert_eq!(oracle["soft_assertion"], true);
            let evidence = oracle["evidence"].as_array().unwrap();
            let messages = evidence.iter().find(|item| item["label"] == "messages").unwrap();
            let bytes = crate::campaign::scenario::canonical_base64(messages["value"].as_str().unwrap()).unwrap();
            let actual: Vec<String> = serde_json::from_slice(&bytes).unwrap();
            assert_eq!(actual, message.into_iter().map(str::to_owned).collect::<Vec<_>>());
            assert_eq!(messages["full_sha256"], canonical::digest(&bytes));
            observation(&format!("soft/{case}/report-only={report_only}"), execution_input(&execution, &configuration), value);
        }
    }
}

#[test]
fn soft_assertion_opt_out_requires_bound_recorded_test_workload() {
    for workload in ["", "go-run", "go-test", "unknown"] {
        let mut configuration = soft_configuration(true);
        configuration.incarnations[0].workload_kind = workload.into();
        let error = oracles::validate(&configuration).err();
        assert_eq!(error.is_none(), workload == "go-test");
        observation(
            &format!("policy/{workload}"),
            json!({"workload_kind":workload}),
            json!({"error":error}),
        );
    }
    for case in ["unknown-handle", "duplicate-policy"] {
        let mut configuration = soft_configuration(true);
        let mut plan = serde_json::to_value(configuration.oracle_plan.take().unwrap()).unwrap();
        if case == "unknown-handle" {
            plan["soft_assertions"][0]["handle"]["node"] = json!("foreign");
        } else {
            let policy = plan["soft_assertions"][0].clone();
            plan["soft_assertions"].as_array_mut().unwrap().push(policy);
        }
        configuration.oracle_plan = Some(serde_json::from_value(plan).unwrap());
        let error = oracles::validate(&configuration).unwrap_err();
        observation(
            &format!("policy/{case}"),
            json!({"oracle_plan":configuration.oracle_plan}),
            json!({"error":error}),
        );
    }
}

#[cfg(feature = "determinism-fixture")]
#[test]
fn output_capacity_annotation_is_narrow_and_preserves_oracle_budget_errors() {
    for report_only in [false, true] {
        for maximum in [1, 4096] {
            let mut configuration = soft_configuration(report_only);
            let mut plan = serde_json::to_value(configuration.oracle_plan.take().unwrap()).unwrap();
            plan["maximum_bytes"] = json!(maximum);
            configuration.oracle_plan = Some(serde_json::from_value(plan).unwrap());
            for (termination, message) in [
                ("infrastructure", "output capacity"),
                ("capacity", "transcript capacity"),
                ("exit", ""),
                ("capacity", "output capacity"),
            ] {
                let mut execution = soft_execution(
                    &configuration,
                    b"PASS",
                    b"failed assertion: invalid {state}",
                );
                execution.incarnations[0].execution.outcome.termination = termination;
                execution.incarnations[0].execution.outcome.message = message.into();
                let value = oracles::capacity_observation(&execution, &configuration);
                let capacity = termination == "capacity" && message == "output capacity";
                assert_eq!(!value["capacity_failure"].is_null(), capacity);
                if capacity {
                    assert_eq!(value["capacity_failure"]["termination"], "capacity");
                    assert_eq!(value["capacity_failure"]["message"], "output capacity");
                    assert_eq!(
                        value["capacity_failure"]["soft_assertions"]["inconclusive"],
                        true
                    );
                    assert_eq!(
                        value["capacity_failure"]["soft_assertions"]["report_only"],
                        report_only
                    );
                    assert_eq!(
                        value["capacity_failure"]["soft_assertions"]["messages"],
                        json!(["invalid {state}"])
                    );
                    assert!(
                        value["capacity_failure"]["detail"]
                            .as_str()
                            .unwrap()
                            .contains(if maximum == 1 {
                                "resource=oracle_evidence_bytes"
                            } else {
                                "choice trace is not complete"
                            })
                    );
                }
                observation(
                    &format!(
                        "capacity/{termination}/{message}/report-only={report_only}/maximum={maximum}"
                    ),
                    execution_input(&execution, &configuration),
                    value,
                );
            }
        }
    }
}

fn metadata_publication() -> crate::evidence::store::Publication {
    use crate::evidence::{
        identity::Identity,
        record::{Outcome, Record},
    };
    let module = b"\0asm\x01\0\0\0".to_vec();
    let (mut configuration, _) = super::projection_tests::configuration();
    configuration.workload = None;
    configuration.oracle_plan = None;
    configuration.engine = linked::Conditions::compiled().configuration().into();
    configuration.incarnations.truncate(1);
    let item = &mut configuration.incarnations[0];
    item.module = Payload {
        file: "target/module.wasm".into(),
        bytes: module.len() as u64,
        sha256: canonical::digest(&module),
        limit: 512 << 20,
    };
    let toolchain = canonical::digest(b"retained-reader-test-toolchain");
    item.choice_identity.target_sha256 = sha2::Sha256::digest(&module).into();
    item.choice_identity.toolchain_build_key = toolchain.clone();
    let identity = Identity {
        target: canonical::digest(&module),
        toolchain: toolchain.clone(),
        platform: super::super::preflight::platform(),
        implementation: super::super::preflight::implementation().unwrap(),
        configuration: canonical::digest(&canonical::encode(&configuration).unwrap()),
    };
    let record = Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest().unwrap(),
        identity,
        profile: configuration.profile.clone(),
        seed: configuration.broker.seed,
        arguments: vec![],
        environment: BTreeMap::new(),
        limits: configuration.limits.clone(),
        virtual_time_nanos: 0,
        outcome: Outcome {
            class: "success".into(),
            termination: "exit".into(),
            exit_code: Some(0),
            reason: String::new(),
        },
        streams: BTreeMap::new(),
    };
    let inputs = Inputs {
        schema: "gomad.cluster-replay-inputs/v1".into(),
        helper_sha256: linked::IMPLEMENTATION_SHA256.into(),
        host_sha256: env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into(),
        runtime_implementation_sha256: env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256").into(),
        toolchain_sha256: toolchain,
        retained: vec![Retained {
            handle: configuration.incarnations[0].handle.clone(),
            interrupted: false,
            evidence: BTreeMap::new(),
        }],
        configuration,
    };
    crate::evidence::store::Publication {
        record,
        payloads: BTreeMap::from([
            (FILE.into(), canonical::encode(&inputs).unwrap()),
            ("target/module.wasm".into(), module),
            ("evidence/world".into(), vec![]),
        ]),
    }
}

struct ReaderFixture {
    root: std::path::PathBuf,
    path: std::path::PathBuf,
}

impl ReaderFixture {
    fn new(publication: crate::evidence::store::Publication) -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-retained-reader-{}",
            crate::campaign::origin::identity().unwrap()
        ));
        let published = crate::evidence::store::Store {
            root: root.clone(),
            maximum_bytes: 128 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        }
        .publish(publication)
        .unwrap();
        Self {
            root,
            path: published.path,
        }
    }
    fn open(&self) -> Opened {
        Opened::open(&self.path).unwrap()
    }
}

impl Drop for ReaderFixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}

#[test]
fn retained_raw_authentication_binds_bytes_length_digest_and_path() {
    let fixture = ReaderFixture::new(metadata_publication());
    let opened = fixture.open();
    let original = opened.payload(FILE, 16 << 20).unwrap();
    let inputs = super::inputs(&opened).unwrap();
    super::bind(&opened, &inputs).unwrap();
    for case in ["digest", "bytes", "path", "limit"] {
        let mut descriptor = Payload {
            file: FILE.into(),
            bytes: original.len() as u64,
            sha256: canonical::digest(&original),
            limit: 16 << 20,
        };
        match case {
            "digest" => descriptor.sha256 = canonical::digest(b"foreign bytes"),
            "bytes" => descriptor.bytes += 1,
            "path" => descriptor.file = "evidence/foreign.json".into(),
            "limit" => descriptor.limit = 32 << 20,
            _ => unreachable!(),
        }
        let error = descriptor.read(&opened, 16 << 20).unwrap_err();
        observation(
            &format!("authentication/{case}"),
            json!({"payload_base64":crate::campaign::exploration::base64(&original),"descriptor":descriptor}),
            json!({"error":error}),
        );
    }
    let mut changed = original.clone();
    changed.push(b'\n');
    std::fs::write(fixture.path.join(FILE), &changed).unwrap();
    let error = opened.payload(FILE, 16 << 20).unwrap_err();
    observation(
        "authentication/raw-bytes",
        json!({"original_base64":crate::campaign::exploration::base64(&original),"changed_base64":crate::campaign::exploration::base64(&changed)}),
        json!({"error":error}),
    );
}

#[test]
fn retained_native_decoder_is_closed_and_preserves_the_existing_lineage() {
    let baseline = metadata_publication();
    let raw = baseline.payloads[FILE].clone();
    let original: Inputs = serde_json::from_slice(&raw).unwrap();
    assert_eq!(canonical::encode(&original).unwrap(), raw);
    for case in [
        "ordinary-spelling",
        "duplicate-top",
        "duplicate-nested",
        "unknown-top",
        "unknown-nested",
        "missing-module",
        "wrong-module-type",
        "foreign-schema",
        "legacy-schema",
        "trailing-value",
        "truncated",
        "invalid-utf8",
        "host-faults",
        "host-scenarios",
        "host-history",
        "host-observations",
        "host-oracles",
        "go-served-calls",
        "guest-callbacks",
    ] {
        let mut publication = metadata_publication();
        let mut value: Value = serde_json::from_slice(&raw).unwrap();
        let changed = match case {
            "ordinary-spelling" => serde_json::to_vec_pretty(&value).unwrap(),
            "duplicate-top" => {
                let text = String::from_utf8(raw.clone()).unwrap();
                format!("{{\"schema\":\"duplicate\",{}", &text[1..]).into_bytes()
            }
            "duplicate-nested" => {
                let text = String::from_utf8(raw.clone()).unwrap();
                text.replacen(
                    "\"configuration\":{",
                    "\"configuration\":{\"engine\":\"duplicate\",",
                    1,
                )
                .into_bytes()
            }
            "unknown-top" => {
                value["unknown"] = json!(true);
                canonical::encode(&value).unwrap()
            }
            "unknown-nested" => {
                value["configuration"]["unknown"] = json!(true);
                canonical::encode(&value).unwrap()
            }
            "missing-module" => {
                value["configuration"]["incarnations"][0]
                    .as_object_mut()
                    .unwrap()
                    .remove("module");
                canonical::encode(&value).unwrap()
            }
            "wrong-module-type" => {
                value["configuration"]["incarnations"][0]["module"] = json!(1);
                canonical::encode(&value).unwrap()
            }
            "foreign-schema" => {
                value["schema"] = json!("gomad.other/v1");
                canonical::encode(&value).unwrap()
            }
            "legacy-schema" => {
                value["schema"] = json!("gomad.wasm-simulation-retained/v1");
                canonical::encode(&value).unwrap()
            }
            "trailing-value" => {
                let mut bytes = raw.clone();
                bytes.extend_from_slice(b"{}");
                bytes
            }
            "truncated" => raw[..raw.len() - 1].to_vec(),
            "invalid-utf8" => {
                let mut bytes = raw.clone();
                bytes[2] = 0xff;
                bytes
            }
            field => {
                let name = match field {
                    "host-faults" => "faults",
                    "host-scenarios" => "scenarios",
                    "host-history" => "history",
                    "host-observations" => "observations",
                    "host-oracles" => "oracles",
                    "go-served-calls" => "go_filesystem_calls",
                    "guest-callbacks" => "guest_callbacks",
                    _ => unreachable!(),
                };
                value["configuration"][name] = json!([{}]);
                canonical::encode(&value).unwrap()
            }
        };
        publication.payloads.insert(FILE.into(), changed.clone());
        let fixture = ReaderFixture::new(publication);
        let opened = fixture.open();
        opened.authenticate_all().unwrap();
        let error = super::inputs(&opened).err().expect(case);
        observation(
            &format!("decode/{case}"),
            json!({"payload_base64":crate::campaign::exploration::base64(&changed)}),
            json!({"error":error}),
        );
    }
    observation(
        "decode/native-valid",
        json!({"payload_base64":crate::campaign::exploration::base64(&raw)}),
        json!({"reencoded_base64":crate::campaign::exploration::base64(&canonical::encode(&original).unwrap())}),
    );
}

#[test]
fn retained_implementation_and_primary_binding_reject_before_preparing_helpers() {
    for case in [
        "helper",
        "host",
        "runtime",
        "toolchain",
        "engine",
        "profile",
        "primary-path",
        "primary-digest",
        "missing-incarnations",
        "missing-retained",
        "foreign-retained",
        "duplicate-incarnation",
    ] {
        let mut publication = metadata_publication();
        let mut inputs: Inputs = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
        match case {
            "helper" => inputs.helper_sha256 = canonical::digest(b"resealed helper"),
            "host" => inputs.host_sha256 = canonical::digest(b"foreign host"),
            "runtime" => {
                inputs.runtime_implementation_sha256 = canonical::digest(b"foreign runtime")
            }
            "toolchain" => inputs.toolchain_sha256 = canonical::digest(b"foreign toolchain"),
            "engine" => inputs.configuration.engine.push_str("-changed"),
            "profile" => inputs.configuration.profile = "foreign".into(),
            "primary-path" => {
                inputs.configuration.incarnations[0].module.file = "target/foreign.wasm".into()
            }
            "primary-digest" => {
                inputs.configuration.incarnations[0].module.sha256 =
                    canonical::digest(b"foreign module")
            }
            "missing-incarnations" => inputs.configuration.incarnations.clear(),
            "missing-retained" => inputs.retained.clear(),
            "foreign-retained" => inputs.retained[0].handle.node = "foreign".into(),
            "duplicate-incarnation" => inputs
                .configuration
                .incarnations
                .push(inputs.configuration.incarnations[0].clone()),
            _ => unreachable!(),
        }
        publication.record.identity.configuration =
            canonical::digest(&canonical::encode(&inputs.configuration).unwrap());
        publication.record.execution_identity = publication.record.identity.digest().unwrap();
        publication
            .payloads
            .insert(FILE.into(), canonical::encode(&inputs).unwrap());
        let fixture = ReaderFixture::new(publication);
        let opened = fixture.open();
        let before = std::fs::read_dir(&fixture.path).unwrap().count();
        let error = super::bind(&opened, &inputs).unwrap_err();
        let prepare_error = super::prepare(opened.try_clone().unwrap(), None)
            .err()
            .unwrap();
        assert_eq!(prepare_error, error);
        assert_eq!(std::fs::read_dir(&fixture.path).unwrap().count(), before);
        assert!(!fixture.path.join("compiled-cache").exists());
        observation(
            &format!("binding/{case}"),
            json!({"retained_inputs":inputs,"record":opened.manifest().record}),
            json!({"error":error,"prepare_error":prepare_error,"entries_before":before,"entries_after":std::fs::read_dir(&fixture.path).unwrap().count(),"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
        );
    }
}

#[test]
fn absent_recorded_test_execution_is_inconclusive_under_both_policies() {
    for report_only in [false, true] {
        let configuration = soft_configuration(report_only);
        let mut execution = soft_execution(&configuration, b"PASS", b"");
        execution.incarnations.clear();
        let report =
            serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
        assert_eq!(report["oracles"].as_array().unwrap().len(), 1);
        assert_eq!(report["oracles"][0]["passed"], false);
        assert_eq!(report["oracles"][0]["inconclusive"], true);
        assert_eq!(report["oracles"][0]["report_only"], report_only);
        assert_eq!(report["outcome"], "oracle_failed");
        observation(
            &format!("soft/missing-execution/report-only={report_only}"),
            json!({"incarnations":[],"oracle_plan":configuration.oracle_plan,"configured_handle":configuration.incarnations[0].handle}),
            report,
        );
    }
}

#[cfg(feature = "determinism-fixture")]
#[test]
fn actual_retained_admission_uses_private_bound_inputs_and_replays_without_source() {
    let corpus = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let publication = crate::fixture_cluster::publication(&corpus, 7).unwrap();
    let fixture = ReaderFixture::new(publication);
    let opened = fixture.open();
    opened.authenticate_all().unwrap();
    let original = opened.payload(FILE, 16 << 20).unwrap();
    let before = std::fs::read_dir(&fixture.path).unwrap().count();
    let prepared = super::prepare(opened.try_clone().unwrap(), None).unwrap();
    assert!(prepared.exact && prepared.available);
    let actual = super::execute_projected(prepared.execution).unwrap();
    let divergence = crate::replay::divergence::first(&opened.manifest().record, &actual);
    assert_eq!(divergence, None);
    assert_eq!(opened.payload(FILE, 16 << 20).unwrap(), original);
    assert_eq!(std::fs::read_dir(&fixture.path).unwrap().count(), before);
    assert!(!fixture.path.join("compiled-cache").exists());
    observation(
        "admission/actual-replay",
        json!({"retained_inputs_base64":crate::campaign::exploration::base64(&original),"record":opened.manifest().record}),
        json!({"divergence":divergence,"termination":actual.outcome.termination,"exit_code":actual.outcome.exit_code,"entries_before":before,"entries_after":std::fs::read_dir(&fixture.path).unwrap().count(),"reopened_inputs_base64":crate::campaign::exploration::base64(&opened.payload(FILE,16<<20).unwrap()),"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
    );
}

#[test]
fn retained_target_reader_refuses_missing_tampered_and_symbolic_sources() {
    use std::os::unix::fs::symlink;
    for case in [
        "tampered-bytes",
        "truncated",
        "missing-source",
        "symbolic-link-source",
    ] {
        let fixture = ReaderFixture::new(metadata_publication());
        let opened = fixture.open();
        let module = fixture.path.join("target/module.wasm");
        let original = opened.payload("target/module.wasm", 512 << 20).unwrap();
        match case {
            "tampered-bytes" => std::fs::write(&module, b"foreign!").unwrap(),
            "truncated" => std::fs::write(&module, &original[..original.len() - 1]).unwrap(),
            "missing-source" => std::fs::remove_file(&module).unwrap(),
            "symbolic-link-source" => {
                let genuine = fixture.root.join("genuine-module");
                std::fs::rename(&module, &genuine).unwrap();
                symlink(&genuine, &module).unwrap();
            }
            _ => unreachable!(),
        }
        let error = opened.payload("target/module.wasm", 512 << 20).unwrap_err();
        assert!(!fixture.path.join("compiled-cache").exists());
        observation(
            &format!("module/{case}"),
            json!({"original_base64":crate::campaign::exploration::base64(&original),"file":"target/module.wasm","descriptor":opened.manifest().files["target/module.wasm"]}),
            json!({"error":error,"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
        );
    }
}

#[test]
fn native_storage_selection_projection_is_detached() {
    let (mut configuration, _) = super::projection_tests::configuration();
    let item = &mut configuration.incarnations[0];
    let selection = BTreeMap::from([
        ("volume".to_owned(), vec![1, 2]),
        ("empty".to_owned(), vec![]),
    ]);
    item.storage_crash_selection = Some(selection.clone());
    assert!(
        super::storage_input(item)
            .err()
            .unwrap()
            .contains("declared storage volumes")
    );
    item.storage_crash_selection = None;
    assert!(super::storage_input(item).unwrap().is_none());
    item.storage_crash_selection = Some(BTreeMap::new());
    assert!(super::storage_input(item).unwrap().is_none());
    let mut detached = item.clone();
    detached.storage_crash_selection = Some(selection.clone());
    detached
        .storage_crash_selection
        .as_mut()
        .unwrap()
        .get_mut("volume")
        .unwrap()[0] = 99;
    assert!(item.storage_crash_selection.as_ref().unwrap().is_empty());
    observation(
        "storage-selection/native-ownership",
        json!({"selection":selection}),
        json!({"original":item.storage_crash_selection,"detached":detached.storage_crash_selection,"durable_default":super::storage_input(item).unwrap().is_none()}),
    );
}

#[test]
fn sqlite_oracle_intent_is_sealed_by_the_native_workload_owner() {
    use crate::qualify::workloads::Workload;
    let original = json!({"kind":"sqlite-durable-recovery","selection":null});
    let workload: Workload = serde_json::from_value(original.clone()).unwrap();
    workload.validate().unwrap();
    for (case, field, value) in [
        ("unknown-schema", "schema", json!("unknown")),
        ("arbitrary-prefix", "prefix", json!("arbitrary ")),
        ("mixed-history", "histories", json!([{"name":"mixed"}])),
        (
            "missing-ack-node",
            "acknowledged",
            json!({"node":"missing","incarnation":2}),
        ),
        (
            "same-handles",
            "recovered",
            json!({"node":"sqlite","incarnation":2}),
        ),
        ("empty-events", "expected_events", json!([])),
        ("noncanonical-event", "expected_events", json!(["01"])),
        (
            "ambiguous-value",
            "expected_value",
            json!("comma,ambiguous"),
        ),
        (
            "unchecked-integrity",
            "expected_integrity",
            json!("unchecked"),
        ),
    ] {
        let mut changed = original.clone();
        changed[field] = value;
        let error = serde_json::from_value::<Workload>(changed.clone())
            .err()
            .unwrap()
            .to_string();
        observation(
            &format!("sqlite-intent/{case}"),
            changed,
            json!({"error":error}),
        );
    }
    observation(
        "sqlite-intent/native-valid",
        original,
        json!({"workload":workload}),
    );
}

#[test]
fn native_storage_projection_detaches_selection_context_and_limits() {
    let (mut configuration, _) = super::projection_tests::configuration();
    let item = &mut configuration.incarnations[0];
    let context=canonical::encode(&json!({"volume":{"PendingSHA256":"a".repeat(64),"Operations":[{"ID":2,"Dependencies":[1]}]}})).unwrap();
    item.storage=Some(serde_json::from_value(json!({"descriptor":{
        "model":{"volumes":[{"ID":"volume","Path":"/volume","CapacityBytes":4096}],"limits":{"PendingOperations":10,"Transitions":100}},
        "model_identity":"a".repeat(64),"return_image":false,
        "crash_limits":{"States":32,"Operations":8,"Depth":8,"Bytes":4096,"Work":65536},
        "crash_context":{"chunks":1,"bytes":context.len(),"sha256":canonical::digest(&context)}},"context":context})).unwrap());
    item.storage_crash_selection = Some(BTreeMap::from([
        ("volume".into(), vec![1, 2]),
        ("empty".into(), vec![]),
    ]));
    let before = canonical::encode(&item).unwrap();
    let projected = super::storage_input(item).unwrap().unwrap();
    let projected = serde_json::to_value(projected).unwrap();
    assert_eq!(
        projected["descriptor"]["crash_selection"]["volume"],
        json!([1, 2])
    );
    assert_eq!(
        projected["descriptor"]["crash_selection"]["empty"],
        json!([])
    );
    let mut detached = projected.clone();
    detached["descriptor"]["crash_selection"]["volume"][0] = json!(99);
    detached["descriptor"]["crash_selection"]
        .as_object_mut()
        .unwrap()
        .remove("empty");
    detached["descriptor"]["crash_limits"]["States"] = json!(1);
    detached["context"][0] = json!(99);
    assert_eq!(canonical::encode(&item).unwrap(), before);
    let again = serde_json::to_value(super::storage_input(item).unwrap()).unwrap();
    assert_eq!(again, projected);
    observation(
        "storage-selection/native-deep-detachment",
        json!({"original_incarnation_base64":crate::campaign::exploration::base64(&before)}),
        json!({"projected":projected,"detached":detached,"reprojected":again,"original_after_base64":crate::campaign::exploration::base64(&canonical::encode(&item).unwrap())}),
    );
}
