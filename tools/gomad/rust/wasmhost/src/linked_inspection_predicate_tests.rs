use super::*;
use crate::model::inspection::{Recorder, Registration};
use sha2::{Digest, Sha256};

fn record(ordinal: u64, kind: u8, selected: u32) -> Vec<u8> {
    let mut bytes = vec![0; 96];
    bytes[..8].copy_from_slice(&ordinal.to_be_bytes());
    bytes[8] = kind;
    bytes[9] = match kind {
        3 => 2,
        4 => 0,
        _ => 1,
    };
    bytes[12..16].copy_from_slice(&2u32.to_be_bytes());
    bytes[16..20].copy_from_slice(&selected.to_be_bytes());
    bytes[24..32].copy_from_slice(&7u64.to_be_bytes());
    if kind == 3 {
        bytes[10..12].copy_from_slice(&1u16.to_be_bytes());
        bytes[12..16].copy_from_slice(&1u32.to_be_bytes());
        bytes[20..24].copy_from_slice(&1u32.to_be_bytes());
        bytes[24..32].copy_from_slice(&15u64.to_be_bytes());
        bytes[32..40].copy_from_slice(&1u64.to_be_bytes());
    } else {
        bytes[32..64].fill(if selected == 0 { 1 } else { 2 });
        if kind != 4 {
            let mut hash = Sha256::new();
            hash.update(b"gomad-choice-alternative-set/v1\0");
            hash.update(2u64.to_be_bytes());
            hash.update([1; 32]);
            hash.update([2; 32]);
            bytes[64..].copy_from_slice(&hash.finalize());
        }
    }
    bytes
}

fn fixture(records: &[Vec<u8>], coordinates: &[u64]) -> (Evidence, Vec<u8>) {
    let handle = Handle {
        node: "node".into(),
        incarnation: 2,
    };
    let mut recorder = Recorder::default();
    let mut previous = 0;
    for (decision, coordinate) in coordinates.iter().copied().enumerate() {
        for bytes in &records[previous..usize::try_from(coordinate).unwrap()] {
            recorder.stream(&handle, "choice", bytes);
        }
        previous = usize::try_from(coordinate).unwrap();
        let mut runtime_state = vec![0; 128];
        runtime_state[..8].copy_from_slice(&coordinate.to_be_bytes());
        runtime_state[104..112].copy_from_slice(&(decision as u64).to_be_bytes());
        runtime_state[112..120].copy_from_slice(&2u64.to_be_bytes());
        let witness = Witness::decode(&runtime_state).unwrap();
        recorder
            .snapshot(
                Cut {
                    handle: handle.clone(),
                    choice_records: coordinate,
                    choice_bytes: coordinate * 96,
                    witness: witness.clone(),
                    ..Cut::default()
                },
                &[],
                &[],
            )
            .unwrap();
        recorder
            .boundary(&handle, witness, 10, vec![], &[0; 32])
            .unwrap();
    }
    for bytes in &records[previous..] {
        recorder.stream(&handle, "choice", bytes);
    }
    recorder.terminal(&handle, false);
    let owner = &mut recorder.evidence.incarnations[0];
    owner.registry = Registry {
        closed: true,
        closed_order: 1,
        entries: vec![Registration {
            id: 0,
            kind: "check".into(),
            name: "healthy".into(),
            available_order: 1,
            monotonic: true,
        }],
    };
    let raw = recorder.bytes(true).unwrap();
    (Evidence::decode(&raw).unwrap(), raw)
}

fn material(evidence: &Evidence) -> Value {
    json!({"evidence":evidence,"boundaries":evidence.boundaries})
}

fn raw(root: &str, variant: &str, value: Value) {
    let mut row = json!({"schema":"gomad.native-inspection-predicate/v1","original_root":root,"variant":variant});
    row.as_object_mut()
        .unwrap()
        .insert("observed".into(), value);
    println!("{row}");
}

