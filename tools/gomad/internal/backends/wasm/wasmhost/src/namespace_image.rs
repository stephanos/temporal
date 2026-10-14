use crate::namespace::{Boundary, Image, ImageEntry, Limits};
use std::collections::{BTreeMap, BTreeSet};
use std::io::{ErrorKind, Read};

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
            Err(error) if error.kind() == ErrorKind::Interrupted => continue,
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
        let length = remaining.min(chunk.len() as u64) as usize;
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

    const ROOT_IMAGE: &[u8] = b"GMWN\x01\0\0\0\
        \x01\0\0\0\0\0\0\0\x05\0\0\0\0\0\0\0\
        \x02\0\0\0\0\0\0\0\x03\0\0\0\0\0\0\0\
        \x04\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
        \0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
        \0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0/\
        \x01\0\0\0\0\0\0\0x\0";

    #[test]
    fn decodes_hand_checked_root_image() {
        let image = decode(&mut &ROOT_IMAGE[..], 1, 3, 4).unwrap();
        assert!(image.entries.is_empty());
        assert!(image.preopens.is_empty());
        assert!(image.writable_directories.is_empty());
        assert_eq!(image.working_directory, b"/");
        assert_eq!(image.stdin, b"x");
        assert_eq!(image.limits.files, 1);
        assert_eq!(image.limits.descriptors, 5);
        assert_eq!(image.limits.filesystem_bytes, 2);
        assert_eq!(image.limits.output_bytes, 3);
        assert_eq!(image.epoch_nanos, 4);
    }

    #[test]
    fn decodes_go_literal_to_expected_complete_namespace() {
        let bytes = b"GMWN\x01\0\0\0\
            \x0a\0\0\0\0\0\0\0\x05\0\0\0\0\0\0\0\
            \x10\0\0\0\0\0\0\0\x20\0\0\0\0\0\0\0\
            \x09\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
            \0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
            \0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0/\
            \x02\0\0\0\0\0\0\0hi\0";
        let decoded =
            crate::namespace::Namespace::new(decode(&mut &bytes[..], 1, 32, 9).unwrap()).unwrap();
        let expected = crate::namespace::Namespace::new(Image {
            entries: vec![],
            writable_directories: vec![],
            preopens: vec![],
            working_directory: b"/".to_vec(),
            stdin: b"hi".to_vec(),
            limits: Limits {
                files: 10,
                descriptors: 5,
                filesystem_bytes: 16,
                output_bytes: 32,
            },
            epoch_nanos: 9,
        })
        .unwrap();
        assert_eq!(decoded.canonical_state(), expected.canonical_state());
    }

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

        fn decode(&self) -> Result<Image, Boundary> {
            decode(&mut &self.bytes()[..], self.entries.len() as u64 + 1, 3, 4)
        }
    }

    fn blob(bytes: &mut Vec<u8>, data: &[u8]) {
        bytes.extend_from_slice(&(data.len() as u64).to_le_bytes());
        bytes.extend_from_slice(data);
    }

    #[test]
    fn preserves_inode_order_and_captured_readonly_under_writable_ancestor() {
        let image = Fixture::new().decode().unwrap();
        assert_eq!(
            image
                .entries
                .iter()
                .map(|entry| &entry.path[..])
                .collect::<Vec<_>>(),
            [
                b"/workspace".as_slice(),
                b"/workspace/input",
                b"/workspace/input/a"
            ]
        );
        let mut ns = crate::namespace::Namespace::new(image).unwrap();
        let reply = ns
            .execute(
                crate::namespace::Operation::UnlinkFile {
                    fd: 3,
                    path: b"input/a".to_vec(),
                },
                0,
            )
            .unwrap();
        assert_eq!(reply.errno, crate::namespace::ROFS);
    }

    #[test]
    fn rejects_invalid_header_limits_and_expected_identity() {
        for (index, value) in [
            (0, 0),
            (0, (1 << 20) + 1),
            (1, 4),
            (1, (1 << 20) + 1),
            (2, 0),
            (2, (1 << 30) + 1),
            (3, 0),
            (3, (1 << 30) + 1),
            (4, i64::MAX as u64 + 1),
        ] {
            let mut fixture = Fixture::new();
            fixture.limits[index] = value;
            assert_eq!(
                fixture.decode().unwrap_err().kind,
                "invalid",
                "{index}: {value}"
            );
        }
        for (files, output, epoch) in [(5, 3, 4), (4, 2, 4), (4, 3, 5)] {
            assert_eq!(
                decode(&mut &Fixture::new().bytes()[..], files, output, epoch)
                    .unwrap_err()
                    .kind,
                "invalid"
            );
        }
    }

    #[test]
    fn rejects_invalid_entry_structure() {
        let cases: &[fn(&mut Fixture)] = &[
            |f| f.entries[0].0 = 7,
            |f| f.entries[0].1 = 2,
            |f| f.entries[0].3 = b"bad".to_vec(),
            |f| f.entries.swap(0, 1),
            |f| f.entries[0].0 = 2,
            |f| f.entries[2].2 = f.entries[1].2.clone(),
            |f| f.entries[2].2 = b"/".to_vec(),
            |f| f.entries[2].2 = b"relative".to_vec(),
            |f| f.entries[2].2 = b"/workspace//a".to_vec(),
            |f| f.entries[2].2 = b"/workspace/./a".to_vec(),
            |f| f.entries[2].2 = b"/workspace/../a".to_vec(),
            |f| f.entries[2].2 = b"/workspace/a/".to_vec(),
            |f| f.entries[2].2 = b"/workspace/a\0".to_vec(),
            |f| f.entries[2].2 = b"/workspace/\xff".to_vec(),
        ];
        for (index, change) in cases.iter().enumerate() {
            let mut fixture = Fixture::new();
            change(&mut fixture);
            assert_eq!(
                fixture.decode().unwrap_err().kind,
                "invalid",
                "case {index}"
            );
        }
    }

    #[test]
    fn rejects_invalid_mounts_and_working_directory() {
        let cases: &[fn(&mut Fixture)] = &[
            |f| f.captured.push(b"/workspace/input"),
            |f| f.captured.push(b"/a"),
            |f| f.captured[0] = b"/missing",
            |f| f.captured[0] = b"/workspace/input/a",
            |f| f.entries[1].1 = 0,
            |f| f.entries[2].1 = 0,
            |f| f.captured.clear(),
            |f| f.writable[0] = b"/workspace/input",
            |f| {
                f.writable[0] = b"/workspace/input/sub";
                f.entries
                    .push((1, 0, b"/workspace/input/sub".to_vec(), vec![]));
            },
            |f| f.writable.push(b"/workspace"),
            |f| f.writable[0] = b"/missing",
            |f| f.preopens.clear(),
            |f| f.preopens.push(b"/workspace/input"),
            |f| f.preopens.swap(0, 1),
            |f| f.preopens[0] = b"/missing",
            |f| f.preopens[1] = b"/workspace/input/a",
            |f| f.cwd = b"/missing".to_vec(),
            |f| f.cwd = b"/workspace/input/a".to_vec(),
            |f| f.cwd = b"/workspace/".to_vec(),
        ];
        for (index, change) in cases.iter().enumerate() {
            let mut fixture = Fixture::new();
            change(&mut fixture);
            assert_eq!(
                fixture.decode().unwrap_err().kind,
                "invalid",
                "case {index}"
            );
        }
    }

    #[test]
    fn rejects_file_and_data_budgets_before_constructor_capacity() {
        for (index, limit) in [(0, 3), (1, 5), (2, 4)] {
            let mut fixture = Fixture::new();
            fixture.limits[index] = limit;
            if index == 1 {
                fixture.preopens.push(b"/workspace/third");
                fixture
                    .entries
                    .push((1, 0, b"/workspace/third".to_vec(), vec![]));
            }
            assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
        }
        for offset in [48, 56, 64, 72] {
            let mut bytes = ROOT_IMAGE.to_vec();
            bytes[offset..offset + 8].copy_from_slice(&u64::MAX.to_le_bytes());
            assert_eq!(
                decode(&mut &bytes[..], 1, 3, 4).unwrap_err().kind,
                "invalid"
            );
        }
    }

    #[test]
    fn rejects_magic_terminator_trailing_data_and_every_truncation() {
        for length in 0..ROOT_IMAGE.len() {
            assert_eq!(
                decode(&mut &ROOT_IMAGE[..length], 1, 3, 4)
                    .unwrap_err()
                    .kind,
                "invalid"
            );
        }
        for offset in [0, 4, ROOT_IMAGE.len() - 1] {
            let mut bytes = ROOT_IMAGE.to_vec();
            bytes[offset] = 9;
            assert_eq!(
                decode(&mut &bytes[..], 1, 3, 4).unwrap_err().kind,
                "invalid"
            );
        }
        let mut bytes = ROOT_IMAGE.to_vec();
        bytes.push(0);
        assert_eq!(
            decode(&mut &bytes[..], 1, 3, 4).unwrap_err().kind,
            "invalid"
        );
    }

    #[test]
    fn huge_blob_claim_does_not_allocate_from_length() {
        let mut bytes = ROOT_IMAGE[..80].to_vec();
        bytes.extend_from_slice(&u64::MAX.to_le_bytes());
        assert_eq!(
            decode(&mut &bytes[..], 1, 3, 4).unwrap_err().kind,
            "invalid"
        );
        let mut bytes = ROOT_IMAGE[..89].to_vec();
        bytes.extend_from_slice(&u64::MAX.to_le_bytes());
        assert_eq!(
            decode(&mut &bytes[..], 1, 3, 4).unwrap_err().kind,
            "invalid"
        );
    }

    #[test]
    fn tiny_filesystem_budget_allows_large_path_metadata() {
        let mut fixture = Fixture::new();
        fixture.limits[2] = 2;
        fixture.stdin.clear();
        fixture.entries[2].0 = 1;
        fixture.entries[2].3.clear();
        fixture.entries[2].2 =
            [b"/workspace/input/".as_slice(), &vec![b'a'; (4 << 20) + 1]].concat();
        let image = fixture.decode().unwrap();
        assert!(image.entries[2].path.len() > 4 << 20);
        assert_eq!(image.limits.filesystem_bytes, 2);
    }

    #[test]
    fn file_and_data_counts_include_root_and_stdin_but_exclude_standard_fds() {
        let mut fixture = Fixture::new();
        fixture.limits[0] = 4;
        fixture.limits[2] = 5;
        let image = fixture.decode().unwrap();
        assert_eq!(image.entries.len(), 3);
        assert_eq!(image.stdin, b"in");
        assert!(crate::namespace::Namespace::new(image).is_ok());
        fixture.limits[2] = 3;
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
        fixture.stdin.clear();
        assert!(fixture.decode().is_ok());
    }

    #[test]
    fn preserves_unsorted_writable_config_and_entry_order() {
        let mut fixture = Fixture::new();
        fixture.captured.clear();
        fixture.writable = vec![b"/z", b"/a"];
        fixture.preopens = vec![b"/a", b"/z"];
        fixture.cwd = b"/z".to_vec();
        fixture.entries = vec![
            (1, 0, b"/z".to_vec(), vec![]),
            (1, 0, b"/a".to_vec(), vec![]),
            (2, 0, b"/z/writable".to_vec(), b"bytes".to_vec()),
        ];
        let image = fixture.decode().unwrap();
        assert_eq!(image.writable_directories, [b"/z".to_vec(), b"/a".to_vec()]);
        assert_eq!(image.entries[0].path, b"/z");
        assert_eq!(image.entries[1].path, b"/a");
        assert!(!image.entries[2].readonly);
    }

    #[test]
    fn captured_root_matches_components_instead_of_raw_prefix() {
        let mut fixture = Fixture::new();
        fixture.writable.push(b"/workspace/input2");
        fixture.preopens.push(b"/workspace/input2");
        fixture
            .entries
            .push((1, 0, b"/workspace/input2".to_vec(), vec![]));
        assert!(fixture.decode().is_ok());
        fixture
            .entries
            .push((2, 1, b"/workspace/input2/a".to_vec(), vec![]));
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
    }

    #[test]
    fn rejects_root_mounts() {
        let mut fixture = Fixture::new();
        fixture.entries.clear();
        fixture.captured = vec![b"/"];
        fixture.writable.clear();
        fixture.preopens = vec![b"/"];
        fixture.cwd = b"/".to_vec();
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
        fixture.captured.clear();
        fixture.writable = vec![b"/"];
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
        fixture.captured = vec![b"/"];
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
    }

    #[test]
    fn rejects_overlapping_captured_roots() {
        let mut fixture = Fixture::new();
        fixture.captured.push(b"/workspace/input/sub");
        fixture.preopens.push(b"/workspace/input/sub");
        fixture
            .entries
            .push((1, 1, b"/workspace/input/sub".to_vec(), vec![]));
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
        fixture.captured.insert(1, b"/workspace/input-");
        fixture.preopens.insert(2, b"/workspace/input-");
        fixture
            .entries
            .push((1, 1, b"/workspace/input-".to_vec(), vec![]));
        assert_eq!(fixture.decode().unwrap_err().kind, "invalid");
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

    #[test]
    fn refuses_truncated_entry_payloads() {
        let bytes = Fixture::new().bytes();
        for length in 80..bytes.len() {
            assert_eq!(
                decode(&mut &bytes[..length], 4, 3, 4).unwrap_err().kind,
                "invalid"
            );
        }
    }
}
