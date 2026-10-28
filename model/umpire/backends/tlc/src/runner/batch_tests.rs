use super::*;

fn message(code: u32, error: bool, body: &str) -> String {
    format!("@!@!@STARTMSG {code}:{} @!@!@\n{body}\n@!@!@ENDMSG {code} @!@!@\n", u8::from(error))
}

fn owner() -> Result<Scratch, Refusal> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    Scratch::new(&root)
}

fn fake_tlc(directory: &Path, case: &fixture::Case) -> Result<Tlc, Refusal> {
    let origin = InvocationOrigin::ProtocolFixture(fixture::install(directory, case)?);
    Ok(Tlc {
        root: Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?,
        java: directory.join("java"),
        jar: directory.join("unused.jar"),
        limits: Limits::default(),
        property_workers: 2,
        origin,
    })
}

#[test]
fn batch_protocol_requires_complete_selected_evidence() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let finished = message(2186, false, "Finished");
        let success = message(2193, false, "Model checking completed. No error has been found.");
        let evaluation = message(2111, true, "Evaluating invariant A failed.\nOverflow");
        let properties = vec![("INVARIANT".into(), "A".into()), ("PROPERTY".into(), "B".into())];
        let selected = message(2110, true, "Invariant A is violated.");
        let unknown = message(2110, true, "Invariant Unknown is violated.");
        let liveness = message(2116, true, "Liveness properties were violated.");
        let located = message(
            2110,
            true,
            "Invariant line 2, col 1 to line 2, col 8 of module M is violated.",
        );
        for (label, output, status, expected) in [
            ("complete", format!("{success}{finished}"), 0, "ok"),
            ("empty", String::new(), 0, "refusal"),
            ("missing_success", finished.clone(), 0, "refusal"),
            ("missing_finished", success.clone(), 0, "refusal"),
            ("nonzero_success", format!("{success}{finished}"), 1, "refusal"),
            ("safety", format!("{selected}{finished}"), 12, "ok"),
            ("unknown", format!("{unknown}{finished}"), 12, "refusal"),
            ("liveness", format!("{liveness}{finished}"), 13, "ok"),
            ("located", format!("{located}{finished}"), 12, "ok"),
            ("unfinished_violation", selected, 12, "refusal"),
            ("evaluation", format!("{evaluation}{finished}"), 1, "evaluation"),
            ("unfinished_evaluation", evaluation, 1, "refusal"),
            ("truncated", "@!@!@STARTMSG 2193:0 @!@!@\n".into(), 0, "refusal"),
            (
                "unsupported_error",
                format!("{}{finished}", message(1000, true, "Unexpected failure")),
                1,
                "refusal",
            ),
        ] {
            let tlc = fake_tlc(scratch.path(), &fixture::Case::Protocol { output, status })?;
            let answer = tlc.execution(
                "M",
                "---- MODULE M ----\nA == TRUE\nB == []TRUE\n====\n",
                "SPECIFICATION Spec\nINVARIANT A\nPROPERTY B\n",
                Selection::Batch(&properties),
                &Schema::default(),
                None,
            );
            let actual = match &answer {
                Ok(_) => "ok",
                Err(Error::Evaluation(_)) => "evaluation",
                Err(Error::Refusal(_)) => "refusal",
            };
            assert_eq!(actual, expected, "{label}: {answer:?}");
        }
        let invariants = vec![("INVARIANT".into(), "A".into())];
        assert!(batch_violation(&[(2116, "unnamed liveness".into())], &invariants, "").is_err());
        Ok(())
    })();
    let result = result.and_then(|()| {
        fixture::complete(scratch.path())?;
        Ok(())
    });
    scratch.finish(result)
}

