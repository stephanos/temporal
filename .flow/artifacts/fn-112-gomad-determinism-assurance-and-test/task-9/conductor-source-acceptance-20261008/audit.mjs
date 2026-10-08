import assert from 'node:assert/strict';
import { existsSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { out, worker, root, read, sha, sources } from './capture.mjs';

const json = p => JSON.parse(read(p));
const git = args => {
  const result = spawnSync('git', args, { cwd: root, maxBuffer: 128 << 20 });
  assert.equal(result.status, 0, result.stderr?.toString());
  return result.stdout;
};
const workerSources = () => Object.fromEntries(git(['ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration']).toString().trim().split('\n').filter(p => existsSync(resolve(root, p))).map(p => [p, sha(read(p))]));
const manifest = json(worker + '/manifest.json'), seal = json(worker + '/terminal-seal.json');
assert.equal(sha(read(worker + '/manifest.json')), '25358815c9030c72348f560c6e17c646a8173bbb2e2c490c571ba3cab5accbe7');
assert.equal(sha(read(worker + '/terminal-seal.json')), 'a8fb30b9892076ac10fa9e45f976387c53cf25f2d8b8f4bb44c20b88aefebf1f');
assert.equal(seal.lane, 'RELEASED');
assert.equal(seal.commands_running, 0);
assert.equal(seal.delegates_running, 0);
const walk = prefix => readdirSync(resolve(worker, prefix), { withFileTypes: true }).flatMap(e => {
  const path = prefix ? prefix + '/' + e.name : e.name;
  assert(!e.isSymbolicLink(), path);
  return e.isDirectory() ? walk(path) : [path];
});
assert.deepEqual(walk('').sort(), [...manifest.files.map(f => f.path), ...manifest.exclusions].sort());
assert.equal(manifest.files.length, 448);
for (const file of [...manifest.files, ...seal.terminal_files]) {
  assert(!file.path.startsWith('/') && !file.path.split('/').includes('..'));
  assert.equal(sha(read(worker + '/' + file.path)), file.sha256, file.path);
  assert.equal(statSync(resolve(worker, file.path)).size, file.bytes, file.path);
}
assert.equal(seal.manifest_sha256, sha(read(worker + '/manifest.json')));
const frozen = json(worker + '/sealed-source-proof.json');
assert.deepEqual(workerSources(), frozen.current_sources);
let recomputed;
const producer = read(worker + '/source-proof.mjs').toString().replace(/^import .*;\n/gm, '').replace(/^export /gm, '');
new Function('assert', 'existsSync', 'resolve', 'read', 'writeOriginal', 'git', 'sha', 'sources', 'root', 'process', 'console', producer)(
  assert, existsSync, resolve, read, (name, value) => { assert.equal(name, 'sealed-source-proof.json'); recomputed = value; }, git, sha, workerSources, root,
  { argv: ['node', 'source-proof.mjs', '--final', '--output=sealed-source-proof.json'] }, { log() {} },
);
assert.deepEqual(JSON.parse(JSON.stringify(recomputed)), frozen);
const restoration = json(out + '/world-and-cause-restoration-proof.json');
for (const file of restoration.changed_test_files) assert.equal(sha(read(file.path)), file.sha256, file.path);
assert.deepEqual(git(['diff', '--name-only', 'a7d99f4e64990839f81df4acd9bc150b842f8ea6', '--', 'tools/gomad3']).toString().trim().split('\n'), restoration.changed_test_files.map(f => f.path));
for (const file of restoration.user_files) {
  assert.equal(sha(read(file.path)), file.sha256);
  assert.equal(git(['ls-files', '--', file.path]).length, 0);
}
const identity = sha(JSON.stringify(sources()));
const events = path => read(path).toString().split('\n').flatMap(line => { try { return [JSON.parse(line)]; } catch { return []; } });
const sorted = list => list.map(x => JSON.stringify(x)).sort();
const receipt = label => {
  const value = json(out + '/' + label + '.json');
  assert.equal(value.signal, null, label);
  assert.equal(value.error, null, label);
  assert.equal(value.native, false, label);
  assert.deepEqual(value.source_changes, []);
  assert.equal(value.sources_before_sha256, identity, label);
  assert.equal(value.sources_after_sha256, identity, label);
  for (const suffix of ['stdout', 'stderr']) assert.equal(sha(read(out + '/' + label + '.' + suffix)), value[suffix + '_sha256']);
  const terminal = events(out + '/' + label + '.stdout').filter(e => e.Test && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => ({ package: e.Package, test: e.Test, action: e.Action }));
  assert.deepEqual(value.tests, terminal);
  for (const action of ['pass', 'fail', 'skip']) assert.equal(value.top_level_counts[action], terminal.filter(e => e.action === action && !e.test.includes('/')).length);
  for (const [tool, path] of Object.entries({ go: '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go', gofmt: '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt', golangci_lint: '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', errortype: '/tmp/fn109-lint-tools.ZdNe1t50/errortype' })) assert.equal(value.tool_sha256[tool], sha(read(path)));
  return value;
};
assert.equal(receipt('independent-worker-verification').exit, 0);
const rootResult = receipt('independent-isolated-cache-root'), oldRoot = json(worker + '/sealed-root.json');
assert.equal(rootResult.exit, 0);
assert.deepEqual(rootResult.top_level_counts, { pass: 19, fail: 0, skip: 0 });
assert.equal(rootResult.tests.filter(t => t.test.includes('/') && t.action === 'pass').length, 59);
assert.deepEqual(rootResult.argv, oldRoot.argv);
const environmentChanges = Object.keys(rootResult.environment).filter(k => (oldRoot.environment[k] ?? null) !== rootResult.environment[k]);
assert.deepEqual(environmentChanges, ['PATH', 'GOCACHE']);
assert.equal(oldRoot.exit, 1);
const controlledFixtures = receipt('independent-controlled-cache-fixtures');
assert.equal(controlledFixtures.exit, 0);
const controlledEnvironmentChanges = Object.keys(controlledFixtures.environment).filter(k => (oldRoot.environment[k] ?? null) !== controlledFixtures.environment[k]);
assert.deepEqual(controlledEnvironmentChanges, ['GOCACHE']);
assert.deepEqual(controlledFixtures.top_level_counts, { pass: 2, fail: 0, skip: 0 });
const portable = [];
for (const [label, original, count] of [['independent-portable-deterministicio', 'sealed-final-deterministicio', 36], ['independent-portable-cli', 'sealed-final-cli', 90], ['independent-portable-runner', 'sealed-final-runner-focused', 6]]) {
  const actual = receipt(label), old = json(worker + '/' + original + '.json');
  assert.equal(actual.exit, 0);
  assert.deepEqual(actual.top_level_counts, { pass: count, fail: 0, skip: 0 });
  const selected = old.tests.filter(t => t.action === 'pass' && !t.test.includes('/')).map(t => t.test);
  assert.deepEqual(sorted(actual.tests), sorted(old.tests.filter(t => selected.includes(t.test.split('/')[0]))));
  portable.push({ label, counts: actual.top_level_counts, terminal_identities: actual.tests.length });
}
for (const label of ['independent-empty-cache-normal', 'independent-recovery-failure-recheck', 'independent-validate', 'independent-scoped-vet', 'independent-scoped-errortype', 'independent-focused-format', 'independent-checker-controls']) assert.equal(receipt(label).exit, 0, label);
for (const [label, failedChildren] of [['independent-empty-cache-mutant', 1], ['independent-completion-cause-mutant', 2], ['independent-world-seed-mutant', 1]]) {
  const result = receipt(label);
  assert.equal(result.exit, 1);
  assert.equal(result.tests.filter(t => t.test.includes('/') && t.action === 'fail').length, failedChildren);
}
const observations = ['independent-deterministicio', 'independent-cli', 'independent-runner-mapped'].map(label => {
  const r = receipt(label);
  assert.equal(r.exit, 1);
  return { label, counts: r.top_level_counts, child_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(a => [a, r.tests.filter(t => t.test.includes('/') && t.action === a).length])), failed_tops: r.tests.filter(t => !t.test.includes('/') && t.action === 'fail').map(t => t.test) };
});
const diagnostics = path => [...read(path).toString().matchAll(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)].map(m => ({ path: m[1], line: Number(m[2]), column: Number(m[3]), message: m[4], rule: m[5] }));
for (const [label, original, count] of [['independent-fast-lint', 'sealed-final-configured-fast-lint', 3], ['independent-scoped-lint', 'sealed-final-unfiltered-scoped-lint', 92]]) {
  assert.equal(receipt(label).exit, 2);
  const actual = diagnostics(out + '/' + label + '.stdout');
  assert.equal(actual.length, count);
  assert.deepEqual(actual, diagnostics(worker + '/' + original + '.stdout'));
}
const lint = json(worker + '/final-execution-analysis.json').lint;
for (const finding of [...lint.fast, ...lint.scoped]) {
  assert.equal(read(finding.path).toString().split('\n')[finding.line - 1], finding.statement);
  assert.equal(sha(read(finding.path)), finding.file_sha256);
  assert.equal(sha(git(['diff', '--no-ext-diff', finding.parent, finding.owner, '--', finding.path])), finding.owner_patch_sha256);
  assert.equal(finding.inherited_statement, true);
  assert.equal(finding.waived, false);
}
assert.equal(receipt('independent-nested-format').exit, 1);
assert.equal(json(out + '/independent-nested-format.stdout').raw_exit, 0);
assert.equal(json(out + '/independent-nested-format.stdout').stdout, 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go\n');
const result = {
  verified: true, timestamp: new Date().toISOString(), head: git(['rev-parse', 'HEAD']).toString().trim(), source_identity_sha256: identity,
  worker_manifest_files: 448, worker_terminal_seal_sha256: sha(read(worker + '/terminal-seal.json')), source_proof_recomputed_in_memory_without_writes: true,
  source_identity_scopes: { conductor: 'all tracked non-.flow files, sorted', worker: 'three tracked Gomad module roots; every entry checked exact' },
  mapping_edges: 282, original_unaffected_functions: 165, accepted_task6_exact_bodies: 383, accepted_task3_exact_bodies: 4,
  cache_check: { full_root_environment_changes: environmentChanges, launcher_path_difference_explicit: true, controlled_fixture_environment_changes: controlledEnvironmentChanges, earlier_root_exit: 1, isolated_root_exit: 0, isolated_counts: rootResult.top_level_counts, isolated_children_pass: 59, controlled_fixture_counts: controlledFixtures.top_level_counts, diagnosis: 'Current isolated-cache root and exact-PATH fixture selectors pass. Cache-local failure supported; exhaustion versus corruption not distinguished. Shared cache untouched by conductor.', native: false },
  portable, complete_root_plus_portable_passing_top_identities: 151, empty_cache_partial_parent_control_separate: true,
  mixed_observations_retained: observations, recovery: { package_run: 'FAIL permission denied at directory.Sync; unchanged source', bounded_recheck: 'PASS', portable_selection: 'PASS', cause_established: false, fixed_claim: false },
  exact_restored_mutations_rejected: ['empty-cache identity ordering', 'two private SemanticChoice Causes', 'original World7/8 mismatch'],
  scoped_lint_findings: 92, fast_lint_findings: 3, global_lint_pass: false, whole_source_format_pass: false, native_pass: false, full_host_pass: false,
  imported_audits: ['restoration-source-audit.mjs', 'three-file-restoration-audit.mjs', 'world-and-cause-restoration-audit.mjs', 'capture.mjs', 'independent-gates.mjs', 'portable-controls.mjs', 'controlled-cache-fixtures.mjs'].map(path => ({ path, sha256: sha(read(out + '/' + path)) })),
  review_verdict: null, approved_new_exceptions: [], production_edits: [],
};
writeFileSync(resolve(out, 'audit.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(result));
