use super::*;
use crate::cycle::{build::DryRun, plan::leaf::Binding, process::CycleOwner};
use std::{fs, path::PathBuf, time::Duration};

fn command(
    config: &Value,
    runner: &mut Runner,
    cancelled: &AtomicBool,
    argv: Vec<String>,
) -> Result<(), String> {
    let private = private_context(runner)?;
    let output = custody::observe_command(
        runner,
        &private,
        roles::path(config, "repo")?,
        argv,
        BTreeMap::from([
            (
                "PATH".into(),
                std::env::var("PATH").map_err(|e| e.to_string())?,
            ),
            (
                "HOME".into(),
                std::env::var("HOME").map_err(|e| e.to_string())?,
            ),
            ("GOENV".into(), "off".into()),
            ("GOTOOLCHAIN".into(), "local".into()),
            ("GOWORK".into(), "off".into()),
        ]),
        Duration::from_secs(300),
        cancelled,
    )?;
    require(
        output.completed.accepted(),
        &format!(
            "binding fixture command refused: {}",
            String::from_utf8_lossy(&output.stderr)
        ),
    )
}
fn record(path: &Path) -> Result<Value, String> {
    roles::read(path)
}
fn emit(root: &Path, name: &str, value: &Value) -> Result<(), String> {
    fs::write(root.join(name), inputs::canonical(value)?).map_err(|e| e.to_string())
}
fn reject<T>(
    root: &Path,
    name: &str,
    input: &Value,
    ids: &[&str],
    result: Result<T, String>,
    needle: &str,
) -> Result<(), String> {
    let error = result
        .err()
        .ok_or_else(|| format!("{name} incorrectly accepted"))?;
    require(
        error.contains(needle),
        &format!("{name}: wrong predicate {error}, expected {needle}"),
    )?;
    let bytes = inputs::canonical(input)?;
    let path = root.join(format!("mutant-{name}.json"));
    fs::write(&path, &bytes).map_err(|e| e.to_string())?;
    println!(
        "B03_OWNER_PROOF {}",
        json!({"ids":ids,"case":name,"input":path,"input_sha256":custody::digest(&bytes),"error":error,"control":"fresh freeze and independent reopen returned"})
    );
    Ok(())
}
fn mismatched_consumed_roles(
    prepared: &PreparedInputs,
    roles: &CandidateRoles,
    stages: &StageBindings,
    root: &Path,
    context: &mut FreezeContext<'_>,
) -> Result<(), String> {
    for (binding, role) in CONSUMED_LIVE_ROLES {
        let mut inputs = stages.inputs.clone();
        let replacement = roles::path(roles.config(), "binary")?.display().to_string();
        inputs.leaves.paths.insert(binding, replacement.clone());
        let mismatched = super::super::composition::compose(inputs)?;
        reject(
            root,
            &format!("mismatched-consumed-{role}"),
            &json!({"binding":binding,"replacement":replacement,"authenticated":roles.config()["live_corpus"]["roles"][role]}),
            &["review:R2:consumed-role-path"],
            freeze(prepared.clone(), roles.clone(), mismatched, context),
            "consumed role executable differs from authenticated",
        )?;
    }
    Ok(())
}
fn changed_consumed_roles(
    frozen: &FrozenConfiguration,
    root: &Path,
    context: &mut FreezeContext<'_>,
) -> Result<(), String> {
    for (binding, role) in CONSUMED_LIVE_ROLES {
        let binary = roles::path(
            &frozen.configured.value()["live_corpus"]["roles"][role],
            "binary",
        )?;
        let original = custody::read_file(binary, 128 << 20)?;
        let mut changed = original.clone();
        changed.push(0);
        fs::write(binary, &changed).map_err(|e| e.to_string())?;
        let observed =
            FrozenConfiguration::open(context.output, frozen.configured.sha256(), context);
        fs::write(binary, &original).map_err(|e| e.to_string())?;
        reject(
            root,
            &format!("changed-consumed-{role}"),
            &json!({"binding":binding,"binary":binary,"control_sha256":custody::digest(&original),"mutant_sha256":custody::digest(&changed)}),
            &["review:R2:consumed-role-bytes"],
            observed,
            "frozen input changed",
        )?;
        let parked = root.join(format!("parked-{role}"));
        fs::rename(binary, &parked).map_err(|e| e.to_string())?;
        let observed =
            FrozenConfiguration::open(context.output, frozen.configured.sha256(), context);
        fs::rename(&parked, binary).map_err(|e| e.to_string())?;
        reject(
            root,
            &format!("deleted-consumed-{role}"),
            &json!({"binding":binding,"binary":binary,"expected_sha256":custody::digest(&original)}),
            &["review:R2:consumed-role-exists"],
            observed,
            "custody file:",
        )?;
    }
    FrozenConfiguration::open(context.output, frozen.configured.sha256(), context)?;
    Ok(())
}
fn source_cases(root: &Path, runner: &mut Runner, cancelled: &AtomicBool) -> Result<(), String> {
    let private = private_context(runner)?;
    let repo = root.join("source-fixture");
    fs::create_dir(&repo).map_err(|e| e.to_string())?;
    let env = BTreeMap::from([
        ("PATH".into(), "/usr/bin:/bin".into()),
        ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
        ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
    ]);
    let mut git = |args: &[&str]| -> Result<(), String> {
        let mut argv = vec![
            "/usr/bin/git".into(),
            "-c".into(),
            "maintenance.auto=false".into(),
            "-c".into(),
            "gc.auto=0".into(),
            "-C".into(),
            repo.display().to_string(),
        ];
        argv.extend(args.iter().map(|s| (*s).into()));
        let output = custody::observe_command(
            runner,
            &private,
            &repo,
            argv,
            env.clone(),
            Duration::from_secs(20),
            cancelled,
        )?;
        require(
            output.completed.accepted(),
            "source fixture Git command failed",
        )
    };
    git(&["init", "--quiet"])?;
    git(&[
        "-c",
        "user.name=fixture",
        "-c",
        "user.email=fixture@example.invalid",
        "commit",
        "--quiet",
        "--allow-empty",
        "-m",
        "fixture",
    ])?;
    let error = Snapshot::observe(&repo, runner, cancelled, &private).unwrap_err();
    require(error.contains("empty actual"), &error)?;
    println!(
        "B03_OWNER_PROOF {}",
        json!({"ids":["orchestration:configure:186"],"case":"empty actual Git inventory","input":repo,"error":error})
    );
    fs::write(repo.join("source.rs"), b"fn current() {}\n").map_err(|e| e.to_string())?;
    let invoke = |runner: &mut Runner, args: &[&str]| -> Result<(), String> {
        let mut argv = vec![
            "/usr/bin/git".into(),
            "-c".into(),
            "maintenance.auto=false".into(),
            "-c".into(),
            "gc.auto=0".into(),
            "-C".into(),
            repo.display().to_string(),
        ];
        argv.extend(args.iter().map(|s| (*s).into()));
        let output = custody::observe_command(
            runner,
            &private,
            &repo,
            argv,
            env.clone(),
            Duration::from_secs(20),
            cancelled,
        )?;
        require(
            output.completed.accepted(),
            "source fixture Git command failed",
        )
    };
    invoke(runner, &["add", "source.rs"])?;
    let control = Snapshot::observe(&repo, runner, cancelled, &private)?;
    require(
        control.files.len() == 1,
        "source fixture control wrong selection",
    )?;
    fs::write(repo.join("line\nbreak.rs"), b"fn other() {}\n").map_err(|e| e.to_string())?;
    invoke(runner, &["add", "line\nbreak.rs"])?;
    let error = Snapshot::observe(&repo, runner, cancelled, &private).unwrap_err();
    require(error.contains("clean custody"), &error)?;
    println!(
        "B03_OWNER_PROOF {}",
        json!({"ids":["orchestration:configure:189"],"case":"tracked newline filename","control":control,"mutant_filename":"line\nbreak.rs","error":error})
    );
    Ok(())
}
fn public_cases(
    config: &mut Value,
    root: &Path,
    source: &Snapshot,
    release: &Path,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<(), String> {
    let native_root =
        PathBuf::from(std::env::var("GOMAD_BINDING_NATIVE_ROOT").map_err(|e| e.to_string())?);
    let native = record(&native_root.join("native-controls.json"))?;
    let diagnostic = roles::path(&native, "diagnostic_receipt")?;
    let diagnostic_record = record(diagnostic)?;
    let source_inventory = root.join("public-source.json");
    emit(
        root,
        "public-source.json",
        &serde_json::to_value(source).map_err(|e| e.to_string())?,
    )?;
    let builds = root.join("public-builds.json");
    let inventory = json!({"schema":"gomad.rust-public-builds/v1","source_inventory":source_inventory,"source_inventory_sha256":custody::digest(&fs::read(&source_inventory).map_err(|e|e.to_string())?),"roles":{"release":{"kind":"cycle","build_receipt":release,"sha256":custody::digest(&fs::read(release).map_err(|e|e.to_string())?)},"diagnostic":{"kind":"native","build_receipt":diagnostic,"sha256":custody::digest(&fs::read(diagnostic).map_err(|e|e.to_string())?)}}});
    emit(root, "public-builds.json", &inventory)?;
    let golden_root =
        roles::path(config, "repo")?.join("tools/gomad/testdata/parity/goldens-inspect");
    let mut lines = String::new();
    let mut names = fs::read_dir(&golden_root)
        .map_err(|e| e.to_string())?
        .map(|e| e.map(|e| e.path()).map_err(|e| e.to_string()))
        .collect::<Result<Vec<_>, _>>()?;
    names.sort();
    for path in names {
        if path.is_file() {
            lines.push_str(&custody::digest(&custody::read_file(&path, 16 << 20)?));
            lines.push_str("  ");
            lines.push_str(
                path.file_name()
                    .and_then(|v| v.to_str())
                    .ok_or("golden filename UTF8")?,
            );
            lines.push('\n');
        }
    }
    let golden_inventory = root.join("public-goldens.sha256");
    publication::write(&golden_inventory, lines.as_bytes())?;
    let template = root.join("public-template.json");
    let declared = json!({"schema":"gomad.rust-public-bindings/v1","files":{"release":{"path":config["binary"],"sha256":config["binary_sha256"]},"diagnostic":{"path":diagnostic_record["executable"],"sha256":diagnostic_record["executable_sha256"]},"build-receipts":{"path":builds,"sha256":custody::digest(&fs::read(&builds).map_err(|e|e.to_string())?)}},"trees":{"goldens":{"path":golden_root,"inventory":golden_inventory,"sha256":custody::digest(lines.as_bytes())}}});
    emit(root, "public-template.json", &declared)?;
    config["public_verbs"] = json!({"bindings":template,"required_roles":["release","diagnostic"]});
    config["files"][template.to_str().ok_or("template path UTF8")?] = json!(custody::digest(
        &fs::read(&template).map_err(|e| e.to_string())?
    ));
    let mut unsealed = config.clone();
    unsealed["files"][template.to_str().unwrap()] = json!("0".repeat(64));
    reject(
        root,
        "unsealed-public-template",
        &unsealed,
        &["orchestration:configure:150"],
        public::freeze_public_binding(
            &mut unsealed.clone(),
            &root.join("cycle.json"),
            runner,
            cancelled,
        ),
        "unsealed public",
    )?;
    let mut foreign = declared;
    foreign["schema"] = json!("foreign");
    let bad = root.join("public-foreign.json");
    emit(root, "public-foreign.json", &foreign)?;
    reject(
        root,
        "foreign-public-template",
        &foreign,
        &["orchestration:configure:152"],
        public::public_inputs(config, &bad, runner, cancelled),
        "foreign public",
    )?;
    public::freeze_public_binding(config, &root.join("cycle.json"), runner, cancelled)?;
    reject(
        root,
        "duplicate-public-publication",
        config,
        &["orchestration:configure:161"],
        public::freeze_public_binding(
            &mut config.clone(),
            &root.join("cycle.json"),
            runner,
            cancelled,
        ),
        "File exists",
    )?;
    Ok(())
}
fn source_bound_case(
    root: &Path,
    files: &BTreeMap<String, String>,
    runner: &mut Runner,
    cancelled: &AtomicBool,
) -> Result<(), String> {
    let private = private_context(runner)?;
    let (origin, size) = files
        .keys()
        .filter_map(|name| fs::metadata(name).ok().map(|meta| (name, meta.len())))
        .filter(|(_, size)| *size >= 1 << 20 && *size <= 128 << 20)
        .max_by_key(|(_, size)| *size)
        .ok_or("genuine source bound input missing")?;
    let repo = root.join("source-byte-bound");
    fs::create_dir(&repo).map_err(|e| e.to_string())?;
    let config = json!({"repo":repo});
    let git = |runner: &mut Runner, args: &[&str]| -> Result<(), String> {
        let mut argv = vec![
            "/usr/bin/git".into(),
            "-c".into(),
            "maintenance.auto=false".into(),
            "-c".into(),
            "gc.auto=0".into(),
            "-C".into(),
            repo.display().to_string(),
        ];
        argv.extend(args.iter().map(|arg| (*arg).into()));
        command(&config, runner, cancelled, argv)
    };
    git(runner, &["init", "--quiet"])?;
    git(
        runner,
        &[
            "-c",
            "user.name=fixture",
            "-c",
            "user.email=fixture@example.invalid",
            "commit",
            "--quiet",
            "--allow-empty",
            "-m",
            "source bound fixture",
        ],
    )?;
    fs::write(repo.join("source.rs"), b"fn actual() {}\n").map_err(|e| e.to_string())?;
    git(runner, &["add", "source.rs"])?;
    let source_control = Snapshot::observe(&repo, runner, cancelled, &private)?;
    let aliases = (512u64 << 20) / size + 1;
    let mut paths = Vec::new();
    for n in 0..aliases {
        let target = repo.join(format!("genuine-input-{n}.bin"));
        fs::hard_link(origin, &target).map_err(|e| e.to_string())?;
        paths.push(target);
    }
    git(runner, &["add", "--all"])?;
    let refused = Snapshot::observe(&repo, runner, cancelled, &private)
        .err()
        .ok_or("source512MiB bound disappeared")?;
    require(refused.contains("source inventory byte bound"), &refused)?;
    emit(
        root,
        "source-bound-control.json",
        &json!({"source_control":source_control,"existing_genuine_origin":origin,"origin_sha256":files[origin],"alias_paths":paths,"logical_bytes":size*aliases,"refusal":refused,"mutation":"distinct tracked hard links to retained genuine immutable control bytes; no fabricated bulk payload"}),
    )?;
    Ok(())
}
fn file_boundary_cases(owner: &CycleOwner) -> Result<(), String> {
    let root = PathBuf::from(std::env::var("GOMAD_BINDING_CASE_ROOT").map_err(|e| e.to_string())?);
    fs::create_dir(&root).map_err(|e| e.to_string())?;
    let mut runner = Runner::create(
        owner,
        &root.join("commands"),
        Some("file-boundary-proof".into()),
    )?;
    let cancelled = AtomicBool::new(false);
    let missing = Runner::create(owner, &root.join("unbound-commands"), None)?;
    require(
        private_context(&missing)
            .unwrap_err()
            .contains("bound private"),
        "unbound caller context accepted",
    )?;
    let private = private_context(&runner)?;
    let observe = |runner: &mut Runner, context: &str| {
        custody::observe_command(
            runner,
            context,
            &root,
            vec!["/usr/bin/true".into()],
            BTreeMap::new(),
            Duration::from_secs(10),
            &cancelled,
        )
    };
    require(
        observe(&mut runner, &private)?.completed.accepted(),
        "private caller control failed",
    )?;
    let wrong = observe(&mut runner, "foreign-binding-context")
        .err()
        .ok_or("foreign context accepted")?;
    require(
        wrong.contains("foreign private observation context"),
        &wrong,
    )?;
    require(
        observe(&mut runner, &private)?.completed.accepted(),
        "private caller restoration failed",
    )?;
    emit(
        &root,
        "context-control.json",
        &json!({"private":private,"wrong_context":"foreign-binding-context","refusal":wrong,"valid_before_and_after":true,"missing_context":"binding operations require a bound private observation context"}),
    )?;
    source_cases(&root, &mut runner, &cancelled)?;
    let inventory =
        PathBuf::from(std::env::var("GOMAD_BINDING_FILE_INPUTS").map_err(|e| e.to_string())?);
    let files: BTreeMap<String, String> =
        serde_json::from_value(record(&inventory)?).map_err(|e| e.to_string())?;
    source_bound_case(&root, &files, &mut runner, &cancelled)?;
    let total = files.keys().try_fold(0u64, |sum, name| {
        sum.checked_add(fs::metadata(name).map_err(|e| e.to_string())?.len())
            .ok_or_else(|| "fixture byte overflow".to_owned())
    })?;
    require(
        total > 512 << 20,
        "aggregate fixture does not exceed old erroneous cap",
    )?;
    custody::check_files(&files)?;
    emit(
        &root,
        "aggregate-control.json",
        &json!({"inputs":inventory,"input_sha256":custody::digest(&custody::read_file(&inventory,64<<20)?),"file_count":files.len(),"total_bytes":total,"normal_return":true,"retained_bounds":{"per_file_bytes":128<<20,"count":50000,"source_bytes":512<<20}}),
    )?;
    println!("B03_EVENT genuine-aggregate-and-private-context-controls-returned");
    Ok(())
}

fn native_cases(phase: &str) -> Result<(), String> {
    let root =
        PathBuf::from(std::env::var("GOMAD_BINDING_NATIVE_ROOT").map_err(|e| e.to_string())?);
    fs::create_dir_all(&root).map_err(|e| e.to_string())?;
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let proof = crate::measurement::inputs::authenticate_checkout(&repo)?;
    let cargo_home = root.join("cargo-home");
    fs::create_dir_all(&cargo_home).map_err(|e| e.to_string())?;
    fs::write(
        cargo_home.join("config.toml"),
        b"[build]\njobs=1\n[profile.release]\nincremental=false\n",
    )
    .map_err(|e| e.to_string())?;
    let registry = cargo_home.join("registry");
    if !registry.exists() {
        std::os::unix::fs::symlink("/home/agent/.cargo/registry", &registry)
            .map_err(|e| e.to_string())?;
    }
    let mut environ = std::env::vars()
        .map(|(k, v)| format!("{k}={v}"))
        .collect::<Vec<_>>();
    environ.retain(|row| !row.starts_with("CARGO_HOME="));
    environ.push(format!("CARGO_HOME={}", cargo_home.display()));
    if phase == "native-release" || phase == "native-diagnostic" {
        let role = if phase == "native-release" {
            crate::measurement::build::Role::Release
        } else {
            crate::measurement::build::Role::Diagnostic(vec!["diagnostic-profile".into()])
        };
        let receipt = crate::measurement::build::build_role(&proof, &root, &role, &environ)?;
        emit(&root, &format!("{phase}.json"), &json!({"receipt":receipt}))?;
        println!("B03_EVENT {phase}-returned");
        return Ok(());
    }
    let release = roles::path(&record(&root.join("native-release.json"))?, "receipt")?.to_owned();
    let diagnostic =
        roles::path(&record(&root.join("native-diagnostic.json"))?, "receipt")?.to_owned();
    let witness = roles::native_owner_cases(&json!({"repo":repo}), &root, &release, &diagnostic)?;
    emit(&root, "native-controls.json", &witness)?;
    println!("B03_EVENT genuine-native-controls-returned");
    Ok(())
}
fn role_case(owner: &CycleOwner) -> Result<(), String> {
    let root = PathBuf::from(std::env::var("GOMAD_BINDING_ROLES_ROOT").map_err(|e| e.to_string())?);
    fs::create_dir_all(&root).map_err(|e| e.to_string())?;
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let target = PathBuf::from(std::env::var("CARGO_TARGET_DIR").map_err(|e| e.to_string())?);
    let cargo = PathBuf::from(std::env::var("GOMAD_BUILD_CASE_CARGO").map_err(|e| e.to_string())?);
    let role = std::env::var("GOMAD_BINDING_ROLE").map_err(|e| e.to_string())?;
    let mut runner = Runner::create(
        owner,
        &root.join(format!("commands-{role}")),
        Some("binding-role-proof".into()),
    )?;
    let private = private_context(&runner)?;
    let receipt = DryRun::build(
        &role,
        &repo,
        &root,
        &cargo,
        &target,
        &mut runner,
        &AtomicBool::new(false),
        &private,
    )?;
    println!(
        "B03_ROLE_CONTROL {}",
        json!({"role":role,"receipt":receipt,"receipt_sha256":custody::digest(&custody::read_file(&receipt,64 << 20)?)})
    );
    Ok(())
}
#[expect(
    clippy::too_many_lines,
    reason = "One owner process reuses genuine role/Go fixtures for all independent freeze mutations"
)]
pub(super) fn run_cases(owner: &CycleOwner) -> Result<(), String> {
    let phase = std::env::var("GOMAD_BINDING_PHASE").unwrap_or_else(|_| "freeze".into());
    if phase == "file-boundary" {
        return file_boundary_cases(owner);
    }
    if phase.starts_with("native-") {
        return native_cases(&phase);
    }
    if phase == "role" {
        return role_case(owner);
    }

    let root = PathBuf::from(std::env::var("GOMAD_BINDING_CASE_ROOT").map_err(|e| e.to_string())?);
    fs::create_dir(&root).map_err(|e| e.to_string())?;
    let repo = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .map_err(|e| e.to_string())?;
    let compiler = PathBuf::from(std::env::var("GOMAD_BINDING_GO").map_err(|e| e.to_string())?);
    let mut runner = Runner::create(owner, &root.join("commands"), Some("binding-proof".into()))?;
    let cancelled = AtomicBool::new(false);
    source_cases(&root, &mut runner, &cancelled)?;
    let roles_root =
        PathBuf::from(std::env::var("GOMAD_BINDING_ROLES_ROOT").map_err(|e| e.to_string())?);
    let release = roles_root.join("release/dry-build.json");
    let runtime = roles_root.join("runtime-harness/dry-build.json");
    let mut live_roles = serde_json::Map::new();
    for role in [
        "release",
        "host-fixture",
        "ordinary-helper-test",
        "ordinary-host-test",
    ] {
        let receipt = roles_root.join(role).join("dry-build.json");
        live_roles.insert(
            role.into(),
            json!({"binary":roles::path(&record(&receipt)?,"binary")?,"build_receipt":receipt}),
        );
    }
    let release_record = record(&release)?;
    let runtime_record = record(&runtime)?;
    let private = private_context(&runner)?;
    let source = Snapshot::observe(&repo, &mut runner, &cancelled, &private)?;
    let staged_manifest = root.join("staged-source.manifest");
    publication::write(&staged_manifest, &source.manifest())?;
    let mut fixture = super::super::composition::tests::Fixture::new();
    fixture.inputs.static_inputs.repo = repo.clone();
    let output = root.join("cycle.json");
    fixture.inputs.static_inputs.driver_config = output.clone();
    fixture
        .inputs
        .leaves
        .paths
        .insert(Binding::Repo, repo.display().to_string());
    fixture
        .inputs
        .leaves
        .paths
        .insert(Binding::Config, output.display().to_string());
    fixture.inputs.leaves.paths.insert(
        Binding::Binary,
        roles::path(&release_record, "binary")?
            .display()
            .to_string(),
    );
    fixture.inputs.leaves.paths.insert(
        Binding::RuntimeHarness,
        roles::path(&runtime_record, "binary")?
            .display()
            .to_string(),
    );
    fixture.inputs.leaves.paths.insert(
        Binding::SourceManifest,
        staged_manifest.display().to_string(),
    );
    fixture.inputs.leaves.paths.insert(
        Binding::TerminalSpec,
        root.join("terminal-preparation.json").display().to_string(),
    );
    for (binding, role) in CONSUMED_LIVE_ROLES {
        let receipt = roles_root.join(role).join("dry-build.json");
        fixture.inputs.leaves.paths.insert(
            binding,
            roles::path(&record(&receipt)?, "binary")?
                .display()
                .to_string(),
        );
    }
    fixture.inputs.runtime_build_receipt = runtime;
    fixture.inputs.leaves.environment = BTreeMap::from([
        ("GOCACHE".into(), "/home/agent/.cache/go-build".into()),
        ("GOMODCACHE".into(), "/home/agent/go/pkg/mod".into()),
    ]);
    let install_cache = root.join("install-cache");
    fs::create_dir(&install_cache).map_err(|e| e.to_string())?;
    let installation = root.join("installation.json");
    let mut base = fixture.inputs.base.clone();
    base["repo"] = json!(repo);
    base["root"] = json!(root.join("cycle"));
    base["binary"] = release_record["binary"].clone();
    base["binary_sha256"] = release_record["binary_sha256"].clone();
    base["build_receipt"] = json!(release);
    base["identity"] =
        json!({"host":release_record["host_sha256"],"helper":release_record["helper_sha256"]});
    base["source_manifest"] = json!(staged_manifest);
    base["source_sha256"] = json!(custody::digest(&source.manifest()));
    base["head"] = json!(source.head);
    base["symlinks"] = json!(source.symlinks);
    base["live_corpus"] = json!({"roles":live_roles,"families":{}});
    base["primary"] = json!({"compiler":compiler,"cache":install_cache,"runtime_root":repo.join("tools/gomad/internal/model/runtimesource")});
    base["environment"] = json!(fixture.inputs.leaves.environment);
    base["reference_environment"] = json!(fixture.inputs.leaves.reference_environment);
    let scratch = root.join("scratch");
    fs::create_dir(&scratch).map_err(|e| e.to_string())?;
    base["scratch"] = json!(scratch);
    base["temporary_paths"] =
        json!({"TMPDIR":scratch.join("tmp"),"GOTMPDIR":scratch.join("go-tmp")});
    base["cores"] = json!(1);
    base["mem_gib"] = json!(4);
    command(
        &base,
        &mut runner,
        &cancelled,
        vec![
            roles::path(&release_record, "binary")?
                .display()
                .to_string(),
            "install".into(),
            "--compiler".into(),
            compiler.display().to_string(),
            "--runtime-root".into(),
            repo.join("tools/gomad/internal/model/runtimesource")
                .display()
                .to_string(),
            "--profile".into(),
            "gomad.wasi-cooperative/v1".into(),
            "--cache".into(),
            install_cache.display().to_string(),
            "--output".into(),
            installation.display().to_string(),
        ],
    )?;
    for path in fixture.inputs.leaves.installations.values_mut() {
        *path = installation.display().to_string();
    }
    fixture
        .inputs
        .leaves
        .paths
        .insert(Binding::Root, root.join("cycle").display().to_string());
    let specs = fixture.inputs.leaves.specifications.clone();
    for (label, path) in specs {
        let actual = root.join(format!("{label}.json"));
        publication::write(
            &actual,
            &inputs::canonical(&cell_spec(
                &label,
                &root.join("cycle/cells").join(&label),
                &staged_manifest,
                base["source_sha256"].as_str().ok_or("source sha missing")?,
                &base,
            )?)?,
        )?;
        let _ = path;
        fixture
            .inputs
            .leaves
            .specifications
            .insert(label, actual.display().to_string());
    }
    println!("B03_EVENT actual-sqlite-build-start");
    let sqlite = roles::SqliteBuild::build(&base, &root.join("sqlite"), &mut runner, &cancelled)?;
    fixture.inputs.sqlite_build_receipt = sqlite.clone();
    fixture.inputs.base = base;
    let initial = super::super::composition::compose(fixture.inputs.clone())?;
    let mut public_config = initial.config().clone();
    public_config["files"] = json!(source.files);
    public_cases(
        &mut public_config,
        &root,
        &source,
        &release,
        &mut runner,
        &cancelled,
    )?;
    fixture.inputs.base["public_verbs"] = public_config["public_verbs"].clone();
    let stages = super::super::composition::compose(fixture.inputs.clone())?;
    let mut candidate = stages.config().clone();
    candidate["files"] = json!(source.files);
    let roles = CandidateRoles::authenticate(&candidate, &mut runner, &cancelled)?;
    let mut missing_live = candidate.clone();
    missing_live["live_corpus"]["roles"]
        .as_object_mut()
        .ok_or("live fixture roles absent")?
        .remove("host-fixture");
    reject(
        &root,
        "missing-live-role",
        &missing_live,
        &["orchestration:configure:109"],
        roles::live_role_inputs(&missing_live, &mut runner, &cancelled),
        "unknown binding fields",
    )?;

    emit(
        &root,
        "roles-catchers.json",
        &roles::run_owner_tests(&candidate, &mut runner, &cancelled)?,
    )?;
    let sqlite_record = record(&sqlite)?;
    let prepared = PreparedInputs {
        compiler: serde_json::from_value(sqlite_record["compiler_inputs"].clone())
            .map_err(|e| e.to_string())?,
        sources: serde_json::from_value(sqlite_record["sources"].clone())
            .map_err(|e| e.to_string())?,
    };
    let context = TrustedContext {
        repo_root: &repo,
        now_epoch: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_err(|e| e.to_string())?
            .as_secs_f64(),
        available_cores: std::thread::available_parallelism()
            .map_err(|e| e.to_string())?
            .get(),
    };
    let mut context = FreezeContext {
        output: &output,
        trusted: context,
        runner: &mut runner,
        cancelled: &cancelled,
    };
    mismatched_consumed_roles(&prepared, &roles, &stages, &root, &mut context)?;
    println!("B03_EVENT genuine-freeze-start");
    let frozen = freeze(prepared, roles, stages, &mut context)?;
    let sha = frozen.configured.sha256().to_owned();
    let control_installation = record(&installation)?;
    for (name, inventory, ids) in [
        (
            "empty-installation-compiler",
            json!({}),
            vec!["orchestration:configure:205"],
        ),
        (
            "relative-installation-compiler",
            json!({"relative-compiler-input":"a".repeat(64)}),
            vec!["orchestration:configure:207"],
        ),
    ] {
        let mut mutant = control_installation.clone();
        mutant["admission"]["compiler_inputs"] = inventory.clone();
        mutant["admission"]["inputs_sha256"] =
            json!(custody::digest(&inputs::canonical(&inventory)?));
        fs::write(&installation, inputs::canonical(&mutant)?).map_err(|e| e.to_string())?;
        let result = CandidateRoles::authenticate(&candidate, context.runner, &cancelled);
        fs::write(&installation, inputs::canonical(&control_installation)?)
            .map_err(|e| e.to_string())?;
        reject(
            &root,
            name,
            &mutant,
            &ids,
            result,
            "installation admission identity mismatch",
        )?;
    }
    let mut oracle = candidate.clone();
    oracle["stages"]["sqlite-catalogue"]["oracle"] = json!(["/untrusted/oracle"]);
    reject(
        &root,
        "caller-sqlite-oracle",
        &oracle,
        &["orchestration:configure:129"],
        roles::sqlite_oracle_inputs(&oracle, context.runner, &cancelled),
        "legacy SQLite oracle argv is forbidden",
    )?;

    let reopened = FrozenConfiguration::open(&output, &sha, &mut context)?;
    reopened.recheck(&mut context)?;
    emit(
        &root,
        "control.json",
        &json!({"config":output,"sha256":sha,"source_sha256":frozen.configured.value()["source_sha256"],"plan":frozen.plan,"scope":"genuine bounded dry-run preparation/freeze; final inputs explicitly unexecuted; no qualification"}),
    )?;
    changed_consumed_roles(&frozen, &root, &mut context)?;
    let original = frozen.configured.bytes().to_vec();
    let control = frozen.configured.value().clone();
    let mut unsealed_endpoint = control["files"].clone();
    unsealed_endpoint
        .as_object_mut()
        .ok_or("frozen files missing")?
        .remove(control["binary"].as_str().ok_or("frozen binary missing")?);
    for (name, pointer, value, needle, ids) in [
        (
            "changed-role",
            "/identity/host",
            json!("0".repeat(64)),
            "candidate",
            vec!["orchestration:configure:120"],
        ),
        (
            "stress-cycle",
            "/stress_cycle",
            json!({}),
            "stress matrix",
            vec!["orchestration:configure:196"],
        ),
        (
            "unsealed-endpoint",
            "/files",
            unsealed_endpoint,
            "unsealed actual final endpoint",
            vec!["orchestration:configure:295"],
        ),
    ] {
        let mut mutant = control.clone();
        if pointer == "/stress_cycle" {
            mutant["stress_cycle"] = value;
        } else {
            *mutant
                .pointer_mut(pointer)
                .ok_or("fixture mutation field absent")? = value;
        }
        let bytes = inputs::canonical(&mutant)?;
        fs::write(&output, &bytes).map_err(|e| e.to_string())?;
        reject(
            &root,
            name,
            &mutant,
            &ids,
            FrozenConfiguration::open(&output, &custody::digest(&bytes), &mut context),
            needle,
        )?;
        fs::write(&output, &original).map_err(|e| e.to_string())?;
    }
    let error = publication::fresh(&output).unwrap_err();
    require(error.contains("fresh"), "duplicate freeze output accepted")?;
    reject(
        &root,
        "changed-config-sha",
        &control,
        &["orchestration:configure:182"],
        FrozenConfiguration::open(&output, &"0".repeat(64), &mut context),
        "SHA256 mismatch",
    )?;
    let source_file = repo.join("tools/gomad/rust/gomad/src/cycle/preparation/bindings/mod.rs");
    let original_source = custody::read_file(&source_file, 4 << 20)?;
    let mut changed = original_source.clone();
    changed.extend_from_slice(b"\n");
    fs::write(&source_file, &changed).map_err(|e| e.to_string())?;
    let refused = FrozenConfiguration::open(&output, &sha, &mut context);
    fs::write(&source_file, &original_source).map_err(|e| e.to_string())?;
    reject(
        &root,
        "changed-source",
        &json!({"path":source_file,"control_sha256":custody::digest(&original_source),"mutant_sha256":custody::digest(&changed)}),
        &[],
        refused,
        "frozen input changed",
    )?;
    FrozenConfiguration::open(&output, &sha, &mut context)?;
    println!("B03_EVENT genuine-freeze-and-independent-reopen-returned");
    Ok(())
}
