use crate::{evidence::canonical, install::manifest};
use serde_json::json;
use std::{
    io::Write,
    path::{Path, PathBuf},
};

/// # Errors
/// Preserves the unavailable default doctor and installation-selected WASM refusal.
pub fn run(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    let (selected, source) = manifest::selected(installation)?;
    if source != "executable-sibling" || std::fs::symlink_metadata(selected).is_ok() {
        return Err("unsupported command: doctor".into());
    }
    let mut json_output = false;
    let mut artifacts = PathBuf::from(".gomad/artifacts");
    let mut index = 0;
    while index < arguments.len() {
        let flag = &arguments[index];
        if flag == "--" && index + 1 == arguments.len() {
            break;
        }
        let name = flag
            .strip_prefix("--")
            .or_else(|| flag.strip_prefix('-'))
            .ok_or("doctor accepts no positional arguments")?;
        let (name, value) = name
            .split_once('=')
            .map_or((name, None), |(name, value)| (name, Some(value)));
        match name {
            "json" => {
                json_output = match value {
                    None | Some("true" | "1" | "t" | "TRUE" | "True" | "T") => true,
                    Some("false" | "0" | "f" | "FALSE" | "False" | "F") => false,
                    _ => return Err("invalid value for --json".into()),
                }
            }
            "artifacts" => {
                let value = if let Some(value) = value {
                    value
                } else {
                    index += 1;
                    arguments
                        .get(index)
                        .ok_or("flag needs an argument: -artifacts")?
                };
                artifacts = PathBuf::from(value);
            }
            _ => return Err(format!("flag provided but not defined: -{name}").into()),
        }
        index += 1;
    }
    let os = match std::env::consts::OS {
        "macos" => "darwin",
        os => os,
    };
    let arch = match std::env::consts::ARCH {
        "aarch64" => "arm64",
        "x86_64" => "amd64",
        arch => arch,
    };
    let host = format!("{os}/{arch}");
    let repair = "use gomad discover to inspect a WASM installation";
    if json_output {
        let report = json!({"schema":"gomad.doctor/v3","available":false,"host":host,"supported_platforms":[],"boundary_manifest_version":"","io_inventory_sha256":"","io_implementation_sha256":"","adapters":[],"installation_source":"","toolchain_root":"","artifact_directory":std::path::absolute(artifacts).map_err(|e|e.to_string())?,"repair_instruction":repair,"checks":[]});
        output
            .write_all(&canonical::encode(&report)?)
            .and_then(|()| output.write_all(b"\n"))
            .map_err(|e| super::Error {
                status: 3,
                message: e.to_string(),
            })?;
    } else {
        writeln!(output,"gomad doctor: available=false host={host} go= toolchain= runner= boundary=\ninstallation: source= toolchain=\nrepair: {repair}").map_err(|e|super::Error{status:3,message:e.to_string()})?;
    }
    Ok(2)
}
