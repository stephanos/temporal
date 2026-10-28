//! Model gate timing and source timestamp invalidation for the package benchmark.
use std::{
    env, fs,
    io::{self, Write},
    os::unix::{ffi::OsStrExt, process::ExitStatusExt},
    path::Path,
    process::Command,
    time::Instant,
};

fn touch_source(path: &Path) -> io::Result<()> {
    let file = fs::OpenOptions::new().write(true).open(path)?;
    file.set_modified(std::time::SystemTime::now())
}

pub(super) fn warnings(
    writer: &mut impl Write,
    total: f64,
    inner: f64,
    gate_budget: Option<&str>,
    inner_budget: Option<&str>,
) -> io::Result<()> {
    for (label, elapsed, value, default) in
        [("gate", total, gate_budget, 60.0), ("inner loop", inner, inner_budget, 10.0)]
    {
        let budget = value.and_then(|value| value.parse::<f64>().ok()).unwrap_or(default);
        if elapsed > budget {
            writeln!(writer, "note: {label} over soft target (fn-210 R30)")?;
        }
    }
    Ok(())
}

fn benchmark_command(
    root: &Path,
    framework: &std::ffi::OsStr,
    package: &str,
) -> io::Result<Command> {
    let mut command = Command::new(framework);
    command
        .args(["test-run", "--package", package])
        .current_dir(root.join("umpire"))
        .env("UMPIRE_TOOL_ROOT", root.join("umpire"))
        .env("UMPIRE_FRAMEWORK_ROOT", root.join("umpire"));
    super::admit_command("tests", &mut command)?;
    Ok(command)
}

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    let repository = root.parent().ok_or_else(|| io::Error::other("missing repository root"))?;
    let budget = |key, default: &str| {
        env::var(key).ok().filter(|value| !value.is_empty()).unwrap_or_else(|| default.into())
    };
    let gate_budget = budget("UMPIRE_GATE_BUDGET", "60");
    let inner_budget = budget("UMPIRE_INNER_BUDGET", "10");
    let package = budget("UMPIRE_INNER_CRATE", "umpire-checker");
    let source = env::var_os("UMPIRE_INNER_SOURCE")
        .filter(|value| !value.is_empty())
        .map_or_else(|| root.join("umpire/backends/native/src/lib.rs"), |value| root.join(value));
    writeln!(io::stdout(), "{:<32} {:>7}", "step (warm)", "time")?;
    writeln!(
        io::stdout(),
        "gate command: UMPIRE_TIME_STEPS=1 make -C {} model-check",
        repository.display()
    )?;
    let mut command = Command::new("make");
    command.arg("-C").arg(repository).arg("model-check").env("UMPIRE_TIME_STEPS", "1");
    super::admit_command("gate", &mut command)?;
    let started = Instant::now();
    let code = xtask::gate::timing::execute("gate total", &command, false, None)?;
    if code != 0 {
        return Ok(code);
    }
    let total = started.elapsed().as_secs_f64();
    let bootstrap = super::framework_executable(root)?;
    if !bootstrap.status.success() {
        return Ok(bootstrap
            .status
            .code()
            .unwrap_or_else(|| 128 + bootstrap.status.signal().unwrap_or(0)));
    }
    let framework = std::ffi::OsStr::from_bytes(
        bootstrap.stdout.strip_suffix(b"\n").unwrap_or(&bootstrap.stdout),
    );
    let command = benchmark_command(root, framework, &package)?;
    writeln!(io::stdout(), "inner source: {}", source.display())?;
    writeln!(
        io::stdout(),
        "inner command: {} test-run --package {package}",
        framework.to_string_lossy()
    )?;
    touch_source(&source)?;
    let started = Instant::now();
    let result = xtask::gate::timing::execute("inner loop (edit+test)", &command, true, None);
    let inner = started.elapsed().as_secs_f64();
    let code = result?;
    if code != 0 {
        return Ok(code);
    }
    writeln!(io::stdout(), "{:<32} {:6.3}s  (soft target {}s)", "gate total", total, gate_budget)?;
    writeln!(io::stdout(), "{:<32} {:6.3}s  (soft target {}s)", "inner loop", inner, inner_budget)?;
    warnings(&mut io::stdout(), total, inner, Some(&gate_budget), Some(&inner_budget))?;
    Ok(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn benchmark_selects_authoritative_package_runner_and_disables_caches() -> io::Result<()> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
        let command =
            benchmark_command(&root, std::ffi::OsStr::new("framework"), "selected-package")?;
        assert_eq!(
            command.get_args().collect::<Vec<_>>(),
            ["test-run", "--package", "selected-package"]
        );
        assert_eq!(command.get_current_dir(), Some(root.join("umpire").as_path()));
        for (key, expected) in [
            ("UMPIRE_QUICK_CACHE", "0"),
            ("UMPIRE_CACHE_DISABLE", "1"),
            ("MISE_AUTO_INSTALL", "false"),
            ("MISE_EXEC_AUTO_INSTALL", "false"),
        ] {
            assert!(
                command
                    .get_envs()
                    .any(|(name, value)| name == key
                        && value == Some(std::ffi::OsStr::new(expected)))
            );
        }
        Ok(())
    }

    #[test]
    fn source_edit_restores_original_content_after_failed_child() -> io::Result<()> {
        let directory =
            xtask::gate::scheduler::directory(&env::temp_dir().join("model-timing-controls"))?;
        let path = directory.join("source.rs");
        fs::write(&path, b"original content\n")?;
        let old_mtime = std::time::SystemTime::UNIX_EPOCH + std::time::Duration::from_secs(10);
        fs::OpenOptions::new().write(true).open(&path)?.set_modified(old_mtime)?;
        touch_source(&path)?;
        assert_eq!(fs::read(&path)?, b"original content\n");
        assert!(fs::metadata(&path)?.modified()? > old_mtime);
        let mut command = Command::new("/bin/sh");
        command.args(["-c", "exit 37"]);
        assert_eq!(xtask::gate::timing::execute("failed edit", &command, true, None)?, 37);
        assert_eq!(fs::read(&path)?, b"original content\n");
        fs::remove_dir_all(directory)?;
        Ok(())
    }
}

#[cfg(test)]
#[path = "timing_interrupt_controls.rs"]
mod interrupt_controls;
