//! Temporal stages of the combined Model gate.
use std::os::unix::ffi::OsStringExt;
use std::{
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    path::Path,
    process::Command,
};
use xtask::gate::{
    count_dispatch::{Contract, NoWorkReason},
    work_counts,
};

/// The reviewed Temporal stage budget table, relative to the Model root.
const TEMPORAL_BUDGETS: &str = "dev/xtask/src/gate/budgets.toml";

pub(crate) mod timing;
#[cfg(test)]
pub(crate) mod timing_controls;

fn admit_command(name: &str, command: &mut Command) -> io::Result<()> {
    work_counts::configure_scope(command);
    let caller = std::env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    command
        .env("CARGO_HOME", home)
        .env("UMPIRE_QUICK_CACHE", "0")
        .env("UMPIRE_CACHE_DISABLE", "1")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("MISE_TASK_PGID_MANAGED", "1");
    if name == "tests" || name == "source-boundary" {
        crate::test_run::configure(
            command,
            std::env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require"),
            std::env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
        );
    }

    if let Some(target) = std::env::var_os("CARGO_TARGET_DIR") {
        let target = std::path::PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    xtask::gate::admission::configure(
        command,
        std::env::var_os("UMPIRE_BACKENDS").is_some_and(|v| v == "require")
            || std::env::var_os("UMPIRE_APALACHE").is_some_and(|v| v == "require"),
    );
    Ok(())
}

pub(crate) fn stage(root: &Path, name: &str, command: Command) -> io::Result<i32> {
    stage_with_inventory(root, name, command, None)
}

fn stage_with_inventory(
    root: &Path,
    name: &str,
    mut command: Command,
    inventory: Option<&[u8]>,
) -> io::Result<i32> {
    xtask::gate::source_context::verify_consumer(root)?;
    admit_command(name, &mut command)?;
    command.current_dir(root);
    let started = std::time::Instant::now();
    let observed = if work_counts::owns_controller()? {
        work_counts::observe_streamed(&command, inventory, true, true)?
    } else {
        None
    };
    let output = if let Some(output) = observed {
        io::stdout().write_all(&output.stdout)?;
        io::stderr().write_all(&output.stderr)?;
        Ok(output)
    } else {
        xtask::gate::timing::capture(
            &command,
            std::env::var_os("UMPIRE_TIME_STEPS").is_some_and(|value| value == "1"),
            inventory,
        )
    }
    .map_err(|error| io::Error::other(format!("Temporal stage {name}: {error}")))?;
    let code = output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0));
    if code == 0 && std::env::var_os("UMPIRE_TIME_STEPS").is_some_and(|value| value == "1") {
        writeln!(io::stdout(), "{name:<32} {:6.3}s", started.elapsed().as_secs_f64())?;
    }
    if code != 0 {
        let _ = writeln!(io::stderr(), "Temporal stage {name} failed (exit {code})");
    }
    Ok(code)
}

/// `cargo` for owned argument lists, choosing the pinned nextest runner the same way.
fn cargo_owned(arguments: &[String]) -> Command {
    cargo(&arguments.iter().map(String::as_str).collect::<Vec<_>>())
}

pub(crate) fn cargo(arguments: &[&str]) -> Command {
    // Nextest runs as the mise-pinned `cargo-nextest` itself: `cargo nextest` would resolve
    // `$CARGO_HOME/bin` first, where a stale host install could shadow the pin.
    let program = if arguments.first() == Some(&"nextest") { "cargo-nextest" } else { "cargo" };
    let mut command = Command::new("mise");
    command.args(["exec", "--", program]).args(arguments);
    command
}

use xtask::gate::scheduler;

fn mise(root: &Path) -> io::Result<Command> {
    let caller = std::env::current_dir()?;
    let home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    let mut command = Command::new("mise");
    command
        .current_dir(root.parent().ok_or_else(|| io::Error::other("missing repository root"))?)
        .env("CARGO_HOME", home)
        .env("UMPIRE_FRAMEWORK_ROOT", root.join("../../umpire"))
        .env("UMPIRE_TEMPORAL_ROOT", root)
        .env("UMPIRE_QUICK_CACHE", "0")
        .env("UMPIRE_CACHE_DISABLE", "1")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("MISE_TASK_PGID_MANAGED", "1");
    if let Some(target) = std::env::var_os("CARGO_TARGET_DIR") {
        let target = std::path::PathBuf::from(target);
        command.env(
            "CARGO_TARGET_DIR",
            if target.is_absolute() { target } else { caller.join(target) },
        );
    }
    admit_command("graph", &mut command)?;
    Ok(command)
}

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    let bootstrap = framework_executable(root)?;
    if !bootstrap.status.success() {
        writeln!(
            io::stderr(),
            "framework bootstrap failed (exit {})",
            bootstrap.status.code().unwrap_or(1)
        )?;
        return Ok(bootstrap.status.code().unwrap_or(1));
    }
    let framework = std::ffi::OsString::from_vec(
        bootstrap.stdout.strip_suffix(b"\n").unwrap_or(&bootstrap.stdout).to_owned(),
    );
    if std::env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require") {
        let mut command = Command::new(&framework);
        command.arg("fetch-apalache").arg(root.join("../../umpire"));
        let code = stage(root, "apalache", command)?;
        if code != 0 {
            return Ok(code);
        }
    }
    let expected = scheduler::inventory(mise(root)?, "")?;
    let caller = std::env::current_dir()?;
    let target = std::env::var_os("CARGO_TARGET_DIR").map_or_else(
        || root.join("target"),
        |path| {
            let path = std::path::PathBuf::from(path);
            if path.is_absolute() { path } else { caller.join(path) }
        },
    );
    let records = scheduler::directory(&target.join("gate-receipts"))?;
    let framework = framework_stage_executable(root, &target, framework)?;
    let mut command = mise(root)?;
    command
        .env("UMPIRE_TEMPORAL_GATE_EXECUTABLE", std::env::current_exe()?)
        .env("UMPIRE_GATE_CONTROLLER_EXECUTABLE", std::env::current_exe()?)
        .env("UMPIRE_GATE_EXECUTABLE", &framework);
    // Delegated framework stages are budgeted by the framework's own table.
    let mut tables = Vec::new();
    if expected.iter().any(|stage| stage.starts_with("framework/")) {
        tables.push(xtask::gate::budgets::read_table(
            &root.join("../../umpire").join(xtask::gate::budgets::FRAMEWORK_TABLE_PATH),
        )?);
    }
    tables.push(xtask::gate::budgets::read_table(&root.join(TEMPORAL_BUDGETS))?);
    // The tier follows the admitted resource profile, which Apalache alone also selects.
    let tier = if std::env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require")
        || std::env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require")
    {
        xtask::gate::budgets::Tier::Validate
    } else {
        xtask::gate::budgets::Tier::Check
    };
    let result = scheduler::run(
        command,
        "model:all",
        &records,
        &expected,
        &tables.iter().map(String::as_str).collect::<Vec<_>>(),
        Some(tier),
    );
    if umpire_guest_driver::managed::enabled() {
        remove_if_present(Path::new(&framework))?;
    }
    result
}

