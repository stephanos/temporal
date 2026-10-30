//! Invocation-local payload admission; mise continues to own task eligibility.
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeSet,
    fmt, fs,
    fs::{File, OpenOptions},
    io::{self, Read, Write},
    os::unix::fs::{DirBuilderExt, MetadataExt, OpenOptionsExt},
    path::{Path, PathBuf},
    sync::atomic::{AtomicU64, Ordering},
    thread,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

const MAX_STAGES: usize = 256;
const MAX_BYTES: u64 = 262_144;
const POLL: Duration = Duration::from_millis(5);
static NEXT: AtomicU64 = AtomicU64::new(0);

#[derive(Clone, Debug, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
#[serde(transparent)]
pub(super) struct StageId(String);

impl StageId {
    pub(super) fn new(value: &str) -> io::Result<Self> {
        let parts: Vec<_> = value.split('/').collect();
        if value.len() > 128
            || parts.len() != 2
            || parts.iter().any(|part| {
                part.is_empty()
                    || !part.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
            })
        {
            return Err(io::Error::other("invalid qualified budget stage"));
        }
        Ok(Self(value.into()))
    }

    pub(super) fn as_str(&self) -> &str {
        &self.0
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(super) struct FailureCause {
    pub(super) stage: StageId,
    pub(super) code: i32,
    pub(super) reason: String,
}

#[derive(Debug)]
pub(super) enum Error {
    Io(io::Error),
    Aborted(FailureCause),
}

impl Error {
    pub(super) fn original_failure(&self) -> Option<&FailureCause> {
        match self {
            Self::Aborted(cause) => Some(cause),
            Self::Io(_) => None,
        }
    }
}

impl fmt::Display for Error {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(error) => error.fmt(out),
            Self::Aborted(cause) => {
                write!(out, "{}: exit {}: {}", cause.stage.as_str(), cause.code, cause.reason)
            }
        }
    }
}

impl std::error::Error for Error {}

