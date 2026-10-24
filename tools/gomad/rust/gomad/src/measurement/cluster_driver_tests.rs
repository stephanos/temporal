use super::super::evidence::{NativeExecution, NativeRun};
use super::*;
use std::collections::BTreeMap;

fn committed() -> NativeRun {
    let mut incarnations = Vec::new();
    for generation in [1, 2] {
        for node in ["active", "standby"] {
            let observation = json!({"cluster_id":node,"history":"aGk=","event_ids":[1,2],"types":["started","completed"]});
            let mut streams = BTreeMap::from([(
                "stdout".into(),
                format!("GOMAD_TEMPORAL_OBSERVATION {observation}\n").into_bytes(),
            )]);
            if generation == 1 {
                streams.insert("storage_image".into(), b"retained-image".to_vec());
            }
            incarnations.push(NativeExecution { handle:Some(json!({"node":node,"incarnation":generation})), outcome:json!({"termination":if generation == 1 {"crash"} else {"exit"},"exit_code":if generation == 1 {Value::Null} else {json!(0)}}), streams });
        }
    }
    NativeRun {
        request: Value::Null,
        report: Value::Null,
        profile: Value::Null,
        root: NativeExecution {
            handle: None,
            outcome: json!({"termination":"exit","exit_code":0}),
            streams: BTreeMap::new(),
        },
        incarnations,
        executed_steps: None,
        cutoff: None,
        modules: Vec::new(),
    }
}
fn change(run: &mut NativeRun, index: usize, field: &str, value: Value) {
    let stdout = run.incarnations[index].streams.get_mut("stdout").unwrap();
    let mut observation = crate::stress::cluster::observation(stdout).unwrap();
    observation[field] = value;
    *stdout = format!("GOMAD_TEMPORAL_OBSERVATION {observation}\n").into_bytes();
}
#[test]
fn cluster_acceptance_restores_independent_convergence_and_durable_recovery_predicates() {
    success(&committed()).unwrap();
    for kind in 0..9 {
        let mut run = committed();
        match kind {
            0 => {
                run.incarnations[0].streams.remove("storage_image");
            }
            1 => run.incarnations[0].outcome["termination"] = json!("exit"),
            2 => change(&mut run, 1, "cluster_id", json!("active")),
            3 => change(&mut run, 1, "history", json!("Ynll")),
            4 => change(&mut run, 2, "history", json!("Ynll")),
            5 => change(&mut run, 2, "event_ids", json!([1, 3])),
            6 => change(&mut run, 2, "types", json!(["started", "changed"])),
            7 => change(&mut run, 3, "cluster_id", json!("changed")),
            8 => {
                run.incarnations[2].streams.insert(
                    "stdout".into(),
                    b"GOMAD_TEMPORAL_OBSERVATION invalid\n".to_vec(),
                );
            }
            _ => unreachable!(),
        }
        assert!(success(&run).is_err(), "mutation {kind}");
    }
}

fn staging_snapshot(root: &Path) -> BTreeMap<PathBuf, Vec<u8>> {
    let mut entries = BTreeMap::new();
    let mut pending = vec![root.to_path_buf()];
    while let Some(directory) = pending.pop() {
        for entry in fs::read_dir(&directory).unwrap() {
            let path = entry.unwrap().path();
            let metadata = fs::symlink_metadata(&path).unwrap();
            if metadata.is_dir() {
                pending.push(path.clone());
                entries.insert(path, Vec::new());
            } else {
                entries.insert(path.clone(), fs::read(&path).unwrap());
            }
        }
    }
    entries
}

/// Original `TestTemporalFixtureStagingRejectsLinks`: a linked fixture
/// source or staging destination, at its root or through a parent, is refused
/// and nothing outside the trusted checkout changes.
#[test]
fn broker_fixture_staging_refuses_links_without_touching_outside_bytes() {
    use std::os::unix::fs::symlink;
    let sources = [
        "tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go",
        "tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go",
    ];
    let scratch = Path::new(env!("CARGO_MANIFEST_DIR")).join("target/staging-link-tests");
    let checkout = |root: &Path| {
        for (index, source) in sources.iter().enumerate() {
            let path = root.join(source);
            fs::create_dir_all(path.parent().unwrap()).unwrap();
            fs::write(path, format!("package fixture // {index}\n")).unwrap();
        }
    };
    let clean = crate::campaign::prepare::process::Temporary::new(&scratch).unwrap();
    checkout(&clean.0);
    let staged = stage_broker(&clean.0).unwrap();
    assert_eq!(stage_broker(&clean.0).unwrap(), staged);
    let staged = staged.trim_start_matches("./").to_owned();
    for (index, name) in ["recovery_test.go", "replication_test.go"]
        .iter()
        .enumerate()
    {
        assert_eq!(
            fs::read(clean.0.join(&staged).join(name)).unwrap(),
            format!("package fixture // {index}\n").into_bytes()
        );
    }
    for case in [
        "source-root",
        "source-parent",
        "destination-root",
        "destination-parent",
        "destination-anchor",
    ] {
        let repo = crate::campaign::prepare::process::Temporary::new(&scratch).unwrap();
        let outside = crate::campaign::prepare::process::Temporary::new(&scratch).unwrap();
        checkout(&repo.0);
        checkout(&outside.0);
        fs::create_dir_all(outside.0.join("staged")).unwrap();
        match case {
            "source-root" => {
                let source = repo.0.join(sources[0]);
                fs::remove_file(&source).unwrap();
                symlink(outside.0.join(sources[0]), source).unwrap();
            }
            "source-parent" => {
                let parent = repo.0.join(Path::new(sources[1]).parent().unwrap());
                fs::remove_dir_all(&parent).unwrap();
                symlink(
                    outside.0.join(Path::new(sources[1]).parent().unwrap()),
                    parent,
                )
                .unwrap();
            }
            "destination-root" => {
                let destination = repo.0.join(&staged);
                fs::create_dir_all(destination.parent().unwrap()).unwrap();
                symlink(outside.0.join("staged"), destination).unwrap();
            }
            "destination-parent" => {
                fs::create_dir_all(repo.0.join(".tmp")).unwrap();
                symlink(
                    outside.0.join("staged"),
                    repo.0.join(".tmp/wasm-backend-temporal"),
                )
                .unwrap();
            }
            "destination-anchor" => symlink(outside.0.join("staged"), repo.0.join(".tmp")).unwrap(),
            _ => unreachable!(),
        }
        let before = staging_snapshot(&outside.0);
        assert!(
            stage_broker(&repo.0).is_err(),
            "{case}: linked fixture path staged"
        );
        assert_eq!(
            staging_snapshot(&outside.0),
            before,
            "{case}: staging changed bytes or entries outside the trusted checkout"
        );
    }
}
