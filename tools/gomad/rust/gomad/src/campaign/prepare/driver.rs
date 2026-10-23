use super::*;

pub(crate) mod collection;

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Recipe {
    pub schema: String,
    pub import_path: String,
    pub options: ExploreOptions,
    pub installed: Installation,
    pub directory: PathBuf,
    pub environment: BTreeMap<String, String>,
    pub root: PathBuf,
    pub compiler: BTreeMap<PathBuf, String>,
    pub replacements: BTreeMap<String, String>,
    pub flags: Vec<String>,
    pub list: Vec<String>,
    pub sources: BTreeMap<PathBuf, String>,
    pub configuration: Configuration,
    pub inputs: Value,
}

#[expect(
    clippy::too_many_lines,
    reason = "Recipe generation inventories the compiler, overlay and sources in the existing ordered admission"
)]
pub(crate) fn prepare(options: &ExploreOptions, deadline: Instant) -> Result<Recipe, String> {
    process::check_deadline(deadline)?;
    let installed = installation(options)?;
    if options.soft_assertion_report_only && options.target_kind != "go-test" {
        return Err("unsupported_target: soft assertion report-only requires go-test".into());
    }
    if !matches!(options.target_kind.as_str(), "go-run" | "go-test")
        || !options.provenance.is_empty()
    {
        return Err("unsupported_target: unsupported source preparation request".into());
    }
    let scenario = options
        .scenario
        .as_deref()
        .map(super::super::scenario::normalize)
        .transpose()?;
    let cooperative = installed.profile == "gomad.wasi-cooperative/v1";
    if cooperative && !options.choices && scenario.is_none()
        || !cooperative && (options.choices || options.diagnostics || scenario.is_some())
    {
        return Err("unsupported_target: choices/scenario do not match installed profile".into());
    }
    let directory = fs::canonicalize(&options.working_dir).map_err(failure)?;
    let temporary = process::Temporary::new(&installed.cache_root)?;
    let mut environment = process::environment()?;
    let version = process::command(
        &installed.compiler_path,
        &["version".into()],
        &directory,
        &environment,
        &temporary.0,
        64 << 10,
        deadline,
    )?;
    if !String::from_utf8_lossy(&version).contains(&format!("{} ", installed.go_version)) {
        return Err(failure("compiler version does not match installation"));
    }
    let (mut compiler, root) = inventory::compiler(
        &installed.compiler_path,
        &directory,
        &environment,
        &temporary.0,
        deadline,
    )?;
    let mut replacements = BTreeMap::new();
    let mut runtime = None;
    let mut overlay = None;
    if cooperative {
        let cache = process::command(
            &installed.compiler_path,
            &["env".into(), "GOMODCACHE".into()],
            &directory,
            &environment,
            &temporary.0,
            64 << 10,
            deadline,
        )?;
        let cache = PathBuf::from(String::from_utf8_lossy(&cache).trim());
        if root.starts_with(fs::canonicalize(&cache).unwrap_or(cache)) {
            return Err(failure("cooperative compiler must be outside module cache"));
        }
        environment.insert("GOROOT".into(), root.to_string_lossy().into_owned());
        let module_root = installed
            .overlay_module_root
            .as_deref()
            .or_else(|| {
                installed
                    .runtime_root
                    .parent()
                    .and_then(Path::parent)
                    .and_then(Path::parent)
            })
            .ok_or_else(|| failure("overlay module owner is required"))?;
        if !module_root.is_absolute() {
            return Err(failure("absolute overlay module owner is required"));
        }
        let generated = crate::install::toolchain::overlay(
            &installed,
            &root,
            &environment,
            &temporary.0,
            deadline,
        )?;
        let value: Value =
            serde_json::from_slice(&read(&generated.path, 16 << 20)?).map_err(failure)?;
        replacements = serde_json::from_value(
            value
                .get("Replace")
                .cloned()
                .ok_or_else(|| failure("overlay replacements missing"))?,
        )
        .map_err(failure)?;
        for (original, replacement) in &replacements {
            for path in [Path::new(original), Path::new(replacement)] {
                if path.is_file() {
                    compiler.insert(path.into(), canonical::digest(&read(path, MAXIMUM)?));
                }
            }
        }
        for name in [
            "internal/model/choice/schema/choicewire.json",
            "internal/model/choice/schema/choicewire.go.tmpl",
            "internal/model/choice/schema/choicewire_runtime.go.tmpl",
            "internal/model/choice/schema/choicewire_test.go.tmpl",
            "internal/model/choice/trace.go",
            "internal/model/choice/tape.go",
            "internal/backends/wasm/toolchain/overlay.go",
            "internal/backends/wasm/toolchain/runtime_wasm.go.txt",
            "internal/backends/wasm/toolchain/runtime_inspect_wasm.go.txt",
            "internal/backends/wasm/toolchain/net_stream_wasm.go.txt",
            "internal/backends/wasm/toolchain/net_stream_status_generated.go.txt",
            "internal/backends/wasm/toolchain/cmd/overlay/main.go",
        ] {
            let path = module_root.join(name);
            compiler.insert(path.clone(), canonical::digest(&read(&path, MAXIMUM)?));
        }
        for path in [
            &generated.path,
            &installed.runtime_root.join("gomad_semantics.go.txt"),
            &installed
                .runtime_root
                .join("gomad_choicewire_generated.go.txt"),
        ] {
            compiler.insert(path.clone(), canonical::digest(&read(path, MAXIMUM)?));
        }
        runtime = Some(generated.identity);
        overlay = Some(generated.path);
    }
    let mut tags = options.build_tags.clone();
    tags.extend(installed.build_tags.iter().cloned());
    tags.sort();
    tags.dedup();
    let mut flags = vec!["-mod=readonly".into(), "-pgo=off".into()];
    if let Some(path) = &overlay {
        flags.extend(["-overlay".into(), path.to_string_lossy().into_owned()]);
    }
    if !tags.is_empty() {
        flags.extend(["-tags".into(), tags.join(",")]);
    }
    let mut list = vec!["list".into()];
    list.extend(flags.clone());
    list.extend(["-deps".into(), "-json".into()]);
    if options.target_kind == "go-test" {
        list.push("-test".into());
    }
    list.push(options.source.clone());
    let mut package_arguments = vec!["list".into()];
    package_arguments.extend(flags.iter().cloned());
    package_arguments.push(options.source.clone());
    let package = process::command(
        &installed.compiler_path,
        &package_arguments,
        &directory,
        &environment,
        &temporary.0,
        1 << 20,
        deadline,
    )?;
    let import_path = String::from_utf8(package)
        .map_err(failure)?
        .trim()
        .to_owned();
    if import_path.is_empty() || import_path.contains(char::is_whitespace) {
        return Err(failure("driver recipe requires one concrete package"));
    }
    let sources = inventory::sources(
        &process::command(
            &installed.compiler_path,
            &list,
            &directory,
            &environment,
            &temporary.0,
            128 << 20,
            deadline,
        )?,
        &root.join("pkg/include"),
        &replacements,
    )?;
    let configuration = configuration(options, &installed)?;
    let build_environment = environment
        .iter()
        .filter(|(name, _)| {
            matches!(
                name.as_str(),
                "GOENV"
                    | "GOPROXY"
                    | "GOSUMDB"
                    | "GOPRIVATE"
                    | "GONOPROXY"
                    | "GOVCS"
                    | "GOFLAGS"
                    | "GOTOOLCHAIN"
                    | "GOWORK"
                    | "GOOS"
                    | "GOARCH"
                    | "CGO_ENABLED"
                    | "GOEXPERIMENT"
                    | "GOCACHEPROG"
                    | "GOWASM"
                    | "GODEBUG"
            )
        })
        .collect::<BTreeMap<_, _>>();
    let inputs = json!({"schema":"gomad.rust-wasm-build-inputs/v1", "compiler":compiler,"compiler_path":installed.compiler_path,"compiler_sha256":installed.compiler_sha256,
        "go_version":installed.go_version,"sources":sources,"runtime":runtime,"runtime_implementation":installed.runtime_implementation_sha256,
        "helper":linked::IMPLEMENTATION_SHA256,"host":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),"kind":options.target_kind,"source":options.source,
        "directory":directory,"arguments":options.target_args,"tags":tags,"configuration":configuration,"scenario":scenario,"network":installed.network,
        "build_flags":["-mod=readonly","-trimpath","-buildvcs=false","-pgo=off"],"environment":build_environment,"soft_assertion_report_only":options.soft_assertion_report_only});
    Ok(Recipe {
        schema: "gomad.driver-recipe/v1".into(),
        import_path,
        options: options.clone(),
        installed,
        directory,
        environment,
        root,
        compiler,
        replacements,
        flags,
        list,
        sources,
        configuration,
        inputs,
    })
}

