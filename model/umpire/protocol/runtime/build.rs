//! Generate shared protocols using prost's native well-known types.

fn main() -> std::io::Result<()> {
    let schema = "../proto/umpire/model/v1/model.proto";
    println!("cargo:rerun-if-changed={schema}");
    println!("cargo:rerun-if-changed=../proto/umpire/realization/v1/realization.proto");
    println!("cargo:rerun-if-changed=../proto/umpire/annotations/v1/annotations.proto");
    let mut config = prost_build::Config::new();
    config
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .include_file("protocol.rs")
        .disable_comments([".google.protobuf"]);
    config.compile_protos(
        &[
            schema,
            "../proto/umpire/realization/v1/realization.proto",
            "../proto/umpire/annotations/v1/annotations.proto",
            "google/protobuf/any.proto",
            "google/protobuf/api.proto",
            "google/protobuf/descriptor.proto",
            "google/protobuf/duration.proto",
            "google/protobuf/empty.proto",
            "google/protobuf/field_mask.proto",
            "google/protobuf/source_context.proto",
            "google/protobuf/struct.proto",
            "google/protobuf/timestamp.proto",
            "google/protobuf/type.proto",
            "google/protobuf/wrappers.proto",
        ],
        &["../proto"],
    )
}
