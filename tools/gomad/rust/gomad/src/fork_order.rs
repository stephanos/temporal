//! Orders child creation against advisory-lock release within this process.
//!
//! A `flock` belongs to the open file description, not the descriptor. A child
//! forked by another thread shares every description until its exec closes the
//! close-on-exec descriptors, so a lock this process drops inside that window
//! stays held: collection then skips a released run, or a lease that was let go
//! still looks busy (`EWOULDBLOCK`). The same window holds a freshly written
//! executable open for writing, so executing it fails with `ETXTBSY`.
//!
//! Every child is created inside [`spawning`], which holds the shared side until
//! `Command::spawn` returns; std returns only after the child has exec'd (or
//! failed to), so no inherited copy outlives the call. Every lock is released
//! through [`Locked`], which takes the exclusive side, so a release waits for
//! in-flight spawns and no later spawn can copy the released description.
use std::{
    fs::File,
    ops::Deref,
    sync::{PoisonError, RwLock},
};
#[cfg(test)]
use std::{
    io,
    process::{Child, Command, ExitStatus, Output, Stdio},
};

static ORDER: RwLock<()> = RwLock::new(());

/// Runs `create`, which creates children, once no lock release is in
/// progress; releases wait until it returns. `create` must return as soon as
/// the children exist (`Command::spawn`), or for short-lived children only.
pub fn spawning<T>(create: impl FnOnce() -> T) -> T {
    let _spawning = ORDER.read().unwrap_or_else(PoisonError::into_inner);
    create()
}

/// `Command::spawn` through [`spawning`].
///
/// # Errors
/// Returns the spawn error.
#[cfg(test)]
pub(crate) fn spawn(command: &mut Command) -> io::Result<Child> {
    spawning(|| command.spawn())
}

/// `Command::status` with only the spawn ordered; the child inherits stdio.
///
/// # Errors
/// Returns spawn or wait errors.
#[cfg(test)]
pub(crate) fn status(command: &mut Command) -> io::Result<ExitStatus> {
    spawn(command)?.wait()
}

/// `Command::output` with only the spawn ordered: stdin is null, stdout and
/// stderr are captured.
///
/// # Errors
/// Returns spawn or wait errors.
#[cfg(test)]
pub(crate) fn output(command: &mut Command) -> io::Result<Output> {
    spawn(
        command
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped()),
    )?
    .wait_with_output()
}

/// A file whose advisory lock is released only when no child creation is in
/// flight. Holding a `Locked` is holding the lock.
#[derive(Debug)]
pub struct Locked(Option<File>);

impl Locked {
    /// Takes ownership of a file whose lock is held (or about to be).
    #[must_use]
    pub fn new(file: File) -> Self {
        Self(Some(file))
    }
}

impl Deref for Locked {
    type Target = File;
    fn deref(&self) -> &File {
        self.0.as_ref().expect("lock file is present until drop")
    }
}

impl Drop for Locked {
    fn drop(&mut self) {
        let _releasing = ORDER.write().unwrap_or_else(PoisonError::into_inner);
        drop(self.0.take());
    }
}

/// Makes a written test fixture executable on a fresh inode whose only write
/// descriptor was open while no child could be created. The original write
/// may have been copied into a concurrently forked child, which holds it open
/// until its exec; executing that inode in the window fails with `ETXTBSY`.
#[cfg(test)]
pub(crate) fn make_executable(path: &std::path::Path, mode: u32) {
    use std::{io::Write as _, os::unix::fs::OpenOptionsExt as _};
    let contents = std::fs::read(path).unwrap();
    let mut staged = path.as_os_str().to_owned();
    staged.push(".executable");
    let staged = std::path::PathBuf::from(staged);
    {
        let _writing = ORDER.write().unwrap_or_else(PoisonError::into_inner);
        let mut file = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(mode)
            .open(&staged)
            .unwrap();
        file.write_all(&contents).unwrap();
    }
    std::fs::set_permissions(&staged, std::os::unix::fs::PermissionsExt::from_mode(mode)).unwrap();
    std::fs::rename(&staged, path).unwrap();
}
