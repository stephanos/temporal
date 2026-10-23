use super::super::evidence::{NativeExecution, NativeRun};
use super::*;
use std::collections::BTreeMap;

fn committed() -> NativeRun {
    let mut incarnations = Vec::new();
    for generation in [1, 2] {
        for node in ["active", "standby"] {
            let observation = json!({"cluster_id":node,"history":"aGk=","event_ids":[1,2],"types":["started","completed"]});
            let mut streams = BTreeMap::from([(
                "stdout".into(),
                format!("GOMAD_TEMPORAL_OBSERVATION {observation}\n").into_bytes(),
            )]);
            if generation == 1 {
                streams.insert("storage_image".into(), b"retained-image".to_vec());
            }
            incarnations.push(NativeExecution { handle:Some(json!({"node":node,"incarnation":generation})), outcome:json!({"termination":if generation == 1 {"crash"} else {"exit"},"exit_code":if generation == 1 {Value::Null} else {json!(0)}}), streams });
        }
    }
    NativeRun {
        request: Value::Null,
        report: Value::Null,
        profile: Value::Null,
        root: NativeExecution {
            handle: None,
            outcome: json!({"termination":"exit","exit_code":0}),
            streams: BTreeMap::new(),
        },
        incarnations,
        executed_steps: None,
        cutoff: None,
        modules: Vec::new(),
    }
}
fn change(run: &mut NativeRun, index: usize, field: &str, value: Value) {
    let stdout = run.incarnations[index].streams.get_mut("stdout").unwrap();
    let mut observation = crate::stress::cluster::observation(stdout).unwrap();
    observation[field] = value;
    *stdout = format!("GOMAD_TEMPORAL_OBSERVATION {observation}\n").into_bytes();
}
#[test]
fn cluster_acceptance_restores_independent_convergence_and_durable_recovery_predicates() {
    success(&committed()).unwrap();
    for kind in 0..9 {
        let mut run = committed();
        match kind {
            0 => {
                run.incarnations[0].streams.remove("storage_image");
            }
            1 => run.incarnations[0].outcome["termination"] = json!("exit"),
            2 => change(&mut run, 1, "cluster_id", json!("active")),
            3 => change(&mut run, 1, "history", json!("Ynll")),
            4 => change(&mut run, 2, "history", json!("Ynll")),
            5 => change(&mut run, 2, "event_ids", json!([1, 3])),
            6 => change(&mut run, 2, "types", json!(["started", "changed"])),
            7 => change(&mut run, 3, "cluster_id", json!("changed")),
            8 => {
                run.incarnations[2].streams.insert(
                    "stdout".into(),
                    b"GOMAD_TEMPORAL_OBSERVATION invalid\n".to_vec(),
                );
            }
            _ => unreachable!(),
        }
        assert!(success(&run).is_err(), "mutation {kind}");
    }
}
