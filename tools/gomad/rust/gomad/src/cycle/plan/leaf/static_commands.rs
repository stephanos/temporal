use super::{Binding, LeafBindings, LeafCommand, LeafFlag, extended, owned, strings};

const RETAINED: &[&str] = &[
    "./internal/model/...",
    "./internal/backends/wasm/cluster",
    "./internal/backends/wasm/toolchain/...",
    "./internal/support/...",
];
const HOST_FEATURES: &[&str] = &[
    "",
    "determinism-fixture",
    "helper-stress-jitter",
    "helper-determinism-fixture",
    "determinism-fixture,helper-stress-jitter",
    "determinism-fixture,helper-determinism-fixture",
];
const HELPER_FEATURES: &[&str] = &[
    "reference-helper",
    "conformance",
    "reference-helper,diagnostic-profile",
    "reference-helper,diagnostic-fuel-off",
    "reference-helper,guest-profile",
    "reference-helper,guest-profile,diagnostic-fuel-off",
    "reference-helper,stress-jitter",
];

fn step(kind: &str, argv: Vec<String>, cwd: &str) -> LeafCommand {
    let mut row = owned(argv);
    row.kind = Some(kind.into());
    row.cwd = Some(cwd.into());
    row
}
fn common(b: &LeafBindings, verb: &str) -> Result<Vec<String>, String> {
    Ok(strings(&[
        b.get(Binding::Cargo)?,
        verb,
        "--manifest-path",
        &b.join(Binding::Repo, "tools/gomad/rust/Cargo.toml")?,
        "--release",
        "--locked",
        "--offline",
        "--no-default-features",
    ]))
}
fn role(b: &LeafBindings, features: &str, role: &str, cwd: &str) -> Result<LeafCommand, String> {
    let mut args = extended(
        &common(b, "build")?,
        &["-p", "gomad-wasmhost", "--features", features],
    );
    if role == "release" {
        args = extended(&common(b, "build")?, &["-p", "gomad"]);
    }
    let mut row = step("build-role", args, cwd);
    row.role = Some(role.into());
    row.features = if features.is_empty() {
        vec![]
    } else {
        strings(&[features])
    };
    Ok(row)
}
fn validate(b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let module = b.join(Binding::Repo, "tools/gomad")?;
    let go = b.get(Binding::Go)?;
    let gc = strings(&[go, "test", "-tags", "test_dep", "-count=1"]);
    let cc = common(b, "test")?;
    let mut rows = vec![
        step("versions", strings(&[go, "version"]), &module),
        step(
            "rust-version",
            strings(&[b.get(Binding::Rustc)?, "--version"]),
            &module,
        ),
        step(
            "cargo-version",
            strings(&[b.get(Binding::Cargo)?, "--version"]),
            &module,
        ),
        step(
            "exit",
            strings(&[go, "run", "./internal/model/cmd/protocolgen", "-check"]),
            &module,
        ),
        step(
            "go-tests",
            extended(&gc, &["./internal/model/generation/protocol"]),
            &module,
        ),
        step(
            "replay-regeneration",
            extended(
                &gc,
                &[
                    "-run",
                    "^TestReplayPlanConformance$",
                    "./internal/model/choice",
                ],
            ),
            &module,
        ),
        step(
            "cargo-tests",
            extended(&cc, &["-p", "gomad-wasmhost", "--lib"]),
            &module,
        ),
        role(b, "conformance", "conformance-reference", &module)?,
        role(b, "reference-helper", "ordinary-reference", &module)?,
        step(
            "go-tests",
            extended(
                &strings(&[go, "test", "-tags", "test_dep", "-count=2"]),
                RETAINED,
            ),
            &module,
        ),
        step(
            "cargo-tests",
            extended(&cc, &["-p", "gomad-wasmhost", "--lib"]),
            &module,
        ),
        step(
            "cargo-tests",
            extended(&cc, &["-p", "gomad-wasmhost", "--test", "inventory"]),
            &module,
        ),
    ];
    rows[6].test_filter = Some("replay_plan".into());
    rows[9].repeats = Some(2);
    rows[9].flags.insert(LeafFlag::Reference);
    rows[10].repeats = Some(2);
    Ok(rows)
}

