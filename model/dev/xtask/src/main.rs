//! Native development command dispatcher.
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
        _ => Err(xtask::Error::Invalid(
            "commands: guest, guest-freshness, guest-lock, guest-rustc".into(),
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