impl From<io::Error> for Error {
    fn from(error: io::Error) -> Self {
        Self::Io(error)
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Context {
    invocation: String,
    stages: Vec<StageId>,
    drain_nanos: u64,
}

impl Context {
    fn new(invocation: &str, stages: &[StageId], drain: Duration) -> io::Result<Self> {
        let mut selected = stages.to_vec();
        selected.sort();
        if invocation.is_empty()
            || invocation.len() > 128
            || !invocation.bytes().all(|byte| byte.is_ascii_alphanumeric() || b"-_".contains(&byte))
            || selected.is_empty()
            || selected.len() > MAX_STAGES
            || selected.windows(2).any(|pair| pair[0] == pair[1])
            || drain.is_zero()
        {
            return Err(invalid("invalid coordinator invocation, inventory or drain bound"));
        }
        for stage in &selected {
            StageId::new(stage.as_str())?;
        }
        let drain_nanos = u64::try_from(drain.as_nanos()).map_err(io::Error::other)?;
        Instant::now()
            .checked_add(drain)
            .ok_or_else(|| invalid("unrepresentable drain deadline"))?;
        Ok(Self { invocation: invocation.into(), stages: selected, drain_nanos })
    }
    fn contains(&self, stage: &StageId) -> bool {
        self.stages.binary_search(stage).is_ok()
    }
    fn drain(&self) -> Duration {
        Duration::from_nanos(self.drain_nanos)
    }
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Identity {
    device: u64,
    inode: u64,
}
impl Identity {
    fn of(metadata: &fs::Metadata) -> Self {
        Self { device: metadata.dev(), inode: metadata.ino() }
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Manifest {
    version: u8,
    context: Context,
    directory: Identity,
    lock: Identity,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "kebab-case")]
enum Kind {
    SharedFirst,
    ExclusiveFirst,
    Retry,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Entry {
    stage: StageId,
    owner: String,
    kind: Kind,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "kebab-case")]
enum AbortOrigin {
    Original,
    Incomplete,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct Abort {
    cause: FailureCause,
    origin: AbortOrigin,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
struct State {
    version: u8,
    manifest_sha256: String,
    active: Vec<Entry>,
    pending: Vec<Entry>,
    entered: Vec<StageId>,
    consumed_retries: Vec<StageId>,
    finished: Vec<StageId>,
    aborted: Option<Abort>,
}

impl State {
    fn validate(&self, coordinator: &Coordinator) -> io::Result<()> {
        let context = &coordinator.manifest.context;
        if self.version != 1 || self.manifest_sha256 != coordinator.manifest_sha256 {
            return Err(invalid("crossed coordinator state manifest"));
        }
        for stages in [&self.entered, &self.consumed_retries, &self.finished] {
            let unique: BTreeSet<_> = stages.iter().collect();
            if unique.len() != stages.len()
                || stages.len() > context.stages.len()
                || stages.iter().any(|stage| !context.contains(stage))
            {
                return Err(invalid("invalid coordinator stage set"));
            }
        }
        let mut represented = BTreeSet::new();
        for entry in self.active.iter().chain(&self.pending) {
            if !self.entered.contains(&entry.stage)
                || !represented.insert(&entry.stage)
                || !valid_owner(&entry.owner)
                || self.finished.contains(&entry.stage)
                || (entry.kind == Kind::Retry) != self.consumed_retries.contains(&entry.stage)
            {
                return Err(invalid("invalid coordinator active or pending ownership"));
            }
        }
        for stage in &self.finished {
            if !represented.insert(stage) {
                return Err(invalid("finished coordinator stage still active"));
            }
        }
        if represented.len() != self.entered.len()
            || self.entered.iter().any(|stage| !represented.contains(stage))
            || self.consumed_retries.iter().any(|stage| !self.entered.contains(stage))
            || (self.active.iter().any(|entry| entry.kind != Kind::SharedFirst)
                && self.active.len() != 1)
        {
            return Err(invalid("incomplete coordinator ownership partition"));
        }
        if let Some(cause) = &self.aborted {
            validate_cause(context, &cause.cause)?;
        }
        Ok(())
    }
    fn aborted(&self) -> Result<(), Error> {
        self.aborted.as_ref().map_or(Ok(()), |cause| Err(Error::Aborted(cause.cause.clone())))
    }
    fn eligible(&self, entry: &Entry) -> bool {
        if self.active.iter().any(|active| active.kind != Kind::SharedFirst) {
            return false;
        }
        match entry.kind {
            Kind::SharedFirst => {
                !self.pending.iter().any(|request| request.kind != Kind::SharedFirst)
            }
            Kind::ExclusiveFirst => {
                self.active.is_empty()
                    && !self.pending.iter().any(|request| request.kind == Kind::Retry)
                    && self.pending.iter().find(|request| request.kind == Kind::ExclusiveFirst)
                        == Some(entry)
            }
            Kind::Retry => {
                self.active.is_empty()
                    && self.pending.iter().find(|request| request.kind == Kind::Retry)
                        == Some(entry)
            }
        }
    }
    fn remove_active(&mut self, entry: &Entry) -> io::Result<()> {
        let position = self
            .active
            .iter()
            .position(|active| active == entry)
            .ok_or_else(|| invalid("permit does not own an active payload"))?;
        self.active.remove(position);
        Ok(())
    }
    fn abort(&mut self, cause: FailureCause, origin: AbortOrigin) {
        if self.aborted.as_ref().is_none_or(|existing| {
            existing.origin == AbortOrigin::Incomplete && origin == AbortOrigin::Original
        }) {
            self.aborted = Some(Abort { cause, origin });
        }
    }
}

#[derive(Clone, Debug)]
pub(super) struct Coordinator {
    directory: PathBuf,
    manifest: Manifest,
    manifest_sha256: String,
}

impl Coordinator {
    pub(super) fn create(
        directory: &Path,
        invocation: &str,
        stages: &[StageId],
        drain: Duration,
    ) -> io::Result<Self> {
        let context = Context::new(invocation, stages, drain)?;
        match fs::DirBuilder::new().mode(0o700).create(directory) {
            Ok(()) => (),
            Err(error) if error.kind() == io::ErrorKind::AlreadyExists => (),
            Err(error) => return Err(error),
        }
        let identity = directory_identity(directory)?;
        if fs::read_dir(directory)?.next().is_some() {
            return Err(invalid("coordinator directory is not fresh"));
        }
        let lock = OpenOptions::new()
            .read(true)
            .write(true)
            .create_new(true)
            .mode(0o600)
            .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
            .open(directory.join("lock"))?;
        let manifest = Manifest {
            version: 1,
            context,
            directory: identity,
            lock: Identity::of(&lock.metadata()?),
        };
        let bytes = serde_json::to_vec(&manifest)?;
        create_record(&directory.join("manifest.json"), &bytes)?;
        let coordinator = Self {
            directory: directory.to_owned(),
            manifest,
            manifest_sha256: umpire_backend_process::bytes_sha256(&bytes),
        };
        let state = State {
            version: 1,
            manifest_sha256: coordinator.manifest_sha256.clone(),
            active: vec![],
            pending: vec![],
            entered: vec![],
            consumed_retries: vec![],
            finished: vec![],
            aborted: None,
        };
        create_record(&directory.join("state.json"), &serde_json::to_vec(&state)?)?;
        File::open(directory)?.sync_all()?;
        Ok(coordinator)
    }

    pub(super) fn open(
        directory: &Path,
        invocation: &str,
        stages: &[StageId],
        drain: Duration,
    ) -> io::Result<Self> {
        let context = Context::new(invocation, stages, drain)?;
        let bytes = read_record(&directory.join("manifest.json"))?;
        let manifest: Manifest = serde_json::from_slice(&bytes)?;
        if manifest.version != 1
            || manifest.context != context
            || manifest.directory != directory_identity(directory)?
        {
            return Err(invalid("coordinator manifest differs from admitted invocation"));
        }
        let coordinator = Self {
            directory: directory.to_owned(),
            manifest,
            manifest_sha256: umpire_backend_process::bytes_sha256(&bytes),
        };
        coordinator.update(|_| Ok(())).map_err(|error| io::Error::other(error.to_string()))?;
        Ok(coordinator)
    }

    pub(super) fn admit_shared(&self, stage: &StageId) -> Result<Permit, Error> {
        self.admit(stage, Kind::SharedFirst)
    }
    pub(super) fn admit_exclusive_first(&self, stage: &StageId) -> Result<Permit, Error> {
        self.admit(stage, Kind::ExclusiveFirst)
    }

    fn admit(&self, stage: &StageId, kind: Kind) -> Result<Permit, Error> {
        if !self.manifest.context.contains(stage) {
            return Err(invalid("stage is outside coordinator selection").into());
        }
        let entry = Entry { stage: stage.clone(), owner: new_owner()?, kind };
        self.update(|state| {
            state.aborted()?;
            if state.entered.contains(stage) {
                return Err(invalid("stage already entered coordinator").into());
            }
            state.entered.push(stage.clone());
            state.pending.push(entry.clone());
            Ok(())
        })?;
        self.wait(&entry)
    }

    pub(super) fn failure(&self) -> Result<Option<FailureCause>, Error> {
        self.update(|state| Ok(state.aborted.as_ref().map(|abort| abort.cause.clone())))
    }

    pub(super) fn abort(&self, cause: FailureCause) -> Result<(), Error> {
        validate_cause(&self.manifest.context, &cause)?;
        self.update(|state| {
            state.abort(cause, AbortOrigin::Original);
            Ok(())
        })
    }

    fn abort_incomplete(&self, cause: FailureCause) -> Result<(), Error> {
        validate_cause(&self.manifest.context, &cause)?;
        self.update(|state| {
            state.abort(cause, AbortOrigin::Incomplete);
            Ok(())
        })
    }

    fn wait(&self, entry: &Entry) -> Result<Permit, Error> {
        let deadline = Instant::now() + self.manifest.context.drain();
        loop {
            let admitted = self.update_until(deadline, |state| {
                state.aborted()?;
                if !state.pending.contains(entry) {
                    return Err(invalid("request lost coordinator ownership").into());
                }
                if !state.eligible(entry) {
                    return Ok(false);
                }
                state.pending.retain(|pending| pending != entry);
                state.active.push(entry.clone());
                Ok(true)
            })?;
            if admitted {
                return Ok(Permit {
                    coordinator: self.clone(),
                    entry: entry.clone(),
                    completed: false,
                });
            }
            if Instant::now() >= deadline {
                let cause = FailureCause {
                    stage: entry.stage.clone(),
                    code: 1,
                    reason: "bounded payload admission/drain expired".into(),
                };
                return self
                    .update_until(Instant::now(), |state| {
                        state.abort(cause, AbortOrigin::Incomplete);
                        state.aborted()
                    })
                    .and_then(|()| Err(invalid("drain failed to abort coordinator").into()));
            }
            thread::sleep(POLL.min(deadline.saturating_duration_since(Instant::now())));
        }
    }

    fn update<T>(&self, body: impl FnOnce(&mut State) -> Result<T, Error>) -> Result<T, Error> {
        self.update_until(Instant::now() + self.manifest.context.drain(), body)
    }

    fn update_until<T>(
        &self,
        deadline: Instant,
        body: impl FnOnce(&mut State) -> Result<T, Error>,
    ) -> Result<T, Error> {
        let _lock = self.lock(deadline)?;
        let mut state: State =
            serde_json::from_slice(&read_record(&self.directory.join("state.json"))?)
                .map_err(io::Error::other)?;
        state.validate(self)?;
        let original = state.clone();
        let result = body(&mut state);
        state.validate(self)?;
        if state != original {
            self.publish(&state)?;
        }
        result
    }

    fn lock(&self, deadline: Instant) -> io::Result<File> {
        if directory_identity(&self.directory)? != self.manifest.directory {
            return Err(invalid("coordinator directory identity changed"));
        }
        let file = open_record(&self.directory.join("lock"))?;
        if Identity::of(&file.metadata()?) != self.manifest.lock {
            return Err(invalid("coordinator lock identity changed"));
        }
        loop {
            match file.try_lock() {
                Ok(()) => break,
                Err(fs::TryLockError::WouldBlock) if Instant::now() < deadline => {
                    thread::sleep(POLL.min(deadline.saturating_duration_since(Instant::now())))
                }
                Err(fs::TryLockError::WouldBlock) => {
                    return Err(io::Error::new(
                        io::ErrorKind::TimedOut,
                        "coordinator lock deadline expired",
                    ));
                }
                Err(fs::TryLockError::Error(error)) => return Err(error),
            }
        }
        self.verify_files()?;
        if Identity::of(&fs::symlink_metadata(self.directory.join("lock"))?) != self.manifest.lock {
            return Err(invalid("coordinator lock replaced while waiting"));
        }
        Ok(file)
    }

    fn verify_files(&self) -> io::Result<()> {
        for entry in fs::read_dir(&self.directory)? {
            let name = entry?.file_name();
            if !["lock", "manifest.json", "state.json"].iter().any(|expected| name == *expected) {
                return Err(invalid("partial or foreign coordinator file"));
            }
        }
        if umpire_backend_process::bytes_sha256(&read_record(
            &self.directory.join("manifest.json"),
        )?) != self.manifest_sha256
        {
            return Err(invalid("coordinator manifest identity changed"));
        }
        Ok(())
    }

    fn publish(&self, state: &State) -> io::Result<()> {
        let bytes = serde_json::to_vec(state)?;
        if u64::try_from(bytes.len()).unwrap_or(u64::MAX) > MAX_BYTES {
            return Err(invalid("coordinator state exceeds byte bound"));
        }
        let mut temporary =
            tempfile::Builder::new().prefix(".state-").tempfile_in(&self.directory)?;
        temporary.write_all(&bytes)?;
        temporary.as_file().sync_all()?;
        temporary.persist(self.directory.join("state.json")).map_err(|error| error.error)?;
        File::open(&self.directory)?.sync_all()
    }
}

#[derive(Debug)]
pub(super) struct Permit {
    coordinator: Coordinator,
    entry: Entry,
    completed: bool,
}
impl Permit {
    pub(super) fn finish_fast(mut self) -> Result<(), Error> {
        let result = self.coordinator.update(|state| {
            state.remove_active(&self.entry)?;
            state.finished.push(self.entry.stage.clone());
            state.aborted()
        });
        self.completed = result.is_ok() || matches!(result, Err(Error::Aborted(_)));
        result
    }

    pub(super) fn finish_slow(mut self) -> Result<RetryToken, Error> {
        let entry = Entry { kind: Kind::Retry, ..self.entry.clone() };
        let result = self.coordinator.update(|state| {
            state.remove_active(&self.entry)?;
            if state.aborted.is_some() || self.entry.kind == Kind::Retry {
                state.finished.push(self.entry.stage.clone());
                state.abort(
                    FailureCause {
                        stage: self.entry.stage.clone(),
                        code: 1,
                        reason: "retry payload exceeded its wall bound".into(),
                    },
                    AbortOrigin::Incomplete,
                );
                return state.aborted();
            }
            state.consumed_retries.push(self.entry.stage.clone());
            state.pending.push(entry.clone());
            Ok(())
        });
        self.completed = result.is_ok() || matches!(result, Err(Error::Aborted(_)));
        result?;
        Ok(RetryToken { coordinator: self.coordinator.clone(), entry, completed: false })
    }

    pub(super) fn abort(mut self, cause: FailureCause) -> Result<(), Error> {
        validate_cause(&self.coordinator.manifest.context, &cause)?;
        if cause.stage != self.entry.stage {
            return Err(invalid("failure does not belong to permit stage").into());
        }
        let result = self.coordinator.update(|state| {
            state.remove_active(&self.entry)?;
            state.finished.push(self.entry.stage.clone());
            state.abort(cause, AbortOrigin::Original);
            Ok(())
        });
        self.completed = result.is_ok();
        result
    }
}
impl Drop for Permit {
    fn drop(&mut self) {
        if !self.completed {
            let _ = self.coordinator.abort_incomplete(FailureCause {
                stage: self.entry.stage.clone(),
                code: 1,
                reason: "payload permit dropped incomplete".into(),
            });
        }
    }
}

#[derive(Debug)]
pub(super) struct RetryToken {
    coordinator: Coordinator,
    entry: Entry,
    completed: bool,
}
impl RetryToken {
    pub(super) fn wait(mut self) -> Result<Permit, Error> {
        let result = self.coordinator.wait(&self.entry);
        self.completed = result.is_ok() || matches!(result, Err(Error::Aborted(_)));
        result
    }
}
impl Drop for RetryToken {
    fn drop(&mut self) {
        if !self.completed {
            let _ = self.coordinator.abort_incomplete(FailureCause {
                stage: self.entry.stage.clone(),
                code: 1,
                reason: "retry token dropped incomplete".into(),
            });
        }
    }
}

fn invalid(message: &str) -> io::Error {
    io::Error::other(message)
}
fn valid_owner(owner: &str) -> bool {
    !owner.is_empty()
        && owner.len() <= 96
        && owner.bytes().all(|byte| byte.is_ascii_hexdigit() || byte == b'-')
}
fn new_owner() -> io::Result<String> {
    let time = SystemTime::now().duration_since(UNIX_EPOCH).map_err(io::Error::other)?.as_nanos();
    Ok(format!("{:x}-{time:x}-{:x}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed)))
}
fn validate_cause(context: &Context, cause: &FailureCause) -> io::Result<()> {
    if !context.contains(&cause.stage)
        || !(1..=255).contains(&cause.code)
        || cause.reason.is_empty()
        || cause.reason.len() > 4096
    {
        return Err(invalid("invalid original coordinator failure"));
    }
    Ok(())
}
fn directory_identity(path: &Path) -> io::Result<Identity> {
    let metadata = fs::symlink_metadata(path)?;
    if !metadata.is_dir() {
        return Err(invalid("coordinator path is not a directory"));
    }
    let mode = metadata.mode() & 0o777;
    if mode != 0o700 {
        return Err(io::Error::other(format!(
            "coordinator directory requires mode0700, found {mode:04o}: {}",
            path.display()
        )));
    }
    let canonical = path.canonicalize()?;
    if canonical != path {
        return Err(io::Error::other(format!(
            "coordinator directory is not canonical: {} resolves to {}",
            path.display(),
            canonical.display()
        )));
    }
    Ok(Identity::of(&metadata))
}
fn open_record(path: &Path) -> io::Result<File> {
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)?;
    let metadata = file.metadata()?;
    if !metadata.is_file()
        || metadata.nlink() != 1
        || metadata.mode() & 0o777 != 0o600
        || Identity::of(&metadata) != Identity::of(&fs::symlink_metadata(path)?)
    {
        return Err(invalid("coordinator record must be a stable private regular file"));
    }
    Ok(file)
}
fn read_record(path: &Path) -> io::Result<Vec<u8>> {
    let file = open_record(path)?;
    let mut bytes = Vec::new();
    file.take(MAX_BYTES + 1).read_to_end(&mut bytes)?;
    if u64::try_from(bytes.len()).unwrap_or(u64::MAX) > MAX_BYTES {
        return Err(invalid("coordinator record exceeds byte bound"));
    }
    Ok(bytes)
}
fn create_record(path: &Path, bytes: &[u8]) -> io::Result<()> {
    if u64::try_from(bytes.len()).unwrap_or(u64::MAX) > MAX_BYTES {
        return Err(invalid("coordinator record exceeds byte bound"));
    }
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(path)?;
    file.write_all(bytes)?;
    file.sync_all()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{os::unix::fs::PermissionsExt, sync::mpsc, thread};

    fn private_root() -> io::Result<tempfile::TempDir> {
        tempfile::Builder::new()
            .permissions(fs::Permissions::from_mode(0o700))
            .tempdir_in(std::env::temp_dir().canonicalize()?)
    }

    fn stages() -> io::Result<Vec<StageId>> {
        ["framework/a", "framework/b", "consumer/c", "consumer/standalone"]
            .into_iter()
            .map(StageId::new)
            .collect()
    }

    #[test]
    fn two_slow_payloads_remove_themselves_before_bounded_drain()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(2))?;
        let a = coordinator.admit_shared(&ids[0])?;
        let b = coordinator.admit_shared(&ids[1])?;
        let competitor = coordinator.admit_shared(&ids[2])?;
        let first = a.finish_slow()?;
        let second = b.finish_slow()?;
        let (sender, receiver) = mpsc::channel();
        thread::scope(|scope| -> Result<(), Box<dyn std::error::Error>> {
            let sender2 = sender.clone();
            let a = scope.spawn(move || -> Result<(), Error> {
                let retry = first.wait()?;
                sender.send("a").map_err(io::Error::other)?;
                retry.finish_fast()
            });
            let b = scope.spawn(move || -> Result<(), Error> {
                let retry = second.wait()?;
                sender2.send("b").map_err(io::Error::other)?;
                retry.finish_fast()
            });
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            competitor.finish_fast()?;
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "a");
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "b");
            a.join().map_err(|_| io::Error::other("first retry panicked"))??;
            b.join().map_err(|_| io::Error::other("second retry panicked"))??;
            Ok(())
        })
    }

    #[test]
    fn queued_exclusive_first_cannot_overtake_requested_retry()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(2))?;
        let active = coordinator.admit_shared(&ids[0])?;
        let (sender, receiver) = mpsc::channel();
        thread::scope(|scope| -> Result<(), Box<dyn std::error::Error>> {
            let queued = scope.spawn(|| -> Result<(), Error> {
                let first = coordinator.admit_exclusive_first(&ids[3])?;
                sender.send("exclusive").map_err(io::Error::other)?;
                first.finish_fast()
            });
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            let retry = active.finish_slow()?.wait()?;
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            retry.finish_fast()?;
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "exclusive");
            queued.join().map_err(|_| io::Error::other("exclusive first panicked"))??;
            Ok(())
        })
    }

