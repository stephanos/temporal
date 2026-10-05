import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const base = 'b1054ecc0968cb1d9b957c3b98ef6dd8c340c414';
const historical = '56148912df17e105dab3ec4b9e250ff5ef813318';
const prefix = 'tools/gomad3/internal/compatibilitypack/';
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const git = args => {
  const result = spawnSync('git', args, { cwd: root, maxBuffer: 32 << 20 });
  if (result.status !== 0) throw new Error(result.stderr.toString());
  return result.stdout;
};
const actual = path => readFileSync(resolve(root, path));
const expectedChanges = [
  'authoring/generate_test.go', 'authoring/refresh_test.go', 'evidence_test.go',
  'policy_test.go', 'generation.json', 'packs_generated_test.go', 'working-directories.json',
].map(path => prefix + path);
expectedChanges.push('tools/gomad3/internal/gomadtool/architecture/architecture.go', 'tools/gomad3/architecture_test.go');
const restored = [
  'packs/modernc-libc-xsys-v041.json', 'requests/modernc-libc-xsys-v041.json',
  'reports/modernc-libc-xsys-v041.md', 'testdata/v041/go.mod',
  'testdata/v041/go.sum', 'testdata/v041/libc_test.go',
].map(path => prefix + path);
const assert = (condition, message) => { if (!condition) throw new Error(message); };
const historicalFiles = restored.map(path => {
  const before = git(['show', historical + ':' + path]), after = actual(path);
  const equal = before.equals(after);
  assert(equal, 'historical bytes changed: ' + path);
  return { path, historical_sha256: hash(before), final_sha256: hash(after), byte_equal: equal };
});
const checks = JSON.parse(actual(artifacts + '/checks.json'));
const baselineOutputs = checks.freezes.BASE.generated;
assert(JSON.stringify(checks.freezes.FINAL.generated) === JSON.stringify(checks.freezes.FINAL3.generated), 'classification or metadata refinement changed pack/fixture outputs');
const finalOutputs = checks.freezes.FINAL3.generated;
const aggregate = [prefix + 'generation.json', prefix + 'packs_generated_test.go', prefix + 'working-directories.json'];
const unrelatedOutputs = Object.keys(baselineOutputs).filter(path => !aggregate.includes(path));
assert(unrelatedOutputs.every(path => finalOutputs[path] === baselineOutputs[path]), 'unrelated output changed');
assert(Object.keys(finalOutputs).filter(path => !Object.hasOwn(baselineOutputs, path)).sort().join('\n') === [...restored].sort().join('\n'), 'unexpected new generated/fixture path');
const scope = ['tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'Makefile', '.github/.golangci.yml'];
const basePaths = git(['ls-tree', '-r', '--name-only', base, '--', ...scope]).toString().trim().split('\n').sort();
const changed = [];
const changedBindings = {};
for (const path of basePaths) {
  const before = git(['show', base + ':' + path]), after = actual(path);
  if (!before.equals(after)) {
    changed.push(path);
    changedBindings[path] = { base_sha256: hash(before), final_sha256: hash(after) };
  }
}
assert(changed.sort().join('\n') === [...expectedChanges].sort().join('\n'), 'changed tracked source outside expected closure');
const finalPaths = git(['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', ...scope]).toString().split('\0').filter(Boolean).sort();
assert(finalPaths.filter(path => !basePaths.includes(path)).join('\n') === [...restored].sort().join('\n'), 'unexpected added source');
const digest = createHash('sha256');
for (const path of finalPaths) digest.update(path + '\0' + hash(actual(path)) + '\0');
assert(digest.digest('hex') === checks.freezes.FINAL3.source.sha256, 'final source no longer frozen');
const beforeState = JSON.parse(git(['show', base + ':' + prefix + 'generation.json']));
const afterState = JSON.parse(actual(prefix + 'generation.json'));
const statePreserved = beforeState.outputs.filter(output => output.path !== 'packs_generated_test.go').every(output => afterState.outputs.some(next => next.path === output.path && next.sha256 === output.sha256));
assert(statePreserved, 'unrelated generation output binding changed');
const mutationCount = actual(prefix + 'packs_generated_test.go').toString().split('\n').filter(line => line.includes('PackID: "modernc-libc-xsys-v041"')).length;
assert(mutationCount === 21, 'wrong v041 mutation count');
const oldRegistry = git(['show', base + ':' + prefix + 'packs_generated_test.go']).toString();
const currentRegistry = actual(prefix + 'packs_generated_test.go').toString();
assert(currentRegistry.split('\n').filter(line => !line.includes('PackID: "modernc-libc-xsys-v041"')).join('\n') === oldRegistry, 'unrelated registry bytes changed');
const oldMapping = JSON.parse(git(['show', base + ':' + prefix + 'working-directories.json']));
const mapping = JSON.parse(actual(prefix + 'working-directories.json'));
assert(JSON.stringify(mapping.requests.filter(entry => entry.request !== 'modernc-libc-xsys-v041')) === JSON.stringify(oldMapping.requests), 'unrelated mapping changed');
assert(mapping.requests.find(entry => entry.request === 'modernc-libc-xsys-v041')?.directory === 'testdata/v041', 'wrong v041 directory');
const request = JSON.parse(actual(prefix + 'requests/modernc-libc-xsys-v041.json'));
assert(request.approval_sha256 === 'sha256:0ea7c3348fb859e204f7cbfd36db3a735b99261597a3e33260e725b375fe8820', 'historical approval changed');
const packs = ['v041', 'v047'].map(version => JSON.parse(actual(prefix + 'packs/modernc-libc-xsys-' + version + '.json')));
const adapterPins = pack => pack.activation.filter(module => module.replacement.kind === 'adapter').map(module => ({path: module.path, version: module.version, sum: module.sum, profile: module.replacement.adapter.profile_implementation_sha256}));
assert(JSON.stringify(adapterPins(packs[0])) === JSON.stringify(adapterPins(packs[1])), 'demonstrated adapter/profile pin mismatch');
const taskPath = '.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.3.md';
const architecturePath = 'tools/gomad3/internal/gomadtool/architecture/architecture.go';
const architectureTestPath = 'tools/gomad3/architecture_test.go';
assert(actual(architecturePath).toString().replace('"internal/compatibilitypack/testdata/v041/go.mod": true, ', '') === git(['show', base + ':' + architecturePath]).toString(), 'architecture change exceeds exact module classification');
assert(actual(architectureTestPath).toString().replace('"internal/compatibilitypack/testdata/v041", ', '') === git(['show', base + ':' + architectureTestPath]).toString(), 'architecture test change exceeds matching fixture directory');
const suffix = bytes => bytes.toString().slice(bytes.toString().indexOf('## Done summary')).replace(/^Blocked:\n[\s\S]*?(?=^## Evidence\n)/m, 'Blocked:\n<Flow-managed runtime blocker excluded>\n');
assert(suffix(git(['show', base + ':' + taskPath])) === suffix(actual(taskPath)), 'historical Done narrative/Evidence changed');
const repeat = checks.runs.find(run => run.label === 'generate-repeat');
assert(repeat.exit_code === 0 && repeat.source_and_tool_inputs_stable, 'repeat generation not idempotent');
const proof = {
  base_commit: base, recovery_parent: historical,
  deletion_commit: '7fd67d5aaeca5d658ca45d571b7671a67ddbfc31',
  historical_files: historicalFiles, changed_source_bindings: changedBindings,
  source_closure: {base_path_count: basePaths.length, final_path_count: finalPaths.length, unchanged_existing_paths: basePaths.length - changed.length, added_paths: restored, final_sha256: checks.freezes.FINAL3.source.sha256, scope: 'All cached tracked plus nonignored untracked source paths under tools/gomad3, tools/gomad3sim, tools/gomad3integration, root Makefile and .github/.golangci.yml; includes all six restored files; no artifacts. Capture legacy tracked_path_count key counts this complete closure.'},
  outputs: {unrelated_output_count: unrelatedOutputs.length, unrelated_outputs_unchanged: true, unrelated_generation_manifest_bindings_unchanged: true, unrelated_registry_bytes_unchanged: true, unrelated_mapping_entries_unchanged: true, v041_mutation_count: mutationCount},
  approval: {historical: request.approval_sha256, preserved: true, new_approval_granted: false, discovery_performed: false, workload_qualification_performed: false, adapter_profile_pins_equal_current_v047: true, pins: adapterPins(packs[0])},
  task_historical_done_and_evidence_unchanged: true,
  architecture_classification: {exact_literal_module_entry_only: true, exact_synthetic_fixture_directory_only: true, source_exclusions_and_refusal_predicates_unchanged: true},
  task_historical_comparison_scope: 'Done narrative and Evidence retained exactly; only the Flow-managed Blocked: segment before ## Evidence is excluded from comparison so the conductor can update current runtime blockers.',
  idempotence: {run: repeat.label, exit_code: repeat.exit_code, source_sha256: repeat.source_after},
};
writeFileSync(resolve(artifacts, 'preservation.json'), JSON.stringify(proof, null, 2) + '\n');
console.log(JSON.stringify({final_source_sha256: proof.source_closure.final_sha256, historical_files_equal: historicalFiles.length, changed_existing_paths: changed.length, unchanged_existing_paths: basePaths.length - changed.length, unrelated_outputs_preserved: unrelatedOutputs.length, v041_mutation_count: mutationCount, architecture_classification: proof.architecture_classification, task_historical_comparison_scope: proof.task_historical_comparison_scope}));
