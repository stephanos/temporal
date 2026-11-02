use std::{
    fs, io,
    path::PathBuf,
    sync::atomic::{AtomicU64, Ordering},
};

pub(crate) struct Scratch(pub(crate) PathBuf);
impl Scratch {
    pub(crate) fn new() -> io::Result<Self> {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "temporal native script controls {} {}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        fs::create_dir(&root)?;
        Ok(Self(root.canonicalize()?))
    }
}
impl Drop for Scratch {
    fn drop(&mut self) {
        if let Err(error) = fs::remove_dir_all(&self.0) {
            let _ = io::Write::write_fmt(
                &mut io::stderr(),
                format_args!("control scratch cleanup failed: {error}\n"),
            );
        }
    }
}

#[cfg(target_os = "macos")]
#[test]
fn filesystem_refuses_invalid_utf8_names() -> io::Result<()> {
    use std::{ffi::OsString, os::unix::ffi::OsStringExt};
    let scratch = Scratch::new()?;
    fs::write(scratch.0.join("valid-file"), b"control")?;
    fs::create_dir(scratch.0.join("valid-directory"))?;
    let raw = scratch.0.join(OsString::from_vec(b"invalid-\xff".to_vec()));
    for result in [fs::write(&raw, b"control"), fs::create_dir(&raw)] {
        assert!(result.is_err(), "invalid filename accepted");
    }
    Ok(())
}
