//! Current snapshots belong to fixture passes or named live/native readers.

#![forbid(unsafe_code)]

use std::{
    collections::BTreeSet,
    fs,
    path::{Path, PathBuf},
};
use umpire_testing::TestResult;

const FIXTURES: &[&str] = &[
    "umpire/author/dsl/macros/tests/fixtures",
    "umpire/author/dsl/tests/fixtures",
    "umpire/author/suite/tests/fixtures",
    "umpire/backends/native/tests/fixtures",
    "umpire/dev/fixtures/fixtures",
    "temporal/feature",
];

const READERS: &[(&str, &str)] = &[
    (
        "temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__typed-action-registry.snap",
        "realization macros::tests::typed_registry_expansion",
    ),
    (
        "umpire/catalog/syntax/tests/snapshots/expansion__typed-catalog.snap",
        "cargo test -p umpire-catalog-syntax --test expansion executable_expansion_and_source_coverage",
    ),
    (
        "umpire/catalog/tests/snapshots/controls__generated-catalog.snap",
        "controls::generated_catalogs_retain_executable_declaration_metadata",
    ),
    (
        "temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization-authored.snap",
        "realization macros::tests::activity_expansion",
    ),
    (
        "temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization-dispatch.snap",
        "realization macros::tests::activity_expansion",
    ),
    (
        "temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization-metadata.snap",
        "realization macros::tests::activity_expansion",
    ),
    (
        "umpire/catalog/tests/snapshots/catalog__admission.snap",
        "catalog::request_admission_retains_typed_diagnostics",
    ),
    (
        "umpire/catalog/tests/snapshots/catalog__gaps.snap",
        "catalog::gap_goldens_are_derived_from_validation_and_redact_secret_values",
    ),
    (
        "umpire/catalog/tests/snapshots/catalog__kind-violations.snap",
        "catalog::kind_constraints_have_positive_and_negative_controls",
    ),
    (
        "umpire/catalog/tests/snapshots/catalog__presence-violations.snap",
        "catalog::presence_operators_validate_and_guide_fill",
    ),
    (
        "umpire/reference/kettle/expected/coverage.txt.snap",
        "probes::coverage_matches_catalog_golden; Go TestKettleAnnotationCoverageAndActualWireGaps",
    ),
    (
        "umpire/reference/kettle/expected/report.txt.snap",
        "realization::explore_retains_canonical_report; runtime dispatch/caught-panic native payload readers; Go TestKettleJudgeAnswerTranscript",
    ),
    (
        "umpire/reference/kettle/expected/gaps.txt.snap",
        "Go TestKettleAnnotationCoverageAndActualWireGaps; live_snapshots::import_live_candidates",
    ),
    (
        "umpire/reference/kettle/expected/mutants.txt.snap",
        "Go TestKettleMutants; live_snapshots::import_live_candidates",
    ),
    (
        "umpire/reference/kettle/expected/transcript.txt.snap",
        "Go TestKettleJudgeAnswerTranscript; live_snapshots::import_live_candidates",
    ),
    (
        "umpire/backends/native/tests/fixtures/sample_kettle_trait/expected/report.txt.snap",
        "runtime dispatch::explore; Go TestKettleParity",
    ),
    (
        "umpire/backends/native/tests/fixtures/sample_kettle_trait/expected/rows.txt.snap",
        "runtime dispatch::rows; Go TestKettleParity",
    ),
    (
        "umpire/backends/native/tests/fixtures/effect_panic/expected/report.txt.snap",
        "panic-control-reports native comparison",
    ),
    (
        "umpire/backends/native/tests/fixtures/effect_overflow/expected/report.txt.snap",
        "panic-control-reports native comparison",
    ),
    (
        "temporal/feature/activity/expected/report.txt.snap",
        "realization::named_guest_adapter_preserves_the_independent_report_and_ordered_rows",
    ),
    (
        "temporal/feature/activity/expected/rows.txt.snap",
        "realization::named_guest_adapter_preserves_the_independent_report_and_ordered_rows",
    ),
    (
        "temporal/feature/activity/expected/gaps.txt.snap",
        "Go TestTestpilotModelGaps; fixtures::import_live_gap_candidate (import only)",
    ),
];

