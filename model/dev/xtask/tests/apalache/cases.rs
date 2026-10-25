#[cfg(unix)]
mod tests {
    use super::*;
    use std::{
        io::Write,
        os::unix::fs::{PermissionsExt, symlink},
    };
    const JAR: &[u8] = b"native installer fixture jar";

    fn archive(name: &str, kind: tar::EntryType, bytes: &[u8]) -> Vec<u8> {
        let gzip = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
        let mut target = tar::Builder::new(gzip);
        let mut header = tar::Header::new_gnu();
        header.set_size(bytes.len() as u64);
        header.set_entry_type(kind);
        header.set_mode(0o644);
        header.set_cksum();
        target.append_data(&mut header, name, bytes).unwrap();
        target.into_inner().unwrap().finish().unwrap()
    }
    fn payload() -> Vec<u8> {
        archive(&format!("apalache-{VERSION}/lib/apalache.jar"), tar::EntryType::Regular, JAR)
    }
    fn pins(payload: &[u8]) -> Pins {
        Pins { version: VERSION, archive: hash(payload), jar: hash(JAR) }
    }
    fn fake(root: &Path, payload: &[u8]) -> Result<PathBuf> {
        install_with(root, &pins(payload), |path| {
            fs::write(path, payload)?;
            Ok(())
        })
    }
    fn cache(root: &Path) -> PathBuf {
        root.join("model/build/apalache")
    }
    fn destination(root: &Path) -> PathBuf {
        cache(root).join(format!("apalache-{VERSION}"))
    }
    fn cleaned(root: &Path) {
        let mut entries: Vec<_> =
            fs::read_dir(cache(root)).unwrap().map(|p| p.unwrap().file_name()).collect();
        entries.sort();
        assert_eq!(entries, [".install.lock"]);
    }

    #[test]
    fn hash_before_parse_and_exact_regular_member_before_publication() {
        let name = format!("apalache-{VERSION}/lib/apalache.jar");
        let cases = [
            (b"not gzip".to_vec(), Some("bad"), "archive integrity"),
            (b"not gzip".to_vec(), None, ""),
            (archive("wrong/lib/apalache.jar", tar::EntryType::Regular, JAR), None, "missing"),
            (archive(&name, tar::EntryType::Symlink, b""), None, "regular archive"),
            (
                archive(&name, tar::EntryType::Regular, b"wrong jar"),
                None,
                "extracted jar integrity",
            ),
        ];
        for (payload, override_hash, expected) in cases {
            let root = tempfile::tempdir().unwrap();
            let mut pins = pins(&payload);
            if let Some(digest) = override_hash {
                pins.archive = digest.into();
            }
            let error = install_with(root.path(), &pins, |p| {
                fs::write(p, &payload)?;
                Ok(())
            })
            .unwrap_err();
            if expected.is_empty() {
                assert!(matches!(error, Error::Io(_)), "{error}");
            } else {
                assert!(error.to_string().contains(expected), "{error}");
            }
            cleaned(root.path());
        }
    }

    #[test]
    fn symlinks_in_every_component_including_dangling_are_refused() {
        for component in [
            "model".into(),
            "model/build".into(),
            "model/build/apalache".into(),
            format!("model/build/apalache/apalache-{VERSION}"),
            format!("model/build/apalache/apalache-{VERSION}/lib"),
            format!("model/build/apalache/apalache-{VERSION}/lib/apalache.jar"),
        ] {
            for dangling in [false, true] {
                let root = tempfile::tempdir().unwrap();
                let outside = root.path().join("outside");
                if !dangling {
                    fs::create_dir(&outside).unwrap();
                }
                let link = root.path().join(&component);
                fs::create_dir_all(link.parent().unwrap()).unwrap();
                symlink(&outside, &link).unwrap();
                let error =
                    install_with(root.path(), &pins(&payload()), |_| panic!("must not download"))
                        .unwrap_err();
                assert!(error.to_string().contains("symlink"), "{component}: {error}");
                if dangling {
                    assert!(!outside.exists());
                }
            }
        }
    }

    #[test]
    fn partial_wrong_and_nonregular_installations_never_download() {
        for kind in ["partial", "wrong", "directory"] {
            let root = tempfile::tempdir().unwrap();
            let lib = destination(root.path()).join("lib");
            fs::create_dir_all(&lib).unwrap();
            match kind {
                "wrong" => fs::write(lib.join("apalache.jar"), b"wrong").unwrap(),
                "directory" => fs::create_dir(lib.join("apalache.jar")).unwrap(),
                _ => (),
            }
            let error =
                install_with(root.path(), &pins(&payload()), |_| panic!("must not download"))
                    .unwrap_err();
            assert!(
                error.to_string().contains(if kind == "wrong" { "integrity" } else { "partial" }),
                "{error}"
            );
        }
    }

    #[test]
    fn malicious_lock_types_refuse_without_following_or_blocking() {
        for kind in ["symlink", "fifo", "directory"] {
            let root = tempfile::tempdir().unwrap();
            fs::create_dir_all(cache(root.path())).unwrap();
            let lock = cache(root.path()).join(".install.lock");
            match kind {
                "symlink" => symlink(root.path().join("missing"), &lock).unwrap(),
                "fifo" => assert!(Command::new("mkfifo").arg(&lock).status().unwrap().success()),
                _ => fs::create_dir(&lock).unwrap(),
            }
            assert!(
                install_with(root.path(), &pins(&payload()), |_| panic!("must not download"))
                    .is_err()
            );
            assert!(!root.path().join("missing").exists());
        }
    }

