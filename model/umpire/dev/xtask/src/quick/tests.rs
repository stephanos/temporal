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
fn stage_receipt_reports_missing_tool_as_failed_status() -> TestResult {
    let root = tempfile::tempdir()?;
    let mut log = Vec::new();
    let result = super::execute(
        "test",
        Path::new("/selected/workspace"),
        &["selected-package".into()],
        Command::new(root.path().join("missing-program")),
        &mut log,
    );
    assert_eq!(result?, 1);
    assert_stage_receipt(&log, "test", "1")?;
    Ok(())
}

#[test]
fn stage_receipt_reports_elapsed_and_spawn_failure_without_success_status() -> TestResult {
    let root = tempfile::tempdir()?;
    let mut log = Vec::new();
    let mut command = Command::new("/bin/true");
    command.current_dir(root.path().join("missing-directory"));
    let result = super::execute(
        "test",
        Path::new("/selected/workspace"),
        &["selected-package".into()],
        command,
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
    assert!(graph.compiler_only(Path::new("/framework"), "umpire-kettle-guest"));
    assert!(!graph.compiler_only(Path::new("/framework"), "umpire-reference-kettle"));
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

#[test]
fn metadata_preserves_doctests_without_requesting_binary_only_docs() -> TestResult {
    let mut metadata: serde_json::Value = serde_json::from_slice(&metadata(
        "system",
        &[("library", "library", &[]), ("guest", "guest", &[]), ("disabled", "disabled", &[])],
        &["library", "guest", "disabled"],
    ))?;
    metadata["packages"][0]["targets"] = serde_json::json!([
        {"kind": ["lib"], "doctest": true}
    ]);
    metadata["packages"][1]["targets"] = serde_json::json!([
        {"kind": ["bin"], "doctest": false}
    ]);
    metadata["packages"][2]["targets"] = serde_json::json!([
        {"kind": ["lib"], "doctest": false}
    ]);
    let mut graph = Graph::default();
    graph.add(Path::new("/system"), &serde_json::to_vec(&metadata)?)?;
    let options = Options { always: BTreeSet::from(["library".into()]), ..Options::default() };
    let selection = graph.select(&options, vec!["/system/Cargo.toml".into()])?;
    assert_eq!(selection.workspaces[0].packages, ["disabled", "guest", "library"]);
    assert_eq!(selection.workspaces[0].doctests, BTreeSet::from(["library".into()]));
    Ok(())
}

#[test]
fn explicit_compiler_only_packages_must_belong_to_an_admitted_workspace() -> TestResult {
    let mut options = options();
    options.compiler_only.insert("missing".into());
    assert!(graph()?.select(&options, vec![]).is_err());
    Ok(())
}

#[test]
fn framework_abi_guests_are_explicit_compiler_only_consumers() -> TestResult {
    let guests = [
        ("umpire-kettle-guest", "reference/kettle/guest"),
        ("umpire-stock-guest", "protocol/runtime/guests/stock"),
        ("umpire-control-guest", "protocol/runtime/guests/control"),
        ("umpire-caught-panic-guest", "protocol/runtime/guests/caught-panic"),
        ("umpire-overflow-guest", "protocol/runtime/guests/overflow"),
    ];
    let mut data: serde_json::Value = serde_json::from_slice(&metadata(
        "framework",
        &guests.map(|(name, _)| (name, name, &[][..])),
        &guests.map(|(name, _)| name),
    ))?;
    for (index, (_, directory)) in guests.iter().enumerate() {
        data["packages"][index]["manifest_path"] =
            serde_json::json!(format!("/framework/{directory}/Cargo.toml"));
    }
    let mut graph = Graph::default();
    graph.add(Path::new("/framework"), &serde_json::to_vec(&data)?)?;
    for (name, _) in guests {
        assert!(graph.compiler_only(Path::new("/framework"), name));
        assert!(!graph.compiler_only(Path::new("/other-workspace"), name));
    }
    Ok(())
}

#[test]
fn workspace_execution_checks_build_only_consumers_then_batches_tests_and_scoped_docs() -> TestResult
{
    let directory = tempfile::tempdir()?;
    let root = directory.path();
    let inventory = serde_json::json!({"rust-suites": {
        "api": {"package-name": "api", "binary-name": "api", "kind": "lib", "status": "listed", "testcases": {
            "api_test": {"ignored": false, "filter-match": {"status": "matches"}}
        }},
        "fixtures": {"package-name": "fixtures", "binary-name": "fixtures", "kind": "test", "status": "listed", "testcases": {
            "fixtures": {"ignored": false, "filter-match": {"status": "matches"}}
        }}
    }});
    fs::write(root.join("inventory.json"), serde_json::to_vec(&inventory)?)?;
    let script = root.join("cargo.sh");
    fs::write(
        &script,
        r#"set -eu
printf '%s\n' "$*" >> "$CONTROL_ROOT/calls"
case "$1" in
 check) exit "$CONTROL_COMPILE_EXIT" ;;
 nextest) case "$2" in
  list) cat "$CONTROL_ROOT/inventory.json"; exit "$CONTROL_LIST_EXIT" ;;
  run) exit "$CONTROL_RUN_EXIT" ;;
  *) exit 91 ;;
 esac ;;
 test) [ "$2" = --doc ]; exit "$CONTROL_DOC_EXIT" ;;
 *) exit 92 ;;
