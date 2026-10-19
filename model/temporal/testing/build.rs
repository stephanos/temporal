//! Discover Temporal Models through the shared fixture discovery source.

#[path = "../../umpire/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("../features")
}
