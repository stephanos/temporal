fn main() {
    if let Err(error) = gomad::support_tool(
        &std::env::args().skip(1).collect::<Vec<_>>(),
        &mut std::io::stdout().lock(),
    ) {
        eprintln!("fn222-support: {error}");
        std::process::exit(1);
    }
}
