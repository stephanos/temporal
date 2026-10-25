use std::{env, fs, path::PathBuf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let source = "../../umpire/dev/fixtures/fixtures/sample_kettle/model.rs";
    println!("cargo:rerun-if-changed={source}");
    let authored = fs::read_to_string(source)?;
    let baseline = "const BOILING_TEMPERATURE: i32 = 100;";
    if authored.matches(baseline).count() != 1 {
        return Err("shared Kettle must declare exactly one boiling threshold of 100".into());
    }
    let scaled = authored.replace(baseline, "const BOILING_TEMPERATURE: i32 = 1_000_000;");
    // include! supplies the module documentation context at the call site.
    let scaled =
        scaled.strip_prefix("//! Model authored with the attribute surface.\n").unwrap_or(&scaled);
    fs::write(
        PathBuf::from(env::var_os("OUT_DIR").ok_or("missing OUT_DIR")?).join("kettle_big.rs"),
        scaled,
    )?;
    Ok(())
}
