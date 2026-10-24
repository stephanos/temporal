# Primary inspection capacity diagnosis

Task .29 isolates the cycle8 inspection refusal. Two separately labelled diagnostic
executions reproduce `transition-bytes limit=1048576 used=1048560 delta=188` with
byte-identical accepted World prefixes and guest streams. The exhausted owner is
wasm-broker. Its 6,408 transitions represent 2,136 visible calls:
`2124 * (188 + 131 + 172) + 12 * (182 + 125 + 166) = 1048560` bytes.
The next 188-byte register is refused before mutation. Accounting matches the
existing register/ready/quiesce contract.

The bounded network trace contains 1,647 served polls, of which 1,474 remain
pending and 173 complete. Repeated scans of pending requests consume the budget
after 471,133,454 nanoseconds of virtual time. The guest runtime polls before and
after idle; late trace timestamps contain two passes over the same 20 pending
request IDs. This supports polling amplification. It establishes neither an
accounting defect nor a safe optimization, and does not establish DNS warnings
as the cause. Removing one repeated scan does not prove the full canary fits.

Raw evidence and instrumentation provenance remain outside Git at
`/home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-r1.json`
(SHA-256 `db625ed2acb5e7f14af4b9577d76f50ff25d9959d0c572492082135a7810a5f7`).
The companion Markdown and attempts02/03 retain the execution details; the
parent authentication record verifies all 13 listed raw hashes. An independent
bounded audit verified the counts, matching prefixes, original scenario and
settings, and the distinction between diagnostic and qualification identities.
Temporary instrumentation remains isolated and uncommitted. Original cycle8
evidence is unchanged; these diagnostics supply no qualification pass.

The historical scope comparison is retained separately in
`/home/agent/wt/evidence/fn232-inspection-capacity-diagnostic/actual-diagnosis-scope-addendum-r3.md`
and its hashed source manifest. Retired Go did not activate recording on either
internal World. Rust commit `93be185443` explicitly broadened the public limit
to each native owner. Current `wasmhost/protocol.md` retains that active-bound
contract and separately preserves earlier absent-field artifacts. Restoring the
old inactive scope for new preparations would waive the current contract.
The repair must therefore preserve active enforcement and reduce unnecessary
visible operations safely; a guest-only time or dirty-bit cache is insufficient
because visible calls process queued controller commands.

Task .30 receives the repair and full original canary obligations formerly in
.29. Its unchanged 1 MiB bound, timers, retries, assertions, broker accounting,
replay and identity requirements still apply. Task .13 depends on .30 and retains
all 96 stages, the final 145-ID reader and tasks .22–.25 receiving. The original
diagnostic report's closing statement that repair is still needed remains true
of the defect; after this split, diagnosis closure and repair closure are separate.
