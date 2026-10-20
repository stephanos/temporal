fn main() {
    let arguments = std::env::args().skip(1).collect::<Vec<_>>();
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
