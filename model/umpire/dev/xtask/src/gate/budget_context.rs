use serde::Serialize;
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeSet,
    fs::{self, File, Metadata, OpenOptions},
    io::{self, Write},
    os::unix::{
        ffi::OsStrExt,
        fs::{FileExt, MetadataExt, OpenOptionsExt, PermissionsExt},
    },
    path::{Path, PathBuf},
};

const MAX_ENTRIES: usize = 1_000_000;
const MAX_FILE_BYTES: u64 = 2 * 1024 * 1024 * 1024;
const MAX_TOTAL_BYTES: u64 = 64 * 1024 * 1024 * 1024;
const DEFAULT_EXCLUDED: [&str; 4] = [".git", "target", "build", ".bin"];

#[derive(Clone, Debug)]
pub(super) struct SourceRoot {
    pub label: String,
    pub root: PathBuf,
    pub excluded: Vec<PathBuf>,
}
#[derive(Debug, Serialize, PartialEq, Eq)]
pub(super) struct FileInput {
    pub path_bytes: Vec<u8>,
    pub sha256: String,
    pub executable_mode: u32,
}
#[derive(Debug, Serialize, PartialEq, Eq)]
pub(super) struct SourceInput {
    pub label: String,
    pub excluded: Vec<Vec<u8>>,
    pub files: Vec<FileInput>,
}
#[derive(Debug, Serialize, PartialEq, Eq)]
pub(super) struct Authority {
    pub version: u32,
    pub sources: Vec<SourceInput>,
    pub tools: Vec<FileInput>,
    pub semantic: serde_json::Value,
}
#[derive(Debug, PartialEq, Eq)]
struct Identity {
    device: u64,
    inode: u64,
    size: u64,
    mode: u32,
    modified: (i64, i64),
    changed: (i64, i64),
}
impl Identity {
    fn of(metadata: &Metadata) -> Self {
        Self {
            device: metadata.dev(),
            inode: metadata.ino(),
            size: metadata.size(),
            mode: metadata.mode(),
            modified: (metadata.mtime(), metadata.mtime_nsec()),
            changed: (metadata.ctime(), metadata.ctime_nsec()),
        }
    }
}
#[derive(Debug, PartialEq, Eq)]
struct Guard {
    path: PathBuf,
    identity: Identity,
}
struct Tool {
    path: PathBuf,
    file: File,
    identity: Identity,
}
pub(super) struct Inputs {
    roots: Vec<SourceRoot>,
    tools: Vec<Tool>,
    guards: Vec<Guard>,
    authority: Authority,
    digest: String,
}
fn invalid(message: impl Into<String>) -> io::Error {
    io::Error::new(io::ErrorKind::InvalidData, message.into())
}
fn serialized(value: &impl Serialize) -> io::Result<Vec<u8>> {
    serde_json::to_vec(value).map_err(invalid_json)
}
fn invalid_json(error: serde_json::Error) -> io::Error {
    invalid(error.to_string())
}
fn hex_digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
fn normal_relative(path: &Path) -> bool {
    let bytes = path.as_os_str().as_bytes();
    !bytes.is_empty()
        && bytes
            .split(|byte| *byte == b'/')
            .all(|part| !part.is_empty() && part != b"." && part != b"..")
}
fn canonical(path: &Path) -> io::Result<()> {
    if !path.is_absolute()
        || fs::canonicalize(path)?.as_os_str().as_bytes() != path.as_os_str().as_bytes()
    {
        return Err(invalid(format!("input path must be canonical: {}", path.display())));
    }
    Ok(())
}
fn plain_path(root: &Path, relative: &Path) -> io::Result<()> {
    let mut path = root.to_owned();
    for component in relative.components() {
        path.push(component);
        match fs::symlink_metadata(&path) {
            Ok(metadata) if metadata.file_type().is_symlink() => {
                return Err(invalid(format!("input symlink refused: {}", path.display())));
            }
            Ok(_) => (),
            Err(error) if error.kind() == io::ErrorKind::NotFound => break,
            Err(error) => return Err(error),
        }
    }
    Ok(())
}
fn exclusions(source: &SourceRoot) -> io::Result<Vec<PathBuf>> {
    canonical(&source.root)?;
    if !fs::symlink_metadata(&source.root)?.is_dir() {
        return Err(invalid("source root must be a directory"));
    }
    let mut paths = DEFAULT_EXCLUDED.map(PathBuf::from).to_vec();
    for path in &source.excluded {
        if !normal_relative(path) {
            return Err(invalid("source exclusion must contain only normal relative components"));
        }
        paths.push(path.clone());
    }
    paths.sort_by(|a, b| a.as_os_str().as_bytes().cmp(b.as_os_str().as_bytes()));
    paths.dedup();
    for path in &paths {
        plain_path(&source.root, path)?;
    }
    Ok(paths)
}
fn label_valid(label: &str) -> bool {
    !label.is_empty()
        && label.len() <= 128
        && label.as_bytes()[0].is_ascii_lowercase()
        && label.bytes().all(|byte| {
            byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-' || byte == b'_'
        })
}
fn hash_file(file: &File) -> io::Result<(String, Identity)> {
    let before = file.metadata()?;
    if !before.is_file() || before.len() > MAX_FILE_BYTES {
        return Err(invalid("input must be a bounded regular file"));
    }
    let identity = Identity::of(&before);
    let mut digest = Sha256::new();
    let mut offset = 0_u64;
    let mut buffer = [0_u8; 64 * 1024];
    loop {
        let size = file.read_at(&mut buffer, offset)?;
        if size == 0 {
            break;
        }
        offset += u64::try_from(size).map_err(|_| invalid("input read size overflow"))?;
        if offset > MAX_FILE_BYTES {
            return Err(invalid("input exceeds file byte bound"));
        }
        digest.update(&buffer[..size]);
    }
    if offset != before.len() || identity != Identity::of(&file.metadata()?) {
        return Err(invalid("input changed while hashing"));
    }
    Ok((format!("{:x}", digest.finalize()), identity))
}
fn open_input(path: &Path) -> io::Result<(File, String, Identity)> {
    canonical(path)?;
    let before = fs::symlink_metadata(path)?;
    if !before.is_file() {
        return Err(invalid("input must be a regular file"));
    }
    let file = File::open(path)?;
    let (digest, identity) = hash_file(&file)?;
    if identity != Identity::of(&before) || identity != Identity::of(&fs::symlink_metadata(path)?) {
        return Err(invalid("input path changed while opening"));
    }
    Ok((file, digest, identity))
}
#[derive(Default)]
struct Inventory {
    guards: Vec<Guard>,
    entries: usize,
    bytes: u64,
}
impl Inventory {
    fn walk(
        &mut self,
        root: &Path,
        path: &Path,
        excluded: &[PathBuf],
        files: &mut Vec<FileInput>,
    ) -> io::Result<()> {
        self.entries += 1;
        if self.entries > MAX_ENTRIES {
            return Err(invalid("source inventory exceeds entry bound"));
        }
        let before = fs::symlink_metadata(path)?;
        if before.file_type().is_symlink() {
            return Err(invalid(format!("source symlink refused: {}", path.display())));
        }
        let identity = Identity::of(&before);
        if before.is_dir() {
            let mut children = fs::read_dir(path)?
                .take(MAX_ENTRIES.saturating_sub(self.entries) + 1)
                .map(|entry| entry.map(|entry| entry.path()))
                .collect::<io::Result<Vec<_>>>()?;
            if children.len() > MAX_ENTRIES - self.entries {
                return Err(invalid("source directory exceeds entry bound"));
            }
            children.sort_by(|a, b| a.as_os_str().as_bytes().cmp(b.as_os_str().as_bytes()));
            for child in children {
                let relative =
                    child.strip_prefix(root).map_err(|error| invalid(error.to_string()))?;
                if !excluded.iter().any(|excluded| relative.starts_with(excluded)) {
                    self.walk(root, &child, excluded, files)?;
                }
            }
        } else if before.is_file() {
            self.bytes = self
                .bytes
                .checked_add(before.len())
                .ok_or_else(|| invalid("source byte count overflow"))?;
            if self.bytes > MAX_TOTAL_BYTES {
                return Err(invalid("source inventory exceeds byte bound"));
            }
            let (_, sha256, opened) = open_input(path)?;
            if opened != identity {
                return Err(invalid("source changed while opening"));
            }
            files.push(FileInput {
                path_bytes: path
                    .strip_prefix(root)
                    .map_err(|error| invalid(error.to_string()))?
                    .as_os_str()
                    .as_bytes()
                    .to_vec(),
                sha256,
                executable_mode: before.mode() & 0o111,
            });
        } else {
            return Err(invalid("source is not a regular file or directory"));
        }
        if identity != Identity::of(&fs::symlink_metadata(path)?) {
            return Err(invalid("source changed while inventorying"));
        }
        let guard_identity = if before.is_dir() {
            Identity { size: 0, modified: (0, 0), changed: (0, 0), ..identity }
        } else {
            identity
        };
        self.guards.push(Guard { path: path.to_owned(), identity: guard_identity });
        Ok(())
    }
}
fn snapshot(roots: &[SourceRoot]) -> io::Result<(Vec<SourceInput>, Vec<Guard>)> {
    let mut labels = BTreeSet::new();
    let mut ordered = roots.iter().collect::<Vec<_>>();
    ordered.sort_by(|a, b| a.label.cmp(&b.label));
    let mut inventory = Inventory::default();
    let mut sources = Vec::new();
    for source in ordered {
        if !label_valid(&source.label) || !labels.insert(&source.label) {
            return Err(invalid("source labels must be unique normalized identifiers"));
        }
        let excluded = exclusions(source)?;
        let mut files = Vec::new();
        inventory.walk(&source.root, &source.root, &excluded, &mut files)?;
        files.sort_by(|a, b| a.path_bytes.cmp(&b.path_bytes));
        sources.push(SourceInput {
            label: source.label.clone(),
            excluded: excluded.iter().map(|path| path.as_os_str().as_bytes().to_vec()).collect(),
            files,
        });
    }
    Ok((sources, inventory.guards))
}
impl Inputs {
    pub fn capture(
        roots: &[SourceRoot],
        tools: &[PathBuf],
        semantic: &impl Serialize,
    ) -> io::Result<Self> {
        let semantic = serde_json::to_value(semantic).map_err(invalid_json)?;
        let (sources, guards) = snapshot(roots)?;
        let mut retained = Vec::new();
        let mut tool_inputs = Vec::new();
        for path in tools {
            let (file, sha256, identity) = open_input(path)?;
            tool_inputs.push(FileInput {
                path_bytes: path.as_os_str().as_bytes().to_vec(),
                sha256,
                executable_mode: identity.mode & 0o111,
            });
            retained.push(Tool { path: path.clone(), file, identity });
        }
        let authority = Authority { version: 1, sources, tools: tool_inputs, semantic };
        let digest = hex_digest(&serialized(&authority)?);
        let inputs = Self { roots: roots.to_vec(), tools: retained, guards, authority, digest };
        inputs.verify_unchanged()?;
        Ok(inputs)
    }
    pub fn verify_unchanged(&self) -> io::Result<()> {
        let (sources, guards) = snapshot(&self.roots)?;
        if sources != self.authority.sources || guards != self.guards {
            return Err(invalid("admitted source inputs changed"));
        }
        for (tool, admitted) in self.tools.iter().zip(&self.authority.tools) {
            let (_, path_digest, path_identity) = open_input(&tool.path)?;
            let (retained_digest, retained_identity) = hash_file(&tool.file)?;
            if path_digest != admitted.sha256
                || retained_digest != admitted.sha256
                || path_identity != tool.identity
                || retained_identity != tool.identity
            {
                return Err(invalid("admitted tool input changed"));
            }
        }
        Ok(())
    }
    pub fn digest(&self) -> &str {
        &self.digest
    }
    pub fn authority(&self) -> &Authority {
        &self.authority
    }
}

