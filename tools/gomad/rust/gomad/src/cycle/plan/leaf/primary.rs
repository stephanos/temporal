use super::{Binding, LeafBindings, LeafCommand, LeafFlag, named, strings};

pub(super) fn derive(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    match label {
        "ordinary" => return super::ordinary::derive(b),
        "cluster-provider" | "cluster-functional" | "retained-cluster" => {
            return super::broad::derive(label, b);
        }
        "wasmmeasure-build"
        | "helper-build"
        | "ordinary-helper-publication"
        | "conformance-helper-build"
        | "terminal-module"
        | "terminal-host-admission"
        | "terminal-exit" => return native(label, b),
        "artifact" | "operator" => return retained(label, b),
        "retained-durable-restart" | "busy-grouped" | "busy-boundaries" | "network-streams" => {
            return parity(label, b);
        }
        _ => (),
    }
    let work = b.work(label)?;
    if label == "terminal-controls" {
        return Ok(vec![script(
            b,
            "primary-final",
            "driver/terminal_controls.py",
            &[
                "produce",
                "--config",
                b.get(Binding::Config)?,
                "--root",
                &work,
            ],
        )?]);
    }
    workload(label, b)
}

fn script(
    b: &LeafBindings,
    name: &str,
    script: &str,
    tail: &[&str],
) -> Result<LeafCommand, String> {
    let mut row = named(
        name,
        &[
            b.get(Binding::Python)?,
            &b.join(Binding::Repo, &format!("tools/gomad/rust/{script}"))?,
        ],
    );
    row.argv.extend(strings(tail));
    Ok(row)
}

fn native(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    if label == "conformance-helper-build" {
        return conformance(b);
    }
    let work = b.work(label)?;
    let bootstrap = b.get(Binding::Bootstrap)?;
    let repo = b.get(Binding::Repo)?;
    let row = match label {
        "wasmmeasure-build" | "helper-build" => named(
            "primary-final",
            &[
                bootstrap,
                "terminal-exit-helper-build",
                "--repo",
                repo,
                "--terminal-exit-helper-build-root",
                &work,
            ],
        ),
        "terminal-host-admission" => named(
            "primary-final",
            &[
                bootstrap,
                "--capture-host-evidence",
                &format!("{work}/host-evidence"),
            ],
        ),
        "terminal-module" => {
            let mut row = named(
                "primary-final",
                &[
                    bootstrap,
                    "--prepare-terminal-module",
                    b.get(Binding::TerminalSpec)?,
                    &format!("{work}/module"),
                ],
            );
            row.flags.insert(LeafFlag::PrivateObservations);
            row
        }
        "terminal-exit" => named(
            "primary-final",
            &[
                bootstrap,
                "terminal-exit",
                "--repo",
                repo,
                "--stock-go",
                b.get(Binding::StockGo)?,
                "--test-helper",
                b.get(Binding::FreshOrdinary)?,
                "--terminal-exit-root",
                &work,
                "--terminal-exit-helper-receipt",
                b.get(Binding::FreshAdmission)?,
                "--terminal-exit-module",
                b.get(Binding::Module)?,
                "--terminal-exit-module-receipt",
                b.get(Binding::ModuleReceipt)?,
                "--terminal-exit-diagnostic-helper",
                b.get(Binding::Diagnostic)?,
                "--terminal-exit-diagnostic-helper-receipt",
                b.get(Binding::DiagnosticAdmission)?,
            ],
        ),
        "ordinary-helper-publication" => {
            return Ok(vec![
                named(
                    "primary-final",
                    &[
                        bootstrap,
                        "terminal-exit-helper-stage",
                        "--repo",
                        repo,
                        "--terminal-exit-helper-stage-root",
                        &work,
                        "--terminal-exit-helper-build-receipt",
                        b.get(Binding::FreshBuild)?,
                    ],
                ),
                named(
                    "primary-discover",
                    &[
                        &format!("{work}/ordinary-host/gomad"),
                        "--installation",
                        LeafBindings::selected(&b.installations, "cooperative-runtime")?,
                        "discover",
                    ],
                ),
            ]);
        }
        _ => return Err("unknown native primary leaf".into()),
    };
    Ok(vec![row])
}

fn retained(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let work = b.work(label)?;
    let mut rows = vec![];
    let families: &[&str] = if label == "artifact" {
        &["typed-detached", "kill-12", "kill-24"]
    } else {
        &["failure"]
    };
    for family in families {
        let name = if *family == "failure" {
            "installed-cluster-failure".into()
        } else {
            format!("installed-lifecycle-{family}")
        };
        let endpoint = if *family == "failure" {
            "driver/installed_cluster_failure.py"
        } else {
            "driver/installed_lifecycle_families.py"
        };
        let mut row = script(
            b,
            &name,
            endpoint,
            &[
                "produce",
                "--config",
                b.get(Binding::Config)?,
                "--main-build",
                b.get(Binding::BuildReceipt)?,
                "--compiler-root",
                b.get(Binding::CompilerRoot)?,
                "--guest-config",
                b.get(Binding::GuestConfig)?,
                "--module-cache",
                b.get(Binding::ModuleCache)?,
                "--root",
                &format!("{work}/{name}"),
            ],
        )?;
        if *family != "failure" {
            row.argv.extend(strings(&["--family", family]));
        }
        rows.push(row);
    }
    rows.push(script(
        b,
        "primary-final",
        "driver/final_retained.py",
        &[
            "--config",
            b.get(Binding::Config)?,
            "--section",
            if label == "artifact" {
                "retained"
            } else {
                "operator"
            },
            "--root",
            b.get(Binding::Root)?,
        ],
    )?);
    Ok(rows)
}

