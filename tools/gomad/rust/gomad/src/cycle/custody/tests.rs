use super::*;
use std::{fs, os::unix::fs::symlink, path::PathBuf};
fn root() -> PathBuf {
    let path = PathBuf::from(format!(
        "/home/agent/wt/evidence/fn242-b01-custody-inputs/file-test-{}",
        format!(
            "{}-{}",
            std::process::id(),
            super::COMMAND_SEQUENCE.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        )
    ));
    fs::create_dir_all(&path).unwrap();
    path
}
fn unit_proof(id: &str, variant: &str, input: Value, error: &str) {
    let bytes = serde_json::to_vec(&input).unwrap();
    eprintln!(
        "CUSTODY_PROOF {}",
        serde_json::json!({"id":id,"variant":variant,"input":input,"input_sha256":digest(&bytes),"catching_error":error,"control":"read_file(file,8) returns original / check_files valid map returns Ok","original_source_sha256":"153c0a969c57bfe983c2b22412c56c88618afc1eb5b00ed9e795ea6fa64c78de","current_source_sha256":digest(include_bytes!("mod.rs")),"current_test_sha256":digest(include_bytes!("tests.rs"))})
    );
}
#[test]
fn immutable_file_control_and_refusals() {
    let root = root();
    let file = root.join("input");
    fs::write(&file, b"original").unwrap();
    assert_eq!(read_file(&file, 8).unwrap(), b"original");
    let error = read_file(&file, 7).unwrap_err();
    assert!(error.contains("bound"));
    unit_proof(
        "orchestration:driver:76",
        "bounded file read",
        serde_json::json!({"bytes":"original","valid_bound":8,"mutant_bound":7}),
        &error,
    );
    let link = root.join("link");
    symlink(&file, &link).unwrap();
    for (variant, path) in [
        ("symlink", link.as_path()),
        ("directory", root.as_path()),
        ("relative", Path::new("input")),
    ] {
        let error = read_file(path, 8).unwrap_err();
        unit_proof(
            "orchestration:driver:76",
            variant,
            serde_json::json!({"valid":file,"mutant":path,"bytes":"original"}),
            &error,
        );
        assert_eq!(read_file(&file, 8).unwrap(), b"original");
    }
    let ancestor = root.join("alias");
    symlink(&root, &ancestor).unwrap();
    assert!(read_file(&ancestor.join("input"), 8).is_err());
    fs::remove_file(ancestor).unwrap();
    fs::remove_file(link).unwrap();
    fs::remove_file(file).unwrap();
    fs::remove_dir(root).unwrap();
}
#[test]
fn strict_json_and_frozen_files_control_and_refusals() {
    assert_eq!(strict_json(br#"{"n":1}"#).unwrap()["n"], 1);
    for input in [
        br#"{"n":1,"n":2}"#.as_slice(),
        b"NaN",
        b"Infinity",
        b"{}{}",
        b"{",
    ] {
        assert!(strict_json(input).is_err());
    }
    let root = root();
    let file = root.join("frozen");
    fs::write(&file, b"original").unwrap();
    let mut files = BTreeMap::from([(file.to_str().unwrap().to_owned(), digest(b"original"))]);
    check_files(&files).unwrap();
    let error = check_files(&BTreeMap::new()).unwrap_err();
    assert!(error.contains("empty"));
    for id in ["orchestration:driver:45", "orchestration:driver:73"] {
        unit_proof(
            id,
            "empty frozen input set",
            serde_json::json!({"valid":files,"mutant":{}}),
            &error,
        );
    }
    check_files(&files).unwrap();
    fs::write(&file, b"changed!").unwrap();
    let error = check_files(&files).unwrap_err();
    assert!(error.contains("changed"));
    unit_proof(
        "orchestration:driver:77",
        "changed bytes",
        serde_json::json!({"path":file,"expected_bytes":"original","mutant_bytes":"changed!","sha256":files}),
        &error,
    );
    fs::write(&file, b"original").unwrap();
    check_files(&files).unwrap();
    files.insert(file.to_str().unwrap().to_owned(), "X".repeat(64));
    assert!(check_files(&files).unwrap_err().contains("digest"));
    fs::remove_file(file).unwrap();
    fs::remove_dir(root).unwrap();
}

pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    use crate::cycle::{
        model::{Mode, PlanConfig},
        plan, process,
    };
    use std::{collections::BTreeMap, sync::atomic::AtomicBool, time::Duration};
    let root =
        PathBuf::from(std::env::var("GOMAD_SOURCE_CASE_ROOT").unwrap_or_else(|_| {
            "/home/agent/wt/evidence/fn242-b01-custody-inputs/source-run1".into()
        }));
    fs::create_dir(&root).map_err(|e| e.to_string())?;
    let repo = root.join("repo");
    fs::create_dir(&repo).map_err(|e| e.to_string())?;
    let context = "cycle-source-proof";
    let cancelled = AtomicBool::new(false);
    let mut runner = process::Runner::create(owner, &root.join("registry"), Some(context.into()))?;
    git(&mut runner, context, &repo, &["init", "-q"], &cancelled)?;
    git(
        &mut runner,
        context,
        &repo,
        &["config", "user.name", "Bounded fixture"],
        &cancelled,
    )?;
    git(
        &mut runner,
        context,
        &repo,
        &["config", "user.email", "fixture@example.invalid"],
        &cancelled,
    )?;
    git(
        &mut runner,
        context,
        &repo,
        &["config", "gc.auto", "0"],
        &cancelled,
    )?;
    git(
        &mut runner,
        context,
        &repo,
        &["config", "maintenance.auto", "false"],
        &cancelled,
    )?;
    let file = repo.join("source");
    fs::write(&file, b"original source\n").map_err(|e| e.to_string())?;
    let tracked_scratch = repo.join("common/gomaddebug/testdata/inspection-tracked");
    fs::create_dir_all(tracked_scratch.parent().unwrap()).map_err(|e| e.to_string())?;
    fs::write(&tracked_scratch, b"tracked source").map_err(|e| e.to_string())?;
    let ignored_flow = repo.join(".flow/ignored");
    let reference = repo.join(".flow/reference");
    fs::create_dir(reference.parent().unwrap()).map_err(|e| e.to_string())?;
    fs::write(&reference, b"frozen reference").map_err(|e| e.to_string())?;
    fs::write(&ignored_flow, b"Flow state exemption").map_err(|e| e.to_string())?;
    symlink("source", repo.join("source-link")).map_err(|e| e.to_string())?;
    git(&mut runner, context, &repo, &["add", "-A"], &cancelled)?;
    git(
        &mut runner,
        context,
        &repo,
        &["commit", "-qm", "bounded source fixture"],
        &cancelled,
    )?;
    let gitlink = repo.join("gitlink");
    fs::create_dir(&gitlink).map_err(|e| e.to_string())?;
    let initial_head = String::from_utf8(git(
        &mut runner,
        context,
        &repo,
        &["rev-parse", "HEAD"],
        &cancelled,
    )?)
    .map_err(|e| e.to_string())?;
    let gitlink_entry = format!("160000,{},gitlink", initial_head.trim());
    git(
        &mut runner,
        context,
        &repo,
        &["update-index", "--add", "--cacheinfo", &gitlink_entry],
        &cancelled,
    )?;
    git(
        &mut runner,
        context,
        &repo,
        &["commit", "-qm", "bounded gitlink fixture"],
        &cancelled,
    )?;
    let snapshot = Snapshot::observe(&repo, &mut runner, &cancelled, context)?;
    assert!(!snapshot.files.contains_key(gitlink.to_str().unwrap()));
    fs::remove_dir(&gitlink).map_err(|e| e.to_string())?;
    fs::write(&gitlink, b"substituted gitlink").map_err(|e| e.to_string())?;
    let gitlink_error = Snapshot::observe(&repo, &mut runner, &cancelled, context).unwrap_err();
    assert!(gitlink_error.contains("gitlink is not a directory"));
    fs::remove_file(&gitlink).map_err(|e| e.to_string())?;
    symlink("source", &gitlink).map_err(|e| e.to_string())?;
    let gitlink_symlink_error =
        Snapshot::observe(&repo, &mut runner, &cancelled, context).unwrap_err();
    assert!(gitlink_symlink_error.contains("gitlink is not a directory"));
    fs::remove_file(&gitlink).map_err(|e| e.to_string())?;
    fs::create_dir(&gitlink).map_err(|e| e.to_string())?;
    snapshot.verify(&repo, &mut runner, &cancelled, context)?;
    eprintln!(
        "SOURCE_GITLINK_CONTROL directory=accepted regular-substitution=refused symlink-substitution=refused restored=accepted error={gitlink_error} symlink_error={gitlink_symlink_error}"
    );
    let manifest = root.join("source.manifest");
    fs::write(&manifest, snapshot.manifest()).map_err(|e| e.to_string())?;
    let config_path = root.join("original-config");
    fs::write(&config_path, br#"{"_freezing":true,"fixture":"bounded"}"#)
        .map_err(|e| e.to_string())?;
    let mut files = snapshot.files.clone();
    files.insert(reference.display().to_string(), digest(b"frozen reference"));
    files.insert(manifest.display().to_string(), digest(&snapshot.manifest()));
    let mut frozen = Frozen {
        repo: repo.clone(),
        config_path: config_path.clone(),
        config_sha256: digest(&read_file(&config_path, 1024)?),
        files,
        head: snapshot.head.clone(),
        symlinks: snapshot.symlinks.clone(),
        source_manifest: manifest.clone(),
        source_sha256: digest(&snapshot.manifest()),
        required: vec![reference.clone()],
    };
    let mut proofs = Vec::new();
    let mut emit = |id: &str, variant: &str, error: &str, input: Value| {
        let bytes = serde_json::to_vec(&input).unwrap();
        let row = serde_json::json!({"id":id,"variant":variant,"original_source_sha256":"153c0a969c57bfe983c2b22412c56c88618afc1eb5b00ed9e795ea6fa64c78de","current_source_sha256":digest(include_bytes!("mod.rs")),"current_test_sha256":digest(include_bytes!("tests.rs")),"control":"normal Frozen::guard returns Ok on same fixture before/after mutation","catching_error":error,"input":input,"input_sha256":digest(&bytes),"boundary":"authenticated config / current Git / immutable files / current typed Runner intervals"});
        eprintln!("CUSTODY_PROOF {}", row);
        proofs.push(row);
    };
    frozen.guard(&mut runner, &cancelled, context)?;
    fs::remove_file(&ignored_flow).map_err(|e| e.to_string())?;
    frozen.guard(&mut runner, &cancelled, context)?;
    fs::write(&ignored_flow, b"Flow state exemption").map_err(|e| e.to_string())?;
    frozen.guard(&mut runner, &cancelled, context)?;
    let config = read_file(&config_path, 1024)?;
    fs::write(&config_path, b"changed original config").map_err(|e| e.to_string())?;
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("driver config changed"));
    emit(
        "orchestration:driver:451",
        "changed bytes even with freezing JSON",
        &error,
        serde_json::json!({"original":String::from_utf8_lossy(&config),"mutant":"changed original config"}),
    );
    fs::write(&config_path, &config).map_err(|e| e.to_string())?;
    frozen.guard(&mut runner, &cancelled, context)?;
    let old = frozen.head.clone();
    frozen.head = "0".repeat(40);
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("frozen HEAD changed"));
    emit(
        "orchestration:driver:455",
        "wrong HEAD",
        &error,
        serde_json::json!({"expected":frozen.head,"actual":old}),
    );
    frozen.head = old;
    frozen.guard(&mut runner, &cancelled, context)?;
    for (id, path, message) in [
        (
            "orchestration:driver:459",
            file.clone(),
            "incomplete non-Flow source inventory",
        ),
        (
            "orchestration:driver:464",
            reference.clone(),
            "unsealed driver/reference input",
        ),
        (
            "orchestration:driver:484",
            manifest.clone(),
            "frozen source manifest changed or unsealed",
        ),
    ] {
        let expected = frozen.files.remove(path.to_str().unwrap()).unwrap();
        let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
        assert!(error.contains(message), "{id}: {error}");
        emit(
            id,
            "missing frozen binding",
            &error,
            serde_json::json!({"removed":path,"digest":expected}),
        );
        frozen.files.insert(path.display().to_string(), expected);
        frozen.guard(&mut runner, &cancelled, context)?;
    }
    let saved = frozen.source_sha256.clone();
    frozen.source_sha256 = "0".repeat(64);
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("frozen source manifest changed or unsealed"));
    emit(
        "orchestration:driver:484",
        "wrong manifest digest",
        &error,
        serde_json::json!({"manifest":manifest,"original":saved,"mutant":frozen.source_sha256}),
    );
    frozen.source_sha256 = saved;
    frozen.guard(&mut runner, &cancelled, context)?;
    let link = repo.join("source-link").display().to_string();
    frozen.symlinks.insert(link.clone(), "foreign".into());
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("frozen symlink changed"));
    emit(
        "orchestration:driver:466",
        "wrong literal",
        &error,
        serde_json::json!({"path":link,"original":"source","mutant":"foreign"}),
    );
    frozen.symlinks = snapshot.symlinks.clone();
    frozen.guard(&mut runner, &cancelled, context)?;
    frozen.symlinks.clear();
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("symlink inventory changed"));
    emit(
        "orchestration:driver:468",
        "missing symlink inventory",
        &error,
        serde_json::json!({"original":snapshot.symlinks,"mutant":{}}),
    );
    frozen.symlinks = snapshot.symlinks.clone();
    frozen.guard(&mut runner, &cancelled, context)?;
    let untracked = repo.join("common/gomaddebug/testdata/inspection-owned/proof");
    let plan = plan::derive(
        &PlanConfig {
            binary: "/bin/sh".into(),
            config_path: config_path.clone(),
            config_sha256: frozen.config_sha256.clone(),
        },
        Mode::Parallel,
    )?;
    let stage = plan
        .stages
        .iter()
        .find(|s| s.contract.label == "ordinary")
        .unwrap();
    let command=process::Command{argv:vec!["/bin/sh".into(),"-c".into(),"mkdir -p common/gomaddebug/testdata/inspection-owned; printf child > common/gomaddebug/testdata/inspection-owned/proof; exec /bin/sleep 60".into()],cwd:repo.clone(),environment:process::Environment::Exact(BTreeMap::from([("PATH".into(),"/usr/bin:/bin".into())])),declared_environment:BTreeMap::new(),temporary_paths:BTreeMap::new(),accepted:vec![0],timeout:Duration::from_secs(90),private_context:Some(context.into())};
    let mut running = runner
        .start_stage(stage, 1, command, &cancelled, |_| Ok(()))
        .map_err(|e| e.to_string())?;
    let started = std::time::Instant::now();
    while !untracked.exists() {
        if started.elapsed() > Duration::from_secs(5) {
            return Err("actual owned scratch child not ready".into());
        }
        std::thread::park_timeout(Duration::from_millis(5));
    }
    frozen.guard(&mut runner, &cancelled, context)?;
    fs::write(&tracked_scratch, b"changed tracked source").map_err(|e| e.to_string())?;
    let old = frozen
        .files
        .insert(
            tracked_scratch.display().to_string(),
            digest(b"changed tracked source"),
        )
        .unwrap();
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("dirty frozen source"));
    emit(
        "orchestration:driver:482",
        "tracked change under active matching scratch",
        &error,
        serde_json::json!({"path":tracked_scratch,"mutant":"changed tracked source","active_pid":running.pid(),"scratch":stage.contract.scratch}),
    );
    fs::write(&tracked_scratch, b"tracked source").map_err(|e| e.to_string())?;
    frozen
        .files
        .insert(tracked_scratch.display().to_string(), old);
    frozen.guard(&mut runner, &cancelled, context)?;
    running.cancel();
    let completed = running.wait(&cancelled)?;
    assert!(completed.cleanup_complete);
    assert!(!PathBuf::from(format!("/proc/{}", completed.pid)).exists());
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("dirty frozen source"));
    emit(
        "orchestration:driver:482",
        "ended interval grants no exception",
        &error,
        serde_json::json!({"path":untracked,"completed_pid":completed.pid,"active_intervals":runner.active_intervals()?.len()}),
    );
    fs::write(
        root.join("registry/forged.json"),
        br#"{"active":true,"scratch":["common/gomaddebug/testdata/inspection-*"]}"#,
    )
    .map_err(|e| e.to_string())?;
    let error = frozen.guard(&mut runner, &cancelled, context).unwrap_err();
    assert!(error.contains("dirty frozen source"));
    emit(
        "orchestration:driver:482",
        "JSON cannot grant interval authority",
        &error,
        serde_json::json!({"forged_registry":true,"active_intervals":runner.active_intervals()?.len()}),
    );
    fs::remove_file(untracked).map_err(|e| e.to_string())?;
    fs::remove_dir(repo.join("common/gomaddebug/testdata/inspection-owned"))
        .map_err(|e| e.to_string())?;
    frozen.guard(&mut runner, &cancelled, context)?;
    fs::write(
        root.join("proofs.json"),
        serde_json::to_vec_pretty(&proofs).map_err(|e| e.to_string())?,
    )
    .map_err(|e| e.to_string())?;
    eprintln!("CUSTODY_OWNER_CASES {}", proofs.len());
    Ok(())
}
