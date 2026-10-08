import assert from 'node:assert/strict';
import { writeFileSync, statSync } from 'node:fs';
import { resolve } from 'node:path';
import { root, worker, out, read, sha, sources } from './capture.mjs';
import { finalProof } from '../source-acceptance-20261008/final-proof.mjs';
import { verify } from '../source-acceptance-20261008/verify.mjs';
import { spans } from '../../task-4/source-acceptance-20261008/proof.mjs';
import { git } from '../source-acceptance-20261008/capture.mjs';
const json = p => JSON.parse(read(p)), normalize = x => JSON.parse(JSON.stringify(x));
const frozen = json(resolve(worker, 'final-source-proof.json'));
const seal = json(resolve(worker, 'terminal-seal.json'));
for (const f of seal.files) {
  const bytes = read(resolve(worker, f.name));
  assert.equal(bytes.length, f.bytes, f.name);
  assert.equal(sha(bytes), f.sha256, f.name);
}
assert.equal(seal.files.length, 150);
assert.deepEqual(normalize(finalProof()), frozen);
const verified = verify();
assert.equal(verified.frozen_files, 139);
assert.equal(frozen.test_function_mapping.length, 403);
for (const f of frozen.imported_helpers.concat(frozen.user_files)) assert.equal(sha(read(f.path)), f.sha256, f.path);
for (const row of frozen.owners) assert.equal(sha(read(row.path)), row.current_sha256, row.path);
for (const row of frozen.test_function_mapping) {
  assert.equal(sha(row.original.body), row.original_sha256);
  if (row.current) {
    assert.equal(sha(row.current.body), row.current_sha256);
    const currentPath = row.current.path ?? row.path;
    assert.equal(spans(read(currentPath).toString()).find(f => f.signature === row.current.signature)?.body, row.current.body, row.test);
  } else {
    assert(row.consolidation_mapping.length > 0);
    assert(row.consolidation_mapping.every(m => m[3] === '-' && m[5].startsWith('removed:')));
  }
}
for (const row of frozen.housekeeping_current_stat) {
  let code = null;
  try { statSync(resolve(root, row.path)); } catch (e) { code = e.code; }
  assert.equal(code, 'ENOENT', row.path);
}
const identity = sha(JSON.stringify(sources()));
assert.equal(identity, frozen.source_identity_sha256);
const events = path => read(path).toString().split('\n').flatMap(l => { try { return [JSON.parse(l)]; } catch { return []; } });
const sort = xs => xs.map(x => JSON.stringify(x)).sort();
const receipt = label => {
  const r = json(resolve(out, label + '.json'));
  assert.equal(sha(read(resolve(out, label + '.stdout'))), r.stdout_sha256);
  assert.equal(sha(read(resolve(out, label + '.stderr'))), r.stderr_sha256);
  assert.equal(r.sources_before_sha256, identity);
  assert.equal(r.sources_after_sha256, identity);
  assert.deepEqual(r.source_changes, []);
  assert.equal(r.signal, null);
  assert.equal(r.error, null);
  assert.equal(r.native, false);
  const actual = events(resolve(out, label + '.stdout')).filter(e => e.Test && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => ({ package: e.Package, test: e.Test, action: e.Action }));
  assert.deepEqual(r.tests, actual);
  for (const action of ['pass', 'fail', 'skip']) assert.equal(r.top_level_counts[action], actual.filter(e => e.action === action && !e.test.includes('/')).length);
  return r;
};
const suites = ['portable-private-operations', 'portable-api-external', 'portable-execution-controls', 'portable-minimizer-plan'];
const topIdentities = [], rootGates = [];
for (const label of suites) {
  const original = json(resolve(worker, label + '.json')), actual = receipt('independent-' + label);
  assert.equal(actual.exit, 0);
  assert(actual.argv.includes('-tags') && actual.argv.includes('test_dep'));
  assert.deepEqual(actual.tool_sha256, original.tool_sha256);
  assert.deepEqual(actual.environment, original.environment);
  const selected = original.tests.filter(e => e.action === 'pass' && !e.test.includes('/')).map(e => e.test);
  const expected = original.tests.filter(e => selected.includes(e.test.split('/')[0]));
  assert.deepEqual(sort(actual.tests), sort(expected), label + ' exact terminal identity multiset');
  assert.equal(actual.top_level_counts.fail, 0);
  assert.equal(actual.top_level_counts.skip, 0);
  topIdentities.push(...actual.tests.filter(e => !e.test.includes('/')));
  rootGates.push({ label, counts: actual.top_level_counts, exit: actual.exit });
}
assert.equal(topIdentities.length, 81);
assert.equal(new Set(topIdentities.map(e => e.package + ':' + e.test)).size, 81);
for (const label of ['configured-generators', 'configured-vet', 'configured-errortype', 'public-api-ast', 'current-godoc', 'whole-format', 'task6-format']) assert.equal(receipt('independent-' + label).exit, 0, label);
assert.equal(read(resolve(out, 'independent-task6-format.stdout')).length, 0);
assert.equal(read(resolve(out, 'independent-whole-format.stdout')).toString().trim(), 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go');
for (const [label, message] of [['external-negative-field', 'unknown field Executor'], ['external-negative-internal', 'use of internal package']]) {
  assert.equal(receipt('independent-' + label).exit, 1);
  assert(read(resolve(out, 'independent-' + label + '.stderr')).toString().includes(message));
}
const lint = json(resolve(worker, 'lint-attribution.json'));
const diagnostics = text => [...text.matchAll(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/gm)].map(m => ({ path: m[1], line: Number(m[2]), column: Number(m[3]), message: m[4], rule: m[5] }));
for (const [label, count] of [['unfiltered-scoped-lint', 91], ['configured-fast-lint', 66]]) {
  assert.equal(receipt('independent-' + label).exit, 2);
  const actual = diagnostics(read(resolve(out, 'independent-' + label + '.stdout')).toString());
  assert.equal(actual.length, count);
  assert.deepEqual(sort(actual), sort(diagnostics(read(resolve(worker, label + '.stdout')).toString())));
}
for (const f of lint.findings) {
  assert.equal(read(f.path).toString().split('\n')[f.line - 1], f.exact_line);
  assert.equal(spans(read(f.path).toString()).find(g => g.signature === f.current_function.signature)?.body, f.current_function.body);
  assert.equal(sha(f.current_function.body), f.current_function.sha256);
  assert.equal(git(['blame', '--porcelain', '-L', f.line + ',' + f.line, '--', f.path]).toString(), f.blame);
  assert.equal(f.worker_introduced, false);
  assert.equal(f.exception_applied, false);
}
const observations = json(resolve(worker, 'verified-observations.json'));
assert.deepEqual(observations.suites.map(s => s.failures.filter(f => !f.test.includes('/')).length), [41, 0, 4, 0]);
assert.deepEqual(observations.suites.map(s => s.failures.length), [53, 0, 4, 0]);
for (const suite of observations.suites) for (const f of suite.failures) {
  assert.equal(f.requested_operation_behavior_validated, false);
  assert(f.raw_events.length > 0);
  assert(!f.causality.startsWith('Unclassified'), f.test);
}
for (const source of observations.source) {
  assert.equal(sha(read(source.binding.path)), source.binding.sha256);
  for (const f of source.functions) assert.equal(spans(read(source.binding.path).toString()).find(g => g.signature === f.signature)?.body, f.body);
}
const result = {
  verified: true, timestamp: new Date().toISOString(), head: git(['rev-parse', 'HEAD']).toString().trim(),
  source_identity_sha256: identity, terminal_seal_sha256: sha(read(resolve(worker, 'terminal-seal.json'))), sealed_files: 150,
  independently_recomputed_source_proof: true, original_test_mappings: 403, native_failures_retained: 45,
  exact_portable_top_level_tests: 81, exact_terminal_identity_multisets: true, root_gates: rootGates,
  scoped_lint_findings: 91, fast_lint_findings: 66, whole_source_format_pass: false,
  original_task6_source_format_pass: true, native_pass: false, full_host_pass: false,
  product_source_changes: [], approved_new_exceptions: [], review_verdict: null,
  board_scope: 'Frozen in-progress claim; later lifecycle changes require narrow rebinding, not rewriting worker evidence.',
};
writeFileSync(resolve(out, 'audit.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(result));
