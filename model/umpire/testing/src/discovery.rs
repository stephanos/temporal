//! Build-time fixture discovery.

use std::{
    collections::BTreeMap,
    fmt::Write,
    fs, io,
    path::{Path, PathBuf},
};

/// A synthetic control: `model.rs` holds both the Model and the golden hook.
pub const MODEL: &[&str] = &["model.rs"];
/// A shared fixture: the Model in `model.rs`, its golden hook in `tests.rs` beside it.
#[allow(dead_code)]
pub const MODEL_AND_TESTS: &[&str] = &["model.rs", "tests.rs"];

/// A discovered fixture folder.
#[derive(Debug, Clone)]
pub struct Fixture {
    /// Folder name.
    pub name: String,
    /// Rust module the folder's `model.rs` compiles as.
    pub module: String,
    /// Folder path.
    pub path: PathBuf,
}

/// Discover every fixture directory in sorted order and validate its inputs and output folder.
pub fn discover(root: &Path, inputs: &[&str]) -> io::Result<Vec<Fixture>> {
    let mut fixtures = Vec::new();
    for entry in fs::read_dir(root)? {
        let entry = entry?;
        let path = entry.path();
        if !entry.file_type()?.is_dir() {
            return Err(io::Error::other(format!("unexpected fixture entry: {}", path.display())));
        }
        for required in inputs.iter().copied().chain(["expected"]) {
            let exists = if required == "expected" {
                path.join(required).is_dir()
            } else {
                path.join(required).is_file()
            };
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
        // Authors name a shared Model's module in its `tests.rs`; synthetic controls share their
        // crate with other items and keep the collision-free escaped form.
        let module = if inputs.contains(&"tests.rs") { module_name(&name) } else { escaped(&name) };
        fixtures.push(Fixture { name, module, path });
    }
    fixtures.sort_by(|a, b| a.name.cmp(&b.name));
    if fixtures.is_empty() {
        return Err(io::Error::other(format!("no fixtures in {}", root.display())));
    }
    Ok(fixtures)
}

/// Generate fixture modules, refusing incomplete fixture directories.
///
/// `model.rs` compiles as `<module>`, `tests.rs` as `<module>_tests`; the last input holds `run`.
pub fn generate(root: &Path, inputs: &[&str], output: &Path) -> io::Result<()> {
    generate_with_overrides(root, inputs, output, &BTreeMap::new())
}

/// Generate fixture modules with selected compile-only copies of authored inputs.
/// Override keys name a fixture folder and one of its required input filenames.
pub fn generate_with_overrides(
    root: &Path,
    inputs: &[&str],
    output: &Path,
    overrides: &BTreeMap<(String, String), PathBuf>,
) -> io::Result<()> {
    let fixtures = discover(root, inputs)?;
    for ((name, input), path) in overrides {
        if !fixtures.iter().any(|fixture| fixture.name == *name)
            || !inputs.contains(&input.as_str())
        {
            return Err(io::Error::other(format!(
                "unknown fixture input override: {name}/{input}"
            )));
        }
        if !path.is_file() {
            return Err(io::Error::other(format!(
                "fixture input override is not a file: {}",
                path.display()
            )));
        }
    }
    let mut source = String::new();
    for fixture in &fixtures {
        for input in inputs {
            let module = input_module(&fixture.module, input);
            let file = overrides
                .get(&(fixture.name.clone(), (*input).to_owned()))
                .cloned()
                .unwrap_or_else(|| fixture.path.join(input));
            let file = fs::canonicalize(file)?;
            let file = file.to_str().ok_or_else(|| io::Error::other("invalid fixture path"))?;
            writeln!(source, "#[allow(unused_imports, missing_docs, unreachable_pub, clippy::module_inception)]\n#[path = {file:?}] mod {module};").map_err(io::Error::other)?;
        }
    }
    source.push_str("fn registered() -> Vec<umpire_testing::Fixture> { vec![\n");
    let hook = inputs.last().ok_or_else(|| io::Error::other("no fixture inputs"))?;
    for fixture in &fixtures {
        let module = input_module(&fixture.module, hook);
        writeln!(source, "({:?}, {module}::run),", fixture.name).map_err(io::Error::other)?;
    }
    source.push_str("] }\n");
    fs::write(output, source)
}

fn input_module(module: &str, input: &str) -> String {
    match input.strip_suffix(".rs") {
        Some("model") | None => module.into(),
        Some(stem) => format!("{module}_{stem}"),
    }
}

/// The folder name itself when it is a plain lowercase identifier, an escaped `fixture_` name
/// otherwise; the two forms and their `_tests` siblings never collide.
fn module_name(name: &str) -> String {
    let plain = name.bytes().next().is_some_and(|byte| byte.is_ascii_lowercase())
        && name
            .bytes()
            .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'_')
        && !name.starts_with("fixture_")
        && !name.ends_with("_tests")
        && !KEYWORDS.contains(&name);
    if plain { name.into() } else { escaped(name) }
}

fn escaped(name: &str) -> String {
    let mut module = String::from("fixture_");
    for byte in name.bytes() {
        if byte.is_ascii_lowercase() || byte.is_ascii_digit() {
            module.push(char::from(byte));
        } else {
            write!(module, "_{byte:02x}").ok();
        }
    }
    module
}

const KEYWORDS: &[&str] = &[
    "abstract", "as", "async", "await", "become", "box", "break", "const", "continue", "crate",
    "do", "dyn", "else", "enum", "extern", "false", "final", "fn", "for", "gen", "if", "impl",
    "in", "let", "loop", "macro", "match", "mod", "move", "mut", "override", "priv", "pub", "ref",
    "return", "self", "static", "struct", "super", "trait", "true", "try", "type", "typeof",
    "unsafe", "unsized", "use", "virtual", "where", "while", "yield",
];

/// Cargo build-script entry point, shared as source to avoid a checker build-dependency cycle.
#[allow(dead_code, clippy::print_stdout)]
pub fn build() -> io::Result<()> {
    build_at("tests/fixtures", MODEL)
}

/// Cargo build-script entry point for a crate with an explicit fixture root and inputs.
#[allow(dead_code, clippy::print_stdout)]
pub fn build_at(relative: &str, inputs: &[&str]) -> io::Result<()> {
    println!("cargo:rerun-if-changed={relative}");
    let manifest = std::env::var_os("CARGO_MANIFEST_DIR")
        .ok_or_else(|| io::Error::other("missing CARGO_MANIFEST_DIR"))?;
    let output = std::env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("missing OUT_DIR"))?;
    generate(
        &PathBuf::from(manifest).join(relative),
        inputs,
        &PathBuf::from(output).join("fixtures.rs"),
    )
}
