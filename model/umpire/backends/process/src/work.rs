//! Invocation-scoped observations from native producers.
use std::{
    env, fs,
    io::{self, Write},
    path::PathBuf,
    sync::{
        OnceLock,
        atomic::{AtomicU64, Ordering},
    },
};

static NEXT: AtomicU64 = AtomicU64::new(0);
static STAGE: OnceLock<StageScope> = OnceLock::new();

#[derive(PartialEq, Eq)]
struct StageScope {
    root: PathBuf,
    scope: String,
    stage: String,
}

fn identifier(value: &str) -> bool {
    value.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
}

/// Freeze the scheduler's explicit scope without mutating process environment.
/// A native callback process owns exactly one stage; later rebinding is refused.
pub fn admit_stage(root: &std::path::Path, scope: &str, stage: &str) -> io::Result<()> {
    if !root.is_absolute() || stage.is_empty() || !identifier(stage) || !identifier(scope) {
        return Err(io::Error::other("invalid explicit work observation scope"));
    }
    let admitted =
        StageScope { root: root.canonicalize()?, scope: scope.into(), stage: stage.into() };
    let frozen = STAGE.get_or_init(|| admitted);
    if frozen.root != root.canonicalize()? || frozen.scope != scope || frozen.stage != stage {
        return Err(io::Error::other("work observation scope cannot be rebound"));
    }
    directory()?.ok_or_else(|| io::Error::other("missing explicit observation scope"))?;
    Ok(())
}

/// Carry an explicitly admitted native stage into its ordinary child commands.
/// Clean child constructors may subsequently remove these variables as usual.
pub fn configure_scope(command: &mut std::process::Command) {
    if let Some(admitted) = STAGE.get() {
        command
            .env("UMPIRE_GATE_RECEIPTS", &admitted.root)
            .env("UMPIRE_GATE_SCOPE", &admitted.scope)
            .env("UMPIRE_GATE_STAGE", &admitted.stage);
    }
}
/// Maximum completed producer events in one stage.
pub const MAX_EVENTS: u64 = 100_000;
/// Maximum bytes in one producer observation.
pub const MAX_EVENT_BYTES: usize = 16_384;

/// Resolve only the current stage's already-created observation directory.
pub fn directory() -> io::Result<Option<PathBuf>> {
    if let Some(admitted) = STAGE.get() {
        if env::var_os("UMPIRE_GATE_RECEIPTS").is_some_and(|root| {
            !PathBuf::from(root).canonicalize().is_ok_and(|root| root == admitted.root)
        }) || env::var_os("UMPIRE_GATE_STAGE")
            .is_some_and(|stage| stage != admitted.stage.as_str())
            || env::var_os("UMPIRE_GATE_SCOPE")
                .is_some_and(|scope| scope != admitted.scope.as_str())
        {
            return Err(io::Error::other("environment conflicts with admitted observation scope"));
        }
        let directory = admitted.root.join("work").join(&admitted.scope).join(&admitted.stage);
        let metadata = fs::symlink_metadata(&directory)?;
        if !metadata.is_dir() || metadata.is_symlink() {
            return Err(io::Error::other("invalid admitted observation directory"));
        }
        return Ok(Some(directory));
    }
    let Some(root) = env::var_os("UMPIRE_GATE_RECEIPTS") else { return Ok(None) };
    let Some(stage) = env::var_os("UMPIRE_GATE_STAGE") else {
        return Err(io::Error::other("work observation scope lacks stage"));
    };
    let stage = stage
        .to_str()
        .filter(|stage| {
            !stage.is_empty()
                && stage.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
        })
        .ok_or_else(|| io::Error::other("invalid work observation stage"))?;
    let root = PathBuf::from(root);
    if !root.is_absolute() {
        return Err(io::Error::other("work observations need an absolute invocation directory"));
    }
    let scope = env::var("UMPIRE_GATE_SCOPE").unwrap_or_default();
    if !scope.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-') {
        return Err(io::Error::other("invalid work observation scope"));
    }
    let directory = root.join("work").join(scope).join(stage);
    let metadata = fs::symlink_metadata(&directory)?;
    if !metadata.is_dir() || metadata.is_symlink() {
        return Err(io::Error::other("invalid work observation directory"));
    }
    Ok(Some(directory))
}

