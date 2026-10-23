fn main() {
    if let Err(error) = gomad_wasmhost::transport() {
        #[cfg(feature = "conformance")]
        eprintln!("wasmhost conformance transport: {error}");
        #[cfg(not(feature = "conformance"))]
        eprintln!("wasmhost transport: {error}");
        std::process::exit(1);
    }
}
