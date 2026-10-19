//! Discover the shared end-to-end fixtures and lift their authored source.

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("fixtures", discovery::MODEL_AND_TESTS)?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    let fixtures =
        discovery::discover(std::path::Path::new("fixtures"), discovery::MODEL_AND_TESTS)?
            .into_iter()
            .map(|fixture| (fixture.name, fixture.module, fixture.path))
            .collect::<Vec<_>>();
    umpire_lift::build::generate(&fixtures, std::path::Path::new(&output))
}
