//! Applicability contracts declared by the native dispatch branches.
pub use super::work_counts::NoWorkReason;
use std::{
    io,
    sync::atomic::{AtomicU64, Ordering},
};
use umpire_backend_process::work;

static NEXT_DISPATCH: AtomicU64 = AtomicU64::new(0);

impl NoWorkReason {
    const fn wire(self) -> &'static str {
        match self {
            Self::NativeScan => "native_scan",
            Self::CompileOnly => "compile_only",
            Self::Formatting => "formatting",
            Self::ProtobufGeneration => "protobuf_generation",
            Self::GuestBuild => "guest_build",
            Self::ToolProvisioning => "tool_provisioning",
            Self::ChildExclusive => "child_exclusive",
            Self::SkippedByMode => "skipped_by_mode",
        }
    }
}

/// Evidence required from a dispatch for one metric.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum MetricContract {
    /// An actual test producer must report its inventory and outcome.
    ObservedTests,
    /// Actual process observations must establish TLC coverage.
    ObservedTlc,
    /// This dispatch cannot perform work for the metric.
    NotApplicable(NoWorkReason),
}
impl MetricContract {
    const fn wire(self) -> &'static str {
        match self {
            Self::ObservedTests => "observed-tests",
            Self::ObservedTlc => "observed-tlc",
            Self::NotApplicable(reason) => reason.wire(),
        }
    }
}

/// The metric roles declared at an actual native dispatch branch.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct Contract {
    tests: MetricContract,
    tlc: MetricContract,
}
impl Contract {
    /// Admit a contract without allowing a test observer in the TLC role or the reverse.
    pub fn new(tests: MetricContract, tlc: MetricContract) -> io::Result<Self> {
        if tests == MetricContract::ObservedTlc || tlc == MetricContract::ObservedTests {
            return Err(io::Error::other("invalid dispatch metric role"));
        }
        Ok(Self { tests, tlc })
    }

    /// Require actual test and TLC observations; this declaration proves neither count.
    pub const fn observed() -> Self {
        Self { tests: MetricContract::ObservedTests, tlc: MetricContract::ObservedTlc }
    }

    /// Declare that neither metric applies to this dispatch.
    pub const fn not_applicable(reason: NoWorkReason) -> Self {
        Self {
            tests: MetricContract::NotApplicable(reason),
            tlc: MetricContract::NotApplicable(reason),
        }
    }

    /// The test producer requirement.
    pub const fn tests(self) -> MetricContract {
        self.tests
    }

    /// The TLC observation requirement.
    pub const fn tlc(self) -> MetricContract {
        self.tlc
    }

    /// Bracket the entire dispatch; errors leave it incomplete for the collector to refuse.
    /// Outside a work scope, execute the callback unchanged.
    pub fn run<E: From<io::Error>>(self, body: impl FnOnce() -> Result<i32, E>) -> Result<i32, E> {
        if work::directory()?.is_none() {
            return body();
        }
        let id = format!(
            "dispatch-{}-{}",
            std::process::id(),
            NEXT_DISPATCH.fetch_add(1, Ordering::Relaxed)
        );
        self.run_recorded(&id, work::record, body)
    }

    fn run_recorded<E: From<io::Error>>(
        self,
        id: &str,
        mut record: impl FnMut(&str, &str) -> io::Result<()>,
        body: impl FnOnce() -> Result<i32, E>,
    ) -> Result<i32, E> {
        record("dispatch-begin", &format!("{id}\t{}\t{}", self.tests.wire(), self.tlc.wire()))?;
        let code = body()?;
        if let Err(error) = record("dispatch-finish", &format!("{id}\t{code}"))
            && code == 0
        {
            return Err(error.into());
        }
        Ok(code)
    }
}

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
                    if kind == "dispatch-finish" {
                        Err(io::Error::other("finish refused"))
                    } else {
                        Ok(())
                    }
                },
                || Ok::<_, io::Error>(0),
            )
            .unwrap_err();
        assert_eq!(error.to_string(), "finish refused");
        let code = Contract::observed()
            .run_recorded(
                "dispatch-42-12",
                |kind, _| {
                    if kind == "dispatch-finish" {
                        Err(io::Error::other("finish refused"))
                    } else {
                        Ok(())
                    }
                },
                || Ok::<_, io::Error>(37),
            )
            .unwrap();
        assert_eq!(code, 37, "recording failure replaced the original failed payload status");
    }

    #[test]
    fn metric_roles_refuse_swapped_observers_and_allow_mixed_applicability() {
        assert!(Contract::new(MetricContract::ObservedTlc, MetricContract::ObservedTlc).is_err());
        assert!(
            Contract::new(MetricContract::ObservedTests, MetricContract::ObservedTests).is_err()
        );
        let contract = Contract::new(
            MetricContract::ObservedTests,
            MetricContract::NotApplicable(NoWorkReason::CompileOnly),
        )
        .unwrap();
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
            assert_eq!(
                serde_json::from_str::<NoWorkReason>(&format!("\"{wire}\"")).unwrap(),
                reason
            );
            let mut events = Vec::new();
            Contract::not_applicable(reason)
                .run_recorded(
                    "dispatch-42-11",
                    |kind, values| {
                        events.push(format!("{kind}\t{values}"));
                        Ok(())
                    },
                    || Ok::<_, io::Error>(0),
                )
                .unwrap();
            assert_eq!(events[0], format!("dispatch-begin\tdispatch-42-11\t{wire}\t{wire}"));
        }
        assert!(serde_json::from_str::<NoWorkReason>("\"unknown\"").is_err());
    }
}
