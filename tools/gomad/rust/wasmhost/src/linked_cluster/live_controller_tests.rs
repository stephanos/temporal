use super::*;
use serde_json::{Value, json};
use std::{fs, path::Path};

fn number(value: &Value, field: &str) -> u64 {
    value[field]
        .as_u64()
        .unwrap_or_else(|| panic!("missing actual request {field}"))
}

fn prepared(input: &Value) -> Prepared {
    let config = &input["configuration"];
    let broker: BrokerConfig = serde_json::from_value(config["broker"].clone()).unwrap();
    let incarnations = config["incarnations"]
        .as_array()
        .unwrap()
        .iter()
        .map(|item| {
            let module_file = item["module"]["file"].as_str().unwrap();
            let path = input["modules"][module_file].as_str().unwrap();
            let module = fs::read(path).unwrap();
            assert!(
                module.starts_with(b"\0asm"),
                "root35 requires actual unchanged compiled service guest"
            );
            assert!(
                module.len() <= 8 << 20,
                "original service module admission bound"
            );
            let limits = &item["limits"];
            let mut model = item["model"].clone();
            model["seed"] = json!(broker.seed);
            Incarnation {
                handle: serde_json::from_value(item["handle"].clone()).unwrap(),
                network: serde_json::from_value(item["network"].clone()).unwrap(),
                request: linked::Request {
                    module,
                    module_sha256: item["module"]["sha256"].as_str().unwrap().into(),
                    limits: linked::Limits {
                        memory_bytes: number(limits, "memory_bytes"),
                        fuel: number(limits, "fuel"),
                        output_bytes: number(limits, "output_bytes"),
                        evidence_bytes: number(limits, "evidence_bytes"),
                        wall_timeout: Duration::from_millis(number(limits, "wall_millis")),
                    },
                    model: Some(linked::ModelInput {
                        configuration: serde_json::from_value(model).unwrap(),
                        arguments: serde_json::from_value(item["arguments"].clone()).unwrap(),
                        environment: item["environment"]
                            .as_object()
                            .unwrap()
                            .iter()
                            .map(|(name, value)| format!("{name}={}", value.as_str().unwrap()))
                            .collect(),
                        replay_trace: None,
                    }),
                    namespace: Some(serde_json::from_value(item["namespace"].clone()).unwrap()),
                    storage: serde_json::from_value(item["storage"].clone()).unwrap(),
                },
            }
        })
        .collect();
    let mut prepared = prepare(Request {
        nodes: serde_json::from_value(config["nodes"].clone()).unwrap(),
        broker,
        network: serde_json::from_value(config["network"].clone()).unwrap(),
        incarnations,
        steps: serde_json::from_value(config["steps"].clone()).unwrap(),
        boundaries: serde_json::from_value(config["boundaries"].clone()).unwrap(),
        crash_groups: serde_json::from_value(config["crash_groups"].clone()).unwrap(),
        wall_timeout: Duration::from_millis(number(&config["limits"], "wall_millis")),
        evidence_bytes: number(&config["limits"], "evidence_bytes"),
    })
    .unwrap();
    assert_eq!(prepared.steps.len(), 2);
    assert_eq!(prepared.steps[0].handle.node, "a");
    assert_eq!(prepared.steps[1].handle.node, "b");
    assert!(
        prepared
            .steps
            .iter()
            .all(|step| step.action == Action::Start)
    );
    prepared
        .cleanup_after_starts(CleanupMode::Shutdown)
        .unwrap();
    prepared
}

#[test]
#[ignore = "requires original root35 compiled service input; mandatory live corpus invokes explicitly"]
fn shutdown_reinsertion_preserves_actual_live_guest_world() {
    let input_path = std::env::var_os("GOMAD_LIVE_CLUSTER_SHUTDOWN_INPUT")
        .expect("mandatory original root35 request path");
    let output_path = std::env::var_os("GOMAD_LIVE_CLUSTER_SHUTDOWN_OUTPUT")
        .expect("mandatory new root35 evidence directory");
    let input: Value = serde_json::from_slice(&fs::read(input_path).unwrap()).unwrap();
    assert_eq!(input["schema"], "gomad.live-cluster-observation/v1");
    let output = Path::new(&output_path);
    fs::create_dir(output).unwrap();
    let mut baseline = None;
    let mut receipts = Vec::new();
    for iteration in 0..16 {
        let mut prepared = prepared(&input);
        let live = Arc::new(AtomicBool::new(false));
        prepared.shutdown_reinsertion = Some((iteration % 2 != 0, live.clone()));
        let observed = execute_prepared(prepared).unwrap();
        assert!(
            live.load(Ordering::Acquire),
            "both original started handles must be live at reinsertion"
        );
        assert_eq!(observed.incarnations.len(), 2);
        assert_eq!(
            observed.executed_steps.len(),
            2,
            "shutdown invented lifecycle actions"
        );
        assert_eq!(observed.outcome.termination, "exit");
        assert_eq!(observed.outcome.exit_code, Some(0));
        assert!(!observed.world.is_empty());
        let actual = (observed.world, observed.network);
        if let Some(expected) = &baseline {
            assert_eq!(&actual, expected, "iteration {iteration}");
        } else {
            baseline = Some(actual.clone());
        }
        fs::write(output.join(format!("world-{iteration}.bin")), &actual.0).unwrap();
        fs::write(output.join(format!("network-{iteration}.bin")), &actual.1).unwrap();
        receipts.push(
            json!({"iteration":iteration,"reverse_insertion":iteration % 2 != 0,
            "live_verified":live.load(Ordering::Acquire),"completed":observed.incarnations.len(),
            "world_bytes":actual.0.len(),"network_bytes":actual.1.len()}),
        );
    }
    fs::write(
        output.join("result.json"),
        serde_json::to_vec(&json!({
            "schema":"gomad.live-cluster-shutdown-reinsertion/v1","iterations":receipts,
            "conditions":"ordinary","unchanged_guest":true,
        }))
        .unwrap(),
    )
    .unwrap();
    println!("root35: 16 actual post-Start live-map reconstruction and shutdown comparisons");
}
