use super::*;
use std::fmt::Write as _;
use std::{
    ffi::OsStr,
    io,
    os::unix::fs::{PermissionsExt, symlink},
    path::PathBuf,
    process::Output,
};

fn executable(path: &Path, source: &str) -> io::Result<()> {
    fs::write(path, source)?;
    fs::set_permissions(path, fs::Permissions::from_mode(0o755))
}

#[test]
fn isolated_build_child() -> Result<()> {
    let Some(root) = env::var_os("UMPIRE_GUEST_PATH_ROOT") else {
        return Ok(());
    };
    let driver = env::var_os("UMPIRE_GUEST_PATH_DRIVER")
        .ok_or_else(|| crate::Error::Invalid("missing child driver".into()))?;
    let status = run(Path::new(&root), Path::new(&driver))?;
    std::process::exit(status)
}

struct Fixture {
    scratch: tempfile::TempDir,
    root: PathBuf,
    caller: PathBuf,
    driver: PathBuf,
}
impl Fixture {
    fn new() -> Result<Self> {
        let scratch = tempfile::tempdir()?;
        let base = scratch.path().canonicalize()?;
        let root = base.join("framework with spaces");
        let caller = base.join("caller with spaces");
        fs::create_dir_all(&root)?;
        fs::create_dir_all(caller.join("physical cargo home"))?;
        fs::create_dir_all(caller.join("physical target"))?;
        fs::create_dir_all(base.join("bin"))?;
        symlink("physical cargo home", caller.join("cargo home"))?;
        symlink("physical target", caller.join("target link"))?;
        let driver = base.join("native guest driver");
        executable(
            &driver,
            r#"#!/bin/bash
set -euo pipefail
[[ "$PWD" = "$EXPECTED_ROOT" && "$CARGO_HOME" = "$EXPECTED_HOME" && "$CARGO_TARGET_DIR" = "$EXPECTED_TARGET" ]]
[[ "$MISE_AUTO_INSTALL" = false && "$MISE_EXEC_AUTO_INSTALL" = false ]]
[[ "$1" = guest ]]
printf 'guest|%s|%s\n' "$2" "$3" >> "$CONTROL_LOG"
[[ "${FAIL_GUEST:-}" != "$2" ]] || exit 37
"#,
        )?;
        executable(
            &base.join("bin/mise"),
            r#"#!/bin/bash
set -euo pipefail
[[ "$PWD" = "$EXPECTED_ROOT" && "$CARGO_HOME" = "$EXPECTED_HOME" && "$CARGO_TARGET_DIR" = "$EXPECTED_TARGET" ]]
[[ "$MISE_AUTO_INSTALL" = false && "$MISE_EXEC_AUTO_INSTALL" = false ]]
printf 'native|%s\n' "$*" >> "$CONTROL_LOG"
[[ "${FAIL_NATIVE:-}" != "$9" ]] || exit 42
"#,
        )?;
        Ok(Self { scratch, root, caller, driver })
    }
    fn command(
        &self,
        home: &OsStr,
        target: Option<&OsStr>,
        expected_home: &Path,
        expected_target: &Path,
    ) -> Result<Command> {
        let mut command = Command::new(env::current_exe()?);
        command
            .args(["--exact", "build_guests::path_controls::isolated_build_child", "--nocapture"])
            .current_dir(&self.caller)
            .env("UMPIRE_GUEST_PATH_ROOT", &self.root)
            .env("UMPIRE_GUEST_PATH_DRIVER", &self.driver)
            .env("PATH", format!("{}:/usr/bin:/bin", self.scratch.path().join("bin").display()))
            .env("CARGO_HOME", home)
            .env("EXPECTED_HOME", expected_home)
            .env("EXPECTED_ROOT", &self.root)
            .env("EXPECTED_TARGET", expected_target)
            .env("CONTROL_LOG", self.scratch.path().join("calls"))
            .env("MISE_AUTO_INSTALL", "true")
            .env("MISE_EXEC_AUTO_INSTALL", "true")
            .env_remove("BASH_ENV")
            .env_remove("FAIL_GUEST")
            .env_remove("FAIL_NATIVE");
        if let Some(target) = target {
            command.env("CARGO_TARGET_DIR", target);
        } else {
            command.env_remove("CARGO_TARGET_DIR");
        }
        Ok(command)
    }
    fn check(&self, output: &Output, status: i32, expected: &str) -> Result<()> {
        assert_eq!(
            output.status.code(),
            Some(status),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
        assert_eq!(fs::read_to_string(self.scratch.path().join("calls"))?, expected);
        Ok(())
    }
    fn expected(target: &Path) -> io::Result<String> {
        let mut result = String::new();
        for (package, artifact) in GUESTS {
            writeln!(
                result,
                "guest|{package}|{}/wasm32-wasip1/guest/{artifact}.wasm",
                target.display()
            )
            .map_err(io::Error::other)?;
        }
        writeln!(result, "native|exec -- cargo run --locked --profile guest -p umpire-caught-panic-guest --bin panic-control-reports -- {}/panic-native", target.display()).map_err(io::Error::other)?;
        writeln!(result, "native|exec -- cargo run --locked --profile guest -p umpire-control-guest --bin current-contract-reports -- {}/current-contract-native", target.display()).map_err(io::Error::other)?;
        Ok(result)
    }
}

#[test]
fn relative_absolute_default_and_symlink_cargo_paths_preserve_seven_children() -> Result<()> {
    let fixture = Fixture::new()?;
    let absolute_home = fixture.caller.join("cargo home");
    let physical_home = fixture.caller.join("physical cargo home");
    let nonexistent_home = fixture.caller.join("missing cargo");
    let absolute_target = fixture.caller.join("target link");
    let physical_target = fixture.caller.join("physical target");
    let default_target = fixture.root.join("target");
    for (home, expected_home) in [
        (OsStr::new("cargo home"), &physical_home),
        (absolute_home.as_os_str(), &physical_home),
        (OsStr::new("missing cargo"), &nonexistent_home),
    ] {
        for (target, expected_target) in [
            (Some(OsStr::new("target link")), &physical_target),
            (Some(absolute_target.as_os_str()), &physical_target),
            (None, &default_target),
        ] {
            let log = fixture.scratch.path().join("calls");
            fs::write(&log, "")?;
            let output = crate::gate::scheduler::execute(
                &fixture.command(home, target, expected_home, expected_target)?,
                false,
                false,
            )?;
            fixture.check(&output, 0, &Fixture::expected(expected_target)?)?;
        }
    }
    Ok(())
}

#[test]
fn guest_and_native_statuses_stop_without_following_children() -> Result<()> {
    let fixture = Fixture::new()?;
    let target = fixture.caller.join("physical target");
    let home = fixture.caller.join("physical cargo home");
    let expected = Fixture::expected(&target)?;
    fs::write(fixture.scratch.path().join("calls"), "")?;
    let output = crate::gate::scheduler::execute(
        fixture
            .command(OsStr::new("cargo home"), Some(OsStr::new("target link")), &home, &target)?
            .env("FAIL_GUEST", "umpire-control-guest"),
        false,
        false,
    )?;
    fixture.check(&output, 37, &expected.split_inclusive('\n').take(3).collect::<String>())?;
    fs::write(fixture.scratch.path().join("calls"), "")?;
    let output = crate::gate::scheduler::execute(
        fixture
            .command(OsStr::new("cargo home"), Some(OsStr::new("target link")), &home, &target)?
            .env("FAIL_NATIVE", "umpire-caught-panic-guest"),
        false,
        false,
    )?;
    fixture.check(&output, 42, &expected.split_inclusive('\n').take(6).collect::<String>())?;
    let retained = fixture.scratch.path().to_owned();
    drop(fixture);
    assert!(!retained.exists());
    Ok(())
}
