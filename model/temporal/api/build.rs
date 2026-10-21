//! Generate the upstream descriptor closure with shared well-known types.

use std::{
    env, fs, io,
    path::{Path, PathBuf},
    process::Command,
};

use umpire_temporal_annotate_syntax::{
    catalogs,
    descriptor::{self, Coverage},
    realization::catalog_inputs as declaration_inputs,
    services::{Classification, classifications},
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
    let mut coverage = Coverage::default();
    let mut classifications = Vec::new();
    catalog_inputs(&root.join("model/temporal"), &mut coverage, &mut classifications)?;
    let descriptors = fs::read(&linked)?;
    descriptor::check(&descriptors, &coverage).map_err(io::Error::other)?;
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

fn catalog_inputs(
    path: &Path,
    coverage: &mut Coverage,
    service_classifications: &mut Vec<Classification>,
) -> io::Result<()> {
    if !path.exists() {
        println!("cargo:rerun-if-changed={}", path.display());
        return Ok(());
    }
    let inputs = declaration_inputs(path).map_err(io::Error::other)?;
    for directory in inputs.directories {
        println!("cargo:rerun-if-changed={}", directory.display());
    }
    for path in inputs.files {
        println!("cargo:rerun-if-changed={}", path.display());
        let source = fs::read_to_string(&path)?;
        for catalog in catalogs(&source)
            .map_err(|error| io::Error::other(format!("{}: {error}", path.display())))?
        {
            let projected = catalog.coverage();
            coverage.messages.extend(projected.messages);
            coverage.enums.extend(projected.enums);
            coverage.services.extend(projected.services);
        }
        for classification in classifications(&source)
            .map_err(|error| io::Error::other(format!("{}: {error}", path.display())))?
        {
            coverage.services.push(classification.coverage.clone());
            service_classifications.push(classification);
        }
    }
    Ok(())
}
