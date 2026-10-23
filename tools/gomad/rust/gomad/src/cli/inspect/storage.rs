use crate::evidence::store::Opened;
use serde::Serialize;

#[derive(Serialize)]
pub(super) struct Storage {
    volume_count: usize,
    instances: Vec<Instance>,
}

#[derive(Serialize)]
struct Instance {
    node: String,
    incarnation: u64,
    volume_ids: Vec<String>,
}

pub(super) fn read(opened: &Opened) -> Result<Option<Storage>, String> {
    use crate::{evidence::canonical, replay::cluster};
    let manifest = opened.manifest();
    if !manifest.files.contains_key(cluster::FILE) {
        return Ok(None);
    }
    let bytes = opened.payload(cluster::FILE, 128 << 20)?;
    let inputs: cluster::Inputs =
        serde_json::from_slice(&bytes).map_err(|error| error.to_string())?;
    if inputs.schema != "gomad.cluster-replay-inputs/v1"
        || canonical::encode(&inputs)? != bytes
        || manifest.record.identity.configuration
            != canonical::digest(&canonical::encode(&inputs.configuration)?)
    {
        return Err("invalid retained cluster storage context".into());
    }
    let configured = inputs
        .configuration
        .incarnations
        .iter()
        .map(|item| ((item.handle.node.clone(), item.handle.incarnation), item))
        .collect::<std::collections::BTreeMap<_, _>>();
    if configured.len() != inputs.configuration.incarnations.len() {
        return Err("duplicate configured storage incarnation".into());
    }
    let handles = inputs
        .retained
        .iter()
        .map(|item| (item.handle.node.clone(), item.handle.incarnation))
        .collect::<Vec<_>>();
    if handles
        .iter()
        .any(|handle| !configured.contains_key(handle))
    {
        return Err("unconfigured retained storage owner".into());
    }
    let mut instances = Vec::new();
    for index in latest(&handles)? {
        let retained = &inputs.retained[index];
        let key = &handles[index];
        let configured = configured
            .get(key)
            .ok_or("unconfigured retained storage owner")?;
        let Some(storage) = &configured.storage else {
            if retained.evidence.contains_key("storage_image") {
                return Err("unbound retained storage image".into());
            }
            continue;
        };
        let image = retained
            .evidence
            .get("storage_image")
            .ok_or("missing actual terminal storage image")?;
        if image.file != format!("evidence/cluster/{}/{}/storage_image", key.0, key.1) {
            return Err("retained storage image belongs to a different incarnation".into());
        }
        let bytes = image.read(opened, 128 << 20)?;
        let volume_ids = gomad_wasmhost::linked_storage_inspection::recovery_volumes(
            storage,
            &bytes,
            &image.sha256,
            image.limit,
        )?;
        instances.push(Instance {
            node: key.0.clone(),
            incarnation: key.1,
            volume_ids,
        });
    }
    let volume_count = instances
        .iter()
        .map(|instance| instance.volume_ids.len())
        .sum();
    Ok(Some(Storage {
        volume_count,
        instances,
    }))
}

fn latest(handles: &[(String, u64)]) -> Result<Vec<usize>, String> {
    let mut owners = std::collections::BTreeMap::<&str, (u64, usize)>::new();
    let mut seen = std::collections::BTreeSet::new();
    for (index, (node, incarnation)) in handles.iter().enumerate() {
        if node.is_empty() || *incarnation == 0 || !seen.insert((node, incarnation)) {
            return Err("invalid/duplicate retained storage owner".into());
        }
        if owners
            .get(node.as_str())
            .is_none_or(|(prior, _)| incarnation > prior)
        {
            owners.insert(node, (*incarnation, index));
        }
    }
    Ok(owners.into_values().map(|(_, index)| index).collect())
}

