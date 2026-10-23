use std::collections::{BTreeMap, BTreeSet};

use serde::{Deserialize, Serialize};

use super::hash::{FileBytes, Hasher, bytes_digest};

pub(super) const MAXIMUM_FILE_BYTES: u64 = 256 << 20;
const MAXIMUM_TOTAL_BYTES: u64 = 1 << 30;
const MAXIMUM_NODES: usize = 100_000;
const MAXIMUM_HANDLES: u64 = 100_000;

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum Error {
    Errno(&'static str),
    Capacity {
        resource: &'static str,
        required: u64,
        maximum: u64,
    },
    Invalid(&'static str),
}

pub(crate) type Result<T> = std::result::Result<T, Error>;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u64)]
pub(crate) enum Kind {
    File = 1,
    Directory = 2,
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct VolumeConfig {
    #[serde(rename = "ID")]
    pub(crate) id: String,
    #[serde(rename = "Path")]
    pub(crate) mount: String,
    pub(crate) capacity_bytes: u64,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct VolumeLimits {
    pub(crate) pending_operations: u64,
    pub(crate) transitions: u64,
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct Operation {
    #[serde(rename = "ID")]
    pub(crate) id: u64,
    pub(crate) kind: String,
    pub(crate) dependencies: Option<Vec<u64>>,
    pub(crate) inode: u64,
    pub(crate) offset: u64,
    pub(crate) bytes: u64,
    #[serde(rename = "PayloadSHA256")]
    pub(crate) payload_sha256: String,
    #[serde(rename = "EffectSHA256")]
    pub(crate) effect_sha256: String,
    pub(crate) path: String,
    pub(crate) destination: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct SnapshotEntry {
    pub(crate) path: String,
    pub(crate) mode: u32,
    pub(crate) kind: String,
    pub(crate) mod_time: i64,
    pub(crate) size: u64,
    #[serde(rename = "DataSHA256")]
    pub(crate) data_sha256: String,
}

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub(crate) struct VolumeSnapshot {
    #[serde(rename = "ID")]
    pub(crate) id: String,
    pub(crate) mount: String,
    pub(crate) capacity_bytes: u64,
    pub(crate) persisted: Vec<SnapshotEntry>,
    pub(crate) volatile: Vec<SnapshotEntry>,
    pub(crate) pending_operations: u64,
    #[serde(rename = "PendingSHA256")]
    pub(crate) pending_sha256: String,
    pub(crate) next_operation: u64,
    pub(crate) identity: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct DiskObject {
    pub(super) inode: u64,
    pub(super) mode: u32,
    pub(super) kind: Kind,
    pub(super) mod_time: i64,
    pub(super) data: FileBytes,
}

#[derive(Debug, Default, PartialEq, Eq)]
pub(super) struct DiskState {
    pub(super) objects: BTreeMap<u64, DiskObject>,
    pub(super) paths: BTreeMap<String, u64>,
    pub(super) owned: BTreeSet<u64>,
}

impl Clone for DiskState {
    fn clone(&self) -> Self {
        Self {
            objects: self.objects.clone(),
            paths: self.paths.clone(),
            owned: BTreeSet::new(),
        }
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(super) struct Effect {
    pub(super) kind: String,
    pub(super) inode: u64,
    pub(super) parent: u64,
    pub(super) parents: Vec<u64>,
    pub(super) offset: u64,
    pub(super) size: u64,
    pub(super) mode: u32,
    pub(super) mod_time: i64,
    pub(super) data: Vec<u8>,
    pub(super) object: Option<DiskObject>,
    pub(super) namespace: Vec<NamespaceEffect>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct NamespaceEffect {
    pub(super) path: String,
    pub(super) inode: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct PersistenceOperation {
    pub(super) id: u64,
    pub(super) kind: String,
    pub(super) deps: Vec<u64>,
    pub(super) stores: Vec<String>,
    pub(super) effect: Effect,
}

#[derive(Clone, Debug)]
pub(super) struct OperationPlan {
    pub(super) kind: String,
    pub(super) effect: Effect,
    pub(super) stores: Vec<String>,
    pub(super) depends: Vec<String>,
}

#[derive(Clone, Debug)]
pub(super) struct VolumeState {
    pub(super) id: String,
    pub(super) mount: String,
    pub(super) capacity_bytes: u64,
    pub(super) persisted: DiskState,
    pub(super) pending: Vec<PersistenceOperation>,
    pub(super) history: Vec<PersistenceOperation>,
    pub(super) next_operation: u64,
    pub(super) last: BTreeMap<String, u64>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct Node {
    pub(crate) inode: u64,
    pub(crate) mode: u32,
    pub(crate) kind: Kind,
    pub(crate) mod_time: i64,
    pub(crate) data: FileBytes,
    pub(crate) volume: String,
    pub(crate) linked: bool,
    pub(crate) mount_root: bool,
    pub(crate) readonly: bool,
    pub(super) handles: u64,
}

#[derive(Clone, Copy, Debug, Default)]
#[expect(
    clippy::struct_excessive_bools,
    reason = "Flags mirror independent Go filesystem open bits"
)]
pub(crate) struct OpenFlags {
    pub(crate) read: bool,
    pub(crate) write: bool,
    pub(crate) append: bool,
    pub(crate) create: bool,
    pub(crate) exclusive: bool,
    pub(crate) truncate: bool,
}

#[derive(Debug)]
#[expect(
    clippy::struct_excessive_bools,
    reason = "Read, write, append and close flags are independent handle state"
)]
pub(crate) struct Handle {
    pub(crate) inode: u64,
    pub(crate) name: String,
    pub(crate) offset: u64,
    pub(super) directory_offset: usize,
    pub(super) readable: bool,
    pub(super) writable: bool,
    pub(super) append: bool,
    pub(super) closed: bool,
    pub(super) generation: u64,
}

#[derive(Debug)]
pub(crate) struct Fs {
    pub(super) volumes: BTreeMap<String, VolumeState>,
    pub(super) next_inode: u64,
    pub(super) nodes: BTreeMap<String, Node>,
    pub(super) detached: BTreeMap<u64, Node>,
    pub(super) limits: VolumeLimits,
    pub(super) generation: u64,
    pub(super) unavailable: bool,
    pub(super) cwd: String,
    pub(super) used_bytes: u64,
    pub(super) configured: bool,
}

impl Fs {
    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn chmod(&mut self, name: &str, mode: u32, now: i64) -> Result<()> {
        let path = self.resolve(name)?;
        self.metadata(self.lookup(&path)?.inode, mode & 0o777, now)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn chtimes(&mut self, name: &str, mod_time: i64) -> Result<()> {
        let path = self.resolve(name)?;
        let node = self.lookup(&path)?;
        self.metadata(node.inode, node.mode, mod_time)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn chdir(&mut self, name: &str) -> Result<()> {
        let path = self.resolve(name)?;
        if self.lookup(&path)?.kind != Kind::Directory {
            return Err(Error::Errno("not-directory"));
        }
        self.cwd = path;
        Ok(())
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_chdir(&mut self, handle: &Handle) -> Result<()> {
        self.check_handle(handle)?;
        let node = self.node(handle.inode)?;
        if node.kind != Kind::Directory {
            return Err(Error::Errno("not-directory"));
        }
        if !node.linked {
            return Err(Error::Errno("no-entry"));
        }
        self.cwd = self
            .nodes
            .iter()
            .find_map(|(path, node)| (node.inode == handle.inode).then(|| path.clone()))
            .ok_or(Error::Errno("no-entry"))?;
        Ok(())
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_seek(&self, handle: &mut Handle, offset: i64, whence: u8) -> Result<u64> {
        self.check_handle(handle)?;
        let next = match whence {
            0 => offset,
            1 => handle.offset.cast_signed().wrapping_add(offset),
            2 => (i64::try_from(self.node(handle.inode)?.data.len())
                .expect("file length is at most 256 MiB"))
            .wrapping_add(offset),
            _ => return Err(Error::Errno("invalid")),
        };
        if next < 0 {
            return Err(Error::Errno("invalid"));
        }
        handle.offset = next.cast_unsigned();
        Ok(handle.offset)
    }

    pub(crate) fn handle_read_dir(
        &self,
        handle: &mut Handle,
        count: i64,
    ) -> Result<Vec<(String, Node)>> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        if self.node(handle.inode)?.kind != Kind::Directory {
            return Err(Error::Errno("not-directory"));
        }
        let entries: Vec<_> = self
            .nodes
            .iter()
            .filter(|(path, _)| path.as_str() != handle.name && parent_path(path) == handle.name)
            .map(|(path, node)| (path.rsplit('/').next().unwrap().into(), node.clone()))
            .collect();
        if handle.directory_offset == entries.len() {
            return if count > 0 {
                Err(Error::Errno("eof"))
            } else {
                Ok(Vec::new())
            };
        }
        let remaining = entries
            .len()
            .checked_sub(handle.directory_offset)
            .ok_or(Error::Errno("invalid"))?;
        let take = if count > 0 {
            remaining.min(usize::try_from(count).unwrap_or(usize::MAX))
        } else {
            remaining
        };
        let result = entries[handle.directory_offset..handle.directory_offset + take].to_vec();
        handle.directory_offset += take;
        Ok(result)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_write(
        &mut self,
        handle: &mut Handle,
        data: &[u8],
        now: i64,
    ) -> Result<usize> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        if !handle.writable {
            return Err(Error::Errno("bad-descriptor"));
        }
        let node = self.node(handle.inode)?;
        if node.readonly {
            return Err(Error::Errno("read-only"));
        }
        if handle.append {
            handle.offset = node.data.len() as u64;
        }
        let written = self.write_at(handle.inode, handle.offset, data, now)?;
        handle.offset += written as u64;
        Ok(written)
    }

    pub(crate) fn handle_write_at(
        &mut self,
        handle: &Handle,
        offset: i64,
        data: &[u8],
        now: i64,
    ) -> Result<usize> {
        self.check_handle(handle)?;
        if !handle.writable {
            return Err(Error::Errno("bad-descriptor"));
        }
        if handle.append || offset < 0 {
            return Err(Error::Errno("invalid"));
        }
        self.write_at(handle.inode, offset.cast_unsigned(), data, now)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_read(&self, handle: &mut Handle, length: usize) -> Result<Vec<u8>> {
        self.check_handle(handle)?;
        if !handle.readable {
            return Err(Error::Errno("bad-descriptor"));
        }
        let result = self.read_at(handle.inode, handle.offset, length)?;
        handle.offset += result.len() as u64;
        Ok(result)
    }

    /// Go `Handle.ReadAt`: a short read at EOF is not an error.
    pub(crate) fn handle_read_at(
        &self,
        handle: &Handle,
        offset: i64,
        length: usize,
    ) -> Result<Vec<u8>> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        if !handle.readable {
            return Err(Error::Errno("bad-descriptor"));
        }
        if offset < 0 {
            return Err(Error::Errno("invalid"));
        }
        self.read_at(handle.inode, offset.cast_unsigned(), length)
    }

    /// Go `Handle.Metadata`: modification time and size without data.
    pub(crate) fn handle_metadata(&self, handle: &Handle) -> Result<(i64, u64)> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        let node = self.node(handle.inode)?;
        Ok((node.mod_time, node.data.len() as u64))
    }

    pub(crate) fn handle_resize(&mut self, handle: &Handle, size: i64, now: i64) -> Result<()> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        if !handle.writable {
            return Err(Error::Errno("bad-descriptor"));
        }
        if self.node(handle.inode)?.readonly {
            return Err(Error::Errno("read-only"));
        }
        if size < 0 {
            return Err(Error::Errno("invalid"));
        }
        self.resize(handle.inode, size.cast_unsigned(), now)
    }

    pub(crate) fn handle_sync(&mut self, handle: &Handle) -> Result<Vec<u64>> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        self.sync_node(handle.inode)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_chmod(&mut self, handle: &Handle, mode: u32, now: i64) -> Result<()> {
        self.check_handle(handle)?;
        self.metadata(handle.inode, mode & 0o777, now)
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn handle_chtimes(&mut self, handle: &Handle, mod_time: i64) -> Result<()> {
        self.check_handle(handle)?;
        let mode = self.node(handle.inode)?.mode;
        self.metadata(handle.inode, mode, mod_time)
    }

    pub(crate) fn mkdir(&mut self, path: &str, mode: u32, now: i64) -> Result<()> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        let path = self.resolve(path).map_err(|_| Error::Errno("invalid"))?;
        self.create(&path, Kind::Directory, mode, now).map(|_| ())
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn mkdir_all(&mut self, name: &str, mode: u32, now: i64) -> Result<()> {
        let path = self.resolve(name)?;
        let mut planned: BTreeMap<String, Node> = BTreeMap::new();
        let mut links = Vec::new();
        let mut new_entries: BTreeMap<String, usize> = BTreeMap::new();
        let mut current = String::new();
        for component in path[1..]
            .split('/')
            .filter(|component| !component.is_empty())
        {
            current.push('/');
            current.push_str(component);
            if let Some(existing) = planned.get(&current).or_else(|| self.nodes.get(&current)) {
                if existing.kind != Kind::Directory {
                    return Err(Error::Errno("not-directory"));
                }
                continue;
            }
            let parent_name = parent_path(&current);
            let parent = planned
                .get(parent_name)
                .or_else(|| self.nodes.get(parent_name))
                .ok_or(Error::Errno("no-entry"))?
                .clone();
            if parent.readonly {
                return Err(Error::Errno("read-only"));
            }
            if self.nodes.len() + self.detached.len() + links.len() + 1 > MAXIMUM_NODES
                || self.directory_entries(parent_name)
                    + new_entries.get(parent_name).copied().unwrap_or(0)
                    == MAXIMUM_NODES
            {
                return Err(Error::Errno("no-space"));
            }
            let node = Node::new(
                self.allocate_inode()?,
                mode & 0o777,
                Kind::Directory,
                now,
                &parent.volume,
            );
            *new_entries.entry(parent_name.into()).or_default() += 1;
            planned.insert(current.clone(), node.clone());
            links.push((current.clone(), node, parent));
        }
        let mut operations: BTreeMap<String, u64> = BTreeMap::new();
        for (_, node, _) in &links {
            *operations.entry(node.volume.clone()).or_default() += 2;
        }
        for (volume, count) in operations {
            self.preflight_operations(&volume, count)?;
        }
        for (path, node, parent) in links {
            self.record_allocation(&path, &node, &parent);
            self.nodes.insert(path, node);
        }
        Ok(())
    }

    pub(crate) fn open(
        &mut self,
        name: &str,
        flags: OpenFlags,
        mode: u32,
        now: i64,
    ) -> Result<Handle> {
        let path = self.resolve(name)?;
        if self.open_handles() == MAXIMUM_HANDLES {
            return Err(Error::Errno("too-many-files"));
        }
        let inode = if let Some(node) = self.nodes.get(&path) {
            if flags.create && flags.exclusive {
                return Err(Error::Errno("exists"));
            }
            node.inode
        } else {
            if !flags.create {
                return Err(Error::Errno("no-entry"));
            }
            self.create(&path, Kind::File, mode, now)?
        };
        let node = self.node(inode)?;
        if node.readonly && (flags.write || flags.truncate || flags.create) {
            return Err(Error::Errno("read-only"));
        }
        if node.kind == Kind::Directory && flags.write {
            return Err(Error::Errno("is-directory"));
        }
        if flags.truncate && flags.write && !node.data.is_empty() {
            self.resize(inode, 0, now)?;
        }
        self.node_mut(inode)?.handles += 1;
        Ok(Handle {
            inode,
            name: path,
            offset: 0,
            directory_offset: 0,
            readable: flags.read,
            writable: flags.write,
            append: flags.append,
            closed: false,
            generation: self.generation,
        })
    }

    pub(crate) fn close(&mut self, handle: &mut Handle) -> Result<()> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.check_handle(handle)?;
        handle.closed = true;
        let node = self.node_mut(handle.inode)?;
        node.handles -= 1;
        let released_bytes = (!node.linked && node.handles == 0).then_some(node.data.len() as u64);
        if let Some(bytes) = released_bytes {
            self.used_bytes -= bytes;
            self.detached.remove(&handle.inode);
        }
        Ok(())
    }

    pub(crate) fn check_handle(&self, handle: &Handle) -> Result<()> {
        if self.unavailable || handle.generation != self.generation {
            return Err(Error::Errno("stale"));
        }
        if handle.closed {
            return Err(Error::Errno("bad-descriptor"));
        }
        Ok(())
    }

    fn detach(&mut self, path: &str) {
        if let Some(mut node) = self.nodes.remove(path) {
            node.linked = false;
            if node.handles == 0 {
                self.used_bytes -= node.data.len() as u64;
            } else {
                self.detached.insert(node.inode, node);
            }
        }
    }

    pub(crate) fn rename(&mut self, old_name: &str, new_name: &str, now: i64) -> Result<()> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        let old_path = self.resolve(old_name)?;
        let new_path = self.resolve(new_name)?;
        if old_path == "/" || new_path == "/" {
            return Err(Error::Errno("invalid"));
        }
        let node = self.lookup(&old_path)?.clone();
        let parent = self.lookup(parent_path(&new_path))?.clone();
        if node.readonly || parent.readonly || node.mount_root || node.volume != parent.volume {
            return Err(Error::Errno("cross-device"));
        }
        if parent.kind != Kind::Directory {
            return Err(Error::Errno("not-directory"));
        }
        if node.kind == Kind::Directory && new_path.starts_with(&format!("{old_path}/")) {
            return Err(Error::Errno("invalid"));
        }
        let existing = self.nodes.get(&new_path);
        if existing.is_some_and(|existing| {
            existing.readonly || existing.mount_root || existing.volume != node.volume
        }) {
            return Err(Error::Errno("cross-device"));
        }
        let descendants: Vec<_> = if node.kind == Kind::Directory {
            self.nodes
                .iter()
                .filter(|(path, _)| path.starts_with(&format!("{old_path}/")))
                .map(|(path, node)| (path.clone(), node.clone()))
                .collect()
        } else {
            Vec::new()
        };
        if descendants.iter().any(|(_, node)| node.readonly) {
            return Err(Error::Errno("cross-device"));
        }
        if old_path == new_path {
            return Ok(());
        }
        if existing.is_some_and(|existing| existing.kind == Kind::Directory) {
            return Err(Error::Errno("exists"));
        }
        if existing.is_none()
            && parent_path(&old_path) != parent_path(&new_path)
            && self.directory_entries(parent_path(&new_path)) == MAXIMUM_NODES
        {
            return Err(Error::Errno("no-space"));
        }
        self.preflight_operations(&node.volume, 1)?;
        let old_parent = self.lookup(parent_path(&old_path))?;
        let mut parents = vec![old_parent.inode];
        if parent.inode != old_parent.inode {
            parents.push(parent.inode);
        }
        let mut effects = Vec::new();
        if !node.volume.is_empty() {
            let volume = &self.volumes[&node.volume];
            effects.push(NamespaceEffect {
                path: volume.relative(&old_path),
                inode: 0,
            });
            for (path, _) in &descendants {
                effects.push(NamespaceEffect {
                    path: volume.relative(path),
                    inode: 0,
                });
            }
            effects.push(NamespaceEffect {
                path: volume.relative(&new_path),
                inode: node.inode,
            });
            for (path, child) in &descendants {
                effects.push(NamespaceEffect {
                    path: volume.relative(&format!("{new_path}{}", &path[old_path.len()..])),
                    inode: child.inode,
                });
            }
        }
        self.record_namespace(&node.volume, parents, effects, node.inode, now);
        self.detach(&new_path);
        let mut moved = self.nodes.remove(&old_path).unwrap();
        moved.mod_time = now;
        self.nodes.insert(new_path.clone(), moved);
        for (path, descendant) in descendants {
            self.nodes.remove(&path);
            self.nodes
                .insert(format!("{new_path}{}", &path[old_path.len()..]), descendant);
        }
        if node.kind == Kind::Directory
            && (self.cwd == old_path || self.cwd.starts_with(&format!("{old_path}/")))
        {
            self.cwd = format!("{new_path}{}", &self.cwd[old_path.len()..]);
        }
        Ok(())
    }

    fn removal_allowed(&self, path: &str, node: &Node) -> Result<()> {
        if node.readonly {
            return Err(Error::Errno("read-only"));
        }
        if node.mount_root || self.cwd == path || self.cwd.starts_with(&format!("{path}/")) {
            return Err(Error::Errno("busy"));
        }
        Ok(())
    }

    pub(crate) fn remove(&mut self, name: &str) -> Result<()> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        let path = self.resolve(name).map_err(|_| Error::Errno("invalid"))?;
        if path == "/" {
            return Err(Error::Errno("invalid"));
        }
        let node = self.lookup(&path)?.clone();
        self.removal_allowed(&path, &node)?;
        if node.kind == Kind::Directory
            && self
                .nodes
                .keys()
                .any(|candidate| candidate.starts_with(&format!("{path}/")))
        {
            return Err(Error::Errno("not-empty"));
        }
        self.preflight_operations(&node.volume, 1)?;
        let parent = self.lookup(parent_path(&path))?.inode;
        let effect_path = self
            .volumes
            .get(&node.volume)
            .map(|volume| volume.relative(&path))
            .unwrap_or_default();
        self.record_namespace(
            &node.volume,
            vec![parent],
            vec![NamespaceEffect {
                path: effect_path,
                inode: 0,
            }],
            node.inode,
            node.mod_time,
        );
        self.detach(&path);
        Ok(())
    }

    #[allow(
        dead_code,
        reason = "Complete volume projection port; the WASI namespace routes a subset of these operations."
    )]
    pub(crate) fn remove_all(&mut self, name: &str) -> Result<()> {
        let path = self.resolve(name).map_err(|_| Error::Errno("invalid"))?;
        if path == "/" {
            return Err(Error::Errno("invalid"));
        }
        let Some(node) = self.nodes.get(&path).cloned() else {
            return Ok(());
        };
        if self.cwd == path || self.cwd.starts_with(&format!("{path}/")) {
            return Err(Error::Errno("busy"));
        }
        self.removal_allowed(&path, &node)?;
        let paths: Vec<_> = self
            .nodes
            .iter()
            .filter(|(candidate, _)| {
                candidate.as_str() == path || candidate.starts_with(&format!("{path}/"))
            })
            .map(|(candidate, child)| (candidate.clone(), child.readonly))
            .collect();
        if paths.iter().any(|(_, readonly)| *readonly) {
            return Err(Error::Errno("read-only"));
        }
        self.preflight_operations(&node.volume, 1)?;
        let parent = self.lookup(parent_path(&path))?.inode;
        let effects = self
            .volumes
            .get(&node.volume)
            .map(|volume| {
                paths
                    .iter()
                    .rev()
                    .map(|(path, _)| NamespaceEffect {
                        path: volume.relative(path),
                        inode: 0,
                    })
                    .collect()
            })
            .unwrap_or_default();
        self.record_namespace(
            &node.volume,
            vec![parent],
            effects,
            node.inode,
            node.mod_time,
        );
        for (path, _) in paths {
            self.detach(&path);
        }
        Ok(())
    }

    pub(super) fn record_namespace(
        &mut self,
        volume_id: &str,
        parents: Vec<u64>,
        namespace: Vec<NamespaceEffect>,
        inode: u64,
        mod_time: i64,
    ) {
        if volume_id.is_empty() {
            return;
        }
        let mut depends: Vec<_> = parents.iter().map(|parent| alloc_key(*parent)).collect();
        if inode != 0 {
            depends.push(alloc_key(inode));
        }
        let stores: Vec<_> = namespace
            .iter()
            .map(|part| namespace_key(&part.path))
            .collect();
        depends.extend(stores.iter().cloned());
        self.volumes
            .get_mut(volume_id)
            .unwrap()
            .add_operations(vec![OperationPlan {
                kind: "namespace".into(),
                effect: Effect {
                    kind: "namespace".into(),
                    inode,
                    parents,
                    mod_time,
                    namespace,
                    ..Effect::default()
                },
                stores,
                depends,
            }]);
    }

    pub(crate) fn metadata(&mut self, inode: u64, mode: u32, mod_time: i64) -> Result<()> {
        // Scalar fields only: cloning a node would copy its file data.
        let node = self.node(inode)?;
        if node.readonly {
            return Err(Error::Errno("read-only"));
        }
        let volume = node.volume.clone();
        self.preflight_operations(&volume, 1)?;
        if !volume.is_empty() {
            self.volumes
                .get_mut(&volume)
                .unwrap()
                .add_operations(vec![OperationPlan {
                    kind: "metadata".into(),
                    effect: Effect {
                        kind: "metadata".into(),
                        inode,
                        mode,
                        mod_time,
                        ..Effect::default()
                    },
                    stores: vec![metadata_key(inode)],
                    depends: vec![alloc_key(inode), metadata_key(inode)],
                }]);
        }
        let node = self.node_mut(inode)?;
        node.mode = mode;
        node.mod_time = mod_time;
        Ok(())
    }

    pub(crate) fn read_at(&self, inode: u64, offset: u64, length: usize) -> Result<Vec<u8>> {
        let node = self.node(inode)?;
        if node.kind != Kind::File {
            return Err(Error::Errno("is-directory"));
        }
        if offset >= node.data.len() as u64 {
            return Ok(Vec::new());
        }
        let start = usize::try_from(offset).expect("offset is bounded by the file length");
        let end = start.saturating_add(length).min(node.data.len());
        Ok(node.data[start..end].to_vec())
    }

    pub(crate) fn new(now: i64) -> Self {
        let mut result = Self {
            volumes: BTreeMap::new(),
            next_inode: 2,
            nodes: BTreeMap::new(),
            detached: BTreeMap::new(),
            limits: VolumeLimits::default(),
            generation: 1,
            unavailable: false,
            cwd: "/".into(),
            used_bytes: 0,
            configured: false,
        };
        result
            .reset_nodes(now)
            .expect("new filesystem has inode headroom");
        result
    }

    fn allocate_inode(&mut self) -> Result<u64> {
        let inode = self.next_inode;
        self.next_inode = inode.checked_add(1).ok_or(Error::Errno("no-space"))?;
        Ok(inode)
    }

    fn reset_nodes(&mut self, now: i64) -> Result<()> {
        let temp_inode = self.allocate_inode()?;
        self.nodes = BTreeMap::from([
            ("/".into(), Node::new(1, 0o755, Kind::Directory, 0, "")),
            (
                "/tmp".into(),
                Node::new(temp_inode, 0o777, Kind::Directory, now, ""),
            ),
        ]);
        Ok(())
    }

    pub(super) fn preflight_inode_allocations(&self, count: u64) -> Result<()> {
        self.next_inode
            .checked_add(count)
            .ok_or(Error::Errno("no-space"))?;
        Ok(())
    }

    pub(super) fn rebuild_inode_allocations(&self) -> u64 {
        1 + mount_inode_allocations(
            self.volumes.values().map(|volume| volume.mount.as_str()),
            BTreeSet::from(["/".into(), "/tmp".into()]),
        )
    }

    pub(super) fn preflight_rebuild(&self) -> Result<()> {
        self.preflight_inode_allocations(self.rebuild_inode_allocations())?;
        self.generation
            .checked_add(1)
            .ok_or(Error::Errno("no-space"))?;
        Ok(())
    }

    pub(crate) fn configure_volumes(
        &mut self,
        configs: &[VolumeConfig],
        limits: VolumeLimits,
        now: i64,
    ) -> Result<()> {
        if self.configured
            || self.nodes.len() != 2
            || !self.nodes.contains_key("/")
            || !self.nodes.contains_key("/tmp")
            || self.open_handles() != 0
            || self.used_bytes != 0
        {
            return Err(Error::Invalid(
                "simulation volumes must be configured on a new filesystem",
            ));
        }
        if limits.pending_operations == 0 || limits.transitions == 0 {
            return Err(Error::Invalid("simulation volume limits must be nonzero"));
        }
        let mut ids = BTreeSet::new();
        let mut mounts: BTreeSet<&str> = BTreeSet::new();
        let mut previous = "";
        for config in configs {
            if normalize(&config.mount).ok().as_deref() != Some(config.mount.as_str())
                || config.mount == "/"
            {
                return Err(Error::Invalid(
                    "simulation volume has an invalid mount path",
                ));
            }
            if config.id.is_empty() || config.id.contains('\0') {
                return Err(Error::Invalid("simulation volume ID is invalid"));
            }
            if !ids.insert(config.id.as_str()) {
                return Err(Error::Invalid("simulation volume ID is duplicated"));
            }
            if config.mount.as_str() <= previous {
                return Err(Error::Invalid(
                    "simulation volume mount paths must be strictly sorted",
                ));
            }
            if config.capacity_bytes == 0 {
                return Err(Error::Invalid("simulation volume has zero capacity"));
            }
            if mounts.iter().any(|mount| {
                config.mount.starts_with(&format!("{mount}/"))
                    || mount.starts_with(&format!("{}/", config.mount))
            }) {
                return Err(Error::Invalid(
                    "simulation volume mount paths must not overlap",
                ));
            }
            mounts.insert(&config.mount);
            previous = &config.mount;
        }
        self.preflight_inode_allocations(mount_inode_allocations(
            configs.iter().map(|config| config.mount.as_str()),
            self.nodes.keys().cloned().collect(),
        ))?;
        self.configured = true;
        self.limits = limits;
        for config in configs {
            let mut current = String::new();
            for component in config.mount[1..].split('/') {
                current.push('/');
                current.push_str(component);
                if let Some(existing) = self.nodes.get(&current) {
                    if existing.kind != Kind::Directory {
                        return Err(Error::Errno("not-directory"));
                    }
                    if current == config.mount {
                        return Err(Error::Errno("exists"));
                    }
                    continue;
                }
                let mut node = Node::new(self.allocate_inode()?, 0o755, Kind::Directory, now, "");
                if current == config.mount {
                    node.volume.clone_from(&config.id);
                    node.mount_root = true;
                }
                self.nodes.insert(current.clone(), node);
            }
            let root = &self.nodes[&config.mount];
            self.volumes.insert(
                config.id.clone(),
                VolumeState {
                    id: config.id.clone(),
                    mount: config.mount.clone(),
                    capacity_bytes: config.capacity_bytes,
                    persisted: DiskState {
                        objects: BTreeMap::from([(root.inode, disk_object_for_node(root))]),
                        paths: BTreeMap::from([("/".into(), root.inode)]),
                        owned: BTreeSet::from([root.inode]),
                    },
                    pending: Vec::new(),
                    history: Vec::new(),
                    next_operation: 1,
                    last: BTreeMap::new(),
                },
            );
        }
        Ok(())
    }

    pub(crate) fn resolve(&self, path: &str) -> Result<String> {
        if self.unavailable {
            return Err(Error::Errno("stale"));
        }
        normalize(&if path.starts_with('/') {
            path.to_owned()
        } else {
            format!("{}/{path}", self.cwd)
        })
    }

    pub(crate) fn lookup(&self, path: &str) -> Result<&Node> {
        if self.unavailable {
            return Err(Error::Errno("stale"));
        }
        self.nodes.get(path).ok_or(Error::Errno("no-entry"))
    }

    pub(crate) fn node(&self, inode: u64) -> Result<&Node> {
        if self.unavailable {
            return Err(Error::Errno("stale"));
        }
        self.nodes
            .values()
            .find(|node| node.inode == inode)
            .or_else(|| self.detached.get(&inode))
            .ok_or(Error::Errno("bad-descriptor"))
    }

    fn node_mut(&mut self, inode: u64) -> Result<&mut Node> {
        if self.unavailable {
            return Err(Error::Errno("stale"));
        }
        if let Some(path) = self
            .nodes
            .iter()
            .find_map(|(path, node)| (node.inode == inode).then(|| path.clone()))
        {
            return Ok(self.nodes.get_mut(&path).unwrap());
        }
        self.detached
            .get_mut(&inode)
            .ok_or(Error::Errno("bad-descriptor"))
    }

    fn open_handles(&self) -> u64 {
        self.nodes
            .values()
            .chain(self.detached.values())
            .map(|node| node.handles)
            .sum()
    }

    fn directory_entries(&self, path: &str) -> usize {
        self.nodes
            .keys()
            .filter(|name| name.as_str() != path && parent_path(name) == path)
            .count()
    }

    pub(super) fn preflight_operations(&self, volume_id: &str, count: u64) -> Result<()> {
        if volume_id.is_empty() || count == 0 {
            return Ok(());
        }
        let required = self.volumes[volume_id].pending.len() as u64 + count;
        if required > self.limits.pending_operations {
            return Err(Error::Capacity {
                resource: "pending_operations",
                required,
                maximum: self.limits.pending_operations,
            });
        }
        Ok(())
    }

    fn preflight_growth(&self, node: &Node, final_size: u64) -> Result<()> {
        if node.volume.is_empty() || final_size <= node.data.len() as u64 {
            return Ok(());
        }
        let mut seen = BTreeSet::new();
        let used: u64 = self
            .nodes
            .values()
            .filter(|candidate| candidate.volume == node.volume && seen.insert(candidate.inode))
            .map(|candidate| candidate.data.len() as u64)
            .sum();
        let required = used + final_size - node.data.len() as u64;
        let maximum = self.volumes[&node.volume].capacity_bytes;
        if required > maximum {
            return Err(Error::Capacity {
                resource: "volume_bytes",
                required,
                maximum,
            });
        }
        Ok(())
    }

    pub(crate) fn create(&mut self, name: &str, kind: Kind, mode: u32, now: i64) -> Result<u64> {
        let path = self.resolve(name)?;
        if path == "/" {
            return Err(Error::Errno("invalid"));
        }
        if self.nodes.contains_key(&path) {
            return Err(Error::Errno("exists"));
        }
        let parent = self.lookup(parent_path(&path))?.clone();
        if parent.kind != Kind::Directory {
            return Err(Error::Errno("not-directory"));
        }
        if parent.readonly {
            return Err(Error::Errno("read-only"));
        }
        if self.nodes.len() + self.detached.len() == MAXIMUM_NODES
            || self.directory_entries(parent_path(&path)) == MAXIMUM_NODES
        {
            return Err(Error::Errno("no-space"));
        }
        self.preflight_operations(&parent.volume, 2)?;
        let inode = self.allocate_inode()?;
        let node = Node::new(inode, mode & 0o777, kind, now, &parent.volume);
        self.record_allocation(&path, &node, &parent);
        self.nodes.insert(path, node);
        Ok(inode)
    }

    fn record_allocation(&mut self, path: &str, node: &Node, parent: &Node) {
        if node.volume.is_empty() {
            return;
        }
        let volume = self.volumes.get_mut(&node.volume).unwrap();
        let relative = volume.relative(path);
        volume.add_operations(vec![
            OperationPlan {
                kind: "alloc".into(),
                effect: Effect {
                    kind: "alloc".into(),
                    inode: node.inode,
                    object: Some(disk_object_for_node(node)),
                    ..Effect::default()
                },
                stores: vec![alloc_key(node.inode)],
                depends: vec![],
            },
            OperationPlan {
                kind: "namespace".into(),
                effect: Effect {
                    kind: "namespace".into(),
                    inode: node.inode,
                    parent: parent.inode,
                    namespace: vec![NamespaceEffect {
                        path: relative.clone(),
                        inode: node.inode,
                    }],
                    ..Effect::default()
                },
                stores: vec![namespace_key(&relative)],
                depends: vec![
                    alloc_key(node.inode),
                    alloc_key(parent.inode),
                    namespace_key(&relative),
                ],
            },
        ]);
    }

    pub(crate) fn write_at(
        &mut self,
        inode: u64,
        offset: u64,
        data: &[u8],
        now: i64,
    ) -> Result<usize> {
        let node = self.node(inode)?;
        if node.readonly {
            return Err(Error::Errno("read-only"));
        }
        let end = offset
            .checked_add(data.len() as u64)
            .filter(|end| *end <= MAXIMUM_FILE_BYTES)
            .ok_or(Error::Errno("file-too-large"))?;
        let length = node.data.len() as u64;
        self.preflight_operations(&node.volume, 1 + u64::from(end > length))?;
        self.preflight_growth(node, end)?;
        let volume = node.volume.clone();
        if !volume.is_empty() {
            let mut plans = Vec::new();
            if end > length {
                plans.push(resize_plan(inode, end, now));
            }
            plans.push(OperationPlan {
                kind: "write".into(),
                effect: Effect {
                    kind: "write".into(),
                    inode,
                    offset,
                    data: data.to_vec(),
                    mod_time: now,
                    ..Effect::default()
                },
                stores: vec![write_key(inode)],
                depends: vec![alloc_key(inode), resize_key(inode)],
            });
            self.volumes.get_mut(&volume).unwrap().add_operations(plans);
        }
        let growth = end.saturating_sub(length);
        if growth > MAXIMUM_TOTAL_BYTES - self.used_bytes {
            return Err(Error::Errno("no-space"));
        }
        self.used_bytes += growth;
        let node = self.node_mut(inode)?;
        if end > node.data.len() as u64 {
            node.data.resize(
                usize::try_from(end).expect("file length is at most 256 MiB"),
                0,
            );
        }
        node.data[usize::try_from(offset).expect("offset is bounded by the file length")
            ..usize::try_from(end).expect("file length is at most 256 MiB")]
            .copy_from_slice(data);
        node.mod_time = now;
        Ok(data.len())
    }

    pub(crate) fn resize(&mut self, inode: u64, size: u64, now: i64) -> Result<()> {
        let node = self.node(inode)?;
        if node.readonly {
            return Err(Error::Errno("read-only"));
        }
        if size > MAXIMUM_FILE_BYTES {
            return Err(Error::Errno("file-too-large"));
        }
        self.preflight_operations(&node.volume, 1)?;
        self.preflight_growth(node, size)?;
        let (volume, previous) = (node.volume.clone(), node.data.len() as u64);
        if !volume.is_empty() {
            self.volumes
                .get_mut(&volume)
                .unwrap()
                .add_operations(vec![resize_plan(inode, size, now)]);
        }
        if size > previous && size - previous > MAXIMUM_TOTAL_BYTES - self.used_bytes {
            return Err(Error::Errno("no-space"));
        }
        self.used_bytes = self.used_bytes - previous + size;
        let node = self.node_mut(inode)?;
        node.data.resize(
            usize::try_from(size).expect("file length is at most 256 MiB"),
            0,
        );
        node.mod_time = now;
        Ok(())
    }

    pub(crate) fn sync_node(&mut self, inode: u64) -> Result<Vec<u64>> {
        let node = self.node(inode)?;
        if node.volume.is_empty() {
            return Ok(Vec::new());
        }
        let directory = node.kind == Kind::Directory;
        let id = node.volume.clone();
        let volume = self.volumes.get_mut(&id).unwrap();
        let mut selected = BTreeSet::new();
        for operation in &volume.pending {
            let effect = &operation.effect;
            let matches = if directory {
                effect.parents.contains(&inode)
                    || effect.parent == inode
                    || effect.inode == inode && operation.kind == "metadata"
            } else {
                effect.inode == inode && operation.kind != "namespace"
            };
            if matches {
                selected.insert(operation.id);
            }
        }
        volume.add_dependency_closure(&mut selected);
        let selected_ids = volume
            .pending
            .iter()
            .filter(|operation| selected.contains(&operation.id))
            .map(|operation| operation.id)
            .collect();
        volume.persist_selected(&selected)?;
        Ok(selected_ids)
    }

    pub(crate) fn operations(&self) -> BTreeMap<String, Vec<Operation>> {
        self.volumes
            .iter()
            .map(|(id, volume)| {
                (
                    id.clone(),
                    volume.history.iter().map(observed_operation).collect(),
                )
            })
            .collect()
    }

    #[cfg_attr(
        feature = "conformance",
        allow(
            dead_code,
            reason = "The complete volume port retains pending operations for storage transport, which conformance builds exclude."
        )
    )]
    pub(crate) fn pending_operations(&self) -> BTreeMap<String, Vec<Operation>> {
        self.volumes
            .iter()
            .map(|(id, volume)| {
                (
                    id.clone(),
                    volume.pending.iter().map(observed_operation).collect(),
                )
            })
            .collect()
    }

    pub(crate) fn volume_snapshots(&self) -> Vec<VolumeSnapshot> {
        self.volumes
            .values()
            .map(|volume| {
                let mut snapshot = VolumeSnapshot {
                    id: volume.id.clone(),
                    mount: volume.mount.clone(),
                    capacity_bytes: volume.capacity_bytes,
                    persisted: snapshot_entries(&volume.persisted),
                    volatile: self
                        .nodes
                        .iter()
                        .filter(|(_, node)| node.volume == volume.id)
                        .map(|(path, node)| {
                            snapshot_entry(
                                volume.relative(path),
                                node.mode,
                                node.kind,
                                node.mod_time,
                                &node.data,
                            )
                        })
                        .collect(),
                    pending_operations: volume.pending.len() as u64,
                    pending_sha256: volume.pending_identity(),
                    next_operation: volume.next_operation,
                    identity: String::new(),
                };
                snapshot.identity = volume_snapshot_identity(&snapshot);
                snapshot
            })
            .collect()
    }

    pub(super) fn rebuild_after_lifecycle(&mut self) -> Result<()> {
        self.preflight_rebuild()?;
        self.detached.clear();
        self.generation += 1;
        self.cwd = "/".into();
        self.reset_nodes(0)?;
        self.used_bytes = 0;
        let mut mounts: Vec<_> = self
            .volumes
            .values()
            .map(|volume| (volume.mount.clone(), volume.id.clone()))
            .collect();
        mounts.sort();
        for (mount, id) in mounts {
            let mut current = String::new();
            for component in mount[1..].split('/') {
                current.push('/');
                current.push_str(component);
                if self.nodes.contains_key(&current) {
                    continue;
                }
                // Go consumes an inode even when replacing this provisional mount node.
                let mut node = Node::new(self.allocate_inode()?, 0o755, Kind::Directory, 0, "");
                if current == mount {
                    let volume = &self.volumes[&id];
                    node = node_for_disk_object(
                        &volume.persisted.objects[&volume.persisted.paths["/"]],
                        &id,
                    );
                    node.mount_root = true;
                }
                self.nodes.insert(current.clone(), node);
            }
            let volume = &self.volumes[&id];
            for (relative, inode) in &volume.persisted.paths {
                if relative == "/" {
                    continue;
                }
                let node = node_for_disk_object(&volume.persisted.objects[inode], &id);
                self.used_bytes += node.data.len() as u64;
                self.nodes.insert(format!("{mount}{relative}"), node);
            }
        }
        Ok(())
    }
}

fn mount_inode_allocations<'a>(
    mounts: impl Iterator<Item = &'a str>,
    mut paths: BTreeSet<String>,
) -> u64 {
    let existing = paths.len();
    for mount in mounts {
        let mut current = String::new();
        for component in mount[1..].split('/') {
            current.push('/');
            current.push_str(component);
            paths.insert(current.clone());
        }
    }
    (paths.len() - existing) as u64
}

