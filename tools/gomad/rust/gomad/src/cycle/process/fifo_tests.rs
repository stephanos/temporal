use super::{Registration, registry::Registry};
use rustix::fs::{Mode, OFlags};
use std::{
    collections::BTreeMap,
    fs::File,
    path::PathBuf,
    sync::{Arc, mpsc},
    time::{Duration, SystemTime, UNIX_EPOCH},
};

struct Fixture(Arc<Registry>);
impl Fixture {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "cycle-process-fifo-{}-{}",
            std::process::id(),
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        Self(Arc::new(Registry::create(&path).unwrap()))
    }
    fn replace_with_fifo(&self, name: &str) -> PathBuf {
        let path = self.0.path.join(name);
        std::fs::remove_file(&path).unwrap();
        rustix::fs::mkfifoat(rustix::fs::CWD, &path, Mode::from_raw_mode(0o600)).unwrap();
        path
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.0.path).unwrap();
    }
}

fn bounded_fifo_refusal(
    path: PathBuf,
    operation: impl FnOnce() -> Result<(), String> + Send + 'static,
) {
    let (sender, receiver) = mpsc::channel();
    let worker = std::thread::spawn(move || sender.send(operation()).unwrap());
    let timely = receiver.recv_timeout(Duration::from_secs(1));
    let returned_without_writer = timely.is_ok();
    let result = match timely {
        Ok(result) => result,
        Err(mpsc::RecvTimeoutError::Timeout) => {
            let rescue = File::from(
                rustix::fs::open(
                    &path,
                    OFlags::RDWR | OFlags::NONBLOCK | OFlags::NOFOLLOW | OFlags::CLOEXEC,
                    Mode::empty(),
                )
                .unwrap(),
            );
            drop(rescue);
            receiver.recv_timeout(Duration::from_secs(2)).unwrap()
        }
        Err(error) => panic!("FIFO refusal worker disconnected: {error}"),
    };
    worker.join().unwrap();
    assert!(
        returned_without_writer,
        "named FIFO reopen blocked until rescue writer opened"
    );
    assert!(
        result.is_err(),
        "named FIFO was accepted as a regular registry file"
    );
}

#[test]
fn fifo_stream_replacements_are_refused_without_a_writer() {
    for name in ["command.stdout", "command.stderr"] {
        let fixture = Fixture::new();
        drop(fixture.0.create_file(name).unwrap());
        let path = fixture.replace_with_fifo(name);
        let registry = Arc::clone(&fixture.0);
        bounded_fifo_refusal(path, move || registry.open_read(name).map(drop));
    }
}

#[test]
fn fifo_registration_replacement_is_refused_without_a_writer() {
    let fixture = Fixture::new();
    let registration = Registration {
        name: "command".into(),
        interval: None,
        pid: std::process::id(),
        parent_pid: std::process::id(),
        stage_pid: std::process::id(),
        active: true,
        supervisor_pid: std::process::id(),
        supervisor_parent_pid: std::process::id(),
        start_ns: 1,
        clock: "CLOCK_MONOTONIC",
        argv: vec!["/bin/true".into()],
        cwd: "/".into(),
        environment: BTreeMap::new(),
        temporary_paths: BTreeMap::new(),
        effective_environment: None,
        execution_environment: BTreeMap::new(),
        environment_mode: "exact",
        private_context: None,
        started_monotonic_nanos: 1,
    };
    fixture.0.register(&registration).unwrap();
    let path = fixture.replace_with_fifo("command.process.json");
    let registry = Arc::clone(&fixture.0);
    bounded_fifo_refusal(path, move || registry.verify(&registration));
}
