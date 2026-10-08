import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import assert from 'node:assert/strict';
import { root, out, worker, stock } from './capture.mjs';
const load = label => JSON.parse(readFileSync(resolve(worker, label + '.json')));
function capture(label, argv, expected = 0) {
  const result = spawnSync('node', [resolve(out, 'capture.mjs'), label, ...argv], { cwd: root, encoding: 'utf8', maxBuffer: 64 << 20 });
  process.stdout.write(result.stdout ?? '');
  process.stderr.write(result.stderr ?? '');
  assert.equal(result.status, expected, label);
}
capture('worker-integrity', ['node', resolve(worker, 'verify.mjs')]);
capture('writer-negative-control', [stock + '/go', '-C', 'tools/gomad3', 'test', '-count=1', '-tags', 'test_dep', '-json', '-overlay=' + resolve(out, 'writer-negative-overlay.json'), './cmd/gomad/internal/cli', '-run', '^TestExploreErrorJoinsDiagnosticAndReporterWriterFailures$', '-timeout', '90s'], 1);
for (const label of ['restored-portable-cli', 'restored-portable-runner', 'restored-architecture-installation', 'restored-qualification-control', 'restored-generators', 'restored-errortype', 'restored-format-check']) capture(label, load(label).argv);
for (const label of ['restored-fast-lint', 'restored-unfiltered-lint']) capture(label, load(label).argv, 2);
