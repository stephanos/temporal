mod broad;
mod ordinary;
mod primary;
mod static_commands;

use crate::cycle::model::{Adapter, Stage};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Copy, Debug, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
pub(crate) enum Binding {
    Repo,
    Root,
    Config,
    Binary,
    Python,
    Go,
    Cargo,
    Rustc,
    Clippy,
    Make,
    Golangci,
    Errortype,
    True,
    CargoHome,
    RuntimeHarness,
    Bootstrap,
    Compiler,
    RuntimeRoot,
    Cache,
    CapturedConfig,
    SourceManifest,
    BuildReceipt,
    CompilerRoot,
    GuestConfig,
    ModuleCache,
    ClusterTemplate,
    BrokerObservation,
    HostFixture,
    HostTests,
    HelperTests,
    StockGo,
    FreshOrdinary,
    FreshAdmission,
    FreshBuild,
    Diagnostic,
    DiagnosticAdmission,
    Module,
    ModuleReceipt,
    TerminalSpec,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct LeafBindings {
    pub paths: BTreeMap<Binding, String>,
    pub installations: BTreeMap<String, String>,
    pub specifications: BTreeMap<String, String>,
    pub lint_base: String,
    pub environment: BTreeMap<String, String>,
    pub reference_environment: BTreeMap<String, String>,
}

impl LeafBindings {
    fn get(&self, key: Binding) -> Result<&str, String> {
        let value = self
            .paths
            .get(&key)
            .ok_or_else(|| format!("missing leaf binding {key:?}"))?;
        super::absolute(std::path::Path::new(value))?;
        Ok(value)
    }

