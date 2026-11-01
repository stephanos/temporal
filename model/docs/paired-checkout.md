# Local paired checkout

Umpire is internal. The public Temporal branch cannot access it, so this integration
currently works locally. The owner has deferred GitHub workflow integration;
no Umpire checkout action, CI credential or revision variable is required.

Place both repositories under one parent directory:

```text
<dev-root>/
  temporal/
  umpire/
  go.work
```

Cargo dependencies resolve the sibling Umpire source directly. From the parent,
create a Go workspace only if one does not already exist:

```sh
go work init ./temporal ./umpire/hosts/go ./umpire/reference/kettle ./temporal/model/temporal/hosts/go
```

For an existing workspace, add the four modules with `go work use` and preserve
unrelated entries. Check `go env GOWORK` and `go list -m -json` from Temporal to
confirm that the host resolves to the local sibling checkout.

Run Temporal's `make model-check` and `make model-validate` locally with both
checkouts present. Refresh its owned guest with `make umpire-guest`. Framework
checks run from the Umpire checkout. Uncommitted framework edits are consumed
locally; no package publication or remote checkout fallback replaces that source.

The migration retains fresh paired source, generation, guest reproduction and
Rust/Go edit proofs. CI support remains separate future work; local passing checks
do not establish public CI compatibility.

A checkout without the sibling `../umpire` cannot resolve `github.com/temporalio/umpire/hosts/go`.
On such a checkout, including public CI, packages that import it (`tools/umpire/...`,
`tools/canary/...`, `common/testing/testpilot/temporal/act` and the `./tests` functional
package) and `go list -m all` fail with a missing replacement directory. This is the
expected cost of keeping Umpire local-only.
