//! Run trybuild's private targets outside the managed workspace selection.

use std::{env, ffi::OsStr, process::Command};

const CHILD: &str = "UMPIRE_TEMPORAL_COMPILER_CHILD";

#[expect(clippy::print_stdout, reason = "The captured marker proves the child executed its checks")]
pub(crate) fn run(name: &str, check: impl FnOnce()) {
    let marker = format!("umpire compiler test completed: {name}");
    if env::var_os(CHILD).as_deref() == Some(OsStr::new(name)) {
        assert!(env::var_os("BUILD_CACHE_MANAGED").is_none(), "compiler child remained managed");
        check();
        println!("\n{marker}");
        return;
    }
    if env::var_os("BUILD_CACHE_MANAGED").is_none() {
        check();
        return;
    }
    let executable = env::current_exe().unwrap_or_else(|error| panic!("test executable: {error}"));
    let output = Command::new(executable)
        .args(["--exact", name, "--nocapture"])
        .env_remove("BUILD_CACHE_MANAGED")
        .env_remove("BUILD_CACHE_TOOL")
        .env(CHILD, name)
        .output()
        .unwrap_or_else(|error| panic!("compiler test {name}: {error}"));
    assert!(
        output.status.success(),
        "compiler test {name} failed:\n{}\n{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );
    assert!(
        String::from_utf8_lossy(&output.stdout).lines().any(|line| line == marker),
        "compiler test {name} did not complete its checks",
    );
}
