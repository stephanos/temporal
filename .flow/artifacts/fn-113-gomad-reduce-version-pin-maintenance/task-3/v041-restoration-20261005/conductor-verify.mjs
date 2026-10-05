import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const directory = dirname(fileURLToPath(import.meta.url));
const base = 'b1054ecc0968cb1d9b957c3b98ef6dd8c340c414';
const recovery = '56148912df17e105dab3ec4b9e250ff5ef813318';
const scope = ['tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'Makefile', '.github/.golangci.yml'];
const prefix = 'tools/gomad3/internal/compatibilitypack/';
const added = ['packs/modernc-libc-xsys-v041.json', 'requests/modernc-libc-xsys-v041.json', 'reports/modernc-libc-xsys-v041.md', 'testdata/v041/go.mod', 'testdata/v041/go.sum', 'testdata/v041/libc_test.go'].map(path => prefix + path).sort();
const changed = ['authoring/generate_test.go', 'authoring/refresh_test.go', 'evidence_test.go', 'generation.json', 'packs_generated_test.go', 'policy_test.go', 'working-directories.json'].map(path => prefix + path).concat(['tools/gomad3/architecture_test.go', 'tools/gomad3/internal/gomadtool/architecture/architecture.go']).sort();
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const read = path => readFileSync(resolve(root, path));
const git = (args, input) => {
  const result = spawnSync('git', args, { cwd: root, input, maxBuffer: 128 << 20 });
  assert.equal(result.status, 0, result.stderr.toString());
  return result.stdout;
};
const checks = JSON.parse(readFileSync(resolve(directory, 'checks.json')));
const paths = git(['ls-tree', '-r', '--name-only', '-z', base, '--', ...scope]).toString().split('\0').filter(Boolean).sort();
const batch = git(['cat-file', '--batch'], paths.map(path => base + ':' + path + '\n').join(''));
let offset = 0;
const baselineDigest = createHash('sha256');
const foundChanges = [];
for (const path of paths) {
  const end = batch.indexOf(10, offset);
  const header = batch.subarray(offset, end).toString().split(' ');
  assert.equal(header[1], 'blob', path);
  const size = Number(header[2]);
  assert.ok(Number.isSafeInteger(size) && size >= 0);
  const bytes = batch.subarray(end + 1, end + 1 + size);
  offset = end + size + 2;
  baselineDigest.update(path + '\0' + hash(bytes) + '\0');
  if (!bytes.equals(read(path))) foundChanges.push(path);
}
assert.equal(offset, batch.length);
assert.equal(paths.length, 1052);
assert.equal(baselineDigest.digest('hex'), checks.freezes.BASE.source.sha256);
assert.deepEqual(foundChanges.sort(), changed);
const currentPaths = git(['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', ...scope]).toString().split('\0').filter(Boolean).sort();
assert.equal(currentPaths.length, 1058);
assert.deepEqual(currentPaths.filter(path => !paths.includes(path)), added);
const currentDigest = createHash('sha256');
for (const path of currentPaths) currentDigest.update(path + '\0' + hash(read(path)) + '\0');
assert.equal(currentDigest.digest('hex'), checks.freezes.FINAL3.source.sha256);
for (const path of added) assert.ok(read(path).equals(git(['show', recovery + ':' + path])), path);
for (const freeze of Object.values(checks.freezes)) assert.deepEqual(freeze.tools, checks.freezes.BASE.tools);
for (const [path, digest] of Object.entries(checks.freezes.FINAL3.tools)) assert.equal(hash(readFileSync(path)), digest, path);
for (const run of checks.runs) {
  assert.equal(run.signal, null, run.label);
  assert.equal(run.error, undefined, run.label);
  assert.equal(hash(readFileSync(resolve(directory, run.label + '.stdout.log'))), run.stdout_sha256, run.label);
  assert.equal(hash(readFileSync(resolve(directory, run.label + '.stderr.log'))), run.stderr_sha256, run.label);
  assert.ok(Date.parse(run.finished_at) >= Date.parse(run.started_at), run.label);
  if (run.label === 'generate-first') {
    assert.equal(run.source_before, checks.freezes.RESTORED.source.sha256);
    assert.equal(run.source_after, checks.freezes.GENERATED.source.sha256);
  } else {
    assert.equal(run.source_and_tool_inputs_stable, true, run.label);
    assert.equal(run.source_before, checks.freezes[run.stage].source.sha256, run.label);
    assert.equal(run.source_after, run.source_before, run.label);
  }
}
const task = '.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.3.md';
const archival = bytes => bytes.toString().split('## Done summary')[1].replace(/^Blocked:\n[\s\S]*?(?=^## Evidence)/m, '');
assert.equal(archival(read(task)), archival(git(['show', base + ':' + task])));
console.log(JSON.stringify({ base_git_paths_verified: paths.length, unchanged_existing_source_paths: paths.length - foundChanges.length, modified_existing_paths: foundChanges, exact_historical_recoveries: added.length, final_source_paths: currentPaths.length, final_source_sha256: checks.freezes.FINAL3.source.sha256, retained_command_bindings: checks.runs.length, archival_done_and_evidence_unchanged: true, current_flow_blocker_excluded_from_archival_comparison: true }, null, 2));
