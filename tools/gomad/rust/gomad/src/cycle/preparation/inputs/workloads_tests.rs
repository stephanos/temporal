use super::*;
use serde_json::json;

#[test]
fn workload_contracts_and_refusals() {
    let scenario = sqlite_scenario(false);
    assert_eq!(scenario["boots"][1]["args"], json!(["append-crash", "WAL"]));
    assert_eq!(sqlite_scenario(true)["actions"][3]["action"], "wait");
    assert_eq!(
        scenario["spec"]["limits"]["network_stall_nanos"],
        10_000_000_000_u64
    );
    assert!(policy("replication", None).is_err());
    assert!(inspection_policy("onebox", None).is_err());
    assert!(inspection_policy("unknown", None).is_err());
    assert!(guest_build_tags("unknown").is_err());
    assert_eq!(
        guest_build_tags("full").unwrap(),
        vec!["disable_grpc_modules", "sqlite3_dotlk", "test_dep"]
    );
    let capture = json!({"schema": ["original"]});
    assert_eq!(
        policy("recovery", Some(&capture)).unwrap()["CapturedInputs"],
        capture
    );
    assert_eq!(
        support_policy("retry", &capture).unwrap()["network_v2"]["faults"][0]["at_nanos"],
        40_000_000_000_u64
    );
    let spec = qualification_spec(
        std::path::Path::new("."),
        std::path::Path::new("./fresh-root"),
        "replication",
        &QualificationOptions {
            source: Some("./unchanged"),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(spec["Campaign"]["OverallTimeout"], 3_600_000_000_000_u64);
    assert_eq!(spec["Manifest"]["Repeat"], 2);
    assert!(
        !spec["Campaign"]["Environment"]
            .as_array()
            .unwrap()
            .iter()
            .any(|v| v.as_str().unwrap().starts_with("TZ="))
    );
    assert!(
        qualification_spec(
            std::path::Path::new("."),
            std::path::Path::new("."),
            "recovery",
            &QualificationOptions {
                source: Some("./unchanged"),
                canary: true,
                ..Default::default()
            }
        )
        .is_err()
    );
}

#[test]
fn unchanged_staging_reopens_bytes_and_refuses_replacement() {
    use std::{fs, os::unix::fs::symlink};
    let root = std::env::temp_dir().join(format!("gomad-workload-custody-{}", std::process::id()));
    fs::create_dir_all(root.join(WASM).join("testdata/temporal_storage")).unwrap();
    let original = root
        .join(WASM)
        .join("testdata/temporal_storage/recovery_test.go");
    fs::write(&original, b"package unchanged\n").unwrap();
    let destination = root.join("staged");
    let manifest = stage_temporal(&root, &destination, "recovery").unwrap();
    assert_eq!(
        manifest[original.to_str().unwrap()],
        super::super::digest(b"package unchanged\n")
    );
    assert_eq!(
        stage_temporal(&root, &destination, "recovery").unwrap(),
        manifest
    );
    fs::write(destination.join("recovery_test.go"), b"replacement").unwrap();
    assert!(
        stage_temporal(&root, &destination, "recovery")
            .unwrap_err()
            .contains("differs")
    );
    fs::remove_file(destination.join("recovery_test.go")).unwrap();
    symlink(&original, destination.join("recovery_test.go")).unwrap();
    assert!(stage_temporal(&root, &destination, "recovery").is_err());
    fs::remove_file(destination.join("recovery_test.go")).unwrap();
    fs::write(destination.join("unexpected.go"), b"package extra").unwrap();
    assert!(
        stage_temporal(&root, &destination, "recovery")
            .unwrap_err()
            .contains("unexpected")
    );
    fs::remove_dir_all(&root).unwrap();
}

#[test]
fn original_scenario_and_policy_bytes() {
    let capture = json!({"x": 1});
    let cases = [
        (
            sqlite_scenario(false),
            "d46ab0bcadd591d85db8edb32ca69b775530178749c83fee6ae6f286260be675",
        ),
        (
            sqlite_scenario(true),
            "9d57e44cfe00ab8c11059c80ee4fc6aa555e0b6d2355ba92cd4c5ac7022ab169",
        ),
        (
            temporal_replication_scenario(false),
            "e08cae5d4cf9fc79f247e721cf2c1ee12166e09d3f2d95aa84b714969effd9bd",
        ),
        (
            temporal_replication_scenario(true),
            "16635b53a8e3b0a9a996e72e865db59504c25e6dc6e3a8f3e36253796b7bd2f2",
        ),
        (
            temporal_recovery_scenario(),
            "92e884af479422059d3cca3327655faf965b77bc13beb236646f1f6166e8b65a",
        ),
        (
            inspection_scenario(json!(["-test.v"])),
            "4478bc4279758a7ed40868ae34134f7978d21a21f0bb9bccd77ea7a9028b5a3b",
        ),
        (
            policy("sqlite", None).unwrap(),
            "0c344004a04970c94ed698a3e6c3ffc32c5b2a2bea51627746ce1b60a48296aa",
        ),
        (
            policy("replication", Some(&capture)).unwrap(),
            "2b0feffb547319d81cc5a3974b1396a58d72cd7db93531082f015b532af3af6e",
        ),
        (
            policy("recovery", Some(&capture)).unwrap(),
            "826c5935f5a2606e5b893483a563a8ad17911f7036a46638b8501105026ae99c",
        ),
        (
            inspection_policy("shard", None).unwrap(),
            "05345b57d5f30c62cc6cdc532dce8e421d7c03972309070f05b23fad19ba5939",
        ),
        (
            inspection_policy("workflow", None).unwrap(),
            "05345b57d5f30c62cc6cdc532dce8e421d7c03972309070f05b23fad19ba5939",
        ),
        (
            inspection_policy("onebox", Some(&capture)).unwrap(),
            "bcde40cb055979270a71cab42fdbc437f97a237ae04c3be97008e3054ca5a433",
        ),
        (
            support_policy("smoke", &capture).unwrap(),
            "f3fe708e21b502e2e3a33d9fdfa7aaffa7891ed94aed2de22dcee0fd9169d43a",
        ),
        (
            support_policy("frontend", &capture).unwrap(),
            "5c5a3c6f2525d6e877998d32116332d767ba5478cb50062fdc93bea0fbc6cbce",
        ),
        (
            support_policy("retry", &capture).unwrap(),
            "5ff1f6416a8bbebc2cb3eaa7a7722562889297a3f8d62db6181267ab2c165a9d",
        ),
    ];
    for (value, expected) in cases {
        assert_eq!(
            super::super::digest(&super::super::canonical(&value).unwrap()),
            expected
        );
    }
}

fn guard_proof(
    line: u64,
    case: &str,
    control_input: Value,
    control: Value,
    mutation: Value,
    refusal: Result<Value, String>,
    predicate: &str,
) {
    let refusal = refusal.expect_err(case);
    assert!(
        refusal.contains(predicate),
        "{case}: expected {predicate:?}, got {refusal:?}"
    );
    println!(
        "WORKLOAD_PROOF {}",
        json!({
            "original_id": format!("workload-callback:workloads:raise:{line}:{}", match line {107|225=>12,112|205=>16,_=>8}),
            "case":case,"control_input":control_input,"control_input_sha256":digest(&canonical(&control_input).unwrap()),
            "control_outcome":"normal-return","control_output":control,"control_output_sha256":digest(&canonical(&control).unwrap()),
            "mutation_input":mutation,"mutation_input_sha256":digest(&canonical(&mutation).unwrap()),
            "mutation_outcome":"refusal","refusal":refusal,"intended_predicate":predicate
        })
    );
}

#[test]
fn original_policy_guard_controls_and_mutations() {
    let capture = json!({"x":1});
    for (line, family, message) in [
        (
            125,
            "policy",
            "actual authenticated schema/sqlite captured inputs required",
        ),
        (
            225,
            "inspection",
            "unchanged authenticated Temporal schema inputs required",
        ),
        (
            263,
            "support",
            "authenticated complete schema capture required",
        ),
    ] {
        let call = |value: Option<&Value>| match family {
            "policy" => policy("replication", value),
            "inspection" => inspection_policy("onebox", value),
            _ => support_policy("retry", value.unwrap_or(&Value::Null)),
        };
        let control = call(Some(&capture)).unwrap();
        guard_proof(
            line,
            &format!("{family}-missing-capture"),
            json!({"family":family,"captured_inputs":capture}),
            control.clone(),
            json!({"family":family,"captured_inputs":"absent"}),
            call(None),
            message,
        );
        for changed in [
            Value::Null,
            json!(false),
            json!(0),
            json!(""),
            json!([]),
            json!({}),
        ] {
            guard_proof(
                line,
                &format!("{family}-false-capture-{changed}"),
                json!({"family":family,"captured_inputs":capture}),
                control.clone(),
                json!({"family":family,"captured_inputs":changed}),
                call(Some(&changed)),
                message,
            );
        }
    }
    for profile in ["", "slim", "full"] {
        let control = json!(guest_build_tags(profile).unwrap());
        guard_proof(
            151,
            &format!("guest-profile-{profile}"),
            json!({"profile":profile}),
            control,
            json!({"profile":"FULL"}),
            guest_build_tags("FULL").map(|v| json!(v)),
            "qualification guest profile must be full or slim",
        );
    }
    guard_proof(
        218,
        "inspection-owner",
        json!({"owner":"shard","captured_inputs":null}),
        inspection_policy("shard", None).unwrap(),
        json!({"owner":"unknown","captured_inputs":null}),
        inspection_policy("unknown", None),
        "unknown inspection owner",
    );
}

#[test]
fn original_qualification_guard_controls_and_mutations() {
    let path = Path::new(".");
    let root = Path::new("./fresh-root");
    let successful = QualificationOptions {
        source: Some("./unchanged"),
        ..Default::default()
    };
    let control_input = json!({"repo":".","root":"./fresh-root","workload":"replication","source":"./unchanged","canary":false,"original_clusters":false});
    let control = qualification_spec(path, root, "replication", &successful).unwrap();
    let clusters = QualificationOptions {
        source: Some("./unchanged"),
        original_clusters: true,
        ..Default::default()
    };
    let clusters_control = qualification_spec(path, root, "replication", &clusters).unwrap();
    assert_eq!(clusters_control["Manifest"]["Repeat"], 1);
    for (workload, canary) in [
        ("sqlite", false),
        ("recovery", false),
        ("replication", true),
    ] {
        let options = QualificationOptions {
            source: Some("./unchanged"),
            original_clusters: true,
            canary,
            ..Default::default()
        };
        guard_proof(
            160,
            &format!("original-clusters-{workload}-{canary}"),
            json!({"workload":"replication","source":"./unchanged","original_clusters":true,"canary":false}),
            clusters_control.clone(),
            json!({"workload":workload,"source":"./unchanged","original_clusters":true,"canary":canary}),
            qualification_spec(path, root, workload, &options),
            "original clusters requires one successful replication run",
        );
    }
    guard_proof(
        162,
        "unknown-workload",
        control_input.clone(),
        control.clone(),
        json!({"workload":"unknown","source":"./unchanged"}),
        qualification_spec(path, root, "unknown", &successful),
        "unknown mandatory workload",
    );
    for workload in ["replication", "recovery"] {
        let valid = qualification_spec(path, root, workload, &successful).unwrap();
        for source in [None, Some("")] {
            guard_proof(
                164,
                &format!("missing-source-{workload}-{source:?}"),
                json!({"workload":workload,"source":"./unchanged"}),
                valid.clone(),
                json!({"workload":workload,"source":source}),
                qualification_spec(
                    path,
                    root,
                    workload,
                    &QualificationOptions {
                        source,
                        ..Default::default()
                    },
                ),
                "fresh unchanged staged Temporal source required",
            );
        }
    }
    guard_proof(
        166,
        "recovery-canary",
        json!({"workload":"recovery","source":"./unchanged","canary":false}),
        qualification_spec(path, root, "recovery", &successful).unwrap(),
        json!({"workload":"recovery","source":"./unchanged","canary":true}),
        qualification_spec(
            path,
            root,
            "recovery",
            &QualificationOptions {
                source: Some("./unchanged"),
                canary: true,
                ..Default::default()
            },
        ),
        "recovery catalogue is an original success gate",
    );
    let support_control = support_spec(
        path,
        root,
        "retry",
        &SupportOptions {
            source: Some("./unchanged"),
            ..Default::default()
        },
    )
    .unwrap();
    for (source, canary) in [
        (None, false),
        (Some(""), false),
        (Some("./unchanged"), true),
    ] {
        guard_proof(
            279,
            &format!("retry-target-{source:?}-{canary}"),
            json!({"row":"retry","source":"./unchanged","canary":false}),
            support_control.clone(),
            json!({"row":"retry","source":source,"canary":canary}),
            support_spec(
                path,
                root,
                "retry",
                &SupportOptions {
                    source,
                    canary,
                    ..Default::default()
                },
            ),
            "named reset requires unchanged staged original success target",
        );
    }
    for row in ["smoke", "frontend"] {
        let value = support_spec(
            path,
            root,
            row,
            &SupportOptions {
                canary: true,
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(value["Campaign"]["Target"]["Args"][3], "-test.run=[");
    }
    let sqlite =
        qualification_spec(path, root, "sqlite", &QualificationOptions::default()).unwrap();
    assert_eq!(
        sqlite["Campaign"]["Target"]["BuildTags"],
        json!(["sqlite3_dotlk"])
    );
    assert_eq!(sqlite["Campaign"]["OverallTimeout"], 180_000_000_000_u64);
}

struct WorkloadFixture(PathBuf);
impl WorkloadFixture {
    fn new(name: &str) -> Self {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "gomad-workload-{name}-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&root).unwrap();
        Self(root)
    }
    fn write(&self, path: &str, body: &[u8]) -> PathBuf {
        let target = self.0.join(path);
        fs::create_dir_all(target.parent().unwrap()).unwrap();
        fs::write(&target, body).unwrap();
        target
    }
}
impl Drop for WorkloadFixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn original_temporal_staging_guard_controls_and_mutations() {
    use std::os::unix::fs::symlink;
    let fixture = WorkloadFixture::new("temporal");
    let source = fixture.write(
        &format!("{WASM}/testdata/temporal_storage/recovery_test.go"),
        b"package unchanged\n",
    );
    let destination = fixture.0.join("staged");
    let control = stage_temporal(&fixture.0, &destination, "recovery").unwrap();
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(
        destination.join("recovery_test.go"),
        fs::Permissions::from_mode(0o644),
    )
    .unwrap();
    assert_eq!(
        stage_temporal(&fixture.0, &destination, "recovery").unwrap(),
        control
    );
    let input = json!({"repo":fixture.0,"destination":destination,"workload":"recovery","source_bytes":b"package unchanged\n","source_sha256":digest(b"package unchanged\n")});
    fixture.write("staged/extra.go", b"package extra\n");
    guard_proof(
        102,
        "temporal-unexpected-file",
        input.clone(),
        control.clone(),
        json!({"base":input,"added_entry":"extra.go","added_bytes":b"package extra\n"}),
        stage_temporal(&fixture.0, &destination, "recovery"),
        "unexpected staged Temporal input",
    );
    fs::remove_file(destination.join("extra.go")).unwrap();
    for kind in ["bytes", "symlink", "directory"] {
        let target = destination.join("recovery_test.go");
        fs::remove_file(&target).unwrap();
        match kind {
            "bytes" => {
                fs::write(&target, b"replacement").unwrap();
            }
            "symlink" => symlink(&source, &target).unwrap(),
            _ => fs::create_dir(&target).unwrap(),
        }
        guard_proof(
            112,
            &format!("temporal-staged-{kind}"),
            input.clone(),
            control.clone(),
            json!({"base":input,"target_kind":kind,"replacement_bytes":b"replacement","symlink_target":source}),
            stage_temporal(&fixture.0, &destination, "recovery"),
            "staged Temporal source differs from unchanged original",
        );
        if kind == "directory" {
            fs::remove_dir(&target).unwrap();
        } else {
            fs::remove_file(&target).unwrap();
        }
        stage_temporal(&fixture.0, &destination, "recovery").unwrap();
    }
    fs::remove_dir_all(&destination).unwrap();
    for kind in ["directory", "symlink"] {
        fs::remove_file(&source).unwrap();
        if kind == "directory" {
            fs::create_dir(&source).unwrap();
        } else {
            let other = fixture.write("unchanged.go", b"package unchanged\n");
            symlink(&other, &source).unwrap();
        }
        guard_proof(
            107,
            &format!("temporal-original-{kind}"),
            input.clone(),
            control.clone(),
            json!({"base":input,"source_kind":kind,"symlink_target":"unchanged.go"}),
            stage_temporal(&fixture.0, &destination, "recovery"),
            "original Temporal input is not regular",
        );
        if kind == "directory" {
            fs::remove_dir(&source).unwrap();
        } else {
            fs::remove_file(&source).unwrap();
        }
        fs::write(&source, b"package unchanged\n").unwrap();
    }
    fs::remove_dir_all(&destination).unwrap();
    symlink(&fixture.0, &destination).unwrap();
    guard_proof(
        102,
        "temporal-destination-symlink",
        input.clone(),
        control.clone(),
        json!({"base":input,"destination_kind":"symlink","symlink_target":fixture.0}),
        stage_temporal(&fixture.0, &destination, "recovery"),
        "unexpected staged Temporal input",
    );
    fs::remove_file(&destination).unwrap();
    fixture.write(
        &format!("{WASM}/testdata/temporal_broker/replication_test.go"),
        b"package replication\n",
    );
    assert_eq!(
        stage_temporal(&fixture.0, &destination, "replication")
            .unwrap()
            .as_object()
            .unwrap()
            .len(),
        2
    );
    fixture.write(
        &format!("{WASM}/testdata/network_streams/z.go"),
        b"package retry\n",
    );
    fixture.write(
        &format!("{WASM}/testdata/network_streams/a.go"),
        b"package retry\n",
    );
    let retry = stage_temporal(&fixture.0, &fixture.0.join("retry"), "retry").unwrap();
    assert_eq!(retry.as_object().unwrap().len(), 2);
}

#[test]
fn original_inspection_staging_guard_controls_and_mutations() {
    use std::os::unix::fs::symlink;
    let fixture = WorkloadFixture::new("inspection");
    let source = fixture.write("service/history/shard/original.go", b"package shard\n");
    fixture.write("service/history/shard/excluded_test.go", b"package shard\n");
    fixture.write(
        "service/history/shard/gomad_debug_test.go",
        b"package shard\n",
    );
    let destination = fixture.0.join("staged");
    let input = json!({"owner":"shard","repo":fixture.0,"destination":destination,"source_bytes":b"package shard\n","source_sha256":digest(b"package shard\n"),"source_names":["original.go","excluded_test.go","gomad_debug_test.go"]});
    let control = stage_inspection(&fixture.0, &destination, "shard").unwrap();
    assert_eq!(control.as_object().unwrap().len(), 2);
    assert!(!destination.join("excluded_test.go").exists());
    fixture.write("staged/extra.go", b"package extra\n");
    guard_proof(
        198,
        "inspection-unexpected-file",
        input.clone(),
        control.clone(),
        json!({"base":input,"added_entry":"extra.go","added_bytes":b"package extra\n"}),
        stage_inspection(&fixture.0, &destination, "shard"),
        "unexpected staged inspection input",
    );
    fs::remove_file(destination.join("extra.go")).unwrap();
    for kind in ["bytes", "symlink", "directory"] {
        let target = destination.join("original.go");
        fs::remove_file(&target).unwrap();
        match kind {
            "bytes" => {
                fs::write(&target, b"replacement").unwrap();
            }
            "symlink" => symlink(&source, &target).unwrap(),
            _ => fs::create_dir(&target).unwrap(),
        }
        guard_proof(
            205,
            &format!("inspection-staged-{kind}"),
            input.clone(),
            control.clone(),
            json!({"base":input,"target_kind":kind,"replacement_bytes":b"replacement","symlink_target":source}),
            stage_inspection(&fixture.0, &destination, "shard"),
            "staged inspection source differs from unchanged original",
        );
        if kind == "directory" {
            fs::remove_dir(&target).unwrap();
        } else {
            fs::remove_file(&target).unwrap();
        }
        stage_inspection(&fixture.0, &destination, "shard").unwrap();
    }
    fs::remove_dir_all(&destination).unwrap();
    symlink(&fixture.0, &destination).unwrap();
    guard_proof(
        198,
        "inspection-destination-symlink",
        input.clone(),
        control.clone(),
        json!({"base":input,"destination_kind":"symlink","symlink_target":fixture.0}),
        stage_inspection(&fixture.0, &destination, "shard"),
        "unexpected staged inspection input",
    );
    fs::remove_file(&destination).unwrap();
    stage_inspection(&fixture.0, &destination, "shard").unwrap();
    fs::remove_file(&source).unwrap();
    fs::remove_file(fixture.0.join("service/history/shard/gomad_debug_test.go")).unwrap();
    fs::remove_dir_all(&destination).unwrap();
    guard_proof(
        212,
        "inspection-empty-selection",
        input.clone(),
        control,
        json!({"base":input,"removed_sources":["original.go","gomad_debug_test.go"],"remaining_sources":["excluded_test.go"]}),
        stage_inspection(&fixture.0, &destination, "shard"),
        "no original inspection source selected",
    );
    fixture.write(
        &format!("{WASM}/testdata/prototype_temporal/onebox_test.go"),
        b"package onebox\n",
    );
    assert_eq!(
        stage_inspection(&fixture.0, &fixture.0.join("onebox"), "onebox")
            .unwrap()
            .as_object()
            .unwrap()
            .len(),
        1
    );
    fixture.write(
        "service/history/workflow/original.go",
        b"package workflow\n",
    );
    assert_eq!(
        stage_inspection(&fixture.0, &fixture.0.join("workflow"), "workflow")
            .unwrap()
            .as_object()
            .unwrap()
            .len(),
        1
    );
}

#[test]
fn original_qualification_and_support_request_bytes() {
    let repo = Path::new("/gomad-workload-contract/repo");
    let root = Path::new("/gomad-workload-contract/out");
    for (workload, canary, original_clusters, guest_profile, expected) in [
        (
            "sqlite",
            false,
            false,
            "",
            "90fcb4bebb4eec1eb7e5ffacd5f09716d1c51cc6cae634fca6c155c46ecd3e8c",
        ),
        (
            "sqlite",
            true,
            false,
            "",
            "f433cc165d3b4eb13f570eed84d8ac181a5337861071186bfabd521f861cf7d7",
        ),
        (
            "recovery",
            false,
            false,
            "",
            "f168e5d463c4eaf86db02894ccc2dc38adc7f5bd32d70512ce973e2123f62aa9",
        ),
        (
            "replication",
            false,
            false,
            "",
            "63a51b228c62fa305589becd294c30e6bc8b66c7ce70d59a3f6320e3a4f4c7ba",
        ),
        (
            "replication",
            true,
            false,
            "",
            "4486f25665fa7aeb203728a94d789710ebf50e94858a30992221f2258293ba74",
        ),
        (
            "replication",
            false,
            true,
            "full",
            "e94e5d8d9c2110bbbff45a3c3598429aa2c09fc0398d212cee42ff307ee45956",
        ),
    ] {
        let value = qualification_spec(
            repo,
            root,
            workload,
            &QualificationOptions {
                source: Some("./unchanged"),
                canary,
                original_clusters,
                guest_profile,
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(
            digest(&canonical(&value).unwrap()),
            expected,
            "{workload}/{canary}/{original_clusters}/{guest_profile}"
        );
    }
    for (row, canary, expected) in [
        (
            "smoke",
            false,
            "71612b00335650f972bd59fe68037cb27cf67fd3979be2c15fa79ab473e3f6f2",
        ),
        (
            "smoke",
            true,
            "71d08f8ce10c6ac579739c1f644acd46c547bfef07d4bf5add9a363097a75120",
        ),
        (
            "frontend",
            false,
            "654d3520cbe261237aead552d6be883ab63e3af5e98b1a0ae303e203cd5f26ba",
        ),
        (
            "frontend",
            true,
            "81178ef5d25d2ca59d88db4b27976c02d563c06ddc5737e7e88828e331442596",
        ),
        (
            "retry",
            false,
            "ed69708d72e9413beba5c2dd8fc1fdac59eb45dc0fc0112594fc735f17f45e63",
        ),
    ] {
        let value = support_spec(
            repo,
            root,
            row,
            &SupportOptions {
                source: if row == "retry" {
                    Some("./unchanged")
                } else {
                    None
                },
                canary,
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(
            digest(&canonical(&value).unwrap()),
            expected,
            "{row}/{canary}"
        );
    }
    let fixture = WorkloadFixture::new("manifest");
    let manifest = fixture.write("original.json", b"{ \"b\":2, \"a\":1 }\n");
    let expected = "sha256:1b4e8ba7af3723a8ec7a07775aaa8a50e264fad675f4763d0d733eb38b1df673";
    let value = qualification_spec(
        repo,
        root,
        "sqlite",
        &QualificationOptions {
            source_manifest: Some(&manifest),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(value["Manifest"]["IntegratedSourceSHA256"], expected);
    assert_eq!(value["Manifest"]["Provisional"], false);
    let value = support_spec(
        repo,
        root,
        "smoke",
        &SupportOptions {
            source_manifest: Some(&manifest),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(value["Manifest"]["IntegratedSourceSHA256"], expected);
}