esac
"#,
    )?;
    let workspace = super::WorkspaceSelection {
        workspace: root.into(),
        packages: vec![
            "api".into(),
            "guest".into(),
            "fixtures".into(),
            "umpire".into(),
            "umpire-ir-derive".into(),
        ],
        doctests: BTreeSet::from(["api".into(), "umpire".into(), "umpire-ir-derive".into()]),
    };
    let options = Options {
        mappings: vec![("/feature".into(), "fixtures".into())],
        targets: [("fixtures".into(), "fixtures".into())].into(),
        filters: [("fixtures".into(), "fixtures".into())].into(),
        ..Options::default()
    };
    for (compile, list, run, doc, expected, count) in [
        (0, 0, 0, 0, 0, 4),
        (13, 0, 0, 0, 13, 1),
        (0, 17, 0, 0, 17, 2),
        (0, 0, 29, 0, 29, 3),
        (0, 0, 0, 37, 37, 4),
    ] {
        fs::write(root.join("calls"), "")?;
        let status = super::test_workspace(
            &workspace,
            &options,
            &["/feature/activity/model.rs".into()],
            &BTreeSet::from(["guest".into(), "umpire".into(), "umpire-ir-derive".into()]),
            || {
                let mut command = Command::new("/bin/sh");
                command
                    .arg(&script)
                    .current_dir(root)
                    .env("CONTROL_ROOT", root)
                    .env("CONTROL_COMPILE_EXIT", compile.to_string())
                    .env("CONTROL_LIST_EXIT", list.to_string())
                    .env("CONTROL_RUN_EXIT", run.to_string())
                    .env("CONTROL_DOC_EXIT", doc.to_string());
                command
            },
        )?;
        assert_eq!(status, expected);
        let calls = fs::read_to_string(root.join("calls"))?;
        let lines = calls.lines().collect::<Vec<_>>();
        assert_eq!(lines.len(), count, "{calls}");
        assert_eq!(lines[0], "check --locked --all-targets -p guest -p umpire -p umpire-ir-derive");
        if count >= 3 {
            assert!(lines[2].starts_with("nextest run "), "{calls}");
            assert!(
                lines[2].contains("-p api -p guest -p fixtures -p umpire -p umpire-ir-derive"),
                "{calls}"
            );
            assert!(
                lines[2].contains("kind(test) & binary(=fixtures) & test(=fixtures)"),
                "{calls}"
            );
        }
        if count == 4 {
            assert_eq!(
                lines[3],
                "test --doc --locked -p api -p umpire -p umpire-ir-derive -- --nocapture"
            );
        }
    }
    Ok(())
}

#[test]
#[cfg(target_os = "linux")]
fn quick_leaf_reaps_background_children_and_preserves_failed_status() -> TestResult {
    let directory = tempfile::tempdir()?;
    let pid = directory.path().join("child.pid");
    let mut command = Command::new("/bin/sh");
    command
        .args(["-c", "sleep 30 & printf '%s' $! > \"$CONTROL_PID\"; exit 37"])
        .env("CONTROL_PID", &pid);
    assert_eq!(
        super::execute("test", directory.path(), &["member".into()], command, Vec::new())?,
        37
    );
    let pid = fs::read_to_string(pid)?;
    let alive = Command::new("/bin/kill")
        .args(["-0", &pid])
        .stderr(std::process::Stdio::null())
        .status()?
        .success();
    if alive {
        Command::new("/bin/kill").args(["-KILL", &pid]).status()?;
    }
    assert!(!alive, "quick returned while its background child was alive");
    Ok(())
}

