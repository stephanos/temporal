use super::*;

#[test]
fn terminal_preparation_preserves_original_backend_tuple_without_execution_arguments() {
    let recipe = terminal_exit(Path::new("/frozen/checkout")).unwrap();
    assert_eq!(recipe.module.kind, "go-test");
    assert_eq!(recipe.module.source, "./tests");
    assert_eq!(recipe.module.directory, Path::new("/frozen/checkout"));
    assert_eq!(recipe.module.build_tags, ORIGINAL_TEMPORAL_TAGS);
    assert!(recipe.module.target_arguments.is_empty());
    assert_eq!(recipe.memory_bytes, 4 << 30);
    assert_eq!(recipe.fuel, 1 << 50);
    let config = recipe.configuration;
    assert!(config["Args"].is_null());
    assert_eq!(config["Environment"], json!(["PWD=/workspace"]));
    assert_eq!(config["WorkingDirectory"], "/workspace");
    assert_eq!(config["WritableDirectories"], json!(["/workspace", "/tmp"]));
    assert_eq!(config["EntropyKey"], json!(vec![0_u8; 32]));
    assert_eq!(
        config["Clock"],
        json!({"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":0})
    );
    assert_eq!(
        config["Limits"],
        json!({"OutputBytes":1<<20,"TranscriptBytes":8<<20,"Calls":10_000,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096})
    );
    assert!(config.get("network_v2").is_none());
}

#[test]
fn original_temporal_compile_selects_full_three_tags_without_run_selection() {
    let full = temporal_test(Path::new("/repo"), "./tests");
    assert_eq!(full.kind, "go-test");
    assert_eq!(full.build_tags, ORIGINAL_TEMPORAL_TAGS);
    assert!(full.target_arguments.is_empty());
    assert!(!full.build_tags.iter().any(|tag| tag == "gomad_wasm_slim"));
    let derived = derived_temporal_test(Path::new("/external/source-module"), "./.tmp/derived");
    assert_eq!(derived.directory, Path::new("/external/source-module"));
    assert_eq!(derived.source, "./.tmp/derived");
    assert_eq!(
        derived.build_tags,
        ["test_dep", "disable_grpc_modules", "sqlite3_dotlk"]
    );
}

#[test]
fn runtime_and_probe_compile_preserve_the_original_mapped_fixture_and_no_tags() {
    let runtime = runtime_cost(Path::new("/repo"));
    assert_eq!(runtime.kind, "go-run");
    assert_eq!(runtime.directory, Path::new("/repo/tools/gomad"));
    assert_eq!(
        runtime.source,
        "./internal/model/wasi/testdata/runtime_cost"
    );
    assert!(runtime.build_tags.is_empty());
    assert!(runtime.target_arguments.is_empty());
    let probe = diagnostic_probe(Path::new("/repo"));
    assert_eq!(probe.kind, "go-run");
    assert_eq!(probe.directory, runtime.directory);
    assert_eq!(
        probe.source,
        "./internal/backends/wasm/toolchain/testdata/choices"
    );
    assert!(probe.build_tags.is_empty());
    assert!(probe.target_arguments.is_empty());
}

#[test]
fn runtime_cost_request_has_original_profile_bounds_and_captured_arguments() {
    let request = runtime_request().unwrap();
    assert_eq!(request.memory_bytes, 256 << 20);
    assert_eq!(request.fuel, 1 << 40);
    assert_eq!(request.timeout, Duration::from_secs(180));
    assert_eq!(request.choice_bytes, 64 << 20);
    assert!(!request.diagnostics);
    assert!(!request.digest_transcript);
    assert_eq!(
        request.configuration["Args"],
        json!(["guest", "literal $arg"])
    );
    assert_eq!(
        request.configuration["Environment"],
        json!(["PWD=/workspace", "TZ=UTC", "ONLY=captured"])
    );
    assert_eq!(
        request.configuration["Limits"],
        json!({"OutputBytes":1<<20,"TranscriptBytes":64<<20,"Calls":1<<20,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096})
    );
    assert_eq!(request.configuration["EntropyKey"][0], 1);
    assert_eq!(request.configuration["EntropyKey"][1], 2);
    assert_eq!(request.configuration["EntropyKey"][2], 3);
    assert_eq!(request.configuration["Clock"]["ReadStepNanos"], 0);
}

