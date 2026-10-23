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
    pub(crate) fn volume_ids(&self) -> Vec<String> {
        self.fs.volumes.keys().cloned().collect()
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

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{Engine, engine::general_purpose::STANDARD};

    #[test]
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
