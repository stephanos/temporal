//! Generate the upstream API closure and local envelopes.

use std::{io, path::Path};

/// Generate a deterministic descriptor closure extended by a local schema.
pub fn generate(
    config: &mut prost_build::Config,
    upstream: &Path,
    schema: &Path,
    includes: &[&Path],
) -> io::Result<()> {
    let mut files = prost_build::Config::new()
        .file_descriptor_set_path(upstream)
        .skip_protoc_run()
        .load_fds(&[] as &[&str], &[] as &[&str])?;
    config
        .enable_type_names()
        .type_name_domain(["."], "type.googleapis.com")
        .btree_map(["."])
        .include_file("protocol.rs")
        .disable_comments(["."])
        .protoc_arg(format!("--descriptor_set_in={}", upstream.display()));
    let extension = config.load_fds(&[schema], includes)?;
    for file in extension.file {
        if !files.file.iter().any(|existing| existing.name == file.name) {
            files.file.push(file);
        }
    }
    // Same-package definitions are emitted in descriptor file order.
    files.file.sort_by(|a, b| a.name.cmp(&b.name));
    config.compile_fds(files)
}
