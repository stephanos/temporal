use super::{counterfactual, execution, ledger, prepare::Prepared};
use crate::{evidence::canonical, install::config, replay::preflight::Configuration};
use gomad_wasmhost::linked;
use std::{collections::BTreeMap, time::Duration};

fn prepared() -> Prepared {
    let mut configured = config::build("gomad.wasi-cooperative/v1", None, None).unwrap();
    configured.arguments = vec!["guest".into(), "literal $arg".into()];
    configured.environment = BTreeMap::from([
        ("PWD".into(), "/workspace".into()),
        ("DECLARED".into(), "unchanged".into()),
    ]);
    configured.model.as_object_mut().unwrap().remove("seed");
    let configuration = Configuration {
        engine: linked::Conditions::Ordinary.configuration(),
        profile: "gomad.wasi-cooperative/v1".into(),
        limits: crate::evidence::record::Limits {
            fuel: 10000,
            memory_bytes: 65536,
            wall_millis: 5000,
            output_bytes: configured.namespace.limits.output_bytes,
            evidence_bytes: 1 << 20,
        },
        world_transition_bytes: None,
        wall_nanos: None,
        arguments: configured.arguments,
        environment: configured.environment,
        derive_seed_environment: false,
        network: None,
        runtime_sha256: None,
        model: Some(configured.model),
        namespace: Some(configured.namespace),
        storage: configured.storage,
    };
    let module = br#"(module
        (import "wasi_snapshot_preview1" "args_sizes_get" (func $args_sizes (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "args_get" (func $args (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "environ_sizes_get" (func $env_sizes (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "environ_get" (func $env (param i32 i32) (result i32)))
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (memory (export "memory") 1)
        (func (export "_start")
            (if (call $args_sizes (i32.const 0) (i32.const 4)) (then unreachable))
            (if (i32.ne (i32.load (i32.const 0)) (i32.const 2)) (then unreachable))
            (if (i32.ne (i32.load (i32.const 4)) (i32.const 19)) (then unreachable))
            (if (call $args (i32.const 8) (i32.const 32)) (then unreachable))
            (if (call $env_sizes (i32.const 16) (i32.const 20)) (then unreachable))
            (if (i32.ne (i32.load (i32.const 16)) (i32.const 2)) (then unreachable))
            (if (i32.ne (i32.load (i32.const 20)) (i32.const 34)) (then unreachable))
            (if (call $env (i32.const 64) (i32.const 96)) (then unreachable))
            (i32.store (i32.const 160) (i32.const 32))
            (i32.store (i32.const 164) (i32.const 19))
            (if (call $write (i32.const 1) (i32.const 160) (i32.const 1) (i32.const 168)) (then unreachable))
            (i32.store (i32.const 160) (i32.const 96))
            (i32.store (i32.const 164) (i32.const 34))
            (if (call $write (i32.const 1) (i32.const 160) (i32.const 1) (i32.const 168)) (then unreachable))))"#.to_vec();
    Prepared {
        module,
        configuration,
        toolchain: "a".repeat(64),
        choice_identity: None,
        scenario: None,
        provenance: br#"{"fixture":"declared-input-controls"}"#.to_vec(),
        network: None,
        soft_assertion_report_only: false,
        compile_cache_root: None,
        auxiliary: BTreeMap::new(),
    }
}

#[test]
fn every_private_fixture_fault_keeps_complete_declared_input_serialization_and_identity() {
    let prepared = prepared();
    let before = canonical::encode(&prepared.configuration).unwrap();
    let declared: serde_json::Value = serde_json::from_slice(&before).unwrap();
    for private in [
        "guest_profile",
        "fixture_fault",
        "diagnostic_configuration_revision",
    ] {
        assert!(!declared.as_object().unwrap().contains_key(private));
    }
    let module_before = prepared.module.clone();
    let provenance_before = prepared.provenance.clone();
    let job = ledger::Job {
        ordinal: 0,
        seed: 7,
    };
    for kind in [
        linked::FixtureFault::None,
        linked::FixtureFault::Draw,
        linked::FixtureFault::Clock,
        linked::FixtureFault::Handoff,
        linked::FixtureFault::OutputClock,
    ] {
        let result = execution::run_counterfactual(
            &prepared,
            &job,
            Duration::from_secs(5),
            &linked::Cancellation::default(),
            counterfactual::Fault { kind, owner: None },
        )
        .unwrap()
        .finalized()
        .unwrap();
        assert_eq!(canonical::encode(&prepared.configuration).unwrap(), before);
        assert_eq!(prepared.module, module_before);
        assert_eq!(prepared.provenance, provenance_before);
        assert_eq!(
            result.publication.payloads["target/module.wasm"],
            module_before
        );
        assert_eq!(
            result.publication.payloads["target/provenance.json"],
            provenance_before
        );
        let retained: crate::replay::preflight::Inputs =
            serde_json::from_slice(&result.publication.payloads[crate::replay::preflight::FILE])
                .unwrap();
        assert_eq!(canonical::encode(&retained.configuration).unwrap(), before);
        let record = &result.execution.record;
        assert_eq!(record.identity.configuration, canonical::digest(&before));
        assert_eq!(record.arguments, prepared.configuration.arguments);
        assert_eq!(record.environment, prepared.configuration.environment);
        assert_eq!(record.seed, 7);
        assert_eq!(record.outcome.termination, "exit");
        assert_eq!(record.outcome.exit_code, Some(0));
        assert_eq!(
            result.publication.payloads["output/stdout"],
            b"guest\0literal $arg\0DECLARED=unchanged\0PWD=/workspace\0"
        );
    }
}
