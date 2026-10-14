# Scala 3 macros spike (2026-10-13)

The prototype behind `model/design/scala-macros-export.md`: a compile-time reifier (`proto/reify/`)
that lifts Kettle's rules, guards and effects to Umpire IR, byte-identical to the TASTy lifter's
output, plus a compiler-plugin host, refusal probes and the timing scripts.

Archived from `/tmp/umpire-macro-spike/`. The following were not kept:
- jars;
- scala-cli/Bloop build state;
- the copied repository (`repo/`) and `qi/`;
- today's lift outputs (`out/today0`, `out/sanity-today`);
- the large files under `out/measure/` (its logs and timings are kept).

The scripts still use the original `/tmp/umpire-macro-spike/` paths. The report's §8 says how to
rerun it.
