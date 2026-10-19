//! Generate the Model protocol and its shared well-known types.

fn main() -> std::io::Result<()> {
    let schema = "../proto/umpire/model/v1/model.proto";
    println!("cargo:rerun-if-changed={schema}");
    let mut config = prost_build::Config::new();
    config
        .compile_well_known_types()
        .include_file("protocol.rs")
        .disable_comments([".google.protobuf"]);
    config.compile_protos(
        &[
            schema,
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