impl Recipe {
    #[expect(
        clippy::too_many_lines,
        reason = "Actual-module admission keeps post-build inventory and provenance commitments together"
    )]
    pub(crate) fn admit(
        self,
        module: Vec<u8>,
        arguments: Vec<String>,
        deadline: Instant,
    ) -> Result<Prepared, String> {
        let Self {
            installed,
            mut options,
            directory,
            environment,
            root,
            compiler,
            replacements,
            list,
            sources,
            mut configuration,
            mut inputs,
            ..
        } = self;
        options.target_args = arguments;
        let program = configuration
            .arguments
            .first()
            .cloned()
            .ok_or_else(|| failure("prepared configuration has no program argument"))?;
        configuration.arguments = std::iter::once(program)
            .chain(options.target_args.iter().cloned())
            .collect();
        inputs["arguments"] = json!(options.target_args);
        inputs["configuration"] = serde_json::to_value(&configuration).map_err(failure)?;
        let cooperative = installed.profile == "gomad.wasi-cooperative/v1";
        let scenario = options
            .scenario
            .as_deref()
            .map(super::super::scenario::normalize)
            .transpose()?;
        let temporary = process::Temporary::new(&installed.cache_root)?;
        let module_metadata = linked::module_metadata(&module).map_err(failure)?;
        inputs["module_metadata"] = serde_json::to_value(&module_metadata).map_err(failure)?;
        let toolchain = canonical::digest(&canonical::encode(&inputs)?);
        let after = inventory::sources(
            &process::command(
                &installed.compiler_path,
                &list,
                &directory,
                &environment,
                &temporary.0,
                128 << 20,
                deadline,
            )?,
            &root.join("pkg/include"),
            &replacements,
        )?;
        if sources != after {
            return Err(failure(
                "selected source inventory changed during compilation",
            ));
        }
        inventory::verify(&compiler)?;
        inventory::verify(&sources)?;
        crate::install::manifest::executable(&installed.compiler_path, &installed.compiler_sha256)?;
        let (after_compiler, after_root) = inventory::compiler(
            &installed.compiler_path,
            &directory,
            &environment,
            &temporary.0,
            deadline,
        )?;
        if after_root != root
            || after_compiler
                .iter()
                .any(|(path, digest)| compiler.get(path) != Some(digest))
        {
            return Err(failure("compiler inventory changed during compilation"));
        }
        let choice_identity = if cooperative {
            let implementation = choice_implementation(&toolchain)?;
            inventory::verify(&compiler)?;
            configuration.runtime_sha256 = Some(format!("{implementation:x}"));
            Some(linked::ChoiceIdentity {
                target_sha256: Sha256::digest(&module).into(),
                toolchain_build_key: toolchain.clone(),
                goos: "wasip1".into(),
                goarch: "wasm".into(),
                implementation_sha256: implementation.into(),
            })
        } else {
            None
        };
        let provenance = canonical::encode(&Provenance {
            schema: "gomad.rust-wasm-provenance/v1".into(),
            origin: None,
            runtime_workload: None,
            inputs,
            build_key: toolchain.clone(),
            module_sha256: canonical::digest(&module),
            module_metadata: Some(module_metadata),
            configuration,
            choice_identity,
            scenario,
            network: installed.network,
        })?;
        let decoded: Provenance = serde_json::from_slice(&provenance).map_err(failure)?;
        Ok(Prepared {
            module,
            configuration: decoded.configuration,
            toolchain,
            choice_identity: decoded.choice_identity,
            scenario: decoded.scenario,
            provenance,
            network: decoded.network,
            soft_assertion_report_only: options.soft_assertion_report_only,
        })
    }
}

