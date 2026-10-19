//! Generate the upstream descriptor closure with shared well-known types.

use std::{env, io, path::PathBuf, process::Command};

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
