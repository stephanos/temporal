//! Framework tests across binaries, followed by Cargo's workspace doctests.
use std::{env, ffi::OsStr, os::unix::process::ExitStatusExt, process::Command};

/// Run workspace tests with the caller's admitted gate environment and working directory.
pub fn run() -> crate::Result<i32> {
    run_with(|arguments| {
        let mut command = cargo();
        command.args(arguments);
        let status = command.status()?;
        Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
    })
}

fn cargo() -> Command {
    let mut command = Command::new("mise");
    command
        .args(["exec", "--", "cargo"])
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    configure(
        &mut command,
        env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require"),
        env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
    );
    command
}

#[derive(serde::Deserialize)]
struct Metadata {
    packages: Vec<Package>,
    workspace_members: Vec<String>,
}
#[derive(serde::Deserialize)]
struct Package {
    id: String,
    name: String,
    targets: Vec<Target>,
}
#[derive(serde::Deserialize)]
struct Target {
    kind: Vec<String>,
    doctest: bool,
}

fn package_doctests(package: &str, bytes: &[u8]) -> crate::Result<bool> {
    let metadata: Metadata = serde_json::from_slice(bytes).map_err(|error| {
        crate::Error::Invalid(format!("invalid scoped Cargo metadata: {error}"))
    })?;
    let member = metadata
        .packages
        .iter()
        .find(|candidate| {
            candidate.name == package && metadata.workspace_members.contains(&candidate.id)
        })
        .ok_or_else(|| {
            crate::Error::Invalid(format!(
                "scoped test package {package:?} is not a workspace member"
            ))
        })?;
    Ok(member.targets.iter().any(|target| {
        target.doctest && target.kind.iter().any(|kind| kind == "lib" || kind == "proc-macro")
    }))
}

/// Run one workspace member's admitted nextest inventory, tests and eligible doctests.
///
/// # Errors
/// Rejects missing members, malformed metadata/inventories and unexpectedly empty selections.
pub fn run_package(package: &str) -> crate::Result<i32> {
    if package.is_empty() {
        return Err(crate::Error::Invalid("scoped test package is empty".into()));
    }
    let output = cargo()
        .args(["metadata", "--locked", "--no-deps", "--format-version", "1"])
        .stderr(std::process::Stdio::inherit())
        .output()?;
    if !output.status.success() {
        return Ok(output
            .status
            .code()
            .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)));
    }
    let selections = [Selection {
        package,
        target: None,
        filter: None,
        doctests: package_doctests(package, &output.stdout)?,
    }];
    run_selected_checked_with(
        &selections,
        &std::collections::BTreeSet::new(),
        |arguments| {
            let output = cargo().args(arguments).stderr(std::process::Stdio::inherit()).output()?;
            Ok((
                output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)),
                output.stdout,
            ))
        },
        |_, arguments| {
            let status = cargo().args(arguments).status()?;
            Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
        },
    )
}

pub(crate) fn test_threads(apalache: bool, inherited: Option<&OsStr>) -> usize {
    // Optional mixed-backend allocation still needs its real resource proof.
    let cap = if apalache { 1 } else { 4 };
    inherited
        .and_then(OsStr::to_str)
        .and_then(|value| value.parse::<usize>().ok())
        .filter(|threads| *threads > 0)
        .map_or(cap, |threads| threads.min(cap))
}

pub(crate) fn configure(command: &mut Command, required: bool, apalache: bool) {
    if required || apalache {
        let inherited =
            command.get_envs().find(|(name, _)| *name == "NEXTEST_TEST_THREADS").map_or_else(
                || env::var_os("NEXTEST_TEST_THREADS"),
                |(_, value)| value.map(OsStr::to_os_string),
            );
        command
            .env("NEXTEST_PROFILE", "required")
            .env("NEXTEST_TEST_THREADS", test_threads(apalache, inherited.as_deref()).to_string());
    }
}

fn run_with(mut execute: impl FnMut(&[&str]) -> crate::Result<i32>) -> crate::Result<i32> {
    let status = execute(&[
        "nextest",
        "run",
        "--workspace",
        "--locked",
        "--fail-fast",
        "--run-ignored",
        "default",
        "--status-level",
        "slow",
        "--final-status-level",
        "slow",
        "--success-output",
        "never",
        "--failure-output",
        "immediate-final",
        "--no-tests",
        "fail",
    ])?;
    if status != 0 {
        return Ok(status);
    }
    execute(&["test", "--doc", "--workspace", "--locked", "--quiet"])
}

