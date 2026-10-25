fn bounded_recovery_script(
    events: &Path,
    exploration: &str,
    batch: &str,
    singletons: [(&str, u8); 2],
) -> String {
    let [(p0, p0_status), (p1, p1_status)] = singletons;
    format!(
        "events={}\ncount=0\nproperty=''\nwhile read -r kind value; do\n case $kind in INVARIANT|PROPERTY) count=$((count + 1)); property=$value;; esac\ndone < Run.cfg\ncase $count in\n 0) printf 'explore\\n' >> \"$events\"; printf '%s' {};;\n 1) printf '%s\\n' \"$property\" >> \"$events\"\n    case $property in\n     P0) printf '%s' {}; exit {p0_status};;\n     P1) printf '%s' {}; exit {p1_status};;\n     *) printf '%s' {};;\n    esac;;\n *) printf 'batch\\n' >> \"$events\"; printf '%s' {}; exit 12;;\nesac",
        quoted(&events.to_string_lossy()),
        quoted(exploration),
        quoted(p0),
        quoted(p1),
        quoted(&format!("{}{}", message(2193, false, "Success"), message(2186, false, "Finished"))),
        quoted(batch),
    )
}

#[test]
fn singleton_output_budget_precedes_later_evaluation_and_stops_new_waves() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let finished = message(2186, false, "Finished");
        let complete = format!("{}{finished}", message(2193, false, "Success"));
        let exploration = format!(
            "{}{complete}",
            message(2199, false, "0 states generated, 0 distinct states found, 0 states left on queue.")
        );
        let batch_output = format!("{}{finished}", message(2110, true, "Invariant P0 is violated."));
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
            tlc.limits.output_bytes = u64::try_from(limit).map_err(|error| Refusal(error.to_string()))?;
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
                assert!(started.is_subset(&BTreeSet::from(["explore", "batch", "P0", "P1"])), "{observed}");
                assert!(observed.lines().count() <= 4, "{observed}");
            } else {
                assert_eq!(observed, "explore\nP0\n");
            }
        }
        Ok(())
    })();
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
            message(2199, false, "0 states generated, 0 distinct states found, 0 states left on queue.")
        );
        let batch_output = format!("{}{finished}", message(2110, true, "Invariant P0 is violated."));
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
    scratch.finish(result)
}
