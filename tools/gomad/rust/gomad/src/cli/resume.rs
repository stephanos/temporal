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
        crate::campaign::resume(Path::new(&arguments[index]), json, output)
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => report_error(error, json, output),
    }
}
