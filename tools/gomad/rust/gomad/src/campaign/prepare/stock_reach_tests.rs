//! Receiving tests for original stock-compiler roots. They run the stock Go
//! 1.27.1 that the merge gate and the ordinary stage put first on PATH, and
//! fail closed when it is absent.
use super::*;
use std::{collections::BTreeSet, process::Command, time::Duration};

fn stock_go() -> PathBuf {
    let path = std::env::split_paths(&std::env::var_os("PATH").expect("PATH"))
        .map(|directory| directory.join("go"))
        .find(|candidate| candidate.is_file())
        .expect("stock Go 1.27.1 must be first on PATH");
    let version = crate::fork_order::output(
        Command::new(&path)
            .arg("version")
            .env("GOTOOLCHAIN", "local"),
    )
    .unwrap();
    assert!(
        String::from_utf8_lossy(&version.stdout).contains("go version go1.27.1 "),
        "stock compiler identity: {:?}",
        String::from_utf8_lossy(&version.stdout)
    );
    fs::canonicalize(path).unwrap()
}

fn checkout() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../../..")
        .canonicalize()
        .unwrap()
}

fn install(compiler: &Path, cache: &Path, config: Option<&Path>, output: &Path) {
    install_with(compiler, cache, config, &[], output);
}

fn install_with(
    compiler: &Path,
    cache: &Path,
    config: Option<&Path>,
    extra: &[String],
    output: &Path,
) {
    let mut arguments = vec![
        "--compiler".into(),
        compiler.to_string_lossy().into_owned(),
        "--cache".into(),
        cache.to_string_lossy().into_owned(),
        "--go-version=go1.27.1".into(),
        "--output".into(),
        output.to_string_lossy().into_owned(),
    ];
    arguments.extend_from_slice(extra);
    if let Some(config) = config {
        arguments.extend(["--config".into(), config.to_string_lossy().into_owned()]);
    }
    crate::cli::install::run(&arguments, &mut Vec::new())
        .unwrap_or_else(|error| panic!("{}", error.message));
}

fn go_run(installation: &Path, directory: &Path, source: &str) -> ExploreOptions {
    let mut options =
        crate::cli::explore::parse(&["go-run".into(), source.into()], &mut false).unwrap();
    options.installation = Some(installation.to_path_buf());
    options.working_dir = directory.to_path_buf();
    options
}

/// Original `backend.TestStockPreparationDisablesAmbientPGO`: malformed,
/// changed and removed ambient `default.pgo` profiles never reach an uncached
/// preparation, its identity, or its bound `-pgo=off` build flags.
#[test]
fn stock_preparation_disables_ambient_pgo() {
    let compiler = stock_go();
    let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
    let source = root.0.join("source");
    fs::create_dir(&source).unwrap();
    fs::write(
        source.join("go.mod"),
        "module cache.pgo.fixture\n\ngo 1.27.1\n",
    )
    .unwrap();
    fs::write(source.join("main.go"), "package main\nfunc main(){}\n").unwrap();
    let prepare_in = |name: &str| {
        let installation = root.0.join(format!("{name}.json"));
        install(&compiler, &root.0.join(name), None, &installation);
        let mut options = go_run(&installation, &source, ".");
        options.build_tags = vec!["test_dep".into()];
        prepare(&options)
    };
    let first = prepare_in("first").unwrap();
    let profile = source.join("default.pgo");
    for (index, data) in [
        "malformed ambient profile",
        "changed malformed ambient profile",
        "",
    ]
    .into_iter()
    .enumerate()
    {
        if data.is_empty() {
            fs::remove_file(&profile).unwrap();
        } else {
            fs::write(&profile, data).unwrap();
        }
        let prepared = prepare_in(&format!("uncached-{index}"))
            .unwrap_or_else(|error| panic!("disabled ambient PGO affected preparation: {error}"));
        assert!(
            prepared.module == first.module && prepared.toolchain == first.toolchain,
            "disabled ambient PGO changed build identity"
        );
        let provenance: Value = serde_json::from_slice(&prepared.provenance).unwrap();
        assert!(
            provenance["inputs"]["build_flags"]
                .as_array()
                .unwrap()
                .contains(&json!("-pgo=off")),
            "disabled PGO not bound by provenance"
        );
    }
}