#[test]
fn decision_mapping_skips_observations_and_refuses_missing_or_relabelled_boundaries() {
    let mut records = vec![record(0, 1, 0), record(1, 3, 0)];
    records.extend((2..6).map(|ordinal| record(ordinal, 1, 0)));
    let (retained, _) = fixture(&records, &[0, 2, 3, 4, 5]);
    validate_choice_mapping(&retained).unwrap();
    let mut wrong = retained.clone();
    wrong.boundaries[1].trace = 1;
    wrong.boundaries[1].cuts[0].choice_records = 1;
    wrong.boundaries[1].cuts[0].choice_bytes = 96;
    assert!(validate_choice_mapping(&wrong).is_err());
    let mut missing = retained.clone();
    missing.boundaries.truncate(1);
    assert!(validate_choice_mapping(&missing).is_err());
    let mut wrong_bytes = retained.clone();
    wrong_bytes.boundaries[1].cuts[0].choice_bytes -= 1;
    assert!(validate_choice_mapping(&wrong_bytes).is_err());
    raw(
        "TestInspectionChoiceMappingUsesDecisionsRatherThanObservations",
        "observation-and-inventory",
        json!({
            "raw":material(&retained),"invalid":[material(&wrong),material(&missing),material(&wrong_bytes)],
            "errors":[validate_choice_mapping(&wrong).unwrap_err().to_string(),validate_choice_mapping(&missing).unwrap_err().to_string(),validate_choice_mapping(&wrong_bytes).unwrap_err().to_string()],
            "accepted_coordinates":[0,2,3,4,5],"observation_coordinate_refused":true,
            "missing_boundaries_refused":true,"wrong_prefix_bytes_refused":true
        }),
    );
}

#[test]
fn interrupted_select_poll_keeps_its_presented_decision_coordinate() {
    let (retained, _) = fixture(&[record(0, 2, 0)], &[0]);
    validate_choice_mapping(&retained).unwrap();
    raw(
        "TestInspectionInterruptedSelectPrefixKeepsDecisionMapping",
        "unmatched-select-poll",
        json!({
            "raw":material(&retained),"coordinates":[0],"choice_records":1,"boundaries":retained.boundaries.len()
        }),
    );
}

#[test]
fn creation_records_use_the_first_creation_coordinate_not_the_decision_record() {
    let records = (0..5)
        .flat_map(|index| {
            [
                record(index * 3, 4, 0),
                record(index * 3 + 1, 4, 1),
                record(index * 3 + 2, 1, 0),
            ]
        })
        .collect::<Vec<_>>();
    let (retained, _) = fixture(&records, &[0, 3, 6, 9, 12]);
    validate_choice_mapping(&retained).unwrap();
    let mut wrong = retained.clone();
    wrong.boundaries[1].trace = 5;
    wrong.boundaries[1].cuts[0].choice_records = 5;
    wrong.boundaries[1].cuts[0].choice_bytes = 5 * 96;
    assert!(validate_choice_mapping(&wrong).is_err());
    raw(
        "TestInspectionChoiceMappingUsesPredecisionCoordinateForCreationSites",
        "five-creation-groups",
        json!({
            "raw":material(&retained),"invalid":material(&wrong),"error":validate_choice_mapping(&wrong).unwrap_err().to_string(),"coordinates":[0,3,6,9,12],"decision_record_coordinate5_refused":true
        }),
    );
}

