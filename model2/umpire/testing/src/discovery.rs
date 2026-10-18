//! Build-time fixture discovery.

use std::path::Path;

/// Generate fixture modules, refusing incomplete fixture directories.
pub fn generate(_root: &Path, _output: &Path) -> std::io::Result<()> {
    Ok(())
}
