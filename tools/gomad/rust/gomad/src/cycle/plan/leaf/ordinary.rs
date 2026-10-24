use super::{Binding, LeafBindings, LeafCommand, command, strings};

pub(super) fn derive(b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let mut rows = vec![];
    for (name, binding) in [
        ("host", Binding::HostTests),
        ("helper", Binding::HelperTests),
    ] {
        for (suffix, args) in [
            ("list", &["--list"][..]),
            ("ignored", &["--list", "--ignored"][..]),
            ("run", &["--test-threads=1"][..]),
        ] {
            let mut row = command(&[b.get(binding)?]);
            row.argv.extend(strings(args));
            row.name = Some(format!("ordinary-{name}-{suffix}"));
            row.environment.clone_from(&b.environment);
            row.timeout_s = Some(if suffix == "run" { 900 } else { 60 });
            rows.push(row);
        }
    }
    let mut row = command(&[
        b.get(Binding::HelperTests)?,
        "--exact",
        "conformance::tests::volume_conformance_corpus",
        "volume::image::tests::rehashed_images_refuse_persisted_inode_aliases_across_volumes_and_reserved_nodes",
        "--test-threads=1",
    ]);
    row.name = Some("ordinary-helper-volume-run".into());
    row.environment.clone_from(&b.environment);
    row.timeout_s = Some(900);
    rows.push(row);
    rows.extend(go(b)?);
    let mut row = command(&[
        b.get(Binding::Python)?,
        "-m",
        "unittest",
        "discover",
        "-v",
        "-s",
        &b.join(Binding::Repo, "tools/gomad/rust/workload_tools")?,
        "-p",
        "test_*.py",
    ]);
    row.name = Some("ordinary-workload-tools-run".into());
    python_context(&mut row, b);
    rows.push(row);
    let mut row = command(&[
        b.get(Binding::Python)?,
        &b.join(
            Binding::Repo,
            "tools/gomad/testdata/parity/installed-publication-test.py",
        )?,
        "-v",
    ]);
    row.name = Some("ordinary-installed-publication-run".into());
    python_context(&mut row, b);
    rows.push(row);
    Ok(rows)
}

fn python_context(row: &mut LeafCommand, b: &LeafBindings) {
    row.environment.clone_from(&b.environment);
    row.environment
        .insert("PYTHONDONTWRITEBYTECODE".into(), "1".into());
    row.timeout_s = Some(300);
}
fn go(b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let module = b.join(Binding::Repo, "tools/gomad")?;
    let mut rows = vec![];
    for (name, cwd, selector, packages, tags, timeout, reference) in [
        (
            "namespace",
            module.as_str(),
            "^(TestNamespaceConformance|TestVolumeConformance)",
            &["./internal/model/wasi"][..],
            "test_dep",
            600,
            true,
        ),
        (
            "builder",
            &module,
            "^TestRetainedWASMSourcesDoNotImportNativeBuilder$",
            &["./internal/archcheck"][..],
            "test_dep",
            900,
            false,
        ),
        (
            "recursive",
            &module,
            ".",
            &[
                "./internal/model/...",
                "./internal/backends/wasm/cluster",
                "./internal/backends/wasm/toolchain/...",
                "./internal/support/...",
            ][..],
            "test_dep",
            900,
            true,
        ),
        (
            "recursive-builder",
            &module,
            "^TestRetainedWASMSourcesDoNotImportNativeBuilder$",
            &["./internal/archcheck"][..],
            "test_dep",
            900,
            false,
        ),
        (
            "slim",
            b.get(Binding::Repo)?,
            "^TestCloudArchiversRefuseConfiguredSchemesWithoutFallback$|^TestDefaultMembershipLinksRingpop$|^TestSlimGuestRefusesUnlinkedSQLDriversWithoutFallback$",
            &[
                "./common/archiver/provider",
                "./temporal",
                "./tests/testcore",
            ][..],
            "gomad_wasm_slim,test_dep",
            900,
            false,
        ),
    ] {
        for suffix in ["list", "run"] {
            let mut row = command(&[
                b.get(Binding::Go)?,
                "-C",
                cwd,
                "test",
                "-json",
                "-tags",
                tags,
                "-count=1",
                &format!("-timeout={timeout}s"),
                "-run",
                selector,
            ]);
            if suffix == "list" {
                row.argv.extend(strings(&["-list", selector]));
            }
            if name == "recursive" {
                row.argv
                    .extend(strings(&["-skip", "^TestNamespaceConformance"]));
            }
            row.argv.extend(strings(packages));
            row.name = Some(format!("ordinary-{name}-{suffix}"));
            row.timeout_s = Some(timeout);
            row.environment.clone_from(&b.environment);
            if reference {
                row.environment.extend(b.reference_environment.clone());
            }
            rows.push(row);
        }
    }
    Ok(rows)
}
