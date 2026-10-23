use super::*;
use crate::campaign::origin;
use std::collections::BTreeSet;

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Shared {
    pub schema: String,
    pub packages: Vec<Recipe>,
    pub driver_argv: Vec<String>,
    pub installations: Vec<String>,
}

impl Shared {
    pub(crate) fn new(packages: Vec<Recipe>, driver_argv: Vec<String>) -> Result<Self, String> {
        let installations = packages
            .iter()
            .map(|recipe| {
                crate::install::manifest::resolve(recipe.options.installation.as_deref())
                    .map(|selected| selected.sha256)
            })
            .collect::<Result<Vec<_>, _>>()?;
        let value = Self {
            schema: "gomad.shim-driver/v1".into(),
            packages,
            driver_argv,
            installations,
        };
        value.validate()?;
        Ok(value)
    }

    pub(crate) fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.shim-driver/v1"
            || self.packages.is_empty()
            || self.packages.len() > 65536
            || self.installations.len() != self.packages.len()
            || self.driver_argv.len() < 2
            || self.driver_argv.len() > 65536
        {
            return Err(failure("invalid shared Go driver recipe"));
        }
        let mut imports = BTreeSet::new();
        let first = &self.packages[0];
        for (recipe, installation) in self.packages.iter().zip(&self.installations) {
            if recipe.schema != "gomad.driver-recipe/v1"
                || recipe.import_path.is_empty()
                || !imports.insert(&recipe.import_path)
                || !canonical::valid_digest(installation)
                || recipe.installed.compiler_path != first.installed.compiler_path
                || recipe.installed.compiler_sha256 != first.installed.compiler_sha256
                || recipe.options.target_kind != first.options.target_kind
                || recipe.directory != first.directory
                || recipe.environment != first.environment
                || self.driver_argv[0]
                    != origin::argument(recipe.installed.compiler_path.as_os_str())
            {
                return Err(failure("shared driver package association mismatch"));
            }
        }
        let expected_command = if first.options.target_kind == "go-test" {
            "test"
        } else {
            "run"
        };
        let mut command = 1;
        if self
            .driver_argv
            .get(command)
            .is_some_and(|value| value == "2d43")
        {
            command += 2;
        } else if self
            .driver_argv
            .get(command)
            .is_some_and(|value| value.starts_with("2d433d"))
        {
            command += 1;
        }
        let actual = self.driver_argv.get(command);
        let expected = origin::argument(std::ffi::OsStr::new(expected_command));
        let build = origin::argument(std::ffi::OsStr::new("build"));
        if actual != Some(&expected)
            && !(first.options.target_kind == "go-run" && actual == Some(&build))
        {
            return Err(failure("shared driver command mismatch"));
        }
        Ok(())
    }

    pub(crate) fn installation_bindings(&self) -> Vec<(String, String)> {
        self.packages
            .iter()
            .zip(&self.installations)
            .map(|(recipe, digest)| (recipe.import_path.clone(), digest.clone()))
            .collect()
    }

    pub(crate) fn installation_commitment(&self) -> Result<String, String> {
        self.validate()?;
        let bindings = self
            .packages
            .iter()
            .zip(&self.installations)
            .map(|(recipe, digest)| (&recipe.import_path, digest))
            .collect::<Vec<_>>();
        canonical::encode(&bindings).map(|bytes| canonical::digest(&bytes))
    }

    pub(crate) fn select(&self, path: &str) -> Result<usize, String> {
        self.validate()?;
        let mut matches = self.packages.iter().enumerate().filter(|(_, recipe)| {
            let expected = if recipe.options.target_kind == "go-test" {
                format!("{}.test", recipe.import_path)
            } else {
                recipe.import_path.clone()
            };
            path == expected
        });
        let selected = matches
            .next()
            .map(|(index, _)| index)
            .ok_or_else(|| failure("actual module is not a member of the shared Go driver"))?;
        if matches.next().is_some() {
            return Err(failure("ambiguous shared Go driver module"));
        }
        Ok(selected)
    }

    pub(crate) fn admit(
        mut self,
        selected: usize,
        module: Vec<u8>,
        arguments: Vec<String>,
        invocation: &origin::Invocation,
        metadata: &crate::evidence::symbolize::build_info::BuildInfo,
        deadline: Instant,
    ) -> Result<Prepared, String> {
        if self.installation_commitment()? != invocation.installation_sha256
            || self.driver_argv != invocation.driver_argv
            || selected != self.select(&metadata.path)?
        {
            return Err(failure("shared Go driver commitment mismatch"));
        }
        let bindings = self.installation_bindings();
        let installation = self.installations.remove(selected);
        let mut recipe = self.packages.remove(selected);
        recipe.inputs["shared_driver_installations"] = json!(bindings);
        recipe.admit_registered(
            module,
            arguments,
            invocation,
            &self.driver_argv,
            &installation,
            metadata,
            deadline,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn shared(names: &[&str]) -> Shared {
        let packages = names
            .iter()
            .map(|name| super::super::collection::tests::recipe(name))
            .collect::<Vec<_>>();
        let driver_argv = vec![
            origin::argument(packages[0].installed.compiler_path.as_os_str()),
            origin::argument(std::ffi::OsStr::new("test")),
        ];
        Shared {
            schema: "gomad.shim-driver/v1".into(),
            installations: names.iter().map(|_| "a".repeat(64)).collect(),
            packages,
            driver_argv,
        }
    }

    #[test]
    fn public_shared_driver_has_no_private_two_package_restriction() {
        let value = shared(&["./one", "./two", "./three", "./four"]);
        value.validate().unwrap();
        for (index, package) in value.packages.iter().enumerate() {
            assert_eq!(
                value
                    .select(&format!("{}.test", package.import_path))
                    .unwrap(),
                index
            );
        }
        assert!(value.select("unregistered/package.test").is_err());
    }

    #[test]
    fn catalog_installation_commitment_binds_each_package_and_order() {
        let mut value = shared(&["./one", "./two", "./three"]);
        let original = value.installation_commitment().unwrap();
        value.installations[1] = "b".repeat(64);
        assert_ne!(original, value.installation_commitment().unwrap());
        value.packages.swap(0, 1);
        assert_ne!(original, value.installation_commitment().unwrap());
        value.installations.pop();
        assert!(value.validate().is_err());
    }

    #[test]
    fn duplicate_packages_compiler_disagreement_and_command_aliases_refuse() {
        let mut duplicate = shared(&["./one", "./one"]);
        assert!(duplicate.validate().is_err());
        duplicate = shared(&["./one", "./two"]);
        duplicate.packages[1].installed.compiler_sha256 = "f".repeat(64);
        assert!(duplicate.validate().is_err());
        let mut command = shared(&["./one"]);
        command.driver_argv[1] = origin::argument(std::ffi::OsStr::new("build"));
        assert!(command.validate().is_err());
        command
            .driver_argv
            .insert(1, origin::argument(std::ffi::OsStr::new("-C")));
        command
            .driver_argv
            .insert(2, origin::argument(std::ffi::OsStr::new("test")));
        assert!(command.validate().is_err());
        command.driver_argv[3] = origin::argument(std::ffi::OsStr::new("test"));
        command.validate().unwrap();
    }
    #[test]
    fn actual_build_metadata_must_match_ordered_compiler_flags() {
        let mut recipe = super::super::collection::tests::recipe("./one");
        let settings = [
            ("GOOS", "wasip1"),
            ("GOARCH", "wasm"),
            ("CGO_ENABLED", "0"),
            ("-trimpath", "true"),
            ("-compiler", "gc"),
            ("-buildmode", "exe"),
            ("GOEXPERIMENT", "nogreenteagc"),
        ]
        .into_iter()
        .map(|(name, value)| (name.to_owned(), value.to_owned()))
        .collect();
        let mut info = crate::evidence::symbolize::build_info::BuildInfo {
            compiler_version: "go1.27.1".into(),
            path: format!("{}.test", recipe.import_path),
            settings,
        };
        recipe.authenticate_build_info(&info).unwrap();
        info.settings.insert("-gcflags".into(), "all=-N".into());
        assert!(recipe.authenticate_build_info(&info).is_err());
        recipe
            .flags
            .extend(["-gcflags=all=-l".into(), "-gcflags=all=-N".into()]);
        recipe.authenticate_build_info(&info).unwrap();
        info.settings
            .insert("-ldflags".into(), "-X pkg.value=unexpected".into());
        assert!(recipe.authenticate_build_info(&info).is_err());
        info.settings.remove("-ldflags");
        info.settings
            .insert("vcs.revision".into(), "unexpected".into());
        assert!(recipe.authenticate_build_info(&info).is_err());
    }
    #[test]
    fn file_list_driver_operands_remain_distinct_without_changing_legacy_encoding() {
        let mut recipe = super::super::collection::tests::recipe("one");
        let legacy = canonical::encode(&recipe).unwrap();
        assert!(
            serde_json::from_slice::<serde_json::Value>(&legacy)
                .unwrap()
                .get("source_arguments")
                .is_none()
        );
        recipe.options.target_kind = "go-run".into();
        recipe.options.target_args.clear();
        recipe.options.source = "main.go".into();
        recipe.import_path = "command-line-arguments".into();
        recipe.source_arguments = vec!["main.go".into(), "other.go".into()];
        let argv = recipe.driver_arguments(Path::new("/gomad")).unwrap();
        assert_eq!(&argv[argv.len() - 2..], ["main.go", "other.go"]);
        let value = Shared {
            schema: "gomad.shim-driver/v1".into(),
            installations: vec!["a".repeat(64)],
            driver_argv: argv
                .iter()
                .map(|arg| origin::argument(std::ffi::OsStr::new(arg)))
                .collect(),
            packages: vec![recipe],
        };
        value.validate().unwrap();
        assert_eq!(value.select("command-line-arguments").unwrap(), 0);
    }
}
