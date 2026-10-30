#![allow(clippy::unwrap_used, clippy::expect_used)]
use std::process::Command;

#[test]
fn child_requires_copy_provenance_instead_of_parent_fallback() {
    let directory = tempfile::tempdir().unwrap();
    let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["gate", "--standalone-child"])
        .arg(directory.path().join("absent.json"))
        .env_remove("UMPIRE_STANDALONE_CHILD")
        .output()
        .unwrap();
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr).contains("standalone provenance"),
        "{output:?}"
    );
}

fn fixture() -> tempfile::TempDir {
    use std::{fs, path::Path};
    let directory = tempfile::tempdir().unwrap();
    let root = directory.path().join("framework");
    fs::create_dir_all(root.join("dev/gate/policy")).unwrap();
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    fs::copy(
        source.join("dev/gate/policy/framework-boundary.txt"),
        root.join("dev/gate/policy/framework-boundary.txt"),
    )
    .unwrap();
    fs::write(root.join("Cargo.toml"), "[workspace]\nmembers=['control']\n").unwrap();
    fs::create_dir_all(root.join("control/src")).unwrap();
    fs::write(
        root.join("control/Cargo.toml"),
        "[package]\nname='local-control'\nversion='0.0.0'\n",
    )
    .unwrap();
    fs::write(root.join("control/src/lib.rs"), "").unwrap();
    fs::write(root.join("mise.toml"), "[tools]\n").unwrap();
    fs::write(root.join("rust-toolchain.toml"), "[toolchain]\nchannel='stable'\n").unwrap();
    directory
}
fn refusal(root: &std::path::Path, reason: &str) {
    let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["gate", "--stage", "source-boundary"])
        .env("UMPIRE_TOOL_ROOT", root)
        .env("CARGO_TARGET_DIR", root.join("target"))
        .env_remove("UMPIRE_STANDALONE_CHILD")
        .output()
        .unwrap();
    assert!(!output.status.success(), "{output:?}");
    assert!(String::from_utf8_lossy(&output.stderr).contains(reason), "{output:?}");
}
#[test]
fn declared_existing_outside_dependency_fails_containment() {
    use std::fs;
    let directory = fixture();
    let root = directory.path().join("framework");
    let foreign = directory.path().join("foreign");
    fs::create_dir_all(foreign.join("src")).unwrap();
    fs::write(foreign.join("Cargo.toml"), "[package]\nname='outside-control'\nversion='0.0.0'\n")
        .unwrap();
    fs::write(foreign.join("src/lib.rs"), "").unwrap();
    fs::write(root.join("control/Cargo.toml"), "[package]\nname='local-control'\nversion='0.0.0'\n[target.'cfg(any())'.dependencies]\noutside-control={path='../../foreign'}\n").unwrap();
    refusal(&root, "framework dependency containment: path escapes framework root");
}
#[test]
fn policy_planted_identifier_fails_named_scanner() {
    use std::fs;
    let directory = fixture();
    let root = directory.path().join("framework");
    let policy: toml::Value = toml::from_str(
        &fs::read_to_string(root.join("dev/gate/policy/framework-boundary.txt")).unwrap(),
    )
    .unwrap();
    let vocabulary = policy["forbidden"][0].as_str().unwrap();
    fs::write(root.join("planted.rs"), format!("fn {vocabulary}_control() {{}}\n")).unwrap();
    refusal(&root, "framework forbidden name: planted.rs:1");
}
#[test]
fn manifest_root_escape_fails_source_containment() {
    let directory = fixture();
    let root = directory.path().join("framework");
    std::fs::write(root.join("planted.rs"), "fn probe() { let _ = std::path::Path::new(env!(\"CARGO_MANIFEST_DIR\")).join(\"../../../..\"); }\n").unwrap();
    refusal(&root, "framework source-path containment: path escapes framework root");
}
#[test]
fn copied_missing_local_configuration_refuses_parent_configuration() {
    let directory = fixture();
    let root = directory.path().join("framework");
    std::fs::rename(root.join("mise.toml"), directory.path().join("mise.toml")).unwrap();
    let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .arg("gate")
        .env("UMPIRE_TOOL_ROOT", &root)
        .env_remove("UMPIRE_STANDALONE_CHILD")
        .output()
        .unwrap();
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr)
            .contains("missing or escaping framework configuration: mise.toml"),
        "{output:?}"
    );
}
#[test]
fn child_cannot_reenter_copy_stage() {
    let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
        .args(["gate", "--stage", "standalone"])
        .env("UMPIRE_STANDALONE_CHILD", "1")
        .output()
        .unwrap();
    assert!(!output.status.success());
    assert!(
        String::from_utf8_lossy(&output.stderr).contains("standalone recursion refused"),
        "{output:?}"
    );
}
