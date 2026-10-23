use super::{Candidate, admit, manifest, same_directory};
use rustix::fs::{self, AtFlags};
use std::{fs::File, path::Path};

pub(super) fn discard_stage(directory: &File, name: &str) -> Result<(), String> {
    let metadata = match fs::statat(directory, name, AtFlags::SYMLINK_NOFOLLOW) {
        Ok(metadata) => metadata,
        Err(rustix::io::Errno::NOENT) => return Ok(()),
        Err(error) => return Err(error.to_string()),
    };
    if fs::FileType::from_raw_mode(metadata.st_mode) != fs::FileType::RegularFile
        || metadata.st_uid != rustix::process::geteuid().as_raw()
        || metadata.st_mode & 0o777 != 0o600
        || !(1..=2).contains(&metadata.st_nlink)
    {
        return Err("unsafe interrupted derived installation stage".into());
    }
    fs::unlinkat(directory, name, AtFlags::empty()).map_err(|e| e.to_string())?;
    directory.sync_all().map_err(|e| e.to_string())
}

struct Stage<'a> {
    directory: &'a File,
    name: String,
}

impl Drop for Stage<'_> {
    fn drop(&mut self) {
        let _ = discard_stage(self.directory, &self.name);
    }
}

pub(super) fn publish(
    root: &Path,
    directory: &File,
    path: &Path,
    candidate: &Candidate,
) -> Result<(), String> {
    publish_with(
        root,
        directory,
        path,
        candidate,
        |directory, stage, name| fs::linkat(directory, stage, directory, name, AtFlags::empty()),
    )
}

pub(super) fn publish_with(
    root: &Path,
    directory: &File,
    path: &Path,
    candidate: &Candidate,
    link: impl FnOnce(&File, &str, &str) -> Result<(), rustix::io::Errno>,
) -> Result<(), String> {
    let stage = Stage {
        directory,
        name: format!(".{}.staged", candidate.key),
    };
    let result = (|| {
        manifest::write(&root.join(&stage.name), &candidate.installed)?;
        admit(&root.join(&stage.name), candidate)?;
        same_directory(root, directory)?;
        let name = path
            .file_name()
            .and_then(|name| name.to_str())
            .ok_or("invalid derived installation name")?;
        match link(directory, &stage.name, name) {
            Ok(()) => directory.sync_all().map_err(|e| e.to_string())?,
            Err(rustix::io::Errno::EXIST) => admit(path, candidate)?,
            Err(error) => return Err(error.to_string()),
        }
        admit(path, candidate)
    })();
    let cleanup = discard_stage(directory, &stage.name);
    result.and(cleanup)
}
