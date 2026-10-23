fn main() {
    if cfg!(feature = "helper-diagnostic-profile") {
        eprintln!(
            "gomad: diagnostic builds only support the gomad-profile producer; qualification is refused"
        );
        std::process::exit(2);
    }
    #[cfg(not(feature = "determinism-fixture"))]
    if std::env::vars_os().any(|(name, _)| name.to_string_lossy().starts_with("GOMAD_FIXTURE_")) {
        eprintln!("gomad: unsupported fixture control");
        std::process::exit(2);
    }
    let mut arguments = std::env::args_os().skip(1).collect::<Vec<_>>();
    let installation = take_installation(&mut arguments);
    if arguments
        .first()
        .is_some_and(|argument| argument == "run-guest")
    {
        let status = gomad::cli::run_guest::run(&arguments[1..], installation.as_deref());
        std::process::exit(status);
    }
    let arguments = utf8_arguments(arguments);
    if let Some(command) = operator_command(&arguments) {
        let mut output = std::io::stdout().lock();
        let result = match command {
            "install" => gomad::cli::install::run(&arguments[1..], &mut output),
            "discover" => {
                gomad::cli::discover::run(&arguments[1..], &mut output, installation.as_deref())
            }
            "doctor" => {
                gomad::cli::doctor::run(&arguments[1..], &mut output, installation.as_deref())
            }
            "plan" => gomad::cli::plan::run(&arguments[1..], &mut output, installation.as_deref()),
            "recover" => gomad::cli::recover::run(&arguments[1..], &mut output),
            "explore" => gomad::cli::explore::run_with_installation(
                &arguments[1..],
                &mut output,
                installation.as_deref(),
            ),
            "resume" => gomad::cli::resume::run(&arguments[1..], &mut output),
            "minimize" => gomad::cli::minimize::run(&arguments[1..], &mut output),
            "stress" => gomad::cli::stress::run(&arguments[1..], &mut output),
            "qualify" => {
                gomad::cli::qualify::run(&arguments[1..], &mut output, installation.as_deref())
            }
            "report" => gomad::cli::qualify::report(&arguments[1..], &mut output),
            "execute-shard" => gomad::cli::merge::execute_shard(&arguments[1..], &mut output),
            _ => gomad::cli::merge::run(&arguments[1..], &mut output),
        };
        let status = match result {
            Ok(status) => status,
            Err(error) => {
                eprintln!("gomad: {}", error.message);
                error.status
            }
        };
        std::process::exit(status);
    }
    if arguments
        .first()
        .is_some_and(|argument| argument == "replay")
    {
        let status = match gomad::cli::replay::run(&arguments[1..], &mut std::io::stdout().lock()) {
            Ok(status) => status,
            Err(error) => {
                eprintln!("gomad: {}", error.message);
                error.status
            }
        };
        std::process::exit(status);
    }
    if arguments
        .first()
        .is_some_and(|argument| argument == "inspect")
    {
        if let Err(error) = gomad::cli::inspect::run(&arguments[1..], &mut std::io::stdout().lock())
        {
            eprintln!("gomad: {}", error.message);
            std::process::exit(error.status);
        }
        return;
    }
    #[cfg(feature = "determinism-fixture")]
    if let Some(result) = gomad::fixture::dispatch(&arguments) {
        if let Err(error) = result {
            eprintln!("gomad: {}", error.message);
            std::process::exit(error.status);
        }
        return;
    }
    eprintln!("gomad: unsupported command");
    std::process::exit(2);
}

fn operator_command(arguments: &[String]) -> Option<&str> {
    arguments.first().map(String::as_str).filter(|command| {
        matches!(
            *command,
            "explore"
                | "resume"
                | "execute-shard"
                | "merge"
                | "minimize"
                | "qualify"
                | "report"
                | "stress"
                | "install"
                | "discover"
                | "doctor"
                | "plan"
                | "recover"
        )
    })
}

fn utf8_arguments(arguments: Vec<std::ffi::OsString>) -> Vec<String> {
    arguments
        .into_iter()
        .map(|argument| {
            argument.into_string().unwrap_or_else(|_| {
                eprintln!("gomad: command arguments must be UTF-8");
                std::process::exit(2);
            })
        })
        .collect::<Vec<_>>()
}

fn take_installation(arguments: &mut Vec<std::ffi::OsString>) -> Option<std::path::PathBuf> {
    let mut installation = None;
    loop {
        if arguments
            .first()
            .is_some_and(|value| value == "--installation" || value == "-installation")
        {
            if arguments.len() < 2 {
                eprintln!("gomad: flag needs an argument: -installation");
                std::process::exit(2);
            }
            installation = Some(std::path::PathBuf::from(&arguments[1]));
            arguments.drain(..2);
        } else if let Some(value) = arguments.first().and_then(|value| {
            use std::os::unix::ffi::OsStrExt as _;
            value
                .as_os_str()
                .as_bytes()
                .strip_prefix(b"--installation=")
                .or_else(|| value.as_os_str().as_bytes().strip_prefix(b"-installation="))
                .map(std::ffi::OsStr::from_bytes)
        }) {
            installation = Some(std::path::PathBuf::from(value));
            arguments.remove(0);
        } else {
            break;
        }
    }
    installation
}