fn remove_if_present(path: &Path) -> io::Result<()> {
    match std::fs::remove_file(path) {
        Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(()),
        result => result,
    }
}

/// A managed framework gate refuses Temporal's launcher target, so its stages re-enter the
/// launcher for the sibling framework root and receive that root's target and lease.
fn framework_stage_executable(
    root: &Path,
    target: &Path,
    framework: std::ffi::OsString,
) -> io::Result<std::ffi::OsString> {
    use std::os::unix::fs::PermissionsExt;
    if !umpire_guest_driver::managed::enabled() {
        return Ok(framework);
    }
    let quote = |value: &std::ffi::OsStr| {
        let value = value
            .to_str()
            .ok_or_else(|| io::Error::other("managed framework stage path is not UTF-8"))?;
        Ok::<_, io::Error>(format!("'{}'", value.replace('\'', r"'\''")))
    };
    let required = |name: &str| {
        std::env::var_os(name)
            .filter(|value| !value.is_empty())
            .ok_or_else(|| io::Error::other(format!("managed framework stage requires {name}")))
    };
    let workspace = root.join("../../umpire").canonicalize()?;
    let script = format!(
        "#!/bin/bash\nexport UMPIRE_GATE_CONTROLLER_EXECUTABLE={}\nexec {} exec --project {} --workspace {} -- {} \"$@\"\n",
        quote(&framework)?,
        quote(&required("BUILD_CACHE_TOOL")?)?,
        quote(&required("BUILD_CACHE_PROJECT")?)?,
        quote(workspace.as_os_str())?,
        quote(&framework)?,
    );
    let directory = target.join("framework-stages");
    std::fs::create_dir_all(&directory)?;
    let executable = directory.join(format!("gate-{}", std::process::id()));
    // Never follow a leftover entry, including a dangling symlink.
    remove_if_present(&executable)?;
    std::fs::write(&executable, script)?;
    std::fs::set_permissions(&executable, std::fs::Permissions::from_mode(0o555))?;
    Ok(executable.into_os_string())
}

pub(crate) fn framework_executable(root: &Path) -> io::Result<std::process::Output> {
    let mut command = Command::new("/bin/bash");
    command
        .arg(root.join("scripts/xtask.sh"))
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("MISE_TASK_PGID_MANAGED", "1");
    admit_command("bootstrap", &mut command)?;
    let output = scheduler::execute(&command, false, false)?;
    io::stderr().write_all(&output.stderr)?;
    Ok(output)
}

pub(crate) fn list(root: &Path) -> io::Result<i32> {
    for (name, deps) in scheduler::listing(mise(root)?, "")? {
        writeln!(io::stdout(), "{name}\t{}", deps.join(","))?;
    }
    Ok(0)
}

pub(crate) fn scheduled_stage(root: &Path, name: &str) -> io::Result<i32> {
    scheduler::scheduled(name, || run_stage(root, name))
}

pub(crate) fn control_tests(root: &Path, name: &str) -> io::Result<i32> {
    let filters: &[&str] = match name {
        "timing-controls" => &[
            "gate::timing::interrupt_controls::interrupted_benchmark_preserves_original_source_bytes",
            "gate::timing_controls::tests::soft_budget_defaults_equality_overrides_and_warning_order",
            "model_controls::shipping_model_recipes_with_disposable_children",
            "guest_routes::tests::seven_packages_two_entries_preserve_caller_paths_and_exact_arguments",
            "guest_routes::tests::non_unicode_output_and_target_are_lossless",
            "guest_routes::tests::raw_arguments_check_output_and_failure_status_survive_execution",
            "build_guests::path_controls::relative_absolute_default_and_symlink_paths_preserve_temporal_guest",
        ],
        "proto-controls" => &[
            "proto::bridge_controls::temporal_spec_composes_with_native_generation_freshness_and_refusals",
            "proto::tests::raw_repository_and_tool_paths_remain_lossless",
            "proto::tests::only_temporal_and_optional_check_are_admitted_before_execution",
            "proto::tests::temporal_inputs_subtree_and_repository_plugin_preserve_raw_paths_and_status",
        ],
        _ => return Err(io::Error::other(format!("unknown native control family: {name}"))),
    };
    let selections: Vec<_> = filters
        .iter()
        .map(|filter| xtask::test_run::Selection {
            package: "temporal-xtask",
            target: None,
            filter: Some(filter),
            doctests: false,
        })
        .collect();
    let command = |arguments: &[String]| {
        let mut command = cargo_owned(arguments);
        command.current_dir(root);
        admit_command("tests", &mut command)?;
        Ok::<_, io::Error>(command)
    };
    xtask::test_run::run_selected_observed_with(
        &selections,
        &std::collections::BTreeSet::new(),
        |arguments| {
            let output = scheduler::execute(&command(arguments)?, false, false)?;
            io::stderr().write_all(&output.stderr)?;
            Ok((
                output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)),
                output.stdout,
            ))
        },
        |_, arguments, inventory| {
            stage_with_inventory(root, name, command(arguments)?, inventory).map_err(Into::into)
        },
    )
    .map_err(io::Error::other)
}

pub(crate) fn run_stage(root: &Path, name: &str) -> io::Result<i32> {
    run_stage_with_tool(root, name, &std::env::current_exe()?)
}

