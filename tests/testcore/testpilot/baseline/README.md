# Typed-unary Contract baseline

`typed-unary-contract.json` is the `contract` of `../testdata/typed-unary-case.json` as the
generator (`umpire-gen-case-runtime-conformance`) wrote it at fn-86 task .1: the same bytes,
indented the same way, cut out of the fixture. It is a scaffold for task .2's proof point, which
compares the field reads of the typed unary Case re-authored with the commands against these; no
generator writes it and no gate regenerates it, so task .2 deletes this directory once that
comparison has passed. It lives beside `testdata/`, not inside it, because
`umpire-check-case-runtime-conformance` diffs that directory against the generator's output and
rejects any file the generator did not write. Do not edit it by hand and do not add a reader.
