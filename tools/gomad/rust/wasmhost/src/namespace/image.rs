use crate::namespace::{Boundary, Image, ImageEntry, Limits};
use std::collections::{BTreeMap, BTreeSet};
use std::io::{ErrorKind, Read};

#[expect(
    clippy::too_many_lines,
    reason = "The image decoder validates wire lengths, topology, and descriptor limits before returning an image."
)]
pub fn decode(
    reader: &mut impl Read,
    expected_files: u64,
    expected_output: u64,
    expected_epoch: u64,
) -> Result<Image, Boundary> {
    let mut magic = [0; 8];
    reader.read_exact(&mut magic).map_err(|_| invalid())?;
    if &magic != b"GMWN\x01\0\0\0" {
        return Err(invalid());
    }
    let limits = Limits {
        files: read_u64(reader)?,
        descriptors: read_u64(reader)?,
        filesystem_bytes: read_u64(reader)?,
        output_bytes: read_u64(reader)?,
    };
    let epoch_nanos = read_u64(reader)?;
    let entries_count = read_u64(reader)?;
    let writable_count = read_u64(reader)?;
    let captured_count = read_u64(reader)?;
    let preopen_count = read_u64(reader)?;
    let files = entries_count.checked_add(1).ok_or_else(invalid)?;
    if limits.files == 0
        || limits.files > 1 << 20
        || limits.descriptors < 5
        || limits.descriptors > 1 << 20
        || limits.filesystem_bytes == 0
        || limits.filesystem_bytes > 1 << 30
        || limits.output_bytes == 0
        || limits.output_bytes > 1 << 30
        || epoch_nanos > i64::MAX as u64
        || files != expected_files
        || files > limits.files
        || limits.output_bytes != expected_output
        || epoch_nanos != expected_epoch
        || writable_count > files
        || captured_count > files
        || preopen_count > files
        || preopen_count
            .checked_add(3)
            .is_none_or(|count| count > limits.descriptors)
    {
        return Err(invalid());
    }
    let captured_roots = read_paths(reader, captured_count, true)?;
    if captured_roots
        .iter()
        .enumerate()
        .any(|(index, root)| root == b"/" || beneath_captured_root(root, &captured_roots[..index]))
    {
        return Err(invalid());
    }
    let writable_directories = read_paths(reader, writable_count, false)?;
    let preopens = read_paths(reader, preopen_count, true)?;
    let working_directory = read_path(reader)?;
    let stdin = read_blob(reader, limits.filesystem_bytes)?;
    let mut remaining_bytes = limits.filesystem_bytes - stdin.len() as u64;
    let mut entries = Vec::new();
    for _ in 0..entries_count {
        let kind = read_byte(reader)?;
        let readonly = read_byte(reader)?;
        if !matches!(kind, 1 | 2) || readonly > 1 {
            return Err(invalid());
        }
        let path = read_path(reader)?;
        let data = read_blob(reader, if kind == 1 { 0 } else { remaining_bytes })?;
        remaining_bytes = remaining_bytes
            .checked_sub(data.len() as u64)
            .ok_or_else(invalid)?;
        entries.try_reserve(1).map_err(|_| invalid())?;
        entries.push(ImageEntry {
            path,
            directory: kind == 1,
            readonly: readonly == 1,
            data,
        });
    }
    if read_byte(reader)? != 0 {
        return Err(invalid());
    }
    let mut trailing = [0];
    loop {
        match reader.read(&mut trailing) {
            Ok(0) => break,
            Err(error) if error.kind() == ErrorKind::Interrupted => (),
            _ => return Err(invalid()),
        }
    }
    let writable: BTreeSet<_> = writable_directories.iter().map(Vec::as_slice).collect();
    if writable.len() != writable_directories.len() {
        return Err(invalid());
    }
    let mut nodes = BTreeMap::new();
    nodes.insert(b"/".as_slice(), (true, true));
    for entry in &entries {
        let slash = entry.path.iter().rposition(|byte| *byte == b'/').unwrap();
        let parent = if slash == 0 {
            b"/"
        } else {
            &entry.path[..slash]
        };
        let captured = beneath_captured_root(&entry.path, &captured_roots);
        if entry.path == b"/"
            || nodes.get(parent).is_none_or(|(directory, _)| !directory)
            || nodes.contains_key(entry.path.as_slice())
            || captured && !entry.readonly
            || !entry.directory && entry.readonly && !captured
        {
            return Err(invalid());
        }
        nodes.insert(entry.path.as_slice(), (entry.directory, entry.readonly));
    }
    if nodes
        .get(working_directory.as_slice())
        .is_none_or(|(directory, _)| !directory)
    {
        return Err(invalid());
    }
    for root in &captured_roots {
        if nodes.get(root.as_slice()) != Some(&(true, true)) {
            return Err(invalid());
        }
    }
    for path in &writable_directories {
        if nodes.get(path.as_slice()) != Some(&(true, false))
            || beneath_captured_root(path, &captured_roots)
        {
            return Err(invalid());
        }
    }
    let mounts: BTreeSet<_> = captured_roots
        .iter()
        .chain(&writable_directories)
        .map(Vec::as_slice)
        .collect();
    if !mounts
        .iter()
        .copied()
        .eq(preopens.iter().map(Vec::as_slice))
    {
        return Err(invalid());
    }
    Ok(Image {
        entries,
        writable_directories,
        preopens,
        working_directory,
        stdin,
        limits,
        epoch_nanos,
    })
}

fn read_byte(reader: &mut impl Read) -> Result<u8, Boundary> {
    let mut value = [0];
    reader.read_exact(&mut value).map_err(|_| invalid())?;
    Ok(value[0])
}