#[expect(clippy::too_many_lines, reason = "One dispatch arm per scheduled Temporal stage")]
fn run_stage_with_tool(root: &Path, name: &str, tool: &Path) -> io::Result<i32> {
    let native = |arguments: &[&str]| {
        let mut command = Command::new(tool);
        command.args(arguments);
        command
    };
    match name {
        "temporal-prerequisites" => {
            Contract::not_applicable(NoWorkReason::ToolProvisioning).run(|| {
                let command = {
                    let mut command = Command::new("mise");
                    command
                        .args(["exec", "--", "make", "-C"])
                        .arg(
                            root.parent()
                                .ok_or_else(|| io::Error::other("missing repository root"))?,
                        )
                        .args(["-j1", "model-prerequisites"])
                        .env_remove("MAKEFLAGS")
                        .env_remove("MFLAGS");
                    command
                };
                stage(root, name, command)
            })
        }
        "temporal-guest-freshness" => {
            Contract::not_applicable(NoWorkReason::GuestBuild).run(|| {
                let command = {
                    let mut command = native(&["guest-freshness", "umpire-temporal-guest"]);
                    command.arg(root.join("hosts/go/umpire_temporal_guest.wasm"));
                    command
                };
                stage(root, name, command)
            })
        }
        "retired-names" => Contract::not_applicable(NoWorkReason::NativeScan).run(|| {
            let command = native(&["retired-names"]);
            stage(root, name, command)
        }),
        "timing-controls" => Contract::observed().run(|| {
            let command = native(&["timing-controls"]);
            stage(root, name, command)
        }),
        "source-boundary" => Contract::observed().run(|| {
            use std::collections::BTreeSet;
            let selection = xtask::test_run::Selection {
                package: "umpire-temporal-tests",
                target: Some("boundary"),
                filter: None,
                doctests: false,
            };
            xtask::test_run::run_selected_observed_with(
                &[selection],
                &BTreeSet::new(),
                |args| {
                    let mut command = cargo_owned(args);
                    command.current_dir(root);
                    admit_command("source-boundary", &mut command)?;
                    let output = scheduler::execute(&command, false, false)?;
                    io::stderr().write_all(&output.stderr)?;
                    let code = output.status.code().unwrap_or(1);
                    if code != 0 {
                        writeln!(
                            io::stderr(),
                            "Temporal stage source-boundary failed (exit {code})"
                        )?;
                    }
                    Ok((code, output.stdout))
                },
                |_, args, inventory| {
                    let command = cargo_owned(args);
                    stage_with_inventory(root, "source-boundary", command, inventory)
                        .map_err(Into::into)
                },
            )
            .map_err(io::Error::other)
        }),
        "protobuf-controls" => Contract::observed().run(|| {
            let command = native(&["proto-controls"]);
            stage(root, name, command)
        }),
        "temporal-protobuf-freshness" => Contract::not_applicable(NoWorkReason::ProtobufGeneration)
            .run(|| {
                let command = native(&["proto-go", "temporal", "--check"]);
                stage(root, name, command)
            }),
        "fmt" => Contract::not_applicable(NoWorkReason::Formatting).run(|| {
            let command = cargo(&["fmt", "--all", "--check"]);
            stage(root, name, command)
        }),
        "clippy" => Contract::not_applicable(NoWorkReason::CompileOnly).run(|| {
            let command = cargo(&[
                "clippy",
                "--workspace",
                "--all-targets",
                "--all-features",
                "--",
                "-D",
                "warnings",
            ]);
            stage(root, name, command)
        }),
        "docs" => Contract::not_applicable(NoWorkReason::CompileOnly).run(|| {
            let command = {
                let mut command = cargo(&["doc", "--workspace", "--no-deps", "--quiet"]);
                command.env("RUSTDOCFLAGS", "-D warnings");
                command
            };
            stage(root, name, command)
        }),
        "tests" => Contract::observed().run(|| crate::test_run::run(root)),
        "wasm-guest" => Contract::not_applicable(NoWorkReason::GuestBuild).run(|| {
            let command = native(&["build-guests"]);
            stage(root, name, command)
        }),
        "temporal-go-host" => Contract::observed().run(|| {
            let command = {
                let mut command = Command::new("mise");
                command
                    .args(["exec", "--", "go", "-C"])
                    .arg(root.join("hosts/go"))
                    .args(["test", "-tags", "test_dep", "-count=1", "./..."])
                    .env("CGO_ENABLED", "0");
                command
            };
            stage(root, name, command)
        }),
        _ => Err(io::Error::other(format!("unknown Temporal stage: {name}"))),
    }
}

#[cfg(test)]
#[allow(clippy::unwrap_used, clippy::expect_used)]
mod tests {
    use super::*;

    #[test]
    fn managed_framework_wrapper_controller_child() {
        if std::env::var_os("TEMPORAL_WRAPPED_CONTROLLER_CHILD").is_some() {
            assert!(work_counts::owns_controller().unwrap());
        }
    }

    #[test]
    #[ignore = "explicit real managed-wrapper admission probe; no gate or build children"]
    fn real_managed_framework_wrapper_retains_controller_identity() {
        assert!(umpire_guest_driver::managed::enabled());
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize().unwrap();
        let target = std::path::PathBuf::from(std::env::var_os("CARGO_TARGET_DIR").unwrap());
        let wrapper = framework_stage_executable(
            &root,
            &target,
            std::env::current_exe().unwrap().into_os_string(),
        )
        .unwrap();
        let output = Command::new(&wrapper)
            .args([
                "--exact",
                "gate::tests::managed_framework_wrapper_controller_child",
                "--nocapture",
            ])
            .env("TEMPORAL_WRAPPED_CONTROLLER_CHILD", "1")
            .env("UMPIRE_GATE_EXECUTABLE", &wrapper)
            .env("UMPIRE_GATE_CONTROLLER_EXECUTABLE", "/bin/false")
            .output()
            .unwrap();
        std::fs::remove_file(wrapper).unwrap();
        assert!(output.status.success(), "{output:?}");
        io::stderr().write_all(&output.stderr).unwrap();
    }

