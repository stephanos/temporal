//! Framework validation stages and their native command executor.
use crate::{Error, Result};
use std::{
    env,
    ffi::{OsStr, OsString},
    fs,
    io::{self, Write},
    os::unix::process::ExitStatusExt,
    path::PathBuf,
    process::Command,
    time::Instant,
};
use umpire_guest_driver::context::{ToolRoot, cargo_home};
pub mod admission;
mod boundary;
mod budget_coordinator;
mod budgets;
mod go_command;
mod go_counts;
pub mod lease;
pub mod scheduler;
mod standalone;
pub mod timing;
pub mod work_counts;

struct Gate {
    root: PathBuf,
    target: PathBuf,
    home: PathBuf,
    required: bool,
    timed: bool,
    child_records: Option<PathBuf>,
}
impl Gate {
    fn new(tools: &ToolRoot, required: bool) -> Result<Self> {
        let caller = env::current_dir()?;
        let target = env::var_os("CARGO_TARGET_DIR").map_or_else(
            || tools.path().join("target"),
            |p| {
                let p = PathBuf::from(p);
                if p.is_absolute() { p } else { caller.join(p) }
            },
        );
        Ok(Self {
            root: tools.path().to_owned(),
            target,
            home: cargo_home(&caller)?,
            required,
            child_records: None,
            timed: env::var_os("UMPIRE_TIME_STEPS").as_deref() == Some(OsStr::new("1")),
        })
    }
    fn command(&self, program: impl AsRef<OsStr>) -> Command {
        let mut command = Command::new(program);
        command
            .current_dir(&self.root)
            .env("UMPIRE_TOOL_ROOT", &self.root)
            .env("UMPIRE_FRAMEWORK_ROOT", &self.root)
            .env("CARGO_HOME", &self.home)
            .env("CARGO_TARGET_DIR", &self.target)
            .env("UMPIRE_QUICK_CACHE", "0")
            .env("UMPIRE_CACHE_DISABLE", "1")
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false")
            .env("MISE_TASK_PGID_MANAGED", "1");
        if self.required {
            command.env("UMPIRE_BACKENDS", "require");
        }
        admission::configure(
            &mut command,
            self.required || env::var_os("UMPIRE_APALACHE").is_some_and(|v| v == "require"),
        );
        command
    }
    fn cargo(&self, args: &[&str]) -> Command {
        let mut command = self.command("mise");
        command.args(["exec", "--", "cargo"]).args(args);
        command
    }
    fn tests(&self) -> Result<Command> {
        let mut command = self.command(env::current_exe()?);
        command.arg("test-run");
        crate::test_run::configure(
            &mut command,
            self.required,
            env::var_os("UMPIRE_APALACHE").as_deref() == Some(OsStr::new("require")),
        );
        Ok(command)
    }
    fn native(&self, args: &[&str]) -> Result<Command> {
        let mut command = self.command(env::current_exe()?);
        command.args(args);
        Ok(command)
    }
    fn controls(&self, name: &str, filters: &[&str]) -> Result<i32> {
        let selections: Vec<_> = filters
            .iter()
            .map(|filter| crate::test_run::Selection {
                package: "xtask",
                target: None,
                filter: Some(filter),
                doctests: false,
            })
            .collect();
        let command = |args: &[String]| {
            let mut command = self.cargo(&[]);
            command.args(args);
            crate::test_run::configure(
                &mut command,
                self.required,
                env::var_os("UMPIRE_APALACHE").as_deref() == Some(OsStr::new("require")),
            );
            command
        };
        crate::test_run::run_selected_observed_with(
            &selections,
            &std::collections::BTreeSet::new(),
            |args| {
                let output = scheduler::execute(&command(args), false, true)?;
                Ok((
                    output
                        .status
                        .code()
                        .unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0)),
                    output.stdout,
                ))
            },
            |_, args, inventory| {
                let started = Instant::now();
                let status = timing::capture(&command(args), self.timed, inventory)?.status;
                let code = status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
                self.report(name, code, started.elapsed().as_secs_f64())?;
                Ok(code)
            },
        )
    }
    fn execute(&self, name: &str, command: &Command) -> Result<i32> {
        let started = Instant::now();
        let status = timing::capture(command, self.timed, None)?.status;
        let code = status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
        self.report(name, code, started.elapsed().as_secs_f64())?;
        Ok(code)
    }
    fn execute_clean(&self, name: &str, command: &Command) -> Result<i32> {
        let started = Instant::now();
        let directory = scheduler::directory(&self.target.join("command-receipts"))?;
        let code = scheduler::supervised_clean(command, &directory, !self.timed)?;
        self.report(name, code, started.elapsed().as_secs_f64())?;
        Ok(code)
    }
    fn report(&self, name: &str, code: i32, seconds: f64) -> io::Result<()> {
        if code == 0 && self.timed {
            writeln!(io::stdout(), "{name:<32} {seconds:6.3}s")?;
        }
        if code != 0 {
            writeln!(io::stderr(), "framework stage {name} failed (exit {code})")?;
        }
        Ok(())
    }
    fn timing_controls(&self, name: &str) -> Result<i32> {
        let code = self.controls(
            name,
            &["native_leaves::public_timing_front_door_preserves_status_diagnostics_and_cleanup"],
        )?;
        if code == 0 {
            self.controls(name, &[
                        "build_guests::path_controls::relative_absolute_default_and_symlink_cargo_paths_preserve_seven_children",
                        "build_guests::path_controls::guest_and_native_statuses_stop_without_following_children",
                    ])
        } else {
            Ok(code)
        }
    }
    fn stage(&self, name: &str) -> Result<i32> {
        let command = match name {
            "proto-tools" => self.native(&["proto-tools"])?,
            "retired-names" => {
                let code = boundary::retired(&self.root, &self.target)?;
                if code != 0 {
                    writeln!(io::stderr(), "framework stage {name} failed (exit {code})")?;
                }
                return Ok(code);
            }
            "source-boundary" => return self.boundary(),
            "standalone" => return standalone::run(self),
            "timing-controls" => return self.timing_controls(name),
            "capability-controls" => {
                return self.controls(
                    name,
                    &["native_leaves::capability_controls_enforce_core_and_unsafe_policies"],
                );
            }
            "protobuf-controls" => {
                return self.controls(
                    name,
                    &["proto::controls::tests::native_protobuf_behavior_controls"],
                );
            }
            "protobuf-freshness" => self.native(&["proto-go", "core", "--check"])?,
            "kettle-protobuf-freshness" => self.native(&["proto-go", "kettle", "--check"])?,
            "fmt" => self.cargo(&["fmt", "--all", "--check"]),
            "clippy" => self.cargo(&[
                "clippy",
                "--workspace",
                "--all-targets",
                "--all-features",
                "--",
                "-D",
                "warnings",
            ]),
            "docs" => {
                let mut c = self.cargo(&["doc", "--workspace", "--no-deps", "--quiet"]);
                c.env("RUSTDOCFLAGS", "-D warnings");
                c
            }
            "tests" => self.tests()?,
            "wasm32-wasip1" => self.cargo(&[
                "build",
                "--target",
                "wasm32-wasip1",
                "-p",
                "umpire-dsl",
                "-p",
                "umpire-checker",
                "--quiet",
            ]),
            "core-wasm-clippy" => self.cargo(&[
                "clippy",
                "--target",
                "wasm32-wasip1",
                "-p",
                "umpire-dsl",
                "-p",
                "umpire-checker",
                "--lib",
                "--locked",
                "--",
                "-D",
                "warnings",
            ]),
            "wasm-guest" => self.native(&["build-guests"])?,
            "go-host" | "kettle-go-host" => {
                let mut c = self.command("mise");
                c.args(["exec", "--", "go", "-C"])
                    .arg(self.root.join(if name == "go-host" {
                        "hosts/go"
                    } else {
                        "reference/kettle"
                    }))
                    .args(["test", "-tags", "test_dep", "-count=1", "./..."])
                    .env("CGO_ENABLED", "0");
                for (key, file) in [
                    ("UMPIRE_WASM", "wasm32-wasip1/guest/umpire_stock_guest.wasm"),
                    ("UMPIRE_KETTLE_WASM", "wasm32-wasip1/guest/umpire_kettle_guest.wasm"),
                    ("UMPIRE_STATEFUL_WASM", "wasm32-wasip1/guest/stateful_control.wasm"),
                    ("UMPIRE_CAUGHT_PANIC_WASM", "wasm32-wasip1/guest/caught_panic_control.wasm"),
                    ("UMPIRE_OVERFLOW_WASM", "wasm32-wasip1/guest/overflow_control.wasm"),
                    ("UMPIRE_PANIC_NATIVE", "panic-native"),
                    ("UMPIRE_CURRENT_CONTRACT_NATIVE", "current-contract-native"),
                ] {
                    c.env(key, self.target.join(file));
                }
                c
            }
            _ => return Err(Error::Invalid(format!("unknown framework stage: {name}"))),
        };
        self.execute(name, &command)
    }
    fn boundary(&self) -> Result<i32> {
        boundary::check(&self.root, &self.target)?;
        let output = scheduler::execute(
            &self.cargo(&["metadata", "--locked", "--format-version", "1"]),
            false,
            false,
        )?;
        if !output.status.success() {
            io::stderr().write_all(&output.stderr)?;
            let code =
                output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0));
            writeln!(io::stderr(), "framework stage source-boundary failed (exit {code})")?;
            return Ok(code);
        }
        boundary::metadata(&self.root, &output.stdout)?;
        Ok(0)
    }
    fn time(&self) -> Result<i32> {
        let start = Instant::now();
        let code = self.full()?;
        if code != 0 {
            return Ok(code);
        }
        let total = start.elapsed().as_secs_f64();
        let source = env::var_os("UMPIRE_INNER_SOURCE")
            .map_or_else(|| self.root.join("backends/native/src/lib.rs"), PathBuf::from);
        let source = if source.is_absolute() { source } else { self.root.join(source) };
        let file = fs::OpenOptions::new().write(true).open(&source)?;
        file.set_modified(std::time::SystemTime::now())?;
        let package = env::var("UMPIRE_INNER_CRATE").unwrap_or_else(|_| "umpire-checker".into());
        let inner = Instant::now();
        let mut command = self.command(env::current_exe()?);
        command.args(["test-run", "--package", &package]);
        let code = self.execute("inner loop (edit+test)", &command)?;
        if code != 0 {
            return Ok(code);
        }
        writeln!(
            io::stdout(),
            "gate total {:.3}s; inner loop {:.3}s",
            total,
            inner.elapsed().as_secs_f64()
        )?;
        for (label, elapsed, key, default) in [
            ("gate", total, "UMPIRE_GATE_BUDGET", 60.0),
            ("inner loop", inner.elapsed().as_secs_f64(), "UMPIRE_INNER_BUDGET", 10.0),
        ] {
            let budget = env::var(key).ok().and_then(|v| v.parse::<f64>().ok()).unwrap_or(default);
            if elapsed > budget {
                writeln!(io::stdout(), "note: {label} over soft target ({budget}s)")?;
            }
        }
        Ok(code)
    }
    fn full(&self) -> Result<i32> {
        self.full_inner(None)
    }
    fn full_child(&self, manifest: &std::path::Path) -> Result<i32> {
        self.full_inner(Some(manifest))
    }
    fn full_inner(&self, manifest: Option<&std::path::Path>) -> Result<i32> {
        writeln!(
            io::stdout(),
            "backend modes: UMPIRE_BACKENDS={} UMPIRE_APALACHE={}",
            if self.required {
                "require".into()
            } else {
                env::var("UMPIRE_BACKENDS").unwrap_or_else(|_| "skip (default)".into())
            },
            env::var("UMPIRE_APALACHE").unwrap_or_else(|_| "skip (default)".into())
        )?;
        if env::var_os("UMPIRE_APALACHE").as_deref() == Some(OsStr::new("require")) {
            let mut command = self.command(env::current_exe()?);
            command.arg("fetch-apalache").arg(&self.root);
            let code = self.execute("apalache", &command)?;
            if code != 0 {
                return Ok(code);
            }
        }
        let mut expected = scheduler::inventory(self.command("mise"), "gate:")?;
        let child = self.child_records.is_some();
        if child {
            expected.retain(|name| !name.ends_with("/standalone") && name != "standalone");
        }
        let directory = scheduler::directory(&self.target.join("gate-receipts"))?;
        let mut command = self.command("mise");
        command
            .env("UMPIRE_GATE_EXECUTABLE", env::current_exe()?)
            .env("UMPIRE_GATE_CONTROLLER_EXECUTABLE", env::current_exe()?);
        if self.timed {
            command.env("UMPIRE_TIME_STEPS", "1");
        }
        if let Some(manifest) = manifest {
            command.env("UMPIRE_STANDALONE_MANIFEST", manifest);
        }
        let code = scheduler::run(
            command,
            if child { "gate:child" } else { "gate:all" },
            &directory,
            &expected,
        )?;
        if let Some(records) = &self.child_records {
            fs::write(records, fs::read(directory.join("stages.jsonl")).unwrap_or_default())?;
        }
        Ok(code)
    }
}

