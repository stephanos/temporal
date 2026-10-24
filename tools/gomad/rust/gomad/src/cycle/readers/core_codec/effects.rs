use super::{BTreeMap, BTreeSet, Digest, Sha256, Value, codec, json};
use codec::{
    RecoveryEffect, RecoveryObject, RecoveryObservation, RecoveryOperation, RecoveryVolume,
    StateObservation,
};

struct Hash(Sha256);
impl Hash {
    fn new(domain: &str) -> Self {
        let mut h = Self(Sha256::new());
        h.text(domain);
        h
    }
    fn number(&mut self, n: u64) {
        self.0.update(n.to_le_bytes());
    }
    fn bytes(&mut self, b: &[u8]) {
        self.number(b.len() as u64);
        self.0.update(b);
    }
    fn text(&mut self, s: &str) {
        self.bytes(s.as_bytes());
    }
    fn finish(self) -> String {
        format!("sha256:{:x}", self.0.finalize())
    }
}
fn effect(h: &mut Hash, e: &RecoveryEffect) {
    h.text(e.kind());
    h.number(e.inode());
    h.number(e.parent());
    for p in e.parents() {
        h.number(*p);
    }
    for n in [
        e.offset(),
        e.size(),
        u64::from(e.mode()),
        e.mod_time().cast_unsigned(),
    ] {
        h.number(n);
    }
    h.bytes(e.data());
    if let Some(o) = e.object() {
        for n in [
            o.inode(),
            u64::from(o.mode()),
            o.kind(),
            o.mod_time().cast_unsigned(),
        ] {
            h.number(n);
        }
        h.bytes(o.data());
    }
    for p in e.namespace() {
        h.text(p.path());
        h.number(p.inode());
    }
}
fn pending(id: &str, ops: &[RecoveryOperation]) -> String {
    let mut h = Hash::new("gomad-volume-pending/v1");
    h.text(id);
    for op in ops {
        h.number(op.id());
        h.text(op.kind());
        for d in op.dependencies() {
            h.number(*d);
        }
        effect(&mut h, op.effect());
    }
    h.finish()
}
fn operation(op: &RecoveryOperation) -> Value {
    let e = op.effect();
    let mut h = Hash::new("gomad-volume-effect/v1");
    effect(&mut h, e);
    json!({"ID":op.id(),"Kind":op.kind(),"Dependencies":if op.dependencies().is_empty(){Value::Null}else{json!(op.dependencies())},
        "Inode":e.inode(),"Offset":e.offset(),"Bytes":if e.data().is_empty(){e.size()}else{e.data().len() as u64},
        "PayloadSHA256":if e.data().is_empty(){String::new()}else{format!("sha256:{:x}",Sha256::digest(e.data()))},
        "EffectSHA256":h.finish(),"Path":e.namespace().first().map_or("",|p|p.path()),
        "Destination":if e.namespace().len()>1 {e.namespace().last().unwrap().path()}else{""}})
}
#[derive(Clone)]
struct Object {
    mode: u32,
    kind: u64,
    time: i64,
    data: Vec<u8>,
}
impl From<&RecoveryObject> for Object {
    fn from(o: &RecoveryObject) -> Self {
        Self {
            mode: o.mode(),
            kind: o.kind(),
            time: o.mod_time(),
            data: o.data().to_vec(),
        }
    }
}
#[derive(Clone)]
struct Durable {
    objects: BTreeMap<u64, Object>,
    paths: BTreeMap<String, u64>,
}
impl Durable {
    fn new(v: &RecoveryVolume) -> Self {
        Self {
            objects: v
                .objects()
                .iter()
                .map(|(id, o)| (*id, Object::from(o)))
                .collect(),
            paths: v.paths().clone(),
        }
    }
    fn apply(&mut self, e: &RecoveryEffect) -> Result<(), String> {
        match e.kind() {
            "alloc" => {
                let o = e.object().ok_or("selected allocation object absent")?;
                if o.inode() != e.inode()
                    || self.objects.insert(e.inode(), Object::from(o)).is_some()
                {
                    return Err("selected allocation reused inode".into());
                }
            }
            "write" | "resize" | "metadata" => {
                let o = self
                    .objects
                    .get_mut(&e.inode())
                    .ok_or("selected effect inode absent")?;
                if e.kind() == "metadata" {
                    o.mode = e.mode();
                } else {
                    if o.kind != 1 {
                        return Err("selected file effect used directory".into());
                    }
                    if e.size() > 256 << 20
                        || e.offset()
                            .checked_add(e.data().len() as u64)
                            .is_none_or(|end| end > 256 << 20)
                    {
                        return Err("selected persistence bound".into());
                    }
                    if e.kind() == "resize" {
                        o.data.resize(
                            usize::try_from(e.size()).map_err(|_| "selected size overflow")?,
                            0,
                        );
                    } else {
                        let start =
                            usize::try_from(e.offset()).map_err(|_| "selected offset overflow")?;
                        let end = start
                            .checked_add(e.data().len())
                            .ok_or("selected write overflow")?;
                        o.data
                            .get_mut(start..end)
                            .ok_or("selected write exceeds resized file")?
                            .copy_from_slice(e.data());
                    }
                }
                o.time = e.mod_time();
            }
            "namespace" => {
                for part in e.namespace() {
                    if !part.path().starts_with('/') || part.path() == "/" {
                        return Err("selected namespace path".into());
                    }
                    if part.inode() == 0 {
                        self.paths.remove(part.path());
                    } else {
                        if !self.objects.contains_key(&part.inode()) {
                            return Err("selected namespace inode absent".into());
                        }
                        self.paths.insert(part.path().into(), part.inode());
                    }
                }
                if let Some(o) = self.objects.get_mut(&e.inode()) {
                    o.time = e.mod_time();
                }
            }
            _ => return Err("unknown selected effect".into()),
        }
        Ok(())
    }
    fn entries(&self, crash: bool) -> Result<Value, String> {
        self.paths.iter().map(|(path,id)|{
            let o=self.objects.get(id).ok_or("durable path inode absent")?;
            Ok(if crash {json!({"Path":path,"Mode":o.mode,"Kind":o.kind,"ModTime":o.time,"Data":if o.data.is_empty(){Value::Null}else{json!(base64(&o.data))}})}
                else {json!({"Path":path,"Mode":o.mode,"Kind":if o.kind==1{"file"}else{"directory"},"ModTime":o.time,"Size":o.data.len(),"DataSHA256":if o.kind==1{format!("sha256:{:x}",Sha256::digest(&o.data))}else{String::new()}})})
        }).collect::<Result<Vec<_>,String>>().map(Value::Array)
    }
}
fn base64(raw: &[u8]) -> String {
    const TABLE: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::new();
    for chunk in raw.chunks(3) {
        let n = (u32::from(chunk[0]) << 16)
            | (u32::from(*chunk.get(1).unwrap_or(&0)) << 8)
            | u32::from(*chunk.get(2).unwrap_or(&0));
        out.push(TABLE[((n >> 18) & 63) as usize] as char);
        out.push(TABLE[((n >> 12) & 63) as usize] as char);
        out.push(if chunk.len() > 1 {
            TABLE[((n >> 6) & 63) as usize] as char
        } else {
            '='
        });
        out.push(if chunk.len() > 2 {
            TABLE[(n & 63) as usize] as char
        } else {
            '='
        });
    }
    out
}
fn crash_identity(id: &str, d: &Durable) -> Result<String, String> {
    let mut h = Hash::new("gomad-volume-crash-state/v1");
    h.text(id);
    for (path, inode) in &d.paths {
        let o = d.objects.get(inode).ok_or("crash inode absent")?;
        h.text(path);
        h.number(u64::from(o.mode));
        h.number(o.kind);
        h.number(o.time.cast_unsigned());
        h.bytes(&o.data);
    }
    Ok(h.finish())
}
fn snapshot_identity(snapshot: &Value) -> Result<String, String> {
    let text = |v: &Value, k: &str| {
        v[k].as_str()
            .map(str::to_owned)
            .ok_or_else(|| format!("snapshot {k}"))
    };
    let number = |v: &Value, k: &str| {
        v[k].as_u64()
            .or_else(|| v[k].as_i64().map(i64::cast_unsigned))
            .ok_or_else(|| format!("snapshot {k}"))
    };
    let mut h = Hash::new("gomad-volume-snapshot/v1");
    h.text(&text(snapshot, "ID")?);
    h.text(&text(snapshot, "Mount")?);
    h.number(number(snapshot, "CapacityBytes")?);
    for field in ["Persisted", "Volatile"] {
        let entries = snapshot[field].as_array().ok_or("snapshot entries")?;
        h.number(entries.len() as u64);
        for e in entries {
            h.text(&text(e, "Path")?);
            h.number(number(e, "Mode")?);
            h.text(&text(e, "Kind")?);
            h.number(number(e, "ModTime")?);
            h.number(number(e, "Size")?);
            h.text(&text(e, "DataSHA256")?);
        }
    }
    h.number(number(snapshot, "PendingOperations")?);
    h.text(&text(snapshot, "PendingSHA256")?);
    h.number(number(snapshot, "NextOperation")?);
    Ok(h.finish())
}

