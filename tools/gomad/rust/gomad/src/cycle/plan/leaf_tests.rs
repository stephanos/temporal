use super::leaf::{Binding, Expansion, LeafBindings, LeafFlag};
use super::*;
use serde_json::Value;

fn bindings() -> LeafBindings {
    use Binding::*;
    let mut result = LeafBindings::default();
    for (key, value) in [
        (Repo, "/home/agent/wt/fn242-obligation-inventory"),
        (Root, "/sealed/cycle"),
        (Config, "/sealed/driver.json"),
        (Binary, "/sealed/release"),
        (Python, "/sealed/python"),
        (Go, "/sealed/tools/go"),
        (Cargo, "/sealed/tools/cargo"),
        (Rustc, "/sealed/tools/rustc"),
        (Clippy, "/sealed/tools/cargo_clippy"),
        (Make, "/sealed/tools/make"),
        (Golangci, "/sealed/tools/golangci"),
        (Errortype, "/sealed/tools/errortype"),
        (True, "/sealed/tools/true"),
        (CargoHome, "/sealed/cache/cargo_home"),
        (RuntimeHarness, "/sealed/runtime-harness"),
        (Bootstrap, "/sealed/bootstrap"),
        (Compiler, "/sealed/compiler"),
        (RuntimeRoot, "/sealed/runtime-root"),
        (Cache, "/sealed/cache"),
        (CapturedConfig, "/sealed/guest-config"),
        (SourceManifest, "/sealed/source.sha256"),
        (BuildReceipt, "/sealed/release-build.json"),
        (CompilerRoot, "/sealed/compiler-root"),
        (GuestConfig, "/sealed/guest-config"),
        (ModuleCache, "/sealed/module-cache"),
        (ClusterTemplate, "/sealed/cluster-template"),
        (BrokerObservation, "/sealed/observation"),
        (HostFixture, "/sealed/roles/host-fixture"),
        (HostTests, "/sealed/roles/ordinary-host-test"),
        (HelperTests, "/sealed/roles/ordinary-helper-test"),
        (StockGo, "/sealed/go"),
        (FreshOrdinary, "/sealed/fresh-ordinary"),
        (FreshAdmission, "/sealed/fresh-ordinary-admission"),
        (FreshBuild, "/sealed/fresh-build/helper-build.json"),
        (Diagnostic, "/sealed/diag"),
        (DiagnosticAdmission, "/sealed/diag-admission"),
        (Module, "/sealed/module"),
        (ModuleReceipt, "/sealed/module-receipt"),
        (TerminalSpec, "/sealed/terminal-spec.json"),
    ] {
        result.paths.insert(key, value.into());
    }
    for profile in [
        "fixture-controlled",
        "fixture-cooperative",
        "sqlite-cooperative",
        "onebox-cooperative",
        "multi-instance-cooperative",
        "cooperative-runtime",
    ] {
        result
            .installations
            .insert(profile.into(), format!("/sealed/installations/{profile}"));
        if profile != "cooperative-runtime" {
            for cell in ["success", "canary"] {
                let label = format!("workload-{profile}-{cell}");
                result
                    .specifications
                    .insert(label.clone(), format!("/sealed/specs/{label}.json"));
            }
        }
    }
    result.lint_base = "a".repeat(40);
    result
}

fn expected_argv(
    old: &Value,
    original: &LeafBindings,
    current: &LeafBindings,
    inventory: &Value,
) -> Vec<String> {
    let mut argv: Vec<String> = old
        .as_array()
        .unwrap()
        .iter()
        .map(|item| {
            let raw = item.as_str().unwrap();
            for (key, old_path) in &original.paths {
                if raw == old_path {
                    return current.paths[key].clone();
                }
            }
            for (before, after) in [
                (&original.installations, &current.installations),
                (&original.specifications, &current.specifications),
            ] {
                for (key, path) in before {
                    if raw == path {
                        return after[key].clone();
                    }
                }
            }
            for key in [Binding::Repo, Binding::Root] {
                if let Some(tail) = raw.strip_prefix(&(original.paths[&key].clone() + "/")) {
                    return format!("{}/{tail}", current.paths[&key]);
                }
            }
            for (prefix, key) in [
                ("GOLANGCI_LINT=", Binding::Golangci),
                ("ERRORTYPE=", Binding::Errortype),
            ] {
                if raw == format!("{prefix}{}", original.paths[&key]) {
                    return format!("{prefix}{}", current.paths[&key]);
                }
            }
            raw.into()
        })
        .collect();
    for endpoint in inventory["endpoint_substitutions"].as_array().unwrap() {
        if argv.get(1)
            == Some(&format!(
                "{}/{}",
                current.paths[&Binding::Repo],
                endpoint["old_endpoint"].as_str().unwrap()
            ))
        {
            assert_eq!(argv[0], current.paths[&Binding::Python]);
            let mut mapped = vec![current.paths[&Binding::Binary].clone()];
            mapped.extend(
                endpoint["proposed_rust_endpoint"].as_array().unwrap()[1..]
                    .iter()
                    .map(|v| v.as_str().unwrap().to_owned()),
            );
            mapped.extend(argv.into_iter().skip(2));
            argv = mapped;
            break;
        }
    }
    argv
}

