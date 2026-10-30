//! Discover Temporal Models through the shared fixture discovery source.

#![forbid(unsafe_code)]

#[path = "../../umpire/dev/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    println!("cargo:rerun-if-changed=../shared/protocol");
    discovery::build_at("../feature", discovery::MODEL_AND_TESTS)?;
    let output =
        std::env::var_os("OUT_DIR").ok_or_else(|| std::io::Error::other("missing OUT_DIR"))?;
    discovery::generate_lifted(
        std::path::Path::new("../feature"),
        &std::path::Path::new(&output).join("lifted.rs"),
    )?;
    discovery::generate(
        std::path::Path::new("../shared/protocol"),
        discovery::MODEL_AND_TESTS,
        &std::path::Path::new(&output).join("protocol_fixtures.rs"),
    )
}
