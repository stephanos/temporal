mod classify;
mod enumerate;
mod inventory;
mod options;
mod process;
mod record;
mod run;

use std::io::Write;
fn command(
    args: &[String],
    environ: &[(String, String)],
    stdout: &mut impl Write,
    stderr: &mut impl Write,
    start: impl FnOnce(&options::Options, &str) -> Result<Vec<String>, String>,
) -> i32 {
    let error = |stderr: &mut dyn Write, message: &str, status| {
        let _ = writeln!(stderr, "wasmsweep: {message}");
        status
    };
    if args
        .iter()
        .any(|arg| matches!(arg.as_str(), "--help" | "-help" | "-h"))
    {
        let _ = writeln!(
            stderr,
            "Usage of wasmsweep:\n  --verify FILE\n  --checkout DIR --gomad FILE --compiler FILE --helper FILE --runtime-root DIR\n  --config-source throwaway|temporal-install --packages ./tests/...\n  --max-tests N (default 0: all) --seed N (default 0)\n  --per-test-timeout DURATION --prepare-timeout DURATION --overall-budget DURATION\n  --retain-bytes BYTES --out DIR --work DIR --artifacts DIR"
        );
        return 2;
    }
    let flags = match options::flags(args) {
        Ok(flags) => flags,
        Err(message) => return error(stderr, &message, 2),
    };
    if let Some(path) = flags.get("verify").filter(|p| !p.is_empty()) {
        if flags.len() != 1 {
            return error(stderr, "--verify takes no other flags", 2);
        }
        return match record::verify_file(path) {
            Ok(findings) => report(stdout, path, &findings),
            Err(message) => error(stderr, &format!("verify: {message}"), 1),
        };
    }
    let options = match options::Options::parse(args, environ) {
        Ok(value) => value,
        Err(message) => return error(stderr, &message, 2),
    };
    let harness = match std::env::current_exe()
        .map_err(|e| e.to_string())
        .and_then(|path| run::file_sha256(&path))
    {
        Ok(value) => value,
        Err(message) => return error(stderr, &format!("identify harness build: {message}"), 1),
    };
    match start(&options, &harness) {
        Ok(findings) => report(
            stdout,
            &options.out.join("sweep.json").display().to_string(),
            &findings,
        ),
        Err(message) => error(
            stderr,
            &message,
            if message.starts_with("refused:") {
                2
            } else {
                1
            },
        ),
    }
}
fn report(stdout: &mut impl Write, path: &str, findings: &[String]) -> i32 {
    for finding in findings {
        if writeln!(stdout, "wasmsweep: finding: {finding}").is_err() {
            return 1;
        }
    }
    let status = if findings.is_empty() {
        "verified".into()
    } else {
        format!("{} findings", findings.len())
    };
    if writeln!(stdout, "wasmsweep: {path}: {status}").is_err() {
        return 1;
    }
    i32::from(!findings.is_empty())
}

/// Runs the standalone bounded WASM sweep developer command.
pub fn command_line() -> i32 {
    let _signals = match process::Signals::install() {
        Ok(guard) => guard,
        Err(message) => {
            eprintln!("wasmsweep: {message}");
            return 1;
        }
    };
    let args = std::env::args().skip(1).collect::<Vec<_>>();
    let environ = std::env::vars().collect::<Vec<_>>();
    command(
        &args,
        &environ,
        &mut std::io::stdout(),
        &mut std::io::stderr(),
        run::run,
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    fn arguments() -> Vec<String> {
        [
            "checkout",
            "gomad",
            "compiler",
            "helper",
            "runtime-root",
            "out",
            "work",
            "artifacts",
        ]
        .into_iter()
        .flat_map(|name| [format!("--{name}"), format!("/fixture/{name}")])
        .chain(
            [
                "--config-source=throwaway",
                "--packages=./tests/a,./tests",
                "--max-tests=10",
                "--per-test-timeout=10m",
                "--prepare-timeout=30m",
                "--overall-budget=24h",
                "--retain-bytes=5GiB",
            ]
            .map(str::to_owned),
        )
        .collect()
    }
    #[test]
    fn validated_standalone_options_reach_runner_with_harness_identity() {
        assert_eq!(
            command(
                &arguments(),
                &[],
                &mut Vec::new(),
                &mut Vec::new(),
                |options, harness| {
                    assert_eq!((options.retain_bytes, options.max_tests), (5 << 30, 10));
                    assert_eq!(options.packages, ["./tests/a", "./tests"]);
                    assert!(harness.starts_with("sha256:"));
                    Ok(Vec::new())
                }
            ),
            0
        );
    }
    #[test]
    fn standalone_refuses_ambient_missing_and_mixed_verify_inputs() {
        for (args, env) in [
            (
                arguments(),
                vec![("GOMAD_WASM_INSTALLATION".into(), "/bad".into())],
            ),
            (
                vec!["--verify=/x".into(), "--retain-bytes=0".into()],
                vec![],
            ),
            (vec!["--unknown=1".into()], vec![]),
            (vec!["extra".into()], vec![]),
        ] {
            assert_eq!(
                command(
                    &args,
                    &env,
                    &mut Vec::new(),
                    &mut Vec::new(),
                    |_, _| panic!("runner started after refusal")
                ),
                2
            );
        }
        let mut args = arguments();
        args.retain(|arg| !arg.starts_with("--retain-bytes"));
        assert_eq!(
            command(&args, &[], &mut Vec::new(), &mut Vec::new(), |_, _| panic!(
                "runner started without retention bound"
            )),
            2
        );
    }
    #[test]
    fn sweep_errors_and_verification_findings_keep_exit_classes() {
        for (result, expected) in [
            (Ok(vec![]), 0),
            (Ok(vec!["TestA: missing row".into()]), 1),
            (Err("enumeration mismatch".into()), 1),
            (Err("refused: incompatible helper".into()), 2),
            (Err("install failed".into()), 1),
        ] {
            assert_eq!(
                command(
                    &arguments(),
                    &[],
                    &mut Vec::new(),
                    &mut Vec::new(),
                    |_, _| result
                ),
                expected
            );
        }
    }
    #[test]
    fn verify_mode_reports_native_sidecar_findings() {
        static NEXT: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);
        let path = std::env::temp_dir().join(format!(
            "gomad-sweep-cli-{}-{}.json",
            std::process::id(),
            NEXT.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        ));
        let sidecar = serde_json::json!({"schema":record::SCHEMA,"statement":record::OBSERVATION_STATEMENT,"candidate":{},"candidate_key":"","hosts":[],"packages":[],"options":{},"harness_sha256":"","rows":[],"by_class":{},"by_package":{},"reasons":[]});
        std::fs::write(&path, serde_json::to_vec(&sidecar).unwrap()).unwrap();
        let mut output = Vec::new();
        assert_eq!(
            command(
                &[format!("--verify={}", path.display())],
                &[],
                &mut output,
                &mut Vec::new(),
                |_, _| panic!("verify started runner")
            ),
            1
        );
        assert!(
            String::from_utf8(output)
                .unwrap()
                .contains("candidate identity or options missing")
        );
        std::fs::remove_file(path).unwrap();
    }
    #[test]
    fn verify_mode_has_no_runner_and_missing_sidecar_is_failure() {
        assert_eq!(
            command(
                &["--verify=/absent/sweep.json".into()],
                &[],
                &mut Vec::new(),
                &mut Vec::new(),
                |_, _| panic!("verify started runner")
            ),
            1
        );
    }
}
