//! Affected-package validation across explicitly selected Cargo workspaces.
use crate::{Error, Result};
use serde::Serialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    env,
    ffi::OsString,
    io::{self, Write},
    os::unix::{ffi::OsStringExt, process::ExitStatusExt},
    path::{Path, PathBuf},
    process::Command,
    time::Instant,
};
use umpire_guest_driver::context::{ToolRoot, cargo_home};

mod selection;
#[cfg(test)]
mod tests;

#[derive(Default)]
struct Options {
    workspaces: Vec<PathBuf>,
    base: Option<OsString>,
    mappings: Vec<(PathBuf, String)>,
    always: BTreeSet<String>,
    compiler_only: BTreeSet<String>,
    targets: BTreeMap<String, String>,
    filters: BTreeMap<String, String>,
    select_only: bool,
}

fn pair(argument: OsString, flag: &str) -> Result<(String, String)> {
    let argument =
        argument.into_string().map_err(|_| Error::Invalid(format!("{flag} needs UTF-8")))?;
    let (left, right) = argument
        .split_once('=')
        .filter(|(left, right)| !left.is_empty() && !right.is_empty())
        .ok_or_else(|| Error::Invalid(format!("{flag} needs NAME=VALUE")))?;
    Ok((left.into(), right.into()))
}

impl Options {
    fn parse(arguments: Vec<OsString>) -> Result<Self> {
        let mut options = Self::default();
        let mut arguments = arguments.into_iter();
        while let Some(flag) = arguments.next() {
            if flag == "--select-only" {
                options.select_only = true;
                continue;
            }
            let value = arguments
                .next()
                .ok_or_else(|| Error::Invalid(format!("missing value for {}", flag.display())))?;
            match flag.to_str() {
                Some("--workspace") => options.workspaces.push(PathBuf::from(value)),
                Some("--base") => options.base = Some(value),
                Some("--map") => {
                    let (path, name) = pair(value, "--map")?;
                    options.mappings.push((PathBuf::from(path), name));
                }
                Some("--always") => {
                    options.always.insert(
                        value
                            .into_string()
                            .map_err(|_| Error::Invalid("package must be UTF-8".into()))?,
                    );
                }
                Some("--compile-only") => {
                    options.compiler_only.insert(
                        value
                            .into_string()
                            .map_err(|_| Error::Invalid("package must be UTF-8".into()))?,
                    );
                }
                Some("--test-target") => {
                    let (name, target) = pair(value, "--test-target")?;
                    options.targets.insert(name, target);
                }
                Some("--test-filter") => {
                    let (name, filter) = pair(value, "--test-filter")?;
                    options.filters.insert(name, filter);
                }
                _ => {
                    return Err(Error::Invalid(format!("unknown quick option {}", flag.display())));
                }
            }
        }
        if options.workspaces.is_empty() {
            options.workspaces.push(ToolRoot::local()?.path().to_owned());
        }
        if options.always.is_empty() {
            options.always.insert("umpire-fixtures".into());
        }
        options.base =
            Some(options.base.or_else(|| env::var_os("BASE")).unwrap_or_else(|| "HEAD".into()));
        for path in &mut options.workspaces {
            if !path.is_absolute() {
                return Err(Error::Invalid("quick --workspace must be absolute".into()));
            }
            *path = path.canonicalize()?;
        }
        options.workspaces.sort();
        options.workspaces.dedup();
        for (path, _) in &options.mappings {
            if !path.is_absolute()
                || path.components().any(|part| part == std::path::Component::ParentDir)
            {
                return Err(Error::Invalid(
                    "quick --map needs an absolute normalized source directory".into(),
                ));
            }
        }
        Ok(options)
    }
}

#[derive(Serialize)]
struct WorkspaceSelection {
    workspace: PathBuf,
    packages: Vec<String>,
    #[serde(skip)]
    doctests: BTreeSet<String>,
}
#[derive(Serialize)]
struct Selection {
    base: String,
    changed: Vec<PathBuf>,
    reasons: BTreeSet<String>,
    workspaces: Vec<WorkspaceSelection>,
}

fn captured(command: &mut Command) -> Result<Vec<u8>> {
    let output = crate::gate::scheduler::execute(command, false, false)?;
    if !output.status.success() {
        return Err(Error::Invalid(format!(
            "{command:?} failed: {}",
            String::from_utf8_lossy(&output.stderr)
        )));
    }
    Ok(output.stdout)
}

