use super::{
    BTreeMap, BTreeSet, Path, PathBuf, RetainedSession, Value, custody, digest, require, text,
};
use crate::cycle::build::helper_identity as implementation;
use serde_json::json;

const EXTRA: &[&str] = &[
    "build.rs",
    "build_identity.rs",
    "Cargo.toml",
    "../Cargo.toml",
    "../Cargo.lock",
    "../rust-toolchain.toml",
];
fn source_names(
    root: &Path,
    path: &Path,
    files: &mut BTreeMap<String, PathBuf>,
) -> Result<(), String> {
    require(files.len() < 200_000, "identity source inventory bound")?;
    for entry in std::fs::read_dir(path).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        let kind = entry.file_type().map_err(|e| e.to_string())?;
        require(!kind.is_symlink(), "identity source tree contains aliases")?;
        if kind.is_dir() {
            source_names(root, &entry.path(), files)?;
        } else if kind.is_file() && entry.path().extension().is_some_and(|v| v == "rs") {
            files.insert(
                entry
                    .path()
                    .strip_prefix(root)
                    .map_err(|e| e.to_string())?
                    .to_string_lossy()
                    .into_owned(),
                entry.path(),
            );
        }
    }
    Ok(())
}
impl RetainedSession {
    #[expect(
        clippy::too_many_lines,
        reason = "One identity transcript binds every case to the same frozen closure"
    )]
    pub(crate) fn core_identity(&self, stdout: &[u8], context: &Value) -> Result<Value, String> {
        require(
            context.as_object().is_some_and(|v| v.len() == 2)
                && context["bindings"]
                    .as_object()
                    .is_some_and(|v| v.keys().map(String::as_str).eq(["PROFILE", "TARGET"]))
                && context["bindings"]["PROFILE"] == "release"
                && context["bindings"]["TARGET"]
                    .as_str()
                    .is_some_and(|v| !v.is_empty())
                && text(context, "compiler_utf8")?.starts_with("rustc 1.99.1 "),
            "foreign identity compiled context",
        )?;
        let root = Path::new(text(&self.config, "repo")?).join("tools/gomad/rust/wasmhost");
        let mut names = BTreeMap::new();
        source_names(&root, &root.join("src"), &mut names)?;
        require(!names.is_empty(), "identity source closure empty")?;
        for name in EXTRA {
            names.insert(
                (*name).into(),
                root.join(name).canonicalize().map_err(|e| e.to_string())?,
            );
        }
        let mut files = BTreeMap::new();
        let mut inventory = Vec::new();
        let mut total = 0u64;
        for (name, path) in names {
            let raw = custody::read_file(&path, 128 << 20)?;
            total = total
                .checked_add(raw.len() as u64)
                .ok_or("identity bytes overflow")?;
            require(
                total <= 512 << 20
                    && self.config["files"][path.to_string_lossy().as_ref()] == digest(&raw),
                "identity source not frozen/oversized",
            )?;
            inventory.push(json!({"name":name,"path":root.join(&name),"bytes":raw.len(),"sha256":digest(&raw)}));
            files.insert(name, raw);
        }
        let bindings: BTreeMap<String, String> =
            serde_json::from_value(context["bindings"].clone()).map_err(|e| e.to_string())?;
        let before = implementation::implementation(
            &files,
            &bindings,
            text(context, "compiler_utf8")?.as_bytes(),
        )?;
        let required = implementation::REQUIRED_OWNERS;
        let groups = [
            ("identity.storage-owner", &required[1..10]),
            (
                "identity.inspection-owner",
                &[
                    "src/linked_inspection.rs",
                    "src/linked_storage_inspection.rs",
                    "src/model/inspection.rs",
                    "src/cluster/network/evidence.rs",
                ][..],
            ),
            (
                "identity.strict-json-owner",
                &["src/protocol.rs", "src/model.rs", "src/linked_storage.rs"][..],
            ),
        ];
        let mut expected = BTreeSet::from([
            ("identity.compiled-native".to_owned(), None),
            ("identity.empty-closure".to_owned(), None),
        ]);
        for (case, owners) in groups {
            for owner in owners {
                expected.insert((case.to_owned(), Some((*owner).to_owned())));
            }
        }
        let mut seen = BTreeSet::new();
        for line in std::str::from_utf8(stdout)
            .map_err(|e| e.to_string())?
            .lines()
        {
            let Some((_, raw)) = line.split_once("GOMAD_CORE_OBSERVATION ") else {
                continue;
            };
            require(raw.len() <= 1 << 20, "identity observation byte bound")?;
            let row = custody::strict_json(raw.as_bytes())?;
            let inputs = &row["inputs"];
            let observed = &row["observed"];
            let case = text(&row, "case")?;
            let owner = inputs.get("owner").and_then(Value::as_str);
            require(
                row.as_object().is_some_and(|v| v.len() == 4)
                    && row["schema"] == "gomad.core-predicate-observation/v1"
                    && expected.contains(&(case.into(), owner.map(str::to_owned)))
                    && seen.insert((case.to_owned(), owner.map(str::to_owned))),
                "unknown/duplicate identity case or owner",
            )?;
            let input_keys = if owner.is_some() {
                vec![
                    "bindings",
                    "compiler_utf8",
                    "files",
                    "mutation_append_utf8",
                    "owner",
                    "required_owners",
                ]
            } else {
                vec!["bindings", "compiler_utf8", "files", "required_owners"]
            };
            require(
                inputs
                    .as_object()
                    .is_some_and(|v| v.keys().map(String::as_str).eq(input_keys)),
                "identity input fields changed",
            )?;
            require(
                inputs["bindings"] == context["bindings"]
                    && inputs["compiler_utf8"] == context["compiler_utf8"]
                    && inputs["required_owners"] == json!(required),
                "identity substituted compiled context",
            )?;
            require(
                inputs["files"]
                    == if case == "identity.empty-closure" {
                        json!([])
                    } else {
                        json!(inventory)
                    },
                "identity file closure differs from frozen source",
            )?;
            if case == "identity.compiled-native" {
                require(
                    observed == &json!({"compiled":before,"recomputed":before}),
                    "compiled identity differs from actual context",
                )?;
            } else if case == "identity.empty-closure" {
                require(
                    observed == &json!({"error":"missing implementation owner src/lib.rs"}),
                    "empty identity closure did not refuse",
                )?;
            } else {
                let owner = owner.ok_or("identity mutation owner missing")?;
                require(
                    inputs["mutation_append_utf8"] == "changed implementation",
                    "identity perturbation changed",
                )?;
                let mut changed = files.clone();
                changed
                    .get_mut(owner)
                    .ok_or("identity owner absent")?
                    .extend_from_slice(b"changed implementation");
                let after = implementation::implementation(
                    &changed,
                    &bindings,
                    text(context, "compiler_utf8")?.as_bytes(),
                )?;
                require(
                    after != before
                        && observed
                            == &json!({"before":before,"after":after,"missing_error":format!("missing implementation owner {owner}")}),
                    "identity mutation/missing owner differs from raw source",
                )?;
            }
        }
        require(
            seen == expected && seen.len() == 18,
            "identity omitted exact eighteen cases",
        )?;
        Ok(json!({"implementation_sha256":before,"case_count":seen.len()}))
    }
}
