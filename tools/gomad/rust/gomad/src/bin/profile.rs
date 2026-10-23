//! Explicit-input diagnostic producer; excluded from ordinary release builds.
use std::path::Path;

fn main() {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    let result = if arguments.len() == 2 && arguments[0] == "check" {
        std::fs::read(&arguments[1])
            .map_err(|error| error.to_string())
            .and_then(|raw| gomad::profile::decode(&raw).map(|_| ()))
    } else if (arguments.len() == 3 || arguments.len() == 4 && arguments[3] == "--keep")
        && arguments[0] == "retained"
    {
        gomad::profile::producer::run_retained(
            Path::new(&arguments[1]),
            Path::new(&arguments[2]),
            arguments.len() == 4,
        )
        .map(|_| ())
    } else if arguments.len() == 2 {
        gomad::profile::producer::run(Path::new(&arguments[0]), Path::new(&arguments[1]))
    } else {
        Err("usage: gomad-profile INPUT.json NEW-OUTPUT-DIRECTORY".into())
    };
    if let Err(error) = result {
        eprintln!("gomad-profile: {error}");
        std::process::exit(3);
    }
}
