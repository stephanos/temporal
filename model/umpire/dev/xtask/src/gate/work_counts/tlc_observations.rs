//! Integrity-linked TLC process and assertion observations.
use std::{io, path::Path};

#[derive(Default)]
pub(super) struct Collector;

impl Collector {
    pub(super) fn ingest(&mut self, _fields: &[&str]) -> io::Result<bool> {
        Ok(false)
    }

    pub(super) fn record_pass(&mut self, _name: String, _binary_sha: String) -> io::Result<()> {
        Ok(())
    }

    pub(super) fn finish(&self, _directory: &Path) -> io::Result<(Option<u64>, Option<u64>)> {
        Ok((None, None))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn observe(lines: &[&str]) -> io::Result<(Option<u64>, Option<u64>)> {
        let mut collector = Collector::default();
        for line in lines {
            assert!(collector.ingest(&line.split('\t').collect::<Vec<_>>())?);
        }
        collector.finish(Path::new("/unused"))
    }

    #[test]
    fn counts_confirmed_v1_launches_and_deduplicates_configurations() -> io::Result<()> {
        let sha = "a".repeat(64);
        assert_eq!(observe(&[
            &format!("1\ttlc-attempt\t100-0\t{sha}"), "1\ttlc-finish\t100-0\t1",
            &format!("1\ttlc-attempt\t100-1\t{sha}"), "1\ttlc-finish\t100-1\t1",
        ])?, (Some(2), Some(1)));
        Ok(())
    }

    #[test]
    fn counts_admitted_v2_but_preserves_unknown_legacy_completion() -> io::Result<()> {
        let identity = "b".repeat(64);
        assert_eq!(observe(&[
            &format!("2\ttlc-attempt\t100-0\tadmitted\tprobe\t{identity}"),
            "2\ttlc-finish\t100-0\tstarted\tcompleted\t-\t-",
        ])?, (Some(1), Some(0)));
        assert_eq!(observe(&["1\ttlc-attempt\t100-0\t-", "1\ttlc-finish\t100-0\t0"])?, (None, None));
        Ok(())
    }

    #[test]
    fn rejects_missing_duplicate_mixed_and_unknown_records() {
        for lines in [
            vec!["1\ttlc-attempt\t100-0\tprobe"],
            vec!["1\ttlc-finish\t100-0\t1"],
            vec!["1\ttlc-attempt\t100-0\tprobe", "1\ttlc-attempt\t100-0\tprobe"],
            vec!["1\ttlc-attempt\t100-0\tprobe", "2\ttlc-finish\t100-0\tstarted\tcompleted\t-\t-"],
            vec!["3\ttlc-attempt\t100-0\tprobe"],
            vec!["2\ttlc-forged\t100-0"],
        ] {
            assert!(observe(&lines).is_err(), "accepted {lines:?}");
        }
    }

    #[test]
    fn empty_observations_do_not_prove_zero() -> io::Result<()> {
        assert_eq!(Collector::default().finish(Path::new("/unused"))?, (None, None));
        assert!(!Collector::default().ingest(&["1", "tests", "other"])?);
        Ok(())
    }
    #[test]
    fn refuses_fixture_without_assertions_and_observed_parent_pass() -> io::Result<()> {
        let mut collector = Collector::default();
        assert!(collector.ingest(&["2", "tlc-attempt", "100-0", "fixture", "probe", &"b".repeat(64)])?);
        assert!(collector.ingest(&["2", "tlc-finish", "100-0", "fixture", "completed", "2f746d702f3130302d302e6a736f6e", &"c".repeat(64)])?);
        assert!(collector.finish(Path::new("/tmp/receipts/work/scope/tests")).is_err());
        Ok(())
    }

    #[test]
    fn rejects_invalid_record_id_configuration_origin_cause_and_identity() {
        for fields in [
            vec!["1", "tlc-attempt", "bad", "probe"],
            vec!["1", "tlc-attempt", "100-0", "bad"],
            vec!["2", "tlc-attempt", "100-0", "other", "probe", "bad"],
            vec!["2", "tlc-finish", "100-0", "started", "other", "-", "-"],
            vec!["2", "tlc-fixture-complete", "bad", "bad", "bad"],
            vec!["2", "tlc-attempt"],
        ] {
            assert!(Collector::default().ingest(&fields).is_err(), "accepted {fields:?}");
        }
    }

}
