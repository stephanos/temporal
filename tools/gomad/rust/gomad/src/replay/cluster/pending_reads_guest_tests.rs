use super::*;

fn polling_producer() -> Producer {
    require_release_guest_role();
    let root = PathBuf::from(
        std::env::var_os("GOMAD_TEST_BROAD_GUEST_ROOT").expect("explicit retained root"),
    )
    .join("pending-reads");
    assert!(root.is_absolute());
    fs::create_dir(&root).unwrap();
    let compiler = PathBuf::from(std::env::var_os("GOMAD_STOCK_GO").expect("stock Go 1.27.1"));
    assert!(compiler.is_absolute());
    let sources = vec![SourceSeal {
        path: "tools/gomad/internal/backends/wasm/cluster/testdata/pending_reads/main.go",
        sha256: "5a475d2ae7fec8a3417dd75e8d06dd397974975d50b9432455c00e1ddfa38da6",
    }];
    verify_sources(&sources);
    let config = root.join("original-config.json");
    fs::write(&config, canonical::encode(&original_config()).unwrap()).unwrap();
    let installation = root.join("installation.json");
    let arguments = vec![
        "--compiler".into(),
        compiler.to_string_lossy().into_owned(),
        "--runtime-root".into(),
        checkout()
            .join("tools/gomad/internal/model/runtimesource")
            .to_string_lossy()
            .into_owned(),
        "--profile=gomad.wasi-cooperative/v1".into(),
        "--memory-bytes=268435456".into(),
        "--fuel=4000000000".into(),
        "--config".into(),
        config.to_string_lossy().into_owned(),
        "--cache".into(),
        root.join("preparation").to_string_lossy().into_owned(),
        "--output".into(),
        installation.to_string_lossy().into_owned(),
    ];
    crate::cli::install::run(&arguments, &mut Vec::new())
        .unwrap_or_else(|error| panic!("{}", error.message));
    let mut options = crate::cli::explore::parse(
        &[
            "--choices".into(),
            "--diagnostics".into(),
            "go-run".into(),
            "./internal/backends/wasm/cluster/testdata/pending_reads".into(),
        ],
        &mut false,
    )
    .unwrap();
    options.installation = Some(installation);
    options.working_dir = checkout().join("tools/gomad");
    options.world_transition_limit = 1 << 20;
    options.output_limit = 1 << 20;
    options.execution_timeout = Duration::from_secs(60);
    let prepared = prepare::prepare(&options).unwrap();
    verify_sources(&sources);
    assert_eq!(prepared.configuration.limits.memory_bytes, 256 << 20);
    assert_eq!(prepared.configuration.limits.fuel, 4_000_000_000);
    assert_eq!(
        prepared.configuration.model.as_ref().unwrap()["limit"],
        8 << 20
    );
    retain_preparation(&root, &prepared);
    Producer {
        root,
        prepared,
        sources,
    }
}

#[test]
#[ignore = "actual pinned Go compilation and live guests require the parent execution lease"]
fn actual_pending_reads_complete_and_replay_under_one_mib() {
    let producer = polling_producer();
    let scenario = scenario(
        json!([{"id":"a","boot":"fixture","address":"10.0.0.1"}]),
        json!([{"id":"fixture","args":[]}]),
        json!([{"action":"start","handle":{"node":"a","incarnation":1}},
               {"action":"wait","handle":{"node":"a","incarnation":1}}]),
        json!([]),
    );
    let configuration = bind(&producer, &scenario, 7);
    assert_eq!(configuration.broker.transition_bytes, Some(1 << 20));
    let (execution, proof) = observe_execution(&producer, &configuration);
    publication_diagnostic(&producer, "pending", "execution", &proof);
    fs::write(producer.root.join("broker-world.bin"), &execution.world).unwrap();
    assert_eq!(
        execution.outcome.termination, "exit",
        "{:?}",
        execution.outcome
    );
    assert_eq!(execution.incarnations.len(), 1);
    let guest = &execution.incarnations[0].execution;
    assert_eq!(guest.outcome.exit_code, Some(0), "{:?}", guest.outcome);
    let stdout = std::str::from_utf8(&guest.evidence["stdout"]).unwrap();
    assert_eq!(
        stdout,
        "pending reads survived 48 scans at 48ms\ncompleted 32 reads in order\n"
    );
    helper::validate_world(&execution.world, &configuration.broker).unwrap();
    assert_broker_accounting(&execution.world);
    let (report, replay) = publish_replay(&producer, "pending/replay", &configuration, true);
    assert_no_oracle_success(&configuration, &report, &replay);
    assert_eq!(
        replay["retained_inputs"]["configuration"]["broker"]["transition_bytes"],
        1 << 20
    );
}

fn assert_broker_accounting(world: &[u8]) {
    assert_eq!(&world[..12], b"GMWSNAP1\0\0\0\x01");
    let number = |offset| u64::from_be_bytes(world[offset..offset + 8].try_into().unwrap());
    let calls = number(72) - 1;
    let transitions = number(88) - 1;
    assert!(calls > 48);
    assert_eq!(number(80) - 1, calls);
    assert_eq!(
        transitions,
        calls * 3,
        "every visible call retains register/ready/quiesce accounting"
    );
}