    #[test]
    fn managed_framework_wrapper_selects_native_controller_identity() {
        use std::os::unix::fs::PermissionsExt;
        if let Some(root) = std::env::var_os("TEMPORAL_WRAPPER_CONTROLLER_CONTROL") {
            let root = std::path::PathBuf::from(root);
            let executable = std::env::current_exe().unwrap();
            let wrapper = framework_stage_executable(
                &root,
                &root.join("target"),
                executable.into_os_string(),
            )
            .unwrap();
            let output = Command::new(wrapper)
                .args([
                    "--exact",
                    "gate::tests::managed_framework_wrapper_controller_child",
                    "--nocapture",
                ])
                .env("TEMPORAL_WRAPPED_CONTROLLER_CHILD", "1")
                .env("UMPIRE_GATE_EXECUTABLE", "/bin/false")
                .env("UMPIRE_GATE_CONTROLLER_EXECUTABLE", "/bin/false")
                .output()
                .unwrap();
            assert!(output.status.success(), "{output:?}");
            return;
        }
        let root = control_directory();
        let launcher = root.join("bin/build-cache");
        std::fs::write(
            &launcher,
            "#!/bin/bash\nset -eu\n[ \"$1\" = exec ]\n[ \"$6\" = -- ]\nshift 6\nexec \"$@\"\n",
        )
        .unwrap();
        std::fs::set_permissions(&launcher, std::fs::Permissions::from_mode(0o755)).unwrap();
        let output = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                "gate::tests::managed_framework_wrapper_selects_native_controller_identity",
                "--nocapture",
            ])
            .env("TEMPORAL_WRAPPER_CONTROLLER_CONTROL", &root)
            .env("BUILD_CACHE_MANAGED", "1")
            .env("BUILD_CACHE_TOOL", launcher)
            .env("BUILD_CACHE_PROJECT", "umpire")
            .output()
            .unwrap();
        assert!(output.status.success(), "{output:?}");
        std::fs::remove_dir_all(root.parent().unwrap().parent().unwrap()).unwrap();
    }

    #[test]
    fn committed_temporal_budgets_cover_exactly_the_registered_temporal_stages() {
        let config: toml::Value = toml::from_str(include_str!("../../../../mise.toml")).unwrap();
        let registered = config["tasks"]
            .as_table()
            .unwrap()
            .values()
            .filter(|task| {
                task.get("env").and_then(|env| env.get("UMPIRE_GATE_SCOPE"))
                    == Some(&toml::Value::String("temporal".into()))
            })
            .map(|task| format!("temporal/{}", task["env"]["UMPIRE_GATE_STAGE"].as_str().unwrap()))
            .collect::<Vec<_>>();
        assert_eq!(registered.len(), 13);
        let table = include_str!("gate/budgets.toml");
        xtask::gate::budgets::validate(table, &registered).unwrap();
        let mut planted = registered.clone();
        planted.push("temporal/planted-stage".into());
        let error = xtask::gate::budgets::validate(table, &planted).unwrap_err().to_string();
        assert!(error.contains("temporal/planted-stage"), "{error}");
    }

    #[test]
    fn paired_inventory_admits_both_committed_tables_without_a_default_scope() {
        /// Removes its receipt directory even when an assertion fails first.
        struct Receipts(std::path::PathBuf);
        impl Drop for Receipts {
            fn drop(&mut self) {
                let _ = std::fs::remove_dir_all(&self.0);
            }
        }
        let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
        let framework = manifest.join("../../../../umpire");
        let graph: toml::Value = toml::from_str(
            &std::fs::read_to_string(framework.join("tools/xtask/src/gate/tasks.toml")).unwrap(),
        )
        .unwrap();
        let config: toml::Value = toml::from_str(include_str!("../../../../mise.toml")).unwrap();
        let mut inventory = Vec::new();
        for (tasks, scope) in [(&graph, "framework"), (&config["tasks"], "temporal")] {
            for task in tasks.as_table().unwrap().values() {
                let env = task.get("env");
                if env.and_then(|env| env.get("UMPIRE_GATE_SCOPE")).and_then(toml::Value::as_str)
                    == Some(scope)
                {
                    inventory.push(format!(
                        "{scope}/{}",
                        env.unwrap()["UMPIRE_GATE_STAGE"].as_str().unwrap()
                    ));
                }
            }
        }
        assert_eq!(inventory.len(), 31);
        let tables = [
            xtask::gate::budgets::read_table(
                &framework.join(xtask::gate::budgets::FRAMEWORK_TABLE_PATH),
            )
            .unwrap(),
            include_str!("gate/budgets.toml").to_owned(),
        ];
        let receipts = |label: &str| {
            let path = std::env::temp_dir().join(format!(
                "temporal-budget-{label}-{}-{}",
                std::process::id(),
                std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap()
                    .as_nanos()
            ));
            std::fs::create_dir(&path).unwrap();
            Receipts(path)
        };
        let paired = receipts("paired");
        xtask::gate::budget_runtime::admit(
            &paired.0,
            &inventory,
            &[&tables[0], &tables[1]],
            Some(xtask::gate::budgets::Tier::Validate),
        )
        .unwrap();
        // Without a default scope, an unscoped stage is refused by name.
        drop(paired);
        let unscoped = receipts("unscoped");
        inventory.push("unscoped".into());
        let error = xtask::gate::budget_runtime::admit(
            &unscoped.0,
            &inventory,
            &[&tables[0], &tables[1]],
            Some(xtask::gate::budgets::Tier::Validate),
        )
        .unwrap_err()
        .to_string();
        assert!(error.contains("unscoped"), "{error}");
    }

    #[test]
    fn paired_gate_waits_for_primary_framework_without_implicit_isolation() {
        let config: toml::Value = toml::from_str(include_str!("../../../../mise.toml")).unwrap();
        let dependencies = config["tasks"]["model:temporal-go-host"]["depends"].as_array().unwrap();
        assert!(dependencies.iter().any(|value| value.as_str() == Some("gate:all")));
        assert!(!include_str!("../../../../mise.toml").contains("gate:standalone"));
    }

    #[test]
    fn full_gate_overrides_cache_and_preserves_failure() {
        let mut command = Command::new("/bin/bash");
        command.args(["-c", "test \"$UMPIRE_QUICK_CACHE\" = 0 && test \"$UMPIRE_CACHE_DISABLE\" = 1 || exit 9; exit 37"])
            .env("UMPIRE_QUICK_CACHE", "1").env("UMPIRE_CACHE_DISABLE", "0");
        assert_eq!(
            stage(&Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."), "control", command)
                .unwrap(),
            37
        );
    }

    #[test]
    fn test_policy_control_child() {
        let Ok(expected) = std::env::var("TEMPORAL_EXPECT_TEST_POLICY") else { return };
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "test \"${NEXTEST_PROFILE:-unset}:${NEXTEST_TEST_THREADS:-unset}\" = \"$TEMPORAL_EXPECT_TEST_POLICY\""]);
        assert_eq!(stage(Path::new("/"), "tests", command).unwrap(), 0, "{expected}");
    }

    #[test]
    fn test_stage_propagates_required_policy_and_preserves_skipped_defaults() {
        for (backend, apalache, inherited, expected) in [
            ("skip", "skip", None, "unset:unset"),
            ("skip", "skip", Some("2"), "unset:2"),
            ("require", "skip", None, "required:4"),
            ("require", "skip", Some("99"), "required:4"),
            ("require", "skip", Some("2"), "required:2"),
            ("require", "skip", Some("1"), "required:1"),
            ("require", "skip", Some("malformed"), "required:4"),
            ("require", "skip", Some("0"), "required:4"),
            ("skip", "require", Some("2"), "required:1"),
            ("require", "require", Some("99"), "required:1"),
        ] {
            let mut child = Command::new(std::env::current_exe().unwrap());
            child
                .args(["--exact", "gate::tests::test_policy_control_child", "--nocapture"])
                .env("TEMPORAL_EXPECT_TEST_POLICY", expected)
                .env("UMPIRE_BACKENDS", backend)
                .env("UMPIRE_APALACHE", apalache)
                .env_remove("NEXTEST_PROFILE")
                .env_remove("NEXTEST_TEST_THREADS");
            if backend == "require" || apalache == "require" {
                child.env("NEXTEST_PROFILE", "default");
            }
            if let Some(inherited) = inherited {
                child.env("NEXTEST_TEST_THREADS", inherited);
            }
            let output = child.output().unwrap();
            assert!(output.status.success(), "{output:?}");
        }
    }

    #[test]
    fn missing_tool_control_child() {
        if std::env::var_os("TEMPORAL_MISSING_TOOL_CONTROL").is_none() {
            return;
        }
        std::process::exit(
            stage(Path::new("/"), "missing-tool", Command::new("/missing/temporal-tool")).unwrap(),
        );
    }

    #[test]
    fn missing_tool_reports_the_failed_stage() {
        let output = Command::new(std::env::current_exe().unwrap())
            .args(["--exact", "gate::tests::missing_tool_control_child", "--nocapture"])
            .env("TEMPORAL_MISSING_TOOL_CONTROL", "1")
            .output()
            .unwrap();
        assert_eq!(output.status.code(), Some(1));
        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(stderr.contains("cannot execute backend subprocess"));
        assert!(stderr.contains("Temporal stage missing-tool failed (exit 1)"));
    }

    #[test]
    fn invalid_working_directory_names_the_stage() {
        let missing = std::env::temp_dir().join(format!(
            "temporal-missing-cwd-{}-{}",
            std::process::id(),
            std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos()
        ));
        assert!(!missing.exists());
        let error = stage(&missing, "missing-cwd", Command::new("/bin/sh")).unwrap_err();
        assert!(error.to_string().contains("missing-cwd"));
    }

    #[test]
    fn run_control_child() {
        let Some(root) = std::env::var_os("UMPIRE_GATE_CONTROL_ROOT") else { return };
        if let Some(name) = std::env::var_os("UMPIRE_GATE_CONTROL_STAGE") {
            let name = name.to_str().unwrap();
            let root = Path::new(&root);
            let tool = root.join("bin/native");
            let status =
                scheduler::scheduled(name, || run_stage_with_tool(root, name, &tool)).unwrap();
            std::process::exit(status);
        }
        let status = run(Path::new(&root)).unwrap();
        match std::env::var("UMPIRE_GATE_CONTROL_STATUS").unwrap().as_str() {
            "nonzero" => assert_ne!(status, 0),
            expected => assert_eq!(status, expected.parse::<i32>().unwrap()),
        }
    }

    const CONTROL_BODY: &str = r#"