    #[test]
    fn first_real_failure_and_cancellation_status_survive_other_readers()
    -> Result<(), Box<dyn std::error::Error>> {
        for code in [37, 143] {
            let root = private_root()?;
            let ids = stages()?;
            let coordinator =
                Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let active = coordinator.admit_shared(&ids[0])?;
            let cause =
                FailureCause { stage: ids[0].clone(), code, reason: "original failure".into() };
            active.abort(cause.clone())?;
            coordinator.abort(FailureCause {
                stage: ids[1].clone(),
                code: 1,
                reason: "later incomplete callback".into(),
            })?;
            assert_eq!(coordinator.failure()?, Some(cause.clone()));
            let reader =
                Coordinator::open(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let failure = reader.admit_shared(&ids[1]).expect_err("aborted invocation reopened");
            assert_eq!(failure.original_failure(), Some(&cause));
        }
        Ok(())
    }

    #[test]
    fn second_slow_and_incomplete_permit_abort_without_reopening()
    -> Result<(), Box<dyn std::error::Error>> {
        for second_slow in [true, false] {
            let root = private_root()?;
            let ids = stages()?;
            let coordinator =
                Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let active = coordinator.admit_shared(&ids[0])?;
            if second_slow {
                let retry = active.finish_slow()?.wait()?;
                assert!(retry.finish_slow().is_err(), "second retry was manufactured");
            } else {
                drop(active);
            }
            assert!(coordinator.admit_shared(&ids[1]).is_err());
        }
        Ok(())
    }

    #[test]
    fn duplicate_stage_scope_mismatch_and_partial_state_fail_closed()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_millis(50))?;
        let active = coordinator.admit_shared(&ids[0])?;
        assert!(coordinator.admit_shared(&ids[0]).is_err());
        assert!(
            Coordinator::open(root.path(), "crossed", &ids, Duration::from_millis(50)).is_err()
        );
        active.finish_fast()?;
        fs::write(root.path().join("state.json"), b"{\"partial\":")?;
        assert!(
            Coordinator::open(root.path(), "invocation", &ids, Duration::from_millis(50)).is_err()
        );
        Ok(())
    }

    #[test]
    fn unrelated_invocation_admission_remains_independent() -> Result<(), Box<dyn std::error::Error>>
    {
        let first = private_root()?;
        let second = private_root()?;
        let ids = stages()?;
        let a = Coordinator::create(first.path(), "first", &ids, Duration::from_secs(1))?;
        let b = Coordinator::create(second.path(), "second", &ids, Duration::from_secs(1))?;
        let retry = a.admit_shared(&ids[0])?.finish_slow()?;
        b.admit_shared(&ids[0])?.finish_fast()?;
        retry.wait()?.finish_fast()?;
        Ok(())
    }

    #[test]
    fn missing_owner_never_clears_active_work_and_drain_is_finite()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_millis(30))?;
        let missing = coordinator.admit_shared(&ids[0])?;
        let slow = coordinator.admit_shared(&ids[1])?.finish_slow()?;
        std::mem::forget(missing);
        let failure = slow.wait().expect_err("missing owner's active entry was cleared");
        assert_eq!(failure.original_failure().map(|cause| cause.code), Some(1));
        let state: State = serde_json::from_slice(&read_record(&root.path().join("state.json"))?)?;
        assert!(state.active.iter().any(|entry| entry.stage == ids[0]));
        assert!(state.consumed_retries.contains(&ids[1]));
        assert!(coordinator.admit_shared(&ids[2]).is_err());
        Ok(())
    }

    #[test]
    fn replaced_lock_symlink_state_and_abandoned_partial_fail_closed()
    -> Result<(), Box<dyn std::error::Error>> {
        for mode in ["lock", "symlink", "partial", "unknown", "duplicate"] {
            let root = private_root()?;
            let ids = stages()?;
            let coordinator =
                Coordinator::create(root.path(), "invocation", &ids, Duration::from_millis(30))?;
            match mode {
                "lock" => {
                    let lock = root.path().join("lock");
                    fs::rename(&lock, root.path().join("saved-lock"))?;
                    create_record(&lock, b"")?;
                    fs::remove_file(root.path().join("saved-lock"))?;
                }
                "symlink" => {
                    let foreign = root.path().join("foreign");
                    fs::write(&foreign, b"unchanged")?;
                    fs::remove_file(root.path().join("state.json"))?;
                    std::os::unix::fs::symlink(&foreign, root.path().join("state.json"))?;
                }
                "partial" => fs::write(root.path().join(".state-abandoned"), b"partial")?,
                _ => {
                    let mut state: serde_json::Value =
                        serde_json::from_slice(&read_record(&root.path().join("state.json"))?)?;
                    if mode == "unknown" {
                        state["unknown"] = true.into();
                    } else {
                        state["entered"] = serde_json::json!([ids[0], ids[0]]);
                    }
                    fs::write(root.path().join("state.json"), serde_json::to_vec(&state)?)?;
                }
            }
            assert!(coordinator.admit_shared(&ids[0]).is_err(), "{mode}");
            assert!(
                Coordinator::open(root.path(), "invocation", &ids, Duration::from_millis(30))
                    .is_err(),
                "{mode}"
            );
            if mode == "symlink" {
                assert_eq!(fs::read(root.path().join("foreign"))?, b"unchanged");
            }
        }
        Ok(())
    }

    #[test]
    fn zero_failure_code_and_unselected_stage_cannot_abort_other_work()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))?;
        for (stage, code) in [(ids[0].clone(), 0), (StageId::new("framework/not-selected")?, 37)] {
            assert!(
                coordinator.abort(FailureCause { stage, code, reason: "invalid".into() }).is_err()
            );
        }
        assert!(coordinator.failure()?.is_none());
        coordinator.admit_shared(&ids[0])?.finish_fast()?;
        assert!(coordinator.admit_shared(&ids[0]).is_err());
        Ok(())
    }

    #[test]
    fn public_fixture_directory_is_refused_without_repairing_permissions()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = private_root()?;
        fs::set_permissions(root.path(), fs::Permissions::from_mode(0o755))?;
        let ids = stages()?;
        let error = Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))
            .expect_err("public directory was admitted");
        assert!(error.to_string().contains("mode0700, found 0755"));
        assert_eq!(fs::symlink_metadata(root.path())?.mode() & 0o777, 0o755);
        assert!(fs::read_dir(root.path())?.next().is_none());
        Ok(())
    }
}
