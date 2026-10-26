use super::{Options, changed, selection::Graph};
use std::{
    collections::BTreeSet,
    ffi::OsString,
    fs,
    path::{Path, PathBuf},
    process::Command,
};

type TestResult = Result<(), Box<dyn std::error::Error>>;

fn metadata(root: &str, packages: &[(&str, &str, &[&str])], members: &[&str]) -> Vec<u8> {
    serde_json::to_vec(&serde_json::json!({
        "packages": packages.iter().map(|(id, name, _)| serde_json::json!({
            "id": id, "name": name, "manifest_path": format!("/{root}/{name}/Cargo.toml"), "source": null
        })).collect::<Vec<_>>(),
        "workspace_members": members,
        "resolve": {"nodes": packages.iter().map(|(id, _, dependencies)| serde_json::json!({"id": id, "dependencies": dependencies})).collect::<Vec<_>>()}
    })).unwrap_or_default()
}

fn graph() -> Result<Graph, Box<dyn std::error::Error>> {
    let mut graph = Graph::default();
    graph.add(
        Path::new("/framework"),
        &metadata(
            "framework",
            &[("core1", "core", &[]), ("fixture1", "fixtures", &["core1"])],
            &["core1", "fixture1"],
        ),
    )?;
    let mut system: serde_json::Value = serde_json::from_slice(&metadata(
        "system",
        &[("core2", "core", &[]), ("activity2", "activity", &["core2"]), ("other2", "other", &[])],
        &["activity2", "other2"],
    ))?;
    system["packages"][0]["manifest_path"] = serde_json::json!("/framework/core/Cargo.toml");
    graph.add(Path::new("/system"), &serde_json::to_vec(&system)?)?;
    Ok(graph)
}

fn options() -> Options {
    Options {
        base: Some("HEAD".into()),
        always: BTreeSet::from(["fixtures".into()]),
        ..Options::default()
    }
}

fn selected(
    graph: &Graph,
    options: &Options,
    paths: &[&str],
) -> Result<BTreeSet<String>, Box<dyn std::error::Error>> {
    let selection = graph.select(options, paths.iter().map(PathBuf::from).collect())?;
    Ok(selection.workspaces.into_iter().flat_map(|workspace| workspace.packages).collect())
}

#[test]
fn reverse_dependencies_merge_workspace_specific_ids_by_manifest() -> TestResult {
    assert_eq!(
        selected(&graph()?, &options(), &["/framework/core/src/lib.rs"])?,
        BTreeSet::from(["core".into(), "fixtures".into(), "activity".into()])
    );
    Ok(())
}

#[test]
fn deleted_source_ownership_does_not_require_the_file_to_exist() -> TestResult {
    assert_eq!(
        selected(&graph()?, &options(), &["/system/other/src/deleted.rs"])?,
        BTreeSet::from(["fixtures".into(), "other".into()])
    );
    Ok(())
}

#[test]
fn global_and_unmapped_inputs_select_whole_workspace_and_reverse_dependents() -> TestResult {
    let graph = graph()?;
    for path in [
        "/framework/Cargo.toml",
        "/framework/Cargo.lock",
        "/framework/.cargo/config.toml",
        "/framework/scripts/check.sh",
        "/framework/unknown/file",
        "/framework/core/build.rs",
        "/framework/dev/xtask/src/main.rs",
        "/framework/dev/guest-driver/src/lib.rs",
        "/framework/core/rustfmt.toml",
    ] {
        assert_eq!(
            selected(&graph, &options(), &[path])?,
            BTreeSet::from(["core".into(), "fixtures".into(), "activity".into()]),
            "{path}"
        );
    }
    assert_eq!(
        selected(&graph, &options(), &["/Makefile"])?,
        BTreeSet::from(["core".into(), "fixtures".into(), "activity".into(), "other".into()])
    );
    Ok(())
}

#[test]
fn mapped_external_sources_and_unchanged_invocations_keep_explicit_execution() -> TestResult {
    let mut options = options();
    options.mappings.push(("/system/feature".into(), "activity".into()));
    assert_eq!(
        selected(&graph()?, &options, &["/system/feature/activity/model.rs"])?,
        BTreeSet::from(["fixtures".into(), "activity".into()])
    );
    assert_eq!(selected(&graph()?, &options, &[])?, BTreeSet::from(["fixtures".into()]));
    options.always.insert("missing".into());
    assert!(graph()?.select(&options, vec![]).is_err());
    Ok(())
}

fn git(root: &Path, arguments: &[&str]) -> TestResult {
    let output = Command::new("git").current_dir(root).args(arguments).output()?;
    if !output.status.success() {
        return Err(format!("git failed: {}", String::from_utf8_lossy(&output.stderr)).into());
    }
    Ok(())
}

