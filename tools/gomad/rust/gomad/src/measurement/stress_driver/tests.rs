use super::*;

fn call() -> Invocation {
    Invocation {
        name: "stress".into(),
        repo: "/candidate".into(),
        go_command: "go".into(),
        inputs: BTreeMap::from([(
            "GOMAD_WASM_STRESS_HOST_DESCRIPTION".into(),
            "quiet test host".into(),
        )]),
        environ: Vec::new(),
        keep: false,
    }
}

#[test]
fn catalogue_defaults_and_original_controls_are_admitted_before_dispatch() {
    let mut call = call();
    let controls = Controls::parse(&call, 8).unwrap();
    assert_eq!(
        (
            controls.repetitions,
            controls.parallel,
            controls.cpu_workers
        ),
        (3, 2, 7)
    );
    for (key, values) in [
        (
            "GOMAD_WASM_STRESS_REPETITIONS",
            vec!["invalid", "0", "1", "257"],
        ),
        ("GOMAD_WASM_STRESS_PARALLEL", vec!["NaN", "0", "1", "257"]),
        ("GOMAD_WASM_STRESS_CPU_WORKERS", vec!["NaN", "0", "257"]),
        ("GOMAD_WASM_STRESS_ONLY", vec!["other"]),
        ("GOMAD_WASM_STRESS_AXIS", vec!["other"]),
        ("GOMAD_WASM_STRESS_SEED", vec!["-1", "18446744073709551616"]),
    ] {
        for value in values {
            call.inputs.insert(key.into(), value.into());
            assert!(Controls::parse(&call, 8).is_err(), "{key}={value}");
        }
        call.inputs.remove(key);
    }
    call.inputs
        .insert("GOMAD_WASM_STRESS_ONLY".into(), "runtime".into());
    call.inputs
        .insert("GOMAD_WASM_STRESS_SEED".into(), "1000003".into());
    call.inputs
        .insert("GOMAD_WASM_STRESS_AXIS".into(), "reference".into());
    let controls = Controls::parse(&call, 8).unwrap();
    assert!(controls.selected("runtime", 1000003));
    assert!(!controls.selected("runtime", 0));
    assert!(!controls.selected("workflow", 1000003));
}

#[test]
fn stress_runtime_recipe_is_the_original_probe_and_bounds_not_runtime_cost() {
    let config = configuration(Path::new("/unused"), "runtime").unwrap();
    assert_eq!(
        config["Limits"],
        json!({"OutputBytes":1<<20,"TranscriptBytes":64<<20,"Calls":10000,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096})
    );
    assert_eq!(
        config["Environment"],
        json!([
            "PWD=/workspace",
            "TZ=UTC",
            "ONLY=captured",
            "EXTRA_ENTROPY=no_"
        ])
    );
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    let arguments = arguments(
        repo,
        Path::new("/retained"),
        "runtime",
        "./internal/backends/wasm/testdata/runtime_probes",
    );
    let options = crate::cli::explore::parse(&arguments, &mut false).unwrap();
    assert_eq!(options.target_kind, "go-run");
    assert_eq!(options.target_args, Vec::<String>::new());
    assert_eq!(options.execution_timeout, Duration::from_secs(60));
    assert_eq!(options.overall_timeout, Duration::from_secs(3660));
    assert_eq!(options.choice_bytes, 1 << 20);
    assert_eq!(options.output_limit, 1 << 20);
    assert!(options.choices && options.diagnostics);
    assert_eq!(options.environment, ["ONLY=captured", "EXTRA_ENTROPY=no_"]);
    assert!(!arguments.iter().any(|arg| arg.contains("runtime_cost")));
}

#[test]
fn all_temporal_recipes_retain_original_selection_tags_and_fifteen_minute_bounds() {
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    for (name, source, selection, output) in [
        (
            "workflow",
            "./tests",
            "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
            256 << 20,
        ),
        ("activity", "./tests", "^TestActivityTestSuite$", 256 << 20),
        (
            "two-cluster",
            "./.tmp/wasm-backend-temporal/temporal_broker-original",
            "^TestIndependentClusterReplicationRecovery$",
            8 << 20,
        ),
    ] {
        let options = crate::cli::explore::parse(
            &arguments(repo, Path::new("/retained"), name, source),
            &mut false,
        )
        .unwrap();
        assert_eq!(options.target_kind, "go-test");
        assert_eq!(options.source, source);
        assert_eq!(
            options.target_args,
            [
                "-test.v",
                "-test.count=1",
                if name == "two-cluster" {
                    "-test.timeout=1h"
                } else {
                    "-test.timeout=72h"
                },
                &format!("-test.run={selection}"),
                "-persistenceType=sql",
                "-persistenceDriver=sqlite"
            ]
        );
        assert_eq!(
            options.build_tags,
            [
                "disable_grpc_modules",
                "gomad_wasm_slim",
                "sqlite3_dotlk",
                "test_dep"
            ]
        );
        assert_eq!(options.execution_timeout, Duration::from_secs(900));
        assert_eq!(options.overall_timeout, Duration::from_secs(4500));
        assert_eq!(options.choice_bytes, 32 << 20);
        assert_eq!(options.output_limit, output);
    }
}

