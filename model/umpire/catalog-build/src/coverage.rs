//! Catalog input discovery and descriptor closure admission.
use std::{
    fs, io,
    path::{Path, PathBuf},
};
use umpire_catalog_syntax::{
    catalogs,
    descriptor::{self, Coverage},
    services::{Classification, Declarations, declarations},
};

/// Catalog source and directory inputs in deterministic traversal order.
#[derive(Debug, Default)]
pub struct Inputs {
    /// Admitted declaration source files.
    pub files: Vec<PathBuf>,
    /// Visited directories, including empty ones.
    pub directories: Vec<PathBuf>,
}

/// Discover declarations and directories whose additions invalidate generation.
pub fn inputs(root: &Path) -> io::Result<Inputs> {
    let mut inputs = Inputs::default();
    collect(root, &mut inputs)?;
    Ok(inputs)
}

fn collect(root: &Path, inputs: &mut Inputs) -> io::Result<()> {
    inputs.directories.push(root.to_owned());
    if !root.exists() {
        return Ok(());
    }
    let mut entries = fs::read_dir(root)?.collect::<Result<Vec<_>, _>>()?;
    entries.sort_by_key(fs::DirEntry::path);
    for entry in entries {
        let path = entry.path();
        if entry.file_type()?.is_dir() {
            collect(&path, inputs)?;
        } else if path
            .file_name()
            .and_then(std::ffi::OsStr::to_str)
            .is_some_and(|name| ["api.rs", "enum_inventory.rs", "service.rs"].contains(&name))
        {
            inputs.files.push(path);
        }
    }
    Ok(())
}

/// Check explicitly supplied descriptors against all declarations in a catalog root.
#[allow(clippy::print_stdout, reason = "Cargo consumes build input directives on stdout")]
pub fn check(
    root: &Path,
    descriptor_path: &Path,
    service: &str,
    method_enum: &str,
) -> io::Result<(Coverage, Vec<Classification>)> {
    println!("cargo:rerun-if-changed={}", descriptor_path.display());
    let mut coverage = Coverage::default();
    let mut services = Declarations::default();
    let inputs = inputs(root)?;
    for directory in inputs.directories {
        println!("cargo:rerun-if-changed={}", directory.display());
    }
    for path in inputs.files {
        println!("cargo:rerun-if-changed={}", path.display());
        let source = fs::read_to_string(&path)?;
        let error = |error| io::Error::other(format!("{}: {error}", path.display()));
        for catalog in catalogs(&source).map_err(error)? {
            let projected = catalog.coverage();
            coverage.messages.extend(projected.messages);
            coverage.enums.extend(projected.enums);
            coverage.services.extend(projected.services);
        }
        services.extend(declarations(&source, service, method_enum).map_err(error)?);
    }
    let service_classifications = services.resolve(service).map_err(io::Error::other)?;
    coverage.services.extend(
        service_classifications.iter().map(|classification| classification.coverage.clone()),
    );
    descriptor::check(&fs::read(descriptor_path)?, &coverage).map_err(io::Error::other)?;
    Ok((coverage, service_classifications))
}
