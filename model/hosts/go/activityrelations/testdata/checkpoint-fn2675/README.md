# fn-267.5 checkpoint corpus

Captured 2026-11-06 from the live in-process harnesses with `UMPIRE_NOVELTY_CAPTURE`, at framework
`1d49f11` and Temporal `683ea97c64`, with the fn-267.3 run-identity declarations only:

- `gaps-*`: `go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelGap(s|LoggerPreservesFailuresWithoutPrivateOutput)$'`
  (corpus gzip-compressed; `corpus_sha256` covers the plain JSON lines).
- `slice-*`: `go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice(ThroughUmpire)?$'`
  (the Transport explore subtest is the wired consumer).

Secret-claimed values are replaced by presence-preserving placeholders before storage; raw-distinct
digests are HMACs under a key discarded after capture. Recompute with
`UMPIRE_NOVELTY_CORPUS=<this directory> UMPIRE_NOVELTY_HARNESS=gaps|slice go test -tags test_dep -run '^TestNoveltyCorpusCounts$' ./tests`.
