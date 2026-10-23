use super::*;
use serde_json::{Value, json};

fn descriptor(pending: u64) -> StorageDescriptor {
    let model: StorageModel = serde_json::from_value(json!({
        "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1 << 20}],
        "limits":{"PendingOperations":pending,"Transitions":10000}
    }))
    .unwrap();
    serde_json::from_value(json!({"model":model,"model_identity":model.identity(),
        "return_image":true,"crash_limits":{"States":16,"Operations":8,"Depth":8,
        "Bytes":1 << 20,"Work":256}}))
    .unwrap()
}

fn operation(id: u64) -> Value {
    json!({"ID":id,"Kind":"write","Dependencies":null,"Inode":0,"Offset":0,
        "Bytes":0,"PayloadSHA256":"","EffectSHA256":"","Path":"","Destination":""})
}

fn context(operations: Vec<Value>) -> Value {
    json!({"data":{"PendingSHA256":format!("sha256:{}", "a".repeat(64)),
        "Operations":operations}})
}

fn attach(descriptor: &mut StorageDescriptor, context: &Value) -> Result<()> {
    let bytes = serde_json::to_vec(context).unwrap();
    descriptor.crash_context = Some(ChunkedDescriptor {
        bytes: bytes.len() as u64,
        chunks: (bytes.len() as u64).div_ceil(CHUNK),
        sha256: format!("{:x}", Sha256::digest(&bytes)),
    });
    descriptor.validate()?;
    descriptor.attach_context_bytes(Some(&bytes), 64 << 20)
}

#[test]
fn original_crash_context_admission_positive_and_negative_predicates() {
    let mut second = operation(2);
    second["Dependencies"] = json!([1]);
    let base = context(vec![operation(1), second]);
    for (name, selection, edit, want) in [
        ("future-operations", json!([1, 2]), "", ""),
        ("unknown-operation", json!([3]), "", "unknown operation"),
        ("dependency-closure", json!([2]), "", "dependency-closed"),
        ("forged-hash", json!([]), "hash", "inventory"),
        ("nil-inventory", json!([]), "nil", "inventory"),
        (
            "repeated-inventory",
            json!([]),
            "repeated",
            "operation order",
        ),
        (
            "invented-dependency",
            json!([1]),
            "dependency",
            "dependency",
        ),
    ] {
        let mut descriptor = descriptor(1000);
        descriptor.crash_selection = serde_json::from_value(json!({"data":selection})).unwrap();
        let mut input = base.clone();
        match edit {
            "hash" => input["data"]["PendingSHA256"] = json!("a"),
            "repeated" => input["data"]["Operations"] = json!([operation(1), operation(1)]),
            "nil" => input["data"]["Operations"] = Value::Null,
            "dependency" => {
                let mut op = operation(1);
                op["Dependencies"] = json!([2]);
                input["data"]["Operations"] = json!([op]);
            }
            _ => {}
        }
        let result = attach(&mut descriptor, &input);
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-context-admission","inputs":{"variant":name,"context":input,"selection":descriptor.crash_selection},"observed":{"error":result.as_ref().err().map(ToString::to_string)}})
        );
        if want.is_empty() {
            assert!(result.is_ok(), "{name}: {result:?}");
        } else {
            assert!(result.unwrap_err().to_string().contains(want), "{name}");
        }
    }
    let mut missing = descriptor(1000);
    missing.crash_selection = serde_json::from_value(json!({"data":[1]})).unwrap();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-context-admission-missing","inputs":{"descriptor":missing},"observed":{"refusal":missing.validate().unwrap_err().to_string()}})
    );
    assert!(
        missing
            .validate()
            .unwrap_err()
            .to_string()
            .contains("prior observation")
    );
}

