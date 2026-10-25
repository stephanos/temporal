use crate::cycle::custody;
use rustix::fs::{Mode, OFlags, open, openat};
use std::{
    fs::{self, File},
    io::Write,
    path::{Component, Path, PathBuf},
};
fn parent(path: &Path) -> Result<(File, Vec<(PathBuf, File)>), String> {
    custody::clean_absolute(path)?;
    let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
    let mut dirs = vec![(
        PathBuf::from("/"),
        File::from(open("/", flags, Mode::empty()).map_err(|e| e.to_string())?),
    )];
    for part in path
        .parent()
        .ok_or("publication parent missing")?
        .components()
    {
        if let Component::Normal(name) = part {
            let (parent, fd) = dirs.last().ok_or("publication parent absent")?;
            let next =
                File::from(openat(fd, name, flags, Mode::empty()).map_err(|e| e.to_string())?);
            dirs.push((parent.join(name), next));
        }
    }
    let fd = dirs
        .last()
        .ok_or("publication parent absent")?
        .1
        .try_clone()
        .map_err(|e| e.to_string())?;
    Ok((fd, dirs))
}
pub(super) fn fresh(path: &Path) -> Result<(), String> {
    let _ = parent(path)?;
    match fs::symlink_metadata(path) {
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
        Ok(_) => Err("freeze output must be fresh".into()),
        Err(e) => Err(e.to_string()),
    }
}
pub(super) fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let (fd, dirs) = parent(path)?;
    let name = path.file_name().ok_or("publication filename missing")?;
    let mut file = File::from(
        openat(
            &fd,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::RUSR | Mode::WUSR,
        )
        .map_err(|e| format!("fresh freeze publication: {e}"))?,
    );
    file.write_all(bytes).map_err(|e| e.to_string())?;
    file.sync_all().map_err(|e| e.to_string())?;
    for (name, dir) in dirs {
        use std::os::unix::fs::MetadataExt;
        let actual = dir.metadata().map_err(|e| e.to_string())?;
        let named = fs::symlink_metadata(name).map_err(|e| e.to_string())?;
        if !named.is_dir() || actual.dev() != named.dev() || actual.ino() != named.ino() {
            return Err("freeze publication ancestor changed".into());
        }
    }
    fd.sync_all().map_err(|e| e.to_string())?;
    if custody::read_file(path, 64 << 20)? != bytes {
        return Err("freeze publication changed".into());
    }
    Ok(())
}
