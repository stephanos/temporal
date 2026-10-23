use super::explore::{boolean, option, report_error, value};
use std::{io::Write, path::Path};

/// # Errors
/// Reports invalid resume requests and campaign recovery failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    run_with_installation(arguments, output, None)
}

/// # Errors
/// Refuses explicit installation changes before opening the campaign writer.
pub fn run_with_installation(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    let mut json = false;
    let result = (|| {
        let mut index = 0;
        let mut site_scope = None;
        while let Some((name, supplied)) = option(arguments, &mut index) {
            let name = name?;
            match name {
                "json" => json = boolean(supplied.unwrap_or("true"))?,
                "site-scope" => {
                    site_scope = Some(crate::campaign::exploration::site_scope::parse_patterns(
                        value(arguments, &mut index, name, supplied)?,
                    )?);
                }
                _ => return Err(format!("flag provided but not defined: -{name}")),
            }
        }
        if arguments.len() - index != 1 || arguments[index].is_empty() {
            return Err("resume requires one interrupted campaign path".into());
        }
        let path = Path::new(&arguments[index]);
        let compile_cache_root = installation_cache(path, installation)?;

        if crate::campaign::exploration::plan::is_round_campaign(path)? {
            use crate::campaign::exploration::round::{RoundJournal, RoundStrategy};
            let strategy = {
                let journal = RoundJournal::open(path, true)?;
                journal.plan().strategy.clone()
            };
            return match strategy {
                RoundStrategy::Choice(config) => {
                    validate_site_scope(
                        site_scope.as_deref(),
                        config
                            .site_scope
                            .as_ref()
                            .map(|scope| scope.patterns.as_slice()),
                    )?;
                    if compile_cache_root.is_some() {
                        crate::campaign::choice::resume_with_cache(
                            path,
                            json,
                            output,
                            compile_cache_root,
                        )
                    } else {
                        crate::campaign::choice::resume(path, json, output)
                    }
                }
                RoundStrategy::Simulation(_) => {
                    if site_scope.is_some() {
                        return Err("--site-scope requires a choice-exploration campaign".into());
                    }
                    if compile_cache_root.is_some() {
                        crate::campaign::simulation::resume_with_cache(
                            path,
                            json,
                            output,
                            compile_cache_root,
                        )
                    } else {
                        crate::campaign::simulation::resume(path, json, output)
                    }
                }
            };
        }
        if site_scope.is_some() {
            return Err("--site-scope requires a choice-exploration campaign".into());
        }
        if compile_cache_root.is_some() {
            crate::campaign::resume_with_cache(path, json, output, compile_cache_root)
        } else {
            crate::campaign::resume(path, json, output)
        }
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => report_error(&error, json, output),
    }
}

fn installation_cache(path: &Path, installation: Option<&Path>) -> Result<Option<String>, String> {
    let Some(selected) = installation else {
        return Ok(None);
    };
    let description = crate::install::manifest::resolve(Some(selected))?;
    let opened = if crate::campaign::exploration::plan::is_round_campaign(path)? {
        let journal = crate::campaign::exploration::round::RoundJournal::open(path, false)?;
        journal.open_artifact(
            journal.plan().prepared["prepared_artifact"]
                .as_str()
                .ok_or("campaign prepared artifact missing")?,
        )?
    } else {
        let journal = crate::campaign::ledger::Journal::open(path, false)?;
        journal.open_artifact(
            journal.plan()["prepared_artifact"]
                .as_str()
                .ok_or("campaign prepared artifact missing")?,
        )?
    };
    opened.authenticate_all()?;
    super::replay::validate_installation(&description.manifest, &opened)?;
    crate::campaign::prepare::compile_cache_root(&description.manifest)
}

fn validate_site_scope(
    requested: Option<&[String]>,
    persisted: Option<&[String]>,
) -> Result<(), String> {
    if requested.is_some_and(|patterns| Some(patterns) != persisted) {
        return Err("exploration plan identity or bounds changed".into());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn site_scope_resume_refuses_changed_bound() {
        let persisted = vec!["net/http".to_owned()];
        let changed = vec!["net/...".to_owned()];
        assert!(validate_site_scope(None, Some(&persisted)).is_ok());
        assert!(validate_site_scope(Some(&persisted), Some(&persisted)).is_ok());
        assert_eq!(
            validate_site_scope(Some(&changed), Some(&persisted)),
            Err("exploration plan identity or bounds changed".into())
        );
        assert_eq!(
            validate_site_scope(Some(&persisted), None),
            Err("exploration plan identity or bounds changed".into())
        );
    }

    #[test]
    fn site_scope_resume_refuses_malformed_pattern_before_open() {
        let arguments = vec![
            "--json".into(),
            "--site-scope=net/http,".into(),
            "/missing/campaign".into(),
        ];
        let mut output = Vec::new();
        assert!(matches!(run(&arguments, &mut output), Ok(2)));
        let event: serde_json::Value = serde_json::from_slice(&output).unwrap();
        assert_eq!(event["classification"], "invalid_input");
        let message = event["message"].as_str().unwrap();
        assert!(message.contains("site-scope"));
        assert!(message.contains("package pattern"));
    }
}
