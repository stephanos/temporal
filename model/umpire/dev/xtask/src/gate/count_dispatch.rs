//! Applicability contracts declared by the native dispatch branches.

#[cfg(test)]
#[allow(clippy::unwrap_used, clippy::expect_used)]
mod tests {
    use super::*;
    use std::cell::RefCell;

    #[test]
    fn completed_dispatch_records_modalities_before_body_and_exact_failure_status() {
        let events = RefCell::new(Vec::new());
        let code = Contract::observed()
            .run_recorded(
                "dispatch-42-7",
                |kind, values| {
                    events.borrow_mut().push(format!("{kind}\t{values}"));
                    Ok(())
                },
                || {
                    events.borrow_mut().push("body".into());
                    Ok::<_, io::Error>(37)
                },
            )
            .unwrap();
        assert_eq!(code, 37);
        assert_eq!(
            *events.borrow(),
            [
                "dispatch-begin\tdispatch-42-7\tobserved-tests\tobserved-tlc",
                "body",
                "dispatch-finish\tdispatch-42-7\t37",
            ]
        );
    }

    #[test]
    fn body_error_keeps_original_error_and_leaves_dispatch_incomplete() {
        let mut events = Vec::new();
        let error = Contract::not_applicable(NoWorkReason::NativeScan)
            .run_recorded(
                "dispatch-42-8",
                |kind, values| {
                    events.push(format!("{kind}\t{values}"));
                    Ok(())
                },
                || Err::<i32, _>(io::Error::new(io::ErrorKind::PermissionDenied, "original")),
            )
            .unwrap_err();
        assert_eq!(error.kind(), io::ErrorKind::PermissionDenied);
        assert_eq!(error.to_string(), "original");
        assert_eq!(events, ["dispatch-begin\tdispatch-42-8\tnative_scan\tnative_scan"]);
    }

    #[test]
    fn failed_begin_refuses_body_and_failed_finish_refuses_completion() {
        let mut called = false;
        let error = Contract::observed()
            .run_recorded(
                "dispatch-42-9",
                |_, _| Err(io::Error::other("begin refused")),
                || {
                    called = true;
                    Ok::<_, io::Error>(0)
                },
            )
            .unwrap_err();
        assert!(!called);
        assert_eq!(error.to_string(), "begin refused");

        let error = Contract::observed()
            .run_recorded(
                "dispatch-42-10",
                |kind, _| {
                    if kind == "dispatch-finish" { Err(io::Error::other("finish refused")) }
                    else { Ok(()) }
                },
                || Ok::<_, io::Error>(0),
            )
            .unwrap_err();
        assert_eq!(error.to_string(), "finish refused");
    }

    #[test]
    fn metric_roles_refuse_swapped_observers_and_allow_mixed_applicability() {
        assert!(Contract::new(MetricContract::ObservedTlc, MetricContract::ObservedTlc).is_err());
        assert!(Contract::new(MetricContract::ObservedTests, MetricContract::ObservedTests).is_err());
        let contract = Contract::new(
            MetricContract::ObservedTests,
            MetricContract::NotApplicable(NoWorkReason::CompileOnly),
        ).unwrap();
        assert_eq!(contract.tests(), MetricContract::ObservedTests);
        assert_eq!(contract.tlc(), MetricContract::NotApplicable(NoWorkReason::CompileOnly));
    }

    #[test]
    fn no_work_reasons_have_closed_snake_case_wire_and_serde_names() {
        for (reason, wire) in [
            (NoWorkReason::NativeScan, "native_scan"),
            (NoWorkReason::CompileOnly, "compile_only"),
            (NoWorkReason::Formatting, "formatting"),
            (NoWorkReason::ProtobufGeneration, "protobuf_generation"),
            (NoWorkReason::GuestBuild, "guest_build"),
            (NoWorkReason::ToolProvisioning, "tool_provisioning"),
            (NoWorkReason::ChildExclusive, "child_exclusive"),
            (NoWorkReason::SkippedByMode, "skipped_by_mode"),
        ] {
            assert_eq!(serde_json::to_string(&reason).unwrap(), format!("\"{wire}\""));
            assert_eq!(serde_json::from_str::<NoWorkReason>(&format!("\"{wire}\"")).unwrap(), reason);
            let mut events = Vec::new();
            Contract::not_applicable(reason).run_recorded(
                "dispatch-42-11",
                |kind, values| {
                    events.push(format!("{kind}\t{values}"));
                    Ok(())
                },
                || Ok::<_, io::Error>(0),
            ).unwrap();
            assert_eq!(events[0], format!("dispatch-begin\tdispatch-42-11\t{wire}\t{wire}"));
        }
        assert!(serde_json::from_str::<NoWorkReason>("\"unknown\"").is_err());
    }
}
