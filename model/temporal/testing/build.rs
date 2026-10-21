//! Discover Temporal Models through the shared fixture discovery source.

#[path = "../../umpire/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("../feature", discovery::MODEL_AND_TESTS)?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    let fixtures =
        discovery::discover(std::path::Path::new("../feature"), discovery::MODEL_AND_TESTS)?
            .into_iter()
            .map(|fixture| (fixture.name, fixture.module, fixture.path))
            .collect::<Vec<_>>();
    umpire_lift::build::generate(&fixtures, std::path::Path::new(&output))
}
