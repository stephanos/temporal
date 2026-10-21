fn main() {
    #[cfg(not(feature = "determinism-fixture"))]
    if std::env::vars_os().any(|(name, _)| name.to_string_lossy().starts_with("GOMAD_FIXTURE_")) {
        eprintln!("gomad: unsupported fixture control");
        std::process::exit(2);
    }
    let mut arguments = std::env::args().skip(1).collect::<Vec<_>>();
    let mut installation = None;
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
        value
            .strip_prefix("--installation=")
            .or_else(|| value.strip_prefix("-installation="))
    }) {
        installation = Some(std::path::PathBuf::from(value));
        arguments.remove(0);
    }
    if let Some(command) = arguments
        .first()
        .map(String::as_str)
        .filter(|command| matches!(*command, "explore" | "resume" | "execute-shard" | "merge"))
    {
        let mut output = std::io::stdout().lock();
        let result = match command {
            "explore" => gomad::cli::explore::run_with_installation(
                &arguments[1..],
                &mut output,
                installation.as_deref(),
            ),
            "resume" => gomad::cli::resume::run(&arguments[1..], &mut output),
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
