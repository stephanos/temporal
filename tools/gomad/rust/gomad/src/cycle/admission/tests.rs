use super::*;
use std::sync::atomic::{AtomicU64, Ordering};

static NEXT: AtomicU64 = AtomicU64::new(0);
struct Fixture {
    root: PathBuf,
    bindings: Bindings,
}
impl Fixture {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "gomad-admission-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&root).unwrap();
        let root = fs::canonicalize(root).unwrap();
        for name in ["repo", "evidence", "host", "overlay"] {
            fs::create_dir(root.join(name)).unwrap();
        }
        let scratch = root.join("scratch");
        let bindings = Bindings {
            repo: root.join("repo"),
            evidence: root.join("evidence"),
            scratch: scratch.clone(),
            temporary_paths: BTreeMap::from([
                ("TMPDIR".into(), scratch.join("tmp")),
                ("GOTMPDIR".into(), scratch.join("go")),
            ]),
            environment: BTreeMap::new(),
            config_sha256: "authenticated-fixture-config".into(),
            host: root.join("host"),
            overlay: root.join("overlay"),
        };
        Self { root, bindings }
    }
    fn prepare(&self) -> Admission {
        Admission::prepare(self.bindings.clone(), &Facts::default()).unwrap()
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}
#[derive(Debug)]
struct Facts {
    change: Option<(PathBuf, Filesystem)>,
    fail: bool,
}
impl Default for Facts {
    fn default() -> Self {
        Self {
            change: None,
            fail: false,
        }
    }
}
impl Probe for Facts {
    fn filesystem(&self, path: &Path) -> Result<Filesystem, String> {
        if self.fail {
            return Err("fixture probe failed".into());
        }
        Ok(self.change.as_ref().filter(|(p, _)| p == path).map_or(
            Filesystem {
                kind: "overlayfs".into(),
                available_bytes: 64 * GIB,
            },
            |(_, f)| f.clone(),
        ))
    }
}
fn refused<T: std::fmt::Debug>(result: Result<T, String>, expected: &str) -> String {
    let error = result.unwrap_err();
    assert!(error.contains(expected), "expected {expected}, got {error}");
    error
}
fn proof(id: &str, control: impl std::fmt::Debug, input: impl std::fmt::Debug, error: &str) {
    let control = format!("{control:?}");
    let input = format!("{input:?}");
    println!(
        "PROOF_JSON {}",
        serde_json::json!({"id":id,"control":control,"input":input,"control_sha256":crate::evidence::canonical::digest(control.as_bytes()),"input_sha256":crate::evidence::canonical::digest(input.as_bytes()),"control_result":"accepted","result":"refused","failure":error})
    );
}

#[test]
fn host_bindings_original_19_21_23_24_26_29_33_35_37() {
    for id in [19, 21, 23, 24, 26, 29, 33, 35, 37] {
        let f = Fixture::new();
        f.bindings.validate().unwrap();
        let mut b = f.bindings.clone();
        let message = match id {
            19 => {
                b.evidence = PathBuf::from("relative");
                "evidence root"
            }
            21 => {
                b.scratch = PathBuf::from("relative");
                "scratch must"
            }
            23 => {
                b.scratch = f.root.join("absent/child");
                "scratch parent"
            }
            24 => {
                b.scratch = b.repo.join("scratch");
                "candidate source"
            }
            26 => {
                b.scratch = b.evidence.join("scratch");
                "evidence root"
            }
            29 => {
                b.temporary_paths.remove("TMPDIR");
                "explicit TMPDIR"
            }
            33 => {
                b.temporary_paths
                    .insert("TMPDIR".into(), f.root.join("elsewhere"));
                "direct canonical"
            }
            35 => {
                b.environment.insert("TMPDIR".into(), "conflict".into());
                "conflicting"
            }
            37 => {
                b.temporary_paths
                    .insert("TMPDIR".into(), b.temporary_paths["GOTMPDIR"].clone());
                "distinct"
            }
            _ => unreachable!(),
        };
        let error = refused(b.validate(), message);
        proof(
            &format!("orchestration:host_admission:{id}"),
            &f.bindings,
            &b,
            &error,
        );
    }
    let f = Fixture::new();
    std::os::unix::fs::symlink(&f.bindings.evidence, f.root.join("link")).unwrap();
    let mut b = f.bindings.clone();
    b.evidence = f.root.join("link");
    refused(b.validate(), "evidence root");
    let mut b = f.bindings.clone();
    b.scratch = f.root.join("repo/../scratch");
    refused(b.validate(), "scratch must");
}

