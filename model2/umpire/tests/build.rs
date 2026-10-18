//! Discover the shared end-to-end fixtures.

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build_at("fixtures")
}