#[test]
fn git_selection_includes_staged_unstaged_untracked_renamed_and_deleted_paths() -> TestResult {
    let root = tempfile::tempdir()?;
    git(root.path(), &["init", "--quiet"])?;
    for name in ["old.rs", "deleted.rs", "staged.rs", "unstaged.rs"] {
        fs::write(root.path().join(name), "before\n")?;
    }
    git(root.path(), &["add", "."])?;
    git(
        root.path(),
        &[
            "-c",
            "user.name=Quick",
            "-c",
            "user.email=quick@example.test",
            "commit",
            "--quiet",
            "-m",
            "base",
        ],
    )?;
    git(root.path(), &["mv", "old.rs", "new.rs"])?;
    fs::remove_file(root.path().join("deleted.rs"))?;
    fs::write(root.path().join("staged.rs"), "staged\n")?;
    git(root.path(), &["add", "staged.rs"])?;
    fs::write(root.path().join("unstaged.rs"), "unstaged\n")?;
    fs::write(root.path().join("untracked.rs"), "untracked\n")?;
    let paths = changed(root.path(), &OsString::from("HEAD"))?;
    let expected = ["old.rs", "new.rs", "deleted.rs", "staged.rs", "unstaged.rs", "untracked.rs"]
        .map(|name| root.path().join(name))
        .into_iter()
        .collect::<BTreeSet<_>>();
    assert_eq!(paths.into_iter().collect::<BTreeSet<_>>(), expected);
    assert!(changed(root.path(), &OsString::from("missing-base")).is_err());
    assert!(changed(root.path(), &OsString::from("--help")).is_err());
    Ok(())
}

#[test]
fn options_reject_unknown_flags_and_relative_workspaces() {
    assert!(Options::parse(vec!["--workspace".into(), ".".into()]).is_err());
    assert!(Options::parse(vec!["--unknown".into(), "value".into()]).is_err());
    assert!(Options::parse(vec!["--base".into()]).is_err());
}

#[test]
fn package_filters_only_narrow_unchanged_or_exclusively_mapped_feature_edits() {
    let mut options = options();
    options.mappings.push(("/system/feature".into(), "activity".into()));
    assert!(super::focused(&options, "activity", &[]));
    assert!(super::focused(&options, "activity", &["/system/feature/activity/model.rs".into()]));
    for path in [
        "/system/activity/src/lib.rs",
        "/system/feature/activity/tests.rs",
        "/system/feature/activity/proof_checks.rs",
        "/system/feature/activity/compatibility/model.rs",
        "/system/feature/activity/api.rs",
        "/framework/core/src/lib.rs",
        "/system/feature/Cargo.toml",
        "/system/unknown",
    ] {
        assert!(!super::focused(&options, "activity", &[path.into()]), "{path}");
    }
    assert!(!super::focused(
        &options,
        "activity",
        &["/system/feature/activity/model.rs".into(), "/system/feature/activity/tests.rs".into(),]
    ));
    assert!(!super::focused(&options, "other", &["/system/feature/activity/model.rs".into()]));
}

fn assert_stage_receipt(log: &[u8], phase: &str, exit: &str) -> TestResult {
    let log = std::str::from_utf8(log)?;
    let receipt = log.lines().last().ok_or("missing stage receipt")?;
    assert!(
        receipt.starts_with(&format!(
            "quick: stage={phase} workspace=/selected/workspace packages=selected-package elapsed="
        )),
        "{receipt}"
    );
    let elapsed = receipt
        .split(" elapsed=")
        .nth(1)
        .ok_or("missing elapsed time")?
        .split('s')
        .next()
        .ok_or("missing seconds unit")?
        .parse::<f64>()?;
    assert!(elapsed.is_finite() && elapsed >= 0.0, "{receipt}");
    assert!(receipt.contains(&format!(" exit={exit}")), "{receipt}");
    Ok(())
}

#[test]
fn stage_receipts_label_formatting_and_preserve_success_and_failure_statuses() -> TestResult {
    for (program, phase, expected) in
        [("/bin/true", "fmt", 0), ("/bin/false", "clippy", 1), ("/bin/false", "test", 1)]
    {
        let mut log = Vec::new();
        let actual = super::execute(
            phase,
            Path::new("/selected/workspace"),
            &["selected-package".into()],
            Command::new(program),
            &mut log,
        )?;
        assert_eq!(actual, expected);
        assert_stage_receipt(&log, phase, &expected.to_string())?;
    }
    Ok(())
}

#[test]
fn stage_receipt_reports_elapsed_and_spawn_failure_without_success_status() -> TestResult {
    let root = tempfile::tempdir()?;
    let mut log = Vec::new();
    let result = super::execute(
        "test",
        Path::new("/selected/workspace"),
        &["selected-package".into()],
        Command::new(root.path().join("missing-program")),
        &mut log,
    );
    assert!(result.is_err());
    assert_stage_receipt(&log, "test", "spawn-error")?;
    assert!(std::str::from_utf8(&log)?.contains(" error="));
    Ok(())
}