#[test]
fn inspection_control_binds_raw_prefix_registry_and_exact_boundary() {
    let records = (0..5)
        .map(|ordinal| record(ordinal, 1, 0))
        .collect::<Vec<_>>();
    let (retained, bytes) = fixture(&records, &[0, 1, 2, 3, 4]);
    let control = resolve(
        &retained,
        Handle::default(),
        &Stop {
            kind: "choice".into(),
            value: 2,
        },
        &[],
        &[],
        "healthy",
    )
    .unwrap();
    assert_eq!(
        serde_json::to_value(&control.expected).unwrap(),
        serde_json::to_value(&retained.boundaries[2]).unwrap()
    );
    let boundary = &control.expected;
    assert_eq!(
        control.world_sha256,
        inspection::digest(&retained.world[..usize::try_from(boundary.world_bytes).unwrap()])
    );
    assert_eq!(
        control.stream_sha256[0]["choice"],
        inspection::digest(
            &retained.incarnations[0].streams.choice
                [..usize::try_from(boundary.cuts[0].choice_bytes).unwrap()]
        )
    );
    assert_eq!(control.registries[0].entries[0].name, "healthy");
    assert_eq!(control.commands.len(), 1);
    assert_eq!(control.commands[0].kind, 4);
    let mut changed = boundary.clone();
    changed.order += 1;
    assert!(verify_prefix(&retained, &bytes, &changed).is_err());
    raw(
        "TestInspectionControlAuthenticatesRawCutAndRegistry",
        "choice2",
        json!({
            "raw":material(&retained),"expected":control.expected,"changed_boundary":changed,"error":verify_prefix(&retained,&bytes,&changed).unwrap_err().to_string(),
            "world_sha256":control.world_sha256,"stream_sha256":control.stream_sha256,
            "registries":control.registries,"commands":[{"kind":control.commands[0].kind,"id":control.commands[0].id}],
            "changed_order_refused":true
        }),
    );
    assert!(
        resolve(
            &retained,
            Handle::default(),
            &Stop {
                kind: "choice".into(),
                value: 0
            },
            &[],
            &[],
            "missing"
        )
        .err()
        .unwrap()
        .to_string()
        .contains("unknown or unavailable")
    );
    let mut incomplete = retained.clone();
    incomplete.complete = false;
    assert!(
        resolve(
            &incomplete,
            Handle::default(),
            &Stop {
                kind: "choice".into(),
                value: 0
            },
            &[],
            &[],
            "healthy"
        )
        .is_err()
    );
    raw(
        "TestInspectionControlRefusesUnregisteredAndIncompleteBeforeReplay",
        "missing-check-and-incomplete",
        json!({"raw":material(&retained),"incomplete":material(&incomplete),
            "errors":[resolve(&retained,Handle::default(),&Stop{kind:"choice".into(),value:0},&[],&[],"missing").err().unwrap().to_string(),resolve(&incomplete,Handle::default(),&Stop{kind:"choice".into(),value:0},&[],&[],"healthy").err().unwrap().to_string()],
            "missing_check_refused":true,"incomplete_refused":true}),
    );
}

#[test]
fn terminal_network_raw_stream_cannot_be_replaced_by_an_unrelated_state_commitment() {
    let network = include_bytes!("../tests/testdata/cluster_network/a.gmnetev1");
    assert_eq!(
        inspection::digest(network),
        "00a0074b5731ede6751ec6de87fdcdc3df56ebd6ef2bf2b1357cfef11c97465a"
    );
    let model: crate::linked_cluster::NetworkModel = serde_json::from_value(json!({
        "profile":"gomad.simulation-network-stream/v3",
        "limits":{"listeners":128,"connections":128,"bytes":8<<20,
            "connection_bytes":4<<20,"stall_nanos":10_000_000_000u64,"transitions":8192}
    }))
    .unwrap();
    crate::linked_cluster::validate_network_evidence(network, 1 << 20, &model, &model).unwrap();
    let mut trailing = network.to_vec();
    trailing.push(0);
    assert!(
        crate::linked_cluster::validate_network_evidence(&trailing, 1 << 20, &model, &model)
            .is_err()
    );
    let mut recorder = Recorder::default();
    for node in ["a", "b"] {
        let handle = Handle {
            node: node.into(),
            incarnation: 1,
        };
        let mut state = vec![0; 128];
        state[112..120].copy_from_slice(&2u64.to_be_bytes());
        let witness = Witness::decode(&state).unwrap();
        recorder
            .snapshot(
                Cut {
                    handle: handle.clone(),
                    witness: witness.clone(),
                    ..Cut::default()
                },
                &[],
                &[],
            )
            .unwrap();
        recorder
            .boundary(&handle, witness, 10, vec![], &[0; 32])
            .unwrap();
        recorder.stream(&handle, "choice", &record(0, 1, 0));
        state[..8].copy_from_slice(&1u64.to_be_bytes());
        state[104..112].copy_from_slice(&1u64.to_be_bytes());
        recorder
            .snapshot(
                Cut {
                    handle: handle.clone(),
                    choice_records: 1,
                    choice_bytes: 96,
                    witness: Witness::decode(&state).unwrap(),
                    ..Cut::default()
                },
                &[],
                &[],
            )
            .unwrap();
        if node == "a" {
            recorder.terminal_network(&handle, network.to_vec());
        }
        recorder.terminal(&handle, false);
    }
    let bytes = recorder.bytes(true).unwrap();
    let mut retained = Evidence::decode(&bytes).unwrap();
    validate_choice_mapping(&retained).unwrap();
    let valid_material = material(&retained);
    let owner = &retained.incarnations[0];
    assert_eq!(owner.streams.network, network);
    let terminal = retained.boundaries[1]
        .cuts
        .iter_mut()
        .find(|cut| cut.handle.node == "a")
        .unwrap();
    assert_eq!(terminal.network_bytes, network.len() as u64);
    terminal.network_state[0] ^= 1;
    assert!(
        validate(&retained)
            .err()
            .unwrap()
            .to_string()
            .contains("terminal network state commitment")
    );
    raw(
        "TestInspectionNetworkRawStreamBindsCanonicalIdentity",
        "native-terminal-raw-state",
        json!({
            "raw":valid_material,"invalid":material(&retained),"error":validate(&retained).unwrap_err().to_string(),"network_base64":STANDARD.encode(network),"trailing_error":crate::linked_cluster::validate_network_evidence(&trailing,1<<20,&model,&model).unwrap_err(),
            "raw_sha256":inspection::digest(network),"raw_bytes":network.len(),
            "trailing_raw_byte_refused":true,"changed_terminal_commitment_refused":true
        }),
    );
}