/// Per-package quick selection; target and filter retain Cargo's integration/exact semantics.
pub struct Selection<'a> {
    /// Exact Cargo workspace package name.
    pub package: &'a str,
    /// Optional integration-test binary, preserving `cargo test --test` semantics.
    pub target: Option<&'a str>,
    /// Optional exact libtest identity.
    pub filter: Option<&'a str>,
    /// Whether this package declares an enabled library doctest target.
    pub doctests: bool,
}

fn predicate(name: &str, value: &str) -> String {
    let escaped: String = value
        .chars()
        .map(|character| match character {
            '\\' => r"\\".into(),
            ')' => r"\)".into(),
            ',' => r"\,".into(),
            '\n' => r"\n".into(),
            '\r' => r"\r".into(),
            '\t' => r"\t".into(),
            character if character.is_whitespace() => format!("\\u{{{:x}}}", u32::from(character)),
            character => character.to_string(),
        })
        .collect();
    format!("{name}(={escaped})")
}

fn selection_arguments(selections: &[Selection<'_>]) -> Vec<String> {
    let mut arguments = Vec::new();
    let mut expressions = Vec::new();
    for selection in selections {
        arguments.extend(["-p".into(), selection.package.into()]);
        let mut expression = predicate("package", selection.package);
        if let Some(target) = selection.target {
            expression.push_str(" & kind(test) & ");
            expression.push_str(&predicate("binary", target));
        }
        if let Some(filter) = selection.filter {
            expression.push_str(" & ");
            expression.push_str(&predicate("test", filter));
        }
        expressions.push(format!("({expression})"));
    }
    arguments.extend(["-E".into(), expressions.join(" | ")]);
    arguments
}

#[derive(serde::Deserialize)]
struct Inventory {
    #[serde(rename = "rust-suites")]
    suites: std::collections::BTreeMap<String, InventorySuite>,
}

#[derive(serde::Deserialize)]
struct InventorySuite {
    #[serde(rename = "package-name")]
    package: String,
    #[serde(rename = "binary-name")]
    binary: String,
    kind: String,
    status: String,
    #[serde(default)]
    testcases: std::collections::BTreeMap<String, InventoryTest>,
}

#[derive(serde::Deserialize)]
struct InventoryTest {
    ignored: bool,
    #[serde(rename = "filter-match")]
    filter_match: InventoryMatch,
}

#[derive(serde::Deserialize)]
struct InventoryMatch {
    status: String,
}

fn admit_inventory(
    selections: &[Selection<'_>],
    compiler_only: &std::collections::BTreeSet<String>,
    bytes: &[u8],
) -> crate::Result<bool> {
    let inventory: Inventory = serde_json::from_slice(bytes)
        .map_err(|error| crate::Error::Invalid(format!("invalid nextest inventory: {error}")))?;
    let mut any_runnable = false;
    for selection in selections {
        let runnable = inventory.suites.values().any(|suite| {
            suite.package == selection.package
                && suite.status == "listed"
                && selection
                    .target
                    .is_none_or(|target| suite.kind == "test" && suite.binary == target)
                && suite.testcases.iter().any(|(name, test)| {
                    !test.ignored
                        && test.filter_match.status == "matches"
                        && selection.filter.is_none_or(|filter| name == filter)
                })
        });
        any_runnable |= runnable;
        let admitted_build = compiler_only.contains(selection.package)
            && selection.target.is_none()
            && selection.filter.is_none();
        if !(runnable || admitted_build) {
            return Err(crate::Error::Invalid(format!(
                "quick nextest selection has no runnable tests: package={} target={:?} filter={:?}",
                selection.package, selection.target, selection.filter
            )));
        }
    }
    Ok(any_runnable)
}

/// Admit each selected package against the same nextest inventory before executing the batch.
/// Explicit compiler-only packages must already have passed Cargo compilation and retain any tests.
///
/// # Errors
/// Rejects malformed inventories and each unexpectedly empty package/target/test selection.
pub fn run_selected_checked_with(
    selections: &[Selection<'_>],
    compiler_only: &std::collections::BTreeSet<String>,
    mut inventory: impl FnMut(&[String]) -> crate::Result<(i32, Vec<u8>)>,
    execute: impl FnMut(&str, &[String]) -> crate::Result<i32>,
) -> crate::Result<i32> {
    if selections.is_empty() {
        return Err(crate::Error::Invalid("quick test selection is empty".into()));
    }
    let mut arguments =
        ["nextest", "list", "--locked", "--message-format", "json", "--run-ignored", "default"]
            .map(String::from)
            .to_vec();
    arguments.extend(selection_arguments(selections));
    let (status, bytes) = inventory(&arguments)?;
    if status != 0 {
        return Ok(status);
    }
    let binaries = admit_inventory(selections, compiler_only, &bytes)?;
    run_selected(selections, binaries, execute)
}

/// Run one selected nextest batch, then corresponding Cargo doctests.
///
/// # Errors
/// Rejects an empty package set and propagates command execution errors.
pub fn run_selected_with(
    selections: &[Selection<'_>],
    execute: impl FnMut(&str, &[String]) -> crate::Result<i32>,
) -> crate::Result<i32> {
    run_selected(selections, true, execute)
}

fn run_selected(
    selections: &[Selection<'_>],
    binaries: bool,
    mut execute: impl FnMut(&str, &[String]) -> crate::Result<i32>,
) -> crate::Result<i32> {
    if selections.is_empty() {
        return Err(crate::Error::Invalid("quick test selection is empty".into()));
    }
    let mut arguments = [
        "nextest",
        "run",
        "--locked",
        "--fail-fast",
        "--run-ignored",
        "default",
        "--status-level",
        "slow",
        "--final-status-level",
        "slow",
        "--success-output",
        "immediate",
        "--failure-output",
        "immediate-final",
        "--no-tests",
        "fail",
    ]
    .map(String::from)
    .to_vec();
    arguments.extend(selection_arguments(selections));
    let mut doctests = std::collections::BTreeMap::<Option<&str>, Vec<&str>>::new();
    for selection in selections {
        if selection.target.is_none() && selection.doctests {
            doctests.entry(selection.filter).or_default().push(selection.package);
        }
    }
    if binaries {
        let status = execute("test", &arguments)?;
        if status != 0 {
            return Ok(status);
        }
    }
    for (filter, packages) in doctests {
        let mut arguments = ["test", "--doc", "--locked"].map(String::from).to_vec();
        for package in packages {
            arguments.extend(["-p".into(), package.into()]);
        }
        if let Some(filter) = filter {
            arguments.push(filter.into());
        }
        arguments.extend(["--".into(), "--nocapture".into()]);
        if filter.is_some() {
            arguments.push("--exact".into());
        }
        let status = execute("doctest", &arguments)?;
        if status != 0 {
            return Ok(status);
        }
    }
    Ok(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    #[allow(clippy::unwrap_used)]
    fn required_backend_policy_caps_inherited_threads_and_preserves_skipped_defaults() {
        for (required, apalache, cap) in [(true, false, "4"), (false, true, "1"), (true, true, "1")]
        {
            for (inherited, lower) in [
                (None, None),
                (Some("99"), None),
                (Some("2"), Some("2")),
                (Some("1"), Some("1")),
                (Some("malformed"), None),
                (Some("0"), None),
                (Some("-1"), None),
            ] {
                let mut command = Command::new("/bin/sh");
                command.env("NEXTEST_PROFILE", "default").env_remove("NEXTEST_TEST_THREADS");
                if let Some(inherited) = inherited {
                    command.env("NEXTEST_TEST_THREADS", inherited);
                }
                configure(&mut command, required, apalache);
                configure(&mut command, required, apalache);
                let output = command
                    .args(["-c", "printf '%s:%s' \"$NEXTEST_PROFILE\" \"$NEXTEST_TEST_THREADS\""])
                    .output()
                    .unwrap();
                assert!(output.status.success());
                let threads = if apalache { "1" } else { lower.unwrap_or(cap) };
                assert_eq!(
                    String::from_utf8(output.stdout).unwrap(),
                    format!("required:{threads}"),
                    "inherited={inherited:?}, apalache={apalache}"
                );
            }
        }
        let mut command = Command::new("/bin/sh");
        configure(&mut command, false, false);
        assert!(
            !command
                .get_envs()
                .any(|(key, _)| key == "NEXTEST_PROFILE" || key == "NEXTEST_TEST_THREADS")
        );
        command.env("NEXTEST_PROFILE", "default").env("NEXTEST_TEST_THREADS", "2");
        configure(&mut command, false, false);
        let output = command
            .args(["-c", "printf '%s:%s' \"$NEXTEST_PROFILE\" \"$NEXTEST_TEST_THREADS\""])
            .output()
            .unwrap();
        assert!(output.status.success());
        assert_eq!(String::from_utf8(output.stdout).unwrap(), "default:2");
    }

    #[test]
    fn workspace_selection_rejects_an_unexpected_zero_test_run() {
        let result = run_with(|arguments| {
            if arguments.first() == Some(&"nextest") {
                assert!(arguments.windows(2).any(|pair| pair == ["--no-tests", "fail"]));
            }
            Ok(0)
        });
        assert!(result.is_ok());
    }

    #[test]
    fn scoped_package_admission_uses_membership_and_actual_doctest_targets() -> crate::Result<()> {
        let metadata = serde_json::to_vec(&serde_json::json!({
            "workspace_members": ["library", "binary", "disabled"],
            "packages": [
                {"id": "library", "name": "library", "targets": [{"kind": ["lib"], "doctest": true}]},
                {"id": "binary", "name": "binary", "targets": [{"kind": ["bin"], "doctest": false}]},
                {"id": "disabled", "name": "disabled", "targets": [{"kind": ["lib"], "doctest": false}]},
                {"id": "external", "name": "dependency", "targets": [{"kind": ["lib"], "doctest": true}]}
            ]
        }))?;
        assert!(package_doctests("library", &metadata)?);
        assert!(!package_doctests("binary", &metadata)?);
        assert!(!package_doctests("disabled", &metadata)?);
        assert!(package_doctests("dependency", &metadata).is_err());
        assert!(package_doctests("", &metadata).is_err());
        assert!(package_doctests("missing", &metadata).is_err());
        assert!(package_doctests("library", b"{}").is_err());
        Ok(())
    }

    #[test]
    fn scoped_runner_child() -> Result<(), Box<dyn std::error::Error>> {
        let Ok(package) = env::var("UMPIRE_SCOPED_TEST_PACKAGE") else { return Ok(()) };
        assert_eq!(run_package(&package)?, env::var("UMPIRE_SCOPED_TEST_EXIT")?.parse::<i32>()?);
        Ok(())
    }

    #[test]
    #[allow(clippy::unwrap_used)]
    fn scoped_runner_executes_inventory_tests_and_docs_with_the_same_admitted_policy()
    -> crate::Result<()> {
        use std::{fs, os::unix::fs::PermissionsExt};
        let directory = tempfile::tempdir()?;
        let root = fs::canonicalize(directory.path())?;
        fs::write(
            root.join("metadata.json"),
            serde_json::to_vec(&serde_json::json!({
                "workspace_members": ["member"],
                "packages": [{"id": "member", "name": "member", "targets": [{"kind": ["lib"], "doctest": true}]}]
            }))?,
        )?;
        fs::write(
            root.join("inventory.json"),
            inventory(&[("member", "member", "lib", "real_test", false, "matches")]),
        )?;
        let script = root.join("mise");
        fs::write(
            &script,
            r#"#!/bin/sh
set -eu
[ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL" = false:false ]
[ "${NEXTEST_PROFILE:-unset}:${NEXTEST_TEST_THREADS:-unset}" = "$CONTROL_POLICY" ]
[ "$PWD" = "$CONTROL_ROOT" ]
printf '%s\n' "$*" >> "$CONTROL_ROOT/calls"
case "$*" in
 'exec -- cargo metadata --locked --no-deps --format-version 1') cat "$CONTROL_ROOT/metadata.json" ;;
 'exec -- cargo nextest list '*) cat "$CONTROL_ROOT/inventory.json"; exit "$CONTROL_LIST_EXIT" ;;
 'exec -- cargo nextest run '*) exit "$CONTROL_RUN_EXIT" ;;
 'exec -- cargo test --doc '*) exit "$CONTROL_DOC_EXIT" ;;
 *) exit 91 ;;
esac
"#,
        )?;
        fs::set_permissions(&script, fs::Permissions::from_mode(0o755))?;
        for (backend, apalache, policy) in [
            ("skip", "skip", "unset:unset"),
            ("require", "skip", "required:2"),
            ("skip", "require", "required:1"),
        ] {
            for (list, run, doc, expected, calls) in
                [(0, 0, 0, 0, 4), (29, 0, 0, 29, 2), (0, 37, 0, 37, 3), (0, 0, 41, 41, 4)]
            {
                fs::write(root.join("calls"), "")?;
                let mut command = Command::new(env::current_exe()?);
                command
                    .args(["--exact", "test_run::tests::scoped_runner_child", "--nocapture"])
                    .current_dir(&root)
                    .env("PATH", format!("{}:/usr/bin:/bin", root.display()))
                    .env("CONTROL_ROOT", &root)
                    .env("CONTROL_POLICY", policy)
                    .env("CONTROL_LIST_EXIT", list.to_string())
                    .env("CONTROL_RUN_EXIT", run.to_string())
                    .env("CONTROL_DOC_EXIT", doc.to_string())
                    .env("UMPIRE_SCOPED_TEST_PACKAGE", "member")
                    .env("UMPIRE_SCOPED_TEST_EXIT", expected.to_string())
                    .env("UMPIRE_BACKENDS", backend)
                    .env("UMPIRE_APALACHE", apalache)
                    .env("MISE_AUTO_INSTALL", "true")
                    .env("MISE_EXEC_AUTO_INSTALL", "true")
                    .env_remove("NEXTEST_PROFILE")
                    .env_remove("NEXTEST_TEST_THREADS");
                if backend == "require" || apalache == "require" {
                    command.env("NEXTEST_PROFILE", "default").env("NEXTEST_TEST_THREADS", "2");
                }
                let output = command.output()?;
                assert!(output.status.success(), "{output:?}");
                let actual = fs::read_to_string(root.join("calls"))?;
                assert_eq!(actual.lines().count(), calls, "{actual}");
                assert!(actual.contains("--run-ignored default"));
                if calls >= 3 {
                    assert!(actual.contains("--fail-fast"));
                }
            }
        }
        assert!(run_package("").is_err());
        Ok(())
    }

    #[test]
    fn exact_predicates_use_nextest_escape_sequences_without_embedded_quotes() {
        assert_eq!(predicate("package", "xtask"), "package(=xtask)");
        assert_eq!(predicate("test", "case),"), r"test(=case\)\,)");
        assert_eq!(predicate("test", "\\"), r"test(=\\)");
        assert_eq!(predicate("test", "\n\t "), r"test(=\n\t\u{20})");
        assert_eq!(predicate("test", "case.*[one]?"), "test(=case.*[one]?)");
    }

    #[test]
    fn binaries_then_doctests_exactly_once() -> crate::Result<()> {
        let mut calls = Vec::new();
        assert_eq!(
            run_with(|args| {
                calls.push(args.join(" "));
                Ok(0)
            })?,
            0
        );
        assert_eq!(
            calls,
            [
                "nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail",
                "test --doc --workspace --locked --quiet",
            ]
        );
        Ok(())
    }

    #[test]
    fn binary_failure_stops_before_doctests() -> crate::Result<()> {
        let mut calls = 0;
        assert_eq!(
            run_with(|_| {
                calls += 1;
                Ok(42)
            })?,
            42
        );
        assert_eq!(calls, 1);
        Ok(())
    }

    #[test]
    fn doctest_failure_preserves_status() -> crate::Result<()> {
        let mut calls = 0;
        assert_eq!(
            run_with(|_| {
                calls += 1;
                Ok(if calls == 1 { 0 } else { 37 })
            })?,
            37
        );
        assert_eq!(calls, 2);
        Ok(())
    }

    #[test]
    fn selected_packages_share_one_nextest_batch_and_only_eligible_doctests() -> crate::Result<()> {
        let selections = [
            Selection { package: "api", target: None, filter: None, doctests: true },
            Selection { package: "realization", target: None, filter: None, doctests: true },
            Selection { package: "macros", target: None, filter: None, doctests: true },
            Selection { package: "guest", target: None, filter: None, doctests: false },
            Selection {
                package: "temporal-tests",
                target: Some("fixtures"),
                filter: Some("fixtures"),
                doctests: true,
            },
        ];
        let mut calls = Vec::new();
        assert_eq!(
            run_selected_with(&selections, |phase, args| {
                calls.push((phase.to_owned(), args.to_vec()));
                Ok(0)
            })?,
            0
        );
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0].0, "test");
        assert_eq!(calls[0].1.iter().filter(|arg| *arg == "-p").count(), 5);
        let expression = calls[0]
            .1
            .last()
            .ok_or_else(|| crate::Error::Invalid("missing filter expression".into()))?;
        assert_eq!(
            expression,
            concat!(
                "(package(=api)) | (package(=realization)) | ",
                "(package(=macros)) | (package(=guest)) | ",
                "(package(=temporal-tests) & kind(test) & binary(=fixtures) & test(=fixtures))"
            )
        );
        assert!(calls[0].1.windows(2).any(|args| args == ["--no-tests", "fail"]));
        assert!(calls[0].1.windows(2).any(|args| args == ["--run-ignored", "default"]));
        assert!(!calls[0].1.iter().any(|argument| argument == "--profile"));
        assert!(calls[0].1.iter().any(|arg| arg == "--fail-fast"));
        assert_eq!(calls[1].0, "doctest");
        assert_eq!(
            calls[1].1.join(" "),
            "test --doc --locked -p api -p realization -p macros -- --nocapture"
        );
        Ok(())
    }

    #[test]
    fn selected_binary_failure_and_doctest_failure_preserve_status() -> crate::Result<()> {
        let selections = [Selection { package: "api", target: None, filter: None, doctests: true }];
        for (failure, expected_calls) in [(1, 1), (2, 2)] {
            let mut calls = 0;
            assert_eq!(
                run_selected_with(&selections, |_, _| {
                    calls += 1;
                    Ok(if calls == failure { 37 } else { 0 })
                })?,
                37
            );
            assert_eq!(calls, expected_calls);
        }
        Ok(())
    }

    #[test]
    fn selected_empty_batch_and_spawn_failure_do_not_fall_back() {
        assert!(run_selected_with(&[], |_, _| panic!("empty selection executed")).is_err());
        let selections = [Selection { package: "api", target: None, filter: None, doctests: true }];
        let mut calls = 0;
        assert!(
            run_selected_with(&selections, |_, _| {
                calls += 1;
                Err(std::io::Error::new(std::io::ErrorKind::NotFound, "missing nextest").into())
            })
            .is_err()
        );
        assert_eq!(calls, 1);
    }

    #[test]
    fn filtered_doctests_keep_exact_selection() -> crate::Result<()> {
        let selections =
            [Selection { package: "api", target: None, filter: Some("example"), doctests: true }];
        let mut calls = Vec::new();
        run_selected_with(&selections, |_, args| {
            calls.push(args.join(" "));
            Ok(0)
        })?;
        assert_eq!(calls[1], "test --doc --locked -p api example -- --nocapture --exact");
        Ok(())
    }

    fn inventory(rows: &[(&str, &str, &str, &str, bool, &str)]) -> Vec<u8> {
        let suites = rows.iter().enumerate().map(|(index, (package, binary, kind, test, ignored, status))| {
            (index.to_string(), serde_json::json!({
                "package-name": package,
                "binary-name": binary,
                "kind": kind,
                "status": "listed",
                "testcases": { (test.to_string()): {"ignored": ignored, "filter-match": {"status": status}} }
            }))
        }).collect::<std::collections::BTreeMap<_, _>>();
        serde_json::to_vec(&serde_json::json!({"rust-suites": suites})).unwrap_or_default()
    }

    #[test]
    fn inventory_admission_refuses_each_empty_package_and_exact_fixture_selection() {
        let selections = [
            Selection { package: "api", target: None, filter: None, doctests: true },
            Selection {
                package: "temporal-tests",
                target: Some("fixtures"),
                filter: Some("fixtures"),
                doctests: false,
            },
        ];
        let compiler_only = std::collections::BTreeSet::new();
        let api = ("api", "api", "lib", "api_test", false, "matches");
        for fixture in [
            ("other", "fixtures", "test", "fixtures", false, "matches"),
            ("temporal-tests", "other", "test", "fixtures", false, "matches"),
            ("temporal-tests", "fixtures", "lib", "fixtures", false, "matches"),
            ("temporal-tests", "fixtures", "test", "other", false, "matches"),
            ("temporal-tests", "fixtures", "test", "fixtures", true, "matches"),
            ("temporal-tests", "fixtures", "test", "fixtures", false, "mismatch"),
        ] {
            assert!(
                admit_inventory(&selections, &compiler_only, &inventory(&[api, fixture])).is_err()
            );
        }
        assert!(admit_inventory(&selections, &compiler_only, &inventory(&[api])).is_err());
        assert!(admit_inventory(&selections, &compiler_only, b"{}").is_err());
        assert!(admit_inventory(&selections, &compiler_only, b"truncated").is_err());
        let fixture = ("temporal-tests", "fixtures", "test", "fixtures", false, "matches");
        assert!(admit_inventory(&selections, &compiler_only, &inventory(&[fixture])).is_err());
        assert!(admit_inventory(&selections, &compiler_only, &inventory(&[api, fixture])).is_ok());
    }

    #[test]
    fn inventory_failure_preserves_status_and_never_executes_the_batch() -> crate::Result<()> {
        let selections = [Selection { package: "api", target: None, filter: None, doctests: true }];
        let compiler_only = std::collections::BTreeSet::new();
        assert_eq!(
            run_selected_checked_with(
                &selections,
                &compiler_only,
                |_| Ok((29, Vec::new())),
                |_, _| panic!("failed inventory executed")
            )?,
            29
        );
        assert!(
            run_selected_checked_with(
                &selections,
                &compiler_only,
                |_| Ok((0, inventory(&[]))),
                |_, _| panic!("empty package executed")
            )
            .is_err()
        );
        Ok(())
    }

    #[test]
    fn compiler_only_admission_is_explicit_and_keeps_discovered_tests_in_the_batch()
    -> crate::Result<()> {
        let selections = [
            Selection { package: "api", target: None, filter: None, doctests: true },
            Selection { package: "guest", target: None, filter: None, doctests: false },
        ];
        let compiler_only = std::collections::BTreeSet::from(["guest".into()]);
        let api = ("api", "api", "lib", "api_test", false, "matches");
        assert!(
            admit_inventory(&selections, &std::collections::BTreeSet::new(), &inventory(&[api]))
                .is_err()
        );
        assert!(admit_inventory(&selections, &compiler_only, &inventory(&[api])).is_ok());
        let mut calls = Vec::new();
        run_selected_checked_with(
            &selections,
            &compiler_only,
            |arguments| {
                assert_eq!(arguments[1], "list");
                assert!(arguments.windows(2).any(|pair| pair == ["-p", "guest"]));
                Ok((0, inventory(&[api, ("guest", "guest", "lib", "new_test", false, "matches")])))
            },
            |phase, arguments| {
                calls.push((phase.to_owned(), arguments.to_vec()));
                Ok(0)
            },
        )?;
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0].1[1], "run");
        assert!(calls[0].1.windows(2).any(|pair| pair == ["-p", "guest"]));
        Ok(())
    }

    #[test]
    fn explicit_compiler_only_batches_do_not_claim_zero_test_execution() -> crate::Result<()> {
        let selections =
            [Selection { package: "guest", target: None, filter: None, doctests: false }];
        let compiler_only = std::collections::BTreeSet::from(["guest".into()]);
        assert_eq!(
            run_selected_checked_with(
                &selections,
                &compiler_only,
                |_| Ok((0, inventory(&[]))),
                |_, _| panic!("compile-only selection claimed test execution"),
            )?,
            0
        );
        assert!(
            run_selected_checked_with(
                &selections,
                &std::collections::BTreeSet::new(),
                |_| Ok((0, inventory(&[]))),
                |_, _| panic!("unexpected empty selection executed"),
            )
            .is_err()
        );
        Ok(())
    }

    #[test]
    fn missing_runner_fails_without_fallback() {
        let mut calls = 0;
        let result = run_with(|_| {
            calls += 1;
            Err(std::io::Error::new(std::io::ErrorKind::NotFound, "missing runner").into())
        });
        assert!(result.is_err());
        assert_eq!(calls, 1);
    }
}
