use crate::{Error, Result, support::hash};
use std::{
    fs,
    path::{Path, PathBuf},
};
use umpire_backend_process::tool_provider as provider;

/// Provision the optional Apalache release through its named mise environment.
pub fn install(root: &Path) -> Result<PathBuf> {
    install_tool(root, provider::JarTool::Apalache, crate::apalache_pins::JAR_SHA256)
}

/// Provision the required TLC release through the default jar environment.
pub fn install_tlc(root: &Path) -> Result<PathBuf> {
    install_tool(
        root,
        provider::JarTool::Tlc,
        "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88",
    )
}

fn install_tool(root: &Path, tool: provider::JarTool, expected: &str) -> Result<PathBuf> {
    let jar = provider::install(root, tool)?;
    if hash(&fs::read(&jar)?) != expected {
        return Err(Error::Invalid(format!("pinned jar integrity mismatch: {}", jar.display())));
    }
    Ok(jar)
}