impl Node {
    pub(super) fn new(inode: u64, mode: u32, kind: Kind, mod_time: i64, volume: &str) -> Self {
        Self {
            inode,
            mode,
            kind,
            mod_time,
            data: FileBytes::default(),
            volume: volume.into(),
            linked: true,
            mount_root: false,
            readonly: false,
            handles: 0,
        }
    }
}

impl VolumeState {
    pub(super) fn relative(&self, path: &str) -> String {
        if path == self.mount {
            "/".into()
        } else {
            path.strip_prefix(&self.mount).unwrap_or(path).into()
        }
    }

    pub(super) fn add_operations(&mut self, plans: Vec<OperationPlan>) {
        for plan in plans {
            let deps: BTreeSet<_> = plan
                .depends
                .iter()
                .filter_map(|key| self.last.get(key).copied())
                .filter(|id| *id != 0)
                .collect();
            let operation = PersistenceOperation {
                id: self.next_operation,
                kind: plan.kind,
                deps: deps.into_iter().collect(),
                stores: plan.stores,
                effect: plan.effect,
            };
            for key in &operation.stores {
                self.last.insert(key.clone(), operation.id);
            }
            self.next_operation += 1;
            self.pending.push(operation.clone());
            self.history.push(operation);
        }
    }

    pub(super) fn has_pending(&self, id: u64) -> bool {
        self.pending.iter().any(|operation| operation.id == id)
    }

