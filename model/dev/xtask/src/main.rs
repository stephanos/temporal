//! Native development command dispatcher.
#![forbid(unsafe_code)]

use std::{
    env,
    ffi::OsStr,
    io::{self, Read, Write},
    path::Path,
};

fn run() -> xtask::Result<i32> {
    let mut argv = env::args_os();
    if argv.next().as_deref().and_then(|p| Path::new(p).file_name())
        == Some(OsStr::new("guest-rustc"))
    {
        return xtask::compiler::run(argv.collect());
    }
    let mut arguments = argv;
    match arguments.next().as_deref() {
        Some(command) if command == OsStr::new("fetch-apalache") => {
            let root = if let Some(root) = arguments.next() {
                std::path::PathBuf::from(root)
            } else {
                let current = env::current_dir()?;
                current
                    .ancestors()
                    .find(|path| path.join("model/Cargo.toml").is_file())
                    .unwrap_or(&current)
                    .to_owned()
            };
            if arguments.next().is_some() {
                return Err(xtask::Error::Invalid(
                    "usage: cargo xtask fetch-apalache [workspace-root]".into(),
                ));
            }
            let path = xtask::fetch_apalache::install(Path::new(&root))?;
            io::stdout().write_all(path.as_os_str().as_encoded_bytes())?;
            io::stdout().write_all(b"\n")?;
            Ok(0)
        }
        Some(command) if command == OsStr::new("guest") => {
            xtask::guest::run(&arguments.collect::<Vec<_>>(), false)
        }
        Some(command) if command == OsStr::new("guest-freshness") => {
            xtask::guest::run(&arguments.collect::<Vec<_>>(), true)
        }
        Some(command) if command == OsStr::new("guest-lock") => {
            let paths: Vec<_> = arguments.collect();
            if paths.len() != 2 {
                return Err(xtask::Error::Invalid("usage: cargo xtask guest-lock <Cargo.lock> <directory> (metadata JSON on stdin)".into()));
            }
            let mut metadata = Vec::new();
            io::stdin().read_to_end(&mut metadata)?;
            let path = xtask::lock::publish(&metadata, Path::new(&paths[0]), Path::new(&paths[1]))?;
            io::stdout().write_all(path.as_os_str().as_encoded_bytes())?;
            io::stdout().write_all(b"\n")?;
            Ok(0)
        }
        Some(command) if command == OsStr::new("guest-rustc") => {
            xtask::compiler::run(arguments.collect())
        }
        Some(command) if command == OsStr::new("retired-names") => {
            let pattern = arguments.next();
            if pattern.is_none() || arguments.next().is_some() {
                return Ok(2);
            }
            Ok(xtask::retired_names::run(
                pattern.as_deref().unwrap_or_default(),
                io::stdin().lock(),
                io::stdout().lock(),
            ))
        }
        _ => Err(xtask::Error::Invalid(
            "commands: fetch-apalache, guest, guest-freshness, guest-lock, guest-rustc, retired-names".into(),
        )),
    }
}

fn main() {
    let status = match run() {
        Ok(status) => status,
        Err(error) => {
            let _ = writeln!(io::stderr(), "xtask: {error}");
            1
        }
    };
    std::process::exit(status);
}
