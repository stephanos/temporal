use super::explore::{boolean, option, report_error, value};
use std::{
    io::Write,
    path::{Path, PathBuf},
};

/// # Errors
/// Reports malformed merge commands and shard validation failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let mut json = false;
    let result = (|| {
        let mut index = 0;
        let mut partial = false;
        let mut destination = "";
        while let Some((name, supplied)) = option(arguments, &mut index) {
            let name = name?;
            match name {
                "json" => json = boolean(supplied.unwrap_or("true"))?,
                "partial" => partial = boolean(supplied.unwrap_or("true"))?,
                "output" => destination = value(arguments, &mut index, name, supplied)?,
                _ => return Err(format!("flag provided but not defined: -{name}")),
            }
        }
        if destination.is_empty() || arguments.len() - index < 2 {
            return Err("usage: gomad merge --output DIR [--partial] PLAN SHARD...".into());
        }
        let shards: Vec<PathBuf> = arguments[index + 1..].iter().map(PathBuf::from).collect();
        crate::campaign::merge(
            Path::new(&arguments[index]),
            &shards,
            Path::new(destination),
            partial,
            json,
            output,
        )
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => report_error(error, json, output),
    }
}

/// # Errors
/// Reports malformed assignments and campaign shard execution failures.
pub fn execute_shard(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let mut json = false;
    let result = (|| {
        let mut index = 0;
        let mut assignment = "";
        let mut artifacts = ".gomad/artifacts";
        while let Some((name, supplied)) = option(arguments, &mut index) {
            let name = name?;
            match name {
                "json" => json = boolean(supplied.unwrap_or("true"))?,
                "shard" => assignment = value(arguments, &mut index, name, supplied)?,
                "artifacts" => artifacts = value(arguments, &mut index, name, supplied)?,
                _ => return Err(format!("flag provided but not defined: -{name}")),
            }
        }
        if arguments.len() - index != 1 {
            return Err("usage: gomad execute-shard --shard INDEX/COUNT PLAN".into());
        }
        let invalid = || format!("invalid shard {assignment:?}: want zero-based INDEX/COUNT");
        let (shard, count) = assignment.split_once('/').ok_or_else(invalid)?;
        let parse = |text: &str| -> Result<u64, String> {
            let value = text.parse::<u64>().map_err(|_| invalid())?;
            if value.to_string() != text {
                return Err(invalid());
            }
            Ok(value)
        };
        let shard = parse(shard)?;
        let count = parse(count)?;
        if count == 0 || shard >= count {
            return Err(
                "campaign shard requires a positive count and index less than count".into(),
            );
        }
        crate::campaign::execute_shard(
            Path::new(&arguments[index]),
            shard,
            count,
            Path::new(artifacts),
            json,
            output,
        )
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => report_error(error, json, output),
    }
}