/// Original `TestInspectionOutputBoundsIncludeJSONAndHeap`: one aggregate
/// bound covers the JSON document and heap profile, truncation is reported in
/// both flags and the dump status, and a bound too small for the truncation
/// metadata refuses.
#[test]
fn inspection_output_bound_includes_json_and_heap() {
    let diagnostics = || Diagnostics {
        output: json!({"dumps": [{"name": "snapshot", "status": "ok", "data_base64": STANDARD.encode([2u8; 4096])}]}),
        heap: vec![1; 1024],
    };
    let (encoded, heap, truncated) = diagnostics().bound(512).unwrap();
    assert!(truncated);
    assert!(encoded.len() + heap.len() <= 512);
    let result: Value = serde_json::from_slice(&encoded).unwrap();
    assert_eq!(result["truncated"], true);
    assert_eq!(result["heap_truncated"], true);
    assert_eq!(result["dumps"].as_array().unwrap().len(), 1);
    assert_eq!(result["dumps"][0]["status"], "truncated");
    assert!(diagnostics().bound(1).is_err());
}

fn debug_envelope(operation: u32, payload: &[u8]) -> Vec<u8> {
    let mut bytes = operation.to_le_bytes().to_vec();
    bytes.extend_from_slice(&u32::try_from(payload.len()).unwrap().to_le_bytes());
    for value in [7u64, 9, 11] {
        bytes.extend_from_slice(&value.to_le_bytes());
    }
    bytes.extend_from_slice(payload);
    bytes
}

fn debug_frame(identity: u64, index: u32, function: &str, file: &str) -> Vec<u8> {
    let mut bytes = identity.to_le_bytes().to_vec();
    for value in [0u32, index] {
        bytes.extend_from_slice(&value.to_le_bytes());
    }
    bytes.extend_from_slice(&0x1234u64.to_le_bytes());
    for value in [42, function.len(), file.len()] {
        bytes.extend_from_slice(&u32::try_from(value).unwrap().to_le_bytes());
    }
    bytes.extend_from_slice(function.as_bytes());
    bytes.extend_from_slice(file.as_bytes());
    bytes
}

fn debug_target() -> Handle {
    Handle {
        node: "a".into(),
        incarnation: 1,
    }
}