    pub(super) fn selection_valid(&self, selected: &BTreeSet<u64>) -> bool {
        let pending: BTreeSet<_> = self.pending.iter().map(|operation| operation.id).collect();
        self.pending
            .iter()
            .filter(|operation| selected.contains(&operation.id))
            .all(|operation| {
                operation.deps.iter().all(|dependency| {
                    !pending.contains(dependency) || selected.contains(dependency)
                })
            })
    }

    pub(super) fn add_dependency_closure(&self, selected: &mut BTreeSet<u64>) {
        let by_id: BTreeMap<_, _> = self
            .pending
            .iter()
            .map(|operation| (operation.id, operation))
            .collect();
        let mut queue: Vec<_> = selected.iter().copied().collect();
        while let Some(id) = queue.pop() {
            if let Some(operation) = by_id.get(&id) {
                for dependency in &operation.deps {
                    if by_id.contains_key(dependency) && selected.insert(*dependency) {
                        queue.push(*dependency);
                    }
                }
            }
        }
    }

    pub(super) fn persist_selected(&mut self, selected: &BTreeSet<u64>) -> Result<()> {
        for operation in &self.pending {
            if selected.contains(&operation.id) {
                apply_persistence_effect(&mut self.persisted, &operation.effect)?;
            }
        }
        self.pending
            .retain(|operation| !selected.contains(&operation.id));
        self.rebuild_last();
        Ok(())
    }

