use super::*;
use std::io;
fn args(values: &[&str]) -> Vec<String> {
    values.iter().map(|value| (*value).into()).collect()
}
fn repo() -> String {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(4)
        .unwrap()
        .to_string_lossy()
        .into_owned()
}
fn valid(name: &str) -> Vec<String> {
    let mut result = vec![name.into(), "--repo".into(), repo()];
    for input in inputs(name).unwrap().iter().filter(|input| input.required) {
        result.extend([
            format!("--{}", flag(input.variable)),
            format!("/{}", flag(input.variable)),
        ]);
    }
    result
}
#[test]
fn flags_and_commands_are_derived_from_inputs() {
    for (variable, expected) in [
        ("GOMAD_WASM_PROFILE_ROOT", "profile-root"),
        ("GOMAD_STOCK_GO", "stock-go"),
        ("GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES", "features"),
    ] {
        assert_eq!(flag(variable), expected);
    }
    let values = BTreeMap::from([
        ("GOMAD_WASM_PROFILE_ROOT".into(), "/root".into()),
        ("GOMAD_STOCK_GO".into(), String::new()),
        ("PATH".into(), "/bin".into()),
    ]);
    assert_eq!(
        command("gomad-measure", "/repo", "runtime-cost-profile", &values),
        args(&[
            "gomad-measure",
            "runtime-cost-profile",
            "--repo",
            "/repo",
            "--profile-root",
            "/root"
        ])
    );
    let values = BTreeMap::from([
        ("GOMAD_WASM_PROFILE_ROOT".into(), "/a b".into()),
        ("PATH".into(), "/bin".into()),
    ]);
    let record = rerun_record(
        "gomad-measure",
        "/repo",
        "runtime-cost-profile",
        &values,
        true,
    );
    assert_eq!(record.working_directory, Path::new("/repo/tools/gomad"));
    assert_eq!(record.environment["PATH"], "/bin");
    assert_eq!(record.environment["GOROOT"], "");
    assert_eq!(record.command,RerunCommand::Shell("env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off gomad-measure runtime-cost-profile --repo /repo --profile-root '/a b'".into()));
    assert_eq!(
        shell(&args(&["", "a'b", "snow☃", "safe/a-b_c.d=e:f,g"])),
        "'' 'a'\\''b' 'snow☃' safe/a-b_c.d=e:f,g"
    );
    for name in names() {
        let catalogue = inputs(name).unwrap();
        assert!(catalogue[0].required);
        let mut seen = std::collections::BTreeSet::new();
        for input in catalogue {
            assert!(input.variable.starts_with("GOMAD_"));
            let derived = flag(input.variable);
            assert!(derived != "repo" && derived != "go");
            assert!(seen.insert(derived));
        }
    }
}
struct FailingOutput {
    writes: usize,
}
impl Write for FailingOutput {
    fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
        if self.writes == 0 {
            Err(io::Error::other("output unavailable"))
        } else {
            self.writes -= 1;
            Ok(bytes.len())
        }
    }
    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}
