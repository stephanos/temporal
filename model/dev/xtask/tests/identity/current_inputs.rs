use std::{fs, path::Path, process::Command};

#[test]
fn native_toml_accepts_current_documents_and_rejects_malformed_input() {
    let model = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize().unwrap();
    let repo = model.parent().unwrap();
    for path in [
        model.join("Cargo.toml"),
        model.join("Cargo.lock"),
        model.join("rust-toolchain.toml"),
        repo.join("mise.toml"),
    ] {
        let content = fs::read_to_string(&path).unwrap();
        toml::from_str::<toml::Value>(&content)
            .unwrap_or_else(|error| panic!("{}: {error}", path.display()));
    }
    let output = Command::new("rustc").args(["--print", "sysroot"]).output().unwrap();
    assert!(output.status.success());
    let sysroot = String::from_utf8(output.stdout).unwrap();
    let lock = Path::new(sysroot.trim()).join("lib/rustlib/src/rust/library/Cargo.lock");
    let value: toml::Value = toml::from_str(&fs::read_to_string(lock).unwrap()).unwrap();
    assert_ne!(value["package"].as_array().unwrap().as_slice(), []);
    for bad in ["[unfinished", "a = 1\na = 2", "a = [1,"] {
        assert!(toml::from_str::<toml::Value>(bad).is_err(), "{bad}");
    }
}
