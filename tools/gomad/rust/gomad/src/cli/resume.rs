use super::explore::{boolean, option, report_error};
use std::{io::Write, path::Path};

/// # Errors
/// Reports invalid resume requests and campaign recovery failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let mut json = false;
    let result = (|| {
        let mut index = 0;
        while let Some((name, supplied)) = option(arguments, &mut index) {
            let name = name?;
            if name != "json" {
                return Err(format!("flag provided but not defined: -{name}"));
            }
            json = boolean(supplied.unwrap_or("true"))?;
        }
        if arguments.len() - index != 1 || arguments[index].is_empty() {
            return Err("resume requires one interrupted campaign path".into());
        }
        let path = Path::new(&arguments[index]);
        if crate::campaign::exploration::plan::is_round_campaign(path)? {
            use crate::campaign::exploration::round::{RoundJournal, RoundStrategy};
            let strategy = {
                let journal = RoundJournal::open(path, true)?;
                journal.plan().strategy.clone()
            };
            return match strategy {
                RoundStrategy::Choice(_) => crate::campaign::choice::resume(path, json, output),
                RoundStrategy::Simulation(_) => {
                    crate::campaign::simulation::resume(path, json, output)
                }
            };
        }
        crate::campaign::resume(path, json, output)
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => report_error(&error, json, output),
    }
}
