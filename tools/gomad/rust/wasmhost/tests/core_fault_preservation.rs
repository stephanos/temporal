#![cfg(all(
    feature = "determinism-fixture",
    not(any(
        feature = "conformance",
        feature = "diagnostic-profile",
        feature = "stress-jitter"
    ))
))]

use gomad_wasmhost::linked::{self, Conditions, FixtureFault};
#[path = "common/core_preservation_input.rs"]
mod input;

#[test]
fn the_same_actual_guest_observes_only_the_explicit_execution_fault_bootstrap() {
    let guest = r#"
        (drop (call $config (i32.const 0) (i32.const 16)))
        (i32.store (i32.const 64) (i32.const 8))
        (i32.store (i32.const 68) (i32.const 8))
        (drop (call $write (i32.const 1) (i32.const 64) (i32.const 1) (i32.const 80)))
        (i32.store (i32.const 128) (i32.const 1))
        (drop (call $finish (i32.const 128) (i32.const 16)))
    "#;
    let original_module = input::request(guest).module;
    for (fault, expected) in [
        (None, 7_u64),
        (Some(FixtureFault::Draw), 7 | (1 << 63)),
        (None, 7),
        (Some(FixtureFault::OutputClock), 7 | (1 << 62)),
        (None, 7),
    ] {
        let request = input::request(guest);
        assert_eq!(request.module, original_module);
        assert_eq!(request.model.as_ref().unwrap().configuration.seed, 7);
        let mut prepared = linked::prepare(request).unwrap();
        prepared.conditions(Conditions::DeterminismFixtureReplay);
        if let Some(fault) = fault {
            prepared.fixture_fault(fault);
        }
        let observed = linked::execute_prepared(prepared).unwrap();
        assert_eq!(observed.outcome.termination, "exit");
        assert_eq!(observed.outcome.exit_code, Some(0));
        assert_eq!(observed.evidence["stdout"], expected.to_le_bytes());
        assert_eq!(observed.outcome.output_bytes, 8);
        let model = observed.outcome.local_model.as_ref().unwrap();
        assert_eq!(model["configured"], true);
        assert_eq!(model["finished"], true);
        assert_eq!(model["records"], 0);
        let events = observed.event_order.unwrap();
        assert!(events.starts_with(b"gomad-wasm-event-order-v1 "));
        assert!(
            std::str::from_utf8(&events)
                .unwrap()
                .lines()
                .last()
                .unwrap()
                .contains("\"type\":\"complete\"")
        );
    }
}
