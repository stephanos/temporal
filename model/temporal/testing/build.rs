//! Discover Temporal Models through the shared fixture discovery source.

#[path = "../../umpire/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("../features")?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    umpire_lift::build::generate(
        &discovery::discover(std::path::Path::new("../features"))?,
        std::path::Path::new(&output),
    )
}
