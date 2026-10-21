fn main() {
    #[cfg(not(feature = "determinism-fixture"))]
    if std::env::var_os("GOMAD_FIXTURE_GUEST_START_FILE").is_some() {
        eprintln!("gomad: unsupported fixture control");
        std::process::exit(2);
    }
    let arguments = std::env::args().skip(1).collect::<Vec<_>>();
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
