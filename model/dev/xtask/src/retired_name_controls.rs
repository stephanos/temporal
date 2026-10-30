use super::*;
use crate::script_control_support::Scratch;
use std::{fs, os::unix::fs::PermissionsExt};

#[test]
fn git_failures_malformed_inventory_and_raw_paths_are_preserved() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let git = scratch.0.join("git control");
    for (body, expected) in [
        ("exit 7", 7),
        ("printf malformed", 2),
        ("printf '/escape\\000'", 2),
        ("printf '../escape\\000'", 2),
    ] {
        fs::write(&git, format!("#!/bin/sh\n{body}\n"))?;
        fs::set_permissions(&git, fs::Permissions::from_mode(0o755))?;
        let status = match inventory_with(&scratch.0, &[], git.as_os_str()) {
            Ok(Err(code)) => code,
            Err(_) => 2,
            Ok(Ok(paths)) => panic!("invalid inventory accepted: {paths:?}"),
        };
        assert_eq!(status, expected);
    }
    fs::write(&git, b"#!/bin/sh\nprintf 'raw-\\377\\000raw-\\377\\000'\n")?;
    let paths = inventory_with(&scratch.0, &[], git.as_os_str())?
        .map_err(|code| io::Error::other(format!("Git failed: {code}")))?;
    assert_eq!(paths.len(), 1);
    assert_eq!(paths[0].1, b"raw-\xff");
    assert_eq!(paths[0].0, scratch.0.join(OsString::from_vec(b"raw-\xff".to_vec())));
    Ok(())
}

#[test]
fn planted_names_fail_and_historical_paths_remain_excluded() -> io::Result<()> {
    let scratch = Scratch::new()?;
    let mut git = Command::new("git");
    git.args(["init", "-q"]).arg(&scratch.0);
    assert!(xtask::gate::scheduler::execute(&git, false, false)?.status.success());
    fs::create_dir_all(scratch.0.join("model/archived"))?;
    fs::write(scratch.0.join("model/archived/history.rs"), ["umpire", "-wasm"].concat())?;
    assert_eq!(check(&scratch.0)?, 0);
    for suffix in ["-wasm", "_tests", "/checker/src/lib.rs", "-annotations"] {
        let name = format!("umpire{suffix}");
        fs::write(scratch.0.join("active.rs"), &name)?;
        assert_eq!(check(&scratch.0)?, 1, "{name}");
    }
    fs::write(scratch.0.join("active.rs"), "umpire/author/dsl umpire/protocol/proto")?;
    assert_eq!(check(&scratch.0)?, 0);
    Ok(())
}
