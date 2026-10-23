use sha2::{Digest, Sha256};
use std::fmt::Write as _;
use std::{collections::BTreeMap, env, fs, path::Path};

mod build_identity;

fn main() {
    let root = env::var_os("CARGO_MANIFEST_DIR").expect("helper manifest directory");
    let root = Path::new(&root);
    let mut files = BTreeMap::new();
    build_identity::sources(root, &root.join("src"), &mut files);
    for name in [
        "build.rs",
        "build_identity.rs",
        "Cargo.toml",
        "../Cargo.toml",
        "../Cargo.lock",
        "../rust-toolchain.toml",
    ] {
        files.insert(
            name.into(),
            fs::read(root.join(name)).expect("read helper build identity input"),
        );
    }
    let mut source_inputs = String::new();
    for (name, bytes) in &files {
        writeln!(&mut source_inputs, "{:x}\t{name}", Sha256::digest(bytes))
            .expect("source input manifest string");
    }
    fs::write(
        Path::new(&env::var_os("OUT_DIR").expect("build output directory"))
            .join("measurement-source-inputs.tsv"),
        source_inputs,
    )
    .expect("measurement source closure");
    let bindings = env::vars()
        .filter(|(name, _)| {
            name.starts_with("CARGO_FEATURE_") || matches!(name.as_str(), "TARGET" | "PROFILE")
        })
        .collect::<BTreeMap<_, _>>();
    let compiler = std::process::Command::new(env::var_os("RUSTC").expect("Rust compiler"))
        .arg("--version")
        .output()
        .expect("Rust compiler identity");
    assert!(compiler.status.success(), "Rust compiler identity failed");
    let output = env::var_os("OUT_DIR").expect("build output directory");
    fs::write(
        Path::new(&output).join("identity-compiler.txt"),
        &compiler.stdout,
    )
    .expect("compiler identity capture");
    fs::write(
        Path::new(&output).join("identity-bindings.tsv"),
        bindings
            .iter()
            .map(|(name, value)| format!("{name}\t{value}\n"))
            .collect::<String>(),
    )
    .expect("feature identity capture");
    for name in files.keys() {
        println!("cargo:rerun-if-changed={name}");
    }
    let identity = build_identity::implementation(&files, &bindings, &compiler.stdout)
        .expect("complete helper implementation identity");
    println!("cargo:rustc-env=GOMAD_LINKED_IMPLEMENTATION_SHA256={identity}");
}
