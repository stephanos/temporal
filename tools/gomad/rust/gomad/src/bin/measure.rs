use gomad::measurement;
use std::{io::Write, path::Path};

fn main() {
    let arguments: Vec<String> = std::env::args().skip(1).collect();
    let code = if arguments == ["--identity"] {
        match measurement::identity().and_then(|value| {
            writeln!(std::io::stdout(), "{value}").map_err(|error| error.to_string())
        }) {
            Ok(()) => 0,
            Err(error) => {
                eprintln!("gomad-measure: {error}");
                2
            }
        }
    } else if arguments.first().is_some_and(|arg| arg == "stress")
        && arguments
            .get(1)
            .is_some_and(|arg| matches!(arg.as_str(), "--spec" | "--cell" | "--rerun"))
    {
        match measurement::stress_run(&arguments[1..], &mut std::io::stdout()) {
            Ok(code) => code,
            Err(error) => {
                eprintln!("gomad-measure: {error}");
                1
            }
        }
    } else if arguments.len() == 4 && arguments[0] == "--linked-run-admitted" {
        finish(measurement::execution::run_admitted(
            Path::new(&arguments[1]),
            Path::new(&arguments[2]),
            Path::new(&arguments[3]),
        ))
    } else if arguments.len() == 3 && arguments[0] == "--linked-run" {
        finish(measurement::execution::run(
            Path::new(&arguments[1]),
            Path::new(&arguments[2]),
        ))
    } else if (arguments.len() == 3 || arguments.len() == 4 && arguments[3] == "--keep")
        && arguments[0] == "--retained-linked-run"
    {
        finish(
            measurement::execution::run_retained(
                Path::new(&arguments[1]),
                Path::new(&arguments[2]),
                arguments.len() == 4,
            )
            .map(|_| ()),
        )
    } else {
        let environ: Vec<_> = std::env::vars()
            .map(|(name, value)| format!("{name}={value}"))
            .collect();
        measurement::inputs::run(
            &arguments,
            &environ,
            &mut std::io::stdout(),
            &mut std::io::stderr(),
            |call, stdout, stderr| {
                measurement::inputs::execute(call, stdout, stderr, measurement::dispatch)
            },
        )
    };
    std::process::exit(code);
}
fn finish(result: Result<(), String>) -> i32 {
    match result {
        Ok(()) => 0,
        Err(error) => {
            eprintln!("gomad-measure: {error}");
            1
        }
    }
}
