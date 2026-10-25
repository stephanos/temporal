use super::static_inputs::{
    Tool, ignored_attributes, ignored_inventory, plan_config, prepare_plan, recipe, static_plan,
    tree_hashes, workflow_pair,
};
use super::*;
use std::{
    fs,
    sync::atomic::{AtomicU64, Ordering},
};
static NEXT: AtomicU64 = AtomicU64::new(0);
pub(crate) struct Fixture {
    pub(crate) root: PathBuf,
    pub(crate) inputs: CompositionInputs,
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}
impl Fixture {
    pub(crate) fn new() -> Self {
        let root = PathBuf::from(format!(
            "/home/agent/wt/evidence/composition-fixture-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir_all(&root).unwrap();
        let repo = root.join("repo");
        fs::create_dir_all(repo.join(".github/workflows")).unwrap();
        for name in ["gomad.yml", "gomad-behavioral-parity.yml"] {
            fs::write(
                repo.join(".github/workflows").join(name),
                "paths: [tools/gomad/**]\n",
            )
            .unwrap();
        }
        let rust = repo.join("tools/gomad/rust");
        fs::create_dir_all(rust.join("driver")).unwrap();
        fs::create_dir_all(rust.join("gomad/src")).unwrap();
        fs::write(
            rust.join("gomad/src/lib.rs"),
            "#[test]\n#[ignore = \"operator fixture\"]\nfn controlled_probe() {}\n",
        )
        .unwrap();
        fs::write(rust.join("driver/static-ignored.json"),serde_json::to_vec(&json!({"schema":"gomad.driver-static-ignored/v1","tests":[{"crate":"gomad","source":"gomad/src/lib.rs","name":"controlled_probe","reason":"operator fixture","final_id":"actual/current"}]})).unwrap()).unwrap();
        fs::write(
            rust.join("driver/required-final.json"),
            b"{\"subgates\":[\"actual/current\"]}",
        )
        .unwrap();
        let tools = root.join("tools");
        fs::create_dir(&tools).unwrap();
        let mut pinned = BTreeMap::new();
        for (key, file) in [
            ("python", "python"),
            ("go", "go"),
            ("cargo", "cargo"),
            ("rustc", "rustc"),
            ("rustfmt", "rustfmt"),
            ("make", "make"),
            ("golangci", "golangci"),
            ("errortype", "errortype"),
            ("true", "true"),
            ("rustdoc", "rustdoc"),
            ("cargo_clippy", "cargo-clippy"),
            ("clippy_driver", "clippy-driver"),
            ("cargo_fmt", "cargo-fmt"),
        ] {
            let p = tools.join(file);
            fs::copy("/bin/true", &p).unwrap();
            let sha256 = custody::digest(&fs::read(&p).unwrap());
            pinned.insert(key.into(), Tool { path: p, sha256 });
        }
        let mut caches = BTreeMap::new();
        for key in ["cargo_target", "cargo_home", "go_cache", "go_mod_cache"] {
            let p = root.join(key);
            fs::create_dir(&p).unwrap();
            caches.insert(key.into(), p);
        }
        let mut b = LeafBindings::default();
        for key in [
            Binding::Repo,
            Binding::Root,
            Binding::Config,
            Binding::Binary,
            Binding::Python,
            Binding::Go,
            Binding::Cargo,
            Binding::Rustc,
            Binding::Clippy,
            Binding::Make,
            Binding::Golangci,
            Binding::Errortype,
            Binding::True,
            Binding::CargoHome,
            Binding::RuntimeHarness,
            Binding::Bootstrap,
            Binding::Compiler,
            Binding::RuntimeRoot,
            Binding::Cache,
            Binding::CapturedConfig,
            Binding::SourceManifest,
            Binding::BuildReceipt,
            Binding::CompilerRoot,
            Binding::GuestConfig,
            Binding::ModuleCache,
            Binding::ClusterTemplate,
            Binding::BrokerObservation,
            Binding::HostFixture,
            Binding::HostTests,
            Binding::HelperTests,
            Binding::StockGo,
            Binding::FreshOrdinary,
            Binding::FreshAdmission,
            Binding::FreshBuild,
            Binding::Diagnostic,
            Binding::DiagnosticAdmission,
            Binding::Module,
            Binding::ModuleReceipt,
            Binding::TerminalSpec,
        ] {
            let p = root.join(format!("{key:?}"));
            fs::write(&p, b"controlled-current-input\n").unwrap();
            b.paths.insert(key, p.to_str().unwrap().into());
        }
        b.paths.insert(Binding::Repo, repo.to_str().unwrap().into());
        b.paths
            .insert(Binding::Root, root.join("cycle").to_str().unwrap().into());
        b.paths.insert(
            Binding::Config,
            root.join("config.json").to_str().unwrap().into(),
        );
        for (key, binding) in [
            ("python", Binding::Python),
            ("go", Binding::Go),
            ("cargo", Binding::Cargo),
            ("rustc", Binding::Rustc),
            ("cargo_clippy", Binding::Clippy),
            ("make", Binding::Make),
            ("golangci", Binding::Golangci),
            ("errortype", Binding::Errortype),
            ("true", Binding::True),
        ] {
            b.paths
                .insert(binding, pinned[key].path.to_str().unwrap().into());
        }
        b.paths.insert(
            Binding::CargoHome,
            caches["cargo_home"].to_str().unwrap().into(),
        );
        b.lint_base = "a".repeat(40);
        for profile in [
            "fixture-controlled",
            "fixture-cooperative",
            "sqlite-cooperative",
            "onebox-cooperative",
            "multi-instance-cooperative",
            "cooperative-runtime",
        ] {
            let p = root.join(profile);
            fs::write(&p, serde_json::to_vec(&json!({"profile":profile})).unwrap()).unwrap();
            b.installations
                .insert(profile.into(), p.to_str().unwrap().into());
            if profile != "cooperative-runtime" {
                for cell in ["success", "canary"] {
                    let label = format!("workload-{profile}-{cell}");
                    let p = root.join(format!("{label}.json"));
                    fs::write(&p, serde_json::to_vec(&json!({"cell":label})).unwrap()).unwrap();
                    b.specifications.insert(label, p.to_str().unwrap().into());
                }
            }
        }
        let static_inputs = StaticInputs {
            repo,
            driver_config: root.join("config.json"),
            tools: pinned,
            caches,
            lint_base: b.lint_base.clone(),
        };
        let plan = plan::derive(
            &PlanConfig {
                binary: b.paths[&Binding::Binary].clone().into(),
                config_path: static_inputs.driver_config.clone(),
                config_sha256: "0".repeat(64),
            },
            Mode::Parallel,
        )
        .unwrap();
        let stages: BTreeMap<_, _> = plan
            .stages
            .iter()
            .map(|s| (s.contract.label.clone(), json!({"adapter":"pending"})))
            .collect();
        let base = json!({"schema":"gomad.rust-qualification-driver/v1","stages":stages,"sent_epoch":1,"deadline_s":3600,"dry_run":true,"identity":{"host":"a".repeat(64),"helper":"b".repeat(64)}});
        let runtime_build_receipt = root.join("runtime-build.json");
        fs::write(
            &runtime_build_receipt,
            b"{\"controlled\":\"runtime-receipt\"}",
        )
        .unwrap();
        let sqlite_build_receipt = root.join("sqlite-build.json");
        fs::write(
            &sqlite_build_receipt,
            b"{\"controlled\":\"sqlite-receipt\"}",
        )
        .unwrap();
        Self {
            root,
            inputs: CompositionInputs {
                schema: SCHEMA.into(),
                leaves: b,
                base,
                runtime_build_receipt,
                sqlite_build_receipt,
                static_inputs,
            },
        }
    }
}
fn refuses(result: Result<(), String>, needle: &str) {
    let error = result.unwrap_err();
    assert!(
        error.contains(needle),
        "{error:?} does not contain {needle:?}"
    );
    println!("catcher={needle:?} refused={error:?}");
}
#[test]
fn current_composition_reopens_all_stages_and_static_recipes() {
    let f = Fixture::new();
    let bound = compose(f.inputs.clone()).unwrap();
    bound.validate().unwrap();
    proof(
        "all 36 original guard normal-return control",
        &[
            "orchestration:compose:46",
            "orchestration:compose:51",
            "orchestration:compose:52",
            "orchestration:compose:62",
            "orchestration:compose:72",
            "orchestration:compose:74",
            "orchestration:compose:75",
            "orchestration:compose:79",
            "orchestration:compose:81",
            "orchestration:compose:82",
            "orchestration:compose:85",
            "orchestration:compose:87",
            "orchestration:compose:88",
            "orchestration:compose:90",
            "orchestration:compose:94",
            "orchestration:compose:96",
            "orchestration:compose:97",
            "orchestration:compose:102",
            "orchestration:compose:110",
            "orchestration:compose:125",
            "orchestration:compose:128",
            "orchestration:compose:137",
            "orchestration:compose:147",
            "orchestration:compose:148",
            "orchestration:compose:151",
            "orchestration:compose:152",
            "orchestration:compose:153",
            "orchestration:compose:157",
            "orchestration:compose:174",
            "orchestration:compose:178",
            "orchestration:compose:183",
            "orchestration:compose:190",
            "orchestration:compose:192",
            "orchestration:static_stages:184",
            "orchestration:static_stages:202",
            "orchestration:static_stages:272",
        ],
        &bound,
        &Ok(()),
    );
    let output = f.root.join("composition.json");
    assert_eq!(compose_to(f.inputs.clone(), &output).unwrap(), bound);
    assert_eq!(
        StageBindings::decode(&fs::read(&output).unwrap()).unwrap(),
        bound
    );
    let result = compose_to(f.inputs.clone(), &output).map(|_| ());
    proof(
        "fresh composition refuses existing output",
        &["orchestration:compose:157"],
        &json!({"inputs":f.inputs,"output":output,"existing_sha256":custody::digest(&fs::read(&output).unwrap())}),
        &result,
    );
    refuses(result, "must be fresh");
    assert_eq!(bound.stages.len(), 96);
    assert_eq!(bound.static_plan.stages.len(), 24);
    let bytes = serde_json::to_vec(&bound).unwrap();
    assert_eq!(StageBindings::decode(&bytes).unwrap(), bound);
    println!(
        "valid-control bytes_sha256={} bytes={}",
        custody::digest(&bytes),
        String::from_utf8(bytes).unwrap()
    );
    for stage in &bound.plan.stages {
        assert_eq!(
            bound.stages[&stage.contract.label],
            stage.leaf_commands(&bound.inputs.leaves).unwrap()
        );
    }
    assert_eq!(bound.final_sections.len(), 5);
    assert_eq!(bound.config["stages"]["runtime-0"]["adapter"], "runtime");
    assert_eq!(
        bound.config["stages"]["workload-sqlite-cooperative-canary"]["private_observations"],
        true
    );
    assert_eq!(
        bound.config["stages"]["validate"]["stop_at_deadline"],
        false
    );
    assert_eq!(bound.config["stages"]["validate"]["timeout_s"], 3600);
    assert!(
        !bound.config["dry_run_unexecuted"]
            .as_array()
            .unwrap()
            .is_empty()
    );
}
#[test]
fn consumed_role_executables_are_sealed_and_reopened() {
    let f = Fixture::new();
    let bound = compose(f.inputs.clone()).unwrap();
    for binding in [
        Binding::HostTests,
        Binding::HelperTests,
        Binding::HostFixture,
    ] {
        let path = Path::new(&f.inputs.leaves.paths[&binding]);
        let original = fs::read(path).unwrap();
        assert_eq!(
            bound.files[path.to_str().unwrap()],
            custody::digest(&original)
        );
        let mut changed = original.clone();
        changed.push(0);
        fs::write(path, changed).unwrap();
        assert!(bound.validate().unwrap_err().contains("changed composed"));
        fs::write(path, &original).unwrap();
        let parked = path.with_extension("parked");
        fs::rename(path, &parked).unwrap();
        let missing = bound.validate().unwrap_err();
        fs::rename(&parked, path).unwrap();
        assert!(missing.contains("custody file:"), "{missing}");
    }
    bound.validate().unwrap();
}
#[test]
fn composition_refuses_missing_identity_labels_bounds_and_bindings() {
    let f = Fixture::new();
    type Mutate = fn(&mut CompositionInputs);
    let cases: &[(&str, Mutate, &str)] = &[
        ("schema", |v| v.schema = "old".into(), "schema"),
        (
            "labels missing",
            |v| {
                v.base["stages"].as_object_mut().unwrap().remove("validate");
            },
            "stage labels",
        ),
        (
            "labels foreign",
            |v| {
                v.base["stages"]["foreign"] = json!({});
            },
            "stage labels",
        ),
        (
            "identity absent",
            |v| v.base["identity"] = Value::Null,
            "identities",
        ),
        (
            "identity zero",
            |v| v.base["identity"]["host"] = json!("0".repeat(64)),
            "identities",
        ),
        (
            "identity foreign",
            |v| v.base["identity"]["other"] = json!("a".repeat(64)),
            "identities",
        ),
        ("deadline", |v| v.base["deadline_s"] = json!(3599), "3600"),
        ("epoch", |v| v.base["sent_epoch"] = json!(0), "epoch"),
        ("stress", |v| v.base["stress"] = json!({}), "post-freeze"),
        (
            "driver config",
            |v| v.static_inputs.driver_config = "/wrong/config".into(),
            "differs",
        ),
        (
            "mandatory installation",
            |v| {
                v.leaves.installations.remove("fixture-controlled");
            },
            "leaf input",
        ),
        (
            "mandatory spec",
            |v| {
                v.leaves
                    .specifications
                    .remove("workload-fixture-controlled-success");
            },
            "leaf input",
        ),
        (
            "live foreign",
            |v| v.base["live_corpus"] = json!({"families":{"foreign":{"stage":"ordinary"}}}),
            "receiver families",
        ),
        (
            "live stage",
            |v| v.base["live_corpus"] = json!({"families":{"choice":{"stage":"unknown"}}}),
            "receiver stage",
        ),
        (
            "full live required",
            |v| v.base["dry_run"] = json!(false),
            "receiver families",
        ),
    ];
    compose(f.inputs.clone()).unwrap();
    for (name, mutate, needle) in cases {
        let mut input = f.inputs.clone();
        mutate(&mut input);
        let ids: &[&str] = match *name {
            "schema" => &["orchestration:compose:148"],
            "labels missing" | "labels foreign" => &[
                "orchestration:compose:51",
                "orchestration:compose:152",
                "orchestration:static_stages:184",
            ],
            "identity absent" | "identity zero" | "identity foreign" => {
                &["orchestration:compose:52"]
            }
            "deadline" => &["orchestration:compose:82"],
            "epoch" => &["orchestration:compose:151"],
            "stress" => &["orchestration:compose:190"],
            "driver config" => &["orchestration:compose:153"],
            "mandatory installation" => &["orchestration:compose:110"],
            "mandatory spec" => &["orchestration:compose:87"],
            "live foreign" | "full live required" => &["orchestration:compose:174"],
            "live stage" => &["orchestration:compose:178"],
            _ => &[],
        };
        let result = compose(input.clone()).map(|_| ());
        proof(name, ids, &input, &result);
        refuses(result, needle);
    }
}
#[test]
fn composition_directly_refuses_changed_adapter_partition_recipe_and_expansion() {
    let f = Fixture::new();
    let bound = compose(f.inputs.clone()).unwrap();
    for (name, mutate) in [
        (
            "adapter",
            (|b: &mut StageBindings| b.config["stages"]["validate"]["adapter"] = json!("runtime"))
                as fn(&mut StageBindings),
        ),
        ("partition", |b: &mut StageBindings| {
            b.final_sections
                .get_mut("20-profiles")
                .unwrap()
                .required
                .push("foreign".into());
        }),
        ("static argv", |b: &mut StageBindings| {
            b.static_plan.stages.get_mut("validate").unwrap().commands[0]
                .argv
                .push("foreign".into());
        }),
        ("stage argv", |b: &mut StageBindings| {
            b.stages.get_mut("runtime-0").unwrap().commands[0]
                .argv
                .push("foreign".into());
        }),
        ("expansion", |b: &mut StageBindings| {
            b.stages.get_mut("sqlite-catalogue").unwrap().expansion = None;
        }),
        ("deleted stage", |b: &mut StageBindings| {
            b.stages.remove("ordinary");
        }),
    ] {
        let mut altered = bound.clone();
        mutate(&mut altered);
        println!("mutation={name}");
        refuses(altered.validate(), "changed composed");
    }
    let mut leaf = bound.stages["validate"].clone();
    leaf.commands[0].argv = vec![
        "/python".into(),
        "/repo/tools/gomad/rust/driver/deleted.py".into(),
    ];
    let result = validate_endpoints(&f.inputs.leaves, &leaf);
    proof(
        "deleted or unknown managed endpoint",
        &["orchestration:compose:81"],
        &leaf,
        &result,
    );
    refuses(result, "unmapped");
    leaf.commands[0].argv = vec![
        f.inputs.leaves.paths[&Binding::Binary].clone(),
        "qualification-cycle".into(),
        "internal".into(),
        "unknown".into(),
    ];
    let result = validate_endpoints(&f.inputs.leaves, &leaf);
    proof(
        "deleted or unknown managed endpoint",
        &["orchestration:compose:81"],
        &leaf,
        &result,
    );
    refuses(result, "unmapped");
    leaf.commands[0].argv[0] = "/foreign".into();
    refuses(
        validate_endpoints(&f.inputs.leaves, &leaf),
        "foreign orchestration",
    );
}
#[test]
fn static_inputs_authenticate_tool_cache_and_immutable_plan_reopening() {
    let f = Fixture::new();
    let b = &f.inputs.leaves;
    let inputs = &f.inputs.static_inputs;
    let output = f.root.join("static-plan.json");
    let plan = prepare_plan(inputs, b, &output).unwrap();
    assert_eq!(prepare_plan(inputs, b, &output).unwrap(), plan);
    let mut files = plan.files.clone();
    files.insert(
        output.to_str().unwrap().into(),
        custody::digest(&fs::read(&output).unwrap()),
    );
    fs::write(
        &inputs.driver_config,
        serde_json::to_vec(&json!({"repo":inputs.repo,"files":files})).unwrap(),
    )
    .unwrap();
    assert_eq!(
        plan_config(&output, "validate", b, &inputs.driver_config)
            .unwrap()
            .0,
        plan
    );
    let result = recipe(&plan, "runtime-0", b).map(|_| ());
    proof(
        "foreign static recipe label",
        &[
            "orchestration:compose:75",
            "orchestration:static_stages:202",
            "orchestration:static_stages:272",
        ],
        &json!({"plan":plan,"requested_label":"runtime-0"}),
        &result,
    );
    refuses(result, "static stages");
    let mut changed = plan.clone();
    changed.stages.get_mut("validate").unwrap().commands[0]
        .argv
        .push("foreign".into());
    let result = recipe(&changed, "validate", b).map(|_| ());
    proof(
        "standalone recipe refuses changed stored argv",
        &[
            "orchestration:static_stages:202",
            "orchestration:static_stages:272",
        ],
        &changed,
        &result,
    );
    refuses(result, "changed immutable static recipe");
    changed = plan.clone();
    changed
        .stages
        .insert("runtime-0".into(), changed.stages["validate"].clone());
    let result = recipe(&changed, "runtime-0", b).map(|_| ());
    proof(
        "standalone recipe refuses injected nonstatic label",
        &["orchestration:static_stages:202"],
        &changed,
        &result,
    );
    refuses(result, "static stages");
    let mut wrong = inputs.clone();
    wrong.tools.remove("rustdoc");
    refuses(static_plan(&wrong, b).map(|_| ()), "tools");
    wrong = inputs.clone();
    wrong.tools.get_mut("go").unwrap().sha256 = "0".repeat(64);
    refuses(static_plan(&wrong, b).map(|_| ()), "changed tool");
    wrong = inputs.clone();
    wrong.caches.insert("go_cache".into(), inputs.repo.clone());
    refuses(static_plan(&wrong, b).map(|_| ()), "outside frozen");
    wrong = inputs.clone();
    wrong.lint_base = "main".into();
    refuses(static_plan(&wrong, b).map(|_| ()), "exact bound commit");
    fs::write(&output, b"{}").unwrap();
    let result = prepare_plan(inputs, b, &output).map(|_| ());
    proof(
        "changed provisioned immutable static plan",
        &["orchestration:compose:137"],
        &json!({"inputs":inputs,"leaves":b,"path":output,"published_bytes":"{}"}),
        &result,
    );
    refuses(result, "provisioned static plan");
    let bound = compose(f.inputs.clone()).unwrap();
    fs::write(&f.inputs.runtime_build_receipt, b"changed runtime receipt").unwrap();
    refuses(bound.validate(), "changed composed");
}
#[test]
fn tokenizer_distinguishes_real_attributes_from_nested_comments_and_literal_fixtures() {
    let source = r####"// #[ignore] fn comment() {}
/* outer /* #[ignore] */ */
const FIXTURE:&str=r###"#[ignore] fn raw() {}"###;
const BYTES:&[u8]=br##"#[ignore] fn bytes() {}"##;
fn outer<'a>(s:&'a str) { let c='\''; }
mod nested { #[test] #[ignore = "explicit operator reason"] pub async fn actual() {} }
"####;
    assert_eq!(
        ignored_attributes(source).unwrap(),
        vec![("actual".into(), "explicit operator reason".into())]
    );
    for text in [
        "#[ignore] fn missing() {}",
        "#[ignore = false] fn wrong() {}",
        "#[ignore = \"why\"] struct Wrong;",
        "/* unterminated",
    ] {
        refuses(
            ignored_attributes(text).map(|_| ()),
            if text.starts_with("/*") {
                "token stream"
            } else {
                "ignored"
            },
        );
    }
    let f = Fixture::new();
    ignored_inventory(&f.inputs.static_inputs.repo).unwrap();
    fs::write(
        f.inputs
            .static_inputs
            .repo
            .join("tools/gomad/rust/gomad/src/lib.rs"),
        "#[ignore = \"unregistered\"] fn controlled_probe() {}\n",
    )
    .unwrap();
    refuses(
        ignored_inventory(&f.inputs.static_inputs.repo).map(|_| ()),
        "disposition",
    );
}
#[test]
fn current_checkout_ignored_and_workflow_inventory_is_actual_source() {
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap();
    let rows = ignored_inventory(repo).unwrap();
    assert!(!rows["tests"].as_array().unwrap().is_empty());
    assert_eq!(workflow_pair(repo).unwrap().len(), 2);
}
#[test]
fn workflows_and_tree_hashes_reject_unreviewed_inventory_and_linked_sources() {
    let f = Fixture::new();
    let repo = &f.inputs.static_inputs.repo;
    let before = tree_hashes(&repo.join(".github/workflows")).unwrap();
    assert_eq!(before.len(), 2);
    fs::write(
        repo.join(".github/workflows/foreign.yml"),
        "paths: [tools/gomad/**]",
    )
    .unwrap();
    refuses(workflow_pair(repo).map(|_| ()), "ambiguous");
    let linked = f.root.join("linked");
    std::os::unix::fs::symlink(&repo, &linked).unwrap();
    refuses(tree_hashes(&linked).map(|_| ()), "canonical");
}

fn proof(case: &str, ids: &[&str], input: &impl serde::Serialize, result: &Result<(), String>) {
    let bytes = serde_json::to_vec(input).unwrap();
    let sources = json!({
        "composition/mod.rs":custody::digest(include_bytes!("mod.rs")),
        "composition/static_inputs.rs":custody::digest(include_bytes!("static_inputs.rs")),
        "composition/tests.rs":custody::digest(include_bytes!("tests.rs")),
        "plan/leaf/mod.rs":custody::digest(include_bytes!("../../plan/leaf/mod.rs")),
    });
    println!(
        "B03_PROOF {}",
        json!({"case":case,"original_ids":ids,"original_source_sha256":{"compose.py":"25c95841212b7ad61a2cd8ddb2ebd515472f9c226ba519b0990e83a30a4b475f","static_stages.py":"3bc7379bbbc4c8d4eb0a6b0ab4437bee111533e0f3e9d7d8ebd59ffcb7ba8f2d"},"current_sources":sources,"input_sha256":custody::digest(&bytes),"input_bytes":String::from_utf8(bytes).unwrap(),"returned":if result.is_ok(){json!("ok")}else{json!(result.as_ref().unwrap_err())}})
    );
}
fn full_inputs(f: &Fixture) -> CompositionInputs {
    let mut inputs = f.inputs.clone();
    inputs.base["dry_run"] = json!(false);
    inputs.base["live_corpus"] = json!({"families":LIVE_FAMILIES.iter().map(|name|((*name).to_owned(),json!({"stage":"ordinary"}))).collect::<BTreeMap<_,_>>()});
    inputs.base["final_profiles"] = json!({"runtime_recipe":{"receipt":inputs.runtime_build_receipt},"qualified":{"installation":inputs.leaves.installations["fixture-controlled"]},"runtime_reference":{"receipt":inputs.runtime_build_receipt}});
    for key in [
        "final_retained",
        "public_verbs",
        "private_observations",
        "fn222",
        "terminal_controls_reference",
    ] {
        inputs.base[key] = json!({"receipt":inputs.runtime_build_receipt});
    }
    inputs
}
#[test]
fn final_inputs_refuse_missing_or_false_group() {
    let f = Fixture::new();
    let inputs = full_inputs(&f);
    let control = compose(inputs.clone()).map(|_| ());
    proof(
        "complete final-group composition control",
        &["orchestration:compose:174", "orchestration:compose:192"],
        &inputs,
        &control,
    );
    control.unwrap();
    for (key, value) in [
        ("final_retained", Value::Null),
        ("public_verbs", json!({})),
        ("private_observations", json!(false)),
        ("final_profiles", json!({"qualified":true})),
    ] {
        let mut altered = inputs.clone();
        altered.base[key] = value;
        let result = compose(altered.clone()).map(|_| ());
        proof(
            &format!("missing/false/incomplete final group {key}"),
            &["orchestration:compose:192"],
            &altered,
            &result,
        );
        refuses(result, "missing actual final input binding");
    }
}
#[test]
fn composition_proof_rows_directly_catch_each_current_descriptor_predicate() {
    let f = Fixture::new();
    let control = compose(f.inputs.clone()).unwrap();
    proof(
        "current descriptor normal-return control",
        &[
            "orchestration:compose:46",
            "orchestration:compose:62",
            "orchestration:compose:72",
            "orchestration:compose:74",
            "orchestration:compose:82",
            "orchestration:compose:88",
            "orchestration:compose:90",
            "orchestration:compose:94",
            "orchestration:compose:96",
            "orchestration:compose:97",
        ],
        &control,
        &Ok(()),
    );
    type Mutate = fn(&mut StageBindings);
    let cases: &[(&str, &[&str], Mutate)] = &[
        ("fixed primary fields", &["orchestration:compose:62"], |b| {
            b.config["stages"]["ordinary"]["unexpected"] = json!(true)
        }),
        (
            "wrong semantic adapter",
            &["orchestration:compose:72"],
            |b| b.config["stages"]["runtime-0"]["adapter"] = json!("primary"),
        ),
        (
            "missing actual static action",
            &["orchestration:compose:74"],
            |b| {
                b.static_plan
                    .stages
                    .get_mut("validate")
                    .unwrap()
                    .commands
                    .remove(0);
            },
        ),
        ("fixed static timeout", &["orchestration:compose:82"], |b| {
            b.config["stages"]["validate"]["timeout_s"] = json!(3601)
        }),
        (
            "stop_at_deadline timing feedback",
            &["orchestration:compose:82"],
            |b| b.config["stages"]["validate"]["stop_at_deadline"] = json!(true),
        ),
        (
            "private observation required",
            &["orchestration:compose:88"],
            |b| {
                b.config["stages"]["workload-sqlite-cooperative-success"]["private_observations"] =
                    json!(false)
            },
        ),
        (
            "durable cell semantic reader",
            &["orchestration:compose:90"],
            |b| {
                b.config["stages"]["workload-sqlite-cooperative-success"]["checker"] =
                    json!(["/foreign/cell-reader"])
            },
        ),
        (
            "fixed catalogue fields",
            &["orchestration:compose:94"],
            |b| b.config["stages"]["sqlite-catalogue"]["oracle"] = json!(["/foreign/oracle"]),
        ),
        (
            "replaced retained adapter",
            &["orchestration:compose:96"],
            |b| b.config["stages"]["terminal-module"]["adapter"] = json!("plan"),
        ),
        ("unbound stage", &["orchestration:compose:97"], |b| {
            b.config["stages"]["ordinary"]["adapter"] = json!("pending")
        }),
        (
            "wrong final obligation partition",
            &["orchestration:compose:46"],
            |b| {
                b.final_sections.get_mut("50-operator").unwrap().required =
                    vec!["retained/20-fresh-observed-replays".into()];
            },
        ),
    ];
    for (name, ids, mutate) in cases {
        let mut altered = control.clone();
        mutate(&mut altered);
        let result = altered.validate();
        proof(name, ids, &altered, &result);
        refuses(result, "changed composed");
    }
}
#[test]
fn composition_proof_rows_catch_strict_input_identity_custody_and_environment() {
    let f = Fixture::new();
    let control = compose(f.inputs.clone()).map(|_| ());
    proof(
        "strict input normal-return control",
        &[
            "orchestration:compose:52",
            "orchestration:compose:102",
            "orchestration:compose:125",
            "orchestration:compose:128",
            "orchestration:compose:147",
            "orchestration:compose:153",
        ],
        &f.inputs,
        &control,
    );
    control.unwrap();
    type Mutate = fn(&mut CompositionInputs);
    let cases: &[(&str, &[&str], Mutate, &str)] = &[
        (
            "malformed nonzero identity",
            &["orchestration:compose:52"],
            |i| i.base["identity"]["host"] = json!("g".repeat(64)),
            "identities",
        ),
        (
            "missing helper identity",
            &["orchestration:compose:52"],
            |i| {
                i.base["identity"].as_object_mut().unwrap().remove("helper");
            },
            "identities",
        ),
        (
            "relative SQLite build receipt",
            &["orchestration:compose:125"],
            |i| i.sqlite_build_receipt = "relative/receipt".into(),
            "absolute",
        ),
        (
            "caller oracle argv",
            &["orchestration:compose:128"],
            |i| i.base["stages"]["sqlite-catalogue"]["oracle"] = json!(["/foreign/oracle"]),
            "caller SQLite oracle",
        ),
        (
            "base/leaf installation mismatch",
            &["orchestration:compose:110"],
            |i| i.base["installations"] = json!({}),
            "installation bindings differ",
        ),
        (
            "base/leaf environment mismatch",
            &["orchestration:compose:153"],
            |i| i.base["environment"] = json!({"GOCACHE":"/foreign"}),
            "environment bindings differ",
        ),
        (
            "base/leaf reference environment mismatch",
            &["orchestration:compose:153"],
            |i| i.base["reference_environment"] = json!({"HOME":"/foreign"}),
            "environment bindings differ",
        ),
    ];
    for (name, ids, mutate, needle) in cases {
        let mut altered = f.inputs.clone();
        mutate(&mut altered);
        let result = compose(altered.clone()).map(|_| ());
        proof(name, ids, &altered, &result);
        refuses(result, needle);
    }
    let bound = compose(f.inputs.clone()).unwrap();
    for key in ["unknown", "runtime"] {
        let mut raw = serde_json::to_value(&bound).unwrap();
        raw["inputs"][key] = json!({"harness":"/foreign"});
        let bytes = serde_json::to_vec(&raw).unwrap();
        let result = StageBindings::decode(&bytes).map(|_| ());
        proof(
            "unknown typed input field",
            &["orchestration:compose:102", "orchestration:compose:147"],
            &raw,
            &result,
        );
        refuses(result, "unknown field");
    }
    let raw = serde_json::to_string(&bound).unwrap().replacen(
        "\"inputs\":",
        "\"inputs\":null,\"inputs\":",
        1,
    );
    let result = StageBindings::decode(raw.as_bytes()).map(|_| ());
    proof(
        "duplicate typed input key",
        &["orchestration:compose:147"],
        &raw,
        &result,
    );
    assert!(result.is_err());
    let mut raw = serde_json::to_value(&bound).unwrap();
    raw["inputs"]
        .as_object_mut()
        .unwrap()
        .remove("runtime_build_receipt");
    let result = StageBindings::decode(&serde_json::to_vec(&raw).unwrap()).map(|_| ());
    proof(
        "missing typed runtime input",
        &["orchestration:compose:102"],
        &raw,
        &result,
    );
    refuses(result, "missing field");
}
#[test]
fn composition_reopens_deleted_bound_files_before_publication() {
    for (name, ids, selected) in [
        (
            "runtime harness",
            &["orchestration:compose:85", "orchestration:compose:102"][..],
            0,
        ),
        (
            "runtime receipt",
            &["orchestration:compose:85", "orchestration:compose:102"][..],
            1,
        ),
        ("SQLite receipt", &["orchestration:compose:125"][..], 2),
        ("mandatory spec", &["orchestration:compose:87"][..], 3),
        (
            "managed executable",
            &["orchestration:compose:79", "orchestration:compose:183"][..],
            4,
        ),
    ] {
        let f = Fixture::new();
        let bound = compose(f.inputs.clone()).unwrap();
        let control = bound.validate();
        proof(&format!("{name} valid control"), ids, &bound, &control);
        control.unwrap();
        let selected = match selected {
            0 => PathBuf::from(&f.inputs.leaves.paths[&Binding::RuntimeHarness]),
            1 => f.inputs.runtime_build_receipt.clone(),
            2 => f.inputs.sqlite_build_receipt.clone(),
            3 => PathBuf::from(
                &f.inputs.leaves.specifications["workload-fixture-controlled-success"],
            ),
            _ => PathBuf::from(&f.inputs.leaves.paths[&Binding::Binary]),
        };
        let old = fs::read(&selected).unwrap();
        fs::remove_file(&selected).unwrap();
        let result = bound.validate();
        proof(
            &format!("{name} deleted bound file"),
            ids,
            &json!({"bindings":bound,"deleted_path":selected,"original_bytes":old}),
            &result,
        );
        assert!(result.is_err(), "deleted {name} accepted");
    }
}

#[test]
fn static_source_helpers_catch_changed_dispositions_routes_and_tool_identity() {
    let f = Fixture::new();
    let inputs = &f.inputs.static_inputs;
    let b = &f.inputs.leaves;
    let control = static_plan(inputs, b).map(|_| ());
    proof(
        "static construction normal-return control",
        &[
            "orchestration:static_stages:184",
            "orchestration:static_stages:202",
            "orchestration:static_stages:272",
        ],
        inputs,
        &control,
    );
    control.unwrap();
    for mode in 0..4 {
        let mut altered = inputs.clone();
        let needle = match mode {
            0 => {
                altered
                    .tools
                    .insert("foreign".into(), inputs.tools["go"].clone());
                "tools"
            }
            1 => {
                altered.tools.get_mut("cargo_clippy").unwrap().path =
                    inputs.tools["cargo"].path.clone();
                "original tool names"
            }
            2 => {
                altered.caches.remove("go_mod_cache");
                "caches"
            }
            _ => {
                let outside = f.root.join("outside-rustc");
                fs::copy(&inputs.tools["rustc"].path, &outside).unwrap();
                altered.tools.get_mut("rustc").unwrap().path = outside;
                "share pinned directory"
            }
        };
        let result = static_plan(&altered, b).map(|_| ());
        proof(
            &format!("static tool/cache predicate {mode}"),
            &[],
            &altered,
            &result,
        );
        refuses(result, needle);
    }
    let root = inputs.repo.join("tools/gomad/rust");
    let registry = root.join("driver/static-ignored.json");
    let original = custody::strict_json(&fs::read(&registry).unwrap()).unwrap();
    for (name, mutate, needle) in [
        (
            "duplicate disposition",
            (|v: &mut Value| {
                let row = v["tests"][0].clone();
                v["tests"].as_array_mut().unwrap().push(row);
            }) as fn(&mut Value),
            "duplicate",
        ),
        (
            "unknown final obligation",
            |v: &mut Value| v["tests"][0]["final_id"] = json!("foreign/obligation"),
            "receiving obligation",
        ),
        (
            "crate/source mismatch",
            |v: &mut Value| v["tests"][0]["crate"] = json!("wasmhost"),
            "crate/source",
        ),
        (
            "ignored schema",
            |v: &mut Value| v["schema"] = json!("old/v1"),
            "schema",
        ),
    ] {
        let mut altered = original.clone();
        mutate(&mut altered);
        fs::write(&registry, serde_json::to_vec(&altered).unwrap()).unwrap();
        let result = ignored_inventory(&inputs.repo).map(|_| ());
        proof(name, &[], &altered, &result);
        refuses(result, needle);
    }
    fs::write(&registry, serde_json::to_vec(&original).unwrap()).unwrap();
    let old_source = root.join("gomad/src/lib.rs");
    let source = root.join("gomad/src/replay/cluster/broad_guest_tests.rs");
    fs::create_dir_all(source.parent().unwrap()).unwrap();
    fs::rename(&old_source, &source).unwrap();
    let mut broad = original.clone();
    broad["tests"][0]["source"] = json!("gomad/src/replay/cluster/broad_guest_tests.rs");
    broad["tests"][0]["broad_routes"] = json!([
        {"stage":"cluster-provider","invocation":"host-broad-guest","selector":"replay::cluster::broad_guest_tests","features":["determinism-fixture"]},
        {"stage":"retained-cluster","invocation":"host-broad-guest","selector":"replay::cluster::broad_guest_tests","features":["determinism-fixture"]}
    ]);
    fs::write(&registry, serde_json::to_vec(&broad).unwrap()).unwrap();
    let control = ignored_inventory(&inputs.repo).map(|_| ());
    proof("ignored actual broad route control", &[], &broad, &control);
    control.unwrap();
    broad["tests"][0]["broad_routes"][0]["features"] = json!([]);
    fs::write(&registry, serde_json::to_vec(&broad).unwrap()).unwrap();
    let result = ignored_inventory(&inputs.repo).map(|_| ());
    proof(
        "changed ignored broad feature selection",
        &[],
        &broad,
        &result,
    );
    refuses(result, "fixed executable, features or final selectors");
    let workflow = inputs.repo.join(".github/workflows/gomad.yml");
    fs::write(&workflow, "paths: [tools/foreign/**]\n").unwrap();
    let result = workflow_pair(&inputs.repo).map(|_| ());
    proof(
        "changed fixed workflow module routing",
        &[],
        &json!({"path":workflow,"bytes":"paths: [tools/foreign/**]\n"}),
        &result,
    );
    refuses(result, "module routing");
}
