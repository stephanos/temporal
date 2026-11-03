//! Temporal-owned entry point for the shared compiler inheritance control.
#![forbid(unsafe_code)]

#[path = "../../../../../umpire/tools/guest-driver/tests/identity/compiler_fixture_body.rs"]
mod implementation;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    implementation::run()
}
