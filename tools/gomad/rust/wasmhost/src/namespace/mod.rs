pub(crate) mod call;
mod image;
#[cfg(not(feature = "conformance"))]
pub(crate) mod transport;
#[cfg(any(test, feature = "conformance"))]
pub(crate) use image::decode as decode_image;

use std::collections::{BTreeMap, BTreeSet};

use crate::volume::{self, Storage, StorageStop};

pub const BADF: u16 = 8;
pub const EXIST: u16 = 20;
pub const INVAL: u16 = 28;
pub const ISDIR: u16 = 31;
pub const NOENT: u16 = 44;
pub const NOTDIR: u16 = 54;
pub const NOTEMPTY: u16 = 55;
pub const ROFS: u16 = 69;
pub const NOTCAPABLE: u16 = 76;
pub const EXDEV: u16 = 75;
pub const STORAGE_PATH_UNSUPPORTED: &str = "storage path outside UTF-8 profile";
pub const READ: u64 = 1 << 1;
pub const SEEK: u64 = 1 << 2;
pub const SET_FLAGS: u64 = 1 << 3;
pub const SYNC: u64 = 1 << 4;
pub const WRITE: u64 = 1 << 6;
pub const CREATE_DIRECTORY: u64 = 1 << 9;
pub const CREATE_FILE: u64 = 1 << 10;
pub const PATH_OPEN: u64 = 1 << 13;
pub const READDIR: u64 = 1 << 14;
pub const RENAME_SOURCE: u64 = 1 << 16;
pub const RENAME_TARGET: u64 = 1 << 17;
pub const PATH_STAT: u64 = 1 << 18;
pub const PATH_SET_SIZE: u64 = 1 << 19;
pub const FILE_STAT: u64 = 1 << 21;
pub const SET_SIZE: u64 = 1 << 22;
pub const REMOVE_DIRECTORY: u64 = 1 << 25;
pub const UNLINK: u64 = 1 << 26;
pub const POLL: u64 = 1 << 27;
pub const RIGHTS_ALL: u64 = (1 << 29) - 1;
pub const FD_FLAGS_UNSUPPORTED: &str = "descriptor flags outside volatile profile";
pub const OPEN_FLAGS_UNSUPPORTED: &str = "open flags outside volatile profile";
pub const READLINK_UNSUPPORTED: &str = "operation outside stock WASI profile: path_readlink";
const BUFFER_LIMIT: usize = 4 << 20;