    fn selected<'a>(paths: &'a BTreeMap<String, String>, key: &str) -> Result<&'a str, String> {
        let path = paths
            .get(key)
            .ok_or_else(|| format!("missing selected leaf input {key}"))?;
        super::absolute(std::path::Path::new(path))?;
        Ok(path)
    }

    fn join(&self, key: Binding, suffix: &str) -> Result<String, String> {
        Ok(format!("{}/{suffix}", self.get(key)?))
    }

    fn work(&self, stage: &str) -> Result<String, String> {
        self.join(Binding::Root, &format!("stages/{stage}/primary"))
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) enum Expansion {
    RuntimeShard {
        index: u32,
        instances: u32,
        replays: u32,
        wall_seconds: u32,
    },
    RuntimeAggregate {
        shards: u32,
    },
    Corpus {
        after_relocation: bool,
    },
    SqliteSelection {
        index: u32,
    },
    SqliteCatalogue,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct LeafPlan {
    pub commands: Vec<LeafCommand>,
    pub expansion: Option<Expansion>,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
pub(crate) enum LeafFlag {
    Reference,
    AllPlatforms,
    LintPolicy,
    Regenerate,
    PrivateObservations,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct LeafCommand {
    pub argv: Vec<String>,
    pub name: Option<String>,
    pub kind: Option<String>,
    pub cwd: Option<String>,
    pub environment: BTreeMap<String, String>,
    pub accepted_exit: Vec<i32>,
    pub timeout_s: Option<u64>,
    pub repeats: Option<u32>,
    pub flags: BTreeSet<LeafFlag>,
    pub role: Option<String>,
    pub features: Vec<String>,
    pub test_filter: Option<String>,
}

fn strings(items: &[&str]) -> Vec<String> {
    items.iter().map(|item| (*item).into()).collect()
}
fn extended(base: &[String], tail: &[&str]) -> Vec<String> {
    base.iter().cloned().chain(strings(tail)).collect()
}
fn command(argv: &[&str]) -> LeafCommand {
    owned(strings(argv))
}
fn owned(argv: Vec<String>) -> LeafCommand {
    LeafCommand {
        argv,
        name: None,
        kind: None,
        cwd: None,
        environment: BTreeMap::new(),
        accepted_exit: vec![0],
        timeout_s: None,
        repeats: None,
        flags: BTreeSet::new(),
        role: None,
        features: vec![],
        test_filter: None,
    }
}
fn named(name: &str, argv: &[&str]) -> LeafCommand {
    let mut row = command(argv);
    row.name = Some(name.into());
    row
}

impl Stage {
    pub(crate) fn leaf_commands(&self, bindings: &LeafBindings) -> Result<LeafPlan, String> {
        let label = self.contract.label.as_str();
        if !super::contracts().iter().any(|row| row.label == label) {
            return Err("unknown leaf stage".into());
        }
        let mut plan = match self.contract.adapter {
            Adapter::CheckedCommand => LeafPlan {
                commands: static_commands::derive(label, bindings)?,
                expansion: None,
            },
            Adapter::Primary => LeafPlan {
                commands: primary::derive(label, bindings)?,
                expansion: None,
            },
            Adapter::Runtime => runtime(label, bindings)?,
            Adapter::Qualify => qualify(label, bindings)?,
            Adapter::RuntimeAggregate => LeafPlan {
                commands: vec![],
                expansion: Some(Expansion::RuntimeAggregate { shards: 32 }),
            },
            Adapter::Corpus => LeafPlan {
                commands: vec![],
                expansion: Some(Expansion::Corpus {
                    after_relocation: label == "workload-corpus-after-relocation",
                }),
            },
            Adapter::SqliteSelection => LeafPlan {
                commands: vec![],
                expansion: Some(Expansion::SqliteSelection {
                    index: label
                        .strip_prefix("sqlite-selection-")
                        .and_then(|s| s.parse().ok())
                        .ok_or("invalid SQLite selection index")?,
                }),
            },
            Adapter::SqliteCatalogue => LeafPlan {
                commands: vec![],
                expansion: Some(Expansion::SqliteCatalogue),
            },
        };
        for row in &mut plan.commands {
            map_endpoint(row, bindings)?;
        }
        Ok(plan)
    }

    pub(crate) fn validate_leaf_commands(
        &self,
        bindings: &LeafBindings,
        plan: &LeafPlan,
    ) -> Result<(), String> {
        if *plan != self.leaf_commands(bindings)? {
            return Err("changed leaf command contract".into());
        }
        Ok(())
    }
}

fn runtime(label: &str, b: &LeafBindings) -> Result<LeafPlan, String> {
    let index = label
        .strip_prefix("runtime-")
        .and_then(|v| v.parse::<u32>().ok())
        .filter(|v| *v < 32)
        .ok_or("invalid runtime shard")?;
    let executable = b.get(Binding::RuntimeHarness)?;
    let root = b.join(Binding::Root, &format!("stages/{label}"))?;
    Ok(LeafPlan {
        commands: vec![
            command(&[executable, "identity"]),
            command(&[
                executable,
                "produce",
                &format!("{root}/runtime-installation.json"),
                &b.join(
                    Binding::Repo,
                    "tools/gomad/internal/backends/wasm/testdata/runtime_probes",
                )?,
                &index.to_string(),
                &format!("{root}/recordings"),
            ]),
        ],
        expansion: Some(Expansion::RuntimeShard {
            index,
            instances: 25,
            replays: 1,
            wall_seconds: 600,
        }),
    })
}

fn qualify(label: &str, b: &LeafBindings) -> Result<LeafPlan, String> {
    let installation = label
        .strip_prefix("workload-")
        .and_then(|s| {
            s.strip_suffix("-success")
                .or_else(|| s.strip_suffix("-canary"))
        })
        .ok_or("invalid mandatory cell")?;
    let mut row = command(&[
        b.get(Binding::Binary)?,
        "--installation",
        LeafBindings::selected(&b.installations, installation)?,
        "qualify",
        "--spec",
        LeafBindings::selected(&b.specifications, label)?,
    ]);
    row.accepted_exit = vec![i32::from(label.ends_with("-canary"))];
    Ok(LeafPlan {
        commands: vec![row],
        expansion: None,
    })
}

fn map_endpoint(row: &mut LeafCommand, b: &LeafBindings) -> Result<(), String> {
    if row.argv.len() < 2 {
        return Ok(());
    }
    for (path, endpoint) in ENDPOINTS {
        if row.argv[1] == b.join(Binding::Repo, path)? {
            if row.argv[0] != b.get(Binding::Python)? {
                return Err("foreign orchestration interpreter".into());
            }
            let mut argv = strings(&[
                b.get(Binding::Binary)?,
                "qualification-cycle",
                "internal",
                endpoint,
            ]);
            argv.extend(row.argv.iter().skip(2).cloned());
            row.argv = argv;
            break;
        }
    }
    Ok(())
}

pub(crate) fn endpoint_catalog() -> &'static [(&'static str, &'static str)] {
    ENDPOINTS
}

const ENDPOINTS: &[(&str, &str)] = &[
    (
        "tools/gomad/rust/driver/allocation_receiving.py",
        "allocation-receiving",
    ),
    ("tools/gomad/rust/driver/broad_harness.py", "broad-harness"),
    ("tools/gomad/rust/driver/cell_reader.py", "cell-reader"),
    (
        "tools/gomad/rust/driver/cluster_wire_reference.py",
        "cluster-wire-reference",
    ),
    (
        "tools/gomad/rust/driver/clusters_receiving.py",
        "clusters-receiving",
    ),
    ("tools/gomad/rust/driver/compose.py", "compose"),
    ("tools/gomad/rust/driver/configure.py", "configure"),
    ("tools/gomad/rust/driver/driver.py", "driver"),
    ("tools/gomad/rust/driver/dry_run.py", "dry-run"),
    (
        "tools/gomad/rust/driver/executed_divergence.py",
        "executed-divergence",
    ),
    ("tools/gomad/rust/driver/final.py", "final"),
    (
        "tools/gomad/rust/driver/final_profiles.py",
        "final-profiles",
    ),
    (
        "tools/gomad/rust/driver/final_public_verbs.py",
        "final-public-verbs",
    ),
    (
        "tools/gomad/rust/driver/final_retained.py",
        "final-retained",
    ),
    (
        "tools/gomad/rust/driver/identity_harness.py",
        "identity-harness",
    ),
    (
        "tools/gomad/rust/driver/installed_cluster_failure.py",
        "installed-cluster-failure",
    ),
    (
        "tools/gomad/rust/driver/installed_lifecycle_families.py",
        "installed-lifecycle-families",
    ),
    (
        "tools/gomad/rust/driver/interception_callbacks.py",
        "interception-callbacks",
    ),
    (
        "tools/gomad/rust/driver/interception_streaming.py",
        "interception-streaming",
    ),
    ("tools/gomad/rust/driver/launch.py", "launch"),
    (
        "tools/gomad/rust/driver/main_cli_build.py",
        "main-cli-build",
    ),
    ("tools/gomad/rust/driver/native_gnu.py", "native-gnu"),
    (
        "tools/gomad/rust/driver/original_ringpop.py",
        "original-ringpop",
    ),
    (
        "tools/gomad/rust/driver/primary_stages.py",
        "primary-stages",
    ),
    ("tools/gomad/rust/driver/public_roles.py", "public-roles"),
    ("tools/gomad/rust/driver/receiving.py", "receiving"),
    (
        "tools/gomad/rust/driver/reference_profile.py",
        "reference-profile",
    ),
    ("tools/gomad/rust/driver/sqlite_oracle.py", "sqlite-oracle"),
    ("tools/gomad/rust/driver/static_stages.py", "static-stages"),
    (
        "tools/gomad/rust/driver/terminal_controls.py",
        "terminal-controls",
    ),
    ("tools/gomad/rust/workload_tools/producer.py", "producer"),
    (
        "tools/gomad/rust/workload_tools/source_custody.py",
        "source-custody",
    ),
    (
        "tools/gomad/rust/workload_tools/temporal-inspection.py",
        "temporal-inspection",
    ),
    (
        "tools/gomad/rust/workload_tools/temporal-recovery.py",
        "temporal-recovery",
    ),
    (
        "tools/gomad/rust/workload_tools/temporal-support.py",
        "temporal-support",
    ),
    ("tools/gomad/rust/workload_tools/workloads.py", "workloads"),
    ("tools/gomad/rust/driver/preparation.py", "preparation"),
    (
        "tools/gomad/rust/driver/ordinary_receiving.py",
        "ordinary-receiving",
    ),
];