/// Original `TestDebugLibraryRecordsPreserveFailuresAndRawStrings`: each dump
/// keeps its own failure status, a panicking check never becomes predicate
/// truth, raw string bytes and host coordinates survive, and every malformed
/// record is refused.
#[test]
fn debug_library_records_preserve_failures_raw_strings_and_coordinates() {
    let record = |kind: u32, id: u32, status: u32, data: &[u8]| {
        let mut bytes = Vec::new();
        for value in [kind, id, status, u32::try_from(data.len()).unwrap()] {
            bytes.extend_from_slice(&value.to_le_bytes());
        }
        bytes.extend_from_slice(data);
        debug_envelope(4, &bytes)
    };
    let raw = [
        record(2, 3, 1, b"panic"),
        record(2, 4, 2, b"partial"),
        record(2, 5, 0, b"next dump"),
        record(3, 6, 2, b""),
        record(1, 7, 1, &[5, 2, 0, 0, 0, 0xff, 0]),
    ]
    .concat();
    let output = debug_output(&raw, &debug_target(), &Registry::default(), 1 << 20)
        .unwrap()
        .output;
    let dumps = output["dumps"].as_array().unwrap();
    assert_eq!(dumps.len(), 3);
    assert_eq!(
        [&dumps[0]["status"], &dumps[1]["status"]],
        ["panic", "truncated"]
    );
    assert_eq!(dumps[2]["data_base64"], STANDARD.encode(b"next dump"));
    assert_eq!(output["checks"].as_array().unwrap().len(), 1);
    assert_eq!(output["checks"][0]["status"], "panic");
    let probes = output["probes"].as_array().unwrap();
    assert_eq!(probes.len(), 1);
    assert_eq!(
        probes[0]["fields"][0]["bytes_base64"],
        STANDARD.encode([0xff, 0])
    );
    assert_eq!(
        (
            &probes[0]["decision"],
            &probes[0]["trace"],
            &probes[0]["virtual_time_nanos"]
        ),
        (&json!(7), &json!(9), &json!(11))
    );
    for malformed in [
        vec![4, 0, 0],
        debug_envelope(99, &[]),
        debug_envelope(4, &[1]),
        record(1, 0, 1, &[3, 1, 0, 0, 0, 2]),
        record(3, 0, 0, &[1]),
    ] {
        assert!(
            debug_output(&malformed, &debug_target(), &Registry::default(), 1 << 20).is_err(),
            "malformed debug record admitted: {malformed:x?}"
        );
    }
}

/// Original `TestDebugHeapProfileOpensInPprof`: the sampling policy, the
/// goroutine stack and a gzip pprof profile whose `inuse_space` sample is the
/// 3072 live bytes attributed to `main.workload`.
#[test]
fn debug_heap_profile_keeps_rate_stack_and_inuse_sample() {
    let mut sample = Vec::new();
    for value in [1u64, 4, 1, 4096, 1024] {
        sample.extend_from_slice(&value.to_le_bytes());
    }
    sample.extend_from_slice(&0u32.to_le_bytes());
    sample.extend_from_slice(&1u32.to_le_bytes());
    sample.extend_from_slice(&0x1235u64.to_le_bytes());
    let mut metadata = vec![0; 48];
    metadata[..8].copy_from_slice(&1u64.to_le_bytes());
    metadata[40..44].copy_from_slice(&u32::MAX.to_le_bytes());
    let raw = [
        debug_envelope(9, &metadata),
        debug_envelope(9, &sample),
        debug_envelope(11, &debug_frame(0, 0, "main.workload", "fixture.go")),
        debug_envelope(8, &debug_frame(17, 0, "main.locked", "fixture.go")),
    ]
    .concat();
    let diagnostics = debug_output(&raw, &debug_target(), &Registry::default(), 1 << 20).unwrap();
    assert_eq!(diagnostics.output["heap_sample_rate"], 1);
    let stacks = diagnostics.output["stacks"].as_array().unwrap();
    assert_eq!(stacks.len(), 1);
    assert_eq!(stacks[0]["goroutine"], 17);
    assert_eq!(stacks[0]["frames"][0]["function"], "main.locked");
    validate_heap_profile(&diagnostics.heap, 1 << 20).unwrap();
    let (profile, _) = crate::world::gzip::decode(&diagnostics.heap, 1 << 20).unwrap();
    // Sample values: alloc_objects 4, alloc_space 4096, inuse_objects 3,
    // inuse_space 3072, packed as field 2 of the sample message.
    let values = [0x12, 6, 4, 0x80, 0x20, 3, 0x80, 0x18];
    assert!(profile.windows(values.len()).any(|window| window == values));
    assert!(
        profile
            .windows(b"main.workload".len())
            .any(|window| window == b"main.workload")
    );
}
