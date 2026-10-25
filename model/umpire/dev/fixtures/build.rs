//! Discover the shared end-to-end fixtures and their lift hooks.

#![forbid(unsafe_code)]

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

#[allow(clippy::print_stdout, reason = "Cargo build-script rerun protocol")]
fn main() -> std::io::Result<()> {
    println!("cargo:rerun-if-changed=fixtures");
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    let output = std::path::Path::new(&output);
    discovery::generate_lifted(std::path::Path::new("fixtures"), &output.join("lifted.rs"))?;
    discovery::generate_interpreted(
        std::path::Path::new("fixtures"),
        &output.join("interpreted.rs"),
    )?;
    discovery::generate(
        std::path::Path::new("fixtures"),
        discovery::MODEL_AND_TESTS,
        &output.join("fixtures.rs"),
    )
}
