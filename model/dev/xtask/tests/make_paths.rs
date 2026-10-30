use std::{env, fs, os::unix::fs::symlink, path::Path, process::Command};

#[test]
fn shipping_framework_make_bootstrap_preserves_relative_cargo_paths() {
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..").canonicalize().unwrap();
    let caller = env::current_dir().unwrap();
    let home = umpire_guest_driver::context::cargo_home(&caller).unwrap();
    let target = env::var_os("CARGO_TARGET_DIR")
        .map_or_else(|| source.join("model/target"), |value| caller.join(value));
    fs::create_dir_all(&target).unwrap();
    let target = target.canonicalize().unwrap();
    let f = super::Fixture::new();
    fs::create_dir_all(f.root.join("make/model")).unwrap();
    let root = f.root.join("make").canonicalize().unwrap();
    symlink(source.join("model/umpire"), root.join("model/umpire")).unwrap();
    symlink(&home, root.join("cargo-home")).unwrap();
    symlink(&target, root.join("build-output")).unwrap();
    let makefile = fs::read_to_string(source.join("Makefile")).unwrap();
    let definition =
        makefile.lines().find(|line| line.starts_with("RUST_MODEL_FRAMEWORK =")).unwrap();
    fs::write(
        root.join("Makefile"),
        format!(
            "{definition}\ncontrol:\n\t@$(RUST_MODEL_FRAMEWORK) gate --stage source-boundary\n"
        ),
    )
    .unwrap();
    let mise = env::split_paths(&env::var_os("PATH").unwrap())
        .map(|path| path.join("mise"))
        .find(|path| path.is_file())
        .unwrap()
        .canonicalize()
        .unwrap();
    super::support::executable(
        &root.join("mise"),
        r#"#!/bin/bash
set -euo pipefail
printf '%s|%s|%s|%s\n' "$PWD" "$CARGO_TARGET_DIR" "$CARGO_HOME" "$*" >> "$PATH_CONTROL_LOG"
exec "$PATH_CONTROL_MISE" "$@"
"#,
    );
    let output = Command::new("make")
        .arg("--no-print-directory")
        .arg("control")
        .current_dir(&root)
        .env("PATH", format!("{}:{}", root.display(), env::var("PATH").unwrap()))
        .env("PATH_CONTROL_MISE", mise)
        .env("PATH_CONTROL_LOG", root.join("calls"))
        .env("CARGO_TARGET_DIR", "build-output")
        .env("CARGO_HOME", "cargo-home")
        .env("CARGO_NET_OFFLINE", "true")
        .env_remove("BASH_ENV")
        .env_remove("UMPIRE_TOOL_ROOT")
        .env_remove("MAKEFLAGS")
        .env_remove("MFLAGS")
        .env_remove("MAKEOVERRIDES")
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "stdout: {}\nstderr: {}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    let calls = fs::read_to_string(root.join("calls")).unwrap();
    let framework = source.join("model/umpire");
    let bootstrap = format!(
        "{}|{}|{}|exec -- cargo build --locked",
        root.join("model/umpire").display(),
        root.join("build-output").display(),
        home.display()
    );
    let metadata = format!(
        "{}|{}|{}|exec -- cargo metadata --locked --format-version 1",
        framework.display(),
        root.join("build-output").display(),
        home.display()
    );
    assert!(calls.lines().any(|line| line.starts_with(&bootstrap)), "{calls}");
    assert!(calls.lines().any(|line| line == metadata), "{calls}");
    assert!(target.join("xtask-bootstrap/debug/xtask").is_file());
}

#[test]
fn shipping_temporal_make_loads_the_workspace_linker_config_without_changing_caller_paths() {
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..").canonicalize().unwrap();
    let config: toml::Value =
        toml::from_str(&fs::read_to_string(source.join("model/.cargo/config.toml")).unwrap())
            .unwrap();
    let targets = config["target"].as_table().unwrap();
    assert_eq!(targets.len(), 2);
    for target in ["aarch64-unknown-linux-gnu", "x86_64-unknown-linux-gnu"] {
        let settings = targets[target].as_table().unwrap();
        assert_eq!(settings.len(), 1);
        assert_eq!(settings["linker"].as_str(), Some("umpire/scripts/mold-linker.sh"));
    }
    let f = super::Fixture::new();
    let root = f.root.join("make");
    fs::create_dir_all(root.join("model")).unwrap();
    let makefile = fs::read_to_string(source.join("Makefile")).unwrap();
    let definition = makefile.lines().find(|line| line.starts_with("RUST_MODEL_CARGO =")).unwrap();
    let recipe =
        makefile.lines().find(|line| line.starts_with('\t') && line.ends_with("-- quick")).unwrap();
    fs::write(root.join("Makefile"), format!("{definition}\ncontrol:\n{recipe}\n")).unwrap();
    super::support::executable(
        &root.join("mise"),
        "#!/bin/sh\nprintf '%s|%s|%s|%s|%s|%s\\n' \"$PWD\" \"$CARGO_HOME\" \"$CARGO_TARGET_DIR\" \"$MISE_AUTO_INSTALL\" \"$MISE_EXEC_AUTO_INSTALL\" \"$*\" > \"$MAKE_PATH_LOG\"\n",
    );
    let output = Command::new("make")
        .args(["--no-print-directory", "control"])
        .current_dir(&root)
        .env("PATH", format!("{}:{}", root.display(), env::var("PATH").unwrap()))
        .env("MAKE_PATH_LOG", root.join("calls"))
        .env("CARGO_HOME", "caller-home")
        .env("CARGO_TARGET_DIR", "caller-target")
        .env_remove("BASH_ENV")
        .env_remove("MAKEFLAGS")
        .env_remove("MFLAGS")
        .env_remove("MAKEOVERRIDES")
        .output()
        .unwrap();
    assert!(output.status.success(), "{output:?}");
    assert_eq!(
        fs::read_to_string(root.join("calls")).unwrap(),
        format!(
            "{}|caller-home|caller-target|false|false|exec -- cargo --config {}/model/.cargo/config.toml run --manifest-path {}/model/Cargo.toml --package temporal-xtask --bin temporal-xtask --profile dev -- quick\n",
            root.display(),
            root.display(),
            root.display()
        )
    );
}
