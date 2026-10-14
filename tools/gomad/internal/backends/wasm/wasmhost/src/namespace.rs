use std::collections::{BTreeMap, BTreeSet};

pub const BADF: u16 = 8;
pub const EXIST: u16 = 20;
pub const INVAL: u16 = 28;
pub const ISDIR: u16 = 31;
pub const NOENT: u16 = 44;
pub const NOTDIR: u16 = 54;
pub const NOTEMPTY: u16 = 55;
pub const ROFS: u16 = 69;
pub const NOTCAPABLE: u16 = 76;
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
const BUFFER_LIMIT: usize = 4 << 20;

#[derive(Clone, Debug)]
pub struct Limits {
    pub files: u64,
    pub descriptors: u64,
    pub filesystem_bytes: u64,
    pub output_bytes: u64,
}
#[derive(Clone, Debug)]
pub struct ImageEntry {
    pub path: Vec<u8>,
    pub directory: bool,
    pub readonly: bool,
    pub data: Vec<u8>,
}
#[derive(Clone, Debug)]
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
}
impl Namespace {
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
            size: n.data.len() as u64,
            atim: timestamp,
            mtim: timestamp,
            ctim: timestamp,
        }
    }
    pub fn execute(&mut self, op: Operation, now: u64) -> std::result::Result<Reply, Boundary> {
        match self.execute_inner(op, now) {
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
            Err(Failure::Boundary(boundary)) => Err(boundary),
        }
    }
    fn execute_inner(&mut self, op: Operation, now: u64) -> Result<Output> {
        match op {
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
                self.descriptors.remove(&fd);
                Ok(Output::Empty)
            }
            Operation::Sync(fd) => {
                self.get_fd(fd, SYNC)?;
                Ok(Output::Empty)
            }
            Operation::SetFlags { fd, flags } => {
                self.get_fd(fd, SET_FLAGS)?;
                if flags & !(1 | 4) != 0 {
                    return Err(Failure::Unsupported(
                        "descriptor flags outside volatile profile".into(),
                    ));
                }
                self.descriptors.get_mut(&fd).unwrap().flags = flags;
                Ok(Output::Empty)
            }
            Operation::Read { fd, length, offset } => self.read(fd, length, offset),
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
                self.resize(ino, size)?;
                self.nodes.get_mut(&ino).unwrap().modified = now;
                Ok(Output::Empty)
            }
            Operation::Seek { fd, offset, whence } => self.seek(fd, offset, whence),
            Operation::Readdir { fd, length, cookie } => self.readdir(fd, length, cookie),
            Operation::Open(input) => self.open(input, now),
            Operation::CreateDirectory { fd, path } => {
                self.path_operation(fd, &path, 0, CREATE_DIRECTORY)
            }
            Operation::RemoveDirectory { fd, path } => {
                self.path_operation(fd, &path, 0, REMOVE_DIRECTORY)
            }
            Operation::UnlinkFile { fd, path } => self.path_operation(fd, &path, 0, UNLINK),
            Operation::PathStat { fd, path, flags } => {
                self.path_operation(fd, &path, flags, PATH_STAT)
            }
            Operation::Rename {
                fd,
                path,
                new_fd,
                new_path,
            } => self.rename(fd, &path, new_fd, &new_path),
            Operation::Unsupported(op) => Err(Failure::Unsupported(format!(
                "operation outside stock WASI profile: {op}"
            ))),
        }
    }
    fn read(&mut self, fd: u32, length: u32, positional: Option<u64>) -> Result<Output> {
        let d = self.get_fd(fd, READ | if positional.is_some() { SEEK } else { 0 })?;
        let n = &self.nodes[&d.node];
        if n.directory {
            return Err(Failure::Errno(ISDIR));
        }
        let offset = positional.unwrap_or(d.offset);
        if offset > i64::MAX as u64 {
            return Err(Failure::Errno(INVAL));
        }
        let data = if offset < n.data.len() as u64 {
            n.data[offset as usize..(n.data.len() as u64).min(offset + length as u64) as usize]
                .to_vec()
        } else {
            Vec::new()
        };
        if positional.is_none() {
            self.descriptors.get_mut(&fd).unwrap().offset += data.len() as u64;
        }
        Ok(Output::Bytes(data))
    }
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
            return Ok(Output::Written(data.len() as u32));
        }
        if d.std == 3 {
            self.stderr.extend_from_slice(data);
            return Ok(Output::Written(data.len() as u32));
        }
        let offset = if d.flags & 1 != 0 {
            n.data.len() as u64
        } else {
            positional.unwrap_or(d.offset)
        };
        if offset > i64::MAX as u64 || data.len() as u64 > i64::MAX as u64 - offset {
            return Err(Failure::Errno(INVAL));
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
            2 => self.nodes[&d.node].data.len() as u64,
            _ => return Err(Failure::Errno(INVAL)),
        };
        let position = if offset >= 0 {
            if base > i64::MAX as u64 - offset as u64 {
                return Err(Failure::Errno(INVAL));
            }
            base + offset as u64
        } else {
            base.checked_sub(offset.unsigned_abs())
                .ok_or(Failure::Errno(INVAL))?
        };
        self.descriptors.get_mut(&fd).unwrap().offset = position;
        Ok(Output::Offset(position))
    }
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
    fn open(&mut self, input: Open, now: u64) -> Result<Output> {
        let d = self.get_fd(input.fd, PATH_OPEN)?;
        if input.dirflags & !1 != 0
            || input.oflags & !15 != 0
            || input.base & !RIGHTS_ALL != 0
            || input.inheriting & !RIGHTS_ALL != 0
        {
            return Err(Failure::Errno(INVAL));
        }
        if input.flags & !(1 | 4) != 0 {
            return Err(Failure::Unsupported(
                "open flags outside volatile profile".into(),
            ));
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
        let ino = match node {
            Some(ino) => ino,
            None => {
                let ino = self.new_node(false, false);
                self.nodes.get_mut(&ino).unwrap().modified = now;
                self.nodes
                    .get_mut(&parent)
                    .unwrap()
                    .children
                    .insert(name, ino);
                ino
            }
        };
        if input.oflags & 8 != 0 {
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
        Ok(Output::Fd(fd))
    }
    fn path_operation(&mut self, fd: u32, path: &[u8], flags: u32, right: u64) -> Result<Output> {
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
            let ino = self.new_node(true, false);
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
        self.nodes.get_mut(&parent).unwrap().children.remove(&name);
        Ok(Output::Empty)
    }
    fn rename(&mut self, fd: u32, path: &[u8], new_fd: u32, new_path: &[u8]) -> Result<Output> {
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
                    if d.offset < n.data.len() as u64 {
                        event.nbytes = n.data.len() as u64 - d.offset;
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
        let mut bytes = b"gomad3.namespace-conformance/v1\0".to_vec();
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
                *fd as u64,
                d.node,
                d.offset,
                d.rights,
                d.inheriting,
                d.flags as u64,
            ] {
                put_u64(&mut bytes, number);
            }
            put_bytes(&mut bytes, &d.preopen);
            put_u64(&mut bytes, d.std as u64);
        }
        bytes.push(b'S');
        for number in [
            self.next_fd as u64,
            self.next_inode,
            self.files,
            self.bytes,
            self.offered_output(),
        ] {
            put_u64(&mut bytes, number);
        }
        put_bytes(&mut bytes, self.stdout());
        put_bytes(&mut bytes, self.stderr());
        bytes
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
        bytes.push(n.directory as u8);
        bytes.push(n.readonly as u8);
        put_u64(bytes, n.data.len() as u64);
        put_bytes(bytes, &n.data);
        put_u64(bytes, n.modified);
    }
    pub fn stdout(&self) -> &[u8] {
        &self.stdout
    }
    pub fn stderr(&self) -> &[u8] {
        &self.stderr
    }
    pub fn offered_output(&self) -> u64 {
        self.offered_output
    }
}
fn node_type(node: &Node) -> u8 {
    if node.directory { 3 } else { 4 }
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
    fn opened(ns: &mut Namespace, name: &[u8]) -> u32 {
        match run(ns, open(3, name, 1, RIGHTS_ALL)).output.unwrap() {
            Output::Fd(fd) => fd,
            other => panic!("{other:?}"),
        }
    }
    fn bytes(reply: Reply) -> Vec<u8> {
        match reply.output.unwrap() {
            Output::Bytes(data) => data,
            other => panic!("{other:?}"),
        }
    }

    #[test]
    fn path_containment_precedes_missing_components() {
        let mut n = ns();
        for path in [b"../x".as_slice(), b"/x", b"missing/../../x", b"x\0y"] {
            assert_eq!(run(&mut n, open(3, path, 0, READ)).errno, NOTCAPABLE);
        }
        assert_eq!(run(&mut n, open(3, b"missing/../x", 0, READ)).errno, NOENT);
        opened(&mut n, b"file");
        for path in [b"file/".as_slice(), b"file/../x", b"file/."] {
            assert_eq!(run(&mut n, open(3, path, 0, READ)).errno, NOTDIR);
        }
    }
    #[test]
    fn open_checks_descriptor_flags_and_create_rights_in_reference_order() {
        let mut n = ns();
        let mut input = Open {
            fd: 999,
            path: b"../x".to_vec(),
            dirflags: 2,
            oflags: 3,
            base: RIGHTS_ALL,
            inheriting: 0,
            flags: 2,
        };
        assert_eq!(run(&mut n, Operation::Open(input.clone())).errno, BADF);
        input.fd = 3;
        assert_eq!(run(&mut n, Operation::Open(input.clone())).errno, INVAL);
        input.dirflags = 0;
        assert_eq!(
            run(&mut n, Operation::Open(input.clone()))
                .unsupported
                .as_deref(),
            Some("open flags outside volatile profile")
        );
        input.flags = 0;
        assert_eq!(run(&mut n, Operation::Open(input)).errno, INVAL);
        let dir = match run(&mut n, open(3, b".", 2, PATH_OPEN)).output.unwrap() {
            Output::Fd(fd) => fd,
            _ => unreachable!(),
        };
        assert_eq!(run(&mut n, open(dir, b"x", 1, READ)).errno, NOTCAPABLE);
        assert_eq!(run(&mut n, open(3, b"missing/", 1, READ)).errno, NOTDIR);
    }
    #[test]
    fn readdir_sorts_raw_names_and_preserves_partial_dirent_cookies() {
        let mut n = ns();
        opened(&mut n, b"b");
        opened(&mut n, b"a");
        opened(&mut n, &[255]);
        let data = bytes(run(
            &mut n,
            Operation::Readdir {
                fd: 3,
                length: 25,
                cookie: 0,
            },
        ));
        assert_eq!(data.len(), 25);
        assert_eq!(&data[24..], b"a");
        assert_eq!(&data[..8], &1u64.to_le_bytes());
        let data = bytes(run(
            &mut n,
            Operation::Readdir {
                fd: 3,
                length: 100,
                cookie: 1,
            },
        ));
        assert_eq!(&data[24..25], b"b");
        assert_eq!(data[49], 255);
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Readdir {
                    fd: 3,
                    length: 1,
                    cookie: 3
                }
            )),
            Vec::<u8>::new()
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Readdir {
                    fd: 3,
                    length: 0,
                    cookie: 4
                }
            )
            .errno,
            INVAL
        );
    }
    #[test]
    fn rejected_stdout_writes_charge_offered_bytes_before_rights() {
        let mut input = image();
        input.limits.output_bytes = 2;
        let mut n = Namespace::new(input).unwrap();
        run(&mut n, Operation::Close(1));
        assert_eq!(
            run(
                &mut n,
                Operation::Write {
                    fd: 1,
                    data: b"xy".to_vec(),
                    offset: None
                }
            )
            .errno,
            BADF
        );
        assert_eq!(n.offered_output(), 2);
        assert!(n.stdout().is_empty());
        assert_eq!(
            n.execute(
                Operation::Write {
                    fd: 1,
                    data: b"x".to_vec(),
                    offset: None
                },
                7
            )
            .unwrap_err(),
            Boundary {
                kind: "capacity",
                dimension: "offered-output-bytes"
            }
        );
        let mut n = ns();
        assert_eq!(
            run(
                &mut n,
                Operation::Write {
                    fd: 2,
                    data: b"x".to_vec(),
                    offset: Some(0)
                }
            )
            .errno,
            NOTCAPABLE
        );
        assert_eq!(n.offered_output(), 1);
        assert!(n.stderr().is_empty());
        assert_eq!(
            run(
                &mut n,
                Operation::Write {
                    fd: 1,
                    data: b"ok".to_vec(),
                    offset: None
                }
            )
            .output,
            Some(Output::Written(2))
        );
        assert_eq!(n.stdout(), b"ok");
    }
    #[test]
    fn data_offsets_append_zero_fill_and_times_match_reference() {
        let mut n = ns();
        let fd = opened(&mut n, b"file");
        run(
            &mut n,
            Operation::Write {
                fd,
                data: b"xyz".to_vec(),
                offset: Some(2),
            },
        );
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 20,
                    offset: None
                }
            )),
            b"\0\0xyz"
        );
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 20,
                    offset: None
                }
            )),
            b""
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Seek {
                    fd,
                    offset: -2,
                    whence: 2
                }
            )
            .output,
            Some(Output::Offset(3))
        );
        run(&mut n, Operation::SetFlags { fd, flags: 1 });
        run(
            &mut n,
            Operation::Write {
                fd,
                data: b"!".to_vec(),
                offset: Some(0),
            },
        );
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 20,
                    offset: Some(0)
                }
            )),
            b"\0\0xyz!"
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Seek {
                    fd,
                    offset: 0,
                    whence: 1
                }
            )
            .output,
            Some(Output::Offset(3))
        );
        let stat = match run(&mut n, Operation::FilestatGet(fd)).output.unwrap() {
            Output::Filestat(s) => s,
            _ => unreachable!(),
        };
        assert_eq!(
            (stat.size, stat.atim, stat.mtim, stat.ctim),
            (6, 1007, 1007, 1007)
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 1,
                    offset: Some(u64::MAX)
                }
            )
            .errno,
            INVAL
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Seek {
                    fd,
                    offset: i64::MIN,
                    whence: 0
                }
            )
            .errno,
            INVAL
        );
        run(&mut n, Operation::SetSize { fd, size: 2 });
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 20,
                    offset: Some(0)
                }
            )),
            b"\0\0"
        );
        assert_eq!(run(&mut n, Operation::Sync(fd)).errno, 0);
    }
    #[test]
    fn capacity_keeps_data_and_offset_and_includes_stdin() {
        let mut input = image();
        input.limits.filesystem_bytes = 5;
        let mut n = Namespace::new(input).unwrap();
        let fd = opened(&mut n, b"file");
        let before = n.canonical_state();
        assert_eq!(
            n.execute(
                Operation::Write {
                    fd,
                    data: vec![1],
                    offset: None
                },
                7
            )
            .unwrap_err()
            .dimension,
            "filesystem-bytes"
        );
        assert_eq!(n.canonical_state(), before);
        assert_eq!(
            n.execute(
                Operation::Write {
                    fd: 1,
                    data: vec![0; BUFFER_LIMIT + 1],
                    offset: None
                },
                7
            )
            .unwrap_err()
            .dimension,
            "write-buffer"
        );
        assert_eq!(n.offered_output(), 0);
    }
    #[test]
    fn unlink_keeps_open_node_and_allocation_charge() {
        let mut input = image();
        input.limits.files = 3;
        input.limits.filesystem_bytes = 6;
        let mut n = Namespace::new(input).unwrap();
        let fd = opened(&mut n, b"file");
        run(
            &mut n,
            Operation::Write {
                fd,
                data: b"x".to_vec(),
                offset: None,
            },
        );
        run(
            &mut n,
            Operation::UnlinkFile {
                fd: 3,
                path: b"file".to_vec(),
            },
        );
        assert_eq!(
            bytes(run(
                &mut n,
                Operation::Read {
                    fd,
                    length: 10,
                    offset: Some(0)
                }
            )),
            b"x"
        );
        assert_eq!(
            n.execute(open(3, b"new", 1, READ), 7)
                .unwrap_err()
                .dimension,
            "files"
        );
        assert_eq!(run(&mut n, open(3, b"file", 0, READ)).errno, NOENT);
    }
    #[test]
    fn rename_preserves_fds_and_rejects_cycles_without_mutation() {
        let mut n = ns();
        let fd = opened(&mut n, b"source");
        let before = n.canonical_state();
        assert_eq!(
            run(
                &mut n,
                Operation::Rename {
                    fd: 3,
                    path: b"source".to_vec(),
                    new_fd: 3,
                    new_path: b"missing/".to_vec()
                }
            )
            .errno,
            NOTDIR
        );
        assert_eq!(n.canonical_state(), before);
        run(
            &mut n,
            Operation::Rename {
                fd: 3,
                path: b"source".to_vec(),
                new_fd: 3,
                new_path: b"target".to_vec(),
            },
        );
        assert_eq!(run(&mut n, Operation::FilestatGet(fd)).errno, 0);
        run(
            &mut n,
            Operation::CreateDirectory {
                fd: 3,
                path: b"d".to_vec(),
            },
        );
        run(
            &mut n,
            Operation::CreateDirectory {
                fd: 3,
                path: b"d/e".to_vec(),
            },
        );
        let before = n.canonical_state();
        assert_eq!(
            run(
                &mut n,
                Operation::Rename {
                    fd: 3,
                    path: b"d".to_vec(),
                    new_fd: 3,
                    new_path: b"d/e/f".to_vec()
                }
            )
            .errno,
            INVAL
        );
        assert_eq!(n.canonical_state(), before);
        assert_eq!(
            run(
                &mut n,
                Operation::RemoveDirectory {
                    fd: 3,
                    path: b"d".to_vec()
                }
            )
            .errno,
            NOTEMPTY
        );
        assert_eq!(
            run(
                &mut n,
                Operation::UnlinkFile {
                    fd: 3,
                    path: b"d".to_vec()
                }
            )
            .errno,
            ISDIR
        );
    }
    #[test]
    fn readonly_files_and_directory_error_precedence() {
        let mut input = image();
        input.entries.extend([
            ImageEntry {
                path: b"/frozen".to_vec(),
                directory: true,
                readonly: true,
                data: vec![],
            },
            ImageEntry {
                path: b"/frozen/file".to_vec(),
                directory: false,
                readonly: true,
                data: b"x".to_vec(),
            },
        ]);
        input.preopens.push(b"/frozen".to_vec());
        let mut n = Namespace::new(input).unwrap();
        assert_eq!(run(&mut n, open(3, b"file", 0, WRITE)).errno, ROFS);
        assert_eq!(
            run(
                &mut n,
                Operation::CreateDirectory {
                    fd: 3,
                    path: b"file".to_vec()
                }
            )
            .errno,
            ROFS
        );
        let fd = match run(&mut n, open(3, b"file", 0, READ | SEEK | SET_SIZE))
            .output
            .unwrap()
        {
            Output::Fd(fd) => fd,
            _ => unreachable!(),
        };
        assert_eq!(
            run(&mut n, Operation::SetSize { fd, size: u64::MAX }).errno,
            ROFS
        );
        assert_eq!(
            run(
                &mut n,
                Operation::Write {
                    fd,
                    data: vec![1],
                    offset: None
                }
            )
            .errno,
            NOTCAPABLE
        );
    }
    #[test]
    fn fd_poll_reports_rights_directories_and_eof_without_mutating() {
        let mut n = ns();
        assert_eq!(
            n.fd_readiness(0, false),
            FdReadiness {
                errno: 0,
                nbytes: 5,
                flags: 0
            }
        );
        assert_eq!(n.fd_readiness(1, false).errno, NOTCAPABLE);
        assert_eq!(n.fd_readiness(3, false).errno, INVAL);
        assert_eq!(n.fd_readiness(999, true).errno, BADF);
        run(
            &mut n,
            Operation::Read {
                fd: 0,
                length: 5,
                offset: None,
            },
        );
        assert_eq!(
            n.fd_readiness(0, false),
            FdReadiness {
                errno: 0,
                nbytes: 0,
                flags: 1
            }
        );
    }
    #[test]
    fn preopens_and_std_stat_have_reference_metadata() {
        let mut n = ns();
        assert_eq!(
            run(&mut n, Operation::PrestatGet(3)).output,
            Some(Output::Prestat(4))
        );
        assert_eq!(
            run(&mut n, Operation::PrestatDirName { fd: 3, length: 3 }).errno,
            INVAL
        );
        assert_eq!(
            bytes(run(&mut n, Operation::PrestatDirName { fd: 3, length: 4 })),
            b"/tmp"
        );
        assert_eq!(run(&mut n, Operation::PrestatGet(0)).errno, BADF);
        assert_eq!(
            run(&mut n, Operation::FdstatGet(0)).output,
            Some(Output::Fdstat(Fdstat {
                filetype: 2,
                flags: 0,
                base: READ | SET_FLAGS | FILE_STAT | POLL,
                inheriting: 0
            }))
        );
        assert_eq!(
            run(&mut n, Operation::SetFlags { fd: 0, flags: 2 })
                .unsupported
                .as_deref(),
            Some("descriptor flags outside volatile profile")
        );
        assert_eq!(
            run(&mut n, Operation::Unsupported("sock_accept".into()))
                .unsupported
                .as_deref(),
            Some("operation outside stock WASI profile: sock_accept")
        );
        run(&mut n, Operation::Close(3));
        assert_eq!(run(&mut n, Operation::Close(3)).errno, BADF);
    }
    #[test]
    fn malformed_images_fail_before_creating_descriptors() {
        let mut input = image();
        input.entries[0].path = b"/tmp/../escape".to_vec();
        assert_eq!(Namespace::new(input).err().unwrap().kind, "invalid");
        let mut input = image();
        input.limits.descriptors = 3;
        assert_eq!(
            Namespace::new(input).err().unwrap().dimension,
            "preopen-descriptors"
        );
        let mut input = image();
        input.limits.filesystem_bytes = 4;
        assert_eq!(
            Namespace::new(input).err().unwrap().dimension,
            "initial-namespace"
        );
    }
    #[test]
    fn constructor_refuses_to_thaw_readonly_nodes_and_builds_missing_writable_directories() {
        let mut input = image();
        input.entries[0].readonly = true;
        assert_eq!(
            Namespace::new(input).err().unwrap().dimension,
            "writable mount overlaps captured input"
        );
        let mut input = image();
        input.entries.clear();
        let mut n = Namespace::new(input).unwrap();
        assert_eq!(run(&mut n, open(3, b"file", 1, WRITE)).errno, 0);
        let mut input = image();
        input.limits.files = 1;
        assert_eq!(
            Namespace::new(input).err().unwrap().dimension,
            "initial-namespace"
        );
    }
    #[test]
    fn configured_preopen_path_length_does_not_use_runtime_resolution_limit() {
        let mut input = image();
        let path = [b"/".as_slice(), &vec![b'x'; 4096]].concat();
        input.entries[0].path = path.clone();
        input.writable_directories = vec![path.clone()];
        input.working_directory = path.clone();
        input.preopens = vec![path];
        let mut n = Namespace::new(input).unwrap();
        assert_eq!(
            run(&mut n, Operation::PrestatGet(3)).output,
            Some(Output::Prestat(4097))
        );
        assert_eq!(run(&mut n, open(3, b".", 2, READ)).errno, 0);
        assert_eq!(
            run(&mut n, open(3, &vec![b'x'; 4097], 0, READ)).errno,
            NOTCAPABLE
        );
    }
    #[test]
    fn canonical_digest_includes_detached_directory_children_and_unopened_descendants() {
        let mut n = ns();
        run(
            &mut n,
            Operation::CreateDirectory {
                fd: 3,
                path: b"d".to_vec(),
            },
        );
        let fd = match run(
            &mut n,
            Operation::Open(Open {
                fd: 3,
                path: b"d".to_vec(),
                dirflags: 0,
                oflags: 2,
                base: RIGHTS_ALL,
                inheriting: RIGHTS_ALL,
                flags: 0,
            }),
        )
        .output
        .unwrap()
        {
            Output::Fd(fd) => fd,
            _ => unreachable!(),
        };
        run(
            &mut n,
            Operation::RemoveDirectory {
                fd: 3,
                path: b"d".to_vec(),
            },
        );
        run(
            &mut n,
            Operation::CreateDirectory {
                fd,
                path: b"child".to_vec(),
            },
        );
        let data = n.canonical_state();
        let mut at = b"gomad3.namespace-conformance/v1\0".len();
        fn number(data: &[u8], at: &mut usize) -> u64 {
            let n = u64::from_le_bytes(data[*at..*at + 8].try_into().unwrap());
            *at += 8;
            n
        }
        fn raw<'a>(data: &'a [u8], at: &mut usize) -> &'a [u8] {
            let size = number(data, at) as usize;
            let value = &data[*at..*at + size];
            *at += size;
            value
        }
        fn node(data: &[u8], at: &mut usize) -> u64 {
            let ino = number(data, at);
            *at += 2;
            number(data, at);
            raw(data, at);
            number(data, at);
            ino
        }
        for path in [b"/".as_slice(), b"/tmp"] {
            assert_eq!(data[at], b'N');
            at += 1;
            assert_eq!(raw(&data, &mut at), path);
            node(&data, &mut at);
        }
        for ino in [3, 4, 5, 6, 7] {
            assert_eq!(data[at], b'O');
            at += 1;
            assert_eq!(node(&data, &mut at), ino);
            assert_eq!(number(&data, &mut at), if ino == 6 { 1 } else { 0 });
            if ino == 6 {
                assert_eq!(raw(&data, &mut at), b"child");
                assert_eq!(number(&data, &mut at), 7);
            }
        }
        assert_eq!(data[at], b'D');
    }
}