#[test]
fn batch_resource_refusal_does_not_launch_singleton_recovery() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let events = scratch.path().join("events");
        let exploration = format!(
            "{}{}{}",
            message(
                2199,
                false,
                "0 states generated, 0 distinct states found, 0 states left on queue."
            ),
            message(2193, false, "Success"),
            message(2186, false, "Finished")
        );
        let mut tlc = fake_tlc(
            scratch.path(),
            &fixture::Case::Resource {
                events: events.clone(),
                exploration,
                oversized: "x".repeat(2048),
            },
        )?;
        tlc.limits.output_bytes = 1024;
        let result = tlc.check_strategy(
            "M",
            "---- MODULE M ----\n====\n",
            "SPECIFICATION Spec\nINVARIANT A\nINVARIANT B\n",
            &Schema {
                fields: BTreeMap::from([("x".into(), umpire_tla::Type::Bool)]),
                observers: BTreeMap::new(),
            },
            true,
        );
        assert!(matches!(result, Err(Error::Refusal(_))), "{result:?}");
        assert_eq!(
            fs::read_to_string(events)
                .map_err(|error| Refusal(format!("events: {error}; check: {result:?}")))?,
            "0\n2\n"
        );
        Ok(())
    })();
    let result = result.and_then(|()| {
        fixture::complete(scratch.path())?;
        Ok(())
    });
    scratch.finish(result)
}

#[test]
fn singleton_recovery_measures_worker_peak_and_owns_each_scratch() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let events = scratch.path().join("events");
        let complete =
            format!("{}{}", message(2193, false, "Success"), message(2186, false, "Finished"));
        let tlc = fake_tlc(
            scratch.path(),
            &fixture::Case::WorkerPeak { events: events.clone(), complete },
        )?;
        let properties =
            (0..4).map(|index| ("INVARIANT".into(), format!("P{index}"))).collect::<Vec<_>>();
        let mut results = Vec::new();
        for wave in properties.chunks(tlc.property_workers) {
            results.extend(tlc.recover_properties(
                "M",
                "",
                "SPECIFICATION Spec\n",
                wave,
                &Schema::default(),
            )?);
        }
        for ((_, property), answer) in properties.iter().zip(results) {
            let (answer, output) = answer?;
            assert_eq!(answer.verdict, Verdict::Verified);
            assert!(output.starts_with(&format!("{property}\n")), "{output}");
        }
        let mut active = BTreeSet::new();
        let mut owned = BTreeSet::new();
        let mut peak = 0;
        for event in fs::read_to_string(events)?.lines() {
            let (event, directory) =
                event.split_once(' ').ok_or_else(|| Refusal("invalid process event".into()))?;
            match event {
                "start" => {
                    assert!(owned.insert(directory.to_owned()), "scratch reused: {directory}");
                    assert!(active.insert(directory.to_owned()));
                    peak = peak.max(active.len());
                }
                "end" => assert!(active.remove(directory)),
                _ => return Err(Refusal("unknown process event".into()).into()),
            }
        }
        assert_eq!(peak, 2, "measured fake Java process peak");
        assert!(active.is_empty());
        assert_eq!(owned.len(), properties.len());
        for directory in owned {
            assert!(!Path::new(&directory).exists(), "scratch not cleaned: {directory}");
        }
        Ok(())
    })();
    let result = result.and_then(|()| {
        fixture::complete(scratch.path())?;
        Ok(())
    });
    scratch.finish(result)
}

fn bounded_recovery_script(
    events: &Path,
    exploration: &str,
    batch: &str,
    singletons: [(&str, u8); 2],
) -> fixture::Case {
    fixture::Case::Recovery {
        events: events.to_owned(),
        exploration: exploration.into(),
        batch: batch.into(),
        singletons: singletons.map(|(output, status)| (output.into(), status)),
        fallback: format!(
            "{}{}",
            message(2193, false, "Success"),
            message(2186, false, "Finished")
        ),
    }
}