#[test]
fn temporal_request_retains_full_prototype_limits_independently_of_module_preparation() {
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    let request = temporal_request(
        repo,
        "tests",
        &["-test.run=^Exact$".into()],
        Duration::from_secs(1800),
    )
    .unwrap();
    assert_eq!(request.memory_bytes, 4 << 30);
    assert_eq!(request.fuel, 1 << 50);
    assert_eq!(request.timeout, Duration::from_secs(1800));
    assert_eq!(request.choice_bytes, 1 << 30);
    assert!(!request.diagnostics);
    assert!(request.digest_transcript);
    assert_eq!(
        request.configuration["Args"],
        json!([
            "temporal.test",
            "-test.v",
            "-test.count=1",
            "-test.timeout=72h",
            "-test.run=^Exact$"
        ])
    );
    assert_eq!(
        request.configuration["WorkingDirectory"],
        "/workspace/tests"
    );
    assert_eq!(
        request.configuration["Limits"],
        json!({"OutputBytes":256<<20,"TranscriptBytes":1<<30,"Calls":1<<28,"Descriptors":4096,"Files":100_000,"FilesystemBytes":512<<20,"PendingEvents":4096})
    );
    assert!(request.configuration.get("network_v2").is_none());
    let admitted = crate::install::config::convert(request.configuration, COOPERATIVE).unwrap();
    assert_eq!(admitted.arguments.last().unwrap(), "-test.run=^Exact$");
    let preparation = terminal_exit(repo).unwrap();
    assert!(preparation.module.target_arguments.is_empty());
    assert!(preparation.configuration["Args"].is_null());
    assert_eq!(preparation.configuration["Limits"]["OutputBytes"], 1 << 20);
}

#[test]
fn original_configurations_pass_the_real_native_configuration_admission() {
    for configuration in [
        terminal_exit(Path::new("/repo")).unwrap().configuration,
        runtime_request().unwrap().configuration,
    ] {
        crate::install::config::convert(configuration, COOPERATIVE).unwrap();
    }
    assert!(
        temporal_request(
            Path::new("/repo"),
            "/absolute",
            &[],
            Duration::from_secs(1800)
        )
        .is_err()
    );
    assert!(
        temporal_request(
            Path::new("/repo"),
            "../escape",
            &[],
            Duration::from_secs(1800)
        )
        .is_err()
    );
}

#[test]
fn host_owner_alone_preserves_its_original_explicit_slim_build_profile() {
    let recipe = slim_host_owner(Path::new("/repo"));
    assert_eq!(recipe.kind, "go-test");
    assert_eq!(recipe.source, "./tests");
    assert_eq!(
        recipe.build_tags,
        [
            "disable_grpc_modules",
            "gomad_wasm_slim",
            "sqlite3_dotlk",
            "test_dep"
        ]
    );
    assert!(recipe.target_arguments.is_empty());
    assert!(
        !temporal_test(Path::new("/repo"), "./tests")
            .build_tags
            .iter()
            .any(|tag| tag == "gomad_wasm_slim")
    );
}

#[test]
fn diagnostic_probe_preserves_unmodified_cooperative_request_defaults() {
    let request = diagnostic_probe_request().unwrap();
    assert_eq!(request.memory_bytes, 256 << 20);
    assert_eq!(request.fuel, 4_000_000_000);
    assert_eq!(request.timeout, Duration::from_secs(60));
    assert_eq!(request.choice_bytes, 1 << 20);
    assert!(request.diagnostics);
    assert!(!request.digest_transcript);
    assert_eq!(request.configuration["Clock"]["ReadStepNanos"], 1000);
    assert_eq!(
        request.configuration["Limits"],
        json!({"OutputBytes":1<<20,"TranscriptBytes":8<<20,"Calls":10_000,"Descriptors":64,"Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096})
    );
    assert_eq!(
        request.configuration["Args"],
        json!(["guest", "literal $arg"])
    );
    crate::install::config::convert(request.configuration, COOPERATIVE).unwrap();
}
