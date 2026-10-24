//! Discover Temporal Models through the shared fixture discovery source.

#[path = "../../umpire/dev/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("../feature", discovery::MODEL_AND_TESTS)?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    discovery::generate_lifted(
        std::path::Path::new("../feature"),
        &std::path::Path::new(&output).join("lifted.rs"),
    )
}
