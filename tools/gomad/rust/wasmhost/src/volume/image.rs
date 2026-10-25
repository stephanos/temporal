use super::hash::bytes_digest;
use super::state::{
    DiskObject, DiskState, Effect, Error, Fs, Kind, NamespaceEffect, PersistenceOperation, Result,
    VolumeConfig, VolumeLimits, VolumeState, normalize,
};
use std::collections::{BTreeMap, BTreeSet};

#[cfg(any(test, feature = "conformance"))]
pub(crate) const IMAGE_VERSION: &str = "gomad.volume-recovery-image/v1";

pub(crate) struct RecoveryImage {
    pub(super) fs: Fs,
    pub(crate) occurrences: BTreeMap<String, u64>,
    #[allow(
        dead_code,
        reason = "Retains the validated model identity for recovery-image conformance checks."
    )]
    pub(crate) model_identity: String,
    #[allow(
        dead_code,
        reason = "Retains the validated image identity for recovery-image conformance checks."
    )]
    pub(crate) identity: String,
}

impl RecoveryImage {
    #[cfg(not(feature = "conformance"))]
    pub(crate) fn volume_ids(&self) -> Vec<String> {
        self.fs.volumes.keys().cloned().collect()
    }

    #[cfg(not(feature = "conformance"))]
    pub(crate) fn observation(self) -> crate::linked_storage_inspection::RecoveryObservation {
        use crate::linked_storage_inspection::{RecoveryObservation, RecoveryVolume};
        RecoveryObservation {
            model_identity: self.model_identity,
            identity: self.identity,
            next_inode: self.fs.next_inode,
            generation: self.fs.generation,
            occurrences: self.occurrences,
            volumes: self
                .fs
                .volumes
                .into_values()
                .map(|volume| RecoveryVolume {
                    id: volume.id,
                    mount: volume.mount,
                    capacity_bytes: volume.capacity_bytes,
                    objects: volume
                        .persisted
                        .objects
                        .into_iter()
                        .map(|(inode, object)| (inode, observe_object(object)))
                        .collect(),
                    paths: volume.persisted.paths,
                    owned: volume.persisted.owned,
                    pending: volume.pending.into_iter().map(observe_operation).collect(),
                    history: volume.history.into_iter().map(observe_operation).collect(),
                    next_operation: volume.next_operation,
                    last: volume.last,
                })
                .collect(),
        }
    }

    pub(crate) fn encode(
        fs: &Fs,
        model: &str,
        occurrences: &BTreeMap<String, u64>,
        maximum: u64,
    ) -> Result<Vec<u8>> {
        let mut model_section = Writer::new(maximum);
        model_section.text(model)?;
        let mut occurrence_section = Writer::new(maximum);
        occurrence_section.number(occurrences.len() as u64)?;
        for (key, occurrence) in occurrences {
            occurrence_section.text(key)?;
            occurrence_section.number(*occurrence)?;
        }
        let mut state = Writer::new(maximum);
        state.number(fs.next_inode)?;
        state.number(fs.generation)?;
        state.number(fs.volumes.len() as u64)?;
        for volume in fs.volumes.values() {
            state.text(&volume.id)?;
            state.text(&volume.mount)?;
            state.number(volume.capacity_bytes)?;
            state.number(volume.persisted.objects.len() as u64)?;
            for object in volume.persisted.objects.values() {
                state.object(object)?;
            }
            state.number(volume.persisted.paths.len() as u64)?;
            for (path, inode) in &volume.persisted.paths {
                state.text(path)?;
                state.number(*inode)?;
            }
            state.numbers(volume.persisted.owned.iter().copied())?;
            state.operations(&volume.pending)?;
            state.operations(&volume.history)?;
            state.number(volume.next_operation)?;
            state.number(volume.last.len() as u64)?;
            for (key, operation) in &volume.last {
                state.text(key)?;
                state.number(*operation)?;
            }
        }
        let mut result = Writer::new(maximum);
        result.raw(b"GMVOLIMG")?;
        result.raw(&1u32.to_le_bytes())?;
        result.raw(&3u32.to_le_bytes())?;
        for (index, section) in [model_section.data, occurrence_section.data, state.data]
            .iter()
            .enumerate()
        {
            result.raw(
                &(u32::try_from(index).expect("the image contains three sections") + 1)
                    .to_le_bytes(),
            )?;
            result.blob(section)?;
        }
        Ok(result.data)
    }

    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub(crate) fn decode(
        bytes: &[u8],
        digest: &str,
        model: &str,
        configs: &[VolumeConfig],
        limits: VolumeLimits,
        lock_databases: &[String],
        maximum: u64,
    ) -> Result<Self> {
        if bytes.len() as u64 > maximum {
            return Err(Error::Capacity {
                resource: "image-bytes",
                required: bytes.len() as u64,
                maximum,
            });
        }
        if digest.len() != 64 || image_digest(bytes) != digest {
            return Err(Error::Invalid("recovery image digest mismatch"));
        }
        let mut frame = Reader::new(bytes);
        if frame.take(8)? != b"GMVOLIMG" || frame.small_number()? != 1 || frame.small_number()? != 3
        {
            return Err(Error::Invalid("recovery image format"));
        }
        let mut sections = Vec::new();
        for tag in 1..=3 {
            if frame.small_number()? != tag {
                return Err(Error::Invalid("recovery image sections"));
            }
            sections.push(frame.blob()?);
        }
        frame.finish()?;
        let mut model_section = Reader::new(sections[0]);
        let model_identity = model_section.text()?;
        model_section.finish()?;
        if model_identity != model || !super::crash::valid_digest(model) {
            return Err(Error::Invalid("recovery storage model identity"));
        }
        let mut occurrence_section = Reader::new(sections[1]);
        let mut occurrences = BTreeMap::new();
        let mut previous = None;
        for _ in 0..occurrence_section.count()? {
            let key = occurrence_section.text()?;
            sorted(&mut previous, &key)?;
            let count = occurrence_section.number()?;
            let (resource, operation) = key
                .split_once('\0')
                .ok_or(Error::Invalid("recovery fault occurrence"))?;
            if count == 0
                || normalize(resource).ok().as_deref() != Some(resource)
                || ![
                    "read", "write", "sync", "truncate", "open", "mkdir", "remove", "rename",
                ]
                .contains(&operation)
            {
                return Err(Error::Invalid("recovery fault occurrence"));
            }
            refuse_lock(resource, lock_databases)?;
            occurrences.insert(key, count);
        }
        occurrence_section.finish()?;
        let mut state = Reader::new(sections[2]);
        let next_inode = state.number()?;
        let generation = state.number()?;
        if generation == 0 || next_inode == 0 {
            return Err(Error::Invalid("recovery filesystem counters"));
        }
        let mut fs = Fs::new(0);
        fs.configure_volumes(configs, limits, 0)
            .map_err(|_| Error::Invalid("recovery volume configuration"))?;
        fs.volumes.clear();
        let mut previous = None;
        let mut maximum_inode = 0;
        let mut durable_bytes = 0u64;
        // Go's initial root and temp inode IDs never belong to volumes.
        let mut object_inodes = BTreeSet::from([1, 2]);
        for _ in 0..state.count()? {
            let id = state.text()?;
            sorted(&mut previous, &id)?;
            let mount = state.text()?;
            let capacity_bytes = state.number()?;
            if !configs.iter().any(|config| {
                config.id == id && config.mount == mount && config.capacity_bytes == capacity_bytes
            }) {
                return Err(Error::Invalid("recovery volume configuration"));
            }
            let mut persisted = DiskState::default();
            let mut object_previous = None;
            for _ in 0..state.count()? {
                let object = state.object()?;
                sorted_number(&mut object_previous, object.inode)?;
                if !object_inodes.insert(object.inode) {
                    return Err(Error::Invalid("recovery object inode alias"));
                }
                maximum_inode = maximum_inode.max(object.inode);
                persisted.objects.insert(object.inode, object);
            }
            let mut path_previous = None;
            let mut path_inodes = BTreeSet::new();
            for _ in 0..state.count()? {
                let path = state.text()?;
                sorted(&mut path_previous, &path)?;
                let inode = state.number()?;
                if normalize(&path).ok().as_deref() != Some(&path)
                    || !path.starts_with('/')
                    || !persisted.objects.contains_key(&inode)
                {
                    return Err(Error::Invalid("recovery durable tree"));
                }
                if !path_inodes.insert(inode) {
                    return Err(Error::Invalid("recovery path inode alias"));
                }
                refuse_lock(
                    &if path == "/" {
                        mount.clone()
                    } else {
                        format!("{mount}{path}")
                    },
                    lock_databases,
                )?;
                persisted.paths.insert(path, inode);
            }
            let mut owned_previous = None;
            for inode in state.numbers()? {
                sorted_number(&mut owned_previous, inode)?;
                if !persisted.objects.contains_key(&inode) {
                    return Err(Error::Invalid("recovery object ownership"));
                }
                persisted.owned.insert(inode);
            }
            let pending = state.operations()?;
            let history = state.operations()?;
            let next_operation = state.number()?;
            if next_operation != history.len() as u64 + 1 {
                return Err(Error::Invalid("recovery operation counter"));
            }
            for (index, operation) in history.iter().enumerate() {
                if operation.id != index as u64 + 1 {
                    return Err(Error::Invalid("recovery operation history"));
                }
                validate_operation(operation)?;
                maximum_inode = maximum_inode.max(operation.effect.inode).max(
                    operation
                        .effect
                        .object
                        .as_ref()
                        .map_or(0, |object| object.inode),
                );
                for effect in &operation.effect.namespace {
                    refuse_lock(&format!("{mount}{}", effect.path), lock_databases)?;
                }
            }
            let mut pending_previous = None;
            for operation in &pending {
                sorted_number(&mut pending_previous, operation.id)?;
                if !history.iter().any(|record| record == operation) {
                    return Err(Error::Invalid("recovery pending operation"));
                }
            }
            let mut last = BTreeMap::new();
            let mut last_previous = None;
            for _ in 0..state.count()? {
                let key = state.text()?;
                sorted(&mut last_previous, &key)?;
                last.insert(key, state.number()?);
            }
            let mut expected_last = BTreeMap::new();
            for operation in &pending {
                for key in &operation.stores {
                    expected_last.insert(key.clone(), operation.id);
                }
            }
            if last != expected_last {
                return Err(Error::Invalid("recovery pending operation stores"));
            }
            durable_bytes = durable_bytes
                .checked_add(validate_tree(&persisted)?)
                .ok_or(Error::Invalid("recovery durable tree"))?;
            fs.volumes.insert(
                id.clone(),
                VolumeState {
                    id,
                    mount,
                    capacity_bytes,
                    persisted,
                    pending,
                    history,
                    next_operation,
                    last,
                },
            );
        }
        state.finish()?;
        if fs.volumes.len() != configs.len() {
            return Err(Error::Invalid("recovery volume configuration"));
        }
        if next_inode <= maximum_inode
            || next_inode
                .checked_add(fs.rebuild_inode_allocations())
                .is_none()
        {
            return Err(Error::Invalid("recovery filesystem counters"));
        }
        fs.next_inode = next_inode;
        fs.generation = generation;
        fs.nodes.retain(|path, _| path == "/" || path == "/tmp");
        Ok(Self {
            fs,
            occurrences,
            model_identity,
            identity: digest.into(),
        })
    }

