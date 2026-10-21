//! Discover the shared end-to-end fixtures and lift their authored source.

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

#[allow(clippy::print_stdout, reason = "Cargo build-script rerun protocol")]
fn main() -> std::io::Result<()> {
    println!("cargo:rerun-if-changed=fixtures");
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    let fixtures =
        discovery::discover(std::path::Path::new("fixtures"), discovery::MODEL_AND_TESTS)?
            .into_iter()
            .map(|fixture| (fixture.name, fixture.module, fixture.path))
            .collect::<Vec<_>>();
    let output = std::path::Path::new(&output);
    let overrides = umpire_lift::build::prepare(&fixtures, output)?;
    discovery::generate_with_overrides(
        std::path::Path::new("fixtures"),
        discovery::MODEL_AND_TESTS,
        &output.join("fixtures.rs"),
        &overrides,
    )?;
    umpire_lift::build::generate(&fixtures, output)
}