#[test]
fn global_fallback_compiles_zero_harness_members_and_keeps_their_tests_and_doctests() -> TestResult
{
    let root = Path::new("/framework");
    let mut data: serde_json::Value = serde_json::from_slice(&metadata(
        "framework",
        &[
            ("api", "api", &[]),
            ("facade", "umpire", &["api"]),
            ("derive", "umpire-ir-derive", &[]),
            ("unknown", "unknown", &[]),
        ],
        &["api", "facade", "derive", "unknown"],
    ))?;
    for (index, directory, kind) in [(1, "author/umpire", "lib"), (2, "ir/derive", "proc-macro")] {
        data["packages"][index]["manifest_path"] =
            serde_json::json!(root.join(directory).join("Cargo.toml"));
        data["packages"][index]["targets"] = serde_json::json!([{"kind": [kind], "doctest": true}]);
    }
    let mut graph = Graph::default();
    graph.add(root, &serde_json::to_vec(&data)?)?;
    let options = Options { always: BTreeSet::from(["api".into()]), ..Options::default() };
    let selection = graph.select(&options, vec![root.join("Cargo.toml")])?;
    let workspace = &selection.workspaces[0];
    assert_eq!(workspace.packages, ["api", "umpire", "umpire-ir-derive", "unknown"]);
    assert_eq!(workspace.doctests, BTreeSet::from(["umpire".into(), "umpire-ir-derive".into()]));
    let compiler_only = workspace
        .packages
        .iter()
        .filter(|name| graph.compiler_only(root, name))
        .cloned()
        .collect::<BTreeSet<_>>();
    assert!(!graph.compiler_only(Path::new("/other-workspace"), "umpire"));
    let selections = workspace
        .packages
        .iter()
        .map(|name| crate::test_run::Selection {
            package: name,
            target: None,
            filter: None,
            doctests: workspace.doctests.contains(name),
        })
        .collect::<Vec<_>>();
    let inventory = |names: &[&str]| {
        serde_json::to_vec(&serde_json::json!({"rust-suites": names.iter().map(|name| {
        (*name, serde_json::json!({"package-name": name, "binary-name": name, "kind": "lib", "status": "listed", "testcases": {
            "case": {"ignored": false, "filter-match": {"status": "matches"}}
        }}))
    }).collect::<std::collections::BTreeMap<_, _>>()}))
    };
    assert!(
        crate::test_run::run_selected_checked_with(
            &selections,
            &compiler_only,
            |_| Ok((0, inventory(&["api"])?)),
            |_, _| panic!("unexpected zero-test package admitted"),
        )
        .is_err()
    );
    for (target, filter) in [(Some("missing"), None), (None, Some("missing"))] {
        let scoped =
            [crate::test_run::Selection { package: "umpire", target, filter, doctests: true }];
        assert!(
            crate::test_run::run_selected_checked_with(
                &scoped,
                &compiler_only,
                |_| Ok((0, inventory(&[])?)),
                |_, _| panic!("empty exact selection was admitted as compiler-only"),
            )
            .is_err()
        );
    }
    for packages_with_tests in
        [vec!["api", "unknown"], vec!["api", "umpire", "umpire-ir-derive", "unknown"]]
    {
        let mut calls = Vec::new();
        crate::test_run::run_selected_checked_with(
            &selections,
            &compiler_only,
            |_| Ok((0, inventory(&packages_with_tests)?)),
            |phase, args| {
                calls.push((phase.to_owned(), args.join(" ")));
                Ok(0)
            },
        )?;
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0].0, "test");
        assert!(calls[0].1.contains("-p api -p umpire -p umpire-ir-derive -p unknown"));
        assert_eq!(
            calls[1],
            (
                "doctest".into(),
                "test --doc --locked -p umpire -p umpire-ir-derive -- --nocapture".into()
            )
        );
    }
    Ok(())
}
