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

#[test]
fn shared_feature_inputs_select_every_direct_consumer_and_reverse_guest() -> TestResult {
    let mut graph = Graph::default();
    graph.add(
        Path::new("/system"),
        &metadata(
            "system",
            &[
                ("tests", "fixture-tests", &["realization", "api"]),
                ("realization", "realization", &["api", "macros"]),
                ("api", "api", &[]),
                ("macros", "realization-macros", &[]),
                ("guest", "guest", &["realization"]),
            ],
            &["tests", "realization", "api", "macros", "guest"],
        ),
    )?;
    let mut options =
        Options { always: BTreeSet::from(["fixture-tests".into()]), ..Options::default() };
    options.mappings.push(("/system/feature".into(), "fixture-tests".into()));
    assert_eq!(
        selected(&graph, &options, &["/system/feature/activity/model.rs"])?,
        BTreeSet::from(["fixture-tests".into()]),
        "a test consumer seed cannot traverse forward dependencies to realization or its guest"
    );
    for package in ["realization", "api", "realization-macros"] {
        options.mappings.push(("/system/feature".into(), package.into()));
    }
    let expected = BTreeSet::from([
        "fixture-tests".into(),
        "realization".into(),
        "api".into(),
        "realization-macros".into(),
        "guest".into(),
    ]);
    for file in ["model.rs", "phase.rs", "probe_gaps.rs"] {
        let path = format!("/system/feature/activity/{file}");
        assert_eq!(selected(&graph, &options, &[&path])?, expected, "{file}");
    }
    Ok(())
}

#[test]
fn shared_kettle_model_seeds_source_consumers_and_reverse_guest() -> TestResult {
    let mut metadata: serde_json::Value = serde_json::from_slice(&metadata(
        "framework",
        &[
            ("fixtures", "umpire-fixtures", &["testing"]),
            ("reference", "umpire-reference-kettle", &[]),
            ("guest", "umpire-kettle-guest", &["reference"]),
            ("suite", "umpire-suite", &[]),
            ("tla", "umpire-tla", &[]),
            ("apalache", "umpire-apalache", &[]),
            ("testing", "umpire-testing", &["tla"]),
            ("other", "unrelated", &["testing"]),
        ],
        &["fixtures", "reference", "guest", "suite", "tla", "apalache", "testing", "other"],
    ))?;
    metadata["packages"][0]["manifest_path"] =
        serde_json::json!("/framework/dev/fixtures/Cargo.toml");
    metadata["packages"][1]["manifest_path"] =
        serde_json::json!("/framework/reference/kettle/Cargo.toml");
    metadata["packages"][2]["manifest_path"] =
        serde_json::json!("/framework/reference/kettle/guest/Cargo.toml");
    metadata["packages"][3]["manifest_path"] =
        serde_json::json!("/framework/author/suite/Cargo.toml");
    metadata["packages"][4]["manifest_path"] =
        serde_json::json!("/framework/backends/tla/Cargo.toml");
    metadata["packages"][5]["manifest_path"] =
        serde_json::json!("/framework/backends/apalache/Cargo.toml");
    let mut graph = Graph::default();
    graph.add(Path::new("/framework"), &serde_json::to_vec(&metadata)?)?;
    let options =
        Options { always: BTreeSet::from(["umpire-fixtures".into()]), ..Options::default() };
    assert_eq!(
        selected(&graph, &options, &["/framework/dev/fixtures/fixtures/sample_kettle/model.rs"])?,
        BTreeSet::from([
            "umpire-fixtures".into(),
            "umpire-reference-kettle".into(),
            "umpire-kettle-guest".into(),
            "umpire-suite".into(),
            "umpire-tla".into(),
            "umpire-apalache".into(),
        ])
    );
    assert_eq!(
        selected(&graph, &options, &["/framework/backends/tla/src/lib.rs"])?,
        BTreeSet::from([
            "umpire-fixtures".into(),
            "umpire-tla".into(),
            "umpire-testing".into(),
            "unrelated".into(),
        ]),
        "an actual library edit still expands its reverse dependencies"
    );
    assert_eq!(
        selected(&graph, &options, &["/framework/dev/fixtures/fixtures/sample_kettle/tests.rs"])?,
        BTreeSet::from(["umpire-fixtures".into()])
    );
    Ok(())
}

#[test]
fn model_edits_ignore_reported_non_model_metadata_and_docs() -> TestResult {
    let graph = graph()?;
    let mut options = options();
    options.mappings.push(("/system/feature".into(), "activity".into()));
    for path in [
        "/.flow/tasks/task.json",
        "/.flow/artifacts/task/proof.log",
        "/.plans/UMPIRE4_SPEC.md",
        "/.turbo/report.md",
        "/docs/development.md",
        "/MILESTONES.md",
        "/README.md",
    ] {
        let selection = graph
            .select(&options, vec!["/system/feature/activity/model.rs".into(), path.into()])?;
        assert_eq!(
            selection
                .workspaces
                .into_iter()
                .flat_map(|workspace| workspace.packages)
                .collect::<BTreeSet<_>>(),
            BTreeSet::from(["fixtures".into(), "activity".into()]),
            "{path}"
        );
        assert!(
            selection
                .reasons
                .iter()
                .any(|reason| reason.contains("ignored") && reason.contains(path)),
            "{path}"
        );
    }
    Ok(())
}

