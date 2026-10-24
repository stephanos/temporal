#![cfg_attr(
    not(test),
    allow(
        dead_code,
        reason = "Cycle endpoint integration belongs to successor batches"
    )
)]
use crate::cycle::process::{Command, Completed, Environment, Runner};
use rustix::fs::{Mode, OFlags, open, openat};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeMap,
    fs::{self, File, Metadata},
    io::Read,
    os::unix::fs::MetadataExt,
    path::{Component, Path, PathBuf},
    sync::atomic::{AtomicBool, AtomicU64, Ordering},
    time::Duration,
};

#[cfg(test)]
mod tests;

pub(crate) fn digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
pub(crate) fn strict_json(bytes: &[u8]) -> Result<Value, String> {
    crate::qualify::spec::strict(bytes)
}
pub(crate) fn valid_digest(text: &str) -> bool {
    text.len() == 64
        && text
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
}
fn identity(meta: &Metadata) -> (u64, u64, u64, i64, i64, i64, i64) {
    (
        meta.dev(),
        meta.ino(),
        meta.len(),
        meta.mtime(),
        meta.mtime_nsec(),
        meta.ctime(),
        meta.ctime_nsec(),
    )
}
pub(crate) fn clean_absolute(path: &Path) -> Result<(), String> {
    let text = path.to_str().ok_or("non-UTF8 custody path")?;
    if !path.is_absolute()
        || text.contains(['\0', '\n', '\r'])
        || text.contains("//")
        || text.contains("/./")
        || text.ends_with("/.")
        || path
            .components()
            .any(|p| matches!(p, Component::ParentDir | Component::CurDir))
    {
        return Err("absolute clean custody path required".into());
    }
    Ok(())
}
pub(crate) fn read_file(path: &Path, maximum: u64) -> Result<Vec<u8>, String> {
    read_file_observed(path, maximum).map(|(bytes, _)| bytes)
}
pub(crate) fn read_file_observed(path: &Path, maximum: u64) -> Result<(Vec<u8>, Metadata), String> {
    clean_absolute(path)?;
    let components = path
        .components()
        .filter_map(|c| {
            if let Component::Normal(name) = c {
                Some(name)
            } else {
                None
            }
        })
        .collect::<Vec<_>>();
    let (name, parents) = components
        .split_last()
        .ok_or("regular custody file required")?;
    let flags = OFlags::RDONLY | OFlags::DIRECTORY | OFlags::NOFOLLOW | OFlags::CLOEXEC;
    let mut dirs = vec![(
        PathBuf::from("/"),
        File::from(open("/", flags, Mode::empty()).map_err(|e| e.to_string())?),
    )];
    for part in parents {
        let (parent, handle) = dirs.last().ok_or("custody directory absent")?;
        let next = File::from(
            openat(handle, *part, flags, Mode::empty())
                .map_err(|e| format!("custody ancestor: {e}"))?,
        );
        dirs.push((parent.join(part), next));
    }
    let mut file = File::from(
        openat(
            &dirs.last().ok_or("custody directory absent")?.1,
            *name,
            OFlags::RDONLY | OFlags::NONBLOCK | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::empty(),
        )
        .map_err(|e| format!("custody file: {e}"))?,
    );
    let before = file.metadata().map_err(|e| e.to_string())?;
    if !before.is_file() {
        return Err("regular nonsymlink custody file required".into());
    }
    if before.len() > maximum {
        return Err("custody file byte bound exceeded".into());
    }
    let mut bytes = Vec::new();
    (&mut file)
        .take(maximum.saturating_add(1))
        .read_to_end(&mut bytes)
        .map_err(|e| e.to_string())?;
    if u64::try_from(bytes.len()).map_err(|e| e.to_string())? > maximum {
        return Err("custody file byte bound exceeded".into());
    }
    let after = file.metadata().map_err(|e| e.to_string())?;
    let named = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    if !named.is_file()
        || identity(&before) != identity(&after)
        || identity(&after) != identity(&named)
        || after.len() != bytes.len() as u64
    {
        return Err("custody file changed during read".into());
    }
    for (name, handle) in dirs {
        let actual = handle.metadata().map_err(|e| e.to_string())?;
        let named = fs::symlink_metadata(name).map_err(|e| e.to_string())?;
        if !named.is_dir() || actual.dev() != named.dev() || actual.ino() != named.ino() {
            return Err("custody ancestor changed during read".into());
        }
    }
    Ok((bytes, after))
}
pub(crate) fn check_files(files: &BTreeMap<String, String>) -> Result<(), String> {
    if files.is_empty() {
        return Err("empty frozen input set".into());
    }
    if files.len() > 50_000 {
        return Err("frozen input count bound".into());
    }
    let mut total = 0u64;
    for (name, expected) in files {
        if !valid_digest(expected) {
            return Err("malformed frozen input digest".into());
        }
        let bytes = read_file(Path::new(name), 128 << 20)?;
        total = total
            .checked_add(bytes.len() as u64)
            .ok_or("frozen input byte overflow")?;
        if total > 512 << 20 {
            return Err("frozen input aggregate byte bound".into());
        }
        if digest(&bytes) != *expected {
            return Err(format!("frozen input changed: {name}"));
        }
    }
    Ok(())
}

