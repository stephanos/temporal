use super::{Model, get32, replay, replay_plan, set32, set64};
use sha2::{Digest as _, Sha256};

fn model() -> Model {
    let mut model = super::tests::model();
    model.config.limit = 8192;
    model
}

fn decision(reversed: bool) -> Vec<u8> {
    let mut bytes = vec![0; 96];
    set32(&mut bytes, 0, 1);
    set32(&mut bytes, 4, 1);
    set64(&mut bytes, 8, 10);
    set32(&mut bytes, 20, 2);
    bytes[32..64].fill(if reversed { 2 } else { 1 });
    bytes[64..96].fill(if reversed { 1 } else { 2 });
    bytes
}

fn finish() -> Vec<u8> {
    let mut bytes = vec![0; 16];
    set32(&mut bytes, 0, 1);
    bytes
}

fn identity() -> replay_plan::ExecutionIdentity {
    replay_plan::ExecutionIdentity {
        target_sha256: [1; 32],
        toolchain_build_key: "02".repeat(32),
        goos: "wasip1".into(),
        goarch: "wasm".into(),
        implementation_sha256: [3; 32],
    }
}

fn admitted(trace: &[u8]) -> wasmtime::Result<replay_plan::ReplayPlan> {
    let plan = replay_plan::project(trace, &identity()).map_err(wasmtime::Error::msg)?;
    replay::preflight(
        trace,
        &replay::Request {
            schema: replay::SCHEMA.into(),
            execution_identity: identity(),
            trace_sha256: format!("{:x}", Sha256::digest(trace)),
            tape_sha256: format!("{:x}", Sha256::digest(&plan.bytes)),
            decisions: plan.decisions.len().try_into().unwrap(),
            trace_bytes: trace.len().try_into().unwrap(),
            trace_frames: u64::from(!trace.is_empty()),
            prefix_control: None,
        },
        2,
        8192,
    )
}

#[test]
fn runtime_replay_forces_logical_identity_and_refuses_each_original_boundary_mismatch() {
    let mut recorded = model();
    recorded.runtime("runtime_config", vec![0; 16]).unwrap();
    let selected = recorded
        .runtime("runtime_decision", decision(false))
        .unwrap();
    assert_eq!(get32(&selected, 24), 0);
    recorded.runtime("runtime_finish", finish()).unwrap();
    let trace = recorded.pending["choice"].clone();
    assert_eq!(trace.len(), 96);
    assert_eq!(
        admitted(&trace).unwrap().decisions[0].selected_identity,
        [1; 32]
    );

    for case in [
        "exact",
        "missing",
        "extra",
        "kind",
        "high-kind",
        "site",
        "enabled",
        "duplicate",
        "unconfigured",
    ] {
        let source = if case == "missing" {
            &[][..]
        } else {
            trace.as_slice()
        };
        let mut replayed = model();
        replayed.config.mode = 2;
        replayed.config.tape = Some(admitted(source).unwrap().decisions);
        if case != "unconfigured" {
            replayed.runtime("runtime_config", vec![0; 16]).unwrap();
        }
        let mut input = decision(true);
        match case {
            "kind" => set32(&mut input, 4, 2),
            "high-kind" => set32(&mut input, 4, 257),
            "site" => set64(&mut input, 8, 11),
            "enabled" => input[32..64].fill(3),
            "duplicate" => input[32..64].fill(1),
            _ => {}
        }
        if case == "extra" {
            assert!(replayed.runtime("runtime_finish", finish()).is_err());
            assert_eq!(replayed.cursor, 0);
        } else if case == "exact" {
            let forced = replayed.runtime("runtime_decision", input).unwrap();
            assert_eq!(get32(&forced, 24), 1);
            replayed.runtime("runtime_finish", finish()).unwrap();
            assert_eq!(replayed.cursor, 1);
            assert_eq!(replayed.pending["choice"], trace);
            assert_eq!(replayed.terminal, finish());
        } else {
            assert!(
                replayed.runtime("runtime_decision", input).is_err(),
                "{case}"
            );
            assert_eq!(replayed.records, 0, "{case}");
            assert_eq!(replayed.cursor, 0, "{case}");
            assert!(
                replayed.pending.get("choice").is_none_or(Vec::is_empty),
                "{case}"
            );
            assert!(!replayed.finished, "{case}");
        }
    }
    for case in ["rank", "readiness"] {
        let mut malformed = trace.clone();
        if case == "rank" {
            set32(&mut malformed, 16, 99);
        } else {
            malformed[10..12].copy_from_slice(&1_u16.to_le_bytes());
        }
        assert!(admitted(&malformed).is_err(), "{case}");
    }
}
