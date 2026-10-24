use super::{Binding, LeafBindings, LeafCommand, named};

pub(super) fn derive(label: &str, b: &LeafBindings) -> Result<Vec<LeafCommand>, String> {
    let mut rows = vec![
        invocation(b, "helper-lib", "helper-lib", "", false, false)?,
        invocation(
            b,
            "host-inspection-scale",
            "host-lib",
            "replay::inspection::predicate_tests::native_retained_scale_admission_preserves_complete_inspection_and_owner_payloads",
            true,
            true,
        )?,
    ];
    for (name, selector) in HOST_SELECTORS {
        rows.push(invocation(b, name, "host-lib", selector, false, false)?);
    }
    for (name, selector) in [
        (
            "helper-world-probe",
            "world::owner_preservation_tests::world_owner_codec_measured_cardinality",
        ),
        (
            "helper-compressed-world-probe",
            "world::owner_preservation_tests::compressed_world_owner_measured_cardinality",
        ),
        (
            "helper-compressed-world-large-probe",
            "world::owner_preservation_tests::compressed_world_owner_measured_cardinality",
        ),
    ] {
        rows.push(invocation(b, name, "helper-lib", selector, true, true)?);
    }
    rows.push(invocation(
        b,
        "helper-allocation",
        "helper-allocation",
        "allocation_receiving",
        true,
        true,
    )?);
    rows.push(invocation(
        b,
        "host-allocation",
        "host-allocation",
        "allocation_receiving",
        true,
        true,
    )?);
    rows.push(invocation(
        b,
        "identity",
        "implementation-identity",
        "",
        false,
        false,
    )?);
    if label != "cluster-functional" {
        rows.push(invocation(
            b,
            "host-broad-guest",
            "host-lib",
            "replay::cluster::broad_guest_tests",
            true,
            false,
        )?);
    }
    Ok(rows)
}
fn invocation(
    b: &LeafBindings,
    name: &str,
    role: &str,
    selector: &str,
    ignored: bool,
    exact: bool,
) -> Result<LeafCommand, String> {
    let mut row = named(
        name,
        &[&b.join(Binding::Root, &format!("broad-harness/admitted/{role}"))?],
    );
    if ignored && exact {
        row.argv.push("--ignored".into());
    }
    if exact {
        row.argv.push("--exact".into());
    }
    if !selector.is_empty() {
        row.argv.push(selector.into());
    }
    if ignored && !exact {
        row.argv.push("--ignored".into());
    }
    row.argv
        .extend(super::strings(&["--nocapture", "--test-threads=1"]));
    Ok(row)
}

const HOST_SELECTORS: &[(&str, &str)] = &[
    ("host-retained", "replay::cluster::retained_tests"),
    ("host-presence", "replay::cluster::retained_presence_tests"),
    (
        "host-lifecycle-matrix",
        "replay::cluster::retained_lifecycle_matrix_tests",
    ),
    ("host-projection", "replay::cluster::projection_tests"),
    ("host-scenario", "campaign::scenario::preservation_tests"),
    ("host-fixture-cluster", "fixture_cluster"),
    (
        "host-writer",
        "evidence::store::publish::write_failure_tests",
    ),
    ("host-inspection", "replay::inspection::"),
    ("host-stress-reach", "stress::reach_tests"),
    ("host-stress", "stress::tests"),
];