/// Run the framework gate or one of its formatting, update and timing entry points.
pub fn run(command: &str, args: Vec<OsString>) -> Result<i32> {
    if command == "gate" && args == [OsString::from("--list")] {
        let tools = ToolRoot::local()?;
        let gate = Gate::new(&tools, false)?;
        for (name, dependencies) in scheduler::listing(gate.command("mise"), "gate:")? {
            writeln!(
                io::stdout(),
                "{}\t{}",
                name.rsplit('/').next().unwrap_or(&name),
                dependencies.join(",")
            )?;
        }
        return Ok(0);
    }
    let args = args.into_boxed_slice();
    let tools = ToolRoot::local()?;
    let required = command == "validate"
        || env::var_os("UMPIRE_BACKENDS").as_deref() == Some(OsStr::new("require"));
    let mut gate = Gate::new(&tools, required)?;
    if matches!(command, "time" | "validate") {
        gate.timed = true;
    }
    if command == "gate"
        && args.len() == 2
        && (args[0] == "--stage" || args[0] == "--scheduled-stage")
    {
        let stage = args[1].to_str().ok_or_else(|| Error::Invalid("stage must be UTF-8".into()))?;
        if env::var_os("UMPIRE_STANDALONE_CHILD").is_some() {
            if stage == "standalone" {
                return Err(Error::Invalid("standalone recursion refused".into()));
            }
            let manifest = env::var_os("UMPIRE_STANDALONE_MANIFEST").ok_or_else(|| {
                Error::Invalid("standalone recursion or invalid child mode refused".into())
            })?;
            standalone::verify(&PathBuf::from(manifest), &gate)?;
        }
        if args[0] == "--scheduled-stage" && stage == "standalone" && !gate.required {
            return scheduler::skipped(stage, "standalone runs only with TLC required")
                .map_err(Into::into);
        }
        let run = || {
            gate.stage(stage).map_err(|error| {
                io::Error::other(format!("framework stage {stage} failed: {error}"))
            })
        };
        return if args[0] == "--scheduled-stage" {
            scheduler::scheduled(stage, run).map_err(Into::into)
        } else {
            run().map_err(Into::into)
        };
    }
    if command == "gate" && args.len() == 2 && args[0] == "--standalone-child" {
        gate.required = true;
        gate.child_records = Some(standalone::admit(&PathBuf::from(&args[1]), &gate)?);
        return gate.full_child(&PathBuf::from(&args[1]));
    }
    if env::var_os("UMPIRE_STANDALONE_CHILD").is_some() {
        return Err(Error::Invalid("standalone recursion or invalid child mode refused".into()));
    }
    if !args.is_empty() {
        return Err(Error::Invalid(format!(
            "usage: {command} (gate also accepts --list or --stage NAME)"
        )));
    }
    match command {
        "gate" => gate.full(),
        "fmt" => gate.execute("fmt", &gate.cargo(&["fmt", "--all"])),
        "update" => {
            let mut c = gate.tests()?;
            c.env("INSTA_UPDATE", "always").env("TRYBUILD", "overwrite");
            gate.execute("update", &c)
        }
        "time" | "validate" => gate.time(),
        _ => Err(Error::Invalid(format!("unknown framework command: {command}"))),
    }
}