#[test]
fn malformed_controls_fail_inside_the_parent_owned_stage_and_preserve_requested_rerun_root() {
    let root = std::env::temp_dir().join(format!("gomad-stress-catalogue-{}", std::process::id()));
    let mut call = call();
    call.repo = root.clone();
    call.inputs.insert(
        "GOMAD_WASM_STRESS_ROOT".into(),
        root.to_string_lossy().into_owned(),
    );
    call.inputs
        .insert("GOMAD_WASM_STRESS_REPETITIONS".into(), "invalid".into());
    let proof = super::super::inputs::fixture_checkout(root.clone());
    let mut stderr = Vec::new();
    assert_eq!(
        super::super::dispatch(call, proof, &mut Vec::new(), &mut stderr),
        2
    );
    assert!(
        String::from_utf8(stderr)
            .unwrap()
            .contains("--stress-repetitions must be 1..256")
    );
    let runs = fs::read_dir(root.join(".gomad-measure/stress"))
        .unwrap()
        .filter_map(Result::ok)
        .filter(|entry| entry.file_name().to_string_lossy().starts_with("run-"))
        .collect::<Vec<_>>();
    assert_eq!(runs.len(), 1);
    let rerun: Value =
        serde_json::from_slice(&fs::read(runs[0].path().join("rerun.json")).unwrap()).unwrap();
    let argv = rerun["command"].as_array().unwrap();
    let at = argv.iter().position(|arg| arg == "--stress-root").unwrap();
    assert_eq!(argv[at + 1], root.to_string_lossy().as_ref());
    assert!(!runs[0].path().join(".complete").exists());
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn retained_control_bounds_cannot_bypass_the_catalogue_parser() {
    let controls = Controls::parse(&call(), 8).unwrap();
    for (repetitions, parallel, cpu_workers) in
        [(257, 2, 1), (2, 257, 1), (2, 2, 0), (usize::MAX, 2, 1)]
    {
        let malformed = Controls {
            repetitions,
            parallel,
            cpu_workers,
            ..controls.clone()
        };
        assert!(malformed.validate().is_err());
    }
}

#[test]
fn excluded_seed_keeps_all_cells_unexecuted_and_preserves_the_requested_root() {
    let repo = std::env::temp_dir().join(format!("gomad-stress-excluded-{}", std::process::id()));
    fs::create_dir(&repo).unwrap();
    fs::write(repo.join("go.mod"), b"module stress-selection-fixture\n").unwrap();
    for arguments in [vec!["init", "-q"], vec!["add", "go.mod"]] {
        assert!(
            std::process::Command::new("git")
                .arg("-C")
                .arg(&repo)
                .args(arguments)
                .status()
                .unwrap()
                .success()
        );
    }
    let root = repo.join("stress output");
    let mut call = call();
    call.repo = repo.clone();
    call.inputs.extend(BTreeMap::from([
        (
            "GOMAD_WASM_STRESS_ROOT".into(),
            root.to_string_lossy().into_owned(),
        ),
        ("GOMAD_WASM_STRESS_ONLY".into(), "runtime".into()),
        ("GOMAD_WASM_STRESS_SEED".into(), "1".into()),
        ("GOMAD_WASM_TEST_HELPER".into(), "/unused-helper".into()),
        ("GOMAD_STOCK_GO".into(), "/unused-compiler".into()),
    ]));
    let proof = super::super::inputs::fixture_checkout(repo.clone());
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    assert_eq!(
        super::super::dispatch(call, proof, &mut stdout, &mut stderr),
        0,
        "{}",
        String::from_utf8_lossy(&stderr)
    );
    let output: Value = serde_json::from_slice(&stdout).unwrap();
    let report = value(Path::new(output["result"].as_str().unwrap())).unwrap();
    assert_eq!(report["complete"], false);
    assert_eq!(report["executed"], false);
    assert_eq!(report["cells"].as_array().unwrap().len(), 164);
    assert_eq!(report["requested_root"], root.to_string_lossy().as_ref());
    assert!(
        report["cells"]
            .as_array()
            .unwrap()
            .iter()
            .all(|cell| cell["status"] == "skipped" && cell["completed"] == 0)
    );
    assert!(!root.join("stress-producer.json").exists());
    fs::remove_dir_all(repo).unwrap();
}

#[test]
fn actual_retained_bounds_and_cluster_topology_are_checked_apart_from_installation() {
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    for name in ["runtime", "two-cluster"] {
        let root = std::env::temp_dir().join(format!(
            "gomad-stress-retained-{name}-{}",
            std::process::id()
        ));
        fs::create_dir(&root).unwrap();
        let config_path = root.join("config.json");
        write(
            &config_path,
            &canonical::encode(&configuration(repo, name).unwrap()).unwrap(),
        )
        .unwrap();
        let configured = crate::install::config::build(PROFILE, Some(&config_path), None).unwrap();
        let mut model = configured.model.clone();
        model["mode"] = json!(1);
        model["limit"] = json!(if name == "runtime" { 1 << 20 } else { 32 << 20 });
        model["diagnostics"] = json!(true);
        model.as_object_mut().unwrap().remove("seed");
        let base = crate::replay::preflight::Configuration {
            engine: linked::Conditions::compiled().configuration(),
            profile: PROFILE.into(),
            limits: crate::evidence::record::Limits {
                fuel: if name == "runtime" {
                    4_000_000_000
                } else {
                    1 << 50
                },
                memory_bytes: if name == "runtime" {
                    256 << 20
                } else {
                    4 << 30
                },
                wall_millis: if name == "runtime" { 60000 } else { 900000 },
                output_bytes: if name == "runtime" { 1 << 20 } else { 8 << 20 },
                evidence_bytes: 128 << 20,
            },
            world_transition_bytes: None,
            wall_nanos: Some(if name == "runtime" {
                60_000_000_000
            } else {
                900_000_000_000
            }),
            arguments: configured.arguments,
            environment: configured.environment,
            derive_seed_environment: true,
            network: configured.network.clone(),
            runtime_sha256: Some("a".repeat(64)),
            model: Some(model),
            namespace: Some(configured.namespace),
            storage: configured.storage,
        };
        let value = if name == "runtime" {
            serde_json::to_value(base).unwrap()
        } else {
            let identity = linked::ChoiceIdentity {
                target_sha256: [0; 32],
                toolchain_build_key: "00".repeat(32),
                goos: "wasip1".into(),
                goarch: "wasm".into(),
                implementation_sha256: [0; 32],
            };
            let mut bound = crate::campaign::scenario::bind(
                &cluster_driver::scenario_bytes().unwrap(),
                &base,
                b"\0asm\x01\0\0\0\x05\x03\x01\x00\x01",
                &identity,
                configured.network.as_ref(),
                7,
            )
            .unwrap();
            crate::stress::cluster::bind(&mut bound).unwrap();
            serde_json::to_value(bound.configuration).unwrap()
        };
        crate::stress::admit_retained_configuration(name, value.clone()).unwrap();
        let mut changed = value.clone();
        if name == "runtime" {
            changed["limits"]["fuel"] = json!(1);
        } else {
            changed["incarnations"][0]["limits"]["fuel"] = json!(1);
        }
        assert!(crate::stress::admit_retained_configuration(name, changed).is_err());
        let mut changed = value.clone();
        if name == "runtime" {
            changed["model"]["limits"]["transcript_bytes"] = json!(1);
        } else {
            changed["network"]["faults"][0]["at_nanos"] = json!(2);
        }
        assert!(crate::stress::admit_retained_configuration(name, changed).is_err());
        fs::remove_dir_all(root).unwrap();
    }
}

#[test]
fn retained_workflow_and_activity_cannot_broaden_the_original_test_selection() {
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    for (name, selection) in [
        (
            "workflow",
            "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
        ),
        ("activity", "^TestActivityTestSuite$"),
    ] {
        let root =
            std::env::temp_dir().join(format!("gomad-stress-argv-{name}-{}", std::process::id()));
        fs::create_dir(&root).unwrap();
        let config_path = root.join("config.json");
        write(
            &config_path,
            &canonical::encode(&configuration(repo, name).unwrap()).unwrap(),
        )
        .unwrap();
        let configured = crate::install::config::build(PROFILE, Some(&config_path), None).unwrap();
        let mut model = configured.model;
        model["mode"] = json!(1);
        model["limit"] = json!(32 << 20);
        model["diagnostics"] = json!(true);
        model.as_object_mut().unwrap().remove("seed");
        let original = json!({"engine":linked::Conditions::compiled().configuration(),"profile":PROFILE,
            "limits":{"fuel":1_u64<<50,"memory_bytes":4_u64<<30,"wall_millis":900000,"output_bytes":256<<20,"evidence_bytes":1<<30},
            "wall_nanos":900_000_000_000_u64,"arguments":["gomad-target","-test.v","-test.count=1","-test.timeout=72h",format!("-test.run={selection}"),"-persistenceType=sql","-persistenceDriver=sqlite"],
            "environment":configured.environment,"runtime_sha256":"a".repeat(64),"model":model,"namespace":configured.namespace,"storage":configured.storage,"network":configured.network});
        crate::stress::admit_retained_configuration(name, original.clone()).unwrap();
        let mut changed = original;
        changed["arguments"][4] = json!("-test.run=.");
        assert!(
            crate::stress::admit_retained_configuration(name, changed)
                .unwrap_err()
                .contains("original guest selection")
        );
        fs::remove_dir_all(root).unwrap();
    }
}
