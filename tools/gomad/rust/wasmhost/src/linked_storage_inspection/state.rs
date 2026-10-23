use crate::{linked::StorageInput, storage_transport::StorageCrashContext, volume::Operation};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
struct Snapshot {
    #[serde(rename = "ID")]
    id: String,
    mount: String,
    capacity_bytes: u64,
    persisted: serde_json::Value,
    volatile: serde_json::Value,
    pending_operations: u64,
    #[serde(rename = "PendingSHA256")]
    pending_sha256: String,
    next_operation: u64,
    identity: String,
}

fn sections(raw: &[u8], maximum: u64) -> Result<(u32, Vec<&[u8]>), String> {
    if maximum == 0
        || raw.len() > 256 << 20
        || raw.len() as u64 > maximum
        || raw.len() < 56
        || raw.get(..8) != Some(crate::storage_transport::STATE_MAGIC)
    {
        return Err("retained storage state header/capacity".into());
    }
    let body = raw.len() - 40;
    if u64::from_le_bytes(raw[body..body + 8].try_into().unwrap()) != body as u64
        || Sha256::digest(&raw[..body]).as_slice() != &raw[body + 8..]
    {
        return Err("retained storage state integrity".into());
    }
    let version = u32::from_le_bytes(raw[8..12].try_into().unwrap());
    let count = u32::from_le_bytes(raw[12..16].try_into().unwrap());
    if !matches!((version, count), (1, 5) | (2 | 3, 6)) {
        return Err("retained storage state version/sections".into());
    }
    let mut offset = 16usize;
    let mut sections = Vec::new();
    for expected in 1..=count {
        let header = raw
            .get(offset..offset + 12)
            .filter(|_| offset + 12 <= body)
            .ok_or("truncated retained storage section")?;
        if u32::from_le_bytes(header[..4].try_into().unwrap()) != expected {
            return Err("unordered retained storage sections".into());
        }
        let size = usize::try_from(u64::from_le_bytes(header[4..].try_into().unwrap()))
            .map_err(|_| "retained storage section capacity")?;
        offset += 12;
        let end = offset
            .checked_add(size)
            .filter(|end| *end <= body)
            .ok_or("retained storage section capacity")?;
        sections.push(&raw[offset..end]);
        offset = end;
    }
    if offset != body {
        return Err("trailing retained storage state".into());
    }
    Ok((version, sections))
}

