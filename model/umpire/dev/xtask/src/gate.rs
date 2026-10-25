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
mod boundary;

/// Ordered framework stages. Dependencies are conservative prerequisite edges.
pub const STAGES: &[(&str, &[&str])] = &[
    ("retired-names", &[]),
    ("timing-controls", &[]),
    ("capability-controls", &[]),
    ("source-boundary", &[]),
    ("protobuf-controls", &["proto-tools"]),
    ("protobuf-freshness", &["proto-tools"]),
    ("kettle-protobuf-freshness", &["proto-tools"]),
    ("fmt", &[]),
    ("clippy", &[]),
    ("docs", &[]),
    ("tests", &[]),
    ("wasm32-wasip1", &[]),
    ("core-wasm-clippy", &[]),
    ("wasm-guest", &[]),
    ("go-host", &["wasm-guest"]),
    ("kettle-go-host", &["wasm-guest", "kettle-protobuf-freshness"]),
];

struct Gate {
    root: PathBuf,
    target: PathBuf,
    home: PathBuf,
    required: bool,
    timed: bool,
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
            timed: env::var_os("UMPIRE_TIME_STEPS").as_deref() == Some(OsStr::new("1")),
        })
    }
    fn command(&self, program: impl AsRef<OsStr>) -> Command {
        let mut command = Command::new(program);
        command
            .current_dir(&self.root)
            .env("UMPIRE_TOOL_ROOT", &self.root)
            .env("CARGO_HOME", &self.home)
            .env("CARGO_TARGET_DIR", &self.target)
            .env("UMPIRE_QUICK_CACHE", "0")
            .env("UMPIRE_CACHE_DISABLE", "1")
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false");
        if self.required {
            command.env("UMPIRE_BACKENDS", "require");
        }
        command
    }
    fn cargo(&self, args: &[&str]) -> Command {
        let mut command = self.command("mise");
        command.args(["exec", "--", "cargo"]).args(args);
        command
    }
    fn script(&self, path: &str, args: &[&str]) -> Command {
        let mut command = self.command("/bin/bash");
        command.arg(self.root.join("scripts").join(path)).args(args);
        command
    }
    fn execute(&self, name: &str, mut command: Command) -> Result<i32> {
        let started = Instant::now();
        let timed = self.timed;
        let status = if timed {
            let output = command.output()?;
            if !output.status.success() {
                io::stderr().write_all(&output.stdout)?;
                io::stderr().write_all(&output.stderr)?;
            }
            output.status
        } else {
            command.status()?
        };
        let code = status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0));
        if code == 0 && timed {
            writeln!(io::stdout(), "{name:<32} {:6.3}s", started.elapsed().as_secs_f64())?;
        }
        if code != 0 {
            writeln!(io::stderr(), "framework stage {name} failed (exit {code})")?;
        }
        Ok(code)
    }
    fn stage(&self, name: &str) -> Result<i32> {
        let command = match name {
            "proto-tools" => self.script("proto-tools.sh", &[]),
            "retired-names" => {
                let code = boundary::retired(&self.root, &self.target)?;
                if code != 0 {
                    writeln!(io::stderr(), "framework stage {name} failed (exit {code})")?;
                }
                return Ok(code);
            }
            "source-boundary" => return self.boundary(),
            "timing-controls" => {
                let code = self.execute(name, self.script("test-time-gate.sh", &[]))?;
                if code != 0 {
                    return Ok(code);
                }
                self.script("tests/build-guests-controls.sh", &[])
            }
            "capability-controls" => self.script("test-capabilities.sh", &[]),
            "protobuf-controls" => self.script("tests/proto-go.sh", &[]),
            "protobuf-freshness" => self.script("proto-go.sh", &["core", "--check"]),
            "kettle-protobuf-freshness" => self.script("proto-go.sh", &["kettle", "--check"]),
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
            "tests" => {
                let mut c = self.cargo(&[]);
                crate::test_run::configure(&mut c);
                c
            }
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
            "wasm-guest" => self.script("tests/build-guests.sh", &[]),
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
        self.execute(name, command)
    }
    fn boundary(&self) -> Result<i32> {
        boundary::check(&self.root, &self.target)?;
        let output = self.cargo(&["metadata", "--locked", "--format-version", "1"]).output()?;
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
    fn full(&self) -> Result<i32> {
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
            let code = self.execute("apalache", command)?;
            if code != 0 {
                return Ok(code);
            }
        }
        let code = self.stage("proto-tools")?;
        if code != 0 {
            return Ok(code);
        }
        for (name, _) in STAGES {
            let started = Instant::now();
            let result = self
                .stage(name)
                .map_err(|e| Error::Invalid(format!("framework stage {name} failed: {e}")))?;
            if result != 0 {
                return Ok(result);
            }
            if ["source-boundary", "retired-names"].contains(name) && self.timed {
                writeln!(io::stdout(), "{name:<32} {:6.3}s", started.elapsed().as_secs_f64())?;
            }
        }
        Ok(0)
    }
}