fn git(root: &Path, arguments: &[&str]) -> Command {
    let mut command = Command::new("git");
    command.current_dir(root).args(arguments);
    command
}

fn changed(root: &Path, base: &std::ffi::OsStr) -> Result<Vec<PathBuf>> {
    let mut revision = base.to_os_string();
    revision.push("^{commit}");
    let commit = captured(git(root, &["rev-parse", "--verify", "--end-of-options"]).arg(revision))?;
    let commit = std::str::from_utf8(&commit)
        .map_err(|_| Error::Invalid("Git commit must be UTF-8".into()))?
        .trim();
    let mut paths =
        captured(&mut git(root, &["diff", "--name-only", "--no-renames", "-z", commit, "--"]))?;
    paths.extend(captured(&mut git(root, &["ls-files", "--others", "--exclude-standard", "-z"]))?);
    Ok(paths
        .split(|byte| *byte == 0)
        .filter(|path| !path.is_empty())
        .map(|path| root.join(OsString::from_vec(path.to_vec())))
        .collect::<BTreeSet<_>>()
        .into_iter()
        .collect())
}

fn cargo(root: &Path, home: &Path, target: &Path) -> Command {
    let mut command = Command::new("mise");
    command
        .current_dir(root)
        .args(["exec", "--", "cargo"])
        .env("CARGO_HOME", home)
        .env("CARGO_TARGET_DIR", target)
        .env("UMPIRE_QUICK_CACHE", "1")
        .env("UMPIRE_QUICK_CACHE_ROOT", target.join("quick-cache"))
        .env("UMPIRE_BACKENDS", "require")
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false")
        .env("MISE_TASK_PGID_MANAGED", "1");
    for name in [
        "RUSTC_WRAPPER",
        "RUSTC_WORKSPACE_WRAPPER",
        "RUSTFLAGS",
        "CARGO_ENCODED_RUSTFLAGS",
        "RUSTC_BOOTSTRAP",
        "CARGO_BUILD_TARGET",
    ] {
        command.env_remove(name);
    }
    for (name, _) in env::vars_os() {
        if name.as_encoded_bytes().starts_with(b"CARGO_PROFILE_") {
            command.env_remove(name);
        }
    }
    crate::test_run::configure(
        &mut command,
        true,
        env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require"),
    );
    command
}

fn focused(options: &Options, package: &str, changed: &[PathBuf]) -> bool {
    changed.iter().all(|path| {
        !selection::global_input(path, None)
            && options.mappings.iter().any(|(directory, name)| {
                name == package
                    && path.file_name().is_some_and(|name| name == "model.rs")
                    && path.parent().and_then(Path::parent) == Some(directory.as_path())
            })
    })
}

fn execute(
    phase: &str,
    workspace: &Path,
    packages: &[String],
    command: Command,
    output: impl Write,
) -> Result<i32> {
    execute_with(phase, workspace, packages, command, output, |command| {
        crate::gate::scheduler::execute(command, true, true)
            .map(|output| (output.status, output.stdout))
    })
    .map(|(status, _)| status)
}

fn execute_with(
    phase: &str,
    workspace: &Path,
    packages: &[String],
    mut command: Command,
    mut output: impl Write,
    run: impl FnOnce(&mut Command) -> io::Result<(std::process::ExitStatus, Vec<u8>)>,
) -> Result<(i32, Vec<u8>)> {
    let stage = format!(
        "quick: stage={phase} workspace={} packages={}",
        workspace.display(),
        packages.join(",")
    );
    writeln!(output, "{stage} command={command:?}")?;
    let started = Instant::now();
    match run(&mut command) {
        Ok((status, bytes)) => {
            let code = status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
            writeln!(
                output,
                "{stage} elapsed={:.3}s exit={code}",
                started.elapsed().as_secs_f64()
            )?;
            Ok((code, bytes))
        }
        Err(error) => {
            writeln!(
                output,
                "{stage} elapsed={:.3}s exit=spawn-error error={error}",
                started.elapsed().as_secs_f64()
            )?;
            Err(error.into())
        }
    }
}

