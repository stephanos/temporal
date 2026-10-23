use super::*;

#[test]
fn original_namespace_and_workflow_success_witnesses_are_required() {
    let workflow = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$";
    let valid = b"=== RUN   TestWorkflowTypeEncodingSuite\n--- PASS: TestWorkflowTypeEncodingSuite\nTestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds\n";
    selected_workload_witness(valid, workflow).unwrap();
    for invalid in [
        b"=== RUN   TestWorkflowTypeEncodingSuite\n--- FAIL: TestWorkflowTypeEncodingSuite\nTestPlainASCII/Succeeds".as_slice(),
        b"=== RUN   TestWorkflowTypeEncodingSuite\nTestPlainASCII/Succeeds",
        b"=== RUN   TestWorkflowTypeEncodingSuite\n--- PASS: TestWorkflowTypeEncodingSuite",
        b"PASS",
    ] {
        assert!(selected_workload_witness(invalid, workflow).is_err());
    }
    selected_workload_witness(
        b"=== RUN   TestActivityTestSuite\n--- PASS: TestActivityTestSuite",
        "^TestActivityTestSuite$",
    )
    .unwrap();
    for invalid in [
        b"=== RUN   TestActivityTestSuite\n--- FAIL: TestActivityTestSuite".as_slice(),
        b"PASS",
    ] {
        assert!(selected_workload_witness(invalid, "^TestActivityTestSuite$").is_err());
    }
    selected_workload_witness(b"runtime-cost 100000\n", "runtime").unwrap();
}
