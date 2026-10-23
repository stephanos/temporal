//! API closure generation with shared protocol imports.
use std::{io, path::Path};
pub(crate) fn generate(
    config: &mut prost_build::Config,
    upstream: &Path,
    schema: &Path,
    include: &Path,
) -> io::Result<()> {
    config
        .extern_path(".umpire.realization.v1", "::umpire_runtime::realization")
        .extern_path(".umpire.annotations.v1", "::umpire_runtime::annotations");
    umpire_catalog_build::codegen::generate(
        config,
        upstream,
        schema,
        &[include, &include.join("../../umpire/proto")],
    )
}
