//! Immutable event partitions for the two attempts of one native payload.
use serde::{Deserialize, Serialize};
use std::{io, path::Path};

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

pub(super) struct Boundary;

impl Boundary {
    pub(super) fn begin(_work: &Path) -> io::Result<Self> {
        Err(io::Error::other("attempt partition implementation pending"))
    }

    pub(super) fn finish(
        self,
        _work: &Path,
        _ordinal: u8,
        _destination: &Path,
    ) -> io::Result<Attempt> {
        Err(io::Error::other("attempt partition implementation pending"))
    }
}

pub(super) fn verify(_work: &Path, _attempts: &[Attempt]) -> io::Result<()> {
    Err(io::Error::other("attempt partition implementation pending"))
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
        fs::write(work.join("0.event"), vec![b'x'; umpire_backend_process::work::MAX_EVENT_BYTES + 1])?;
        assert!(Boundary::begin(&work).is_err());
        Ok(())
    }
}
