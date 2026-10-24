use super::{canonical_path, clean, digest, regular, require, strict_json};
use crate::evidence::{canonical, store::open};
use rustix::fs::{self as descriptor, AtFlags, FlockOperation, Mode, OFlags};
use serde::{Deserialize, Serialize};
use std::{
    collections::{BTreeMap, BTreeSet},
    fs::{self, File},
    io::Write,
    os::unix::fs::MetadataExt as _,
    path::{Path, PathBuf},
};

pub(crate) const MAX_FILES: usize = 50_000;
pub(crate) const MAX_BYTES: u64 = 512 << 20;
const EXCLUDED: &[&str] = &[
    ".flow",
    ".git",
    ".tmp",
    ".turbo",
    ".worktrees",
    ".bin",
    ".stamp",
    ".gomad",
    ".toolchain",
    "dist",
    "evidence",
    "output",
    "outputs",
    "build",
    "target",
    "tools/gomad/rust/target",
    "tools/gomad/scripts/__pycache__",
];
const BASE: &str = "tools/gomad/internal/backends/wasm/testdata";
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct SourceFile {
    pub original: String,
    pub sha256: String,
    pub bytes: usize,
}
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct SourceRecord {
    pub schema: String,
    pub root: PathBuf,
    pub repo: PathBuf,
    pub source_manifest: PathBuf,
    pub source_manifest_sha256: String,
    pub workload: String,
    pub source: String,
    pub working_dir: PathBuf,
    pub files: BTreeMap<String, SourceFile>,
    pub original_wrappers: BTreeMap<String, String>,
    pub copied_files: usize,
    pub copied_bytes: u64,
    pub max_files: usize,
    pub max_bytes: u64,
}
type Copied = BTreeMap<PathBuf, (String, Vec<u8>)>;
fn private(path: &Path) -> Result<(), String> {
    canonical_path(path)?;
    let info = fs::symlink_metadata(path).map_err(|e| e.to_string())?;
    require(
        info.is_dir() && info.mode() & 0o7777 == 0o700,
        "derived directory mode changed",
    )?;
    require(
        info.uid() == rustix::process::geteuid().as_raw(),
        "derived directory owner changed",
    )
}
#[expect(
    clippy::too_many_lines,
    reason = "The byte exact source recipe retains the complete frozen derivation in one authority"
)]
pub(crate) fn recipe(
    root: &Path,
    repo: &Path,
    pins: &BTreeMap<String, String>,
    workload: &str,
    manifest: &Path,
) -> Result<(SourceRecord, Copied), String> {
    private(root)?;
    canonical_path(repo)?;
    require(
        !root.starts_with(repo) && !repo.starts_with(root),
        "source and producer roots overlap",
    )?;
    let raw = regular(manifest, MAX_BYTES, None)?;
    require(
        pins.get(manifest.to_str().ok_or("manifest path UTF8")?) == Some(&digest(&raw)),
        "unsealed source manifest",
    )?;
    let text = std::str::from_utf8(&raw).map_err(|e| e.to_string())?;
    let mut entries = BTreeMap::new();
    for line in text.lines() {
        let (pin, name) = line
            .split_once("  ")
            .ok_or("invalid frozen source manifest")?;
        require(
            canonical::valid_digest(pin) && name.starts_with('/'),
            "invalid frozen source manifest",
        )?;
        let original = Path::new(name);
        require(
            clean(original) && original.starts_with(repo) && original != repo,
            "foreign or unclean frozen source",
        )?;
        require(
            entries
                .insert(
                    name.to_owned(),
                    (
                        original
                            .strip_prefix(repo)
                            .map_err(|e| e.to_string())?
                            .to_path_buf(),
                        pin.to_owned(),
                    ),
                )
                .is_none(),
            "duplicate frozen source",
        )?;
    }
    require(!entries.is_empty(), "empty frozen source manifest")?;
    let mut copied = Copied::new();
    let mut copied_bytes = 0u64;
    for (name, (relative, pin)) in entries {
        if EXCLUDED.iter().any(|prefix| relative.starts_with(prefix)) {
            continue;
        }
        copied_file_bound(copied.len(), false)?;
        let body = regular(Path::new(&name), MAX_BYTES, None)?;
        copied_bytes = copied_bytes
            .checked_add(body.len() as u64)
            .ok_or("copied byte bound exceeded")?;
        copied_byte_bound(copied_bytes)?;
        require(
            pins.get(&name) == Some(&pin) && digest(&body) == pin,
            "original source changed or unsealed",
        )?;
        copied.insert(relative, (name, body));
    }
    require(
        copied.contains_key(Path::new("go.mod")) && copied.contains_key(Path::new("go.sum")),
        "original module files missing",
    )?;
    let module = &copied[Path::new("go.mod")].1;
    require(
        !module.split_inclusive(|b| *b == b'\n').any(|line| {
            let trimmed = line.trim_ascii_start();
            trimmed.starts_with(b"replace") && trimmed.get(7).is_some_and(u8::is_ascii_whitespace)
        }),
        "ambient module replacement refused",
    )?;
    let names: Vec<PathBuf> = match workload {
        "recovery" => vec![Path::new(BASE).join("temporal_storage/recovery_test.go")],
        "replication" => vec![
            Path::new(BASE).join("temporal_storage/recovery_test.go"),
            Path::new(BASE).join("temporal_broker/replication_test.go"),
        ],
        "retry" => copied
            .keys()
            .filter(|path| {
                path.parent() == Some(Path::new(BASE).join("network_streams").as_path())
                    && path.extension().is_some_and(|v| v == "go")
            })
            .cloned()
            .collect(),
        _ => return Err("unknown or empty wrapper workload".into()),
    };
    require(!names.is_empty(), "unknown or empty wrapper workload")?;
    require(
        names.iter().all(|name| copied.contains_key(name)),
        "wrapper missing from frozen module",
    )?;
    let keys = if workload == "retry" {
        vec![Path::new(BASE).join("network_streams/temporal_retry_test.go")]
    } else {
        names.clone()
    };
    require(
        keys.iter().all(|name| copied.contains_key(name)),
        "wrapper key missing",
    )?;
    let key_bytes: Vec<u8> = keys
        .iter()
        .flat_map(|name| copied[name].1.iter().copied())
        .collect();
    let relative_stage = format!(
        ".tmp/fn232-temporal-{workload}-{}",
        &digest(&key_bytes)[..16]
    );
    let original_wrappers = add_wrappers(&mut copied, names, &relative_stage)?;
    copied_file_bound(copied.len(), true)?;
    let size = copied.values().try_fold(0u64, |n, (_, body)| {
        n.checked_add(body.len() as u64)
            .ok_or("copied byte bound exceeded")
    })?;
    copied_byte_bound(size)?;
    let working_dir = root.join("source-module");
    let files = copied
        .iter()
        .map(|(relative, (original, body))| {
            Ok((
                working_dir
                    .join(relative)
                    .to_str()
                    .ok_or("source path UTF8")?
                    .to_owned(),
                SourceFile {
                    original: original.clone(),
                    sha256: digest(body),
                    bytes: body.len(),
                },
            ))
        })
        .collect::<Result<_, String>>()?;
    Ok((
        SourceRecord {
            schema: "gomad.source-custody/v1".into(),
            root: root.into(),
            repo: repo.into(),
            source_manifest: manifest.into(),
            source_manifest_sha256: digest(&raw),
            workload: workload.into(),
            source: format!("./{relative_stage}"),
            working_dir,
            files,
            original_wrappers,
            copied_files: copied.len(),
            copied_bytes: size,
            max_files: MAX_FILES,
            max_bytes: MAX_BYTES,
        },
        copied,
    ))
}
fn copied_file_bound(count: usize, complete: bool) -> Result<(), String> {
    require(
        if complete {
            count <= MAX_FILES
        } else {
            count < MAX_FILES
        },
        "copied file bound exceeded",
    )
}
fn copied_byte_bound(bytes: u64) -> Result<(), String> {
    require(bytes <= MAX_BYTES, "copied byte bound exceeded")
}
fn add_wrappers(
    copied: &mut Copied,
    names: Vec<PathBuf>,
    relative_stage: &str,
) -> Result<BTreeMap<String, String>, String> {
    let mut original_wrappers = BTreeMap::new();
    for name in names {
        let (original, body) = copied[&name].clone();
        let relative = Path::new(relative_stage).join(name.file_name().ok_or("wrapper filename")?);
        require(!copied.contains_key(&relative), "wrapper collision")?;
        original_wrappers.insert(original.clone(), digest(&body));
        copied.insert(relative, (original, body));
    }
    Ok(original_wrappers)
}
pub(crate) fn inspect(root: &Path, record: &SourceRecord) -> Result<(), String> {
    let module = root.join("source-module");
    private(&module)?;
    let expected_files: BTreeSet<PathBuf> = record.files.keys().map(PathBuf::from).collect();
    let mut expected_dirs = BTreeSet::from([module.clone()]);
    for name in &expected_files {
        require(
            name.starts_with(&module) && clean(name),
            "foreign derived source",
        )?;
        expected_dirs.extend(
            name.ancestors()
                .skip(1)
                .take_while(|p| p.starts_with(&module))
                .map(Path::to_path_buf),
        );
    }
    let mut actual_files = BTreeSet::new();
    let mut actual_dirs = BTreeSet::from([module.clone()]);
    let mut pending = vec![module];
    while let Some(directory) = pending.pop() {
        private(&directory)?;
        let before = fs::symlink_metadata(&directory).map_err(|e| e.to_string())?;
        for child in fs::read_dir(&directory).map_err(|e| e.to_string())? {
            let path = child.map_err(|e| e.to_string())?.path();
            let info = fs::symlink_metadata(&path).map_err(|e| e.to_string())?;
            require(!info.file_type().is_symlink(), "derived symlink")?;
            if info.is_dir() {
                require(
                    expected_dirs.contains(&path),
                    "unexpected derived directory",
                )?;
                private(&path)?;
                actual_dirs.insert(path.clone());
                pending.push(path);
            } else {
                require(expected_files.contains(&path), "unexpected derived source")?;
                let body = regular(&path, MAX_BYTES, Some(0o600))?;
                require(
                    digest(&body) == record.files[path.to_str().ok_or("source path UTF8")?].sha256,
                    "derived source changed",
                )?;
                actual_files.insert(path);
            }
        }
        let after = fs::symlink_metadata(&directory).map_err(|e| e.to_string())?;
        require(
            (
                before.dev(),
                before.ino(),
                before.mode(),
                before.uid(),
                before.mtime(),
                before.mtime_nsec(),
                before.ctime(),
                before.ctime_nsec(),
            ) == (
                after.dev(),
                after.ino(),
                after.mode(),
                after.uid(),
                after.mtime(),
                after.mtime_nsec(),
                after.ctime(),
                after.ctime_nsec(),
            ),
            "derived directory changed during inspection",
        )?;
    }
    require(
        actual_files == expected_files && actual_dirs == expected_dirs,
        "derived inventory changed",
    )
}
fn verify_unlocked(
    root: &Path,
    repo: &Path,
    pins: &BTreeMap<String, String>,
    workload: &str,
    target_source: &str,
    target_working_dir: &Path,
    frozen: Option<&Path>,
) -> Result<SourceRecord, String> {
    let raw = regular(&root.join("source-custody.json"), MAX_BYTES, Some(0o600))?;
    let value = strict_json(&raw)?;
    require(
        value.is_object() && value["source_manifest"].is_string(),
        "invalid custody receipt",
    )?;
    let observed: SourceRecord = serde_json::from_value(value).map_err(|e| e.to_string())?;
    if let Some(frozen) = frozen {
        canonical_path(frozen)?;
        require(
            frozen == observed.source_manifest,
            "foreign source manifest",
        )?;
    }
    let (expected, _) = recipe(root, repo, pins, workload, &observed.source_manifest)?;
    let mut encoded = canonical::encode(&expected)?;
    encoded.push(b'\n');
    require(
        observed == expected && raw == encoded,
        "custody recipe changed",
    )?;
    require(
        target_source == expected.source && target_working_dir == expected.working_dir,
        "target custody differs",
    )?;
    inspect(root, &expected)?;
    Ok(expected)
}
pub(crate) fn verify(
    root: &Path,
    repo: &Path,
    pins: &BTreeMap<String, String>,
    workload: &str,
    target_source: &str,
    target_working_dir: &Path,
    frozen: Option<&Path>,
) -> Result<BTreeMap<String, String>, String> {
    private(root)?;
    let lock = open::pin(root)?;
    descriptor::flock(&lock, FlockOperation::LockShared).map_err(|e| e.to_string())?;
    let expected = verify_unlocked(
        root,
        repo,
        pins,
        workload,
        target_source,
        target_working_dir,
        frozen,
    )?;
    Ok(expected
        .files
        .into_iter()
        .map(|(path, file)| (path, file.sha256))
        .collect())
}
fn publish(root: &File, name: &str, bytes: &[u8]) -> Result<(), String> {
    let mut file = File::from(
        descriptor::openat(
            root,
            name,
            OFlags::WRONLY | OFlags::CREATE | OFlags::EXCL | OFlags::NOFOLLOW | OFlags::CLOEXEC,
            Mode::from_raw_mode(0o600),
        )
        .map_err(|e| e.to_string())?,
    );
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|e| e.to_string())?;
    root.sync_all().map_err(|e| e.to_string())
}
pub(crate) fn stage(
    root: &Path,
    repo: &Path,
    pins: &BTreeMap<String, String>,
    workload: &str,
    manifest: &Path,
) -> Result<SourceRecord, String> {
    private(root)?;
    let lock = open::pin(root)?;
    descriptor::flock(&lock, FlockOperation::LockExclusive).map_err(|e| e.to_string())?;
    let (record, copied) = recipe(root, repo, pins, workload, manifest)?;
    match descriptor::statat(&lock, "source-custody.json", AtFlags::SYMLINK_NOFOLLOW) {
        Ok(_) => {
            return verify_unlocked(
                root,
                repo,
                pins,
                workload,
                &record.source,
                &record.working_dir,
                Some(manifest),
            );
        }
        Err(rustix::io::Errno::NOENT) => (),
        Err(error) => return Err(error.to_string()),
    }
    descriptor::mkdirat(&lock, "source-module", Mode::from_raw_mode(0o700))
        .map_err(|e| format!("source module collision: {e}"))?;
    let module = open::child(&lock, "source-module", true, false)?;
    for (relative, (_, body)) in copied {
        let parts: Vec<_> = relative.iter().collect();
        let mut directory = module.try_clone().map_err(|e| e.to_string())?;
        for part in &parts[..parts.len() - 1] {
            let name = part.to_str().ok_or("source directory UTF8")?;
            match descriptor::mkdirat(&directory, name, Mode::from_raw_mode(0o700)) {
                Ok(()) | Err(rustix::io::Errno::EXIST) => (),
                Err(e) => return Err(e.to_string()),
            }
            directory = open::child(&directory, name, true, false)?;
        }
        publish(
            &directory,
            parts
                .last()
                .and_then(|n| n.to_str())
                .ok_or("source name UTF8")?,
            &body,
        )?;
    }
    let mut encoded = canonical::encode(&record)?;
    encoded.push(b'\n');
    publish(&lock, "source-custody.json", &encoded)?;
    verify_unlocked(
        root,
        repo,
        pins,
        workload,
        &record.source,
        &record.working_dir,
        Some(manifest),
    )
}

#[cfg(test)]
#[path = "source_tests.rs"]
mod tests;