    pub(super) fn rebuild_last(&mut self) {
        self.last.clear();
        for operation in &self.pending {
            for key in &operation.stores {
                self.last.insert(key.clone(), operation.id);
            }
        }
    }

    pub(super) fn pending_identity(&self) -> String {
        let mut hasher = Hasher::new();
        hasher.string("gomad-volume-pending/v1");
        hasher.string(&self.id);
        for operation in &self.pending {
            hasher.u64(operation.id);
            hasher.string(&operation.kind);
            for dependency in &operation.deps {
                hasher.u64(*dependency);
            }
            write_effect_hash(&mut hasher, &operation.effect);
        }
        hasher.finish()
    }
}

pub(super) fn normalize(name: &str) -> Result<String> {
    if name.is_empty() || name.len() > 4096 || name.contains('\0') {
        return Err(Error::Errno("invalid"));
    }
    let mut components = Vec::new();
    for component in name.split('/') {
        match component {
            "" | "." => {}
            ".." => {
                if components.pop().is_none() {
                    return Err(Error::Errno("invalid"));
                }
            }
            _ => components.push(component),
        }
    }
    Ok(format!("/{}", components.join("/")))
}

pub(super) fn parent_path(path: &str) -> &str {
    match path.rsplit_once('/') {
        Some(("", _)) | None => "/",
        Some((parent, _)) => parent,
    }
}