fn parity(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let work = b.work(label)?;
    let python = b.get(Binding::Python)?;
    let binary = b.get(Binding::HostFixture)?;
    if matches!(label, "retained-durable-restart" | "busy-grouped") {
        let mut rows = vec![];
        for index in 0..if label == "busy-grouped" { 10 } else { 1 } {
            rows.push(named(
                &format!("primary-{index}"),
                &[
                    python,
                    &b.join(
                        Binding::Repo,
                        "tools/gomad/testdata/parity/live-cluster-check.py",
                    )?,
                    "--binary",
                    binary,
                    "--input",
                    b.get(Binding::ClusterTemplate)?,
                    "--work",
                    &format!("{work}/{index}"),
                    "--cell",
                    if label == "busy-grouped" {
                        "grouped"
                    } else {
                        "durable"
                    },
                ],
            ));
        }
        return Ok(rows);
    }
    Ok(vec![named(
        "primary-final",
        &[
            python,
            &b.join(
                Binding::Repo,
                "tools/gomad/testdata/parity/live-broker-cluster-check.py",
            )?,
            "--binary",
            binary,
            "--observation",
            b.get(Binding::BrokerObservation)?,
            "--work",
            &format!("{work}/broker"),
            "--roots",
            if label == "busy-boundaries" {
                "33,34"
            } else {
                "36"
            },
        ],
    )])
}

fn workload(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let work = b.work(label)?;
    let common = [
        "--binary",
        b.get(Binding::Binary)?,
        "--repo",
        b.get(Binding::Repo)?,
        "--compiler",
        b.get(Binding::Compiler)?,
        "--runtime-root",
        b.get(Binding::RuntimeRoot)?,
        "--cache",
        b.get(Binding::Cache)?,
        "--captured-config",
        b.get(Binding::CapturedConfig)?,
        "--private-observation-config",
        b.get(Binding::Config)?,
    ];
    if label == "temporal-inspection" {
        let mut args = vec!["--owner", "all"];
        args.extend(common);
        let root = format!("{work}/inspection");
        args.extend(["--root", &root]);
        return Ok(vec![script(
            b,
            "primary-final",
            "workload_tools/temporal-inspection.py",
            &args,
        )?]);
    }
    let workload = match label {
        "clusters" => "replication",
        "storage" => "recovery",
        "network" => "retry",
        _ => return Err("unknown primary workload leaf".into()),
    };
    let root = format!("{work}/{workload}");
    let mut args = common.to_vec();
    args.extend([
        "--root",
        &root,
        "--workload",
        workload,
        "--mode",
        "success",
        "--source-manifest",
        b.get(Binding::SourceManifest)?,
    ]);
    let mut first = script(
        b,
        if label == "network" {
            "primary-final"
        } else {
            "primary-workload"
        },
        "workload_tools/producer.py",
        &args,
    )?;
    if label == "clusters" {
        first.argv.push("--original-clusters".into());
        return Ok(vec![
            first,
            script(
                b,
                "primary-final",
                "driver/clusters_receiving.py",
                &["--config", b.get(Binding::Config)?],
            )?,
        ]);
    }
    if label == "network" {
        return Ok(vec![first]);
    }
    Ok(vec![
        first,
        script(
            b,
            "primary-final",
            "workload_tools/temporal-recovery.py",
            &[
                "--binary",
                b.get(Binding::Binary)?,
                "--repo",
                b.get(Binding::Repo)?,
                "--private-observation-config",
                b.get(Binding::Config)?,
                "--primary",
                &root,
                "--root",
                &format!("{work}/recovery-selected"),
            ],
        )?,
    ])
}

fn conformance(b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let work = b.work("conformance-helper-build")?;
    let row = {
        let target = format!("{work}/target");
        let mut row = named(
            "primary-final",
            &[
                b.get(Binding::Cargo)?,
                "build",
                "--manifest-path",
                &b.join(Binding::Repo, "tools/gomad/rust/wasmhost/Cargo.toml")?,
                "--release",
                "--locked",
                "--offline",
                "--no-default-features",
                "--features",
                "conformance",
                "--target-dir",
                &target,
                "--message-format=json",
            ],
        );
        for (name, value) in [
            ("RUSTC", b.get(Binding::Rustc)?),
            ("CARGO_HOME", b.get(Binding::CargoHome)?),
            ("CARGO_TARGET_DIR", &target),
            ("CARGO_BUILD_JOBS", "1"),
        ] {
            row.environment.insert(name.into(), value.into());
        }
        row
    };
    Ok(vec![row])
}
