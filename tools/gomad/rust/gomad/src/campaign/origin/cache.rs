use super::{Invocation, argument, canonical};
use rustix::fs::{Dir, Mode, OFlags, open, openat};
use std::{
    ffi::OsStr,
    fs::{self, File},
    os::unix::fs::{MetadataExt as _, PermissionsExt as _},
    path::{Component, Path},
};

fn error(message: impl std::fmt::Display) -> String {
    format!("invalid registered Go executable cache: {message}")
}

fn registered_run(invocation: &Invocation) -> bool {
    let mut offset = 1;
    if invocation.driver_argv.get(offset) == Some(&argument(OsStr::new("-C"))) {
        offset += 2;
    } else if invocation
        .driver_argv
        .get(offset)
        .is_some_and(|value| value.starts_with(&argument(OsStr::new("-C="))))
    {
        offset += 1;
    }
    invocation.driver_argv.get(offset) == Some(&argument(OsStr::new("run")))
}

fn owned(file: &File, directory: bool) -> Result<(), String> {
    let metadata = file.metadata().map_err(error)?;
    if metadata.uid() != rustix::process::geteuid().as_raw()
        || metadata.permissions().mode() & 0o022 != 0
        || if directory {
            !metadata.is_dir()
        } else {
            !metadata.is_file()
        }
    {
        return Err(error("unsafe ownership, permissions or file type"));
    }
    Ok(())
}

pub(super) fn authenticate(
    root: &Path,
    module: &Path,
    bytes: &[u8],
    invocation: &Invocation,
) -> Result<(), String> {
    if !registered_run(invocation) || !root.is_absolute() || !module.is_absolute() {
        return Err(error("cache admission requires registered go run"));
    }
    let relative = module.strip_prefix(root).map_err(error)?;
    let components = relative.components().collect::<Vec<_>>();
    let [
        Component::Normal(shard),
        Component::Normal(address),
        Component::Normal(name),
    ] = components.as_slice()
    else {
        return Err(error("unexpected cached executable layout"));
    };
    let digest = canonical::digest(bytes);
    if shard.as_encoded_bytes() != &digest.as_bytes()[..2]
        || address.as_encoded_bytes() != format!("{digest}-d").as_bytes()
        || fs::canonicalize(module).map_err(error)?
            != fs::canonicalize(root).map_err(error)?.join(relative)
    {
        return Err(error("captured module does not match its cache address"));
    }
    let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
    let root = File::from(open(root, flags, Mode::empty()).map_err(error)?);
    owned(&root, true)?;
    let shard = File::from(openat(&root, *shard, flags, Mode::empty()).map_err(error)?);
    owned(&shard, true)?;
    let directory = File::from(openat(&shard, *address, flags, Mode::empty()).map_err(error)?);
    owned(&directory, true)?;
    let mut entries = 0;
    for entry in Dir::read_from(&directory).map_err(error)? {
        let entry = entry.map_err(error)?;
        let entry = entry.file_name().to_bytes();
        if matches!(entry, b"." | b"..") {
            continue;
        }
        entries += 1;
        if entries != 1 || entry != name.as_encoded_bytes() {
            return Err(error("cached executable directory must contain one file"));
        }
    }
    if entries != 1 {
        return Err(error("cached executable directory is empty"));
    }
    let file = File::from(
        openat(
            &directory,
            *name,
            OFlags::RDONLY | OFlags::NOFOLLOW | OFlags::NONBLOCK | OFlags::CLOEXEC,
            Mode::empty(),
        )
        .map_err(error)?,
    );
    owned(&file, false)?;
    if file.metadata().map_err(error)?.len() != bytes.len() as u64 {
        return Err(error("cached executable changed after capture"));
    }
    Ok(())
}