pub(super) fn alloc_key(inode: u64) -> String {
    format!("alloc:{inode}")
}
pub(super) fn resize_key(inode: u64) -> String {
    format!("resize:{inode}")
}
pub(super) fn write_key(inode: u64) -> String {
    format!("write:{inode}")
}
pub(super) fn metadata_key(inode: u64) -> String {
    format!("metadata:{inode}")
}
pub(super) fn namespace_key(path: &str) -> String {
    format!("namespace:{path}")
}

fn resize_plan(inode: u64, size: u64, mod_time: i64) -> OperationPlan {
    OperationPlan {
        kind: "resize".into(),
        effect: Effect {
            kind: "resize".into(),
            inode,
            size,
            mod_time,
            ..Effect::default()
        },
        stores: vec![resize_key(inode)],
        depends: vec![alloc_key(inode), resize_key(inode)],
    }
}

pub(super) fn disk_object_for_node(node: &Node) -> DiskObject {
    DiskObject {
        inode: node.inode,
        mode: node.mode,
        kind: node.kind,
        mod_time: node.mod_time,
        data: node.data.clone(),
    }
}

pub(super) fn node_for_disk_object(object: &DiskObject, volume: &str) -> Node {
    let mut node = Node::new(
        object.inode,
        object.mode,
        object.kind,
        object.mod_time,
        volume,
    );
    node.data = object.data.clone();
    node
}

impl DiskState {
    pub(super) fn mutable_object(&mut self, inode: u64) -> Option<&mut DiskObject> {
        if !self.objects.contains_key(&inode) {
            return None;
        }
        self.owned.insert(inode);
        self.objects.get_mut(&inode)
    }
}