pub(super) fn baseline(
    image: &RecoveryObservation,
    state: &StateObservation<'_>,
) -> Result<Vec<Value>, String> {
    let [v] = image.volumes() else {
        return Err("baseline volume inventory".into());
    };
    if v.id() != "data"
        || v.mount() != "/workspace"
        || v.capacity_bytes() != 1 << 20
        || !v.pending().is_empty()
        || v.history().len() != 2
        || v.history()[0].kind() != "alloc"
        || v.history()[1].kind() != "namespace"
        || v.history()[0]
            .effect()
            .object()
            .is_none_or(|o| o.kind() != 2 || o.mode() != 0o777)
        || v.history()[1].effect().namespace().len() != 1
        || v.history()[1].effect().namespace()[0].path() != "/db"
    {
        return Err("baseline mkdir effects changed".into());
    }
    let history: Vec<_> = v.history().iter().map(operation).collect();
    let pending = pending(v.id(), v.history());
    let context = json!({"data":{"PendingSHA256":pending,"Operations":history}});
    if state.contexts() != Some(&context) || state.history() != &json!({"data":history}) {
        return Err("baseline independently derived context/history".into());
    }
    let durable = Durable::new(v);
    let mut live = durable.clone();
    for op in v.history() {
        if op.kind() == "alloc" {
            live.apply(op.effect())?;
        } else {
            for part in op.effect().namespace() {
                if part.path() == "/" || !live.objects.contains_key(&part.inode()) {
                    return Err("invalid live creation namespace".into());
                }
                live.paths.insert(part.path().into(), part.inode());
            }
        }
    }
    let mut snapshot = json!({"ID":v.id(),"Mount":v.mount(),"CapacityBytes":v.capacity_bytes(),"Persisted":durable.entries(false)?,"Volatile":live.entries(false)?,"PendingOperations":v.history().len(),"PendingSHA256":pending,"NextOperation":v.next_operation()});
    snapshot["Identity"] = json!(snapshot_identity(&snapshot)?);
    if state.snapshots() != &json!([snapshot]) {
        return Err("baseline independently derived snapshots".into());
    }
    let page = &state.enumerations()["data"];
    if page["Complete"] != true || page["Capacity"] != "" {
        return Err("baseline enumeration incomplete".into());
    }
    let states = page["States"].as_array().ok_or("baseline states absent")?;
    let mut expected = Vec::new();
    let mut seen = BTreeSet::new();
    for mask in 0..(1usize << v.history().len()) {
        let ids: BTreeSet<_> = v
            .history()
            .iter()
            .enumerate()
            .filter(|(index, _)| mask & (1 << index) != 0)
            .map(|(_, op)| op.id())
            .collect();
        if v.history()
            .iter()
            .filter(|op| ids.contains(&op.id()))
            .any(|op| op.dependencies().iter().any(|dep| !ids.contains(dep)))
        {
            continue;
        }
        let mut selected = durable.clone();
        for op in v.history().iter().filter(|op| ids.contains(&op.id())) {
            selected.apply(op.effect())?;
        }
        let identity = crash_identity(v.id(), &selected)?;
        if seen.insert(identity.clone()) {
            expected.push(json!({"Volume":v.id(),"PendingSHA256":pending,"SelectedOperations":ids,"Entries":selected.entries(true)?,"Identity":identity}));
        }
    }
    if expected.len() < 2 || states != &expected {
        return Err("enumeration persistence effects/order/coverage".into());
    }
    Ok(expected)
}

