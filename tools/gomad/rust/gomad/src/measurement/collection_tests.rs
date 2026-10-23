use super::FailureReasons;

#[test]
fn failed_collection_step_allows_next_and_final_report_keeps_both_errors() {
    let mut failures = FailureReasons::default();
    let mut reports = Vec::new();
    reports.push("failed-sample");
    let failed = failures.observe(Err::<(), _>("collection failure".into()));
    assert_eq!(failed, Err("collection failure".into()));
    reports.push("next-sample");
    failures.observe(Ok(())).unwrap();
    let result = failures.publish(|failed| -> Result<(), String> {
        assert_eq!(failed, Some("collection failure"));
        reports.push("final");
        Err("report failure".into())
    });
    assert_eq!(result, Err("collection failure; report failure".into()));
    assert_eq!(reports, ["failed-sample", "next-sample", "final"]);
}

#[test]
fn successful_report_keeps_collection_failure_for_explicit_producer_policy() {
    let mut failures = FailureReasons::default();
    let _ = failures.observe(Err::<(), _>("sample failure".into()));
    assert_eq!(
        failures.publish(|failure| {
            assert_eq!(failure, Some("sample failure"));
            Ok("retained-report")
        }),
        Ok("retained-report")
    );
    assert_eq!(failures.message().as_deref(), Some("sample failure"));
}

#[test]
fn warm_preparation_failure_reasons_keep_original_order_without_duplicates() {
    let mut failures = FailureReasons::default();
    assert_eq!(failures.message(), None);
    failures.push("");
    failures.push("host bindings changed during collection");
    failures.push("baseline-material-gap");
    failures.push("baseline-material-gap");
    failures.push("host bindings changed during collection");
    assert_eq!(
        failures.message().as_deref(),
        Some("host bindings changed during collection; baseline-material-gap")
    );
    failures.push("material-gap");
    assert_eq!(
        failures.message().as_deref(),
        Some("host bindings changed during collection; baseline-material-gap; material-gap")
    );
}

#[test]
fn publication_failure_deduplicates_existing_reason_and_records_new_failure() {
    let mut failures = FailureReasons::default();
    failures.push("report failure");
    let result = failures.publish(|_| Err::<(), _>("report failure".into()));
    assert_eq!(result, Err("report failure".into()));
    let result = failures.publish(|failure| {
        assert_eq!(failure, Some("report failure"));
        Err::<(), _>("cleanup failure".into())
    });
    assert_eq!(result, Err("report failure; cleanup failure".into()));
}
