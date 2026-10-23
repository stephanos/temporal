use crate::{
    campaign::{
        origin,
        prepare::{self, Prepared},
    },
    evidence::canonical,
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Marker {
    schema: String,
    module_sha256: String,
    installation_sha256: String,
    invocation_sha256: String,
    catalog_sha256: String,
    recipe_sha256: String,
}

fn build_command(invocation: &origin::Invocation) -> bool {
    let mut index = 1;
    while let Some(argument) = invocation.driver_argv.get(index) {
        if argument == "2d43" {
            index += 2;
        } else if argument.starts_with("2d433d") {
            index += 1;
        } else {
            return argument == &origin::argument(std::ffi::OsStr::new("build"));
        }
    }
    false
}

impl Marker {
    fn validate(
        &self,
        prepared: &Prepared,
        installation: &str,
        invocation: &origin::Invocation,
    ) -> Result<(), String> {
        invocation.validate()?;
        let provenance: Value =
            serde_json::from_slice(&prepared.provenance).map_err(|e| e.to_string())?;
        let bindings: Vec<(String, String)> =
            serde_json::from_value(provenance["inputs"]["shared_driver_installations"].clone())
                .map_err(|e| e.to_string())?;
        let mut names = std::collections::BTreeSet::new();
        if bindings.is_empty()
            || bindings.len() > 65536
            || bindings.iter().any(|(name, digest)| {
                name.is_empty() || !names.insert(name) || !canonical::valid_digest(digest)
            })
            || canonical::digest(&canonical::encode(&bindings)?) != invocation.installation_sha256
            || !bindings.iter().any(|(name, digest)| {
                provenance["inputs"]["go_build_info"]["path"] == *name && digest == installation
            })
        {
            return Err(
                "build launcher selected installation differs from retained catalog".into(),
            );
        }
        let registration: origin::DriverRegistration =
            serde_json::from_value(provenance["inputs"]["build_driver_registration"].clone())
                .map_err(|e| e.to_string())?;
        if registration.schema != "gomad.driver-registration/v1"
            || registration.process_id == 0
            || registration.invocation_sha256 != self.invocation_sha256
            || registration.compiler_sha256 != invocation.compiler_sha256
            || registration.driver_argv_sha256
                != canonical::digest(&canonical::encode(&invocation.driver_argv)?)
            || !registration.module_root.is_absolute()
        {
            return Err("build launcher original compiler registration mismatch".into());
        }
        if self.schema != "gomad.build-launcher/v1"
            || self.module_sha256 != canonical::digest(&prepared.module)
            || self.installation_sha256 != installation
            || !canonical::valid_digest(installation)
            || self.invocation_sha256 != canonical::digest(&canonical::encode(invocation)?)
            || self.catalog_sha256 != invocation.installation_sha256
            || self.recipe_sha256 != invocation.recipe_sha256
            || !build_command(invocation)
        {
            return Err(
                "build launcher module, installation or original driver commitment mismatch".into(),
            );
        }
        Ok(())
    }
}

pub(crate) fn seal(
    prepared: &mut Prepared,
    installation: &str,
    invocation: &origin::Invocation,
) -> Result<(), String> {
    let mut provenance: Value =
        serde_json::from_slice(&prepared.provenance).map_err(|e| e.to_string())?;
    if provenance.get("origin").is_some() || provenance["inputs"].get("build_launcher").is_some() {
        return Err("build launcher must be sealed before intercepted origin".into());
    }
    let marker = Marker {
        schema: "gomad.build-launcher/v1".into(),
        module_sha256: canonical::digest(&prepared.module),
        installation_sha256: installation.into(),
        invocation_sha256: canonical::digest(&canonical::encode(invocation)?),
        catalog_sha256: invocation.installation_sha256.clone(),
        recipe_sha256: invocation.recipe_sha256.clone(),
    };
    marker.validate(prepared, installation, invocation)?;
    provenance["inputs"]["build_launcher"] =
        serde_json::to_value(marker).map_err(|e| e.to_string())?;
    rebind(prepared, provenance, None)
}

/// Ordinary receipts retain their exact argument binding and are left untouched.
pub(crate) fn admit(
    prepared: &mut Prepared,
    module: &[u8],
    arguments: &[String],
    installation: &str,
) -> Result<bool, String> {
    let mut provenance: Value =
        serde_json::from_slice(&prepared.provenance).map_err(|e| e.to_string())?;
    let Some(value) = provenance["inputs"].get("build_launcher") else {
        return Ok(false);
    };
    let marker: Marker = serde_json::from_value(value.clone()).map_err(|e| e.to_string())?;
    let bound = origin::from_preparation(prepared)?
        .ok_or("build launcher requires sealed intercepted origin")?;
    marker.validate(prepared, installation, &bound.invocation)?;
    if module != prepared.module.as_slice() {
        return Err("build launcher module differs from its prepared receipt".into());
    }
    provenance
        .as_object_mut()
        .ok_or("invalid build launcher provenance")?
        .remove("origin");
    rebind(prepared, provenance, Some(arguments))?;
    origin::intercept(prepared, bound.invocation)?;
    Ok(true)
}

fn rebind(
    prepared: &mut Prepared,
    mut provenance: Value,
    arguments: Option<&[String]>,
) -> Result<(), String> {
    if let Some(arguments) = arguments {
        prepared.configuration.arguments = std::iter::once("gomad-target".into())
            .chain(arguments.iter().cloned())
            .collect();
        provenance["inputs"]["arguments"] = json!(arguments);
    }
    let mut configuration =
        serde_json::to_value(&prepared.configuration).map_err(|e| e.to_string())?;
    configuration["runtime_sha256"] = Value::Null;
    provenance["inputs"]["configuration"] = configuration;
    let key = canonical::digest(&canonical::encode(&provenance["inputs"])?);
    if let Some(identity) = &mut prepared.choice_identity {
        let implementation = prepare::choice_implementation(&key)?;
        identity.toolchain_build_key.clone_from(&key);
        identity.implementation_sha256 = implementation.into();
        prepared.configuration.runtime_sha256 = Some(format!("{implementation:x}"));
    } else if prepared.configuration.profile == "gomad.wasi-cooperative/v1" {
        return Err("build launcher cooperative choice identity missing".into());
    }
    prepared.toolchain.clone_from(&key);
    provenance["build_key"] = json!(key);
    provenance["configuration"] =
        serde_json::to_value(&prepared.configuration).map_err(|e| e.to_string())?;
    provenance["choice_identity"] =
        serde_json::to_value(&prepared.choice_identity).map_err(|e| e.to_string())?;
    prepare::typed_provenance(&provenance)?;
    prepared.provenance = canonical::encode(&provenance)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use sha2::Digest as _;
    use std::ffi::OsStr;

    fn invocation() -> origin::Invocation {
        origin::Invocation {
            schema: "gomad.intercepted-invocation/v1".into(),
            wrapper_run_id: "1".repeat(64),
            id: "2".repeat(64),
            original_argv: vec![
                origin::argument(OsStr::new("go")),
                origin::argument(OsStr::new("build")),
            ],
            driver_argv: vec![
                origin::argument(OsStr::new("/stock/go")),
                origin::argument(OsStr::new("build")),
            ],
            compiler_sha256: "3".repeat(64),
            installation_sha256: canonical::digest(
                &canonical::encode(&vec![("fixture-main", "6".repeat(64))]).unwrap(),
            ),
            recipe_sha256: "5".repeat(64),
            recipe_file: "driver-recipe.json".into(),
        }
    }

    fn prepared(cooperative: bool) -> Prepared {
        let module = b"\0asm\x01\0\0\0".to_vec();
        let configuration: crate::replay::preflight::Configuration = serde_json::from_value(json!({
            "engine":gomad_wasmhost::linked::Conditions::compiled().configuration(),
            "profile": if cooperative {"gomad.wasi-cooperative/v1"} else {"gomad.wasi-stock-exploratory/v1"},
            "limits":{"fuel":1000,"memory_bytes":1_048_576,"wall_millis":1000,"output_bytes":1_048_576,"evidence_bytes":1_048_576},
            "arguments":["gomad-target"],"environment":{},"runtime_sha256":null,"model":null,"namespace":null,"storage":null
        })).unwrap();
        let invocation = invocation();
        let inputs = json!({"kind":"go-run","source":"fixture-main","arguments":[],"configuration":configuration,
            "shared_driver_installations":[["fixture-main","6".repeat(64)]],"go_build_info":{"path":"fixture-main"},
            "build_driver_registration":{"schema":"gomad.driver-registration/v1","process_id":7,
                "invocation_sha256":canonical::digest(&canonical::encode(&invocation).unwrap()),
                "compiler_sha256":invocation.compiler_sha256,"driver_argv_sha256":canonical::digest(&canonical::encode(&invocation.driver_argv).unwrap()),"module_root":"/private/outputs"}});
        let toolchain = canonical::digest(&canonical::encode(&inputs).unwrap());
        let choice_identity = cooperative.then(|| gomad_wasmhost::linked::ChoiceIdentity {
            target_sha256: sha2::Sha256::digest(&module).into(),
            toolchain_build_key: toolchain.clone(),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [0; 32],
        });
        let provenance = canonical::encode(&json!({
            "schema":"gomad.rust-wasm-provenance/v1","inputs":inputs,"build_key":toolchain,
            "module_sha256":canonical::digest(&module),"configuration":configuration,"choice_identity":choice_identity,"scenario":null,"network":null
        })).unwrap();
        Prepared {
            compile_cache_root: None,
            module,
            configuration,
            toolchain,
            choice_identity,
            scenario: None,
            provenance,
            network: None,
            soft_assertion_report_only: false,
        }
    }

    #[test]
    fn ordinary_receipt_is_untouched_and_build_receipt_rebinds_only_authenticated_inputs() {
        for cooperative in [false, true] {
            let mut ordinary = prepared(cooperative);
            let original = ordinary.provenance.clone();
            let module = ordinary.module.clone();
            assert!(!admit(&mut ordinary, &module, &["new".into()], &"6".repeat(64)).unwrap());
            assert_eq!(ordinary.provenance, original);
            let invocation = invocation();
            seal(&mut ordinary, &"6".repeat(64), &invocation).unwrap();
            origin::intercept(&mut ordinary, invocation.clone()).unwrap();
            let sealed = ordinary.provenance.clone();
            assert!(admit(&mut ordinary, b"other", &[], &"6".repeat(64)).is_err());
            assert!(admit(&mut ordinary, &module, &[], &"7".repeat(64)).is_err());
            assert_eq!(ordinary.provenance, sealed);
            let before = ordinary.toolchain.clone();
            assert!(
                admit(
                    &mut ordinary,
                    &module,
                    &["arg".into(), "--flag".into()],
                    &"6".repeat(64)
                )
                .unwrap()
            );
            assert_eq!(
                ordinary.configuration.arguments,
                ["gomad-target", "arg", "--flag"]
            );
            assert_ne!(ordinary.toolchain, before);
            let bound = origin::from_preparation(&ordinary).unwrap().unwrap();
            assert_eq!(bound.invocation, invocation);
            if let Some(identity) = &ordinary.choice_identity {
                assert_eq!(identity.toolchain_build_key, ordinary.toolchain);
                assert_eq!(
                    ordinary.configuration.runtime_sha256.as_deref(),
                    Some(identity.runtime_implementation_sha256().as_str())
                );
            }
        }
    }

    #[test]
    fn build_marker_commits_original_invocation_and_selected_installation() {
        let original = invocation();
        let marker = Marker {
            schema: "gomad.build-launcher/v1".into(),
            module_sha256: canonical::digest(b"module"),
            installation_sha256: "6".repeat(64),
            invocation_sha256: canonical::digest(&canonical::encode(&original).unwrap()),
            catalog_sha256: original.installation_sha256.clone(),
            recipe_sha256: original.recipe_sha256.clone(),
        };
        let mut value = serde_json::to_value(marker).unwrap();
        value["extra"] = json!(true);
        assert!(serde_json::from_value::<Marker>(value).is_err());
    }
}
