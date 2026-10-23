use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, env, fs, path::Path};

fn sources(root: &Path, directory: &Path, files: &mut BTreeMap<String, Vec<u8>>) {
    for entry in fs::read_dir(directory).expect("host source directory") {
        let path = entry.expect("host source entry").path();
        if path.is_dir() {
            sources(root, &path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.insert(
                path.strip_prefix(root)
                    .expect("contained host source")
                    .to_string_lossy()
                    .replace('\\', "/"),
                fs::read(path).expect("host source bytes"),
            );
        }
    }
}

#[expect(
    clippy::needless_collect,
    reason = "Build feature bindings are sorted before hashing"
)]
fn main() {
    let directory = env::var_os("CARGO_MANIFEST_DIR").expect("host manifest directory");
    let root = Path::new(&directory);
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
            fs::read(root.join(name)).expect("host build identity input"),
        );
    }
    for name in [
        "internal/model/choice/schema/choicewire.json",
        "internal/model/choice/schema/choicewire.go.tmpl",
        "internal/model/choice/schema/choicewire_runtime.go.tmpl",
        "internal/model/choice/schema/choicewire_test.go.tmpl",
        "internal/model/runtimesource/gomad_semantics.go.txt",
        "internal/model/choice/trace.go",
        "internal/model/choice/tape.go",
    ] {
        let path = format!("../../{name}");
        files.insert(
            path.clone(),
            fs::read(root.join(path)).expect("choice implementation identity input"),
        );
    }
    let mut runtime = Sha256::new();
    runtime.update(b"gomad-wasm-runtime-implementation/v1\0");
    for name in [
        "overlay.go",
        "runtime_wasm.go.txt",
        "runtime_inspect_wasm.go.txt",
        "net_stream_wasm.go.txt",
        "net_stream_status_generated.go.txt",
    ] {
        let path = format!("../../internal/backends/wasm/toolchain/{name}");
        let bytes = fs::read(root.join(&path)).expect("runtime implementation identity input");
        runtime.update((bytes.len() as u64).to_be_bytes());
        runtime.update(&bytes);
        files.insert(path, bytes);
    }
    println!(
        "cargo:rustc-env=GOMAD_RUNTIME_IMPLEMENTATION_SHA256={:x}",
        runtime.finalize()
    );
    let mut hash = Sha256::new();
    hash.update(b"gomad.host-implementation/v1\0");
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
        "cargo:rustc-env=GOMAD_HOST_IMPLEMENTATION_SHA256={:x}",
        hash.finalize()
    );
}
