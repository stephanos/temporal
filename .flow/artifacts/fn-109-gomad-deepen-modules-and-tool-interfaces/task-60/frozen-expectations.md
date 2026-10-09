# Independent helper expectations

Before editing process_test.go, its SHA256 is 522a5a2319423de0b9b6877c68aa25209ee69e29093487933bd532c246eb230b. Additive control SHA256 is 94b7fb2073c75a46b0835cc2620fc3c522c7d945bf7b6e5d1686c84a58e16572.

| Helper | Healthy stdout | Both stdout dispositions' stderr | Both statuses |
| --- | --- | --- | --- |
| output | target stdout plus newline | target stderr plus newline | 7 |
| choice-marker | post-choice-marker plus newline | empty | 0 |
| choice-tape-readonly | choice tape read-only plus newline | empty | 0 |
| choice-reorder ab | ab or ba plus newline | empty | 0 |
| choice-select | eight a/b choices plus newline | empty | 0 |
| choice-prefix-rng | eight a/b choices plus newline | empty | 0 |

Each failed stdout is an inherited real read-only file. Its unchanged literal bytes are retained; the parent verifies that writing that same open file returns EBADF. The choice-tape marker uses a lawful read-only ExtraFiles descriptor and retains the original Pwrite then Ftruncate checks before its stdout attempt. Stock Go controls establish helper outcomes and real fd1 failure execution. They cannot establish patched-runtime seeded/replay identities; original Run controls retain those obligations and their actual platform outcomes.
