use super::*;

fn observation(case: &str, inputs: Value, value: Value) {
    let mut row = serde_json::Map::from_iter([
        (
            "schema".into(),
            json!("gomad.retained-reader-observation/v1"),
        ),
        ("case".into(), json!(case)),
    ]);
    row.insert("inputs".into(), inputs);
    row.insert("observed".into(), value);
    println!("GOMAD_RETAINED_OBSERVATION {}", Value::Object(row));
}

fn directory_snapshot(root: &std::path::Path) -> Value {
    fn visit(
        root: &std::path::Path,
        path: &std::path::Path,
        entries: &mut BTreeMap<String, Value>,
    ) {
        let metadata = std::fs::symlink_metadata(path).unwrap();
        let relative = path
            .strip_prefix(root)
            .unwrap()
            .to_str()
            .unwrap()
            .to_owned();
        if metadata.is_dir() {
            if !relative.is_empty() {
                entries.insert(relative, json!({"kind":"directory"}));
            }
            for entry in std::fs::read_dir(path).unwrap() {
                visit(root, &entry.unwrap().path(), entries);
            }
        } else if metadata.file_type().is_symlink() {
            entries.insert(
                relative,
                json!({"kind":"symlink","target":std::fs::read_link(path).unwrap()}),
            );
        } else {
            assert!(metadata.is_file());
            let bytes = std::fs::read(path).unwrap();
            entries.insert(
                relative,
                json!({"kind":"file","bytes":bytes.len(),"sha256":canonical::digest(&bytes)}),
            );
        }
    }
    let mut entries = BTreeMap::new();
    visit(root, root, &mut entries);
    json!(entries)
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
    configuration.engine = linked::Conditions::compiled().configuration();
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
    fn new(mut publication: crate::evidence::store::Publication) -> Self {
        if let Some(raw) = publication.payloads.get(FILE) {
            let stream = publication
                .record
                .streams
                .entry(FILE.into())
                .or_insert_with(|| crate::evidence::record::Stream {
                    bytes: raw.len() as u64,
                    sha256: canonical::digest(raw),
                    records: 0,
                });
            stream.bytes = raw.len() as u64;
            stream.sha256 = canonical::digest(raw);
        }
        if publication.payloads.contains_key("record.json") {
            publication
                .payloads
                .insert("record.json".into(), publication.record.bytes().unwrap());
        }
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

fn native_decoder_mutation(case: &str, raw: &[u8]) -> Vec<u8> {
    let mut value: Value = serde_json::from_slice(raw).unwrap();
    match case {
        "ordinary-spelling" => serde_json::to_vec_pretty(&value).unwrap(),
        "duplicate-top" => {
            let text = String::from_utf8(raw.to_vec()).unwrap();
            format!("{{\"schema\":\"duplicate\",{}", &text[1..]).into_bytes()
        }
        "duplicate-nested" => {
            let text = String::from_utf8(raw.to_vec()).unwrap();
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
        "duplicate-model-value" => {
            let text = String::from_utf8(raw.to_vec()).unwrap();
            text.replacen(
                "\"model\":{",
                "\"model\":{\"native_value\":1,\"native_value\":2,",
                1,
            )
            .into_bytes()
        }
        "duplicate-environment-map" => {
            let text = String::from_utf8(raw.to_vec()).unwrap();
            text.replacen(
                "\"environment\":{",
                "\"environment\":{\"PWD\":\"/first\",\"PWD\":\"/second\",",
                1,
            )
            .into_bytes()
        }
        "excessive-nesting" => {
            let text = String::from_utf8(raw.to_vec()).unwrap();
            let nested = format!("{}0{}", "[".repeat(129), "]".repeat(129));
            text.replacen("\"model\":{}", &format!("\"model\":{nested}"), 1)
                .into_bytes()
        }
        "oversized" => vec![b' '; (16 << 20) + 1],
        "foreign-schema" => {
            value["schema"] = json!("gomad.other/v1");
            canonical::encode(&value).unwrap()
        }
        "legacy-schema" => {
            value["schema"] = json!("gomad.wasm-simulation-retained/v1");
            canonical::encode(&value).unwrap()
        }
        "trailing-value" => {
            let mut bytes = raw.to_vec();
            bytes.extend_from_slice(b"{}");
            bytes
        }
        "truncated" => raw[..raw.len() - 1].to_vec(),
        "invalid-utf8" => {
            let mut bytes = raw.to_vec();
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
    }
}

#[test]
fn retained_native_decoder_is_closed_and_preserves_the_existing_lineage() {
    let baseline = metadata_publication();
    let raw = baseline.payloads[FILE].clone();
    let original: Inputs = serde_json::from_slice(&raw).unwrap();
    assert_eq!(canonical::encode(&original).unwrap(), raw);
    for case in [
        "duplicate-top",
        "duplicate-nested",
        "unknown-top",
        "unknown-nested",
        "duplicate-model-value",
        "duplicate-environment-map",
        "excessive-nesting",
        "oversized",
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
        let changed = native_decoder_mutation(case, &raw);
        publication.payloads.insert(FILE.into(), changed.clone());
        let fixture = ReaderFixture::new(publication);
        let opened = fixture.open();
        opened.authenticate_all().unwrap();
        let error = super::inputs(&opened).err().expect(case);
        let prepare_error = super::prepare(opened.try_clone().unwrap(), None)
            .err()
            .expect(case);
        assert_eq!(prepare_error, error);
        let before_snapshot = directory_snapshot(&fixture.path);
        let before = std::fs::read_dir(&fixture.path).unwrap().count();
        let mut cli_output = Vec::new();
        let cli_error =
            crate::cli::replay::run(&[fixture.path.to_str().unwrap().into()], &mut cli_output)
                .unwrap_err();
        assert_eq!(cli_error.status, 2);
        assert!(cli_error.message.contains(&error));
        assert_eq!(std::fs::read_dir(&fixture.path).unwrap().count(), before);
        let expected = match case {
            "foreign-schema" | "legacy-schema" => {
                "unsupported retained cluster replay input schema"
            }
            "duplicate-top" => "duplicate object key",
            "duplicate-nested" | "duplicate-model-value" | "duplicate-environment-map" => {
                "duplicate object key"
            }
            "excessive-nesting" => "nesting",
            "oversized" => "artifact payload exceeds caller bound",
            "unknown-top" | "unknown-nested" => "unknown field `unknown`",
            "missing-module" => "missing field `module`",
            "wrong-module-type" => "invalid type: integer",
            "trailing-value" => "trailing data",
            "truncated" => "invalid syntax",
            "invalid-utf8" => "unicode",
            "host-faults" => "unknown field `faults`",
            "host-scenarios" => "unknown field `scenarios`",
            "host-history" => "unknown field `history`",
            "host-observations" => "unknown field `observations`",
            "host-oracles" => "unknown field `oracles`",
            "go-served-calls" => "unknown field `go_filesystem_calls`",
            "guest-callbacks" => "unknown field `guest_callbacks`",
            _ => unreachable!(),
        };
        assert!(
            error.contains(expected),
            "{case}: expected {expected:?}, got {error:?}"
        );
        observation(
            &format!("decode/{case}"),
            json!({"payload_base64":crate::campaign::exploration::base64(&changed)}),
            json!({"error":error,"prepare_error":prepare_error,"public_status":cli_error.status,"public_error":cli_error.message,"directory_before":before_snapshot,"directory_after":directory_snapshot(&fixture.path),"entries_before":before,"entries_after":std::fs::read_dir(&fixture.path).unwrap().count()}),
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
                inputs.runtime_implementation_sha256 = canonical::digest(b"foreign runtime");
            }
            "toolchain" => inputs.toolchain_sha256 = canonical::digest(b"foreign toolchain"),
            "engine" => inputs.configuration.engine.push_str("-changed"),
            "profile" => inputs.configuration.profile = "foreign".into(),
            "primary-path" => {
                inputs.configuration.incarnations[0].module.file = "target/foreign.wasm".into();
            }
            "primary-digest" => {
                inputs.configuration.incarnations[0].module.sha256 =
                    canonical::digest(b"foreign module");
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
        let before_snapshot = directory_snapshot(&fixture.path);
        let before = std::fs::read_dir(&fixture.path).unwrap().count();
        let error = super::bind(&opened, &inputs).unwrap_err();
        assert_eq!(
            error,
            match case {
                "primary-path" | "primary-digest" =>
                    "retained cluster primary module identity mismatch",
                "missing-incarnations" | "missing-retained" | "duplicate-incarnation" =>
                    "retained cluster incarnation count mismatch",
                "foreign-retained" => "retained cluster handle mismatch",
                _ => "WASM retained cluster implementation/configuration identity mismatch",
            },
            "{case}"
        );
        let prepare_error = super::prepare(opened.try_clone().unwrap(), None)
            .err()
            .unwrap();
        assert_eq!(prepare_error, error);
        assert_eq!(std::fs::read_dir(&fixture.path).unwrap().count(), before);
        assert!(!fixture.path.join("compiled-cache").exists());
        observation(
            &format!("binding/{case}"),
            json!({"retained_inputs":inputs,"record":opened.manifest().record}),
            json!({"error":error,"prepare_error":prepare_error,"directory_before":before_snapshot,"directory_after":directory_snapshot(&fixture.path),"entries_before":before,"entries_after":std::fs::read_dir(&fixture.path).unwrap().count(),"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
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
    let before_snapshot = directory_snapshot(&fixture.path);
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
        json!({"retained_inputs_base64":crate::campaign::exploration::base64(&original),"record":opened.manifest().record,"actual_execution":{"outcome":actual.outcome,"evidence":actual.evidence}}),
        json!({"divergence":divergence,"termination":actual.outcome.termination,"exit_code":actual.outcome.exit_code,"directory_before":before_snapshot,"directory_after":directory_snapshot(&fixture.path),"entries_before":before,"entries_after":std::fs::read_dir(&fixture.path).unwrap().count(),"reopened_inputs_base64":crate::campaign::exploration::base64(&opened.payload(FILE,16<<20).unwrap()),"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
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
        let source_metadata = std::fs::symlink_metadata(&module).ok();
        let source_kind = source_metadata.as_ref().map(|metadata| {
            if metadata.file_type().is_symlink() {
                "symlink"
            } else if metadata.is_file() {
                "file"
            } else {
                "other"
            }
        });
        let actual_bytes = (source_kind == Some("file")).then(|| std::fs::read(&module).unwrap());
        let link_target =
            (source_kind == Some("symlink")).then(|| std::fs::read_link(&module).unwrap());
        let error = opened.payload("target/module.wasm", 512 << 20).unwrap_err();
        assert!(!fixture.path.join("compiled-cache").exists());
        observation(
            &format!("module/{case}"),
            json!({"original_base64":crate::campaign::exploration::base64(&original),"file":"target/module.wasm","descriptor":opened.manifest().files["target/module.wasm"],"source_kind":source_kind,"source_bytes_base64":actual_bytes.as_deref().map(crate::campaign::exploration::base64),"link_target":link_target}),
            json!({"error":error,"compiled_cache_exists":fixture.path.join("compiled-cache").exists()}),
        );
    }
}

#[test]
fn native_storage_selection_requires_declared_volumes_and_preserves_defaults() {
    let (mut configuration, _) = super::projection_tests::configuration();
    let item = &mut configuration.incarnations[0];
    let selection = BTreeMap::from([
        ("volume".to_owned(), vec![1, 2]),
        ("empty".to_owned(), vec![]),
    ]);
    item.storage_crash_selection = Some(selection.clone());
    let requested = canonical::encode(item).unwrap();
    let selection_error = super::storage_input(item).err().unwrap();
    assert!(selection_error.contains("declared storage volumes"));
    item.storage_crash_selection = None;
    let durable_none = super::storage_input(item).unwrap().is_none();
    assert!(durable_none);
    item.storage_crash_selection = Some(BTreeMap::new());
    assert!(super::storage_input(item).unwrap().is_none());
    observation(
        "storage-selection/native-ownership",
        json!({"selection":selection,"requested_incarnation_base64":crate::campaign::exploration::base64(&requested)}),
        json!({"selection_error":selection_error,"durable_none":durable_none,"original":item.storage_crash_selection,"durable_default":super::storage_input(item).unwrap().is_none()}),
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
    let mut dependency_detached = projected.clone();
    let context_bytes: Vec<u8> =
        serde_json::from_value(dependency_detached["context"].clone()).unwrap();
    let mut context: Value = serde_json::from_slice(&context_bytes).unwrap();
    context["volume"]["Operations"][0]["Dependencies"][0] = json!(99);
    dependency_detached["context"] = json!(canonical::encode(&context).unwrap());
    assert_eq!(canonical::encode(&item).unwrap(), before);
    let again = serde_json::to_value(super::storage_input(item).unwrap()).unwrap();
    assert_eq!(again, projected);
    observation(
        "storage-selection/native-deep-detachment",
        json!({"original_incarnation_base64":crate::campaign::exploration::base64(&before)}),
        json!({"projected":projected,"detached":detached,"dependency_detached":dependency_detached,"reprojected":again,"original_after_base64":crate::campaign::exploration::base64(&canonical::encode(&item).unwrap())}),
    );
}

#[test]
fn unstarted_unrequested_test_incarnation_does_not_gain_an_oracle() {
    let mut configuration = soft_configuration(false);
    configuration.oracle_plan = None;
    let mut execution = soft_execution(&configuration, b"PASS", b"");
    execution.incarnations.clear();
    let report =
        serde_json::to_value(oracles::assess(&execution, &configuration).unwrap()).unwrap();
    assert_eq!(
        report["oracles"].as_array().unwrap().as_slice(),
        &[] as &[Value]
    );
    assert_eq!(report["outcome"], "completed");
    assert_eq!(report["failure_identity"], "");
    observation(
        "soft/unstarted-unrequested",
        json!({"configured_handle":configuration.incarnations[0].handle,"incarnations":[],"oracle_plan":null}),
        report,
    );
}

fn soft_publication(
    stdout: &[u8],
    stderr: &[u8],
    offered: u64,
    report_only: bool,
) -> crate::evidence::store::Publication {
    let mut publication = metadata_publication();
    let mut inputs: Inputs = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
    inputs.configuration.incarnations[0].workload_kind = "go-test".into();
    inputs.configuration.oracle_plan = soft_configuration(report_only).oracle_plan;
    let handle = inputs.configuration.incarnations[0].handle.clone();
    let mut execution = soft_execution(&inputs.configuration, stdout, stderr);
    execution.incarnations[0].execution.outcome.output_bytes = offered;
    let mut streams = serde_json::Map::new();
    for (name, raw) in [("stdout", stdout), ("stderr", stderr)] {
        let file = format!(
            "evidence/cluster/{}/{}/{name}",
            handle.node, handle.incarnation
        );
        let descriptor = Payload {
            file: file.clone(),
            bytes: raw.len() as u64,
            sha256: canonical::digest(raw),
            limit: inputs.configuration.limits.output_bytes,
        };
        streams.insert(
            name.into(),
            json!({"bytes":raw.len(),"sha256":descriptor.sha256}),
        );
        inputs.retained[0].evidence.insert(name.into(), descriptor);
        publication.payloads.insert(file, raw.to_vec());
    }
    let summary = json!({"schema":"gomad.cluster-observation/v1","incarnations":[{
        "handle":handle,"termination":"exit","exit_code":0,"reason":"","streams":streams,
        "offered_output_bytes":offered,"interrupted":false,"local_model":null}],
        "world_sha256":canonical::digest(&[]),"network_sha256":canonical::digest(&[]),"virtual_elapsed_nanos":0});
    publication.payloads.insert(
        "evidence/cluster_summary".into(),
        canonical::encode(&summary).unwrap(),
    );
    let report = oracles::assess(&execution, &inputs.configuration).unwrap();
    publication.payloads.insert(
        "evidence/cluster_oracles".into(),
        canonical::encode(&report).unwrap(),
    );
    publication
        .payloads
        .insert(FILE.into(), canonical::encode(&inputs).unwrap());
    publication.record.identity.configuration =
        canonical::digest(&canonical::encode(&inputs.configuration).unwrap());
    publication.record.execution_identity = publication.record.identity.digest().unwrap();
    reseal_evidence(&mut publication);
    publication
}

fn reseal_evidence(publication: &mut crate::evidence::store::Publication) {
    for (file, raw) in &publication.payloads {
        if file.starts_with("evidence/") {
            publication.record.streams.insert(
                file.clone(),
                crate::evidence::record::Stream {
                    bytes: raw.len() as u64,
                    sha256: canonical::digest(raw),
                    records: 0,
                },
            );
        }
    }
}

fn reseal_oracle_report(report: &mut Value) {
    for oracle in report["oracles"].as_array_mut().unwrap() {
        oracle["identity"] = json!("");
        oracle["failure_identity"] = json!("");
        if oracle["passed"] != true {
            oracle["failure_identity"] = json!(canonical::digest(
                &canonical::encode(&json!({"name":oracle["name"],"evidence":oracle["evidence"]}))
                    .unwrap()
            ));
        }
        oracle["identity"] = json!(canonical::digest(&canonical::encode(oracle).unwrap()));
    }
    let failed: Vec<_> = report["oracles"]
        .as_array()
        .unwrap()
        .iter()
        .filter(|item| item["passed"] != true)
        .cloned()
        .collect();
    report["failure_identity"] = json!(if failed.is_empty() {
        String::new()
    } else {
        canonical::digest(&canonical::encode(&failed).unwrap())
    });
}

fn forge_retained_oracle(case: &str, inputs: &mut Inputs, summary: &mut Value, report: &mut Value) {
    match case {
        "valid" | "noncanonical-summary" => (),
        "invalid-stream-summary" => {
            summary["incarnations"][0]["streams"] = json!([]);
        }
        "forged-message" => {
            let raw = canonical::encode(&json!(["forged invariant"])).unwrap();
            report["oracles"][0]["evidence"][1]["value"] =
                json!(crate::campaign::exploration::base64(&raw));
            report["oracles"][0]["evidence"][1]["full_sha256"] = json!(canonical::digest(&raw));
        }
        "forged-instance" => {
            report["oracles"][0]["name"] = json!(format!(
                "soft-assertion.instance-{}",
                canonical::digest(b"foreign/1")
            ));
            report["oracles"][0]["evidence"][0]["value"] =
                json!(crate::campaign::exploration::base64(b"foreign/1"));
            report["oracles"][0]["evidence"][0]["full_sha256"] =
                json!(canonical::digest(b"foreign/1"));
        }
        "missing-output" => {
            inputs.retained[0].evidence.clear();
        }
        "forged-digest" => {
            inputs.retained[0]
                .evidence
                .get_mut("stderr")
                .unwrap()
                .sha256 = canonical::digest(b"forged output");
            summary["incarnations"][0]["streams"]["stderr"]["sha256"] =
                json!(canonical::digest(b"forged output"));
        }
        "generic" | "mixed" => {
            let generic = json!({"name":"generic","passed":false,"inconclusive":false,"soft_assertion":false,"report_only":false,"evidence":[],"failure_identity":"","identity":""});
            if case == "generic" {
                report["oracles"] = json!([generic]);
            } else {
                report["oracles"].as_array_mut().unwrap().push(generic);
            }
        }
        "truncated-pass" => {
            summary["incarnations"][0]["offered_output_bytes"] = json!(100);
        }
        _ => unreachable!(),
    }
}

#[test]
fn retained_oracle_report_is_rederived_from_authenticated_actual_outputs() {
    for case in [
        "valid",
        "forged-message",
        "forged-instance",
        "missing-output",
        "forged-digest",
        "generic",
        "mixed",
        "truncated-pass",
        "noncanonical-summary",
        "invalid-stream-summary",
    ] {
        let stderr = b"failed assertion: invalid state\n";
        let mut publication = soft_publication(b"PASS\n", stderr, (5 + stderr.len()) as u64, false);
        let control = ReaderFixture::new(soft_publication(
            b"PASS\n",
            stderr,
            (5 + stderr.len()) as u64,
            false,
        ));
        let opened = control.open();
        let inputs = super::inputs(&opened).unwrap();
        super::bind(&opened, &inputs).unwrap();
        oracles::authenticate_retained(&opened, &inputs).unwrap();
        let mut inputs: Inputs = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
        let mut summary: Value =
            serde_json::from_slice(&publication.payloads["evidence/cluster_summary"]).unwrap();
        let mut report: Value =
            serde_json::from_slice(&publication.payloads["evidence/cluster_oracles"]).unwrap();
        forge_retained_oracle(case, &mut inputs, &mut summary, &mut report);
        reseal_oracle_report(&mut report);
        publication
            .payloads
            .insert(FILE.into(), canonical::encode(&inputs).unwrap());
        publication.payloads.insert(
            "evidence/cluster_summary".into(),
            if case == "noncanonical-summary" {
                serde_json::to_vec_pretty(&summary).unwrap()
            } else {
                canonical::encode(&summary).unwrap()
            },
        );
        publication.payloads.insert(
            "evidence/cluster_oracles".into(),
            canonical::encode(&report).unwrap(),
        );
        reseal_evidence(&mut publication);
        let fixture = ReaderFixture::new(publication);
        let opened = fixture.open();
        opened.authenticate_all().unwrap();
        let inputs = super::inputs(&opened).unwrap();
        super::bind(&opened, &inputs).unwrap();
        let error = oracles::authenticate_retained(&opened, &inputs).err();
        assert_eq!(error.is_none(), case == "valid", "{case}: {error:?}");
        if let Some(error) = &error {
            assert_eq!(
                error,
                match case {
                    "missing-output" => "retained cluster oracle output summary binding mismatch",
                    "forged-digest" => "retained cluster payload identity mismatch",
                    "noncanonical-summary" => "retained cluster oracle summary is not canonical",
                    "invalid-stream-summary" => "invalid retained cluster oracle stream summary",
                    _ => "retained cluster oracle report differs from actual output",
                }
            );
        }
        observation(
            &format!("retained-oracle/{case}"),
            json!({"retained_inputs":inputs,"summary":summary,"report":report,"stdout_base64":crate::campaign::exploration::base64(b"PASS\n"),"stderr_base64":crate::campaign::exploration::base64(stderr),"summary_base64":crate::campaign::exploration::base64(&opened.payload("evidence/cluster_summary",1<<20).unwrap()),"report_base64":crate::campaign::exploration::base64(&opened.payload("evidence/cluster_oracles",1<<20).unwrap())}),
            json!({"error":error}),
        );
    }
}

#[test]
fn retained_inconclusive_and_literal_braces_are_rederived_without_shortening() {
    for (case, output, message) in [
        (
            "literal-json",
            r#"failed assertion: invalid {"left":1}"#,
            r#"invalid {"left":1}"#,
        ),
        (
            "literal-tagged",
            "2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true}",
            "invalid {state}",
        ),
    ] {
        for shortened in [false, true] {
            let mut publication =
                soft_publication(b"", output.as_bytes(), output.len() as u64, false);
            if shortened {
                let mut report: Value =
                    serde_json::from_slice(&publication.payloads["evidence/cluster_oracles"])
                        .unwrap();
                let raw = canonical::encode(&json!(["invalid"])).unwrap();
                report["oracles"][0]["evidence"][1]["value"] =
                    json!(crate::campaign::exploration::base64(&raw));
                report["oracles"][0]["evidence"][1]["full_sha256"] = json!(canonical::digest(&raw));
                reseal_oracle_report(&mut report);
                publication.payloads.insert(
                    "evidence/cluster_oracles".into(),
                    canonical::encode(&report).unwrap(),
                );
                reseal_evidence(&mut publication);
            }
            let fixture = ReaderFixture::new(publication);
            let opened = fixture.open();
            let inputs = super::inputs(&opened).unwrap();
            let error = oracles::authenticate_retained(&opened, &inputs).err();
            assert_eq!(error.is_some(), shortened);
            if shortened {
                assert_eq!(
                    error.as_deref(),
                    Some("retained cluster oracle report differs from actual output")
                );
            }
            observation(
                &format!("retained-oracle/{case}/shortened={shortened}"),
                json!({"stderr_base64":crate::campaign::exploration::base64(output.as_bytes()),"expected_message":message,"retained_report_base64":crate::campaign::exploration::base64(&opened.payload("evidence/cluster_oracles",1<<20).unwrap())}),
                json!({"error":error}),
            );
        }
    }
    for report_only in [false, true] {
        let fixture = ReaderFixture::new(soft_publication(b"", b"", 100, report_only));
        let opened = fixture.open();
        let inputs = super::inputs(&opened).unwrap();
        oracles::authenticate_retained(&opened, &inputs).unwrap();
        let report: Value =
            serde_json::from_slice(&opened.payload("evidence/cluster_oracles", 1 << 20).unwrap())
                .unwrap();
        assert_eq!(report["oracles"][0]["passed"], false);
        assert_eq!(report["oracles"][0]["inconclusive"], true);
        observation(
            &format!("retained-oracle/truncated-inconclusive/report-only={report_only}"),
            json!({"stdout_base64":"","stderr_base64":"","offered_bytes":100}),
            report,
        );
    }
}

#[cfg(feature = "determinism-fixture")]
const PRODUCER_OUTPUT: &[u8] = b"PASS\nfailed assertion: invalid {state}\n";

#[cfg(feature = "determinism-fixture")]
fn producer_module(corpus: &std::path::Path) -> Vec<u8> {
    let original = std::fs::read_to_string(corpus.join("targets/inspect-choice.wat")).unwrap();
    let module=original.replacen("(module", r#"(module
  (import "wasi_snapshot_preview1" "fd_write" (func $fd_write (param i32 i32 i32 i32) (result i32)))"#,1)
        .replacen(r#"  (func (export "_start")"#,r#"  (data (i32.const 256) "PASS\0afailed assertion: invalid {state}\0a")
  (func (export "_start")"#,1)
        .replacen("    i32.const 16 i32.const 16 call $finish",&format!(
            "    i32.const 240 i32.const 256 i32.store\n    i32.const 244 i32.const {} i32.store\n    i32.const 1 i32.const 240 i32.const 1 i32.const 248 call $fd_write if unreachable end\n    i32.const 248 i32.load i32.const {} i32.ne if unreachable end\n    i32.const 16 i32.const 16 call $finish",PRODUCER_OUTPUT.len(),PRODUCER_OUTPUT.len()),1).into_bytes();
    assert_ne!(module, original.as_bytes());
    module
}

#[cfg(feature = "determinism-fixture")]
fn assert_cluster_producer(
    configuration: &Configuration,
    toolchain: &str,
    modules: &BTreeMap<String, Vec<u8>>,
    report_only: bool,
) {
    let publication = crate::fixture_cluster::execute_configuration(
        configuration.clone(),
        toolchain.to_owned(),
        modules.clone(),
    )
    .unwrap();
    let fixture = ReaderFixture::new(publication);
    let opened = fixture.open();
    let inputs = super::inputs(&opened).unwrap();
    let descriptors: Vec<_> = inputs
        .retained
        .iter()
        .flat_map(|item| item.evidence.iter())
        .filter(|(name, _)| matches!(name.as_str(), "stdout" | "stderr"))
        .map(|(_, value)| value)
        .collect();
    assert!(!descriptors.is_empty());
    assert!(
        descriptors
            .iter()
            .all(|item| item.limit == configuration.limits.evidence_bytes
                && item.limit > configuration.limits.output_bytes)
    );
    let mut actual_outputs = BTreeMap::new();
    for retained in &inputs.retained {
        let descriptor = retained.evidence.get("stdout").unwrap();
        let bytes = opened
            .payload(&descriptor.file, configuration.limits.evidence_bytes)
            .unwrap();
        assert_eq!(bytes, PRODUCER_OUTPUT);
        actual_outputs.insert(
            descriptor.file.clone(),
            crate::campaign::exploration::base64(&bytes),
        );
    }
    let report: Value = serde_json::from_slice(
        &opened
            .payload(
                "evidence/cluster_oracles",
                configuration.limits.evidence_bytes,
            )
            .unwrap(),
    )
    .unwrap();
    assert_eq!(
        report["oracles"].as_array().unwrap().len(),
        configuration.incarnations.len()
    );
    for oracle in report["oracles"].as_array().unwrap() {
        assert_eq!(oracle["passed"], report_only);
        assert_eq!(oracle["report_only"], report_only);
        assert_eq!(oracle["inconclusive"], false);
        let messages = crate::campaign::scenario::canonical_base64(
            oracle["evidence"][1]["value"].as_str().unwrap(),
        )
        .unwrap();
        assert_eq!(
            serde_json::from_slice::<Value>(&messages).unwrap(),
            json!(["invalid {state}"])
        );
    }
    oracles::authenticate_retained(&opened, &inputs).unwrap();
    let prepare_error = super::prepare(opened.try_clone().unwrap(), None).err();
    assert!(prepare_error.is_none(), "{prepare_error:?}");
    observation(
        &format!("producer/cluster-oracles/report-only={report_only}"),
        json!({"retained_inputs":inputs,"manifest":opened.manifest()}),
        json!({"prepare_error":prepare_error,"output_descriptor_count":descriptors.len(),"actual_outputs_base64":actual_outputs,"actual_report":report}),
    );
}

#[cfg(feature = "determinism-fixture")]
fn scenario_prepared(
    configuration: &Configuration,
    toolchain: String,
    module: Vec<u8>,
    report_only: bool,
) -> crate::campaign::prepare::Prepared {
    let scenario_path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../internal/backends/wasm/command/testdata/scenario/network.json");
    let mut scenario: Value =
        serde_json::from_slice(&std::fs::read(scenario_path).unwrap()).unwrap();
    scenario["actions"] = serde_json::to_value(&configuration.steps).unwrap();
    scenario["spec"]["links"] = json!([]);
    for node in scenario["spec"]["nodes"].as_array_mut().unwrap() {
        node["boot"] = json!("fixture");
    }
    scenario["boots"] = json!([{"id":"fixture","args":[]}]);
    scenario["network_faults"] = json!([]);
    scenario["incarnation_boots"] = json!([]);
    let scenario =
        crate::campaign::scenario::normalize(&canonical::encode(&scenario).unwrap()).unwrap();
    let item = &configuration.incarnations[0];
    crate::campaign::prepare::Prepared {
        auxiliary: BTreeMap::new(),
        compile_cache_root: None,
        module,
        configuration: super::super::preflight::Configuration {
            engine: configuration.engine.clone(),
            profile: configuration.profile.clone(),
            limits: configuration.limits.clone(),
            world_transition_bytes: None,
            wall_nanos: None,
            arguments: item.arguments.clone(),
            environment: item.environment.clone(),
            derive_seed_environment: false,
            network: None,
            runtime_sha256: Some(item.runtime_sha256.clone()),
            model: Some(item.model.clone()),
            namespace: Some(item.namespace.clone()),
            storage: None,
        },
        toolchain,
        choice_identity: None,
        scenario: Some(scenario),
        provenance: b"{}".to_vec(),
        network: None,
        soft_assertion_report_only: report_only,
    }
}

#[cfg(feature = "determinism-fixture")]
fn assert_scenario_producer(
    configuration: &Configuration,
    toolchain: String,
    modules: &BTreeMap<String, Vec<u8>>,
    report_only: bool,
) {
    let prepared = scenario_prepared(
        configuration,
        toolchain,
        modules["target/module.wasm"].clone(),
        report_only,
    );
    let execution = helper::execute_prepared(
        crate::fixture_cluster::prepare_configuration(configuration, modules).unwrap(),
    )
    .unwrap();
    let completion = crate::campaign::scenario::publication(
        &prepared,
        &crate::campaign::ledger::Job {
            ordinal: 0,
            seed: 7,
        },
        configuration.clone(),
        execution,
        false,
    )
    .unwrap()
    .finalized()
    .unwrap();
    let fixture = ReaderFixture::new(completion.publication);
    let opened = fixture.open();
    let inputs = super::inputs(&opened).unwrap();
    assert!(
        !opened
            .manifest()
            .record
            .streams
            .contains_key("evidence/cluster_oracles")
    );
    assert!(
        opened
            .manifest()
            .record
            .streams
            .contains_key("evidence/soft-assertions.json")
    );
    let scenario_report: Value = serde_json::from_slice(
        &opened
            .payload(
                "evidence/soft-assertions.json",
                configuration.limits.evidence_bytes,
            )
            .unwrap(),
    )
    .unwrap();
    assert_eq!(
        scenario_report["incarnations"].as_array().unwrap().len(),
        configuration.incarnations.len()
    );
    let mut actual_outputs = BTreeMap::new();
    for retained in &inputs.retained {
        let descriptor = retained.evidence.get("stdout").unwrap();
        assert_eq!(descriptor.limit, configuration.limits.evidence_bytes);
        assert!(descriptor.limit > configuration.limits.output_bytes);
        let bytes = opened
            .payload(&descriptor.file, configuration.limits.evidence_bytes)
            .unwrap();
        assert_eq!(bytes, PRODUCER_OUTPUT);
        actual_outputs.insert(
            descriptor.file.clone(),
            crate::campaign::exploration::base64(&bytes),
        );
    }
    for incarnation in scenario_report["incarnations"].as_array().unwrap() {
        assert_eq!(
            incarnation["report"]["messages"],
            json!(["invalid {state}"])
        );
        assert_eq!(incarnation["report"]["report_only"], report_only);
        assert_eq!(incarnation["report"]["inconclusive"], false);
    }
    let before = opened.payload(FILE, 16 << 20).unwrap();
    let prepare_error = super::prepare(opened.try_clone().unwrap(), None).err();
    assert!(prepare_error.is_none(), "{prepare_error:?}");
    assert_eq!(opened.payload(FILE, 16 << 20).unwrap(), before);
    observation(
        &format!("producer/scenario/report-only={report_only}"),
        json!({"retained_inputs":inputs,"manifest":opened.manifest()}),
        json!({"prepare_error":prepare_error,"cluster_oracles_present":opened.manifest().record.streams.contains_key("evidence/cluster_oracles"),"scenario_report_present":opened.manifest().record.streams.contains_key("evidence/soft-assertions.json"),"actual_outputs_base64":actual_outputs,"actual_report":scenario_report}),
    );
}

#[cfg(feature = "determinism-fixture")]
#[test]
fn actual_native_producers_admit_distinct_output_and_evidence_bounds() {
    let corpus = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let module = producer_module(&corpus);
    for report_only in [false, true] {
        let initial = crate::fixture_cluster::publication(&corpus, 7).unwrap();
        let source: Inputs = serde_json::from_slice(&initial.payloads[FILE]).unwrap();
        let mut configuration = source.configuration;
        assert!(configuration.limits.evidence_bytes > configuration.limits.output_bytes);
        for item in &mut configuration.incarnations {
            item.workload_kind = "go-test".into();
            item.module = Payload {
                file: "target/module.wasm".into(),
                bytes: module.len() as u64,
                sha256: canonical::digest(&module),
                limit: 512 << 20,
            };
            item.choice_identity.target_sha256 = sha2::Sha256::digest(&module).into();
            item.imports = None;
        }
        configuration.oracle_plan=Some(serde_json::from_value(json!({
            "schema":"gomad.wasm.soft-assertion-observation/v1","maximum_bytes":4096,
            "soft_assertions":configuration.incarnations.iter().map(|item|json!({"handle":item.handle,"report_only":report_only})).collect::<Vec<_>>()
        })).unwrap());
        let modules = BTreeMap::from([("target/module.wasm".into(), module.clone())]);
        assert_cluster_producer(
            &configuration,
            &source.toolchain_sha256,
            &modules,
            report_only,
        );
        configuration.oracle_plan = None;
        assert_scenario_producer(
            &configuration,
            source.toolchain_sha256,
            &modules,
            report_only,
        );
    }
}

#[test]
fn retained_ordinary_json_spellings_keep_exact_raw_authentication() {
    let corpus = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let baseline = crate::fixture_cluster::publication(&corpus, 7).unwrap();
    let raw = &baseline.payloads[FILE];
    let expected: Inputs = serde_json::from_slice(raw).unwrap();
    let value: Value = serde_json::from_slice(raw).unwrap();
    let reversed = format!(
        "{{{}}}",
        value
            .as_object()
            .unwrap()
            .iter()
            .rev()
            .map(|(key, value)| format!(
                "{}:{}",
                serde_json::to_string(key).unwrap(),
                serde_json::to_string(value).unwrap()
            ))
            .collect::<Vec<_>>()
            .join(",")
    )
    .into_bytes();
    let mut explicit_null = value.clone();
    explicit_null["configuration"]["oracle_plan"] = Value::Null;
    for (case, bytes) in [
        ("canonical", raw.clone()),
        ("indented", serde_json::to_vec_pretty(&value).unwrap()),
        ("member-order", reversed),
        (
            "escaped-member",
            String::from_utf8(raw.clone())
                .unwrap()
                .replacen("\"schema\":", "\"\\u0073chema\":", 1)
                .into_bytes(),
        ),
        (
            "escaped-value",
            String::from_utf8(raw.clone())
                .unwrap()
                .replacen(
                    "gomad.cluster-replay-inputs",
                    "\\u0067omad.cluster-replay-inputs",
                    1,
                )
                .into_bytes(),
        ),
        ("trailing-space", [raw.as_slice(), b" \n\t\r"].concat()),
        (
            "explicit-optional-null",
            serde_json::to_vec(&explicit_null).unwrap(),
        ),
    ] {
        let mut publication = crate::fixture_cluster::publication(&corpus, 7).unwrap();
        publication.payloads.insert(FILE.into(), bytes.clone());
        let fixture = ReaderFixture::new(publication);
        let opened = fixture.open();
        opened.authenticate_all().unwrap();
        let admitted = super::inputs(&opened).unwrap_or_else(|error| panic!("{case}: {error}"));
        assert_eq!(
            canonical::encode(&admitted).unwrap(),
            canonical::encode(&expected).unwrap()
        );
        super::bind(&opened, &admitted).unwrap();
        for item in &admitted.configuration.incarnations {
            super::storage_input(item).unwrap();
        }
        let prepared = super::prepare(opened.try_clone().unwrap(), None).unwrap();
        assert!(prepared.available && prepared.exact);
        let replay = super::execute_projected(prepared.execution).unwrap();
        assert_eq!(
            crate::replay::divergence::first(&opened.manifest().record, &replay),
            None
        );
        assert_eq!(opened.payload(FILE, 16 << 20).unwrap(), bytes);
        let mut public_output = Vec::new();
        let public_status =
            crate::cli::replay::run(&[fixture.path.to_str().unwrap().into()], &mut public_output)
                .unwrap_or_else(|error| panic!("{}: {}", error.status, error.message));
        assert_eq!(public_status, 0);
        let mut appended = bytes.clone();
        appended.push(b'\n');
        std::fs::write(fixture.path.join(FILE), &appended).unwrap();
        let error = opened.payload(FILE, 16 << 20).unwrap_err();
        observation(
            &format!("ordinary/{case}"),
            json!({"payload_base64":crate::campaign::exploration::base64(&bytes),"appended_base64":crate::campaign::exploration::base64(&appended),"manifest":opened.manifest(),"canonical_inputs":expected}),
            json!({"canonical_inputs":admitted,"raw_sha256":canonical::digest(&bytes),"appended_error":error,"public_status":public_status,"public_output_base64":crate::campaign::exploration::base64(&public_output)}),
        );
    }
}
#[path = "allocation_probe.rs"]
pub(crate) mod allocation_probe;