    pub(crate) fn recover(&self) -> Result<(Fs, BTreeMap<String, u64>)> {
        Ok((self.fs.clone_durable()?, self.occurrences.clone()))
    }
}

pub(crate) fn image_digest(bytes: &[u8]) -> String {
    bytes_digest(bytes).trim_start_matches("sha256:").into()
}

fn sorted(previous: &mut Option<String>, value: &str) -> Result<()> {
    if previous
        .as_ref()
        .is_some_and(|previous| value <= previous.as_str())
    {
        return Err(Error::Invalid("recovery state ordering"));
    }
    *previous = Some(value.into());
    Ok(())
}
fn sorted_number(previous: &mut Option<u64>, value: u64) -> Result<()> {
    if previous.is_some_and(|previous| value <= previous) {
        return Err(Error::Invalid("recovery state ordering"));
    }
    *previous = Some(value);
    Ok(())
}
fn refuse_lock(resource: &str, databases: &[String]) -> Result<()> {
    if databases.iter().any(|database| {
        resource == format!("{database}.lock")
            || resource == format!("{database}-shm")
            || resource.starts_with(&format!("{database}-shm/"))
    }) {
        return Err(Error::Invalid(
            "durable image contains process-owned SQLite lock",
        ));
    }
    Ok(())
}
/// The durable tree must have a directory root and a countable size. Like Go,
/// recovery admits a dependency-closed durable state larger than the volume
/// capacity (only volatile growth is capacity-checked); the recovered
/// namespace applies Go's `recovered-namespace` filesystem-bytes bound.
fn validate_tree(state: &DiskState) -> Result<u64> {
    let root = state
        .paths
        .get("/")
        .and_then(|inode| state.objects.get(inode));
    if root.is_none_or(|root| root.kind != Kind::Directory) {
        return Err(Error::Invalid("recovery durable tree"));
    }
    let mut used = 0u64;
    let mut seen = BTreeSet::new();
    // A dependency-closed partial persistence choice can retain a child path
    // before its parent namespace operation; Go's durable state is a flat map.
    for inode in state.paths.values() {
        if seen.insert(*inode) {
            used = used
                .checked_add(state.objects[inode].data.len() as u64)
                .ok_or(Error::Invalid("recovery durable tree"))?;
        }
    }
    Ok(used)
}
fn validate_operation(operation: &PersistenceOperation) -> Result<()> {
    let effect = &operation.effect;
    if operation.kind != effect.kind
        || !["alloc", "resize", "write", "metadata", "namespace"].contains(&effect.kind.as_str())
    {
        return Err(Error::Invalid("recovery persistence effect"));
    }
    let mut previous = None;
    for dependency in &operation.deps {
        sorted_number(&mut previous, *dependency)?;
        if *dependency == 0 || *dependency >= operation.id {
            return Err(Error::Invalid("recovery operation dependencies"));
        }
    }
    if effect.kind == "alloc"
        && effect
            .object
            .as_ref()
            .is_none_or(|object| object.inode != effect.inode)
        || effect.kind != "alloc" && effect.object.is_some()
    {
        return Err(Error::Invalid("recovery persistence effect"));
    }
    if effect.size > super::state::MAXIMUM_FILE_BYTES
        || effect
            .offset
            .checked_add(effect.data.len() as u64)
            .is_none_or(|end| end > super::state::MAXIMUM_FILE_BYTES)
    {
        return Err(Error::Invalid("recovery persistence effect"));
    }
    for part in &effect.namespace {
        if part.path == "/"
            || normalize(&part.path).ok().as_deref() != Some(&part.path)
            || !part.path.starts_with('/')
        {
            return Err(Error::Invalid("recovery persistence effect"));
        }
    }
    Ok(())
}