/// Run one public command under supervisor-held host admission.
pub fn supervise(arguments: Vec<OsString>) -> Result<i32> {
    use std::os::unix::fs::PermissionsExt;
    let directory =
        tempfile::Builder::new().permissions(fs::Permissions::from_mode(0o700)).tempdir()?;
    let required = arguments.first().is_some_and(|arg| arg == "quick" || arg == "validate")
        || env::var_os("UMPIRE_BACKENDS").is_some_and(|value| value == "require")
        || env::var_os("UMPIRE_APALACHE").is_some_and(|value| value == "require");
    let mut command = Command::new(env::current_exe()?);
    admission::configure(&mut command, required);
    command.arg("__owned-command").args(arguments).current_dir(env::current_dir()?);
    let result = scheduler::supervised(&command, directory.path(), &[]).map_err(Into::into);
    directory.close()?;
    result
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    #[cfg(target_os = "linux")]
    fn stage_reaps_background_children_before_returning_the_failed_status() -> Result<()> {
        let directory = tempfile::tempdir()?;
        let gate = Gate {
            root: directory.path().to_owned(),
            target: directory.path().join("target"),
            home: directory.path().join("cargo"),
            required: false,
            timed: false,
            child_records: None,
        };
        let pid = directory.path().join("child.pid");
        let mut command = gate.command("/bin/sh");
        command
            .args(["-c", "sleep 30 & printf '%s' $! > \"$CONTROL_PID\"; exit 37"])
            .env("CONTROL_PID", &pid);
        assert_eq!(gate.execute("owned-child", &command)?, 37);
        let pid = fs::read_to_string(pid)?;
        let alive = Command::new("/bin/kill")
            .args(["-0", &pid])
            .stderr(std::process::Stdio::null())
            .status()?
            .success();
        if alive {
            Command::new("/bin/kill").args(["-KILL", &pid]).status()?;
        }
        assert!(!alive, "stage returned while its background child was alive");
        Ok(())
    }

    #[test]
    fn real_stage_child_receives_cache_bypass_and_required_mode() -> Result<()> {
        let directory = tempfile::tempdir()?;
        let gate = Gate {
            root: directory.path().to_owned(),
            target: directory.path().join("target"),
            home: directory.path().join("cargo"),
            required: true,
            timed: false,
            child_records: None,
        };
        let mut command = gate.command("/bin/sh");
        command.args(["-c","test \"$UMPIRE_QUICK_CACHE\" = 0 && test \"$UMPIRE_CACHE_DISABLE\" = 1 && test \"$UMPIRE_BACKENDS\" = require"]);
        assert_eq!(gate.execute("cache-control", &command)?, 0);
        let mut command = gate.command("/bin/sh");
        command.args(["-c", "exit 37"]);
        assert_eq!(gate.execute("exit-control", &command)?, 37);
        Ok(())
    }
}
