//! Shipping Make recipes exercised with disposable gate and compiler children.
use std::{
    env,
    ffi::OsStr,
    fs,
    io::{self, Write},
    os::unix::{
        fs::{PermissionsExt, symlink},
        process::ExitStatusExt,
    },
    path::{Path, PathBuf},
    process::{Command, Output},
    sync::atomic::{AtomicU64, Ordering},
};

fn require(condition: bool, label: &str) -> io::Result<()> {
    if condition {
        Ok(())
    } else {
        Err(io::Error::other(format!("Model recipe control failed: {label}")))
    }
}

fn executable(path: &Path, source: &str) -> io::Result<()> {
    fs::write(path, source)?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o755))
}

/// Umpire stays internal: Temporal's GitHub workflows must not check out or run it.
fn local_only(repository: &Path) -> io::Result<()> {
    require(
        !repository.join(".github/workflows/umpire.yml").try_exists()?,
        "Temporal GitHub workflows exclude Umpire",
    )
}

struct Fixture {
    root: PathBuf,
    tool: PathBuf,
    timing: bool,
}
impl Drop for Fixture {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(self.root.parent().unwrap_or(&self.root));
    }
}
impl Fixture {
    fn new(shipping: &str, tool: &Path, timing: bool) -> io::Result<Self> {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let parent = env::temp_dir().canonicalize()?;
        let container = parent.join(format!(
            "model-recipe-controls-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let root = container.join("temporal");
        let tool = tool.canonicalize()?;
        fs::create_dir_all(root.join("../umpire"))?;
        let fixture = Self { root, tool, timing };
        for directory in ["model/scripts", "bin", "output", "physical home", "physical target"] {
            fs::create_dir_all(fixture.root.join(directory))?;
        }
        symlink("physical home", fixture.root.join("relative-home"))?;
        symlink("physical target", fixture.root.join("relative-target"))?;
        fs::write(fixture.root.join("model/inner.rs"), "control source\n")?;
        let section = shipping
            .split_once("##### model: Umpire in Rust")
            .ok_or_else(|| io::Error::other("missing shipping Model Make section"))?
            .1;
        let mut makefile = String::from(
            "COLOR = \"%s\\n\"\nAPI_BINPB = control-api\nPROTOC_GEN_GO = control-go\nPROTOC_GEN_GO_GRPC = control-grpc\n",
        );
        if timing {
            let mut copy = false;
            for line in section.lines() {
                if line.starts_with("RUST_MODEL_")
                    || line.starts_with("model-time:")
                    || line.starts_with("model-validate:")
                {
                    copy = true;
                }
                if copy {
                    makefile.push_str(line);
                    makefile.push('\n');
                }
                if line.is_empty() || line.starts_with("RUST_MODEL_") {
                    copy = false;
                }
            }
            require(
                makefile.contains("model-time:") && makefile.contains("model-validate:"),
                "shipping timer and validation recipes exist",
            )?;
            makefile.push_str(r#"
.PHONY: model-check
model-check:
	@printf 'gate:%s:%s\n' "$${UMPIRE_BACKENDS-unset}" "$${UMPIRE_APALACHE-unset}" >> "$$CONTROL_LOG"
	@printf 'gate stdout diagnostic\n'; printf 'gate stderr diagnostic\n' >&2; exit "$${CONTROL_GATE_STATUS:-0}"
"#);
        } else {
            makefile.push_str("# Model Make recipes\n");
            makefile.push_str(section.split_once('\n').map_or("", |(_, body)| body));
            makefile.push_str(
                r#"
.PHONY: control-api control-go control-grpc
control-api control-go control-grpc:
	@test "$$UMPIRE_QUICK_CACHE:$$UMPIRE_CACHE_DISABLE" = 0:1
	@printf 'producer|%s\n' "$@" >> "$$CONTROL_LOG"
"#,
            );
        }
        fs::write(fixture.root.join("Makefile"), makefile)?;
        executable(&fixture.root.join("model/scripts/xtask.sh"), BOOTSTRAP)?;
        executable(
            &fixture.root.join("model/scripts/framework-root.sh"),
            "#!/bin/sh\nprintf '%s\\n' \"$CONTROL_ROOT/../umpire\"\n",
        )?;
        fs::create_dir_all(fixture.root.join("../umpire/tools/scripts"))?;
        executable(
            &fixture.root.join("../umpire/tools/scripts/run-xtask.sh"),
            "#!/bin/sh\nexec \"$CONTROL_FRAMEWORK\" \"$@\"\n",
        )?;
        executable(&fixture.root.join("bin/framework"), FRAMEWORK)?;
        executable(&fixture.root.join("bin/mise"), MISE)?;
        fs::write(fixture.root.join("calls"), "")?;
        Ok(fixture)
    }

    fn command(&self, program: impl AsRef<OsStr>) -> io::Result<Command> {
        let mut command = Command::new(program);
        let mut path = vec![self.root.join("bin")];
        if let Some(inherited) = env::var_os("PATH") {
            path.extend(env::split_paths(&inherited));
        }
        command
            .current_dir(&self.root)
            .env("PATH", env::join_paths(path).map_err(io::Error::other)?)
            .env("CONTROL_ROOT", &self.root)
            .env("CONTROL_MODEL", &self.root)
            .env("CONTROL_LOG", self.root.join("calls"))
            .env("CONTROL_TOOL", &self.tool)
            .env("CONTROL_NATIVE_TIMER_CHILD", "1")
            .env("CONTROL_FRAMEWORK", self.root.join("bin/framework"))
            .env("CONTROL_TIMING", if self.timing { "1" } else { "0" })
            .env("TMPDIR", self.root.join("output"))
            .env("CARGO_HOME", "relative-home")
            .env("CARGO_TARGET_DIR", "relative-target")
            .env("CARGO_BUILD_JOBS", "99")
            .env("UMPIRE_QUICK_CACHE", "1")
            .env("UMPIRE_CACHE_DISABLE", "0")
            .env("UMPIRE_INNER_SOURCE", "inner.rs")
            .env("UMPIRE_INNER_CRATE", "control-checker")
            .env("UMPIRE_GATE_BUDGET", "-1")
            .env("UMPIRE_INNER_BUDGET", "-1");
        for key in [
            "UMPIRE_BACKENDS",
            "UMPIRE_APALACHE",
            "BUILD_CACHE_MANAGED",
            "BUILD_CACHE_TOOL",
            "MAKEFLAGS",
            "MFLAGS",
            "MAKEOVERRIDES",
            "BASH_ENV",
            "CONTROL_GATE_STATUS",
            "CONTROL_BENCHMARK_STATUS",
            "CONTROL_BOOTSTRAP_STATUS",
            "CONTROL_FRAMEWORK_STATUS",
            "CONTROL_TEMPORAL_STATUS",
        ] {
            command.env_remove(key);
        }
        Ok(command)
    }

    fn check(
        &self,
        label: &str,
        command: &Command,
        expected_status: i32,
        expected: &str,
    ) -> io::Result<Output> {
        fs::write(self.root.join("calls"), "")?;
        let output = xtask::gate::scheduler::execute(command, false, false)?;
        let status =
            output.status.code().unwrap_or_else(|| 128 + output.status.signal().unwrap_or(0));
        let calls = fs::read_to_string(self.root.join("calls"))?;
        let diagnostics = [output.stdout.as_slice(), output.stderr.as_slice()].concat();
        require(
            status == expected_status && calls == expected,
            &format!(
                "{label}: expected status {expected_status} / calls {expected:?}; got {status} / {calls:?}; {}",
                String::from_utf8_lossy(&diagnostics)
            ),
        )?;
        require(
            fs::read_dir(self.root.join("output"))?.next().is_none(),
            &format!("{label}: temporary capture cleanup"),
        )?;
        if self.timing {
            require(
                fs::read(self.root.join("model/inner.rs"))? == b"control source\n",
                &format!("{label}: source edit restored"),
            )?;
            let text = String::from_utf8_lossy(&diagnostics);
            if status == 0 {
                for warning in ["gate over soft target", "inner loop over soft target"] {
                    require(text.contains(warning), &format!("{label}: {warning}"))?;
                }
            } else {
                require(
                    !text.contains("over soft target"),
                    &format!("{label}: failure must not publish success warnings"),
                )?;
                let stage = if calls.contains("benchmark:") {
                    "benchmark"
                } else if label.contains("bootstrap") {
                    "bootstrap"
                } else {
                    "gate"
                };
                for stream in ["stdout", "stderr"] {
                    require(
                        text.contains(&format!("{stage} {stream} diagnostic")),
                        &format!("{label}: preserves {stage} {stream}"),
                    )?;
                }
            }
        }
        Ok(output)
    }

    fn close(self) -> io::Result<()> {
        fs::remove_dir_all(self.root.parent().unwrap_or(&self.root))
    }
}

const BOOTSTRAP: &str = r#"#!/bin/bash
set -euo pipefail
if [ "$CONTROL_TIMING" = 1 ]; then
 [ "$CARGO_HOME:$CARGO_TARGET_DIR" = "$CONTROL_MODEL/physical home:$CONTROL_MODEL/relative-target" ]
 [ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL" = false:false ]
 if [ "${UMPIRE_BACKENDS:-}" = require ] || [ "${UMPIRE_APALACHE:-}" = require ]; then [ "$CARGO_BUILD_JOBS" = 1 ]; else [ "$CARGO_BUILD_JOBS" = 99 ]; fi
 if [ "${CONTROL_BOOTSTRAP_STATUS:-0}" != 0 ]; then
  printf 'bootstrap stdout diagnostic\n' >&2
  printf 'bootstrap stderr diagnostic\n' >&2
  exit "$CONTROL_BOOTSTRAP_STATUS"
 fi
fi
printf '%s\n' "$CONTROL_FRAMEWORK"
"#;

const FRAMEWORK: &str = r#"#!/bin/bash
set -euo pipefail
if [ "$CONTROL_TIMING" = 1 ]; then
 [ "$PWD" = "$(cd "$CONTROL_ROOT/../umpire" && pwd -P)" ]
 [ "$*" = 'test-run --package control-checker' ]
 [ "$CARGO_HOME:$CARGO_TARGET_DIR" = "$CONTROL_MODEL/physical home:$CONTROL_MODEL/relative-target" ]
 [ "$UMPIRE_QUICK_CACHE:$UMPIRE_CACHE_DISABLE" = 0:1 ]
 [ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL" = false:false ]
 printf 'benchmark:%s:%s\n' "${UMPIRE_BACKENDS-unset}" "${UMPIRE_APALACHE-unset}" >> "$CONTROL_LOG"
 printf 'benchmark stdout diagnostic\n'; printf 'benchmark stderr diagnostic\n' >&2
 exit "${CONTROL_BENCHMARK_STATUS:-0}"
fi
[ "$PWD" = "$CONTROL_ROOT" ]
[ "$CARGO_HOME:$CARGO_TARGET_DIR" = relative-home:relative-target ]
printf 'framework|%s|%s|%s|%s\n' "$PWD" "$*" "${UMPIRE_QUICK_CACHE-unset}" "${UMPIRE_CACHE_DISABLE-unset}" >> "$CONTROL_LOG"
exit "${CONTROL_FRAMEWORK_STATUS:-0}"
"#;

const MISE: &str = r#"#!/bin/bash
set -euo pipefail
[ "$PWD" = "$CONTROL_ROOT" ]
[ "$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL" = false:false ]
if [ "$CONTROL_TIMING" = 1 ]; then
 [ "$*" = "exec -- cargo --config $CONTROL_MODEL/model/.cargo/config.toml run --manifest-path $CONTROL_MODEL/model/Cargo.toml --package temporal-xtask --bin temporal-xtask --profile dev -- time" ]
 if [ "${UMPIRE_BACKENDS:-}" = require ] || [ "${UMPIRE_APALACHE:-}" = require ]; then [ "$CARGO_BUILD_JOBS" = 1 ]; else [ "$CARGO_BUILD_JOBS" = 99 ]; fi
 [ "$CARGO_HOME:$CARGO_TARGET_DIR" = relative-home:relative-target ]
 exec "$CONTROL_TOOL" --exact model_controls::time_control_child --nocapture
fi
[ "$CARGO_HOME:$CARGO_TARGET_DIR" = relative-home:relative-target ]
printf 'temporal|%s|%s|%s|%s\n' "$PWD" "$*" "${UMPIRE_QUICK_CACHE-unset}" "${UMPIRE_CACHE_DISABLE-unset}" >> "$CONTROL_LOG"
if [[ "$*" = *' -- quick' ]] || [ "${UMPIRE_BACKENDS:-}" = require ] || [ "${UMPIRE_APALACHE:-}" = require ]; then [ "$CARGO_BUILD_JOBS" = 1 ]; else [ "$CARGO_BUILD_JOBS" = 99 ]; fi
exit "${CONTROL_TEMPORAL_STATUS:-0}"
"#;

fn timer_controls(shipping: &str, tool: &Path) -> io::Result<()> {
    let fixture = Fixture::new(shipping, tool, true)?;
    for conflict in ["absent", "environment", "command-line", "makeflags", "command-line-makeflags"]
    {
        for apalache in ["skip", "require"] {
            let mut command = fixture.command("make")?;
            match conflict {
                "absent" => {
                    command.arg(format!("UMPIRE_APALACHE={apalache}"));
                }
                "environment" => {
                    command.env("UMPIRE_BACKENDS", "skip").env("UMPIRE_APALACHE", apalache);
                }
                "command-line" => {
                    command.args(["UMPIRE_BACKENDS=skip", &format!("UMPIRE_APALACHE={apalache}")]);
                }
                "makeflags" => {
                    command.env(
                        "MAKEFLAGS",
                        format!("UMPIRE_BACKENDS=skip UMPIRE_APALACHE={apalache}"),
                    );
                }
                _ => {
                    command
                        .arg(format!("MAKEFLAGS=UMPIRE_BACKENDS=skip UMPIRE_APALACHE={apalache}"));
                }
            }
            command.arg("model-validate");
            fixture.check(
                &format!("validate precedence {conflict}/{apalache}"),
                &command,
                0,
                &format!("gate:require:{apalache}\nbenchmark:require:{apalache}\n"),
            )?;
        }
    }
    for mode in ["unset", "skip", "require"] {
        let mut command = fixture.command("make")?;
        if mode != "unset" {
            command.arg(format!("UMPIRE_BACKENDS={mode}"));
        }
        command.arg("model-time");
        fixture.check(
            &format!("time preserves {mode}"),
            &command,
            0,
            &format!("gate:{mode}:unset\nbenchmark:{mode}:unset\n"),
        )?;
    }
    let mut apalache = fixture.command("make")?;
    apalache.args(["UMPIRE_APALACHE=require", "model-time"]);
    fixture.check(
        "time Apalache bootstrap admission",
        &apalache,
        0,
        "gate:unset:require\nbenchmark:unset:require\n",
    )?;
    for (label, variable, expected) in [
        ("validate gate failure", "CONTROL_GATE_STATUS", "gate:require:unset\n"),
        (
            "validate benchmark failure",
            "CONTROL_BENCHMARK_STATUS",
            "gate:require:unset\nbenchmark:require:unset\n",
        ),
    ] {
        let mut command = fixture.command("make")?;
        command.arg("model-validate").env(variable, "37");
        fixture.check(label, &command, 2, expected)?;
    }
    for (label, variable, expected) in [
        (
            "raw benchmark failure",
            "CONTROL_BENCHMARK_STATUS",
            "gate:unset:unset\nbenchmark:unset:unset\n",
        ),
        ("raw bootstrap failure", "CONTROL_BOOTSTRAP_STATUS", "gate:unset:unset\n"),
    ] {
        let mut command = fixture.command(&fixture.tool)?;
        command
            .args(["--exact", "model_controls::time_control_child", "--nocapture"])
            .env(variable, "37");
        fixture.check(label, &command, 37, expected)?;
    }
    fixture.close()
}

fn wiring_controls(shipping: &str, tool: &Path) -> io::Result<()> {
    let fixture = Fixture::new(shipping, tool, false)?;
    let temporal = |arguments: &str, quick: &str, disabled: &str| {
        format!(
            "temporal|{}|exec -- cargo --config {}/model/.cargo/config.toml {arguments}|{quick}|{disabled}\n",
            fixture.root.display(),
            fixture.root.display()
        )
    };
    let run = |command: &str| {
        format!(
            "run --manifest-path {}/model/Cargo.toml --package temporal-xtask --bin temporal-xtask --profile dev -- {command}",
            fixture.root.display()
        )
    };
    for timed in ["0", "1"] {
        let mut command = fixture.command("make")?;
        command.args([
            &format!("UMPIRE_TIME_STEPS={timed}"),
            "UMPIRE_QUICK_CACHE=1",
            "UMPIRE_CACHE_DISABLE=0",
            "model-check",
        ]);
        fixture.check(
            "combined gate exactly once",
            &command,
            0,
            &temporal(&run("gate"), "0", "1"),
        )?;
    }
    for required in ["UMPIRE_BACKENDS=require", "UMPIRE_APALACHE=require"] {
        let mut command = fixture.command("make")?;
        command.args([required, "model-check"]);
        fixture.check(
            "required gate exactly once and one bootstrap job",
            &command,
            0,
            &temporal(&run("gate"), "0", "1"),
        )?;
    }
    let mut quick = fixture.command("make")?;
    quick.arg("model-quick");
    fixture.check(
        "quick exactly once and one bootstrap job",
        &quick,
        0,
        &temporal(&run("quick"), "1", "0"),
    )?;
    let mut producers = fixture.command("make")?;
    producers.args([
        "-j1",
        "UMPIRE_QUICK_CACHE=1",
        "UMPIRE_CACHE_DISABLE=0",
        "model-prerequisites",
    ]);
    fixture.check(
        "ordered producer prerequisites",
        &producers,
        0,
        "producer|control-go\nproducer|control-grpc\nproducer|control-api\n",
    )?;
    let mut failure = fixture.command("make")?;
    failure.arg("model-check").env("CONTROL_TEMPORAL_STATUS", "37");
    fixture.check("gate failure through Make", &failure, 2, &temporal(&run("gate"), "0", "1"))?;
    for action in ["fmt", "update"] {
        let framework = format!("framework|{}|{action}|1|0\n", fixture.root.display());
        let arguments = if action == "update" {
            run("test-run")
        } else {
            format!("fmt --manifest-path {}/model/Cargo.toml --all", fixture.root.display())
        };
        let mut command = fixture.command("make")?;
        command.arg(format!("model-{action}"));
        fixture.check(
            &format!("{action} framework before Temporal with inherited caches"),
            &command,
            0,
            &format!("{framework}{}", temporal(&arguments, "1", "0")),
        )?;
        command.env("CONTROL_FRAMEWORK_STATUS", "37");
        fixture.check(
            &format!("{action} framework failure stops Temporal"),
            &command,
            2,
            &framework,
        )?;
    }
    fixture.close()
}

/// Check maintained Make/CI wiring using fake child work, not live Model evidence.
fn run(root: &Path, tool: &Path) -> io::Result<i32> {
    let repository =
        root.parent().ok_or_else(|| io::Error::other("missing Model repository parent"))?;
    local_only(repository)?;
    let shipping = fs::read_to_string(repository.join("Makefile"))?;
    timer_controls(&shipping, tool)?;
    wiring_controls(&shipping, tool)?;
    writeln!(
        io::stdout(),
        "Model orchestration and recipe controls passed (fake children; no execution evidence)."
    )?;
    Ok(0)
}

#[test]
fn time_control_child() -> io::Result<()> {
    if env::var_os("CONTROL_NATIVE_TIMER_CHILD").as_deref() != Some(OsStr::new("1")) {
        return Ok(());
    }
    let fixture = env::var_os("CONTROL_ROOT")
        .ok_or_else(|| io::Error::other("missing timer control root"))?;
    std::process::exit(crate::gate::timing::run(&PathBuf::from(fixture).join("model"))?);
}

#[test]
fn shipping_model_recipes_with_disposable_children() -> io::Result<()> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    require(run(&root, &env::current_exe()?)? == 0, "native Model recipe controls")
}

#[test]
fn missing_sibling_stops_public_recipes_before_tool_execution() -> io::Result<()> {
    let repository = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let shipping = fs::read_to_string(repository.join("Makefile"))?;
    let fixture = Fixture::new(&shipping, &env::current_exe()?, false)?;
    fs::copy(
        repository.join("model/scripts/framework-root.sh"),
        fixture.root.join("model/scripts/framework-root.sh"),
    )?;
    for action in ["check", "quick", "fmt", "update", "time"] {
        let mut command = fixture.command("make")?;
        command.arg(format!("model-{action}"));
        let output = fixture.check(action, &command, 2, "")?;
        require(
            String::from_utf8_lossy(&output.stderr).contains("missing sibling Umpire checkout"),
            action,
        )?;
    }
    fixture.close()
}
