use super::*;
use std::sync::atomic::{AtomicU64, Ordering};
static NEXT: AtomicU64 = AtomicU64::new(0);
struct Fixture {
    root: PathBuf,
    authority: Authority,
}
impl Fixture {
    fn new() -> Self {
        let root = std::env::temp_dir().join(format!(
            "cycle-journal-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        std::fs::create_dir(&root).unwrap();
        let root = root.canonicalize().unwrap();
        std::fs::create_dir(root.join("outputs")).unwrap();
        Self {
            authority: Authority {
                binding: root.join("binding.json"),
            },
            root,
        }
    }
    fn enroll(&self) -> Journal {
        self.authority
            .initial_enroll(&self.root.join("journal"), &self.root.join("outputs"))
            .unwrap()
    }
    fn output(&self, name: &str) -> PathBuf {
        let p = self.root.join("outputs").join(name);
        std::fs::create_dir(&p).unwrap();
        p
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}
fn digest() -> String {
    "a".repeat(64)
}
fn refusal<T>(result: Result<T, String>, expected: &str) {
    let message = result.err().expect("expected intended refusal");
    assert!(
        message.contains(expected),
        "expected {expected}, got {message}"
    );
    println!("JOURNAL_REFUSAL {expected}: {message}");
}

#[test]
fn enrollment_is_explicit_and_missing_binding_never_recreates_history() {
    let f = Fixture::new();
    assert!(f.authority.open().is_err());
    assert!(!f.root.join("journal").exists());
    let mut journal = f.enroll();
    assert_eq!(journal.charged().unwrap(), 0);
    journal
        .reserve(&digest(), &f.output("first"), || Ok(()))
        .unwrap();
    drop(journal);
    let before = std::fs::read(f.root.join("journal")).unwrap();
    std::fs::remove_file(&f.authority.binding).unwrap();
    assert!(f.authority.open().is_err());
    std::fs::create_dir(f.root.join("fresh-outputs")).unwrap();
    assert!(
        f.authority
            .initial_enroll(&f.root.join("fresh"), &f.root.join("fresh-outputs"))
            .is_err()
    );
    assert_eq!(before, std::fs::read(f.root.join("journal")).unwrap());
    assert!(!f.root.join("fresh").exists());
}

#[test]
fn retained_outputs_and_partial_initial_enrollment_refuse() {
    let f = Fixture::new();
    std::fs::write(
        f.output("retained").join(".cycle-attempt-started"),
        "started",
    )
    .unwrap();
    refusal(
        f.authority
            .initial_enroll(&f.root.join("journal"), &f.root.join("outputs")),
        "retained output",
    );
    assert!(!f.authority.sentinel().exists());
    let g = Fixture::new();
    std::fs::write(g.root.join("journal"), "retained highwater").unwrap();
    assert!(
        g.authority
            .initial_enroll(&g.root.join("journal"), &g.root.join("outputs"))
            .is_err()
    );
    assert!(g.authority.sentinel().exists());
    std::fs::remove_file(g.root.join("journal")).unwrap();
    assert!(
        g.authority
            .initial_enroll(&g.root.join("new"), &g.root.join("outputs"))
            .is_err()
    );
}

#[test]
fn stable_lock_survives_binding_publication_and_concurrent_initial_contenders() {
    let f = Fixture::new();
    let barrier = std::sync::Barrier::new(2);
    let wins = std::thread::scope(|scope| {
        let handles: Vec<_> = (0..2)
            .map(|_| {
                scope.spawn(|| {
                    barrier.wait();
                    f.authority
                        .initial_enroll(&f.root.join("journal"), &f.root.join("outputs"))
                        .is_ok()
                })
            })
            .collect();
        handles
            .into_iter()
            .map(|h| usize::from(h.join().unwrap()))
            .sum::<usize>()
    });
    assert_eq!(wins, 1);
    let mut journal = f.authority.open().unwrap();
    let inode = journal.file.metadata().unwrap().ino();
    for i in 0..2 {
        journal
            .reserve(&digest(), &f.output(&format!("attempt{i}")), || Ok(()))
            .unwrap();
        assert_eq!(journal.file.metadata().unwrap().ino(), inode);
        refusal(f.authority.open(), "locked");
    }
    drop(journal);
    let mut reopened = f.authority.open().unwrap();
    assert_eq!(reopened.charged().unwrap(), 2);
    refusal(
        reopened.reserve(&"b".repeat(64), &f.output("third-candidate-output"), || {
            Ok(())
        }),
        "allowance exhausted",
    );
}

#[test]
fn corruption_and_crash_gaps_fail_closed_after_independent_reopen() {
    for mutation in [
        "policy",
        "path",
        "inode",
        "digest",
        "count",
        "missing-journal",
        "symlink-journal",
        "symlink-binding",
        "symlink-sentinel",
        "crash-gap",
        "torn-tail",
        "invalid-sequence",
        "sentinel-policy",
        "sentinel-path",
        "sentinel-inode",
        "missing-sentinel",
        "refund-started",
    ] {
        let f = Fixture::new();
        let mut journal = f.enroll();
        journal
            .reserve(&digest(), &f.output("first"), || Ok(()))
            .unwrap();
        let mut binding = journal.binding.clone();
        drop(journal);
        let journal_path = f.root.join("journal");
        match mutation {
            "policy" => binding.enrollment.policy = "foreign".into(),
            "path" => binding.enrollment.journal = f.root.join("foreign"),
            "inode" => binding.journal_inode += 1,
            "digest" => binding.digest = "0".repeat(64),
            "count" => binding.charged = 0,
            "missing-journal" => std::fs::remove_file(&journal_path).unwrap(),
            "symlink-journal" => {
                std::fs::rename(&journal_path, f.root.join("saved")).unwrap();
                std::os::unix::fs::symlink(f.root.join("saved"), &journal_path).unwrap();
            }
            "symlink-binding" => {
                std::fs::rename(&f.authority.binding, f.root.join("saved")).unwrap();
                std::os::unix::fs::symlink(f.root.join("saved"), &f.authority.binding).unwrap();
            }
            "symlink-sentinel" => {
                let p = f.authority.sentinel();
                std::fs::rename(&p, f.root.join("saved")).unwrap();
                std::os::unix::fs::symlink(f.root.join("saved"), p).unwrap();
            }
            "crash-gap" => {
                write(
                    &open(&journal_path, false, true).unwrap(),
                    &encoded(&Event::Reserve {
                        id: 2,
                        candidate: digest(),
                        output: f.output("unpublished"),
                    })
                    .unwrap(),
                )
                .unwrap();
            }
            "torn-tail" => {
                write(&open(&journal_path, false, true).unwrap(), b"{\"event\":").unwrap();
            }
            "invalid-sequence" => {
                let b = encoded(&Event::Started {
                    id: 77,
                    pid: 1,
                    supervisor: 2,
                })
                .unwrap();
                write(&open(&journal_path, false, true).unwrap(), &b).unwrap();
                binding.digest = canonical::digest(&std::fs::read(&journal_path).unwrap());
                binding.events += 1;
            }
            "sentinel-policy" | "sentinel-path" => {
                let mut enrollment = binding.enrollment.clone();
                if mutation == "sentinel-policy" {
                    enrollment.policy = "foreign".into();
                } else {
                    enrollment.journal = f.root.join("foreign-history");
                }
                std::fs::write(
                    f.authority.sentinel(),
                    canonical::encode(&enrollment).unwrap(),
                )
                .unwrap();
            }
            "sentinel-inode" => {
                let path = f.authority.sentinel();
                let bytes = std::fs::read(&path).unwrap();
                std::fs::rename(&path, f.root.join("saved-sentinel")).unwrap();
                write(&open(&path, true, false).unwrap(), &bytes).unwrap();
            }
            "missing-sentinel" => std::fs::remove_file(f.authority.sentinel()).unwrap(),
            "refund-started" => {
                let file = open(&journal_path, false, true).unwrap();
                write(
                    &file,
                    &encoded(&Event::Started {
                        id: 1,
                        pid: 1,
                        supervisor: 2,
                    })
                    .unwrap(),
                )
                .unwrap();
                write(&file, &encoded(&Event::NoChild { id: 1 }).unwrap()).unwrap();
                binding.digest = canonical::digest(&std::fs::read(&journal_path).unwrap());
                binding.events += 2;
                binding.charged = 0;
            }
            _ => unreachable!(),
        }
        if matches!(
            mutation,
            "policy"
                | "path"
                | "inode"
                | "digest"
                | "count"
                | "invalid-sequence"
                | "refund-started"
        ) {
            std::fs::write(&f.authority.binding, canonical::encode(&binding).unwrap()).unwrap();
        }
        assert!(
            f.authority.open().is_err(),
            "mutation {mutation} incorrectly reopened"
        );
        println!("JOURNAL_MUTATION {mutation} refused");
        assert!(
            f.authority
                .initial_enroll(&f.root.join("new-history"), &f.root.join("outputs"))
                .is_err()
        );
        assert!(!f.root.join("new-history").exists());
    }
}

#[test]
fn live_replacement_and_floor_failure_never_charge() {
    let f = Fixture::new();
    let mut journal = f.enroll();
    let original = std::fs::read(f.root.join("journal")).unwrap();
    refusal(
        journal.reserve(&digest(), &f.output("floor"), || {
            Err("HOST floor lowered".into())
        }),
        "HOST floor",
    );
    assert_eq!(journal.charged().unwrap(), 0);
    assert_eq!(original, std::fs::read(f.root.join("journal")).unwrap());
    std::fs::rename(f.root.join("journal"), f.root.join("saved")).unwrap();
    let replacement = open(&f.root.join("journal"), true, false).unwrap();
    write(&replacement, &original).unwrap();
    refusal(
        journal.reserve(&digest(), &f.output("replaced"), || Ok(())),
        "inode changed",
    );
    assert_eq!(original, std::fs::read(f.root.join("saved")).unwrap());
}

pub(crate) fn owner_cases(owner: &super::super::process::CycleOwner) {
    use super::super::{
        admission::{Admission, Bindings, Filesystem, Probe},
        process::Environment,
    };
    use std::{collections::BTreeMap, time::Duration};
    struct Facts;
    impl Probe for Facts {
        fn filesystem(&self, _: &Path) -> Result<Filesystem, String> {
            Ok(Filesystem {
                kind: "overlayfs".into(),
                available_bytes: 64 << 30,
            })
        }
    }
    let f = Fixture::new();
    for name in ["repo", "evidence", "host", "overlay"] {
        std::fs::create_dir(f.root.join(name)).unwrap();
    }
    let scratch = f.root.join("scratch");
    let admission = Admission::prepare(
        Bindings {
            repo: f.root.join("repo"),
            evidence: f.root.join("evidence"),
            scratch: scratch.clone(),
            temporary_paths: BTreeMap::from([
                ("TMPDIR".into(), scratch.join("tmp")),
                ("GOTMPDIR".into(), scratch.join("go")),
            ]),
            environment: BTreeMap::new(),
            config_sha256: digest(),
            host: f.root.join("host"),
            overlay: f.root.join("overlay"),
        },
        &Facts,
    )
    .unwrap();
    let command = || Command {
        argv: vec!["/bin/sleep".into(), "2".into()],
        cwd: "/".into(),
        environment: Environment::Exact(BTreeMap::new()),
        declared_environment: BTreeMap::new(),
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: Duration::from_secs(3),
        private_context: None,
    };
    let mut runner = Runner::create(owner, &f.root.join("registry"), None).unwrap();
    let mut journal = f.enroll();
    let cancel = AtomicBool::new(true);
    refusal(
        journal.start_inner(
            &mut runner,
            "authenticated-nochild",
            command(),
            &cancel,
            &digest(),
            &f.output("nochild"),
            || admission.recheck(&Facts),
            |_| Ok(()),
        ),
        "NoChild",
    );
    assert_eq!(journal.charged().unwrap(), 0);
    drop(journal);
    let mut seen = Vec::new();
    for (i, candidate) in [digest(), "b".repeat(64)].iter().enumerate() {
        let mut reopened = f.authority.open().unwrap();
        let mut actual = None;
        refusal(
            reopened.start_inner(
                &mut runner,
                &format!("interrupted-{i}"),
                command(),
                &AtomicBool::new(false),
                candidate,
                &f.output(&format!("candidate-{i}")),
                || admission.recheck(&Facts),
                |registration| {
                    assert!(Path::new(&format!("/proc/{}", registration.pid)).exists());
                    assert!(Path::new(&format!("/proc/{}", registration.supervisor_pid)).exists());
                    actual = Some((registration.pid, registration.supervisor_pid));
                    Err("fixture interruption before journal PID publication".into())
                },
            ),
            "Created",
        );
        let (pid, supervisor) = actual.unwrap();
        for p in [pid, supervisor] {
            assert!(
                !Path::new(&format!("/proc/{p}")).exists(),
                "fixture child survived {p}"
            );
            println!("JOURNAL_PID {p} CLEANED");
            seen.push(p);
        }
        assert_eq!(usize::from(reopened.charged().unwrap()), i + 1);
        assert!(
            !std::fs::read_to_string(f.root.join("journal"))
                .unwrap()
                .contains("\"event\":\"Started\"")
        );
    }
    let mut reopened = f.authority.open().unwrap();
    refusal(
        reopened.start_inner(
            &mut runner,
            "third",
            command(),
            &AtomicBool::new(false),
            &"c".repeat(64),
            &f.output("third"),
            || admission.recheck(&Facts),
            |_| panic!("third child admitted"),
        ),
        "allowance exhausted",
    );
    assert_eq!(reopened.charged().unwrap(), 2);
    drop(reopened);
    let mut child = crate::fork_order::spawn(
        std::process::Command::new(std::env::current_exe().unwrap())
            .arg("--reopen")
            .arg(&f.authority.binding),
    )
    .unwrap();
    let restart_pid = child.id();
    assert!(child.wait().unwrap().success());
    assert!(!Path::new(&format!("/proc/{restart_pid}")).exists());
    println!("JOURNAL_PID {restart_pid} CLEANED");
    for p in seen {
        assert!(!Path::new(&format!("/proc/{p}")).exists());
    }
    println!(
        "POLICY policy-ambiguous-spawn CONTROL=two-real-gated-forks REFUSAL=third REOPEN=independent CLEANUP=confirmed"
    );
}

pub(crate) fn restart_case(binding: PathBuf) {
    let authority = Authority { binding };
    let mut journal = authority.open().unwrap();
    assert_eq!(journal.charged().unwrap(), 2);
    refusal(
        journal.reserve(
            &"c".repeat(64),
            Path::new("/unused-third-output"),
            || Ok(()),
        ),
        "allowance exhausted",
    );
    println!(
        "JOURNAL_RESTART charged=2 third=refused pid={}",
        std::process::id()
    );
}

pub(crate) fn floor_cases(owner: &super::super::process::CycleOwner) {
    use super::super::{
        admission::{Admission, Bindings, Filesystem, Probe},
        process::Environment,
    };
    use std::{collections::BTreeMap, time::Duration};
    let f = Fixture::new();
    for name in ["repo", "host", "overlay"] {
        std::fs::create_dir(f.root.join(name)).unwrap();
    }
    struct Facts {
        root: PathBuf,
        change: Option<(PathBuf, u64)>,
        fail: bool,
    }
    impl Probe for Facts {
        fn filesystem(&self, p: &Path) -> Result<Filesystem, String> {
            if self.fail {
                return Err("reservation probe failure".into());
            }
            let bytes = if p == self.root.join("host") {
                45 << 30
            } else if p == self.root.join("overlay") {
                20 << 30
            } else {
                12 << 30
            };
            Ok(Filesystem {
                kind: "overlayfs".into(),
                available_bytes: self
                    .change
                    .as_ref()
                    .filter(|(path, _)| path == p)
                    .map_or(bytes, |(_, v)| *v),
            })
        }
    }
    let facts = Facts {
        root: f.root.clone(),
        change: None,
        fail: false,
    };
    let mut runner = Runner::create(owner, &f.root.join("registry"), None).unwrap();
    let mut journal = f.enroll();
    let output = f.output("admitted");
    let scratch = f.root.join("scratch");
    let admission = Admission::prepare(
        Bindings {
            repo: f.root.join("repo"),
            evidence: output.clone(),
            scratch: scratch.clone(),
            temporary_paths: BTreeMap::from([
                ("TMPDIR".into(), scratch.join("tmp")),
                ("GOTMPDIR".into(), scratch.join("go")),
            ]),
            environment: BTreeMap::new(),
            config_sha256: digest(),
            host: f.root.join("host"),
            overlay: f.root.join("overlay"),
        },
        &facts,
    )
    .unwrap();
    let command = || Command {
        argv: vec!["/bin/true".into()],
        cwd: "/".into(),
        environment: Environment::Exact(BTreeMap::new()),
        declared_environment: BTreeMap::new(),
        temporary_paths: BTreeMap::new(),
        accepted: vec![0],
        timeout: Duration::from_secs(3),
        private_context: None,
    };
    for (path, floor) in [
        (f.root.join("host"), 45u64 << 30),
        (f.root.join("overlay"), 20 << 30),
        (f.root.clone(), 12 << 30),
    ] {
        let below = Facts {
            root: f.root.clone(),
            change: Some((path.clone(), floor - 1)),
            fail: false,
        };
        refusal(
            journal.start(
                &mut runner,
                "floor",
                command(),
                &AtomicBool::new(false),
                &digest(),
                &output,
                &admission,
                &below,
            ),
            "free byte floor",
        );
        assert_eq!(journal.charged().unwrap(), 0);
        assert!(!output.join(".cycle-attempt-started").exists());
        assert!(
            std::fs::read_dir(f.root.join("registry"))
                .unwrap()
                .next()
                .is_none()
        );
        println!(
            "POLICY policy-reservation-resource-floors path={} below={} charged=0 spawn=0",
            path.display(),
            floor - 1
        );
    }
    let broken = Facts {
        root: f.root.clone(),
        change: None,
        fail: true,
    };
    refusal(
        journal.start(
            &mut runner,
            "probe",
            command(),
            &AtomicBool::new(false),
            &digest(),
            &output,
            &admission,
            &broken,
        ),
        "probe failure",
    );
    assert_eq!(journal.charged().unwrap(), 0);
    let foreign = f.output("foreign");
    refusal(
        journal.start(
            &mut runner,
            "foreign",
            command(),
            &AtomicBool::new(false),
            &digest(),
            &foreign,
            &admission,
            &facts,
        ),
        "differs from admitted",
    );
    assert_eq!(journal.charged().unwrap(), 0);
    let done = journal
        .start(
            &mut runner,
            "exact",
            command(),
            &AtomicBool::new(false),
            &digest(),
            &output,
            &admission,
            &facts,
        )
        .unwrap()
        .wait(&AtomicBool::new(false))
        .unwrap();
    assert!(done.accepted());
    assert_eq!(journal.charged().unwrap(), 1);
    for pid in [done.pid, done.supervisor_pid] {
        assert!(!Path::new(&format!("/proc/{pid}")).exists());
        println!("JOURNAL_PID {pid} CLEANED");
    }
    let bytes = std::fs::read_to_string(f.root.join("journal")).unwrap();
    assert!(bytes.contains("\"event\":\"Started\""));
    drop(journal);
    assert_eq!(f.authority.open().unwrap().charged().unwrap(), 1);
    println!(
        "POLICY policy-reservation-resource-floors exact=45/20/12GiB executed-exit=0 charged=1 reopened=1"
    );
}
