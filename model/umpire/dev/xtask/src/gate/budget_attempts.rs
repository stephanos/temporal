//! Immutable event partitions for the two attempts of one native payload.
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs,
    io::{self, Read, Write},
    os::unix::{
        ffi::OsStrExt,
        fs::{MetadataExt, OpenOptionsExt},
    },
    path::Path,
};
use umpire_backend_process::{
    bytes_sha256, structural_sha256,
    work::{MAX_EVENT_BYTES, MAX_EVENTS},
};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Event {
    name: String,
    sha256: String,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct Attempt {
    version: u8,
    scope: String,
    ordinal: u8,
    events: Vec<Event>,
}

pub(super) struct Boundary {
    scope: String,
    before: BTreeMap<String, String>,
}

impl Boundary {
    pub(super) fn begin(work: &Path) -> io::Result<Self> {
        let (scope, before) = snapshot(work)?;
        Ok(Self { scope, before })
    }

    pub(super) fn finish(
        self,
        work: &Path,
        ordinal: u8,
        destination: &Path,
    ) -> io::Result<Attempt> {
        if !(1..=2).contains(&ordinal) || (ordinal == 1 && !self.before.is_empty()) {
            return Err(invalid("invalid attempt boundary or ordinal"));
        }
        let (scope, after) = snapshot(work)?;
        if scope != self.scope || self.before.iter().any(|(name, sha)| after.get(name) != Some(sha))
        {
            return Err(invalid("prior attempt observations changed"));
        }
        let events = after
            .into_iter()
            .filter_map(|(name, sha256)| {
                (!self.before.contains_key(&name)).then_some(Event { name, sha256 })
            })
            .collect();
        let attempt = Attempt { version: 1, scope, ordinal, events };
        let bytes = serde_json::to_vec(&attempt)?;
        let mut file =
            fs::OpenOptions::new().write(true).create_new(true).mode(0o600).open(destination)?;
        file.write_all(&bytes)?;
        file.sync_all()?;
        Ok(attempt)
    }
}

impl Attempt {
    pub(super) fn read(path: &Path) -> io::Result<Self> {
        let bytes = record(path, MAX_EVENTS as usize * 128 + 4096)?;
        serde_json::from_slice(&bytes).map_err(io::Error::other)
    }

    pub(super) fn event_names(&self) -> Vec<String> {
        self.events.iter().map(|event| event.name.clone()).collect()
    }
}

pub(super) fn verify(work: &Path, attempts: &[Attempt]) -> io::Result<()> {
    if attempts.is_empty() || attempts.len() > 2 {
        return Err(invalid("invalid completed attempt count"));
    }
    let (scope, observed) = snapshot(work)?;
    let mut partition = BTreeMap::new();
    for (index, attempt) in attempts.iter().enumerate() {
        if attempt.version != 1
            || attempt.scope != scope
            || usize::from(attempt.ordinal) != index + 1
        {
            return Err(invalid("crossed or out-of-order attempt authority"));
        }
        for event in &attempt.events {
            if partition.insert(event.name.clone(), event.sha256.clone()).is_some() {
                return Err(invalid("overlapping attempt observations"));
            }
        }
    }
    if partition != observed {
        return Err(invalid("attempt observations are not an exact partition"));
    }
    Ok(())
}

fn invalid(message: &str) -> io::Error {
    io::Error::other(message)
}

fn identity(work: &Path) -> io::Result<String> {
    let metadata = fs::symlink_metadata(work)?;
    if !metadata.is_dir() || metadata.is_symlink() || work.canonicalize()? != work {
        return Err(invalid("work directory must be canonical"));
    }
    Ok(structural_sha256(&(work.as_os_str().as_bytes(), metadata.dev(), metadata.ino())))
}

fn record(path: &Path, maximum: usize) -> io::Result<Vec<u8>> {
    let before = fs::symlink_metadata(path)?;
    if !before.is_file()
        || before.is_symlink()
        || before.nlink() != 1
        || before.len() > maximum as u64
    {
        return Err(invalid("invalid attempt observation file"));
    }
    let file = fs::OpenOptions::new()
        .read(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)?;
    let metadata = file.metadata()?;
    if (before.dev(), before.ino(), before.len())
        != (metadata.dev(), metadata.ino(), metadata.len())
    {
        return Err(invalid("attempt observation replaced during read"));
    }
    let mut bytes = Vec::new();
    file.take(maximum as u64 + 1).read_to_end(&mut bytes)?;
    if bytes.len() > maximum {
        return Err(invalid("attempt observation exceeds byte limit"));
    }
    Ok(bytes)
}

fn snapshot(work: &Path) -> io::Result<(String, BTreeMap<String, String>)> {
    let scope = identity(work)?;
    let mut events = BTreeMap::new();
    let mut claims = std::collections::BTreeSet::new();
    for (index, entry) in fs::read_dir(work)?.enumerate() {
        if index as u64 >= MAX_EVENTS * 2 {
            return Err(invalid("attempt observation count exceeds limit"));
        }
        let entry = entry?;
        let name =
            entry.file_name().into_string().map_err(|_| invalid("invalid event filename"))?;
        let (slot, kind) = name.split_once('.').ok_or_else(|| invalid("partial event filename"))?;
        let number = slot.parse::<u64>().map_err(|_| invalid("invalid observation slot"))?;
        if number >= MAX_EVENTS || number.to_string() != slot {
            return Err(invalid("noncanonical observation slot"));
        }
        match kind {
            "claim" => {
                record(&entry.path(), 0)?;
                claims.insert(format!("{slot}.event"));
            }
            "event" => {
                let bytes = record(&entry.path(), MAX_EVENT_BYTES)?;
                if !bytes.ends_with(b"\n") {
                    return Err(invalid("partial observation record"));
                }
                events.insert(name, bytes_sha256(&bytes));
            }
            _ => return Err(invalid("partial or unexpected attempt observation")),
        }
    }
    if !claims.iter().eq(events.keys()) || identity(work)? != scope {
        return Err(invalid("incomplete or replaced attempt observations"));
    }
    Ok((scope, events))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{fs, os::unix::fs::symlink};

    fn event(work: &Path, slot: u32, bytes: &[u8]) -> io::Result<()> {
        fs::write(work.join(format!("{slot}.claim")), b"")?;
        fs::write(work.join(format!("{slot}.event")), bytes)
    }

    #[test]
    fn attempts_partition_exact_events_without_clearing_first_run() -> io::Result<()> {
        let root = tempfile::tempdir()?;
        let work = root.path().join("work");
        fs::create_dir(&work)?;
        let boundary = Boundary::begin(&work)?;
        event(&work, 7, b"1\ttests-begin\tone\n")?;
        let first = boundary.finish(&work, 1, &root.path().join("one.json"))?;
        assert_eq!(first.events.iter().map(|e| e.name.as_str()).collect::<Vec<_>>(), ["7.event"]);
        let boundary = Boundary::begin(&work)?;
        event(&work, 19, b"1\ttests-begin\ttwo\n")?;
        let second = boundary.finish(&work, 2, &root.path().join("two.json"))?;
        assert_eq!(second.events.iter().map(|e| e.name.as_str()).collect::<Vec<_>>(), ["19.event"]);
        verify(&work, &[first.clone(), second.clone()])?;
        assert!(verify(&work, &[second.clone(), first.clone()]).is_err());
        assert!(verify(&work, &[first.clone(), first.clone()]).is_err());
        assert!(verify(&work, &[second.clone()]).is_err());
        assert!(verify(&work, &[first.clone()]).is_err());
        assert!(verify(&work, &[first.clone(), second.clone(), second.clone()]).is_err());
        let decoded: Attempt = serde_json::from_slice(&fs::read(root.path().join("one.json"))?)?;
        assert_eq!(decoded, first);
        fs::write(work.join("7.event"), b"changed\n")?;
        assert!(verify(&work, &[first, second]).is_err());
        Ok(())
    }

    #[test]
    fn changed_prior_events_and_cross_scope_boundaries_refuse() -> io::Result<()> {
        let root = tempfile::tempdir()?;
        let a = root.path().join("a");
        let b = root.path().join("b");
        fs::create_dir(&a)?;
        fs::create_dir(&b)?;
        event(&a, 1, b"before\n")?;
        let boundary = Boundary::begin(&a)?;
        fs::write(a.join("1.event"), b"after\n")?;
        assert!(boundary.finish(&a, 2, &root.path().join("changed.json")).is_err());
        let boundary = Boundary::begin(&a)?;
        assert!(boundary.finish(&b, 2, &root.path().join("crossed.json")).is_err());
        assert!(!root.path().join("changed.json").exists());
        assert!(!root.path().join("crossed.json").exists());
        Ok(())
    }

    #[test]
    fn first_attempt_requires_empty_boundary_and_manifest_never_overwrites() -> io::Result<()> {
        let root = tempfile::tempdir()?;
        let work = root.path().join("work");
        fs::create_dir(&work)?;
        event(&work, 0, b"earlier\n")?;
        assert!(Boundary::begin(&work)?.finish(&work, 1, &root.path().join("first.json")).is_err());
        let boundary = Boundary::begin(&work)?;
        fs::write(root.path().join("existing.json"), b"preserved")?;
        assert!(boundary.finish(&work, 2, &root.path().join("existing.json")).is_err());
        assert_eq!(fs::read(root.path().join("existing.json"))?, b"preserved");
        let boundary = Boundary::begin(&work)?;
        assert!(boundary.finish(&work, 3, &root.path().join("third.json")).is_err());
        Ok(())
    }

    #[test]
    fn partial_noncanonical_symlink_and_unbounded_records_refuse() -> io::Result<()> {
        let root = tempfile::tempdir()?;
        let work = root.path().join("work");
        fs::create_dir(&work)?;
        fs::write(work.join("0.claim"), b"")?;
        assert!(Boundary::begin(&work).is_err());
        fs::write(work.join("0.event"), b"complete\n")?;
        let valid = Boundary::begin(&work)?;
        fs::write(work.join("1.partial"), b"partial")?;
        assert!(valid.finish(&work, 2, &root.path().join("partial.json")).is_err());
        fs::remove_file(work.join("1.partial"))?;
        fs::write(work.join("00.claim"), b"")?;
        fs::write(work.join("00.event"), b"alias\n")?;
        assert!(Boundary::begin(&work).is_err());
        fs::remove_file(work.join("00.claim"))?;
        fs::remove_file(work.join("00.event"))?;
        fs::remove_file(work.join("0.event"))?;
        symlink(root.path().join("external"), work.join("0.event"))?;
        assert!(Boundary::begin(&work).is_err());
        fs::remove_file(work.join("0.event"))?;
        fs::write(work.join("0.event"), vec![b'x'; MAX_EVENT_BYTES + 1])?;
        assert!(Boundary::begin(&work).is_err());
        Ok(())
    }
}