#[test]
fn original_crash_context_equality_preserves_every_operation_field_and_null_shape() {
    let base: StorageCrashContext =
        serde_json::from_value(context(vec![operation(2)])["data"].clone()).unwrap();
    assert_eq!(base, base.clone());
    let encoded = serde_json::to_value(&base).unwrap();
    assert_eq!(encoded["Operations"][0].as_object().unwrap().len(), 10);
    for (field, value) in [
        ("ID", json!(3)),
        ("Kind", json!("forged")),
        ("Dependencies", json!([])),
        ("Inode", json!(1)),
        ("Offset", json!(1)),
        ("Bytes", json!(1)),
        ("PayloadSHA256", json!("x")),
        ("EffectSHA256", json!("x")),
        ("Path", json!("x")),
        ("Destination", json!("x")),
    ] {
        let mut changed = encoded.clone();
        changed["Operations"][0][field] = value;
        let changed: StorageCrashContext = serde_json::from_value(changed).unwrap();
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-context-equality","inputs":{"base":base,"field":field,"changed":changed},"observed":{"equal_forward":base == changed,"equal_reverse":changed == base}})
        );
        assert_ne!(base, changed, "{field}");
        assert_ne!(changed, base, "{field}");
    }
    for value in [Value::Null, json!([])] {
        let mut changed = encoded.clone();
        changed["Operations"] = value;
        let changed: StorageCrashContext = serde_json::from_value(changed).unwrap();
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-context-equality-shape","inputs":{"base":base,"changed":changed},"observed":{"equal_forward":base == changed,"equal_reverse":changed == base}})
        );
        assert_ne!(base, changed);
    }
    let mut null = encoded.clone();
    null["Operations"] = Value::Null;
    let mut empty = encoded.clone();
    empty["Operations"] = json!([]);
    assert_ne!(
        serde_json::from_value::<StorageCrashContext>(null).unwrap(),
        serde_json::from_value::<StorageCrashContext>(empty).unwrap()
    );
    let mut changed = base.clone();
    changed.pending_sha256 = format!("sha256:{}", "b".repeat(64));
    assert_ne!(base, changed);
    assert_ne!(BTreeMap::from([("data", base.clone())]), BTreeMap::new());
}

#[test]
fn original_large_context_transport_round_trips_five_thousand_operations() {
    let operations = (1..=5000)
        .map(|id| {
            let mut op = operation(id);
            op["Path"] = json!("x".repeat(256));
            op
        })
        .collect();
    let input = context(operations);
    let bytes = serde_json::to_vec(&input).unwrap();
    let mut direct = descriptor(5000);
    direct.crash_selection = serde_json::from_value(json!({"data":[1]})).unwrap();
    attach(&mut direct, &input).unwrap();
    assert!(bytes.len() > 1 << 20);
    let declared = direct.crash_context.as_ref().unwrap();
    assert!(declared.chunks >= 17);
    assert!(serde_json::to_vec(&direct).unwrap().len() <= 8192);
    let mut wire = Vec::new();
    for (sequence, chunk) in bytes.chunks(CHUNK as usize).enumerate() {
        let frame = json!({"type":"storage_context_input","sequence":sequence,
            "data_base64":STANDARD.encode(chunk)});
        wire.extend(serde_json::to_vec(&frame).unwrap());
        wire.push(b'\n');
    }
    wire.extend(b"{\"type\":\"storage_context_inputs_end\"}\n");
    let mut framed = direct.clone();
    framed.observed_context = None;
    framed
        .read_context(&mut std::io::Cursor::new(wire), 64 << 20)
        .unwrap();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-large-context-transport","inputs":{"descriptor":direct,"transport_sha256":format!("{:x}",Sha256::digest(&bytes)),"typed_context_sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(direct.observed_context.as_ref().unwrap()).unwrap()))},"observed":{"context_sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(framed.observed_context.as_ref().unwrap()).unwrap())),"operation_ids":framed.observed_context.as_ref().unwrap()["data"].operations.as_ref().unwrap().iter().map(|op|op.id).collect::<Vec<_>>()}})
    );
    assert_eq!(direct.observed_context, framed.observed_context);
    let expected: BTreeMap<String, StorageCrashContext> = serde_json::from_slice(&bytes).unwrap();
    let expected = serde_json::to_vec(&expected).unwrap();
    let actual = serde_json::to_vec(framed.observed_context.as_ref().unwrap()).unwrap();
    assert_eq!(Sha256::digest(&actual), Sha256::digest(&expected));
    assert_eq!(actual, expected);
}