pub(super) fn apply_persistence_effect(state: &mut DiskState, effect: &Effect) -> Result<()> {
    match effect.kind.as_str() {
        "alloc" => {
            let object = effect
                .object
                .as_ref()
                .ok_or(Error::Invalid("invalid durable allocation effect"))?;
            if state.objects.contains_key(&effect.inode) {
                return Err(Error::Invalid("invalid durable allocation effect"));
            }
            state.objects.insert(effect.inode, object.clone());
            state.owned.insert(effect.inode);
        }
        "resize" => {
            let object = state
                .mutable_object(effect.inode)
                .ok_or(Error::Invalid("invalid durable resize effect"))?;
            if object.kind != Kind::File || effect.size > MAXIMUM_FILE_BYTES {
                return Err(Error::Invalid("invalid durable resize effect"));
            }
            object.data.resize(
                usize::try_from(effect.size).expect("file length is at most 256 MiB"),
                0,
            );
            object.mod_time = effect.mod_time;
        }
        "write" => {
            let object = state
                .mutable_object(effect.inode)
                .ok_or(Error::Invalid("invalid durable write effect"))?;
            let end = effect
                .offset
                .checked_add(effect.data.len() as u64)
                .filter(|end| *end <= object.data.len() as u64)
                .ok_or(Error::Invalid("invalid durable write effect"))?;
            if object.kind != Kind::File {
                return Err(Error::Invalid("invalid durable write effect"));
            }
            object.data[usize::try_from(effect.offset)
                .expect("offset is bounded by the file length")
                ..usize::try_from(end).expect("file length is at most 256 MiB")]
                .copy_from_slice(&effect.data);
            object.mod_time = effect.mod_time;
        }
        "metadata" => {
            let object = state
                .mutable_object(effect.inode)
                .ok_or(Error::Invalid("invalid durable metadata effect"))?;
            object.mode = effect.mode;
            object.mod_time = effect.mod_time;
        }
        "namespace" => {
            for part in &effect.namespace {
                if part.path == "/" {
                    return Err(Error::Invalid("invalid durable root namespace effect"));
                }
                if part.inode == 0 {
                    state.paths.remove(&part.path);
                    continue;
                }
                if !state.objects.contains_key(&part.inode) {
                    return Err(Error::Invalid(
                        "durable namespace effect refers to an unavailable object",
                    ));
                }
                state.paths.insert(part.path.clone(), part.inode);
            }
            if effect.inode != 0
                && let Some(object) = state.mutable_object(effect.inode)
            {
                object.mod_time = effect.mod_time;
            }
        }
        _ => return Err(Error::Invalid("unknown durable persistence effect")),
    }
    Ok(())
}

pub(super) fn observed_operation(operation: &PersistenceOperation) -> Operation {
    let effect = &operation.effect;
    Operation {
        id: operation.id,
        kind: operation.kind.clone(),
        dependencies: (!operation.deps.is_empty()).then(|| operation.deps.clone()),
        inode: effect.inode,
        offset: effect.offset,
        bytes: if effect.data.is_empty() {
            effect.size
        } else {
            effect.data.len() as u64
        },
        payload_sha256: if effect.data.is_empty() {
            String::new()
        } else {
            bytes_digest(&effect.data)
        },
        effect_sha256: persistence_effect_identity(effect),
        path: effect
            .namespace
            .first()
            .map(|part| part.path.clone())
            .unwrap_or_default(),
        destination: if effect.namespace.len() > 1 {
            effect.namespace.last().unwrap().path.clone()
        } else {
            String::new()
        },
    }
}

pub(super) fn snapshot_entry(
    path: String,
    mode: u32,
    kind: Kind,
    mod_time: i64,
    data: &FileBytes,
) -> SnapshotEntry {
    SnapshotEntry {
        path,
        mode,
        kind: if kind == Kind::Directory {
            "directory"
        } else {
            "file"
        }
        .into(),
        mod_time,
        size: data.len() as u64,
        data_sha256: if kind == Kind::File {
            data.digest().into()
        } else {
            String::new()
        },
    }
}

pub(super) fn snapshot_entries(state: &DiskState) -> Vec<SnapshotEntry> {
    state
        .paths
        .iter()
        .map(|(path, inode)| {
            let object = &state.objects[inode];
            snapshot_entry(
                path.clone(),
                object.mode,
                object.kind,
                object.mod_time,
                &object.data,
            )
        })
        .collect()
}

pub(super) fn write_snapshot_entries(hasher: &mut Hasher, entries: &[SnapshotEntry]) {
    hasher.u64(entries.len() as u64);
    for entry in entries {
        hasher.string(&entry.path);
        hasher.u64(u64::from(entry.mode));
        hasher.string(&entry.kind);
        hasher.u64(entry.mod_time.cast_unsigned());
        hasher.u64(entry.size);
        hasher.string(&entry.data_sha256);
    }
}

pub(crate) fn volume_snapshot_identity(snapshot: &VolumeSnapshot) -> String {
    let mut hasher = Hasher::new();
    hasher.string("gomad-volume-snapshot/v1");
    hasher.string(&snapshot.id);
    hasher.string(&snapshot.mount);
    hasher.u64(snapshot.capacity_bytes);
    write_snapshot_entries(&mut hasher, &snapshot.persisted);
    write_snapshot_entries(&mut hasher, &snapshot.volatile);
    hasher.u64(snapshot.pending_operations);
    hasher.string(&snapshot.pending_sha256);
    hasher.u64(snapshot.next_operation);
    hasher.finish()
}

pub(super) fn persistence_effect_identity(effect: &Effect) -> String {
    let mut hasher = Hasher::new();
    hasher.string("gomad-volume-effect/v1");
    write_effect_hash(&mut hasher, effect);
    hasher.finish()
}