pub(crate) struct Capture {
    pub completed: Completed,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
}
static COMMAND_SEQUENCE: AtomicU64 = AtomicU64::new(0);
pub(crate) fn observe_command(
    runner: &mut Runner,
    context: &str,
    cwd: &Path,
    argv: Vec<String>,
    environment: BTreeMap<String, String>,
    timeout: Duration,
    cancelled: &AtomicBool,
) -> Result<Capture, String> {
    let name = format!(
        "custody-{}",
        COMMAND_SEQUENCE.fetch_add(1, Ordering::Relaxed)
    );
    let command = Command {
        argv,
        cwd: cwd.to_owned(),
        environment: Environment::Exact(environment.clone()),
        declared_environment: environment,
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout,
        private_context: Some(context.into()),
    };
    let running = runner
        .start(&name, command, cancelled, |_| Ok(()))
        .map_err(|e| e.to_string())?;
    let mut stdout = Vec::new();
    let mut stderr = Vec::new();
    let completed = running.wait_observed(cancelled, |_, _, streams| {
        stdout = read_file(&streams.stdout, 32 << 20)?;
        stderr = read_file(&streams.stderr, 32 << 20)?;
        if digest(&stdout) != streams.stdout_sha256 || digest(&stderr) != streams.stderr_sha256 {
            return Err("command raw bytes changed".into());
        }
        Ok(None)
    })?;
    Ok(Capture {
        completed,
        stdout,
        stderr,
    })
}
fn git(
    runner: &mut Runner,
    context: &str,
    repo: &Path,
    arguments: &[&str],
    cancelled: &AtomicBool,
) -> Result<Vec<u8>, String> {
    let mut argv = vec![
        "/usr/bin/git".into(),
        "-C".into(),
        repo.to_str().ok_or("non-UTF8 checkout")?.into(),
    ];
    argv.extend(arguments.iter().map(|s| (*s).to_owned()));
    let environment = BTreeMap::from([
        ("PATH".into(), "/usr/bin:/bin".into()),
        ("LC_ALL".into(), "C".into()),
        ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
        ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
        ("GIT_CONFIG_SYSTEM".into(), "/dev/null".into()),
        ("GIT_OPTIONAL_LOCKS".into(), "0".into()),
        ("GIT_NO_REPLACE_OBJECTS".into(), "1".into()),
    ]);
    let result = observe_command(
        runner,
        context,
        repo,
        argv,
        environment,
        Duration::from_secs(30),
        cancelled,
    )?;
    if !result.completed.accepted() {
        return Err(format!(
            "source Git command refused: exit {} {:?}",
            result.completed.exit, result.completed.failure
        ));
    }
    Ok(result.stdout)
}
fn head_symlinks(
    repo: &Path,
    runner: &mut Runner,
    head: &str,
    cancelled: &AtomicBool,
    context: &str,
) -> Result<BTreeMap<String, String>, String> {
    let tree = git(
        runner,
        context,
        repo,
        &["ls-tree", "-r", "-z", head],
        cancelled,
    )?;
    let mut committed_links = BTreeMap::new();
    for row in tree.split(|b| *b == 0).filter(|r| !r.is_empty()) {
        let (meta, name) = row.split_at(
            row.iter()
                .position(|b| *b == b'\t')
                .ok_or("invalid Git tree row")?,
        );
        let meta = std::str::from_utf8(meta)
            .map_err(|e| e.to_string())?
            .split(' ')
            .collect::<Vec<_>>();
        let name = std::str::from_utf8(&name[1..]).map_err(|e| e.to_string())?;
        if meta.first() == Some(&"120000") && !name.starts_with(".flow/") {
            if meta.len() != 3 || meta[1] != "blob" {
                return Err("tracked symlink is not blob".into());
            }
            let original = git(
                runner,
                context,
                repo,
                &["cat-file", "blob", meta[2]],
                cancelled,
            )?;
            let literal = String::from_utf8(original).map_err(|e| e.to_string())?;
            let hash = observe_command(
                runner,
                context,
                repo,
                vec![
                    "/bin/sh".into(),
                    "-c".into(),
                    "printf %s \"$1\" | /usr/bin/git hash-object --stdin".into(),
                    "symlink-blob".into(),
                    literal.clone(),
                ],
                BTreeMap::from([
                    ("PATH".into(), "/usr/bin:/bin".into()),
                    ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
                    ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
                    ("GIT_NO_REPLACE_OBJECTS".into(), "1".into()),
                ]),
                Duration::from_secs(30),
                cancelled,
            )?;
            if !hash.completed.accepted() || hash.stdout != format!("{}\n", meta[2]).as_bytes() {
                return Err("Git symlink blob object identity mismatch".into());
            }
            committed_links.insert(name.to_owned(), literal);
        }
    }
    Ok(committed_links)
}
fn git_index(
    runner: &mut Runner,
    context: &str,
    repo: &Path,
    cancelled: &AtomicBool,
) -> Result<Vec<u8>, String> {
    git(
        runner,
        context,
        repo,
        &["ls-files", "--stage", "-z"],
        cancelled,
    )
}
fn index_gitlinks(indexed: &[u8]) -> Result<std::collections::BTreeSet<&str>, String> {
    let mut gitlinks = std::collections::BTreeSet::new();
    for entry in indexed.split(|b| *b == 0).filter(|entry| !entry.is_empty()) {
        let entry = std::str::from_utf8(entry).map_err(|e| e.to_string())?;
        let (header, name) = entry.split_once('\t').ok_or("invalid Git index entry")?;
        if header.starts_with("160000 ") && header.ends_with(" 0") {
            gitlinks.insert(name);
        }
    }
    Ok(gitlinks)
}
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Snapshot {
    pub head: String,
    pub files: BTreeMap<String, String>,
    pub symlinks: BTreeMap<String, String>,
}
impl Snapshot {
    pub(crate) fn observe(
        repo: &Path,
        runner: &mut Runner,
        cancelled: &AtomicBool,
        context: &str,
    ) -> Result<Self, String> {
        clean_absolute(repo)?;
        if repo.canonicalize().map_err(|e| e.to_string())? != repo {
            return Err("noncanonical candidate checkout".into());
        }
        let head = String::from_utf8(git(
            runner,
            context,
            repo,
            &["rev-parse", "HEAD"],
            cancelled,
        )?)
        .map_err(|e| e.to_string())?
        .trim()
        .to_owned();
        if head.len() != 40
            || !head
                .bytes()
                .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
        {
            return Err("invalid actual Git HEAD".into());
        }
        let tracked = git(runner, context, repo, &["ls-files", "-z"], cancelled)?;
        let indexed = git_index(runner, context, repo, cancelled)?;
        let gitlinks = index_gitlinks(&indexed)?;
        let committed_links = head_symlinks(repo, runner, &head, cancelled, context)?;
        let mut files = BTreeMap::new();
        let mut symlinks = BTreeMap::new();
        let mut total = 0u64;
        let mut count = 0usize;
        for raw in tracked.split(|b| *b == 0).filter(|r| !r.is_empty()) {
            let name = std::str::from_utf8(raw).map_err(|e| e.to_string())?;
            if name.starts_with(".flow/") {
                continue;
            }
            let path = repo.join(name);
            clean_absolute(&path)?;
            count += 1;
            if count > 50_000 {
                return Err("source tracked inventory count bound".into());
            }
            let metadata =
                fs::symlink_metadata(&path).map_err(|e| format!("tracked source absent: {e}"))?;
            if gitlinks.contains(name) {
                if !metadata.is_dir() || metadata.file_type().is_symlink() {
                    return Err("tracked Git gitlink is not a directory".into());
                }
                continue;
            }
            if metadata.file_type().is_symlink() {
                let literal = fs::read_link(&path)
                    .map_err(|e| e.to_string())?
                    .to_str()
                    .ok_or("non-UTF8 symlink literal")?
                    .to_owned();
                if committed_links.get(name) != Some(&literal) {
                    return Err("tracked symlink literal differs from actual Git HEAD blob".into());
                }
                symlinks.insert(path.to_str().ok_or("non-UTF8 source")?.into(), literal);
            } else if metadata.is_file() {
                let bytes = read_file(&path, 128 << 20)?;
                total += bytes.len() as u64;
                if total > 512 << 20 {
                    return Err("source inventory byte bound".into());
                }
                files.insert(
                    path.to_str().ok_or("non-UTF8 source")?.into(),
                    digest(&bytes),
                );
            } else {
                return Err("tracked source must be regular or symlink".into());
            }
        }
        if files.is_empty() {
            return Err("empty actual regular tracked source inventory".into());
        }
        let expected_links = committed_links
            .keys()
            .map(|n| repo.join(n).to_string_lossy().into_owned())
            .collect::<std::collections::BTreeSet<_>>();
        if expected_links != symlinks.keys().cloned().collect() {
            return Err("tracked symlink inventory differs from HEAD".into());
        }
        if git(runner, context, repo, &["rev-parse", "HEAD"], cancelled)?
            != format!("{head}\n").as_bytes()
            || git(runner, context, repo, &["ls-files", "-z"], cancelled)? != tracked
            || git_index(runner, context, repo, cancelled)? != indexed
        {
            return Err("source Git state changed during observation".into());
        }
        Ok(Self {
            head,
            files,
            symlinks,
        })
    }
    pub(crate) fn verify(
        &self,
        repo: &Path,
        runner: &mut Runner,
        cancelled: &AtomicBool,
        context: &str,
    ) -> Result<(), String> {
        let actual = Self::observe(repo, runner, cancelled, context)?;
        if actual != *self {
            return Err("current frozen source/HEAD/symlinks changed".into());
        }
        Ok(())
    }
    pub(crate) fn manifest(&self) -> Vec<u8> {
        let mut text = String::new();
        for (path, sha) in &self.files {
            text.push_str(sha);
            text.push_str("  ");
            text.push_str(path);
            text.push('\n');
        }
        text.into_bytes()
    }
}

