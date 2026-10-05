import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const out = path.dirname(fileURLToPath(import.meta.url));
const base = '1b0bc277589d141aca8b534b03135ab3e57fc050';
const cli = '/home/agent/.codex/scripts/flowctl';
const read = p => fs.readFileSync(path.resolve(root, p));
const json = p => JSON.parse(read(p));
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const git = args => execFileSync('git', args, { cwd: root, maxBuffer: 32 << 20 });
const expected = [
  'tools/gomad3/toolchain/runtime/go1.27.1.patch',
  'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go',
  'tools/gomad3/choice/internal/wire/wire_generated.go',
  'tools/gomad3/target/internal/livecap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go',
  'tools/gomad3/runner/testdata/diagnostic-identity-choices.json',
].sort();
assert.equal(git(['rev-parse', 'HEAD']).toString().trim(), base, 'precommit base changed');
const preservation = json(path.join(out, 'preservation.json'));
const freeze = json(path.join(preservation.scratch, 'freeze.json'));
const final = json(path.join(preservation.scratch, 'final-sources.json'));
const prior = json('.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/preservation.json');
for (const file of preservation.files) {
  const previous = prior.files[file.path];
  assert(previous, file.path);
  assert.equal(file.before_sha256, previous.materialized_sha256, 'baseline materialization drift: ' + file.path);
}
assert.equal(preservation.total_renames, 30);
assert.equal(preservation.files.length, 21);
const tracked = git(['ls-files', '-z']).toString().split('\0').filter(p => p && !p.startsWith('.flow/') && fs.existsSync(path.join(root, p)) && fs.statSync(path.join(root, p)).isFile()).sort();
const current = Object.fromEntries(tracked.map(p => [p, sha(read(p))]));
assert.deepEqual(current, final, 'source changed after final preservation');
assert.equal(tracked.length, 5076);
const changed = [...new Set([...Object.keys(freeze.sources), ...tracked])].filter(p => freeze.sources[p] !== current[p]).sort();
assert.deepEqual(changed, expected);
assert.deepEqual(git(['diff', '--name-only', base, '--', '.', ':(exclude).flow/**']).toString().trim().split('\n').sort(), expected);
for (const [p, digest] of Object.entries(freeze.user_hashes)) assert.equal(sha(read(p)), digest, p);
assert.equal(sha(read('tools/gomad3/toolchain/version/version.json')), freeze.descriptor_sha256);
assert.equal(sha(read('tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz')), freeze.archive_sha256);
const measurements = {};
for (const [name, value] of Object.entries(preservation.measurements)) {
  const bytes = read(path.join(preservation.scratch, name + '.patch'));
  assert.equal(sha(bytes), value.sha256);
  assert.equal(bytes.length, value.bytes);
  assert.equal(bytes.toString().split('\n').length - 1, value.lines);
  measurements[name] = value;
}
assert.equal(measurements['final-U1'].bytes, 24117);
assert.equal(measurements['final-U3'].bytes, 33294);
assert.equal(measurements['final-U3'].bytes - 32652, 642);
const allowedEnvironment = new Set(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'GOEXPERIMENT', 'GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOCACHE', 'GOMODCACHE', 'COMPACT_PHASE', 'COMPACT_CAPTURE_DIR']);
let receipts = 0;
for (const name of fs.readdirSync(out).filter(p => p.endsWith('.json'))) {
  const value = json(path.join(out, name));
  if (!('stdout_sha256' in value) || !('exit' in value)) continue;
  const label = name.slice(0, -5);
  assert.equal(sha(read(path.join(out, label + '.stdout'))), value.stdout_sha256, name);
  assert.equal(sha(read(path.join(out, label + '.stderr'))), value.stderr_sha256, name);
  for (const key of Object.keys(value.environment)) assert(allowedEnvironment.has(key), 'unexpected environment key: ' + key);
  receipts++;
}
const audit = json(path.join(out, 'identity-audit/post-publication.receipt.json'));
assert.equal(audit.exit_status, 0);
assert.deepEqual(audit.product_after_changes, {});
const identity = JSON.parse(audit.stdout);
assert.equal(identity.current_golden_matches_candidate, true);
assert.equal(identity.differences.length, 7);
const golden = read('tools/gomad3/runner/testdata/diagnostic-identity-choices.json');
assert.equal('sha256:' + sha(golden), identity.candidate_fixture_sha256);
assert.notEqual(golden.at(-1), 10);
const section = (md, start, end) => md.slice(md.indexOf(start), md.indexOf(end, md.indexOf(start)));
for (const number of [2, 4]) {
  const taskPath = '.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.' + number;
  const historical = git(['show', base + ':' + taskPath + '.md']).toString();
  const live = execFileSync(cli, ['cat', 'fn-110.' + number], { cwd: root, encoding: 'utf8' });
  assert.equal(section(live, '## Acceptance\n', '## Done summary\n'), section(historical, '## Acceptance\n', '## Done summary\n'));
  assert(section(live, '## Done summary\n', '## Evidence\n').startsWith(section(historical, '## Done summary\n', '## Evidence\n')));
  assert.equal(section(live, '## Evidence\n', '## Linux ownership blocker'), section(historical, '## Evidence\n', '## Linux ownership blocker'));
  const before = JSON.parse(git(['show', base + ':' + taskPath + '.json']));
  const after = JSON.parse(execFileSync(cli, ['show', 'fn-110.' + number, '--json'], { cwd: root, encoding: 'utf8' }));
  assert.deepEqual(after.depends_on, before.depends_on);
}
console.log(JSON.stringify({ base, product_files: tracked.length, exact_changed_paths: changed, bound_receipts: receipts, alpha_sites: 30, baseline_sources_bound_to_previous_checkpoint: 21, identity_pointers: 7, measurements, original_U3_gap: 642, original_acceptance_and_historical_evidence_preserved: true, user_files_preserved: true }, null, 2));