#[test]
fn external_build_inputs_and_unknown_sources_remain_conservative() -> TestResult {
    let graph = graph()?;
    for path in [
        "/Makefile",
        "/mise.toml",
        "/mise.lock",
        "/rust-toolchain.toml",
        "/go.mod",
        "/go.sum",
        "/cmd/tools/getproto/main.go",
        "/proto/internal/service.proto",
        "/proto/api.binpb",
        "/api/generated.pb.go",
        "/unknown/build-input.rs",
        "/docs/build.rs",
    ] {
        assert_eq!(
            selected(&graph, &options(), &[path])?,
            BTreeSet::from(["core".into(), "fixtures".into(), "activity".into(), "other".into()]),
            "{path}"
        );
    }
    for path in ["/Makefile", "/mise.lock"] {
        let selection = graph.select(&options(), vec![path.into()])?;
        assert!(selection.reasons.iter().any(|reason| reason.contains("global input")), "{path}");
    }
    Ok(())
}

#[test]
fn renamed_sources_select_both_owners_with_metadata_changes() -> TestResult {
    assert_eq!(
        selected(
            &graph()?,
            &options(),
            &["/framework/core/src/old.rs", "/system/other/src/new.rs", "/.flow/specs/spec.json"],
        )?,
        BTreeSet::from(["core".into(), "fixtures".into(), "activity".into(), "other".into()])
    );
    Ok(())
}

#[test]
fn metadata_named_checkout_ancestors_do_not_hide_external_build_inputs() -> TestResult {
    let root = "/tmp/.flow/candidate/model";
    let mut graph = Graph::default();
    graph.add(
        Path::new(root),
        &metadata(
            "tmp/.flow/candidate/model",
            &[
                ("fixtures", "fixtures", &[]),
                ("activity", "activity", &[]),
                ("other", "other", &[]),
            ],
            &["fixtures", "activity", "other"],
        ),
    )?;
    for path in ["/tmp/.flow/candidate/proto/api.binpb", "/tmp/.flow/candidate/unknown/source.rs"] {
        assert_eq!(
            selected(&graph, &options(), &[path])?,
            BTreeSet::from(["fixtures".into(), "activity".into(), "other".into()]),
            "{path}"
        );
    }
    assert_eq!(
        selected(&graph, &options(), &["/tmp/.flow/candidate/model/.flow/Cargo.toml"])?,
        BTreeSet::from(["fixtures".into(), "activity".into(), "other".into()]),
        "explicit global inputs take precedence over metadata exclusions"
    );
    assert_eq!(
        selected(&graph, &options(), &["/tmp/.flow/candidate/model/activity/.flow/evidence.json"])?,
        BTreeSet::from(["fixtures".into(), "activity".into()]),
        "package ownership takes precedence over metadata exclusions"
    );
    let mut mapped = options();
    mapped.mappings.push(("/tmp/.flow/candidate/.flow".into(), "other".into()));
    assert_eq!(
        selected(&graph, &mapped, &["/tmp/.flow/candidate/.flow/evidence.json"])?,
        BTreeSet::from(["fixtures".into(), "other".into()]),
        "explicit mappings take precedence over metadata exclusions"
    );
    for path in [
        "/tmp/.flow/candidate/.flow/evidence.json",
        "/tmp/.flow/candidate/model/.flow/evidence.json",
    ] {
        let selection = graph.select(
            &options(),
            vec!["/tmp/.flow/candidate/model/activity/model.rs".into(), path.into()],
        )?;
        assert_eq!(
            selection
                .workspaces
                .into_iter()
                .flat_map(|workspace| workspace.packages)
                .collect::<BTreeSet<_>>(),
            BTreeSet::from(["fixtures".into(), "activity".into()]),
            "{path}"
        );
        assert!(
            selection
                .reasons
                .iter()
                .any(|reason| reason.contains("ignored") && reason.contains(path)),
            "{path}"
        );
    }
    Ok(())
}

#[test]
fn workspace_documentation_can_supply_executable_test_inputs() -> TestResult {
    let mut graph = Graph::default();
    graph.add(
        Path::new("/repo/model/umpire"),
        &metadata(
            "repo/model/umpire",
            &[("fixtures", "fixtures", &[]), ("dsl", "dsl", &[]), ("other", "other", &[])],
            &["fixtures", "dsl", "other"],
        ),
    )?;
    for path in ["/repo/model/umpire/README.md", "/repo/model/umpire/docs/examples.md"] {
        assert_eq!(
            selected(&graph, &options(), &[path])?,
            BTreeSet::from(["fixtures".into(), "dsl".into(), "other".into()]),
            "{path}"
        );
    }
    for path in ["/repo/README.md", "/repo/MILESTONES.md", "/repo/docs/development.md"] {
        assert_eq!(
            selected(&graph, &options(), &["/repo/model/umpire/dsl/src/lib.rs", path])?,
            BTreeSet::from(["fixtures".into(), "dsl".into()]),
            "{path}"
        );
    }
    Ok(())
}
