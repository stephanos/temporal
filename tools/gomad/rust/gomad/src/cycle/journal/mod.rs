#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle CLI integration belongs to B15")
)]

use super::process::{Command, Registration, Runner, Running, SpawnOutcome};
use crate::{evidence::canonical, fork_order::Locked};
use rustix::fs::{self, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs::File,
    io::Write,
    os::unix::fs::MetadataExt,
    path::{Path, PathBuf},
    sync::atomic::AtomicBool,
};

pub(crate) const POLICY: &str = "fn232.13-owner-2026-10-24-full96-max2";
const BINDING: &str =
    "/home/agent/wt/evidence/fn232-split-closure/qualification-attempt-journal-binding-v1.json";

pub(crate) struct Authority {
    binding: PathBuf,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Enrollment {
    policy: String,
    max: u8,
    journal: PathBuf,
    outputs: PathBuf,
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Binding {
    enrollment: Enrollment,
    authority_device: u64,
    authority_inode: u64,
    journal_device: u64,
    journal_inode: u64,
    digest: String,
    events: u64,
    charged: u8,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(tag = "event", deny_unknown_fields)]
enum Event {
    Enroll {
        enrollment: Enrollment,
    },
    Reserve {
        id: u64,
        candidate: String,
        output: PathBuf,
    },
    Started {
        id: u64,
        pid: u32,
        supervisor: u32,
    },
    NoChild {
        id: u64,
    },
}
#[derive(Default)]
struct History {
    attempts: BTreeMap<u64, bool>,
    events: u64,
}
impl History {
    fn charged(&self) -> Result<u8, String> {
        u8::try_from(self.attempts.len()).map_err(|_| "attempt count overflow".into())
    }
    fn apply(&mut self, event: &Event, enrollment: &Enrollment) -> Result<(), String> {
        match event {
            Event::Enroll { enrollment: actual } if self.events == 0 && actual == enrollment => {}
            Event::Reserve {
                id,
                candidate,
                output,
            } if self.events > 0
                && *id == self.events
                && self.attempts.len() < 2
                && canonical::valid_digest(candidate)
                && output.starts_with(&enrollment.outputs)
                && output != &enrollment.outputs =>
            {
                self.attempts.insert(*id, false);
            }
            Event::Started {
                id,
                pid,
                supervisor,
            } if *pid > 0 && *supervisor > 0 && self.attempts.get(id) == Some(&false) => {
                self.attempts.insert(*id, true);
            }
            Event::NoChild { id } if self.attempts.get(id) == Some(&false) => {
                self.attempts.remove(id);
            }
            _ => return Err("invalid attempt journal sequence".into()),
        }
        self.events = self.events.checked_add(1).ok_or("attempt event overflow")?;
        Ok(())
    }
}

pub(crate) struct Journal {
    authority: Authority,
    sentinel: File,
    file: Locked,
    binding: Binding,
    history: History,
    bytes: Vec<u8>,
    degraded: bool,
}

fn fail(error: impl std::fmt::Display) -> String {
    error.to_string()
}
fn clean_path(path: &Path) -> Result<(), String> {
    if !path.is_absolute()
        || path.components().any(|part| {
            matches!(
                part,
                std::path::Component::CurDir | std::path::Component::ParentDir
            )
        })
        || path
            .parent()
            .ok_or("missing parent")?
            .canonicalize()
            .map_err(fail)?
            != path.parent().ok_or("missing parent")?
    {
        return Err("attempt authority path must have an absolute canonical parent".into());
    }
    Ok(())
}
fn directory(path: &Path) -> Result<(), String> {
    if !path.is_absolute()
        || path.canonicalize().map_err(fail)? != path
        || !std::fs::symlink_metadata(path).map_err(fail)?.is_dir()
    {
        return Err("attempt output directory must be canonical".into());
    }
    Ok(())
}
fn open(path: &Path, create: bool, append: bool) -> Result<File, String> {
    clean_path(path)?;
    let mut flags = OFlags::RDWR | OFlags::NOFOLLOW | OFlags::NONBLOCK | OFlags::CLOEXEC;
    if create {
        flags |= OFlags::CREATE | OFlags::EXCL;
    }
    if append {
        flags |= OFlags::APPEND;
    }
    let file = File::from(fs::open(path, flags, Mode::from_raw_mode(0o600)).map_err(fail)?);
    let meta = file.metadata().map_err(fail)?;
    if !meta.is_file()
        || meta.nlink() != 1
        || meta.uid() != rustix::process::getuid().as_raw()
        || meta.mode() & 0o777 != 0o600
    {
        return Err("private regular attempt authority file required".into());
    }
    Ok(file)
}
fn read(file: &File) -> Result<Vec<u8>, String> {
    use std::os::unix::fs::FileExt;
    let len = file.metadata().map_err(fail)?.len();
    if len > 8 * 1024 * 1024 {
        return Err("attempt history exceeds intake bound".into());
    }
    let mut bytes = vec![0; usize::try_from(len).map_err(fail)?];
    file.read_exact_at(&mut bytes, 0).map_err(fail)?;
    Ok(bytes)
}
fn write(file: &File, bytes: &[u8]) -> Result<(), String> {
    let mut file = file;
    file.write_all(bytes).map_err(fail)?;
    file.sync_all().map_err(fail)
}
fn sync_parent(path: &Path) -> Result<(), String> {
    File::open(path.parent().ok_or("missing parent")?)
        .map_err(fail)?
        .sync_all()
        .map_err(fail)
}
fn same(path: &Path, file: &File) -> Result<(), String> {
    let named = open(path, false, false)?;
    let a = file.metadata().map_err(fail)?;
    let b = named.metadata().map_err(fail)?;
    if a.dev() != b.dev() || a.ino() != b.ino() {
        return Err("attempt authority inode changed".into());
    }
    Ok(())
}
fn encoded(event: &Event) -> Result<Vec<u8>, String> {
    let mut b = canonical::encode(event)?;
    b.push(b'\n');
    Ok(b)
}

impl Authority {
    pub fn fixed() -> Self {
        Self {
            binding: BINDING.into(),
        }
    }
    fn sentinel(&self) -> PathBuf {
        self.binding.with_extension("enrolled")
    }
    pub fn initial_enroll(&self, journal: &Path, outputs: &Path) -> Result<Journal, String> {
        clean_path(&self.binding)?;
        clean_path(journal)?;
        directory(outputs)?;
        if journal == self.binding
            || journal == self.sentinel()
            || journal.starts_with(outputs)
            || self.binding.starts_with(outputs)
        {
            return Err("attempt authority overlaps output history".into());
        }
        if std::fs::read_dir(outputs).map_err(fail)?.next().is_some() {
            return Err("initial enrollment refuses retained output history".into());
        }
        // This permanent create-new witness is deliberately left on every failed enrollment.
        let sentinel = Locked::new(open(&self.sentinel(), true, false)?);
        fs::flock(&*sentinel, FlockOperation::NonBlockingLockExclusive).map_err(fail)?;
        sync_parent(&self.sentinel())?;
        if self.binding.try_exists().map_err(fail)? {
            return Err("attempt binding already exists".into());
        }
        let enrollment = Enrollment {
            policy: POLICY.into(),
            max: 2,
            journal: journal.into(),
            outputs: outputs.into(),
        };
        write(&sentinel, &canonical::encode(&enrollment)?)?;
        let file = Locked::new(open(journal, true, true)?);
        fs::flock(&*file, FlockOperation::NonBlockingLockExclusive).map_err(fail)?;
        let bytes = encoded(&Event::Enroll {
            enrollment: enrollment.clone(),
        })?;
        write(&file, &bytes)?;
        sync_parent(journal)?;
        let sm = sentinel.metadata().map_err(fail)?;
        let jm = file.metadata().map_err(fail)?;
        let binding = Binding {
            enrollment,
            authority_device: sm.dev(),
            authority_inode: sm.ino(),
            journal_device: jm.dev(),
            journal_inode: jm.ino(),
            digest: canonical::digest(&bytes),
            events: 1,
            charged: 0,
        };
        write(
            &open(&self.binding, true, false)?,
            &canonical::encode(&binding)?,
        )?;
        sync_parent(&self.binding)?;
        drop(file);
        drop(sentinel);
        self.open()
    }
    pub fn open(&self) -> Result<Journal, String> {
        let sentinel = open(&self.sentinel(), false, false)?;
        let enrollment: Enrollment = serde_json::from_slice(&read(&sentinel)?).map_err(fail)?;
        if enrollment.policy != POLICY || enrollment.max != 2 {
            return Err("fixed attempt enrollment policy mismatch".into());
        }
        let file = Locked::new(open(&enrollment.journal, false, true)?);
        fs::flock(&*file, FlockOperation::NonBlockingLockExclusive)
            .map_err(|e| format!("attempt journal locked: {e}"))?;
        let binding: Binding =
            serde_json::from_slice(&read(&open(&self.binding, false, false)?)?).map_err(fail)?;
        let bytes = read(&file)?;
        let mut history = History::default();
        if bytes.last() != Some(&b'\n') {
            return Err("attempt journal torn tail".into());
        }
        for line in bytes[..bytes.len() - 1].split(|b| *b == b'\n') {
            let event: Event = serde_json::from_slice(line).map_err(fail)?;
            history.apply(&event, &enrollment)?;
        }
        let value = Journal {
            authority: Self {
                binding: self.binding.clone(),
            },
            sentinel,
            file,
            binding,
            history,
            bytes,
            degraded: false,
        };
        if value.binding.enrollment != enrollment {
            return Err("attempt binding enrollment mismatch".into());
        }
        value.check()?;
        Ok(value)
    }
}
impl Journal {
    pub fn charged(&self) -> Result<u8, String> {
        self.check()?;
        self.history.charged()
    }
    fn check(&self) -> Result<(), String> {
        if self.degraded {
            return Err("attempt journal degraded".into());
        }
        same(&self.authority.sentinel(), &self.sentinel)?;
        same(&self.binding.enrollment.journal, &self.file)?;
        directory(&self.binding.enrollment.outputs)?;
        let sm = self.sentinel.metadata().map_err(fail)?;
        let jm = self.file.metadata().map_err(fail)?;
        let current: Binding =
            serde_json::from_slice(&read(&open(&self.authority.binding, false, false)?)?)
                .map_err(fail)?;
        if current != self.binding
            || sm.dev() != self.binding.authority_device
            || sm.ino() != self.binding.authority_inode
            || jm.dev() != self.binding.journal_device
            || jm.ino() != self.binding.journal_inode
            || self.bytes != read(&self.file)?
            || canonical::digest(&self.bytes) != self.binding.digest
            || self.history.events != self.binding.events
            || self.history.charged()? != self.binding.charged
            || read(&self.sentinel)? != canonical::encode(&self.binding.enrollment)?
        {
            return Err("attempt journal binding/high-water mismatch".into());
        }
        Ok(())
    }
    fn append(&mut self, event: &Event) -> Result<(), String> {
        self.check()?;
        let previous = self.binding.clone();
        let bytes = encoded(event)?;
        self.history.apply(event, &self.binding.enrollment)?;
        self.degraded = true;
        write(&self.file, &bytes)?;
        self.bytes.extend(bytes);
        self.binding.digest = canonical::digest(&self.bytes);
        self.binding.events = self.history.events;
        self.binding.charged = self.history.charged()?;
        let staged = self
            .authority
            .binding
            .with_extension(format!("pending-{}", self.binding.events));
        write(
            &open(&staged, true, false)?,
            &canonical::encode(&self.binding)?,
        )?;
        same(&self.binding.enrollment.journal, &self.file)?;
        same(&self.authority.sentinel(), &self.sentinel)?;
        let current: Binding =
            serde_json::from_slice(&read(&open(&self.authority.binding, false, false)?)?)
                .map_err(fail)?;
        if current != previous {
            return Err("attempt high-water changed during publication".into());
        }
        std::fs::rename(&staged, &self.authority.binding).map_err(fail)?;
        sync_parent(&self.authority.binding)?;
        self.degraded = false;
        self.check()
    }
    fn reserve(
        &mut self,
        candidate: &str,
        output: &Path,
        recheck: impl FnOnce() -> Result<(), String>,
    ) -> Result<u64, String> {
        self.check()?;
        if self.history.charged()? >= 2 {
            return Err("global two-attempt allowance exhausted".into());
        }
        if !canonical::valid_digest(candidate) {
            return Err("candidate digest required".into());
        }
        directory(output)?;
        if !output.starts_with(&self.binding.enrollment.outputs)
            || output == self.binding.enrollment.outputs
            || std::fs::read_dir(output).map_err(fail)?.next().is_some()
        {
            return Err("fresh enrolled attempt output required".into());
        }
        recheck()?;
        let id = self.history.events;
        self.append(&Event::Reserve {
            id,
            candidate: candidate.into(),
            output: output.into(),
        })?;
        write(
            &open(&output.join(".cycle-attempt-started"), true, false)?,
            &canonical::encode(
                &serde_json::json!({"policy":POLICY,"id":id,"candidate":candidate,"journal":self.binding.enrollment.journal}),
            )?,
        )?;
        sync_parent(&output.join(".cycle-attempt-started"))?;
        Ok(id)
    }
    #[allow(clippy::too_many_arguments)]
    pub fn start(
        &mut self,
        runner: &mut Runner,
        name: &str,
        command: Command,
        cancelled: &AtomicBool,
        candidate: &str,
        output: &Path,
        admission: &super::admission::Admission,
        probe: &impl super::admission::Probe,
    ) -> Result<Running, String> {
        if admission.evidence() != output {
            return Err("attempt output differs from admitted evidence".into());
        }
        self.start_inner(
            runner,
            name,
            command,
            cancelled,
            candidate,
            output,
            || admission.recheck(probe),
            |_| Ok(()),
        )
    }
    #[allow(clippy::too_many_arguments)]
    fn start_inner(
        &mut self,
        runner: &mut Runner,
        name: &str,
        command: Command,
        cancelled: &AtomicBool,
        candidate: &str,
        output: &Path,
        recheck: impl FnOnce() -> Result<(), String>,
        before_bind: impl FnOnce(&Registration) -> Result<(), String>,
    ) -> Result<Running, String> {
        let id = self.reserve(candidate, output, recheck)?;
        match runner.start(name, command, cancelled, |registration| {
            before_bind(registration)?;
            self.append(&Event::Started {
                id,
                pid: registration.pid,
                supervisor: registration.supervisor_pid,
            })
        }) {
            Ok(running) => Ok(running),
            Err(failure) => {
                if failure.outcome == SpawnOutcome::NoChild {
                    self.append(&Event::NoChild { id })?;
                }
                Err(format!(
                    "attempt spawn {:?}: {}",
                    failure.outcome, failure.message
                ))
            }
        }
    }
}

#[cfg(test)]
pub(crate) mod tests;