struct Writer {
    data: Vec<u8>,
    maximum: u64,
}
impl Writer {
    fn new(maximum: u64) -> Self {
        Self {
            data: Vec::new(),
            maximum,
        }
    }
    fn raw(&mut self, bytes: &[u8]) -> Result<()> {
        let required = (self.data.len() as u64)
            .checked_add(bytes.len() as u64)
            .ok_or(Error::Capacity {
                resource: "image-bytes",
                required: u64::MAX,
                maximum: self.maximum,
            })?;
        if required > self.maximum {
            return Err(Error::Capacity {
                resource: "image-bytes",
                required,
                maximum: self.maximum,
            });
        }
        self.data.extend_from_slice(bytes);
        Ok(())
    }
    fn number(&mut self, value: u64) -> Result<()> {
        self.raw(&value.to_le_bytes())
    }
    fn blob(&mut self, value: &[u8]) -> Result<()> {
        self.number(value.len() as u64)?;
        self.raw(value)
    }
    fn text(&mut self, value: &str) -> Result<()> {
        self.blob(value.as_bytes())
    }
    fn numbers(&mut self, values: impl ExactSizeIterator<Item = u64>) -> Result<()> {
        self.number(values.len() as u64)?;
        for value in values {
            self.number(value)?;
        }
        Ok(())
    }
    fn object(&mut self, object: &DiskObject) -> Result<()> {
        self.number(object.inode)?;
        self.number(u64::from(object.mode))?;
        self.number(object.kind as u64)?;
        self.number(object.mod_time.cast_unsigned())?;
        self.blob(&object.data)
    }
    fn operations(&mut self, operations: &[PersistenceOperation]) -> Result<()> {
        self.number(operations.len() as u64)?;
        for operation in operations {
            self.number(operation.id)?;
            self.text(&operation.kind)?;
            self.numbers(operation.deps.iter().copied())?;
            self.number(operation.stores.len() as u64)?;
            for store in &operation.stores {
                self.text(store)?;
            }
            let effect = &operation.effect;
            self.text(&effect.kind)?;
            self.number(effect.inode)?;
            self.number(effect.parent)?;
            self.numbers(effect.parents.iter().copied())?;
            self.number(effect.offset)?;
            self.number(effect.size)?;
            self.number(u64::from(effect.mode))?;
            self.number(effect.mod_time.cast_unsigned())?;
            self.blob(&effect.data)?;
            self.number(u64::from(effect.object.is_some()))?;
            if let Some(object) = &effect.object {
                self.object(object)?;
            }
            self.number(effect.namespace.len() as u64)?;
            for entry in &effect.namespace {
                self.text(&entry.path)?;
                self.number(entry.inode)?;
            }
        }
        Ok(())
    }
}

struct Reader<'a> {
    bytes: &'a [u8],
    offset: usize,
}
impl<'a> Reader<'a> {
    fn new(bytes: &'a [u8]) -> Self {
        Self { bytes, offset: 0 }
    }
    fn take(&mut self, count: usize) -> Result<&'a [u8]> {
        let end = self
            .offset
            .checked_add(count)
            .filter(|end| *end <= self.bytes.len())
            .ok_or(Error::Invalid("truncated recovery image"))?;
        let result = &self.bytes[self.offset..end];
        self.offset = end;
        Ok(result)
    }
    fn number(&mut self) -> Result<u64> {
        Ok(u64::from_le_bytes(self.take(8)?.try_into().unwrap()))
    }
    fn small_number(&mut self) -> Result<u32> {
        Ok(u32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }
    fn count(&mut self) -> Result<usize> {
        let count = usize::try_from(self.number()?)
            .map_err(|_| Error::Invalid("truncated recovery image"))?;
        if count > self.bytes.len() - self.offset {
            return Err(Error::Invalid("truncated recovery image"));
        }
        Ok(count)
    }
    fn blob(&mut self) -> Result<&'a [u8]> {
        let count = self.count()?;
        self.take(count)
    }
    fn text(&mut self) -> Result<String> {
        std::str::from_utf8(self.blob()?)
            .map(str::to_owned)
            .map_err(|_| Error::Invalid("recovery image UTF-8"))
    }
    fn finish(&self) -> Result<()> {
        if self.offset == self.bytes.len() {
            Ok(())
        } else {
            Err(Error::Invalid("recovery image trailing bytes"))
        }
    }
    fn numbers(&mut self) -> Result<Vec<u64>> {
        (0..self.count()?).map(|_| self.number()).collect()
    }
    fn object(&mut self) -> Result<DiskObject> {
        let inode = self.number()?;
        let mode =
            u32::try_from(self.number()?).map_err(|_| Error::Invalid("recovery object mode"))?;
        let kind = match self.number()? {
            1 => Kind::File,
            2 => Kind::Directory,
            _ => return Err(Error::Invalid("recovery object kind")),
        };
        let mod_time = (self.number()?).cast_signed();
        let data = self.blob()?.to_vec();
        if inode == 0
            || data.len() as u64 > super::state::MAXIMUM_FILE_BYTES
            || kind == Kind::Directory && !data.is_empty()
        {
            return Err(Error::Invalid("recovery object state"));
        }
        Ok(DiskObject {
            inode,
            mode,
            kind,
            mod_time,
            data: data.into(),
        })
    }
    fn operations(&mut self) -> Result<Vec<PersistenceOperation>> {
        let mut operations = Vec::new();
        for _ in 0..self.count()? {
            let id = self.number()?;
            let kind = self.text()?;
            let deps = self.numbers()?;
            let stores = (0..self.count()?)
                .map(|_| self.text())
                .collect::<Result<Vec<_>>>()?;
            let effect_kind = self.text()?;
            let inode = self.number()?;
            let parent = self.number()?;
            let parents = self.numbers()?;
            let offset = self.number()?;
            let size = self.number()?;
            let mode = u32::try_from(self.number()?)
                .map_err(|_| Error::Invalid("recovery effect mode"))?;
            let mod_time = (self.number()?).cast_signed();
            let data = self.blob()?.to_vec();
            let object = match self.number()? {
                0 => None,
                1 => Some(self.object()?),
                _ => return Err(Error::Invalid("recovery object presence")),
            };
            let namespace = (0..self.count()?)
                .map(|_| {
                    Ok(NamespaceEffect {
                        path: self.text()?,
                        inode: self.number()?,
                    })
                })
                .collect::<Result<Vec<_>>>()?;
            operations.push(PersistenceOperation {
                id,
                kind,
                deps,
                stores,
                effect: Effect {
                    kind: effect_kind,
                    inode,
                    parent,
                    parents,
                    offset,
                    size,
                    mode,
                    mod_time,
                    data,
                    object,
                    namespace,
                },
            });
        }
        Ok(operations)
    }
}

#[cfg(not(feature = "conformance"))]
fn observe_object(mut object: DiskObject) -> crate::linked_storage_inspection::RecoveryObject {
    crate::linked_storage_inspection::RecoveryObject {
        inode: object.inode,
        kind: object.kind as u64,
        mode: object.mode,
        mod_time: object.mod_time,
        data: std::mem::take(&mut *object.data),
    }
}

