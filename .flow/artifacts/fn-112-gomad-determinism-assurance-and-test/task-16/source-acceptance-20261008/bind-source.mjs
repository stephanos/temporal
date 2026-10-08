import assert from 'node:assert/strict';
import {existsSync, readFileSync, writeFileSync, statSync} from 'node:fs';
import {resolve} from 'node:path';
import {root, out, sha, git, sources, stock} from './capture.mjs';

assert(!existsSync(out + '/input-proof.json'), 'existing input proof; refusing overwrite');
const read = path => readFileSync(resolve(root, path));
const json = path => JSON.parse(read(path));
const bound = path => ({path, bytes: read(path).length, sha256: sha(read(path))});
const base = readFileSync(out + '/base_commit', 'utf8').trim();
const head = git(['rev-parse', 'HEAD']).toString().trim();
assert.equal(head, base, 'HEAD changed during worker source capture');
const nativeBase = 'd635e23f00d926a43b942f25a9d05bd0ccb72025';
const nativeFix = 'a3b9f80efab9356c0be2080779133337e2471ac0';
const executionBase = 'bce0407286458f995f640fba97b4708a5d05b412';
const executionFix = '6be0755fef727146f86bac6658436f6874680cc8';
const reviewPaths = [
  'tools/gomad3/artifact/store.go', 'tools/gomad3/artifact/store_test.go', 'tools/gomad3/artifact/publication.go',
  'tools/gomad3/artifact/open.go', 'tools/gomad3/artifact/manifest_copy.go',
  'tools/gomad3/runner/retention.go', 'tools/gomad3/runner/completion.go', 'tools/gomad3/runner/retention_test.go',
  'tools/gomad3/runner/runner.go', 'tools/gomad3/runner/choice_exploration_campaign.go', 'tools/gomad3/runner/simulation_exploration_campaign.go',
  'tools/gomad3/runner/internal/campaign/open_campaign.go', 'tools/gomad3/runner/internal/campaign/retained_evidence.go',
  'tools/gomad3/runner/internal/campaign/retained_evidence_test.go', 'tools/gomad3/runner/internal/campaign/campaign_journal.go',
  'tools/gomad3/record/identity.go', 'tools/gomad3/record/record.go', 'tools/gomad3/record/record_test.go',
  'tools/gomad3/cmd/gomad/retained_success_e2e_test.go', 'tools/gomad3/cmd/gomad/e2e_test.go',
  'tools/gomad3/cmd/gomad/internal/cli/cli.go', 'tools/gomad3/cmd/gomad/internal/cli/application.go',
];
const tracked = new Set(git(['ls-files', '-z']).toString().split('\0').filter(Boolean));
for (const path of reviewPaths) assert(tracked.has(path), 'review source missing ' + path);
const currentPaths = reviewPaths;
const taskNativePaths = ['tools/gomad3/artifact/store.go', 'tools/gomad3/artifact/store_test.go', 'tools/gomad3/runner/retention_test.go', 'tools/gomad3/cmd/gomad/retained_success_e2e_test.go'];
for (const ref of [nativeBase, nativeFix, executionBase, executionFix]) git(['merge-base', '--is-ancestor', ref, head]);
const history = taskNativePaths.map(path => ({path, fix_sha256: sha(git(['show', nativeFix + ':' + path])), current_sha256: sha(read(path)), subsequent_commits: git(['log', '--format=%H %s', nativeFix + '..' + head, '--', path]).toString().trim().split('\n').filter(Boolean)}));
const literals = ['sha256-8804bc935588b0e0ac9fd7f890e4da67', 'sha256:27c9b74965e1b7cb30ef6f914b8f028eda0072216f5df3794bd84f8536db5f9d', 'sha256:8804bc935588b0e0ac9fd7f890e4da6718d567133466c52672ce1b11e9b454be'];
const oldTest = git(['show', nativeFix + ':tools/gomad3/artifact/store_test.go']).toString();
const currentTest = read('tools/gomad3/artifact/store_test.go').toString();
const retained = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-16';
for (const value of literals) assert(oldTest.includes(value) && currentTest.includes(value) && read(retained + '/red.log').toString().includes(value));
const runtime = '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/source-acceptance-20261007';
const runtimeBindings = json(runtime + '/raw/bindings.json');
for (const [path, expected] of Object.entries(runtimeBindings.scoped_bindings)) assert.equal(sha(read(path)), expected, path);
assert.equal(Object.keys(runtimeBindings.scoped_bindings).length, 87);
const task5BindingPath = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/source-binding.json';
const task5Binding = json(task5BindingPath);
const exact5 = task5Binding.exact_bindings;
for (const entry of exact5) assert.equal(sha(read(entry.path)), entry.sha256, entry.path);
const archive = {...task5Binding.archive};
assert.equal(sha(read(archive.path)), archive.sha256);
assert.equal(statSync(resolve(root, archive.path)).size, 35109201);
const rawManifest = json(runtime + '/raw-manifest.json');
let rawBytes = 0;
for (const entry of rawManifest) {
  const stored = read(runtime + '/raw/' + (entry.retained_name ?? entry.name));
  if (entry.retained_name) assert.equal(sha(stored), entry.retained_sha256);
  const decoded = entry.encoding === 'utf8-json-string' ? Buffer.from(JSON.parse(stored).value, 'utf8') : stored;
  assert.equal(decoded.length, entry.bytes);
  assert.equal(sha(decoded), entry.sha256);
  rawBytes += decoded.length;
}
assert.equal(rawManifest.length, 72);
const retainedReceipts = ['README.md', 'gates.md', 'handover-summary.md', 'handover-evidence.json', 'red.log', 'green.log', 'focused-final.log', 'reachability-fmt.jsonl', 'inspect-before.txt', 'reachability-fixed.jsonl', 'inspect-after.json', 'cli-comparison.json', 'replay-seed-1.txt', 'replay-seed-2.txt', 'final-completion-summary.md', 'final-completion-evidence.json'].map(path => bound(retained + '/' + path));
const integrated = '.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-13';
const integratedBindings = json(integrated + '/integrated-source-hashes.json');
const integratedComparison = Object.entries(integratedBindings.sources).map(([path, expected]) => {
  const present = existsSync(resolve(root, path));
  return {path, retained_sha256: expected, current_sha256: present ? sha(read(path)) : null, equal: present && sha(read(path)) === expected, status: present ? 'current path compared' : 'historical path absent after later module extraction; no current binding claimed'};
});
const cli = json(retained + '/cli-comparison.json');
assert.equal(cli.platform, 'darwin/arm64');
assert.equal(cli.artifacts.length, 2);
assert.equal(cli.artifacts[0].outcome_signature, cli.artifacts[1].outcome_signature);
assert.notEqual(cli.artifacts[0].record_hash, cli.artifacts[1].record_hash);
assert.notEqual(cli.artifacts[0].path, cli.artifacts[1].path);
assert(read(retained + '/inspect-before.txt').toString().includes('validate published success artifact 2: retained success artifact does not match its campaign execution'));
for (const seed of [1, 2]) assert(read(retained + '/replay-seed-' + seed + '.txt').toString().includes('reproduced=true'));
const users = [
  {path: '.turbo/plans/gomad3-glossary-update.md', sha256: '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188'},
  {path: '.turbo/technical-debt.md', sha256: 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'},
];
for (const entry of users) assert.equal(sha(read(entry.path)), entry.sha256);
const finalSources = sources();
assert(!Object.keys(finalSources).some(path => path.startsWith('.turbo/') || path.startsWith('.flow/')));
const changes = git(['diff', '--name-only', base, '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration']).toString().trim().split('\n').filter(Boolean);
assert.deepEqual(changes, ['tools/gomad3/runner/internal/campaign/retained_evidence_test.go']);
const standardsTools = ['/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype', stock + '/go', stock + '/gofmt'].map(path => ({path, sha256: sha(readFileSync(path))}));
const result = {head, base_commit: base, native_fix: nativeFix, native_prechange: nativeBase, experimental_fix: executionFix, experimental_prechange: executionBase, integration: 'ca3345469a2541b7686606268b0f19cde030a8fe', history, review_sources: currentPaths.map(bound), missing_optional_review_paths: reviewPaths.filter(path => !tracked.has(path)), literal_noncollision_identities: literals, sources: finalSources, source_count: Object.keys(finalSources).length, sources_sha256: sha(JSON.stringify(finalSources)), product_changes: changes, historical_receipts: retainedReceipts, historical_integrated_receipts: ['integrated-source-hashes.json', 'integrated-test-host-green.log', 'integrated-review.json'].map(path => bound(integrated + '/' + path)), integrated_source_comparison: integratedComparison, runtime_reuse: {binding: bound(runtime + '/raw/bindings.json'), task5_binding: bound(task5BindingPath), exact_inputs: 87, raw_manifest: bound(runtime + '/raw-manifest.json'), raw_files: rawManifest.length, decoded_bytes: rawBytes, status: 'all exact inputs and retained raw bytes verified; both supported source sets are static, no native qualification; no whole-checkout historical equality claim'}, archive, standards_tools: standardsTools, standards_config: bound('.github/.golangci.yml'), lint_route: [bound('Makefile'), bound('cmd/tools/lintcode/main.go'), bound('tools/gomad3/internal/gomadtool/architecture/architecture.go')], user_files: users, excluded_user_scope: users.map(entry => entry.path), native_qualification: false};
writeFileSync(out + '/input-proof.json', JSON.stringify(result, null, 2) + '\n');
console.log(JSON.stringify({head, source_count: result.source_count, review_sources: currentPaths.length, runtime_exact_inputs: 87, retained_runtime_raw_files: rawManifest.length, native_qualification: false, product_changes: changes, missing_optional_review_paths: result.missing_optional_review_paths}));
