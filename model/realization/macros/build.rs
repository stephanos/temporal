//! Link the shared upstream descriptors without depending on generated API code.

#![forbid(unsafe_code)]

use std::{env, io, path::PathBuf};

#[path = "../../api/system.rs"]
mod system;

fn main() -> io::Result<()> {
    let repository = PathBuf::from(
        env::var_os("CARGO_MANIFEST_DIR")
            .ok_or_else(|| io::Error::other("CARGO_MANIFEST_DIR is missing"))?,
    )
    .join("../../..");
    let root = system::system(&repository).catalog_root;
    let catalogs = umpire_temporal_annotate_syntax::realization::catalog_inputs(&root)
        .map_err(io::Error::other)?;
    for input in catalogs.watched(&root)? {
        println!("cargo:rerun-if-changed={}", input.display());
    }
    system::link_descriptors(
        &repository,
        &PathBuf::from(
            env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("OUT_DIR is missing"))?,
        )
        .join("realization-api.binpb"),
    )
}