/// Publish one bounded event without appending to another process's file.
/// Missing scope disables observations; malformed active scope refuses execution.
pub fn record(kind: &str, values: &str) -> io::Result<()> {
    record_version(1, kind, values)
}

fn record_version(version: u8, kind: &str, values: &str) -> io::Result<()> {
    let Some(directory) = directory()? else { return Ok(()) };
    if kind.is_empty()
        || !kind.bytes().all(|byte| byte.is_ascii_lowercase() || byte == b'-')
        || values.bytes().any(|byte| byte == b'\n' || byte == b'\r')
    {
        return Err(io::Error::other("invalid work observation record"));
    }
    let record = format!("{version}\t{kind}\t{values}\n");
    if record.len() > MAX_EVENT_BYTES {
        return Err(io::Error::other("work observation exceeds byte limit"));
    }
    // Slot names bound all concurrent producers together; PID reuse cannot overwrite evidence.
    for slot in NEXT.fetch_add(1, Ordering::Relaxed)..MAX_EVENTS {
        let claim = directory.join(format!("{slot}.claim"));
        match fs::OpenOptions::new().write(true).create_new(true).open(claim) {
            Ok(_) => {
                NEXT.fetch_max(slot + 1, Ordering::Relaxed);
                let partial = directory.join(format!("{slot}.partial"));
                let mut file =
                    fs::OpenOptions::new().write(true).create_new(true).open(&partial)?;
                file.write_all(record.as_bytes())?;
                file.sync_all()?;
                fs::hard_link(&partial, directory.join(format!("{slot}.event")))?;
                fs::remove_file(partial)?;
                return Ok(());
            }
            Err(error) if error.kind() == io::ErrorKind::AlreadyExists => (),
            Err(error) => return Err(error),
        }
    }
    Err(io::Error::other("work observation event limit exceeded"))
}

/// Startup evidence for an invocation already classified as an admitted checker.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum TlcStartup {
    /// The captured bytes contain the admitted checker's startup output.
    Started,
    /// No complete, decodable checker startup evidence was captured.
    Unknown,
}

/// Classify captured admitted-checker output without treating silence as a non-start.
pub fn checker_startup(output: &[u8]) -> TlcStartup {
    let Ok(output) = std::str::from_utf8(output) else {
        return TlcStartup::Unknown;
    };
    if output.contains("TLC2 Version ")
        || output.contains(
            "TLC - provides model checking and simulation of TLA+ specifications - Version ",
        )
    {
        TlcStartup::Started
    } else {
        TlcStartup::Unknown
    }
}

/// A supervisor fork is an attempt, not proof that the checker executable started.
pub(crate) fn spawned(command: &std::process::Command) -> io::Result<Option<String>> {
    if !command.get_args().any(|argument| argument == "tlc2.TLC") || directory()?.is_none() {
        return Ok(None);
    }
    let configuration = command
        .get_envs()
        .find_map(|(name, value)| (name == "UMPIRE_TLC_CONFIGURATION").then_some(value).flatten());
    let configuration = configuration.and_then(std::ffi::OsStr::to_str).unwrap_or_else(|| {
        if command.get_args().any(|argument| argument == "-h") { "probe" } else { "-" }
    });
    if !["-", "probe"].contains(&configuration)
        && (configuration.len() != 64
            || !configuration.bytes().all(|byte| byte.is_ascii_hexdigit()))
    {
        return Err(io::Error::other("invalid TLC configuration observation"));
    }
    let id = format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
    record("tlc-attempt", &format!("{id}\t{configuration}"))?;
    Ok(Some(id))
}

