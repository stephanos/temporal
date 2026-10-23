#![cfg(not(feature = "conformance"))]

use base64::{Engine as _, engine::general_purpose::STANDARD};
use gomad_wasmhost::linked_inspection::{self, Evidence};
use serde_json::{Value, json};

fn word(out: &mut Vec<u8>, value: u32) {
    out.extend(value.to_le_bytes());
}
fn number(out: &mut Vec<u8>, value: u64) {
    out.extend(value.to_le_bytes());
}
fn handle(out: &mut Vec<u8>, incarnation: u64) {
    word(out, 1);
    out.push(b'a');
    number(out, incarnation);
}

fn fixture(change: &str) -> Vec<u8> {
    let mut world = b"gomad.inspection-world-commitments/v2\0".to_vec();
    let world_header_bytes = world.len() as u64;
    let mut lifecycle = b"GMWIL\x01\0\0".to_vec();
    let mut life_cuts = Vec::new();
    for (action, incarnation) in [(1, 1), (2, 1), (1, 2), (2, 2), (3, 1), (3, 2)] {
        lifecycle.push(action);
        handle(&mut lifecycle, incarnation);
        if action == 2 {
            life_cuts.push(lifecycle.len() as u64);
            number(&mut world, 32);
            world.extend([u8::try_from(incarnation).unwrap(); 32]);
        }
    }
    let mut frames = b"GMWIB\x01\0\0".to_vec();
    word(&mut frames, 2);
    for index in 0..2u64 {
        let mut boundary = Vec::new();
        handle(&mut boundary, index + 1);
        let trace = if index == 0 { 17 } else { 0 };
        for value in [
            0,
            trace,
            2 + index * 2,
            100 + index,
            world_header_bytes + (index + 1) * 40,
            life_cuts[usize::try_from(index).unwrap()],
        ] {
            number(&mut boundary, value);
        }
        word(&mut boundary, u32::try_from(index + 1).unwrap());
        for incarnation in 1..=index + 1 {
            handle(&mut boundary, incarnation);
            let records = if incarnation == 1 { 17 } else { 0 };
            let stdout = if change == "output-cut" { 1 } else { 2 };
            for value in [records, records * 96, 0, 0, 0, stdout, 0] {
                number(&mut boundary, value);
            }
            word(&mut boundary, 0);
            boundary.extend([0; 32]);
            boundary.extend([0; 32]);
            boundary.extend(if change == "recovery-state" {
                [1; 32]
            } else {
                [0; 32]
            });
            let mut witness = [0u8; 128];
            witness[..8].copy_from_slice(&records.to_be_bytes());
            witness[104..112]
                .copy_from_slice(&if incarnation == 1 { 7u64 } else { 0u64 }.to_be_bytes());
            witness[112..120].copy_from_slice(&1u64.to_be_bytes());
            if change == "witness" {
                witness[16..24].copy_from_slice(&1u64.to_be_bytes());
            }
            boundary.extend(witness);
        }
        word(&mut frames, u32::try_from(boundary.len()).unwrap());
        frames.extend(boundary);
    }
    let owners:Vec<Value>=(1..=2).map(|incarnation| {
        let mut choice=vec![0u8;if incarnation==1 {17*96} else {0}];
        if change=="choice" && !choice.is_empty() { choice[16]=1; }
        json!({"handle":{"node":"a","incarnation":incarnation},"started_order":if incarnation==1 {1} else {3},"terminal_order":if incarnation==1 {5} else {6},
            "registry":{"closed":false,"closed_order":0,"entries":[]},
            "streams":{"choice":STANDARD.encode(choice),"wasi":"","network":"","storage":"","stdout":STANDARD.encode(if change=="output" {b"changed".as_slice()} else {b"out".as_slice()}),"stderr":""}})
    }).collect();
    serde_json::to_vec(&json!({"schema":"gomad.wasm-inspection/v3","boundary_encoding":"gomad.inspection-boundaries-binary/v1","lifecycle_encoding":"gomad.inspection-lifecycle-binary/v1","state_encoding":"gomad.inspection-state-sha256/v2","world_encoding":"gomad.inspection-world-sha256/v2","complete":true,"world":STANDARD.encode(world),"lifecycle":STANDARD.encode(lifecycle),"boundary_frames":STANDARD.encode(frames),"incarnations":owners})).unwrap()
}

#[test]
fn native_inspection_preserves_trace_decision_cuts_and_recovery_witness() {
    let raw = fixture("");
    let retained = Evidence::decode(&raw).unwrap();
    assert_eq!(retained.boundaries.len(), 2);
    assert_eq!(retained.boundaries[0].trace, 17);
    assert_eq!(retained.boundaries[0].decision, 0);
    assert_eq!(retained.boundaries[0].order, 2);
    assert_eq!(retained.boundaries[0].cuts[0].stdout_bytes, 2);
    for change in [
        "choice",
        "output",
        "output-cut",
        "recovery-state",
        "witness",
    ] {
        let changed = fixture(change);
        Evidence::decode(&changed).unwrap();
        assert!(
            linked_inspection::verify_prefix(&retained, &changed, &retained.boundaries[0]).is_err(),
            "{change}"
        );
    }
    linked_inspection::verify_prefix(&retained, &raw, &retained.boundaries[0]).unwrap();
}

#[test]
fn native_inspection_rejects_unknown_schema_and_tracks_zero_choice_incarnation() {
    let raw = fixture("");
    let retained = Evidence::decode(&raw).unwrap();
    let zero = &retained.incarnations[1];
    assert_eq!(zero.handle.incarnation, 2);
    assert_eq!(zero.started_order, 3);
    assert_eq!(zero.streams.choice, [] as [u8; 0]);
    let boundary = &retained.boundaries[1];
    assert_eq!(boundary.handle.incarnation, 2);
    assert_eq!(boundary.trace, 0);
    assert_eq!(boundary.decision, 0);
    assert_eq!(boundary.order, 4);
    assert_eq!(boundary.time_nanos, 101);
    assert_eq!(boundary.cuts.len(), 2);
    let mut malformed: Value = serde_json::from_slice(&raw).unwrap();
    malformed["schema"] = "gomad.wasm-inspection/v99".into();
    assert!(Evidence::decode(&serde_json::to_vec(&malformed).unwrap()).is_err());
}
