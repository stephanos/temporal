use super::*;
use std::os::unix::fs::PermissionsExt;
use std::process::{Command, Stdio};
use std::sync::mpsc;

const PRODUCER: &str = "runtime-cost-profile";

struct Scratch(PathBuf);
impl Scratch {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!("gomad-retention-process-{}", unique()));
        std::fs::create_dir(&path).unwrap();
        Self(std::fs::canonicalize(path).unwrap())
    }
    fn begin(&self) -> RetainedRun {
        RetainedRun::begin(&self.0, PRODUCER, &BTreeMap::new(), false, &|| false).unwrap()
    }
    fn store(&self) -> PathBuf {
        self.0.join(".gomad-measure").join(PRODUCER)
    }
    fn latest(&self) -> PathBuf {
        let name: String =
            serde_json::from_slice(&std::fs::read(self.store().join("latest.json")).unwrap())
                .unwrap();
        self.store().join(name)
    }
    fn successful_run(&self) -> PathBuf {
        let run = self.begin();
        let path = run.path().to_owned();
        std::fs::write(path.join("result"), b"complete").unwrap();
        run.finish(true, &|| false).unwrap();
        path
    }
    fn named_directories(&self, prefix: &str) -> Vec<PathBuf> {
        std::fs::read_dir(self.store())
            .unwrap()
            .map(Result::unwrap)
            .filter(|entry| {
                entry.file_type().unwrap().is_dir()
                    && entry.file_name().to_str().unwrap().starts_with(prefix)
            })
            .map(|entry| entry.path())
            .collect()
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        fn writable_tree(path: &Path) {
            std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o700)).unwrap();
            for entry in std::fs::read_dir(path).unwrap() {
                let entry = entry.unwrap();
                if entry.file_type().unwrap().is_dir() {
                    writable_tree(&entry.path());
                }
            }
        }
        writable_tree(&self.0);
        std::fs::remove_dir_all(&self.0).unwrap();
    }
}

#[test]
fn interrupted_child() {
    let Some(root) = std::env::var_os("GOMAD_RETENTION_CRASH_ROOT") else {
        return;
    };
    let run = RetainedRun::begin(Path::new(&root), PRODUCER, &BTreeMap::new(), false, &|| {
        false
    })
    .unwrap();
    std::fs::write(run.path().join("partial"), b"interrupted").unwrap();
    std::process::exit(23);
}

#[test]
fn interrupted_run_recovery_preserves_latest_after_three_abrupt_exits() {
    let scratch = Scratch::new();
    let prior = scratch.successful_run();
    let prior_pointer = std::fs::read(scratch.store().join("latest.json")).unwrap();
    let module = module_path!().split_once("::").unwrap().1;
    for _ in 0..3 {
        let output = Command::new(std::env::current_exe().unwrap())
            .args([
                "--exact",
                &format!("{module}::interrupted_child"),
                "--nocapture",
            ])
            .env("GOMAD_RETENTION_CRASH_ROOT", &scratch.0)
            .stdin(Stdio::null())
            .output()
            .unwrap();
        assert_eq!(output.status.code(), Some(23), "{output:?}");
        assert_eq!(scratch.latest(), prior);
        assert_eq!(
            std::fs::read(scratch.store().join("latest.json")).unwrap(),
            prior_pointer
        );
        assert_eq!(std::fs::read(prior.join("result")).unwrap(), b"complete");
    }
    let latest = scratch.successful_run();
    assert_eq!(scratch.latest(), latest);
    assert!(!prior.exists());
    let attempts = scratch.named_directories("run-");
    assert_eq!(attempts.len(), 2, "{attempts:?}");
    let abandoned = attempts.iter().find(|path| **path != latest).unwrap();
    assert_eq!(
        std::fs::read(abandoned.join("partial")).unwrap(),
        b"interrupted"
    );
    assert!(!abandoned.join(".complete").exists());
    assert_eq!(scratch.named_directories(".stage-"), Vec::<PathBuf>::new());
    assert_eq!(scratch.named_directories(".trash-"), Vec::<PathBuf>::new());
}

#[test]
fn publication_waits_for_store_lock_then_publishes_the_completed_run() {
    let scratch = Scratch::new();
    let prior = scratch.successful_run();
    let run = scratch.begin();
    let path = run.path().to_owned();
    std::fs::write(path.join("result"), b"complete").unwrap();
    let held = store_lock(&run.store, &|| false).unwrap();
    let (send, receive) = mpsc::channel();
    let publisher = std::thread::spawn(move || send.send(run.finish(true, &|| false)).unwrap());
    let deadline = Instant::now() + Duration::from_secs(10);
    while !path.join(".complete").is_file() && Instant::now() < deadline {
        std::thread::sleep(Duration::from_millis(1));
    }
    let completed = path.join(".complete").is_file();
    let premature = receive.try_recv().ok();
    let latest_while_locked = scratch.latest();
    drop(held);
    let finished = premature.as_ref().map_or_else(
        || receive.recv_timeout(Duration::from_secs(10)).unwrap(),
        Clone::clone,
    );
    publisher.join().unwrap();
    assert!(
        completed,
        "run never completed before acquiring the publication lock"
    );
    assert!(
        premature.is_none(),
        "publication finished while store lock remained held: {premature:?}"
    );
    assert_eq!(latest_while_locked, prior);
    finished.unwrap();
    assert_eq!(scratch.latest(), path);
    assert_eq!(std::fs::read(path.join("result")).unwrap(), b"complete");
    assert!(!prior.exists());
}

#[test]
fn reclaim_permission_failure_keeps_new_latest_and_next_publication_recovers() {
    if rustix::process::geteuid().as_raw() == 0 {
        eprintln!(
            "SKIP: root ignores the directory permissions required for an actual reclamation failure"
        );
        return;
    }
    let scratch = Scratch::new();
    let first = scratch.begin();
    let prior = first.path().to_owned();
    let cache = prior.join("cache");
    std::fs::create_dir(&cache).unwrap();
    std::fs::write(cache.join("module.cwasm"), b"compiled").unwrap();
    std::fs::write(prior.join("result"), b"complete").unwrap();
    first.finish(true, &|| false).unwrap();
    std::fs::set_permissions(&cache, std::fs::Permissions::from_mode(0o500)).unwrap();
    assert_eq!(
        std::fs::remove_file(cache.join("module.cwasm"))
            .unwrap_err()
            .kind(),
        std::io::ErrorKind::PermissionDenied,
        "the fixture must cause an actual filesystem refusal"
    );
    let latest = scratch.successful_run();
    assert_eq!(scratch.latest(), latest);
    assert_eq!(std::fs::read(latest.join("result")).unwrap(), b"complete");
    assert!(
        !prior.exists(),
        "failed reclamation kept the old authenticated path"
    );
    let trash = scratch.named_directories(".trash-");
    assert_eq!(trash.len(), 1, "{trash:?}");
    assert_eq!(
        std::fs::read(trash[0].join("cache/module.cwasm")).unwrap(),
        b"compiled"
    );
    std::fs::set_permissions(
        trash[0].join("cache"),
        std::fs::Permissions::from_mode(0o700),
    )
    .unwrap();
    let recovered = scratch.successful_run();
    assert_eq!(scratch.latest(), recovered);
    assert_eq!(
        std::fs::read(recovered.join("result")).unwrap(),
        b"complete"
    );
    assert_eq!(scratch.named_directories(".trash-"), Vec::<PathBuf>::new());
    assert_eq!(scratch.named_directories("run-"), vec![recovered]);
}
