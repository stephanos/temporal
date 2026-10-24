//! Generate the upstream descriptor closure with shared well-known types.

use std::{
    env, fs, io,
    path::{Path, PathBuf},
    process::Command,
};

mod codegen;
mod services;
mod wire_schema;

fn main() -> io::Result<()> {
    let root = PathBuf::from(
        env::var_os("CARGO_MANIFEST_DIR")
            .ok_or_else(|| io::Error::other("CARGO_MANIFEST_DIR is missing"))?,
    )
    .join("../../..");
    let upstream = root.join("proto/api.binpb");
    for input in ["Makefile", "go.mod", "go.sum", "cmd/tools/getproto", "proto/internal"] {
        println!("cargo:rerun-if-changed={}", root.join(input).display());
    }
    if !Command::new("make").current_dir(&root).arg("proto/api.binpb").status()?.success() {
        return Err(io::Error::other("make proto/api.binpb failed"));
    }
    let schema = "../proto/umpire/temporal/v1/temporal.proto";
    println!("cargo:rerun-if-changed={}", upstream.display());
    println!("cargo:rerun-if-changed={schema}");
    println!("cargo:rerun-if-changed=../proto/umpire/temporal/v1/probe.proto");
    println!("cargo:rerun-if-changed=../proto/umpire/temporal/v1/activity.proto");
    let linked = PathBuf::from(
        env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("OUT_DIR is missing"))?,
    )
    .join("annotation-api.binpb");
    if !Command::new("sh")
        .current_dir(&root)
        .arg("cmd/tools/getproto/run.sh")
        .arg("--linked-api")
        .arg(&upstream)
        .arg("--out")
        .arg(&linked)
        .status()?
        .success()
    {
        return Err(io::Error::other("linking annotation API descriptors failed"));
    }
    let (coverage, classifications) = umpire_catalog_build::coverage::check(
        &root.join("model/temporal"),
        &linked,
        "temporal.api.workflowservice.v1.WorkflowService",
        "WorkflowServiceMethod",
    )?;
    let descriptors = fs::read(&linked)?;
    fs::write(
        linked.with_file_name("services.rs"),
        services::generate(&descriptors, &coverage, &classifications).map_err(io::Error::other)?,
    )?;
    fs::write(
        linked.with_file_name("wire_schema.rs"),
        wire_schema::generate(&descriptors).map_err(io::Error::other)?,
    )?;

    codegen::generate(
        &mut prost_build::Config::new(),
        &upstream,
        Path::new(schema),
        Path::new("../proto"),
    )
}
