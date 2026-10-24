#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle endpoint integration belongs to successor batches"
    )
)]
use serde_json::Value;
use std::{
    fs,
    os::unix::fs::MetadataExt as _,
    path::{Component, Path},
};
pub(crate) mod gnu;
mod inventory;
pub(crate) mod source;
pub(crate) mod workloads;
pub(crate) use crate::cycle::custody::{digest, read_file as read, strict_json};
pub(crate) use inventory::{Inventory, PreparedInputs};

fn require(condition: bool, message: &str) -> Result<(), String> {
    if condition {
        Ok(())
    } else {
        Err(message.into())
    }
}
pub(crate) fn canonical(value: &Value) -> Result<Vec<u8>, String> {
    crate::evidence::canonical::encode(value)
}
fn clean(path: &Path) -> bool {
    path.is_absolute()
        && !path.as_os_str().as_encoded_bytes().contains(&0)
        && path
            .components()
            .all(|part| matches!(part, Component::RootDir | Component::Normal(_)))
        && path
            .components()
            .collect::<std::path::PathBuf>()
            .as_os_str()
            == path.as_os_str()
}
fn canonical_path(path: &Path) -> Result<(), String> {
    require(
        clean(path)
            && fs::canonicalize(path)
                .map_err(|e| e.to_string())?
                .as_os_str()
                == path.as_os_str(),
        "noncanonical or linked path",
    )
}
pub(crate) fn byte_bound(length: u64, maximum: u64) -> Result<(), String> {
    require(length <= maximum, "input byte bound exceeded")
}
pub(crate) fn regular_metadata(
    info: &fs::Metadata,
    maximum: u64,
    mode: Option<u32>,
) -> Result<(), String> {
    require(
        info.is_file() && info.nlink() == 1,
        "nonregular or linked source",
    )?;
    byte_bound(info.len(), maximum)?;
    if let Some(mode) = mode {
        require(
            info.uid() == rustix::process::geteuid().as_raw(),
            "derived file owner changed",
        )?;
        require(info.mode() & 0o7777 == mode, "derived file mode changed")?;
    }
    Ok(())
}
pub(crate) fn same_file(before: &fs::Metadata, after: &fs::Metadata) -> bool {
    (
        before.dev(),
        before.ino(),
        before.mode(),
        before.nlink(),
        before.uid(),
        before.gid(),
        before.size(),
        before.mtime(),
        before.mtime_nsec(),
        before.ctime(),
        before.ctime_nsec(),
    ) == (
        after.dev(),
        after.ino(),
        after.mode(),
        after.nlink(),
        after.uid(),
        after.gid(),
        after.size(),
        after.mtime(),
        after.mtime_nsec(),
        after.ctime(),
        after.ctime_nsec(),
    )
}
fn regular(path: &Path, maximum: u64, mode: Option<u32>) -> Result<Vec<u8>, String> {
    let before = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    regular_metadata(&before, maximum, mode)?;
    let (raw, observed) = crate::cycle::custody::read_file_observed(path, maximum)?;
    byte_bound(raw.len() as u64, maximum)?;
    require(
        same_file(&before, &observed),
        "source changed before descriptor read",
    )?;
    let after = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    require(
        same_file(&observed, &after) && raw.len() as u64 == observed.len(),
        "source changed during descriptor read",
    )?;
    canonical_path(path)?;
    Ok(raw)
}

pub(crate) fn verify_preparation(
    config: &Value,
    artifact: &Path,
    installation: &Value,
    inputs: &Value,
    owner_pid: Option<u64>,
) -> Result<PreparedInputs, String> {
    Inventory::new(config)?.project(artifact, installation, inputs, owner_pid)
}
#[cfg(test)]
mod tests;
#[cfg(test)]
pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    tests::run_cases(owner)
}