/// Run the framework gate or one of its formatting, update and timing entry points.
pub fn run(command: &str, args: Vec<OsString>) -> Result<i32> {
    if command == "gate" && args == [OsString::from("--list")] {
        for (name, deps) in STAGES {
            writeln!(io::stdout(), "{name}\t{}", deps.join(","))?;
        }
        return Ok(0);
    }
    let args = args.into_boxed_slice();
    let tools = ToolRoot::local()?;
    let mut gate = Gate::new(&tools, command == "validate")?;
    if matches!(command, "time" | "validate") {
        gate.timed = true;
    }
    if command == "gate" && args.len() == 2 && args[0] == "--stage" {
        let stage = args[1].to_str().ok_or_else(|| Error::Invalid("stage must be UTF-8".into()))?;
        return gate
            .stage(stage)
            .map_err(|e| Error::Invalid(format!("framework stage {stage} failed: {e}")));
    }
    if !args.is_empty() {
        return Err(Error::Invalid(format!(
            "usage: {command} (gate also accepts --list or --stage NAME)"
        )));
    }
    match command {
        "gate" => gate.full(),
        "fmt" => gate.execute("fmt", gate.cargo(&["fmt", "--all"])),
        "update" => {
            let mut c = gate.cargo(&[]);
            crate::test_run::configure(&mut c);
            c.env("INSTA_UPDATE", "always").env("TRYBUILD", "overwrite");
            gate.execute("update", c)
        }
        "time" | "validate" => {
            let start = Instant::now();
            let code = gate.full()?;
            if code != 0 {
                return Ok(code);
            }
            let total = start.elapsed().as_secs_f64();
            let source = env::var_os("UMPIRE_INNER_SOURCE")
                .map_or_else(|| gate.root.join("backends/native/src/lib.rs"), PathBuf::from);
            let source = if source.is_absolute() { source } else { gate.root.join(source) };
            let file = fs::OpenOptions::new().write(true).open(&source)?;
            file.set_modified(std::time::SystemTime::now())?;
            let package =
                env::var("UMPIRE_INNER_CRATE").unwrap_or_else(|_| "umpire-checker".into());
            let inner = Instant::now();
            let code = gate.execute(
                "inner loop (edit+test)",
                gate.cargo(&["test", "-p", &package, "--quiet"]),
            )?;
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
                let budget =
                    env::var(key).ok().and_then(|v| v.parse::<f64>().ok()).unwrap_or(default);
                if elapsed > budget {
                    writeln!(io::stdout(), "note: {label} over soft target ({budget}s)")?;
                }
            }
            Ok(code)
        }
        _ => Err(Error::Invalid(format!("unknown framework command: {command}"))),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn real_stage_child_receives_cache_bypass_and_required_mode() -> Result<()> {
        let directory = tempfile::tempdir()?;
        let gate = Gate {
            root: directory.path().to_owned(),
            target: directory.path().join("target"),
            home: directory.path().join("cargo"),
            required: true,
            timed: false,
        };
        let mut command = gate.command("/bin/sh");
        command.args(["-c","test \"$UMPIRE_QUICK_CACHE\" = 0 && test \"$UMPIRE_CACHE_DISABLE\" = 1 && test \"$UMPIRE_BACKENDS\" = require"]);
        assert_eq!(gate.execute("cache-control", command)?, 0);
        let mut command = gate.command("/bin/sh");
        command.args(["-c", "exit 37"]);
        assert_eq!(gate.execute("exit-control", command)?, 37);
        Ok(())
    }
}