#[cfg(test)]
mod tests {
    #[test]
    fn installed_lifecycle_latest_storage_ownership_retains_independent_nodes() {
        assert_eq!(super::latest(&[]).unwrap(), Vec::<usize>::new());
        let handles = [("a".into(), 1), ("a".into(), 2), ("b".into(), 1)];
        assert_eq!(super::latest(&handles).unwrap(), vec![1, 2]);
        assert!(super::latest(&[("a".into(), 1), ("a".into(), 1)]).is_err());
    }
    fn storage_fixture(
        extra_owner: bool,
    ) -> (
        crate::installed_lifecycle_tests::Retained,
        crate::evidence::store::Published,
    ) {
        use crate::evidence::{
            canonical,
            record::Stream,
            store::{Opened, Publication, Store},
        };
        use serde_json::json;
        use std::collections::BTreeMap;
        let fixture = crate::installed_lifecycle_tests::retained(false);
        let data: serde_json::Value = serde_json::from_slice(include_bytes!(
            "../../../../../testdata/parity/installed-lifecycle-storage-fixture.json"
        ))
        .unwrap();
        let image: Vec<u8> = serde_json::from_value(data["image"].clone()).unwrap();
        let file = "evidence/cluster/a/2/storage_image";
        let namespace = json!({"entries":[],"writable_directories":[],"preopens":[],"working_directory":[],"stdin":[],
            "limits":{"files":16,"descriptors":16,"filesystem_bytes":1_048_576,"output_bytes":1_048_576},"epoch_nanos":0});
        let configuration = json!({"engine":"inspection-only","profile":"gomad.wasi-stock-exploratory/v1",
            "limits":fixture.record.limits,"broker":{"seed":7,"limits":{"max_requests":16,"max_events":16,
                "max_queued_events":16,"max_transitions":128,"max_payload_bytes":1_048_576,"max_string_bytes":1024}},
            "network":null,"incarnations":[{"handle":{"node":"a","incarnation":2},
                "module":{"file":"target/module.wasm","bytes":8,"sha256":fixture.record.identity.target,"limit":512<<20},
                "limits":fixture.record.limits,"arguments":[],"environment":{},"runtime_sha256":"0".repeat(64),
                "model":{},"namespace":namespace,"storage":data["storage"],"network":null,
                "choice_identity":{"TargetSHA256":vec![0u8;32],"ToolchainBuildKey":"1".repeat(64),
                    "GOOS":"wasip1","GOARCH":"wasm","ImplementationSHA256":vec![0u8;32]}}],
            "steps":[],"boundaries":[],"crash_groups":[]});
        let mut inputs: crate::replay::cluster::Inputs = serde_json::from_value(json!({"schema":"gomad.cluster-replay-inputs/v1",
            "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
            "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
            "runtime_implementation_sha256":env!("GOMAD_RUNTIME_IMPLEMENTATION_SHA256"),
            "toolchain_sha256":fixture.record.identity.toolchain,"configuration":configuration,
            "retained":[{"handle":{"node":"a","incarnation":2},"interrupted":false,"evidence":{
                "storage_image":{"file":file,"bytes":image.len(),"sha256":canonical::digest(&image),"limit":1<<20}}}]})).unwrap();
        if extra_owner {
            let mut extra = inputs.configuration.incarnations[0].clone();
            extra.handle.node = "b".into();
            inputs.configuration.incarnations.push(extra);
        }
        let raw = canonical::encode(&inputs).unwrap();
        let opened = Opened::open(&fixture.published.path).unwrap();
        let mut payloads = BTreeMap::new();
        for (name, entry) in &opened.manifest().files {
            if name != "record.json" {
                payloads.insert(name.clone(), opened.payload(name, entry.bytes).unwrap());
            }
        }
        payloads.insert("evidence/cluster-replay.json".into(), raw);
        payloads.insert(
            "evidence/cluster_summary".into(),
            canonical::encode(&json!({
                "schema":"gomad.cluster-observation/v1", "virtual_elapsed_nanos":42,
                "incarnations":[{"handle":{"node":"a","incarnation":2},
                    "interrupted":false,"offered_output_bytes":0,
                    "streams":{"storage_image":{"bytes":image.len(),"sha256":canonical::digest(&image)}}}]
            })).unwrap(),
        );
        payloads.insert(file.into(), image);
        let mut record = fixture.record.clone();
        record.identity.configuration =
            canonical::digest(&canonical::encode(&inputs.configuration).unwrap());
        record.execution_identity = record.identity.digest().unwrap();
        for (name, bytes) in &payloads {
            if name != "target/module.wasm" {
                record.streams.insert(
                    name.clone(),
                    Stream {
                        bytes: bytes.len() as u64,
                        sha256: canonical::digest(bytes),
                        records: 1,
                    },
                );
            }
        }
        payloads.insert("record.json".into(), record.bytes().unwrap());
        let store = Store {
            root: fixture.root.join("volume-store"),
            maximum_bytes: 64 << 20,
            target_pool: None,
            #[cfg(feature = "determinism-fixture")]
            fault: None,
        };
        let published = store.publish(Publication { record, payloads }).unwrap();
        (fixture, published)
    }

    #[test]
    fn installed_lifecycle_inspection_refuses_configured_but_unretained_owner() {
        let (_fixture, published) = storage_fixture(true);
        let error = crate::cli::inspect::run(
            &["--json".into(), published.path.to_str().unwrap().into()],
            &mut Vec::new(),
        )
        .expect_err("configured but unretained owner was accepted");
        assert!(
            error.message.contains("cluster incarnation count mismatch"),
            "{}",
            error.message
        );
    }

    #[test]
    fn installed_lifecycle_inspection_reads_actual_volume_image_not_configured_count() {
        use serde_json::json;
        let (_fixture, published) = storage_fixture(false);
        let file = "evidence/cluster/a/2/storage_image";
        let arguments = vec!["--json".into(), published.path.to_str().unwrap().to_owned()];
        let mut output = Vec::new();
        crate::cli::inspect::run(&arguments, &mut output)
            .unwrap_or_else(|error| panic!("{}", error.message));
        let report: serde_json::Value = serde_json::from_slice(&output).unwrap();
        assert_eq!(report["artifact"]["storage"]["volume_count"], 1);
        assert_eq!(
            report["artifact"]["storage"]["instances"],
            json!([
            {"node":"a","incarnation":2,"volume_ids":["data"]}])
        );
        std::fs::write(published.path.join(file), b"corrupt image").unwrap();
        assert!(crate::cli::inspect::run(&arguments, &mut Vec::new()).is_err());
    }
}