/// Original `backend.TestSlimGuestClosureCutsOnlyOptionalProviderSeams`: the
/// slim guest closure of `./tests` differs from the full closure by exactly the
/// two optional-provider seams, keeps every retained package, and its
/// preparation inputs follow the excluded modules.
#[test]
#[expect(
    clippy::too_many_lines,
    reason = "One original root: closure listing, seam prediction and module inputs."
)]
fn slim_guest_closure_cuts_only_optional_provider_seams() {
    type Graph = BTreeMap<String, Vec<String>>;
    const SERVER: &str = "go.temporal.io/server/";
    let compiler = stock_go();
    let repo = checkout();
    let environment = process::environment().unwrap();
    let scratch = process::Temporary::new(&std::env::temp_dir()).unwrap();
    let deadline = Instant::now() + Duration::from_secs(600);
    let base = |path: &str| path.split(" [").next().unwrap().to_owned();
    let list = |tags: &[String]| -> (Graph, Vec<String>, Vec<u8>) {
        let data = process::command(
            &compiler,
            &[
                "list".into(),
                "-mod=readonly".into(),
                "-pgo=off".into(),
                "-deps".into(),
                "-test".into(),
                "-json=ImportPath,Imports,DepOnly,Dir,GoFiles,EmbedFiles,TestGoFiles,XTestGoFiles,TestEmbedFiles,XTestEmbedFiles,SFiles,SysoFiles,HFiles,Standard,Module".into(),
                "-tags".into(),
                tags.join(","),
                "./tests".into(),
            ],
            &repo,
            &environment,
            &scratch.0,
            128 << 20,
            deadline,
        )
        .unwrap();
        let (mut graph, mut roots) = (Graph::new(), Vec::new());
        for listed in serde_json::Deserializer::from_slice(&data).into_iter::<Value>() {
            let listed = listed.unwrap();
            let path = listed["ImportPath"].as_str().unwrap().to_owned();
            let imports = listed["Imports"]
                .as_array()
                .map(|items| {
                    items
                        .iter()
                        .map(|item| item.as_str().unwrap().to_owned())
                        .collect()
                })
                .unwrap_or_default();
            if listed["DepOnly"] != true {
                roots.push(path.clone());
            }
            graph.insert(path, imports);
        }
        (graph, roots, data)
    };
    let seams: BTreeMap<String, Vec<String>> = [
        (
            "common/archiver/provider",
            ["common/archiver/gcloud", "common/archiver/s3store"],
        ),
        (
            "tests/testcore",
            [
                "common/persistence/sql/sqlplugin/mysql",
                "common/persistence/sql/sqlplugin/postgresql",
            ],
        ),
    ]
    .into_iter()
    .map(|(from, to)| {
        (
            format!("{SERVER}{from}"),
            to.into_iter().map(|to| format!("{SERVER}{to}")).collect(),
        )
    })
    .collect();
    let cut = |from: &str, to: &str| {
        seams
            .get(&base(from))
            .is_some_and(|targets| targets.contains(&base(to)))
    };
    let reach = |graph: &Graph, roots: &[String], cutting: bool| {
        let (mut seen, mut packages) = (BTreeSet::new(), BTreeSet::new());
        let mut pending = roots.to_vec();
        while let Some(next) = pending.pop() {
            if !seen.insert(next.clone()) {
                continue;
            }
            packages.insert(base(&next));
            for imported in &graph[&next] {
                if graph.contains_key(imported) && !(cutting && cut(&next, imported)) {
                    pending.push(imported.clone());
                }
            }
        }
        packages
    };
    let full_tags: Vec<String> = ["disable_grpc_modules", "sqlite3_dotlk", "test_dep"]
        .map(str::to_owned)
        .into();
    let slim_tags = crate::measurement::module_recipes::slim_host_owner(&repo).build_tags;
    let (full_graph, full_roots, full_listing) = list(&full_tags);
    let (slim_graph, slim_roots, slim_listing) = list(&slim_tags);
    let full = reach(&full_graph, &full_roots, false);
    let predicted = reach(&full_graph, &full_roots, true);
    let slim = reach(&slim_graph, &slim_roots, false);
    for (from, targets) in &seams {
        for to in targets {
            assert!(
                full.contains(from)
                    && full.contains(to)
                    && !slim.contains(to)
                    && slim.contains(from),
                "seam {from} -> {to} is not the measured optional-provider edge"
            );
        }
    }
    let unexpected: Vec<String> = predicted
        .difference(&slim)
        .map(|path| format!("-{path}"))
        .chain(slim.difference(&predicted).map(|path| format!("+{path}")))
        .collect();
    assert!(
        unexpected.is_empty(),
        "slim guest closure differs from the cut seams: {unexpected:?}"
    );
    for required in [
        "common/archiver/provider",
        "common/archiver/filestore",
        "common/membership/static",
        "common/membership/ringpop",
        "common/persistence/sql/sqlplugin/sqlite",
        "common/persistence/visibility/store/elasticsearch",
        "common/persistence/cassandra",
    ]
    .map(|path| format!("{SERVER}{path}"))
    .into_iter()
    .chain(
        [
            "github.com/olivere/elastic/v7",
            "go.temporal.io/auto-scaled-workers/wci/workercomponent",
        ]
        .map(str::to_owned),
    ) {
        assert!(
            slim.contains(&required),
            "slim guest closure dropped retained package {required}"
        );
    }
    let (_, goroot) =
        inventory::compiler(&compiler, &repo, &environment, &scratch.0, deadline).unwrap();
    let include = goroot.join("pkg/include");
    let module_input = |listing: &[u8], module: &str| {
        let marker = format!("/cache/download/{module}/@v/");
        inventory::sources(listing, &include, &BTreeMap::new())
            .unwrap()
            .keys()
            .any(|path| {
                let path = path.to_string_lossy();
                path.contains(&marker) && path.ends_with(".mod")
            })
    };
    for module in [
        "cloud.google.com/go/storage",
        "github.com/go-sql-driver/mysql",
        "github.com/jackc/pgx/v5",
    ] {
        assert!(
            module_input(&full_listing, module) && !module_input(&slim_listing, module),
            "preparation inputs do not follow the {module} exclusion"
        );
    }
    assert!(
        module_input(&full_listing, "github.com/temporalio/ringpop-go")
            && module_input(&slim_listing, "github.com/temporalio/ringpop-go"),
        "preparation inputs do not retain github.com/temporalio/ringpop-go"
    );
    println!(
        "default ./tests guest closure {} packages; slim {} packages",
        full.len(),
        slim.len()
    );
}

