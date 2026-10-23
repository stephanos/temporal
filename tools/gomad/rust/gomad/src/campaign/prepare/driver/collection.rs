use super::*;

#[derive(Deserialize, Serialize)]
#[serde(untagged)]
pub(crate) enum Recipes {
    Single(Box<Recipe>),
    Collection(Collection),
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Collection {
    schema: String,
    packages: Vec<Recipe>,
}

impl Recipes {
    #[cfg(any(test, feature = "determinism-fixture"))]
    pub(crate) fn new(packages: Vec<Recipe>) -> Result<Self, String> {
        let recipes = Self::Collection(Collection {
            schema: "gomad.driver-recipes/v1".into(),
            packages,
        });
        recipes.validate()?;
        Ok(recipes)
    }

    fn packages(&self) -> &[Recipe] {
        match self {
            Self::Single(recipe) => std::slice::from_ref(recipe.as_ref()),
            Self::Collection(collection) => &collection.packages,
        }
    }

    pub(crate) fn packages_mut(&mut self) -> &mut [Recipe] {
        match self {
            Self::Single(recipe) => std::slice::from_mut(recipe.as_mut()),
            Self::Collection(collection) => &mut collection.packages,
        }
    }

    pub(crate) fn first(&self) -> Result<&Recipe, String> {
        self.validate()?;
        self.packages()
            .first()
            .ok_or_else(|| failure("missing driver recipe"))
    }

    pub(crate) fn validate(&self) -> Result<(), String> {
        let packages = self.packages();
        if let Self::Collection(collection) = self {
            if collection.schema != "gomad.driver-recipes/v1" || packages.len() != 2 {
                return Err(failure(
                    "private driver collection requires exactly two packages",
                ));
            }
            let first = &packages[0];
            let second = &packages[1];
            if first.options.target_kind != "go-test"
                || second.options.target_kind != "go-test"
                || first.import_path == second.import_path
                || first.options.source == second.options.source
                || common_identity(first)? != common_identity(second)?
            {
                return Err(failure(
                    "private driver collection package identities disagree",
                ));
            }
        }
        for recipe in packages {
            if recipe.schema != "gomad.driver-recipe/v1"
                || recipe.import_path.is_empty()
                || recipe.import_path.contains(char::is_whitespace)
                || recipe.list.last() != Some(&recipe.options.source)
            {
                return Err(failure("invalid private package recipe"));
            }
        }
        Ok(())
    }

    pub(crate) fn driver_arguments(&self, launcher: &Path) -> Result<Vec<String>, String> {
        let first = self.first()?;
        let mut argv = first.driver_arguments(launcher)?;
        if let Self::Collection(collection) = self {
            let source = argv.len() - first.options.target_args.len() - 1;
            if argv[source] != first.options.source {
                return Err(failure("private driver package operand missing"));
            }
            argv.splice(
                source..=source,
                std::iter::once("-p=2".into()).chain(
                    collection
                        .packages
                        .iter()
                        .map(|recipe| recipe.options.source.clone()),
                ),
            );
        }
        Ok(argv)
    }