fn read_u64(reader: &mut impl Read) -> Result<u64, Boundary> {
    let mut value = [0; 8];
    reader.read_exact(&mut value).map_err(|_| invalid())?;
    Ok(u64::from_le_bytes(value))
}

fn read_blob(reader: &mut impl Read, limit: u64) -> Result<Vec<u8>, Boundary> {
    let mut remaining = read_u64(reader)?;
    if remaining > limit {
        return Err(invalid());
    }
    let mut bytes = Vec::new();
    let mut chunk = [0; 4096];
    while remaining > 0 {
        let length = chunk
            .len()
            .min(usize::try_from(remaining).unwrap_or(usize::MAX));
        reader
            .read_exact(&mut chunk[..length])
            .map_err(|_| invalid())?;
        bytes.try_reserve(length).map_err(|_| invalid())?;
        bytes.extend_from_slice(&chunk[..length]);
        remaining -= length as u64;
    }
    Ok(bytes)
}

fn read_path(reader: &mut impl Read) -> Result<Vec<u8>, Boundary> {
    let path = read_blob(reader, u64::MAX)?;
    if std::str::from_utf8(&path).is_err()
        || path.contains(&0)
        || !(path == b"/"
            || path.first() == Some(&b'/')
                && path[1..]
                    .split(|byte| *byte == b'/')
                    .all(|name| !name.is_empty() && name != b"." && name != b".."))
    {
        return Err(invalid());
    }
    Ok(path)
}

fn read_paths(reader: &mut impl Read, count: u64, sorted: bool) -> Result<Vec<Vec<u8>>, Boundary> {
    let mut paths = Vec::new();
    for _ in 0..count {
        let path = read_path(reader)?;
        if sorted && paths.last().is_some_and(|previous| previous >= &path) {
            return Err(invalid());
        }
        paths.try_reserve(1).map_err(|_| invalid())?;
        paths.push(path);
    }
    Ok(paths)
}

fn beneath_captured_root(path: &[u8], roots: &[Vec<u8>]) -> bool {
    if roots
        .binary_search_by(|root| root.as_slice().cmp(path))
        .is_ok()
        || roots.first().is_some_and(|root| root == b"/")
    {
        return true;
    }
    path.iter().enumerate().skip(1).any(|(index, byte)| {
        *byte == b'/'
            && roots
                .binary_search_by(|root| root.as_slice().cmp(&path[..index]))
                .is_ok()
    })
}

fn invalid() -> Boundary {
    Boundary {
        kind: "invalid",
        dimension: "namespace image",
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Fixture {
        limits: [u64; 5],
        captured: Vec<&'static [u8]>,
        writable: Vec<&'static [u8]>,
        preopens: Vec<&'static [u8]>,
        cwd: Vec<u8>,
        stdin: Vec<u8>,
        entries: Vec<(u8, u8, Vec<u8>, Vec<u8>)>,
    }

    impl Fixture {
        fn new() -> Self {
            Self {
                limits: [10, 10, 20, 3, 4],
                captured: vec![b"/workspace/input"],
                writable: vec![b"/workspace"],
                preopens: vec![b"/workspace", b"/workspace/input"],
                cwd: b"/workspace".to_vec(),
                stdin: b"in".to_vec(),
                entries: vec![
                    (1, 0, b"/workspace".to_vec(), vec![]),
                    (1, 1, b"/workspace/input".to_vec(), vec![]),
                    (2, 1, b"/workspace/input/a".to_vec(), b"abc".to_vec()),
                ],
            }
        }

        fn bytes(&self) -> Vec<u8> {
            let mut bytes = b"GMWN\x01\0\0\0".to_vec();
            for value in self.limits.into_iter().chain([
                self.entries.len() as u64,
                self.writable.len() as u64,
                self.captured.len() as u64,
                self.preopens.len() as u64,
            ]) {
                bytes.extend_from_slice(&value.to_le_bytes());
            }
            for path in self
                .captured
                .iter()
                .chain(&self.writable)
                .chain(&self.preopens)
            {
                blob(&mut bytes, path);
            }
            blob(&mut bytes, &self.cwd);
            blob(&mut bytes, &self.stdin);
            for (kind, readonly, path, data) in &self.entries {
                bytes.extend_from_slice(&[*kind, *readonly]);
                blob(&mut bytes, path);
                blob(&mut bytes, data);
            }
            bytes.push(0);
            bytes
        }
    }

    fn blob(bytes: &mut Vec<u8>, data: &[u8]) {
        bytes.extend_from_slice(&(data.len() as u64).to_le_bytes());
        bytes.extend_from_slice(data);
    }

    #[test]
    fn handles_fragmented_and_interrupted_reads_without_large_requests() {
        struct Fragmented<'a> {
            bytes: &'a [u8],
            interrupt: bool,
        }
        impl Read for Fragmented<'_> {
            fn read(&mut self, buffer: &mut [u8]) -> std::io::Result<usize> {
                assert!(buffer.len() <= 4096);
                self.interrupt = !self.interrupt;
                if self.interrupt {
                    return Err(ErrorKind::Interrupted.into());
                }
                let length = self.bytes.len().min(buffer.len()).min(7);
                buffer[..length].copy_from_slice(&self.bytes[..length]);
                self.bytes = &self.bytes[length..];
                Ok(length)
            }
        }
        let mut fixture = Fixture::new();
        fixture.limits[2] = 10_000;
        fixture.entries[2].3 = vec![5; 9000];
        let bytes = fixture.bytes();
        let image = decode(
            &mut Fragmented {
                bytes: &bytes,
                interrupt: false,
            },
            4,
            3,
            4,
        )
        .unwrap();
        assert_eq!(image.entries[2].data, vec![5; 9000]);
    }
}