/// Original `wasi.TestStockGoGuestRepeatsIn100FreshInstances`: one stock
/// preparation of the environment probe guest runs in 100 fresh stores with
/// identical stdout and WASI transcript.
#[test]
fn stock_go_guest_repeats_in_100_fresh_instances() {
    let compiler = stock_go();
    let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
    let mut entropy = [0_u8; 32];
    entropy[..3].copy_from_slice(&[1, 2, 3]);
    let config = root.0.join("config.json");
    fs::write(
        &config,
        canonical::encode(&json!({"Profile":"gomad.wasi-stock-exploratory/v1",
            "Args":["guest","literal $arg"],"Environment":["PWD=/workspace","TZ=UTC","ONLY=captured"],
            "WorkingDirectory":"/workspace","WritableDirectories":["/workspace","/tmp"],"EntropyKey":entropy,
            "Clock":{"EpochNanos":946_684_800_000_000_000_u64,"ReadStepNanos":1_000_000},
            "Limits":{"OutputBytes":1<<20,"TranscriptBytes":8<<20,"Calls":10000,"Descriptors":64,
                "Files":1000,"FilesystemBytes":8<<20,"PendingEvents":4096}}))
        .unwrap(),
    )
    .unwrap();
    let installation = root.0.join("installation.json");
    // Each run is a fresh store; the installation compile cache only spares
    // recompiling the same module bytes, as qualification replay does.
    install_with(
        &compiler,
        &root.0.join("cache"),
        Some(&config),
        &[
            "--compile-cache".into(),
            root.0.join("compile-cache").to_string_lossy().into_owned(),
        ],
        &installation,
    );
    let mut options = go_run(
        &installation,
        &checkout().join("tools/gomad"),
        "./internal/backends/wasm/testdata/environment",
    );
    options.target_args = vec!["literal $arg".into()];
    options.execution_timeout = Duration::from_secs(60);
    let prepared = prepare(&options).unwrap();
    let mut first: Option<(Vec<u8>, Vec<u8>)> = None;
    for index in 0..100 {
        let completion = super::super::execution::run(
            &prepared,
            &crate::campaign::ledger::Job {
                ordinal: index,
                seed: 7,
            },
            Duration::from_secs(60),
            &linked::Cancellation::default(),
        )
        .unwrap();
        let super::super::execution::Completion::Finalized(result) = completion else {
            panic!("guest {index} did not finalize");
        };
        let record = &result.publication.record;
        assert!(
            record.outcome.termination == "exit" && record.outcome.exit_code == Some(0),
            "guest {index}: {} {:?}: {}",
            record.outcome.termination,
            record.outcome.exit_code,
            record.outcome.reason
        );
        let payloads = &result.publication.payloads;
        let observed = (
            payloads["output/stdout"].clone(),
            payloads["evidence/transcript"].clone(),
        );
        match &first {
            None => {
                let stdout = String::from_utf8_lossy(&observed.0);
                for marker in [
                    "map ",
                    "select ",
                    "random ",
                    "timer ",
                    "fakeTCP echo",
                    "cwd /workspace env captured arg literal $arg",
                    "file scratch",
                ] {
                    assert!(
                        stdout.contains(marker),
                        "missing guest probe {marker:?}: {stdout}"
                    );
                }
                first = Some(observed);
            }
            Some(expected) => assert!(
                *expected == observed,
                "fresh guest {index} diverged from guest 0"
            ),
        }
    }
}
