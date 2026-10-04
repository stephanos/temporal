# Round 3 retained same-input gates

Round3 changes from the immutable round2 manifest are exactly:
f2cc79df4d21a556402c3161dffe2b634dfefececbb7f799731aa46cafb5adca  tools/gomad3/internal/gomadtool/architecture/effects.go
401ca9b50b3d9ad610ecbd4729bd0b4b06e6a10b859b2be3eae199cb75f53f79  tools/gomad3/internal/gomadtool/architecture/range_test.go

The new iterator-slot tests and shared assignment handling affect the private
architecture checker and its test binary. Fresh full checker-unit and root
Quick/HostVet gates are required. Root Quick includes every architecture fixture,
the actual pure roots on both qualified source sets, public signatures, and the
actual external consumer.

Round2 make validate and the 26-test preservation command are retained green:
round2-final-quick-validate.log (exit0, 07:05:08–07:05:11Z) and
round2-final-preservation.log (exit0, 07:07:01–07:07:02Z).
Their production/test/input files are byte identical under both full manifests.
Makefile validate consists of cmd/gomadtool version/protocol/boundary generators,
patch/script validation, compatibility-pack check, the existing compatibility
profile test, and qualification generation. Stock Go1.27.1 go list -deps for
./cmd/gomadtool and ./internal/compatibilitypack does not include
internal/gomadtool/architecture. No generator, patch, script, profile, schema,
qualification source/input or test used by these checks changed in round3.
The focused preservation command selects only the 26 retained named tests in
record, World/process, compatibilitypack, target, Runner, pinimpact and refresh;
neither changed checker path is imported by those packages. Their canonical
schemas, old-source comparisons and literal expected bytes are unchanged.

This is explicit retained same-input evidence, not a fresh execution, native
qualification or a generic green-receipt skip.