#[cfg(not(feature = "conformance"))]
fn observe_operation(
    operation: PersistenceOperation,
) -> crate::linked_storage_inspection::RecoveryOperation {
    use crate::linked_storage_inspection::{RecoveryEffect, RecoveryNamespace, RecoveryOperation};
    let effect = operation.effect;
    RecoveryOperation {
        id: operation.id,
        kind: operation.kind,
        dependencies: operation.deps,
        stores: operation.stores,
        effect: RecoveryEffect {
            kind: effect.kind,
            inode: effect.inode,
            parent: effect.parent,
            parents: effect.parents,
            offset: effect.offset,
            size: effect.size,
            mode: effect.mode,
            mod_time: effect.mod_time,
            data: effect.data,
            object: effect.object.map(observe_object),
            namespace: effect
                .namespace
                .into_iter()
                .map(|entry| RecoveryNamespace {
                    path: entry.path,
                    inode: entry.inode,
                })
                .collect(),
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{Engine, engine::general_purpose::STANDARD};

    #[test]
    #[cfg(not(feature = "conformance"))]
    fn installed_lifecycle_readonly_volume_inventory_uses_actual_decoded_images() {
        for count in 0..=2 {
            let configs = (0..count)
                .map(|index| VolumeConfig {
                    id: format!("volume-{index}"),
                    mount: format!("/volume-{index}"),
                    capacity_bytes: 4096,
                })
                .collect::<Vec<_>>();
            let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
                "volumes":configs,"limits":{"PendingOperations":16,"Transitions":1024}
            }))
            .unwrap();
            let input: crate::linked::StorageInput = serde_json::from_value(serde_json::json!({
                "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true},
                "image":null,"context":null
            }))
            .unwrap();
            let mut fs = Fs::new(0);
            fs.configure_volumes(&configs, model.limits, 0).unwrap();
            let raw =
                RecoveryImage::encode(&fs, &model.identity(), &BTreeMap::new(), 1 << 20).unwrap();
            let digest = image_digest(&raw);
            let expected = (0..count)
                .map(|index| format!("volume-{index}"))
                .collect::<Vec<_>>();
            assert_eq!(
                crate::linked_storage_inspection::recovery_volumes(&input, &raw, &digest, 1 << 20)
                    .unwrap(),
                expected
            );
            for length in [0, 7, raw.len() - 1] {
                assert!(
                    crate::linked_storage_inspection::recovery_volumes(
                        &input,
                        &raw[..length],
                        &image_digest(&raw[..length]),
                        1 << 20
                    )
                    .is_err()
                );
            }
            let mut corrupt = raw.clone();
            corrupt[0] ^= 1;
            assert!(
                crate::linked_storage_inspection::recovery_volumes(
                    &input,
                    &corrupt,
                    &digest,
                    1 << 20
                )
                .is_err()
            );
            let mut wrong_version = raw.clone();
            wrong_version[8] = 2;
            assert!(
                crate::linked_storage_inspection::recovery_volumes(
                    &input,
                    &wrong_version,
                    &image_digest(&wrong_version),
                    1 << 20
                )
                .is_err()
            );
            assert!(
                crate::linked_storage_inspection::recovery_volumes(
                    &input,
                    &raw,
                    &digest,
                    raw.len() as u64 - 1
                )
                .is_err()
            );
        }
    }

    #[test]
    #[cfg(not(feature = "conformance"))]
    fn admitted_image_projection_retains_durable_objects_and_complete_effects() {
        let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":configs(),"limits":limits()
        }))
        .unwrap();
        let input: crate::linked::StorageInput = serde_json::from_value(serde_json::json!({
            "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true}
        }))
        .unwrap();
        let mut fs = Fs::new(0);
        fs.configure_volumes(&configs(), limits(), 0).unwrap();
        let inode = fs
            .create("/workspace/retained", Kind::File, 0o640, 11)
            .unwrap();
        fs.write_at(inode, 0, b"durable", 12).unwrap();
        fs.sync_node(inode).unwrap();
        fs.write_at(inode, 2, b"pending", 13).unwrap();
        fs.chmod("/workspace/retained", 0o600, 14).unwrap();
        let occurrences = BTreeMap::from([("/workspace/retained\0write".into(), 3)]);
        let raw = RecoveryImage::encode(&fs, &model.identity(), &occurrences, 8 << 20).unwrap();
        let observed = crate::linked_storage_inspection::recovery_image(
            &input,
            &raw,
            &image_digest(&raw),
            8 << 20,
        )
        .unwrap();
        assert_eq!(observed.identity(), image_digest(&raw));
        assert_eq!(observed.model_identity(), model.identity());
        assert_eq!(observed.next_inode(), fs.next_inode);
        assert_eq!(observed.generation(), fs.generation);
        assert_eq!(observed.occurrences(), &occurrences);
        let volume = &observed.volumes()[0];
        let source = &fs.volumes["data"];
        assert_eq!(volume.id(), source.id);
        assert_eq!(volume.mount(), source.mount);
        assert_eq!(volume.capacity_bytes(), source.capacity_bytes);
        assert_eq!(volume.paths(), &source.persisted.paths);
        assert_eq!(volume.owned(), &source.persisted.owned);
        assert_eq!(volume.last(), &source.last);
        assert_eq!(volume.next_operation(), source.next_operation);
        assert_eq!(volume.objects().len(), source.persisted.objects.len());
        for (id, object) in volume.objects() {
            let expected = &source.persisted.objects[id];
            assert_eq!(
                (
                    object.inode(),
                    object.kind(),
                    object.mode(),
                    object.mod_time(),
                    object.data()
                ),
                (
                    expected.inode,
                    expected.kind as u64,
                    expected.mode,
                    expected.mod_time,
                    expected.data.as_ref()
                )
            );
        }
        for (operations, expected) in [
            (volume.pending(), &source.pending),
            (volume.history(), &source.history),
        ] {
            assert_eq!(operations.len(), expected.len());
            for (operation, expected) in operations.iter().zip(expected) {
                assert_eq!(
                    (
                        operation.id(),
                        operation.kind(),
                        operation.dependencies(),
                        operation.stores()
                    ),
                    (
                        expected.id,
                        expected.kind.as_str(),
                        expected.deps.as_slice(),
                        expected.stores.as_slice()
                    )
                );
                let effect = operation.effect();
                let expected = &expected.effect;
                assert_eq!(
                    (
                        effect.kind(),
                        effect.inode(),
                        effect.parent(),
                        effect.parents(),
                        effect.offset(),
                        effect.size()
                    ),
                    (
                        expected.kind.as_str(),
                        expected.inode,
                        expected.parent,
                        expected.parents.as_slice(),
                        expected.offset,
                        expected.size
                    )
                );
                assert_eq!(
                    (effect.mode(), effect.mod_time(), effect.data()),
                    (expected.mode, expected.mod_time, expected.data.as_slice())
                );
                assert_eq!(
                    effect.object().map(|object| (
                        object.inode(),
                        object.kind(),
                        object.mode(),
                        object.mod_time(),
                        object.data()
                    )),
                    expected.object.as_ref().map(|object| (
                        object.inode,
                        object.kind as u64,
                        object.mode,
                        object.mod_time,
                        object.data.as_ref()
                    ))
                );
                assert_eq!(
                    effect
                        .namespace()
                        .iter()
                        .map(|entry| (entry.path(), entry.inode()))
                        .collect::<Vec<_>>(),
                    expected
                        .namespace
                        .iter()
                        .map(|entry| (entry.path.as_str(), entry.inode))
                        .collect::<Vec<_>>()
                );
            }
        }
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-image-projection","inputs":{"storage":input,"image_base64":STANDARD.encode(&raw),"image_digest":image_digest(&raw)},"observed":{"identity":observed.identity(),"model_identity":observed.model_identity(),"volumes":observed.volumes().len(),"pending":volume.pending().len(),"history":volume.history().len()}})
        );
        let selection = BTreeMap::from([(
            "data".to_string(),
            volume
                .pending()
                .iter()
                .map(|operation| operation.id())
                .collect(),
        )]);
        fs.crash_volumes(&selection).unwrap();
        let selected_raw =
            RecoveryImage::encode(&fs, &model.identity(), &occurrences, 8 << 20).unwrap();
        let selected = crate::linked_storage_inspection::recovery_image(
            &input,
            &selected_raw,
            &image_digest(&selected_raw),
            8 << 20,
        )
        .unwrap();
        assert!(selected.volumes()[0].pending().is_empty());
        assert_eq!(selected.volumes()[0].objects()[&inode].data(), b"dupending");
        assert_eq!(selected.volumes()[0].objects()[&inode].mode(), 0o600);
        if let Some(directory) = std::env::var_os("FN242_CORE_PROOF_DIRECTORY") {
            let directory = std::path::PathBuf::from(directory);
            std::fs::create_dir_all(&directory).unwrap();
            std::fs::write(directory.join("effect-baseline-image.bin"), &raw).unwrap();
            std::fs::write(directory.join("effect-selected-image.bin"), &selected_raw).unwrap();
            std::fs::write(
                directory.join("effect-input.json"),
                serde_json::to_vec(&input).unwrap(),
            )
            .unwrap();
        }
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-effects-selected-control","inputs":{"storage":input,"baseline_image_base64":STANDARD.encode(&raw),"baseline_image_digest":image_digest(&raw),"selected_image_base64":STANDARD.encode(&selected_raw),"selected_image_digest":image_digest(&selected_raw),"selection":selection},"observed":{"accepted":true,"pending":selected.volumes()[0].pending().len()}})
        );
        for (name, bytes, maximum) in [
            ("trailing", [raw.as_slice(), &[0]].concat(), 8 << 20),
            ("oversized", raw.clone(), raw.len() as u64 - 1),
            ("malformed", vec![0; 56], 8 << 20),
        ] {
            let error = crate::linked_storage_inspection::recovery_image(
                &input,
                &bytes,
                &image_digest(&bytes),
                maximum,
            )
            .unwrap_err();
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-image-refusal","inputs":{"mutation":name,"storage":input,"image_base64":STANDARD.encode(&bytes),"image_digest":image_digest(&bytes),"maximum":maximum},"observed":{"error":error}})
            );
        }
    }

    #[test]
    #[cfg(not(feature = "conformance"))]
    #[expect(
        clippy::too_many_lines,
        reason = "One table binds the production image decoder's semantic guards to genuine encoded controls and exact mutated bytes."
    )]
    fn admitted_image_semantic_guard_mutations_reach_common_decoder() {
        let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":configs(),"limits":limits()
        }))
        .unwrap();
        let input: crate::linked::StorageInput = serde_json::from_value(serde_json::json!({
            "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true}
        }))
        .unwrap();
        for (mutation, expected) in [
            ("object-zero-inode", "recovery object state"),
            ("object-mode-range", "recovery object mode"),
            ("object-kind", "recovery object kind"),
            ("directory-payload", "recovery object state"),
            ("effect-mode-range", "recovery effect mode"),
            ("object-presence", "recovery object presence"),
            ("effect-kind-mismatch", "recovery persistence effect"),
            ("effect-kind-unsupported", "recovery persistence effect"),
            ("dependency-zero", "recovery operation dependencies"),
            ("dependency-future", "recovery operation dependencies"),
            ("dependency-order", "recovery state ordering"),
            ("allocation-missing", "recovery persistence effect"),
            ("allocation-inode", "recovery persistence effect"),
            ("nonallocation-object", "recovery persistence effect"),
            ("effect-size", "recovery persistence effect"),
            ("effect-offset-overflow", "recovery persistence effect"),
            ("effect-write-bound", "recovery persistence effect"),
            ("namespace-root", "recovery persistence effect"),
            ("namespace-relative", "recovery persistence effect"),
            ("namespace-noncanonical", "recovery persistence effect"),
            ("owned-missing-object", "recovery object ownership"),
            ("durable-relative", "recovery durable tree"),
            ("durable-noncanonical", "recovery durable tree"),
            ("durable-missing-object", "recovery durable tree"),
            ("durable-inode-alias", "recovery path inode alias"),
            ("durable-root-missing", "recovery durable tree"),
            ("durable-root-file", "recovery durable tree"),
            ("history-id", "recovery operation history"),
            ("pending-order", "recovery state ordering"),
            ("pending-history-mismatch", "recovery pending operation"),
            ("operation-counter", "recovery operation counter"),
            ("last-store-value", "recovery pending operation stores"),
            ("last-store-missing", "recovery pending operation stores"),
            ("volume-id", "recovery volume configuration"),
            ("volume-mount", "recovery volume configuration"),
            ("volume-capacity", "recovery volume configuration"),
            ("volume-inventory", "recovery volume configuration"),
            ("occurrence-count", "recovery fault occurrence"),
            ("occurrence-key", "recovery fault occurrence"),
            ("occurrence-path", "recovery fault occurrence"),
            ("occurrence-operation", "recovery fault occurrence"),
        ] {
            let mut fs = Fs::new(0);
            fs.configure_volumes(&configs(), limits(), 0).unwrap();
            let inode = fs
                .create("/workspace/retained", Kind::File, 0o640, 11)
                .unwrap();
            fs.write_at(inode, 0, b"durable", 12).unwrap();
            fs.sync_node(inode).unwrap();
            let root = fs.lookup("/workspace").unwrap().inode;
            fs.sync_node(root).unwrap();
            fs.create("/workspace/pending", Kind::File, 0o600, 13)
                .unwrap();
            let mut occurrences = BTreeMap::from([("/workspace/retained\0write".into(), 3)]);
            let original =
                RecoveryImage::encode(&fs, &model.identity(), &occurrences, 8 << 20).unwrap();
            crate::linked_storage_inspection::recovery_image(
                &input,
                &original,
                &image_digest(&original),
                8 << 20,
            )
            .unwrap();
            let volume = fs.volumes.get_mut("data").unwrap();
            match mutation {
                "object-zero-inode" => volume.persisted.objects.get_mut(&root).unwrap().inode = 0,
                "directory-payload" => {
                    volume.persisted.objects.get_mut(&root).unwrap().data = vec![1].into()
                }
                "effect-kind-mismatch" => volume.history[0].effect.kind = "metadata".into(),
                "effect-kind-unsupported" => {
                    volume.history[0].kind = "unsupported".into();
                    volume.history[0].effect.kind = "unsupported".into();
                }
                "dependency-zero" => volume.history[0].deps = vec![0],
                "dependency-future" => volume.history[0].deps = vec![1],
                "dependency-order" => {
                    let last = volume.history.last_mut().unwrap();
                    last.deps = vec![2, 1];
                }
                "allocation-missing" => volume.history[0].effect.object = None,
                "allocation-inode" => volume.history[0].effect.object.as_mut().unwrap().inode += 1,
                "nonallocation-object" => {
                    let object = volume.history[0].effect.object.clone();
                    volume.history[1].effect.object = object;
                }
                "effect-size" => volume.history[0].effect.size = (256 << 20) + 1,
                "effect-offset-overflow" => {
                    volume.history[0].effect.offset = u64::MAX;
                    volume.history[0].effect.data = vec![1];
                }
                "effect-write-bound" => {
                    volume.history[0].effect.offset = 256 << 20;
                    volume.history[0].effect.data = vec![1];
                }
                "namespace-root" | "namespace-relative" | "namespace-noncanonical" => {
                    volume.history[1].effect.namespace[0].path = match mutation {
                        "namespace-root" => "/",
                        "namespace-relative" => "relative",
                        _ => "/a/../b",
                    }
                    .into();
                }
                "owned-missing-object" => {
                    volume.persisted.owned.insert(u64::MAX);
                }
                "durable-relative"
                | "durable-noncanonical"
                | "durable-missing-object"
                | "durable-inode-alias" => {
                    let (path, id) = match mutation {
                        "durable-relative" => ("relative", inode),
                        "durable-noncanonical" => ("/a/../b", inode),
                        "durable-missing-object" => ("/missing", u64::MAX),
                        _ => ("/alias", root),
                    };
                    volume.persisted.paths.insert(path.into(), id);
                }
                "durable-root-missing" => {
                    volume.persisted.paths.remove("/");
                }
                "durable-root-file" => {
                    volume.persisted.objects.get_mut(&root).unwrap().kind = Kind::File
                }
                "history-id" => volume.history[0].id = 2,
                "pending-order" => volume.pending.reverse(),
                "pending-history-mismatch" => volume.pending[0].effect.mod_time += 1,
                "operation-counter" => volume.next_operation += 1,
                "last-store-value" => {
                    *volume.last.values_mut().next().unwrap() += 1;
                }
                "last-store-missing" => volume.last.clear(),
                "volume-id" => volume.id = "foreign".into(),
                "volume-mount" => volume.mount = "/foreign".into(),
                "volume-capacity" => volume.capacity_bytes += 1,
                "volume-inventory" => fs.volumes.clear(),
                "occurrence-count" => {
                    *occurrences.values_mut().next().unwrap() = 0;
                }
                "occurrence-key" | "occurrence-path" | "occurrence-operation" => {
                    occurrences.clear();
                    occurrences.insert(
                        match mutation {
                            "occurrence-key" => "no-separator",
                            "occurrence-path" => "/a/../b\0write",
                            _ => "/workspace/retained\0unsupported",
                        }
                        .into(),
                        1,
                    );
                }
                "object-mode-range" | "object-kind" | "effect-mode-range" | "object-presence" => {}
                _ => unreachable!("table mutation"),
            }
            let mut raw =
                RecoveryImage::encode(&fs, &model.identity(), &occurrences, 8 << 20).unwrap();
            if matches!(
                mutation,
                "object-mode-range" | "object-kind" | "effect-mode-range" | "object-presence"
            ) {
                patch_image_scalar(&mut raw, mutation);
            }
            let error = crate::linked_storage_inspection::recovery_image(
                &input,
                &raw,
                &image_digest(&raw),
                8 << 20,
            )
            .unwrap_err();
            assert!(
                error.contains(expected),
                "{mutation}: expected {expected}: {error}"
            );
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-image-semantic-guard","inputs":{"mutation":mutation,"storage":input,"control_image_base64":STANDARD.encode(&original),"control_image_digest":image_digest(&original),"image_base64":STANDARD.encode(&raw),"image_digest":image_digest(&raw),"maximum":8 << 20},"observed":{"error":error,"predicate":expected}})
            );
        }
    }

    #[cfg(not(feature = "conformance"))]
    fn patch_image_scalar(raw: &mut [u8], mutation: &str) {
        let mut frame = Reader::new(raw);
        frame.take(16).unwrap();
        for _ in 0..2 {
            frame.small_number().unwrap();
            frame.blob().unwrap();
        }
        frame.small_number().unwrap();
        let state_start = frame.offset + 8;
        let state = frame.blob().unwrap();
        let mut reader = Reader::new(state);
        reader.number().unwrap();
        reader.number().unwrap();
        reader.count().unwrap();
        reader.text().unwrap();
        reader.text().unwrap();
        reader.number().unwrap();
        let objects = reader.count().unwrap();
        let first_object = reader.offset;
        for _ in 0..objects {
            reader.object().unwrap();
        }
        for _ in 0..reader.count().unwrap() {
            reader.text().unwrap();
            reader.number().unwrap();
        }
        reader.numbers().unwrap();
        reader.count().unwrap();
        reader.number().unwrap();
        reader.text().unwrap();
        reader.numbers().unwrap();
        for _ in 0..reader.count().unwrap() {
            reader.text().unwrap();
        }
        reader.text().unwrap();
        reader.number().unwrap();
        reader.number().unwrap();
        reader.numbers().unwrap();
        reader.number().unwrap();
        reader.number().unwrap();
        let effect_mode = reader.offset;
        reader.number().unwrap();
        reader.number().unwrap();
        reader.blob().unwrap();
        let object_presence = reader.offset;
        let (offset, value) = match mutation {
            "object-mode-range" => (first_object + 8, u64::MAX),
            "object-kind" => (first_object + 16, 3),
            "effect-mode-range" => (effect_mode, u64::MAX),
            "object-presence" => (object_presence, 2),
            _ => unreachable!("scalar mutation"),
        };
        raw[state_start + offset..state_start + offset + 8].copy_from_slice(&value.to_le_bytes());
    }

    #[test]
    #[cfg(not(feature = "conformance"))]
    fn admitted_image_ordering_and_framing_mutations_reach_common_decoder() {
        let mut configuration = configs();
        configuration.insert(
            0,
            VolumeConfig {
                id: "data2".into(),
                mount: "/other".into(),
                capacity_bytes: 1 << 20,
            },
        );
        let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":configuration,"limits":limits()
        }))
        .unwrap();
        let input: crate::linked::StorageInput = serde_json::from_value(serde_json::json!({
            "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true}
        }))
        .unwrap();
        let mut fs = Fs::new(0);
        fs.configure_volumes(&configuration, limits(), 0).unwrap();
        let inode = fs
            .create("/workspace/retained", Kind::File, 0o640, 11)
            .unwrap();
        fs.write_at(inode, 0, b"durable", 12).unwrap();
        fs.sync_node(inode).unwrap();
        let root = fs.lookup("/workspace").unwrap().inode;
        fs.sync_node(root).unwrap();
        fs.create("/workspace/pending", Kind::File, 0o600, 13)
            .unwrap();
        fs.volumes
            .get_mut("data")
            .unwrap()
            .persisted
            .owned
            .extend([root, inode]);
        let original =
            RecoveryImage::encode(&fs, &model.identity(), &BTreeMap::new(), 8 << 20).unwrap();
        crate::linked_storage_inspection::recovery_image(
            &input,
            &original,
            &image_digest(&original),
            8 << 20,
        )
        .unwrap();
        let mut frame = Reader::new(&original);
        frame.take(16).unwrap();
        let mut sections = Vec::new();
        for _ in 0..3 {
            frame.small_number().unwrap();
            sections.push(frame.blob().unwrap().to_vec());
        }
        let mut reader = Reader::new(&sections[2]);
        reader.number().unwrap();
        reader.number().unwrap();
        reader.count().unwrap();
        let first_id = reader.offset + 8;
        reader.text().unwrap();
        reader.text().unwrap();
        reader.number().unwrap();
        let count = reader.count().unwrap();
        let mut object_offsets = Vec::new();
        for _ in 0..count {
            object_offsets.push(reader.offset);
            reader.object().unwrap();
        }
        let count = reader.count().unwrap();
        let mut path_offsets = Vec::new();
        for _ in 0..count {
            path_offsets.push(reader.offset + 8);
            reader.text().unwrap();
            reader.number().unwrap();
        }
        let owned = reader.offset + 8;
        assert!(reader.numbers().unwrap().len() >= 2);
        let pending_count = reader.offset;
        reader.operations().unwrap();
        reader.operations().unwrap();
        reader.number().unwrap();
        let count = reader.count().unwrap();
        let mut last_offsets = Vec::new();
        let mut last_keys = Vec::new();
        for _ in 0..count {
            last_offsets.push(reader.offset + 8);
            last_keys.push(reader.text().unwrap());
            reader.number().unwrap();
        }
        let second_id = reader.offset + 8;
        assert!(object_offsets.len() >= 2 && path_offsets.len() >= 2 && last_offsets.len() >= 2);
        for (mutation, expected) in [
            ("text-utf8", "recovery image UTF-8"),
            ("state-count-bound", "truncated recovery image"),
            ("operation-count-bound", "truncated recovery image"),
            ("model-inner-finish", "recovery image trailing bytes"),
            ("occurrence-inner-finish", "recovery image trailing bytes"),
            ("state-inner-finish", "recovery image trailing bytes"),
            ("generation-zero", "recovery filesystem counters"),
            ("object-order", "recovery state ordering"),
            ("owned-order", "recovery state ordering"),
            ("path-order", "recovery state ordering"),
            ("volume-order", "recovery state ordering"),
            ("last-map-order", "recovery state ordering"),
            ("path-duplicate", "recovery state ordering"),
            ("volume-duplicate", "recovery state ordering"),
            ("last-map-duplicate", "recovery state ordering"),
        ] {
            let mut parts = sections.clone();
            match mutation {
                "text-utf8" => parts[2][first_id] = 0xff,
                "state-count-bound" => parts[2][16..24].copy_from_slice(&u64::MAX.to_le_bytes()),
                "operation-count-bound" => parts[2][pending_count..pending_count + 8]
                    .copy_from_slice(&u64::MAX.to_le_bytes()),
                "model-inner-finish" => parts[0].push(0),
                "occurrence-inner-finish" => parts[1].push(0),
                "state-inner-finish" => parts[2].push(0),
                "generation-zero" => parts[2][8..16].copy_from_slice(&0u64.to_le_bytes()),
                "object-order" => parts[2][object_offsets[1]..object_offsets[1] + 8]
                    .copy_from_slice(&root.to_le_bytes()),
                "owned-order" => {
                    parts[2][owned + 8..owned + 16].copy_from_slice(&root.to_le_bytes())
                }
                "path-order" => parts[2][path_offsets[1]] = b'!',
                "volume-order" => parts[2][second_id] = b'a',
                "last-map-order" => parts[2][last_offsets[1]] = 0,
                "path-duplicate" | "volume-duplicate" | "last-map-duplicate" => {
                    let (offset, replacement) = match mutation {
                        "path-duplicate" => (path_offsets[1], "/"),
                        "volume-duplicate" => (second_id, "data"),
                        _ => (last_offsets[1], last_keys[0].as_str()),
                    };
                    let length = usize::try_from(u64::from_le_bytes(
                        parts[2][offset - 8..offset].try_into().unwrap(),
                    ))
                    .unwrap();
                    let mut field = (replacement.len() as u64).to_le_bytes().to_vec();
                    field.extend_from_slice(replacement.as_bytes());
                    parts[2].splice(offset - 8..offset + length, field);
                }
                _ => unreachable!("table mutation"),
            }
            let mut writer = Writer::new(8 << 20);
            writer.raw(b"GMVOLIMG").unwrap();
            writer.raw(&1u32.to_le_bytes()).unwrap();
            writer.raw(&3u32.to_le_bytes()).unwrap();
            for (tag, section) in parts.iter().enumerate() {
                writer
                    .raw(&(u32::try_from(tag).unwrap() + 1).to_le_bytes())
                    .unwrap();
                writer.blob(section).unwrap();
            }
            let raw = writer.data;
            let error = crate::linked_storage_inspection::recovery_image(
                &input,
                &raw,
                &image_digest(&raw),
                8 << 20,
            )
            .unwrap_err();
            assert!(
                error.contains(expected),
                "{mutation}: expected {expected}: {error}"
            );
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-image-framing-guard","inputs":{"mutation":mutation,"storage":input,"control_image_base64":STANDARD.encode(&original),"control_image_digest":image_digest(&original),"image_base64":STANDARD.encode(&raw),"image_digest":image_digest(&raw),"maximum":8 << 20},"observed":{"error":error,"predicate":expected}})
            );
        }
    }

    #[test]
    #[cfg(not(feature = "conformance"))]
    fn admitted_image_preserves_genuine_partial_durability_above_volatile_capacity() {
        let configuration = vec![VolumeConfig {
            id: "data".into(),
            mount: "/workspace".into(),
            capacity_bytes: 8,
        }];
        let model: crate::volume::StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":configuration,"limits":limits()
        }))
        .unwrap();
        let input: crate::linked::StorageInput = serde_json::from_value(serde_json::json!({
            "descriptor":{"model":model,"model_identity":model.identity(),"return_image":true}
        }))
        .unwrap();
        let mut fs = Fs::new(0);
        fs.configure_volumes(&configuration, limits(), 0).unwrap();
        let root = fs.lookup("/workspace").unwrap().inode;
        let old = fs.create("/workspace/old", Kind::File, 0o600, 1).unwrap();
        fs.write_at(old, 0, b"old-data", 2).unwrap();
        fs.sync_node(old).unwrap();
        fs.sync_node(root).unwrap();
        fs.resize(old, 0, 3).unwrap();
        let new = fs.create("/workspace/new", Kind::File, 0o600, 4).unwrap();
        fs.write_at(new, 0, b"new-data", 5).unwrap();
        fs.sync_node(new).unwrap();
        fs.sync_node(root).unwrap();
        let raw = RecoveryImage::encode(&fs, &model.identity(), &BTreeMap::new(), 8 << 20).unwrap();
        let observed = crate::linked_storage_inspection::recovery_image(
            &input,
            &raw,
            &image_digest(&raw),
            8 << 20,
        )
        .unwrap();
        let volume = &observed.volumes()[0];
        let durable_bytes: usize = volume
            .objects()
            .values()
            .map(|object| object.data().len())
            .sum();
        assert_eq!(durable_bytes, 16);
        assert_eq!(volume.capacity_bytes(), 8);
        assert_eq!(volume.paths()["/old"], old);
        assert_eq!(volume.paths()["/new"], new);
        assert!(
            volume
                .objects()
                .values()
                .all(|object| object.inode() < observed.next_inode())
        );
        assert_eq!(fs.lookup("/workspace/old").unwrap().data.len(), 0);
        assert_eq!(fs.lookup("/workspace/new").unwrap().data.len(), 8);
        if let Some(directory) = std::env::var_os("FN242_CORE_PROOF_DIRECTORY") {
            let directory = std::path::PathBuf::from(directory);
            std::fs::create_dir_all(&directory).unwrap();
            std::fs::write(directory.join("over-capacity-image.bin"), &raw).unwrap();
            std::fs::write(
                directory.join("over-capacity-input.json"),
                serde_json::to_vec(&input).unwrap(),
            )
            .unwrap();
        }
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            serde_json::json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.admitted-partial-durability-capacity","inputs":{"storage":input,"image_base64":STANDARD.encode(&raw),"image_digest":image_digest(&raw),"maximum":8 << 20},"observed":{"durable_bytes":durable_bytes,"capacity":volume.capacity_bytes(),"next_inode":observed.next_inode(),"accepted":true}})
        );
    }

    fn configs() -> Vec<VolumeConfig> {
        vec![VolumeConfig {
            id: "data".into(),
            mount: "/workspace".into(),
            capacity_bytes: 1 << 20,
        }]
    }
    fn limits() -> VolumeLimits {
        VolumeLimits {
            pending_operations: 1000,
            transitions: 1000,
        }
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The corpus validator preserves its complete ordered literal and observation checks."
    )]
    fn go_generated_image_admission_corpus() {
        let corpus: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/image-admission.json"
        ))
        .unwrap();
        for case in corpus["cases"].as_array().unwrap() {
            let bytes = STANDARD
                .decode(case["image_base64"].as_str().unwrap())
                .unwrap();
            let configs: Vec<VolumeConfig> =
                serde_json::from_value(case["storage"]["volumes"].clone()).unwrap();
            let limits: VolumeLimits =
                serde_json::from_value(case["storage"]["limits"].clone()).unwrap();
            let locks: Vec<String> = case["storage"]["sqlite_lock_databases"]
                .as_array()
                .map(|paths| {
                    paths
                        .iter()
                        .map(|path| path.as_str().unwrap().into())
                        .collect()
                })
                .unwrap_or_default();
            let maximum = case["maximum"].as_u64().unwrap();
            let result = RecoveryImage::decode(
                &bytes,
                case["image_digest"].as_str().unwrap(),
                case["model"].as_str().unwrap(),
                &configs,
                limits,
                &locks,
                maximum,
            );
            let error = match result {
                Ok(image) => {
                    assert_eq!(
                        RecoveryImage::encode(
                            &image.fs,
                            &image.model_identity,
                            &image.occurrences,
                            maximum
                        )
                        .unwrap(),
                        bytes
                    );
                    let before = image.fs.volume_snapshots();
                    let (mut recovered, _) = image.recover().unwrap();
                    for (id, volume) in &image.fs.volumes {
                        assert_eq!(
                            recovered.volumes[id].persisted.objects.len(),
                            volume.persisted.objects.len(),
                            "{} recovered object count: {id}",
                            case["name"]
                        );
                    }
                    if let Some(next) = case["recovered_next_inode"].as_u64() {
                        assert_eq!(recovered.next_inode, next);
                        assert_eq!(
                            u64::from(recovered.lookup("/tmp").unwrap().mode),
                            case["recovered_temp_mode"].as_u64().unwrap()
                        );
                        for config in &configs {
                            if let Some(inode) = image.fs.volumes[&config.id]
                                .persisted
                                .paths
                                .get("/retained")
                            {
                                assert_eq!(
                                    recovered
                                        .lookup(&format!("{}/retained", config.mount))
                                        .unwrap()
                                        .data,
                                    image.fs.volumes[&config.id].persisted.objects[inode].data
                                );
                            }
                        }
                    }
                    if case["refuse_allocation"].as_bool() == Some(true) {
                        let mount = configs
                            .first()
                            .map_or("/tmp", |config| config.mount.as_str());
                        let snapshot = recovered.volume_snapshots();
                        assert_eq!(
                            recovered.create(&format!("{mount}/refused"), Kind::File, 0o600, 1),
                            Err(Error::Errno("no-space"))
                        );
                        assert_eq!(
                            recovered.mkdir_all(&format!("{mount}/a/b"), 0o755, 1),
                            Err(Error::Errno("no-space"))
                        );
                        assert!(matches!(
                            recovered.advance_lifecycle(false, &BTreeMap::new()),
                            Err(Error::Errno("no-space"))
                        ));
                        assert_eq!(recovered.volume_snapshots(), snapshot);
                    }
                    assert_eq!(image.fs.volume_snapshots(), before);
                    assert_eq!(
                        RecoveryImage::encode(
                            &image.fs,
                            &image.model_identity,
                            &image.occurrences,
                            maximum
                        )
                        .unwrap(),
                        bytes
                    );
                    String::new()
                }
                Err(Error::Invalid(message)) => message.into(),
                Err(Error::Capacity { resource, .. }) => format!("capacity:{resource}"),
                Err(error) => panic!("{}: unexpected error {error:?}", case["name"]),
            };
            assert_eq!(error, case["error"].as_str().unwrap(), "{}", case["name"]);
        }
    }

    #[test]
    fn go_generated_image_identity_corpus() {
        let cases: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/image-identity.json"
        ))
        .unwrap();
        let mut digests = Vec::new();
        let mut counters = Vec::new();
        for case in cases.as_array().unwrap() {
            let bytes = STANDARD
                .decode(case["image_base64"].as_str().unwrap())
                .unwrap();
            let digest = case["image_digest"].as_str().unwrap();
            let mut reader = Reader::new(&bytes);
            reader.take(20).unwrap();
            let mut model_section = Reader::new(reader.blob().unwrap());
            let model = model_section.text().unwrap();
            let image =
                RecoveryImage::decode(&bytes, digest, &model, &configs(), limits(), &[], 1 << 20)
                    .unwrap();
            let expected: BTreeMap<String, u64> =
                serde_json::from_value(case["occurrences"].clone()).unwrap();
            assert_eq!(image.occurrences, expected);
            assert_eq!(image.fs.next_inode, case["next_inode"].as_u64().unwrap());
            assert_eq!(
                RecoveryImage::encode(&image.fs, &model, &image.occurrences, 1 << 20).unwrap(),
                bytes
            );
            assert!(matches!(
                RecoveryImage::encode(
                    &image.fs,
                    &model,
                    &image.occurrences,
                    bytes.len() as u64 - 1
                ),
                Err(Error::Capacity {
                    resource: "image-bytes",
                    ..
                })
            ));
            assert_eq!(image_digest(&bytes), digest);
            digests.push(digest.to_owned());
            counters.push(image.fs.next_inode);
        }
        assert_ne!(digests[0], digests[1]);
        assert_ne!(digests[0], digests[2]);
        assert_ne!(counters[0], counters[1]);
        assert_eq!(counters[0], counters[2]);
    }

    #[test]
    fn decoding_keeps_durable_payloads_without_constructing_discarded_namespace_nodes() {
        let model = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        let mut fs = Fs::new(0);
        fs.configure_volumes(&configs(), limits(), 0).unwrap();
        let inode = fs.create("/workspace/db", Kind::File, 0o600, 1).unwrap();
        fs.write_at(inode, 0, b"durable", 2).unwrap();
        fs.sync_node(inode).unwrap();
        fs.sync_node(fs.lookup("/workspace").unwrap().inode)
            .unwrap();
        let bytes = RecoveryImage::encode(&fs, model, &BTreeMap::new(), 1 << 20).unwrap();
        let image = RecoveryImage::decode(
            &bytes,
            &image_digest(&bytes),
            model,
            &configs(),
            limits(),
            &[],
            1 << 20,
        )
        .unwrap();
        assert!(image.fs.nodes.values().all(|node| node.volume.is_empty()));
        assert_eq!(image.fs.used_bytes, 0);
    }

    #[test]
    fn rehashed_images_refuse_persisted_inode_aliases_across_volumes_and_reserved_nodes() {
        let model = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff";
        let configs = vec![
            VolumeConfig {
                id: "left".into(),
                mount: "/left".into(),
                capacity_bytes: 4096,
            },
            VolumeConfig {
                id: "right".into(),
                mount: "/right".into(),
                capacity_bytes: 4096,
            },
        ];
        for (alias, orphan) in [(3, false), (3, true), (1, false), (2, false)] {
            let mut fs = Fs::new(0);
            fs.configure_volumes(&configs, limits(), 0).unwrap();
            assert_eq!(fs.volumes["left"].persisted.paths["/"], 3);
            let right = fs.volumes.get_mut("right").unwrap();
            let old_inode = right.persisted.paths["/"];
            let mut object = right.persisted.objects[&old_inode].clone();
            object.inode = alias;
            right.persisted.objects.insert(alias, object);
            right.persisted.owned.insert(alias);
            if !orphan {
                right.persisted.objects.remove(&old_inode);
                right.persisted.owned.remove(&old_inode);
                right.persisted.paths.insert("/".into(), alias);
            }
            let before = fs.volume_snapshots();
            let _bytes = RecoveryImage::encode(&fs, model, &BTreeMap::new(), 1 << 20).unwrap();
            assert_eq!(fs.volume_snapshots(), before);
        }
    }

    fn roundtrip_go_image(step: &serde_json::Value, storage: &serde_json::Value) {
        let bytes = STANDARD
            .decode(step["image_base64"].as_str().unwrap())
            .unwrap();
        let digest = step["image_digest"].as_str().unwrap();
        let mut frame = Reader::new(&bytes);
        frame.take(20).unwrap();
        let mut model_section = Reader::new(frame.blob().unwrap());
        let model = model_section.text().unwrap();
        let configs: Vec<VolumeConfig> =
            serde_json::from_value(storage["volumes"].clone()).unwrap();
        let limits: VolumeLimits = serde_json::from_value(storage["limits"].clone()).unwrap();
        let result = RecoveryImage::decode(&bytes, digest, &model, &configs, limits, &[], 1 << 26);
        if step["op"] == "configure" && step["error"].as_str().is_some() {
            assert!(matches!(
                result,
                Err(Error::Invalid("recovery volume configuration"))
            ));
            return;
        }
        let image = result.unwrap();
        assert_eq!(image_digest(&bytes), digest);
        assert_eq!(
            RecoveryImage::encode(&image.fs, &model, &image.occurrences, 1 << 26).unwrap(),
            bytes
        );
        let (first, occurrences) = image.recover().unwrap();
        let (second, second_occurrences) = image.recover().unwrap();
        for (id, volume) in &image.fs.volumes {
            assert_eq!(
                first.volumes[id].persisted.objects.len(),
                volume.persisted.objects.len()
            );
        }
        assert_eq!(first.volume_snapshots(), second.volume_snapshots());
        assert_eq!(occurrences, second_occurrences);
    }

    #[test]
    fn every_go_corpus_image_roundtrips_full_state_and_fault_occurrences() {
        let directory = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../internal/model/wasi/testdata/volume-conformance");
        let mut checked = 0;
        let mut corpus_files = 0;
        for entry in std::fs::read_dir(directory).unwrap() {
            let path = entry.unwrap().path();
            if path.extension().is_none_or(|extension| extension != "json") {
                continue;
            }
            let corpus: serde_json::Value =
                serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
            corpus_files += 1;
            // Rejected session configurations have no executed operations.
            for step in corpus["observations"].as_array().into_iter().flatten() {
                if step["image_base64"]
                    .as_str()
                    .is_some_and(|data| !data.is_empty())
                {
                    roundtrip_go_image(step, &corpus["session"]["storage"]);
                    checked += 1;
                }
            }
        }
        assert!(corpus_files >= 46, "volume corpus files: {corpus_files}");
        assert!(
            checked >= 100,
            "only {checked} actual Go image observations checked"
        );
        let core: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/core.json"
        ))
        .unwrap();
        for scenario in core["scenarios"].as_array().unwrap() {
            for step in scenario["steps"].as_array().unwrap() {
                if step.get("image_base64").is_some() {
                    roundtrip_go_image(step, &scenario["storage"]);
                }
            }
        }
    }
}
