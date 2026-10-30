//! Linux linker configuration and failure controls.
use std::{
    ffi::OsString,
    fs,
    os::unix::{ffi::OsStringExt, fs::PermissionsExt},
    process::Command,
};

type TestResult = Result<(), Box<dyn std::error::Error>>;

#[test]
fn linux_linker_is_scoped_and_requires_the_pinned_tool() -> TestResult {
    let root = crate::framework_root();
    let config: toml::Value =
        toml::from_str(&fs::read_to_string(root.join(".cargo/config.toml"))?)?;
    let targets = config.get("target").and_then(toml::Value::as_table).ok_or("target settings")?;
    assert_eq!(targets.len(), 2);
    for target in ["aarch64-unknown-linux-gnu", "x86_64-unknown-linux-gnu"] {
        let settings = targets[target].as_table().ok_or("target table")?;
        assert_eq!(settings.len(), 1);
        assert_eq!(settings["linker"].as_str(), Some("scripts/mold-linker.sh"));
    }
    let temporary = tempfile::tempdir()?;
    let bin = temporary.path().join("bin");
    let pinned = temporary.path().join("pinned mold");
    fs::create_dir_all(&bin)?;
    fs::create_dir_all(pinned.join("bin"))?;
    let linker = pinned.join("bin/ld.mold");
    fs::write(&linker, "#!/bin/sh\nexit 0\n")?;
    fs::set_permissions(&linker, fs::Permissions::from_mode(0o755))?;
    let log = temporary.path().join("argv");
    for (name, script) in [
        (
            "mise",
            "#!/bin/sh\n[ \"$MISE_AUTO_INSTALL:$MISE_EXEC_AUTO_INSTALL\" = false:false ] || exit 93\n[ \"$1\" = -C ] && [ \"$2\" = \"$TOOL_CONFIG\" ] || exit 95\n[ \"$3 $4\" = 'where github:rui314/mold' ] || exit 94\n[ -z \"${FAIL_MISE:-}\" ] || exit 37\nprintf '%s\\n' \"$PINNED\"\n",
        ),
        ("cc", "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LINK_LOG\"\n"),
    ] {
        let path = bin.join(name);
        fs::write(&path, script)?;
        fs::set_permissions(&path, fs::Permissions::from_mode(0o755))?;
    }
    let command = || {
        let mut command = Command::new("/bin/bash");
        command
            .current_dir(temporary.path())
            .env("BASH_ENV", "/dev/null")
            .env("TOOL_CONFIG", root.join("scripts/.."))
            .arg(root.join("scripts/mold-linker.sh"))
            .args(["input with spaces.o", "-o", "output"])
            .arg(OsString::from_vec(b"object-\xfe.o".to_vec()))
            .env("PATH", format!("{}:/usr/bin:/bin", bin.display()))
            .env("PINNED", &pinned)
            .env("LINK_LOG", &log)
            .env("MISE_AUTO_INSTALL", "true")
            .env("MISE_EXEC_AUTO_INSTALL", "true");
        command
    };
    assert!(command().status()?.success());
    let mut expected =
        format!("-B{}/bin/\n-fuse-ld=mold\ninput with spaces.o\n-o\noutput\n", pinned.display())
            .into_bytes();
    expected.extend(b"object-\xfe.o\n");
    assert_eq!(fs::read(&log)?, expected);
    fs::remove_file(&log)?;
    assert_eq!(command().env("FAIL_MISE", "1").status()?.code(), Some(37));
    assert!(!log.exists());
    fs::remove_file(linker)?;
    assert!(!command().status()?.success());
    assert!(!log.exists());
    Ok(())
}
