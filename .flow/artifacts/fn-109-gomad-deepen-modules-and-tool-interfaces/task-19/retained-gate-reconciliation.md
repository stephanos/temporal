# Final retained-gate reconciliation

Round4 differs from immutable round2 only by these two private-checker paths:
9796dff5b296f6c0cdadaa47434e30ee064eaf4a94a8210e3608dbb035fcd9eb  tools/gomad3/internal/gomadtool/architecture/effects.go
f807d65cab62673106ee6779a6d36b7195caa475504417a2ff5d871b95ff448f  tools/gomad3/internal/gomadtool/architecture/range_test.go

All other whole-module source/input hashes are identical. Fresh full private
checker units, root Quick and complete HostVet cover the changed paths, including
the new native-causal iterator/assignment operand tests. The full root Quick
also executes all36 effect fixtures, both actual qualified effect roots, package
and public-signature rules, and the actual external-consumer compilation.

Retained SAME-INPUT green results:
- make validate: round2-final-quick-validate.log, exit0, 07:05:08–07:05:11Z.
- All26 focused preservation tests: round2-final-preservation.log, exit0,
  07:07:01–07:07:02Z.
- fn105.4 read-only reference: round2-final-quick-reference.log, exit0,
  07:05:34–07:05:35Z; status blocked, conductor owns closure.

Makefile validation runs cmd/gomadtool generation/checks and the compatibility
profile test. Stock Go1.27.1 go list -deps for ./cmd/gomadtool and
./internal/compatibilitypack does not include internal/gomadtool/architecture.
The only changed production file is that unimported private checker; only its
own test file is newly added. Makefile, generator/validation/tool commands,
compatibility/profile/qualification/patch/script inputs and every focused
preservation package/test/preimage/literal expectation remain byte identical.
This is retained same-input evidence, not a fresh run or a generic receipt skip.

Round1, round2 and round3 manifests/logs stay immutable, including each discovered
RED and inconclusive selection. Round4's freeze HEAD is 9c0438b314b5a8368b1067a02abb865c24077f70.
Root subsequently committed exact task18/source evidence at
b7c26a2a5cf15fb3626b8114abdd44ccb2dd7005 without changing any task19 input.
The handover status file records that current HEAD and exact module delta.
Native qualification and formal acceptance remain open.
