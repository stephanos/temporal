//! System workspace command bridge to the independent framework tools.
#![forbid(unsafe_code)]
use std::{
    env,
    ffi::OsString,
    io::{self, Write},
    os::unix::{
        ffi::OsStringExt,
        process::{CommandExt, ExitStatusExt},
    },
    path::{Path, PathBuf},
    process::Command,
};

mod build_guests;
mod gate;
mod guest_routes;
mod mapping_count;
#[cfg(test)]
mod model_controls;
mod proto;
mod quick;
#[cfg(test)]
#[path = "../tests/retired_names.rs"]
mod retired_name_policy_controls;
mod retired_names;
#[cfg(test)]
mod script_control_support;
mod test_run;

fn bridge(root: &Path, mut arguments: Vec<OsString>) -> io::Result<i32> {
    let framework = root.join("../../umpire");
    let caller = env::current_dir()?;
    let cargo_home = umpire_guest_driver::context::cargo_home(&caller).map_err(io::Error::other)?;
    if arguments.first().is_some_and(|command| command == "quick") {
        arguments = quick::arguments(root, arguments.into_iter().skip(1).collect())?;
    }
    let guest =
        arguments.first().is_some_and(|command| command == "guest" || command == "guest-freshness");
    let workspace =
        if guest { guest_routes::prepare(root, &caller, &mut arguments) } else { caller.clone() };
    let output = gate::framework_executable(root)?;
    if !output.status.success() {
        return Ok(output
            .status
            .code()
            .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)));
    }
    let path = output.stdout.strip_suffix(b"\n").unwrap_or(&output.stdout);
    let tool = PathBuf::from(OsString::from_vec(path.to_owned()));
    if arguments.first().is_some_and(|command| command == "proto-go") {
        return proto::run(root, &tool, &arguments[1..]);
    }
    if arguments.first().is_some_and(|command| command == "build-guests") {
        if arguments.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask build-guests"));
        }
        return build_guests::run(root, &tool.with_file_name("umpire-guest-driver"));
    }
    let mut command = if guest {
        guest_routes::command(
            &tool,
            &workspace,
            &caller,
            &arguments,
            &cargo_home,
            env::var_os("CARGO_TARGET_DIR"),
        )
    } else {
        let mut command = Command::new(tool);
        command.args(arguments).current_dir(workspace).env("CARGO_HOME", cargo_home);
        command
    };
    command.env("UMPIRE_TOOL_ROOT", framework);
    Err(command.exec())
}

/// Whether this consumer entry admits the paired source itself (see `run`).
fn admits_source(supplied: &[OsString]) -> bool {
    supplied
        .iter()
        .find(|argument| argument.as_os_str() != "__owned-command")
        .is_some_and(|command| command == "gate")
}

fn run() -> io::Result<i32> {
    xtask::gate::controller_context::initialize()?;
    xtask::gate::work_counts::owns_controller()?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let mut supplied: Vec<_> = env::args_os().skip(1).collect();
    // Freeze the paired source before the consumer's own gate work, so stage retries and
    // completion can refuse changed inputs. Only `gate` is admitted here: the timing parent
    // owns its benchmark touch between separately admitted gate legs, `test-run` may rewrite
    // goldens under `model-update`, and bridged commands such as `validate` are admitted by
    // the framework entry they hand off to, which owns its own benchmark touch.
    if admits_source(&supplied) {
        xtask::gate::source_context::admit_consumer(&root, &supplied)?;
    }
    let _lease = if supplied.first().is_some_and(|arg| arg == "__owned-command") {
        supplied.remove(0);
        if env::var_os("UMPIRE_JVM_LEASE_FD").is_none() {
            return Err(io::Error::other("missing owned-command descriptor"));
        }
        Some(xtask::gate::lease::JvmLease::inherited_or_acquire()?)
    } else if supplied.first().is_some_and(|arg| {
        [
            "gate",
            "test-run",
            "quick",
            "update",
            "time",
            "validate",
            "timing-controls",
            "proto-controls",
        ]
        .iter()
        .any(|name| arg == name)
    }) && !supplied.iter().any(|arg| arg == "--list")
    {
        return xtask::gate::supervise(supplied).map_err(io::Error::other);
    } else {
        None
    };
    if supplied.first().is_some_and(|command| command == "test-run") {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask test-run"));
        }
        return test_run::run(&root);
    }
    if supplied.first().is_some_and(|command| command == "gate") {
        let arguments: Vec<_> = supplied.into_iter().skip(1).collect();
        return match arguments.as_slice() {
            [] => gate::run(&root),
            [flag] if flag == "--list" => gate::list(&root),
            [flag, name] if flag == "--stage" || flag == "--scheduled-stage" => {
                let name =
                    name.to_str().ok_or_else(|| io::Error::other("gate stage must be UTF-8"))?;
                if flag == "--stage" {
                    gate::run_stage(&root, name)
                } else {
                    gate::scheduled_stage(&root, name)
                }
            }
            _ => Err(io::Error::other(
                "usage: cargo xtask gate [--list | --stage NAME | --scheduled-stage NAME]",
            )),
        };
    }
    if supplied.first().is_some_and(|command| command == "mapping-count") {
        return match supplied.as_slice() {
            [_] => mapping_count::run(&root),
            [_, model] => mapping_count::run(Path::new(model)),
            _ => Err(io::Error::other("usage: cargo xtask mapping-count [MODEL_DIR]")),
        };
    }
    if supplied.first().is_some_and(|command| command == "time" || command == "retired-names") {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask <time|retired-names>"));
        }
        return if supplied[0] == "time" {
            gate::timing::run(&root)
        } else {
            Ok(retired_names::run(
                root.parent().ok_or_else(|| io::Error::other("missing repository root"))?,
            ))
        };
    }
    if supplied
        .first()
        .is_some_and(|command| command == "timing-controls" || command == "proto-controls")
    {
        if supplied.len() != 1 {
            return Err(io::Error::other("usage: cargo xtask <timing-controls|proto-controls>"));
        }
        return gate::control_tests(
            &root,
            supplied[0].to_str().ok_or_else(|| io::Error::other("control name must be UTF-8"))?,
        );
    }
    bridge(&root, supplied)
}

fn main() {
    let result = run();
    // A completed gate also requires its admitted source to be unchanged.
    let result = match (result, xtask::gate::source_context::verify()) {
        (Ok(0), Err(error)) => Err(error),
        (primary, Err(error)) => {
            let _ = writeln!(io::stderr(), "temporal-xtask: source custody: {error}");
            primary
        }
        (primary, Ok(())) => primary,
    };
    let status = match result {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "temporal-xtask: {error}");
            1
        }
    };
    std::process::exit(status);
}

#[cfg(test)]
mod admission_selection_controls {
    use super::admits_source;
    use std::ffi::OsString;

    fn arguments(values: &[&str]) -> Vec<OsString> {
        values.iter().map(OsString::from).collect()
    }

    #[test]
    fn only_consumer_gate_commands_admit_the_paired_source() {
        assert!(admits_source(&arguments(&["gate"])));
        assert!(admits_source(&arguments(&["__owned-command", "gate"])));
        assert!(admits_source(&arguments(&["gate", "--scheduled-stage", "fmt"])));
        for refused in [
            &["time"][..],
            &["__owned-command", "time"],
            &["test-run"],
            &["__owned-command", "test-run"],
            &["validate"],
            &["quick"],
            &["update"],
            &[],
        ] {
            assert!(!admits_source(&arguments(refused)), "{refused:?}");
        }
    }
}