/// Confirm execution from the admitted checker's own startup output.
pub(crate) fn finished(id: Option<&str>, output: &str) -> io::Result<()> {
    if let Some(id) = id {
        let started = output.contains("TLC2 Version ")
            || output.contains(
                "TLC - provides model checking and simulation of TLA+ specifications - Version ",
            );
        record("tlc-finish", &format!("{id}\t{}", u8::from(started)))?;
    }
    Ok(())
}

/// Begin one actual test-run command; completion must use the same producer identity.
pub fn begin_tests() -> io::Result<String> {
    let id = format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
    record("tests-begin", &id)?;
    Ok(id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn admitted_checker_startup_requires_positive_captured_evidence() {
        for output in [
            b"TLC2 Version 2.19 of 08 August 2024\n".as_slice(),
            b"TLC - provides model checking and simulation of TLA+ specifications - Version 2.19\n",
            b"TLC2 Version 2.19\nEvaluating invariant A failed.\n",
        ] {
            assert_eq!(checker_startup(output), TlcStartup::Started);
        }
    }

    #[test]
    fn admitted_checker_absence_partial_or_non_utf8_stays_unknown() {
        for output in [
            b"".as_slice(),
            b"Error: Could not find or load main class tlc2.TLC\n",
            b"TLC2 Vers",
            b"TLC2 Version 2.19\n\xff",
        ] {
            assert_eq!(checker_startup(output), TlcStartup::Unknown);
        }
    }

    #[test]
    fn scope_probe_child() -> io::Result<()> {
        let Ok(mode) = env::var("WORK_SCOPE_PROBE") else { return Ok(()) };
        let mut command = std::process::Command::new("java");
        if mode == "tlc" {
            command.arg("tlc2.TLC");
        }
        let result = spawned(&command);
        if mode == "tlc" {
            assert!(result.is_err());
        } else {
            assert_eq!(result?, None);
        }
        Ok(())
    }

    #[test]
    fn unrelated_processes_do_not_validate_tlc_observation_scope() -> io::Result<()> {
        for mode in ["other", "tlc"] {
            let status = std::process::Command::new(env::current_exe()?)
                .args(["--exact", "work::tests::scope_probe_child", "--nocapture"])
                .env("WORK_SCOPE_PROBE", mode)
                .env("UMPIRE_GATE_RECEIPTS", "relative-invalid-scope")
                .env("UMPIRE_GATE_STAGE", "tests")
                .status()?;
            assert!(status.success());
        }
        Ok(())
    }
}

/// Immutable file identity checked on both sides of an owned invocation.
#[derive(Clone, Debug)]
struct Artifact {
    path: PathBuf,
    sha256: String,
}
impl Artifact {
    fn pin(path: &std::path::Path) -> io::Result<Self> {
        let path = path.canonicalize()?;
        let metadata = fs::symlink_metadata(&path)?;
        if !metadata.is_file() || metadata.is_symlink() {
            return Err(io::Error::other("accounting identity is not a regular file"));
        }
        Ok(Self { sha256: crate::bytes_sha256(&fs::read(&path)?), path })
    }
    fn verify(&self) -> io::Result<()> {
        let current = Self::pin(&self.path)?;
        if current.path != self.path || current.sha256 != self.sha256 {
            return Err(io::Error::other("accounting executable identity changed"));
        }
        Ok(())
    }
}

fn hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut text = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        text.push(char::from(HEX[usize::from(byte >> 4)]));
        text.push(char::from(HEX[usize::from(byte & 15)]));
    }
    text
}
fn path_hex(path: &std::path::Path) -> String {
    use std::os::unix::ffi::OsStrExt;
    hex(path.as_os_str().as_bytes())
}