pub(super) fn selected(
    baseline: &RecoveryObservation,
    image: &RecoveryObservation,
    state: &Value,
) -> Result<(), String> {
    let [prior] = baseline.volumes() else {
        return Err("baseline volume inventory".into());
    };
    let [actual] = image.volumes() else {
        return Err("selected volume inventory".into());
    };
    if image.model_identity() != baseline.model_identity()
        || actual.id() != prior.id()
        || actual.mount() != prior.mount()
        || actual.capacity_bytes() != prior.capacity_bytes()
        || actual.history() != prior.history()
        || actual.next_operation() != prior.next_operation()
    {
        return Err("selected image baseline provenance".into());
    }
    if Durable::new(actual).entries(true)? != state["Entries"] {
        return Err("selected image durable effects".into());
    }
    Ok(())
}

pub(super) fn large_selection(
    baseline: &RecoveryObservation,
    selected: &RecoveryObservation,
    state: &StateObservation<'_>,
) -> Result<(), String> {
    let [prior] = baseline.volumes() else {
        return Err("large volume inventory".into());
    };
    let [actual] = selected.volumes() else {
        return Err("large selected volume inventory".into());
    };
    if prior.history().len() != 100
        || actual.history() != prior.history()
        || !actual.pending().is_empty()
    {
        return Err("large selected history inventory".into());
    }
    let context = json!({"data":{"PendingSHA256":pending("data",prior.history()),"Operations":prior.history().iter().map(operation).collect::<Vec<_>>()}});
    if serde_json::to_vec(&context)
        .map_err(|e| e.to_string())?
        .len()
        <= 1 << 20
        || state.contexts() != Some(&context)
    {
        return Err("large context raw/history binding".into());
    }
    let mut durable = Durable::new(prior);
    for (index, pair) in prior.history().as_chunks::<2>().0.iter().enumerate() {
        let expected = format!("/{}{:02}", "\x01".repeat(3998), index);
        if pair[0].kind() != "alloc"
            || pair[1].kind() != "namespace"
            || pair[0]
                .effect()
                .object()
                .is_none_or(|o| o.kind() != 1 || !o.data().is_empty())
            || pair[1].effect().namespace().len() != 1
            || pair[1].effect().namespace()[0].path() != expected
            || pair[1].effect().namespace()[0].inode() != pair[0].effect().inode()
        {
            return Err("large creation effects changed".into());
        }
        durable.apply(pair[0].effect())?;
        durable.apply(pair[1].effect())?;
    }
    if actual.paths().len() != 51 || durable.entries(true)? != Durable::new(actual).entries(true)? {
        return Err("large selected durable entries".into());
    }
    Ok(())
}

