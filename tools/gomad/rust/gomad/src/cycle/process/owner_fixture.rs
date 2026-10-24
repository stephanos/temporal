#![allow(
    dead_code,
    unused_imports,
    reason = "Standalone fixture includes the actual private production module graph"
)]
include!("../../lib.rs");

fn main() {
    cycle::process::tests::fixture_main();
}