#[derive(Clone, Debug)]
struct Scope {
    receipts: PathBuf,
    namespace: String,
    stage: String,
}
impl Scope {
    fn current() -> io::Result<Option<Self>> {
        if directory()?.is_none() {
            return Ok(None);
        }
        let receipts = PathBuf::from(
            env::var_os("UMPIRE_GATE_RECEIPTS")
                .ok_or_else(|| io::Error::other("missing work receipts"))?,
        )
        .canonicalize()?;
        Ok(Some(Self {
            receipts,
            namespace: env::var("UMPIRE_GATE_SCOPE").unwrap_or_default(),
            stage: env::var("UMPIRE_GATE_STAGE").map_err(io::Error::other)?,
        }))
    }
    fn identity(&self) -> String {
        crate::structural_sha256(&(path_hex(&self.receipts), &self.namespace, &self.stage))
    }
    fn authorities(&self) -> io::Result<PathBuf> {
        let path = self.receipts.join("authorities").join(&self.namespace).join(&self.stage);
        fs::create_dir_all(&path)?;
        if path.canonicalize()? != path {
            return Err(io::Error::other("escaping accounting authority directory"));
        }
        Ok(path)
    }
}

/// Private-renderer assertion identity shared by one protocol-control family.
/// The backend must call `complete` only after its original assertions succeed.
#[derive(Clone, Debug)]
pub struct FixtureAssertions {
    owner: PathBuf,
    test: String,
    executable: Artifact,
    scope: Option<Scope>,
    id: String,
}
impl FixtureAssertions {
    /// Bind a backend's closed fixture assertions to this test executable and owned scratch.
    pub fn pin(owner: &std::path::Path, test: &str) -> io::Result<Self> {
        let owner = owner.canonicalize()?;
        let executable = Artifact::pin(&env::current_exe()?)?;
        let scope = Scope::current()?;
        let id = crate::structural_sha256(&(
            path_hex(&owner),
            test,
            &executable.sha256,
            scope.as_ref().map(Scope::identity),
        ));
        Ok(Self { owner, test: test.into(), executable, scope, id })
    }
    /// Publish integrity-linked completion after the enclosing fixture assertions succeed.
    pub fn complete(&self) -> io::Result<()> {
        let Some(scope) = &self.scope else {
            return Ok(());
        };
        self.executable.verify()?;
        if Scope::current()?.as_ref().map(Scope::identity) != Some(scope.identity()) {
            return Err(io::Error::other("crossed fixture assertion scope"));
        }
        let index = self.owner.join(".fixture-attempts").join(&self.id);
        if !index.exists() {
            return Ok(());
        }
        let mut attempts = Vec::new();
        for entry in fs::read_dir(index)? {
            let path = entry?.path();
            let text = fs::read_to_string(&path)?;
            let fields: Vec<_> = text.trim_end().split('\t').collect();
            if fields.len() != 3
                || fields[0] != path.file_name().unwrap_or_default().to_string_lossy()
            {
                return Err(io::Error::other("malformed fixture attempt completion index"));
            }
            attempts.push(format!(
                "{{\"attempt\":\"{}\",\"path_hex\":\"{}\",\"sha256\":\"{}\"}}",
                fields[0], fields[1], fields[2]
            ));
        }
        attempts.sort();
        let bytes = format!(
            "{{\"version\":1,\"kind\":\"tlc-fixture-assertions\",\"assertion_id\":\"{}\",\"scope_sha256\":\"{}\",\"test_hex\":\"{}\",\"test_executable_sha256\":\"{}\",\"outcome\":\"assertions-completed\",\"attempts\":[{}]}}\n",
            self.id,
            scope.identity(),
            hex(self.test.as_bytes()),
            self.executable.sha256,
            attempts.join(",")
        );
        let path = scope.authorities()?.join(format!("assertions-{}.json", self.id));
        publish_authority(&path, bytes.as_bytes())?;
        record_v2(
            "tlc-fixture-complete",
            &format!("{}\t{}\t{}", self.id, path_hex(&path), crate::bytes_sha256(bytes.as_bytes())),
        )
    }
}

