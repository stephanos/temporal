//! Link the shared upstream descriptors without depending on generated API code.

use std::{env, io, path::PathBuf, process::Command};

fn main() -> io::Result<()> {
    let root = PathBuf::from(
        env::var_os("CARGO_MANIFEST_DIR")
            .ok_or_else(|| io::Error::other("CARGO_MANIFEST_DIR is missing"))?,
    )
    .join("../../../..");
    for input in
        ["Makefile", "go.mod", "go.sum", "cmd/tools/getproto", "proto/internal", "proto/api.binpb"]
    {
        println!("cargo:rerun-if-changed={}", root.join(input).display());
    }
    let catalogs =
        umpire_temporal_annotate_syntax::realization::catalog_inputs(&root.join("model/temporal"))
            .map_err(io::Error::other)?;
    for input in catalogs.directories.iter().chain(&catalogs.files) {
        println!("cargo:rerun-if-changed={}", input.display());
    }
    if !Command::new("make").current_dir(&root).arg("proto/api.binpb").status()?.success() {
        return Err(io::Error::other("make proto/api.binpb failed"));
    }
    let linked = PathBuf::from(
        env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("OUT_DIR is missing"))?,
    )
    .join("realization-api.binpb");
    if !Command::new("sh")
        .current_dir(&root)
        .arg("cmd/tools/getproto/run.sh")
        .arg("--linked-api")
        .arg(root.join("proto/api.binpb"))
        .arg("--out")
        .arg(&linked)
        .status()?
        .success()
    {
        return Err(io::Error::other("linking realization API descriptors failed"));
    }
    Ok(())
}