#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub struct Limits {
    pub files: u64,
    pub descriptors: u64,
    pub filesystem_bytes: u64,
    pub output_bytes: u64,
}
#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub struct ImageEntry {
    pub path: Vec<u8>,
    pub directory: bool,
    pub readonly: bool,
    pub data: Vec<u8>,
}
#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub struct Image {
    pub entries: Vec<ImageEntry>,
    pub writable_directories: Vec<Vec<u8>>,
    pub preopens: Vec<Vec<u8>>,
    pub working_directory: Vec<u8>,
    pub stdin: Vec<u8>,
    pub limits: Limits,
    pub epoch_nanos: u64,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Boundary {
    pub kind: &'static str,
    pub dimension: &'static str,
}
impl std::fmt::Display for Boundary {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "WASI {}: {}", self.kind, self.dimension)
    }
}
impl std::error::Error for Boundary {}
#[derive(Clone, Debug)]
pub struct Open {
    pub fd: u32,
    pub path: Vec<u8>,
    pub dirflags: u32,
    pub oflags: u16,
    pub base: u64,
    pub inheriting: u64,
    pub flags: u16,
}
#[derive(Clone, Debug)]
pub enum Operation {
    PrestatGet(u32),
    PrestatDirName {
        fd: u32,
        length: u32,
    },
    FdstatGet(u32),
    FilestatGet(u32),
    Close(u32),
    Sync(u32),
    SetFlags {
        fd: u32,
        flags: u16,
    },
    Read {
        fd: u32,
        length: u32,
        offset: Option<u64>,
    },
    Write {
        fd: u32,
        data: Vec<u8>,
        offset: Option<u64>,
    },
    SetSize {
        fd: u32,
        size: u64,
    },
    Seek {
        fd: u32,
        offset: i64,
        whence: u8,
    },
    Readdir {
        fd: u32,
        length: u32,
        cookie: u64,
    },
    Open(Open),
    CreateDirectory {
        fd: u32,
        path: Vec<u8>,
    },
    RemoveDirectory {
        fd: u32,
        path: Vec<u8>,
    },
    UnlinkFile {
        fd: u32,
        path: Vec<u8>,
    },
    PathStat {
        fd: u32,
        path: Vec<u8>,
        flags: u32,
    },
    Rename {
        fd: u32,
        path: Vec<u8>,
        new_fd: u32,
        new_path: Vec<u8>,
    },
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    Readlink {
        fd: u32,
        path: Vec<u8>,
        length: u32,
    },
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    Symlink {
        old_path: Vec<u8>,
        fd: u32,
        path: Vec<u8>,
    },
    #[cfg(any(test, feature = "conformance"))]
    Unsupported(String),
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Fdstat {
    pub filetype: u8,
    pub flags: u16,
    pub base: u64,
    pub inheriting: u64,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Filestat {
    pub dev: u64,
    pub ino: u64,
    pub filetype: u8,
    pub nlink: u64,
    pub size: u64,
    pub atim: u64,
    pub mtim: u64,
    pub ctim: u64,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Output {
    Empty,
    Prestat(u32),
    Fdstat(Fdstat),
    Filestat(Filestat),
    Bytes(Vec<u8>),
    Fd(u32),
    Written(u32),
    Offset(u64),
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Reply {
    pub errno: u16,
    pub unsupported: Option<String>,
    pub output: Option<Output>,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FdReadiness {
    pub errno: u16,
    pub nbytes: u64,
    pub flags: u16,
}

struct Node {
    ino: u64,
    directory: bool,
    readonly: bool,
    data: Vec<u8>,
    children: BTreeMap<Vec<u8>, u64>,
    modified: u64,
    /// Go `node.storagePath`: the volume resource, empty outside volumes.
    storage_path: String,
    storage_size: u64,
}
#[derive(Clone)]
struct Descriptor {
    node: u64,
    offset: u64,
    rights: u64,
    inheriting: u64,
    flags: u16,
    preopen: Vec<u8>,
    std: u32,
}
enum Failure {
    Errno(u16),
    Boundary(Boundary),
    Unsupported(String),
    Storage(StorageStop),
}

/// Why a namespace operation ends the run instead of returning a reply.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Stop {
    Boundary(Boundary),
    /// A modeled `crash-before` storage fault, by fault name.
    StorageCrash(String),
    /// A Go error outside the boundary taxonomy, by its `Error()` text.
    Fatal(String),
}
impl std::fmt::Display for Stop {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Stop::Boundary(boundary) => boundary.fmt(f),
            Stop::StorageCrash(fault) => write!(f, "modeled storage crash: {fault}"),
            Stop::Fatal(message) => f.write_str(message),
        }
    }
}
impl std::error::Error for Stop {}
impl Stop {
    #[cfg(test)]
    fn dimension(&self) -> &str {
        match self {
            Stop::Boundary(boundary) => boundary.dimension,
            Stop::StorageCrash(fault) | Stop::Fatal(fault) => fault,
        }
    }
    /// The engine error: boundaries keep their own type for classification.
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    pub fn into_error(self) -> wasmtime::Error {
        match self {
            Stop::Boundary(boundary) => wasmtime::Error::new(boundary),
            other => wasmtime::Error::new(other),
        }
    }
}
impl From<StorageStop> for Failure {
    fn from(stop: StorageStop) -> Self {
        Failure::Storage(stop)
    }
}
fn storage_failure(error: &volume::Error) -> Failure {
    match volume::storage_boundary(error) {
        Some(stop) => Failure::Storage(stop),
        None => Failure::Errno(volume::storage_errno(error)),
    }
}
type Result<T> = std::result::Result<T, Failure>;

pub struct Namespace {
    nodes: BTreeMap<u64, Node>,
    descriptors: BTreeMap<u32, Descriptor>,
    next_fd: u32,
    next_inode: u64,
    files: u64,
    bytes: u64,
    limits: Limits,
    epoch_nanos: u64,
    offered_output: u64,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    output_committed: [u64; 2],
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    event_state_cache: Option<((u64, u64), serde_json::Value)>,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    event_mutations: u64,
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    writable: Vec<Vec<u8>>,
    storage: Option<Storage>,
    /// Go `descriptor.storage`: open volume handles by descriptor.
    handles: BTreeMap<u32, volume::Handle>,
    /// This namespace's process-unique identity and a revision that every
    /// mutating entry point advances: equal pairs denote identical
    /// inspection material, so recording reuses its commitments.
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    inspection_revision: (u64, u64),
}

/// Process-unique namespace identities for inspection revisions.
static NAMESPACE_IDENTITIES: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(1);
impl Namespace {
    #[expect(
        clippy::too_many_lines,
        reason = "Namespace construction validates image topology and admits resources before publishing descriptors."
    )]
    pub fn new(image: Image) -> std::result::Result<Self, Boundary> {
        let mut ns = Self {
            nodes: BTreeMap::new(),
            descriptors: BTreeMap::new(),
            next_fd: 3,
            next_inode: 1,
            files: 0,
            bytes: 0,
            limits: image.limits,
            epoch_nanos: image.epoch_nanos,
            offered_output: 0,
            stdout: Vec::new(),
            stderr: Vec::new(),
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            output_committed: [0; 2],
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_state_cache: None,
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            event_mutations: 0,
            writable: image.writable_directories.clone(),
            storage: None,
            handles: BTreeMap::new(),
            inspection_revision: (
                NAMESPACE_IDENTITIES.fetch_add(1, std::sync::atomic::Ordering::Relaxed),
                0,
            ),
        };
        ns.new_node(true, true);
        for entry in image.entries {
            if !valid_image_path(&entry.path)
                || entry.path == b"/"
                || entry.directory && !entry.data.is_empty()
            {
                return Err(invalid("namespace image entry"));
            }
            let slash = entry.path.iter().rposition(|b| *b == b'/').unwrap();
            let parent = ns
                .lookup(if slash == 0 {
                    b"/"
                } else {
                    &entry.path[..slash]
                })
                .ok_or_else(|| invalid("namespace image parent"))?;
            let name = &entry.path[slash + 1..];
            if !ns.nodes[&parent].directory || ns.nodes[&parent].children.contains_key(name) {
                return Err(invalid("namespace image tree"));
            }
            let ino = ns.new_node(entry.directory, entry.readonly);
            ns.bytes = ns
                .bytes
                .checked_add(entry.data.len() as u64)
                .ok_or_else(|| capacity("initial-namespace"))?;
            ns.nodes.get_mut(&ino).unwrap().data = entry.data;
            ns.nodes
                .get_mut(&parent)
                .unwrap()
                .children
                .insert(name.to_vec(), ino);
        }
        for path in &image.writable_directories {
            if !valid_image_path(path) {
                return Err(invalid("writable directory"));
            }
            if ns.lookup(path).is_some_and(|ino| ns.nodes[&ino].readonly) {
                return Err(invalid("writable mount overlaps captured input"));
            }
            let ino = ns
                .ensure_directories(path)
                .ok_or_else(|| invalid("writable directory beneath file"))?;
            if !ns.nodes[&ino].directory {
                return Err(invalid("writable directory beneath file"));
            }
            ns.nodes.get_mut(&ino).unwrap().readonly = false;
        }
        if !valid_image_path(&image.working_directory)
            || ns
                .lookup(&image.working_directory)
                .is_none_or(|ino| !ns.nodes[&ino].directory)
        {
            return Err(invalid("working directory absent from captured namespace"));
        }
        ns.bytes = ns
            .bytes
            .checked_add(image.stdin.len() as u64)
            .ok_or_else(|| capacity("initial-namespace"))?;
        if ns.files > ns.limits.files || ns.bytes > ns.limits.filesystem_bytes {
            return Err(capacity("initial-namespace"));
        }
        for fd in 0..3 {
            let ino = ns.next_inode;
            ns.next_inode += 1;
            ns.nodes.insert(
                ino,
                Node {
                    ino,
                    directory: false,
                    readonly: false,
                    data: if fd == 0 {
                        image.stdin.clone()
                    } else {
                        Vec::new()
                    },
                    children: BTreeMap::new(),
                    modified: 0,
                    storage_path: String::new(),
                    storage_size: 0,
                },
            );
            let rights = if fd == 0 { READ } else { WRITE } | SET_FLAGS | FILE_STAT | POLL;
            ns.descriptors.insert(
                fd,
                Descriptor {
                    node: ino,
                    offset: 0,
                    rights,
                    inheriting: 0,
                    flags: 0,
                    preopen: Vec::new(),
                    std: fd + 1,
                },
            );
        }
        let preopens: BTreeSet<_> = image.preopens.into_iter().collect();
        if preopens.len() as u64 + 3 > ns.limits.descriptors {
            return Err(capacity("preopen-descriptors"));
        }
        for path in preopens {
            if !valid_image_path(&path) {
                return Err(invalid("preopen directory"));
            }
            let ino = ns
                .lookup(&path)
                .ok_or_else(|| invalid("preopen directory"))?;
            if !ns.nodes[&ino].directory {
                return Err(invalid("preopen directory"));
            }
            ns.descriptors.insert(
                ns.next_fd,
                Descriptor {
                    node: ino,
                    offset: 0,
                    rights: RIGHTS_ALL,
                    inheriting: RIGHTS_ALL,
                    flags: 0,
                    preopen: path,
                    std: 0,
                },
            );
            ns.next_fd += 1;
        }
        Ok(ns)
    }
    fn new_node(&mut self, directory: bool, readonly: bool) -> u64 {
        let ino = self.next_inode;
        self.next_inode += 1;
        self.files += 1;
        self.nodes.insert(
            ino,
            Node {
                ino,
                directory,
                readonly,
                data: Vec::new(),
                children: BTreeMap::new(),
                modified: 0,
                storage_path: String::new(),
                storage_size: 0,
            },
        );
        ino
    }
    fn lookup(&self, path: &[u8]) -> Option<u64> {
        let mut ino = 1;
        for name in path.split(|b| *b == b'/').filter(|name| !name.is_empty()) {
            let n = &self.nodes[&ino];
            if !n.directory {
                return None;
            }
            ino = *n.children.get(name)?;
        }
        Some(ino)
    }
    fn ensure_directories(&mut self, path: &[u8]) -> Option<u64> {
        let mut ino = 1;
        for name in path.split(|b| *b == b'/').filter(|name| !name.is_empty()) {
            if !self.nodes[&ino].directory {
                return None;
            }
            let child = if let Some(child) = self.nodes[&ino].children.get(name) {
                *child
            } else {
                let child = self.new_node(true, false);
                self.nodes
                    .get_mut(&ino)
                    .unwrap()
                    .children
                    .insert(name.to_vec(), child);
                child
            };
            ino = child;
        }
        Some(ino)
    }
    fn get_fd(&self, fd: u32, rights: u64) -> Result<&Descriptor> {
        let d = self.descriptors.get(&fd).ok_or(Failure::Errno(BADF))?;
        if d.rights & rights != rights {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        Ok(d)
    }
    fn resolve_path(&self, d: &Descriptor, path: &[u8]) -> Result<(u64, Vec<u8>)> {
        if !self.nodes[&d.node].directory {
            return Err(Failure::Errno(NOTDIR));
        }
        if path.is_empty() {
            return Err(Failure::Errno(NOENT));
        }
        if path.len() > 4096 || path[0] == b'/' || path.contains(&0) {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        let mut depth = 0;
        for name in path.split(|b| *b == b'/') {
            match name {
                b"" | b"." => {}
                b".." => {
                    if depth == 0 {
                        return Err(Failure::Errno(NOTCAPABLE));
                    }
                    depth -= 1;
                }
                _ => depth += 1,
            }
        }
        let mut stack = Vec::new();
        let mut nodes = vec![Some(d.node)];
        for name in path.split(|b| *b == b'/') {
            if name.is_empty() {
                continue;
            }
            let current = nodes
                .last()
                .copied()
                .flatten()
                .ok_or(Failure::Errno(NOENT))?;
            let n = &self.nodes[&current];
            if !n.directory {
                return Err(Failure::Errno(NOTDIR));
            }
            match name {
                b"." => {}
                b".." => {
                    stack.pop();
                    nodes.pop();
                }
                _ => {
                    stack.push(name);
                    nodes.push(n.children.get(name).copied());
                }
            }
        }
        if stack.is_empty() {
            return Ok((d.node, Vec::new()));
        }
        if path.ends_with(b"/")
            && nodes
                .last()
                .copied()
                .flatten()
                .is_some_and(|ino| !self.nodes[&ino].directory)
        {
            return Err(Failure::Errno(NOTDIR));
        }
        Ok((
            nodes[nodes.len() - 2].unwrap(),
            stack.last().unwrap().to_vec(),
        ))
    }
    fn child_at(&self, parent: u64, name: &[u8]) -> Option<u64> {
        if name.is_empty() {
            Some(parent)
        } else {
            self.nodes[&parent].children.get(name).copied()
        }
    }
    fn stat(&self, ino: u64) -> Filestat {
        let n = &self.nodes[&ino];
        let timestamp = self.epoch_nanos.wrapping_add(n.modified);
        Filestat {
            dev: 1,
            ino: n.ino,
            filetype: node_type(n),
            nlink: 1,
            size: node_size(n),
            atim: timestamp,
            mtim: timestamp,
            ctim: timestamp,
        }
    }
    /// Go `initializeStorage` after validation: mounts the volumes on their
    /// empty writable directories, loads the durable tree (recovery) and
    /// opens a volume handle for every descriptor already on a volume.
    pub(crate) fn attach_storage(&mut self, storage: Storage) -> std::result::Result<(), Stop> {
        self.touch();
        let mounts: Vec<String> = storage
            .model
            .volumes()
            .iter()
            .map(|config| config.mount.clone())
            .collect();
        self.storage = Some(storage);
        let fail = |failure: Failure| match failure {
            Failure::Boundary(boundary) => Stop::Boundary(boundary),
            Failure::Storage(StorageStop::Boundary(kind, dimension)) => {
                Stop::Boundary(Boundary { kind, dimension })
            }
            Failure::Storage(StorageStop::Crash(fault)) => Stop::StorageCrash(fault),
            Failure::Storage(StorageStop::Fatal(message)) => Stop::Fatal(message),
            Failure::Errno(_) | Failure::Unsupported(_) => {
                Stop::Fatal("storage initialization".into())
            }
        };
        for mount in mounts {
            let ino = self
                .lookup(mount.as_bytes())
                .filter(|ino| self.nodes[ino].directory && self.nodes[ino].children.is_empty())
                .ok_or(Stop::Boundary(invalid(
                    "storage mount must be an empty writable directory",
                )))?;
            self.nodes.get_mut(&ino).unwrap().storage_path = mount;
            self.load_storage_tree(ino).map_err(fail)?;
        }
        if self.files > self.limits.files || self.bytes > self.limits.filesystem_bytes {
            return Err(Stop::Boundary(capacity("recovered-namespace")));
        }
        let descriptors: Vec<(u32, u64)> = self
            .descriptors
            .iter()
            .map(|(fd, d)| (*fd, d.node))
            .collect();
        for (fd, ino) in descriptors {
            let resource = self.nodes[&ino].storage_path.clone();
            if resource.is_empty() {
                continue;
            }
            let storage = self.storage.as_mut().unwrap();
            if let Some(mut previous) = self.handles.remove(&fd) {
                storage
                    .fs
                    .close(&mut previous)
                    .map_err(|error| Stop::Fatal(volume::storage_error_text(&error)))?;
            }
            let handle = storage
                .open(
                    &resource,
                    volume::OpenFlags {
                        read: true,
                        ..volume::OpenFlags::default()
                    },
                )
                .map_err(|error| Stop::Fatal(volume::storage_error_text(&error)))?;
            self.handles.insert(fd, handle);
        }
        Ok(())
    }
    fn load_storage_tree(&mut self, ino: u64) -> Result<()> {
        let resource = self.nodes[&ino].storage_path.clone();
        let storage = self.storage.as_mut().unwrap();
        let fatal = |error: volume::Error| {
            Failure::Storage(StorageStop::Fatal(volume::storage_error_text(&error)))
        };
        let mut handle = storage
            .open(
                &resource,
                volume::OpenFlags {
                    read: true,
                    ..volume::OpenFlags::default()
                },
            )
            .map_err(fatal)?;
        let entries = storage.fs.handle_read_dir(&mut handle, -1);
        let closed = storage.fs.close(&mut handle);
        let entries = entries.map_err(fatal)?;
        closed.map_err(fatal)?;
        for (name, entry) in entries {
            let directory = entry.kind == volume::Kind::Directory;
            let child = self.new_node(directory, false);
            let path = format!("{resource}/{name}");
            if self.storage.as_ref().unwrap().model.lock_path(&path) {
                return Err(Failure::Boundary(invalid(
                    "durable image contains process-owned SQLite lock",
                )));
            }
            let size = entry.data.len() as u64;
            let node = self.nodes.get_mut(&child).unwrap();
            node.storage_path = path;
            node.modified = entry.mod_time.max(0).cast_unsigned();
            node.storage_size = size;
            self.bytes += size;
            self.nodes
                .get_mut(&ino)
                .unwrap()
                .children
                .insert(name.into_bytes(), child);
            if directory {
                self.load_storage_tree(child)?;
            }
        }
        Ok(())
    }
    /// Go `storagePath`: the volume resource for `name` beneath `parent`, or
    /// empty outside volumes and for process-owned SQLite lock artifacts.
    fn storage_path(&self, parent: u64, name: &[u8]) -> Result<String> {
        let parent_path = &self.nodes[&parent].storage_path;
        if parent_path.is_empty() {
            return Ok(String::new());
        }
        if name.is_empty() {
            return Ok(parent_path.clone());
        }
        let name = std::str::from_utf8(name)
            .map_err(|_| Failure::Unsupported(STORAGE_PATH_UNSUPPORTED.into()))?;
        let resource = format!("{parent_path}/{name}");
        if self.storage.as_ref().unwrap().model.lock_path(&resource) {
            return Ok(String::new());
        }
        Ok(resource)
    }
    /// Go `storageWrite`: returns the written count and the errno to report
    /// after the node refresh, or a stop that skips the refresh.
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Storage decisions return a prefix of the BUFFER_LIMIT-bounded input, and writes return a count within that prefix."
    )]
    fn storage_write(
        &mut self,
        fd: u32,
        ino: u64,
        input: &[u8],
        offset: u64,
        now: u64,
    ) -> Result<(u32, u16)> {
        let resource = self.nodes[&ino].storage_path.clone();
        let storage = self.storage.as_mut().unwrap();
        let (count, errno) = storage.decision(&resource, "write", input.len() as u64, now)?;
        if errno != 0 && count == input.len() as u64 {
            return Ok((0, errno));
        }
        let mut final_size = self.nodes[&ino].storage_size;
        if count > 0 {
            final_size = final_size.max(offset + count);
        }
        self.storage_resize_allowed(ino, final_size)?;
        let storage = self.storage.as_mut().unwrap();
        let clock = storage.clock;
        match storage.fs.handle_write_at(
            &self.handles[&fd],
            offset.cast_signed(),
            &input[..count as usize],
            clock,
        ) {
            Err(error) => match volume::storage_boundary(&error) {
                Some(stop) => Err(Failure::Storage(stop)),
                None => Ok((0, volume::storage_errno(&error))),
            },
            Ok(_) if errno != 0 => Ok((0, errno)),
            Ok(written) => Ok((written as u32, 0)),
        }
    }
    fn storage_resize_allowed(&self, ino: u64, size: u64) -> Result<()> {
        let maximum = self.limits.filesystem_bytes;
        if size > maximum || self.bytes - self.nodes[&ino].storage_size > maximum - size {
            return Err(Failure::Boundary(capacity("filesystem-bytes")));
        }
        Ok(())
    }
    fn refresh_storage_node(&mut self, fd: u32) -> Result<()> {
        let ino = self.descriptors[&fd].node;
        let (mod_time, size) = self
            .storage
            .as_ref()
            .unwrap()
            .fs
            .handle_metadata(&self.handles[&fd])
            .map_err(|error| {
                Failure::Storage(StorageStop::Fatal(volume::storage_error_text(&error)))
            })?;
        let node = self.nodes.get_mut(&ino).unwrap();
        self.bytes = self.bytes - node.storage_size + size;
        node.storage_size = size;
        node.modified = mod_time.max(0).cast_unsigned();
        Ok(())
    }
    fn relocate_storage_node(&mut self, ino: u64, name: &str) {
        let node = self.nodes.get_mut(&ino).unwrap();
        name.clone_into(&mut node.storage_path);
        let children: Vec<(Vec<u8>, u64)> = node
            .children
            .iter()
            .map(|(child, ino)| (child.clone(), *ino))
            .collect();
        for (child, child_ino) in children {
            self.relocate_storage_node(
                child_ino,
                &format!("{name}/{}", String::from_utf8_lossy(&child)),
            );
        }
    }
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    pub fn writable_directories(&self) -> &[Vec<u8>] {
        &self.writable
    }
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    pub fn filesystem_bytes(&self) -> u64 {
        self.limits.filesystem_bytes
    }
    pub(crate) fn storage(&self) -> Option<&Storage> {
        self.storage.as_ref()
    }
    #[cfg(not(feature = "conformance"))]
    pub(crate) fn storage_mut(&mut self) -> Option<&mut Storage> {
        self.touch();
        self.storage.as_mut()
    }
    /// Go `CrashStorage`: the image of the selected durable state. Descriptor
    /// numbers and open volume handles belong to the discarded incarnation.
    pub(crate) fn crash_storage(
        &mut self,
        selections: &BTreeMap<String, Vec<u64>>,
        maximum: u64,
    ) -> std::result::Result<volume::Image, volume::Error> {
        self.touch();
        let image = self.storage.as_mut().unwrap().crash(selections, maximum);
        if image.is_ok() {
            self.descriptors.clear();
            self.handles.clear();
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        self.diagnostic_state(None);
        image
    }
    pub fn execute(&mut self, op: Operation, now: u64) -> std::result::Result<Reply, Stop> {
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        let mutation = match &op {
            Operation::PrestatGet(_)
            | Operation::PrestatDirName { .. }
            | Operation::FdstatGet(_)
            | Operation::FilestatGet(_)
            | Operation::Readdir { .. }
            | Operation::PathStat { .. }
            | Operation::Readlink { .. }
            | Operation::Symlink { .. } => false,
            Operation::Read { fd, offset, .. } => offset.is_none() || self.handles.contains_key(fd),
            #[cfg(test)]
            Operation::Unsupported(_) => false,
            _ => true,
        };
        self.touch();
        let result = self.execute_inner(op, now);
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if mutation {
            self.diagnostic_state(Some(now));
        }
        match result {
            Ok(output) => Ok(Reply {
                errno: 0,
                unsupported: None,
                output: Some(output),
            }),
            Err(Failure::Errno(errno)) => Ok(Reply {
                errno,
                unsupported: None,
                output: None,
            }),
            Err(Failure::Unsupported(message)) => Ok(Reply {
                errno: 0,
                unsupported: Some(message),
                output: None,
            }),
            Err(Failure::Boundary(boundary)) => Err(Stop::Boundary(boundary)),
            Err(Failure::Storage(StorageStop::Boundary(kind, dimension))) => {
                Err(Stop::Boundary(Boundary { kind, dimension }))
            }
            Err(Failure::Storage(StorageStop::Crash(fault))) => Err(Stop::StorageCrash(fault)),
            Err(Failure::Storage(StorageStop::Fatal(message))) => Err(Stop::Fatal(message)),
        }
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Keep namespace operation dispatch and its admission-before-mutation ordering explicit."
    )]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Validated image paths and guest path buffers fit within the WASI u32 length field."
    )]
    fn execute_inner(&mut self, op: Operation, now: u64) -> Result<Output> {
        match op {
            Operation::Readlink { .. } => Err(Failure::Unsupported(READLINK_UNSUPPORTED.into())),
            Operation::Symlink { .. } => Err(Failure::Unsupported(
                "operation outside stock WASI profile: path_symlink".into(),
            )),
            Operation::PrestatGet(fd) => {
                let d = self.get_fd(fd, 0)?;
                if d.preopen.is_empty() {
                    return Err(Failure::Errno(BADF));
                }
                Ok(Output::Prestat(d.preopen.len() as u32))
            }
            Operation::PrestatDirName { fd, length } => {
                let d = self.get_fd(fd, 0)?;
                if d.preopen.is_empty() {
                    return Err(Failure::Errno(BADF));
                }
                if length as usize != d.preopen.len() {
                    return Err(Failure::Errno(INVAL));
                }
                Ok(Output::Bytes(d.preopen.clone()))
            }
            Operation::FdstatGet(fd) => {
                let d = self.get_fd(fd, 0)?;
                Ok(Output::Fdstat(Fdstat {
                    filetype: if d.std != 0 {
                        2
                    } else {
                        node_type(&self.nodes[&d.node])
                    },
                    flags: d.flags,
                    base: d.rights,
                    inheriting: d.inheriting,
                }))
            }
            Operation::FilestatGet(fd) => Ok(Output::Filestat(
                self.stat(self.get_fd(fd, FILE_STAT)?.node),
            )),
            Operation::Close(fd) => {
                self.get_fd(fd, 0)?;
                if let Some(mut handle) = self.handles.remove(&fd) {
                    let storage = self.storage.as_mut().unwrap();
                    if let Err(error) = storage.fs.close(&mut handle) {
                        self.handles.insert(fd, handle);
                        return Err(Failure::Errno(volume::storage_errno(&error)));
                    }
                }
                self.descriptors.remove(&fd);
                Ok(Output::Empty)
            }
            Operation::Sync(fd) => {
                let ino = self.get_fd(fd, SYNC)?.node;
                if self.handles.contains_key(&fd) {
                    let resource = self.nodes[&ino].storage_path.clone();
                    let storage = self.storage.as_mut().unwrap();
                    let (_, errno) = storage.decision(&resource, "sync", 0, now)?;
                    if errno != 0 {
                        return Err(Failure::Errno(errno));
                    }
                    storage
                        .fs
                        .handle_sync(&self.handles[&fd])
                        .map_err(|error| storage_failure(&error))?;
                }
                Ok(Output::Empty)
            }
            Operation::SetFlags { fd, flags } => {
                self.get_fd(fd, SET_FLAGS)?;
                if flags & !(1 | 4) != 0 {
                    return Err(Failure::Unsupported(FD_FLAGS_UNSUPPORTED.into()));
                }
                self.descriptors.get_mut(&fd).unwrap().flags = flags;
                Ok(Output::Empty)
            }
            Operation::Read { fd, length, offset } => self.read(fd, length, offset, now),
            Operation::Write { fd, data, offset } => self.write(fd, &data, offset, now),
            Operation::SetSize { fd, size } => {
                let d = self.get_fd(fd, SET_SIZE)?;
                let ino = d.node;
                let n = &self.nodes[&ino];
                if n.directory {
                    return Err(Failure::Errno(ISDIR));
                }
                if n.readonly {
                    return Err(Failure::Errno(ROFS));
                }
                if self.handles.contains_key(&fd) {
                    if size > i64::MAX as u64 {
                        return Err(Failure::Errno(INVAL));
                    }
                    self.storage_resize_allowed(ino, size)?;
                    let resource = self.nodes[&ino].storage_path.clone();
                    let storage = self.storage.as_mut().unwrap();
                    let (_, errno) = storage.decision(&resource, "truncate", size, now)?;
                    if errno != 0 {
                        return Err(Failure::Errno(errno));
                    }
                    let clock = storage.clock;
                    storage
                        .fs
                        .handle_resize(&self.handles[&fd], size.cast_signed(), clock)
                        .map_err(|error| storage_failure(&error))?;
                    self.refresh_storage_node(fd)?;
                    return Ok(Output::Empty);
                }
                self.resize(ino, size)?;
                self.nodes.get_mut(&ino).unwrap().modified = now;
                Ok(Output::Empty)
            }
            Operation::Seek { fd, offset, whence } => self.seek(fd, offset, whence),
            Operation::Readdir { fd, length, cookie } => self.readdir(fd, length, cookie),
            Operation::Open(input) => self.open(&input, now),
            Operation::CreateDirectory { fd, path } => {
                self.path_operation(fd, &path, 0, CREATE_DIRECTORY, now)
            }
            Operation::RemoveDirectory { fd, path } => {
                self.path_operation(fd, &path, 0, REMOVE_DIRECTORY, now)
            }
            Operation::UnlinkFile { fd, path } => self.path_operation(fd, &path, 0, UNLINK, now),
            Operation::PathStat { fd, path, flags } => {
                self.path_operation(fd, &path, flags, PATH_STAT, now)
            }
            Operation::Rename {
                fd,
                path,
                new_fd,
                new_path,
            } => self.rename(fd, &path, new_fd, &new_path, now),
            #[cfg(any(test, feature = "conformance"))]
            Operation::Unsupported(op) => Err(Failure::Unsupported(format!(
                "operation outside stock WASI profile: {op}"
            ))),
        }
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The slice offset is below the existing Vec length and its endpoint is capped by that length."
    )]
    fn read(&mut self, fd: u32, length: u32, positional: Option<u64>, now: u64) -> Result<Output> {
        let d = self.get_fd(fd, READ | if positional.is_some() { SEEK } else { 0 })?;
        let n = &self.nodes[&d.node];
        if n.directory {
            return Err(Failure::Errno(ISDIR));
        }
        let offset = positional.unwrap_or(d.offset);
        if offset > i64::MAX as u64 {
            return Err(Failure::Errno(INVAL));
        }
        if self.handles.contains_key(&fd) {
            let resource = n.storage_path.clone();
            let storage = self.storage.as_mut().unwrap();
            let (_, errno) = storage.decision(&resource, "read", u64::from(length), now)?;
            if errno != 0 {
                return Err(Failure::Errno(errno));
            }
            let data = storage
                .fs
                .handle_read_at(&self.handles[&fd], offset.cast_signed(), length as usize)
                .map_err(|error| storage_failure(&error))?;
            if positional.is_none() {
                self.descriptors.get_mut(&fd).unwrap().offset += data.len() as u64;
            }
            return Ok(Output::Bytes(data));
        }
        let data = if offset < n.data.len() as u64 {
            n.data[offset as usize..(n.data.len() as u64).min(offset + u64::from(length)) as usize]
                .to_vec()
        } else {
            Vec::new()
        };
        if positional.is_none() {
            self.descriptors.get_mut(&fd).unwrap().offset += data.len() as u64;
        }
        Ok(Output::Bytes(data))
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Write buffers are capped at BUFFER_LIMIT and resize admits the backing Vec before offset indexing."
    )]
    fn write(&mut self, fd: u32, data: &[u8], positional: Option<u64>, now: u64) -> Result<Output> {
        if data.len() > BUFFER_LIMIT {
            return Err(Failure::Boundary(capacity("write-buffer")));
        }
        if fd == 1 || fd == 2 {
            if data.len() as u64 > self.limits.output_bytes.saturating_sub(self.offered_output) {
                return Err(Failure::Boundary(capacity("offered-output-bytes")));
            }
            self.offered_output += data.len() as u64;
        }
        let d = self
            .get_fd(fd, WRITE | if positional.is_some() { SEEK } else { 0 })?
            .clone();
        let n = &self.nodes[&d.node];
        if n.directory {
            return Err(Failure::Errno(ISDIR));
        }
        if n.readonly {
            return Err(Failure::Errno(ROFS));
        }
        if d.std == 2 {
            self.stdout.extend_from_slice(data);
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            {
                self.output_committed[0] += data.len() as u64;
            }
            #[cfg(feature = "determinism-fixture")]
            crate::determinism_fixture::output_committed();
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            crate::event_order::emit_current(
                "output",
                self.output_committed[0],
                now.cast_signed(),
                serde_json::json!({"stream": "stdout", "data": crate::event_order::bytes_value(data)}),
            );
            return Ok(Output::Written(data.len() as u32));
        }
        if d.std == 3 {
            self.stderr.extend_from_slice(data);
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            {
                self.output_committed[1] += data.len() as u64;
            }
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            crate::event_order::emit_current(
                "output",
                self.output_committed[1],
                now.cast_signed(),
                serde_json::json!({"stream": "stderr", "data": crate::event_order::bytes_value(data)}),
            );
            return Ok(Output::Written(data.len() as u32));
        }
        let offset = if d.flags & 1 != 0 {
            node_size(n)
        } else {
            positional.unwrap_or(d.offset)
        };
        if offset > i64::MAX as u64 || data.len() as u64 > i64::MAX as u64 - offset {
            return Err(Failure::Errno(INVAL));
        }
        if self.handles.contains_key(&fd) {
            let (written, errno) = self.storage_write(fd, d.node, data, offset, now)?;
            self.refresh_storage_node(fd)?;
            if errno != 0 {
                return Err(Failure::Errno(errno));
            }
            if positional.is_none() {
                self.descriptors.get_mut(&fd).unwrap().offset = offset + u64::from(written);
            }
            return Ok(Output::Written(written));
        }
        let size = if data.is_empty() {
            n.data.len() as u64
        } else {
            (offset + data.len() as u64).max(n.data.len() as u64)
        };
        self.resize(d.node, size)?;
        if !data.is_empty() {
            let n = self.nodes.get_mut(&d.node).unwrap();
            n.data[offset as usize..offset as usize + data.len()].copy_from_slice(data);
            n.modified = now;
        }
        if positional.is_none() {
            self.descriptors.get_mut(&fd).unwrap().offset = offset + data.len() as u64;
        }
        Ok(Output::Written(data.len() as u32))
    }
    fn resize(&mut self, ino: u64, size: u64) -> Result<()> {
        let old = self.nodes[&ino].data.len() as u64;
        if size > self.limits.filesystem_bytes
            || self.bytes - old > self.limits.filesystem_bytes - size
        {
            return Err(Failure::Boundary(capacity("filesystem-bytes")));
        }
        let length =
            usize::try_from(size).map_err(|_| Failure::Boundary(capacity("filesystem-bytes")))?;
        self.nodes.get_mut(&ino).unwrap().data.resize(length, 0);
        self.bytes = self.bytes - old + size;
        Ok(())
    }
    fn seek(&mut self, fd: u32, offset: i64, whence: u8) -> Result<Output> {
        let d = self.get_fd(fd, SEEK)?;
        if self.nodes[&d.node].directory || d.std != 0 {
            return Err(Failure::Errno(INVAL));
        }
        let base = match whence {
            0 => 0,
            1 => d.offset,
            2 => node_size(&self.nodes[&d.node]),
            _ => return Err(Failure::Errno(INVAL)),
        };
        let position = if offset >= 0 {
            if base > i64::MAX as u64 - offset.cast_unsigned() {
                return Err(Failure::Errno(INVAL));
            }
            base + offset.cast_unsigned()
        } else {
            base.checked_sub(offset.unsigned_abs())
                .ok_or(Failure::Errno(INVAL))?
        };
        self.descriptors.get_mut(&fd).unwrap().offset = position;
        Ok(Output::Offset(position))
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "Cookies are checked against the child count, and names come from bounded image or guest paths."
    )]
    fn readdir(&self, fd: u32, length: u32, cookie: u64) -> Result<Output> {
        let d = self.get_fd(fd, READDIR)?;
        let n = &self.nodes[&d.node];
        if !n.directory {
            return Err(Failure::Errno(NOTDIR));
        }
        if cookie > n.children.len() as u64 {
            return Err(Failure::Errno(INVAL));
        }
        let mut data = Vec::new();
        for (i, (name, ino)) in n.children.iter().enumerate().skip(cookie as usize) {
            if data.len() >= length as usize {
                break;
            }
            let mut entry = Vec::with_capacity(24 + name.len());
            entry.extend_from_slice(&(i as u64 + 1).to_le_bytes());
            entry.extend_from_slice(&ino.to_le_bytes());
            entry.extend_from_slice(&(name.len() as u32).to_le_bytes());
            entry.push(node_type(&self.nodes[ino]));
            entry.extend_from_slice(&[0; 3]);
            entry.extend_from_slice(name);
            data.extend_from_slice(&entry[..entry.len().min(length as usize - data.len())]);
        }
        Ok(Output::Bytes(data))
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Path opening preserves the Go model\'s ordered permission, capacity and storage checks."
    )]
    fn open(&mut self, input: &Open, now: u64) -> Result<Output> {
        let d = self.get_fd(input.fd, PATH_OPEN)?;
        if input.dirflags & !1 != 0
            || input.oflags & !15 != 0
            || input.base & !RIGHTS_ALL != 0
            || input.inheriting & !RIGHTS_ALL != 0
        {
            return Err(Failure::Errno(INVAL));
        }
        if input.flags & !(1 | 4) != 0 {
            return Err(Failure::Unsupported(OPEN_FLAGS_UNSUPPORTED.into()));
        }
        if input.oflags & 3 == 3 {
            return Err(Failure::Errno(INVAL));
        }
        if input.oflags & 1 != 0 && d.rights & CREATE_FILE == 0
            || input.oflags & 8 != 0 && d.rights & PATH_SET_SIZE == 0
        {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        if input.base & d.inheriting != input.base
            || input.inheriting & d.inheriting != input.inheriting
        {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        let (parent, name) = self.resolve_path(d, &input.path)?;
        let node = self.child_at(parent, &name);
        if node.is_none() && input.path.ends_with(b"/") {
            return Err(Failure::Errno(NOTDIR));
        }
        if node.is_some() && input.oflags & 5 == 5 {
            return Err(Failure::Errno(EXIST));
        }
        if node.is_none() && input.oflags & 1 == 0 {
            return Err(Failure::Errno(NOENT));
        }
        if node.is_some_and(|ino| input.oflags & 2 != 0 && !self.nodes[&ino].directory) {
            return Err(Failure::Errno(NOTDIR));
        }
        if node.is_some_and(|ino| self.nodes[&ino].directory && input.oflags & 8 != 0) {
            return Err(Failure::Errno(ISDIR));
        }
        if self.nodes[&parent].readonly
            && (node.is_none() || input.oflags & 8 != 0 || input.base & WRITE != 0)
        {
            return Err(Failure::Errno(ROFS));
        }
        if input.oflags & 8 != 0 && input.base & SET_SIZE == 0 {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        if self.descriptors.len() as u64 >= self.limits.descriptors || self.next_fd == u32::MAX {
            return Err(Failure::Boundary(capacity("descriptors")));
        }
        if node.is_none() && self.files == self.limits.files {
            return Err(Failure::Boundary(capacity("files")));
        }
        let resource = self.storage_path(parent, &name)?;
        let mut handle = None;
        if !resource.is_empty() {
            let storage = self.storage.as_mut().unwrap();
            let (_, errno) = storage.decision(&resource, "open", 0, now)?;
            if errno != 0 {
                return Err(Failure::Errno(errno));
            }
            let flags = volume::OpenFlags {
                read: input.base & READ != 0 || input.oflags & 2 != 0,
                write: input.base & WRITE != 0,
                append: false,
                create: input.oflags & 1 != 0,
                exclusive: input.oflags & 4 != 0,
                truncate: input.oflags & 8 != 0,
            };
            handle = Some(
                storage
                    .open(&resource, flags)
                    .map_err(|error| storage_failure(&error))?,
            );
        }
        let ino = if let Some(ino) = node {
            ino
        } else {
            let ino = self.new_node(false, false);
            let created = self.nodes.get_mut(&ino).unwrap();
            created.modified = now;
            created.storage_path = resource;
            self.nodes
                .get_mut(&parent)
                .unwrap()
                .children
                .insert(name, ino);
            ino
        };
        if input.oflags & 8 != 0 && handle.is_none() {
            self.resize(ino, 0)?;
            self.nodes.get_mut(&ino).unwrap().modified = now;
        }
        let fd = self.next_fd;
        self.next_fd += 1;
        self.descriptors.insert(
            fd,
            Descriptor {
                node: ino,
                offset: 0,
                rights: input.base,
                inheriting: input.inheriting,
                flags: input.flags,
                preopen: Vec::new(),
                std: 0,
            },
        );
        if let Some(handle) = handle {
            self.handles.insert(fd, handle);
            self.refresh_storage_node(fd)?;
        }
        Ok(Output::Fd(fd))
    }
    fn path_operation(
        &mut self,
        fd: u32,
        path: &[u8],
        flags: u32,
        right: u64,
        now: u64,
    ) -> Result<Output> {
        let d = self.get_fd(fd, right)?;
        if flags & !1 != 0 {
            return Err(Failure::Errno(INVAL));
        }
        let (parent, name) = self.resolve_path(d, path)?;
        let node = self.child_at(parent, &name);
        if right == PATH_STAT {
            return Ok(Output::Filestat(
                self.stat(node.ok_or(Failure::Errno(NOENT))?),
            ));
        }
        if self.nodes[&parent].readonly || node.is_some_and(|ino| self.nodes[&ino].readonly) {
            return Err(Failure::Errno(ROFS));
        }
        if right == CREATE_DIRECTORY {
            if node.is_some() {
                return Err(Failure::Errno(EXIST));
            }
            if self.files == self.limits.files {
                return Err(Failure::Boundary(capacity("files")));
            }
            let resource = self.storage_path(parent, &name)?;
            if !resource.is_empty() {
                let storage = self.storage.as_mut().unwrap();
                let (_, errno) = storage.decision(&resource, "mkdir", 0, now)?;
                if errno != 0 {
                    return Err(Failure::Errno(errno));
                }
                let clock = storage.clock;
                storage
                    .fs
                    .mkdir(&resource, 0o777, clock)
                    .map_err(|error| storage_failure(&error))?;
            }
            let ino = self.new_node(true, false);
            self.nodes.get_mut(&ino).unwrap().storage_path = resource;
            self.nodes
                .get_mut(&parent)
                .unwrap()
                .children
                .insert(name, ino);
            return Ok(Output::Empty);
        }
        let ino = node.ok_or(Failure::Errno(NOENT))?;
        if name.is_empty() {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        let n = &self.nodes[&ino];
        if right == REMOVE_DIRECTORY {
            if !n.directory {
                return Err(Failure::Errno(NOTDIR));
            }
            if !n.children.is_empty() {
                return Err(Failure::Errno(NOTEMPTY));
            }
        } else if n.directory {
            return Err(Failure::Errno(ISDIR));
        }
        if !n.storage_path.is_empty() {
            let resource = n.storage_path.clone();
            let storage = self.storage.as_mut().unwrap();
            let (_, errno) = storage.decision(&resource, "remove", 0, now)?;
            if errno != 0 {
                return Err(Failure::Errno(errno));
            }
            storage
                .fs
                .remove(&resource)
                .map_err(|error| storage_failure(&error))?;
        }
        self.nodes.get_mut(&parent).unwrap().children.remove(&name);
        Ok(Output::Empty)
    }
    fn rename(
        &mut self,
        fd: u32,
        path: &[u8],
        new_fd: u32,
        new_path: &[u8],
        now: u64,
    ) -> Result<Output> {
        let source = self.get_fd(fd, RENAME_SOURCE)?;
        let target = self.get_fd(new_fd, RENAME_TARGET)?;
        let (parent, name) = self.resolve_path(source, path)?;
        let (new_parent, new_name) = self.resolve_path(target, new_path)?;
        let ino = self.child_at(parent, &name).ok_or(Failure::Errno(NOENT))?;
        let n = &self.nodes[&ino];
        if new_path.ends_with(b"/") && !n.directory {
            return Err(Failure::Errno(NOTDIR));
        }
        if name.is_empty() || new_name.is_empty() {
            return Err(Failure::Errno(NOTCAPABLE));
        }
        let old = self.nodes[&new_parent].children.get(&new_name).copied();
        if self.nodes[&parent].readonly
            || self.nodes[&new_parent].readonly
            || n.readonly
            || old.is_some_and(|old| self.nodes[&old].readonly)
        {
            return Err(Failure::Errno(ROFS));
        }
        if parent == new_parent && name == new_name {
            return Ok(Output::Empty);
        }
        if n.directory && self.contains_node(ino, new_parent) {
            return Err(Failure::Errno(INVAL));
        }
        if let Some(old) = old {
            let old = &self.nodes[&old];
            if old.directory != n.directory {
                return Err(Failure::Errno(if old.directory { ISDIR } else { NOTDIR }));
            }
            if old.directory && !old.children.is_empty() {
                return Err(Failure::Errno(NOTEMPTY));
            }
        }
        if !n.storage_path.is_empty() || !self.nodes[&new_parent].storage_path.is_empty() {
            let source = n.storage_path.clone();
            let resource = self.storage_path(new_parent, &new_name)?;
            if source.is_empty() || resource.is_empty() {
                return Err(Failure::Errno(EXDEV));
            }
            let storage = self.storage.as_mut().unwrap();
            let (_, errno) = storage.decision(&source, "rename", 0, now)?;
            if errno != 0 {
                return Err(Failure::Errno(errno));
            }
            let clock = storage.clock;
            storage
                .fs
                .rename(&source, &resource, clock)
                .map_err(|error| storage_failure(&error))?;
            self.relocate_storage_node(ino, &resource);
        }
        self.nodes.get_mut(&parent).unwrap().children.remove(&name);
        self.nodes
            .get_mut(&new_parent)
            .unwrap()
            .children
            .insert(new_name, ino);
        Ok(Output::Empty)
    }
    fn contains_node(&self, root: u64, query: u64) -> bool {
        root == query
            || self.nodes[&root]
                .children
                .values()
                .any(|ino| self.nodes[ino].directory && self.contains_node(*ino, query))
    }
    pub fn fd_readiness(&self, fd: u32, write: bool) -> FdReadiness {
        let mut event = FdReadiness {
            errno: 0,
            nbytes: 0,
            flags: 0,
        };
        match self.get_fd(fd, POLL | if write { WRITE } else { READ }) {
            Ok(d) => {
                let n = &self.nodes[&d.node];
                if n.directory {
                    event.errno = INVAL;
                } else if !write {
                    if d.offset < node_size(n) {
                        event.nbytes = node_size(n) - d.offset;
                    } else {
                        event.flags = 1;
                    }
                }
            }
            Err(Failure::Errno(errno)) => event.errno = errno,
            _ => unreachable!(),
        }
        event
    }
    pub fn canonical_state(&self) -> Vec<u8> {
        let mut bytes = b"gomad.namespace-conformance/v1\0".to_vec();
        let mut seen = BTreeSet::new();
        self.encode_tree(1, b"/", &mut seen, &mut bytes);
        let mut orphans = BTreeSet::new();
        for d in self.descriptors.values() {
            self.collect_orphans(d.node, &seen, &mut orphans);
        }
        for ino in orphans {
            bytes.push(b'O');
            self.encode_node(ino, &mut bytes);
            let children = &self.nodes[&ino].children;
            put_u64(&mut bytes, children.len() as u64);
            for (name, child) in children {
                put_bytes(&mut bytes, name);
                put_u64(&mut bytes, *child);
            }
        }
        for (fd, d) in &self.descriptors {
            bytes.push(b'D');
            for number in [
                u64::from(*fd),
                d.node,
                d.offset,
                d.rights,
                d.inheriting,
                u64::from(d.flags),
            ] {
                put_u64(&mut bytes, number);
            }
            put_bytes(&mut bytes, &d.preopen);
            put_u64(&mut bytes, u64::from(d.std));
        }
        bytes.push(b'S');
        for number in [
            u64::from(self.next_fd),
            self.next_inode,
            self.files,
            self.bytes,
            self.offered_output(),
        ] {
            put_u64(&mut bytes, number);
        }
        put_bytes(&mut bytes, &self.stdout);
        put_bytes(&mut bytes, &self.stderr);
        bytes
    }
    /// Every public `&mut self` entry point calls this before it can change
    /// anything [`Self::inspection_material`] encodes; fields are private.
    fn touch(&mut self) {
        self.inspection_revision.1 += 1;
    }
    /// Equal values denote identical [`Self::inspection_material`].
    #[cfg(not(feature = "conformance"))]
    pub fn inspection_revision(&self) -> (u64, u64) {
        self.inspection_revision
    }
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn diagnostic_value(&mut self) -> wasmtime::Result<serde_json::Value> {
        let revision = self.inspection_revision();
        if self
            .event_state_cache
            .as_ref()
            .is_none_or(|(cached, _)| *cached != revision)
        {
            let (state, recovery, log) = self.inspection_material()?;
            self.event_state_cache = Some((
                revision,
                serde_json::json!({"model":"namespace-volume","state":crate::event_order::bytes_value(&state),"recovery":crate::event_order::bytes_value(&recovery),"storage_log":crate::event_order::bytes_value(&log)}),
            ));
        }
        Ok(self.event_state_cache.as_ref().unwrap().1.clone())
    }
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    pub fn diagnostic_state(&mut self, now: Option<u64>) {
        if !crate::event_order::is_enabled() {
            return;
        }
        let Ok(value) = self.diagnostic_value() else {
            crate::event_order::incomplete("namespace or volume diagnostic state capacity");
            return;
        };
        let ordinal = self.event_mutations;
        self.event_mutations += 1;
        if let Some(now) = now {
            crate::event_order::emit_current("model-transition", ordinal, now.cast_signed(), value);
        } else {
            crate::event_order::emit_current_context("model-transition", ordinal, value);
        }
    }
    #[cfg(not(feature = "conformance"))]
    pub fn inspection_material(&self) -> wasmtime::Result<(Vec<u8>, Vec<u8>, Vec<u8>)> {
        const MAXIMUM: u64 = 8 << 20;
        if self.bytes > MAXIMUM || self.nodes.len() > 65536 {
            return Err(wasmtime::Error::msg("inspection namespace state capacity"));
        }
        let mut state = self.canonical_state();
        let mut recovery = Vec::new();
        let mut log = Vec::new();
        if let Some(storage) = &self.storage {
            let storage_state = serde_json::to_vec(&(
                storage.snapshots(),
                storage.operations(),
                &storage.occurrences,
                storage.clock,
            ))
            .map_err(wasmtime::Error::new)?;
            if storage_state.len() as u64 > MAXIMUM {
                return Err(wasmtime::Error::msg("inspection volume state capacity"));
            }
            put_bytes(&mut state, &storage_state);
            recovery = volume::RecoveryImage::encode(
                &storage.fs,
                &storage.model_identity,
                &storage.occurrences,
                MAXIMUM,
            )
            .map_err(|error| {
                wasmtime::Error::msg(format!("inspection recovery state: {error:?}"))
            })?;
            log.clone_from(&storage.log);
        }
        Ok((state, recovery, log))
    }
    fn collect_orphans(&self, ino: u64, seen: &BTreeSet<u64>, orphans: &mut BTreeSet<u64>) {
        if seen.contains(&ino) || !orphans.insert(ino) {
            return;
        }
        for child in self.nodes[&ino].children.values() {
            self.collect_orphans(*child, seen, orphans);
        }
    }
    fn encode_tree(&self, ino: u64, path: &[u8], seen: &mut BTreeSet<u64>, bytes: &mut Vec<u8>) {
        seen.insert(ino);
        bytes.push(b'N');
        put_bytes(bytes, path);
        self.encode_node(ino, bytes);
        for (name, child) in &self.nodes[&ino].children {
            let mut child_path = path.to_vec();
            if path != b"/" {
                child_path.push(b'/');
            }
            child_path.extend_from_slice(name);
            self.encode_tree(*child, &child_path, seen, bytes);
        }
    }
    fn encode_node(&self, ino: u64, bytes: &mut Vec<u8>) {
        let n = &self.nodes[&ino];
        put_u64(bytes, n.ino);
        bytes.push(u8::from(n.directory));
        bytes.push(u8::from(n.readonly));
        put_u64(bytes, node_size(n));
        put_bytes(bytes, &n.data);
        put_u64(bytes, n.modified);
    }
    /// Whether `fd` is still the original guest stdout/stderr descriptor.
    #[cfg_attr(feature = "conformance", allow(dead_code))]
    pub fn standard_output(&self, fd: u32) -> bool {
        matches!(fd, 1 | 2) && self.descriptors.get(&fd).is_some_and(|d| d.std == fd + 1)
    }
    pub fn offered_output(&self) -> u64 {
        self.offered_output
    }
    #[cfg(not(feature = "conformance"))]
    pub fn adopt_offered_output(&mut self, offered: u64) {
        assert!(offered >= self.offered_output && offered <= self.limits.output_bytes);
        if offered != self.offered_output {
            self.touch();
        }
        self.offered_output = offered;
    }
    #[cfg_attr(all(feature = "conformance", not(test)), allow(dead_code))]
    pub fn take_output(&mut self) -> (Vec<u8>, Vec<u8>) {
        if !self.stdout.is_empty() || !self.stderr.is_empty() {
            self.touch();
        }
        (
            std::mem::take(&mut self.stdout),
            std::mem::take(&mut self.stderr),
        )
    }
}
fn node_type(node: &Node) -> u8 {
    if node.directory { 3 } else { 4 }
}
/// Go `nodeSize`: volume nodes report their volume size.
fn node_size(node: &Node) -> u64 {
    if node.storage_path.is_empty() {
        node.data.len() as u64
    } else {
        node.storage_size
    }
}
fn capacity(dimension: &'static str) -> Boundary {
    Boundary {
        kind: "capacity",
        dimension,
    }
}
fn invalid(dimension: &'static str) -> Boundary {
    Boundary {
        kind: "invalid",
        dimension,
    }
}
fn valid_image_path(path: &[u8]) -> bool {
    path == b"/"
        || path.first() == Some(&b'/')
            && !path.contains(&0)
            && path[1..]
                .split(|b| *b == b'/')
                .all(|name| !name.is_empty() && name != b"." && name != b"..")
}
fn put_u64(bytes: &mut Vec<u8>, value: u64) {
    bytes.extend_from_slice(&value.to_le_bytes());
}
fn put_bytes(bytes: &mut Vec<u8>, value: &[u8]) {
    put_u64(bytes, value.len() as u64);
    bytes.extend_from_slice(value);
}

#[cfg(test)]
mod tests {
    use super::*;

    fn image() -> Image {
        Image {
            entries: vec![ImageEntry {
                path: b"/tmp".to_vec(),
                directory: true,
                readonly: false,
                data: vec![],
            }],
            writable_directories: vec![b"/tmp".to_vec()],
            preopens: vec![b"/tmp".to_vec()],
            working_directory: b"/tmp".to_vec(),
            stdin: b"input".to_vec(),
            limits: Limits {
                files: 100,
                descriptors: 100,
                filesystem_bytes: 100,
                output_bytes: 100,
            },
            epoch_nanos: 1000,
        }
    }
    fn ns() -> Namespace {
        Namespace::new(image()).unwrap()
    }
    #[cfg(not(feature = "conformance"))]
    fn open(fd: u32, name: &[u8], oflags: u16, base: u64) -> Operation {
        Operation::Open(Open {
            fd,
            path: name.to_vec(),
            dirflags: 0,
            oflags,
            base,
            inheriting: 0,
            flags: 0,
        })
    }
    fn run(ns: &mut Namespace, op: Operation) -> Reply {
        ns.execute(op, 7).unwrap()
    }
    #[cfg(not(feature = "conformance"))]
    fn opened(ns: &mut Namespace, name: &[u8]) -> u32 {
        match run(ns, open(3, name, 1, RIGHTS_ALL)).output.unwrap() {
            Output::Fd(fd) => fd,
            other => panic!("{other:?}"),
        }
    }

    #[test]
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn diagnostic_output_ordinals_survive_draining_each_stream() {
        let mut n = Namespace::new(image()).unwrap();
        for fd in [1, 2, 1, 2] {
            run(
                &mut n,
                Operation::Write {
                    fd,
                    data: b"four".to_vec(),
                    offset: None,
                },
            );
            let _ = n.take_output();
        }
        assert_eq!(n.output_committed, [8, 8]);
    }

    #[test]
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn diagnostic_namespace_state_tracks_mutations_and_caches_by_revision() {
        let mut n = Namespace::new(image()).unwrap();
        let before = n.diagnostic_value().unwrap();
        let cached = n.event_state_cache.clone();
        assert_eq!(n.diagnostic_value().unwrap(), before);
        assert_eq!(n.event_state_cache, cached);
        run(&mut n, Operation::FdstatGet(1));
        assert_eq!(
            n.event_state_cache, cached,
            "read-only metadata must not walk the namespace"
        );
        run(
            &mut n,
            Operation::Write {
                fd: 1,
                data: b"changed".to_vec(),
                offset: None,
            },
        );
        assert_ne!(n.diagnostic_value().unwrap(), before);
        assert_ne!(n.event_state_cache.as_ref().unwrap().0, cached.unwrap().0);
    }

    #[test]
    fn drained_output_keeps_charge_and_cannot_be_taken_twice() {
        let mut input = image();
        input.limits.output_bytes = 3;
        let mut n = Namespace::new(input).unwrap();
        run(
            &mut n,
            Operation::Write {
                fd: 1,
                data: b"ab".to_vec(),
                offset: None,
            },
        );
        run(
            &mut n,
            Operation::Write {
                fd: 2,
                data: b"x".to_vec(),
                offset: None,
            },
        );
        let charged = n.offered_output();
        assert_ne!(charged, 0);
        let drained = n.take_output();
        assert!(!drained.0.is_empty() && !drained.1.is_empty());
        assert_eq!(n.take_output(), (Vec::new(), Vec::new()));
        assert_eq!(n.offered_output(), charged);
        assert!(
            n.execute(
                Operation::Write {
                    fd: 1,
                    data: b"y".to_vec(),
                    offset: None,
                },
                7,
            )
            .is_err()
        );
        assert_eq!(n.offered_output(), charged);
        assert_eq!(n.take_output(), (Vec::new(), Vec::new()));
    }

    #[test]
    fn oversized_typed_write_refuses_before_output_accounting() {
        let mut n = ns();
        assert_eq!(
            n.execute(
                Operation::Write {
                    fd: 1,
                    data: vec![0; BUFFER_LIMIT + 1],
                    offset: None,
                },
                7,
            )
            .unwrap_err()
            .dimension(),
            "write-buffer"
        );
        assert_eq!(n.offered_output(), 0);
    }

    #[test]
    fn constructor_checks_preopen_capacity_below_public_minimum() {
        let mut input = image();
        input.limits.descriptors = 3;
        assert_eq!(
            Namespace::new(input).err().unwrap().dimension,
            "preopen-descriptors"
        );
    }

    /// Every public mutator advances the inspection revision before it can
    /// change the material; reads and no-op output handling do not.
    #[test]
    #[cfg(not(feature = "conformance"))]
    fn inspection_revision_advances_on_every_mutating_entry_point() {
        let mut n = ns();
        let other = ns();
        assert_ne!(n.inspection_revision().0, other.inspection_revision().0);
        let mut revision = n.inspection_revision();
        let mut advanced = |n: &Namespace, expected: bool| {
            let current = n.inspection_revision();
            assert_eq!(current != revision, expected);
            assert_eq!(current.0, revision.0);
            revision = current;
        };
        let material = n.inspection_material().unwrap();
        n.canonical_state();
        advanced(&n, false);
        assert_eq!(n.take_output(), (Vec::new(), Vec::new()));
        advanced(&n, false);
        n.adopt_offered_output(n.offered_output());
        advanced(&n, false);
        run(&mut n, Operation::FilestatGet(3));
        advanced(&n, true);
        assert_eq!(n.inspection_material().unwrap(), material);
        let fd = opened(&mut n, b"f");
        advanced(&n, true);
        run(
            &mut n,
            Operation::Write {
                fd,
                data: b"x".to_vec(),
                offset: None,
            },
        );
        advanced(&n, true);
        assert_ne!(n.inspection_material().unwrap(), material);
        run(
            &mut n,
            Operation::Write {
                fd: 1,
                data: b"o".to_vec(),
                offset: None,
            },
        );
        advanced(&n, true);
        n.adopt_offered_output(n.offered_output() + 1);
        advanced(&n, true);
        assert_eq!(n.take_output().0, b"o");
        advanced(&n, true);

        let mut storage_image = image();
        storage_image.entries[0].path = b"/workspace".to_vec();
        storage_image.writable_directories = vec![b"/workspace".to_vec()];
        storage_image.preopens = vec![b"/workspace".to_vec()];
        storage_image.working_directory = b"/workspace".to_vec();
        let mut n = Namespace::new(storage_image).unwrap();
        let mut revision = n.inspection_revision();
        let mut advanced = |n: &Namespace| {
            let current = n.inspection_revision();
            assert_ne!(current, revision);
            assert_eq!(current.0, revision.0);
            revision = current;
        };
        let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1 << 20}],
            "limits":{"PendingOperations":1000,"Transitions":100_000}
        }))
        .unwrap();
        n.attach_storage(crate::volume::Storage::new(model, 1 << 24).unwrap())
            .unwrap();
        advanced(&n);
        n.crash_storage(&BTreeMap::new(), 1 << 20).unwrap();
        advanced(&n);
    }
}
