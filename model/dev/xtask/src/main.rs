//! Native development command dispatcher.
use std::{
    env,
    ffi::OsStr,
    io::{self, Read, Write},
    path::Path,
};

fn run() -> xtask::Result<i32> {
    let mut arguments = env::args_os().skip(1);
    match arguments.next().as_deref() {
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
        _ => Err(xtask::Error::Invalid("commands: guest-lock, guest-rustc, retired-names".into())),
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
