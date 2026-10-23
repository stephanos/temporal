use super::{directory, publish_bytes};
use std::{
    fs::{self, OpenOptions},
    io::Write,
    os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt, symlink},
    path::PathBuf,
    sync::{
        Arc, Barrier,
        atomic::{AtomicBool, Ordering},
    },
    thread,
};

struct Root(PathBuf);

impl Root {
    fn new() -> Self {
        let path = std::env::temp_dir().join(format!(
            "gomad-qualification-publication-{}-{}",
            std::process::id(),
            super::STAGING.fetch_add(1, Ordering::Relaxed)
        ));
        fs::DirBuilder::new().mode(0o700).create(&path).unwrap();
        Self(path)
    }
}

impl Drop for Root {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.0).unwrap();
    }
}

fn private(path: &std::path::Path, bytes: &[u8]) {
    let mut file = OpenOptions::new().create_new(true).write(true).mode(0o600).open(path).unwrap();
    file.write_all(bytes).unwrap();
}

#[test]
fn result_publication_is_complete_concurrent_private_and_no_clobber() {
    let root = Root::new();
    let (path, pinned) = directory(&root.0).unwrap();
    let pinned = Arc::new(pinned);
    let data = Arc::new(b"measured result\n".repeat(16384));
    let barrier = Arc::new(Barrier::new(9));
    let mut workers = Vec::new();
    for _ in 0..8 {
        let (pinned, data, barrier) = (Arc::clone(&pinned), Arc::clone(&data), Arc::clone(&barrier));
        workers.push(thread::spawn(move || {
            barrier.wait();
            publish_bytes(&pinned, "result.json", &data, data.len() as u64).unwrap();
        }));
    }
    let done = Arc::new(AtomicBool::new(false));
    let observer_done = Arc::clone(&done);
    let observed = path.join("result.json");
    let expected = Arc::clone(&data);
    let observer = thread::spawn(move || {
        while !observer_done.load(Ordering::Acquire) {
            match fs::read(&observed) {
                Ok(actual) => assert_eq!(actual, *expected),
                Err(error) => assert_eq!(error.kind(), std::io::ErrorKind::NotFound),
            }
            thread::yield_now();
        }
    });
    barrier.wait();
    for worker in workers {
        worker.join().unwrap();
    }
    done.store(true, Ordering::Release);
    observer.join().unwrap();
    assert_eq!(fs::read(path.join("result.json")).unwrap(), *data);
    assert_eq!(fs::metadata(path.join("result.json")).unwrap().permissions().mode() & 0o777, 0o600);
    assert_eq!(fs::read_dir(path).unwrap().count(), 1);
}

#[test]
fn result_publication_preserves_partial_conflicts_and_symlinks() {
    for redirected in [false, true] {
        let root = Root::new();
        let (path, pinned) = directory(&root.0).unwrap();
        let outside = root.0.join("outside");
        if redirected {
            private(&outside, b"complete");
            symlink(&outside, path.join("result.json")).unwrap();
        } else {
            private(&path.join("result.json"), b"par");
        }
        assert!(publish_bytes(&pinned, "result.json", b"complete", 8).is_err());
        assert_eq!(fs::symlink_metadata(path.join("result.json")).unwrap().file_type().is_symlink(), redirected);
        assert_eq!(fs::read(path.join("result.json")).unwrap(), if redirected { b"complete".as_slice() } else { b"par".as_slice() });
    }
}

#[test]
fn result_publication_refuses_persistent_foreign_hardlink() {
    let root = Root::new();
    let (path, pinned) = directory(&root.0).unwrap();
    private(&path.join("result.json"), b"complete");
    fs::hard_link(path.join("result.json"), root.0.join("foreign")).unwrap();
    assert!(publish_bytes(&pinned, "result.json", b"complete", 8).is_err());
    assert_eq!(fs::read(root.0.join("foreign")).unwrap(), b"complete");
    assert_eq!(fs::read_dir(path).unwrap().count(), 1);
}

#[test]
fn result_publication_preserves_abandoned_foreign_staging() {
    let root = Root::new();
    let (path, pinned) = directory(&root.0).unwrap();
    private(&path.join(".foreign.partial"), b"interrupted");
    publish_bytes(&pinned, "result.json", b"complete", 8).unwrap();
    assert_eq!(fs::read(path.join(".foreign.partial")).unwrap(), b"interrupted");
    assert_eq!(fs::read(path.join("result.json")).unwrap(), b"complete");
    assert_eq!(fs::read_dir(path).unwrap().count(), 2);
}

#[test]
fn result_publication_rejects_redirected_namespace_before_writing() {
    let root = Root::new();
    let outside = Root::new();
    symlink(&outside.0, root.0.join("rust-qualifications")).unwrap();
    assert!(directory(&root.0).is_err());
    assert_eq!(fs::read_dir(&outside.0).unwrap().count(), 0);
}