    #[test]
    fn download_failure_cleans_private_staging_and_never_publishes() {
        let root = tempfile::tempdir().unwrap();
        let error = install_with(root.path(), &pins(&payload()), |path| {
            assert_eq!(fs::metadata(path.parent().unwrap())?.permissions().mode() & 0o777, 0o700);
            assert!(!destination(root.path()).exists());
            let mut file = File::create(path)?;
            file.write_all(b"partial")?;
            Err(refuse("download fixture failed"))
        })
        .unwrap_err();
        assert!(error.to_string().contains("download fixture failed"));
        cleaned(root.path());
    }

    #[test]
    fn concurrent_installers_publish_once_and_return_same_path() {
        const CHILD_ROOT: &str = "UMPIRE_APALACHE_INSTALL_FIXTURE_ROOT";
        if let Some(root) = std::env::var_os(CHILD_ROOT) {
            let root = PathBuf::from(root);
            let payload = payload();
            let path = install_with(&root, &pins(&payload), |path| {
                let mut log =
                    OpenOptions::new().append(true).create(true).open(root.join("downloads"))?;
                log.write_all(b"download\n")?;
                fs::write(path, &payload)?;
                Ok(())
            })
            .unwrap();
            assert_eq!(path, destination(&root).join("lib/apalache.jar"));
            return;
        }
        let root = tempfile::tempdir().unwrap();
        let mut workers: Vec<_> = (0..4)
            .map(|_| {
                Command::new(std::env::current_exe().unwrap())
                    .args([
                        "--exact",
                        "installer::tests::concurrent_installers_publish_once_and_return_same_path",
                    ])
                    .env(CHILD_ROOT, root.path())
                    .stdout(std::process::Stdio::null())
                    .spawn()
                    .unwrap()
            })
            .collect();
        let deadline = std::time::Instant::now() + std::time::Duration::from_secs(10);
        for worker in &mut workers {
            loop {
                if let Some(status) = worker.try_wait().unwrap() {
                    assert!(status.success(), "{status}");
                    break;
                }
                if std::time::Instant::now() >= deadline {
                    for worker in &mut workers {
                        let _ = worker.kill();
                        let _ = worker.wait();
                    }
                    panic!("concurrent installers timed out");
                }
                std::thread::sleep(std::time::Duration::from_millis(10));
            }
        }
        assert_eq!(fs::read(root.path().join("downloads")).unwrap(), b"download\n");
        let path = destination(root.path()).join("lib/apalache.jar");
        assert_eq!(fs::read(&path).unwrap(), JAR);
        assert_eq!(fs::read_dir(cache(root.path())).unwrap().count(), 2);
        assert_eq!(fake(root.path(), &payload()).unwrap(), path);
    }

    #[test]
    fn native_curl_contract_and_download_failure_cleanup() {
        let root = tempfile::tempdir().unwrap();
        let bin = root.path().join("bin");
        fs::create_dir(&bin).unwrap();
        let curl = bin.join("curl");
        fs::write(&curl, b"#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FIXTURE_CURL_LOG\"\nexit 42\n")
            .unwrap();
        fs::set_permissions(&curl, fs::Permissions::from_mode(0o700)).unwrap();
        let log = root.path().join("curl-args");
        fs::create_dir(root.path().join("model")).unwrap();
        fs::write(root.path().join("model/Cargo.toml"), b"fixture repository marker").unwrap();
        for explicit in [false, true] {
            let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
            command
                .arg("fetch-apalache")
                .current_dir(root.path().join("model"))
                .env("PATH", &bin)
                .env("FIXTURE_CURL_LOG", &log);
            if explicit {
                command.arg(root.path());
            }
            let output = command.output().unwrap();
            assert_eq!(output.status.code(), Some(1));
            assert_eq!(output.stdout, [] as [u8; 0]);
            assert!(
                String::from_utf8(output.stderr)
                    .unwrap()
                    .starts_with("xtask: Apalache download failed:")
            );
            let arguments = fs::read_to_string(&log).unwrap();
            let arguments: Vec<_> = arguments.lines().collect();
            assert_eq!(
                &arguments[..11],
                [
                    "--proto",
                    "=https",
                    "--tlsv1.2",
                    "--fail",
                    "--location",
                    "--retry",
                    "2",
                    "--max-time",
                    "180",
                    &format!(
                        "https://github.com/apalache-mc/apalache/releases/download/v{VERSION}/apalache-{VERSION}.tgz"
                    ),
                    "-o"
                ]
            );
            let archive = Path::new(arguments[11]);
            assert!(archive.starts_with(cache(root.path())));
            assert_eq!(archive.file_name().unwrap(), "release.tgz");
            cleaned(root.path());
        }
    }

    #[test]
    fn native_dispatch_usage_failure_is_nonzero_and_structured() {
        let output = Command::new(env!("CARGO_BIN_EXE_xtask"))
            .args(["fetch-apalache", ".", "extra"])
            .output()
            .unwrap();
        assert_eq!(output.status.code(), Some(1));
        assert_eq!(output.stdout, [] as [u8; 0]);
        let error = String::from_utf8(output.stderr).unwrap();
        assert!(error.starts_with("xtask: usage: cargo xtask fetch-apalache"));
    }
}