impl Recipe {
    pub(crate) fn driver_arguments(&self, launcher: &Path) -> Result<Vec<String>, String> {
        let compiler = self
            .installed
            .compiler_path
            .to_str()
            .ok_or("compiler path is not UTF-8")?;
        let launcher = launcher.to_str().ok_or("launcher path is not UTF-8")?;
        let mut arguments = vec![
            compiler.into(),
            if self.options.target_kind == "go-test" {
                "test"
            } else {
                "run"
            }
            .into(),
        ];
        arguments.extend(self.flags.iter().cloned());
        arguments.extend(["-trimpath".into(), "-buildvcs=false".into()]);
        if self.options.target_kind == "go-test" {
            arguments.push("-count=1".into());
        }
        let launcher = if !launcher.contains('\'') {
            format!("'{launcher}'")
        } else if !launcher.contains('"') {
            format!("\"{launcher}\"")
        } else {
            return Err(
                "launcher path contains both quote kinds and cannot be represented by Go -exec"
                    .into(),
            );
        };
        arguments.extend([
            "-exec".into(),
            format!("{launcher} run-guest"),
            self.options.source.clone(),
        ]);
        arguments.extend(self.options.target_args.iter().cloned());
        Ok(arguments)
    }

    pub(crate) fn admit_external(
        self,
        module: Vec<u8>,
        arguments: Vec<String>,
        invocation: &super::super::origin::Invocation,
        deadline: Instant,
    ) -> Result<Prepared, String> {
        let expected = self.driver_arguments(&std::env::current_exe().map_err(failure)?)?;
        let metadata = crate::evidence::symbolize::build_info::read(&module)?;
        self.admit_external_bound(
            module, arguments, invocation, &expected, &metadata, deadline,
        )
    }