fn namespace(descriptor: &StorageDescriptor) -> Namespace {
    let mut namespace = Namespace::new(crate::namespace::Image {
        entries: Vec::new(),
        writable_directories: vec![b"/workspace".to_vec()],
        preopens: vec![b"/workspace".to_vec()],
        working_directory: b"/workspace".to_vec(),
        stdin: Vec::new(),
        epoch_nanos: 0,
        limits: crate::namespace::Limits {
            files: 1000,
            descriptors: 128,
            filesystem_bytes: 1 << 20,
            output_bytes: 1 << 20,
        },
    })
    .unwrap();
    descriptor.attach(&mut namespace, None, 8 << 20).unwrap();
    namespace
}

fn observed(namespace: &Namespace) -> Value {
    let storage = namespace.storage().unwrap();
    let operations = storage.fs.pending_operations();
    let snapshots = storage.snapshots();
    let snapshot = &snapshots[0];
    json!({"data":{"PendingSHA256":snapshot.pending_sha256,
        "Operations":operations["data"]}})
}

#[test]
fn original_selected_states_recover_and_divergence_keeps_enumeration_without_image() {
    let baseline = descriptor(1000);
    let make = || {
        let mut namespace = namespace(&baseline);
        namespace
            .storage_mut()
            .unwrap()
            .fs
            .create("/workspace/db", volume::Kind::Directory, 0o700, 1)
            .unwrap();
        namespace
    };
    let mut first = make();
    let context = observed(&first);
    let page = first
        .storage()
        .unwrap()
        .enumerate("data", baseline.crash_limits.unwrap(), None)
        .unwrap();
    assert!(page.complete);
    let states = page.states.unwrap();
    assert!(states.len() >= 2);
    let original = evidence(&mut first, &baseline);
    assert_eq!(original.failure, None);
    for state in states {
        let mut selected = baseline.clone();
        selected.crash_selection = Some(BTreeMap::from([(
            "data".into(),
            Some(state.selected_operations.clone()),
        )]));
        attach(&mut selected, &context).unwrap();
        let selected = evidence(&mut make(), &selected);
        assert_eq!(selected.failure, None);
        let image = selected.image.unwrap();
        let recovered = volume::Storage::recover(
            baseline.model.clone(),
            8 << 20,
            &image,
            &format!("{:x}", Sha256::digest(&image)),
            2 << 20,
        )
        .unwrap();
        let present = recovered.fs.lookup("/workspace/db").is_ok();
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-selected-state-recovery","inputs":{"context":context,"state":state},"observed":{"recovery_image_sha256":format!("{:x}",Sha256::digest(&image)),"db_present":present,"snapshots":recovered.snapshots()}})
        );
        assert_eq!(
            present,
            state.entries.iter().any(|entry| entry.path == "/db")
        );
    }
    for edit in ["hash", "history", "missing"] {
        let mut forged = context.clone();
        match edit {
            "hash" => forged["data"]["PendingSHA256"] = json!(format!("sha256:{}", "0".repeat(64))),
            "history" => forged["data"]["Operations"][0]["Kind"] = json!("forged"),
            _ => forged["data"]["Operations"] = json!([]),
        }
        let mut requested = baseline.clone();
        attach(&mut requested, &forged).unwrap();
        let refused = evidence(&mut make(), &requested);
        assert_eq!(
            refused.failure,
            Some((
                "invalid",
                "storage crash context diverged from prior observation".into()
            )),
            "{edit}"
        );
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-context-divergence","inputs":{"variant":edit,"context":forged},"observed":{"failure":refused.failure,"image":refused.image,"state_sha256":format!("{:x}",Sha256::digest(&refused.state))}})
        );
        assert!(refused.image.is_none());
        assert_eq!(refused.log, original.log);
        let sections = |state: &[u8]| {
            let count = u32::from_le_bytes(state[12..16].try_into().unwrap());
            let mut cursor = 16;
            let mut sections = Vec::new();
            for _ in 0..count {
                let length = usize::try_from(u64::from_le_bytes(
                    state[cursor + 4..cursor + 12].try_into().unwrap(),
                ))
                .unwrap();
                sections.push(state[cursor + 12..cursor + 12 + length].to_vec());
                cursor += 12 + length;
            }
            sections
        };
        assert_eq!(
            &sections(&refused.state)[..3],
            &sections(&original.state)[..3]
        );
    }
}

