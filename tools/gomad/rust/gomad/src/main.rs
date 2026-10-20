fn main() {
    #[cfg(feature = "determinism-fixture")]
    if let Some(result) = gomad::fixture::dispatch(&std::env::args().skip(1).collect::<Vec<_>>()) {
        if let Err(error) = result {
            eprintln!("gomad: {}", error.message);
            std::process::exit(error.status);
        }
        return;
    }
    eprintln!("gomad: unsupported command");
    std::process::exit(2);
}
