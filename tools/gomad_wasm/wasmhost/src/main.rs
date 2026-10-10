mod abi;
mod engine;
mod protocol;

fn main() {
    if let Err(error) = engine::execute() {
        eprintln!("wasmhost transport: {error}");
        std::process::exit(1);
    }
}