[ "$UMPIRE_QUICK_CACHE:$UMPIRE_CACHE_DISABLE" = 0:1 ] || exit 9
[ "$CARGO_HOME" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-home" ] || exit 8
[ "$CARGO_TARGET_DIR" = "$UMPIRE_GATE_CONTROL_CALLER/control-relative-target" ] || exit 7
printf '%s\n' "$label" >> "$UMPIRE_GATE_CONTROL_LOG"
[ "$label" != "${UMPIRE_GATE_CONTROL_FAILURE:-}" ] || exit 37
case " $* " in
 *' cargo-nextest nextest run '*' --message-format '*) printf '%s\n' '{"type":"test","event":"ok","name":"umpire-temporal-tests::boundary$control"}' ;;
 *' cargo test --doc '*) printf '%s\n' 'running 0 tests' 'test result: ok. 0 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.1s' ;;
 *' go -C '*' test -json -list . '*) ;;
 *' go -C '*' test -json '*) printf '%s\n' '{"Action":"start","Package":"control"}' '{"Action":"run","Package":"control","Test":"TestControl"}' '{"Action":"pass","Package":"control","Test":"TestControl"}' '{"Action":"pass","Package":"control"}' ;;
esac

"#;

    const CONTROL_STAGES: [(&str, &str); 17] = [
        ("temporal-prerequisites", "prerequisites"),
        ("temporal-guest-freshness", "guest"),
        ("retired-names", "retired"),
        ("timing-controls", "timing"),
        ("source-boundary", "boundary-inventory"),
        ("source-boundary", "boundary-tests"),
        ("protobuf-controls", "proto-controls"),
        ("temporal-protobuf-freshness", "proto"),
        ("fmt", "exec -- cargo fmt --all --check"),
        ("clippy", "exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings"),
        ("docs", "exec -- cargo doc --workspace --no-deps --quiet"),
        (
            "tests",
            "exec -- cargo-nextest nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final --no-tests fail",
        ),
        ("tests", "exec -- cargo test --doc --workspace --locked --quiet"),
        ("wasm-guest", "guest-build"),
        ("temporal-go-host", "go-environment"),
        ("temporal-go-host", "go-inventory"),
        (
            "temporal-go-host",
            "exec -- go -C PLACEHOLDER/hosts/go test -json -tags test_dep -count=1 ./...",
        ),
    ];

    /// Write the committed Temporal budgets for exactly these fixture stages, with bounds
    /// that admit the synthetic leaves, and return their mise inventory.
    fn control_inventory(root: &Path, stage_names: &[&str]) -> String {
        let committed = include_str!("gate/budgets.toml");
        let (header, sections) = committed.split_once("\n[stages.").unwrap();
        let mut table = header.to_owned();
        for section in sections.split("\n[stages.") {
            let name = section.split('"').nth(1).unwrap().trim_start_matches("temporal/");
            if !stage_names.contains(&name) {
                continue;
            }
            table.push_str("\n[stages.");
            for line in section.lines() {
                let relaxed = match line.split_once(" = ") {
                    Some(("warm_seconds", _)) => {
                        "warm_seconds = { check = 600.0, validate = 600.0 }".to_owned()
                    }
                    // The synthetic control leaves run without test events.
                    Some((key, _))
                        if ["timing-controls", "protobuf-controls"].contains(&name)
                            && ["discovered", "executed", "ignored", "skipped"].contains(&key) =>
                    {
                        format!("{key} = \"unobserved\"")
                    }
                    Some((key, value)) if key != "reason" && value.parse::<u64>().is_ok() => {
                        format!("{key} = 100000")
                    }
                    _ => line.to_owned(),
                };
                table.push_str(&relaxed);
                table.push('\n');
            }
        }
        let path = root.join(TEMPORAL_BUDGETS);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(path, table).unwrap();
        serde_json::to_string(
            &stage_names
                .iter()
                .enumerate()
                .map(|(order, name)| {
                    serde_json::json!({"name": format!("model:{name}"), "env": [
                        format!("UMPIRE_GATE_STAGE={name}"), format!("UMPIRE_GATE_ORDER={order}")
                    ], "depends": []})
                })
                .collect::<Vec<_>>(),
        )
        .unwrap()
    }

    fn control_directory() -> std::path::PathBuf {
        use std::{fs, os::unix::fs::PermissionsExt};
        let root = std::env::temp_dir().join(format!(
            "temporal-gate-{}-{}",
            std::process::id(),
            std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos()
        ));
        let root = root.join("temporal/model");
        fs::create_dir_all(root.join("scripts")).unwrap();
        fs::create_dir_all(root.join("bin")).unwrap();
        fs::create_dir_all(root.join("../../umpire")).unwrap();
        let root = root.canonicalize().unwrap();
        fs::write(root.join("scripts/xtask.sh"), "#!/bin/bash\nprintf '/bin/true\\n'\n").unwrap();
        let body = CONTROL_BODY;
        fs::write(
            root.join("bin/native"),
            format!(r#"#!/bin/bash
set -eu
[ "$PWD" = "$UMPIRE_GATE_CONTROL_ROOT" ] || exit 90
[ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL:$MISE_TASK_PGID_MANAGED" = false:false:1 ] || exit 91
case "$*" in
 'guest-freshness umpire-temporal-guest '*)
   [ "$#" = 3 ] && [ "$3" = "$UMPIRE_GATE_CONTROL_ROOT/hosts/go/umpire_temporal_guest.wasm" ] || exit 92
   label=guest ;;
 'retired-names') label=retired ;;
 'timing-controls') label=timing ;;
 'proto-controls') label=proto-controls ;;
 'proto-go temporal --check') label=proto ;;
 'build-guests') label=guest-build ;;
 *) exit 93 ;;
