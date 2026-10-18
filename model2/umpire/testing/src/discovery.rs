//! Build-time fixture discovery.

use std::{
    fmt::Write,
    fs, io,
    path::{Path, PathBuf},
};

/// Discover every fixture directory in sorted order and validate its input and output folder.
pub fn discover(root: &Path) -> io::Result<Vec<(String, PathBuf)>> {
    let mut fixtures = Vec::new();
    for entry in fs::read_dir(root)? {
        let entry = entry?;
        let path = entry.path();
        if !entry.file_type()?.is_dir() {
            return Err(io::Error::other(format!("unexpected fixture entry: {}", path.display())));
        }
        for (required, exists) in [
            ("model.rs", path.join("model.rs").is_file()),
            ("expected", path.join("expected").is_dir()),
        ] {
            if !exists {
                return Err(io::Error::other(format!(
                    "fixture {} is missing {required}",
                    path.display()
                )));
            }
        }
        let name = entry
            .file_name()
            .into_string()
            .map_err(|_| io::Error::other("invalid fixture name"))?;
        fixtures.push((name, path));
    }
    fixtures.sort_by(|a, b| a.0.cmp(&b.0));
    if fixtures.is_empty() {
        return Err(io::Error::other(format!("no fixtures in {}", root.display())));
    }
    Ok(fixtures)
}

/// Generate fixture modules, refusing incomplete fixture directories.
pub fn generate(root: &Path, output: &Path) -> io::Result<()> {
    let fixtures = discover(root)?;
    let mut source = String::new();
    for (name, path) in &fixtures {
        let model = fs::canonicalize(path.join("model.rs"))?;
        let model = model.to_str().ok_or_else(|| io::Error::other("invalid fixture path"))?;
        writeln!(source, "#[allow(unused_imports, missing_docs, unreachable_pub)]\n#[path = {model:?}] mod fixture_{name};").map_err(io::Error::other)?;
    }
    source.push_str("fn registered() -> Vec<umpire_testing::Fixture> { vec![\n");
    for (name, _) in &fixtures {
        writeln!(source, "({name:?}, fixture_{name}::run),").map_err(io::Error::other)?;
    }
    source.push_str("] }\n");
    fs::write(output, source)
}

/// Cargo build-script entry point, shared as source to avoid a checker build-dependency cycle.
#[allow(dead_code, clippy::print_stdout)]
pub fn build() -> io::Result<()> {
    build_at("tests/fixtures")
}

/// Cargo build-script entry point for a crate with an explicit fixture root.
#[allow(dead_code, clippy::print_stdout)]
pub fn build_at(relative: &str) -> io::Result<()> {
    println!("cargo:rerun-if-changed={relative}");
    let manifest = std::env::var_os("CARGO_MANIFEST_DIR")
        .ok_or_else(|| io::Error::other("missing CARGO_MANIFEST_DIR"))?;
    let output = std::env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("missing OUT_DIR"))?;
    generate(&PathBuf::from(manifest).join(relative), &PathBuf::from(output).join("fixtures.rs"))
}