/// Identity supplied only by a backend-owned closed renderer, never by a process environment flag.
#[derive(Clone, Debug)]
pub struct ProtocolFixture {
    assertions: FixtureAssertions,
    script: Artifact,
    interpreter: Artifact,
    tools: Vec<Artifact>,
    family: String,
    renderer_sha256: String,
    case_hex: String,
    script_hex: String,
    data: Vec<PathBuf>,
}
impl ProtocolFixture {
    /// Pin a closed renderer's output and typed case. The caller owns renderer closure proof.
    pub fn pin(
        assertions: FixtureAssertions,
        script: &std::path::Path,
        family: &str,
        renderer_source: &[u8],
        typed_case: &[u8],
        data: &[PathBuf],
    ) -> io::Result<Self> {
        let script = Artifact::pin(script)?;
        let bytes = fs::read(&script.path)?;
        if !bytes.starts_with(b"#!/bin/bash\nset -eu\n")
            || !matches!(family, "tlc-batch-v1" | "tlc-cache-v1")
        {
            return Err(io::Error::other("unrecognized closed protocol renderer"));
        }
        if script.path.parent() != Some(assertions.owner.as_path()) {
            return Err(io::Error::other("fixture executable escapes assertion owner"));
        }
        Ok(Self {
            assertions,
            script_hex: hex(&bytes),
            script,
            data: data.to_vec(),
            interpreter: Artifact::pin(std::path::Path::new("/bin/bash"))?,
            tools: ["/usr/bin/dirname", "/usr/bin/cat", "/usr/bin/grep"]
                .into_iter()
                .map(|path| Artifact::pin(std::path::Path::new(path)))
                .collect::<io::Result<_>>()?,
            family: family.into(),
            renderer_sha256: crate::bytes_sha256(renderer_source),
            case_hex: hex(typed_case),
        })
    }
    fn verify(&self) -> io::Result<()> {
        self.assertions.executable.verify()?;
        self.script.verify()?;
        self.interpreter.verify()?;
        for tool in &self.tools {
            tool.verify()?;
        }
        Ok(())
    }
}

/// Invocation provenance supplied by the backend at its admitted executable sites.
#[derive(Clone, Debug)]
pub struct TlcInvocation {
    origin: Origin,
}
#[derive(Clone, Debug)]
enum Origin {
    Admitted { java: Artifact, jar: Artifact },
    Fixture(Box<ProtocolFixture>),
}
impl TlcInvocation {
    /// Bind the Java/JAR identities inside the backend's successful admission closure.
    pub fn admitted(java: &std::path::Path, jar: &std::path::Path) -> io::Result<Self> {
        Ok(Self {
            origin: Origin::Admitted { java: Artifact::pin(java)?, jar: Artifact::pin(jar)? },
        })
    }
    /// Carry the private backend renderer's pinned, positively evidenced protocol authority.
    pub fn fixture(fixture: ProtocolFixture) -> Self {
        Self { origin: Origin::Fixture(Box::new(fixture)) }
    }
    pub(crate) fn clean_command(
        &self,
        command: &std::process::Command,
    ) -> Option<std::process::Command> {
        if !matches!(self.origin, Origin::Fixture(_)) {
            return None;
        }
        let mut clean = std::process::Command::new(command.get_program());
        clean
            .args(command.get_args())
            .env("PATH", "/usr/bin:/bin")
            .env("LANG", "C")
            .env("LC_ALL", "C");
        if let Some(cwd) = command.get_current_dir() {
            clean.current_dir(cwd);
        }
        Some(clean)
    }
    fn verify(&self, command: &std::process::Command) -> io::Result<()> {
        let program = std::path::Path::new(command.get_program()).canonicalize()?;
        match &self.origin {
            Origin::Admitted { java, jar } => {
                java.verify()?;
                jar.verify()?;
                if program != java.path {
                    return Err(io::Error::other("crossed checker executable identity"));
                }
            }
            Origin::Fixture(fixture) => {
                fixture.verify()?;
                if program != fixture.script.path {
                    return Err(io::Error::other("crossed fixture executable identity"));
                }
            }
        }
        Ok(())
    }
    fn identity(&self) -> String {
        match &self.origin {
            Origin::Admitted { java, jar } => crate::structural_sha256(&(
                path_hex(&java.path),
                &java.sha256,
                path_hex(&jar.path),
                &jar.sha256,
            )),
            Origin::Fixture(fixture) => crate::structural_sha256(&(
                &fixture.assertions.id,
                &fixture.family,
                &fixture.renderer_sha256,
                &fixture.case_hex,
                &fixture.script.sha256,
                &fixture.interpreter.sha256,
            )),
        }
    }
}

