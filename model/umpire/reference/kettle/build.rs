//! Generate the reference service with the shared protobuf well-known types.
fn main() -> std::io::Result<()> {
    println!("cargo:rerun-if-changed=proto/kettle/v1/kettle.proto");
    prost_build::Config::new()
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .compile_well_known_types()
        .extern_path(".google.protobuf", "::umpire_wasm::google::protobuf")
        .compile_protos(&["proto/kettle/v1/kettle.proto"], &["proto"])
}