#[test]
fn all_235_fixed_leaf_recipes_match_independent_golden() {
    let oracle = super::tests::oracle();
    let inventory_bytes = include_bytes!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../../../.flow/artifacts/fn242-obligation-inventory.json"
    ));
    assert_eq!(
        crate::evidence::canonical::digest(inventory_bytes),
        "21b500b6272f5b303c22e921225235ef6917d87f141a41d1429641f9d81bd4e3"
    );
    let inventory: Value = crate::qualify::spec::strict(inventory_bytes).unwrap();

    let original = bindings();
    let mut rebound = original.clone();
    for (key, path) in &mut rebound.paths {
        if matches!(key, Binding::Repo | Binding::Root) {
            *path = format!("/candidate/{key:?}");
        }
    }
    rebound
        .paths
        .insert(Binding::Go, "/candidate/toolchain/go".into());
    rebound.installations.insert(
        "fixture-controlled".into(),
        "/candidate/custom.installation".into(),
    );
    rebound.specifications.insert(
        "workload-fixture-controlled-success".into(),
        "/candidate/custom-spec".into(),
    );
    for current in [&original, &rebound] {
        for mode in [Mode::Parallel, Mode::Serial] {
            let plan = derive(&super::tests::config(), mode).unwrap();
            let mut count = 0;
            let mut dependent = 0;
            for (stage, expected) in plan.stages.iter().zip(oracle["stages"].as_array().unwrap()) {
                let leaves = stage.leaf_commands(current).unwrap();
                let recipes = expected["leaf_recipes"].as_array().unwrap();
                assert_eq!(leaves.expansion, expansion(&stage.contract.label));
                assert_eq!(
                    leaves.commands.len(),
                    recipes.len(),
                    "{}",
                    stage.contract.label
                );
                if recipes.is_empty() {
                    assert!(leaves.expansion.is_some());
                    dependent += 1;
                }
                for (actual, recipe) in leaves.commands.iter().zip(recipes) {
                    assert_eq!(
                        actual.argv,
                        expected_argv(&recipe["argv"], &original, current, &inventory),
                        "{}",
                        stage.contract.label
                    );
                    check_metadata(
                        actual,
                        recipe,
                        &original,
                        current,
                        &inventory,
                        &stage.contract.label,
                    );
                    count += 1;
                }
                stage.validate_leaf_commands(current, &leaves).unwrap();
            }
            assert_eq!((count, dependent), (235, 8));
            println!(
                "leaf_recipes mode={mode:?} fixed=235 fixed_rows=88 result_dependent=8 runtime_expansions=32"
            );
        }
    }
}

#[test]
fn changed_leaf_argument_is_refused_with_outer_wrapper_unchanged() {
    let stage = derive(&super::tests::config(), Mode::Parallel)
        .unwrap()
        .stages
        .remove(1);
    let bindings = bindings();
    let wrapper = stage.argv.clone();
    let mut leaves = stage.leaf_commands(&bindings).unwrap();
    stage.validate_leaf_commands(&bindings, &leaves).unwrap();
    let argument = leaves.commands[0]
        .argv
        .iter_mut()
        .find(|arg| *arg == "-timeout=20m")
        .unwrap();
    *argument = "-timeout=19m".into();
    assert_eq!(stage.argv, wrapper);
    assert_eq!(
        stage
            .validate_leaf_commands(&bindings, &leaves)
            .unwrap_err(),
        "changed leaf command contract"
    );
    println!(
        "leaf_argument control=accepted mutation=-timeout=20m-to-19m outcome=refused outer_wrapper=unchanged"
    );
}

fn expansion(label: &str) -> Option<Expansion> {
    match label {
        "runtime-aggregate" => Some(Expansion::RuntimeAggregate { shards: 32 }),
        "workload-corpus" => Some(Expansion::Corpus {
            after_relocation: false,
        }),
        "workload-corpus-after-relocation" => Some(Expansion::Corpus {
            after_relocation: true,
        }),
        "sqlite-selection-0" => Some(Expansion::SqliteSelection { index: 0 }),
        "sqlite-selection-1" => Some(Expansion::SqliteSelection { index: 1 }),
        "sqlite-selection-2" => Some(Expansion::SqliteSelection { index: 2 }),
        "sqlite-selection-3" => Some(Expansion::SqliteSelection { index: 3 }),
        "sqlite-catalogue" => Some(Expansion::SqliteCatalogue),
        runtime if runtime.starts_with("runtime-") => Some(Expansion::RuntimeShard {
            index: runtime[8..].parse().unwrap(),
            instances: 25,
            replays: 1,
            wall_seconds: 600,
        }),
        _ => None,
    }
}

fn check_metadata(
    actual: &super::leaf::LeafCommand,
    recipe: &Value,
    original: &LeafBindings,
    current: &LeafBindings,
    inventory: &Value,
    label: &str,
) {
    let value = serde_json::to_value(actual).unwrap();
    for (key, expected) in recipe.as_object().unwrap() {
        if key == "argv" {
            continue;
        }
        let expected = if key == "cwd" {
            Value::String(
                expected_argv(
                    &serde_json::json!([expected]),
                    &original,
                    current,
                    &inventory,
                )
                .remove(0),
            )
        } else if key == "environment" {
            let mut env = expected.as_object().unwrap().clone();
            for value in env.values_mut() {
                *value = Value::String(
                    expected_argv(&serde_json::json!([value]), &original, current, &inventory)
                        .remove(0),
                );
            }
            Value::Object(env)
        } else {
            expected.clone()
        };
        let flag = match key.as_str() {
            "reference" => Some(LeafFlag::Reference),
            "all_platforms" => Some(LeafFlag::AllPlatforms),
            "lint_policy" => Some(LeafFlag::LintPolicy),
            "regenerate" => Some(LeafFlag::Regenerate),
            "private_observations" => Some(LeafFlag::PrivateObservations),
            _ => None,
        };
        let actual_value = flag.map_or_else(
            || value[key].clone(),
            |flag| Value::Bool(actual.flags.contains(&flag)),
        );
        assert_eq!(actual_value, expected, "{} {key}", label);
    }
}
