use super::*;
use std::os::unix::fs::PermissionsExt as _;

fn policy_fixture(selected: Selection) -> QualificationResult {
    let mut manifest = json!({});
    for field in [
        "schema",
        "id",
        "workload",
        "profile",
        "integrated_source_sha256",
        "workload_source_sha256",
        "runtime_sha256",
        "compiler_sha256",
        "helper_sha256",
        "host_sha256",
        "model_sha256",
        "engine_sha256",
        "host_config_sha256",
        "module_sha256",
        "toolchain_sha256",
        "host_os",
        "host_arch",
    ] {
        manifest[field] = json!("fixture");
    }
    manifest["provisional"] = json!(false);
    manifest["cell"] = json!("success");
    manifest["profile"] = json!("cooperative");
    manifest["seed"] = json!(7);
    manifest["repeat"] = json!(2);
    manifest["fuel"] = json!(1_u64 << 50);
    manifest["memory_bytes"] = json!(4_u64 << 30);
    manifest["unsupported"] = json!([]);
    // Policy tests use a minimal captured namespace; retained Result tests
    // separately authenticate its complete payload inventory.
    let entropy = hex_bytes(&canonical::digest(b"gomad-wasm-prototype/v1")).unwrap();
    let mut preopens = vec![
        b"/workspace".to_vec(),
        b"/tmp".to_vec(),
        selected.directory.as_bytes().to_vec(),
        b"/workspace/schema/sqlite".to_vec(),
    ];
    preopens.sort();
    let model = json!({"entropy_key":entropy,"working_directory_sha256":canonical::digest(selected.directory.as_bytes()),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},"mode":1,"limit":64<<20,"diagnostics":false,"epoch_nanos":946_684_800_000_000_000_u64,"read_step_nanos":0,"now":1,"limits":{"calls":1<<28,"transcript_bytes":1<<30,"pending_events":4096}});
    let config = json!({"engine":"fixture","profile":"gomad.wasi-cooperative/v1","limits":{"fuel":1_u64<<50,"memory_bytes":4_u64<<30,"wall_millis":900_000,"output_bytes":256<<20,"evidence_bytes":704<<20},"wall_nanos":900_000_000_000_u64,"arguments":argv(selected,false),"environment":{"PWD":selected.directory,"TEMPORAL_ROOT":"/workspace","TMPDIR":"/tmp","TZ":"UTC","TEMPORAL_TEST_LOG_LEVEL":"warn","GOMAD_IO_PROFILE":"wasm","GOMAD_CHOICE_PROFILE":"gomad-choice-trace/v4"},"derive_seed_environment":true,"runtime_sha256":"full-raw-choice-runtime-identity","model":model,"namespace":{"entries":[{"path":b"/workspace/schema/sqlite","directory":true,"readonly":true,"data":[]}],"writable_directories":[b"/workspace".to_vec(),b"/tmp".to_vec(),selected.directory.as_bytes().to_vec()],"preopens":preopens,"working_directory":selected.directory.as_bytes(),"stdin":[],"limits":{"files":100_000,"descriptors":4096,"filesystem_bytes":512<<20,"output_bytes":256<<20},"epoch_nanos":946_684_800_000_000_000_u64},"storage":null});
    serde_json::from_value(json!({"schema":"fixture","manifest":manifest,"report":{"schema":"fixture","qualified":true,"deterministic":true,"target_success":true,"seed":7,"repeat":2,"command":["gomad"],"record_sha256":null,"first_divergence":null,"executions":[],"failure":null},"observations":{"operator_process_id":1,"resident_stores":1,"start":null,"prepared":null,"executed":null,"preparation_wall_nanos":1,"samples":[]},"cost":null,"cost_failure":null,"budget":null,"budget_failure":null,"integrated_source_manifest":null,"provenance":{"configuration":config,"inputs":{"kind":"go-test","source":selected.package,"tags":TAGS,"arguments":&argv(selected,false)[1..],"network":null,"soft_assertion_report_only":false}}})).unwrap()
}

#[test]
fn support_policy_rejects_changed_evidence_capacity() {
    let mut result = policy_fixture(SELECTIONS[0]);
    assert!(policy(&result, SELECTIONS[0], false).is_ok());
    result.provenance["configuration"]["limits"]["evidence_bytes"] = json!(1);
    assert!(policy(&result, SELECTIONS[0], false).is_err());
}

