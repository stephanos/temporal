# Source-size closeout

Run the counter after integration and final source freeze, from the repository
root, with the authenticated Go source revision and the final Rust revision:

```sh
python3 tools/gomad/scripts/source-size.py --baseline GO_SOURCE_SHA --candidate RUST_SOURCE_SHA --output /absolute/fresh/source-size.json
```

The JSON stores both full commit IDs, this counter's version and SHA-256, the
command, sorted complete path inventories and their hashes, each blob's Git ID
and SHA-256, and bytes/files/total lines/nonblank lines/physical code lines.
It groups source by subsystem, language and production/test/generated file.
Generated markers take precedence; tests embedded in production files remain in
that file's group. Template and runtime `.go.txt` sources are included. The
whole-repository totals include every tracked blob, including non-source inputs.
The complete file list permits a separately reported non-Flow inventory.

The physical code-line metric excludes recognized comments outside literals;
Go raw strings, Python triple strings, Rust raw strings/nested comments, shell
hash comments and SQL comments have focused fixtures. It is a lexical estimate,
not cloc or a compiler metric. Raw byte/line counts remain independently useful.
Unsupported source extensions stay in the repository totals without guessed
code-line values. Run the counter fixture checks with:

```sh
python3 -m unittest discover -s tools/gomad/scripts/tests -p test_source_size.py -v
```

Report the preexisting Rust helper separately from the new Rust host, retained
Go model/oracle/toolchain sources and supporting languages. The prospective
retirement tree has `internal/model/wasi` as a distinct retained reference group.
Do not report the entire preexisting helper as newly written replacement code,
or subtract its removal while adding the relocated copy. Use whole-repository
bytes/files and both path inventories for the net comparison after deletion.
No preliminary worktree count is the requested final integrated comparison.