    fn admit_external_bound(
        mut self,
        module: Vec<u8>,
        arguments: Vec<String>,
        invocation: &super::super::origin::Invocation,
        expected_argv: &[String],
        metadata: &crate::evidence::symbolize::build_info::BuildInfo,
        deadline: Instant,
    ) -> Result<Prepared, String> {
        invocation.validate()?;
        let selected = crate::install::manifest::resolve(self.options.installation.as_deref())?;
        crate::install::manifest::admit_build(&selected.manifest)?;
        if self.schema != "gomad.driver-recipe/v1"
            || canonical::encode(&selected.manifest)? != canonical::encode(&self.installed)?
            || selected.sha256 != invocation.installation_sha256
            || self.installed.compiler_sha256 != invocation.compiler_sha256
            || self.inputs["host"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
            || self.inputs["helper"] != linked::IMPLEMENTATION_SHA256
            || self.inputs["compiler"] != json!(self.compiler)
            || self.inputs["sources"] != json!(self.sources)
            || self.inputs["configuration"]
                != serde_json::to_value(&self.configuration).map_err(failure)?
            || self.inputs["source"] != self.options.source
            || self.inputs["kind"] != self.options.target_kind
            || self.inputs["directory"] != json!(self.directory)
        {
            return Err(failure("intercepted driver recipe identity mismatch"));
        }
        let expected = expected_argv
            .iter()
            .map(|arg| super::super::origin::argument(std::ffi::OsStr::new(arg)))
            .collect::<Vec<_>>();
        if invocation.driver_argv != expected {
            return Err(failure(
                "intercepted driver invocation does not match its recipe",
            ));
        }
        inventory::verify(&self.compiler)?;
        inventory::verify(&self.sources)?;
        self.authenticate_build_info(metadata)?;
        self.inputs["go_build_info"] = serde_json::to_value(metadata).map_err(failure)?;
        self.admit(module, arguments, deadline)
    }

    fn authenticate_build_info(
        &self,
        info: &crate::evidence::symbolize::build_info::BuildInfo,
    ) -> Result<(), String> {
        let expected_path = if self.options.target_kind == "go-test" {
            format!("{}.test", self.import_path)
        } else {
            self.import_path.clone()
        };
        if info.compiler_version != self.installed.go_version || info.path != expected_path {
            return Err(failure("intercepted module package mismatch"));
        }
        let values = &info.settings;
        for (name, expected) in [
            ("GOOS", "wasip1"),
            ("GOARCH", "wasm"),
            ("CGO_ENABLED", "0"),
            ("-trimpath", "true"),
            ("-compiler", "gc"),
            ("-buildmode", "exe"),
            ("GOEXPERIMENT", "nogreenteagc"),
        ] {
            if values.get(name).map(String::as_str) != Some(expected) {
                return Err(failure(format!(
                    "intercepted module setting {name} mismatch"
                )));
            }
        }
        for name in ["-race", "-msan", "-asan", "-cover"] {
            if values.get(name).is_some_and(|value| value != "false") {
                return Err(failure(
                    "intercepted module has unsupported instrumentation",
                ));
            }
        }
        let expected_tags = self.inputs["tags"]
            .as_array()
            .ok_or_else(|| failure("recipe tags missing"))?
            .iter()
            .map(|v| v.as_str().ok_or_else(|| failure("invalid recipe tag")))
            .collect::<Result<Vec<_>, _>>()?
            .join(",");
        if values.get("-tags").map_or("", String::as_str) != expected_tags {
            return Err(failure("intercepted module build tags mismatch"));
        }
        Ok(())
    }
}

#[cfg(test)]
mod retained_input_tests {
    use super::*;
    use crate::install::manifest;