pub(super) fn write_effect_hash(hasher: &mut Hasher, effect: &Effect) {
    hasher.string(&effect.kind);
    hasher.u64(effect.inode);
    hasher.u64(effect.parent);
    for parent in &effect.parents {
        hasher.u64(*parent);
    }
    hasher.u64(effect.offset);
    hasher.u64(effect.size);
    hasher.u64(u64::from(effect.mode));
    hasher.u64(effect.mod_time.cast_unsigned());
    hasher.bytes(&effect.data);
    if let Some(object) = &effect.object {
        hasher.u64(object.inode);
        hasher.u64(u64::from(object.mode));
        hasher.u64(object.kind as u64);
        hasher.u64(object.mod_time.cast_unsigned());
        hasher.bytes(&object.data);
    }
    for part in &effect.namespace {
        hasher.string(&part.path);
        hasher.u64(part.inode);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{Engine, engine::general_purpose::STANDARD};

    fn filesystem(capacity_bytes: u64, pending_operations: u64) -> Fs {
        let mut fs = Fs::new(0);
        fs.configure_volumes(
            &[VolumeConfig {
                id: "db".into(),
                mount: "/data".into(),
                capacity_bytes,
            }],
            VolumeLimits {
                pending_operations,
                transitions: 100,
            },
            0,
        )
        .unwrap();
        fs
    }

    #[test]
    #[cfg(not(feature = "conformance"))]
    fn exploration_persistence_survives_a_crash_without_pending_selection() {
        use crate::linked_simulation::{Controller, Plan};
        use std::sync::{Arc, Mutex};

        let fixture = || {
            let mut fs = filesystem(4096, 100);
            let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
            fs.write_at(inode, 0, b"original", 1).unwrap();
            fs.sync_node(inode).unwrap();
            let root = fs.lookup("/data").unwrap().inode;
            fs.sync_node(root).unwrap();
            fs.write_at(inode, 0, b"survives", 2).unwrap();
            fs
        };
        let mut fs = fixture();
        let mut probe = fixture();
        let controller = Arc::new(Mutex::new(
            Controller::new(
                Plan::new(
                    format!("sha256:{}", "a".repeat(64)),
                    format!("sha256:{}", "b".repeat(64)),
                    7,
                    vec![],
                )
                .unwrap(),
                100,
            )
            .unwrap(),
        ));
        probe.explore_persistence(&controller).unwrap();
        let decision = controller.lock().unwrap().decisions()[0].clone();
        let overridden = Arc::new(Mutex::new(
            Controller::new(
                Plan::new(
                    format!("sha256:{}", "a".repeat(64)),
                    format!("sha256:{}", "b".repeat(64)),
                    7,
                    vec![crate::linked_simulation::Override::from_decision(&decision, 1).unwrap()],
                )
                .unwrap(),
                100,
            )
            .unwrap(),
        ));
        fs.explore_persistence(&overridden).unwrap();
        fs.crash_volumes(&BTreeMap::new()).unwrap();
        let recovered = fs.lookup("/data/file").unwrap().inode;
        assert_eq!(fs.read_at(recovered, 0, 32).unwrap(), b"survives");
        probe.crash_volumes(&BTreeMap::new()).unwrap();
        let lost = probe.lookup("/data/file").unwrap().inode;
        assert_eq!(probe.read_at(lost, 0, 32).unwrap(), b"original");
    }

    #[test]
    fn repeated_snapshots_do_not_rehash_unchanged_file_bytes() {
        use super::super::hash::CONTENT_HASHES;
        let mut fs = filesystem(4096, 100);
        let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
        fs.write_at(inode, 0, b"abc", 2).unwrap();
        fs.sync_node(inode).unwrap();
        fs.sync_node(3).unwrap();
        CONTENT_HASHES.with(|count| count.set(0));
        fs.volume_snapshots();
        let hashed = CONTENT_HASHES.with(std::cell::Cell::get);
        assert_eq!(
            hashed, 2,
            "volatile and persisted byte owners each hash once"
        );
        for _ in 0..3 {
            fs.volume_snapshots();
        }
        assert_eq!(CONTENT_HASHES.with(std::cell::Cell::get), hashed);
        fs.chmod("/data/file", 0o640, 3).unwrap();
        fs.volume_snapshots();
        assert_eq!(CONTENT_HASHES.with(std::cell::Cell::get), hashed);
    }

    #[test]
    fn detached_open_file_preserves_data_until_the_last_close() {
        let mut fs = filesystem(5, 100);
        let mut handle = fs
            .open(
                "/data/file",
                OpenFlags {
                    read: true,
                    write: true,
                    create: true,
                    ..OpenFlags::default()
                },
                0o600,
                1,
            )
            .unwrap();
        fs.write_at(handle.inode, 0, b"abc", 2).unwrap();
        fs.remove("/data/file").unwrap();
        fs.write_at(handle.inode, 3, b"xy", 3).unwrap();
        assert_eq!(fs.sync_node(handle.inode).unwrap(), vec![1, 3, 4, 6, 7]);
        fs.close(&mut handle).unwrap();
        assert!(fs.detached.is_empty());
    }

    #[test]
    fn allocation_write_and_sync_keep_link_separate_from_data() {
        let mut fs = filesystem(100, 100);
        let inode = fs.create("/data/file", Kind::File, 0o640, 7).unwrap();
        fs.write_at(inode, 2, b"xy", 9).unwrap();
        assert_eq!(fs.sync_node(inode).unwrap(), vec![1, 3, 4]);
        assert_eq!(fs.sync_node(3).unwrap(), vec![2]);
    }

    #[test]
    fn directory_metadata_and_unlink_sync_match_go_selection() {
        let mut fs = filesystem(100, 100);
        let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
        fs.metadata(inode, 0o644, -7).unwrap();
        assert_eq!(fs.sync_node(3).unwrap(), vec![1, 2]);
        fs.remove("/data/file").unwrap();
        assert_eq!(fs.sync_node(3).unwrap(), vec![4]);
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The scenario retains the complete ordered Go conformance sequence"
    )]
    fn pure_core_matches_every_go_metadata_namespace_and_lifecycle_step() {
        let vectors: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/core.json"
        ))
        .unwrap();
        for scenario in vectors["scenarios"].as_array().unwrap() {
            let mut fs = Fs::new(0);
            let mut handles: BTreeMap<String, Handle> = BTreeMap::new();
            let mut source: Option<Fs> = None;
            for (index, step) in scenario["steps"].as_array().unwrap().iter().enumerate() {
                let input = &step["input"];
                let now = step["now"].as_i64().unwrap();
                let text = |key: &str| input[key].as_str().unwrap_or_default();
                let integer = |key: &str| input[key].as_i64().unwrap_or(0);
                let op = step["op"].as_str().unwrap();
                match op {
                    op if op.starts_with("api-") => {
                        let mut value = 0i64;
                        let mut data = Vec::new();
                        let mut names: Vec<String> = Vec::new();
                        let result: Result<()> = (|| {
                            match op {
                                "api-open" => {
                                    let flags = &input["flags"];
                                    let flag = |key: &str| flags[key].as_bool().unwrap();
                                    let handle = fs.open(
                                        text("path"),
                                        OpenFlags {
                                            read: flag("Read"),
                                            write: flag("Write"),
                                            append: flag("Append"),
                                            create: flag("Create"),
                                            exclusive: flag("Exclusive"),
                                            truncate: flag("Truncate"),
                                        },
                                        u32::try_from(integer("mode")).unwrap(),
                                        now,
                                    )?;
                                    handles.insert(text("handle").into(), handle);
                                }
                                "api-write" => {
                                    value = i64::try_from(fs.handle_write(
                                        handles.get_mut(text("handle")).unwrap(),
                                        &STANDARD.decode(text("data_base64")).unwrap(),
                                        now,
                                    )?)
                                    .unwrap();
                                }
                                "api-write-at" => {
                                    value = i64::try_from(fs.handle_write_at(
                                        &handles[text("handle")],
                                        integer("offset"),
                                        &STANDARD.decode(text("data_base64")).unwrap(),
                                        now,
                                    )?)
                                    .unwrap();
                                }
                                "api-seek" => {
                                    value = (fs.handle_seek(
                                        handles.get_mut(text("handle")).unwrap(),
                                        integer("offset"),
                                        integer("whence").to_le_bytes()[0],
                                    )?)
                                    .cast_signed();
                                }
                                "api-read" => {
                                    data = fs.handle_read(
                                        handles.get_mut(text("handle")).unwrap(),
                                        usize::try_from(integer("size")).unwrap(),
                                    )?;
                                    value = i64::try_from(data.len()).unwrap();
                                }
                                "api-read-dir" => {
                                    names = fs
                                        .handle_read_dir(
                                            handles.get_mut(text("handle")).unwrap(),
                                            integer("size"),
                                        )?
                                        .into_iter()
                                        .map(|entry| entry.0)
                                        .collect();
                                }
                                "api-handle-chmod" => fs.handle_chmod(
                                    &handles[text("handle")],
                                    u32::try_from(integer("mode")).unwrap(),
                                    now,
                                )?,
                                "api-handle-chtimes" => fs.handle_chtimes(
                                    &handles[text("handle")],
                                    integer("mod_time"),
                                )?,
                                "api-handle-chdir" => fs.handle_chdir(&handles[text("handle")])?,
                                "api-chdir" => fs.chdir(text("path"))?,
                                "api-mkdir" => fs.mkdir(
                                    text("path"),
                                    u32::try_from(integer("mode")).unwrap(),
                                    now,
                                )?,
                                "api-mkdir-all" => fs.mkdir_all(
                                    text("path"),
                                    u32::try_from(integer("mode")).unwrap(),
                                    now,
                                )?,
                                "api-remove-all" => fs.remove_all(text("path"))?,
                                "api-remove" => fs.remove(text("path"))?,
                                "api-chmod" => fs.chmod(
                                    text("path"),
                                    u32::try_from(integer("mode")).unwrap(),
                                    now,
                                )?,
                                "api-rename" => {
                                    fs.rename(text("path"), text("destination"), now)?;
                                }
                                "api-resize" => fs.handle_resize(
                                    &handles[text("handle")],
                                    integer("size"),
                                    now,
                                )?,
                                "api-close" => {
                                    fs.close(handles.get_mut(text("handle")).unwrap())?;
                                }
                                "api-sync" => {
                                    fs.handle_sync(&handles[text("handle")])?;
                                }
                                "api-advance" => {
                                    let next = fs.advance_lifecycle(false, &BTreeMap::new())?;
                                    source = Some(std::mem::replace(&mut fs, next));
                                }
                                "api-select" => assert_eq!(
                                    serde_json::to_value(
                                        fs.select_crash_operations(input["seed"].as_u64().unwrap())
                                    )
                                    .unwrap(),
                                    step["selections"]
                                ),
                                "api-source-stat" => {
                                    source.as_ref().unwrap().lookup(text("path"))?;
                                }
                                "api-flush" => fs.flush_volumes()?,
                                "api-crash" => {
                                    let selections = input
                                        .get("selections")
                                        .map(|value| serde_json::from_value(value.clone()).unwrap())
                                        .unwrap_or_default();
                                    fs.crash_volumes(&selections)?;
                                }
                                "api-clone" => {
                                    let next = fs.clone_durable()?;
                                    source = Some(std::mem::replace(&mut fs, next));
                                }
                                "api-source" => std::mem::swap(&mut fs, source.as_mut().unwrap()),
                                "api-observe" => {}
                                "api-captured-file" => {
                                    let path = text("path").to_owned();
                                    assert!(!fs.nodes.contains_key(&path));
                                    let data = STANDARD.decode(text("data_base64")).unwrap();
                                    let inode = fs.allocate_inode()?;
                                    let mut node = Node::new(
                                        inode,
                                        u32::try_from(integer("mode")).unwrap() & 0o777,
                                        Kind::File,
                                        now,
                                        "",
                                    );
                                    node.readonly = true;
                                    fs.used_bytes += data.len() as u64;
                                    node.data = data.into();
                                    fs.nodes.insert(path, node);
                                }
                                "api-revoke" => {
                                    fs.advance_lifecycle(false, &BTreeMap::new())?;
                                }
                                _ => panic!("unknown API vector operation {op}"),
                            }
                            Ok(())
                        })();
                        if let Some(expected) = step.get("capacity") {
                            let Err(Error::Capacity {
                                resource,
                                required,
                                maximum,
                            }) = &result
                            else {
                                panic!("expected capacity error")
                            };
                            assert_eq!(*resource, expected["Resource"].as_str().unwrap());
                            assert_eq!(*required, expected["Required"].as_u64().unwrap());
                            assert_eq!(*maximum, expected["Maximum"].as_u64().unwrap());
                        }
                        let error = match result {
                            Ok(()) => String::new(),
                            Err(Error::Errno(name) | Error::Invalid(name)) => name.into(),
                            Err(error @ Error::Capacity { .. }) => {
                                super::super::faults::error_text(&error)
                            }
                        };
                        assert_eq!(
                            error,
                            step["error"].as_str().unwrap_or_default(),
                            "{} step {index} {op}",
                            scenario["name"]
                        );
                        assert_eq!(
                            value,
                            step["value"].as_i64().unwrap_or_default(),
                            "{} step {index} {op} value",
                            scenario["name"]
                        );
                        assert_eq!(
                            data,
                            STANDARD
                                .decode(step["data_base64"].as_str().unwrap_or_default())
                                .unwrap()
                        );
                        let expected_names: Vec<String> = step["names"]
                            .as_array()
                            .map(|names| {
                                names
                                    .iter()
                                    .map(|name| name.as_str().unwrap().into())
                                    .collect()
                            })
                            .unwrap_or_default();
                        assert_eq!(names, expected_names);
                        if let Some(path) = step["handle_path"].as_str() {
                            assert_eq!(handles[text("handle")].name, path);
                        }
                    }
                    "configure" => {
                        let configs: Vec<VolumeConfig> =
                            serde_json::from_value(scenario["storage"]["volumes"].clone()).unwrap();
                        let limits: VolumeLimits =
                            serde_json::from_value(scenario["storage"]["limits"].clone()).unwrap();
                        let result = fs.configure_volumes(&configs, limits, now);
                        if let Some(expected) = step["error"].as_str() {
                            let Err(Error::Invalid(message)) = result else {
                                panic!("expected configuration refusal")
                            };
                            assert_eq!(message, expected);
                            assert!(!fs.configured);
                        } else {
                            result.unwrap();
                        }
                    }
                    "open" => {
                        let flags = &input["flags"];
                        let flag = |key: &str| flags[key].as_bool().unwrap();
                        let handle = fs
                            .open(
                                text("path"),
                                OpenFlags {
                                    read: flag("Read"),
                                    write: flag("Write"),
                                    append: flag("Append"),
                                    create: flag("Create"),
                                    exclusive: flag("Exclusive"),
                                    truncate: flag("Truncate"),
                                },
                                u32::try_from(integer("mode")).unwrap(),
                                now,
                            )
                            .unwrap();
                        handles.insert(text("handle").into(), handle);
                    }
                    "write-at" => {
                        let data = STANDARD.decode(text("data_base64")).unwrap();
                        let written = fs
                            .handle_write_at(
                                &handles[text("handle")],
                                integer("offset"),
                                &data,
                                now,
                            )
                            .unwrap();
                        assert_eq!(written, data.len());
                    }
                    "chmod" => fs
                        .chmod(text("path"), u32::try_from(integer("mode")).unwrap(), now)
                        .unwrap(),
                    "chtimes" => fs.chtimes(text("path"), integer("mod_time")).unwrap(),
                    "mkdir" => fs
                        .mkdir(text("path"), u32::try_from(integer("mode")).unwrap(), now)
                        .unwrap(),
                    "mkdir-all" => fs
                        .mkdir_all(text("path"), u32::try_from(integer("mode")).unwrap(), now)
                        .unwrap(),
                    "rename" => fs.rename(text("path"), text("destination"), now).unwrap(),
                    "select" => assert_eq!(
                        serde_json::to_value(
                            fs.select_crash_operations(input["seed"].as_u64().unwrap())
                        )
                        .unwrap(),
                        step["selections"],
                        "{} step {index} {op}",
                        scenario["name"]
                    ),
                    "file-sync" => {
                        fs.handle_sync(&handles[text("handle")]).unwrap();
                    }
                    "directory-sync" => {
                        let mut handle = fs
                            .open(
                                text("path"),
                                OpenFlags {
                                    read: true,
                                    ..OpenFlags::default()
                                },
                                0,
                                now,
                            )
                            .unwrap();
                        fs.handle_sync(&handle).unwrap();
                        fs.close(&mut handle).unwrap();
                    }
                    "remove" => fs.remove(text("path")).unwrap(),
                    "truncate" => fs
                        .handle_resize(&handles[text("handle")], integer("size"), now)
                        .unwrap(),
                    "close" => fs.close(handles.get_mut(text("handle")).unwrap()).unwrap(),
                    "clone-durable" => fs = fs.clone_durable().unwrap(),
                    "advance-crash" | "advance-graceful" => {
                        let selections = if input["selections"].is_object() {
                            serde_json::from_value(input["selections"].clone()).unwrap()
                        } else {
                            BTreeMap::new()
                        };
                        fs = fs
                            .advance_lifecycle(op == "advance-graceful", &selections)
                            .unwrap();
                    }
                    _ => panic!("unknown Go core vector operation {op}"),
                }
                if let Some(cwd) = step["cwd"].as_str() {
                    assert_eq!(fs.cwd, cwd, "{} step {index} cwd", scenario["name"]);
                }
                if let Some(used) = step["used_bytes"].as_u64() {
                    assert_eq!(
                        fs.used_bytes, used,
                        "{} step {index} used bytes",
                        scenario["name"]
                    );
                }
                assert_eq!(
                    serde_json::to_value(fs.volume_snapshots()).unwrap(),
                    step["snapshots"],
                    "{} step {index} {op} snapshots",
                    scenario["name"]
                );
                assert_eq!(
                    serde_json::to_value(fs.operations()).unwrap(),
                    step["operations"],
                    "{} step {index} {op} operations",
                    scenario["name"]
                );
                assert_eq!(
                    fs.next_inode,
                    step["state"]["NextInode"].as_u64().unwrap(),
                    "{} step {index} {op} inode counter",
                    scenario["name"]
                );
                assert_eq!(
                    fs.generation,
                    step["state"]["Generation"].as_u64().unwrap(),
                    "{} step {index} {op} generation",
                    scenario["name"]
                );
                if let Some(image) = step["image_base64"].as_str() {
                    let expected = STANDARD.decode(image).unwrap();
                    let model_length =
                        usize::try_from(u64::from_le_bytes(expected[28..36].try_into().unwrap()))
                            .unwrap();
                    let model = std::str::from_utf8(&expected[36..36 + model_length]).unwrap();
                    let got = super::super::image::RecoveryImage::encode(
                        &fs,
                        model,
                        &BTreeMap::new(),
                        1 << 20,
                    )
                    .unwrap();
                    assert_eq!(
                        got, expected,
                        "{} step {index} {op} full recovery state",
                        scenario["name"]
                    );
                    assert_eq!(
                        super::super::image::image_digest(&got),
                        step["image_digest"].as_str().unwrap()
                    );
                }
            }
        }
    }

    #[test]
    fn remove_all_checks_busy_working_directory_before_readonly() {
        let mut fs = filesystem(100, 100);
        fs.mkdir("/data/dir", 0o755, 1).unwrap();
        fs.chdir("/data/dir").unwrap();
        fs.nodes.get_mut("/data/dir").unwrap().readonly = true;
        assert_eq!(fs.remove_all("/data/dir"), Err(Error::Errno("busy")));
        assert_eq!(fs.remove("/data/dir"), Err(Error::Errno("read-only")));
        assert!(fs.nodes.contains_key("/data/dir"));
        assert_eq!(fs.volumes["db"].history.len(), 2);
    }

    #[test]
    fn exhausted_inode_counter_refuses_create_and_planned_mkdir_without_wrap() {
        let mut fs = filesystem(100, 100);
        fs.next_inode = u64::MAX;
        let before = fs.volume_snapshots();
        assert_eq!(
            fs.create("/data/refused", Kind::File, 0o600, 1),
            Err(Error::Errno("no-space"))
        );
        assert_eq!(fs.next_inode, u64::MAX);
        assert_eq!(fs.volume_snapshots(), before);
        fs.next_inode = u64::MAX - 1;
        assert_eq!(
            fs.mkdir_all("/data/a/b", 0o755, 1),
            Err(Error::Errno("no-space"))
        );
        assert_eq!(fs.next_inode, u64::MAX);
        assert_eq!(fs.volume_snapshots(), before);
        assert!(fs.lookup("/data/a").is_err());
    }

    #[test]
    fn exhausted_lifecycle_inode_headroom_refuses_before_durable_or_volatile_mutation() {
        let mut fs = filesystem(100, 100);
        let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
        fs.write_at(inode, 0, b"pending", 2).unwrap();
        fs.next_inode = u64::MAX - 1;
        let before = fs.volume_snapshots();
        assert_eq!(fs.flush_volumes(), Err(Error::Errno("no-space")));
        assert_eq!(fs.volume_snapshots(), before);
        assert_eq!(
            fs.crash_volumes(&BTreeMap::new()),
            Err(Error::Errno("no-space"))
        );
        assert_eq!(fs.volume_snapshots(), before);
        fs.next_inode = u64::MAX - 3;
        assert!(matches!(
            fs.advance_lifecycle(true, &BTreeMap::new()),
            Err(Error::Errno("no-space"))
        ));
        assert_eq!(fs.volume_snapshots(), before);
        assert_eq!(fs.next_inode, u64::MAX - 3);
        assert!(!fs.unavailable);
        fs.next_inode = u64::MAX;
        assert!(matches!(fs.clone_durable(), Err(Error::Errno("no-space"))));
    }

    #[test]
    fn rejected_file_bounds_leave_data_history_and_offsets_unchanged() {
        let mut fs = filesystem(100, 100);
        let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
        fs.write_at(inode, 0, b"abc", 2).unwrap();
        let snapshot = fs.volume_snapshots();
        let operations = fs.operations();
        assert_eq!(
            fs.write_at(inode, u64::MAX, b"xy", 3),
            Err(Error::Errno("file-too-large"))
        );
        assert_eq!(fs.volume_snapshots(), snapshot);
        assert_eq!(fs.operations(), operations);
    }

    #[test]
    fn replacing_open_target_retains_its_inode_until_last_close() {
        let mut fs = filesystem(100, 100);
        let mut source = fs
            .open(
                "/data/source",
                OpenFlags {
                    read: true,
                    write: true,
                    create: true,
                    ..OpenFlags::default()
                },
                0o600,
                1,
            )
            .unwrap();
        let mut target = fs
            .open(
                "/data/target",
                OpenFlags {
                    read: true,
                    write: true,
                    create: true,
                    ..OpenFlags::default()
                },
                0o600,
                2,
            )
            .unwrap();
        fs.handle_write(&mut source, b"new", 3).unwrap();
        fs.handle_write(&mut target, b"old", 4).unwrap();
        fs.rename("/data/source", "/data/target", 5).unwrap();
        fs.close(&mut target).unwrap();
        assert!(fs.detached.is_empty());
        fs.close(&mut source).unwrap();
    }

    #[test]
    fn aggregate_byte_failure_keeps_go_record_before_growth_order() {
        let mut fs = filesystem(MAXIMUM_TOTAL_BYTES + 1, 100);
        let inode = fs.create("/data/file", Kind::File, 0o600, 1).unwrap();
        // Simulate a full aggregate charge without allocating a gigabyte in a unit test.
        fs.used_bytes = MAXIMUM_TOTAL_BYTES;
        assert_eq!(
            fs.write_at(inode, 0, b"x", 2),
            Err(Error::Errno("no-space"))
        );
        assert!(fs.node(inode).unwrap().data.is_empty());
        assert_eq!(fs.node(inode).unwrap().mod_time, 1);
        assert_eq!(fs.volumes["db"].next_operation, 5);
        assert_eq!(
            fs.volumes["db"]
                .pending
                .iter()
                .map(|operation| (operation.kind.as_str(), operation.deps.clone()))
                .collect::<Vec<_>>(),
            vec![
                ("alloc", vec![]),
                ("namespace", vec![1]),
                ("resize", vec![1]),
                ("write", vec![1, 3])
            ]
        );
        assert_eq!(fs.resize(inode, 1, 3), Err(Error::Errno("no-space")));
        let last = fs.volumes["db"].history.last().unwrap();
        assert_eq!(last.id, 5);
        assert_eq!(last.kind, "resize");
        assert_eq!(last.deps, vec![1, 3]);
        assert_eq!(fs.volumes["db"].next_operation, 6);
        assert_eq!(fs.used_bytes, MAXIMUM_TOTAL_BYTES);
    }
}
