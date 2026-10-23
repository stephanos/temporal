//! Generate the reference service and enforce the shared catalog descriptor closure.
use std::{env, fs, io, path::PathBuf};

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
        .compile_well_known_types()
        .btree_map(["."])
        .extern_path(".google.protobuf", "::umpire_runtime::google::protobuf")
        .extern_path(".umpire.annotations.v1", "::umpire_runtime::annotations")
        .file_descriptor_set_path(&descriptor)
        .compile_protos(
            &["proto/kettle/v1/kettle.proto", "proto/kettle/v1/probe.proto"],
            &["proto", "../../proto"],
        )?;
    let root = PathBuf::from(
        env::var_os("CARGO_MANIFEST_DIR").ok_or_else(|| io::Error::other("manifest missing"))?,
    );
    let (coverage, classifications) = umpire_catalog_build::coverage::check(
        &root,
        &descriptor,
        "umpire.reference.kettle.v1.KettleService",
        "KettleServiceMethod",
    )?;
    let bytes = fs::read(&descriptor)?;
    fs::write(
        out.join("services.rs"),
        umpire_catalog_build::services::generate(
            &bytes,
            &coverage,
            &classifications,
            &umpire_catalog_build::services::Config {
                service: "umpire.reference.kettle.v1.KettleService",
                rust_type_root: "crate::wire",
                macro_type_root: "$crate::wire",
                method_enum: "KettleServiceMethod",
                method_info: "KettleServiceMethodInfo",
                methods_const: "KETTLE_SERVICE_METHODS",
                dispatch_macro: "dispatch_kettle_annotations",
                bind_macro: "bind_kettle_action",
            },
        )
        .map_err(io::Error::other)?,
    )?;
    fs::write(
        out.join("wire_schema.rs"),
        umpire_catalog_build::wire_schema::generate(
            &bytes,
            &umpire_catalog_build::wire_schema::Config {
                service: "umpire.reference.kettle.v1.KettleService",
                rust_type_root: "crate::wire",
                method_enum: "KettleServiceMethod",
                fields_function: "kettle_request_fields",
                mutate_function: "mutate_kettle_request",
                request_roots: &[
                    "Fill",
                    "Heat",
                    "Pour",
                    "Describe",
                    "PollHeating",
                    "RespondHeating",
                ],
            },
        )
        .map_err(io::Error::other)?,
    )?;
    Ok(())
}