const HISTORICAL: &[&str] = &[
    "coverage.snap",
    "expansion/closings.snap",
    "lift.snap",
    "realization.snap",
    "report.snap",
    "rows.snap",
    "states.snap",
];

fn collect(root: &Path, files: &mut Vec<PathBuf>) -> TestResult {
    for entry in fs::read_dir(root)? {
        let entry = entry?;
        let path = entry.path();
        if entry.file_type()?.is_dir() {
            collect(&path, files)?;
        } else if path.extension().is_some_and(|extension| extension == "snap")
            || path.to_string_lossy().ends_with(".snap.new")
        {
            files.push(path);
        }
    }
    Ok(())
}

fn validate(
    root: &Path,
    fixture_expected: &[PathBuf],
    readers: &[(&str, &str)],
    historical: &BTreeSet<PathBuf>,
) -> TestResult {
    let mut files = Vec::new();
    collect(root, &mut files)?;
    for path in files {
        if historical.contains(&path) {
            continue;
        }
        if path.to_string_lossy().ends_with(".snap.new") {
            return Err(format!("pending golden: {}", path.display()).into());
        }
        if !fixture_expected.iter().any(|expected| path.starts_with(expected))
            && !readers
                .iter()
                .any(|(relative, owner)| !owner.trim().is_empty() && path == root.join(relative))
        {
            return Err(format!("snapshot has no consumer: {}", path.display()).into());
        }
        let content = fs::read_to_string(&path)?;
        umpire_testing::snapshot_payload(&content)
            .map_err(|error| format!("{}: {error}", path.display()))?;
        insta::Snapshot::from_file(&path)?;
    }
    for (relative, owner) in readers {
        if owner.trim().is_empty() || !root.join(relative).is_file() {
            return Err(format!("missing registered snapshot: {relative}").into());
        }
    }
    Ok(())
}

#[test]
fn current_snapshots_have_explicit_owners() -> TestResult {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let mut expected = Vec::new();
    for fixtures in FIXTURES {
        expected.extend(
            umpire_testing::discovery::discover(
                &root.join(fixtures),
                umpire_testing::discovery::MODEL,
            )?
            .into_iter()
            .map(|fixture| fixture.path.join("expected")),
        );
    }
    let historical = HISTORICAL
        .iter()
        .map(|file| root.join("temporal/feature/activity/compatibility/original").join(file))
        .collect();
    validate(&root, &expected, READERS, &historical)
}

#[test]
fn nonfixture_orphan_pending_and_missing_reader_refuse() -> TestResult {
    let root =
        std::env::temp_dir().join(format!("umpire-snapshot-registry-{}", std::process::id()));
    fs::create_dir_all(root.join("outside"))?;
    let result = (|| -> TestResult {
        let owned = root.join("outside/live.snap");
        fs::write(&owned, "---\nsource: control.rs\n---\nlive\n")?;
        let readers = [("outside/live.snap", "Go TestLive; native payload reader")];
        validate(&root, &[], &readers, &BTreeSet::new())?;
        let orphan = root.join("outside/orphan.snap");
        fs::write(&orphan, "---\nsource: control.rs\n---\norphan\n")?;
        assert!(validate(&root, &[], &readers, &BTreeSet::new()).is_err());
        fs::remove_file(orphan)?;
        let pending = root.join("outside/live.snap.new");
        fs::write(&pending, "pending")?;
        assert!(
            validate(&root, &[], &readers, &BTreeSet::new())
                .err()
                .ok_or("pending accepted")?
                .to_string()
                .contains("pending golden")
        );
        fs::remove_file(pending)?;
        fs::remove_file(owned)?;
        assert!(validate(&root, &[], &readers, &BTreeSet::new()).is_err());
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