    #[test]
    #[ignore = "requires explicitly retained unaccepted input and its original private recipe"]
    fn retained_actual_wasm_metadata_matches_only_its_original_recipe() {
        let module_path =
            std::env::var_os("GOMAD_TEST_UNACCEPTED_MODULE").expect("retained module");
        let recipe_path =
            std::env::var_os("GOMAD_TEST_UNACCEPTED_RECIPE").expect("original recipe");
        let module = manifest::read(Path::new(&module_path), 512 << 20).unwrap();
        let expected = std::env::var("GOMAD_TEST_UNACCEPTED_SHA256").expect("retained digest");
        assert_eq!(canonical::digest(&module), expected);
        let recipe: Recipe =
            serde_json::from_slice(&manifest::read(Path::new(&recipe_path), 32 << 20).unwrap())
                .unwrap();
        let start = Instant::now();
        let (mut info, work) = crate::evidence::symbolize::build_info::measured(&module).unwrap();
        recipe.authenticate_build_info(&info).unwrap();
        println!(
            "unaccepted input metadata only: module_bytes={} elapsed_ms={} {work}",
            module.len(),
            start.elapsed().as_millis()
        );
        let path = info.path.clone();
        info.path.push_str(".unrelated");
        assert!(recipe.authenticate_build_info(&info).is_err());
        info.path = path;
        info.settings.insert("-tags".into(), "unrelated".into());
        assert!(recipe.authenticate_build_info(&info).is_err());
        // This does not authenticate a live Go parent or produce an accepted artifact.
    }
}
