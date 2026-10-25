//! Installer fixture controls use private pins; the production command has no overrides.
#![forbid(unsafe_code)]
#![allow(clippy::unwrap_used, clippy::expect_used)]
#[allow(dead_code, unreachable_pub)]
mod installer {
    include!("../src/fetch_apalache.rs");
    include!("apalache/cases.rs");
}