pub(crate) struct TlcAttempt {
    id: String,
    invocation: TlcInvocation,
    scope: Scope,
    configuration: String,
    data: Vec<(Artifact, String)>,
}
impl TlcAttempt {
    pub(crate) fn begin(
        invocation: &TlcInvocation,
        command: &std::process::Command,
    ) -> io::Result<Option<Self>> {
        if !command.get_args().any(|arg| arg == "tlc2.TLC") {
            if let Err(error) = invocation.verify(command) {
                if Scope::current()?.is_some() {
                    let id =
                        format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
                    record_v2(
                        "tlc-attempt",
                        &format!("{id}\tfixture\t-\t{}", invocation.identity()),
                    )?;
                }
                return Err(error);
            }
            return Ok(None);
        }
        let Some(scope) = Scope::current()? else {
            invocation.verify(command)?;
            return Ok(None);
        };
        let configuration = configuration(command)?;
        let id = format!("{}-{}", std::process::id(), NEXT.fetch_add(1, Ordering::Relaxed));
        let origin =
            if matches!(invocation.origin, Origin::Fixture(_)) { "fixture" } else { "admitted" };
        record_v2(
            "tlc-attempt",
            &format!("{id}\t{origin}\t{configuration}\t{}", invocation.identity()),
        )?;
        // Once an attempt is published, identity corruption leaves it unmatched: accounting
        // must refuse even when a negative backend test expects the process refusal.
        invocation.verify(command)?;
        let data = if let Origin::Fixture(fixture) = &invocation.origin {
            fixture
                .data
                .iter()
                .map(|path| {
                    let artifact = Artifact::pin(path)?;
                    let bytes = fs::read(&artifact.path)?;
                    if bytes.len() > MAX_EVENT_BYTES {
                        return Err(io::Error::other("fixture input exceeds byte limit"));
                    }
                    Ok((artifact, hex(&bytes)))
                })
                .collect::<io::Result<_>>()?
        } else {
            Vec::new()
        };
        Ok(Some(Self { id, invocation: invocation.clone(), scope, configuration, data }))
    }
    pub(crate) fn finish(
        self,
        command: &std::process::Command,
        bytes: &[u8],
        cause: &str,
        status: Option<i32>,
    ) -> io::Result<()> {
        self.invocation.verify(command)?;
        if Scope::current()?.as_ref().map(Scope::identity) != Some(self.scope.identity()) {
            return Err(io::Error::other("crossed TLC invocation scope"));
        }
        if let Origin::Fixture(fixture) = &self.invocation.origin {
            if fixture.assertions.scope.as_ref().map(Scope::identity) != Some(self.scope.identity())
            {
                return Err(io::Error::other("crossed fixture observation scope"));
            }
            for (artifact, _) in &self.data {
                artifact.verify()?;
            }
            let data = self
                .data
                .iter()
                .map(|(artifact, bytes)| {
                    format!(
                        "{{\"path_hex\":\"{}\",\"sha256\":\"{}\",\"bytes_hex\":\"{}\"}}",
                        path_hex(&artifact.path),
                        artifact.sha256,
                        bytes
                    )
                })
                .collect::<Vec<_>>()
                .join(",");
            let tools = fixture
                .tools
                .iter()
                .map(|tool| {
                    format!(
                        "{{\"path_hex\":\"{}\",\"sha256\":\"{}\"}}",
                        path_hex(&tool.path),
                        tool.sha256
                    )
                })
                .collect::<Vec<_>>()
                .join(",");
            let document = format!(
                "{{\"version\":1,\"kind\":\"closed-tlc-protocol-fixture\",\"attempt\":\"{}\",\"assertion_id\":\"{}\",\"scope_sha256\":\"{}\",\"configuration\":\"{}\",\"family\":\"{}\",\"renderer_sha256\":\"{}\",\"case_hex\":\"{}\",\"script_path_hex\":\"{}\",\"script_sha256\":\"{}\",\"script_hex\":\"{}\",\"interpreter_path_hex\":\"{}\",\"interpreter_sha256\":\"{}\",\"tools\":[{}],\"data\":[{}],\"test_hex\":\"{}\",\"test_executable_path_hex\":\"{}\",\"test_executable_sha256\":\"{}\",\"environment_policy\":\"clean-path-c-v1\",\"cause\":\"{}\",\"exit\":{}}}\n",
                self.id,
                fixture.assertions.id,
                self.scope.identity(),
                self.configuration,
                fixture.family,
                fixture.renderer_sha256,
                fixture.case_hex,
                path_hex(&fixture.script.path),
                fixture.script.sha256,
                fixture.script_hex,
                path_hex(&fixture.interpreter.path),
                fixture.interpreter.sha256,
                tools,
                data,
                hex(fixture.assertions.test.as_bytes()),
                path_hex(&fixture.assertions.executable.path),
                fixture.assertions.executable.sha256,
                cause,
                status.map_or_else(|| "null".into(), |code| code.to_string())
            );
            let path = self.scope.authorities()?.join(format!("{}.json", self.id));
            publish_authority(&path, document.as_bytes())?;
            let sha = crate::bytes_sha256(document.as_bytes());
            let index =
                fixture.assertions.owner.join(".fixture-attempts").join(&fixture.assertions.id);
            fs::create_dir_all(&index)?;
            publish_authority(
                &index.join(&self.id),
                format!("{}\t{}\t{}\n", self.id, path_hex(&path), sha).as_bytes(),
            )?;
            record_v2(
                "tlc-finish",
                &format!("{}\tfixture\t{cause}\t{}\t{sha}", self.id, path_hex(&path)),
            )
        } else {
            let outcome =
                if checker_startup(bytes) == TlcStartup::Started { "started" } else { "unknown" };
            record_v2("tlc-finish", &format!("{}\t{outcome}\t{cause}\t-\t-", self.id))
        }
    }
}

fn configuration(command: &std::process::Command) -> io::Result<String> {
    let value = command
        .get_envs()
        .find_map(|(key, value)| (key == "UMPIRE_TLC_CONFIGURATION").then_some(value).flatten())
        .and_then(std::ffi::OsStr::to_str)
        .unwrap_or_else(|| if command.get_args().any(|arg| arg == "-h") { "probe" } else { "-" });
    if !["probe", "-"].contains(&value)
        && (value.len() != 64 || !value.bytes().all(|byte| byte.is_ascii_hexdigit()))
    {
        return Err(io::Error::other("invalid TLC configuration identity"));
    }
    Ok(value.into())
}
fn publish_authority(path: &std::path::Path, bytes: &[u8]) -> io::Result<()> {
    if bytes.len() > 4 * 1024 * 1024 {
        return Err(io::Error::other("fixture authority exceeds byte limit"));
    }
    let mut file = fs::OpenOptions::new().write(true).create_new(true).open(path)?;
    file.write_all(bytes)?;
    file.sync_all()
}
fn record_v2(kind: &str, values: &str) -> io::Result<()> {
    record_version(2, kind, values)
}
