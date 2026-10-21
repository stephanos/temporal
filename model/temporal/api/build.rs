//! Generate the upstream descriptor closure with shared well-known types.

use std::{
    env, fs, io,
    path::{Path, PathBuf},
    process::Command,
};

use umpire_temporal_annotate_syntax::{
    catalogs,
    descriptor::{self, Coverage},
};

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
    catalog_inputs(&root.join("model/temporal/features"), &mut coverage)?;
    descriptor::check(&fs::read(&linked)?, &coverage).map_err(io::Error::other)?;

    let mut files = prost_build::Config::new()
        .file_descriptor_set_path(&upstream)
        .skip_protoc_run()
        .load_fds(&[] as &[&str], &[] as &[&str])?;
    let mut config = prost_build::Config::new();
    config
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .compile_well_known_types()
        .btree_map(["."])
        .extern_path(".google.protobuf", "::umpire_wasm::google::protobuf")
        .include_file("protocol.rs")
        .disable_comments(["."])
        .protoc_arg(format!("--descriptor_set_in={}", upstream.display()));
    let extension = config.load_fds(&[schema], &["../proto"])?;
    for file in extension.file {
        if !files.file.iter().any(|existing| existing.name == file.name) {
            files.file.push(file);
        }
    }
    config.compile_fds(files)
}

fn catalog_inputs(path: &Path, coverage: &mut Coverage) -> io::Result<()> {
    println!("cargo:rerun-if-changed={}", path.display());
    if !path.exists() {
        return Ok(());
    }
    let mut entries: Vec<_> = fs::read_dir(path)?.collect::<Result<_, _>>()?;
    entries.sort_by_key(fs::DirEntry::path);
    for entry in entries {
        let path = entry.path();
        if entry.file_type()?.is_dir() {
            catalog_inputs(&path, coverage)?;
        } else if path.file_name().is_some_and(|name| name == "api.rs") {
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
        }
    }
    Ok(())
}
