//! A renamed facade and renamed extension retain generated capability paths and admission.

#![forbid(unsafe_code)]

#[test]
fn renamed_dependencies_preserve_realization_and_enum_admission() {
    let fixture = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/renamed");
    let target = std::env::var_os("CARGO_TARGET_DIR")
        .map_or_else(|| fixture.join("target"), std::path::PathBuf::from)
        .join("tests/renamed-realization");
    for invalid in [false, true] {
        let mut command = std::process::Command::new(env!("CARGO"));
        command.current_dir(&fixture).env("CARGO_TARGET_DIR", &target).args([
            "check",
            "--offline",
            "--locked",
        ]);
        if invalid {
            command.args(["--features", "invalid-enum"]);
        }
        let output = command
            .output()
            .unwrap_or_else(|error| panic!("cannot check renamed fixture: {error}"));
        let stderr = String::from_utf8_lossy(&output.stderr);
        if invalid {
            assert!(!output.status.success(), "renamed generated-enum wildcard was admitted");
            assert!(stderr.contains("generated enum matches must name every variant"), "{stderr}");
        } else {
            assert!(output.status.success(), "{stderr}");
        }
    }
}
