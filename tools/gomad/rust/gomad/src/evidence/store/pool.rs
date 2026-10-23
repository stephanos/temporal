use super::{canonical, open, publish};
use rustix::fs::{self, AtFlags};
use std::fs::File;
#[cfg(test)]
use std::io::Write;

#[cfg(test)]
pub(super) fn place(
    store: &publish::Store,
    pool: &File,
    destination: &File,
    name: &str,
    bytes: &[u8],
) -> Result<(), String> {
    place_with(store, pool, destination, name, bytes, &mut |file, bytes| {
        file.write_all(bytes)
    })
}

pub(super) fn place_with(
    store: &publish::Store,
    pool: &File,
    destination: &File,
    name: &str,
    bytes: &[u8],
    writer: &mut publish::ByteWriter<'_>,
) -> Result<(), String> {
    let digest = canonical::digest(bytes);
    store.immutable_file_with(pool, &digest, bytes, true, writer)?;
    let source = open::exclusive(pool, &digest, true)?;
    let linked = fs::linkat(pool, digest.as_str(), destination, name, AtFlags::empty());
    drop(source);
    match linked {
        Ok(()) => {}
        Err(error) if error == rustix::io::Errno::EXIST => {
            if open::read(destination, name, bytes.len() as u64, true)? != bytes {
                return Err("target pool collision identity mismatch".into());
            }
        }
        Err(error) if error == rustix::io::Errno::XDEV => {
            store.immutable_file_with(destination, name, bytes, true, writer)?;
        }
        Err(error) => return Err(error.to_string()),
    }
    destination.sync_all().map_err(|error| error.to_string())
}
