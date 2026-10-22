use crate::{evidence::canonical, install::manifest};
use serde_json::json;
use std::{io::Write, path::Path};

/// # Errors
/// Refuses invalid command grammar, incompatible selected installations and output errors.
pub fn run(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    if !arguments.is_empty() {
        return Err("discover accepts no command arguments".into());
    }
    let description =
        manifest::resolve(installation).map_err(|message| super::Error { status: 3, message })?;
    let cooperative = description.manifest.profile == "gomad.wasi-cooperative/v1";
    let mut response = serde_json::to_value(description).map_err(|error| error.to_string())?;
    response["capabilities"] = json!({"Name":"wasm", "Commands":["install","discover","plan","explore","replay","resume","recover","inspect","minimize","qualify","report"],
        "ReplayModes":if cooperative {vec!["exact","observed"]} else {vec!["observed"]},
        "ChoiceProfiles":if cooperative {json!(["gomad-choice-trace/v4"])} else {json!(null)},"TypedScenarios":cooperative,"Simulation":false,"ReadOnlyMounts":false,"IOTranscript":false});
    output
        .write_all(
            &canonical::encode(&response).map_err(|message| super::Error { status: 3, message })?,
        )
        .and_then(|()| output.write_all(b"\n"))
        .map_err(|error| super::Error {
            status: 3,
            message: error.to_string(),
        })?;
    Ok(0)
}