pub(super) fn validate(
    input: &StorageInput,
    raw: &[u8],
    log: &[u8],
    maximum: u64,
) -> Result<(), String> {
    let (version, sections) = sections(raw, maximum)?;
    let binding = sections[4];
    if binding.len() != 40
        || u64::from_le_bytes(binding[..8].try_into().unwrap()) != log.len() as u64
        || Sha256::digest(log).as_slice() != &binding[8..]
    {
        return Err("retained storage log binding".into());
    }
    let descriptor = input.descriptor();
    descriptor.validate().map_err(|error| error.to_string())?;
    if version == 2 {
        if !matches!(sections[5], [1] | [2]) || !sections[3].is_empty() {
            return Err("retained storage failed stage".into());
        }
    } else if (version == 3) != descriptor.crash_limits.is_some() {
        return Err("retained storage evidence: crash context".into());
    }
    let snapshots: Vec<Snapshot> =
        serde_json::from_slice(sections[0]).map_err(|error| error.to_string())?;
    let history: BTreeMap<String, Vec<Operation>> =
        serde_json::from_slice(sections[1]).map_err(|error| error.to_string())?;
    let configured = descriptor.model.volumes();
    if snapshots.len() != configured.len() || history.len() != configured.len() {
        return Err("retained storage volume inventory".into());
    }
    let mut prior = None;
    for snapshot in &snapshots {
        let config = configured
            .iter()
            .find(|config| config.id == snapshot.id)
            .ok_or("retained storage unconfigured volume")?;
        if prior.is_some_and(|id| id >= snapshot.id.as_str())
            || snapshot.mount != config.mount
            || snapshot.capacity_bytes != config.capacity_bytes
            || !snapshot.persisted.is_array()
            || !snapshot.volatile.is_array()
            || snapshot.next_operation == 0
            || snapshot.identity.is_empty()
            || snapshot.pending_operations > descriptor.model.limits.pending_operations
            || !history.contains_key(&snapshot.id)
        {
            return Err("retained storage snapshot inventory".into());
        }
        prior = Some(snapshot.id.as_str());
    }
    if version != 3 {
        return Ok(());
    }
    let contexts: BTreeMap<String, StorageCrashContext> =
        serde_json::from_slice(sections[5]).map_err(|error| error.to_string())?;
    if contexts.len() != configured.len() {
        return Err("retained storage evidence: crash context".into());
    }
    for snapshot in snapshots {
        let context = contexts
            .get(&snapshot.id)
            .ok_or("retained storage evidence: crash context")?;
        let operations = context
            .operations
            .as_ref()
            .ok_or("retained storage evidence: crash context")?;
        if context.pending_sha256 != snapshot.pending_sha256
            || operations.len() as u64 != snapshot.pending_operations
            || !context
                .pending_sha256
                .strip_prefix("sha256:")
                .is_some_and(|digest| {
                    digest.len() == 64
                        && digest
                            .bytes()
                            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
                })
        {
            return Err("retained storage evidence: crash context".into());
        }
        let observed = &history[&snapshot.id];
        let mut previous = 0;
        for operation in operations {
            if operation.id <= previous || !observed.iter().any(|actual| actual == operation) {
                return Err("retained storage evidence: crash context".into());
            }
            previous = operation.id;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{namespace, storage_transport};
    use serde_json::{Value, json};

    fn actual() -> (StorageInput, storage_transport::Evidence) {
        let model: crate::volume::StorageModel = serde_json::from_value(json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1 << 20}],
            "limits":{"PendingOperations":1000,"Transitions":10000}
        }))
        .unwrap();
        let descriptor: storage_transport::StorageDescriptor = serde_json::from_value(json!({
            "model":model,"model_identity":model.identity(),"return_image":true,
            "crash_limits":{"States":16,"Operations":8,"Depth":8,"Bytes":1 << 20,"Work":256}
        }))
        .unwrap();
        let input = serde_json::from_value(json!({"descriptor":descriptor})).unwrap();
        let mut namespace = namespace::Namespace::new(namespace::Image {
            entries: Vec::new(),
            writable_directories: vec![b"/workspace".to_vec()],
            preopens: vec![b"/workspace".to_vec()],
            working_directory: b"/workspace".to_vec(),
            stdin: Vec::new(),
            epoch_nanos: 0,
            limits: namespace::Limits {
                files: 1000,
                descriptors: 128,
                filesystem_bytes: 1 << 20,
                output_bytes: 1 << 20,
            },
        })
        .unwrap();
        descriptor.attach(&mut namespace, None, 8 << 20).unwrap();
        namespace
            .storage_mut()
            .unwrap()
            .fs
            .create("/workspace/db", crate::volume::Kind::Directory, 0o700, 1)
            .unwrap();
        let evidence = storage_transport::evidence(&mut namespace, &descriptor);
        assert_eq!(evidence.failure, None);
        (input, evidence)
    }

    #[test]
    fn actual_native_storage_state_refuses_forged_context_and_unrequested_context_presence() {
        let (input, evidence) = actual();
        validate(&input, &evidence.state, &evidence.log, 8 << 20).unwrap();
        let (_, sections) = sections(&evidence.state, 8 << 20).unwrap();
        let original: Value = serde_json::from_slice(sections[5]).unwrap();
        for edit in [
            "valid",
            "snapshot-hash",
            "operation-history",
            "missing-inventory",
        ] {
            let mut context = original.clone();
            match edit {
                "snapshot-hash" => {
                    context["data"]["PendingSHA256"] = json!(format!("sha256:{}", "b".repeat(64)))
                }
                "operation-history" => context["data"]["Operations"][0]["Kind"] = json!("forged"),
                "missing-inventory" => context["data"]["Operations"] = Value::Null,
                _ => {}
            }
            let context = serde_json::to_vec(&context).unwrap();
            let raw = storage_transport::encode_state_sections(
                sections[0],
                sections[1],
                sections[2],
                std::str::from_utf8(sections[3]).unwrap(),
                &evidence.log,
                None,
                Some(&context),
            );
            let result = validate(&input, &raw, &evidence.log, 8 << 20);
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.native-state-context","inputs":{"edit":edit,"state_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,&raw),"log_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,&evidence.log),"storage":input},"observed":{"error":result.as_ref().err()}})
            );
            assert_eq!(result.is_err(), edit != "valid", "{edit}");
        }
        let v1 = storage_transport::encode_state(
            sections[0],
            sections[1],
            sections[2],
            std::str::from_utf8(sections[3]).unwrap(),
            &evidence.log,
        );
        assert_eq!(
            validate(&input, &v1, &evidence.log, 8 << 20).unwrap_err(),
            "retained storage evidence: crash context"
        );
        let mut no_limits = serde_json::to_value(&input).unwrap();
        no_limits["descriptor"]["crash_limits"] = Value::Null;
        let no_limits: StorageInput = serde_json::from_value(no_limits).unwrap();
        assert_eq!(
            validate(&no_limits, &evidence.state, &evidence.log, 8 << 20).unwrap_err(),
            "retained storage evidence: crash context"
        );
        for (name, request, state, log) in [
            (
                "v1-with-limits",
                &input,
                v1.as_slice(),
                evidence.log.as_slice(),
            ),
            (
                "v3-without-limits",
                &no_limits,
                evidence.state.as_slice(),
                evidence.log.as_slice(),
            ),
            (
                "forged-log",
                &input,
                evidence.state.as_slice(),
                b"forged".as_slice(),
            ),
        ] {
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.native-state-context-presence","inputs":{"edit":name,"state_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,state),"log_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,log),"storage":request},"observed":{"error":validate(request,state,log,8 << 20).unwrap_err()}})
            );
        }
        for stage in [1, 2, 3] {
            let partial = storage_transport::encode_state_sections(
                sections[0],
                sections[1],
                sections[2],
                "",
                &evidence.log,
                Some(stage),
                None,
            );
            let result = validate(&input, &partial, &evidence.log, 8 << 20);
            assert_eq!(result.is_err(), stage == 3);
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1","case":"storage.native-partial-state","inputs":{"stage":stage,"state_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,&partial),"log_base64":base64::Engine::encode(&base64::engine::general_purpose::STANDARD,&evidence.log),"storage":input},"observed":{"error":result.as_ref().err()}})
            );
        }
        let mut corrupted = evidence.state.clone();
        corrupted[20] ^= 1;
        assert_eq!(
            validate(&input, &corrupted, &evidence.log, 8 << 20).unwrap_err(),
            "retained storage state integrity"
        );
    }
}