pub(super) fn recovered(
    image: &RecoveryObservation,
    state: &StateObservation<'_>,
) -> Result<(), String> {
    let snapshots=image.volumes().iter().map(|volume|{
        let entries=Durable::new(volume).entries(false)?;
        let mut snapshot=json!({"ID":volume.id(),"Mount":volume.mount(),"CapacityBytes":volume.capacity_bytes(),"Persisted":entries,"Volatile":entries,"PendingOperations":0,"PendingSHA256":pending(volume.id(),&[]),"NextOperation":volume.next_operation()});
        snapshot["Identity"]=json!(snapshot_identity(&snapshot)?);Ok(snapshot)
    }).collect::<Result<Vec<Value>,String>>()?;
    if state.snapshots() != &json!(snapshots) {
        return Err("recovered namespace snapshots differ from image".into());
    }
    Ok(())
}

pub(super) fn pending_effects(
    before: &RecoveryObservation,
    after: &RecoveryObservation,
) -> Result<(), String> {
    if before.model_identity() != after.model_identity()
        || before.volumes().len() != after.volumes().len()
    {
        return Err("effect replay model/volume provenance".into());
    }
    for (original, actual) in before.volumes().iter().zip(after.volumes()) {
        if original.id() != actual.id()
            || original.history() != actual.history()
            || !actual.pending().is_empty()
        {
            return Err("effect replay history provenance".into());
        }
        let mut durable = Durable::new(original);
        for operation in original.pending() {
            durable.apply(operation.effect())?;
        }
        if durable.entries(true)? != Durable::new(actual).entries(true)? {
            return Err("independently replayed pending effects differ".into());
        }
    }
    Ok(())
}

#[cfg(test)]
pub(super) fn semantic_guard_cases(image: &RecoveryObservation) {
    let volume = &image.volumes()[0];
    let mut durable = Durable::new(volume);
    let allocation = volume
        .history()
        .iter()
        .find(|op| op.kind() == "alloc")
        .unwrap()
        .effect();
    let mut fresh = durable.clone();
    fresh.objects.remove(&allocation.inode());
    fresh.apply(allocation).unwrap();
    assert_eq!(
        fresh.apply(allocation).unwrap_err(),
        "selected allocation reused inode"
    );
    let mut kinds = BTreeSet::new();
    for operation in volume.pending() {
        let effect = operation.effect();
        if matches!(effect.kind(), "write" | "resize" | "metadata") {
            kinds.insert(effect.kind());
            let mut missing = durable.clone();
            missing.objects.remove(&effect.inode());
            assert_eq!(
                missing.apply(effect).unwrap_err(),
                "selected effect inode absent"
            );
            if effect.kind() != "metadata" {
                let mut directory = durable.clone();
                directory.objects.get_mut(&effect.inode()).unwrap().kind = 2;
                assert_eq!(
                    directory.apply(effect).unwrap_err(),
                    "selected file effect used directory"
                );
            }
            if effect.kind() == "write" {
                let mut truncated = durable.clone();
                truncated
                    .objects
                    .get_mut(&effect.inode())
                    .unwrap()
                    .data
                    .clear();
                assert_eq!(
                    truncated.apply(effect).unwrap_err(),
                    "selected write exceeds resized file"
                );
            }
        }
        if effect.kind() == "namespace" {
            let mut missing = durable.clone();
            missing.objects.remove(&effect.namespace()[0].inode());
            assert_eq!(
                missing.apply(effect).unwrap_err(),
                "selected namespace inode absent"
            );
        }
        durable.apply(effect).unwrap();
    }
    assert_eq!(kinds, BTreeSet::from(["metadata", "resize", "write"]));
}