fn test_workspace(
    workspace: &WorkspaceSelection,
    options: &Options,
    changed: &[PathBuf],
    compiler_only: &BTreeSet<String>,
    cargo: impl Fn() -> Command,
) -> Result<i32> {
    let selections = workspace
        .packages
        .iter()
        .map(|name| {
            let focused = focused(options, name, changed);
            crate::test_run::Selection {
                package: name,
                target: options.targets.get(name).filter(|_| focused).map(String::as_str),
                filter: options.filters.get(name).filter(|_| focused).map(String::as_str),
                doctests: workspace.doctests.contains(name),
            }
        })
        .collect::<Vec<_>>();
    if !compiler_only.is_empty() {
        let mut command = cargo();
        command.args(["check", "--locked", "--all-targets"]);
        for package in compiler_only {
            command.args(["-p", package]);
        }
        let status = execute(
            "compile",
            &workspace.workspace,
            &compiler_only.iter().cloned().collect::<Vec<_>>(),
            command,
            io::stderr().lock(),
        )?;
        if status != 0 {
            return Ok(status);
        }
    }
    crate::test_run::run_selected_checked_with(
        &selections,
        compiler_only,
        |arguments| {
            let mut command = cargo();
            command.args(arguments).stderr(std::process::Stdio::inherit());
            execute_with(
                "test-inventory",
                &workspace.workspace,
                &workspace.packages,
                command,
                io::stderr().lock(),
                |command| {
                    crate::gate::scheduler::execute(command, false, true)
                        .map(|output| (output.status, output.stdout))
                },
            )
        },
        |phase, arguments| {
            let mut command = cargo();
            command.args(arguments);
            execute(phase, &workspace.workspace, &workspace.packages, command, io::stderr().lock())
        },
    )
}

/// Run affected-package formatting, clippy and tests; `--select-only` reports the selection.
pub fn run(arguments: Vec<OsString>) -> Result<i32> {
    let options = Options::parse(arguments)?;
    let caller = env::current_dir()?;
    let home = cargo_home(&caller)?;
    let target = env::var_os("CARGO_TARGET_DIR").map_or_else(
        || options.workspaces[0].join("target"),
        |path| {
            let path = PathBuf::from(path);
            if path.is_absolute() { path } else { caller.join(path) }
        },
    );
    let repository = captured(&mut git(&options.workspaces[0], &["rev-parse", "--show-toplevel"]))?;
    let repository = repository.strip_suffix(b"\n").unwrap_or(&repository);
    let repository = PathBuf::from(OsString::from_vec(repository.to_vec())).canonicalize()?;
    for workspace in &options.workspaces {
        let actual = captured(&mut git(workspace, &["rev-parse", "--show-toplevel"]))?;
        let actual = actual.strip_suffix(b"\n").unwrap_or(&actual);
        if PathBuf::from(OsString::from_vec(actual.to_vec())).canonicalize()? != repository {
            return Err(Error::Invalid(
                "quick workspaces must belong to the same Git repository".into(),
            ));
        }
    }
    let base = options.base.as_deref().unwrap_or_default();
    let changed = changed(&repository, base)?;
    let mut graph = selection::Graph::default();
    for workspace in &options.workspaces {
        let metadata = captured(cargo(workspace, &home, &target).args([
            "metadata",
            "--locked",
            "--format-version",
            "1",
        ]))?;
        graph.add(workspace, &metadata)?;
    }
    let selection = graph.select(&options, changed)?;
    serde_json::to_writer(io::stdout().lock(), &selection)
        .map_err(|error| Error::Invalid(error.to_string()))?;
    writeln!(io::stdout())?;
    if options.select_only {
        return Ok(0);
    }
    for workspace in &selection.workspaces {
        if workspace.packages.is_empty() {
            continue;
        }
        for operation in ["fmt", "clippy"] {
            let mut command = cargo(&workspace.workspace, &home, &target);
            command.arg(operation);
            if operation == "clippy" {
                command.args(["--locked", "--all-targets", "--all-features"]);
            }
            for name in &workspace.packages {
                command.args(["-p", name]);
            }
            command.args(if operation == "fmt" {
                &["--", "--check"][..]
            } else {
                &["--", "-D", "warnings"][..]
            });
            let status = execute(
                operation,
                &workspace.workspace,
                &workspace.packages,
                command,
                io::stderr().lock(),
            )?;
            if status != 0 {
                return Ok(status);
            }
        }
        let compiler_only = workspace
            .packages
            .iter()
            .filter(|name| {
                options.compiler_only.contains(*name)
                    || graph.compiler_only(&workspace.workspace, name)
            })
            .cloned()
            .collect::<BTreeSet<_>>();
        let status =
            test_workspace(workspace, &options, &selection.changed, &compiler_only, || {
                cargo(&workspace.workspace, &home, &target)
            })?;
        if status != 0 {
            return Ok(status);
        }
    }
    Ok(0)
}
