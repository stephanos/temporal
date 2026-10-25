//! Native child for compiler process inheritance controls.
#![forbid(unsafe_code)]

use std::{
    env,
    ffi::OsStr,
    fs,
    os::unix::ffi::OsStrExt,
    path::Path,
    process::{self, Command},
};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let arguments: Vec<_> = env::args_os().skip(1).collect();
    if let Some(expected) = env::var_os("FIXTURE_EXPECT_LOADER") {
        assert_eq!(env::var_os("DYLD_FALLBACK_LIBRARY_PATH"), Some(expected));
    }
    if env::var_os("FIXTURE_PREVIOUS").is_some()
        && arguments.first().is_some_and(|v| Path::new(v).is_file())
    {
        fs::write(env::var_os("FIXTURE_PREVIOUS").ok_or("previous marker missing")?, b"called")?;
        let status = Command::new(&arguments[0])
            .args(&arguments[1..])
            .env_remove("FIXTURE_PREVIOUS")
            .status()?;
        process::exit(status.code().unwrap_or(1));
    }
    if let Some(sidecar) = env::var_os("FIXTURE_SIDECAR") {
        let record: serde_json::Value = serde_json::from_slice(&fs::read(sidecar)?)?;
        assert_eq!(record["schema"], 2);
        assert_eq!(record["identity"].as_str().ok_or("identity is not a string")?.len(), 64);
    }
    if env::var_os("FIXTURE_FD").is_some() {
        assert_eq!(fs::read("/dev/fd/9")?, b"jobserver");
    }
    if let Some(output) = env::var_os("FIXTURE_OUTPUT") {
        let argv: Vec<_> = arguments.iter().map(|a| a.as_bytes().to_vec()).collect();
        let loader = env::var_os("DYLD_FALLBACK_LIBRARY_PATH").map(|v| v.as_bytes().to_vec());
        let raw = env::var_os("FIXTURE_RAW").map(|v| v.as_bytes().to_vec());
        fs::write(
            output,
            serde_json::to_vec(&serde_json::json!({"argv":argv,"loader":loader,"raw":raw}))?,
        )?;
    }
    if env::var_os("FIXTURE_SIGNAL").is_some() {
        let status = Command::new("/bin/kill")
            .args([OsStr::new("-TERM"), OsStr::new(&process::id().to_string())])
            .status()?;
        assert!(status.success());
        panic!("signal was not delivered");
    }
    process::exit(env::var("FIXTURE_EXIT").unwrap_or_else(|_| "0".into()).parse()?);
}