esac
{body}
"#),
        ).unwrap();
        fs::set_permissions(root.join("bin/native"), fs::Permissions::from_mode(0o755)).unwrap();
        root
    }

    fn control_mise_script(scheduler: &str) -> String {
        let body = CONTROL_BODY;
        let nextest = r#"
case "$*" in
 'exec -- make -C '*' -j1 model-prerequisites')
   [ "$5" = "$(dirname "$UMPIRE_GATE_CONTROL_ROOT")" ] || exit 92
   [ -z "${MAKEFLAGS+x}${MFLAGS+x}" ] || exit 93
   label=prerequisites ;;
 'exec -- cargo-nextest nextest list'*)
   case " $* " in *' --workspace '*) label=work-inventory ;; *) label=boundary-inventory
   BODY
   ;; esac
   printf '%s\n' '{"test-count":1,"rust-suites":{"boundary":{"package-name":"umpire-temporal-tests","binary-name":"boundary","kind":"test","status":"listed","testcases":{"control":{"ignored":false,"filter-match":{"status":"matches"}}}}}}'
   exit 0 ;;
 'exec -- cargo-nextest nextest run'*' -p umpire-temporal-tests '*) label=boundary-tests ;;
 'exec -- go -C '*' env -json GOFLAGS')
   label=go-environment
   BODY
   printf '%s\n' '{"GOFLAGS":"-p=1"}'
   exit 0 ;;
 'exec -- go -C '*' test -json -list . '*)
   label=go-inventory
   BODY
   printf '%s\n' '{"Action":"start","Package":"control"}' '{"Action":"output","Package":"control","Output":"TestControl\n"}' '{"Action":"pass","Package":"control"}'
   exit 0 ;;
 *) label="$*"; label="${label% --message-format libtest-json --message-format-version 0.1}" ;;
