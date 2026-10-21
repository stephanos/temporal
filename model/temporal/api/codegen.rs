//! Generate the upstream API closure and local envelopes.

use std::{io, path::Path};

pub(crate) fn generate(
    config: &mut prost_build::Config,
    upstream: &Path,
    schema: &Path,
    include: &Path,
) -> io::Result<()> {
    let mut files = prost_build::Config::new()
        .file_descriptor_set_path(upstream)
        .skip_protoc_run()
        .load_fds(&[] as &[&str], &[] as &[&str])?;
    config
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .compile_well_known_types()
        .btree_map(["."])
        .extern_path(".google.protobuf", "::umpire_wasm::google::protobuf")
        .include_file("protocol.rs")
        .disable_comments(["."])
        .protoc_arg(format!("--descriptor_set_in={}", upstream.display()));
    let extension = config.load_fds(&[schema], &[include])?;
    for file in extension.file {
        if !files.file.iter().any(|existing| existing.name == file.name) {
            files.file.push(file);
        }
    }
    config.compile_fds(files)
}
