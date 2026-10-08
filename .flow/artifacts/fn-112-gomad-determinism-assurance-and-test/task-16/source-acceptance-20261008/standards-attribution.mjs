import assert from 'node:assert/strict';
import {readFileSync, writeFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {root, out, sha, git} from './capture.mjs';

const base = readFileSync(out + '/base_commit', 'utf8').trim();
const nativeTaskPaths = new Set(['tools/gomad3/artifact/store.go', 'tools/gomad3/artifact/store_test.go', 'tools/gomad3/runner/retention_test.go', 'tools/gomad3/cmd/gomad/retained_success_e2e_test.go']);
const nativeBase = 'd635e23f00d926a43b942f25a9d05bd0ccb72025';
const nativeFix = 'a3b9f80efab9356c0be2080779133337e2471ac0';
const owner = path => {
  if (path.includes('/artifact/manifest_copy.go')) return {task: 'fn-109.12 / fn-109.19 / fn-109.21', disclosure: 'fn-109.33 explicitly preserves this production panic and states it is not waived'};
  if (path.includes('/cmd/gomad/internal/cli/application.go')) return {task: 'fn-109.4 / fn-109.21', disclosure: 'construction owner retains original writer/error behavior; task4 historical summary separately discloses output handling'};
  if (path.includes('/cmd/gomad/internal/cli/cli.go')) return {task: 'fn-109.26 / fn-109.21', disclosure: 'CLI semantic-ownership and aggregate standards reconciliation; no task16 output-error redesign'};
  if (path.includes('/runner/replay_operation.go')) return {task: 'fn-109.12 / fn-109.21', disclosure: 'opened-handle migration source; existing uppercase World diagnostics remain unwaived'};
  if (path.includes('/cmd/gomadtool/soak.go')) return {task: 'fn-112.10', disclosure: 'four writer findings explicitly retained in trusted blocked-task reassessment'};
  if (path.includes('/cmd/gomadtool/') || path.includes('/upgrade/adapterregen/')) return {task: 'fn-113 / fn-109.21', disclosure: 'pin/adapter maintainer and aggregate standards owners; exact correction admission remains theirs'};
  if (path.includes('/internal/gomadtool/architecture/')) return {task: 'fn-109.19 / fn-109.21', disclosure: 'architecture/aggregate standards owners'};
  if (path.includes('/runner/internal/execution/descriptor_dup_linux_test.go')) return {task: 'fn-109.21', disclosure: 'same two ordinary-source descriptor cleanup findings retained in task5 standards attribution; no native transfer excuses them'};
  if (path.includes('/world/')) return {task: 'fn-109.19 / fn-109.21', disclosure: 'World/aggregate standards owners; error strings unwaived'};
  return {task: 'fn-109.21', disclosure: 'aggregate owner must reconcile, no suppression or new ownership invented'};
};
const observations = {};
for (const label of ['configured-fast-lint', 'task-source-lint']) {
  const log = readFileSync(out + '/' + label + '.stdout', 'utf8');
  const issues = [...log.matchAll(/^(tools\/gomad3\/[^:\n]+):(\d+):(\d+): (.+)\n\s+([^\n]+)\n/gm)].map(match => {
    const [_, path, lineText, column, message, printedCode] = match;
    const line = Number(lineText);
    const source = readFileSync(resolve(root, path), 'utf8');
    const baseSource = git(['show', base + ':' + path]).toString();
    const code = source.split('\n')[line - 1].trim();
    assert.equal(printedCode.trim(), code);
    assert.equal(baseSource.split('\n')[line - 1].trim(), code, 'lint finding changed by worker ' + path);
    const blame = git(['blame', '-L', line + ',' + line, '--porcelain', base, '--', path]).toString();
    const commit = blame.split(' ')[0];
    const filename = blame.match(/^filename (.+)$/m)?.[1];
    const summary = blame.match(/^summary (.+)$/m)?.[1];
    assert(!nativeTaskPaths.has(path), 'task16 original implementation file now has a lint finding requiring direct reconciliation');
    return {path, line, column: Number(column), message, code, code_sha256: sha(code), source_sha256: sha(source), unchanged_at_worker_base: true, blame: {commit, filename, summary}, original_native_task_surface: false, external_owner: owner(path), waiver: false};
  });
  assert.equal(issues.length, label === 'configured-fast-lint' ? 68 : 12);
  observations[label] = {receipt: label + '.json', actual_exit: JSON.parse(readFileSync(out + '/' + label + '.json')).exit, issues};
}
const result = {worker_base: base, original_native_base: nativeBase, original_native_fix: nativeFix, original_native_task_paths: [...nativeTaskPaths], worker_product_change: 'tools/gomad3/runner/internal/campaign/retained_evidence_test.go', observations, original_task_findings: 0, worker_introduced_findings: 0, final_changed_package_receipt: 'final-campaign-lint.json', final_changed_package_exit: JSON.parse(readFileSync(out + '/final-campaign-lint.json')).exit, configured_vet_receipt: 'configured-vet.json', aggregate_green: false, suppressions_added: false, authority: 'causal attribution only; original-base aggregate gate remains red with its separately established owners; conductor decides source acceptance'};
writeFileSync(out + '/standards-attribution.json', JSON.stringify(result, null, 2) + '\n');
console.log(JSON.stringify({fast_findings: 68, explicit_task_package_findings: 12, original_task_findings: 0, worker_introduced_findings: 0, final_changed_package_exit: result.final_changed_package_exit, aggregate_green: false}));
