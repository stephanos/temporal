use super::*;

fn retain(
    configuration: &mut super::Configuration,
    steps: &[super::helper::Step],
) -> Result<(), String> {
    crate::campaign::scenario::retain_execution_prefix(configuration, steps, false)
}

#[test]
fn failed_wait_keeps_declared_ordered_join_recipe_without_inventing_actions() {
    let corpus = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let initial = publication(&corpus, 7).unwrap();
    let original: Inputs = serde_json::from_slice(&initial.payloads[cluster::FILE]).unwrap();
    let mut configuration = original.configuration;
    configuration
        .incarnations
        .retain(|item| item.handle.incarnation == 1);
    let source = std::fs::read_to_string(corpus.join("targets/inspect-choice.wat")).unwrap();
    let source = source.replacen(
        "(module",
        "(module (import \"wasi_snapshot_preview1\" \"proc_exit\" (func $exit (param i32)))",
        1,
    );
    let module = format!(
        "{} i32.const 1 call $exit))",
        source.trim_end().strip_suffix("))").unwrap()
    )
    .into_bytes();
    let file = "target/module.wasm";
    for item in &mut configuration.incarnations {
        item.imports = None;
        item.module = descriptor(file.into(), &module, 512 << 20);
        item.choice_identity.target_sha256 = sha2::Sha256::digest(&module).into();
    }
    configuration.steps = [helper::Action::Start, helper::Action::Wait]
        .into_iter()
        .flat_map(|action| {
            configuration
                .incarnations
                .iter()
                .map(move |item| helper::Step {
                    action,
                    handle: item.handle.clone(),
                })
        })
        .collect();
    let modules = BTreeMap::from([(file.into(), module)]);
    let executed =
        helper::execute_prepared(prepare_configuration(&configuration, &modules).unwrap()).unwrap();
    assert_eq!(executed.incarnations.len(), 2);
    assert!(
        executed
            .incarnations
            .iter()
            .any(|item| item.execution.outcome.exit_code == Some(1))
    );
    assert_eq!(executed.executed_steps.len(), 3);
    assert_eq!(configuration.steps.len(), 4);
    assert_eq!(
        executed.executed_steps,
        configuration.steps[..executed.executed_steps.len()]
    );
    let original_bytes = canonical::encode(&configuration).unwrap();
    retain(&mut configuration, &executed.executed_steps).unwrap();
    assert_eq!(canonical::encode(&configuration).unwrap(), original_bytes);

    invalid_suffixes_stay_refused(&configuration, &executed.executed_steps);
    let world = executed.world.clone();
    let publication =
        execute_configuration(configuration, original.toolchain_sha256, modules).unwrap();
    assert_eq!(publication.record.outcome.class, "target_failure");
    assert_eq!(publication.payloads["evidence/world"], world);
    public_replay(publication);
}

fn invalid_suffixes_stay_refused(configuration: &Configuration, actual: &[helper::Step]) {
    for action in [
        helper::Action::Start,
        helper::Action::Restart,
        helper::Action::Crash,
        helper::Action::Stop,
    ] {
        let mut invalid = configuration.clone();
        invalid.steps[3].action = action;
        assert!(retain(&mut invalid, &actual[..2]).is_err());
    }
    for tail in [
        vec![],
        vec![configuration.steps[2].clone()],
        vec![
            configuration.steps[2].clone(),
            configuration.steps[3].clone(),
            configuration.steps[3].clone(),
        ],
        vec![
            configuration.steps[3].clone(),
            configuration.steps[2].clone(),
        ],
    ] {
        let mut invalid = configuration.clone();
        invalid.steps.truncate(2);
        invalid.steps.extend(tail);
        assert!(retain(&mut invalid, &actual[..2]).is_err());
    }
    let mut foreign = actual.to_vec();
    foreign.push(helper::Step {
        handle: helper::Handle {
            node: "foreign".into(),
            incarnation: 1,
        },
        action: helper::Action::Wait,
    });
    assert!(retain(&mut configuration.clone(), &foreign).is_err());
    let mut divergent = actual.to_vec();
    divergent[2].action = helper::Action::Stop;
    assert!(retain(&mut configuration.clone(), &divergent).is_err());
    assert!(retain(&mut configuration.clone(), &actual[..1]).is_err());
}

fn public_replay(publication: Publication) {
    use crate::evidence::store::{Opened, Store};
    let root = std::env::temp_dir().join(format!(
        "gomad-implicit-drain-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    let published = Store {
        root: root.clone(),
        maximum_bytes: 64 << 20,
        target_pool: None,
        fault: None,
    }
    .publish(publication)
    .unwrap();
    let opened = Opened::open(&published.path).unwrap();
    opened.authenticate_all().unwrap();
    let retained = opened.payload(cluster::FILE, 16 << 20).unwrap();
    let input: Inputs = serde_json::from_slice(&retained).unwrap();
    assert_eq!(input.configuration.steps.len(), 4);
    let mut output = Vec::new();
    let status = crate::cli::replay::run(
        &[published.path.to_string_lossy().into_owned()],
        &mut output,
    )
    .map_err(|error| error.message)
    .unwrap();
    assert_eq!(status, 1, "{}", String::from_utf8_lossy(&output));
    assert!(
        String::from_utf8(output)
            .unwrap()
            .contains("reproduced=true")
    );
    assert_eq!(opened.payload(cluster::FILE, 16 << 20).unwrap(), retained);
    std::fs::remove_dir_all(root).unwrap();
}
