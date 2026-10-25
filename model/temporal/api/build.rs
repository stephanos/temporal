//! Generate the upstream descriptor closure with shared well-known types.

use std::{
    env, io,
    path::{Path, PathBuf},
};

mod system;

fn main() -> io::Result<()> {
    let repository = PathBuf::from(
        env::var_os("CARGO_MANIFEST_DIR")
            .ok_or_else(|| io::Error::other("CARGO_MANIFEST_DIR is missing"))?,
    )
    .join("../../..");
    let out = PathBuf::from(
        env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("OUT_DIR is missing"))?,
    );
    let schema = "../proto/umpire/temporal/v1/temporal.proto";
    println!("cargo:rerun-if-changed={schema}");
    println!("cargo:rerun-if-changed=../proto/umpire/temporal/v1/probe.proto");
    println!("cargo:rerun-if-changed=../proto/umpire/temporal/v1/activity.proto");
    let linked = out.join("annotation-api.binpb");
    system::link_descriptors(&repository, &linked)?;
    system::system(&repository).generate(&linked, &out)?;
    let mut config = prost_build::Config::new();
    config
        .extern_path(".umpire.realization.v1", "::umpire_runtime::realization")
        .extern_path(".umpire.annotations.v1", "::umpire_runtime::annotations");
    umpire_catalog_build::codegen::generate(
        &mut config,
        &repository.join("proto/api.binpb"),
        Path::new(schema),
        &[Path::new("../proto"), Path::new("../proto/../../umpire/protocol/proto")],
    )
}
