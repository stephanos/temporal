use super::*;
use crate::script_control_support::Scratch;
use std::{
    ffi::OsStr,
    os::unix::fs::PermissionsExt,
    time::{Duration, SystemTime},
};

#[test]
#[ignore = "isolated actual timing controller"]
fn timing_child() -> io::Result<()> {
    let root = env::var_os("UMPIRE_TIMING_INTERRUPT_ROOT")
        .ok_or_else(|| io::Error::other("missing timing child root"))?;
    let pid = env::var_os("CONTROL_TIMING_PID")
        .ok_or_else(|| io::Error::other("missing timing PID record"))?;
    fs::write(pid, format!("{}\n", std::process::id()))?;
    std::process::exit(run(Path::new(&root))?)
}

fn executable(path: &Path, source: &str) -> io::Result<()> {
    fs::write(path, source)?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o755))
}

#[test]
fn interrupted_benchmark_preserves_original_source_bytes() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let root = scratch.0.join("model");
    for directory in [root.join("scripts"), root.join("umpire"), scratch.0.join("bin")] {
        fs::create_dir_all(directory)?;
    }
    let source = root.join("arbitrary source");
    let original = b"original arbitrary syntax\n\xff";
    fs::write(&source, original)?;
    let old_mtime = SystemTime::UNIX_EPOCH + Duration::from_secs(10);
    fs::OpenOptions::new().write(true).open(&source)?.set_modified(old_mtime)?;
    executable(&scratch.0.join("bin/make"), "#!/bin/sh\nexit 0\n")?;
    executable(
        &root.join("scripts/xtask.sh"),
        "#!/bin/sh\nprintf '%s\\n' \"$CONTROL_TIMING_FRAMEWORK\"\n",
    )?;
    let framework = scratch.0.join("bin/framework");
    executable(
        &framework,
        r#"#!/bin/sh
set -eu
[ "$*" = 'test-run --package control-package' ]
printf 'benchmark reached\n' > "$CONTROL_TIMING_REACHED"
IFS= read -r timing_pid < "$CONTROL_TIMING_PID"
kill -KILL "$timing_pid"
"#,
    )?;
    let mut command = Command::new(env::current_exe()?);
    command
        .args([
            "--exact",
            "gate::timing::interrupt_controls::timing_child",
            "--nocapture",
            "--ignored",
        ])
        .current_dir(&scratch.0)
        .env("UMPIRE_TIMING_INTERRUPT_ROOT", &root)
        .env("CONTROL_TIMING_PID", scratch.0.join("timing.pid"))
        .env("CONTROL_TIMING_REACHED", scratch.0.join("benchmark.reached"))
        .env("CONTROL_TIMING_FRAMEWORK", framework)
        .env("UMPIRE_INNER_SOURCE", OsStr::new("arbitrary source"))
        .env("UMPIRE_INNER_CRATE", "control-package")
        .env("CARGO_HOME", scratch.0.join("cargo home"))
        .env("CARGO_TARGET_DIR", scratch.0.join("target"))
        .env(
            "PATH",
            env::join_paths([scratch.0.join("bin"), "/usr/bin".into(), "/bin".into()])
                .map_err(io::Error::other)?,
        )
        .env_remove("UMPIRE_BACKENDS")
        .env_remove("UMPIRE_APALACHE")
        .env_remove("BASH_ENV");
    let output = xtask::gate::scheduler::execute(&command, false, false)?;
    assert_eq!(output.status.signal(), Some(9), "{output:?}");
    assert_eq!(fs::read(scratch.0.join("benchmark.reached"))?, b"benchmark reached\n");
    assert_eq!(fs::read(&source)?, original);
    assert!(fs::metadata(source)?.modified()? > old_mtime);
    Ok(())
}
