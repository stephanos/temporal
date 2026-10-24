use super::*;

pub(super) fn resolve(
    compiler: &Path,
    directory: &Path,
    environment: &BTreeMap<String, String>,
    module_root: &Path,
    deadline: Instant,
) -> Result<PathBuf, String> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Cache {
        #[serde(rename = "GOCACHE")]
        root: PathBuf,
    }
    let scratch = process::Temporary::new(module_root)?;
    let mut query_environment = environment.clone();
    query_environment.remove("GOFLAGS");
    query_environment.insert(
        "GOTMPDIR".into(),
        module_root
            .to_str()
            .ok_or("Go temporary path is not UTF-8")?
            .into(),
    );
    let raw = process::command(
        compiler,
        &["env".into(), "-json".into(), "GOCACHE".into()],
        directory,
        &query_environment,
        &scratch.0,
        16384,
        deadline,
    )?;
    let cache: Cache = serde_json::from_slice(&raw).map_err(failure)?;
    if !cache.root.is_absolute() {
        return Err(failure("Go executable cache directory must be absolute"));
    }
    Ok(cache.root)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    fn compiler(root: &Path, response: &str) -> PathBuf {
        let path = root.join("pinned-go");
        fs::write(&path, format!("#!/bin/sh\nprintf '%s\\n' \"$@\" \"${{GOFLAGS-unset}}\" \"$GOTOOLCHAIN\" \"$SENTINEL\" \"$GOTMPDIR\" > '{}'\nprintf '%s' '{}'\n", root.join("query").display(), response)).unwrap();
        crate::fork_order::make_executable(&path, 0o700);
        path
    }

    #[test]
    fn query_preserves_driver_environment_and_allows_a_not_yet_created_cache() {
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        let cache = root.0.join("not-created");
        let compiler = compiler(&root.0, &serde_json::json!({"GOCACHE":cache}).to_string());
        let environment = BTreeMap::from([
            ("GOFLAGS".into(), "-race -tags=caller".into()),
            ("GOTOOLCHAIN".into(), "local".into()),
            ("SENTINEL".into(), "kept".into()),
            ("GOTMPDIR".into(), "original".into()),
        ]);
        let original = environment.clone();
        assert_eq!(
            resolve(
                &compiler,
                &root.0,
                &environment,
                &root.0,
                Instant::now() + Duration::from_secs(5)
            )
            .unwrap(),
            cache
        );
        assert!(!cache.exists());
        assert_eq!(environment, original);
        assert_eq!(
            fs::read_to_string(root.0.join("query")).unwrap(),
            format!(
                "env\n-json\nGOCACHE\nunset\nlocal\nkept\n{}\n",
                root.0.display()
            )
        );
    }

    #[test]
    fn query_uses_the_original_deadline_before_spawning() {
        let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
        let compiler = compiler(&root.0, "{}");
        let error = resolve(
            &compiler,
            &root.0,
            &BTreeMap::new(),
            &root.0,
            Instant::now(),
        )
        .unwrap_err();
        assert!(error.contains("overall_timeout"));
        assert!(!root.0.join("query").exists());
    }

    #[test]
    fn query_refuses_malformed_relative_unknown_and_excessive_responses() {
        for response in [
            "not json".to_owned(),
            "{\"GOCACHE\":\"relative\"}".into(),
            "{\"GOCACHE\":\"/cache\",\"extra\":1}".into(),
            "x".repeat(16385),
        ] {
            let root = process::Temporary::new(&std::env::temp_dir()).unwrap();
            let compiler = compiler(&root.0, &response);
            assert!(
                resolve(
                    &compiler,
                    &root.0,
                    &BTreeMap::new(),
                    &root.0,
                    Instant::now() + Duration::from_secs(5)
                )
                .is_err()
            );
        }
    }
}