esac
"#.replace("BODY", body);
        format!("#!/bin/bash\nset -eu\n{scheduler}\n{nextest}\n{body}")
    }

    fn apalache_control_directory() -> std::path::PathBuf {
        use std::{fs, os::unix::fs::PermissionsExt};
        let root = control_directory();
        let framework = root.join("bin/framework");
        let body = CONTROL_BODY;
        fs::write(
            &framework,
            format!(
                r#"#!/bin/bash
set -eu
[ "$#" = 2 ] && [ "$1" = fetch-apalache ] && [ "$2" = "$UMPIRE_GATE_CONTROL_ROOT/../../umpire" ] || exit 91
[ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL:$MISE_TASK_PGID_MANAGED" = false:false:1 ] || exit 92
[ "$UMPIRE_APALACHE:$UMPIRE_GATE_CONTROL_SENTINEL" = require:inherited ] || exit 93
printf 'fetch stdout\n'
printf 'fetch stderr\n' >&2
label=apalache
{body}
"#
            ),
        )
        .unwrap();
        fs::set_permissions(&framework, fs::Permissions::from_mode(0o755)).unwrap();
        fs::write(
            root.join("scripts/xtask.sh"),
            format!(
                "#!/bin/bash\nlabel=bootstrap\n{body}\nprintf '%s\\n' '{}'\n",
                framework.display()
            ),
        )
        .unwrap();
        let inventory = control_inventory(&root, &["retired-names"]);
        let scheduler = format!(
            r#"
if [ "$*" = "tasks ls --json" ]; then
  label=inventory
  BODY
  printf '%s\n' '{inventory}'
  exit 0
fi
if [ "$*" = "run --force --skip-tools --task-cache off --jobs 4 model:all" ]; then
  label=graph
  BODY
  UMPIRE_GATE_STAGE=retired-names UMPIRE_GATE_CONTROL_STAGE=retired-names "$UMPIRE_TEMPORAL_GATE_EXECUTABLE" --exact gate::tests::run_control_child --nocapture
  exit $?
fi
exit 94
"#
        )
        .replace("BODY", body);
        let mise = root.join("bin/mise");
        fs::write(&mise, control_mise_script(&scheduler)).unwrap();
        fs::set_permissions(&mise, fs::Permissions::from_mode(0o755)).unwrap();
        root
    }

    #[test]
    fn optional_apalache_is_prepared_before_combined_scheduling() {
        use std::fs;
        let root = apalache_control_directory();
        let log = root.join("calls");
        for (mode, failure, expected) in [
            (Some("require"), "", "bootstrap\napalache\ninventory\ngraph\nretired\n"),
            (Some("require"), "apalache", "bootstrap\napalache\n"),
            (None, "", "bootstrap\ninventory\ngraph\nretired\n"),
            (Some("skip"), "", "bootstrap\ninventory\ngraph\nretired\n"),
        ] {
            for timed in ["0", "1"] {
                fs::write(&log, "").unwrap();
                let mut child = Command::new(std::env::current_exe().unwrap());
                child
                    .args(["--exact", "gate::tests::run_control_child", "--nocapture"])
                    .env("UMPIRE_GATE_CONTROL_ROOT", &root)
                    .env("UMPIRE_GATE_CONTROL_LOG", &log)
                    .env("UMPIRE_GATE_CONTROL_STATUS", if failure.is_empty() { "0" } else { "37" })
                    .env("UMPIRE_GATE_CONTROL_FAILURE", failure)
                    .env("UMPIRE_GATE_CONTROL_SENTINEL", "inherited")
                    .env("UMPIRE_QUICK_CACHE", "1")
                    .env("UMPIRE_CACHE_DISABLE", "0")
                    .env("CARGO_HOME", "control-relative-home")
                    .env("CARGO_TARGET_DIR", "control-relative-target")
                    .env("UMPIRE_GATE_CONTROL_CALLER", &root)
                    .env("UMPIRE_BACKENDS", "skip")
                    .env_remove("UMPIRE_GATE_SCOPE")
                    .env("UMPIRE_TIME_STEPS", timed)
                    .env(
                        "PATH",
                        format!(
                            "{}:{}",
                            root.join("bin").display(),
                            std::env::var("PATH").unwrap()
                        ),
                    )
                    .current_dir(&root)
                    .env_remove("UMPIRE_APALACHE");
                if let Some(mode) = mode {
                    child.env("UMPIRE_APALACHE", mode);
                }
                let output = child.output().unwrap();
                assert!(output.status.success(), "mode={mode:?}, timed={timed}: {output:?}");
                assert_eq!(
                    fs::read_to_string(&log).unwrap(),
                    expected,
                    "mode={mode:?}, timed={timed}"
                );
                if mode == Some("require") {
                    let stdout = String::from_utf8_lossy(&output.stdout);
                    let stderr = String::from_utf8_lossy(&output.stderr);
                    if timed == "0" {
                        assert!(stdout.contains("fetch stdout"));
                        assert!(stderr.contains("fetch stderr"));
                    } else if failure.is_empty() {
                        assert!(stdout.contains("apalache"));
                        assert!(!stdout.contains("fetch stdout"));
                        assert!(!stderr.contains("fetch stderr"));
                    } else {
                        assert!(stderr.contains("fetch stdout"));
                        assert!(stderr.contains("fetch stderr"));
                    }
                    if !failure.is_empty() {
                        assert!(stderr.contains("Temporal stage apalache failed (exit 37)"));
                    }
                }
            }
        }
        fs::remove_dir_all(root.parent().unwrap().parent().unwrap()).unwrap();
    }

    #[test]
    fn actual_temporal_gate_wiring_and_named_failures() {
        use std::{fs, os::unix::fs::PermissionsExt};
        let root = control_directory();
        let mise = root.join("bin/mise");
        let stages = CONTROL_STAGES;
        let mut stage_names = Vec::new();
        for (name, _) in &stages {
            if !stage_names.contains(name) {
                stage_names.push(*name);
            }
        }
        let inventory = control_inventory(&root, &stage_names);
        let scheduler = format!(
            r#"
if [ "$*" = "tasks ls --json" ]; then
  printf '%s\n' '{inventory}'
  exit 0
fi
if [ "$*" = "run --force --skip-tools --task-cache off --jobs 4 model:all" ]; then
  for stage_name in {names}; do
    UMPIRE_GATE_STAGE=$stage_name UMPIRE_GATE_CONTROL_STAGE=$stage_name "$UMPIRE_TEMPORAL_GATE_EXECUTABLE" --exact gate::tests::run_control_child --nocapture || exit $?
  done
  exit 0
fi
"#,
            names = stage_names.join(" ")
        );
        fs::write(&mise, control_mise_script(&scheduler)).unwrap();
        fs::set_permissions(&mise, fs::Permissions::from_mode(0o755)).unwrap();
        let expected =
            stages.map(|(_, label)| label.replace("PLACEHOLDER", &root.to_string_lossy()));
        let log = root.join("calls");
        for (timed, failure) in std::iter::once(None)
            .chain((0..expected.len()).map(Some))
            .flat_map(|failure| [("0", failure), ("1", failure)])
        {
            fs::write(&log, "").unwrap();
            let mut child = Command::new(std::env::current_exe().unwrap());
            child
                .args(["--exact", "gate::tests::run_control_child", "--nocapture"])
                .env("UMPIRE_GATE_CONTROL_ROOT", &root)
                .env("UMPIRE_GATE_CONTROL_LOG", &log)
                .env("UMPIRE_GATE_CONTROL_STATUS", if failure.is_some() { "37" } else { "0" })
                .env(
                    "UMPIRE_GATE_CONTROL_FAILURE",
                    failure.map_or("", |index| expected[index].as_str()),
                )
                .env("UMPIRE_QUICK_CACHE", "1")
                .env("UMPIRE_CACHE_DISABLE", "0")
                .env("CARGO_HOME", "control-relative-home")
                .env("CARGO_TARGET_DIR", "control-relative-target")
                .env("UMPIRE_GATE_CONTROL_CALLER", &root)
                .env_remove("UMPIRE_GATE_SCOPE")
                .current_dir(&root)
                .env("UMPIRE_TIME_STEPS", timed)
                .env("MAKEFLAGS", "-n -j99")
                .env("MFLAGS", "-n -j99")
                .env(
                    "PATH",
                    format!("{}:{}", root.join("bin").display(), std::env::var("PATH").unwrap()),
                );
            let output = child.output().unwrap();
            assert!(output.status.success(), "{output:?}");
            if let Some(index) = failure {
                assert!(
                    String::from_utf8_lossy(&output.stderr).contains(stages[index].0),
                    "failure={index}, stage={:?}: {output:?}",
                    stages[index]
                );
            }
            let actual = fs::read_to_string(&log).unwrap();
            let end = failure.map_or(expected.len(), |index| index + 1);
            assert_eq!(actual.lines().collect::<Vec<_>>(), expected[..end]);
        }
        fs::remove_dir_all(root.parent().unwrap().parent().unwrap()).unwrap();
    }

    /// Real consumer stage leaves under the committed table: a planted count bound and a
    /// planted warm budget that both attempts overrun each fail the gate by stage name.
    #[test]
    fn consumer_budget_overruns_fail_by_stage_name() {
        use std::{fs, os::unix::fs::PermissionsExt};
        for (stage, key, value, expected) in [
            ("tests", "executed", "0", "stage temporal/tests budget: executed 1 exceeds 0"),
            (
                "fmt",
                "warm_seconds",
                "{ check = 0.0001, validate = 0.0001 }",
                "stage temporal/fmt budget: second attempt",
            ),
        ] {
            let root = control_directory();
            let inventory = control_inventory(&root, &[stage]);
            let path = root.join(TEMPORAL_BUDGETS);
            let table = fs::read_to_string(&path).unwrap();
            let (head, section) =
                table.split_once(&format!("[stages.\"temporal/{stage}\"]")).unwrap();
            let planted = section
                .lines()
                .map(|line| {
                    if line.split_once(" = ").is_some_and(|(name, _)| name == key) {
                        format!("{key} = {value}")
                    } else {
                        line.to_owned()
                    }
                })
                .collect::<Vec<_>>()
                .join("\n");
            fs::write(&path, format!("{head}[stages.\"temporal/{stage}\"]{planted}\n")).unwrap();
            let scheduler = format!(
                r#"
if [ "$*" = "tasks ls --json" ]; then
  printf '%s\n' '{inventory}'
  exit 0
fi
if [ "$*" = "run --force --skip-tools --task-cache off --jobs 4 model:all" ]; then
  UMPIRE_GATE_STAGE={stage} UMPIRE_GATE_CONTROL_STAGE={stage} "$UMPIRE_TEMPORAL_GATE_EXECUTABLE" --exact gate::tests::run_control_child --nocapture || exit $?
  exit 0
fi
"#
            );
            let mise = root.join("bin/mise");
            fs::write(&mise, control_mise_script(&scheduler)).unwrap();
            fs::set_permissions(&mise, fs::Permissions::from_mode(0o755)).unwrap();
            let log = root.join("calls");
            fs::write(&log, "").unwrap();
            let output = Command::new(std::env::current_exe().unwrap())
                .args(["--exact", "gate::tests::run_control_child", "--nocapture"])
                .env("UMPIRE_GATE_CONTROL_ROOT", &root)
                .env("UMPIRE_GATE_CONTROL_LOG", &log)
                .env("UMPIRE_GATE_CONTROL_STATUS", "nonzero")
                .env("UMPIRE_GATE_CONTROL_FAILURE", "")
                .env("UMPIRE_QUICK_CACHE", "1")
                .env("UMPIRE_CACHE_DISABLE", "0")
                .env("CARGO_HOME", "control-relative-home")
                .env("CARGO_TARGET_DIR", "control-relative-target")
                .env("UMPIRE_GATE_CONTROL_CALLER", &root)
                .env_remove("UMPIRE_GATE_SCOPE")
                .env_remove("UMPIRE_BACKENDS")
                .env_remove("UMPIRE_APALACHE")
                .env("UMPIRE_TIME_STEPS", "0")
                .current_dir(&root)
                .env(
                    "PATH",
                    format!("{}:{}", root.join("bin").display(), std::env::var("PATH").unwrap()),
                )
                .output()
                .unwrap();
            let text = format!(
                "{}{}",
                String::from_utf8_lossy(&output.stdout),
                String::from_utf8_lossy(&output.stderr)
            );
            assert!(output.status.success(), "{stage}: {text}");
            assert!(text.contains(expected), "{stage}: {text}");
            fs::remove_dir_all(root.parent().unwrap().parent().unwrap()).unwrap();
        }
    }
}