#[test]
fn singleton_output_budget_precedes_later_evaluation_and_stops_new_waves() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let finished = message(2186, false, "Finished");
        let complete = format!("{}{finished}", message(2193, false, "Success"));
        let exploration = format!(
            "{}{complete}",
            message(
                2199,
                false,
                "0 states generated, 0 distinct states found, 0 states left on queue."
            )
        );
        let batch_output =
            format!("{}{finished}", message(2110, true, "Invariant P0 is violated."));
        let limit = 1024;
        let p0_size = limit - exploration.len() + 1;
        assert!(p0_size >= complete.len());
        let p0_output = format!("{complete}{}", "x".repeat(p0_size - complete.len()));
        let p1_output = format!(
            "{}{finished}",
            message(2111, true, "Evaluating invariant P1 failed.\nOverflow")
        );
        for output in [&exploration, &batch_output, &p0_output, &p1_output] {
            assert!(output.len() < limit, "each process stays below its output budget");
        }
        assert!(exploration.len() + batch_output.len() < limit);
        assert!(exploration.len() + p0_output.len() > limit);
        let schema = Schema {
            fields: BTreeMap::from([("x".into(), umpire_tla::Type::Bool)]),
            observers: BTreeMap::new(),
        };
        for (batch, workers) in [(false, 1), (true, 2)] {
            let events = scratch.path().join(format!("events-{batch}"));
            let script = bounded_recovery_script(
                &events,
                &exploration,
                &batch_output,
                [(&p0_output, 0), (&p1_output, 1)],
            );
            let mut tlc = fake_tlc(scratch.path(), &script)?;
            tlc.property_workers = workers;
            tlc.limits.output_bytes =
                u64::try_from(limit).map_err(|error| Refusal(error.to_string()))?;
            let answer = tlc.check_strategy(
                "M",
                "---- MODULE M ----\n====\n",
                "SPECIFICATION Spec\nINVARIANT P0\nINVARIANT P1\nINVARIANT P2\nINVARIANT P3\n",
                &schema,
                batch,
            );
            assert!(
                matches!(&answer, Err(Error::Refusal(Refusal(text))) if text.contains("combined check output limit exceeded")),
                "batch={batch}: {answer:?}"
            );
            let observed = fs::read_to_string(events)?;
            let started = observed.lines().collect::<BTreeSet<_>>();
            assert!(started.contains("explore") && started.contains("P0"), "{observed}");
            assert!(!started.contains("P2") && !started.contains("P3"), "{observed}");
            if batch {
                assert!(started.contains("batch"), "{observed}");
                assert!(
                    started.is_subset(&BTreeSet::from(["explore", "batch", "P0", "P1"])),
                    "{observed}"
                );
                assert!(observed.lines().count() <= 4, "{observed}");
            } else {
                assert_eq!(observed, "explore\nP0\n");
            }
        }
        Ok(())
    })();
    let result = result.and_then(|()| {
        fixture::complete(scratch.path())?;
        Ok(())
    });
    scratch.finish(result)
}

#[test]
fn singleton_evaluation_stops_recovery_after_current_wave() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let finished = message(2186, false, "Finished");
        let complete = format!("{}{finished}", message(2193, false, "Success"));
        let exploration = format!(
            "{}{complete}",
            message(
                2199,
                false,
                "0 states generated, 0 distinct states found, 0 states left on queue."
            )
        );
        let batch_output =
            format!("{}{finished}", message(2110, true, "Invariant P0 is violated."));
        let evaluation = format!(
            "{}{finished}",
            message(2111, true, "Evaluating invariant P0 failed.\nOverflow")
        );
        let events = scratch.path().join("events");
        let script = bounded_recovery_script(
            &events,
            &exploration,
            &batch_output,
            [(&evaluation, 1), (&complete, 0)],
        );
        let mut tlc = fake_tlc(scratch.path(), &script)?;
        tlc.property_workers = 2;
        let answer = tlc.check_strategy(
            "M",
            "---- MODULE M ----\n====\n",
            "SPECIFICATION Spec\nINVARIANT P0\nINVARIANT P1\nINVARIANT P2\nINVARIANT P3\n",
            &Schema {
                fields: BTreeMap::from([("x".into(), umpire_tla::Type::Bool)]),
                observers: BTreeMap::new(),
            },
            true,
        );
        assert!(
            matches!(&answer, Err(Error::Evaluation(error)) if error.property.as_deref() == Some("P0") && error.message == "Overflow"),
            "{answer:?}"
        );
        let observed = fs::read_to_string(events)?;
        assert_eq!(
            observed.lines().collect::<BTreeSet<_>>(),
            BTreeSet::from(["explore", "batch", "P0", "P1"]),
            "{observed}"
        );
        assert_eq!(observed.lines().count(), 4, "{observed}");
        Ok(())
    })();
    let result = result.and_then(|()| {
        fixture::complete(scratch.path())?;
        Ok(())
    });
    scratch.finish(result)
}
