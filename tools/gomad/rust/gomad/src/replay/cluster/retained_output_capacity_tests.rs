use super::*;

const MODULE: &[u8] = br#"(module
    (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
    (memory (export "memory") 1)
    (data (i32.const 128) "PASSx")
    (func (export "_start")
        i32.const 0 i32.const 128 i32.store
        i32.const 4 i32.const 4 i32.store
        i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop
        i32.const 0 i32.const 132 i32.store
        i32.const 4 i32.const 1 i32.store
        i32.const 1 i32.const 0 i32.const 1 i32.const 8 call $write drop))"#;

fn configuration(report_only: bool, maximum: u64) -> Configuration {
    let mut configuration = soft_configuration(report_only);
    configuration.nodes = vec!["a".into()];
    let item = &mut configuration.incarnations[0];
    item.handle = helper::Handle {
        node: "a".into(),
        incarnation: 1,
    };
    item.module.bytes = MODULE.len() as u64;
    item.module.sha256 = canonical::digest(MODULE);
    item.model = crate::fixture_execute::model_configuration(configuration.broker.seed, 4096, 0);
    item.model.as_object_mut().unwrap().remove("seed");
    item.namespace = crate::fixture_execute::namespace();
    item.environment = BTreeMap::from([("PWD".into(), "/".into())]);
    item.namespace.limits.output_bytes = 4;
    item.limits.output_bytes = 4;
    item.choice_identity.target_sha256 = sha2::Sha256::digest(MODULE).into();
    configuration.limits.output_bytes = 4;
    configuration.steps = [helper::Action::Start, helper::Action::Wait]
        .map(|action| helper::Step {
            action,
            handle: item.handle.clone(),
        })
        .to_vec();
    configuration.oracle_plan = report_only.then(|| {
        serde_json::from_value(json!({
            "schema":"gomad.wasm.soft-assertion-observation/v1","maximum_bytes":maximum,
            "soft_assertions":[{"handle":item.handle,"report_only":true}]
        }))
        .unwrap()
    });
    configuration
}

#[test]
fn actual_wasi_offered_output_capacity_never_acquires_replayable_publication() {
    for (report_only, maximum) in [(false, 4096), (true, 4096), (true, 1)] {
        let configuration = configuration(report_only, maximum);
        let modules = BTreeMap::from([(
            configuration.incarnations[0].module.file.clone(),
            MODULE.to_vec(),
        )]);
        let prepared =
            crate::fixture_cluster::prepare_configuration(&configuration, &modules).unwrap();
        let activity = prepared.activity();
        let execution = helper::execute_prepared(prepared).unwrap();
        assert_eq!(execution.incarnations.len(), 1);
        assert_eq!(execution.executed_steps, configuration.steps);
        let item = &execution.incarnations[0].execution;
        assert_eq!(item.outcome.termination, "capacity");
        assert_eq!(item.outcome.message, "WASI capacity: offered-output-bytes");
        assert_eq!(item.outcome.exit_code, None);
        assert_eq!(item.outcome.output_bytes, 4);
        assert_eq!(item.evidence.get("stdout").unwrap(), b"PASS");
        assert!(item.evidence.get("choice").is_none_or(Vec::is_empty));
        assert!(item.authorization_to_terminal_nanos.is_some());
        assert_eq!(activity.snapshot().launched_stores, 1);
        assert_eq!(activity.snapshot().remote_callback_attempts, 0);
        let annotation = oracles::capacity_observation(&execution, &configuration);
        let failure = &annotation["capacity_failure"];
        assert_eq!(failure["termination"], "capacity");
        assert_eq!(failure["message"], "output capacity");
        assert_eq!(failure["soft_assertions"]["inconclusive"], true);
        assert_eq!(failure["soft_assertions"]["report_only"], report_only);
        assert_eq!(failure["soft_assertions"]["messages"], json!([]));
        let assessment = match oracles::assess(&execution, &configuration) {
            Ok(report) => {
                let report = serde_json::to_value(report).unwrap();
                assert_eq!(report["oracles"].as_array().unwrap().len(), 1);
                assert_eq!(report["oracles"][0]["passed"], false);
                assert_eq!(report["oracles"][0]["inconclusive"], true);
                assert_eq!(report["oracles"][0]["report_only"], report_only);
                assert_ne!(maximum, 1);
                json!({"report":report})
            }
            Err(error) => {
                assert_eq!(maximum, 1);
                assert!(error.contains("resource=oracle_evidence_bytes"));
                json!({"error":error})
            }
        };
        let refusal = crate::fixture_cluster::execute_configuration(
            configuration.clone(),
            canonical::digest(b"native-output-capacity-fixture"),
            modules,
        )
        .err()
        .expect("incomplete output capacity acquired a publication");
        assert!(refusal.starts_with("capacity: output capacity;"));
        assert!(refusal.contains(if maximum == 1 {
            "resource=oracle_evidence_bytes"
        } else {
            "choice trace is not complete"
        }));
        println!(
            "GOMAD_OUTPUT_CAPACITY_OBSERVATION {}",
            json!({
                "schema":"gomad.native-output-capacity-observation/v1",
                "inputs":{"module_bytes":MODULE,"configuration":configuration,"report_only":report_only,"maximum_bytes":maximum},
                "observed":{"execution":execution_input(&execution,&configuration),"steps":execution.executed_steps,
                    "authorization_to_terminal_nanos":item.authorization_to_terminal_nanos,"activity":activity.snapshot(),
                    "annotation":annotation,"assessment":assessment,"publication_refusal":refusal}
            })
        );
    }
}
