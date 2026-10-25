use super::*;
use std::os::unix::fs::PermissionsExt;

fn message(code: u32, error: bool, body: &str) -> String {
    format!("@!@!@STARTMSG {code}:{} @!@!@\n{body}\n@!@!@ENDMSG {code} @!@!@\n", u8::from(error))
}

fn quoted(text: &str) -> String {
    format!("'{}'", text.replace('\'', "'\\''"))
}

fn owner() -> Result<Scratch, Refusal> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?;
    Scratch::new(&root)
}

fn fake_tlc(directory: &Path, body: &str) -> Result<Tlc, Refusal> {
    let java = directory.join("java");
    fs::write(
        &java,
        format!(
            "#!/bin/bash\nset -eu\ndump=''\nwhile [ \"$#\" -gt 0 ]; do\n if [ \"$1\" = '-dump' ]; then shift; dump=$1; fi\n shift\ndone\n: > \"$dump\"\n{body}\n"
        ),
    )?;
    fs::set_permissions(&java, fs::Permissions::from_mode(0o700))?;
    Ok(Tlc {
        root: Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..").canonicalize()?,
        java,
        jar: directory.join("unused.jar"),
        limits: Limits::default(),
        property_workers: 2,
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
        let liveness = message(2116, true, "Temporal properties were violated.");
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
            let tlc = fake_tlc(
                scratch.path(),
                &format!("printf '%s' {}\nexit {status}", quoted(&output)),
            )?;
            let answer = tlc.execution(
                "M",
                "---- MODULE M ----\nA == TRUE\nB == []TRUE\n====\n",
                "SPECIFICATION Spec\nINVARIANT A\nPROPERTY B\n",
                Selection::Batch(&properties),
                &Schema::default(),
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
        let script = format!(
            "count=0\nwhile read -r kind property; do\n case $kind in INVARIANT|PROPERTY) count=$((count + 1));; esac\ndone < Run.cfg\nprintf '%s\\n' \"$count\" >> {}\nif [ \"$count\" = 0 ]; then printf '%s' {}; else printf '%s' {}; fi",
            quoted(&events.to_string_lossy()),
            quoted(&exploration),
            quoted(&"x".repeat(2048)),
        );
        let mut tlc = fake_tlc(scratch.path(), &script)?;
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
    scratch.finish(result)
}

#[test]
fn singleton_recovery_measures_worker_peak_and_owns_each_scratch() -> Result<(), Error> {
    let scratch = owner()?;
    let result = (|| {
        let events = scratch.path().join("events");
        let complete =
            format!("{}{}", message(2193, false, "Success"), message(2186, false, "Finished"));
        let script = format!(
            "events={}\nprintf 'start %s\\n' \"$PWD\" >> \"$events\"\nround=0\nwhile [ \"$round\" -lt 100000 ]; do\n starts=0\n while read -r event path; do\n  if [ \"$event\" = start ]; then starts=$((starts + 1)); fi\n done < \"$events\"\n if [ \"$starts\" -ge 2 ]; then break; fi\n round=$((round + 1))\ndone\nround=0\nwhile [ \"$round\" -lt 10000 ]; do round=$((round + 1)); done\nproperty=''\nwhile read -r kind value; do\n case $kind in INVARIANT|PROPERTY) property=$value;; esac\ndone < Run.cfg\nprintf '%s\\n' \"$property\"\nprintf '%s' {}\nprintf 'end %s\\n' \"$PWD\" >> \"$events\"",
            quoted(&events.to_string_lossy()),
            quoted(&complete),
        );
        let tlc = fake_tlc(scratch.path(), &script)?;
        let properties =
            (0..4).map(|index| ("INVARIANT".into(), format!("P{index}"))).collect::<Vec<_>>();
        let results = tlc.recover_properties(
            "M",
            "",
            "SPECIFICATION Spec\n",
            &properties,
            &Schema::default(),
        )?;
        for ((_, property), (answer, output)) in properties.iter().zip(results) {
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
    scratch.finish(result)
}
