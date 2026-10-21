//! Missing fixture inputs and outputs must never disappear from discovery.

use std::{collections::BTreeMap, error::Error, fs};

#[test]
fn refuses_incomplete_fixtures() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-discovery-{}", std::process::id()));
    fs::create_dir_all(&root)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        for missing in ["model.rs", "tests.rs", "expected"] {
            let fixture = root.join(format!("missing_{}", missing.replace('.', "_")));
            fs::create_dir(&fixture)?;
            for present in ["model.rs", "tests.rs"] {
                if present != missing {
                    fs::write(fixture.join(present), "")?;
                }
            }
            if missing != "expected" {
                fs::create_dir(fixture.join("expected"))?;
            }
            let error = umpire_testing::discovery::generate(
                &root,
                umpire_testing::discovery::MODEL_AND_TESTS,
                &root.join("generated.rs"),
            )
            .err()
            .ok_or("incomplete fixture was silently accepted")?;
            let message = error.to_string();
            assert!(message.contains(missing), "{message}");
            assert!(message.contains(&*fixture.to_string_lossy()), "{message}");
            fs::remove_dir_all(fixture)?;
        }
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn compiles_distinct_non_identifier_folder_names() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-discovery-names-{}", std::process::id()));
    let fixtures = root.join("fixtures");
    fs::create_dir_all(&fixtures)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        for name in [
            "kettle-cut",
            "kettle_cut",
            "kettle_2dcut",
            "fixture_kettle_2dcut",
            "kettle_cut_tests",
            "Kettle",
            "4",
            "λ",
            "type",
        ] {
            let folder = fixtures.join(name);
            fs::create_dir_all(folder.join("expected"))?;
            fs::write(folder.join("model.rs"), "")?;
            fs::write(folder.join("tests.rs"), "pub(crate) fn run() {}\n")?;
        }
        let generated = root.join("generated.rs");
        umpire_testing::discovery::generate(
            &fixtures,
            umpire_testing::discovery::MODEL_AND_TESTS,
            &generated,
        )?;
        let source = root.join("lib.rs");
        fs::write(
            &source,
            "#![allow(dead_code)]\nmod umpire_testing { pub type Fixture = (&'static str, fn()); }\ninclude!(\"generated.rs\");\n",
        )?;
        let output = std::process::Command::new("rustc")
            .args(["--edition=2024", "--crate-type=lib", "--emit=metadata", "--out-dir"])
            .arg(&root)
            .arg(&source)
            .output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn overrides_preserve_module_identity_and_tests_lexical_scope() -> Result<(), Box<dyn Error>> {
    let root =
        std::env::temp_dir().join(format!("umpire-discovery-overrides-{}", std::process::id()));
    let fixtures = root.join("fixtures");
    fs::create_dir_all(&fixtures)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        let model = "pub mod machine {\n    pub struct World { limit: u8 }\n    impl World { pub fn new(limit: u8) -> Self { Self { limit } } }\n}\n";
        let tests = "use crate::alpha::machine::{self, World as Selected};\nfn selected() -> Selected { Selected::new(17) }\npub(crate) fn run() -> u8 { 0 }\n";
        for (name, model, tests) in
            [("beta", "", "pub(crate) fn run() -> u8 { 9 }\n"), ("alpha", model, tests)]
        {
            let folder = fixtures.join(name);
            fs::create_dir_all(folder.join("expected"))?;
            fs::write(folder.join("model.rs"), model)?;
            fs::write(folder.join("tests.rs"), tests)?;
        }
        let generated_model = root.join("alpha-model.rs");
        fs::write(
            &generated_model,
            model.replace(
                "\n}\n",
                "\n    pub fn projection(world: &World) -> u8 { world.limit }\n}\n",
            ),
        )?;
        let generated_tests = root.join("alpha-tests.rs");
        fs::write(
            &generated_tests,
            tests.replace("{ 0 }", "{ machine::projection(&selected()) }"),
        )?;
        let inputs = umpire_testing::discovery::MODEL_AND_TESTS;
        let default = root.join("default.rs");
        let empty = root.join("empty.rs");
        umpire_testing::discovery::generate(&fixtures, inputs, &default)?;
        umpire_testing::discovery::generate_with_overrides(
            &fixtures,
            inputs,
            &empty,
            &BTreeMap::new(),
        )?;
        assert_eq!(fs::read(default)?, fs::read(empty)?);
        let overrides = BTreeMap::from([
            (("alpha".into(), "model.rs".into()), generated_model),
            (("alpha".into(), "tests.rs".into()), generated_tests),
        ]);
        umpire_testing::discovery::generate_with_overrides(
            &fixtures,
            inputs,
            &root.join("generated.rs"),
            &overrides,
        )?;
        let source = root.join("main.rs");
        fs::write(
            &source,
            "#![allow(dead_code)]\nmod umpire_testing { pub type Fixture = (&'static str, fn() -> u8); }\ninclude!(\"generated.rs\");\nfn main() { let results: Vec<_> = registered().into_iter().map(|(name, run)| (name, run())).collect(); assert_eq!(results, vec![(\"alpha\", 17), (\"beta\", 9)]); }\n",
        )?;
        let binary = root.join("control");
        let output = std::process::Command::new("rustc")
            .args(["--edition=2024", "-o"])
            .arg(&binary)
            .arg(source)
            .output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        let output = std::process::Command::new(binary).output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}

#[test]
fn overrides_refuse_unknown_inputs_and_still_require_originals() -> Result<(), Box<dyn Error>> {
    let root =
        std::env::temp_dir().join(format!("umpire-discovery-refusals-{}", std::process::id()));
    let fixtures = root.join("fixtures");
    let folder = fixtures.join("sample");
    fs::create_dir_all(folder.join("expected"))?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        for input in ["model.rs", "tests.rs"] {
            fs::write(folder.join(input), "")?;
        }
        let generated = root.join("copy.rs");
        fs::write(&generated, "")?;
        let inputs = umpire_testing::discovery::MODEL_AND_TESTS;
        for (name, input, path, reason) in [
            ("unknown", "model.rs", generated.clone(), "unknown fixture input override"),
            ("sample", "unknown.rs", generated.clone(), "unknown fixture input override"),
            ("sample", "model.rs", root.join("absent.rs"), "override is not a file"),
            ("sample", "model.rs", folder.clone(), "override is not a file"),
        ] {
            let overrides = BTreeMap::from([((name.into(), input.into()), path)]);
            let error = umpire_testing::discovery::generate_with_overrides(
                &fixtures,
                inputs,
                &root.join("output.rs"),
                &overrides,
            )
            .err()
            .ok_or("invalid override was silently accepted")?;
            assert!(error.to_string().contains(reason), "{error}");
        }
        for missing in inputs {
            fs::remove_file(folder.join(missing))?;
            let overrides =
                BTreeMap::from([(("sample".into(), (*missing).into()), generated.clone())]);
            let error = umpire_testing::discovery::generate_with_overrides(
                &fixtures,
                inputs,
                &root.join("output.rs"),
                &overrides,
            )
            .err()
            .ok_or("generated copy hid missing authored input")?;
            assert!(error.to_string().contains(&format!("missing {missing}")), "{error}");
            fs::write(folder.join(missing), "")?;
        }
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