pub(super) fn derive(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let module = b.join(Binding::Repo, "tools/gomad")?;
    let repo = b.get(Binding::Repo)?;
    let go = b.get(Binding::Go)?;
    let gc = strings(&[go, "test", "-tags", "test_dep", "-count=1"]);
    let cc = common(b, "test")?;
    let rows = match label {
        "validate" => return validate(b),
        "portable" => {
            let mut row = step(
                "go-tests",
                extended(&extended(&gc, &["-timeout=20m"]), RETAINED),
                &module,
            );
            row.flags.insert(LeafFlag::Reference);
            vec![row]
        }
        "default-build" => vec![
            step("exit", strings(&[go, "build", "./..."]), &module),
            role(b, "", "release", &module)?,
        ],
        "default-vet" => vec![step(
            "exit",
            strings(&[go, "vet", "-tags", "test_dep", "./..."]),
            &module,
        )],
        "compile-only" => vec![
            step(
                "exit",
                extended(
                    &gc,
                    &["-exec", b.get(Binding::True)?, "-run", "^$", "./..."],
                ),
                &module,
            ),
            step("exit", extended(&cc, &["--workspace", "--no-run"]), &module),
        ],
        "lint-router" => {
            let mut row = step("go-tests", extended(&gc, &["./cmd/tools/lintcode"]), repo);
            row.flags.insert(LeafFlag::LintPolicy);
            vec![row]
        }
        "lint-router-vet" => vec![step(
            "exit",
            strings(&[go, "vet", "-tags", "test_dep", "./cmd/tools/lintcode"]),
            repo,
        )],
        "actionlint-build" => vec![step(
            "actionlint-build",
            strings(&[
                go,
                "install",
                "github.com/rhysd/actionlint/cmd/actionlint@v1.7.7",
            ]),
            repo,
        )],
        "actionlint-version" => vec![step(
            "actionlint-version",
            strings(&["@actionlint", "-version"]),
            repo,
        )],
        "workflow-lint" => vec![step(
            "workflow-lint",
            strings(&["@actionlint", "@workflows"]),
            repo,
        )],
        "lint-module" | "lint-root" => return lint(label, b),
        "architecture" => {
            let mut row = step(
                "go-tests",
                extended(&gc, &["-timeout=15m", "./internal/archcheck"]),
                &module,
            );
            row.flags.insert(LeafFlag::AllPlatforms);
            vec![row]
        }
        "test-harness" => vec![
            step(
                "go-tests",
                extended(&gc, &["./internal/archcheck", "./internal/support/..."]),
                &module,
            ),
            step("cargo-tests", extended(&cc, &["-p", "gomad"]), &module),
        ],
        "world-race" => vec![step(
            "go-tests",
            strings(&[
                go,
                "test",
                "-race",
                "-tags=test_dep",
                "-p=1",
                "-parallel=1",
                "-count=1",
                "./internal/model/world/...",
            ]),
            &module,
        )],
        _ => return rust(label, b, &module),
    };
    Ok(rows)
}

fn rust(label: &str, b: &LeafBindings, module: &str) -> Result<Vec<LeafCommand>, String> {
    if label == "rust-lint" {
        let workspace = b.join(Binding::Repo, "tools/gomad/rust/Cargo.toml")?;
        let mut rows = vec![step(
            "exit",
            strings(&[
                b.get(Binding::Cargo)?,
                "fmt",
                "--manifest-path",
                &workspace,
                "--all",
                "--",
                "--check",
            ]),
            module,
        )];
        for (package, modes) in [
            ("gomad", HOST_FEATURES),
            ("gomad-wasmhost", HELPER_FEATURES),
        ] {
            for features in modes {
                rows.push(step(
                    "exit",
                    strings(&[
                        b.get(Binding::Clippy)?,
                        "clippy",
                        "--manifest-path",
                        &workspace,
                        "--release",
                        "--locked",
                        "--offline",
                        "--no-default-features",
                        "-p",
                        package,
                        "--all-targets",
                        "--no-deps",
                        "--features",
                        features,
                        "--",
                        "-D",
                        "warnings",
                    ]),
                    module,
                ));
            }
        }
        return Ok(rows);
    }
    let features = match label {
        "rust-conformance-unit" => "conformance",
        "rust-diagnostic-test" => "reference-helper,diagnostic-profile",
        "rust-fuel-off-test" => "reference-helper,diagnostic-fuel-off",
        "rust-guest-profile-test" => "reference-helper,guest-profile",
        "rust-combined-test" => "reference-helper,guest-profile,diagnostic-fuel-off",
        "rust-default-test" | "helper-corpus-regen-1" | "helper-corpus-regen-2" => {
            "reference-helper"
        }
        _ => return Err("unknown static leaf stage".into()),
    };
    let common = common(b, "test")?;
    let mut argv = extended(&common, &["-p", "gomad-wasmhost", "--features", features]);
    if label == "rust-conformance-unit" {
        argv.push("--lib".into());
    }
    let regenerate = label.starts_with("helper-corpus-regen-");
    if regenerate {
        argv.extend(strings(&["--test", "helper_corpus"]));
    }
    let mut row = step("cargo-tests", argv, module);
    if regenerate {
        row.flags.insert(LeafFlag::Regenerate);
    }
    let mut rows = vec![row];
    if label == "rust-default-test" {
        rows.push(step(
            "cargo-tests",
            extended(&common, &["-p", "gomad"]),
            module,
        ));
    }
    Ok(rows)
}

fn lint(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let repo = b.get(Binding::Repo)?;
    Ok(match label {
        "lint-module" | "lint-root" => {
            if b.lint_base.len() != 40
                || !b
                    .lint_base
                    .bytes()
                    .all(|v| v.is_ascii_hexdigit() && !v.is_ascii_uppercase())
            {
                return Err("invalid leaf lint base".into());
            }
            vec![step(
                "exit",
                strings(&[
                    b.get(Binding::Make)?,
                    if label == "lint-module" {
                        "lint-code-gomad"
                    } else {
                        "lint-code-fast"
                    },
                    &format!("GOLANGCI_LINT_BASE_REV={}", b.lint_base),
                    "GOLANGCI_LINT_FIX=false",
                    &format!("GOLANGCI_LINT={}", b.get(Binding::Golangci)?),
                    &format!("ERRORTYPE={}", b.get(Binding::Errortype)?),
                ]),
                repo,
            )]
        }
        _ => return Err("unknown lint leaf".into()),
    })
}
