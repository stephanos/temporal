//! Generate the reference service and enforce the shared catalog descriptor closure.
#![forbid(unsafe_code)]

use std::{env, io, path::PathBuf};

fn main() -> io::Result<()> {
    let out =
        PathBuf::from(env::var_os("OUT_DIR").ok_or_else(|| io::Error::other("OUT_DIR missing"))?);
    let descriptor = out.join("kettle.binpb");
    for schema in ["proto/kettle/v1/kettle.proto", "proto/kettle/v1/probe.proto"] {
        println!("cargo:rerun-if-changed={schema}");
    }
    prost_build::Config::new()
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .btree_map(["."])
        .extern_path(".umpire.annotations.v1", "::umpire_runtime::annotations")
        .file_descriptor_set_path(&descriptor)
        .compile_protos(
            &["proto/kettle/v1/kettle.proto", "proto/kettle/v1/probe.proto"],
            &["proto", "../../protocol/proto"],
        )?;
    umpire_catalog_build::System {
        service: "umpire.reference.kettle.v1.KettleService",
        prefix: "kettle",
        rust_root: "crate::wire",
        request_roots: &["Fill", "Heat", "Pour", "Describe", "PollHeating", "RespondHeating"],
        catalog_root: PathBuf::from(
            env::var_os("CARGO_MANIFEST_DIR")
                .ok_or_else(|| io::Error::other("manifest missing"))?,
        ),
    }
    .generate(&descriptor, &out)
}