#[test]
fn support_policy_rejects_report_only_soft_assertions() {
    let mut result = policy_fixture(SELECTIONS[0]);
    result.provenance["inputs"]["soft_assertion_report_only"] = json!(true);
    assert!(policy(&result, SELECTIONS[0], false).is_err());
}

#[test]
fn preparation_gate_authenticates_receipt_instead_of_only_hash() {
    let root = std::env::temp_dir().join(format!("gomad-support-red-{}", std::process::id()));
    std::fs::create_dir_all(&root).unwrap();
    std::fs::set_permissions(&root, std::fs::Permissions::from_mode(0o700)).unwrap();
    let path = root.join("forged-receipt.json");
    write_new(&path, b"{}\n").unwrap();
    let mut gates = Vec::new();
    for name in GATE_ORDER {
        gates.push(Gate {
            cell: SELECTIONS[0].cell.into(),
            name: name.into(),
            command: vec!["gate".into()],
            exit: 0,
            log: path.clone(),
            log_sha256: canonical::digest(b"{}\n"),
            preparation_module_sha256: None,
            preparation_receipt: None,
            preparation_receipt_sha256: None,
            inspection_receipt: None,
            inspection_receipt_sha256: None,
        });
    }
    gates[0].preparation_module_sha256 = Some(canonical::digest(b"module"));
    gates[0].preparation_receipt = Some(path);
    gates[0].preparation_receipt_sha256 = Some(canonical::digest(b"{}\n"));
    assert!(validate_gates(&gates, SELECTIONS[0].cell, &canonical::digest(b"module")).is_err());
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn selected_test_requires_one_named_pass() {
    let function = "TestFrontendSystemInfo";
    for output in [
        "PASS\n",
        "testing: warning: no tests to run\nPASS\n",
        "--- SKIP: TestFrontendSystemInfo (0.00s)\n",
        "--- PASS: Wrapper (0.01s)\n",
        "--- PASS: TestFrontendSystemInfo (0.00s)\n--- PASS: TestFrontendSystemInfo (0.01s)\n",
        "--- PASS: TestFrontendSystemInfo (oops)\n",
    ] {
        assert!(
            named_pass(output.as_bytes(), function).is_err(),
            "accepted {output:?}"
        );
    }
    assert!(
        named_pass(
            b"    --- PASS: TestFrontendSystemInfo (0.01s)\nPASS\n",
            function
        )
        .is_ok()
    );
}

#[test]
fn candidate_refuses_a_stale_linked_reader() {
    let mut result = policy_fixture(SELECTIONS[0]);
    result.provenance["inputs"]["runtime_implementation"] = json!("a".repeat(64));
    let executable = FileBinding {
        path: PathBuf::from("/retained/gomad"),
        sha256: "a".repeat(64),
    };
    assert!(candidate(&result, &executable, None).is_err());
}

#[test]
fn policy_projection_preserves_raw_preimages_and_removes_only_declared_fields() {
    let success = policy_fixture(SELECTIONS[0]);
    let original = success.provenance.clone();
    let projected = policy(&success, SELECTIONS[0], false).unwrap();
    assert_eq!(success.provenance, original);
    let mut canary = success.clone();
    canary.manifest.cell = Some("canary".into());
    canary.provenance["configuration"]["arguments"] = json!(argv(SELECTIONS[0], true));
    canary.provenance["inputs"]["arguments"] = json!(&argv(SELECTIONS[0], true)[1..]);
    canary.provenance["configuration"]["runtime_sha256"] =
        json!("another-authenticated-build-key-derived-runtime");
    assert_ne!(
        digest(&success.provenance["configuration"]).unwrap(),
        digest(&canary.provenance["configuration"]).unwrap()
    );
    assert_eq!(projected, policy(&canary, SELECTIONS[0], true).unwrap());
}

#[test]
fn each_registry_row_preserves_its_configured_model_directory() {
    let smoke = policy_fixture(SELECTIONS[0]);
    let frontend = policy_fixture(SELECTIONS[1]);
    assert_ne!(
        smoke.provenance["configuration"]["model"],
        frontend.provenance["configuration"]["model"]
    );
    assert!(policy(&smoke, SELECTIONS[0], false).is_ok());
    assert!(policy(&frontend, SELECTIONS[1], false).is_ok());
    assert!(policy(&frontend, SELECTIONS[0], false).is_err());
    assert!(selection("onebox/sweep-promoted-test").is_err());
}

#[test]
fn support_policy_refuses_selection_order_list_flags_and_extra_flags() {
    let selected = SELECTIONS[0];
    let mut wrong_order = argv(selected, false);
    wrong_order.swap(1, 4);
    let mut list = argv(selected, false);
    list[4] = "-test.list=.".into();
    let mut extra = argv(selected, false);
    extra.push("-test.short".into());
    for args in [wrong_order, list, extra] {
        let mut result = policy_fixture(selected);
        result.provenance["configuration"]["arguments"] = json!(args);
        result.provenance["inputs"]["arguments"] = json!(&args[1..]);
        assert!(policy(&result, selected, false).is_err());
    }
}

#[test]
fn support_policy_refuses_changed_complete_policy_fields() {
    for pointer in [
        "/configuration/namespace/limits/files",
        "/configuration/namespace/epoch_nanos",
        "/configuration/model/limits/calls",
        "/configuration/model/read_step_nanos",
        "/configuration/limits/fuel",
        "/configuration/limits/memory_bytes",
        "/configuration/limits/output_bytes",
        "/configuration/limits/wall_millis",
        "/configuration/wall_nanos",
    ] {
        let mut result = policy_fixture(SELECTIONS[0]);
        *result.provenance.pointer_mut(pointer).unwrap() = json!(1);
        assert!(
            policy(&result, SELECTIONS[0], false).is_err(),
            "accepted {pointer}"
        );
    }
}

#[test]
fn empty_and_single_row_requests_cannot_publish_support() {
    for count in [0, 1, 2, 3] {
        let request = Request {
            executable: FileBinding {
                path: PathBuf::from("/retained/gomad"),
                sha256: "a".repeat(64),
            },
            results: vec![PathBuf::from("/retained/result.json"); count],
            process_receipts: vec![],
            gates: vec![],
            inventory: None,
        };
        assert!(build(&request).is_err());
    }
}

#[test]
fn support_paths_refuse_lexical_aliases() {
    for path in [
        "relative",
        "/root//result",
        "/root/./result",
        "/root/../result",
        "/root/result/",
    ] {
        assert!(absolute(Path::new(path)).is_err(), "accepted {path}");
    }
    assert!(absolute(Path::new("/root/result.json")).is_ok());
}

#[test]
fn support_output_is_exclusive_and_round_trips_private_bytes() {
    let root = std::env::temp_dir().join(format!("gomad-support-output-{}", std::process::id()));
    std::fs::create_dir(&root).unwrap();
    std::fs::set_permissions(&root, std::fs::Permissions::from_mode(0o700)).unwrap();
    let path = root.join("list.json");
    write_new(&path, b"original\n").unwrap();
    assert!(write_new(&path, b"replacement\n").is_err());
    assert_eq!(read(&path, 32).unwrap(), b"original\n");
    std::fs::remove_dir_all(root).unwrap();
}

fn compiler_inventory_fixture(compiler: &str, runtime: &str, cache: &str) -> inventory::Inventory {
    let identity = "c".repeat(64);
    let overlay_root = Path::new(cache).join("runtime-overlay").join(&identity);
    let mapping = json!({"Replace":{
        Path::new(compiler).join("src/runtime/proc.go").to_string_lossy():overlay_root.join("proc.go"),
        Path::new(compiler).join("src/runtime/gomad_semantics.go").to_string_lossy():overlay_root.join("gomad_semantics.go")
    }});
    let bytes = canonical::encode(&mapping).unwrap();
    let mut raw = BTreeMap::new();
    for (path, hash) in [
        (Path::new(compiler).join("VERSION"), "a".repeat(64)),
        (
            Path::new(compiler).join("pkg/tool/linux_arm64/compile"),
            "b".repeat(64),
        ),
        (
            Path::new(compiler).join("src/runtime/proc.go"),
            "d".repeat(64),
        ),
        (
            Path::new(runtime).join("gomad_semantics.go.txt"),
            "e".repeat(64),
        ),
        (
            Path::new(runtime).join("gomad_choicewire_generated.go.txt"),
            "f".repeat(64),
        ),
        (overlay_root.join("proc.go"), "1".repeat(64)),
        (overlay_root.join("gomad_semantics.go"), "2".repeat(64)),
        (
            PathBuf::from("/shared/source/toolchain/overlay.go"),
            "3".repeat(64),
        ),
        (overlay_root.join("overlay.json"), canonical::digest(&bytes)),
    ] {
        raw.insert(path, hash);
    }
    inventory::Inventory {
        compiler_path: Path::new(compiler).join("bin/go"),
        runtime_root: PathBuf::from(runtime),
        runtime_identity: identity,
        raw,
        overlay: inventory::Overlay {
            path: overlay_root.join("overlay.json"),
            sha256: canonical::digest(&bytes),
            bytes_base64: crate::campaign::exploration::base64(&bytes),
        },
    }
}

#[test]
fn compiler_alias_mapping_preserves_every_raw_inventory_and_descriptor() {
    let original =
        compiler_inventory_fixture("/original/go", "/original/runtime", "/original/cache");
    let copied =
        compiler_inventory_fixture("/private/compiler", "/private/runtime", "/private/cache");
    let raw_original = original.clone();
    let raw_copied = copied.clone();
    assert_ne!(original.raw, copied.raw);
    assert_ne!(original.overlay.sha256, copied.overlay.sha256);
    assert_eq!(
        inventory::view(&original).unwrap(),
        inventory::view(&copied).unwrap()
    );
    assert_eq!(original, raw_original);
    assert_eq!(copied, raw_copied);
}

#[test]
fn resealed_changed_compiler_runtime_replacement_and_inventory_members_are_refused() {
    let original =
        compiler_inventory_fixture("/original/go", "/original/runtime", "/original/cache");
    let baseline = inventory::view(&original).unwrap();
    for path in [
        Path::new("/original/go/pkg/tool/linux_arm64/compile").to_path_buf(),
        original.runtime_root.join("gomad_semantics.go.txt"),
        original.overlay.path.parent().unwrap().join("proc.go"),
    ] {
        let mut changed = original.clone();
        changed.raw.insert(path, "9".repeat(64));
        assert_ne!(inventory::view(&changed).unwrap(), baseline);
    }
    let mut extra = original.clone();
    extra
        .raw
        .insert(PathBuf::from("/shared/extra.go"), "9".repeat(64));
    assert_ne!(inventory::view(&extra).unwrap(), baseline);
    let mut missing = original.clone();
    missing
        .raw
        .remove(&PathBuf::from("/shared/source/toolchain/overlay.go"));
    assert_ne!(inventory::view(&missing).unwrap(), baseline);
    let mut missing_replacement = original.clone();
    missing_replacement
        .raw
        .remove(&original.overlay.path.parent().unwrap().join("proc.go"));
    assert!(inventory::view(&missing_replacement).is_err());
    let mut remapped = original;
    let mut descriptor: Value =
        spec::strict(&crate::qualify::cells::base64(&remapped.overlay.bytes_base64).unwrap())
            .unwrap();
    let replacement = descriptor["Replace"]
        .as_object_mut()
        .unwrap()
        .remove("/original/go/src/runtime/proc.go")
        .unwrap();
    descriptor["Replace"]["/original/go/src/runtime/different.go"] = replacement;
    let bytes = canonical::encode(&descriptor).unwrap();
    remapped.overlay.bytes_base64 = crate::campaign::exploration::base64(&bytes);
    remapped.overlay.sha256 = canonical::digest(&bytes);
    remapped.raw.insert(
        remapped.overlay.path.clone(),
        remapped.overlay.sha256.clone(),
    );
    assert_ne!(inventory::view(&remapped).unwrap(), baseline);
}

#[test]
fn retained_heap_requires_complete_bounded_gzip_payload() {
    let valid: Vec<u8> = vec![
        31, 139, 8, 0, 0, 0, 0, 0, 4, 255, 1, 4, 0, 251, 255, 104, 101, 97, 112, 225, 34, 41, 189,
        4, 0, 0, 0,
    ];
    gomad_wasmhost::linked_inspection::validate_heap_profile(&valid, 4 << 20).unwrap();
    let mut corrupt = valid.clone();
    let last = corrupt.len() - 8;
    corrupt[last] ^= 1;
    for invalid in [vec![31, 139, 8], corrupt, valid[..valid.len() - 1].to_vec()] {
        assert!(
            gomad_wasmhost::linked_inspection::validate_heap_profile(&invalid, 4 << 20).is_err()
        );
    }
    assert!(gomad_wasmhost::linked_inspection::validate_heap_profile(&valid, 1).is_err());
}

#[test]
fn original_descriptor_reconstruction_uses_retained_bytes_after_inputs_disappear() {
    let original = compiler_inventory_fixture(
        "/unavailable/original/go",
        "/unavailable/original/runtime",
        "/unavailable/original/cache",
    );
    let alias = compiler_inventory_fixture(
        "/unavailable/private/compiler",
        "/unavailable/private/runtime",
        "/unavailable/private/cache",
    );
    let mut provenance = json!({"inputs":{"compiler":original.raw,"compiler_path":original.compiler_path,"runtime":original.runtime_identity}});
    assert_eq!(
        inventory::from_alias(
            &provenance,
            &alias.overlay,
            Path::new("/unavailable/private/compiler")
        )
        .unwrap(),
        original
    );
    provenance["inputs"]["compiler"][original.overlay.path.to_str().unwrap()] =
        json!("9".repeat(64));
    assert!(
        inventory::from_alias(
            &provenance,
            &alias.overlay,
            Path::new("/unavailable/private/compiler")
        )
        .is_err()
    );
}

pub(super) fn source_candidate() -> Candidate {
    let mut result = policy_fixture(SELECTIONS[0]);
    let inventory =
        compiler_inventory_fixture("/original/go", "/original/runtime", "/original/cache");
    result.manifest.host_sha256 = env!("GOMAD_HOST_IMPLEMENTATION_SHA256").into();
    result.manifest.helper_sha256 = gomad_wasmhost::linked::IMPLEMENTATION_SHA256.into();
    result.manifest.engine_sha256 = canonical::digest(
        gomad_wasmhost::linked::Conditions::compiled()
            .configuration()
            .as_bytes(),
    );
    result.manifest.host_os = std::env::consts::OS.into();
    result.manifest.host_arch = std::env::consts::ARCH.into();
    result.manifest.runtime_sha256 = digest(&inventory.raw).unwrap();
    result.provenance["inputs"]["compiler"] = json!(inventory.raw);
    result.provenance["inputs"]["compiler_path"] = json!(inventory.compiler_path);
    result.provenance["inputs"]["runtime"] = json!(inventory.runtime_identity);
    result.provenance["inputs"]["runtime_implementation"] =
        json!(env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"));
    result.provenance["inputs"]["directory"] = json!("/source");
    let manifest = format!(
        "{}  /source/schema/sqlite/schema.sql\n{}  /source/tools/gomad/internal/backends/wasm/testdata/prototype_temporal/inspection_test.go\n{}  /source/tools/gomad/internal/backends/wasm/testdata/prototype_temporal/prototype_test.go\n",
        canonical::digest(b"schema"),
        canonical::digest(b"inspection source"),
        canonical::digest(b"prototype source")
    );
    result.manifest.integrated_source_sha256 = canonical::digest(manifest.as_bytes());
    result.integrated_source_manifest = Some(manifest);
    candidate(
        &result,
        &FileBinding {
            path: PathBuf::from("/retained/gomad"),
            sha256: "a".repeat(64),
        },
        Some(&inventory),
    )
    .unwrap()
}

#[test]
fn schema_capture_must_match_complete_frozen_source_bytes() {
    let candidate = source_candidate();
    let configured: Configuration =
        serde_json::from_value(policy_fixture(SELECTIONS[0]).provenance["configuration"].clone())
            .unwrap();
    let mut namespace = configured.namespace.unwrap();
    assert!(source::schema(&namespace, &candidate).is_err());
    namespace
        .entries
        .push(gomad_wasmhost::linked::NamespaceEntry {
            path: b"/workspace/schema/sqlite/schema.sql".to_vec(),
            directory: false,
            readonly: true,
            data: b"schema".to_vec(),
        });
    for path in [
        "/workspace",
        "/workspace/schema",
        "/tmp",
        "/workspace/tests",
    ] {
        namespace
            .entries
            .push(gomad_wasmhost::linked::NamespaceEntry {
                path: path.as_bytes().to_vec(),
                directory: true,
                readonly: path == "/workspace/schema",
                data: vec![],
            });
    }
    source::schema(&namespace, &candidate).unwrap();
    namespace
        .entries
        .iter_mut()
        .find(|entry| !entry.directory)
        .unwrap()
        .data = b"changed schema".to_vec();
    assert!(source::schema(&namespace, &candidate).is_err());
}

#[test]
fn inspection_stage_requires_exact_unchanged_original_go_files() {
    let candidate = source_candidate();
    let staged = Path::new("/source/service/history/.inspection");
    let mut provenance = json!({"inputs":{"sources":{
        "/source/service/history/.inspection/inspection_test.go":canonical::digest(b"inspection source"),
        "/source/service/history/.inspection/prototype_test.go":canonical::digest(b"prototype source")
    }}});
    source::inspection_staged(&provenance, staged, &candidate).unwrap();
    provenance["inputs"]["sources"]["/source/service/history/.inspection/extra_test.go"] =
        json!(canonical::digest(b"extra"));
    assert!(source::inspection_staged(&provenance, staged, &candidate).is_err());
}

fn specification_fixture() -> (Value, QualificationResult) {
    let mut result = policy_fixture(SELECTIONS[0]);
    result.manifest.id = "unchanged-workflow-type-smoke".into();
    result.manifest.workload = "onebox".into();
    result.manifest.integrated_source_sha256 = "a".repeat(64);
    result.manifest.integrated_source_manifest_path = Some("/retained/source.sha256".into());
    result.provenance["inputs"]["directory"] = json!("/source");
    let spec = json!({"Manifest":{"Provisional":false,"ID":result.manifest.id,"Workload":"onebox","Cell":"success","IntegratedSourceSHA256":format!("sha256:{}",result.manifest.integrated_source_sha256),"IntegratedSourceManifestPath":"/retained/source.sha256","Seed":7,"Repeat":2},"ArtifactRoot":"/retained/results","Campaign":{"Target":{"Backend":"wasm","Kind":"go-test","Source":"./tests","WorkingDir":"/source","Args":&argv(SELECTIONS[0],false)[1..],"BuildTags":TAGS},"ExecutionTimeout":900_000_000_000_u64,"OverallTimeout":1_800_000_000_000_u64,"TerminateGrace":1_000_000_000,"OnFailure":"all","FailureBudget":1,"OutputLimit":256<<20,"WorldTransitionLimit":32<<20,"ChoiceTraceLimit":64<<20,"SuccessBytesLimit":2_u64<<30,"Environment":["PWD=/workspace/tests","TEMPORAL_ROOT=/workspace","TMPDIR=/tmp","TZ=UTC","TEMPORAL_TEST_LOG_LEVEL=warn"],"Artifacts":"/retained/results/artifacts"}});
    (spec, result)
}

#[test]
fn required_raw_specification_refuses_changed_unrecorded_campaign_bounds() {
    let (value, result) = specification_fixture();
    let witness = |value: &Value| {
        let bytes = encoded(value).unwrap();
        SpecWitness {
            path: PathBuf::from("/retained/smoke.spec.json"),
            sha256: canonical::digest(&bytes),
            bytes_base64: crate::campaign::exploration::base64(&bytes),
        }
    };
    specification(&witness(&value), &result).unwrap();
    for field in [
        "WorldTransitionLimit",
        "ChoiceTraceLimit",
        "SuccessBytesLimit",
        "OverallTimeout",
        "TerminateGrace",
        "OutputLimit",
        "ExecutionTimeout",
        "FailureBudget",
    ] {
        let mut changed = value.clone();
        changed["Campaign"][field] = json!(0);
        assert!(
            specification(&witness(&changed), &result).is_err(),
            "accepted resealed {field}"
        );
    }
}

#[test]
fn retained_specification_authenticates_raw_bytes_selection_and_source() {
    let (value, result) = specification_fixture();
    let witness = |value: &Value| {
        let bytes = encoded(value).unwrap();
        SpecWitness {
            path: PathBuf::from("/retained/spec.json"),
            sha256: canonical::digest(&bytes),
            bytes_base64: crate::campaign::exploration::base64(&bytes),
        }
    };
    let mut forged = witness(&value);
    forged.sha256 = "b".repeat(64);
    assert!(specification(&forged, &result).is_err());
    for (pointer, replacement) in [
        (
            "/Manifest/IntegratedSourceSHA256",
            json!(format!("sha256:{}", "b".repeat(64))),
        ),
        (
            "/Manifest/IntegratedSourceManifestPath",
            json!("/other/source.sha256"),
        ),
        ("/Manifest/Cell", json!("canary")),
        ("/Campaign/Target/Args", json!(["-test.list=."])),
        ("/Campaign/Target/WorkingDir", json!("/other/source")),
        ("/Campaign/Environment", json!([])),
    ] {
        let mut changed = value.clone();
        *changed.pointer_mut(pointer).unwrap() = replacement;
        assert!(
            specification(&witness(&changed), &result).is_err(),
            "accepted {pointer}"
        );
    }
}

/// Original `TestHostMemoryPublicationErrorRetainsValidationStagePair`: a
/// Result that fails validation publishes nothing and returns no path. The
/// Go-only stage-observer pair has no Rust counterpart.
#[test]
fn invalid_result_publication_fabricates_nothing() {
    let mut result = policy_fixture(SELECTIONS[0]);
    result.manifest.fuel = 0;
    assert!(result::validate(&result).is_err());
    let root = std::env::temp_dir().join(format!(
        "gomad-invalid-result-publication-{}",
        std::process::id()
    ));
    let _ = std::fs::remove_dir_all(&root);
    std::fs::create_dir(&root).unwrap();
    std::fs::set_permissions(&root, std::fs::Permissions::from_mode(0o700)).unwrap();
    assert!(result::publish(&root, &result).is_err());
    assert_eq!(
        std::fs::read_dir(&root).unwrap().count(),
        0,
        "publication fabricated"
    );
    std::fs::remove_dir(&root).unwrap();
}

fn matched_fixture(cell: &str) -> QualificationResult {
    let mut result = policy_fixture(SELECTIONS[0]);
    let m = &mut result.manifest;
    m.schema = result::MANIFEST_SCHEMA.into();
    m.workload = "onebox".into();
    m.cell = Some(cell.into());
    m.integrated_source_manifest_path = Some("/frozen/source.manifest".into());
    for digest in [
        &mut m.integrated_source_sha256,
        &mut m.workload_source_sha256,
        &mut m.runtime_sha256,
        &mut m.compiler_sha256,
        &mut m.helper_sha256,
        &mut m.host_sha256,
        &mut m.model_sha256,
        &mut m.engine_sha256,
        &mut m.host_config_sha256,
        &mut m.module_sha256,
        &mut m.toolchain_sha256,
    ] {
        *digest = canonical::digest(b"fixture");
    }
    result
}

/// Go `Matched` and the support pair check: a canary only stands beside the
/// success row it was measured against.
#[test]
fn support_success_and_canary_must_be_matched_cells() {
    let success = matched_fixture("success");
    let policy = json!({"policy": "fixture"});
    let pair = |canary: &QualificationResult, policy_b: &Value, bytes: u64| {
        paired((&success, &policy, 9), (canary, policy_b, bytes))
    };
    assert_eq!(pair(&matched_fixture("canary"), &policy, 9), Ok(()));
    type Change = fn(&mut Manifest);
    let unmatched: [Change; 9] = [
        |m| m.workload = "sqlite".into(),
        |m| m.integrated_source_sha256 = canonical::digest(b"other"),
        |m| m.workload_source_sha256 = canonical::digest(b"other"),
        |m| m.host_os = "darwin".into(),
        |m| m.host_arch = "amd64".into(),
        |m| m.seed = 8,
        |m| m.repeat = 3,
        |m| m.fuel = 1,
        |m| m.memory_bytes = 1,
    ];
    for change in unmatched {
        let mut canary = matched_fixture("canary");
        change(&mut canary.manifest);
        assert_eq!(
            pair(&canary, &policy, 9),
            Err("WASM comparison cells are not matched".into())
        );
    }
    let mut canary = matched_fixture("canary");
    canary.manifest.module_sha256 = canonical::digest(b"other");
    let differ = Err("support success/canary policy, source or module differ".into());
    assert_eq!(pair(&canary, &policy, 9), differ);
    let canary = matched_fixture("canary");
    assert_eq!(pair(&canary, &policy, 10), differ);
    assert_eq!(pair(&canary, &json!({"policy": "other"}), 9), differ);
    let mut canary = matched_fixture("canary");
    canary.provenance["inputs"]["tags"] = json!(["other"]);
    assert_eq!(pair(&canary, &policy, 9), differ);
}