#[test]
fn host_initial_and_recheck_original_68_71_73() {
    let f = Fixture::new();
    let a = f.prepare();
    a.recheck(&Facts::default()).unwrap();
    let error = refused(
        Admission::prepare(f.bindings.clone(), &Facts::default()),
        "fresh",
    );
    proof(
        "orchestration:host_admission:68",
        (&f.bindings, "scratch absent"),
        (&f.bindings, "scratch exists"),
        &error,
    );
    let error = refused(
        a.recheck_config("changed", &Facts::default()),
        "authenticated initial",
    );
    proof(
        "orchestration:host_admission:71",
        &f.bindings.config_sha256,
        "changed",
        &error,
    );
    fs::remove_dir(&f.bindings.temporary_paths["TMPDIR"]).unwrap();
    let error = refused(a.recheck(&Facts::default()), "disappeared");
    proof(
        "orchestration:host_admission:73",
        (&f.bindings.temporary_paths, "all exist"),
        (&f.bindings.temporary_paths, "TMPDIR removed"),
        &error,
    );
}

#[test]
fn host_filesystem_original_48_79_81_90() {
    for (id, kind, bytes, fail, message) in [
        (48, "ext4\nspoof", 64 * GIB, false, "unrecognized"),
        (79, "virtiofs", 64 * GIB, false, "scratch filesystem"),
        (79, "fuseblk", 64 * GIB, false, "scratch filesystem"),
        (81, "ext4", 12 * GIB - 1, false, "free byte floor"),
        (90, "ext4", 64 * GIB, true, "probe failed"),
    ] {
        let f = Fixture::new();
        f.bindings.validate().unwrap();
        let facts = Facts {
            change: Some((
                f.root.clone(),
                Filesystem {
                    kind: kind.into(),
                    available_bytes: bytes,
                },
            )),
            fail,
        };
        let error = refused(Admission::prepare(f.bindings.clone(), &facts), message);
        assert!(!f.bindings.scratch.exists());
        f.prepare().recheck(&Facts::default()).unwrap();
        proof(
            &format!("orchestration:host_admission:{id}"),
            Facts::default(),
            &facts,
            &error,
        );
    }
    let f = Fixture::new();
    let facts = Facts {
        change: Some((
            f.bindings.evidence.clone(),
            Filesystem {
                kind: "overlayfs".into(),
                available_bytes: 12 * GIB - 1,
            },
        )),
        fail: false,
    };
    refused(
        Admission::prepare(f.bindings.clone(), &facts),
        "free byte floor",
    );
}

#[test]
fn reservation_resource_floors_exact_boundary_and_one_byte_below() {
    let f = Fixture::new();
    let a = f.prepare();
    for (path, floor) in [
        (&f.bindings.host, 45 * GIB),
        (&f.bindings.overlay, 20 * GIB),
        (&f.root, 12 * GIB),
    ] {
        let facts = |available_bytes| Facts {
            change: Some((
                path.clone(),
                Filesystem {
                    kind: "overlayfs".into(),
                    available_bytes,
                },
            )),
            fail: false,
        };
        a.recheck(&facts(floor)).unwrap();
        refused(a.recheck(&facts(floor - 1)), "free byte floor");
    }
    refused(
        a.recheck(&Facts {
            change: None,
            fail: true,
        }),
        "probe failed",
    );
    let not_overlay = Facts {
        change: Some((
            f.root.clone(),
            Filesystem {
                kind: "ext4".into(),
                available_bytes: 64 * GIB,
            },
        )),
        fail: false,
    };
    refused(a.recheck(&not_overlay), "actual overlay scratch");
    let temporary = &f.bindings.temporary_paths["TMPDIR"];
    fs::rename(temporary, f.bindings.scratch.join("original-tmp")).unwrap();
    fs::create_dir(temporary).unwrap();
    refused(a.recheck(&Facts::default()), "directory identity changed");
}

