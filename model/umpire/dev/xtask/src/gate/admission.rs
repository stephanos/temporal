//! Conservative compiler and Go admission within one supervisor-held host allocation.
use std::{env, ffi::OsString, process::Command};

fn value(command: &Command, name: &str) -> Option<OsString> {
    command
        .get_envs()
        .find(|(key, _)| *key == name)
        .map_or_else(|| env::var_os(name), |(_, value)| value.map(OsString::from))
}

/// Cap required validation work while retaining other explicitly selected tool flags.
/// Native backend-skipped commands retain their caller's defaults.
pub fn configure(command: &mut Command, required: bool) {
    if !required {
        return;
    }
    command.env("CARGO_BUILD_JOBS", "1").env("GOMAXPROCS", "1");
    let mut flags = value(command, "GOFLAGS").unwrap_or_default();
    if flags.as_encoded_bytes().split(u8::is_ascii_whitespace).next_back() != Some(b"-p=1") {
        if !flags.is_empty() {
            flags.push(" ");
        }
        flags.push("-p=1");
    }
    command.env("GOFLAGS", flags);
}

/// Configure a source-only clean child without restoring stripped parent tool settings.
pub fn configure_clean(command: &mut Command) {
    command.env("GOFLAGS", "");
    configure(command, true);
}

pub(super) fn receipt(command: &Command) -> serde_json::Value {
    let text = |name| value(command, name).map(|value| value.to_string_lossy().into_owned());
    let bounded_go = value(command, "GOFLAGS").is_some_and(|flags| {
        flags.as_encoded_bytes().split(u8::is_ascii_whitespace).next_back() == Some(b"-p=1")
    });
    serde_json::json!({
        "cargo_build_jobs": text("CARGO_BUILD_JOBS"),
        "go_max_procs": text("GOMAXPROCS"),
        "go_package_parallelism": bounded_go.then_some(1),
        "meaning": "per-tool admission; no CPU-set or total-RSS limit",
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn required_tool_policy_caps_work_and_retains_other_caller_flags() -> std::io::Result<()> {
        for jobs in ["1", "2", "99", "malformed"] {
            let mut command = Command::new("/bin/sh");
            command
                .args([
                    "-c",
                    "printf '%s|%s|%s' \"$CARGO_BUILD_JOBS\" \"$GOMAXPROCS\" \"$GOFLAGS\"",
                ])
                .env("CARGO_BUILD_JOBS", jobs)
                .env("GOMAXPROCS", "99")
                .env("GOFLAGS", "-tags=control -p=8");
            configure(&mut command, true);
            configure(&mut command, true);
            let output = command.output()?;
            assert!(output.status.success());
            assert_eq!(output.stdout, b"1|1|-tags=control -p=8 -p=1");
            let receipt = receipt(&command);
            assert_eq!(receipt["cargo_build_jobs"], "1");
            assert_eq!(receipt["go_package_parallelism"], 1);
        }
        Ok(())
    }

    #[test]
    fn clean_policy_keeps_stripped_flags_out_and_native_policy_keeps_defaults() {
        let mut clean = Command::new("/bin/true");
        clean.env_clear().env("GOFLAGS", "-tags=stripped");
        configure_clean(&mut clean);
        assert_eq!(value(&clean, "GOFLAGS"), Some("-p=1".into()));
        let mut native = Command::new("/bin/true");
        native.env("CARGO_BUILD_JOBS", "2").env("GOMAXPROCS", "2").env("GOFLAGS", "-p=2");
        configure(&mut native, false);
        assert_eq!(value(&native, "CARGO_BUILD_JOBS"), Some("2".into()));
        assert_eq!(value(&native, "GOFLAGS"), Some("-p=2".into()));
    }
}
