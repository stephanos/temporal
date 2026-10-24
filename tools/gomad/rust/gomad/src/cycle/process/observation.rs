use super::{Registration, registry::Registry};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::{
    fs::File,
    os::unix::fs::{FileExt, MetadataExt},
    path::PathBuf,
    time::{SystemTime, UNIX_EPOCH},
};
#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub(crate) struct RawStreams {
    pub stdout: PathBuf,
    pub stderr: PathBuf,
    pub stdout_sha256: String,
    pub stderr_sha256: String,
    pub argv_sha256: String,
}
pub(super) fn wall_nanos() -> Result<u64, String> {
    u64::try_from(
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|error| error.to_string())?
            .as_nanos(),
    )
    .map_err(|error| error.to_string())
}
#[expect(
    clippy::cast_precision_loss,
    reason = "Compatibility seconds are derived from validated integer wait4 nanoseconds; integer nanoseconds remain authoritative"
)]
pub(super) fn seconds(seconds: i64, micros: i64) -> Result<f64, String> {
    Ok(super::nanos(seconds, micros)? as f64 / 1_000_000_000.0)
}
fn stream(registry: &Registry, file: &File, name: &str) -> Result<(PathBuf, String), String> {
    let named = registry.open_read(name)?;
    let metadata = file.metadata().map_err(|error| error.to_string())?;
    let observed = named.metadata().map_err(|error| error.to_string())?;
    if metadata.dev() != observed.dev()
        || metadata.ino() != observed.ino()
        || !metadata.is_file()
        || metadata.nlink() != 1
        || metadata.mode() & 0o777 != 0o600
    {
        return Err("command raw stream identity changed".into());
    }
    let mut hash = Sha256::new();
    let mut offset = 0u64;
    let mut buffer = [0u8; 16384];
    loop {
        let count = match file.read_at(&mut buffer, offset) {
            Ok(count) => count,
            Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
            Err(error) => return Err(error.to_string()),
        };
        if count == 0 {
            break;
        }
        hash.update(&buffer[..count]);
        offset = offset
            .checked_add(u64::try_from(count).map_err(|error| error.to_string())?)
            .ok_or("raw stream offset overflow")?;
    }
    let after = file.metadata().map_err(|error| error.to_string())?;
    if after.len() != offset
        || after.len() != metadata.len()
        || after.mtime() != metadata.mtime()
        || after.mtime_nsec() != metadata.mtime_nsec()
        || after.ctime() != metadata.ctime()
        || after.ctime_nsec() != metadata.ctime_nsec()
    {
        return Err("command raw stream changed during observation".into());
    }
    Ok((registry.path.join(name), format!("{:x}", hash.finalize())))
}
pub(super) fn bind(
    registry: &Registry,
    registration: &Registration,
    stdout: &File,
    stderr: &File,
) -> Result<RawStreams, String> {
    let (stdout, stdout_sha256) =
        stream(registry, stdout, &format!("{}.stdout", registration.name))?;
    let (stderr, stderr_sha256) =
        stream(registry, stderr, &format!("{}.stderr", registration.name))?;
    Ok(RawStreams {
        stdout,
        stderr,
        stdout_sha256,
        stderr_sha256,
        argv_sha256: crate::evidence::canonical::digest(&crate::evidence::canonical::encode(
            &registration.argv,
        )?),
    })
}
