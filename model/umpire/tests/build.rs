//! Discover the shared end-to-end fixtures and lift their authored source.

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("fixtures")?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    umpire_lift::build::generate(
        &discovery::discover(std::path::Path::new("fixtures"))?,
        std::path::Path::new(&output),
    )
}
