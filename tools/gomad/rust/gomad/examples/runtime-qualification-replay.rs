//! Internal qualification harness; build with the ordinary release library.
fn main() {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    let result = if arguments.len() == 1 && arguments[0] == "identity" {
        println!("{}", gomad::runtime_qualification::identity());
        Ok(())
    } else if arguments.len() == 5 && arguments[0] == "produce" {
        arguments[3]
            .to_string_lossy()
            .parse::<u64>()
            .map_err(|error| error.to_string())
            .and_then(|index| {
                gomad::runtime_qualification::produce(
                    std::path::Path::new(&arguments[1]),
                    std::path::Path::new(&arguments[2]),
                    index,
                    std::path::Path::new(&arguments[4]),
                )
            })
    } else if arguments.len() == 2 {
        gomad::runtime_qualification::replay(
            std::path::Path::new(&arguments[0]),
            std::path::Path::new(&arguments[1]),
        )
    } else {
        Err("usage: runtime-qualification-replay ARTIFACT NEW-OUTPUT-DIRECTORY".into())
    };
    if let Err(error) = result {
        eprintln!("{error}");
        std::process::exit(2);
    }
}
