//! Real runner counts survive Cargo replacing the admitted native executable.
#![forbid(unsafe_code)]
use std::{
    fs,
    os::unix::fs::{MetadataExt, PermissionsExt},
    path::Path,
    process::Command,
};

fn fixture(root: &Path) -> crate::Result<()> {
    fs::create_dir_all(root.join("src"))?;
    fs::create_dir(root.join("bin"))?;
    fs::create_dir(root.join("receipts"))?;
    fs::write(
        root.join("Cargo.toml"),
        "[package]\nname='controller-count-control'\nversion='0.0.0'\nedition='2024'\n[workspace]\n",
    )?;
    fs::write(
        root.join("Cargo.lock"),
        "version = 4\n[[package]]\nname = \"controller-count-control\"\nversion = \"0.0.0\"\n",
    )?;
    fs::write(
        root.join("src/lib.rs"),
        r"/// ```
/// assert_eq!(2 + 2, 4);
/// ```
/// ```ignore
/// assert_eq!(2 + 2, 5);
/// ```
pub fn example() {}
#[test] fn runs() {}
#[test] fn another() {}
#[test] #[ignore] fn ignored() {}
",
    )?;
    let framework = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let config = fs::read_to_string(framework.join("mise.toml"))?;
    let tools = config.split_once("\n[task_config]").map_or(config.as_str(), |(tools, _)| tools);
    fs::write(root.join("mise.toml"), tools)?;
    fs::copy(framework.join("rust-toolchain.toml"), root.join("rust-toolchain.toml"))?;
    fs::write(
        root.join("bin/mise"),
        r#"#!/bin/sh
set -eu
case " $* " in
 *' cargo nextest list '*)
  if [ ! -e "$CONTROL_ROOT/replaced" ]; then
   cp "$CONTROL_CONTROLLER" "$CONTROL_CONTROLLER.next"
   mv "$CONTROL_CONTROLLER.next" "$CONTROL_CONTROLLER"
   printf replaced > "$CONTROL_ROOT/replaced"
  fi ;;
esac
exec "$CONTROL_REAL_MISE" "$@"
"#,
    )?;
    fs::set_permissions(root.join("bin/mise"), fs::Permissions::from_mode(0o755))?;
    Ok(())
}

#[test]
fn copied_controller_keeps_nextest_and_doctest_counts_after_atomic_self_replacement()
-> crate::Result<()> {
    let directory = tempfile::tempdir()?;
    let root = directory.path();
    fixture(root)?;
    let controller = root.join("controller");
    fs::copy(env!("CARGO_BIN_EXE_xtask"), &controller)?;
    let original = fs::metadata(&controller)?.ino();
    let real_mise = std::env::split_paths(&std::env::var_os("PATH").unwrap_or_default())
        .map(|path| path.join("mise"))
        .find(|path| path.is_file())
        .ok_or_else(|| {
            crate::Error::Invalid("pinned mise is required for replacement control".into())
        })?
        .canonicalize()?;
    let mut paths = vec![root.join("bin")];
    paths.extend(std::env::split_paths(&std::env::var_os("PATH").unwrap_or_default()));
    let mut command = Command::new(&controller);
    command
        .args(["gate", "--scheduled-stage", "tests"])
        .current_dir(root)
        .env(
            "PATH",
            std::env::join_paths(paths)
                .map_err(|error| crate::Error::Invalid(error.to_string()))?,
        )
        .env("CONTROL_ROOT", root)
        .env("CONTROL_CONTROLLER", &controller)
        .env("CONTROL_REAL_MISE", real_mise)
        .env("UMPIRE_TOOL_ROOT", root)
        .env("UMPIRE_GATE_EXECUTABLE", &controller)
        .env("UMPIRE_GATE_CONTROLLER_EXECUTABLE", &controller)
        .env("UMPIRE_GATE_RECEIPTS", root.join("receipts"))
        .env("UMPIRE_GATE_SCOPE", "framework")
        .env("UMPIRE_GATE_STAGE", "tests")
        .env("CARGO_TARGET_DIR", root.join("target"))
        .env("CARGO_BUILD_JOBS", "1")
        .env("NEXTEST_TEST_THREADS", "1")
        .env("MISE_TRUSTED_CONFIG_PATHS", root)
        .env("MISE_AUTO_INSTALL", "false")
        .env("MISE_EXEC_AUTO_INSTALL", "false");
    for name in [
        "UMPIRE_STANDALONE_CHILD",
        "UMPIRE_STANDALONE_MANIFEST",
        "UMPIRE_STANDALONE_OUTPUT",
        "UMPIRE_BACKENDS",
        "UMPIRE_APALACHE",
        "UMPIRE_TIME_STEPS",
        "NEXTEST_PROFILE",
    ] {
        command.env_remove(name);
    }
    let lease = umpire_backend_process::JvmLease::inherited_or_acquire()?;
    let output =
        super::gate_wiring::finish_owned(super::gate_wiring::spawn_owned(&command, root, &lease));
    assert!(output.status.success(), "{output:?}");
    assert!(root.join("replaced").is_file());
    assert_ne!(original, fs::metadata(&controller)?.ino());
    let receipt: serde_json::Value =
        serde_json::from_slice(&fs::read(root.join("receipts/framework/tests.json"))?)?;
    assert_eq!(receipt["counts"]["discovered"], 5, "{receipt}");
    assert_eq!(receipt["counts"]["executed"], 3, "{receipt}");
    assert_eq!(receipt["counts"]["ignored"], 2, "{receipt}");
    let observations =
        xtask::gate::work_counts::collect(&root.join("receipts/work/framework/tests"))?;
    assert_eq!(observations.discovered, Some(5));
    let finished = fs::read_dir(root.join("receipts/work/framework/tests"))?
        .map(|entry| {
            entry.and_then(|entry| {
                if entry.path().extension().is_some_and(|ext| ext == "event") {
                    fs::read_to_string(entry.path())
                } else {
                    Ok(String::new())
                }
            })
        })
        .collect::<std::io::Result<Vec<_>>>()?;
    assert_eq!(finished.iter().filter(|record| record.starts_with("1\ttests-finish\t")).count(), 2);
    Ok(())
}
