use super::*;
use std::{fs, path::Path};

const MIGRATIONS: &str =
    include_str!("../../../../internal/backends/wasm/determinism/evidence-migrations.json");
const ORIGINAL: &str = "1d695061b6382009d0659238c652a5a765b1d5f5";
const EXPECTED: &[(&str, &str, &str, &str, usize)] = &[
    (
        "TestDiagnosticTransportRequiresPrivateRevisionOptIn",
        "348de047469451285747b3e990cf0bedaa03e32546549e44e33416a60746da05",
        "internal/backends/wasm/wasi/process_test.go",
        "8c1388bac13802b5adfd962034c0517c7f950f7aa02c56a67fd44b68f3b8e823",
        17,
    ),
    (
        "TestDeterminismFixtureConditionsStayOutsideDeclaredInputs",
        "73a0a1c24bada8c2eee03beaf0c4b8872f7273ecfcdeeac90fcd5ea00c59fb8e",
        "internal/backends/wasm/wasi/stress_test.go",
        "ad9bd1e4b24be8d28e2e882e82b1f7143a92065edfa42bfa4e5eecb3e963f078",
        14,
    ),
    (
        "TestDeterminismRefuseOffPinStarted",
        "db293c31d0b7a27851c7dd206358d47d6c17082bd279366639a1106181f6c9cd",
        "internal/backends/wasm/wasi/determinism_boundary_test.go",
        "1a28d14503dd07fa76e5b506dad6050a54ffe7441bf82cf955dd25f5cef74305",
        5,
    ),
    (
        "TestDeterminismRefuseUnknownHelperFrames",
        "87e1fd1a514369dfcde2c22b14388e627cfd5afc6eed1b83436290444a40322e",
        "internal/backends/wasm/wasi/determinism_boundary_test.go",
        "d3de6ed362bcb5e12e735da70ff89f649c7c12f7b4bcf61baf086def85a51c93",
        40,
    ),
    (
        "TestDeterminismRefuseNestedHelperFields",
        "4d36b62d1dc9c64d58106b118a098e731ee78a445fbfb2b48427af9a3c6f84d9",
        "internal/backends/wasm/wasi/determinism_boundary_test.go",
        "4256fa7b1a00e166e3cdea1f7899b9a6f51c3d1319355f6729706e3c7dd9fa01",
        4,
    ),
    (
        "TestDeterminismRefuseUnknownHelperTerminal",
        "fdaaf9246326247b76e58ed1dd736fc834454a67a128371fd0b7186f556b126a",
        "internal/backends/wasm/wasi/determinism_boundary_test.go",
        "64f267419811889a782a667c7c7a7e3b239f75c5291eff173c3d5b353503395d",
        5,
    ),
    (
        "TestTransportRejectsInvalidTerminalClaims",
        "e6cfdee1a52ef87a1c6b566da3d5307d6c360d5def6419637c50b7fa05d26f3e",
        "internal/backends/wasm/wasi/process_test.go",
        "904d0920aa99b03b6cb309b1b0c6e793bfc41a1a07222eb5437b9faf4cb2bf3f",
        7,
    ),
];

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Migrations {
    schema: String,
    original_commit: String,
    authority: String,
    original_pass: bool,
    contracts: Vec<Contract>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Contract {
    legacy_reference: String,
    original_file: String,
    original_sha256: String,
    original_pass: bool,
    assertions: Vec<Assertion>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Assertion {
    id: String,
    original_assertion: String,
    disposition: String,
    reason: String,
    receiving_tests: Vec<String>,
}

// Bind original predicate custody separately from changeable receiving dispositions.
fn original_assertions_hash(assertions: &[Assertion]) -> String {
    let mut hash = Sha256::new();
    for assertion in assertions {
        for text in [&assertion.id, &assertion.original_assertion] {
            hash.update((text.len() as u64).to_le_bytes());
            hash.update(text.as_bytes());
        }
    }
    format!("{:x}", hash.finalize())
}

fn parse_migrations(text: &str, module: &Path) -> Result<Migrations, String> {
    let value: serde_json::Value = serde_json::from_str(text).map_err(|error| error.to_string())?;
    if format!("{}\n", serde_json::to_string_pretty(&value).unwrap()) != text {
        return Err("evidence migration document is not canonical".into());
    }
    let migrations: Migrations =
        serde_json::from_value(value).map_err(|error| error.to_string())?;
    if migrations.schema != "gomad.determinism-evidence-migrations/v1"
        || migrations.original_commit != ORIGINAL
        || migrations.original_pass
        || migrations.authority
            != "fn-232 Rust host retirement decision and MILESTONES.md; explicit receiving-contract migration, not original test execution"
        || migrations.contracts.len() != EXPECTED.len()
    {
        return Err("evidence migration authority or original custody mismatch".into());
    }
    let mut references = BTreeSet::new();
    for contract in &migrations.contracts {
        let name = contract
            .legacy_reference
            .strip_prefix("go:internal/backends/wasm/wasi:")
            .ok_or("migration does not name a retired Go WASI reference")?;
        let (_, hash, original_file, assertion_hash, count) = EXPECTED
            .iter()
            .find(|(expected, _, _, _, _)| *expected == name)
            .ok_or("unreviewed evidence migration reference")?;
        if contract.original_pass
            || contract.original_sha256 != *hash
            || contract.assertions.len() != *count
            || !references.insert(&contract.legacy_reference)
            || contract.original_file != *original_file
            || original_assertions_hash(&contract.assertions) != *assertion_hash
        {
            return Err("evidence migration original predicate custody mismatch".into());
        }
        let mut ids = BTreeSet::new();
        for assertion in &contract.assertions {
            if assertion.id.is_empty()
                || !ids.insert(&assertion.id)
                || assertion.original_assertion.is_empty()
                || assertion.reason.is_empty()
                || !matches!(
                    assertion.disposition.as_str(),
                    "native" | "retired-peer-wire" | "pending"
                )
                || assertion.disposition == "native" && assertion.receiving_tests.is_empty()
            {
                return Err("incomplete evidence migration assertion".into());
            }
            for reference in &assertion.receiving_tests {
                if !reference.starts_with("rust:") {
                    return Err(
                        "migration receiving evidence must name an actual native test".into(),
                    );
                }
                super::evidence_inventory::check_reference(module, reference)?;
            }
        }
    }
    Ok(migrations)
}

fn require_received(migrations: &Migrations, reference: &str) -> Result<(), String> {
    let contract = migrations
        .contracts
        .iter()
        .find(|contract| contract.legacy_reference == reference)
        .ok_or_else(|| format!("unreviewed evidence migration {reference}"))?;
    let pending: Vec<&str> = contract
        .assertions
        .iter()
        .filter(|assertion| assertion.disposition == "pending")
        .map(|assertion| assertion.id.as_str())
        .collect();
    if !pending.is_empty() {
        return Err(format!(
            "pending original assertions for {reference}: {}",
            pending.join(", ")
        ));
    }
    Ok(())
}

pub(super) fn check_reference(module: &Path, reference: &str) -> Result<(), String> {
    let migrations = parse_migrations(MIGRATIONS, module)?;
    require_received(&migrations, reference)
}

#[test]
fn evidence_migrations_bind_complete_reviewed_original_assertions_without_original_pass() {
    let module = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    parse_migrations(MIGRATIONS, &module).unwrap();
}

#[test]
fn pending_claims_and_missing_receivers_cannot_resolve_as_complete_migrations() {
    let module = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let mut migrations = parse_migrations(MIGRATIONS, &module).unwrap();
    let reference = "go:internal/backends/wasm/wasi:TestTransportRejectsInvalidTerminalClaims";
    let error = require_received(&migrations, reference).unwrap_err();
    assert!(
        error.contains("invented-output") && error.contains("excess-fuel"),
        "{error}"
    );
    let reference = migrations.contracts[0].legacy_reference.clone();
    require_received(&migrations, &reference).unwrap();
    migrations.contracts[0].assertions[0].disposition = "pending".into();
    assert!(require_received(&migrations, &reference).is_err());
    assert!(require_received(&migrations, "go:missing:TestMissing").is_err());
    let text = MIGRATIONS.replace(
        "single_native_role_matrix_refuses_before_guest_output_and_joins_every_execution",
        "removed_native_receiver",
    );
    assert!(parse_migrations(&text, &module).is_err());
    let mut value: serde_json::Value = serde_json::from_str(MIGRATIONS).unwrap();
    value["contracts"][0]["assertions"][0]["id"] = "unrelated-assertion".into();
    value["contracts"][0]["assertions"][0]["original_assertion"] = "Unrelated assertion".into();
    value["contracts"][0]["assertions"][0]["disposition"] = "retired-peer-wire".into();
    value["contracts"][0]["assertions"][0]["receiving_tests"] = serde_json::json!([]);
    let text = format!("{}\n", serde_json::to_string_pretty(&value).unwrap());
    assert!(parse_migrations(&text, &module).is_err());
    let mut value: serde_json::Value = serde_json::from_str(MIGRATIONS).unwrap();
    value["contracts"][0]["original_file"] = "internal/backends/wasm/wasi/stress_test.go".into();
    let text = format!("{}\n", serde_json::to_string_pretty(&value).unwrap());
    assert!(parse_migrations(&text, &module).is_err());
}

#[test]
fn retired_go_peer_wire_has_no_installed_native_entrypoint_and_reference_helper_stays_explicit() {
    let module = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    assert!(!module.join("internal/backends/wasm/wasi").exists());
    let helper_manifest = fs::read_to_string(module.join("rust/wasmhost/Cargo.toml")).unwrap();
    assert!(helper_manifest.contains("required-features = [\"reference-helper\"]"));
    let host_manifest = fs::read_to_string(module.join("rust/gomad/Cargo.toml")).unwrap();
    assert!(!host_manifest.contains("reference-helper"));
    let helper_main = fs::read_to_string(module.join("rust/wasmhost/src/main.rs")).unwrap();
    assert!(helper_main.contains("gomad_wasmhost::transport()"));
    let main = fs::read_to_string(module.join("rust/gomad/src/main.rs")).unwrap();
    assert!(!main.contains("gomad_wasmhost::transport"));
    let execute = fs::read_to_string(module.join("rust/gomad/src/execute.rs")).unwrap();
    assert!(execute.contains("gomad_wasmhost::linked::execute(request)"));
    let campaign = fs::read_to_string(module.join("rust/gomad/src/campaign/execution.rs")).unwrap();
    assert!(campaign.contains("linked::execute_prepared(admitted)"));
    let linked = fs::read_to_string(module.join("rust/wasmhost/src/linked.rs")).unwrap();
    assert!(linked.contains("Box::new(NoRemote"));
    assert!(linked.contains("linked model forbids external owners or caller-projected tapes"));
}
