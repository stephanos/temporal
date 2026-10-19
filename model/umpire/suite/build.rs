//! Discover the suite's message fixtures before compiling the runner.

#[path = "../testing/src/discovery.rs"]
#[allow(unreachable_pub)]
mod discovery;

fn main() -> std::io::Result<()> {
    discovery::build()
}
