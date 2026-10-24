//! Discover all checker fixtures before compiling the runner.

#![allow(clippy::disallowed_methods, clippy::disallowed_types)]

#[path = "../../dev/testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build()
}
