use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, env, fs, path::Path};

fn sources(root: &Path, directory: &Path, files: &mut BTreeMap<String, Vec<u8>>) {
    for entry in fs::read_dir(directory).expect("read helper source directory") {
        let path = entry.expect("read helper source entry").path();
        if path.is_dir() {
            sources(root, &path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.insert(
                path.strip_prefix(root)
                    .expect("contained helper source")
                    .to_string_lossy()
                    .replace('\\', "/"),
                fs::read(&path).expect("read helper source"),
            );
        }
    }
}

#[expect(
    clippy::needless_collect,
    reason = "Feature bindings must be sorted before hashing the implementation identity."
)]
fn main() {
    let root = env::var_os("CARGO_MANIFEST_DIR").expect("helper manifest directory");
    let root = Path::new(&root);
    let mut files = BTreeMap::new();
    sources(root, &root.join("src"), &mut files);
    for name in [
        "build.rs",
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
    let source_inputs: String = files
        .iter()
        .map(|(name, bytes)| format!("{:x}\t{name}\n", Sha256::digest(bytes)))
        .collect();
    fs::write(
        Path::new(&env::var_os("OUT_DIR").expect("build output directory"))
            .join("measurement-source-inputs.tsv"),
        source_inputs,
    )
    .expect("measurement source closure");
    let mut hash = Sha256::new();
    hash.update(b"gomad.linked-implementation/v1\0");
    for (name, value) in env::vars()
        .filter(|(name, _)| {
            name.starts_with("CARGO_FEATURE_") || matches!(name.as_str(), "TARGET" | "PROFILE")
        })
        .collect::<BTreeMap<_, _>>()
    {
        hash.update(name.as_bytes());
        hash.update([0]);
        hash.update(value.as_bytes());
        hash.update([0]);
    }
    let compiler = std::process::Command::new(env::var_os("RUSTC").expect("Rust compiler"))
        .arg("--version")
        .output()
        .expect("Rust compiler identity");
    assert!(compiler.status.success(), "Rust compiler identity failed");
    hash.update(compiler.stdout);
    for (name, bytes) in files {
        println!("cargo:rerun-if-changed={name}");
        hash.update((name.len() as u64).to_be_bytes());
        hash.update(name.as_bytes());
        hash.update((bytes.len() as u64).to_be_bytes());
        hash.update(bytes);
    }
    println!(
        "cargo:rustc-env=GOMAD_LINKED_IMPLEMENTATION_SHA256={:x}",
        hash.finalize()
    );
}