#[test]
fn output_failure_is_infrastructure_failure() {
    for (arguments, stdout_fails, writes) in [
        (args(&["--help"]), true, 0),
        (args(&["--help"]), true, 1),
        (args(&["unknown"]), false, 0),
        (args(&["runtime-cost-profile", "--unknown"]), false, 0),
        (
            args(&["runtime-cost-profile", "--repo", "relative"]),
            false,
            0,
        ),
    ] {
        let mut fail = FailingOutput { writes };
        let mut sink = io::sink();
        let (stdout, stderr): (&mut dyn Write, &mut dyn Write) = if stdout_fails {
            (&mut fail, &mut sink)
        } else {
            (&mut sink, &mut fail)
        };
        assert_eq!(
            run(&arguments, &[], stdout, stderr, |_, _, _| panic!(
                "producer started"
            )),
            EXIT_FAILED
        );
    }
}
#[test]
fn every_subcommand_refuses_without_explicit_inputs() {
    assert_eq!(names().len(), 25);
    for name in names() {
        let prefix = vec![name.to_string(), "--repo".into(), repo()];
        let mut extra = prefix.clone();
        extra.push("positional".into());
        let mut unknown = prefix.clone();
        unknown.extend(args(&["--not-an-input", "x"]));
        for arguments in [
            prefix,
            vec![name.into()],
            args(&[name, "--repo", "relative"]),
            args(&[name, "--repo", "/nonexistent-checkout"]),
            extra,
            unknown,
        ] {
            let mut error = Vec::new();
            assert_eq!(
                run(
                    &arguments,
                    &[],
                    &mut io::sink(),
                    &mut error,
                    |_, _, _| panic!("producer started")
                ),
                EXIT_REFUSED,
                "{name}: {}",
                String::from_utf8_lossy(&error)
            );
        }
        let mut error = Vec::new();
        assert_eq!(
            run(
                &[name.into()],
                &args(&["GOMAD_WASM_PROFILE_ROOT=/tmp/x"]),
                &mut io::sink(),
                &mut error,
                |_, _, _| panic!("producer started")
            ),
            EXIT_REFUSED
        );
        assert!(
            String::from_utf8(error)
                .unwrap()
                .contains("unset GOMAD_WASM_PROFILE_ROOT")
        );
    }
    assert_eq!(
        run(
            &args(&["no-such-measurement"]),
            &[],
            &mut io::sink(),
            &mut io::sink(),
            |_, _, _| panic!()
        ),
        EXIT_REFUSED
    );
}
#[test]
fn flags_become_producer_inputs() {
    for code in [0, EXIT_FAILED, EXIT_REFUSED] {
        let mut arguments = valid("runtime-cost-profile");
        arguments.extend(args(&["--go", "/build/go"]));
        assert_eq!(
            run(
                &arguments,
                &args(&["PATH=/bin"]),
                &mut io::sink(),
                &mut io::sink(),
                |call, _, _| {
                    assert_eq!(call.name, "runtime-cost-profile");
                    assert_eq!(call.repo, Path::new(&repo()));
                    assert_eq!(call.go_command, "/build/go");
                    assert_eq!(call.environ, args(&["PATH=/bin"]));
                    assert_eq!(
                        call.inputs,
                        BTreeMap::from([
                            ("GOMAD_STOCK_GO".into(), "/stock-go".into()),
                            ("GOMAD_WASM_PROFILE_ROOT".into(), "/profile-root".into()),
                            ("GOMAD_WASM_TEST_HELPER".into(), "/test-helper".into())
                        ])
                    );
                    code
                }
            ),
            code
        );
    }
    let mut arguments = valid("diagnostic-helper-build");
    arguments.extend(args(&["--features", "diagnostic-profile,guest-profile"]));
    assert_eq!(
        run(
            &arguments,
            &[],
            &mut io::sink(),
            &mut io::sink(),
            |call, _, _| {
                assert_eq!(
                    call.inputs["GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES"],
                    "diagnostic-profile,guest-profile"
                );
                0
            }
        ),
        0
    );
}
#[test]
fn keep_flag_reaches_producer() {
    let mut arguments = valid("runtime-cost-profile");
    arguments.push("--keep".into());
    assert_eq!(
        run(
            &arguments,
            &[],
            &mut io::sink(),
            &mut io::sink(),
            |call, _, _| {
                assert!(call.keep);
                0
            }
        ),
        0
    );
}
#[test]
fn executor_refuses_another_checkout() {
    assert!(
        authenticate_checkout(Path::new("/nonexistent-checkout"))
            .unwrap_err()
            .contains("executor was not built from --repo")
    );
}
#[test]
fn refusal_order_and_required_order_are_stable() {
    let mut error = Vec::new();
    run(
        &args(&["runtime-cost-profile", "--repo", "relative", "extra"]),
        &[],
        &mut io::sink(),
        &mut error,
        |_, _, _| panic!(),
    );
    assert!(
        String::from_utf8(error)
            .unwrap()
            .contains("unexpected arguments")
    );
    let mut error = Vec::new();
    run(
        &["runtime-cost-profile".into(), "--repo".into(), repo()],
        &[],
        &mut io::sink(),
        &mut error,
        |_, _, _| panic!(),
    );
    assert!(
        String::from_utf8(error)
            .unwrap()
            .ends_with("missing required --stock-go, --profile-root, --test-helper\n")
    );
    for bad in [
        format!("{}//", repo()),
        format!("{}/.", repo()),
        format!("{}/../{}", repo(), "gomad"),
    ] {
        let mut error = Vec::new();
        run(
            &["runtime-cost-profile".into(), "--repo".into(), bad],
            &[],
            &mut io::sink(),
            &mut error,
            |_, _, _| panic!(),
        );
        assert!(String::from_utf8(error).unwrap().contains("absolute clean"));
    }
}

#[test]
fn source_closure_authenticates_bytes_and_membership() {
    let root = std::env::temp_dir().join(format!(
        "gomad-measure-source-contract-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    std::fs::create_dir_all(root.join("src")).unwrap();
    std::fs::write(root.join("src/lib.rs"), b"original").unwrap();
    let manifest = format!(
        "{}\tsrc/lib.rs\n",
        crate::evidence::canonical::digest(b"original")
    );
    assert!(authenticate_source_closure(&root, &manifest).is_ok());
    std::fs::write(root.join("src/lib.rs"), b"changed").unwrap();
    assert!(
        authenticate_source_closure(&root, &manifest)
            .unwrap_err()
            .contains("changed")
    );
    std::fs::write(root.join("src/lib.rs"), b"original").unwrap();
    std::fs::write(root.join("src/injected.rs"), b"additional").unwrap();
    assert!(
        authenticate_source_closure(&root, &manifest)
            .unwrap_err()
            .contains("membership")
    );
    std::fs::remove_file(root.join("src/injected.rs")).unwrap();
    std::fs::remove_file(root.join("src/lib.rs")).unwrap();
    assert!(authenticate_source_closure(&root, &manifest).is_err());
    std::fs::remove_dir_all(root).unwrap();
}
