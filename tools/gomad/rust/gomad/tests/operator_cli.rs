//! End-to-end operator command refusals through the built `gomad` binary.
use std::process::Command;

/// Original `TestQualifyValidatesSpecBeforeResolvingInstallation`: grammar and
/// spec errors exit 2 before the selected installation is resolved, with no
/// response on stdout.
#[test]
fn qualify_validates_spec_before_resolving_installation() {
    let root = std::env::temp_dir().join(format!("gomad-operator-cli-{}", std::process::id()));
    std::fs::create_dir_all(&root).unwrap();
    let invalid = root.join("invalid-spec.json");
    std::fs::write(&invalid, br#"{"Options":{}}"#).unwrap();
    let invalid = invalid.to_str().unwrap();
    for arguments in [
        vec![],
        vec!["--spec", "/missing/spec"],
        vec!["--spec", invalid],
        vec!["--spec", invalid, "extra"],
    ] {
        let output = Command::new(env!("CARGO_BIN_EXE_gomad"))
            .args([
                "--installation",
                "/missing/installation-must-not-be-resolved",
                "qualify",
            ])
            .args(&arguments)
            .env(
                "GOMAD_WASM_INSTALLATION",
                "/missing/environment-must-not-be-resolved",
            )
            .output()
            .unwrap();
        let stderr = String::from_utf8_lossy(&output.stderr);
        assert_eq!(output.status.code(), Some(2), "{arguments:?}: {stderr}");
        assert!(output.stdout.is_empty(), "{arguments:?}");
        assert!(
            !stderr.contains("must-not-be-resolved") && !stderr.contains("unavailable"),
            "{arguments:?}: {stderr}"
        );
    }
    std::fs::remove_dir_all(&root).unwrap();
}
