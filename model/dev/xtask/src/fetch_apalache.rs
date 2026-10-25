// SHA-pinned Apalache admission, without extracting archive paths to disk.
use crate::{
    Error, Result,
    apalache_pins::{ARCHIVE_SHA256, JAR_SHA256, VERSION},
    support::hash,
};
use std::{
    fs::{self, File, OpenOptions},
    io::Read,
    path::{Path, PathBuf},
    process::Command,
};

struct Pins {
    version: &'static str,
    archive: String,
    jar: String,
}

/// Install the official pinned assembly, or validate an existing installation.
/// The root is the repository directory containing `model/`.
pub fn install(root: &Path) -> Result<PathBuf> {
    install_with(
        root,
        &Pins { version: VERSION, archive: ARCHIVE_SHA256.into(), jar: JAR_SHA256.into() },
        |archive| {
            let status = Command::new("curl").args(["--proto", "=https", "--tlsv1.2", "--fail", "--location", "--retry", "2", "--max-time", "180"])
            .arg(format!("https://github.com/apalache-mc/apalache/releases/download/v{VERSION}/apalache-{VERSION}.tgz"))
            .arg("-o").arg(archive).status()?;
            if !status.success() {
                return Err(Error::Invalid(format!("Apalache download failed: {status}")));
            }
            Ok(())
        },
    )
}

fn refuse(message: &str) -> Error {
    Error::Invalid(format!("Apalache {message}"))
}

fn directory(path: &Path) -> Result<()> {
    match fs::create_dir(path) {
        Ok(()) => (),
        Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => (),
        Err(error) => return Err(error.into()),
    }
    if !fs::symlink_metadata(path)?.is_dir() || fs::canonicalize(path)? != path {
        return Err(refuse("build root contains a symlink or escapes workspace"));
    }
    Ok(())
}

#[cfg(unix)]
fn lock(cache: &Path) -> Result<File> {
    use std::os::unix::fs::{MetadataExt, OpenOptionsExt};
    let path = cache.join(".install.lock");
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .truncate(false)
        .mode(0o600)
        .custom_flags(libc::O_NOFOLLOW | libc::O_NONBLOCK)
        .open(&path)?;
    let verify = || -> Result<()> {
        let opened = file.metadata()?;
        let named = fs::symlink_metadata(&path)?;
        if !opened.is_file()
            || !named.is_file()
            || opened.dev() != named.dev()
            || opened.ino() != named.ino()
        {
            return Err(refuse("lock is not a stable regular file"));
        }
        Ok(())
    };
    verify()?;
    file.lock()?;
    verify()?;
    Ok(file)
}

#[cfg(not(unix))]
fn lock(_: &Path) -> Result<File> {
    Err(refuse("installer requires Unix path admission"))
}

fn validate_paths(cache: &Path, destination: &Path) -> Result<()> {
    if fs::canonicalize(cache)? != cache {
        return Err(refuse("cache changed during admission"));
    }
    for path in
        [destination.to_owned(), destination.join("lib"), destination.join("lib/apalache.jar")]
    {
        match fs::symlink_metadata(path) {
            Ok(metadata) if metadata.is_symlink() => {
                return Err(refuse("installation is a symlink"));
            }
            Ok(_) => (),
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => (),
            Err(error) => return Err(error.into()),
        }
    }
    Ok(())
}

fn contents(payload: &[u8], pins: &Pins) -> Result<Vec<u8>> {
    if hash(payload) != pins.archive {
        return Err(refuse("archive integrity mismatch"));
    }
    let decoder = flate2::read::GzDecoder::new(payload);
    let mut archive = tar::Archive::new(decoder);
    let wanted = format!("apalache-{}/lib/apalache.jar", pins.version);
    for entry in archive.entries()? {
        let mut entry = entry?;
        if entry.path()?.as_ref() != Path::new(&wanted) {
            continue;
        }
        if !entry.header().entry_type().is_file() {
            return Err(refuse("jar is not a regular archive member"));
        }
        let mut bytes = Vec::new();
        entry.read_to_end(&mut bytes)?;
        if hash(&bytes) != pins.jar {
            return Err(refuse("extracted jar integrity mismatch"));
        }
        return Ok(bytes);
    }
    Err(refuse("archive is missing the jar member"))
}

fn install_with(
    root: &Path,
    pins: &Pins,
    download: impl FnOnce(&Path) -> Result<()>,
) -> Result<PathBuf> {
    let root = fs::canonicalize(root)?;
    let mut cache = root;
    for part in ["model", "build", "apalache"] {
        cache.push(part);
        directory(&cache)?;
    }
    let _lease = lock(&cache)?;
    let destination = cache.join(format!("apalache-{}", pins.version));
    let jar = destination.join("lib/apalache.jar");
    validate_paths(&cache, &destination)?;
    if jar.is_file() {
        if hash(&fs::read(&jar)?) != pins.jar {
            return Err(refuse("jar integrity mismatch"));
        }
        return Ok(jar);
    }
    if destination.try_exists()? {
        return Err(refuse("partial installation"));
    }
    let mut builder = tempfile::Builder::new();
    builder.prefix("install-");
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        builder.permissions(fs::Permissions::from_mode(0o700));
    }
    let staging = builder.tempdir_in(&cache)?;
    let result = (|| {
        let archive = staging.path().join("release.tgz");
        download(&archive)?;
        let bytes = contents(&fs::read(archive)?, pins)?;
        let private = staging.path().join("validated");
        fs::create_dir(&private)?;
        fs::create_dir(private.join("lib"))?;
        crate::support::publish(&private.join("lib/apalache.jar"), &bytes)?;
        validate_paths(&cache, &destination)?;
        if destination.try_exists()? {
            return Err(refuse("cache changed during admission"));
        }
        fs::rename(private, &destination)?;
        Ok(jar)
    })();
    let cleanup = staging.close();
    match (result, cleanup) {
        (Ok(path), Ok(())) => Ok(path),
        (Err(error), Ok(())) => Err(error),
        (_, Err(error)) => Err(refuse(&format!("staging cleanup failed: {error}"))),
    }
}