pub(super) struct PinnedImage {
    file: File,
    digest: String,
    identity: Identity,
}
impl PinnedImage {
    fn from_file(file: File) -> io::Result<Self> {
        let (digest, identity) = hash_file(&file)?;
        Ok(Self { file, digest, identity })
    }
    pub fn digest(&self) -> &str {
        &self.digest
    }
    pub fn verify_unchanged(&self) -> io::Result<()> {
        let (digest, identity) = hash_file(&self.file)?;
        // Unlinking or replacing the original name changes ctime, not the running image bytes.
        if digest != self.digest
            || identity.device != self.identity.device
            || identity.inode != self.identity.inode
            || identity.mode != self.identity.mode
        {
            return Err(invalid("retained running image changed"));
        }
        Ok(())
    }
    pub fn copy_to(&self, destination: &Path) -> io::Result<()> {
        self.verify_unchanged()?;
        let mut output = OpenOptions::new()
            .write(true)
            .read(true)
            .create_new(true)
            .mode(0o700)
            .open(destination)?;
        let mut offset = 0_u64;
        let mut buffer = [0_u8; 64 * 1024];
        loop {
            let size = self.file.read_at(&mut buffer, offset)?;
            if size == 0 {
                break;
            }
            offset += u64::try_from(size).map_err(|_| invalid("image size overflow"))?;
            if offset > MAX_FILE_BYTES {
                return Err(invalid("image exceeds byte bound"));
            }
            output.write_all(&buffer[..size])?;
        }
        output.set_permissions(fs::Permissions::from_mode(0o700))?;
        output.sync_all()?;
        if hash_file(&output)?.0 != self.digest {
            return Err(invalid("image changed while copying"));
        }
        self.verify_unchanged()
    }
}
pub(super) fn pin_current_image() -> io::Result<PinnedImage> {
    #[cfg(target_os = "linux")]
    {
        PinnedImage::from_file(File::open("/proc/self/exe")?)
    }
    #[cfg(not(target_os = "linux"))]
    {
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            "exact current executable image pinning is supported only on Linux",
        ))
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        ffi::OsString,
        fs,
        os::unix::{
            ffi::OsStringExt,
            fs::{PermissionsExt, symlink},
        },
    };
    fn root(path: &Path) -> SourceRoot {
        SourceRoot { label: "fixture".into(), root: path.canonicalize().unwrap(), excluded: vec![] }
    }
    fn capture(path: &Path) -> io::Result<Inputs> {
        Inputs::capture(
            &[root(path)],
            &[],
            &serde_json::json!({"budget": 10, "selection": ["one"]}),
        )
    }
    #[test]
    fn changes_after_admission_are_refused() -> io::Result<()> {
        for mutation in ["edit", "add", "delete", "mode", "replace", "symlink"] {
            let dir = tempfile::tempdir()?;
            let file = dir.path().join("source");
            fs::write(&file, "before admission")?;
            fs::write(&file, "admitted")?;
            let inputs = capture(dir.path())?;
            inputs.verify_unchanged()?;
            match mutation {
                "edit" => fs::write(&file, "after")?,
                "add" => fs::write(dir.path().join("added"), "new")?,
                "delete" => fs::remove_file(&file)?,
                "mode" => fs::set_permissions(&file, fs::Permissions::from_mode(0o755))?,
                "replace" => {
                    fs::remove_file(&file)?;
                    fs::write(&file, "admitted")?;
                }
                _ => {
                    fs::remove_file(&file)?;
                    symlink("missing", &file)?;
                }
            }
            assert!(inputs.verify_unchanged().is_err(), "accepted {mutation}");
        }
        Ok(())
    }
    #[test]
    fn copied_roots_and_raw_names_have_the_same_semantic_digest() -> io::Result<()> {
        let one = tempfile::tempdir()?;
        let two = tempfile::tempdir()?;
        let name = OsString::from_vec(vec![b'f', 0xff]);
        fs::write(one.path().join(&name), "same")?;
        fs::write(two.path().join(&name), "same")?;
        assert_eq!(capture(one.path())?.digest(), capture(two.path())?.digest());
        Ok(())
    }
    #[test]
    fn budget_and_selection_are_bound_to_the_digest() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let roots = [root(dir.path())];
        let a =
            Inputs::capture(&roots, &[], &serde_json::json!({"budget": 10, "selection": ["one"]}))?;
        for semantic in [
            serde_json::json!({"budget": 11, "selection": ["one"]}),
            serde_json::json!({"budget": 10, "selection": ["two"]}),
        ] {
            assert_ne!(a.digest(), Inputs::capture(&roots, &[], &semantic)?.digest());
        }
        Ok(())
    }
    #[test]
    fn explicit_exclusions_cannot_escape_or_follow_symlinks() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        symlink("missing", dir.path().join("alias"))?;
        for excluded in ["", ".", "..", "../outside", "/outside", "a/../b", "alias", "alias/child"]
        {
            let mut source = root(dir.path());
            source.excluded.push(PathBuf::from(excluded));
            assert!(Inputs::capture(&[source], &[], &()).is_err(), "accepted {excluded}");
        }
        fs::remove_file(dir.path().join("alias"))?;
        fs::create_dir(dir.path().join("outputs"))?;
        let mut source = root(dir.path());
        source.excluded.push("outputs".into());
        let inputs = Inputs::capture(&[source], &[], &())?;
        fs::write(dir.path().join("outputs/generated"), "output")?;
        inputs.verify_unchanged()?;
        Ok(())
    }
    #[test]
    fn root_aliases_and_duplicate_labels_are_refused() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let source = root(dir.path());
        assert!(Inputs::capture(&[source.clone(), source.clone()], &[], &()).is_err());
        let mut invalid = source.clone();
        invalid.label = "../bad".into();
        assert!(Inputs::capture(&[invalid], &[], &()).is_err());
        let mut invalid = source;
        invalid.root.push(".");
        assert!(Inputs::capture(&[invalid], &[], &()).is_err());
        Ok(())
    }
    #[test]
    fn external_tool_replacement_is_refused_even_with_identical_bytes() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let tool = dir.path().join("tool");
        fs::write(&tool, "image")?;
        let inputs = Inputs::capture(&[], &[tool.canonicalize()?], &())?;
        fs::rename(&tool, dir.path().join("old"))?;
        fs::write(&tool, "image")?;
        assert!(inputs.verify_unchanged().is_err());
        Ok(())
    }
    #[test]
    fn retained_image_copies_original_bytes_after_path_replacement() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let path = dir.path().join("image");
        fs::write(&path, "running image")?;
        let image = PinnedImage::from_file(File::open(&path)?)?;
        let digest = image.digest().to_owned();
        fs::rename(&path, dir.path().join("old"))?;
        fs::write(&path, "replacement image")?;
        let destination = dir.path().join("private-image");
        image.copy_to(&destination)?;
        assert_eq!(fs::read(&destination)?, b"running image");
        assert_ne!(fs::metadata(&destination)?.permissions().mode() & 0o111, 0);
        assert_eq!(image.digest(), digest);
        assert!(image.copy_to(&destination).is_err());
        Ok(())
    }
}
