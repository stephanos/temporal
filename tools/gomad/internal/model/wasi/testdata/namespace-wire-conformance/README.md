# Namespace conformance wire corpus

`TestNamespaceWireConformanceCorpus` generates complete input and expected output
strings for the test-only namespace driver. Keeping raw frame text inside JSON
strings preserves duplicate keys, malformed tokens, trailing values and missing
required fields. The Rust loader runs the normal conformance driver's LF reader
and compares the complete output bytes. Each fixture has a strict schema.

From `tools/gomad`, regenerate with:

```sh
go test -tags test_dep -count=1 -run '^TestNamespaceWireConformanceCorpus$' ./internal/model/wasi -args -update-conformance
```

Repeat the command and require a byte-identical corpus before accepting changes.