#[test]
fn original_large_observed_selection_uses_fifty_four_thousand_byte_paths() {
    let mut descriptor = descriptor(1000);
    descriptor.crash_limits = serde_json::from_value(json!({"States":1,"Operations":1,
        "Depth":1,"Bytes":1 << 20,"Work":1}))
    .unwrap();
    let make = || {
        let mut namespace = namespace(&descriptor);
        for index in 0..50 {
            namespace
                .storage_mut()
                .unwrap()
                .fs
                .create(
                    &format!("/workspace/{}{:02}", "\x01".repeat(3998), index),
                    volume::Kind::File,
                    0o600,
                    1,
                )
                .unwrap();
        }
        namespace
    };
    let mut initial = make();
    let context = observed(&initial);
    assert!(serde_json::to_vec(&context).unwrap().len() > 1 << 20);
    let ids: Vec<u64> = initial.storage().unwrap().fs.pending_operations()["data"]
        .iter()
        .map(|op| op.id)
        .collect();
    assert_eq!(ids.len(), 100);
    let original = evidence(&mut initial, &descriptor);
    assert_eq!(original.failure, None);
    let mut selected = descriptor.clone();
    selected.crash_selection = Some(BTreeMap::from([("data".into(), Some(ids))]));
    attach(&mut selected, &context).unwrap();
    let result = evidence(&mut make(), &selected);
    assert_eq!(result.failure, None);
    let image = result.image.unwrap();
    let recovered = volume::Storage::recover(
        descriptor.model.clone(),
        8 << 20,
        &image,
        &format!("{:x}", Sha256::digest(&image)),
        2 << 20,
    )
    .unwrap();
    println!(
        "GOMAD_CORE_OBSERVATION {}",
        json!({"schema":"gomad.core-predicate-observation/v1","case":"storage-large-observed-selection","inputs":{"files":50,"path_bytes":4000,"context_sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(&context).unwrap())),"selection":selected.crash_selection},"observed":{"image_sha256":format!("{:x}",Sha256::digest(&image)),"snapshots_sha256":format!("{:x}",Sha256::digest(serde_json::to_vec(&recovered.snapshots()).unwrap()))}})
    );
    for index in 0..50 {
        assert!(
            recovered
                .fs
                .lookup(&format!("/workspace/{}{:02}", "\x01".repeat(3998), index))
                .is_ok()
        );
    }
}

#[test]
fn original_volume_validation_first_refusal_is_stable_for_sixty_four_orders() {
    for kind in ["selection", "context"] {
        for iteration in 0..64 {
            let model: StorageModel = serde_json::from_value(json!({
                "volumes":[{"ID":"a","Path":"/a","CapacityBytes":512},
                    {"ID":"b","Path":"/b","CapacityBytes":512}],
                "limits":{"PendingOperations":1000,"Transitions":1000}
            }))
            .unwrap();
            let mut descriptor = descriptor(1000);
            descriptor.model_identity = model.identity();
            descriptor.model = model;
            let ids = if iteration % 2 == 0 {
                ["a", "b"]
            } else {
                ["b", "a"]
            };
            let mut context = serde_json::Map::new();
            if kind == "selection" {
                let mut selection = BTreeMap::new();
                for id in ids {
                    selection.insert(
                        id.to_owned(),
                        Some(if id == "a" { vec![0] } else { vec![1, 1] }),
                    );
                }
                descriptor.crash_selection = Some(selection);
            } else {
                for id in ids {
                    context.insert(
                        id.to_owned(),
                        json!({"PendingSHA256":if id == "a" {
                        "invalid".to_owned()} else {format!("sha256:{}","a".repeat(64))},
                        "Operations":[operation(0)]}),
                    );
                }
            }
            let error = if kind == "selection" {
                descriptor.validate().unwrap_err()
            } else {
                attach(&mut descriptor, &Value::Object(context)).unwrap_err()
            };
            println!(
                "GOMAD_CORE_OBSERVATION {}",
                json!({"schema":"gomad.core-predicate-observation/v1",
                "case":"storage-validation-volume-order","inputs":{"kind":kind,"iteration":iteration,
                "insertion_ids":ids,"descriptor":descriptor},"observed":{"refusal":error.to_string()}})
            );
            assert_eq!(
                error.to_string(),
                if kind == "selection" {
                    "WASI invalid: storage crash selection zero operation"
                } else {
                    "WASI invalid: storage crash context inventory"
                }
            );
        }
    }
}