    pub(crate) fn admit_external(
        self,
        module: Vec<u8>,
        arguments: Vec<String>,
        invocation: &super::super::super::origin::Invocation,
        deadline: Instant,
    ) -> Result<Prepared, String> {
        if let Self::Single(recipe) = self {
            return recipe.admit_external(module, arguments, invocation, deadline);
        }
        let expected = self.driver_arguments(&std::env::current_exe().map_err(failure)?)?;
        let metadata = crate::evidence::symbolize::build_info::read(&module)?;
        let selected = select(self.packages(), &metadata.path)?;
        let Self::Collection(mut collection) = self else {
            unreachable!()
        };
        collection.packages.remove(selected).admit_external_bound(
            module, arguments, invocation, &expected, &metadata, deadline,
        )
    }
}

fn select(packages: &[Recipe], path: &str) -> Result<usize, String> {
    let mut matches = packages
        .iter()
        .enumerate()
        .filter(|(_, recipe)| path == format!("{}.test", recipe.import_path));
    let selected = matches.next().map(|(index, _)| index);
    if matches.next().is_some() {
        return Err(failure(
            "actual module does not select exactly one private package recipe",
        ));
    }
    selected
        .ok_or_else(|| failure("actual module does not select exactly one private package recipe"))
}

fn common_identity(recipe: &Recipe) -> Result<Vec<u8>, String> {
    let mut options = recipe.options.clone();
    options.source.clear();
    let mut inputs = recipe.inputs.clone();
    for field in ["source", "sources"] {
        inputs
            .as_object_mut()
            .ok_or_else(|| failure("invalid recipe inputs"))?
            .remove(field);
    }
    canonical::encode(&json!({
        "options": options, "installed": recipe.installed, "directory": recipe.directory,
        "environment": recipe.environment, "root": recipe.root, "compiler": recipe.compiler,
        "replacements": recipe.replacements, "flags": recipe.flags,
        "list": &recipe.list[..recipe.list.len().saturating_sub(1)],
        "configuration": recipe.configuration, "inputs": inputs,
    }))
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;

    pub(crate) fn recipe(package: &str) -> Recipe {
        let mut ignored = false;
        let options = crate::cli::explore::parse(
            &[
                "go-test".into(),
                package.into(),
                "--".into(),
                "-test.run=TestFrontendSystemInfo".into(),
            ],
            &mut ignored,
        )
        .unwrap();
        let configuration = json!({
            "engine":linked::Conditions::compiled().configuration(),
            "profile":"gomad.wasi-stock-exploratory/v1",
            "limits":{"fuel":1000,"memory_bytes":1_048_576,"wall_millis":1000,"output_bytes":1_048_576,"evidence_bytes":16_777_216},
            "wall_nanos":null,"arguments":["gomad-target"],"environment":{},
            "derive_seed_environment":true,"network":null,"runtime_sha256":null,
            "model":null,"namespace":null,"storage":null,
        });
        serde_json::from_value(json!({
            "schema":"gomad.driver-recipe/v1", "import_path":format!("example.test/{package}"),
            "options":options,
            "installed":{
                "schema":crate::install::manifest::SCHEMA,
                "compiler_path":"/pinned/go", "compiler_sha256":"1".repeat(64),
                "go_version":"go1.27.1", "runtime_root":"/runtime", "cache_root":"/cache",
                "profile":"gomad.wasi-stock-exploratory/v1", "fuel":1000,"memory_bytes":1_048_576,
                "helper_sha256":"2".repeat(64),"host_sha256":"3".repeat(64),
                "runtime_implementation_sha256":null, "model":{},
                "namespace":{"entries":[],"writable_directories":[],"preopens":[[47]],"working_directory":[47],"stdin":[],"epoch_nanos":0,
                             "limits":{"files":16,"descriptors":16,"filesystem_bytes":4096,"output_bytes":1_048_576}},
                "storage":null
            },
            "directory":"/source", "environment":{"GOOS":"wasip1","GOARCH":"wasm"},
            "root":"/pinned", "compiler":{"/pinned/go":"1".repeat(64)},
            "replacements":{},"flags":["-mod=readonly","-pgo=off"],
            "list":["list","-test",package],"sources":{format!("/source/{package}/test.go"):"4".repeat(64)},
            "configuration":configuration,"inputs":{"source":package,"sources":{},"configuration":configuration,"tags":[]}
        })).unwrap()
    }

    #[test]
    fn single_recipe_encoding_and_full_two_package_argv_are_preserved() {
        let single = recipe("first");
        let original = canonical::encode(&single).unwrap();
        let parsed: Recipes = serde_json::from_slice(&original).unwrap();
        assert_eq!(canonical::encode(&parsed).unwrap(), original);
        let launcher = Path::new("/proof/gomad");
        assert_eq!(
            parsed.driver_arguments(launcher).unwrap(),
            single.driver_arguments(launcher).unwrap()
        );
        let recipes = Recipes::new(vec![recipe("first"), recipe("second")]).unwrap();
        let argv = recipes.driver_arguments(launcher).unwrap();
        assert_eq!(
            argv,
            [
                "/pinned/go",
                "test",
                "-mod=readonly",
                "-pgo=off",
                "-trimpath",
                "-buildvcs=false",
                "-count=1",
                "-exec",
                "'/proof/gomad' run-guest",
                "-p=2",
                "first",
                "second",
                "-test.run=TestFrontendSystemInfo"
            ]
        );
        assert_eq!(
            select(recipes.packages(), "example.test/first.test").unwrap(),
            0
        );
        assert_eq!(
            select(recipes.packages(), "example.test/second.test").unwrap(),
            1
        );
        for path in [
            "example.test/first",
            "prefix/example.test/first.test",
            "example.test/third.test",
        ] {
            assert!(select(recipes.packages(), path).is_err());
        }
    }

    #[test]
    fn missing_ambiguous_and_mismatched_collection_members_are_refused() {
        assert!(Recipes::new(vec![recipe("first")]).is_err());
        assert!(Recipes::new(vec![recipe("first"), recipe("second"), recipe("third")]).is_err());
        assert!(Recipes::new(vec![recipe("first"), recipe("first")]).is_err());
        let mut second = recipe("second");
        second.import_path = recipe("first").import_path;
        assert!(Recipes::new(vec![recipe("first"), second]).is_err());
        assert!(
            select(
                &[recipe("first"), recipe("first")],
                "example.test/first.test"
            )
            .is_err()
        );
        for field in [
            "environment",
            "installed",
            "configuration",
            "options",
            "flags",
            "compiler",
            "inputs",
        ] {
            let mut value = serde_json::to_value(recipe("second")).unwrap();
            match field {
                "environment" => value[field]["GOARCH"] = json!("amd64"),
                "installed" => value[field]["compiler_sha256"] = json!("9".repeat(64)),
                "configuration" => value[field]["limits"]["fuel"] = json!(999),
                "options" => value[field]["target_args"] = json!(["-test.run=Other"]),
                "flags" => value[field] = json!(["-mod=readonly", "-race"]),
                "compiler" => value[field]["/pinned/go"] = json!("9".repeat(64)),
                "inputs" => value[field]["host"] = json!("9".repeat(64)),
                _ => unreachable!(),
            }
            let second = serde_json::from_value(value).unwrap();
            assert!(
                Recipes::new(vec![recipe("first"), second]).is_err(),
                "{field}"
            );
        }
    }

    #[test]
    fn selected_package_still_requires_pinned_settings_and_exact_package() {
        use crate::evidence::symbolize::build_info::BuildInfo;
        let selected = recipe("second");
        let mut info = BuildInfo {
            compiler_version: "go1.27.1".into(),
            path: "example.test/second.test".into(),
            settings: [
                ("GOOS", "wasip1"),
                ("GOARCH", "wasm"),
                ("CGO_ENABLED", "0"),
                ("-trimpath", "true"),
                ("-compiler", "gc"),
                ("-buildmode", "exe"),
                ("GOEXPERIMENT", "nogreenteagc"),
            ]
            .into_iter()
            .map(|(name, value)| (name.into(), value.into()))
            .collect(),
        };
        selected.authenticate_build_info(&info).unwrap();
        info.path = "example.test/first.test".into();
        assert!(selected.authenticate_build_info(&info).is_err());
        info.path = "example.test/second.test".into();
        info.compiler_version = "go1.27.2".into();
        assert!(selected.authenticate_build_info(&info).is_err());
        info.compiler_version = "go1.27.1".into();
        for (name, value) in [
            ("GOARCH", "amd64"),
            ("GOOS", "linux"),
            ("-tags", "unrelated"),
            ("-race", "true"),
        ] {
            let original = info.settings.insert(name.into(), value.into());
            assert!(selected.authenticate_build_info(&info).is_err(), "{name}");
            if let Some(original) = original {
                info.settings.insert(name.into(), original);
            } else {
                info.settings.remove(name);
            }
        }
    }
}