#[test]
fn memory_original_166_176_179_187_194_198_201_203() {
    let valid = "MemTotal: 20 kB\nMemAvailable: 1024 kB\n";
    for (id, threshold, platform, raw, message) in [
        (166, "+1", "linux", valid.to_owned(), "bounded unsigned"),
        (176, "0", "linux", valid.to_owned(), "positive decimal"),
        (179, "1", "darwin", valid.to_owned(), "requires Linux"),
        (187, "1", "linux", "x".repeat(8193), "exceeds 8KiB"),
        (
            194,
            "1",
            "linux",
            "MemAvailable: 1024 MB\n".into(),
            "unique exact MemAvailable",
        ),
        (
            198,
            "1",
            "linux",
            "MemAvailable: 18446744073709551615 kB\n".into(),
            "byte conversion overflow",
        ),
        (
            201,
            "1",
            "linux",
            "MemTotal: 1024 kB\n".into(),
            "MemAvailable missing",
        ),
        (
            203,
            "1048577",
            "linux",
            valid.to_owned(),
            "insufficient available",
        ),
    ] {
        assert_eq!(
            admit_memory("1048576", "linux", valid.as_bytes()).unwrap(),
            1048576
        );
        let error = refused(admit_memory(threshold, platform, raw.as_bytes()), message);
        proof(
            &format!(
                "workload-callback:producer:raise:{id}:{}",
                if [194, 198].contains(&id) { 12 } else { 8 }
            ),
            ("1048576", "linux", valid),
            (threshold, platform, raw),
            &error,
        );
    }
    for value in [
        "",
        "-1",
        "1.0",
        "18446744073709551616",
        "000000000000000000000",
    ] {
        assert!(unsigned_decimal(value).is_err());
    }
    for raw in [
        "MemAvailable: 1 kB\nMemAvailable: 1 kB\n",
        "MemAvailable: +1 kB\n",
        "MemAvailable: 18446744073709551616 kB\n",
        "MemAvailable : 1 kB\n",
        "MemAvailable: 1 kB\nmemavailable: 1 kB\n",
        "MemAvailable: 1 kB\nMemAvailable : 1 kB\n",
    ] {
        assert!(admit_memory("1", "linux", raw.as_bytes()).is_err());
    }
    struct Broken;
    impl Read for Broken {
        fn read(&mut self, _: &mut [u8]) -> std::io::Result<usize> {
            Err(std::io::Error::other("fixture I/O error"))
        }
    }
    refused(admit_memory("1", "linux", Broken), "fixture I/O error");
}

#[test]
fn serving_filesystems_require_overlay_at_preparation() {
    let host = Fixture::new();
    let host_facts = Facts {
        change: Some((
            host.bindings.host.clone(),
            Filesystem {
                kind: "ext4".into(),
                available_bytes: 64 * GIB,
            },
        )),
        fail: false,
    };
    Admission::prepare(host.bindings.clone(), &host_facts)
        .unwrap()
        .recheck(&host_facts)
        .unwrap();
    println!("SERVING_FS_HOST_CONTROL ext4 preparation=accepted reservation=accepted");
    let mut incorrectly_accepted = Vec::new();
    for role in ["evidence", "declared overlay"] {
        for kind in ["fuse", "ext4"] {
            let f = Fixture::new();
            let control = f.prepare();
            control.recheck(&Facts::default()).unwrap();
            drop(control);
            for path in f.bindings.temporary_paths.values() {
                fs::remove_dir(path).unwrap();
            }
            fs::remove_dir(&f.bindings.scratch).unwrap();
            let path = if role == "evidence" {
                &f.bindings.evidence
            } else {
                &f.bindings.overlay
            };
            let facts = Facts {
                change: Some((
                    path.clone(),
                    Filesystem {
                        kind: kind.into(),
                        available_bytes: 64 * GIB,
                    },
                )),
                fail: false,
            };
            let expected = format!("actual overlay {role} is required");
            match Admission::prepare(f.bindings.clone(), &facts) {
                Err(error) => {
                    assert_eq!(error, expected);
                    assert!(!f.bindings.scratch.exists());
                    proof(
                        &format!("servingfs:prepare:{role}:{kind}"),
                        (&f.bindings, Facts::default()),
                        (&f.bindings, &facts),
                        &error,
                    );
                }
                Ok(_) => {
                    println!("SERVING_FS_BAD_ACCEPTED preparation role={role} kind={kind}");
                    incorrectly_accepted.push((role, kind));
                }
            }
        }
    }
    assert!(
        incorrectly_accepted.is_empty(),
        "forbidden preparation filesystems accepted: {incorrectly_accepted:?}"
    );
}