pub(crate) struct Frozen {
    repo: PathBuf,
    config_path: PathBuf,
    config_sha256: String,
    files: BTreeMap<String, String>,
    head: String,
    symlinks: BTreeMap<String, String>,
    source_manifest: PathBuf,
    source_sha256: String,
    required: Vec<PathBuf>,
}
impl Frozen {
    pub(crate) fn from_config(config: &crate::cycle::config::Configured) -> Result<Self, String> {
        let value = config.value();
        let repo = PathBuf::from(value["repo"].as_str().ok_or("missing candidate checkout")?);
        let required = [
            "parallel/scheduler.py",
            "parallel/dag.tsv",
            "parallel/lib.sh",
            "q1-serial/run-qualification.sh",
            "parallel/required-roots.py",
        ]
        .iter()
        .map(|name| {
            repo.join(".flow/artifacts/wasm-qualification-driver")
                .join(name)
        })
        .collect();
        Ok(Self {
            repo,
            config_path: config.path().to_owned(),
            config_sha256: config.sha256().to_owned(),
            files: serde_json::from_value(value["files"].clone()).map_err(|e| e.to_string())?,
            head: value["head"].as_str().ok_or("missing frozen HEAD")?.into(),
            symlinks: serde_json::from_value(value["symlinks"].clone())
                .map_err(|e| e.to_string())?,
            source_manifest: PathBuf::from(
                value["source_manifest"]
                    .as_str()
                    .ok_or("missing frozen source manifest")?,
            ),
            source_sha256: value["source_sha256"]
                .as_str()
                .ok_or("missing source manifest digest")?
                .into(),
            required,
        })
    }
    pub(crate) fn guard(
        &self,
        runner: &mut Runner,
        cancelled: &AtomicBool,
        context: &str,
    ) -> Result<(), String> {
        if digest(&read_file(&self.config_path, 16 << 20)?) != self.config_sha256 {
            return Err("driver config changed".into());
        }
        check_files(&self.files)?;
        let actual = Snapshot::observe(&self.repo, runner, cancelled, context)?;
        if actual.head != self.head {
            return Err("frozen HEAD changed".into());
        }
        if !actual.files.keys().all(|p| self.files.contains_key(p)) {
            return Err("incomplete non-Flow source inventory".into());
        }
        if !actual
            .files
            .iter()
            .all(|(path, sha)| self.files.get(path) == Some(sha))
        {
            return Err("frozen source bytes changed during guard".into());
        }
        if !self
            .required
            .iter()
            .all(|p| self.files.contains_key(&p.to_string_lossy().into_owned()))
        {
            return Err("unsealed driver/reference input".into());
        }
        for (path, target) in &self.symlinks {
            let path = Path::new(path);
            clean_absolute(path)?;
            if !fs::symlink_metadata(path)
                .map_err(|e| e.to_string())?
                .file_type()
                .is_symlink()
                || fs::read_link(path).map_err(|e| e.to_string())?.to_str() != Some(target)
            {
                return Err("frozen symlink changed".into());
            }
        }
        let tracked = git(runner, context, &self.repo, &["ls-files", "-z"], cancelled)?;
        let mut all_links = BTreeMap::new();
        for row in tracked.split(|b| *b == 0).filter(|r| !r.is_empty()) {
            let path = self
                .repo
                .join(std::str::from_utf8(row).map_err(|e| e.to_string())?);
            let metadata = match fs::symlink_metadata(&path) {
                Ok(metadata) => metadata,
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
                Err(error) => return Err(error.to_string()),
            };
            if metadata.file_type().is_symlink() {
                all_links.insert(
                    path.to_string_lossy().into_owned(),
                    fs::read_link(path)
                        .map_err(|e| e.to_string())?
                        .to_string_lossy()
                        .into_owned(),
                );
            }
        }
        if all_links != self.symlinks {
            return Err("frozen source symlink inventory changed".into());
        }
        let status = git(
            runner,
            context,
            &self.repo,
            &["status", "--porcelain", "--untracked-files=all", "-z"],
            cancelled,
        )?;
        let intervals = runner.active_intervals()?;
        let mut rows = status.split(|b| *b == 0).filter(|r| !r.is_empty());
        while let Some(row) = rows.next() {
            if row.len() < 4 {
                return Err("malformed frozen source status".into());
            }
            let name = std::str::from_utf8(&row[3..]).map_err(|e| e.to_string())?;
            if row[..2].contains(&b'R') || row[..2].contains(&b'C') {
                let original = rows.next().ok_or("missing Git rename source")?;
                if !name.starts_with(".flow/") || !original.starts_with(b".flow/") {
                    return Err("dirty frozen source rename".into());
                }
                continue;
            }
            if name.starts_with(".flow/") {
                continue;
            }
            if !row.starts_with(b"?? ")
                || !intervals.iter().any(|interval| {
                    interval
                        .scratch()
                        .iter()
                        .any(|pattern| glob(pattern, name) || glob(&format!("{pattern}/*"), name))
                })
            {
                return Err(format!(
                    "dirty frozen source: {}",
                    String::from_utf8_lossy(row)
                ));
            }
        }
        // A completed or degraded interval cannot authorize the status just read.
        let current = runner.active_intervals()?;
        if intervals.len() != current.len()
            || intervals.iter().zip(current.iter()).any(|(a, b)| {
                a.pid() != b.pid() || a.started_monotonic_nanos() != b.started_monotonic_nanos()
            })
        {
            return Err("active scratch interval changed during source guard".into());
        }
        let name = self
            .source_manifest
            .to_str()
            .ok_or("non-UTF8 source manifest")?;
        if self.files.get(name) != Some(&self.source_sha256)
            || digest(&read_file(&self.source_manifest, 16 << 20)?) != self.source_sha256
        {
            return Err("frozen source manifest changed or unsealed".into());
        }
        Ok(())
    }
}
fn glob(pattern: &str, value: &str) -> bool {
    fn matches(p: &[u8], v: &[u8], memo: &mut BTreeMap<(usize, usize), bool>) -> bool {
        let key = (p.len(), v.len());
        if let Some(result) = memo.get(&key) {
            return *result;
        }
        let result = match p.first() {
            None => v.is_empty(),
            Some(b'*') => matches(&p[1..], v, memo) || (!v.is_empty() && matches(p, &v[1..], memo)),
            Some(b'?') => !v.is_empty() && matches(&p[1..], &v[1..], memo),
            Some(b'[') => {
                if let Some(end) = p.iter().position(|b| *b == b']') {
                    let mut chars = &p[1..end];
                    let negate = chars.first() == Some(&b'!');
                    if negate {
                        chars = &chars[1..];
                    }
                    let hit = v.first().is_some_and(|b| {
                        let mut found = false;
                        let mut i = 0;
                        while i < chars.len() {
                            if i + 2 < chars.len() && chars[i + 1] == b'-' {
                                found |= chars[i] <= *b && *b <= chars[i + 2];
                                i += 3;
                            } else {
                                found |= chars[i] == *b;
                                i += 1;
                            }
                        }
                        found != negate
                    });
                    hit && matches(&p[end + 1..], &v[1..], memo)
                } else {
                    v.first() == Some(&b'[') && matches(&p[1..], &v[1..], memo)
                }
            }
            Some(c) => v.first() == Some(c) && matches(&p[1..], &v[1..], memo),
        };
        memo.insert(key, result);
        result
    }
    matches(pattern.as_bytes(), value.as_bytes(), &mut BTreeMap::new())
}
#[cfg(test)]
pub(crate) fn run_cases(owner: &crate::cycle::process::CycleOwner) -> Result<(), String> {
    tests::run_cases(owner)
}
