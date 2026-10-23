use super::*;

const ROOT: &str = "TestProviderBindsEverySelectedBootNetworkLimit";
const VARIANTS: &[&str] = &[
    "matching",
    "network free",
    "first byte mismatch",
    "later byte mismatch",
    "first stall mismatch",
    "later stall mismatch",
    "restart byte mismatch",
    "restart stall mismatch",
    "matching explicit broker",
    "matching restart",
    "independent local capacities",
    "broker byte mismatch",
    "broker stall mismatch",
];

fn model() -> NetworkModel {
    serde_json::from_value(json!({
        "profile":"gomad.simulation-network-stream/v3",
        "limits":{"listeners":8,"connections":8,"bytes":1024,"connection_bytes":123,"stall_nanos":456,"transitions":128},
        "faults":[]
    })).unwrap()
}

fn declared(input: &Request) -> Value {
    json!({
        "nodes":input.nodes,
        "shared":input.network,
        "selected":input.incarnations.iter().map(|item| json!({
            "node":item.handle.node,"incarnation":item.handle.incarnation,"network":item.network,
            "model_seed":item.request.model.as_ref().unwrap().configuration.seed
        })).collect::<Vec<_>>(),
        "steps":input.steps,
        "broker_seed":input.broker.seed
    })
}

#[test]
fn selected_boot_network_limits_bind_first_later_restart_and_explicit_shared_models() {
    for variant in VARIANTS {
        let local = model();
        let mut shared = local.clone();
        if variant.starts_with("broker ") {
            if variant.contains("stall") {
                shared.limits.stall_nanos += 1;
            } else {
                shared.limits.connection_bytes += 1;
            }
        }
        let mut selected = vec![
            ("a".to_owned(), 1_u64, local.clone()),
            ("b".to_owned(), 1_u64, local.clone()),
        ];
        if variant.starts_with("restart ") || *variant == "matching restart" {
            selected.push(("a".into(), 2, local.clone()));
        }
        let mismatch = if variant.starts_with("first ") {
            Some(0)
        } else if variant.starts_with("later ") {
            Some(1)
        } else if variant.starts_with("restart ") {
            Some(2)
        } else {
            None
        };
        if let Some(index) = mismatch {
            if variant.contains("stall") {
                selected[index].2.limits.stall_nanos += 1;
            } else {
                selected[index].2.limits.connection_bytes += 1;
            }
        }
        if *variant == "independent local capacities" {
            let limits = &mut selected[1].2.limits;
            limits.bytes *= 2;
            limits.transitions *= 2;
            limits.listeners *= 2;
            limits.connections *= 2;
        }
        let source = json!({"shared":(*variant != "network free").then_some(&shared),"selected":selected.iter().map(|(node,incarnation,network)|json!({"node":node,"incarnation":incarnation,"network":(*variant != "network free").then_some(network)})).collect::<Vec<_>>()});
        let before = source.clone();
        let mut input = output_wait_request(1, "", &["a", "b"], Action::Crash);
        input.boundaries.clear();
        input.network = if *variant == "network free" {
            None
        } else {
            Some(serde_json::from_value(json!({"model":shared,"addresses":{},"links":[],"faults":[],"strict_topology":false})).unwrap())
        };
        input.incarnations.clear();
        input.steps.clear();
        for (node, incarnation, network) in &selected {
            let mut item = output_wait_request(1, "", &[node.as_str()], Action::Crash)
                .incarnations
                .remove(0);
            item.handle.incarnation = *incarnation;
            item.network = (*variant != "network free").then(|| network.clone());
            input.steps.push(Step {
                action: if *incarnation == 1 {
                    Action::Start
                } else {
                    Action::Restart
                },
                handle: item.handle.clone(),
            });
            input.steps.push(Step {
                action: Action::Wait,
                handle: item.handle.clone(),
            });
            input.incarnations.push(item);
        }
        let declaration = declared(&input);
        let result = prepare(input);
        let observed = match result {
            Ok(prepared) => {
                assert!(
                    !variant.contains("mismatch"),
                    "{variant}: admitted mismatched identity-bound network limits"
                );
                let encoded = serde_json::to_vec(&prepared.network).unwrap();
                json!({"shared_json_bytes":encoded,"shared_json_sha256":format!("{:x}",Sha256::digest(&encoded)),"error":"","bound":{"shared":prepared.network,"selected":prepared.incarnations.iter().map(|(handle,sealed)|json!({"node":handle.node,"incarnation":handle.incarnation,"network":sealed.network})).collect::<Vec<_>>(),"owners":prepared.ids},"activity":prepared.activity().snapshot()})
            }
            Err(error) => {
                assert!(variant.contains("mismatch"), "{variant}: {error}");
                assert_eq!(error.to_string(), "cluster boot network limits");
                json!({"error":error.to_string()})
            }
        };
        let after = json!({"shared":(*variant != "network free").then_some(&shared),"selected":selected.iter().map(|(node,incarnation,network)|json!({"node":node,"incarnation":incarnation,"network":(*variant != "network free").then_some(network)})).collect::<Vec<_>>()});
        assert_eq!(source, before);
        assert_eq!(before, after);
        println!(
            "GOMAD_CORE_OBSERVATION {}",
            json!({
                "schema":"gomad.core-predicate-observation/v1","case":"boot-network.selected-model-binding",
                "inputs":{"original_root":ROOT,"variant":variant,"caller_before":before,"caller_after":after,"declared":declaration},
                "observed":observed
            })
        );
    }
}
